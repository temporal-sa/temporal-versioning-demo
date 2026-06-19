package dashboard

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// maxOrdersPerTick caps how many open orders are processed per poll, to keep each
// poll cheap. Orders are sorted oldest-first (see OpenOrders), so the cap keeps the
// newest maxOrdersPerTick orders (the tail) and never drops new ones. Truncation is
// logged (no silent caps).
const maxOrdersPerTick = 50

// openOrdersQuery selects open PizzaOrder workflows for one browser session.
// Results are sorted oldest-first in Go (see OpenOrders) rather than via an
// ORDER BY clause, because the dev server's standard (SQLite) visibility store
// does not support ORDER BY.
func openOrdersQuery(sessionID string) string {
	return fmt.Sprintf(
		"WorkflowType = 'PizzaOrder' AND ExecutionStatus = 'Running' AND %s = '%s'",
		SessionSearchAttribute,
		sessionID,
	)
}

// SDKReader implements TemporalReader using the Temporal SDK client.
type SDKReader struct {
	c              client.Client
	deploymentName string
	logger         *slog.Logger

	labels *LabelResolver
}

// NewSDKReader builds an SDK-backed TemporalReader for the given deployment. The
// label resolver is shared with the actions so the buildID→label cache is not
// duplicated.
func NewSDKReader(c client.Client, deploymentName string, labels *LabelResolver, logger *slog.Logger) *SDKReader {
	return &SDKReader{
		c:              c,
		deploymentName: deploymentName,
		logger:         logger,
		labels:         labels,
	}
}

// DeploymentSnapshot reads the deployment's routing config and version summaries.
func (r *SDKReader) DeploymentSnapshot(ctx context.Context) (Routing, []VersionSummary, error) {
	h := r.c.WorkerDeploymentClient().GetHandle(r.deploymentName)
	resp, err := h.Describe(ctx, client.WorkerDeploymentDescribeOptions{})
	if err != nil {
		return Routing{}, nil, fmt.Errorf("describe worker deployment %q: %w", r.deploymentName, err)
	}

	rc := resp.Info.RoutingConfig
	routing := Routing{
		RampingPct:     int(rc.RampingVersionPercentage),
		CurrentBuildID: currentBuildID(rc),
	}
	if rc.RampingVersion != nil {
		routing.RampingBuildID = rc.RampingVersion.BuildID
	}

	summaries := make([]VersionSummary, 0, len(resp.Info.VersionSummaries))
	for _, s := range resp.Info.VersionSummaries {
		buildID := s.Version.BuildID
		labelVal := r.labels.label(ctx, buildID)
		summaries = append(summaries, VersionSummary{
			BuildID:      buildID,
			PizzaVersion: labelVal,
			CreateTime:   s.CreateTime,
			Draining: s.DrainageStatus == client.WorkerDeploymentVersionDrainageStatusDraining ||
				s.DrainageStatus == client.WorkerDeploymentVersionDrainageStatusDrained,
		})
	}
	return routing, summaries, nil
}

// OpenOrders lists open PizzaOrder workflows for sessionID (capped, oldest
// first), queries getState on each, and returns them with their pinned Build ID
// and elapsed time. Orders that cannot be queried yet (e.g. mid-start) are
// skipped.
func (r *SDKReader) OpenOrders(ctx context.Context, sessionID string) ([]LiveOrder, error) {
	if !validSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID %q", sessionID)
	}
	// Namespace is left empty: the SDK fills it from the client's configuration.
	resp, err := r.c.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Query:    openOrdersQuery(sessionID),
		PageSize: maxOrdersPerTick,
	})
	if err != nil {
		return nil, fmt.Errorf("list open orders: %w", err)
	}

	// Sort oldest-first in Go for stable, jump-free rendering: new orders append at
	// the end instead of pushing every card down. The standard visibility store does
	// not support ORDER BY, so ordering cannot be pushed into the query.
	slices.SortFunc(resp.Executions, func(a, b *workflow.WorkflowExecutionInfo) int {
		return a.GetStartTime().AsTime().Compare(b.GetStartTime().AsTime())
	})

	// Keep the newest maxOrdersPerTick orders: since the slice is oldest-first, those
	// are the last elements. This way new orders are never dropped once the cap is hit.
	executions := resp.Executions
	if len(executions) > maxOrdersPerTick {
		r.logger.Info("truncating open orders for this tick",
			"total", len(executions), "cap", maxOrdersPerTick)
		executions = executions[len(executions)-maxOrdersPerTick:]
	}

	orders := make([]LiveOrder, 0, len(executions))
	for _, exec := range executions {
		wfID := exec.GetExecution().GetWorkflowId()

		state, err := r.queryState(ctx, wfID)
		if err != nil {
			r.logger.Debug("skipping order: query failed", "workflowId", wfID, "err", err)
			continue
		}

		orders = append(orders, LiveOrder{
			WorkflowID: wfID,
			BuildID:    pinnedBuildID(exec),
			State:      state,
			ElapsedSec: elapsedSec(exec.GetStartTime().AsTime()),
		})
	}
	return orders, nil
}

func (r *SDKReader) queryState(ctx context.Context, wfID string) (pizza.OrderState, error) {
	val, err := r.c.QueryWorkflow(ctx, wfID, "", pizza.GetStateQuery)
	if err != nil {
		return pizza.OrderState{}, err
	}
	var state pizza.OrderState
	if err := val.Get(&state); err != nil {
		return pizza.OrderState{}, fmt.Errorf("decode getState: %w", err)
	}
	return state, nil
}

// pinnedBuildID returns the Worker Deployment Version Build ID this run is pinned
// to, preferring an explicit override; "" when the run is not (yet) versioned.
func pinnedBuildID(exec *workflow.WorkflowExecutionInfo) string {
	vi := exec.GetVersioningInfo()
	if vi == nil {
		return ""
	}
	if o := vi.GetVersioningOverride(); o != nil {
		if p := o.GetPinned(); p != nil && p.GetVersion() != nil {
			return p.GetVersion().GetBuildId()
		}
	}
	if dv := vi.GetDeploymentVersion(); dv != nil {
		return dv.GetBuildId()
	}
	return ""
}

func elapsedSec(start time.Time) int {
	if start.IsZero() {
		return 0
	}
	d := time.Since(start)
	if d < 0 {
		return 0
	}
	return int(d.Seconds())
}
