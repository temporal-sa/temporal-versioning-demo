---
name: "kbld digest pinning for deploy determinism"
description: "make deploy pipes kustomize through kbld to pin images to sha256 digests; pods keep IfNotPresent"
type: project
---

# kbld digest pinning for deploy determinism

`make deploy` is `kubectl kustomize k8s/base | kbld -f - | kubectl apply -f -`.
kbld (Carvel) rewrites every `image:` tag — including the WorkerDeployment
CRD's `spec.template.spec.containers[].image` (default kbld search rules match
the `image` key in any YAML doc) — to its immutable `…@sha256:<digest>` form
at apply time. Manifests keep human-readable tags (the worker base is now
`:v1`, not `:latest`) and pods use `imagePullPolicy: IfNotPresent`.

Note: **all** deploy paths go through kbld now. `make deploy` pins the `:v1`
base (`k8s/base`); `make deploy-v2`/`deploy-v3` apply kustomize overlays
(`kubectl kustomize k8s/vN | kbld -f - | kubectl apply -f -`) so v2/v3 are
digest-pinned too. See [[version-shipping-overlays]].

**Why:** with Worker Versioning the controller derives the Build ID from the
pod-template hash. A mutable `:latest` keeps the pod template (hence Build ID)
identical while image content drifts, so one Build ID could run two different
binaries — the exact non-determinism Versioning prevents. A pinned digest makes
a Build ID map to exactly one image; changing the image changes the digest,
the pod-template hash, and the Build ID together. With an immutable digest,
`IfNotPresent` is safe (cached == correct), so `Always` is unnecessary.

**How to apply:** keep tags in the manifests, never hardcode digests there; let
kbld resolve them at deploy. kbld is a deploy-time prerequisite (`brew install
carvel-dev/carvel/kbld`). Do not switch the cluster pods to `imagePullPolicy:
Always`. See [[worker-versioning-model]].
