package pizza

import (
	"context"
	"errors"

	"go.temporal.io/sdk/activity"
)

// Activities groups the pizza step activities. They are intentionally tiny; the
// per-step dwell time is created by workflow.Sleep, not by slow activities.
type Activities struct{}

// Receive acknowledges a new order.
func (Activities) Receive(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("order received", "orderId", in.OrderID, "pizza", in.Pizza)
	return nil
}

// Cook prepares the pizza.
func (Activities) Cook(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("cooking", "orderId", in.OrderID)
	return nil
}

// QualityCheck inspects the pizza before delivery (added in v2).
func (Activities) QualityCheck(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("quality check", "orderId", in.OrderID)
	return nil
}

// OutForDelivery dispatches the pizza to a courier.
func (Activities) OutForDelivery(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("out for delivery", "orderId", in.OrderID)
	return nil
}

// Deliver marks the order as delivered.
func (Activities) Deliver(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Info("delivered", "orderId", in.OrderID)
	return nil
}

// DroneDelivery is the buggy v3 step: it always fails, so v3 orders stall and go
// red until they are recovered onto v2.
func (Activities) DroneDelivery(ctx context.Context, in OrderInput) error {
	activity.GetLogger(ctx).Warn("drone delivery failed", "orderId", in.OrderID)
	return errors.New("drone delivery failed: navigation system offline")
}
