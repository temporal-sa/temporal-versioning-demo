---
name: "Deploy modal is server-rendered and HTMX-driven"
description: "The Deploy modal is fetched/re-rendered server-side via HTMX; the server owns version pre-selection and ramp defaults"
type: project
---

# Deploy modal is server-rendered and HTMX-driven

The Deploy modal in `frontend/index.html` is **rendered server-side and driven
by HTMX**, not static markup. Flow:

- The **Deploy** button issues `GET /api/deploy-modal`, swapping the fragment
  into `#deploy-modal-content` (`hx-swap="innerHTML"`); the backdrop scrim
  (`#deploy-modal`) is revealed via
  `hx-on::after-request="if (event.detail.successful) openDeploy()"`. (Use
  `after-request`, NOT `after-swap`: `htmx:afterSwap` fires on the swap target,
  not the requesting button, so a button-level `after-swap` handler never runs;
  `htmx:afterRequest` fires on the button, and the `successful` guard avoids
  revealing an empty modal on an error response.)
- Each version radio issues `GET /api/deploy-ramp?version=vX` to re-render
  **only** the `#deploy-ramp` section (radio uses `hx-target="#deploy-ramp"`,
  `hx-swap="outerHTML"`, default `change` trigger — the fragment's root is
  `<div id="deploy-ramp">` so the id survives the swap).

The **server owns version selection and ramp defaults**, and the ramp value is
**driven by the selected card's `Status`** (so initial render and radio-change
re-render stay consistent):

- selected version **Ramping** → its in-progress traffic % (`TrafficPct`,
  one of 10/25/50/100);
- selected version **Current** → **100%**;
- otherwise (Inactive/Draining/unknown/empty) → **10%** (start of a new ramp).

**Default pre-selection** when the modal opens (`defaultDeploySelection`):
the **Ramping** version if a ramp is in progress, else the **Current** version,
else the first card. So with a ramp in progress the modal opens with the
ramping radio checked and the slider at the ramping %. This **supersedes** the
earlier client-side DOM-scanning approach (the removed `detectCurrentVersion` /
`syncRampToSelection` / `onDeployVersionChange` JS) and the earlier
server logic that always pre-checked Current at 100%.

Client JS is now minimal: `onRampSlide` (live `%` label while dragging),
`openDeploy`/`closeDeploy` (toggle the scrim), `applyDeploy` (reads the checked
radio + slider stop; POSTs `/api/promote` at 100%, else `/api/ramp`), plus the
rollback and Escape handlers. Apply/rollback still POST via `fetch`, not htmx.

**Why:** the project prefers server-rendered HTML (see [[frontend-rules]]); the
server authoritatively knows the Current version, so pre-selection and ramp
defaults belong there instead of scraping the DOM on the client.

**How to apply:** keep the modal markup in the `deploy-modal` / `deploy-ramp`
named templates in `internal/dashboard/templates/dashboard.tmpl`; do not move
it back into static `index.html`. Server pieces backing this: `Hub.Latest()`
(`sse.go`), the `rampStops` slider stops + `rampDefaultPct` (10) / `rampFullPct`
(100) constants + `rampView` / `deployVersionOption` / `deployModalView` view
models + `stopIndex` / `rampViewFor(state, selected)` /
`defaultDeploySelection(state)` / `buildDeployModalView(state)` helpers +
`Renderer.DeployModal` / `Renderer.DeployRamp` (`render.go`), and the
`GET /api/deploy-modal` / `GET /api/deploy-ramp` routes (`server.go`). The
`currentVersion(state)` helper was removed once the status-driven ramp made it
dead code. The ramp `GET` does not validate the `version` param — an
unknown/empty value just yields the 10% default (harmless for a read-only
fragment). See also
[[deployment-panel-ui]] and [[worker-versioning-model]].
