---
name: "Pizza demo: versioning approach and Temporal wiring"
description: "Single image + PIZZA_VERSION env selects shape; connection/deployment names; rollback & recover semantics; timing"
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
  queue `pizza`. The controller auto-generates the Temporal deployment name as
  `default.pizza` (`<k8s-ns>.<WorkerDeployment-name>`); backend env
  `PIZZA_DEPLOYMENT_NAME` must match that.
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
- **Friendly version labels** in the deployment panel come from CreateTime
  ordering of the Describe version summaries (oldest = v1).
- **Timing:** 15 s dwell between steps (full order ~60-90 s); order generator
  starts one order every 6 s; UI ramp increments 10/25/50/100 %.
- **v3 regression:** the Drone delivery activity always fails; the workflow runs
  a bounded manual retry loop (`maxDroneRetries`) so the order stalls red and
  surfaces a retry count via the query, without unbounded history.

**Why:** These are non-obvious choices (not derivable from a fresh read of the
code) made during planning to satisfy the spec's narrative and timing.

**How to apply:** Follow them when implementing/extending the worker, backend
actions, and manifests. See [[worker-controller-crd-rename]]. Full step-by-step
plan: `docs/superpowers/plans/2026-06-03-pizza-worker-versioning-demo.md`
(gitignored, local only).
