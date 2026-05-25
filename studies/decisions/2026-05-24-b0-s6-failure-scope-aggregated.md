<!-- path: studies/decisions/2026-05-24-b0-s6-failure-scope-aggregated.md -->

# B0-S6 — Failure Scope Aggregated

## Metadata

- **B-item reference:** B0-S6 (Wave-S Phase β, item 3 of 4)
- **Status:** resolved-study (Wave-S, B0-S6; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-24
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch); foundational triplet
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) /
  [ADR-0022](../../docs/adr/0022-kind-catalog.md) /
  [ADR-0023](../../docs/adr/0023-sources-schema.md);
  [ADR-0024](../../docs/adr/0024-window-semantics.md) (tumbling
  windows + record-mode `execution_id`);
  [ADR-0025](../../docs/adr/0025-aggregation-and-runner-shape.md)
  (aggregation seam at kind handler; parallel runners; same
  `dq_executions` schema with `mode` column);
  [ADR-0004](../../docs/adr/0004-failure-scope.md) (set-mode
  status enum: pass / fail / degraded / error);
  [ADR-0003](../../docs/adr/0003-result-write-model.md)
  (append-only write model; evidence policy).
- **Downstream open:** B0-S7 (cost guardrails — consumes the
  evidence-sample-size shape committed here as a cost dimension);
  B1-6 (evidence retention parameters — open at B1; amends
  retention windows for record-mode evidence per the shape
  committed here).
- **Promotion target:** `docs/adr/0026-failure-scope-aggregated.md`
  (subject to ADR-0020 §Decision (Per-item ADR numbering); `0026`
  assumes ADR-0010 amendment ADR has not interleaved).
