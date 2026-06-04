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
