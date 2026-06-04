# temporal-versioning-demo

A customer-facing demo of Temporal **Worker Versioning** on
Kubernetes, built around a live **Pizza Tracker** dashboard.

See [README.md](README.md) for the architecture, tech stack,
module layout, and the full `make` task list.

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
from previous conversations — including the worker-versioning
rules, target cluster, and other architecture decisions that
used to live in this file.

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
