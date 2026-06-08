---
name: "Deploy modal is server-rendered and HTMX-driven"
description: "The Deploy modal is fetched/re-rendered server-side via HTMX; the server owns version pre-selection and ramp defaults"
type: project
---

# Deploy modal is server-rendered and HTMX-driven

The Deploy modal in `frontend/index.html` is **rendered server-side and driven
by HTMX**, with **zero application JavaScript** (see [[frontend-rules]]). Both
modals (Deploy, Rollback) use a single empty **`#modal-host`** container: a
modal is *open* when the server has swapped a `.modal-scrim` fragment into it,
*closed* when the host is emptied. Flow:

- The **Deploy** button issues `GET /api/deploy-modal` → swaps the full scrim
  fragment into `#modal-host` (`hx-swap="innerHTML"`). Its mere presence shows
  the modal — no `hidden` toggling, no JS reveal step.
- **Close** is pure HTMX: Cancel buttons and the scrim's
  `hx-trigger="keyup[key=='Escape'] from:body"` both `GET /api/close` (returns
  an **empty 200**) into `#modal-host`, clearing it. (Browser-verified: an empty
  200 body swapped as innerHTML clears the host; the Escape keyboard trigger
  fires.)
- Each version radio issues `GET /api/deploy-ramp` (no query string — htmx
  auto-includes the checked radio's `name="version"` value) to re-render **only**
  the `#deploy-ramp` section (`hx-target="#deploy-ramp"`, `hx-swap="outerHTML"`,
  default `change` trigger; fragment root is `<div id="deploy-ramp">`).
- The slider (`name="stop"`, value = stop index 0–3) re-renders the ramp section
  on `change` (i.e. on release — the % label is **server-rendered**, not updated
  live; this was an explicit choice to keep zero JS), sending `stop` plus the
  checked `version` via `hx-include="input[name='version']:checked"`. The ramp
  `GET` honours an explicit `stop` param (slider) and otherwise derives the % from
  the version's status (radio change).
- **Apply** is a `<form hx-post="/api/deploy">` (single endpoint that decides
  promote-vs-ramp **server-side**: stop index 3 / 100% → promote, else ramp).
  Success returns an **empty 200** → `#modal-host` cleared → modal closes; errors
  route to `#toast` via `hx-target-error` (response-targets) so the modal stays
  open. **Rollback** confirm posts `/api/rollback` the same way (empty 200 closes
  the modal). NOTE: success must be 200-empty, **not 204** — htmx does not swap on
  204, which would leave the modal open.

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

There is **no `<script>` block and no `onclick`/`oninput`/`hx-on`** in the UI
anymore (only the head's CDN library `<script src>` tags remain). The earlier
client-side helpers (`detectCurrentVersion` / `syncRampToSelection` /
`onDeployVersionChange`, then `openDeploy`/`closeDeploy`/`onRampSlide`/
`applyDeploy`/`confirmRollback` + the Escape keydown listener) are all gone.

**Why:** the project prefers server-rendered HTML and zero bespoke JS (see
[[frontend-rules]]); the server authoritatively knows version/ramp state, so
selection, ramp defaults, and even promote-vs-ramp branching belong there.

**How to apply:** keep all modal markup in the `deploy-modal` / `deploy-ramp` /
`rollback-modal` / `controls` named templates in
`internal/dashboard/templates/dashboard.tmpl`; never move it back into static
`index.html`, and do not reintroduce JS — wire interactions with htmx attributes
and small fragment endpoints. Server pieces: `Hub.Latest()` (`sse.go`); the
`rampStops` stops + `rampDefaultPct`/`rampFullPct` constants + `rampView` /
`deployVersionOption` / `deployModalView` view models + `stopIndex` /
`rampViewFor(state, selected)` / `rampViewForStop(stop)` /
`defaultDeploySelection(state)` / `buildDeployModalView(state)` helpers +
`Renderer.DeployModal` / `Renderer.DeployRamp` / `Renderer.RollbackModal`
(`render.go`); and the routes (`server.go`): `GET /api/deploy-modal`,
`GET /api/rollback-modal`, `GET /api/deploy-ramp`, `GET /api/close` (empty 200),
`POST /api/deploy` (unified promote/ramp), `POST /api/rollback`,
`POST /api/recover`. The old `POST /api/ramp` & `POST /api/promote` routes,
their `handleRamp`/`handlePromote` handlers, and the generic `handleAction`
were removed (superseded by `/api/deploy` and the dedicated `handleRollback`);
the `actions.Ramp`/`Promote`/`Rollback` methods stayed. See also
[[deployment-panel-ui]] and [[worker-versioning-model]].
