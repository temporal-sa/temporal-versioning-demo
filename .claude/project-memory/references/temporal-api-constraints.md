---
name: "Temporal API constraints in this demo"
description: "Non-obvious Temporal limits that break the demo if reintroduced: no dot in deployment name, no ORDER BY on dev visibility, visibility omits VersioningInfo, AllowNoPollers on ramp/promote"
type: feedback
---

# Temporal API constraints in this demo

Non-obvious Temporal limits hit while building this demo. Don't reintroduce the
breakage:

- **No `.` in the worker deployment name.** Temporal reserves `.` as the
  separator in `<deployment>.<build_id>`; a dotted name breaks both worker
  polling and the backend deployment snapshot. The deployment name is `pizza`.
- **The dev-server visibility store rejects `ORDER BY`.** Standard (SQLite)
  visibility fails `ListWorkflowExecutions` queries that contain `ORDER BY`.
  Omit it and sort in Go; when capping results, keep the **newest tail** so
  recent orders are never dropped.
- **`ListWorkflowExecutions` (visibility) does NOT populate `VersioningInfo`.**
  Only `DescribeWorkflowExecution` fills it. Verified on the live dev server: a
  running order's `list` JSON has no `versioningInfo` block, so `pinnedBuildID`
  on a list result is always `""`. To filter open orders by pinned build, push
  the predicate into the query via the built-in **`TemporalWorkerDeploymentVersion`**
  search attribute (value `<deployment>:<buildID>`, e.g. `pizza:v3-local`,
  colon-separated — note this differs from the dotted form in `versioning_info.version`).
  This was the root cause of the "Recover" button always reporting "Recovered 0"
  (originally fixed in `Recover`/`recoverQueryFor`, `internal/dashboard/actions.go`).
  **Update (2026-06-08):** Recover is now a **per-card / per-workflow** action
  (`Actions.RecoverOne`, `POST /api/recover/{id}`); `recoverQueryFor` and the
  bulk in-error filter (`inError`/`hasFailingActivity`) were removed. Recover no
  longer issues a visibility query at all — it resets the one workflow the
  operator clicked. See [[worker-versioning-model]] and [[deployment-panel-ui]].
  **Same gotcha also broke the per-version "N in flight" counts** (now fixed): the
  reader sets `LiveOrder.BuildID = pinnedBuildID(exec)` from a list result
  (`temporal_reader.go`), so it is also always `""`; `BuildState` used to key
  `pinned[o.BuildID]` and so collapsed every order under one empty key, showing
  "0 in flight" everywhere. `BuildState` now counts by the order's resolved
  friendly label (the `pickLabel` value, which falls back to the workflow's
  self-reported `State.Version`) instead of by build ID — never key per-order
  aggregates on `LiveOrder.BuildID`.
- **Ramp/promote need `AllowNoPollers: true` + `IgnoreMissingTaskQueues: true`.**
  The operator clicks ramp/promote right after shipping a new version, before
  its single poller has registered; the default `false` would reject the call
  with `FailedPrecondition` and break the demo flow.

**Why:** Each of these surfaced as a runtime failure, not a compile error, and
is easy to reintroduce by "tidying" a query or a name.

**How to apply:** Keep the deployment name dot-free, never add `ORDER BY` to a
visibility query, never filter open orders by reading `VersioningInfo` off a
`ListWorkflowExecutions` result (use the `TemporalWorkerDeploymentVersion` search
attribute, or `Describe` each run), and keep the no-poller flags on ramp/promote.
See [[worker-versioning-model]].
