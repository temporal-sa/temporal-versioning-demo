---
name: "Write project memory in the present tense"
description: "Memory states the current truth, not a changelog of what changed"
type: feedback
---

# Write project memory in the present tense

Project-memory notes state the **current truth** of the project, in the present
tense. They are not a changelog of what changed.

- Good: "All components support hot-reload."; "The host-dev interface is
  `make dev`."
- Bad: "The worker targets were removed."; "X was added and then removed as a
  duplicate."; before/after migration anecdotes.

**Why:** explicit user instruction — "Rédige la mémoire au présent : on ne
relate pas le passé." Past-tense narration is noise; the repo's git history
already records what changed.

**How to apply:** when writing or editing any note, phrase every fact as the
present state. Drop "was added/removed/renamed", before/after stories, and
migration history. Keep the rationale in **Why**, stated as a present reason.
