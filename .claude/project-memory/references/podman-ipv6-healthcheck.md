---
name: "Podman IPv6 localhost breaks the Temporal healthcheck"
description: "Temporal compose healthcheck must use 127.0.0.1, not the default localhost, under Podman"
type: project
---

# Podman IPv6 localhost breaks the Temporal healthcheck

The `temporal` service healthcheck in `compose.yaml` must target the explicit
IPv4 loopback: `temporal operator cluster health --address 127.0.0.1:7233`.
Without `--address` it defaults to `localhost:7233`, which fails under Podman
and reports the container as `unhealthy`, so `docker compose up --wait` (and
thus `make infra-up`) aborts before `backend`/`worker` ever start.

**Why:** Podman's generated `/etc/hosts` maps `localhost` to both `127.0.0.1`
and `::1`. The gRPC client resolves IPv6 `::1` first, but the dev server runs
with `--ip 0.0.0.0` (IPv4 only) and never answers on `::1`, so the probe hangs
until `context deadline exceeded`. Docker Desktop's `/etc/hosts` omits the
`::1 localhost` line, which is why it only surfaced on Podman.

**How to apply:** keep the explicit `--address 127.0.0.1:7233` on this (and any
similar) healthcheck. If a container is `unhealthy` but `temporal operator
cluster health --address 127.0.0.1:7233` returns `SERVING` from inside it,
suspect IPv6 loopback resolution rather than a real server failure.
