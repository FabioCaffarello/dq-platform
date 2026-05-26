<!-- path: docs/security/evidence-retention.md -->

# Evidence Retention Posture

> **Status:** v1 posture is **single-tier retention via BigQuery
> partition expiration**, with a narrow content allowlist for
> `sample_violating_rows`. The authoritative commitment is
> [ADR-0031](../adr/0031-evidence-retention-parameters.md); this
> note is the operator-facing summary.

---

## What this note is for

Operators reading this note are typically here for one of three
reasons:

- **Reviewing access controls** on the result tables
  (`dq_executions`, `dq_check_results`) — the privacy posture
  drives the access-matrix decisions.
- **Responding to a sample-content incident** (a rule's
  `sample_violating_rows` exposed sensitive data; you need to
  understand what the allowlist allows and who reviews rule
  changes).
- **Adjusting retention values** for a deployment (the per-env
  defaults are conservative starting points; operators can
  tune them via PR review under
  [ADR-0018](../adr/0018-environment-configuration-model.md)
  PAT-4).

The note covers retention, the content allowlist, the
responsibility ladder, and the deferred enhancements.

---

## Retention values

Both `dq_executions` and `dq_check_results` are partitioned by
`recorded_at` (created by `EnsureSchema` at first-contact) and
the partition expiration is set per env:

| Env | `ResultsRetention` | Notes |
|---|---|---|
| local | 30 days | Fast dev iteration without unbounded laptop growth. |
| qa | 90 days | Matches a typical integration cycle (one calendar quarter). |
| prod | 365 days | Standard annual operational forensic-review cycle. |

Partition expiration is a **BigQuery substrate operation**: the
substrate drops expired partitions internally. The engine code
path is unchanged ([ADR-0003](../adr/0003-result-write-model.md)
CC1's append-only commitment is honored — partition expiration
is not an engine UPDATE/DELETE).

### Adjusting retention

Retention values live in
`engine/internal/env/{local,qa,prod}.go` under
`EvidenceRetention.ResultsRetention`. Adjusting a value is a
code change reviewed under the same PR-review surface as any
other `EnvConfig` change per ADR-0018 PAT-4. The reflect-based
exhaustiveness test from ADR-0018 MD-4 catches any per-env file
that forgets the field.

A deployment that has accumulated **non-partitioned** history
before this posture lands cannot get partition expiration via
the new `EnsureSchema` alone — BigQuery cannot add partitioning
in place. The migration is `CREATE TABLE ... AS SELECT ...
PARTITION BY DATE(recorded_at)` followed by a rename. At
ADR-0031's acceptance the migration is effectively green-field
for all deployments (`dq-local` is wiped routinely; qa/prod are
`PLACEHOLDER`-provisioned via the new `EnsureSchema`); the
runbook for a future deployment that needs the in-place
migration is registered as a B2 follow-up.

---

## Sample-content allowlist

`sample_violating_rows` on `dq_check_results` may carry only
content matching the five categories below. Any other content
is forbidden at v1.

### `primary_key` *(set-mode)*

The rule's declared primary-key column value, intended as a
forensic row identifier. **Rule authors are responsible for
ensuring the primary key is not itself a PII field.** If it
is (e.g., `user_email`), the rule must be amended to use a
non-PII identifier or a hashed surrogate. CODEOWNERS PR review
is the v1 gate per
[ADR-0015](../adr/0015-codeowners.md).

### `check_relevant_columns` *(set-mode)*

Columns the check evaluates, declared per-check in the rule
artefact. Foundation 05 §"Evidence Retention" already committed
this as the existing posture — columns NOT relevant to the
check are not captured, as a defense against accidental
exfiltration of PII through unrelated columns.

### `partition_column_value` *(set-mode)*

The partition-column value (typically a date or timestamp) for
forensic context — tells the reader which time-window the
violation fell in.

### `forensic_locator` *(record-mode)*

Kafka partition + offset (`{ partition, offset }`).
[ADR-0026](../adr/0026-failure-scope-aggregated.md) §"Evidence
sample shape" committed the Kafka offset; ADR-0031 extends the
locator to include the partition because partition is part of
the natural Kafka address. The β implementation in
`engine/internal/eval/record_schema_conformance.go` already
carries both. Both fields are substrate-native addressing
primitives; neither is PII.

### `violation_reason` *(record-mode; future set-mode enrichment)*

Handler-derived structured string describing why the violation
fired (e.g., `"missing required field 'id'"`, `"row_count ==
0"`). Today only record-mode handlers emit a structured reason
on each per-record sample; set-mode samples carry the violating
row's columns themselves and do not use this category.

