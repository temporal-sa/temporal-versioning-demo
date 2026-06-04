---
name: "make dev hot-reloads the backend only, not the worker"
description: "Worker runs via go run with no hot reload; restart it to pick up internal/pizza (workflow/activity) changes"
type: feedback
---

# make dev hot-reloads the backend only, not the worker

Under `make dev`, only the **backend** is hot-reloaded (Air rebuilds
`./tmp/backend` on `.go` changes, per `.air.toml`). The **worker** runs
via `go run ./cmd/worker` with **no hot reload**.

The pizza **workflow and activities (`internal/pizza/`) run in the
worker**, not the backend. So edits to workflow/activity code are **not**
picked up until the worker process is restarted (Ctrl-C `make dev` and
relaunch, or restart the `worker` / `worker-v2` / `worker-v3` target).

**Why:** This silently bit the user during this work — changes appeared
to have "no effect" because the running worker was still executing the
previously-compiled code.

**How to apply:** After changing anything under `internal/pizza` (or
`cmd/worker`), restart the worker before testing in the UI. If repeatedly
annoying, consider putting the worker under Air too so `make dev` reloads
both. Related: [[workflow-waits-activity-side]].
