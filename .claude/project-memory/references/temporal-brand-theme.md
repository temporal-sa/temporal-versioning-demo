---
name: "Temporal brand theme + Tailwind-first styling"
description: "Brand palette/font tokens and the rule that all styling is @apply + @theme tokens, with the documented raw-CSS exceptions"
type: project
---

# Temporal brand theme + Tailwind-first styling

The Pizza Tracker UI is themed to Temporal's brand (source:
[temporal.io/brand](https://temporal.io/brand)) and the stylesheet is
Tailwind-first.

**Brand tokens (defined in the `@theme` block of `frontend/index.html`):**

- `--color-uv: #444ce7` — Temporal **UV**, the primary accent (also v1).
- Neutral dark surface scale derived from **Space Black**: `--color-base`
  `#0e0e11`, `--color-surface` `#15151a`, `--color-raised` `#1b1b21`,
  `--color-edge` `#20202a`, `--color-line` `#2c2c36`, `--color-divider`
  `#23232c`.
- `--color-ink: #f8fafc` — **Off White** text.
- Version hues stay distinct: **v1 = UV**, **v2 = green** `#16a34a`,
  **v3 = amber** `#d97706` (`--color-green` / `--color-amber`); amber also
  drives ramping; `--color-red` `#dc2626` for errors.
- `--font-sans` = **Hanken Grotesk** (geometric grotesque close to Temporal's
  Aeonik), loaded from Google Fonts via a `<link>` in `<head>` (accepted
  external CDN dependency / offline-cluster risk, like the other CDN scripts).

**Styling approach:** every semantic component class is styled with `@apply`
Tailwind utilities + the `@theme` tokens — no raw CSS property declarations,
**except** the irreducible exceptions: `@theme` token definitions,
`@keyframes`, the two `@media` blocks (`max-width: 760px` and
`prefers-reduced-motion: reduce`), the stepper connector pseudo-elements
(`.stepper::before/::after`, whose fill is a `calc()` driven by the live
`--fill` custom property), and the current-node inner dot
(`.node.cur .dot::after`).

**Why:** the user wants the demo to look unmistakably Temporal and to minimise
hand-written CSS in favour of Tailwind.

**How to apply:** add/adjust colours by editing the `@theme` tokens (don't
scatter raw hexes); keep new component styling in `@apply`. Do **not** try to
move the media queries into `@apply` variants — see the gotcha in
[[frontend-rules]]. Builds on [[frontend-rules]] and [[deployment-panel-ui]].
