package dashboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
			name:         "no target no current no error waits",
			target:       "",
			current:      "",
			err:          nil,
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
		{
			name:         "target present but describe errored waits",
			target:       "b1",
			current:      "",
			err:          describeErr,
			wantDecision: bootstrapWait,
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

// TestLabelResolverConcurrent hammers the resolver's cache from many goroutines
// so `go test -race` flags any regression that drops the mutex guard. Keys are
// pre-seeded so label() returns from the cache and never dials the nil client.
func TestLabelResolverConcurrent(t *testing.T) {
	r := newLabelResolver(nil, "pizza", slog.New(slog.NewTextHandler(io.Discard, nil)))
	for i := 0; i < 5; i++ {
		r.cache[fmt.Sprintf("b%d", i)] = "v1"
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = r.label(context.Background(), fmt.Sprintf("b%d", n%5))
		}(i)
	}
	wg.Wait()
	if got := r.label(context.Background(), "b0"); got != "v1" {
		t.Errorf("label(b0) = %q, want v1", got)
	}
}
