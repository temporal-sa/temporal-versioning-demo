package dashboard

import "testing"

// drain returns the single buffered frame waiting on ch, failing if none is ready.
func drain(t *testing.T, ch <-chan DashboardState) DashboardState {
	t.Helper()
	select {
	case s := <-ch:
		return s
	default:
		t.Fatal("expected a buffered frame, got none")
		return DashboardState{}
	}
}

func TestHubRecovering(t *testing.T) {
	h := NewHub()

	// A plain publish carries no recover flag.
	h.Publish(DashboardState{Versions: []VersionCard{{Version: "v1"}}})
	if h.Latest().Recovering {
		t.Fatal("Latest().Recovering = true after plain Publish, want false")
	}

	// Subscribe before flipping the flag, so we can observe a re-published frame.
	frames, unsubscribe := h.Subscribe()
	defer unsubscribe()
	drain(t, frames) // pre-loaded frame from Subscribe (Recovering false)

	// Flipping the flag re-publishes the latest state stamped with it.
	h.SetRecovering(true)
	if !h.Latest().Recovering {
		t.Fatal("Latest().Recovering = false after SetRecovering(true), want true")
	}
	if got := drain(t, frames); !got.Recovering {
		t.Fatal("re-published frame after SetRecovering(true) has Recovering = false, want true")
	}

	// A fresh subscriber pre-loads the stamped state.
	fresh, unsubscribeFresh := h.Subscribe()
	defer unsubscribeFresh()
	if got := drain(t, fresh); !got.Recovering {
		t.Fatal("fresh Subscribe pre-load has Recovering = false, want true")
	}

	// A subsequent poll frame is stamped with the flag for existing subscribers.
	h.Publish(DashboardState{Versions: []VersionCard{{Version: "v2"}}})
	if got := drain(t, frames); !got.Recovering {
		t.Fatal("poll frame during recovery has Recovering = false, want true")
	}

	// Clearing the flag returns to the normal state.
	h.SetRecovering(false)
	if h.Latest().Recovering {
		t.Fatal("Latest().Recovering = true after SetRecovering(false), want false")
	}
}
