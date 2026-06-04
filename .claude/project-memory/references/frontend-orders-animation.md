---
name: "Live orders: idiomorph + masonry gotchas"
description: "The #orders region morphs (not replaces) and uses a client-side pinned-column masonry; key gotchas that cause flicker/reshuffle if broken"
type: feedback
---

# Live orders: idiomorph + masonry gotchas

The live `#orders` region is animated; these non-obvious gotchas are easy to
reintroduce:

- **`#orders` morphs, it does not replace.** It uses idiomorph
  (`hx-swap="morph:innerHTML"`); each card has a stable `id="order-{ID}"` so
  existing cards persist as DOM nodes. That persistence is what lets CSS
  transitions and the once-only entry animation work. A plain `innerHTML` swap
  recreates nodes every tick and kills the animations (the other sse-swap
  targets — kpis/versions/controls — do use plain `innerHTML`).
- **CRITICAL idiomorph gotcha:** idiomorph removes any attribute the server
  didn't render. The masonry script owns the inline `style` and all `data-*` on
  `.order`, but the server renders them empty — so each morph stripped them,
  causing flicker (cards flash to 0,0) and column reshuffle (lost `data-col`).
  Fix: a global `Idiomorph.defaults.callbacks.beforeAttributeUpdated` that
  returns `false` for `style` and `data-*` on `.order`. Do not let idiomorph
  touch them.
- **Client-side masonry of pinned columns** (vanilla script in `index.html`, not
  CSS grid — grid can't do per-column collapse). Each order pins to a column for
  its lifetime and never reshuffles; a new card goes to the shortest column; a
  full re-pack happens only when the column count changes on resize. Capped at
  **2 columns** (`MAX_COLS`). `.dleft` is the scroll container with
  `scrollbar-gutter: stable` — without it, a classic scrollbar appearing as the
  list grows shrinks the width and the columns drift.
- **Done → visible-then-collapse:** the server adds class `done`; the card stays
  full-height/all-green for `COLLAPSE_DELAY` (4 s), then a one-shot timer sets
  `data-collapsing`, which drives both the CSS collapse and the masonry's
  zero-height treatment so neighbours glide up. `DeliveredDwell` keeps the
  workflow Running long enough for this to finish — see [[demo-timing]].
- **Layout intents to preserve:** the right Deployment column is pinned to
  exactly one third (`basis-1/3 grow-0 shrink-0`) to align with the 3rd KPI
  cell; the per-order stepper spans the full card width as one progress track
  whose fill is driven server-side by `--fill`.

**Why:** These animations depend on DOM-node persistence and on the script
solely owning layout attributes; the failure modes (flicker, reshuffle, drift)
are subtle and were each hit during development.

**How to apply:** Keep morphing + stable ids for `#orders`; never let idiomorph
write `style`/`data-*` on `.order`; keep the stable scrollbar gutter. General
frontend rules are in [[frontend-rules]].
