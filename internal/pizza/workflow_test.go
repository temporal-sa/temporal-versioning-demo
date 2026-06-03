package pizza_test

import (
	"testing"
	"time"

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

	// Query the state shortly before the drone step would be reached, then again
	// while it is failing.
	env.RegisterDelayedCallback(func() {
		val, err := env.QueryWorkflow(pizza.GetStateQuery)
		if err != nil {
			t.Errorf("query failed: %v", err)
			return
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
	}, 80*time.Second)

	env.ExecuteWorkflow(pizza.PizzaOrderV3, pizza.OrderInput{OrderID: 3, Pizza: "Diavola"})
	// v3 stalls on the bounded retry loop; the test env runs out the loop and the
	// workflow eventually finishes with an error after maxDroneRetries.
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("v3 should ultimately error on drone delivery")
	}
}