- **Loop discipline:** same as B0-S1–S5 — `/resolve-b0` study →
  `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.
- **Significance:** B0-S6 is the **third Phase β study**. It fills
  the aggregation function that ADR-0025 committed only the seam
  for, and resolves OQ-B0S5.5 (per-kind vs per-mode aggregation
  policy granularity).

---

## Context

ADR-0025 committed that **aggregation happens inside the kind
handler** when a window closes — the record-mode runner invokes
the handler with the per-window batch of records, and the handler
returns one `CheckResult` per check, with per-record violations
in evidence. ADR-0025 §C-B0S5.5 explicitly left the **aggregation
function** to this study: how N per-record outcomes within one
window map to a single ADR-0004 status (`pass` / `fail` /
`degraded` / `error`).

The set-mode side of ADR-0004 is unchanged: a set-mode check
(e.g., `set.row_count_positive`) runs one BigQuery query per
window and produces one outcome directly. No aggregation needed.

The record-mode side needs aggregation by construction:

- `record.schema_conformance` (the inaugural record kind per
  ADR-0022) validates each record against a JSON Schema fragment.
  In a one-minute tumbling window (per ADR-0024) over a topic
  emitting 100 records/sec, that's ~6,000 records, each yielding
  a per-record pass/fail. Without an aggregation function, the
  handler cannot return a single `CheckResult`.

ADR-0020 §B0-S6 commits the scope:

> Decides how per-record failures aggregate into an entity-level
> status, given that record-mode lacks the natural batch boundary
> that ADR-0004 currently relies on; how the ADR-0004 status
> policy (`pass` / `fail` / `error` / `degraded`) maps onto a
> continuous stream of records (windowed rollup, sliding-fraction
> threshold, per-watermark aggregation); whether per-record
> evidence is retained, sampled, or dropped (B1-6 retention
> parameters need a record-mode amendment).

Four interlocking sub-decisions live inside B0-S6's scope:

1. **The aggregation function shape** — threshold-based,
   all-must-pass, aggregate-evidence-only, or something else.
2. **Where the aggregation policy lives** — per-kind in the
   catalog, per-mode (one rule for all record kinds), per-rule, or
   hybrid. ADR-0025 OQ-B0S5.5 deferred this; B0-S6 commits it.
3. **How the aggregation outcome maps to ADR-0004's four-state
   enum** — pass/fail/degraded/error in the record-mode context.
4. **Per-record evidence retention shape** — how many per-record
   violations land in `dq_executions` evidence per window; how
   B1-6 (currently open at B1) eventually amends retention
   windows for record-mode.

The platform principles bearing on the design: **P1**
(declarative — aggregation policy and overrides live in the
catalog and the rule, not in engine code); **P2 / foundation 01
§"Determinism"** (same input → same status output, given the
same aggregation policy); **P3** (status is derived from data,
not declared per-entity); **P4 mirror, foundation 01
§"Cost"** (evidence-sample-size is a cost dimension — bound it
explicitly).

---

## Decision Drivers

- **DD-S6.1** — **Honour ADR-0004's status enum exactly.**
  Record-mode `dq_check_results` rows write the same four-state
  status (`pass` / `fail` / `degraded` / `error`) as set-mode.
  No new status values are added. The aggregation function maps
  per-window per-record outcomes into one of those four states.

- **DD-S6.2** — **Honour ADR-0025's seam location.** Aggregation
  happens inside the kind handler, per ADR-0025 §C-B0S5.4. The
  runner invokes the handler with the per-window batch; the
  handler aggregates and returns one `CheckResult` per check; the
  runner writes one `dq_check_results` row per check. B0-S6
  commits what the handler does inside that envelope.

- **DD-S6.3** — **Per-kind aggregation policy.** ADR-0025
  OQ-B0S5.5 raised the question of per-kind vs per-mode. Per-kind
  is the right answer for two reasons: (a) different kinds have
  different natural failure semantics (`record.schema_conformance`
  is binary per-record; a future `record.field_value_in_range`
  kind might be threshold-on-numeric-deviation; a future
  `record.unique_id_within_window` kind might be "any duplicate
  fails"), (b) per-mode policy is too coarse for the platform's
  long-term shape — recording per-mode would foreclose
  per-kind nuance the catalog is built to express. *(New
  contribution proposed here, requires review.)*

- **DD-S6.4** — **Threshold-based aggregation with operator-
  configurable severity bands.** ADR-0004's `degraded` state is
  defined as "data fell into a warning band (between pass and
  fail)". For record-mode, the natural warning band is a
  violation-rate range — e.g., 0% violations passes, ≥5% fails,
  1–5% degrades. Threshold-based aggregation with two operator-
  configurable rates (`warn_if_violation_rate`,
  `fail_if_violation_rate`) maps directly onto ADR-0004's enum.
  Other aggregation shapes (all-must-pass, aggregate-evidence-
  only) are special cases of threshold-based with specific
  parameter values (all-must-pass = `fail_if_violation_rate: 0.0`
  with `warn_if_violation_rate: null`).

- **DD-S6.5** — **Catalog declares the policy type and defaults;
  rule's `params` overrides tunable parameters.** ADR-0022's
  catalog already declares per-kind `params_schema`. B0-S6
  extends per-kind catalog entries with an `aggregation` block
  (type + defaults). The rule's `params` may override the
  aggregation defaults within the schema the catalog declares.
  Authors can tune severity per entity without modifying the
  catalog. *(New contribution proposed here, requires review.)*

- **DD-S6.6** — **`error` status maps to handler runtime
  errors, not data outcomes.** ADR-0004's `error` status is
  "the check's query did not execute successfully". For
  record-mode, `error` covers handler runtime errors (panic,
  schema parse failure, unbounded recursion in the validator),
  not per-record violations. A 100% violation rate maps to
  `fail`, not `error` — the handler executed successfully; the
  data failed.

- **DD-S6.7** — **Evidence is a bounded sample, not the full
  violation set.** Per ADR-0003's evidence-retention policy (the
  evidence field on `dq_executions` is bounded; the current
  scope-noted set-mode shape carries a fixed-size sample). For
  record-mode, the handler samples up to `evidence_sample_size`
  per-record violations into evidence; the rest is dropped
  (not retained). This is a v1 choice that B1-6 (evidence
  retention parameters) may refine.

- **DD-S6.8** — **B1-6 inherits a sample-size default from this
  study; B1-6 commits retention durations and privacy.** B1-6 is
  open at the B1 level; its scope is the **retention duration**
  (how long evidence rows stay in `dq_executions`) and **privacy
  constraints** (which fields can be sampled). B0-S6 commits the
  **shape** (a per-kind sample size with operator override); B1-6
  commits the **duration and privacy bounds** in a future study.
  *(New contribution proposed here, requires review — B0-S6
  pre-scopes B1-6's eventual amendment domain.)*

- **DD-S6.9** — **Late-dropped count from ADR-0024 surfaces in
  evidence, not in status.** ADR-0024 commits that late-arrival
  records are dropped and counted as `late_dropped_count` in
  evidence. The dropped count does **not** participate in the
  aggregation function's violation rate — only records that were
  actually evaluated count. If lateness becomes severe enough to
  threaten data quality, that signal lives in B0-S7's cost
  guardrails (consumer lag, lateness rate), not in B0-S6's
  per-check aggregation. *(New contribution proposed here,
  requires review.)*

---

## Considered Options

The four options below differ on **where the aggregation policy
lives** and **what shape it takes**. All four assume the seam is
at the kind handler per ADR-0025 §C-B0S5.4 (locked); the variation
is policy location and policy shape.

### Option A — Per-kind threshold-based, with rule-level overrides (recommended)

**Shape.** Each catalog entry declares an `aggregation` block
with a `type` (e.g., `none` for set-mode kinds that produce one
result per window directly; `threshold` for record-mode kinds
that need rate-based aggregation) and `defaults` (a JSON-Schema
fragment that the rule's `params` may override).

```yaml
# Catalog entries (R1: code-shaped illustration only)
- name: set.row_count_positive
  mode: set
  source_mode: set
  params_schema:
    type: object
    properties: {}
  aggregation:
    type: none                          # one result per window directly
  description: Verifies the source has at least one row. No parameters.

