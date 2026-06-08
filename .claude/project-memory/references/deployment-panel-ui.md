---
name: "Deployment panel UI and the no-failure-count rule"
description: "Deployment column layout, version-card content, and why version cards show no failure count"
type: project
---

# Deployment panel UI and the no-failure-count rule

The Deployment column (`.dright` in `frontend/index.html`) holds the version
cards followed by a single row of **two** buttons: **Deploy** (opens the
HTMX-rendered Deploy modal — see [[deploy-modal-htmx]]) and **Rollback** (POST
`/api/rollback`). There is no KPI band — it was removed, so there is no global
"In flight" total any more.

The controls row is itself a **server-rendered SSE region** (`#controls`,
`sse-swap="controls"`, `hx-swap="morph:innerHTML"`): the server renders both
buttons every frame (`controls` template in `dashboard.tmpl`) and idiomorph
morphs only changed attributes in place. **Rollback is `disabled` unless a
version is currently Ramping** — gated server-side via the
`hasRamping([]VersionCard)` funcMap helper (`{{if not (hasRamping .Versions)}}
disabled{{end}}`), since rollback only makes sense while a ramp is in flight.

**Recover is per-card, not a global control.** A small amber icon button (↻,
`.recover-btn`) appears on an order card only when that order is failing,
immediately right of the order's version badge in `.oh`. It does `POST
/api/recover/{id}` (`{id}` is the full workflow ID, e.g. `order-123`), routed to
`handleRecoverOne` → `Actions.RecoverOne(ctx, workflowID)` (single
reset-with-move onto the Current build). There is **no server-side busy state**:
the button uses `hx-disabled-elt="this"` to disable itself only during its
in-flight request, and the card simply stops rendering the button on the next
poll once the order is no longer failing. (`Order.Failing` is the only "in
error" signal — orders never reach Failed; see the no-failure-count rule below.)
This replaced an earlier design (a global `/api/recover` button with a
`hasFailing` gate and a server-driven "Recovering…" busy state via a
`Hub.recovering` flag / `DashboardState.Recovering`) — all of that machinery,
plus the bulk `Actions.Recover`/`inError`/`hasFailingActivity` helpers, was
removed.

Each version card is sorted **v1 → v2 → v3** and shows only an in-progress
count: `{{.PinnedCount}} in flight`. Version cards deliberately show **no
failure / "failing" count**.

**Why:** every pizza activity retries forever (native unlimited retry — see
[[demo-timing]]), so no workflow ever reaches a terminal Failed state.
`OrderState.Failing` only means "the current step is retrying", not "failed", so
a per-version failure count would be misleading. The retrying state is still
conveyed per order by the red order card / errored stepper node, just not as a
count.

**How to apply:** do not re-introduce a failure/"failing" count on the version
cards or a global In-flight KPI. Keep the cards to badge + status chip + traffic
bar + `N in flight`. The Deploy modal keeps only Cancel + Apply (Rollback is the
standalone button, not a modal action). Recover stays per-card (no global
Recover button, no server-side busy state). SSE regions are `orders`,
`versions`, and `controls`. See also [[worker-versioning-model]] and
[[frontend-rules]].
