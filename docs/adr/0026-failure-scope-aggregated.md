<!-- path: docs/adr/0026-failure-scope-aggregated.md -->

# ADR-0026 — Failure Scope Aggregated (Record-Mode Status Mapping)

- **Status:** accepted
- **Date:** 2026-05-24

---

## Context

ADR-0025 committed that **aggregation happens inside the kind
handler** when a window closes — the record-mode runner invokes
the handler with the per-window batch of records, and the handler
returns one `CheckResult` per check. ADR-0025 §C-B0S5.4
explicitly left the **aggregation function** (how N per-record
outcomes within a window map to a single ADR-0004 status) to
this ADR.

The set-mode side of [ADR-0004](./0004-failure-scope.md) is
unchanged: a set-mode check (e.g., `set.row_count_positive`) runs
one BigQuery query per window and produces one outcome directly.
No aggregation needed.

The record-mode side needs aggregation by construction. The
inaugural record kind (`record.schema_conformance` per
[ADR-0022](./0022-kind-catalog.md)) validates each record against
a JSON Schema fragment; in a one-minute tumbling window (per
[ADR-0024](./0024-window-semantics.md)) over a topic emitting
100 records/sec, that is ~6,000 records, each yielding a
per-record pass/fail. Without an aggregation function, the
handler cannot return a single `CheckResult`.

The four interlocking sub-decisions resolved by this ADR:

1. **The aggregation function shape** — threshold-based on
   violation rate.
2. **Where the aggregation policy lives** — per-kind in the
   catalog, with per-rule overrides via `params.aggregation`
   (resolves ADR-0025 §OQ-B0S5.5).
3. **How the aggregation outcome maps to ADR-0004's four-state
   enum** — `pass` / `fail` / `degraded` from the threshold;
   `error` from handler runtime errors.
4. **Per-record evidence retention shape** — counts plus a
   bounded sample of per-record violations.

The platform principles bearing on the design: **P1**
(declarative — aggregation policy and overrides live in the
catalog and the rule, not in engine code); **P2 / foundation 01
§"Determinism"** (same input → same status output given the same
aggregation policy); **P3** (status is derived from data, not
declared per-entity); **foundation 01 §"Cost"** (evidence sample
size is a cost dimension — bound it explicitly).

---

## Decision

### Per-kind threshold-based aggregation with rule-level overrides

Each catalog entry declares an `aggregation` block with a `type`
field and a `defaults` block. v1 supports two type values:

- **`none`** — for set-mode kinds that produce one result per
  window directly (no aggregation). The existing
  `set.row_count_positive` entry from ADR-0022 declares this
  type.
- **`threshold`** — for record-mode kinds that aggregate
  per-record outcomes via violation rate. The
  `record.schema_conformance` entry declares this type.

The catalog v1 entries — both `set.row_count_positive` and
`record.schema_conformance` — gain the `aggregation` block as an
**extension to ADR-0022's v1 catalog design** (the catalog v1
file has not yet shipped to disk; the combined implementation
commit lands ADRs 0021–0026 artefacts together, so the v1
catalog is amended in place before shipping). Same pacing
pattern as ADR-0022 §C-B0S2.2 (params field extension) and
ADR-0023 §C-B0S3.1 (source field extension); if any v1-catalog
artefact ships to disk before the combined commit, a v2 catalog
bump is required.

```yaml
# Catalog entries (illustration)
- name: set.row_count_positive
  mode: set
  source_mode: set
  params_schema:
    type: object
    properties: {}
  aggregation:
    type: none

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
```

A rule may override the defaults via `params.aggregation`:

