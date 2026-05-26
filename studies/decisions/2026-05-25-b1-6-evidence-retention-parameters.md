<!-- path: studies/decisions/2026-05-25-b1-6-evidence-retention-parameters.md -->

# B1-6 — Evidence Retention Parameters

## Context

Foundation 05 §"Evidence Retention" commits the shape and the
sensitivity posture of failed-sample evidence on
`dq_check_results`:

- Up to a configured count of violating rows per check
  (default 100).
- Each sample carries the primary key, check-relevant columns,
  and the partition value — **columns not relevant to the
  check are not captured** (defense against accidental PII
  exfiltration).
- 90-day default retention for samples, with reporting tables
  themselves retained longer — exact value deferred to this
  B1 row.
- Samples are treated as potentially sensitive; access
  controls on the reporting tables are tighter than on
  aggregated dashboards.

Concrete cost-discipline ceilings on the sample-count axis
already shipped:

- [ADR-0029](../../docs/adr/0029-bigquery-cost-ceilings.md)
  committed set-mode `MaxEvidenceSampleSize` (5 / 50 / 100 per
  local / qa / prod).
- [ADR-0027](../../docs/adr/0027-record-mode-cost-guardrails.md)
  committed record-mode `MaxEvidenceSampleSize` under
  `RecordModeCost` (100 / 1000 / 10000 per
  local / qa / prod).
- [ADR-0026](../../docs/adr/0026-failure-scope-aggregated.md)
  committed the record-mode sample shape (Kafka partition +
  offset + handler-derived reason; **no raw record bytes or
  field values until B1-6 commits privacy bounds**).

What B1-6 must commit:

