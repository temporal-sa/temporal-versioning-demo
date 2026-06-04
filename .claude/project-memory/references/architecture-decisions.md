---
name: "Pizza demo: versioning approach and Temporal wiring"
description: "Single image + PIZZA_VERSION env selects shape; connection/deployment names; rollback & recover semantics; backend auto-promotes first version; timing"
type: project
---

# Pizza demo: versioning approach and Temporal wiring

Key implementation decisions for the Pizza Worker Versioning demo (resolve spec
open items; verified against Go SDK v1.44.1 / api v1.62.13):

- **One worker image, three shapes.** A single binary compiles v1/v2/v3 pizza
  workflows; the `PIZZA_VERSION` env var (set per pod-template in
  `k8s/workerdeployment.yaml`) selects which shape registers under the shared
  workflow type `PizzaOrder`, all **Pinned**. Shipping a version = change
  `PIZZA_VERSION` (and image tag); the controller derives a new Build ID from
  the pod-template hash. The worker reports its friendly version (v1/v2/v3)
  through the `getState` query so the UI colours orders without decoding Build
  IDs.
- **Connection.** Temporal frontend =
  `temporal-frontend.temporal.svc.cluster.local:7233`, namespace `default`, task
  queue `pizza`. The Temporal worker deployment name is **`pizza`** (the
  `WorkerDeployment` CRD `metadata.name`); backend env `PIZZA_DEPLOYMENT_NAME`
  and worker env `TEMPORAL_DEPLOYMENT_NAME` must be `pizza`. **Correction
  (2026-06-04):** an earlier version of this note claimed the controller
  auto-generates `default.pizza` (`<k8s-ns>.<WorkerDeployment-name>`) — that is
  wrong. Current Temporal **rejects `.` in worker deployment names** (`.` is the
  reserved separator in the canonical version string `<deployment_name>.<build_id>`,
  per the Go SDK `WorkerDeploymentInfo.Name` docs). Using `default.pizza` broke
  both the worker (`Failed to poll for task ... worker deployment name cannot
  contain '.'`) and the backend (`deployment snapshot failed ... reserved
  separator '.'`). The hardcoded `default.pizza` was replaced by `pizza` in
  `Makefile`, `compose.yaml`, `cmd/backend/main.go`, `k8s/backend.yaml`,
  `README.md`, and `.env.local.example`.
- **Visibility queries must not use `ORDER BY`.** The Temporal dev server's
  standard (SQLite) visibility store rejects `ORDER BY` in
  `ListWorkflowExecutions` (`invalid query: operation is not supported: 'ORDER BY'
  clause`). The open-orders and recover queries dropped their `ORDER BY StartTime`
  clauses and sort in Go instead: `openOrdersQuery` newest-first
  (`temporal_reader.go`), `recoverQuery` oldest-first after paging all matches
  (`actions.go`, for deterministic truncation).
- **Routing actions (backend → Temporal):** ramp =
  `SetRampingVersion{BuildID,Percentage}`; promote = `SetCurrentVersion{BuildID}`;
  **rollback = `SetRampingVersion{BuildID:"", Percentage:0}`** (safe because
  Current is non-nil); **recover = per-order reset-with-move**
  (`ResetWorkflowExecution` to the first WorkflowTaskCompleted with a
  `PostResetOperation_UpdateWorkflowOptions` carrying a pinned `VersioningOverride`
  to the current/v2 Build ID).
- **Ramp/Promote use `AllowNoPollers: true` + `IgnoreMissingTaskQueues: true`**
  on purpose: in the demo the operator clicks ramp/promote right after shipping
  a new version, before its (single-replica) poller has registered, and the
  default `false` would reject the call with `FailedPrecondition` and break the
  demo. Recover pages through all open orders up to a 200 cap (oldest-first).
- **Worker versioning rules (canonical home — moved out of `CLAUDE.md`
  2026-06-04):** workflows are **Pinned**; the controller runs in **Manual**
  strategy; shipping new code = K8s image change (`PIZZA_VERSION` + image tag);
  routing (ramp / promote / rollback) of versions *after the first* is
  UI-driven via the Temporal API. Target cluster: the local `temporal-k8s` Kind
  cluster (Temporal Server + Worker Controller already deployed).
- **Backend auto-promotes the FIRST version (decision 2026-06-04, overrides the
  prior "manual first promote" rule).** On startup the backend runs
  `Actions.EnsureCurrentVersion`: if the deployment has **no Current version**,
  it promotes the newest registered version (v1 on first boot) via
  `SetCurrentVersion{AllowNoPollers, IgnoreMissingTaskQueues}`, polling at
  `PIZZA_POLL_INTERVAL` until a version registers, bounded by a ~2 min timeout
  (then it gives up with a warning; manual promote still works). It applies
  **everywhere (local + K8s), with no env flag**, and fires **only when Current
  is nil** — so it runs once at bootstrap and leaves the v2/v3
  ship → ramp → promote → rollback flow fully manual/UI-driven. **This
  supersedes** the earlier behaviour where "the first version starts Inactive
  with no Current version until manually promoted" (old `README` K8s section /
  `CLAUDE.md` convention); confirmed by the user on 2026-06-04. Rationale:
  orders should flow immediately after deploy without a manual first promote —
  the instructive part of the demo is the v2/v3 rollout, not bootstrapping v1.
- **Friendly version labels** in the deployment panel come from CreateTime
  ordering of the Describe version summaries (oldest = v1).
- **Timing:** every work step takes ~15 s of **activity** time (`StepDwell`,
  including the final `Deliver`/"Done" step), so a full order is ~60-90 s; order
  generator starts one order every 6 s; UI ramp increments 10/25/50/100 %. The
  dwell is simulated **inside the activities (no `workflow.Sleep`/timers)** — see
  [[workflow-waits-activity-side]].
- **v3 regression:** the Drone delivery activity always fails; the workflow runs
  a bounded manual retry loop (`maxDroneRetries`) so the order stalls red and
  surfaces a retry count via the query, without unbounded history. Each failing
  attempt takes ~5 s of **activity** time (`droneAttempt`); the loop no longer
  uses `workflow.Sleep` for backoff (see [[workflow-waits-activity-side]]).

**Why:** These are non-obvious choices (not derivable from a fresh read of the
code) made during planning to satisfy the spec's narrative and timing.

**How to apply:** Follow them when implementing/extending the worker, backend
actions, and manifests. See [[worker-controller-crd-rename]]. Full step-by-step
plan: `docs/superpowers/plans/2026-06-03-pizza-worker-versioning-demo.md`
(gitignored, local only).
