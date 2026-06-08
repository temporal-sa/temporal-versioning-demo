---
name: "Podman IPv6 localhost breaks Temporal connections"
description: "Use 127.0.0.1:7233, never localhost:7233, for the compose healthcheck and host dev client defaults under Podman"
type: project
---

# Podman IPv6 localhost breaks Temporal connections

Always use the explicit IPv4 loopback `127.0.0.1:7233`, never `localhost:7233`,
for anything on the host that connects to the Podman-published Temporal dev server
— the compose `temporal` healthcheck and the host dev client defaults (`Makefile`,
`.env.local.example`, the Go fallbacks in `cmd/worker` and `cmd/backend`).

**Why:** Podman maps `localhost` to both `127.0.0.1` and `::1`, and the gRPC client
tries IPv6 `::1` first; but the dev server runs IPv4-only (`--ip 0.0.0.0`) and
never answers on `::1`, so the probe hangs until `context deadline exceeded`. This
made the healthcheck report `unhealthy` (aborting `make infra-up`) and made
`make dev` die on cold start with all three workers dialing at once. `127.0.0.1`
targets the exact address the server listens on — correct and faster on Docker
Desktop too. (In-container compose addresses `temporal:7233` and k8s service DNS
are unaffected and must stay unchanged.)

**How to apply:** keep the explicit `--address 127.0.0.1:7233` on this and any
similar healthcheck. If a container is `unhealthy` but `temporal operator cluster
health --address 127.0.0.1:7233` returns `SERVING` from inside it, suspect IPv6
loopback resolution, not a real failure.