- name: record.schema_conformance
  mode: record
  source_mode: record
  params_schema:
    type: object
    required: [schema]
    properties:
      schema:
        description: JSON Schema fragment to validate each record against.
        type: object
      aggregation:                      # operator overrides (optional)
        type: object
        properties:
          fail_if_violation_rate:
            type: number
            minimum: 0.0
            maximum: 1.0
          warn_if_violation_rate:
            type: [number, "null"]
            minimum: 0.0
            maximum: 1.0
          evidence_sample_size:
            type: integer
            minimum: 0
  aggregation:
    type: threshold
    defaults:
      fail_if_violation_rate: 0.0       # strict by default
      warn_if_violation_rate: null      # no degraded by default
      evidence_sample_size: 10
  description: >
    Validates each record against a JSON Schema fragment passed
    via params.schema. Per-record evaluation; aggregation via
    threshold on violation rate within the window.
```

A rule may override defaults via `params.aggregation`:

```yaml
# Rule overriding default thresholds (illustration)
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema: { type: object, required: [id, event_type] }
      aggregation:
        fail_if_violation_rate: 0.05      # 5% fails
        warn_if_violation_rate: 0.01      # 1% degrades
        evidence_sample_size: 50
```

The aggregation function:
- Compute `violation_rate = violations / records_evaluated` for
  the window, where `records_evaluated` is the count of records
  the handler processed (late-arrival records are excluded by
  ADR-0024's window-close semantics before reaching the handler).
- If `violation_rate >= fail_if_violation_rate` → `fail`.
- Else if `warn_if_violation_rate` is set and
  `violation_rate >= warn_if_violation_rate` → `degraded`.
- Else → `pass`.
- If the handler panics or the schema fails to parse → `error`
  (status from ADR-0004's runtime-error path; bypasses aggregation
  entirely).

**Cost.** Catalog grows by one `aggregation` block per kind. Rule
YAML grows by an optional `aggregation` override block when
authors need per-entity tuning. Lint validates the override against
the catalog's `defaults` schema (existing cross-check #6 from
ADR-0022 covers `params` validation; the `aggregation` override
sits inside `params` and is covered by the same check).

**Verdict.** Recommended.

### Option B — All-must-pass (one failure = window fails)

**Shape.** No catalog `aggregation` block. The handler always
returns `fail` if any per-record violation occurs in the window,
`pass` otherwise. No degraded state. No operator configuration.

**Cost.** Catastrophic for `record.schema_conformance` over noisy
streams: any one malformed record (e.g., a test record, a buggy
producer release) fails every window forever until the producer
is fixed. The lack of `degraded` collapses operational signal —
operators cannot distinguish "1 in 10,000" from "5,000 in 10,000".

**Verdict.** Rejected. Too strict for stream sources where some
data noise is operationally normal; eliminates `degraded`
contrary to ADR-0004's four-state enum.

### Option C — Aggregate-evidence-only (handler discretion, no schema-level threshold)

**Shape.** The catalog declares no aggregation policy. Each
handler decides internally how to map per-record outcomes to a
status, and the policy is implicit in the handler's code.

**Cost.** Hides policy in code. Authors of new record kinds
must learn the handler's decision rule. Lint cannot cross-check
overrides. Different kinds may pick subtly different policies,
fragmenting platform behaviour. Violates P1 (declarative — the
aggregation rule should be visible at the rule-authoring surface).

**Verdict.** Rejected. Reproduces the "kinds dispatch in switch
statements" antipattern that ADR-0022 explicitly retired.

### Option D — Per-mode global policy (one threshold for all record kinds)

**Shape.** Engine-config-level setting declares one aggregation
policy for all record-mode kinds. No per-kind variation; no
rule-level overrides.

**Cost.** Foreclosing per-kind nuance forces a single policy on
kinds with structurally different failure semantics
(schema_conformance is binary per-record; a future
field_value_in_range might be threshold-on-numeric-deviation;
unique_id_within_window might be "any duplicate fails"). One
policy cannot fit all. Authors who add a new kind cannot adjust
the aggregation without engine-config changes — outside the
declarative DSL.

**Verdict.** Rejected. Per-kind expressiveness is the value
ADR-0022's catalog architecture exists to deliver.

---

## Recommendation

**Pick Option A — Per-kind threshold-based with rule-level
overrides.**

Rationale, tied directly to drivers:

- **DD-S6.1 (ADR-0004 enum honoured).** Threshold-based mapping
  produces `pass` / `fail` / `degraded` directly; `error`
  comes from the existing ADR-0004 runtime-error path. No new
  status values.
- **DD-S6.2 (ADR-0025 seam preserved).** Aggregation happens
  inside the kind handler, returning one `CheckResult` per
  check. The runner's view is unchanged from ADR-0025.
- **DD-S6.3 (per-kind aggregation).** Resolves ADR-0025
  OQ-B0S5.5 with per-kind policy declared in the catalog.
- **DD-S6.4 (threshold-based).** Maps cleanly onto ADR-0004's
  four-state enum; supports all-must-pass as a parameterisation
  (`fail_if_violation_rate: 0.0`); supports operator-configurable
  severity bands.
- **DD-S6.5 (catalog + override).** Catalog declares policy type
  + defaults; rule overrides tunable parameters. Lint cross-check
  #6 from ADR-0022 validates overrides via the catalog's
  `params_schema`.
- **DD-S6.6 (`error` semantics).** Handler runtime errors map to
  `error`; data-driven outcomes map to `pass` / `fail` /
  `degraded`.
- **DD-S6.7 (bounded evidence sample).** Per-kind default
  `evidence_sample_size` with rule override; further bounded by
  B1-6 retention windows when those are decided.
- **DD-S6.8 (B1-6 scoping).** Sample-size shape committed here;
  retention duration and privacy committed in B1-6 future study.
- **DD-S6.9 (late-dropped is cost, not status).** Late-dropped
  records do not enter the aggregation; B0-S7 surfaces lateness
  signals via cost guardrails.

### The aggregation function (record-mode threshold)

```
records_evaluated  = count of records the handler processed for this window
violations         = records_evaluated - records_passed
late_dropped_count = (per ADR-0024 — records arrived after watermark closed the window)

