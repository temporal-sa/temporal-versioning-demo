---
name: "Kubernetes namespace: pizza-tracker"
description: "Demo deploys to a dedicated pizza-tracker K8s namespace, owned by Kustomize"
type: project
---

# Kubernetes namespace: pizza-tracker

The demo's K8s resources deploy to a dedicated `pizza-tracker`
namespace, not `default`. Kustomize is the single source of
truth: `k8s/kustomization.yaml` sets `namespace: pizza-tracker`,
and `k8s/namespace.yaml` (first entry in `resources:`) creates
the Namespace before the namespaced objects. No manifest carries
a hardcoded `metadata.namespace`.

**Why:** isolate the demo from `default`; let Kustomize own the
namespace so it lives in exactly one place.

**How to apply:**

- Do NOT add `metadata.namespace` to per-resource manifests —
  the `namespace:` field in kustomization.yaml applies it.
- These are the *Temporal* namespace and unrelated, leave them
  as `default` / `traefik`:
  - `workerdeployment.yaml` `spec.workerOptions.temporalNamespace: default`
  - `backend.yaml` env `TEMPORAL_NAMESPACE: default`
  - `httproute.yaml` `parentRefs[].namespace: traefik` (the gateway)
- `kubectl` commands inspecting the demo need `-n pizza-tracker`.
- The Worker Controller names the Temporal Worker Deployment
  `<k8s-namespace>/<WorkerDeployment-name>` (it injects
  `TEMPORAL_DEPLOYMENT_NAME=pizza-tracker/pizza` into the worker).
  So the backend's `PIZZA_DEPLOYMENT_NAME` MUST carry the namespace
  prefix or it logs "no Worker Deployment found". Don't hardcode it:
  `backend.yaml` derives `POD_NAMESPACE` via the Downward API
  (`fieldRef metadata.namespace`) and sets
  `PIZZA_DEPLOYMENT_NAME: $(POD_NAMESPACE)/pizza` (K8s expands
  `$(VAR)` from an env defined earlier in the list).
