---
name: "Frontend stack: Tailwind Play CDN + HTMX, server-rendered"
description: "Frontend ground rules: Play CDN (no build), styles only in index.html, server-rendered HTML over SSE, SPA embedded, prefer native Tailwind variants over raw @media"
type: feedback
---

# Frontend stack: Tailwind Play CDN + HTMX, server-rendered

The Pizza Tracker SPA was migrated off bespoke vanilla CSS/JS. Follow these
rules when changing the frontend:

- **Tailwind via Play CDN, no build step** (`@tailwindcss/browser@4`). Do NOT
  introduce a Node/PostCSS or standalone-CLI build. Because the Play CDN only
  processes `<style type="text/tailwindcss">` blocks in the HTML (not external
  `.css` files), all styling — `@theme` tokens, `@apply` component classes,
  `@keyframes` — lives in the single `<style>` block in `frontend/index.html`.
  That block is the one place styles are centralized.
- **Go templates carry class names only, no CSS rules.** Rendering moved to Go
  `html/template` (`internal/dashboard/render.go` + `templates/dashboard.tmpl`).
  The only allowed inline `style=` is a CSS custom property carrying a dynamic
  data value (the version traffic bar uses `style="--bar-w:NN%"`, with the
  `width: var(--bar-w)` rule in the CSS block).
- **SSE pushes server-rendered HTML, not JSON.** `GET /events` emits named SSE
  events (`dep`, `kpis`, `orders`, `versions`, `controls`); HTMX
  (`htmx-ext-sse`) swaps each into its `sse-swap` target. Actions are `hx-post`;
  errors/toasts use `htmx-ext-response-targets` into `#toast` with a pure-CSS
  auto-dismiss. The `DashboardState` model and `BuildState` are unchanged — only
  presentation changed.
- **The live `#orders` region morphs, it does not replace (decision
  2026-06-04).** To animate new orders and step transitions smoothly, `#orders`
  uses idiomorph: CDN `idiomorph@0` `idiomorph-ext.min.js` (jsdelivr, floating
  major tag), `hx-ext="sse,response-targets,morph"` on `<body>`, and
  `hx-swap="morph:innerHTML"` on the `#orders` div. **Each order card carries a
  stable `id="order-{ID}"`** (rendered in the `"order"` template) so idiomorph's
  id-set matching keeps existing cards as persistent DOM nodes. That persistence
  is what makes the animations work: a once-only `@keyframes card-in` on `.order`
  (replays only on genuinely new nodes, not on every tick), a `transition: width`
  on `.stepper::after` driven by `--fill`, and color transitions on `.dot`/`.lbl`
  as node classes flip — all disabled under `@media (prefers-reduced-motion:
  reduce)`. **Gotcha:** a plain `innerHTML` swap (still used by the other
  sse-swap targets `kpis`/`versions`/`controls`) destroys/recreates nodes every
  frame, so CSS transitions never fire and entry animations would replay on every
  card; morphing is required for smoothness. Don't revert `#orders` to plain
  `innerHTML` or drop the per-card `id` without losing this.
