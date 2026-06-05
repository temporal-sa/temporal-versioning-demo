package pizza

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// StepDwell is the per-step work time simulated inside the activities (injected
// into Activities.Dwell), sized so a full order lasts ~60-90s.
const StepDwell = 15 * time.Second

// DeliveredDwell is the (shorter) dwell of the final Deliver step. The order is
// marked Done right before Deliver runs, so this is how long the completed
// (all-green) order stays on the dashboard before its workflow closes and the
// card leaves the board. Sized so the frontend's delayed collapse finishes
// before the workflow completes and the card's node is removed.
const DeliveredDwell = 7 * time.Second

// droneAttempt is how long each (failing) drone delivery attempt takes; it paces the
// v3 retry cadence from the activity side so no workflow timer is needed.
const droneAttempt = 5 * time.Second

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
		StartToCloseTimeout: StepDwell + 15*time.Second,
		// Activities now lean on Temporal's native durable retry with no attempt limit
		// (MaximumAttempts: 0 == retry forever). A permanently-broken step — the v3 drone —
		// therefore keeps retrying indefinitely, so the order stays red/Running and never
		// fails/completes. With InitialInterval (1s) and BackoffCoefficient (2.0) left at
		// their defaults, the per-retry interval starts at 1s and doubles (1s→2s→4s→...),
		// but MaximumInterval caps it at droneAttempt so it never backs off to the SDK
		// default maximum (~100s) — keeping the retry cadence lively for the demo.
		RetryPolicy: &temporal.RetryPolicy{MaximumAttempts: 0, MaximumInterval: droneAttempt},
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
			// The v3 drone is deterministically broken, so mark the order failing as it
			// enters this step. With unlimited native retry this .Get blocks until the
			// activity succeeds (it never does) or the workflow is cancelled, so the order
			// stalls red/Running forever — that is intended.
			state.Failing = true
			if err := workflow.ExecuteActivity(ctx, a.DroneDelivery, in).Get(ctx, nil); err != nil {
				return err
			}
		case StepDelivered:
			// The dashboard lists only Running workflows, so an order vanishes the moment
			// its workflow completes. Mark it done before the final activity runs so the
			// completed (all-green) stepper is visible for the (shorter) DeliveredDwell of
			// the Deliver activity, then the workflow closes and the order leaves the board.
			state.Done = true
			if err := workflow.ExecuteActivity(ctx, a.Deliver, in).Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	return nil
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
	w.RegisterActivity(&Activities{Dwell: StepDwell, DroneAttempt: droneAttempt, DeliverDwell: DeliveredDwell})
}
