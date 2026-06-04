package dashboard

import (
	"context"
	"log/slog"
	"time"
)

// TemporalReader is the read surface the poller needs. The production impl wraps
// the Temporal SDK client; tests can fake it.
type TemporalReader interface {
	// DeploymentSnapshot returns routing config + version summaries for the deployment.
	DeploymentSnapshot(ctx context.Context) (Routing, []VersionSummary, error)
	// OpenOrders lists open PizzaOrder workflows, queries getState on each, and
	// returns them with their pinned Build ID and elapsed time.
	OpenOrders(ctx context.Context) ([]LiveOrder, error)
}

// Poller periodically snapshots Temporal and publishes a DashboardState.
type Poller struct {
	reader   TemporalReader
	interval time.Duration
	logger   *slog.Logger
	publish  func(DashboardState)
}

// NewPoller builds a Poller that publishes a fresh DashboardState every interval.
func NewPoller(r TemporalReader, interval time.Duration, logger *slog.Logger,
	publish func(DashboardState),
) *Poller {
	return &Poller{reader: r, interval: interval, logger: logger, publish: publish}
}

// Run polls until ctx is cancelled. On each tick it builds and publishes state;
// transient errors are logged and the previous published state remains in effect.
func (p *Poller) Run(ctx context.Context) {
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		p.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	routing, summaries, err := p.reader.DeploymentSnapshot(ctx)
	if err != nil {
		p.logger.Warn("deployment snapshot failed", "err", err)
		return
	}
	orders, err := p.reader.OpenOrders(ctx)
	if err != nil {
		p.logger.Warn("open orders fetch failed", "err", err)
		return
	}
	p.publish(BuildState(routing, summaries, orders))
}