- **Done cards play an exit animation, then leave (decision 2026-06-04, iterated
  with the user).** An order marked `Done` stays Running ~5 s (`DeliveredDwell`,
  see [[architecture-decisions]]) before idiomorph removes it. The `"order"`
  template adds a conditional `done` class (`{{if .Done}} done{{end}}`, beside
  `fail`); `index.html` defines `@keyframes order-leave` and
  `.order.done { overflow: hidden; animation: order-leave 1s ease forwards; }`.
  The keyframes: **grey-in + tiny settle (0→18 %) → slide out to the LEFT
  (`translateX(-110%)`) + `scale(.85)` + fade to `opacity:0` (18→60 %) → collapse
  `max-height` 110px→0 so the cards below slide up (60→100 %)**, and the 100 %
  keyframe also sets **`display:none`** to drop the card out of the grid flow.
  **Why an animation, not a removal hook:** idiomorph removes the node **instantly**
  when the server stops listing it, so the only way to animate the exit is *during*
  the Done window — the node persists across morphs (same `id`, class stays `done`),
  so a keyframe animation started when `.done` is added plays once, `forwards`-holds
  the gone state, and the now-invisible node is removed later with no visual change.
  **Gotcha A — horizontal clip:** the leftward slide needs clipping or it bleeds
  past the column / adds a horizontal scrollbar, so `.dleft` (the orders column,
  not `.olist`) has `overflow-x: clip` — chosen because `.olist`'s `px-[18px]`
  padding keeps the `pulse`/`errp` shadow rings of active cards visible, whereas
  clipping on `.olist` would crop them. **Gotcha B — residual grid gap (the bug the
  user spotted):** `.olist` is a grid with `gap-3` (12px); a card collapsed to
  `max-height:0` but **still in the DOM** keeps the grid's row-gap on both sides, so
  it left a ~12px offset between neighbors that lingered until idiomorph removed the
  node seconds later. Fix = the `display:none` at the 100 % keyframe (a discrete
  property held by `forwards`: the card stays visible for the whole 1 s exit, then
  drops out of flow at the end), so the grid closes the gap immediately. Don't
  remove that `display:none`. **Gotcha C (general):** `@keyframes card-in` (entry)
  animates `opacity`+`transform`; `.order` applies it with `animation: card-in
  0.35s ease backwards` — a retaining fill (`both`/`forwards`) would pin the 100 %
  values and, per the cascade, **animation values beat normal declarations**, so a
  static state class couldn't override them. (Moot for `.order.done` now that it has
  its own `order-leave` animation, but keep `backwards` to avoid surprising any
  future static state class.) Under `@media (prefers-reduced-motion: reduce)`
  (`.order { animation:none; transition:none }`) the exit is disabled and
  `.order.done { animation:none; transform:none; filter:grayscale(1); opacity:.55 }`
  keeps only the informational grey/dim (no slide/collapse).
- **The SPA is embedded in the backend binary** (`frontend/embed.go` →
  `//go:embed index.html`, served via `http.FileServerFS`). There is no
  `FRONTEND_DIR` env var and no `COPY frontend/` in `Dockerfile.backend`.
- CDN scripts use floating major tags (`@4`, `@2`) with no SRI — an accepted
  trade-off for the no-build demo, including the offline-cluster risk.
- **Prefer native Tailwind variants over hand-written `@media` (decision
  2026-06-04).** Tailwind v4's `@apply` supports variants, so express responsive
  rules with utility variants — named breakpoints (`sm/md/lg/xl/2xl`) or
  arbitrary `min-[1500px]:` / `max-[760px]:` — and arbitrary values like
  `grid-cols-[repeat(auto-fill,minmax(min(420px,100%),1fr))]` (no spaces inside
  the bracket). Reserve a raw `@media` block only when it spans **multiple
  selectors** (e.g. the mobile `max-width: 760px` block touching
  `.app`/`.dbody`/`.olist`/`.dright`/`.oh .nm`). `.olist` was refactored this
  way: its grid + 3-column breakpoint now live entirely in `@apply`.
- **Dashboard layout intents to preserve when retuning widths/breakpoints:**
  live orders render in a responsive grid (`.olist`) **capped at 3 columns at
  ≥1500px**; the right Deployment column (`.dright`) is intentionally aligned to
  the 3rd KPI cell ("Ramping"). **Gotcha:** a naive `2:1` flex split
  (`.dleft { flex: 2 }` vs `.dright` `flex-1`) does NOT render as 2/3 : 1/3 —
  measured in-browser it left `.dright` ~12px too wide, so the divider missed
  the Ramping cell. The fix is to **pin `.dright` to exactly one third**
  (`basis-1/3 grow-0 shrink-0`) and let `.dleft` fill the rest (`flex-1`), which
  matches the 3 equal KPI cells (verified delta ≈ 1px). The per-order stepper
  spans the **full card width** with
  the first/last dots flush to the card edges, drawn as one progress track whose
  fill length is driven server-side by `--fill` (`stepperStyle` /
  `stepperFillPct` in `render.go`). Don't silently undo these when adjusting CSS.

**Why:** These are deliberate choices made on 2026-06-04 at the user's request
(reduce bespoke/proprietary frontend code; keep zero build; centralize styles;
ship a self-contained binary). The Play-CDN-vs-external-`.css` tension is real
and was resolved in favor of keeping the CDN.

**How to apply:** Add styles to the `index.html` `<style>` block (not templates,
not an external file); render new UI in the Go templates; if a future need
demands an external `.css` file, that requires switching to a Tailwind build
step — surface that trade-off before doing it. See
[[architecture-decisions]].