if records_evaluated == 0:
    if late_dropped_count == 0:
        status = "pass"              # vacuous: no data arrived, nothing to evaluate
    else:
        status = "degraded"          # late-drop catastrophe: all in-window data was late
else:
    violation_rate = violations / records_evaluated
    if violation_rate >= fail_if_violation_rate:
        status = "fail"
    elif warn_if_violation_rate is not null
         and violation_rate >= warn_if_violation_rate:
        status = "degraded"
    else:
        status = "pass"

if handler-runtime-error:
    status = "error"                 # overrides above per ADR-0004
```

The vacuous case splits on `late_dropped_count`: a window with
no records evaluated AND no late drops produces `pass` (vacuous:
no data arrived, nothing to evaluate); a window with no records
evaluated BUT positive late drops produces `degraded` (the
late-drop catastrophe — all in-window data arrived after the
watermark closed the window). The `degraded` outcome surfaces
the issue without violating DD-S6.9's architectural separation
from B0-S7 (B0-S7 still owns the lateness-rate alerting and
cost guardrails). ADR-0024's idle-topic rejection ensures the
`pass` case is rare in practice; the split keeps the status
enum semantically honest with what the data shows.

### Per-attempt re-aggregation semantics

Per ADR-0002 retry semantics (extended to record-mode under
ADR-0024 and ADR-0025), a window may produce multiple attempts.
Each attempt **re-aggregates from scratch** — each attempt is a
fresh consumer-group read of the same offset range, producing
its own per-record outcomes and its own aggregation result.
Attempts do not share intermediate aggregation state; the
`dq_check_results` row from attempt N is independent of any
prior attempt. This matches set-mode's per-attempt semantics
from ADR-0002 / ADR-0003 — each attempt is a fresh evaluation
under the same `execution_id`.

### Catalog `aggregation` block — v1 enum values

- `none` — set-mode kinds; one result per window directly; no
  aggregation. Set-mode catalog entries from ADR-0022 use this.
- `threshold` — record-mode kinds with rate-based aggregation;
  carries `fail_if_violation_rate`, `warn_if_violation_rate`,
  `evidence_sample_size` defaults; rules override via
  `params.aggregation`.

Future aggregation types extend additively (e.g.,
`count_threshold` for absolute-count-based aggregation;
`sliding_rate` for sliding-window aggregation tied to future
B0-S4 enum extensions); the type enum is open.

**Cross-validation note.** The catalog's per-kind
`aggregation.defaults` declares the policy defaults; the
catalog's per-kind `params_schema.properties.aggregation`
declares the override shape. By author convention, these two
declarations must agree on the override field set — a
`defaults` key without a corresponding
`params_schema.properties.aggregation` property means the field
is not overridable; an override property without a corresponding
`defaults` key means the override has no floor. A future lint
cross-check (deferred) could enforce this consistency
automatically; for now, the catalog reviewer ensures alignment
per ADR-0015's joint engine-maintainers + rules-authors review.

### Evidence sample shape

Per `dq_executions` evidence (per ADR-0003):

- `evidence.records_evaluated` — count
- `evidence.records_passed` — count
- `evidence.violations` — count
- `evidence.violation_rate` — float (derived)
- `evidence.late_dropped_count` — count (per ADR-0024)
- `evidence.sample_violations` — array of up to
  `evidence_sample_size` per-record violation descriptors. Each
  descriptor carries the record's offset (Kafka offset) and a
  handler-specific violation reason (e.g., "missing required
  field 'id'"). Privacy-sensitive fields are not included by
  default; B1-6 commits the privacy bounds.

### Lint cross-checks added

This study adds **no new lint cross-checks**. The catalog
`aggregation` block validates via the existing JSON Schema
mechanism; cross-check #6 (per-kind params validation, from
ADR-0022) covers rule-level overrides because they sit inside
`params`. Ten cross-checks total remain from ADRs 0021–0024.

### One-line decision summary table

| Decision | Outcome |
|---|---|
| Aggregation function | Threshold-based on violation rate (records_evaluated excluding late-dropped) |
| Status mapping | `pass` / `fail` / `degraded` from threshold; `error` from handler runtime errors (per ADR-0004) |
| Policy location | Per-kind in catalog (defaults); per-rule overrides via `params.aggregation` |
| Catalog `aggregation` type enum (v1) | `none` (set-mode), `threshold` (record-mode) |
| `record.schema_conformance` defaults | `fail_if_violation_rate: 0.0`, `warn_if_violation_rate: null`, `evidence_sample_size: 10` |
| Vacuous case (zero records evaluated) | `pass` (idle topics are rejected by ADR-0024) |
| Evidence shape | Counts + bounded sample of per-record violation descriptors |
| B1-6 interaction | This study commits sample-size shape; B1-6 commits retention duration + privacy |
| Late-dropped records | Excluded from aggregation; signalled via B0-S7 cost guardrails |
| Lint cross-checks added | None (existing cross-check #6 covers overrides) |

---

## Consequences

### Cross-cutting consequences

- **C-B0S6.1** — **ADR-0022's catalog v1 design gains a required
  `aggregation` block per entry.** Each catalog entry — including
  the existing `set.row_count_positive` and
  `record.schema_conformance` v1 entries — declares an
  `aggregation: { type, defaults }` block. This is an
  **extension to ADR-0022's v1 catalog design** (R3 does not
  fully bite, because catalog v1 has not yet shipped to disk;
  the combined implementation commit lands ADRs 0021–0026
  artefacts together). Existing catalog entries gain
  `aggregation: { type: none }` for set-mode and
  `aggregation: { type: threshold, defaults: {...} }` for
  record-mode at the combined commit. Same pacing pattern as
  ADR-0022 §C-B0S2.2 (params field extension) and ADR-0023
  §C-B0S3.1 (source field extension); if any v1-catalog artefact
  ships to disk before the combined commit lands, a v2 catalog
  bump is required to add the `aggregation` field. *(New
  contribution proposed here, requires review.)*

- **C-B0S6.2** — **Rule schema v2's `params` block accepts an
  optional `aggregation` override.** The
  `record.schema_conformance` entry's `params_schema` gains an
  optional `aggregation` property with the override fields. This
  is a second **extension to ADR-0022's v1 catalog design**
  (same pacing as C-B0S6.1: catalog v1 has not shipped, so the
  `params_schema` content for `record.schema_conformance` is
  amended in place before shipping; if catalog v1 ships first, a
  v2 catalog bump applies). Authors may override
  `fail_if_violation_rate`, `warn_if_violation_rate`, and
  `evidence_sample_size` per check. Other record kinds will
  declare their own override shapes in their catalog entries.

- **C-B0S6.3** — **ADR-0004's status enum is unchanged.** Record-
  mode uses the same four-state enum (`pass` / `fail` /
  `degraded` / `error`). The threshold-based aggregation function
  maps per-window per-record outcomes into the existing states; no
  ADR-0004 reopening. The set-mode side of ADR-0004 remains
  scope-noted as set-oriented; this ADR holds the record-mode
  extension of the status-mapping policy.

- **C-B0S6.4** — **B1-6 inherits a sample-size shape.** This
  study commits the per-kind `evidence_sample_size` default
  (`record.schema_conformance` = 10) and the per-rule override
  path. B1-6 (currently open at B1) commits the **retention
  duration** for `dq_executions` evidence rows and the **privacy
  bounds** for sampled violation descriptors. B1-6's eventual
  resolution amends the policy without reopening this ADR.

- **C-B0S6.5** — **Late-dropped records (per ADR-0024) are
  excluded from the aggregation function.** `records_evaluated`
  is the count of records actually evaluated within the closed
  window, not the count of records that arrived. The
  `late_dropped_count` from ADR-0024 surfaces in evidence and in
  B0-S7's cost guardrails (forthcoming), not in B0-S6's status
  determination.

- **C-B0S6.6** — **The vacuous case splits on
  `late_dropped_count`.** A window that closes with zero
  records evaluated AND zero late drops produces `pass`
  (vacuous: no data arrived). A window that closes with zero
  records evaluated BUT positive late drops produces `degraded`
  (late-drop catastrophe — all in-window data arrived after the
  watermark closed the window). The split keeps the status
  enum semantically honest. ADR-0024's idle-topic rejection
  ensures the `pass` case is rare in practice; the `degraded`
  case surfaces the late-drop signal without violating DD-S6.9's
  architectural separation from B0-S7 (B0-S7 still owns the
  lateness-rate alerting and cost guardrails). *(New
  contribution proposed here, requires review.)*

- **C-B0S6.7** — **The catalog `aggregation.type` enum is open
  for additive extension.** Future record kinds that require
  different aggregation shapes (e.g., absolute-count thresholds
  for `record.duplicate_count_within_window`, sliding-window
  aggregations for kinds that consume B0-S4's eventual sliding
  enum extension) extend the type enum additively. The enum
  follows the same additive-within-major pattern as ADR-0001
  governs.

- **C-B0S6.8** — **No new lint cross-checks.** The catalog's
  `aggregation.defaults` JSON-Schema fragment + the rule's
  `params.aggregation` override are validated by the existing
  cross-check #6 from ADR-0022 (per-kind params validation). Ten
  lint cross-checks total remain from ADRs 0021–0024.

- **C-B0S6.9** — **B0-S7 (cost guardrails) inherits the
  evidence-sample-size as a cost dimension.** Each retained
  violation descriptor consumes storage; B0-S7 commits per-runner
  storage and lag budgets that bind to (among other dimensions)
  the configured evidence-sample-size. Larger samples = higher
  storage cost = stricter B0-S7 budgets.

### Per-artefact consequences

- **`engine/internal/dsl/catalog/v1.yaml`** (per ADR-0022) — both
  existing entries gain an `aggregation` block.
  `set.row_count_positive` gains `aggregation: { type: none }`.
  `record.schema_conformance` gains `aggregation: { type: threshold,
  defaults: {...} }`. The byte-equal mirror at
  `rules/_schema/catalog.v1.yaml` (per ADR-0022 §C-B0S2.1) follows.

- **`engine/internal/dsl/schema/v2.schema.json`** (per ADR-0021 +
  ADR-0022 extensions) — the `record.schema_conformance` entry's
  `params_schema` (in the catalog, not the rule schema itself)
  gains the optional `aggregation` property. The rule schema v2
  is unchanged; per-check `params` validation flows through the
  catalog per ADR-0022's existing dispatch.

- **`engine/internal/eval/record_schema_conformance.go`** (the
  handler from ADR-0022) — when implemented in the combined
  implementation commit, reads its `aggregation` policy from
  the catalog entry's `defaults`, merges in any rule-level
  override from `params.aggregation`, runs the threshold-based
  aggregation function on the per-window batch, and returns one
  `CheckResult` per check with the bounded violation sample in
  evidence.

- **`docs/adr/0004-failure-scope.md`** — scope-noted as
  set-oriented. The record-mode status-mapping extension lives
  in this ADR (ADR-0026); ADR-0004 is not reopened.

- **`docs/adr/0003-result-write-model.md`** — scope-noted as
  set-oriented. The evidence-shape additions (`sample_violations`
  array; `late_dropped_count` from ADR-0024) live in this ADR
  and ADR-0024 respectively; ADR-0003 is not reopened.

- **No changes to ADR-0007, ADR-0014, ADR-0006, ADR-0010
  contracts.** All set-mode and runtime contracts hold.

- **No new lint cross-checks.** The lint binary remains at ten
  cross-checks.

---

## Open Questions

- **OQ-B0S6.1** — **Privacy-sensitive fields in evidence
  samples.** B1-6 will commit privacy bounds (which fields can
  be sampled; field-level redaction rules). Until B1-6 lands,
  the v1 implementation **omits all record content from
  `sample_violations`** — each descriptor carries only the
  Kafka offset and a handler-specific violation reason string
  (e.g., "missing required field 'id'"). No raw record bytes,
  no field values. *Out of scope for current cycle;* this v1
  default is the conservative path until B1-6 resolves.

- **OQ-B0S6.2** — **Sliding-window aggregation for future
  kinds.** If B0-S4's `window.type` enum extends to `sliding`
  (per ADR-0024's deferred extensibility), aggregation under
  sliding windows requires its own type — likely `sliding_rate`
  — that B0-S6 does not commit. *Out of scope for current
  cycle;* the v1 `threshold` type covers tumbling windows
  (B0-S4's v1 commitment); sliding aggregation lands when
  sliding windows do.

- **OQ-B0S6.3** — **Pluggable aggregation functions vs hardcoded
  catalog enum.** The catalog's `aggregation.type` is a closed
  enum in v1. A future direction allows operators to register
  pluggable aggregation functions per kind (similar to how
  handlers are registered today). This is a richer extension
  mechanism than additive enum growth. *Out of scope for current
  cycle;* additive enum growth is sufficient until concrete
  operational signal motivates plugins.

- **OQ-B0S6.4** — **Aggregation execution mechanics across retry
  attempts.** Per §Decision (Per-attempt re-aggregation
  semantics), each attempt re-aggregates from scratch by
  re-reading the same offset range. The implementation choice
  of how to organise that re-read — single consumer-group reset
  per attempt; per-attempt consumer-group; snapshot-and-replay
  mechanism — is a runtime-engineering question that does not
  affect the per-attempt semantics this study commits.
  *Defer to the combined implementation commit;* the v1
  implementation picks the simplest path consistent with
  ADR-0007's loader semantics and ADR-0024's window-close
  trigger.

- **OQ-B0S6.5** — **`error` granularity for partial handler
  failures.** Today, `error` is all-or-nothing per ADR-0004.
  Whether a handler that processes 9,990 records successfully
  but errors on the last 10 should produce `error` (current
  semantics) or `degraded` (with evidence of partial success)
  is a future enrichment. *Out of scope for current cycle;*
  the v1 semantics match set-mode: any runtime error → `error`.

- **OQ-B0S6.6** — **Alert routing implications for `degraded`
  on record-mode.** ADR-0006 commits per-attempt dedup for
  alerts. Whether a `degraded` outcome on a record-mode window
  generates a different alert category than `fail` (currently
  both surface as data_quality alerts per ADR-0006 §"category
  mapping") is an alerting-design question. *Out of scope for
  current cycle;* both `fail` and `degraded` map to
  `data_quality` per ADR-0006's existing policy, matching
  set-mode treatment.

---

## Promotion target

**Target:** `docs/adr/0026-failure-scope-aggregated.md`.

This study promotes to **ADR-0026** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0026 is the third Phase β ADR of
Wave-S (after ADR-0024 windowing and ADR-0025 runner shape).
Per ADR-0020 §Decision (Per-item ADR numbering), the `0026` slot
is descriptive; if the forthcoming ADR-0010 amendment (Kafka
substrate-matrix row, flagged by ADR-0023 §C-B0S3.3) lands
first at the `0026` slot, this study promotes to `0027` and the
per-item slugs shift in lockstep.

ADR-0026's promotion commit lands the artefacts committed in
§Consequences above:

1. The catalog `aggregation` block per entry (both existing
   entries gain it; `set.row_count_positive` = `none`,
   `record.schema_conformance` = `threshold` with defaults).
2. The `record.schema_conformance` `params_schema` extension
   for the optional `aggregation` override.
3. The handler implementation (in the combined implementation
   commit per ADR-0025 §C-B0S5.11) reads the aggregation policy
   and computes the threshold-based aggregation function.
4. The evidence-shape extension on `dq_executions` (carries the
   counts and bounded `sample_violations` array — per ADR-0003's
   existing evidence field).

Per R8, the future ADR-0026 will be rewritten from this study,
not linked back to it.
