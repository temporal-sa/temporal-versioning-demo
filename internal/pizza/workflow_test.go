package pizza_test

import (
	"testing"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
	"go.temporal.io/sdk/testsuite"
)

func TestV1CompletesFourSteps(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(&pizza.Activities{})

	env.ExecuteWorkflow(pizza.PizzaOrderV1, pizza.OrderInput{OrderID: 1, Pizza: "Margherita"})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestV2HasQualityCheckStep(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(&pizza.Activities{})

	env.ExecuteWorkflow(pizza.PizzaOrderV2, pizza.OrderInput{OrderID: 2, Pizza: "Pepperoni"})

	if !env.IsWorkflowCompleted() || env.GetWorkflowError() != nil {
		t.Fatalf("v2 should complete cleanly: completed=%v err=%v", env.IsWorkflowCompleted(), env.GetWorkflowError())
	}
	steps := pizza.StepsFor(pizza.V2)
	if len(steps) != 5 || steps[2] != pizza.StepQualityCheck {
		t.Fatalf("v2 must have a Quality check as the 3rd step, got %v", steps)
	}
}

func TestV3StallsOnDrone(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestWorkflowEnvironment()
	env.RegisterActivity(&pizza.Activities{})

	env.ExecuteWorkflow(pizza.PizzaOrderV3, pizza.OrderInput{OrderID: 3, Pizza: "Diavola"})

	// v3 stalls on the always-failing drone: the bounded retry loop runs out and the
	// workflow ultimately finishes with an error. No workflow timers are involved —
	// each attempt is paced by the DroneDelivery activity itself.
	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("v3 should ultimately error on drone delivery")
	}

	// The final state surfaces the stall: stuck on the drone step, marked failing,
	// with the retry count recorded.
	val, err := env.QueryWorkflow(pizza.GetStateQuery)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	var st pizza.OrderState
	_ = val.Get(&st)
	if st.Version != "v3" {
		t.Errorf("expected version v3, got %q", st.Version)
	}
	if !st.Failing || st.RetryCount < 1 {
		t.Errorf("expected drone step to be failing with retries, got %+v", st)
	}
	if st.Steps[st.CurrentStep] != pizza.StepDroneDelivery {
		t.Errorf("expected current step Drone, got %v", st.Steps[st.CurrentStep])
	}
}
