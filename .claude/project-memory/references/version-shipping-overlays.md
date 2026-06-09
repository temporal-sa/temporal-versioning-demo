---
name: "Version shipping via kustomize base + sibling overlays (k8s/)"
description: "k8s/base is the v1 base; k8s/v2 & k8s/v3 overlays reference ../base and repoint the worker image to :vN through kbld; sibling layout avoids the ancestor-cycle"
type: project
---

# Version shipping via kustomize base + sibling overlays (k8s/)

Shipping a later worker version to a running demo uses **kustomize overlays**,
each repointing the worker image tag, applied through kbld (so v2/v3 are
digest-pinned like the base). The version is baked per image — see
[[worker-versioning-model]].

- **Layout (base + sibling overlays, avoids the cycle).** Everything lives under
  `k8s/`: the base is `k8s/base/` (the 5 manifests + their `kustomization.yaml`,
  `namespace: pizza-tracker`); overlays are `k8s/v2` and `k8s/v3`, each with
  `resources: [../base]` + an `images:` transformer
  (`name: …/temporal-versioning-demo-worker`, `newTag: vN`). Overlays reference
  `../base` (a **sibling**, not an ancestor). An overlay that references its
  ancestor base dir triggers a kustomize `cycle detected` error (proven: an
  overlay at `k8s/<x>` with `resources: [..]` fails) — that is why the base is a
  subdir, not `k8s/` root, and why there is **no `k8s/overlays/` directory**.
- `make deploy` ships the base (v1): `kubectl kustomize k8s/base | kbld -f - |
  kubectl apply -f -`. Plain apply is `kubectl apply -k k8s/base`.
  `k8s/base/workerdeployment.yaml` keeps the `:v1` tag.
- `make deploy-v1` re-applies the base (`= make deploy`); **no `k8s/v1`
  overlay** exists.
- `make deploy-v2` / `deploy-v3` run `kubectl kustomize k8s/vN | kbld -f - |
  kubectl apply -f -` — a full, idempotent re-apply of the stack with the worker
  image at `:vN`, kbld-resolved to a digest.
- `make teardown` is `kubectl delete -k k8s/base`.

There is no live `kubectl patch` shipping path and no `k8s/overlays/` or
`deploy/` directory: shipping is purely the overlay re-apply above.
