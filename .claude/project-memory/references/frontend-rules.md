---
name: "Frontend ground rules: Tailwind Play CDN + HTMX, server-rendered"
description: "No build step; all styling in index.html; server-rendered HTML over SSE; templates carry classes only; prefer native Tailwind variants"
type: feedback
---

# Frontend ground rules: Tailwind Play CDN + HTMX, server-rendered

Durable rules for the Pizza Tracker SPA:

- **Tailwind via Play CDN, no build step** (`@tailwindcss/browser@4`). The Play
  CDN only processes `<style type="text/tailwindcss">` blocks in the HTML, so
  **all** styling (`@theme` tokens, `@apply` components, `@keyframes`) lives in
  the single `<style>` block in `frontend/index.html` — the one place styles are
  centralized. Do **not** add a Node/PostCSS build or an external `.css` file; a
  real need for one means switching to a Tailwind build step — surface that
  trade-off first.
- **Go templates carry class names only, no CSS rules.** The only allowed inline
  `style=` is a CSS custom property carrying a dynamic value (e.g. the version
  bar's `--bar-w`).
- **SSE pushes server-rendered HTML, not JSON.** Named SSE events are swapped by
  HTMX into `sse-swap` targets; actions are `hx-post`; errors/toasts use
  response-targets into `#toast`.
- **Zero application JavaScript — all interactivity is HTMX.** There is no
  `<script>` block and no `onclick`/`oninput`/`hx-on` in the UI (only the head's
  CDN library `<script src>` includes). Modals use a single `#modal-host`
  (swap a fragment in to open, empty 200 to close); Escape-to-close is
  `hx-trigger="keyup[key=='Escape'] from:body"`. Do **not** reintroduce JS —
  wire new interactions with htmx attributes + small fragment endpoints. The
  one accepted consequence: inputs that need live value feedback (e.g. the ramp
  slider's % label) update on `change` (server re-render), not continuously.
  See [[deploy-modal-htmx]].
- **Media-based variants do NOT compile via `@apply` in the Play CDN.**
  `@apply max-[760px]:…` (and other `@media`-based variants) emit *nothing* —
  verified in-browser: only an explicit raw `@media` block produces a media
  query. So **responsive and `prefers-reduced-motion` rules must stay as raw
  `@media` blocks**. Non-media variants and selectors (`:hover`, `:has`,
  `:not`), arbitrary-property utilities (`[scrollbar-gutter:stable]`,
  `[outline:…]`), `animate-*` tokens and arbitrary transitions all compile
  fine via `@apply`. (Corrects the earlier "prefer native variants over
  `@media`" guidance.)
- CDN scripts use floating major tags with no SRI — an accepted trade-off for
  the no-build demo (including the offline-cluster risk).

**Why:** Deliberate choices to reduce bespoke frontend code, keep zero build,
and ship a self-contained binary.

**How to apply:** Add styles to the `index.html` `<style>` block (not templates,
not an external file); render new UI in the Go templates. Animation specifics
live in [[frontend-orders-animation]].
