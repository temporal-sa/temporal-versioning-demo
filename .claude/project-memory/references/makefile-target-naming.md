---
name: "Prefer short Makefile target names"
description: "Short target names (app-vN, deploy-vN); make dev is the only host flow and hot-reloads every component"
type: feedback
---

# Prefer short Makefile target names

Make target names are **short and consistent**, with no redundant aliases.

- The Compose targets are `app-up` / `app-v1` / `app-v2` / `app-v3` /
  `app-down` / `app-logs`. The k8s deploy targets are `deploy` / `deploy-vN` /
  `teardown`. These short names are the single interface for each flow.
- The host-dev interface is **`make dev` only**. It runs the backend and all
  three workers under Air, so **every component hot-reloads**. Each worker runs
  from the shared `.air.worker.toml`, overriding `-tmp_dir`/`-build.cmd`/
  `-build.bin` to its own `tmp/worker-vN` dir so the three Air instances don't
  clobber each other's binary.

**Why:** the user wants terse, single-purpose target names and requires
hot-reload for every component.

**How to apply:** favour terse target names over long descriptive ones, and
never add an alias target that merely duplicates another. Keep all four host
components (backend + three workers) under Air — never run workers via `go run`.
See [[version-shipping-overlays]] and [[memory-writing-style]].