```yaml
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

The catalog's per-kind `params_schema.properties.aggregation`
property is a parallel extension to ADR-0022's v1 catalog design
(same pacing as the `aggregation` block above; v2 catalog bump
contingency if v1 ships first).

**Cross-validation note.** The catalog's per-kind
`aggregation.defaults` declares the policy defaults; the
catalog's per-kind `params_schema.properties.aggregation`
declares the override shape. By author convention, these two
declarations must agree on the override field set — a `defaults`
key without a corresponding `params_schema.properties.aggregation`
property means the field is not overridable; an override
property without a corresponding `defaults` key means the
override has no floor. A future lint cross-check (deferred)
could enforce this consistency automatically; for now, the
catalog reviewer ensures alignment per
[ADR-0015](./0015-codeowners.md)'s joint engine-maintainers +
rules-authors review.

The catalog `aggregation.type` enum is open for additive
extension. Future record kinds that require different
aggregation shapes (e.g., `count_threshold` for absolute-count-
based aggregation; `sliding_rate` for sliding-window aggregation
tied to future B0-S4 enum extensions) extend the type enum
additively, following ADR-0001's additive-within-major contract.

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

**Vacuous case split.** A window with no records evaluated AND
no late drops produces `pass` (vacuous: no data arrived). A
window with no records evaluated BUT positive late drops
produces `degraded` (the late-drop catastrophe — all in-window
data arrived after the watermark closed the window). The
`degraded` outcome surfaces the late-drop issue without
collapsing the architectural separation from B0-S7 (B0-S7 still
owns the lateness-rate alerting and cost guardrails). ADR-0024's
idle-topic rejection ensures the `pass` case is rare in
practice; the split keeps the status enum semantically honest
with what the data shows.

**`error` semantics.** ADR-0004's `error` status is "the check's
query did not execute successfully". For record-mode, `error`
covers handler runtime errors (panic, schema parse failure,
unbounded recursion in the validator) — not per-record
violations. A 100% violation rate maps to `fail`, not `error`:
the handler executed successfully; the data failed.

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

### Evidence sample shape

Per ADR-0003's evidence policy, evidence on `dq_executions`
carries:

| Field | Source | Description |
|---|---|---|
| `evidence.records_evaluated` | aggregation function | Count of records the handler processed |
| `evidence.records_passed` | aggregation function | Count passing the per-record check |
| `evidence.violations` | aggregation function | `records_evaluated - records_passed` |
| `evidence.violation_rate` | aggregation function | Derived float |
| `evidence.late_dropped_count` | ADR-0024 | Records arrived after watermark closed the window |
| `evidence.sample_violations` | bounded by `evidence_sample_size` | Array of up to N per-record violation descriptors |

Each `sample_violations` descriptor carries the record's Kafka
offset and a handler-specific violation reason string (e.g.,
`"missing required field 'id'"`). **Privacy-sensitive fields
are omitted by default** — raw record bytes and field values are
not sampled until **B1-6** (evidence retention parameters,
open at B1) commits privacy bounds. B1-6 will commit retention duration and
privacy constraints; this ADR commits the sample-size shape and
the conservative default of no raw content in samples.

### Late-dropped records excluded from aggregation

`records_evaluated` is the count of records the handler
processed within the closed window, **not** the count of records
that arrived. The `late_dropped_count` from ADR-0024 surfaces in
evidence (`evidence.late_dropped_count`) and in the vacuous-case
status split (above), but it does **not** participate in the
violation-rate calculation. If lateness becomes severe enough to
threaten data quality, that signal lives in B0-S7's cost
guardrails (consumer lag, lateness rate), not in B0-S6's
per-check status determination.

### Lint cross-checks unchanged

This ADR adds **no new lint cross-checks**. The catalog
`aggregation` block validates via the existing JSON Schema
mechanism; cross-check #6 from ADR-0022 (per-kind params
validation) covers rule-level overrides because they sit inside
`params`. Ten cross-checks total remain from ADRs 0021–0024.

---

## Consequences

1. **ADR-0022's catalog v1 design gains a required `aggregation`
   block per entry.** This is an extension to ADR-0022's v1
   catalog design (R3 does not fully bite, because catalog v1
   has not yet shipped to disk; the combined implementation
   commit lands ADRs 0021–0026 artefacts together). Same pacing
   pattern as ADR-0022 §C-B0S2.2 and ADR-0023 §C-B0S3.1; v2
   catalog bump contingency if any v1-catalog artefact ships
   first.

2. **The `record.schema_conformance` entry's `params_schema`
   gains an optional `aggregation` property.** Authors may
   override `fail_if_violation_rate`,
   `warn_if_violation_rate`, and `evidence_sample_size` per
   check. Same pacing pattern as Consequence 1.

3. **ADR-0004's status enum is unchanged.** Record-mode uses
   the same four-state enum (`pass` / `fail` / `degraded` /
   `error`). The threshold-based aggregation function maps
   per-window per-record outcomes into the existing states;
   ADR-0004 is not reopened. The set-mode side of ADR-0004
   remains scope-noted as set-oriented; this ADR holds the
   record-mode extension of the status-mapping policy.

4. **The vacuous case splits on `late_dropped_count`.** A
   window with zero records evaluated AND zero late drops →
   `pass` (vacuous); zero records evaluated BUT positive late
   drops → `degraded` (late-drop catastrophe). The status enum
   stays semantically honest; B0-S7 still owns lateness-rate
   alerting.

5. **Per-attempt re-aggregation is the committed semantics.**
   Each attempt re-reads the same offset range and produces its
   own per-record outcomes and aggregation result. Attempts do
   not share intermediate state. Matches set-mode's per-attempt
   semantics from ADR-0002 / ADR-0003.

6. **B1-6 inherits a sample-size shape.** This ADR commits the
   per-kind `evidence_sample_size` default
   (`record.schema_conformance` = 10) and the per-rule override
   path. B1-6 (open at B1) commits the **retention duration**
   for `dq_executions` evidence rows and the **privacy bounds**
   for sampled violation descriptors. B1-6's eventual
   resolution amends the policy without reopening this ADR.

7. **Late-dropped records are excluded from the aggregation
   function** but visible in evidence. `records_evaluated`
   excludes them by construction (late records never reach the
   handler under ADR-0024's window-close semantics);
   `late_dropped_count` surfaces them as evidence. B0-S7 surfaces
   them as cost-guardrail signals.

8. **The catalog `aggregation.type` enum is open for additive
   extension.** Future record kinds that require different
   aggregation shapes extend the type enum additively per
   ADR-0001's compatibility contract.

9. **No new lint cross-checks.** The existing cross-check #6
   from ADR-0022 covers the override mechanism. Ten cross-checks
   total remain from ADRs 0021–0024.

10. **B0-S7 (cost guardrails) inherits the
    evidence-sample-size as a cost dimension.** Each retained
    violation descriptor consumes storage; B0-S7 commits
    per-runner storage and lag budgets that bind to (among
    other dimensions) the configured evidence-sample-size.
    Larger samples = higher storage cost = stricter B0-S7
    budgets.

11. **Combined-commit pacing extends across ADRs 0021–0026.**
    The runtime artefacts committed by ADRs 0021–0025 (schemas
    v2, catalog v1, ten lint cross-checks, dispatcher startup
    invariant, two parallel runners, the `mode` column on
    `dq_executions`, env-constants removal, Kafka emulator +
    ADR-0010 amendment, loader rejection-path removal, atomic
    `customer.yaml` migration) and now this ADR's artefacts
    (catalog `aggregation` block extension; `params_schema`
    `aggregation` override extension; handler-side aggregation
    logic; evidence-shape additions) land together in a single
    combined implementation commit. If any prior v2-schema or
    v1-catalog artefact ships incrementally first, the schema-
    bump and catalog-bump contingencies from prior ADRs apply.

---

## Notes

- The catalog's per-kind `aggregation.defaults` and
  `params_schema.properties.aggregation` must agree on the
  override field set by author convention. A future lint
  cross-check (deferred) could enforce this automatically;
  current enforcement is via the joint engine-maintainers +
  rules-authors review path from ADR-0015.
- B1-6 commits the privacy bounds for `sample_violations`
  content. Until B1-6 lands, the v1 implementation omits all
  raw record content from descriptors — each descriptor carries
  only the Kafka offset and a handler-specific violation reason.
- Sliding-window aggregation (for kinds that consume B0-S4's
  eventual sliding-enum extension) requires a new
  `aggregation.type` value (e.g., `sliding_rate`); not committed
  in v1, additive when needed.
- Pluggable aggregation functions (operator-registered policies
  per kind, similar to how handlers are registered) are a
  richer-extension direction than additive enum growth.
  Out of scope for v1; additive enum growth suffices until
  concrete operational signal motivates plugins.
- The aggregation execution mechanics across retry attempts
  (single consumer-group reset; per-attempt consumer-group;
  snapshot-and-replay) are runtime-engineering choices that do
  not affect the per-attempt semantics committed here; the
  combined implementation commit picks the simplest path
  consistent with ADR-0007's loader semantics and ADR-0024's
  window-close trigger.
- `error` granularity for partial handler failures (a handler
  that processes 9,990 records successfully but errors on the
  last 10 currently produces `error` per ADR-0004's
  all-or-nothing semantics) is a future enrichment. v1 matches
  set-mode: any handler runtime error → `error`.
- Alert routing for `degraded` on record-mode follows ADR-0006's
  existing policy: both `fail` and `degraded` map to the
  `data_quality` category. No separate routing path is added
  by this ADR.
