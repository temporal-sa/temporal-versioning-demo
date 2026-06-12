#!/usr/bin/env bash
# cmux pre-destroy hook: tear down this worktree's Docker stack before the
# worktree is removed, so closing or cancelling a workspace leaves no orphaned
# containers, networks or volumes behind. Mirrors `make app-down` by activating
# the v2/v3 profiles so their workers are removed too.
#
# Best-effort: a teardown failure (e.g. Docker not running) must not block
# workspace removal, so errors are tolerated.
set -uo pipefail

cd "${CMUX_FEATURE_WORKTREE:-$PWD}"

if ! command -v docker >/dev/null 2>&1; then
  echo "pre-destroy: docker not found; nothing to tear down."
  exit 0
fi

echo "pre-destroy: tearing down the Docker stack..."
docker compose --profile v2 --profile v3 down --volumes --remove-orphans \
  || echo "pre-destroy: docker compose down failed (continuing)."

rm -f compose.override.yaml
