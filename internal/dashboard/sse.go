package dashboard

import "sync"

// Hub fans out DashboardState frames to SSE subscribers and remembers the latest.
type Hub struct {
	mu         sync.Mutex
	subs       map[chan DashboardState]struct{}
	latest     DashboardState
	hasData    bool
	recovering bool // a recover action is in progress; stamped onto published frames
}

// NewHub builds an empty Hub.
func NewHub() *Hub { return &Hub{subs: map[chan DashboardState]struct{}{}} }

// Publish stores the latest state and delivers it to all subscribers, dropping
// frames for subscribers that are not keeping up. It stamps the current
// recovering flag so periodic poll frames carry it for the action's duration.
func (h *Hub) Publish(s DashboardState) {
	h.mu.Lock()
	defer h.mu.Unlock()
	s.Recovering = h.recovering
	h.store(s)
}

// store records the latest state and fans it out to subscribers. It assumes the
// caller holds h.mu.
func (h *Hub) store(s DashboardState) {
	h.latest, h.hasData = s, true
	for ch := range h.subs {
		select {
		case ch <- s:
		default:
		}
	}
}

// SetRecovering updates the in-progress flag and immediately re-publishes the
// latest known state stamped with it, so the controls region flips into/out of
// the "Recovering…" busy state without waiting for the next poll frame.
func (h *Hub) SetRecovering(recovering bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recovering = recovering
	if h.hasData {
		s := h.latest
		s.Recovering = recovering
		h.store(s)
	}
}

// Latest returns the most recently published state (zero value if none yet).
func (h *Hub) Latest() DashboardState {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.latest
}

// Subscribe returns a channel pre-loaded with the latest state (if any) and an
// unsubscribe func.
func (h *Hub) Subscribe() (<-chan DashboardState, func()) {
	ch := make(chan DashboardState, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	if h.hasData {
		ch <- h.latest
	}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		close(ch)
		h.mu.Unlock()
	}
}
