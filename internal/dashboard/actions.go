package dashboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// ErrNoTargetVersion is returned when no non-current version exists to act on.
var ErrNoTargetVersion = errors.New("dashboard: no target version to ramp/promote")

// recoverQuery selects every open PizzaOrder workflow; the bad-build filtering is
// done in Go from each run's pinned Build ID (no reliance on an uncertain search
// attribute name). The oldest-first ordering that makes truncation deterministic
// (oldest-stuck orders recovered first) is applied in Go after paging, because the
// dev server's standard (SQLite) visibility store does not support ORDER BY.
const recoverQuery = "WorkflowType = 'PizzaOrder' AND ExecutionStatus = 'Running'"

// maxRecoverPerCall caps how many stuck orders one recover call resets, so a huge
// backlog cannot turn into an unbounded burst of reset RPCs. Truncation is logged.
const maxRecoverPerCall = 200

// Actions turns operator intents into Temporal API calls.
type Actions struct {
	c              client.Client
	deploymentName string
	namespace      string
	logger         *slog.Logger
}

// NewActions builds an Actions over the given client and deployment.
func NewActions(c client.Client, deploymentName, namespace string, logger *slog.Logger) *Actions {
	return &Actions{c: c, deploymentName: deploymentName, namespace: namespace, logger: logger}
}

// Ramp routes pct% of new orders to the newest (target) version.
func (a *Actions) Ramp(ctx context.Context, pct float32) error {
	target, _, err := a.targetAndCurrent(ctx)
	if err != nil {
		return err
	}
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	// Demo: the target version's pollers may not be registered yet (single
	// replica, freshly-shipped version), and its task queues may differ from the
	// current version. We accept routing to it anyway rather than fail the call.
	if _, err := h.SetRampingVersion(ctx, client.WorkerDeploymentSetRampingVersionOptions{
		BuildID:                 target,
		Percentage:              pct,
		AllowNoPollers:          true,
		IgnoreMissingTaskQueues: true,
	}); err != nil {
		return fmt.Errorf("set ramping version %q to %.0f%%: %w", target, pct, err)
	}
	return nil
}

// Promote makes the ramping/newest version Current (full cutover).
func (a *Actions) Promote(ctx context.Context) error {
	target, _, err := a.targetAndCurrent(ctx)
	if err != nil {
		return err
	}
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	// Demo: the target version's pollers may not be registered yet (single
	// replica, freshly-shipped version), and its task queues may differ from the
	// current version. We accept the full cutover anyway rather than fail the call.
	if _, err := h.SetCurrentVersion(ctx, client.WorkerDeploymentSetCurrentVersionOptions{
		BuildID:                 target,
		AllowNoPollers:          true,
		IgnoreMissingTaskQueues: true,
	}); err != nil {
		return fmt.Errorf("set current version %q: %w", target, err)
	}
	return nil
}

// Rollback removes the ramp: 100% of new orders snap back to Current.
func (a *Actions) Rollback(ctx context.Context) error {
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	// Percentage:0 is what clears the ramping version: an empty BuildID alone
	// maps to "unversioned" rather than "no ramp". Clearing the ramp is safe here
	// only because Current is non-nil, so new orders snap back to Current.
	if _, err := h.SetRampingVersion(ctx, client.WorkerDeploymentSetRampingVersionOptions{
		BuildID:    "",
		Percentage: 0,
	}); err != nil {
		return fmt.Errorf("remove ramping version: %w", err)
	}
	return nil
}

// Recover batch-resets every open order pinned to the bad (ramping/newest) build
// and moves each onto the current build, atomically (reset-with-move).
func (a *Actions) Recover(ctx context.Context) (int, error) {
	bad, good, err := a.targetAndCurrent(ctx)
	if err != nil {
		return 0, err
	}
	if good == "" {
		return 0, errors.New("dashboard: no current version to recover onto")
	}

	// Page through every result: the standard visibility store does not support
	// ORDER BY, so we cannot rely on server-side ordering for deterministic
	// truncation and must collect all matches before sorting. The demo's open-order
	// set is small; maxRecoverPerCall bounds the reset RPCs, not the listing.
	var stuck []*workflowpb.WorkflowExecutionInfo
	var pageToken []byte
	for {
		resp, err := a.c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Query:         recoverQuery,
			NextPageToken: pageToken,
		})
		if err != nil {
			return 0, fmt.Errorf("list open orders: %w", err)
		}
		for _, exec := range resp.Executions {
			if pinnedBuildID(exec) == bad {
				stuck = append(stuck, exec)
			}
		}
		pageToken = resp.NextPageToken
		if len(pageToken) == 0 {
			break
		}
	}

	// Sort oldest-first in Go so truncation is deterministic: when the cap is hit,
	// the oldest-stuck orders are recovered first.
	sort.Slice(stuck, func(i, j int) bool {
		return stuck[i].GetStartTime().AsTime().Before(stuck[j].GetStartTime().AsTime())
	})

	truncated := false
	if len(stuck) > maxRecoverPerCall {
		truncated = true
		stuck = stuck[:maxRecoverPerCall]
	}
	if truncated {
		a.logger.Info("capping recover batch", "cap", maxRecoverPerCall)
	}

	recovered := 0
	for _, exec := range stuck {
		if err := a.resetWithMove(ctx, exec.GetExecution(), good); err != nil {
			a.logger.Warn("recover: reset failed", "workflowId", exec.GetExecution().GetWorkflowId(), "err", err)
			continue
		}
		recovered++
	}
	a.logger.Info("recover completed", "badBuild", bad, "goodBuild", good,
		"candidates", len(stuck), "recovered", recovered)
	return recovered, nil
}

