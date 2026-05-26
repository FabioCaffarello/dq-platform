<!-- path: docs/adr/0031-evidence-retention-parameters.md -->

# ADR-0031 — Evidence Retention Parameters

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

The platform's result-write layer
([ADR-0003](./0003-result-write-model.md)) commits `dq_executions`
and `dq_check_results` as append-only tables, with
`sample_violating_rows` carrying per-failure evidence on the
check-results table. The append-only commitment is load-bearing
(CC1): engine code paths never issue `UPDATE` or `DELETE`. The
evidence cost-discipline ceilings on the sample-count axis are
already locked:

- [ADR-0029](./0029-bigquery-cost-ceilings.md) committed set-mode
  `MaxEvidenceSampleSize` (5 / 50 / 100 per local / qa / prod).
- [ADR-0027](./0027-record-mode-cost-guardrails.md) committed
  record-mode `MaxEvidenceSampleSize` under `RecordModeCost`
  (100 / 1000 / 10000 per local / qa / prod) and
  `SampleStorageCapMB` (100 / 1000 / 10000 MB per env).
- [ADR-0026](./0026-failure-scope-aggregated.md) committed the
  record-mode sample shape (Kafka offset + handler-derived
  violation reason) and explicitly deferred the privacy bounds
  on sample content to this row.

The remaining cost-and-privacy dimensions — **retention duration**
for the result tables and **content constraints** on what may
appear in sample evidence — are this ADR's surface. Foundation
05 §"Evidence Retention" anticipated both: the 90-day default
retention was named as B1-resolved, and the "potentially
sensitive by default" privacy posture was sketched as the
existing stance for which this row commits the concrete
allowlist.

The append-only commitment from ADR-0003 CC1 constrains the
mechanism: retention cannot be implemented by engine
UPDATE/DELETE on the result tables. It must instead operate
through a substrate primitive (BigQuery partition expiration)
or through a separate actor outside the engine's write path
(a future redaction tool). This ADR picks the substrate
primitive for v1 and reserves the richer mechanism for a
future amendment.

