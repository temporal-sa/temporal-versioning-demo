package dashboard

import "sync"

// Hub fans out DashboardState frames to SSE subscribers and remembers the latest.
type Hub struct {
	mu      sync.Mutex
	subs    map[chan DashboardState]struct{}
	latest  DashboardState
	hasData bool
}

// NewHub builds an empty Hub.
func NewHub() *Hub { return &Hub{subs: map[chan DashboardState]struct{}{}} }

// Publish stores the latest state and delivers it to all subscribers, dropping
// frames for subscribers that are not keeping up.
func (h *Hub) Publish(s DashboardState) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.latest, h.hasData = s, true
	for ch := range h.subs {
		select {
		case ch <- s:
		default:
		}
	}
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
