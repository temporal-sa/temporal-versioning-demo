package dashboard

import (
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
)

// VersionStatus is the lifecycle chip shown on a deployment card.
type VersionStatus string

// Deployment version lifecycle states.
const (
	StatusCurrent  VersionStatus = "CURRENT"
	StatusRamping  VersionStatus = "RAMPING"
	StatusDraining VersionStatus = "DRAINING"
	StatusInactive VersionStatus = "INACTIVE"
)

// Order is one live order card.
type Order struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"` // friendly: v1/v2/v3
	Pizza       string            `json:"pizza"`
	Steps       []pizza.StepLabel `json:"steps"`
	CurrentStep int               `json:"currentStep"`
	Failing     bool              `json:"failing"`
	Done        bool              `json:"done"`
	ElapsedSec  int               `json:"elapsedSec"`
}

// VersionCard is one deployment-panel card.
type VersionCard struct {
	Version     string        `json:"version"` // friendly: v1/v2/v3
	BuildID     string        `json:"buildId"`
	Status      VersionStatus `json:"status"`
	TrafficPct  int           `json:"trafficPct"`
	PinnedCount int           `json:"pinnedCount"`
}

// KPIs is the top strip.
type KPIs struct {
	InFlight       int    `json:"inFlight"`
	CurrentVersion string `json:"currentVersion"`
	RampingVersion string `json:"rampingVersion"`
	RampingPct     int    `json:"rampingPct"`
}

// DashboardState is the full SSE payload.
type DashboardState struct {
	KPIs     KPIs          `json:"kpis"`
	Orders   []Order       `json:"orders"`
	Versions []VersionCard `json:"versions"`
}

// VersionSummary mirrors the fields BuildState needs from a Temporal version summary.
type VersionSummary struct {
	BuildID      string
	PizzaVersion string // friendly label from version metadata; "" => fall back to CreateTime
	CreateTime   time.Time
	Draining     bool
	Drained      bool
}

// Routing mirrors the routing config BuildState needs.
type Routing struct {
	CurrentBuildID string
	RampingBuildID string
	RampingPct     int
}

// LiveOrder is one open workflow's state plus the build it is pinned to.
type LiveOrder struct {
	WorkflowID string
	BuildID    string // Worker Deployment Version Build ID this run is pinned to
	State      pizza.OrderState
	ElapsedSec int
}

// BuildState maps Temporal data to the SPA payload. Friendly version labels are
// assigned from version metadata when present; otherwise they fall back to
// CreateTime order (oldest summary = v1). Per-version override keeps mixed
// states correct while workers publish their metadata.
func BuildState(routing Routing, summaries []VersionSummary, orders []LiveOrder) DashboardState {
	sorted := slices.Clone(summaries)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].CreateTime.Before(sorted[j].CreateTime) })

	// Label by version metadata when present; otherwise fall back to CreateTime
	// order (oldest = v1). Per-version override keeps mixed states correct while
	// the three workers publish their metadata.
	label := make(map[string]string, len(sorted)) // buildID -> v1/v2/v3
	for i, s := range sorted {
		if s.PizzaVersion != "" {
			label[s.BuildID] = s.PizzaVersion
		} else {
			label[s.BuildID] = friendly(i)
		}
	}

	pinned := make(map[string]int, len(sorted)) // buildID -> open order count
	outOrders := make([]Order, 0, len(orders))
	for _, o := range orders {
		pinned[o.BuildID]++
		outOrders = append(outOrders, Order{
			ID:          o.WorkflowID,
			Version:     pickLabel(label, o.BuildID, o.State.Version),
			Pizza:       o.State.Pizza,
			Steps:       o.State.Steps,
			CurrentStep: o.State.CurrentStep,
			Failing:     o.State.Failing,
			Done:        o.State.Done,
			ElapsedSec:  o.ElapsedSec,
		})
	}

	cards := make([]VersionCard, 0, len(sorted))
	for _, s := range sorted {
		card := VersionCard{
			Version:     label[s.BuildID],
			BuildID:     s.BuildID,
			PinnedCount: pinned[s.BuildID],
			Status:      StatusInactive,
		}
		switch {
		case s.BuildID == routing.CurrentBuildID:
			card.Status = StatusCurrent
			card.TrafficPct = 100 - routing.RampingPct
		case s.BuildID == routing.RampingBuildID:
			card.Status = StatusRamping
			card.TrafficPct = routing.RampingPct
		case s.Draining || s.Drained:
			card.Status = StatusDraining
		}
		cards = append(cards, card)
	}

	kpi := KPIs{
		InFlight:       len(orders),
		CurrentVersion: label[routing.CurrentBuildID],
		RampingVersion: label[routing.RampingBuildID],
		RampingPct:     routing.RampingPct,
	}

	return DashboardState{KPIs: kpi, Orders: outOrders, Versions: cards}
}

func friendly(i int) string {
	return "v" + strconv.Itoa(i+1)
}

// pickLabel prefers the deployment-ordering label; falls back to the order's
// self-reported version when the build is not (yet) in the summaries.
func pickLabel(label map[string]string, buildID, reported string) string {
	if l, ok := label[buildID]; ok && l != "" {
		return l
	}
	return reported
}