The principles bearing on the decision are **P4** (cost is a
first-class constraint — append-only without retention grows
storage cost monotonically), **P3** (ownership is explicit —
the privacy posture must name who is responsible for sensitive
fields), and **R3** (do not revisit settled architecture —
ADR-0003's append-only commitment is honored by routing
retention through substrate primitives, not by reopening the
engine's write contract).

---

## Decision

### Single-tier retention via BigQuery partition expiration

`dq_executions` and `dq_check_results` retention ships as a
single per-env duration governing both tables. The engine
applies partition expiration via the substrate primitive at
table-creation time; the engine's write path is unchanged.

Two coupled schema changes close the retention gap:

1. **Tables become time-partitioned.** Today's `EnsureSchema`
   in `engine/internal/results/bigquery_store.go` creates the
   tables with `bigquery.TableMetadata{Schema: schema}` only —
   no partitioning. This ADR extends `EnsureSchema` to set
   `TimePartitioning{Field: "recorded_at", Type: DAY,
   Expiration: ResultsRetention}` at table-creation time. The
   partition field is `recorded_at`, which every row already
   carries (ADR-0003 CC3).
2. **Pre-existing non-partitioned tables require a one-time
   migration.** BigQuery cannot add partitioning in place; the
   migration is `CREATE TABLE ... AS SELECT ... PARTITION BY
   DATE(recorded_at)` followed by a rename. At this ADR's
   acceptance, the only deployed dataset with non-trivial
   state is `dq-local` (which developers wipe routinely);
   qa/prod datasets are `PLACEHOLDER`-named and have not been
   provisioned, so qa/prod become green-field through the new
   `EnsureSchema` on first contact. A deployment that has
   accumulated non-partitioned history before this ADR
   reaches it requires the migration runbook registered as a
   B2 follow-up below.

Once both schema changes are in place, partitions older than
`ResultsRetention` are auto-dropped by BigQuery. The engine
code path is unchanged: partition expiration is a substrate
operation, not an engine UPDATE/DELETE. ADR-0003 CC1's
append-only commitment for engine code paths is honored.

### `EnvConfig.EvidenceRetention` sub-struct

A new typed sub-struct ships on `EnvConfig` per
[ADR-0018](./0018-environment-configuration-model.md) PAT-4
and is populated in each per-env file
(`engine/internal/env/{local,qa,prod}.go`):

```
// Annotations: "substrate" = enforced by BigQuery partition
// expiration; "policy" = read by operators + rule authors,
// not enforced by engine code.
type EvidenceRetention struct {
    ResultsRetention      time.Duration // substrate
    SampleContentAllowlist []string     // policy
}
```

#### Per-environment values

| Field | local | qa | prod |
|---|---|---|---|
| `ResultsRetention` | 30 days | 90 days | 365 days |
| `SampleContentAllowlist` | (same closed set) | (same closed set) | (same closed set) |

**Per-value rationale.** Local 30 days lets developers iterate
without unbounded laptop growth; qa 90 days matches a typical
integration cycle (one calendar quarter); prod 365 days matches
a standard annual operational forensic-review cycle without
committing to multi-year archival. The 12× spread mirrors
`MaxWindowDuration`'s spread from ADR-0029 intentionally — the
same operational tempo governs both. The 365-day prod value is
conservative; operators needing longer archival extend via the
per-env Go config surface under the same code-review path as
other `EnvConfig` changes per ADR-0018 PAT-4. The allowlist
does not weaken in local just because the dataset is synthetic;
the categories are a contract authors and reviewers learn
once.

### Sample-content allowlist

`sample_violating_rows` may contain only content matching the
five categories below. Any other content is forbidden at v1.
Each category is tagged with the mode(s) it applies to so the
contract is unambiguous:

- **`primary_key`** *(set-mode)* — the rule's declared
  primary-key column value, intended as a forensic row
  identifier. Rule authors are responsible for ensuring the
  primary key is not itself a PII field; if it is
  (e.g., `user_email`), the rule must be amended to use a
  non-PII identifier or a hashed surrogate. CODEOWNERS PR
  review is the v1 gate per
  [ADR-0015](./0015-codeowners.md).
- **`check_relevant_columns`** *(set-mode)* — columns the
  check evaluates, declared per-check in the rule artefact.
  Foundation 05 §"Evidence Retention" already commits this
  posture; this ADR redeems it by naming it as an allowed
  category.
- **`partition_column_value`** *(set-mode)* — the
  partition-column value for forensic context (which
  time-window the violation fell in).
- **`forensic_locator`** *(record-mode)* — Kafka partition +
  offset. ADR-0026 §"Evidence sample shape" committed the
  Kafka offset; this ADR extends the locator to
  `{ partition, offset }` because partition is part of the
  natural Kafka address (the β implementation in
  `engine/internal/eval/record_schema_conformance.go`
  already carries both). Substrate-native addressing
  primitives; neither field is PII.
- **`violation_reason`** *(record-mode; future set-mode
  enrichment)* — handler-derived structured string
  describing why the violation fired (e.g., "missing
  required field 'id'", "row_count == 0"). Today only
  record-mode handlers emit a structured reason on each
  per-record sample; set-mode samples carry the violating
  row's columns themselves and do not use this category. A
  future set-mode handler that emits a per-row reason
  string (e.g., a `set.column_constraint_violated` kind)
  would consume this category. **In every mode, reason
  strings must not quote raw field values** from the data
  that triggered the violation; reasons describe the
  violation structurally.

### Privacy responsibility ladder

The platform commits the following responsibility layout:

- **Rule author.** Picks a non-PII primary key. Picks
  check-relevant columns deliberately. Treats every value
  inside the allowlist categories as potentially sensitive
  even if it does not strictly contain PII.
- **CODEOWNERS reviewer (`@PLACEHOLDER-org/rules-authors`).**
  Reviews each rule change for whether the primary key or
  any check-relevant column would expose sensitive content.
  The CODEOWNERS gate is the v1 enforcement point per
  ADR-0015 and the alert-routing convention from
  [ADR-0006](./0006-alert-routing-contract.md).
- **Operator.** Reviews access controls on the
  `dq_check_results` table — read access is tighter than the
  aggregated dashboard tables (foundation 05's posture).
  Substrate IAM is the enforcement primitive.
- **Platform.** Refuses to capture sample content outside
  the allowlist categories. Documents the categories in the
  operator-facing security note. Does **not** provide
  automated PII detection at v1 — that's a deferred
  enhancement, not a current capability.

The platform's commitment: **the allowlist is narrow; the
responsibility for what falls inside it is on the rule author
and the CODEOWNERS reviewer**. The platform does not pretend
to detect PII automatically.

### Operator-facing security note

A new operator-facing security note ships at
`docs/security/evidence-retention.md` carrying the threat
model, the allowlist, the retention values per env, the
responsibility ladder, and the deferred enhancements. The
`docs/security/` directory was introduced by
[ADR-0030](./0030-manifest-cryptographic-posture.md); this is
the second entry under it.

### Why this does NOT reopen ADR-0003

ADR-0003 CC1 commits that engine code paths never issue
UPDATE or DELETE. With this ADR:

- The engine code path is unchanged. `EnsureSchema` adds a
  table-creation parameter (`TimePartitioning`); it does
  not issue UPDATE or DELETE.
- Partition expiration is a substrate operation, not an
  engine code path. The substrate (BigQuery) drops expired
  partitions internally; the engine does not orchestrate
  the drop.
- The `Writer` / `Reader` / `Store` interfaces from
  `engine/internal/results/results.go` are unchanged.

ADR-0003 stays accepted without amendment.

### Why this does NOT reopen ADR-0026

ADR-0026 explicitly deferred record-mode privacy bounds to
this row. With this ADR:

- The five-category `SampleContentAllowlist` confirms
  ADR-0026's "no raw record bytes or field values" stance
  as the v1 commitment.
- Record-mode samples carry `forensic_locator` and
  `violation_reason`. ADR-0026 §"Evidence sample shape"
  committed Kafka offset + violation reason; this ADR
  extends the locator to `{ partition, offset }` per the
  β implementation, which is additive to ADR-0026's
  commitment rather than a revision.
- The catalog's per-kind `evidence_sample_size` semantics
  are unchanged; SIZE stays bounded by ADR-0027's
  `MaxEvidenceSampleSize`; CONTENT is bounded by this
  ADR's allowlist.

ADR-0026 is satisfied — its B1-6 deferral redeemed without
reopening the underlying ADR.

### Single-tier vs two-tier retention

Foundation 05 §"How long it is kept" sketched a two-tier
scheme: samples expire before the result row drops, with a
`samples_purged: true` marker on the row. v1 ships
**single-tier** retention — samples and rows share the same
expiration window. The two-tier scheme is deferred to a
future amendment when concrete operational signal (an
operator request for aggregate result history longer than the
sample-evidence window) justifies the implementation cost.
The two-tier scheme would require either an engine UPDATE
(violating ADR-0003 CC1) or a separate redaction tool; the
deferred amendment would ship the redaction tool, not amend
ADR-0003.

---

## Consequences

1. **A new `EnvConfig.EvidenceRetention` sub-struct ships in
   `engine/internal/env/config.go`**, populated in
   `local.go` / `qa.go` / `prod.go` with the values committed
   above. The ADR-0018 MD-4 reflect-based exhaustiveness test
   catches any per-env file that forgets a field.

2. **`EnsureSchema` extends to create time-partitioned tables
   with expiration.** Both `dq_executions` and
   `dq_check_results` are created with
   `TimePartitioning{Field: "recorded_at", Type: DAY,
   Expiration: ResultsRetention}`. Two coupled schema
   commitments: tables gain partitioning by `recorded_at`,
   and the partition expiration is set to the per-env
   `ResultsRetention` value. BigQuery drops expired
   partitions internally; ADR-0003 CC1's append-only
   commitment for engine code paths is unchanged (partition
   expiration is a substrate operation, not an engine
   UPDATE/DELETE).

3. **The five-category `SampleContentAllowlist` is the v1
   privacy contract.** The platform refuses to capture sample
   content outside these categories. Rule authors +
   CODEOWNERS reviewers (per ADR-0015) are the responsibility
   layer for whether content within the categories is
   PII-safe.

4. **`docs/security/evidence-retention.md` ships as the
   operator-facing security note** alongside the ADR. The
   note documents the threat model, allowlist, retention
   values, responsibility ladder, and the deferred
   enhancements. Second entry under `docs/security/` (the
   directory introduced by ADR-0030).

5. **A note is added to the existing
   `docs/runbooks/orphan-run-remediation.md` and
   `docs/runbooks/refresh-failure-escalation.md`**
   referencing the security note for the privacy posture
   context. These runbooks already cite ADR-0003 / ADR-0007;
   adding the security-note cross-reference keeps the
   privacy posture visible during incident response.

6. **ADR-0026's record-mode privacy deferral is redeemed.**
   The record-mode evidence shape lands as
   `forensic_locator` + `violation_reason` allowlist
   categories. ADR-0026 §"Evidence sample shape" committed
   Kafka offset + handler-derived reason; this ADR extends
   the locator to `{ partition, offset }` (matching the β
   implementation in
   `engine/internal/eval/record_schema_conformance.go`) and
   confirms "no raw record bytes or field values" as the
   platform's committed posture.

7. **Foundation 05's `samples_purged: true` two-tier scheme
   is deferred.** v1 ships single-tier retention. The
   two-tier scheme is reserved for a future amendment when
   concrete operational signal (an operator request for
   longer aggregate history without sample bytes) justifies
   the implementation cost.

8. **No automated PII detection ships.** Rule authors +
   CODEOWNERS reviewers are the v1 gate. The platform does
   not lint for "this column looks like an email"; that
   detection capability is a future enhancement reserved
   for a B2 row if and when concrete signal surfaces.

9. **The cost-discipline ADRs are unchanged.** ADR-0027
   (`RecordModeCost.MaxEvidenceSampleSize`,
   `SampleStorageCapMB`) and ADR-0029
   (`SetModeCost.MaxEvidenceSampleSize`) commit the SIZE
   ceilings; this ADR commits the DURATION + CONTENT
   constraints. The three together compose the full evidence
   cost-discipline surface.

10. **B2 follow-up: pre-existing-table migration runbook.**
    A future deployment that has accumulated non-partitioned
    history before this ADR reaches it requires a one-time
    migration (`CREATE TABLE ... AS SELECT ... PARTITION BY
    DATE(recorded_at)` followed by a rename). At this ADR's
    acceptance the migration is effectively green-field
    (`dq-local` is wiped routinely; qa/prod are
    `PLACEHOLDER`-provisioned). A B2 row registers the
    migration runbook for the future-deployment case.

11. **The platform's P3 + P4 commitments for evidence are
    now explicit.** P3 (ownership): rule authors + CODEOWNERS
    + operators each carry a named responsibility. P4 (cost):
    retention caps the unbounded-growth path that append-only
    would otherwise create.

---

## Notes

- The 30 / 90 / 365 day spreads are conservative starting
  points, not calibrated targets. The operational session
  that provisions real qa/prod GCP projects has the
  authority to tune them via PR review against
  `engine/internal/env/{qa,prod}.go`.
- A future amendment that ships the two-tier `samples_purged`
  scheme adds a separate `tools/retention` binary (likely
  under `tools/` alongside `tools/lint` and
  `tools/manifest`). The binary issues
  `UPDATE ... SET sample_violating_rows = NULL` on rows
  older than a `SampleRetention` window, and `DELETE` on
  rows older than `ResultsRetention`. The engine code path
  stays append-only; the retention tool is the new actor.
  The deferred amendment ships the tool, not an ADR-0003
  revision.
- The `SampleContentAllowlist` is enumerated rather than
  enforced by code at v1. A future enhancement could ship a
  lint cross-check that scans each rule artefact's referenced
  columns against a per-rule
  `sample_content_categories_declared` field, refusing rules
  that declare categories outside the allowlist. Reserved
  for a B2 row if concrete need surfaces.
- The audit-log retention for the substrate's object-store +
  table-write events is a separate operational-policy
  concern (deployed via the operational session's GCP
  project). This ADR does not commit audit-log retention; it
  is implicit at the substrate level.
