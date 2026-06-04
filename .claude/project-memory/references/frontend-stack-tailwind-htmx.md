---
name: "Frontend stack: Tailwind Play CDN + HTMX, server-rendered"
description: "Frontend ground rules: Play CDN (no build), styles only in index.html, server-rendered HTML over SSE, SPA embedded"
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
- **The SPA is embedded in the backend binary** (`frontend/embed.go` →
  `//go:embed index.html`, served via `http.FileServerFS`). There is no
  `FRONTEND_DIR` env var and no `COPY frontend/` in `Dockerfile.backend`.
- CDN scripts use floating major tags (`@4`, `@2`) with no SRI — an accepted
  trade-off for the no-build demo, including the offline-cluster risk.

**Why:** These are deliberate choices made on 2026-06-04 at the user's request
(reduce bespoke/proprietary frontend code; keep zero build; centralize styles;
ship a self-contained binary). The Play-CDN-vs-external-`.css` tension is real
and was resolved in favor of keeping the CDN.

**How to apply:** Add styles to the `index.html` `<style>` block (not templates,
not an external file); render new UI in the Go templates; if a future need
demands an external `.css` file, that requires switching to a Tailwind build
step — surface that trade-off before doing it. See
[[pizza-demo-architecture-decisions]].
