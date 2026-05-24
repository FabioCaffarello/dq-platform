<!-- path: studies/foundation/06-decision-log.md -->

# 06 — Decision Log

## Metadata

- Purpose: track every platform decision that must be resolved before
  implementation. Each row notes priority, current status, the
  rationale for why it matters, and the location of the resolving
  document.
- Audience: project lead, platform engineers, anyone planning the next
  session of work.
- Status: living document. Update whenever a decision changes state.
- Last updated: 2026-05-23 (W3-P8d closed — Phase 8 closes; B2-9/B2-10 registered from W3-P8b/P8d follow-ups; B1-9 → ADR-0015, B1-10 → ADR-0016, B1-11 → ADR-0017, B1-4 → ADR-0018, B2-8 → ADR-0019; Wave 3 completion gate met)
- Wave-S preparation: 2026-05-23 scope notes applied to ADRs 0002, 0003, 0004, 0006, 0007, 0010, 0014, 0017 declaring set-oriented mode; ADR-0020 (Wave-S launch) lands 2026-05-23 — forward-pointers redeemed.
- Wave-S launch: 2026-05-23 launch study closed at resolved-study (two critique rounds; ref: `studies/decisions/2026-05-23-wave-s-launch.md`) and promoted to [ADR-0020](../../docs/adr/0020-wave-s-launch.md); B0-S1…B0-S7 registered in the Wave-S table below.
- Promotion target: this document stays in `studies/foundation/` for
  the project's lifetime. Resolved decisions are promoted to ADRs
  under `docs/adr/` during Wave 3; rows here keep the link.

---

## Prioritization Model

- **B0** — blocking. Must be resolved before Wave 3 (scaffolding) can
  start. Each B0 gets its own dated decision document in
  `studies/decisions/`.
- **B1** — important. Should be resolved before serious implementation
  starts. Some are refinements of B0 outcomes.
- **B2** — later. Can be resolved as implementation reveals concrete
  needs. Documented here so they are not forgotten.

## Status Vocabulary

- **open** — no work has started.
- **in-progress** — a draft exists in `studies/decisions/` but is not
  yet finalized.
- **resolved-study** — a complete study exists in
  `studies/decisions/`; ready for promotion when Wave 3 starts.
- **resolved-adr** — the study has been promoted to an ADR under
  `docs/adr/`.

---

## B0 — Blocking Decisions (Wave 1 Scope)

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B0-1 | Engine ↔ rules compatibility | [resolved-study](../decisions/2026-05-20-engine-rules-compatibility.md) → [resolved-adr](../../docs/adr/0001-engine-rules-compatibility.md) | How does the rules workspace declare which schema and linter contract it follows, given both live in the same repository? | Without an explicit contract, the monorepo lets boundaries erode silently. | ADR + boundary contract refinement |
| B0-2 | Run identity and idempotency | [resolved-study](../decisions/2026-05-20-run-identity-and-idempotency.md) → [resolved-adr](../../docs/adr/0002-run-identity-and-idempotency.md) | What uniquely defines a run, and how do reruns of the same window behave? | Reporting trust and alert deduplication depend on it. | Execution semantics ADR |
| B0-3 | Result write model | [resolved-study](../decisions/2026-05-20-result-write-model.md) → [resolved-adr](../../docs/adr/0003-result-write-model.md) | Are `dq_executions` and `dq_check_results` append-only, upserted, or hybrid? | Impacts retries, lineage, and dashboard accuracy. | Storage design ADR |
| B0-4 | Failure scope | [resolved-study](../decisions/2026-05-20-failure-scope.md) → [resolved-adr](../../docs/adr/0004-failure-scope.md) | When one check errors operationally, does the entity error, degrade, or partially complete? | Incidents and alerting become inconsistent without a single policy. | Failure-semantics ADR plus runbook |
| B0-5 | Manifest publication semantics | [resolved-study](../decisions/2026-05-20-manifest-publication-semantics.md) → [resolved-adr](../../docs/adr/0005-manifest-publication-semantics.md) | What guarantees atomic, reversible ruleset publication to object storage? | Runtime safety depends on manifest discipline. | Control-plane contract ADR |
| B0-6 | Alert routing contract | [resolved-study](../decisions/2026-05-21-alert-routing-contract.md) → [resolved-adr](../../docs/adr/0006-alert-routing-contract.md) | What fields live in `_owners.yaml`, what stays in engine config, and what is deduplicated on the data itself? | Prevents alerting logic from becoming hardcoded chaos. | Governance doc + owners schema |
| B0-7 | Loader / scheduler / retry failure semantics | [resolved-study](../decisions/2026-05-20-loader-scheduler-retry-failure-semantics.md) → [resolved-adr](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md) | What exactly causes ruleset load failure, scheduler reconciliation failure, and retry budget exhaustion? | The fail-fast registry pattern only helps if failure semantics are explicit and consistent. | Loader and scheduler ADRs |

