---
name: "Temporal host addresses: dev server vs kind cluster"
description: "Local dev server → 127.0.0.1:7233 (IPv4-only); kind-cluster CLI → temporal.127-0-0-1.nip.io:7233 (Traefik authority)"
type: project
---

# Temporal host addresses: dev server vs kind cluster

Two different host-side Temporal endpoints — don't confuse them:

- **Local Docker dev server → `127.0.0.1:7233`** (never `localhost:7233`).
  Used by the Compose `temporal` healthcheck and the host dev client defaults
  (`Makefile`, the Go fallbacks in `cmd/worker`/`cmd/backend`). `localhost`
  resolves to both `127.0.0.1` and `::1`; the gRPC client tries IPv6 `::1`
  first, but the dev server is IPv4-only (`--ip 0.0.0.0`) and never answers on
  `::1`, so the probe hangs until `context deadline exceeded` — the healthcheck
  reports `unhealthy` (aborting `make infra-up`) and `make dev` dies on cold
  start. (First hit under Podman; Docker Desktop can do the same.)

- **temporal-k8s kind-cluster CLI → `temporal.127-0-0-1.nip.io:7233`** (not
  `127.0.0.1:7233`). The cluster exposes Temporal's gRPC API through a Traefik
  `gRPCRoute` keyed on that hostname. The default `127.0.0.1:7233` reaches
  Traefik's gRPC entrypoint (Kind maps host 7233 → NodePort 30723) but its
  `:authority` matches no route, so Traefik returns HTTP 404. The nip.io host
  resolves to 127.0.0.1 too, but carries the authority the route needs.

**How to apply:** for the local dev server keep explicit `--address
127.0.0.1:7233` on healthchecks (if a container is `unhealthy` but `temporal
operator cluster health --address 127.0.0.1:7233` returns `SERVING` from inside
it, suspect IPv6 loopback, not a real failure). For host-side CLI against the
cluster, `export TEMPORAL_ADDRESS=temporal.127-0-0-1.nip.io:7233` (e.g.
`temporal worker deployment describe --name pizza-tracker/pizza-worker`).
In-container compose addresses (`temporal:7233`) and k8s service DNS
(`temporal-frontend.temporal.svc.cluster.local:7233`) are unaffected. See also
[[temporal-api-constraints]].
