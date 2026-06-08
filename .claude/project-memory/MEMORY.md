# Project Memory

> When a new decision **contradicts** an existing memory note, do NOT silently
> override it. Instead: surface the conflict, quote the existing memory, explain
> how the new decision differs, and ask for explicit confirmation before
> updating. **Do NOT take any action** — no tool calls, no file writes — until
> confirmed.

- [Worker versioning model](references/worker-versioning-model.md) — single image + PIZZA_VERSION = Pinned shape; workers publish a `pizzaVersion` metadata label; explicit-target UI routing via a Deploy modal (radios + 4-stop slider, 100%→promote); bootstrap promotes the v1-labelled version and waits for its metadata; `make dev` runs v1/v2/v3 together; target `temporal-k8s` Kind cluster.
- [Temporal API constraints](references/temporal-api-constraints.md) — no `.` in deployment name; no `ORDER BY` on dev visibility (sort in Go, keep newest tail); ramp/promote need AllowNoPollers + IgnoreMissingTaskQueues.
- [Demo timing & v3 drone regression](references/demo-timing.md) — StepDwell 15 s, DeliveredDwell 7 s (sized to outlast UI collapse); order every ~6 s; ramp 10/25/50/100 %; v3 drone always fails and retries forever via native unlimited retry (stays red/Running, never Failed; no retry counter).
- [Workflow waits are activity-side](references/workflow-waits-activity-side.md) — no workflow.Sleep/timers; dwell simulated in activities (injectable, zero in tests); Done set before final activity.
- [Frontend ground rules](references/frontend-rules.md) — Tailwind Play CDN (no build), styles only in index.html, server-rendered HTML over SSE, templates carry classes only, **zero app JS — all interactivity is HTMX**.
- [Temporal brand theme](references/temporal-brand-theme.md) — UV #444CE7 accent, Space Black surfaces, Off White text, Hanken Grotesk font; Tailwind-first (@apply + @theme tokens) with documented raw-CSS exceptions.
- [Live orders animation](references/frontend-orders-animation.md) — #orders morphs via idiomorph with stable ids; single-column normal flow (mb-3 spacing); Done visible-then-collapse is pure CSS (4s delay, no masonry/JS); Deployment stacks above Orders on mobile.
- [Commit message convention](references/commit-convention.md) — subjects start with an imperative action verb; no scope/type prefix (no `Frontend:`/`Fix:`/`feat:`).
- [Podman IPv6 healthcheck](references/podman-ipv6-healthcheck.md) — Use `127.0.0.1:7233` not `localhost:7233` everywhere (compose healthcheck + host dev defaults); `localhost` hits IPv6 `::1` first and stalls/fails under Podman.
- [Deployment panel UI](references/deployment-panel-ui.md) — Deploy/Rollback/Recover row (now an SSE region `#controls`; Rollback disabled unless a version is Ramping), no KPI band, cards sorted v1→v2→v3 showing `N in flight`; no failure count (infinite retry → never Failed).
- [Deploy modal HTMX](references/deploy-modal-htmx.md) — modals are server-rendered, zero JS, via a single #modal-host (swap-in to open, empty-200 to close, Escape via hx-trigger); Apply→POST /api/deploy (server picks promote vs ramp); ramp status-driven (Ramping→its %, Current→100%, else 10%); pre-selects Ramping, else Current, else first.
