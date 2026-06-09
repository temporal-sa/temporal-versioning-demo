---
name: "Prefer short Makefile target names"
description: "Use short target names like app-v1/app-v2/app-v3; no verbose or duplicate-alias targets"
type: feedback
---

# Prefer short Makefile target names

The user prefers **short, consistent Makefile target names** and dislikes
redundant aliases.

- The Compose worker targets are `app-up` / `app-v1` / `app-v2` / `app-v3` /
  `app-down` / `app-logs` (renamed from the earlier verbose
  `app-worker-v2`/`app-worker-v3`).
- A parallel `compose-deploy*` alias family (mirroring the k8s `deploy*`
  targets) was added and then **removed as duplicates** — the short `app-vN`
  targets are the single Compose deploy interface.

**Why:** explicit user preference ("Je préfère utiliser des targets courtes
comme app-v1, app-v2" / "supprime les targets compose-deploy-v1 … en doublon").

**How to apply:** when adding or renaming Make targets, favour terse names over
descriptive-but-long ones, and do not add alias targets that merely duplicate an
existing one. The k8s deploy targets stay `deploy` / `deploy-vN` / `teardown`
(see [[version-shipping-kubectl-patch]]).
