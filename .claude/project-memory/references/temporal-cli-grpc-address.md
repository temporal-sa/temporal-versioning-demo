---
name: "Temporal CLI host address for the kind cluster"
description: "temporal CLI must target temporal.127-0-0-1.nip.io:7233, not 127.0.0.1:7233, or it gets a 404"
type: project
---

# Temporal CLI host address for the kind cluster

To run the `temporal` CLI from the host against the temporal-k8s
Kind cluster, set:

    export TEMPORAL_ADDRESS=temporal.127-0-0-1.nip.io:7233

**Why:** the cluster exposes Temporal's gRPC API through the Traefik
Gateway via a `gRPCRoute` (in the `temporal` namespace) keyed on the
hostname `temporal.127-0-0-1.nip.io`. The CLI's default
`127.0.0.1:7233` reaches Traefik's gRPC entrypoint (Kind maps host
7233 → NodePort 30723 → Traefik) but its `:authority` matches no
route, so Traefik returns HTTP 404 ("unexpected content-type
text/plain"). The nip.io host resolves to 127.0.0.1 too, but carries
the authority the route needs.

**How to apply:** use this address for every host-side `temporal`
command (e.g. `temporal worker deployment describe --name
pizza-tracker/pizza-worker`). In-cluster clients are unaffected —
the backend dials `temporal-frontend.temporal.svc.cluster.local:7233`
directly. Distinct from [[ipv6-localhost-healthcheck]] (which is about
the local Docker dev server, not the cluster). See also
[[temporal-api-constraints]].
