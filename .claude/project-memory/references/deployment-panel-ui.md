---
name: "Deployment panel UI and the no-failure-count rule"
description: "Deployment column layout, version-card content, and why version cards show no failure count"
type: project
---

# Deployment panel UI and the no-failure-count rule

The Deployment column (`.dright` in `frontend/index.html`) holds the version
cards followed by a single row of three buttons: **Deploy** (opens the
HTMX-rendered Deploy modal — see [[deploy-modal-htmx]]), **Rollback** (POST
`/api/rollback`), **Recover** (POST `/api/recover`, htmx). There is no KPI
band — it was removed, so there is no global "In flight" total any more.

The controls row is itself a **server-rendered SSE region** (`#controls`,
`sse-swap="controls"`, `hx-swap="morph:innerHTML"`): the server renders all
three buttons every frame (`controls` template in `dashboard.tmpl`) and
idiomorph morphs only changed attributes in place. **Rollback is `disabled`
unless a version is currently Ramping** — gated server-side via the
`hasRamping([]VersionCard)` funcMap helper (`{{if not (hasRamping .Versions)}}
disabled{{end}}`), since rollback only makes sense while a ramp is in flight.
**Recover is `disabled` unless at least one live order is failing** — gated the
same way via `hasFailing([]Order)` (`{{if not (hasFailing .Orders)}}
disabled{{end}}`); since the controls frame carries the full `DashboardState`
(Orders included), the button enables/disables live as orders enter/leave the
retrying state. (`Order.Failing` is the only "in error" signal — orders never
reach Failed; see the no-failure-count rule below.)

**Recover has a server-driven "Recovering…" busy state** (amber, spinning,
non-clickable) shown for the full duration of the `POST /api/recover` action.
It is gated by a `DashboardState.Recovering` flag, NOT a client-side htmx
indicator: because `#controls` is idiomorph-morphed on every ~1s poll frame, an
`htmx-request` class would be stripped mid-request — and the POST response only
returns after the action finishes, too late to show progress. So the `Hub`
holds a `recovering` bool: `Publish` stamps it onto every frame (so poll frames
during the action keep it set) and `SetRecovering(bool)` flips it AND
immediately re-publishes the latest state (instant flip, no wait for the next
tick). `handleRecover` wraps the call with `SetRecovering(true)` /
`defer SetRecovering(false)`. The template shows a `.recovering` busy button
(`{{if or .Recovering (not (hasFailing .Orders))}} disabled{{end}}`); the
`.btn.recover.recovering` CSS out-specifies `.btn:disabled` to stay amber +
`animate-spin` ring instead of the muted disabled look.

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
standalone button, not a modal action). SSE regions are `orders`, `versions`,
and `controls`. See also [[worker-versioning-model]] and [[frontend-rules]].
