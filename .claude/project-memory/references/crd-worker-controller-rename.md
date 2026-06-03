---
name: "Worker Controller CRD rename (use WorkerDeployment/Connection)"
description: "The temporal-k8s cluster runs controller chart 0.26.0, so use WorkerDeployment/Connection, not the Temporal* kinds"
type: project
---

# Worker Controller CRD rename (use WorkerDeployment/Connection)

The `temporal-k8s` cluster runs Temporal Worker Controller **chart 0.26.0 (app
v1.7.0)** (see `temporal-k8s/infra/temporal-worker-controller/helmrelease.yaml`).
Chart 0.26.0 **renamed** the controller CRDs and the OLD kinds are no longer
reconciled (new objects of the old kinds never become Ready):

- `TemporalWorkerDeployment` → `WorkerDeployment`
- `TemporalConnection` → `Connection`

API group/version stays `temporal.io/v1alpha1`; field layout is otherwise
identical. The worker pod template uses `spec.rollout.strategy: Manual`,
`spec.workerOptions.connectionRef`/`temporalNamespace`, and a `Connection` CR
whose `hostPort` is the Temporal frontend.

**Why:** The demo spec and CLAUDE.md still say `TemporalWorkerDeployment` — that
is out of date for this cluster's chart version and would silently never
reconcile.

**How to apply:** Author `k8s/workerdeployment.yaml` and `k8s/connection.yaml`
with the new kinds. Before applying, verify with `kubectl get crd | grep -i
temporal`; if only the legacy `temporalworkerdeployments` CRD exists, fall back
to the old kinds (same fields). See [[pizza-demo-architecture-decisions]].
