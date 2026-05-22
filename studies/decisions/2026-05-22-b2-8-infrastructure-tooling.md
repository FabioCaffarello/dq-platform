<!-- path: studies/decisions/2026-05-22-b2-8-infrastructure-tooling.md -->

# B2-8 — Infrastructure Tooling

## Metadata

- **B2 reference:** B2-8 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md)
  (row at line 86, "Kustomize, Helm, Terraform, or a combination
  for `deploy/`?", expected output "Infrastructure ADR").
- **Status:** draft (resolved-study after the critique pass).
- **Last updated:** 2026-05-22.
- **Upstream resolved:**
  - Foundation 04 §"PAT-4 — Typed multi-environment configuration"
    — commits the per-env Go file model that landed in W3-P7a.
  - **B1-4** ([`studies/decisions/2026-05-22-b1-4-environment-configuration-model.md`](./2026-05-22-b1-4-environment-configuration-model.md))
    MD-1 (three-bucket configuration), MD-2 (closed env set:
    `local` / `qa` / `prod`), MD-3 (separate GCP project per
    env), MD-4 (`DQ_ENV` is the only application env-var
    surface; per-env values live in
    `engine/internal/env/{local,qa,prod}.go`).
  - **W3-P7a** — engine binary's typed multi-env package +
    `DQ_ENV` selector + 2 emulator-host overrides (PR #14,
    commit `bc4151a`).
  - **W3-P7b** — `deploy/base/` plain Kubernetes manifests
    (Deployment + Service + ConfigMap + ServiceAccount;
    PR #15, commit `b253280`).
  - **ADR-0013 §"Phase 7"** — names `local`, `qa`, `prod` as
    the three first-class environments and explicitly defers
    overlay-tool choice to W3-P7.
- **Downstream open:**
  - **W3-P7c** scaffolding session (per-env overlays under
    `deploy/overlays/{local,qa,prod}/`) cannot begin until
    this study reaches `resolved-study`; W3-P7c's PR commits
    the actual `kustomization.yaml` files this study commits
    the tool choice for.
- **Promotion target:** see final section.

---

## Context

`deploy/base/` (W3-P7b) ships four plain Kubernetes YAML
manifests — `Deployment`, `Service`, `ConfigMap`,
`ServiceAccount`. The base is tool-neutral; **B2-8 picks the
tool that wraps the base + the three per-env overlays
(`local`, `qa`, `prod`) the W3-P7c session will produce**.

The decision is narrow because the per-env delta is small:
- The `dq-engine-env` ConfigMap gains a `data:` block setting
  `DQ_ENV` per env (plus `STORAGE_EMULATOR_HOST` and
  `BIGQUERY_EMULATOR_HOST` for the `local` overlay only).
- The `dq-engine` ServiceAccount gains an annotation
  (`iam.gke.io/gcp-service-account: <env>@<env>.iam.gserviceaccount.com`)
  per env, per B1-4 MD-3.
- Optional per-env tuning of `imagePullPolicy`, `replicas`,
  resource requests / limits, and probe periods.

No per-env templating of values, no per-env conditional
logic, no shared chart distribution to external consumers.

### Out of scope

| Topic | Why deferred |
|---|---|
| **GCP project provisioning / cloud-infra IaC** (Terraform / Pulumi / equivalent) | A separate layer from app deployment. The operational session that provisions each env's GCP project (per B1-4 C-MD-3.3) handles cloud infra; B2-8 picks the app-deployment tool that runs *inside* the provisioned cluster. The two are independent — Terraform-or-equivalent at the infra layer remains possible regardless of B2-8's choice at the app-deployment layer. |
| **External Helm-chart distribution** (e.g., self-hosted DQ Platform consumers) | No such requirement exists at v1; the engine runs in the project's own clusters. If external consumption becomes a goal, a future amendment can layer a Helm chart on top of whichever overlay tool B2-8 picks. |
| **CI deploy pipeline** (apply-to-cluster on merge) | A separate future session (likely B2-3 release-engineering or a Phase-7 follow-up). B2-8 commits the *artifact* shape; whether CI applies it automatically is operational. |
| **Secrets-storage primitive** | A separate future B1 per B1-4's Out-of-scope list. Whichever overlay tool B2-8 picks must coexist with the eventual secrets layer (Kustomize / Helm / Terraform all support secret references). |
| **Multi-region / multi-cluster `prod`** | B1-4 OQ-MD-3.1 deferred v1 to one `prod` GCP project + cluster. B2-8 inherits that deferral. |

---

## Decision Drivers

D1. **Narrow per-env delta.** Per B1-4 MD-4 + W3-P7a, the only
runtime env-var surface is `DQ_ENV` + 2 emulator overrides;
the application config lives in the per-env Go file. The
overlay tool only needs to set 1–3 ConfigMap keys, 1 SA
annotation, and (optionally) tune replicas / resources. A
heavy templating engine is unjustified for this surface
area.

D2. **No external chart-distribution requirement.** The engine
is internal to this project; no Helm-style "ship a chart to
many consumers" use case. The deploy artifacts are
single-target (the project's own clusters).

D3. **`kubectl`-native CI integration.** The Wave-3 phases
already build on `kubectl`-style workflows (the
`deploy/README.md` "CI validity gate" deferral names
`kubectl apply --dry-run=client` as the eventual W3-P7c
validation lane). Picking a tool that **doesn't** require a
separate runtime binary keeps CI light.

D4. **P5 — contract-driven evolution.** The base manifests are
the contract. Evolution channels for per-env values should
be inspectable (the rendered YAML at any moment is a known
input); the tool should not interpose a templating engine
between source and rendered output if it can be avoided.

D5. **P4 — cost discipline.** Operational overhead of the
overlay tool (install, learn, integrate into CI) is a real
cost. The tool should solve the problem at hand without
inviting feature-set creep.

D6. **R5 — borrow patterns, not baggage.** Kustomize ships
inside `kubectl` (which is on CLAUDE.md R5's explicit
commodity-environment list as "Kubernetes"); Helm and
Terraform are similarly ubiquitous Kubernetes-ecosystem
tools that the decision-log B2-8 row itself names as the
candidate set. Naming them here is naming the commodity
substrate the platform runs against, not borrowing a
prior-art system held up as the source of an idea. **New
contribution proposed here, requires review** — this study
asserts the R5 commodity-environment exemption extends to
the K8s-ecosystem overlay-tool candidates beyond the
specific list CLAUDE.md enumerates; a future amendment can
narrow or widen the exemption.

D7. **Doesn't foreclose future tool mix.** Whichever tool
B2-8 picks must coexist with (a) a future infra-layer
Terraform/Pulumi if it lands, (b) a future Helm chart if
external distribution becomes a requirement, (c) per-env
operational customizations the operational session adds at
provisioning time. The chosen tool must compose.

---

## Considered Options

- **(A) Kustomize.** Built into `kubectl` since 1.14
  (`kubectl apply -k`). No separate binary, no templating.
  Overlays are structural patches (strategicMergePatch or
  JSON patch) against the base. The rendered YAML at any
  moment is exactly base + patch — inspectable and
  contract-shaped. **Recommended.**
- **(B) Helm.** Chart templating with `values.yaml`. Strong
  for shared charts distributed to many consumers (the
  `helm install` model); manages release lifecycle from a
  client-local architecture. The templating engine extends
  Go templates with a custom template-helper function set;
  runtime interpolation bugs are possible and the rendered
  output depends on the values.yaml + template combination.
- **(C) Terraform with the `kubernetes` provider.** IaC
  managing Kubernetes resources via the GCP Kubernetes
  provider. Good for "the whole infrastructure stack as one
  artifact" but wrong layer for app-deployment-only.
- **(D) Combination.** Terraform at the infra layer
  (GCP project, GKE cluster, IAM); Kustomize or Helm at the
  app layer. The "what app-deployment tool?" sub-question
  inside (D) reduces to (A) vs (B), so (D) is not a distinct
  option for B2-8.

---

## Recommendation

**(A) Kustomize.** Grounded in D1–D7:

- D1 + D5 — the per-env delta is small enough that strategic
  merge patches handle it cleanly; no templating overhead is
  warranted.
- D2 — no external-consumer use case, so Helm's chart-
  distribution strengths don't apply here.
- D3 — `kubectl apply -k deploy/overlays/<env>/` is the entire
  CI surface; no separate tool install in CI.
- D4 — the rendered YAML is base + patch with no templating
  step; a future maintainer can `kustomize build
  deploy/overlays/local/` and the output is fully inspectable
  alongside the base.
- D6 — Kustomize is commodity tooling; naming it is
  R5-exempt.
- D7 — Kustomize composes:
  - A future Helm chart can wrap a Kustomize output via
    `helm template | kubectl apply -f -` patterns.
  - Terraform at the infra layer is independent (different
    artifact; different operational session).
  - Per-env operational customizations (e.g., Workload Identity
    annotation values) are simple patch additions in the env's
    overlay directory.

The W3-P7b base manifests are already shaped for a Kustomize
base: each resource has the `app.kubernetes.io/name` +
`app.kubernetes.io/component` labels needed for selectors;
the `dq-engine-env` ConfigMap has no `data:` block precisely
so an overlay can `replace` or `merge` data into it.

### Considered and rejected

- **(B) Helm** — chart templating is overkill for the small
  per-env delta. Helm's release-lifecycle features
  (`helm install` / `helm upgrade --atomic` / `helm
  rollback`) are valuable in some operational contexts, but
  the same outcomes are achievable with `kubectl apply -k`
  plus a declarative-Git-as-source-of-truth operational
  discipline. The templating engine adds a fragility surface
  (template-evaluation bugs, helper-function edge cases)
  that the Kustomize "what you write is what you get" model
  avoids.
- **(C) Terraform** — wrong layer. Terraform for cloud-infra
  provisioning (GCP project, GKE cluster, IAM bindings)
  remains a viable infra-layer tool independent of B2-8. At
  the app-deployment layer, the Terraform `kubernetes`
  provider is uncommon outside infra-tooling-heavy teams,
  adds plan/apply state-file ceremony to a domain (Kubernetes
  manifests) that doesn't benefit from declarative state
  tracking the same way cloud resources do.
- **(D) Combination** — premature for B2-8's scope. The
  question reduces to picking the app-deployment tool, which
  is (A) vs (B). A future combination (Terraform for infra +
  Kustomize for app) is not foreclosed by (A); the two are
  orthogonal artifacts at different layers.

---

## Consequences

- **CC1.** W3-P7c scaffolds Kustomize artifacts: a
  `deploy/base/kustomization.yaml` lists the W3-P7b base
  resources, and a `deploy/overlays/{local,qa,prod}/kustomization.yaml`
  per env carries the patches. The patches set `DQ_ENV` (and
  for `local`, the emulator-host overrides) on the
  `dq-engine-env` ConfigMap and annotate the `dq-engine`
  ServiceAccount with the per-env GCP service-account email
  per B1-4 MD-3.

- **CC2.** CI validity-gate lane lands with W3-P7c: a
  `make validate-deploy` Makefile target (or equivalent CI
  step) runs `kubectl apply -k deploy/overlays/<env>/
  --dry-run=client` against each of the three overlays.
  No separate Kustomize binary is required since `kubectl -k`
  is built into kubectl 1.14+.

- **CC3.** The base manifests are not modified by B2-8 itself;
  W3-P7c adds the `kustomization.yaml` files that reference
  them. The W3-P7b base posture (plain YAML, tool-neutral)
  is preserved — a future amendment that picks a different
  app-deployment tool can still consume the base via
  `kubectl apply -f` or `helm template` references.

- **CC4.** **No infrastructure-layer tool is committed by
  B2-8.** Per the Out-of-scope section, GCP project
  provisioning and similar cloud-infra concerns are
  operational-session territory. The operational session is
  free to pick Terraform, Pulumi, or another IaC tool at
  that layer; B2-8 does not foreclose any choice there.

- **CC5.** **No Helm chart for external distribution is
  committed.** If a future requirement to ship the platform
  to external consumers emerges, a follow-up amendment can
  layer a Helm chart on top of the Kustomize-rendered
  output (the `helm template` wrapper pattern); this is
  forward-compatible and does not require unwinding B2-8's
  Kustomize commitment.

- **CC6.** **Secrets** (KMS / Vault / sealed-secrets) remain
  a separate future B1 per B1-4's Out-of-scope list.
  Whichever secret-storage primitive lands must reference
  Kubernetes-native Secret objects that the Kustomize overlay
  can reference (e.g., via `secretGenerator` for synthesized
  secrets or plain Secret references for externally-managed
  ones). Kustomize accommodates all three patterns.

- **CC7.** **`make validate-deploy`** Makefile target is the
  W3-P7c artifact for AC-W3-7's "local build / lint / test
  gate" coverage of the deploy layer. The target wraps
  `kubectl apply -k --dry-run=client` per overlay.

---

## Open Questions

- **OQ-1.** Whether to use Kustomize's
  [`components`](https://kubectl.docs.kubernetes.io/guides/config_management/components/)
  feature for shared cross-overlay logic (e.g., a "uses local
  Compose emulators" component shared between `local` and a
  future "local-test" overlay). **Out-of-scope for current
  cycle — W3-P7c picks the minimum-viable structure (three
  flat overlays referencing one base); components are a
  later optimization if cross-overlay duplication becomes a
  real problem.**

- **OQ-2.** Whether to ship the engine as a Helm chart for
  external consumption. **Out-of-scope for current cycle —
  no external-consumer requirement at v1 (per the Context
  section). Re-visit only if and when a real external
  consumer emerges; the answer at that time is "wrap the
  Kustomize render in a Helm chart" — a forward-compatible
  layer that doesn't unwind B2-8.**

- **OQ-3.** Whether to add a CI workflow that applies the
  rendered manifests against a real (or kind-based) cluster
  on PR merge. **Out-of-scope for current cycle — that is
  release-engineering territory (B2-3). W3-P7c's validity
  gate is `--dry-run=client`, which is sufficient to catch
  syntax / schema issues without standing up a cluster in
  CI.**

- **OQ-4.** Concrete `kubectl` version pinned in CI for the
  `--dry-run` lane. **Out-of-scope for current cycle —
  W3-P7c picks at the moment the CI lane lands; the
  default-latest stance is acceptable for v1 since the
  manifests use core Kubernetes APIs (`apps/v1`, `v1`) that
  haven't moved in years.** (new contribution proposed here,
  requires review — no prior ADR commits the
  default-latest CI posture; W3-P7c session is free to
  override.)

---

## Promotion target

`docs/adr/0017-infrastructure-tooling.md` (provisional).

Last accepted ADR at the time of writing is **ADR-0014**
(W3-P4e trigger-handler-contract from PR #9). Three other
B1 / B2 studies are currently at `resolved-study` awaiting
promotion:

- **B1-10** (workspace tooling) — provisional slot in its own
  study was `0014`, now stale.
- **B1-11** (substrate-posture amendment) — provisional slot
  `0015`, still available.
- **B1-4** (environment configuration model) — provisional
  slot `0016`, still available.

B2-8 takes provisional slot `0017` to leave `0015` and
`0016` for the two B1 studies whose provisional metadata
already reserves them. The promotion session that runs first
picks the actual next free slot at that time; the
provisional slot is a citation hint for cross-study
coherence, not a hard reservation.

On promotion, the ADR carries the single Recommendation as
the Decision section; the per-driver Consequences are
renumbered into a flat ADR-level Consequence list as per the
precedent for the prior promoted ADRs.
