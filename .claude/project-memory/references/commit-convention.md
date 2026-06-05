---
name: "Commit message convention"
description: "Commit subjects start with an imperative action verb; no scope/type prefix"
type: feedback
---

# Commit message convention

Commit **subjects** start with an imperative action verb and describe the
action. **No scope or type prefix** — no `Frontend:`, `Dashboard:`, `Dev:`,
`Worker:`, `Fix:`, `CI:`, `Memory:`, nor Conventional-Commits `feat:`/`fix:`.

- Good: `Add the Deploy modal and stabilize the KPI band`;
  `Fix the Live Orders scroll region`;
  `Publish pizzaVersion as deployment version metadata`.
- Bad: `Frontend: add the Deploy modal`; `Fix: scroll region`;
  `feat: deploy modal`.

**Why:** The user asked for this explicitly and had the entire history rewritten
to match — a subject must read as "a verb with an action", not as a namespaced
changelog entry.

**How to apply:** Write `<Verb> <rest…>` in the imperative. Put any
scope/context in the commit body if it helps, never as a `Scope:` prefix. Keep
the existing body wrapping and `Co-Authored-By` trailer conventions.
