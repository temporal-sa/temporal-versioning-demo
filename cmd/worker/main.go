// Command worker runs the versioned Temporal worker for the Pizza Tracker demo.
//
// The worker is registered with Temporal as part of a Worker Deployment so the
// Temporal Worker Controller can manage its version and route traffic to it.
// The deployment name and build ID are injected by the controller through
// environment variables.
//
// TODO: connect to Temporal, register the pizza workflow and activities with
// the Pinned versioning behavior, and run a versioned worker. See
// docs/superpowers/specs for the full design.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := struct {
		temporalAddress string
		deploymentName  string
		buildID         string
		taskQueue       string
	}{
		temporalAddress: getenv("TEMPORAL_ADDRESS", "localhost:7233"),
		deploymentName:  os.Getenv("TEMPORAL_DEPLOYMENT_NAME"),
		buildID:         os.Getenv("TEMPORAL_WORKER_BUILD_ID"),
		taskQueue:       getenv("PIZZA_TASK_QUEUE", "pizza"),
	}

	logger.Info("starting pizza worker",
		"temporalAddress", cfg.temporalAddress,
		"deploymentName", cfg.deploymentName,
		"buildID", cfg.buildID,
		"taskQueue", cfg.taskQueue,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO: run the versioned Temporal worker until the context is cancelled.
	<-ctx.Done()
	logger.Info("pizza worker stopped")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
