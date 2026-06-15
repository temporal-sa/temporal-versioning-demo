---
name: "Verifying the rollout flow across deployment modes"
description: "How to E2E-verify v1->v2->v3->v2 ramp/rollback/recover in dev, Docker and kind, and why they run one at a time"
type: reference
---

# Verifying the rollout flow across deployment modes

The full demo flow (v1 steady -> ramp v2 25/50/100+promote -> ship+ramp
v3 25% -> rollback to v2 -> recover stuck v3 orders) can be exercised
end-to-end in each mode. Notes from doing it:

## Modes are mutually exclusive on host port 7233

`make dev`, `make app-up` (Docker) and the kind cluster all bind host
**port 7233**: the Compose Temporal dev server publishes 7233, and the
kind cluster's Traefik gRPC listener serves `temporal.127-0-0-1.nip.io:7233`
(which resolves to 127.0.0.1). So only one mode can run at a time —
verify them **sequentially**, tearing each down before the next
(`docker compose down` wipes the in-memory dev-server state). See
[[temporal-host-addresses]].

## Driving the rollout headlessly (the backend's own endpoints)

The Deploy modal maps to plain form POSTs, so the flow is scriptable
without clicking (or via cmux browser clicking the same controls):

- `POST /deploy` with `version=vN` and `stop=IDX`, where the slider stop
  index is **0=25%, 1=50%, 2=100% (promote)**.
- `POST /rollback` drops the ramp (new orders snap back to Current).
- per-order **Fix** button = `POST /orders/{id}/recover` (reset-with-move
  onto the Current build).

In the rendered UI the controls-row buttons are `button[hx-get="/deploy"]`
and `button[hx-get="/rollback"]`; inside the modal the radios are
`input[name="version"][value="vN"]`, the slider is `input[name="stop"]`,
and the recover buttons are `button.recover-btn`.

## Ground-truth via the Temporal CLI

`temporal worker deployment describe --name <DN> -o json` exposes
`.routingConfig.currentVersionBuildID`, `.rampingVersionBuildID` and
`.rampingVersionPercentage`. The friendly v1/v2/v3 labels are NOT in that
output — read each version's `describe-version --build-id <id>` and
base64-decode `.metadata.pizzaVersion.data` (a JSON-string payload).

`<DN>` is **`pizza`** locally (dev/Docker) and
**`pizza-tracker/pizza-worker`** on the cluster. Local Build IDs are the
`vN-local` strings; on kind they are pod-template-hash IDs, so always
resolve labels through the metadata. Cluster CLI address is
`temporal.127-0-0-1.nip.io:7233`; dashboard at
<http://pizza.127-0-0-1.nip.io/>. A stuck v3 order's recovery is
confirmed when its workflow is RUNNING and pinned to the v2 build with no
`DroneDelivery` activity in its fresh history.

## Ground-truth via the WorkerDeployment CR (cluster only)

On the kind cluster, the Worker Controller mirrors the routing config
onto the `WorkerDeployment` CR status, so you can read it with kubectl
instead of the Temporal CLI:

```
kubectl get workerdeployment pizza-worker -n pizza-tracker -o jsonpath='{.status.currentVersion.buildID} {.status.targetVersion.status} {.status.targetVersion.rampPercentage}'
```

`.status.targetVersion.status` is `Ramping` while a ramp is in flight
(with `rampPercentage` 25/50), and flips to `Current` after a promote
(ramp cleared); after a rollback the old target shows `Draining`. The CR
carries Build IDs, not the friendly v1/v2/v3 labels.
