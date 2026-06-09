package dashboard

import (
	"testing"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
)

// TestFormatElapsedGuards covers the negative-seconds guard and the m:ss
// formatting boundaries.
func TestFormatElapsedGuards(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"negative clamps to zero", -30, "0:00"},
		{"zero", 0, "0:00"},
		{"sub-minute", 45, "0:45"},
		{"minute and seconds zero-padded", 72, "1:12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatElapsed(tt.seconds); got != tt.want {
				t.Errorf("formatElapsed(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

// TestStepperFillPctGuards covers the divide-by-(n-1) guard for short pipelines,
// a normal mid-pipeline order, and the done/last-step case.
func TestStepperFillPctGuards(t *testing.T) {
	fiveSteps := []pizza.StepLabel{
		pizza.StepReceived, pizza.StepCooking, pizza.StepQualityCheck, pizza.StepOutForDelivery, pizza.StepDelivered,
	}

	tests := []struct {
		name  string
		order Order
		want  int
	}{
		{
			name:  "fewer than two steps yields zero",
			order: Order{Steps: []pizza.StepLabel{pizza.StepReceived}, CurrentStep: 0},
			want:  0,
		},
		{
			name:  "mid pipeline fills proportionally",
			order: Order{Steps: fiveSteps, CurrentStep: 2},
			want:  50,
		},
		{
			name:  "done fills the full track",
			order: Order{Steps: fiveSteps, CurrentStep: 4, Done: true},
			want:  100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stepperFillPct(tt.order); got != tt.want {
				t.Errorf("stepperFillPct(%+v) = %d, want %d", tt.order, got, tt.want)
			}
		})
	}
}
