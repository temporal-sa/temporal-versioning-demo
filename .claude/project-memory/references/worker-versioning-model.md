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
  the `getState` query so the UI colours orders without decoding Build IDs, and
  at startup it **publishes that label as Worker Deployment Version metadata**
  (`pizzaVersion` key, via `UpdateVersionMetadata`, best-effort retry). The
  backend labels versions from that metadata (cached `DescribeVersion`), with a
  CreateTime-order fallback only when metadata is absent.
- **Local dev runs all three versions at once.** `make dev` starts
  Temporal + backend + workers v1/v2/v3 together (each `PIZZA_VERSION=vN` +
  `TEMPORAL_WORKER_BUILD_ID=vN-local`); the old `dev-all` target was merged into
  `dev` and removed. This is why labels come from metadata, not CreateTime:
  concurrent registration makes CreateTime order racy.
- **Manual controller; routing is UI-driven with an EXPLICIT target.** Actions
  take a friendly version label (resolved to a Build ID via the metadata labels):
  ramp = `SetRampingVersion{BuildID, Pct}`; promote = `SetCurrentVersion{BuildID}`;
  rollback = `SetRampingVersion` empty (safe while Current is non-nil); recover =
  per-order `ResetWorkflowExecution` (reset-with-move) pinning a
  `VersioningOverride` to the current Build ID. The UI is a **Deploy modal**
  (target radios v1/v2/v3 + a 4-stop slider 10/25/50/100; reaching 100 % maps to
  promote; ⟲ Rollback clears the ramp). The KPI band shows Current + the active
  Ramping target/% (read-only, reserved column so the layout never reflows).
  Any-to-any transitions are supported (e.g. v1→v2→v1→v3).
- **Backend bootstraps the v1-labelled version, and WAITS for its metadata.**
  While Current is nil, `EnsureCurrentVersion` promotes the version whose
  *metadata* label is `v1` (`metadataLabels`, **no CreateTime fallback in
  bootstrap**), so starting v1/v2/v3 together via `dev` never promotes an
  arbitrary (oldest-registered) build. Once Current is set it leaves the
  ramp/promote/rollback flow fully manual/UI-driven. (`Ramp`/`Promote` use the
  fallback-bearing `versionIndex`; only bootstrap is metadata-strict.)
- **Target cluster:** the local `temporal-k8s` Kind cluster (Temporal Server +
  Worker Controller already deployed).

**Why:** These are demo-narrative choices, not derivable from a fresh read of
the code.

**How to apply:** Follow them when extending the worker, backend actions, and
manifests. Mind the API limits in [[temporal-api-constraints]]; timing lives in
[[demo-timing]].
