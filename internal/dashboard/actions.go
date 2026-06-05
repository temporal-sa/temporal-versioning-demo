package dashboard

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

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

// ErrUnknownVersion is returned when the UI asks to act on a version label that
// is not registered in the deployment.
var ErrUnknownVersion = errors.New("dashboard: unknown version")

// recoverQuery selects every open PizzaOrder workflow; the bad-build filtering is
// done in Go from each run's pinned Build ID (no reliance on an uncertain search
// attribute name). The oldest-first ordering that makes truncation deterministic
// (oldest-stuck orders recovered first) is applied in Go after paging, because the
// dev server's standard (SQLite) visibility store does not support ORDER BY.
const recoverQuery = "WorkflowType = 'PizzaOrder' AND ExecutionStatus = 'Running'"

// maxRecoverPerCall caps how many stuck orders one recover call resets, so a huge
// backlog cannot turn into an unbounded burst of reset RPCs. Truncation is logged.
const maxRecoverPerCall = 200

// bootstrapTimeout bounds the EnsureCurrentVersion startup loop: if no version is
// registered within this window we give up and leave promotion to the operator.
const bootstrapTimeout = 2 * time.Minute

// Actions turns operator intents into Temporal API calls.
type Actions struct {
	c              client.Client
	deploymentName string
	namespace      string
	logger         *slog.Logger
	labelCache     map[string]string // buildID -> pizzaVersion
}

// NewActions builds an Actions over the given client and deployment.
func NewActions(c client.Client, deploymentName, namespace string, logger *slog.Logger) *Actions {
	return &Actions{
		c:              c,
		deploymentName: deploymentName,
		namespace:      namespace,
		logger:         logger,
		labelCache:     map[string]string{},
	}
}

// Ramp routes pct% of new orders to the named version (v1/v2/v3).
func (a *Actions) Ramp(ctx context.Context, version string, pct float32) error {
	labelToBuild, _, _, _, err := a.versionIndex(ctx)
	if err != nil {
		return err
	}
	buildID, ok := labelToBuild[version]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownVersion, version)
	}
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	if _, err := h.SetRampingVersion(ctx, client.WorkerDeploymentSetRampingVersionOptions{
		BuildID:                 buildID,
		Percentage:              pct,
		AllowNoPollers:          true,
		IgnoreMissingTaskQueues: true,
	}); err != nil {
		return fmt.Errorf("set ramping version %q to %.0f%%: %w", buildID, pct, err)
	}
	return nil
}

// Promote makes the named version Current (full cutover).
func (a *Actions) Promote(ctx context.Context, version string) error {
	labelToBuild, _, _, _, err := a.versionIndex(ctx)
	if err != nil {
		return err
	}
	buildID, ok := labelToBuild[version]
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownVersion, version)
	}
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	if _, err := h.SetCurrentVersion(ctx, client.WorkerDeploymentSetCurrentVersionOptions{
		BuildID:                 buildID,
		AllowNoPollers:          true,
		IgnoreMissingTaskQueues: true,
	}); err != nil {
		return fmt.Errorf("set current version %q: %w", buildID, err)
	}
	return nil
}

// versionIndex resolves the live label↔Build ID mapping (metadata-first, with a
// CreateTime fallback) plus the current/ramping Build IDs.
func (a *Actions) versionIndex(ctx context.Context) (labelToBuild, buildToLabel map[string]string, current, ramping string, err error) {
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	resp, err := h.Describe(ctx, client.WorkerDeploymentDescribeOptions{})
	if err != nil {
		return nil, nil, "", "", fmt.Errorf("describe worker deployment %q: %w", a.deploymentName, err)
	}
	rc := resp.Info.RoutingConfig
	if rc.CurrentVersion != nil {
		current = rc.CurrentVersion.BuildID
	}
	if rc.RampingVersion != nil {
		ramping = rc.RampingVersion.BuildID
	}

	sorted := append([]client.WorkerDeploymentVersionSummary(nil), resp.Info.VersionSummaries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].CreateTime.Before(sorted[j].CreateTime) })

	labelToBuild = map[string]string{}
	buildToLabel = map[string]string{}
	for i, s := range sorted {
		buildID := s.Version.BuildID
		labelVal, cached := a.labelCache[buildID]
		if !cached {
			if v, ferr := fetchVersionLabel(ctx, a.c, a.deploymentName, buildID); ferr == nil && v != "" {
				a.labelCache[buildID] = v
				labelVal = v
			}
		}
		if labelVal == "" {
			labelVal = friendly(i) // CreateTime fallback (oldest = v1)
		}
		labelToBuild[labelVal] = buildID
		buildToLabel[buildID] = labelVal
	}
	return labelToBuild, buildToLabel, current, ramping, nil
}

