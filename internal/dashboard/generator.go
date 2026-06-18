package dashboard

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
	"go.temporal.io/sdk/client"
)

const generatorMaxRun = 10 * time.Minute

// GeneratorControl owns the play/pause state for starting new orders.
type GeneratorControl struct {
	mu       sync.Mutex
	now      func() time.Time
	running  bool
	deadline time.Time
}

// NewGeneratorControl builds a paused generator control. Each Play call runs for
// at most ten minutes before Status/Running auto-pauses it.
func NewGeneratorControl() *GeneratorControl {
	return newGeneratorControl(time.Now)
}

func newGeneratorControl(now func() time.Time) *GeneratorControl {
	return &GeneratorControl{now: now}
}

// Play allows new orders to start for up to generatorMaxRun.
func (c *GeneratorControl) Play() GeneratorStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	c.running = true
	c.deadline = now.Add(generatorMaxRun)
	return c.statusLocked(now)
}

// Pause prevents new orders from starting.
func (c *GeneratorControl) Pause() GeneratorStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	c.deadline = time.Time{}
	return c.statusLocked(c.now())
}

// Status returns the current generator state, auto-pausing expired play sessions.
func (c *GeneratorControl) Status() GeneratorStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statusLocked(c.now())
}

// Running reports whether new orders may start right now.
func (c *GeneratorControl) Running() bool {
	return c.Status().Running
}

func (c *GeneratorControl) statusLocked(now time.Time) GeneratorStatus {
	if c.running && !c.deadline.IsZero() && !now.Before(c.deadline) {
		c.running = false
		c.deadline = time.Time{}
	}
	return GeneratorStatus{Running: c.running}
}

// Generator starts pizza orders while its control is playing.
type Generator struct {
	c         client.Client
	taskQueue string
	interval  time.Duration
	control   *GeneratorControl
	logger    *slog.Logger
	startID   int
}

// NewGenerator builds a Generator that starts one order per interval, numbering
// orders from startID+1.
func NewGenerator(c client.Client, taskQueue string, interval time.Duration, startID int,
	control *GeneratorControl, logger *slog.Logger,
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

// Run starts one order per interval while the control is playing, until ctx is
// cancelled. Orders are started WITHOUT an explicit version so Temporal routes
// them by the deployment's Current/Ramping config; Pinned behaviour then locks
// each to its start version.
func (g *Generator) Run(ctx context.Context) {
	id := g.startID
	t := time.NewTicker(g.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if g.control != nil && !g.control.Running() {
				continue
			}
			id++
			in := pizza.OrderInput{OrderID: id, Pizza: pizza.Menu[id%len(pizza.Menu)]}
			opts := client.StartWorkflowOptions{
				ID:        fmt.Sprintf("order-%d", id),
				TaskQueue: g.taskQueue,
			}
			if _, err := g.c.ExecuteWorkflow(ctx, opts, pizza.WorkflowTypeName, in); err != nil {
				g.logger.Warn("failed to start order", "orderId", id, "err", err)
			}
		}
	}
}