---

## B1 — Important Decisions (Pre-Implementation Scope)

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B1-1 | Baseline strategy | open | Where do moving averages and historical references come from, and what happens with sparse history? | Volume and freshness checks depend on consistent history semantics. | Check design note |
| B1-2 | BigQuery cost ceilings | open | What are the per-environment limits for window size, concurrency, failed samples, and dry-run enforcement? | Cost drift is predictable; designing around it is cheap. | Operations doc + defaults policy |
| B1-3 | Scheduler catchup behavior | open | How are catchup, missed windows, and manual triggers represented? | A scheduler without precise semantics causes duplicate or missing evaluations. | Scheduling design note |
| B1-4 | Environment configuration model | [resolved-study](../decisions/2026-05-22-b1-4-environment-configuration-model.md) → [resolved-adr](../../docs/adr/0018-environment-configuration-model.md) | Which configuration lives in code, deployment, or data, and how are `local`, `qa`, and `prod` isolated? | Prevents configuration sprawl and implicit behavior drift. | Env strategy ADR |
| B1-5 | Local testing strategy | open | What can be tested offline, what needs BigQuery sandbox access, and how is generated SQL inspected? | Developer experience shapes long-term quality. | Dev guide + tooling scope |
| B1-6 | Evidence retention parameters | open | How many violating samples per check, for how long, under what privacy constraints? | Storage cost and privacy compliance depend on it. | Storage and security note |
| B1-7 | Compatibility window duration | open | How long is each schema version supported after its successor is released? | Migration ergonomics for domain teams. | Boundary contract refinement |
| B1-8 | Manifest cryptographic posture | open | Does the manifest carry signatures beyond checksums? Who signs it? | Defense in depth against tampering or accidental publication. | Security note |
| B1-9 | CODEOWNERS finalization | [resolved-study](../decisions/2026-05-22-b1-9-codeowners.md) → [resolved-adr](../../docs/adr/0015-codeowners.md) | Final team names and path rules for the asymmetric review model. | Enforces the boundary at PR-review time. | CODEOWNERS file |
| B1-10 | Workspace tooling stack | [resolved-study](../decisions/2026-05-21-b1-10-workspace-tooling.md) → [resolved-adr](../../docs/adr/0016-workspace-tooling.md) | Confirm Go workspaces (`go.work`) as the tooling choice and finalize the per-tool module structure. | Affects every CI pipeline file. | Topology ADR refinement |
| B1-11 | ADR-0010 substrate-posture amendment — object-store CAS row | [resolved-study](../decisions/2026-05-21-b1-11-substrate-posture-amendment.md) → [resolved-adr](../../docs/adr/0017-substrate-posture-amendment.md) | The ADR-0010 "Object store: generation-conditional pointer write" row is committed as `Yes`, but Phase 2 emulator evaluation found commodity emulators do not faithfully enforce `ifGenerationMatch` (fake-gcs-server accepts stale-generation writes; storage-testbench requires GCP auth; oittaa lacks the media-upload endpoint). The row should be amended to `Partial` so production-shape CAS enforcement is explicitly sandbox-required, matching the existing pattern for the tabular-store lazy-view row. | Without amendment, the ADR-0010 contract and the deployed `docker-compose.yml` disagree on whether local CAS is faithful. | Amendment ADR (or a Wave 3 follow-up adjusting ADR-0010's matrix row). |

---

## B2 — Later Decisions (Implementation-Phase Scope)

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B2-1 | External artifact references in DSL | open | Will the DSL ever allow helper files (auxiliary SQL, reference payloads) resolved relative to the rule origin? | The capability is useful but dangerous if it becomes an escape hatch. | DSL evolution ADR |
| B2-2 | Logging contract specifics | open | Which package names and override syntax are officially supported by `DQ_LOG_LEVELS`? | Good observability patterns help only if standardized. | Operations doc |
| B2-3 | Release engineering invariants | open | Which Docker, Make, and versioning invariants are mandatory across all workspaces? | Long-term repo health depends on consistent ergonomics. | Release engineering doc |
| B2-4 | Stream reporting continuity | open | How will stream-runner results align with batch result tables and identifiers? | Future migration should not fracture observability. | Stream evolution design note |
| B2-5 | Entity onboarding workflow | open | What exact checklist determines when a new entity is ready for test channel and later for production alerting? | Governance quality depends on repeatable onboarding. | Runbook + checklist |
| B2-6 | Dashboard contract | open | Which metrics and dimensions are guaranteed for downstream consumers (Looker, Grafana, etc.)? | Avoids each consumer inventing its own interpretation. | Reporting contract |
| B2-7 | Documentation site generator | open | Does `docs/` get a static site generator, or stay as raw markdown? | Affects how documentation is discovered by non-developers. | Documentation infrastructure note |
| B2-8 | Infrastructure tooling | [resolved-study](../decisions/2026-05-22-b2-8-infrastructure-tooling.md) → [resolved-adr](../../docs/adr/0019-infrastructure-tooling.md) | Kustomize, Helm, Terraform, or a combination for `deploy/`? | Affects deployment ergonomics and environment isolation. | Infrastructure ADR |
| B2-9 | Owner ↔ CODEOWNERS-group linter cross-check | open | Should `dq-lint` parse `.github/CODEOWNERS` and reject `_owners.yaml` entries whose `owner:` does not correspond to an existing CODEOWNERS group? | ADR-0006 §9 commits the linter as the first enforcement point for "no alert without owner"; without the cross-check, a stale or typo'd group reference only fails at PR-review time. Defense-in-depth complement to OQ-B1-9.3 (publisher/loader-side validation). Flagged when W3-P8b closed. | Linter rule design note + `dq-lint` extension |
| B2-10 | `dq-manifest set-pointer` rollback subcommand | open | Should `dq-manifest` expose a first-class `set-pointer <hash>` subcommand to execute the rollback procedure in `docs/runbooks/manifest-rollback.md`? Today the CLI exposes only `publish`, and operators fall back to `gsutil`/console writes that bypass CLI contract enforcement. | Rollback ergonomics during incident response; the runbook's §2 procedure is TBD-blocked on this CLI surface. Flagged when W3-P8d closed. | CLI design note + subcommand under `tools/manifest/` |

---

## Wave 2 — Platform Decisions (Single Consolidated Document)

These are not in the priority backlog because they are a separate
wave with a single decision document covering all of them. They are
listed here so the log is complete.

| # | Topic | Status |
|---|---|---|
| W2-1 | Git host choice (affects CI artifact location and syntax) | [resolved-study](../decisions/2026-05-21-platform-decisions-wave2.md) → [resolved-adr](../../docs/adr/0008-git-host.md) |
| W2-2 | Multi-agent contract — finalize `.claude/`, `.codex/`, `AGENTS.md` | [resolved-study](../decisions/2026-05-21-platform-decisions-wave2.md) → [resolved-adr](../../docs/adr/0009-multi-agent-contract.md) |
| W2-3 | Docker Compose local scope — which services emulated, which sandboxed | [resolved-study](../decisions/2026-05-21-platform-decisions-wave2.md) → [resolved-adr](../../docs/adr/0010-substrate-posture.md) |
| W2-4 | Documentation language policy (English / Portuguese / mixed) | [resolved-study](../decisions/2026-05-21-platform-decisions-wave2.md) → [resolved-adr](../../docs/adr/0011-documentation-language.md) |
| W2-5 | Per-workspace tag prefix conventions (confirm or revise) | [resolved-study](../decisions/2026-05-21-platform-decisions-wave2.md) → [resolved-adr](../../docs/adr/0012-tag-conventions.md) |

---

## Wave 3 — Phases (Scaffolding Sequencing)

Phase structure committed in
[`studies/decisions/2026-05-21-wave3-sequencing.md`](../decisions/2026-05-21-wave3-sequencing.md).
Each phase's sessions run under
[`.claude/playbooks/wave-3-session-loop.md`](../../.claude/playbooks/wave-3-session-loop.md)
and self-verify against
[`.claude/playbooks/wave-3-acceptance-criteria.md`](../../.claude/playbooks/wave-3-acceptance-criteria.md).
Each row resolves to the commit reference that closes the phase.

| # | Phase | Status |
|---|---|---|
| W3-P0 | Protocol — sequencing study at [`studies/decisions/2026-05-21-wave3-sequencing.md`](../decisions/2026-05-21-wave3-sequencing.md) → [`docs/adr/0013-wave3-sequencing.md`](../../docs/adr/0013-wave3-sequencing.md) + Wave 3 playbook + Wave 3 acceptance criteria | closed (commit `25e06ab`, 2026-05-21) |
| W3-P1 | ADR promotion — twelve studies (B0-1…B0-7, W2-1…W2-5) plus this sequencing study, into [`docs/adr/0001–0013`](../../docs/adr/) | closed (Session A `c55799c`, Session B `e411fdb`, Session C lands with this commit; all thirteen ADRs exist) |
| W3-P2 | Root infrastructure — `go.work`, `Makefile`, `docker-compose.yml`, `.github/`, `.codex/AGENTS.md`, top-level `README.md`, empty workspace layout | closed (commit lands with this session; depends on B1-10 resolved-study upstream) |
| W3-P3 | Schema-layer — engine schema source, rules schema mirror, `tools/lint` byte-equality gate (B0-1 C2 / C4 / C10) | closed (commit lands with this session; depends on B1-10 resolved-study upstream) |
| W3-P4 | Engine runtime — loader (B0-7), runner (B0-2), result write (B0-3), failure scope (B0-4), orphan detection (B0-7), HTTP trigger handler (ADR-0002 + W3-P4e contract study) | split into W3-P4a / W3-P4b / W3-P4c / W3-P4d (2026-05-21, resolves OQ-W3-2; see [`studies/decisions/2026-05-21-wave3-sequencing.md`](../decisions/2026-05-21-wave3-sequencing.md) §"Phase 4") + W3-P4e (2026-05-22, renumbered from W3-P6b) |
| W3-P4a | Loader — manifest fetch, hash short-circuit, refuse-swap (B0-7 CC1/CC2/CC9; ADR-0005 §4 consumer side) | closed (commit lands with this session; unit + integration tests against the local Compose stack pass) |
| W3-P4b | Result-write layer — append-only `dq_executions` + `dq_check_results` + lazy `dq_executions_current` view (B0-3 CC1/CC2/CC3/CC7) | closed (lands via PR; unit + integration tests against local bigquery-emulator pass; lazy-view fidelity gap honored per ADR-0010 / B1-11 pattern) |
| W3-P4c | Runner + failure-scope mapping — `execution_id` computation (B0-2 CC1/CC2/CC6); status mapping (B0-4 CC1/CC2/CC3/CC4); pre-check validation (B0-7 CC8); observability emission (B0-7 CC14). Depends on W3-P4a and W3-P4b. | closed (lands via PR; engine binary `cmd/dq-engine` builds and runs the periodic loops; unit + integration tests pass; OTel metric+span pipeline deferred — log signal delivered, metric+span land in a follow-up) |
| W3-P4d | Orphan-run detection — periodic scan, follow-up `aborted` row with orphan-detector's `engine_version` (B0-7 CC10/CC11). Depends on W3-P4b. | closed (lands via PR; unit + integration tests pass; partial-failure tolerance + detector-engine-version invariant exercised) |
| W3-P4e | HTTP trigger handler — exposes POST `/v1/trigger`, dispatches to runner per ADR-0002 with the contract committed by ADR-0014. Depends on W3-P4a/b/c/d. Renumbered from W3-P6b on 2026-05-22. | [resolved-study](../decisions/2026-05-22-trigger-handler-contract.md) → [resolved-adr](../../docs/adr/0014-trigger-handler-contract.md) → closed (lands via PR; `engine/internal/api` HTTP handler with strict decoder, async runner dispatch, `/healthz` + `/readyz`; runner enhanced with optional `AttemptID` and `RulesetVersion` overrides for ADR-0007 §3 in-flight isolation; unit + integration tests pass; AC-W3-1..10 verified) |
| W3-P5 | Alerting — Pub/Sub publisher (B0-6), engine-side dedup, `_owners.yaml` schema, linter rule | closed (lands via PR; `engine/internal/alerts` implements `MapCategory` per ADR-0006 CC7, per-attempt `AttemptDeduper` per CC5, JSON Event payload per §4, Pub/Sub v2 publisher; runner + orphan detector wire emissions; engine binary creates the publisher from `DQ_PUBSUB_TOPIC` env; `_owners.v1.schema.json` lands in `rules/_schema/`; `dq-lint` rejects rules whose entity has no `_owners.yaml` entry per CC9; channel encoding committed as `<type>:<id>` per CC2's Wave 3 deferral; unit + integration tests pass) |
| W3-P6 | `rules/` first onboarded entity — end-to-end flow per W2-3 C-W2-3.4 | split into W3-P6a / W3-P6c / W3-P6d (2026-05-21, amended 2026-05-22 — HTTP trigger handler moved to W3-P4e; remaining sub-phases: manifest publisher, first DSL kind interpreter, end-to-end demo, each getting an independent critique pass and PR) |
| W3-P6a | Manifest publisher tool — `tools/manifest/` implements ADR-0005 §4 verify-write-CAS sequence end-to-end | closed (lands via PR; new Go module + `dq-manifest` CLI binary; in-mem Store fake covers CAS race-loser branch; integration test against fake-gcs-server covers happy path + idempotent re-publish + dry-run; CLI exit codes 0/1/2/3/64 documented; B1-11 CAS fidelity gap honored) |
| W3-P6c | First DSL kind interpreter — real `CheckEvaluator` against BigQuery for at least one check kind. Depends on W3-P4c. | closed (lands via PR; `engine/internal/eval` package with `row_count_positive` kind dispatched on `CheckSpec.Kind`; entity identifier defensively validated in the evaluator; engine binary wires the BigQuery-backed evaluator into the runner via new `DQ_SOURCE_PROJECT` / `DQ_SOURCE_DATASET` env vars; unit + integration tests + a runner-eval wire-level integration test pass; runner-overwrites-EvidenceSummary-on-error gap documented in row_count_positive.go and deferred to a follow-up session) |
| W3-P6d | First entity rule YAML + first `_owners.yaml` entry + end-to-end demo target. Depends on W3-P6a, W3-P6c, and W3-P4e. Closes the C-W2-3.4 invariant locally. | closed (lands via PR; first production `rules/customer.yaml` + `rules/_owners.yaml`; new `engine/internal/dsl/spec` strict YAML parser; HTTP handler wires `ResolveChecks` closure that reads the rule YAML body from the object store at trigger acceptance per ADR-0007 §3; `make demo-p6` + `scripts/smoke/demo-p6.sh` exercises manifest publish → loader refresh-short-circuit → execution write → operational-alert capability locally; Go integration test `TestIntegration_DemoP6_EndToEnd` exercises the same flow programmatically; `tools/lint` walker extended to skip underscore-prefixed files so `_owners.yaml` isn't double-validated against the rule schema; unit + integration + demo all green) |
| W3-P7 | `deploy/` — Kubernetes manifests, environment overlays; depends on B1-4 | split into W3-P7a / W3-P7b / W3-P7c (2026-05-22; the first Phase-7 session decided to split; mirrors the W3-P4 split precedent — PAT-4 refactor, deploy base, per-env overlays each get an independent critique pass and PR) |
| W3-P7a | PAT-4 refactor — `engine/internal/env/{local,qa,prod}.go` per foundation 04 §PAT-4 and B1-4 MD-4; engine binary reads `DQ_ENV` selector; reflect-based CI test enforces every-field-in-every-env rule. Depends on B1-4. | closed (lands via PR; new `engine/internal/env` package with `EnvConfig` typed struct + `Local`/`QA`/`Prod` vars; engine binary's `readEnv()` reduces from 13 env vars to 1 selector + 2 emulator overrides; `scripts/smoke/demo-p6.sh` updated to use `DQ_ENV=local`; demo + integration tests + lint all green; `qa.go` / `prod.go` carry `PLACEHOLDER`-marked values to be replaced by the operational session that provisions the real GCP projects) |
| W3-P7b | Deploy base — Kubernetes manifests for the engine in `deploy/base/`. Depends on W3-P7a. | closed (lands via PR; plain Kubernetes YAML — Deployment + Service + ConfigMap + ServiceAccount — tool-neutral so the overlay tool decision stays with W3-P7c per B2-8; probes wired to `/healthz` + `/readyz` per ADR-0014 §4; port 8080 matches `env.Local.HTTPAddr`; image marked `dq-engine:placeholder` for the future release-engineering pipeline (B2-3); base ConfigMap has no data — overlays patch `DQ_ENV` per env plus emulator-host overrides for `local`; pod runs non-root with `readOnlyRootFilesystem`, dropped capabilities, and the `RuntimeDefault` seccomp profile) |
| W3-P7c | Per-env overlays — `deploy/overlays/{local,qa,prod}/` setting `DQ_ENV` and the emulator overrides per env. Depends on W3-P7a + W3-P7b. | closed (lands via PR; Kustomize per B2-8 — `deploy/base/kustomization.yaml` + three overlay directories under `deploy/overlays/{local,qa,prod}/`; each overlay strategic-merges `DQ_ENV` into the `dq-engine-env` ConfigMap; qa/prod overlays additionally annotate the `dq-engine` ServiceAccount with `iam.gke.io/gcp-service-account` per B1-4 MD-3 — values carry `dq-{qa,prod}-PLACEHOLDER` markers mirroring `engine/internal/env/{qa,prod}.go`; local overlay also sets `STORAGE_EMULATOR_HOST` + `BIGQUERY_EMULATOR_HOST` PLACEHOLDERs; new `make validate-deploy` target renders all three overlays via `kubectl kustomize` and CI workflow `.github/workflows/deploy-validate.yml` runs it on every PR; documented deviation from B2-8 CC2's literal `kubectl apply -k --dry-run=client` — that command still performs API-server discovery on a cluster-free runner; deeper schema validation (`kubeconform` or kind-based cluster lane) deferred to B2-3 or a Phase-7 follow-up; Phase 7 closes with this session) |
| W3-P8 | `docs/` content beyond ADRs — glossary, governance, contribution guide, runbook seeds | split into W3-P8a / W3-P8b / W3-P8c / W3-P8d (2026-05-22; mirrors the W3-P4 / W3-P6 / W3-P7 split precedent — each sub-phase gets an independent critique pass and PR) |
| W3-P8a | Glossary — `docs/glossary.md` collecting terms with codebase-specific meaning across the ADRs, foundation documents, and resolved decisions. Forward-only per R8; no back-links into `studies/` from new content. | closed (lands via PR; ~50 terms across eleven thematic groups — governance, schema boundary, run identity, storage, failure model, manifest plane, loader/runner, trigger plane, alerting, DSL, environment; each entry cites its defining ADR / foundation §) |
| W3-P8b | Governance — finalize CODEOWNERS for the asymmetric review model from ADR-0001 plus contribution-time roles; the study→ADR promotion flow already lives in `.claude/commands/promote-to-adr.md`. Depends on B1-9. | closed (lands via PR; `/.github/CODEOWNERS` published with `PLACEHOLDER-org/` literals per the path-rule table committed by B1-9; `docs/governance.md` lands as a forward-only referential summary of the review model and contribution-time flows; `rules/_owners.yaml`'s `customer` entity updated to `owner: "@PLACEHOLDER-org/rules-authors"` per B1-9 Consequence #4; substitution to the real org slug remains deferred to the operational session that creates the production org per ADR-0008 follow-up + OQ-B1-9.1) |
| W3-P8c | Contribution guide — how to add a rule, how to run `make demo-p6`, how to open a B-item, how a Wave 3 session loop closes. Depends on W3-P8a (terminology baseline). | closed (lands via PR; the stale Wave-1-only `/CONTRIBUTING.md` rewritten in place as the authoritative guide per GitHub convention; covers four practical flows — add a rule, run `make demo-p6`, open a B-item, close a Wave 3 session loop — plus a "what review will look like" pointer to `docs/governance.md`; `docs/README.md` updated to point at the root file; commit conventions section refreshed to mirror the post-Wave-1 `feat(...)`/`docs(...)` taxonomy) |
| W3-P8d | Runbook seeds — operator-facing playbooks for manifest rollback via CAS pointer write, orphan-run remediation, alert-dedup debugging, refresh-failure escalation. Depends on W3-P8a (terminology baseline). | closed (lands via PR; new `docs/runbooks/` directory with index + four runbooks, each following the fixed shape `when to use → preconditions → procedure → verification → rollback → escalation`; each runbook flags TBD markers where a B1 numeric parameter is unresolved (e.g., B1-2 refresh-failure thresholds) or where a CLI subcommand the procedure would prefer does not exist yet (e.g., `dq-manifest set-pointer`, blanket orphan-finalization tool); `docs/README.md` updated to advertise the runbooks directory; **Phase 8 closes with this session — all sub-phases W3-P8a / W3-P8b / W3-P8c / W3-P8d are now closed**) |

---

## Wave-S — Record-Oriented Capability Decisions

Wave-S launches record-oriented (stream-based) validation capability
parallel to the set-oriented capability delivered through Waves 1-3.
Launch study at
[`studies/decisions/2026-05-23-wave-s-launch.md`](../decisions/2026-05-23-wave-s-launch.md)
→ [ADR-0020](../../docs/adr/0020-wave-s-launch.md). Each B0-S item below opens
its own study under the `/resolve-b0` protocol that landed B0-1
through B0-7; sequencing and gate criteria are committed in the
launch study §6.3.

| # | Topic | Status | Key Question | Why It Matters | Expected Output |
|---|---|---|---|---|---|
| B0-S1 | Mode as primitive | [resolved-study](../decisions/2026-05-23-b0-s1-mode-as-primitive.md) → [resolved-adr](../../docs/adr/0021-mode-as-primitive.md) | How is `mode` declared on the rule artefact and entity, and how does capability derive from it (per P3)? | Mode is the architectural primitive (P1); without explicit declaration, downstream capability drifts and the kind-prefix lint gate cannot enforce the boundary. | ADR + mode-field schema + lint rule under `tools/lint/` |
| B0-S2 | Kind catalog | [resolved-study](../decisions/2026-05-24-b0-s2-kind-catalog.md) → [resolved-adr](../../docs/adr/0022-kind-catalog.md) | What is the registry of supported `set.*` and `record.*` kinds, and how is the catalog extended? | Without a catalog, source declarations (S3) cannot validate against a kind's expected shape; schema-version governance has nothing to bump. | ADR + `record.*` schema half under `engine/schema/` and `rules/_schema/` |
| B0-S3 | Sources schema | [resolved-study](../decisions/2026-05-24-b0-s3-sources-schema.md) → [resolved-adr](../../docs/adr/0023-sources-schema.md) | How is a source described per mode — set source (BigQuery table/view) vs record source (stream substrate topic/subscription)? | Sources cross-check against the kind catalog and extend the ADR-0007 loader; last item of the foundational triplet — its promotion meets the partial-Wave-S gate. | ADR + source schema + loader extension |
| B0-S4 | Window semantics | [resolved-study](../decisions/2026-05-24-b0-s4-window-semantics.md) → [resolved-adr](../../docs/adr/0024-window-semantics.md) | What does "window" mean for record-mode (tumbling, sliding, session, watermark-bounded, or per-record), and how do watermarks interact with the ADR-0002 `execution_id` formula? | Windowing reshapes the record-mode halves of ADR-0002 (identity), ADR-0003 (write model), and ADR-0006 (dedup). | ADR + record-mode `execution_id` rule |
| B0-S5 | Aggregation & unified-vs-parallel execution | [resolved-study](../decisions/2026-05-24-b0-s5-aggregation-and-runner-shape.md) → [resolved-adr](../../docs/adr/0025-aggregation-and-runner-shape.md) | One unified runner switching on mode, or two parallel runners — against what objective criterion (satisfies P4's deferral)? | Decides the engine binary layout under `engine/cmd/`; reshapes ADR-0007 (loader/scheduler) and ADR-0014 (trigger handler) on the record side. | ADR + objective criterion + runner-shape decision |
| B0-S6 | Failure scope aggregated | open | How do per-record failures aggregate into an entity-level status (per ADR-0004) when record-mode lacks a natural batch boundary? | Reshapes the record-mode half of ADR-0004 (status policy) and ADR-0006 (alert routing); B1-6 (evidence retention) needs a record-mode amendment. | ADR + record-mode status mapping + B1-6 amendment scope |
| B0-S7 | Record-oriented cost guardrails | open | What throughput, backpressure, dead-letter, and consumer-lag ceilings apply to record-mode under each environment (per ADR-0018)? | Cost couples to substrate (DD-S.6); without ceilings, record-mode lacks the cost discipline (P4) that set-mode gets from B1-2. | ADR + per-env ceilings + ADR-0019 overlay extension under `deploy/overlays/` |

---

## Process

### When a decision moves from `open` to `in-progress`

A draft document exists in `studies/decisions/` (typically created by
the `/resolve-b0` command for B0 items, or by ad-hoc study for B1/B2).
The row is updated with a link to the draft.

### When a decision moves to `resolved-study`

The study is complete: its Open Questions section is either empty or
contains only items explicitly accepted as out-of-scope for the
current cycle. The row is updated with the final study path.

### When a decision moves to `resolved-adr`

The study has been rewritten as an ADR under `docs/adr/` during Wave
3. The row keeps the original study link **and** adds the ADR link.
The study stays in `studies/decisions/` as historical reasoning; it
is not deleted.

### When a new decision is discovered

If a working session reveals a decision that is not yet in the log,
the session adds it before resolving it. Decisions found mid-session
should be tracked here even if not immediately worked on. This keeps
the log honest.

---

## Wave Gates

Use this section to confirm whether the project can advance.

### Wave 1 gate (B0 complete)

Pass when **every B0 row** is at status `resolved-study` or
`resolved-adr`. Currently: **7 of 7 resolved-adr — gate met**
(all B0 studies promoted to `docs/adr/0001-0007` on 2026-05-21).

### Wave 2 gate (platform decisions complete)

Pass when the consolidated Wave 2 decisions document exists in
`studies/decisions/` and addresses every W2 row. Currently:
**5 of 5 W2 rows resolved-adr — gate met**
(study at
[`studies/decisions/2026-05-21-platform-decisions-wave2.md`](../decisions/2026-05-21-platform-decisions-wave2.md);
ADRs at `docs/adr/0008–0012`, promoted 2026-05-21).

### Wave 3 readiness

Pass when both Wave 1 gate and Wave 2 gate have passed. Wave 3 (full
scaffolding) cannot start before this. Currently: **gate met as of
2026-05-21**; phase progression tracked in the
[Wave 3 — Phases](#wave-3--phases-scaffolding-sequencing) table
above.

### Wave 3 completion gate (all phases scaffolded)

Pass when **every row in the [Wave 3 — Phases](#wave-3--phases-scaffolding-sequencing)
table is closed**. Currently: **9 of 9 phases closed — gate met as
of 2026-05-23** (W3-P0 through W3-P8 all closed; the closing
sub-phases are W3-P4a/b/c/d/e, W3-P5, W3-P6a/c/d, W3-P7a/b/c, and
W3-P8a/b/c/d).

Five resolved-study items inside or adjacent to Wave 3 are now
promoted to ADRs alongside this gate: **B1-9 → ADR-0015** (CODEOWNERS),
**B1-10 → ADR-0016** (workspace tooling), **B1-11 → ADR-0017**
(substrate-posture amendment to ADR-0010), **B1-4 → ADR-0018**
(environment configuration model), **B2-8 → ADR-0019** (infrastructure
tooling — Kustomize). The remaining open B1 and B2 rows above are
demand-driven follow-ups, not Wave 3 blockers.

**Post-Wave-3 follow-up backlog** (work that survives Wave 3 closure):

- **Open B1 rows** — B1-1 (baseline strategy), B1-2 (BigQuery cost
  ceilings; referenced by ≥4 runbook TBDs under `docs/runbooks/`),
  B1-3 (scheduler catchup behavior), B1-5 (local testing strategy),
  B1-6 (evidence retention parameters), B1-7 (compatibility window
  duration), B1-8 (manifest cryptographic posture; flagged as a
  potential B0-5 reopener if strengthened).
- **Open B2 rows** — B2-1…B2-7 (long-tail implementation-phase
  items), plus the newly registered B2-9 (owner ↔ CODEOWNERS-group
  linter cross-check) and B2-10 (`dq-manifest set-pointer` rollback
  subcommand).
- **Operational `PLACEHOLDER` substitutions** awaiting the
  GitHub-org / GCP-project provisioning session: `@PLACEHOLDER-org/…`
  in `/.github/CODEOWNERS` and `rules/_owners.yaml`;
  `dq-{qa,prod}-PLACEHOLDER-*` identifiers in
  `engine/internal/env/{qa,prod}.go` and the matching deploy overlays.
- **Runbook TBDs** under `docs/runbooks/` waiting on B1 numeric
  parameters (mostly B1-2) or on B2-10's CLI surface.

These follow-ups do not block any wave gate; each is picked up
on demand under the same study → critique → promotion protocol that
governed Waves 1–3.

---

## Recommended Next Sequence

Waves 1, 2, and 3 are complete (gates met 2026-05-21, 2026-05-21,
and 2026-05-23 respectively). The original 9-step sequence below is
historical; the live sequence from 2026-05-23 onward is demand-
driven against the post-Wave-3 follow-up backlog enumerated in the
Wave 3 completion gate section above.

Suggested triage order when starting a follow-up session:

1. **Operational unblocks first.** If a `PLACEHOLDER` substitution
   or a missing CLI surface (e.g., B2-10) is blocking an imminent
   operational task, resolve it before any B1 study session.
2. **B1 rows with downstream consumers next.** B1-2 (cost ceilings)
   unblocks runbook TBDs; B1-8 (manifest cryptographic posture) is
   the only open B1 with B0-reopener potential and so deserves
   priority over the other open B1 rows.
3. **B2 rows last**, on demand. B2-9 (linter cross-check) and B2-10
   (CLI rollback subcommand) are demand-driven; the other B2 rows
   surface as implementation reveals concrete needs.

### Historical sequence (Waves 1–3)

1. Resolve B0-1 (compatibility model) — it underpins B0-5 and B0-7.
2. Resolve B0-5 (manifest semantics) — required before loader
   semantics can be specified.
3. Resolve B0-2 (run identity) — required before result write model.
4. Resolve B0-3 (result write model) — depends on B0-2.
5. Resolve B0-4 (failure scope) — depends on B0-2 and B0-3.
6. Resolve B0-7 (loader and scheduler failures) — depends on B0-1,
   B0-5, B0-4.
7. Resolve B0-6 (alert routing) — depends on B0-4.
8. Run Wave 2 consolidated decision session.
9. Begin Wave 3 scaffolding.

This ordering minimized rework: each decision built on stable ground
from the previous.
