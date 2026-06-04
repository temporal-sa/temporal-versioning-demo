# temporal-versioning-demo

A customer-facing demo of Temporal **Worker Versioning** on
Kubernetes, built around a live **Pizza Tracker** dashboard.

See [README.md](README.md) for full documentation.

## Tech stack

- **Go** — versioned Temporal worker (`cmd/worker`) and
  backend API (`cmd/backend`); `net/http` + SSE
- **Temporal** — Worker Versioning via the Temporal Worker
  Controller (Pinned behavior, Manual rollout strategy)
- **Frontend** — single-page Pizza Tracker dashboard
- **Kustomize** — K8s manifests under `k8s/`
- **Docker** — images published to ghcr.io
- **Make** — task runner (`Makefile`)

## Build & run

```sh
make build       # build worker + backend binaries
make dev         # Temporal (Docker) + backend + worker v1 on the host
make backend     # run only the backend locally (hot reload)
make worker      # run only the worker v1 locally
make test        # run tests
make deploy      # deploy to the temporal-k8s Kind cluster
```

## Modules

- `cmd/worker/` — versioned Temporal worker (Pinned)
- `cmd/backend/` — REST + SSE API, state poller, rollout actions
- `internal/pizza/` — pizza workflow, activities, shared types
- `frontend/` — Pizza Tracker SPA
- `k8s/` — Kustomize manifests (applied to temporal-k8s)

## Agents

Use the following agents (from the
[skillbox](https://github.com/alexandreroman/skillbox)
plugin) for all code tasks:

- **code-writer** — for ANY task that writes, modifies, or
  refactors code or YAML manifests. This includes one-line
  fixes, import changes, and config tweaks. Never use the
  Edit or Write tools directly on source files — always
  delegate to this agent.
- **code-reviewer** — for read-only code review before
  merging or when investigating issues.

## Memory

At the start of every conversation, read
`.claude/project-memory/MEMORY.md` to load project context
from previous conversations.

Use the **project-memory** skill (from the
[skillbox](https://github.com/alexandreroman/skillbox)
plugin) proactively — without being asked — whenever the
conversation reveals project decisions, deadlines, external
references, workflow preferences, or corrective feedback
worth persisting across conversations.

**Important:** Never use the built-in auto-memory system
(`~/.claude/projects/.../memory/`) for project context — it
is local and not shared with the team.

## Conventions

- Line length: text/Markdown 80 cols, code 120 cols.
- Standard Markdown: blank lines around headings, lists and
  fenced code blocks (with a language tag).
- Always use the latest stable language/framework/library
  versions; verify with context7 before adding a dependency.
- **English only** for all code, comments, docs and UI.
- **Worker versioning rules:** workflows are **Pinned**; the
  controller runs in **Manual** strategy; shipping new code
  goes through K8s (image change), routing (ramp / promote /
  rollback) is driven from the UI via the Temporal API.
- Target cluster: the local `temporal-k8s` Kind cluster
  (Temporal Server + Worker Controller already deployed).
