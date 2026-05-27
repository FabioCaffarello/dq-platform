<!-- path: docs/architecture/adr-reading-order.md -->

# ADR Reading Order

Forty-seven ADRs are a lot to read cold. This page picks the
**seven** that anchor mental model — read these first — and groups
the rest by cluster so you can dive in when a specific question
lands.

---

## Read these seven first

Recommended first-read order; not strictly numerical. Together they
cover entry, identity, outcome, egress, the boundary contract, and
the mode primitive.

1. **[ADR-0001 — Engine ↔ Rules Compatibility](../adr/0001-engine-rules-compatibility.md)** — the boundary contract; without this nothing else has a stable referent.
2. **[ADR-0002 — Run Identity and Idempotency](../adr/0002-run-identity-and-idempotency.md)** — defines `execution_id`; every other concept keys on it.
3. **[ADR-0004 — Failure Scope](../adr/0004-failure-scope.md)** — the outcome enum (`pass / fail / error / aborted`); reporting and alerting both speak it.
4. **[ADR-0006 — Alert Routing Contract](../adr/0006-alert-routing-contract.md)** — egress: `_owners.yaml` + Pub/Sub + dedup; closes the loop to human action.
5. **[ADR-0014 — HTTP Trigger Handler Contract](../adr/0014-trigger-handler-contract.md)** — entry: `/v1/trigger`; where a run begins.
6. **[ADR-0021 — Mode as Primitive](../adr/0021-mode-as-primitive.md)** — `mode: set | record` as the architectural discriminator; the fork.
7. **[ADR-0023 — Sources Schema](../adr/0023-sources-schema.md)** — substrate selection: BigQuery for set, **Kafka** for record; closes the partial-Wave-S gate.

After these seven, the architecture should feel coherent end to end.

---

## Then by cluster (depth-on-demand)

Read these when a question lands in the cluster's territory.

### Manifest semantics
- [ADR-0005 — Manifest Publication Semantics](../adr/0005-manifest-publication-semantics.md)
- [ADR-0030 — Manifest Cryptographic Posture](../adr/0030-manifest-cryptographic-posture.md)
- [ADR-0036 — `set-pointer` Rollback Subcommand](../adr/0036-set-pointer-subcommand.md)

### Result write model
- [ADR-0003 — Result Write Model](../adr/0003-result-write-model.md)
- [ADR-0031 — Evidence Retention Parameters](../adr/0031-evidence-retention-parameters.md)
- [ADR-0041 — Stream Reporting Continuity](../adr/0041-stream-reporting-continuity.md)

### Record-mode mechanics (the Wave-S Phase β cluster)
- [ADR-0020 — Wave-S Launch](../adr/0020-wave-s-launch.md)
- [ADR-0022 — Kind Catalog](../adr/0022-kind-catalog.md)
- [ADR-0024 — Window Semantics](../adr/0024-window-semantics.md)
- [ADR-0025 — Aggregation and Runner Shape](../adr/0025-aggregation-and-runner-shape.md)
- [ADR-0026 — Failure Scope Aggregated](../adr/0026-failure-scope-aggregated.md)
- [ADR-0027 — Record-Mode Cost Guardrails](../adr/0027-record-mode-cost-guardrails.md)
- [ADR-0028 — Kafka Substrate Row](../adr/0028-kafka-substrate-row.md)

### Substrate posture
- [ADR-0010 — Substrate Posture (Local Compose Scope)](../adr/0010-substrate-posture.md)
- [ADR-0017 — Substrate Posture Amendment (CAS)](../adr/0017-substrate-posture-amendment.md)
- [ADR-0018 — Environment Configuration Model](../adr/0018-environment-configuration-model.md)
- [ADR-0029 — BigQuery Cost Ceilings](../adr/0029-bigquery-cost-ceilings.md)

### Schema evolution and the DSL surface
- [ADR-0035 — Compatibility Window Duration](../adr/0035-compatibility-window-duration.md)
- [ADR-0044 — External Artifact References in DSL](../adr/0044-external-artifact-references.md)

### Governance and ownership
- [ADR-0015 — CODEOWNERS Review-Ownership Map](../adr/0015-codeowners.md)
- [ADR-0037 — Owner ↔ CODEOWNERS Linter Cross-Check](../adr/0037-owner-codeowners-cross-check.md)
- [ADR-0040 — Entity Onboarding Workflow](../adr/0040-entity-onboarding-workflow.md)
- [ADR-0046 — Onboarding-Channel Override](../adr/0046-onboarding-channel-override.md)

### Tooling and operations
- [ADR-0007 — Loader / Scheduler / Retry Failure Semantics](../adr/0007-loader-scheduler-retry-failure-semantics.md)
- [ADR-0033 — Scheduler Catchup Behavior](../adr/0033-scheduler-catchup-behavior.md)
- [ADR-0034 — Local Testing Strategy](../adr/0034-local-testing-strategy.md)
- [ADR-0043 — Logging Contract Specifics](../adr/0043-logging-contract-specifics.md)
- [ADR-0047 — `dq-lint` Substrate-Access Posture](../adr/0047-lint-substrate-access.md)

### Dashboards and observability
- [ADR-0039 — Dashboard Contract](../adr/0039-dashboard-contract.md)
- [ADR-0045 — Baseline Dashboard Substrate](../adr/0045-baseline-dashboard-substrate.md)

### Build, release, and CI
- [ADR-0008 — Git Host](../adr/0008-git-host.md)
- [ADR-0012 — Per-Workspace Tag Conventions](../adr/0012-tag-conventions.md)
- [ADR-0013 — Wave 3 Scaffolding Sequencing](../adr/0013-wave3-sequencing.md)
- [ADR-0016 — Workspace Tooling (`go.work`)](../adr/0016-workspace-tooling.md)
- [ADR-0019 — Kustomize for Per-Env Overlays](../adr/0019-infrastructure-tooling.md)
- [ADR-0042 — Release Engineering Invariants](../adr/0042-release-engineering-invariants.md)

### Process and AI collaboration
- [ADR-0009 — Multi-Agent Contract](../adr/0009-multi-agent-contract.md)
- [ADR-0011 — Documentation Language](../adr/0011-documentation-language.md)
- [ADR-0032 — Baseline Strategy](../adr/0032-baseline-strategy.md)
- [ADR-0038 — Documentation Site Generator](../adr/0038-documentation-site-generator.md)

---

## Suggested reading sequence

(1) Skim this directory end to end (~30 min, path in the
[README](./README.md)). (2) Read the seven anchor ADRs above in the
order listed. (3) Pick the cluster matching your question and read it
in numerical order. (4) For timeline ("why this, why now"), read the
**Date** metadata on each ADR — the accept-order is the order each
piece of mental model was committed.

The full ADR index is under [`../adr/`](../adr/). Every ADR not
listed above is either a refinement of one in this catalog or
governs a sub-system the seven anchors do not surface.
