package dashboard

import (
	"errors"
	"testing"
)

func TestDecideBootstrap(t *testing.T) {
	describeErr := errors.New("describe worker deployment: connection refused")

	tests := []struct {
		name         string
		target       string
		current      string
		err          error
		wantDecision bootstrapDecision
		wantBuildID  string
	}{
		{
			name:         "current already set skips even with a target",
			target:       "b2",
			current:      "b1",
			err:          nil,
			wantDecision: bootstrapSkip,
		},
		{
			name:         "no target version waits",
			target:       "",
			current:      "",
			err:          ErrNoTargetVersion,
			wantDecision: bootstrapWait,
		},
		{
			name:         "describe error waits",
			target:       "",
			current:      "",
			err:          describeErr,
			wantDecision: bootstrapWait,
		},
		{
			name:         "target with empty current promotes that build",
			target:       "b1",
			current:      "",
			err:          nil,
			wantDecision: bootstrapPromote,
			wantBuildID:  "b1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDecision, gotBuildID := decideBootstrap(tt.target, tt.current, tt.err)
			if gotDecision != tt.wantDecision {
				t.Errorf("decision = %d, want %d", gotDecision, tt.wantDecision)
			}
			if gotBuildID != tt.wantBuildID {
				t.Errorf("buildID = %q, want %q", gotBuildID, tt.wantBuildID)
			}
		})
	}
}

func TestBootstrapBuildIDPicksV1(t *testing.T) {
	labels := map[string]string{"bX": "v2", "bY": "v1", "bZ": "v3"}
	if got := bootstrapBuildID(labels); got != "bY" {
		t.Errorf("bootstrapBuildID = %q, want bY (the v1 build)", got)
	}
	if got := bootstrapBuildID(map[string]string{"bX": "v2"}); got != "" {
		t.Errorf("no v1 present = %q, want \"\"", got)
	}
}