// resetWithMove resets the listed run to its first completed Workflow Task and
// pins the resulting new run to the good (current) build via a PostResetOperation.
// Assumes these are pinned, non-continue-as-new workflows: the first completed
// Workflow Task is a valid reset point and the run carries a single build ID.
func (a *Actions) resetWithMove(ctx context.Context, exec *commonpb.WorkflowExecution, goodBuild string) error {
	resetPoint, err := a.firstWorkflowTaskCompletedID(ctx, exec)
	if err != nil {
		return err
	}

	req := &workflowservice.ResetWorkflowExecutionRequest{
		Namespace:                 a.namespace,
		WorkflowExecution:         exec,
		Reason:                    "recover stuck orders: bad drone-delivery build",
		RequestId:                 uuid.NewString(),
		WorkflowTaskFinishEventId: resetPoint,
		ResetReapplyType:          enumspb.RESET_REAPPLY_TYPE_SIGNAL,
		PostResetOperations: []*workflowpb.PostResetOperation{{
			Variant: &workflowpb.PostResetOperation_UpdateWorkflowOptions_{
				UpdateWorkflowOptions: &workflowpb.PostResetOperation_UpdateWorkflowOptions{
					WorkflowExecutionOptions: &workflowpb.WorkflowExecutionOptions{
						VersioningOverride: &workflowpb.VersioningOverride{
							Override: &workflowpb.VersioningOverride_Pinned{
								Pinned: &workflowpb.VersioningOverride_PinnedOverride{
									Behavior: workflowpb.VersioningOverride_PINNED_OVERRIDE_BEHAVIOR_PINNED,
									Version: &deploymentpb.WorkerDeploymentVersion{
										DeploymentName: a.deploymentName,
										BuildId:        goodBuild,
									},
								},
							},
						},
					},
					UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"versioning_override"}},
				},
			},
		}},
	}
	if _, err := a.c.ResetWorkflowExecution(ctx, req); err != nil {
		return fmt.Errorf("reset workflow: %w", err)
	}
	return nil
}

// firstWorkflowTaskCompletedID scans history for the first WorkflowTaskCompleted
// event, the canonical reset point for a reset-with-move. Assumes a pinned,
// non-continue-as-new workflow, so the first such event reliably exists in this
// run's history.
func (a *Actions) firstWorkflowTaskCompletedID(ctx context.Context, exec *commonpb.WorkflowExecution) (int64, error) {
	iter := a.c.GetWorkflowHistory(ctx, exec.GetWorkflowId(), exec.GetRunId(), false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		ev, err := iter.Next()
		if err != nil {
			return 0, fmt.Errorf("read history: %w", err)
		}
		if ev.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED {
			return ev.GetEventId(), nil
		}
	}
	return 0, fmt.Errorf("no WorkflowTaskCompleted event for %s", exec.GetWorkflowId())
}

// targetAndCurrent resolves the target build (ramping if set, else newest
// non-current by CreateTime) and the current build from the live routing config.
func (a *Actions) targetAndCurrent(ctx context.Context) (target, current string, err error) {
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	resp, err := h.Describe(ctx, client.WorkerDeploymentDescribeOptions{})
	if err != nil {
		return "", "", fmt.Errorf("describe worker deployment %q: %w", a.deploymentName, err)
	}

	rc := resp.Info.RoutingConfig
	if rc.CurrentVersion != nil {
		current = rc.CurrentVersion.BuildID
	}
	if rc.RampingVersion != nil {
		return rc.RampingVersion.BuildID, current, nil
	}

	// No ramp: target the newest version summary that is not the current build.
	summaries := append([]client.WorkerDeploymentVersionSummary(nil), resp.Info.VersionSummaries...)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].CreateTime.After(summaries[j].CreateTime)
	})
	for _, s := range summaries {
		if s.Version.BuildID != current {
			return s.Version.BuildID, current, nil
		}
	}
	return "", current, ErrNoTargetVersion
}
