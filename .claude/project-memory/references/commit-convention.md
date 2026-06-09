---
name: "Commit message convention"
description: "Commit subjects start with an imperative action verb; no scope/type prefix"
type: feedback
---

# Commit message convention

Commit **subjects** start with an imperative action verb; **no scope or type
prefix** (no `Frontend:`, `Fix:`, nor Conventional-Commits `feat:`/`fix:`).

- Good: `Add the Deploy modal and stabilize the KPI band`.
- Bad: `Frontend: add the Deploy modal`; `feat: deploy modal`.

**Why:** the user requires this — a subject must read as "a verb with an
action", not a namespaced changelog entry.

**How to apply:** write `<Verb> <rest…>` in the imperative; put any scope in the
commit body, never as a `Scope:` prefix.
