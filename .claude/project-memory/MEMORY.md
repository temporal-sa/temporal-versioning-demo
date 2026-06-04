# Project Memory

> When a new decision **contradicts** an existing memory note, do NOT silently
> override it. Instead: surface the conflict, quote the existing memory, explain
> how the new decision differs, and ask for explicit confirmation before
> updating. **Do NOT take any action** — no tool calls, no file writes — until
> confirmed.

- [Worker Controller CRD rename](references/crd-worker-controller-rename.md) — cluster chart 0.26.0 → use WorkerDeployment/Connection, not Temporal* kinds.
- [Architecture decisions](references/architecture-decisions.md) — single image + PIZZA_VERSION; deployment name `pizza` (no `.`); no `ORDER BY`; rollback/recover; worker-versioning rules (moved from CLAUDE.md); backend auto-promotes first version (no flag, everywhere); timing.
- [Frontend stack: Tailwind Play CDN + HTMX](references/frontend-stack-tailwind-htmx.md) — no build; styles only in index.html; server-rendered HTML over SSE; SPA embedded in binary.
- [Workflow waits are activity-side](references/workflow-waits-activity-side.md) — no workflow.Sleep/timers; dwell simulated in activities (injectable, zero in tests); Done set before final activity.
- [make dev hot-reloads backend only](references/make-dev-worker-no-hot-reload.md) — worker runs via `go run`, no reload; restart it to pick up internal/pizza changes.