1. **Retention duration** per environment for the result tables
   (foundation 05's deferred parameter).
2. **Privacy constraints** on sample content (foundation 05's
   "potentially sensitive" posture made explicit; ADR-0026's
   "no raw content until B1-6" commitment redeemed).
3. **The mechanism** that enforces retention against the
   ADR-0003 append-only invariant.

The append-only commitment from
[ADR-0003](../../docs/adr/0003-result-write-model.md) CC1 is
load-bearing: engine code paths never issue `UPDATE` or
`DELETE`. Retention must therefore operate outside the engine's
write surface — via substrate-level partition expiration, a
separate retention tool, or an operator-driven purge. This row
picks the substrate primitive for v1 and reserves a richer
mechanism for a future amendment if needed.

The principles bearing on the decision are **P4** (cost is a
first-class constraint — append-only without retention grows
storage cost monotonically), **P3** (ownership is explicit —
the privacy posture must name who is responsible for sensitive
fields), and **R3** (do not revisit settled architecture —
ADR-0003's append-only commitment is honored by routing
retention through substrate primitives, not by reopening the
engine's write contract).

---

## Decision Drivers

- **DD-1 — Append-only must not be reopened.** ADR-0003 CC1
  commits engine code paths never issue UPDATE or DELETE. The
  retention mechanism must operate outside engine code (via
  substrate primitives or a separate tool), not by amending
  ADR-0003.
- **DD-2 — Per-environment values must diverge.** Local should
  drop fast (developers don't need 90-day history on a laptop);
  qa should match a typical integration cycle; prod should
  match the longest forensic window operators reasonably need.
  ADR-0018 PAT-4's per-env Go-config pattern applies.
- **DD-3 — Privacy posture must close the "no raw content"
  commitment from ADR-0026.** That ADR explicitly deferred
  record-mode privacy bounds to B1-6. The study redeems the
  deferral via the allowlist contract and confirms foundation
  05's set-mode sample-shape posture (primary key +
  check-relevant columns + partition value) inside the same
  allowlist; B1-6 does not modify the set-mode shape, only
  names it as an allowed category.
- **DD-4 — The mechanism must be commodity-friendly.** A
  retention scheme that requires custom tooling (a `tools/retain`
  binary, a scheduled BigQuery DML job) has operational cost.
  A scheme that leans on the substrate's native primitive
  (BigQuery partition expiration) is cheaper. The study picks
  the cheaper option for v1 and reserves the richer option for
  a future amendment if needed.
- **DD-5 — Foundation 05's "samples_purged: true" two-tier
  scheme is desirable but not required at v1.** Foundation 05
  imagined a scheme where samples are redacted before the
  check-result row is dropped, preserving the result history
  longer than the sample evidence. This is appealing but
  requires either an UPDATE (violating ADR-0003 CC1's engine-
  side commitment) or a separate redaction tool. v1 commits
  the single-tier scheme (samples and rows share retention)
  and reserves the two-tier enhancement.
- **DD-6 — Privacy responsibility ladder must be explicit.**
  Foundation 05 says "rule authors choose check-relevant
  columns"; ADR-0015 commits CODEOWNERS on rule changes. B1-6
  must say explicitly that rule authors + CODEOWNERS reviewers
  are the privacy gate at v1, and that the platform does not
  provide automated PII detection or per-column redaction.

---

## Considered Options

### Option 1 — BigQuery partition expiration; same retention for executions + check results (recommended)

This option commits two coupled schema changes (see
Recommendation §`ResultsRetention` for the full mechanism):
`dq_executions` and `dq_check_results` **become partitioned** by
`recorded_at` (today they are not), and BigQuery's table-level
partition-expiration is set to the per-env retention duration.
The engine applies both at table-creation time via
`EnsureSchema` in `engine/internal/results/bigquery_store.go`.
A single per-env duration governs both tables. Sample evidence
(`sample_violating_rows`) is dropped with the row when its
partition expires; no separate redaction step.

**Strengths.** Uses the substrate's native primitive (no
custom retention tool); respects ADR-0003 CC1 (no engine-side
UPDATE/DELETE; partition expiration is a substrate
operation); same per-env divergence pattern as the cost
ADRs (ADR-0027 / ADR-0029); minimal operational surface; one
configuration field per env.

**Trade-offs.** Foundation 05's "samples_purged: true"
two-tier scheme is not implemented — sample evidence and the
result row share their retention window. An operator who
needs aggregate result history beyond the sample-evidence
window cannot get it at v1. The retention is also coarse-
grained (whole partition at once) rather than per-row.

### Option 2 — Two-tier retention via a separate redaction tool

A new `tools/retention` binary runs on a schedule (cron / k8s
CronJob), issues `UPDATE ... SET sample_violating_rows = NULL`
on `dq_check_results` rows older than `SampleRetention`, and
later issues `DELETE` on rows older than `ResultRetention`.
The engine code path stays append-only; the retention tool is
the new actor.

**Strengths.** Implements foundation 05's two-tier scheme —
samples expire before result rows; an operator can query
aggregate failure history without sample bytes for longer.

**Trade-offs.** Introduces a new tool (build + test + CI
lane); the tool has its own credential surface (UPDATE/DELETE
on the results dataset is a meaningful privilege); the
mechanism partially reopens ADR-0003 CC1 (engine code still
honors append-only, but the **system** as a whole now issues
UPDATE/DELETE on the same table). The complexity is real and
the operational benefit (longer result history without
samples) is hypothetical until concrete operational signal
shows it's needed.

### Option 3 — Indefinite retention; defer the decision

Don't set partition expiration at v1; let the tables grow
indefinitely. Defer the retention decision to a future row
when storage cost becomes observable.

**Strengths.** Zero implementation cost. Maximum forensic
history retained.

**Trade-offs.** Violates DD-2 explicitly (no per-env divergence
means local laptops carry unbounded state). The privacy
posture is unaddressed — every sample lives forever,
amplifying the compliance surface. P4 cost discipline is
broken — append-only without retention is unbounded growth.
This option is mentioned only to be rejected: deferral is not
the right answer for retention because storage costs compound
predictably.

---

## Recommendation

**Option 1.** BigQuery partition expiration, single per-env
retention duration for both `dq_executions` and
`dq_check_results`, with sample-content allowlist + privacy
responsibility ladder.

### `EnvConfig.EvidenceRetention` sub-struct

A new typed sub-struct ships on `EnvConfig` per ADR-0018 PAT-4
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

#### Field semantics

- **`ResultsRetention`** — partition-expiration duration
  applied to both `dq_executions` and `dq_check_results`.
  Two schema commitments together close the retention gap:

  1. **Tables become time-partitioned.** Today's
     `EnsureSchema` in
     `engine/internal/results/bigquery_store.go` creates the
     tables with `bigquery.TableMetadata{Schema: schema}`
     only — no partitioning. This row extends `EnsureSchema`
     to set
     `TimePartitioning{Field: "recorded_at", Type: DAY,
     Expiration: ResultsRetention}` at table-creation time.
     The partition field is `recorded_at`, which every
     row already carries (ADR-0003 CC3).
  2. **Existing non-partitioned tables require a one-time
     migration.** BigQuery cannot add partitioning in place;
     the migration is `CREATE TABLE ... AS SELECT ...
     PARTITION BY DATE(recorded_at)` followed by a rename.
     Today the only deployed dataset with non-trivial state
     is `dq-local` (which developers wipe routinely);
     qa/prod datasets are `PLACEHOLDER`-named and have not
     been provisioned, so the one-time migration is
     effectively green-field at v1. Operators provisioning
     qa/prod datasets do so via the new `EnsureSchema`,
     getting partitioned tables on first contact. The
     migration runbook for a pre-existing dataset is
     reserved as a B2 follow-up.

  Once both commitments are in place, partitions older than
  `ResultsRetention` are auto-dropped by BigQuery; the
  engine code path is unchanged (ADR-0003 CC1 honored —
  partition expiration is a substrate operation, not an
  engine UPDATE/DELETE).

- **`SampleContentAllowlist`** — the closed set of content
  categories permitted in `sample_violating_rows`. Each
  category is tagged with the mode(s) it applies to so the
  contract is unambiguous for rule authors reading the
  allowlist:
  - **`primary_key`** *(set-mode)* — the rule's declared
    primary-key column value, intended as a forensic row
    identifier. **Rule authors are responsible for ensuring
    the primary key is not itself a PII field**; if it is
    (e.g., `user_email`), the rule must be amended to use a
    non-PII identifier or a hashed surrogate. CODEOWNERS PR
    review is the v1 gate per ADR-0015.
  - **`check_relevant_columns`** *(set-mode)* — columns the
    check evaluates, declared per-check in the rule
    artefact. Foundation 05 commits this as the existing
    posture; B1-6 redeems it by naming it as an allowed
    category.
  - **`partition_column_value`** *(set-mode)* — the
    partition-column value for forensic context (which
    time-window the violation fell in).
  - **`forensic_locator`** *(record-mode)* — Kafka partition
    + offset. ADR-0026 §"Evidence sample shape" commits the
    Kafka offset; the β implementation in
    `engine/internal/eval/record_schema_conformance.go`
    extended this to carry the partition alongside the
    offset because partition is part of the natural Kafka
    locator. **B1-6 commits the extension explicitly**: the
    allowlist locator is `{ partition, offset }`, not
    offset alone. Both are substrate-native addressing
    primitives; neither is PII.
  - **`violation_reason`** *(record-mode; future set-mode
    enrichment)* — handler-derived structured string
    describing why the violation fired (e.g., "missing
    required field 'id'", "row_count == 0"). Today only
    record-mode handlers (per ADR-0026) emit a structured
    reason on each per-record sample; set-mode samples
    carry the violating row's columns themselves and do not
    use this category. A future set-mode handler that emits
    a per-row reason string (e.g., a future
    `set.column_constraint_violated` kind) would consume
    this category. **In every mode, reason strings must not
    quote raw field values** from the data that triggered
    the violation; reasons describe the violation
    structurally.

  Any sample content outside these five categories is
  forbidden at v1.

#### Per-environment values

| Field | local | qa | prod |
|---|---|---|---|
| `ResultsRetention` | 30 days | 90 days | 365 days |
| `SampleContentAllowlist` | (same closed set) | (same closed set) | (same closed set) |

**Per-value rationale.**

- **`ResultsRetention`** — local 30 days lets developers iterate
  without unbounded laptop growth; qa 90 days matches a typical
  integration cycle (one calendar quarter); prod 365 days
  matches a standard annual operational forensic-review cycle
  without committing to multi-year archival. The 12× spread
  mirrors `MaxWindowDuration`'s spread from ADR-0029,
  intentionally — the same operational tempo governs both. The
  365-day prod value is conservative; operators needing longer
  archival can extend via the per-env Go config surface under
  the same code-review path as other EnvConfig changes per
  ADR-0018 PAT-4.
- **`SampleContentAllowlist`** — the same five categories at
  every env. Privacy posture does not weaken in local just
  because the dataset is synthetic; the categories are a
  contract authors and reviewers learn once.

### Privacy responsibility ladder

The platform commits the following responsibility layout at
v1. Each layer's job is documented in the security note so
contributors know whose call is whose:

- **Rule author.** Picks a non-PII primary key. Picks
  check-relevant columns deliberately. Treats every value
  inside the allowlist categories as potentially sensitive
  even if it does not strictly contain PII.
- **CODEOWNERS reviewer (`@PLACEHOLDER-org/rules-authors`).**
  Reviews each rule change for whether the primary key or
  any check-relevant column would expose sensitive content.
  The CODEOWNERS gate is the v1 enforcement point per
  ADR-0015 + ADR-0006 §"alert routing".
- **Operator.** Reviews access controls on the
  `dq_check_results` table — read access is tighter than the
  aggregated dashboard tables (foundation 05's "access
  controls on the reporting tables are tighter than access
  controls on aggregated dashboards" posture). Substrate
  IAM is the enforcement primitive.
- **Platform.** Refuses to capture sample content outside
  the allowlist categories. Documents the categories in the
  security note. Does **not** provide automated PII
  detection at v1 — that's a deferred enhancement, not a
  current capability.

The platform's commitment is therefore: **the allowlist is
narrow; the responsibility for what falls inside it is on the
rule author and the CODEOWNERS reviewer**. The platform does
not pretend to detect PII automatically.

### Why this does NOT reopen ADR-0003

ADR-0003 CC1 commits that engine code paths never issue
UPDATE or DELETE. With this recommendation:

- The engine code path is unchanged. `EnsureSchema` adds a
  table-creation parameter (`TimePartitioning.Expiration`);
  it does not issue UPDATE or DELETE.
- Partition expiration is a substrate operation, not an
  engine code path. The substrate (BigQuery) drops expired
  partitions internally; the engine does not orchestrate
  the drop.
- The `Writer` / `Reader` / `Store` interfaces from
  `engine/internal/results/results.go` are unchanged.

ADR-0003 stays accepted without amendment. The recommendation
honors the append-only invariant by routing retention through
the substrate primitive rather than through engine code.

### Why this does NOT reopen ADR-0026

ADR-0026 explicitly deferred record-mode privacy bounds to
B1-6. With this recommendation:

- The five-category `SampleContentAllowlist` confirms
  ADR-0026's "no raw record bytes or field values" stance
  as the v1 commitment.
- Record-mode samples carry `forensic_locator` (ADR-0026
  §"Evidence sample shape" committed Kafka offset; B1-6
  extends to `{ partition, offset }` per the β
  implementation in
  `engine/internal/eval/record_schema_conformance.go`) +
  `violation_reason` (the structured reason the handler
  derived); both are in the allowlist.
- The catalog's per-kind `evidence_sample_size` semantics
  are unchanged; the SIZE of the sample stays bounded by
  ADR-0027's `MaxEvidenceSampleSize`, and the CONTENT of
  each sample is bounded by this row's allowlist.

ADR-0026 is satisfied — its B1-6 deferral redeemed without
reopening.

### Operator-facing security note

A new operator-facing security note ships at
`docs/security/evidence-retention.md` carrying the threat
model, the allowlist, the retention values per env, the
responsibility ladder, and the enhancement deferrals. The
`docs/security/` directory was introduced by
[ADR-0030](../../docs/adr/0030-manifest-cryptographic-posture.md);
this is the second entry under it.

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
   Expiration: ResultsRetention}`. The schema change is two
   coupled commitments: tables gain partitioning by
   `recorded_at`, and the partition expiration is set to the
   per-env `ResultsRetention` value. BigQuery drops expired
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
   Kafka offset + handler-derived reason; B1-6 extends the
   locator to `{ partition, offset }` (matching the β
   implementation in
   `engine/internal/eval/record_schema_conformance.go`) and
   confirms "no raw record bytes or field values" as the
   platform's committed posture, not a conservative default.

7. **Foundation 05's "samples_purged: true" two-tier scheme
   is deferred.** v1 ships single-tier retention (samples
   share the row's retention). The two-tier scheme is
   reserved for a future amendment when concrete operational
   signal (an operator request for longer aggregate history
   without sample bytes) justifies the implementation cost.

8. **No automated PII detection ships.** Rule authors +
   CODEOWNERS reviewers are the v1 gate. The platform does
   not lint for "this column looks like an email"; that
   detection capability is a future enhancement reserved for
   a B2 row if and when a concrete need surfaces.

9. **The cost-discipline ADRs are unchanged.** ADR-0027
   (`RecordModeCost.MaxEvidenceSampleSize`,
   `SampleStorageCapMB`) and ADR-0029
   (`SetModeCost.MaxEvidenceSampleSize`) commit the SIZE
   ceilings; this row commits the DURATION + CONTENT
   constraints. The three together compose the full
   evidence cost-discipline surface.

10. **B2 follow-up: pre-existing-table migration runbook.**
    Today the only deployed dataset is `dq-local` (wiped
    routinely); qa/prod datasets are `PLACEHOLDER`-named and
    provisioned green-field via the new `EnsureSchema`. A
    future deployment that has accumulated non-partitioned
    history before this row lands would need a one-time
    migration (`CREATE TABLE ... AS SELECT ... PARTITION BY
    DATE(recorded_at)`, then rename). A B2 row registers the
    migration runbook for that case; v1 deployments do not
    need it. The B2 row is added at close-step assignment of
    a number.

11. **The platform's P3 + P4 commitments for evidence are
    now explicit.** P3 (ownership): rule authors + CODEOWNERS
    + operators each carry a named responsibility. P4 (cost):
    retention caps the unbounded-growth path that append-only
    would otherwise create.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting are explicitly
**out-of-scope for current cycle**:

- **OQ-1: Automated PII detection at lint time.** A future
  lint extension could scan rule artefacts for primary-key
  columns or check-relevant columns whose names match common
  PII patterns (`email`, `ssn`, `phone`, etc.) and refuse
  rules that would publish such columns into samples. The
  detection is heuristic, not exhaustive; v1 leans on rule
  author + CODEOWNERS judgment. Deferred until concrete
  signal (an incident; an audit-finding; a contributor
  request) justifies the heuristic surface.

- **OQ-2: Two-tier sample-vs-row retention (foundation 05's
  `samples_purged: true` scheme).** A future amendment could
  ship a separate `tools/retention` binary that issues
  `UPDATE ... SET sample_violating_rows = NULL` for rows
  older than a `SampleRetention` window and `DELETE` for
  rows older than `ResultsRetention`. The tool would honor
  ADR-0003 CC1 (engine code stays append-only; retention is
  a separate actor). Deferred until concrete operational
  signal (an operator request for longer aggregate result
  history without sample bytes) justifies the tool's
  build/test/CI cost.

---

## Promotion target

`docs/adr/0031-evidence-retention-parameters.md` — ships the
`EnvConfig.EvidenceRetention` sub-struct + per-env defaults,
the `EnsureSchema` extension that sets BigQuery partition
expiration, the five-category `SampleContentAllowlist`, the
rule-author / CODEOWNERS / operator / platform responsibility
ladder, and the two deferred enhancements
(automated PII detection; two-tier sample-vs-row retention).
The security note at `docs/security/evidence-retention.md`
ships alongside as the operator-facing artefact (forward-only
prose; no back-link into `studies/`).
