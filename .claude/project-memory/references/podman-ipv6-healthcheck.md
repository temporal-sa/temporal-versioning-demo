---
name: "Podman IPv6 localhost breaks Temporal connections (healthcheck + host dev)"
description: "Use 127.0.0.1:7233, never localhost:7233, for the compose healthcheck and host dev client defaults under Podman"
type: project
---

# Podman IPv6 localhost breaks Temporal connections (healthcheck + host dev)

Always use the explicit IPv4 loopback `127.0.0.1:7233`, never `localhost:7233`,
for anything on the host (or in-container) that connects to the Podman-published
Temporal dev server. This applies in two places:

1. The `temporal` service healthcheck in `compose.yaml`:
   `temporal operator cluster health --address 127.0.0.1:7233`. Without
   `--address` it defaults to `localhost:7233`, fails, and reports the container
   `unhealthy`, so `docker compose up --wait` (and thus `make infra-up`) aborts
   before `backend`/`worker` ever start.
2. Host dev client defaults: `Makefile` (`TEMPORAL_ADDRESS ?= 127.0.0.1:7233`),
   `.env.local.example`, and the Go fallbacks in `cmd/worker/main.go` and
   `cmd/backend/main.go`. With `localhost`, the worker's eager `client.Dial`
   tries `::1` first (~1.1s vs ~0.03s for `127.0.0.1`), and on a cold start with
   all three workers dialing at once this can exceed the SDK's ~5s connection
   deadline → `make dev` dies with `failed reaching server: context deadline
   exceeded`. In-container compose addresses (`temporal:7233`) and k8s manifests
   are service-DNS based and must stay unchanged.

**Why:** Podman's generated `/etc/hosts` maps `localhost` to both `127.0.0.1`
and `::1`. The gRPC client resolves IPv6 `::1` first, but the dev server runs
with `--ip 0.0.0.0` (IPv4 only) and never answers on `::1`, so the probe hangs
until `context deadline exceeded`. The fix is portable: `127.0.0.1` targets the
exact IPv4 address the server listens on, so it is correct and faster on Docker
Desktop too (zero regression) — `localhost` is just an ambiguous name that may
resolve to `::1` first. The failure was only *observed* under Podman; whether
Docker Desktop hits the same stall has not been verified here, but the IPv4
loopback is the deterministic choice for any runtime.

**How to apply:** keep the explicit `--address 127.0.0.1:7233` on this (and any
similar) healthcheck. If a container is `unhealthy` but `temporal operator
cluster health --address 127.0.0.1:7233` returns `SERVING` from inside it,
suspect IPv6 loopback resolution rather than a real server failure.
