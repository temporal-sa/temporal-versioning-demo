---
name: "Version shipping via kubectl JSON patch (no overlays)"
description: "deploy-vN patch PIZZA_VERSION on the live WorkerDeployment; kustomize overlays were abandoned"
type: project
---

# Version shipping via kubectl JSON patch (no overlays)

Shipping a later worker version to a running demo is done by patching the
live WorkerDeployment, **not** by Kustomize overlays.

- A Kustomize overlay attempt (`k8s/overlays/<v>`) was abandoned because it
  produced a kustomize ancestor-cycle. There must be **no `k8s/overlays`
  directory**.
- `make deploy` ships the committed base (v1): `kubectl kustomize k8s/ |
  kbld -f - | kubectl apply -f -`. The committed base stays v1.
- `make deploy-v1` / `deploy-v2` / `deploy-v3` call `ship-worker
  WORKER_VERSION=vN`, which runs a kubectl `--type=json` patch:
  `replace /spec/template/spec/containers/0/env/0/value` (PIZZA_VERSION is
  `env[0]` in `k8s/workerdeployment.yaml`).
- Only the env changes — the image stays the digest pinned by the last
  `make deploy`. Run `make deploy` once first to create the demo.
- Makefile vars: `K8S_NAMESPACE ?= pizza-tracker`,
  `WORKER_DEPLOYMENT ?= pizza-worker`.
