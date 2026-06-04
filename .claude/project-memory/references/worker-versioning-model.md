---
name: "Worker versioning model"
description: "Single image + PIZZA_VERSION selects a Pinned shape; Manual controller, UI-driven routing, backend auto-promotes only the first version"
type: project
---

# Worker versioning model

How Worker Versioning is wired in this demo:

- **One worker image, three shapes.** A single binary registers v1/v2/v3 of the
  shared workflow type `PizzaOrder`, all **Pinned**; the `PIZZA_VERSION` env var
  (set per pod template) selects which shape a pod runs. Shipping a version =
  bump `PIZZA_VERSION` + image tag; the Worker Controller derives a new Build ID
  from the pod-template hash. The worker reports a friendly label (v1/v2/v3) via
  the `getState` query so the UI colours orders without decoding Build IDs.
- **Manual controller; routing after the first version is UI-driven** via the
  Temporal API: ramp = `SetRampingVersion{BuildID, Pct}`; promote =
  `SetCurrentVersion{BuildID}`; rollback = `SetRampingVersion` empty (safe while
  Current is non-nil); recover = per-order `ResetWorkflowExecution`
  (reset-with-move) carrying a pinned `VersioningOverride` to the current Build
  ID.
- **Backend auto-promotes the first version only.** On startup
  `EnsureCurrentVersion` promotes the newest registered version **iff Current is
  nil** (v1 on first boot), so orders flow immediately after deploy; the v2/v3
  ship → ramp → promote → rollback flow stays fully manual/UI-driven.
- **Target cluster:** the local `temporal-k8s` Kind cluster (Temporal Server +
  Worker Controller already deployed).

**Why:** These are demo-narrative choices, not derivable from a fresh read of
the code.

**How to apply:** Follow them when extending the worker, backend actions, and
manifests. Mind the API limits in [[temporal-api-constraints]]; timing lives in
[[demo-timing]].
