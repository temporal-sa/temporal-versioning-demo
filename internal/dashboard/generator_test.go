package dashboard

import (
	"testing"
	"time"
)

func TestSessionGeneratorManagerPlayPauseAndAutoPause(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	manager := newSessionGeneratorManager(func() time.Time { return now })
	sessionID := "4a1c8f0b-41b2-4c43-8991-75c5950bc04a"

	if manager.Status(sessionID).Running {
		t.Fatal("generator should start paused")
	}

	manager.Play(sessionID)
	if !manager.Status(sessionID).Running {
		t.Fatal("generator should run after Play")
	}

	now = now.Add(generatorMaxRun - time.Second)
	if !manager.Status(sessionID).Running {
		t.Fatal("generator should still run before the 10-minute cap")
	}

	now = now.Add(time.Second)
	if manager.Status(sessionID).Running {
		t.Fatal("generator should auto-pause at the 10-minute cap")
	}

	manager.Play(sessionID)
	manager.Pause(sessionID)
	if manager.Status(sessionID).Running {
		t.Fatal("generator should pause immediately")
	}
}

func TestSessionGeneratorManagerActiveSessionsDropsExpired(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	manager := newSessionGeneratorManager(func() time.Time { return now })
	active := "4a1c8f0b-41b2-4c43-8991-75c5950bc04a"
	expired := "fd10f8d0-70b3-47d8-93f3-c589f7675ea0"

	manager.Play(active)
	manager.Play(expired)
	now = now.Add(generatorMaxRun - time.Second)
	manager.Play(active)
	now = now.Add(time.Second)

	got := manager.ActiveSessions()
	if len(got) != 1 || got[0] != active {
		t.Fatalf("ActiveSessions() = %v, want [%s]", got, active)
	}
	if manager.Status(expired).Running {
		t.Fatal("expired session should have been removed")
	}
}
