# Project Memory

> When a new decision **contradicts** an existing memory note, do NOT silently
> override it. Instead: surface the conflict, quote the existing memory, explain
> how the new decision differs, and ask for explicit confirmation before
> updating. **Do NOT take any action** — no tool calls, no file writes — until
> confirmed.

- [Worker versioning model](references/worker-versioning-model.md) — single image + PIZZA_VERSION = a Pinned shape; workers publish a `pizzaVersion` metadata label; manual UI-driven routing; bootstrap promotes only the v1-labelled version; target `temporal-k8s` Kind cluster.
- [Temporal API constraints](references/temporal-api-constraints.md) — no `.` in deployment name; no `ORDER BY` on dev visibility; visibility omits `VersioningInfo` (use the `TemporalWorkerDeploymentVersion` SA, never key aggregates on a list build ID); ramp/promote need AllowNoPollers + IgnoreMissingTaskQueues.
- [Demo timing & v3 drone regression](references/demo-timing.md) — StepDwell 15 s, DeliveredDwell 7 s (outlasts UI collapse); order every ~6 s; ramp 10/25/50/100 %; v3 drone always fails via native unlimited retry (stays red/Running, never Failed).
- [Workflow waits are activity-side](references/workflow-waits-activity-side.md) — no workflow.Sleep/timers; dwell simulated in activities (injectable, zero in tests); Done set before the final activity.
- [Frontend conventions & gotchas](references/frontend-conventions.md) — no build (Tailwind Play CDN), zero app JS (HTMX only), no per-version failure count; traps: `@media` can't `@apply`, `#orders` must morph not replace, close with empty 200 not 204.
- [Commit message convention](references/commit-convention.md) — subjects start with an imperative action verb; no scope/type prefix.
- [Podman IPv6 healthcheck](references/podman-ipv6-healthcheck.md) — use `127.0.0.1:7233` not `localhost:7233` everywhere on the host; `localhost` hits IPv6 `::1` first and stalls under Podman.
