package dashboard

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/alexandreroman/temporal-versioning-demo/internal/pizza"
	"go.temporal.io/sdk/client"
)

// Generator continuously starts pizza orders so there are always in-flight workflows.
type Generator struct {
	c         client.Client
	taskQueue string
	interval  time.Duration
	logger    *slog.Logger
	startID   int
}

// NewGenerator builds a Generator that starts one order per interval, numbering
// orders from startID+1.
func NewGenerator(c client.Client, taskQueue string, interval time.Duration, startID int,
	logger *slog.Logger,
) *Generator {
	return &Generator{c: c, taskQueue: taskQueue, interval: interval, startID: startID, logger: logger}
}

// Run starts one order per interval until ctx is cancelled. Orders are started
// WITHOUT an explicit version so Temporal routes them by the deployment's
// Current/Ramping config; Pinned behaviour then locks each to its start version.
func (g *Generator) Run(ctx context.Context) {
	id := g.startID
	t := time.NewTicker(g.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
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
