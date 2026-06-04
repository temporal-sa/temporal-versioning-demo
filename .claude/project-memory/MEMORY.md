# Project Memory

> When a new decision **contradicts** an existing memory note, do NOT silently
> override it. Instead: surface the conflict, quote the existing memory, explain
> how the new decision differs, and ask for explicit confirmation before
> updating. **Do NOT take any action** — no tool calls, no file writes — until
> confirmed.

- [Worker Controller CRD rename](references/crd-worker-controller-rename.md) — cluster chart 0.26.0 → use WorkerDeployment/Connection, not Temporal* kinds.
- [Pizza demo architecture decisions](references/pizza-demo-architecture-decisions.md) — single image + PIZZA_VERSION; deployment name is `pizza` (NOT `default.pizza` — no `.` allowed); no `ORDER BY` in visibility queries; rollback/recover; timing.
- [Frontend stack: Tailwind Play CDN + HTMX](references/frontend-stack-tailwind-htmx.md) — no build; styles only in index.html; server-rendered HTML over SSE; SPA embedded in binary.
