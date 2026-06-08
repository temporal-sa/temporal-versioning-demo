---
name: "Worker versioning model"
description: "Single image + PIZZA_VERSION selects a Pinned shape; workers publish a pizzaVersion metadata label; manual UI-driven routing; backend auto-promotes only the v1-labelled version at bootstrap"
type: project
---

# Worker versioning model

How Worker Versioning is wired in this demo:

- **One worker image, three shapes.** A single binary registers v1/v2/v3 of the
  shared workflow type `PizzaOrder`, all **Pinned**; the `PIZZA_VERSION` env var
  (set per pod template) selects which shape a pod runs. Shipping a version =
  bump `PIZZA_VERSION` + image tag; the Worker Controller derives a new Build ID
  from the pod-template hash.
- **Versions are labelled from metadata, not registration order.** At startup
  each worker publishes its friendly label (v1/v2/v3) as Worker Deployment
  Version metadata (`pizzaVersion` key). The backend labels versions from that
  metadata, with a CreateTime-order fallback only when metadata is absent. This
  matters because `make dev` runs v1/v2/v3 **together**, so registration order is
  racy.
- **Manual controller; routing is UI-driven with an explicit target.** Actions
  take a friendly label (resolved to a Build ID via metadata): ramp = set ramping
  version + %, promote = set current version, rollback = clear the ramping version
  (safe while Current is set), recover = a **per-workflow**
  `ResetWorkflowExecution` (rewind to workflow start) that pins the new run's
  versioning override to the current Build ID. Any-to-any transitions are
  supported (e.g. v1→v2→v1→v3).
- **Bootstrap promotes the v1-labelled version and waits for its metadata.** While
  Current is nil the backend promotes the version whose *metadata* label is `v1`
  (no CreateTime fallback in bootstrap), so starting all three together never
  promotes an arbitrary build. Once Current is set, all routing is manual/UI-driven.
- **Target cluster:** the local `temporal-k8s` Kind cluster (Temporal Server +
  Worker Controller already deployed).

**Why:** these are demo-narrative choices, not derivable from a fresh read of the
code.

**How to apply:** mind the API limits in [[temporal-api-constraints]]; timing is
in [[demo-timing]]; the routing UI rules are in [[frontend-conventions]].
