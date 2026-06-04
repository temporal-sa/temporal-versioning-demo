# Project Memory

> When a new decision **contradicts** an existing memory note, do NOT silently
> override it. Instead: surface the conflict, quote the existing memory, explain
> how the new decision differs, and ask for explicit confirmation before
> updating. **Do NOT take any action** — no tool calls, no file writes — until
> confirmed.

- [Worker Controller CRD rename](references/crd-worker-controller-rename.md) — cluster chart 0.26.0 → use WorkerDeployment/Connection, not Temporal* kinds.
- [Architecture decisions](references/architecture-decisions.md) — single image + PIZZA_VERSION; deployment name `pizza` (no `.`); no `ORDER BY`; rollback/recover; worker-versioning rules (moved from CLAUDE.md); backend auto-promotes first version (no flag, everywhere); timing (final Deliver step has own DeliveredDwell 7s, sized for the UI visible-then-collapse delay).
- [Frontend stack: Tailwind Play CDN + HTMX](references/frontend-stack-tailwind-htmx.md) — no build; styles only in index.html; server-rendered HTML over SSE; SPA embedded; prefer native Tailwind variants over raw @media; dashboard layout intents (3-col cap, Deployment↔Ramping align, full-width stepper); #orders morphs via idiomorph (stable id="order-{ID}"); live orders use a client-side absolute-position masonry (pinned columns via data-col, .dleft scrolls w/ stable gutter); Done card stays visible ~4s (COLLAPSE_DELAY) then collapses via data-collapsing; idiomorph beforeAttributeUpdated MUST preserve style+data-* on .order (else flicker/reshuffle).
- [Workflow waits are activity-side](references/workflow-waits-activity-side.md) — no workflow.Sleep/timers; dwell simulated in activities (injectable, zero in tests); Done set before final activity.
- [make dev hot-reloads both backend and worker](references/make-dev-worker-no-hot-reload.md) — backend + v1 worker under Air (.air.worker.toml, own tmp_dir); worker-v2/v3 still go run.
