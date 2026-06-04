---
name: "make dev hot-reloads both backend and worker"
description: "Backend and the v1 worker both run under Air; worker-v2/worker-v3 still use go run (no reload)"
type: feedback
---

# make dev hot-reloads both backend and worker

Under `make dev`, the **backend** (`.air.toml`, rebuilds `./tmp/backend`)
and the **v1 worker** (`.air.worker.toml`, rebuilds `./tmp/worker/worker`)
both run under Air with hot reload. Edits to the pizza **workflow and
activities (`internal/pizza/`)** — which run in the worker, not the
backend — are now picked up automatically; no manual worker restart
needed.

Key details of the worker Air setup (`.air.worker.toml`):

- Own `tmp_dir = "tmp/worker"` so its `build-errors.log` does not collide
  with the backend Air (which uses `tmp_dir = "tmp"`); both run together.
- No live-reload proxy (proxy is backend/dashboard-only).
- `include_dir = ["cmd/worker", "internal/pizza"]` so backend/dashboard
  edits don't trigger a needless worker rebuild + restart.
- `kill_delay = "2s"` + `send_interrupt` to let `worker.Stop` drain
  in-flight tasks before Air force-kills on each reload.
- Required env (`TEMPORAL_DEPLOYMENT_NAME`, `TEMPORAL_WORKER_BUILD_ID`,
  `PIZZA_VERSION`) is set on the `worker` Make target and inherited by
  the Air-spawned binary.

**Still on `go run` (no reload):** the demo `worker-v2` / `worker-v3`
targets. `dev-stop` reaps both Air workers and the go-run workers.

Related: [[workflow-waits-activity-side]].
