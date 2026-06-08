---
name: "Temporal API constraints in this demo"
description: "Non-obvious Temporal limits that break the demo if reintroduced: no dot in deployment name, no ORDER BY on dev visibility, visibility omits VersioningInfo, AllowNoPollers on ramp/promote"
type: feedback
---

# Temporal API constraints in this demo

Non-obvious Temporal limits hit while building this demo — each surfaced as a
runtime failure, not a compile error, so they are easy to reintroduce by
"tidying" a query or a name:

- **No `.` in the worker deployment name.** Temporal reserves `.` as the
  `<deployment>.<build_id>` separator; a dotted name breaks worker polling and the
  deployment snapshot. The name is `pizza`.
- **The dev-server (SQLite) visibility store rejects `ORDER BY`.** Omit it from
  `ListWorkflowExecutions` queries and sort in Go; when capping results, keep the
  **newest tail**.
- **`ListWorkflowExecutions` does NOT populate `VersioningInfo`** (only
  `DescribeWorkflowExecution` does), so `pinnedBuildID` is always empty on a list
  result. To filter open orders by pinned build, push the predicate into the query
  via the **`TemporalWorkerDeploymentVersion`** search attribute (value
  `<deployment>:<buildID>`, e.g. `pizza:v3-local`, colon-separated — differs from
  the dotted form in `versioning_info.version`). For the same reason, **never key
  per-order aggregates on a list result's build ID** — count by the order's
  resolved friendly label instead.
- **Ramp/promote need `AllowNoPollers: true` + `IgnoreMissingTaskQueues: true`.**
  The operator ramps/promotes right after shipping a version, before its poller has
  registered; the default would reject the call with `FailedPrecondition`.

**How to apply:** keep the deployment name dot-free, never add `ORDER BY` to a
visibility query, never read `VersioningInfo` off a list result, and keep the
no-poller flags on ramp/promote. See [[worker-versioning-model]].