// bootstrapBuildID returns the Build ID labelled v1 from a buildID→label map,
// or "" if no version carries the v1 label yet.
func bootstrapBuildID(buildToLabel map[string]string) string {
	for buildID, label := range buildToLabel {
		if label == "v1" {
			return buildID
		}
	}
	return ""
}

// bootstrapDecision is the outcome of evaluating whether to auto-promote a first
// Current version at startup.
type bootstrapDecision int

const (
	// bootstrapSkip means a Current version is already set; never re-promote.
	bootstrapSkip bootstrapDecision = iota
	// bootstrapPromote means no Current version is set and a target is ready.
	bootstrapPromote
	// bootstrapWait means nothing is ready yet; retry later.
	bootstrapWait
)

// decideBootstrap maps the result of targetAndCurrent to a startup action. It is
// pure (no I/O) so the bootstrap rules can be unit-tested in isolation.
//
// Rules:
//   - a Current version already exists -> skip (keeps the manual ramp/promote/
//     rollback flow untouched, even if a newer target also exists);
//   - no Current and a target is ready -> promote that target;
//   - otherwise (ErrNoTargetVersion, describe error, empty target) -> wait.
func decideBootstrap(target, current string, err error) (bootstrapDecision, string) {
	if current != "" {
		return bootstrapSkip, ""
	}
	if err == nil && target != "" {
		return bootstrapPromote, target
	}
	return bootstrapWait, ""
}

// EnsureCurrentVersion makes sure the deployment has a Current version at startup
// by promoting the v1-labelled version (metadata-first, CreateTime fallback) when
// none is set.
//
// It runs everywhere (local and K8s) with no feature flag. Because it fires only
// while Current is nil, it acts once at bootstrap and then leaves later
// ramp/promote/rollback entirely to the operator. The loop is bounded by
// bootstrapTimeout; on timeout it logs and returns, so the operator can still
// promote manually. It always honors ctx cancellation.
func (a *Actions) EnsureCurrentVersion(ctx context.Context, pollInterval time.Duration) {
	deadline := time.NewTimer(bootstrapTimeout)
	defer deadline.Stop()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		_, buildToLabel, current, _, err := a.versionIndex(ctx)
		target := bootstrapBuildID(buildToLabel)
		switch decision, buildID := decideBootstrap(target, current, err); decision {
		case bootstrapSkip:
			a.logger.Info("current version already set, skipping bootstrap", "currentBuild", current)
			return
		case bootstrapPromote:
			if err := a.promoteBootstrap(ctx, buildID); err != nil {
				// Treat as transient: keep retrying until the deadline.
				a.logger.Warn("bootstrap promote failed, will retry", "build", buildID, "err", err)
				break
			}
			a.logger.Info("bootstrapped current version", "build", buildID)
			return
		case bootstrapWait:
			// Nothing registered yet; fall through to wait for the next tick. A
			// non-nil err here means describe is failing (e.g. Temporal unreachable);
			// log it so the cause is diagnosable before the deadline fires. Stay
			// silent on the normal "no version yet" case (err nil) to avoid noise.
			if err != nil {
				a.logger.Debug("bootstrap waiting on describe failure, will retry", "err", err)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-deadline.C:
			a.logger.Warn("no version registered in time for bootstrap; operator can still promote manually",
				"timeout", bootstrapTimeout)
			return
		case <-ticker.C:
		}
	}
}

// promoteBootstrap sets buildID as the Current version, reusing the same lenient
// options as Promote (the freshly-shipped version's pollers may not be registered
// yet, and its task queues may differ from any prior version).
func (a *Actions) promoteBootstrap(ctx context.Context, buildID string) error {
	h := a.c.WorkerDeploymentClient().GetHandle(a.deploymentName)
	if _, err := h.SetCurrentVersion(ctx, client.WorkerDeploymentSetCurrentVersionOptions{
		BuildID:                 buildID,
		AllowNoPollers:          true,
		IgnoreMissingTaskQueues: true,
	}); err != nil {
		return fmt.Errorf("set current version %q: %w", buildID, err)
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
