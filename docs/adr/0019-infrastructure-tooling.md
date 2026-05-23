<!-- path: docs/adr/0019-infrastructure-tooling.md -->

# ADR-0019 — Infrastructure Tooling for Per-Environment Overlays

- **Status:** accepted
- **Date:** 2026-05-23

---

## Context

The `deploy/` workspace carries the platform's Kubernetes
artifacts. `deploy/base/` ships four plain Kubernetes YAML
manifests — `Deployment`, `Service`, `ConfigMap`,
`ServiceAccount` — that are tool-neutral. ADR-0013 §"Phase 7"
defers the choice of overlay tool to Phase 7 itself; ADR-0018
commits the three first-class environments (`local`, `qa`,
`prod`) the overlays must produce, the substrate-isolation
posture (separate GCP project per non-local environment), and
the deployment-bucket contents (per-environment ConfigMap data
plus a Workload Identity annotation on the
`dq-engine` ServiceAccount).

The per-environment delta the overlay tool must encode is
narrow:

- The `dq-engine-env` ConfigMap gains a `data:` block setting
  `DQ_ENV` per environment, plus `STORAGE_EMULATOR_HOST` and
  `BIGQUERY_EMULATOR_HOST` for the `local` overlay only.
- The `dq-engine` ServiceAccount gains an
  `iam.gke.io/gcp-service-account` annotation per environment,
  carrying a per-environment GCP service-account email.
- Optional per-environment tuning of `imagePullPolicy`,
  `replicas`, resource requests/limits, and probe periods may
  follow later.

There is no per-environment templating of values, no per-
environment conditional logic, and no external chart-
distribution requirement at v1. The engine runs in the
project's own clusters; the deploy artifacts are single-target.

This ADR commits the overlay-tool choice and the CI validity-
gate posture that flows from it.

**Out of scope of this ADR:**

- Cloud-infrastructure provisioning (GCP project, GKE cluster,
  IAM bindings). That layer is independent of app deployment;
  the operational session that provisions each environment's
  GCP project per ADR-0018 may pick Terraform, Pulumi, or
  another IaC tool at the infra layer, and this ADR does not
  foreclose any choice there.
- External Helm-chart distribution. If a requirement to ship
  the platform to external consumers emerges, a forward-
  compatible amendment can layer a chart on top of the
  rendered overlay output.
- A CI workflow that applies rendered manifests against a real
  or `kind`-based cluster on PR merge. The v1 validity gate is
  client-side dry-run only; deeper cluster-side validation is
  release-engineering territory.
- Secrets-storage primitive. Kustomize accommodates the three
  common patterns — `secretGenerator`, references to externally
  managed `Secret` objects, and sealed-secrets style flows;
  the specific primitive is a separate future decision.

---

## Decision

### 1. Kustomize is the overlay tool

`deploy/` adopts Kustomize for the base-plus-overlays
composition. The artifact shape is:

- `deploy/base/kustomization.yaml` — lists the four
  tool-neutral resources already in `deploy/base/`.
- `deploy/overlays/{local,qa,prod}/kustomization.yaml` — one
  per environment, each strategic-merging patches into the base
  to set `DQ_ENV` and the per-environment ServiceAccount
  annotation. The `local` overlay additionally patches the
  emulator-host overrides into the `dq-engine-env` ConfigMap.

Three alternatives are rejected:

- **Helm.** Chart templating is overkill for the small per-
  environment delta committed by ADR-0018. Helm's strengths —
  release lifecycle (`helm install` / `helm upgrade --atomic`
  / `helm rollback`), shared-chart distribution to many
  consumers, values-file templating — solve problems the
  platform does not have at v1. The templating engine also
  adds a fragility surface (template-evaluation bugs,
  helper-function edge cases) that the Kustomize "what you
  write is what you get" model avoids. The same operational
  outcomes Helm delivers are achievable with `kubectl apply -k`
  plus a declarative-Git-as-source-of-truth discipline.
- **Terraform with the `kubernetes` provider.** Wrong layer.
  Terraform is well-suited to cloud-resource provisioning
  (GCP projects, GKE clusters, IAM) and remains a viable
  choice at that infra layer, but the Kubernetes provider for
  app-deployment introduces plan/apply state-file ceremony
  that Kubernetes manifests do not benefit from in the same
  way cloud resources do.
- **Combination (Terraform + overlay tool).** Reducible: the
  question of *which* overlay tool inside the combination is
  precisely the question this ADR answers (Kustomize). A
  future combination — Terraform at the infra layer, Kustomize
  at the app layer — is fully compatible with this ADR, since
  the two artifacts live at different layers.

### 2. CI validity gate — `kubectl apply -k --dry-run=client`

The validity gate for the overlay artifacts is a client-side
dry-run rendering of each overlay:

```
kubectl apply -k deploy/overlays/<env>/ --dry-run=client
```

The gate runs against each of the three overlays on every PR
that touches `deploy/`. A successful run validates that the
overlay parses, references its base correctly, and produces
syntactically valid Kubernetes manifests; it does not validate
against a real cluster's API server. Cluster-side validation
(`kubeconform`, `kind`-based lanes) is release-engineering
territory and lands as a future amendment if drift between
client-side validation and cluster admission proves to be a
real problem.

