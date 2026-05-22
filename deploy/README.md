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

The `overlays/` subtree below is forward-looking — those
directories do not exist yet. **W3-P7c will create them**
alongside the overlay-tool choice (per B2-8).

```
deploy/
  base/                # W3-P7b — plain Kubernetes YAML, tool-neutral
    deployment.yaml    # engine Deployment (single replica, probes, resources)
    service.yaml       # ClusterIP Service exposing port 8080
    configmap.yaml     # dq-engine-env ConfigMap (no data — overlays patch)
    serviceaccount.yaml # dq-engine ServiceAccount (per-env IAM in overlays)
  overlays/            # W3-P7c will create — per-environment artifacts (open)
    local/             # W3-P7c will create — local Compose substrate; sets DQ_ENV=local + emulator hosts
    qa/                # W3-P7c will create — qa GCP project; sets DQ_ENV=qa
    prod/              # W3-P7c will create — prod GCP project; sets DQ_ENV=prod
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

The base manifests are **tool-neutral** plain Kubernetes YAML.
They work with `kubectl apply -f`, with Kustomize bases that
reference them, and with Helm chart templates that render
them.

The overlay tool choice (Kustomize / Helm / Terraform / other)
is committed in the **W3-P7c** session per the open
[B2-8](../studies/foundation/06-decision-log.md) infrastructure-
tooling decision. Until W3-P7c lands, the base is fully usable
on its own — `kubectl apply -f deploy/base/` schedules the
resources, and the Deployment stays in `ImagePullBackOff`
until a real container image lands (the
`dq-engine:placeholder` reference is the explicit deferral
marker for the release-engineering pipeline).

**CI validity gate.** No `kubectl apply --dry-run=client` /
`kubeconform` / equivalent CI lane runs against `deploy/base/`
yet — that lands with W3-P7c when overlays make the validation
load-bearing for the merge gate. Until then, contributors
validate manifests locally (visual inspection plus YAML
load).

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