**In every mode, reason strings must not quote raw field
values** from the data that triggered the violation. Reason
strings describe the violation **structurally**, not the data
that triggered it.

### What is forbidden

Any sample content outside the five categories above is
forbidden at v1. In particular:

- Raw record bytes from Kafka payloads (record-mode).
- Field values from inside a record body (record-mode).
- PII fields from a set-mode row that aren't the primary key
  or check-relevant columns (set-mode).
- Free-form text strings that reproduce content from the data
  that triggered the violation.

---

## Privacy responsibility ladder

| Layer | Responsibility |
|---|---|
| **Rule author** | Picks a non-PII primary key. Picks check-relevant columns deliberately. Treats every value inside the allowlist categories as potentially sensitive even if not strictly PII. |
| **CODEOWNERS reviewer (`@PLACEHOLDER-org/rules-authors`)** | Reviews each rule change for whether the primary key or any check-relevant column would expose sensitive content. The CODEOWNERS gate is the v1 enforcement point per ADR-0015. |
| **Operator** | Reviews access controls on the `dq_check_results` table — read access is tighter than the aggregated dashboard tables. Substrate IAM is the enforcement primitive. |
| **Platform** | Refuses to capture sample content outside the allowlist categories. Documents the categories in this note. Does **not** provide automated PII detection at v1. |

The platform's commitment is therefore: **the allowlist is
narrow; the responsibility for what falls inside it is on the
rule author and the CODEOWNERS reviewer**. The platform does
not pretend to detect PII automatically.

---

## Deferred enhancements

The following enhancements are explicitly **not shipped at v1**.
They are reserved for future amendments when concrete
operational signal justifies the implementation cost.

### Two-tier retention (`samples_purged: true` scheme)

Foundation 05 §"How long it is kept" sketched a two-tier
scheme: samples expire before the result row drops, with a
`samples_purged: true` marker on the row. v1 ships single-tier
retention (samples and rows share the same expiration window).
The two-tier scheme is reserved for a future amendment when an
operator requests aggregate result history longer than the
sample-evidence window.

Implementation path if revisited: a separate `tools/retention`
binary issues `UPDATE ... SET sample_violating_rows = NULL` on
rows older than a `SampleRetention` window and `DELETE` on rows
older than `ResultsRetention`. The engine code path stays
append-only; the retention tool is the new actor. The deferred
amendment ships the tool, not an ADR-0003 revision.

### Automated PII detection at lint time

A future `tools/lint` extension could scan rule artefacts for
primary-key columns or check-relevant columns whose names match
common PII patterns (`email`, `ssn`, `phone`, etc.) and refuse
rules that would publish such columns into samples. The
detection is heuristic, not exhaustive; v1 leans on rule
author + CODEOWNERS judgment. Reserved for a B2 row if a
concrete signal (incident, audit finding, contributor request)
surfaces.

### Per-rule sample-category declaration

The five-category allowlist is enumerated in this note and the
ADR; it is not enforced by code at v1. A future enhancement
could ship a lint cross-check that scans each rule artefact's
referenced columns against a per-rule
`sample_content_categories_declared` field, refusing rules that
declare categories outside the allowlist. Reserved for a B2
row if concrete need surfaces.

---

## Cross-references

- [ADR-0031](../adr/0031-evidence-retention-parameters.md) —
  the authoritative retention + allowlist commitment.
- [ADR-0003](../adr/0003-result-write-model.md) — the
  append-only commitment that retention must honor.
- [ADR-0026](../adr/0026-failure-scope-aggregated.md) — the
  record-mode evidence shape (committed Kafka offset; ADR-0031
  extends to partition + offset).
- [ADR-0027](../adr/0027-record-mode-cost-guardrails.md) —
  record-mode `MaxEvidenceSampleSize` (per-sample-count
  ceiling).
- [ADR-0029](../adr/0029-bigquery-cost-ceilings.md) — set-mode
  `MaxEvidenceSampleSize` (per-sample-count ceiling).
- [ADR-0015](../adr/0015-codeowners.md) — CODEOWNERS PR
  review, the v1 privacy enforcement gate.
- [ADR-0018](../adr/0018-environment-configuration-model.md) —
  PAT-4 typed env config; the surface retention values live on.
- [ADR-0030](../adr/0030-manifest-cryptographic-posture.md) —
  introduced the `docs/security/` directory; this is the
  second entry under it.
