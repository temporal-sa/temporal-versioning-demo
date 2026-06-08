package dashboard

import (
	"errors"
	"fmt"
	"sync"
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

func TestRecoverQueryFor(t *testing.T) {
	tests := []struct {
		name           string
		deploymentName string
		badBuild       string
		want           string
	}{
		{
			name:           "builds the bad-build predicate",
			deploymentName: "pizza",
			badBuild:       "v3-local",
			want: "WorkflowType = 'PizzaOrder' AND ExecutionStatus = 'Running' AND " +
				"TemporalWorkerDeploymentVersion = 'pizza:v3-local'",
		},
		{
			name:           "single-quote-escapes the interpolated value",
			deploymentName: "piz'za",
			badBuild:       "v3'local",
			want: "WorkflowType = 'PizzaOrder' AND ExecutionStatus = 'Running' AND " +
				"TemporalWorkerDeploymentVersion = 'piz''za:v3''local'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recoverQueryFor(tt.deploymentName, tt.badBuild); got != tt.want {
				t.Errorf("recoverQueryFor(%q, %q) = %q, want %q", tt.deploymentName, tt.badBuild, got, tt.want)
			}
		})
	}
}

func TestSelectTargetBuild(t *testing.T) {
	tests := []struct {
		name       string
		ramping    string
		current    string
		candidates []targetCandidate
		want       string
		wantErr    error
	}{
		{
			name:    "ramping build wins and ignores candidates",
			ramping: "ramp-build",
			current: "v1-local",
			candidates: []targetCandidate{
				{buildID: "v3-local", label: "v3"},
			},
			want: "ramp-build",
		},
		{
			// Regression: a CreateTime heuristic would pick whatever sorts first
			// here (v2-local), but the highest friendly version number is v3.
			name:    "no ramp picks highest version number not slice order",
			current: "v1-local",
			candidates: []targetCandidate{
				{buildID: "v2-local", label: "v2"},
				{buildID: "v1-local", label: "v1"},
				{buildID: "v3-local", label: "v3"},
			},
			want: "v3-local",
		},
		{
			name:    "all candidates equal current yields ErrNoTargetVersion",
			current: "v1-local",
			candidates: []targetCandidate{
				{buildID: "v1-local", label: "v1"},
			},
			wantErr: ErrNoTargetVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := selectTargetBuild(tt.ramping, tt.current, tt.candidates)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("selectTargetBuild = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestActionsLabelCacheConcurrent hammers the labelCache helpers from many
// goroutines so `go test -race` flags any regression that drops the mutex guard.
func TestActionsLabelCacheConcurrent(t *testing.T) {
	a := &Actions{labelCache: map[string]string{}}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("b%d", n%5)
			a.cacheLabel(id, "v1")
			_, _ = a.cachedLabel(id)
		}(i)
	}
	wg.Wait()
}
