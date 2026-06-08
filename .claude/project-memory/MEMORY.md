# Project Memory

> When a new decision **contradicts** an existing memory note, do NOT silently
> override it. Instead: surface the conflict, quote the existing memory, explain
> how the new decision differs, and ask for explicit confirmation before
> updating. **Do NOT take any action** — no tool calls, no file writes — until
> confirmed.

- [Worker versioning model](references/worker-versioning-model.md) — single image + PIZZA_VERSION = a Pinned shape; workers publish a `pizzaVersion` metadata label; manual UI-driven routing; bootstrap promotes only the v1-labelled version; target `temporal-k8s` Kind cluster.
- [kbld digest pinning for deploy determinism](references/kbld-digest-pinning.md) — `make deploy` pipes kustomize through kbld to pin images to `sha256` digests so each Build ID maps to one image; pods keep `IfNotPresent`, never `Always`.
- [Temporal API constraints](references/temporal-api-constraints.md) — no `.` in deployment name; no `ORDER BY` on dev visibility; visibility omits `VersioningInfo` (use the `TemporalWorkerDeploymentVersion` SA, never key aggregates on a list build ID); ramp/promote need AllowNoPollers + IgnoreMissingTaskQueues.
- [Demo timing & v3 drone regression](references/demo-timing.md) — StepDwell 15 s, DeliveredDwell 7 s (outlasts UI collapse); order every ~6 s; ramp 25/50/100 % (10 % stop dropped); v3 drone always fails via native unlimited retry (stays red/Running, never Failed).
- [Workflow waits are activity-side](references/workflow-waits-activity-side.md) — no workflow.Sleep/timers; dwell simulated in activities (injectable, zero in tests); Done set before the final activity.
- [Frontend conventions & gotchas](references/frontend-conventions.md) — no build (Tailwind Play CDN), zero app JS (HTMX only), hypermedia URLs never `/api/` (`/deploy`, `/rollback`, `DELETE /modal`…), no per-version failure count; traps: `@media` can't `@apply`, `#orders`/`#versions` must morph not replace, content-keyed chip id replays the pulse, close with empty 200 not 204.
- [Commit message convention](references/commit-convention.md) — subjects start with an imperative action verb; no scope/type prefix.
- [Kubernetes namespace: pizza-tracker](references/k8s-namespace.md) — demo deploys to a dedicated `pizza-tracker` namespace owned by Kustomize (not `default`); controller names the deployment `<ns>/<name>` so backend derives `PIZZA_DEPLOYMENT_NAME=$(POD_NAMESPACE)/pizza-worker` via Downward API; the `temporalNamespace`/`TEMPORAL_NAMESPACE`/`parentRefs` namespaces are unrelated and stay as-is.
- [Distroless + runAsNonRoot needs numeric runAsUser](references/distroless-runasnonroot.md) — distroless `:nonroot` images have a non-numeric user, so the hardened pod securityContext must set `runAsUser: 65532` (and `runAsGroup`) or pods hit CreateContainerConfigError.
- [IPv6 localhost healthcheck](references/ipv6-localhost-healthcheck.md) — use `127.0.0.1:7233` not `localhost:7233` everywhere on the host; `localhost` hits IPv6 `::1` first while the dev server is IPv4-only, so it stalls.
