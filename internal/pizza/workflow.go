package pizza

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// StepDwell is the pause between steps, sized so a full order lasts ~60-90s.
const StepDwell = 15 * time.Second

const maxDroneRetries = 100 // bounded so a stuck v3 order can't bloat history forever

// PizzaOrderV1 runs the 4-step baseline pipeline.
func PizzaOrderV1(ctx workflow.Context, in OrderInput) error {
	return run(ctx, in, V1)
}

// PizzaOrderV2 adds a Quality check step.
func PizzaOrderV2(ctx workflow.Context, in OrderInput) error {
	return run(ctx, in, V2)
}

// PizzaOrderV3 replaces delivery with a buggy Drone delivery step.
func PizzaOrderV3(ctx workflow.Context, in OrderInput) error {
	return run(ctx, in, V3)
}

func run(ctx workflow.Context, in OrderInput, v Version) error {
	state := &OrderState{
		Version: string(v),
		Pizza:   in.Pizza,
		Steps:   StepsFor(v),
	}
	if err := workflow.SetQueryHandler(ctx, GetStateQuery, func() (OrderState, error) {
		return *state, nil
	}); err != nil {
		return err
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		// Fail fast: the workflow owns the retry loop so it can surface RetryCount.
		RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 1},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	var a Activities

	for i, label := range state.Steps {
		state.CurrentStep = i
		state.Failing = false

		switch label {
		case StepReceived:
			if err := workflow.ExecuteActivity(ctx, a.Receive, in).Get(ctx, nil); err != nil {
				return err
			}
		case StepCooking:
			if err := workflow.ExecuteActivity(ctx, a.Cook, in).Get(ctx, nil); err != nil {
				return err
			}
		case StepQualityCheck:
			if err := workflow.ExecuteActivity(ctx, a.QualityCheck, in).Get(ctx, nil); err != nil {
				return err
			}
		case StepOutForDelivery:
			if err := workflow.ExecuteActivity(ctx, a.OutForDelivery, in).Get(ctx, nil); err != nil {
				return err
			}
		case StepDroneDelivery:
			if err := runDrone(ctx, state, a, in); err != nil {
				return err
			}
		case StepDelivered:
			if err := workflow.ExecuteActivity(ctx, a.Deliver, in).Get(ctx, nil); err != nil {
				return err
			}
		}

		// Dwell between steps (except after the terminal Delivered step).
		if label != StepDelivered {
			_ = workflow.Sleep(ctx, StepDwell)
		}
	}

	state.Done = true
	return nil
}

// runDrone is the buggy v3 step: it retries the failing drone activity, surfacing
// the attempt count and failing flag through the query, until it gives up.
func runDrone(ctx workflow.Context, state *OrderState, a Activities, in OrderInput) error {
	var lastErr error
	for attempt := 1; attempt <= maxDroneRetries; attempt++ {
		err := workflow.ExecuteActivity(ctx, a.DroneDelivery, in).Get(ctx, nil)
		if err == nil {
			state.Failing = false
			return nil
		}
		lastErr = err
		state.Failing = true
		state.RetryCount = attempt
		_ = workflow.Sleep(ctx, 5*time.Second)
	}
	return lastErr
}

// Register registers the workflow implementation for the given version (under the
// shared type name) plus all activities on the worker.
func Register(w worker.Worker, v Version) {
	var wf any
	switch v {
	case V1:
		wf = PizzaOrderV1
	case V2:
		wf = PizzaOrderV2
	case V3:
		wf = PizzaOrderV3
	}
	w.RegisterWorkflowWithOptions(wf, workflow.RegisterOptions{
		Name:               WorkflowTypeName,
		VersioningBehavior: workflow.VersioningBehaviorPinned,
	})
	w.RegisterActivity(&Activities{})
}
