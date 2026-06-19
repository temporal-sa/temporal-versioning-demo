package dashboard

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
	"go.temporal.io/sdk/client"
)

const generatorMaxRun = 10 * time.Minute

// SessionGeneratorManager owns play/pause state for active browser sessions.
// A session only consumes memory while its generator is playing.
type SessionGeneratorManager struct {
	mu        sync.Mutex
	now       func() time.Time
	deadlines map[string]time.Time
}

// NewSessionGeneratorManager builds a generator manager with no active sessions.
// Each Play call runs that session for at most ten minutes.
func NewSessionGeneratorManager() *SessionGeneratorManager {
	return newSessionGeneratorManager(time.Now)
}

func newSessionGeneratorManager(now func() time.Time) *SessionGeneratorManager {
	return &SessionGeneratorManager{now: now, deadlines: map[string]time.Time{}}
}

// Play allows new orders to start for sessionID for up to generatorMaxRun.
func (m *SessionGeneratorManager) Play(sessionID string) GeneratorStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deadlines[sessionID] = m.now().Add(generatorMaxRun)
	return GeneratorStatus{Running: true}
}

// Pause prevents new orders from starting for sessionID and drops its play state.
func (m *SessionGeneratorManager) Pause(sessionID string) GeneratorStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.deadlines, sessionID)
	return GeneratorStatus{}
}

// Status returns the current generator state for sessionID, deleting expired
// sessions so the map only contains actively playing sessions.
func (m *SessionGeneratorManager) Status(sessionID string) GeneratorStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.statusLocked(sessionID, m.now())
}

// ActiveSessions returns every session that may start an order right now. Expired
// sessions are removed as part of the scan.
func (m *SessionGeneratorManager) ActiveSessions() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	sessions := make([]string, 0, len(m.deadlines))
	for sessionID := range m.deadlines {
		if m.statusLocked(sessionID, now).Running {
			sessions = append(sessions, sessionID)
		}
	}
	slices.Sort(sessions)
	return sessions
}

func (m *SessionGeneratorManager) statusLocked(sessionID string, now time.Time) GeneratorStatus {
	deadline, ok := m.deadlines[sessionID]
	if !ok {
		return GeneratorStatus{}
	}
	if !now.Before(deadline) {
		delete(m.deadlines, sessionID)
		return GeneratorStatus{}
	}
	return GeneratorStatus{Running: true}
}

// Generator starts pizza orders for each session whose control is playing.
type Generator struct {
	c         client.Client
	taskQueue string
	interval  time.Duration
	control   *SessionGeneratorManager
	logger    *slog.Logger
	startID   int
}

// NewGenerator builds a Generator that starts one order per interval, numbering
// orders from startID+1.
func NewGenerator(c client.Client, taskQueue string, interval time.Duration, startID int,
	control *SessionGeneratorManager, logger *slog.Logger,
) *Generator {
	return &Generator{
		c:         c,
		taskQueue: taskQueue,
		interval:  interval,
		control:   control,
		startID:   startID,
		logger:    logger,
	}
}

// Run starts one order per active session every interval, until ctx is cancelled.
// Orders are started WITHOUT an explicit version so Temporal routes them by the
// deployment's Current/Ramping config; Pinned behaviour then locks each to its
// start version.
func (g *Generator) Run(ctx context.Context) {
	id := g.startID
	t := time.NewTicker(g.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if g.control == nil {
				continue
			}
			for _, sessionID := range g.control.ActiveSessions() {
				id++
				in := pizza.OrderInput{OrderID: id, Pizza: pizza.Menu[id%len(pizza.Menu)]}
				opts := client.StartWorkflowOptions{
					ID:        workflowIDForOrder(sessionID, id),
					TaskQueue: g.taskQueue,
					SearchAttributes: map[string]interface{}{
						SessionSearchAttribute: sessionID,
					},
				}
				if _, err := g.c.ExecuteWorkflow(ctx, opts, pizza.WorkflowTypeName, in); err != nil {
					g.logger.Warn("failed to start order", "sessionId", sessionID, "orderId", id, "err", err)
				}
			}
		}
	}
}
