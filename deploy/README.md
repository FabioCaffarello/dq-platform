<!-- path: deploy/README.md -->

# `deploy/` — Infrastructure Configuration

The deploy workspace holds Kubernetes manifests,
environment overlays (`local`, `qa`, `prod`), and
infrastructure-as-code files for the DQ Platform.

Per
[B1-10's resolution](../studies/decisions/2026-05-21-b1-10-workspace-tooling.md),
this workspace is **not a Go module by default**. If a
future Go-based infrastructure tool is introduced (e.g., a
manifest generator), its module is added to
[`go.work`](../go.work) at that time.

## Layout

```
deploy/
  base/                # W3-P7b — plain Kubernetes YAML, tool-neutral
    kustomization.yaml # W3-P7c — Kustomize base listing the 4 resources
    deployment.yaml    # engine Deployment (single replica, probes, resources)
    service.yaml       # ClusterIP Service exposing port 8080
    configmap.yaml     # dq-engine-env ConfigMap (no data — overlays patch)
    serviceaccount.yaml # dq-engine ServiceAccount (per-env IAM in overlays)
  overlays/            # W3-P7c — per-environment Kustomize artifacts
    local/             # local Compose substrate; sets DQ_ENV=local + emulator hosts
    qa/                # qa GCP project; sets DQ_ENV=qa + Workload Identity annotation
    prod/              # prod GCP project; sets DQ_ENV=prod + Workload Identity annotation
```

## What's in scope where

| Concern | Owner |
|---|---|
| Pod shape (image, ports, probes, resources, securityContext) | `deploy/base/deployment.yaml` |
| In-cluster traffic routing | `deploy/base/service.yaml` |
| Engine env-var surface (`DQ_ENV` + emulator overrides) | `deploy/base/configmap.yaml` (shape) + overlays (data) |
| Per-env GCP project / Workload Identity / IAM | Overlays (per-env operational sessions) |
| Per-env scaling (replicas, HPA) | Overlays |
| Ingress / NetworkPolicy / ResourceQuota | Overlays |
| Container build / registry / image tag | Future release-engineering session (B2-3) |
| Secrets (KMS / Vault / sealed-secrets) | Future B1 — flagged in B1-4's Out-of-scope list |

## Tooling

The overlay tool is **Kustomize**, committed in
[B2-8](../studies/decisions/2026-05-22-b2-8-infrastructure-tooling.md)
(provisional ADR-0017). `kubectl` 1.14+ ships Kustomize
natively, so neither workstations nor CI runners need a
separate binary install.

The base manifests under `deploy/base/` remain plain
Kubernetes YAML — the `kustomization.yaml` there only lists
them. A future amendment that switches to a different
app-deployment tool can still consume the base via
`kubectl apply -f` or `helm template` references.

**CI validity gate.** Every PR runs
[`.github/workflows/deploy-validate.yml`](../.github/workflows/deploy-validate.yml),
which invokes `make validate-deploy`. The target renders each
overlay via `kubectl kustomize` and rejects any rendering
error. This is purely client-side (no cluster contact), so it
runs identically on workstations and on the GitHub-hosted
runner.

The gate catches YAML syntax errors, missing-resource
references, patch-target mismatches, and strategic-merge
conflicts. It does **not** catch field-name typos against the
Kubernetes schema (e.g. a `replicass:` instead of `replicas:`
on a Deployment) — `kubectl apply -k --dry-run=client` would
catch that, but empirically the command still performs
API-server discovery before parsing, so it cannot run on a
cluster-free CI runner. Deeper schema validation (`kubeconform`
or a kind-based CI cluster) is deferred to a future
release-engineering session (B2-3) or a Phase-7 follow-up.

## How to apply

Render and inspect (no cluster needed):

```bash
kubectl kustomize deploy/overlays/local/
kubectl kustomize deploy/overlays/qa/
kubectl kustomize deploy/overlays/prod/
```

Apply against a configured cluster context (the Deployment
will sit in `ImagePullBackOff` until a real image lands per
the future release-engineering session, B2-3):

```bash
kubectl apply -k deploy/overlays/local/
```

## Citations

- [ADR-0013 §"Phase 7"](../docs/adr/0013-wave3-sequencing.md) —
  Kubernetes manifests + environment overlays as the Phase-7
  artifact.
- [ADR-0014 §4](../docs/adr/0014-trigger-handler-contract.md) —
  health endpoints `/healthz` (liveness) and `/readyz`
  (readiness); the Deployment wires them as the pod probes.
- [B1-4 MD-3](../studies/decisions/2026-05-22-b1-4-environment-configuration-model.md) —
  separate GCP project per environment; IAM is the isolation
  boundary. Each overlay annotates the SA with its env-specific
  GCP service-account email.
- [B1-4 MD-4](../studies/decisions/2026-05-22-b1-4-environment-configuration-model.md) —
  `DQ_ENV` is the only application-config env var the binary
  reads. The ConfigMap in the base exists so overlays have a
  single patch target.