The gate uses the `kubectl` binary already required for any
Kubernetes-touching CI; Kustomize ships inside `kubectl` since
v1.14 (`kubectl apply -k`), so no separate Kustomize binary is
installed in CI.

### 3. The base stays tool-neutral

`deploy/base/` is not modified to accommodate the overlay tool.
The four resources remain plain Kubernetes YAML with the
`app.kubernetes.io/name` and `app.kubernetes.io/component`
labels in place; the `dq-engine-env` ConfigMap remains data-
empty so overlays can `merge` or `replace` data into it. A
future amendment that picks a different overlay tool can still
consume the base via `kubectl apply -f` or via a `helm template`
wrapper without unwinding this ADR.

---

## Consequences

1. The `deploy/` workspace ships a `deploy/base/kustomization.yaml`
   listing the base resources, plus
   `deploy/overlays/{local,qa,prod}/kustomization.yaml`. Each
   overlay carries the patches that set `DQ_ENV` and the
   per-environment ServiceAccount annotation per ADR-0018; the
   `local` overlay additionally carries the emulator-host
   patches.

2. The `make validate-deploy` target (and the matching CI
   workflow) runs `kubectl apply -k --dry-run=client` against
   each of the three overlays. The target is the platform's
   local-and-CI validity gate for the deploy layer.

3. No separate Kustomize binary installation is required in
   CI. Any CI runner with a recent `kubectl` (≥1.14) can run
   the validity gate.

4. The base manifests retain their tool-neutral posture. A
   future amendment that introduces a second overlay tool
   (e.g., a Helm chart layered on top for external
   distribution) can consume the base via `kubectl apply -f`
   or via `helm template` without touching this ADR's
   commitments.

5. Cloud-infrastructure provisioning remains independent of
   this choice. The operational session that creates each
   environment's GCP project (per ADR-0018 Consequence 6) may
   pick Terraform, Pulumi, or another IaC tool at the infra
   layer; the resulting GKE cluster receives the Kustomize-
   rendered manifests through `kubectl apply -k`. Infra and
   app deployment are orthogonal artifacts.

6. Secrets-storage choice is unconstrained by this ADR.
   Kustomize accommodates the three common Kubernetes-Secret
   patterns — `secretGenerator` for synthesized values,
   references to externally managed `Secret` objects, and
   sealed-secrets style flows. The future B1 that commits the
   secrets-storage primitive selects among them; this ADR's
   choice does not foreclose any.

7. The deploy artifacts are inspectable end-to-end. Running
   `kubectl kustomize deploy/overlays/<env>/` (or `kubectl
   apply -k --dry-run=client -o yaml`) renders the final
   manifests as base + patch with no templating step, so a
   future maintainer sees exactly what will be applied to the
   cluster.

8. Per-entity refinements (e.g., a per-entity Workload
   Identity annotation) compose additively into the relevant
   environment's overlay. The minimum-viable shape is three
   flat overlays referencing one base; cross-overlay
   duplication is not yet a real problem, so Kustomize
   `components` are not used at v1.

9. Cluster-side validation is deferred. The
   `--dry-run=client` gate catches syntax and schema issues
   without standing up a cluster; deeper validation (admission
   webhooks, RBAC checks, dry-run against a real API server)
   lands as a release-engineering follow-up if the client-side
   gate proves insufficient.

10. Reopening this ADR is required to change the overlay tool
    (e.g., to switch from Kustomize to Helm for the primary
    artifact). Adding a Helm chart that *wraps* the
    Kustomize-rendered output for external distribution is
    additive and does not reopen the ADR; the wrapper layer
    consumes Kustomize's output as input.

11. Adding a fourth overlay (e.g., a hypothetical `staging`
    overlay) requires first amending ADR-0018 to introduce the
    fourth first-class environment, then adding the overlay
    directory under `deploy/overlays/staging/`. The two
    amendments are independent: ADR-0018 commits the closed
    environment set, this ADR commits the tool that materializes
    it.

12. The Kustomize choice does not constrain the future Helm-
    chart route. The wrapper pattern (`helm template …` or
    `helm dependency` against a Kustomize source) lets a chart
    re-use the existing base and overlays without unwinding
    them; the chart would carry only the distribution-time
    concerns (chart metadata, version, values defaults) that
    Kustomize does not address.

---

## Notes

- The Kustomize choice is grounded in the narrowness of the
  per-environment delta committed by ADR-0018. A future
  amendment that significantly expands the delta (e.g., by
  introducing per-environment behaviour flags or a complex
  per-environment topology) may need to revisit the
  templating-versus-patching trade-off; the current shape
  doesn't warrant templating.

- The "deep cluster-side validation deferred" deviation from
  the strictest interpretation of "validate before apply" is a
  deliberate v1 simplification, not a permanent posture. The
  release-engineering follow-up (B2-3 or a Phase 7 successor)
  is the natural place to introduce `kubeconform`, a `kind`-
  based lane, or an integration with a real cluster's
  admission API.

- The choice of `kubectl apply -k --dry-run=client` over the
  bare `kubectl kustomize | kubectl apply -f - --dry-run` form
  is a convenience; the two are equivalent in their validation
  output, and the `-k` form is the canonical Kubernetes-native
  invocation.

- The base manifests' tool-neutrality is the property that
  preserves future optionality. A future amendment that picks
  a different overlay tool starts from the same plain YAML;
  the cost of switching is the cost of rewriting the overlay
  layer only, not the base.
