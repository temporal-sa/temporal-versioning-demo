package pizza

import (
	"context"
	"errors"
	"time"

	"go.temporal.io/sdk/activity"
)

// Activities groups the pizza step activities. All three durations are injected by
// the worker and are zero in unit tests so the activities run instantly:
//   - Dwell is the per-step work time the work activities pretend to take.
//   - DroneAttempt is the time the drone spends on each failing delivery attempt.
//   - DeliverDwell is the (shorter) work time of the final Deliver step.
//
// Simulating these waits inside the activities (instead of the workflow sleeping)
// keeps timers out of the workflow history.
type Activities struct {
	// Dwell is how long each work activity pretends to take.
	Dwell time.Duration
	// DroneAttempt is how long each failing drone delivery attempt pretends to take.
	DroneAttempt time.Duration
	// DeliverDwell is the (shorter) dwell of the final Deliver step.
	DeliverDwell time.Duration
}

// dwell simulates the time a real step takes. It is context-aware so a cancelled
// activity returns promptly. A zero or negative duration (e.g. in unit tests) is a no-op.
func (a Activities) dwell(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// Receive acknowledges a new order.
func (a Activities) Receive(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("order received", "orderId", in.OrderID, "pizza", in.Pizza)
	return a.dwell(ctx, a.Dwell)
}

// Cook prepares the pizza.
func (a Activities) Cook(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("cooking", "orderId", in.OrderID)
	return a.dwell(ctx, a.Dwell)
}

// QualityCheck inspects the pizza before delivery (added in v2).
func (a Activities) QualityCheck(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("quality check", "orderId", in.OrderID)
	return a.dwell(ctx, a.Dwell)
}

// OutForDelivery dispatches the pizza to a courier.
func (a Activities) OutForDelivery(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("out for delivery", "orderId", in.OrderID)
	return a.dwell(ctx, a.Dwell)
}

// Deliver marks the order as delivered.
func (a Activities) Deliver(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("delivered", "orderId", in.OrderID)
	return a.dwell(ctx, a.DeliverDwell)
}

// DroneDelivery is the buggy v3 step: each attempt spends DroneAttempt simulating the
// flight, then always fails, so v3 orders stall and go red until they are recovered onto v2.
func (a Activities) DroneDelivery(ctx context.Context, in OrderInput) error {
	if err := a.dwell(ctx, a.DroneAttempt); err != nil {
		return err
	}
	activity.GetLogger(ctx).Warn("drone delivery failed", "orderId", in.OrderID)
	return errors.New("drone delivery failed: navigation system offline")
}
