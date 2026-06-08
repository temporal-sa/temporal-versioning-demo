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
standalone button, not a modal action). SSE regions are `orders` and `versions`
only. See also [[worker-versioning-model]] and [[frontend-rules]].
