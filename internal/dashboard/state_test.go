package dashboard_test

import (
	"testing"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/dashboard"
	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
)

func TestBuildStateLabelsByCreateTimeAndCountsPinned(t *testing.T) {
	base := time.Unix(1_000_000, 0)
	summaries := []dashboard.VersionSummary{
		{BuildID: "b2", CreateTime: base.Add(2 * time.Minute)},
		{BuildID: "b1", CreateTime: base}, // oldest -> v1
		{BuildID: "b3", CreateTime: base.Add(4 * time.Minute), Draining: true},
	}
	routing := dashboard.Routing{CurrentBuildID: "b2", RampingBuildID: "b3", RampingPct: 10}
	orders := []dashboard.LiveOrder{
		{WorkflowID: "order-1", BuildID: "b2", State: pizza.OrderState{
			Version: "v2", Pizza: "Pepperoni",
			Steps: pizza.StepsFor(pizza.V2), CurrentStep: 1,
		}},
		{WorkflowID: "order-2", BuildID: "b3", State: pizza.OrderState{
			Version: "v3", Pizza: "Diavola",
			Steps: pizza.StepsFor(pizza.V3), CurrentStep: 3, Failing: true,
		}},
		{WorkflowID: "order-3", BuildID: "b2", State: pizza.OrderState{
			Version: "v2", Pizza: "Marinara",
			Steps: pizza.StepsFor(pizza.V2), CurrentStep: 0,
		}},
	}

	st := dashboard.BuildState(routing, summaries, orders)

	if st.KPIs.InFlight != 3 {
		t.Errorf("inFlight = %d, want 3", st.KPIs.InFlight)
	}
	if st.KPIs.CurrentVersion != "v2" || st.KPIs.RampingVersion != "v3" || st.KPIs.RampingPct != 10 {
		t.Errorf("KPIs = %+v", st.KPIs)
	}
	byVer := map[string]dashboard.VersionCard{}
	for _, c := range st.Versions {
		byVer[c.Version] = c
	}
	if byVer["v1"].Status != dashboard.StatusInactive {
		t.Errorf("v1 status = %s, want INACTIVE", byVer["v1"].Status)
	}
	if byVer["v2"].Status != dashboard.StatusCurrent || byVer["v2"].TrafficPct != 90 || byVer["v2"].PinnedCount != 2 {
		t.Errorf("v2 card = %+v", byVer["v2"])
	}
	if byVer["v3"].Status != dashboard.StatusRamping || byVer["v3"].TrafficPct != 10 || byVer["v3"].PinnedCount != 1 {
		t.Errorf("v3 card = %+v", byVer["v3"])
	}
}

func TestBuildStateLabelsByMetadataWhenPresent(t *testing.T) {
	base := time.Unix(1_000_000, 0)
	// CreateTime order would label b_new=v1, but metadata says it is v3.
	summaries := []dashboard.VersionSummary{
		{BuildID: "b_new", CreateTime: base, PizzaVersion: "v3"},
		{BuildID: "b_old", CreateTime: base.Add(2 * time.Minute), PizzaVersion: "v1"},
	}
	routing := dashboard.Routing{CurrentBuildID: "b_old"}
	st := dashboard.BuildState(routing, summaries, nil)

	if st.KPIs.CurrentVersion != "v1" {
		t.Errorf("current label = %q, want v1 (from metadata)", st.KPIs.CurrentVersion)
	}
	byVer := map[string]dashboard.VersionCard{}
	for _, c := range st.Versions {
		byVer[c.Version] = c
	}
	if _, ok := byVer["v3"]; !ok {
		t.Errorf("expected a v3 card from metadata, got %+v", st.Versions)
	}
}
