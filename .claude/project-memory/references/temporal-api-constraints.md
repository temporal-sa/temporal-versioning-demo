---
name: "Temporal API constraints in this demo"
description: "Non-obvious Temporal limits that break the demo if reintroduced: no dot in deployment name, no ORDER BY on dev visibility, AllowNoPollers on ramp/promote"
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
- **Ramp/promote need `AllowNoPollers: true` + `IgnoreMissingTaskQueues: true`.**
  The operator clicks ramp/promote right after shipping a new version, before
  its single poller has registered; the default `false` would reject the call
  with `FailedPrecondition` and break the demo flow.

**Why:** Each of these surfaced as a runtime failure, not a compile error, and
is easy to reintroduce by "tidying" a query or a name.

**How to apply:** Keep the deployment name dot-free, never add `ORDER BY` to a
visibility query, and keep the no-poller flags on ramp/promote. See
[[worker-versioning-model]].
