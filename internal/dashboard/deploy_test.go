package dashboard

import (
	"bytes"
	"strings"
	"testing"
)

func versionsState(currentLabel string, labels ...string) DashboardState {
	cards := make([]VersionCard, 0, len(labels))
	for _, l := range labels {
		card := VersionCard{Version: l, Status: StatusInactive}
		if l == currentLabel {
			card.Status = StatusCurrent
			card.TrafficPct = 100
		}
		cards = append(cards, card)
	}
	return DashboardState{Versions: cards}
}

// rampingState returns a state where one version is Ramping at rampPct and
// another is Current taking the remaining traffic, mirroring BuildState.
func rampingState(rampingLabel string, rampPct int, currentLabel string, labels ...string) DashboardState {
	cards := make([]VersionCard, 0, len(labels))
	for _, l := range labels {
		card := VersionCard{Version: l, Status: StatusInactive}
		switch l {
		case rampingLabel:
			card.Status = StatusRamping
			card.TrafficPct = rampPct
		case currentLabel:
			card.Status = StatusCurrent
			card.TrafficPct = 100 - rampPct
		}
		cards = append(cards, card)
	}
	return DashboardState{Versions: cards}
}

func TestRampViewFor(t *testing.T) {
	tests := []struct {
		name        string
		state       DashboardState
		selected    string
		wantPct     int
		wantStopIdx int
	}{
		{"ramping keeps its 25% at idx 1", rampingState("v3", 25, "v2", "v1", "v2", "v3"), "v3", 25, 1},
		{"ramping at 50% at idx 2", rampingState("v3", 50, "v2", "v1", "v2", "v3"), "v3", 50, 2},
		{"current -> 100% at idx 3", versionsState("v2", "v1", "v2", "v3"), "v2", 100, 3},
		{"inactive -> 10% at idx 0", versionsState("v2", "v1", "v2", "v3"), "v1", 10, 0},
		{"unknown selection -> 10% at idx 0", versionsState("v2", "v1", "v2", "v3"), "v9", 10, 0},
		{"empty selection -> 10% at idx 0", versionsState("v2", "v1", "v2", "v3"), "", 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rampViewFor(tt.state, tt.selected)
			if got.Pct != tt.wantPct || got.StopIdx != tt.wantStopIdx {
				t.Errorf("rampViewFor(%q) = %+v, want Pct=%d StopIdx=%d",
					tt.selected, got, tt.wantPct, tt.wantStopIdx)
			}
		})
	}
}

func TestDefaultDeploySelection(t *testing.T) {
	tests := []struct {
		name  string
		state DashboardState
		want  string
	}{
		{"prefers ramping over current", rampingState("v3", 25, "v2", "v1", "v2", "v3"), "v3"},
		{"falls back to current without a ramp", versionsState("v2", "v1", "v2", "v3"), "v2"},
		{"falls back to first card when neither", versionsState("", "v1", "v2", "v3"), "v1"},
		{"empty when no versions", DashboardState{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultDeploySelection(tt.state); got != tt.want {
				t.Errorf("defaultDeploySelection() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildDeployModalView(t *testing.T) {
	t.Run("ramping pre-checked with its in-progress ramp", func(t *testing.T) {
		state := rampingState("v3", 25, "v2", "v1", "v2", "v3")
		view := buildDeployModalView(state)

		want := map[string]bool{"v1": false, "v2": false, "v3": true}
		if len(view.Versions) != len(want) {
			t.Fatalf("got %d options, want %d", len(view.Versions), len(want))
		}
		for _, opt := range view.Versions {
			if opt.Checked != want[opt.Version] {
				t.Errorf("option %q Checked = %v, want %v", opt.Version, opt.Checked, want[opt.Version])
			}
		}
		if view.Ramp.Pct != 25 {
			t.Errorf("Ramp.Pct = %d, want 25", view.Ramp.Pct)
		}
	})

	t.Run("current pre-checked with 100% ramp without a ramp in progress", func(t *testing.T) {
		state := versionsState("v2", "v1", "v2", "v3")
		view := buildDeployModalView(state)

		want := map[string]bool{"v1": false, "v2": true, "v3": false}
		for _, opt := range view.Versions {
			if opt.Checked != want[opt.Version] {
				t.Errorf("option %q Checked = %v, want %v", opt.Version, opt.Checked, want[opt.Version])
			}
		}
		if view.Ramp.Pct != 100 {
			t.Errorf("Ramp.Pct = %d, want 100", view.Ramp.Pct)
		}
	})

	t.Run("falls back to first version when none current", func(t *testing.T) {
		state := versionsState("", "v1", "v2", "v3")
		view := buildDeployModalView(state)

		if !view.Versions[0].Checked {
			t.Errorf("first option should be checked when no Current version")
		}
		for _, opt := range view.Versions[1:] {
			if opt.Checked {
				t.Errorf("option %q should not be checked", opt.Version)
			}
		}
		// The fallback selection is Inactive, so the ramp defaults to 10%.
		if view.Ramp.Pct != 10 {
			t.Errorf("Ramp.Pct = %d, want 10", view.Ramp.Pct)
		}
	})

	t.Run("no versions yields empty options at 10%", func(t *testing.T) {
		view := buildDeployModalView(DashboardState{})
		if len(view.Versions) != 0 {
			t.Errorf("got %d options, want 0", len(view.Versions))
		}
		if view.Ramp.Pct != 10 {
			t.Errorf("Ramp.Pct = %d, want 10", view.Ramp.Pct)
		}
	})
}

func TestRendererDeployModal(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}
	state := versionsState("v2", "v1", "v2", "v3")
	view := buildDeployModalView(state)

	var buf bytes.Buffer
	if err := r.DeployModal(&buf, view); err != nil {
		t.Fatalf("DeployModal: %v", err)
	}
	out := buf.String()

	want := []string{
		`id="deploy-ramp"`,
		`value="v2" checked`,                   // Current version pre-checked
		`hx-get="/api/deploy-ramp?version=v1"`, // radios drive the ramp re-render
		`hx-get="/api/deploy-ramp?version=v3"`,
		`100%`,      // Current => 100% ramp
		`value="3"`, // slider at stop index 3
		`onclick="applyDeploy()"`,
	}
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("deploy modal output missing %q\n--- output ---\n%s", w, out)
		}
	}
	if strings.Contains(out, `value="v1" checked`) || strings.Contains(out, `value="v3" checked`) {
		t.Errorf("only the Current version should be checked\n--- output ---\n%s", out)
	}
}

func TestRendererDeployRamp(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	tests := []struct {
		name    string
		view    rampView
		wantPct string
		wantIdx string
	}{
		{"current at 100%", rampViewFor(versionsState("v2", "v1", "v2"), "v2"), "100%", `value="3"`},
		{"other at 10%", rampViewFor(versionsState("v2", "v1", "v2"), "v1"), "10%", `value="0"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := r.DeployRamp(&buf, tt.view); err != nil {
				t.Fatalf("DeployRamp: %v", err)
			}
			out := buf.String()
			// The ramp root must keep its id so the radio's outerHTML swap stays anchored.
			for _, w := range []string{`id="deploy-ramp"`, tt.wantPct, tt.wantIdx} {
				if !strings.Contains(out, w) {
					t.Errorf("deploy ramp output missing %q\n--- output ---\n%s", w, out)
				}
			}
		})
	}
}
