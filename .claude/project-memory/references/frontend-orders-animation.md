---
name: "Live orders: morph + CSS-only collapse (single column)"
description: "The #orders region morphs (not replaces) and lays cards out as a single normal-flow column; Done collapse and the layout are pure CSS — no masonry/positioning script"
type: feedback
---

# Live orders: morph + CSS-only collapse (single column)

The live `#orders` region is animated; these non-obvious points are easy to get
wrong. **History:** there used to be a client-side multi-column *masonry* script
(absolute-positioned cards, column pinning, a `data-collapsing` JS timer, and an
idiomorph `beforeAttributeUpdated` guard). It was **removed** once the layout
settled to a single column — cards are now plain normal-flow blocks and every
animation is CSS. Do not reintroduce the masonry.

- **`#orders` morphs, it does not replace.** It uses idiomorph
  (`hx-swap="morph:innerHTML"`); each card has a stable `id="order-{ID}"` so
  existing cards persist as DOM nodes. That persistence is what lets the
  once-only entry animation (`card-in`) play only for genuinely new nodes and
  lets the stepper fill (`--fill`, server-rendered on `.stepper`) glide via its
  CSS `width` transition. A plain `innerHTML` swap would recreate nodes every
  tick and kill both (the other sse-swap targets — `versions` — do use plain
  `innerHTML`). idiomorph now syncs **all** attributes normally (no guard): the
  cards carry no client-owned `style`/`data-*`, and we *want* idiomorph to
  update the `done`/`fail` classes and the stepper `--fill`.
- **Single column, normal document flow.** Cards (`.order`) are ordinary blocks
  stacked in `.olist` (just `shrink-0`); inter-card spacing is each card's
  `mb-3` (12px). `.olist` is `shrink-0` so this flex item of the `flex-col`
  `.dleft` keeps its content height and `.dleft` (the `overflow-y-auto` scroll
  container) scrolls instead of compressing the list. `.dleft` keeps
  `scrollbar-gutter: stable` so the centered column does not shift sideways when
  the scrollbar toggles.
- **Done → visible-then-collapse, no JavaScript.** The server adds class `done`
  (the all-green stepper is the cue); the card stays full for ~4s, then the
  `card-collapse` keyframe (`animation: card-collapse 0.5s ease 4s forwards` on
  `.order.done`) shrinks `max-height`/`margin-bottom`/padding/border to 0 and
  fades+greys it, with `overflow: hidden` clipping the content during the
  shrink. Because cards are in normal flow, the cards below simply reflow up as
  the height collapses — no per-card position transition needed.
  `max-height: 600px` on `.order` is the finite start value the collapse
  animates from (real cards are far shorter, so it never clips a live card).
  `DeliveredDwell` keeps the workflow Running long enough for this to finish —
  see [[demo-timing]]. Under `prefers-reduced-motion` the collapse keeps its 4s
  delay but snaps shut (`0.01s`) so done cards still clear the board.
- **Hover lift is masonry-free but still layout-safe.** `.order:hover:not(.done)`
  lifts via `transform: translateY(-2px)` and adds a neutral-grey `outline` ring
  (not box-shadow) — both have zero layout impact. Scoped `:not(.done)` so the
  lift never fights the collapse. `.order.fail` keeps its red identity on hover.
- **Body layout:** `.dbody` is a centered `max-w-5xl` flex row — a flexible
  orders column (`.dleft`, `flex-1`) and a fixed 360px deployment column
  (`.dright`, `shrink-0`); the divider is a 1px `border-left` on `.dright`. On
  mobile (`max-[760px]`) it switches to `flex-col` and **Deployment stacks above
  Orders** via `max-[760px]:order-first` on `.dright` (desktop keeps the natural
  DOM order: orders left, deployment right). The per-order stepper spans the
  full card width as one progress track whose green fill is driven by `--fill`.

**Why:** The animations depend on idiomorph DOM-node persistence; the rest is
deliberately plain CSS so there is no positioning script to keep in sync (the
old masonry's flicker/reshuffle/drift bugs are gone with it).

**How to apply:** Keep morphing + stable ids for `#orders`; keep cards in normal
flow with `mb-3` spacing; do the Done collapse and any responsive ordering in
CSS, not JS. General frontend rules are in [[frontend-rules]]; the Deployment
panel is covered in [[deployment-panel-ui]].
