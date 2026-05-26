<!-- path: docs/adr/0041-stream-reporting-continuity.md -->

# ADR-0041 — Stream Reporting Continuity

- **Status:** accepted
- **Date:** 2026-05-26

---

## Context

Wave-S shipped record-oriented capability alongside
the pre-existing set-oriented capability: the engine
binary now carries two runners (set + record per
ADR-0025), a closed `mode` enum on rules and entities
(ADR-0021), tumbling-window semantics over Kafka
streams (ADR-0024 + ADR-0028), and record-mode
aggregation that maps per-record violations into a
single per-window check result (ADR-0026).

The reporting layer accommodates the second mode in
several committed ways:

- **`execution_id`** is computed from
  `(ruleset_version, entity, window_start, window_end, trigger_source)`
  per [ADR-0002](./0002-run-identity-and-idempotency.md)
  §3 — both runners share the formula. The
  `RecordRunner.closeAndDispatch` builds a
  `TriggerRequest` with the tumbling window's start
  and end and dispatches through the same `Runner.Run`
  set-mode uses.
- **The `dq_executions` table gained an additive
  `mode` column** committed by
  [ADR-0025](./0025-aggregation-and-runner-shape.md)
  §"Result-write schema extension". Every `ExecutionRow`
  carries `Mode Mode`.
- **The dashboard contract metric inventory** in
  [ADR-0039](./0039-dashboard-contract.md) commits
  `mode` as a label on every emitted metric. Cross-mode
  aggregation is possible at the metric layer.
- **The failure-scope mapping** in
  [ADR-0026](./0026-failure-scope-aggregated.md)
  commits the record-mode threshold aggregation that
  produces a single `dq_check_results` row per (window
  × check) with `pass` / `fail` / `degraded` / `error`
  from the same enum set-mode uses.

What none of those ADRs commits *jointly* is the
**consumer-facing reporting-continuity invariant**:
"results from both modes land in the same tables under
the same identifier scheme; downstream observability
queries that don't filter by mode remain valid SQL
across mode transitions; per-mode semantic differences
are documented in this one place." Today the
invariant is true *in code* but the contract is
scattered across five ADRs (0002, 0003, 0025, 0026,
0039) and not articulated as a single consumer
commitment.

Two specific reporting gaps remain at this ADR's
writing:

1. **Window endpoints are not first-class
   `dq_executions` columns.** `window_start` and
   `window_end` are inputs to `execution_id` (ADR-0002
   §3) but not stored as queryable columns. For
   record-mode this is acute — windows are frequent
   (tumbling 1m/5m/15m at typical configurations), and
   an observer querying "show me all windows for
   entity X yesterday" must either re-derive endpoints
   from `(started_at, ruleset_version, ...)` or query
   a separate trigger-handler audit log.

2. **Per-mode time-field semantics are not
   documented.** `started_at` and `completed_at`
   carry different meanings under each runner; the
   `dq_run_duration_seconds` histogram label committed
   by ADR-0039 carries different meaning under each
   mode. A dashboard panel comparing "average run
   duration" mixes apples and oranges without mode
   awareness.

The principles bearing on the decision are **P3**
(ownership is explicit — the consumer-facing
continuity invariant needs to be articulated in one
place rather than left implicit across five ADRs) and
**P5** (evolution must be contract-driven — mode
transitions are an evolution the consumer surface
must handle predictably).

---

## Decision

### Unified-reporting invariant

Both runners write to the same reporting tables under
the same identifier scheme:

| Surface | Set-mode | Record-mode | Source |
|---|---|---|---|
| Result tables | `dq_executions`, `dq_check_results` | same | ADR-0003 + ADR-0025 §"Result-write schema extension" |
| Canonical view | `dq_executions_current` | same | ADR-0003 §2 |
| `execution_id` | SHA256 of (ruleset_version, entity, window_start, window_end, trigger_source) | same | ADR-0002 §3 |
| `mode` column | `set` | `record` | ADR-0025 §"Result-write schema extension" |
| `status` enum | `running` / `success` / `failed` / `error` / `aborted` | same | ADR-0003 §6 |
| `result` enum | `pass` / `fail` / `degraded` / `error` | same | ADR-0004 + ADR-0026 |
| Aggregation rules | per-check → per-execution per ADR-0004 | per-record → per-window → per-check via ADR-0026 threshold | ADR-0004 + ADR-0026 |
| Metric labels | `mode = set` | `mode = record` | ADR-0039 |

A cross-mode query is valid SQL by construction:

```sql
SELECT mode, COUNT(*) AS executions
FROM dq_executions_current
WHERE entity = 'customer'
  AND recorded_at >= '2026-05-26'
GROUP BY mode
```

Mode-aware semantic interpretation is the consumer's
responsibility (see §"Per-mode time-field semantics"
and §"Cross-mode dashboard interpretation" below).

### Window-endpoint columns (design-only)

`window_start` and `window_end` join the
`dq_executions` column inventory as additive stable
columns:

| Column | Type | Stability tier | Source |
|---|---|---|---|
| `window_start` | timestamp (UTC, microsecond) | stable across engine minor versions | new in this ADR |
| `window_end` | timestamp (UTC, microsecond) | stable across engine minor versions | new in this ADR |

The columns are populated by both runners from the
`TriggerRequest.WindowStart` / `WindowEnd` already
available at compute time. For set-mode the trigger
handler sets them per the dispatch convention (typically
"now - cadence" → "now" for batch); for record-mode
the `RecordRunner.closeAndDispatch` sets them from the
closed tumbling window's boundaries.

**Implementation deferred.** The Go struct extension,
DDL migration, view-projection update, and ADR-0039
inventory amendment land in a B2 follow-up consumer
slice (registered in the decision-log update
accompanying this ADR's promotion). The contract this
ADR commits is binding at the contract level; the
implementation is mechanical when the slice lands.

### Per-mode time-field semantics

The interpretation rules in this section are **new
contribution proposed here, requires review**. ADR-0003
§3 declares the `started_at` / `completed_at` columns
and their nullable-on-`running` posture but does not
commit per-mode semantics; this ADR commits them.

| Field | Set-mode | Record-mode |
|---|---|---|
| `started_at` | `Run` entry time, which is effectively trigger-handler acceptance because the trigger handler invokes `Run` synchronously. | `Run` entry time reached via `RecordRunner.closeAndDispatch`, which fires after the watermark advances past `window_end + lateness_tolerance` per ADR-0024. |
| `completed_at` | SQL evaluation completion (the last per-check evaluator returned). | Aggregation completion (per-window threshold aggregation per ADR-0026 finalized). |
| `completed_at - started_at` | "Run-entry-to-SQL-completion latency." | "Window-close-dispatch-to-aggregation-completion latency." Does NOT include the lateness-tolerance wait — that elapses before `started_at` is stamped. |

The two intervals measure different operational
quantities. Dashboard authors comparing
`dq_run_duration_seconds` across modes must filter
by `mode` first or use separate panels.

### Cross-mode dashboard interpretation

Three rules for consumers writing dashboards or
metric queries:

1. **Mode-agnostic queries (count, list, filter by
   entity) are valid as-is.** Example:

   ```sql
   -- Mode-agnostic; total successful runs across both modes.
   SELECT COUNT(*) FROM dq_executions_current
   WHERE entity = 'customer' AND status = 'success'
   ```

2. **Aggregation queries that combine
   per-mode-semantically-different metrics must
   group by mode.** Examples include duration
   histograms (different semantics per mode per the
   table above) and pass-rate-by-mode-meaning queries
   (set-mode "pass" = one SQL OK; record-mode "pass" =
   window aggregation within threshold).

   ```sql
   -- Mode-aware; reports duration per mode.
   SELECT mode,
          APPROX_QUANTILES(
            TIMESTAMP_DIFF(completed_at, started_at, MILLISECOND),
            100
          )[OFFSET(50)] AS p50_duration_ms
   FROM dq_executions_current
   WHERE entity = 'customer'
   GROUP BY mode
   ```

3. **Mixed-mode panels are valid for status/failure
   counts but not for duration/cost comparisons.**
   "How many executions failed yesterday across the
   platform" is mode-agnostic and the answer is
   meaningful. "What's the average run duration
   yesterday" is mode-dependent and the answer is
   misleading without mode awareness.

### Mode-transition rule

When an entity flips mode (set → record or
record → set, e.g., during a substrate migration):

- **Historical rows preserve their original `mode`
  value.** No row is rewritten.
- **The transition is observable** by querying the
  entity's history grouped by `mode` and ordered by
  `recorded_at`. The transition timestamp is the
  earliest `recorded_at` carrying the new `mode`.
- **Cross-transition queries remain valid SQL.**
  `SELECT pass_rate WHERE entity = 'customer' GROUP
  BY DATE(recorded_at)` doesn't break at the
  transition; rows from both sides aggregate
  together. The semantic interpretation (set "pass"
  ≠ record "pass" per cross-mode rule #2 above)
  shifts at the transition; the query doesn't.
- **The dashboard contract's `mode` label and column
  surface the transition.** A consumer wanting to
  separate pre/post-transition behavior filters or
  groups by `mode`.

The transition itself — whether a mode flip re-enters
the entity-onboarding workflow (ADR-0040) or follows
a separate procedure — is governance not committed by
this ADR. This ADR commits only what the *consumer
surface* sees during and after the flip; the
operational procedure is a separate governance
question reserved for a future ADR.

### Why this does not reopen ADR-0002 / ADR-0003 / ADR-0025 / ADR-0026 / ADR-0039

Each prior ADR commits a piece of the reporting
surface this ADR articulates the consumer continuity
invariant over:

- **ADR-0002** commits the `execution_id` formula
  (window-endpoint-keyed). This ADR cites it as the
  identifier-alignment mechanism for both modes
  without amending the formula.
- **ADR-0003** commits the table column inventory.
  This ADR adds `window_start` and `window_end` to
  that inventory as additive columns — additive
  extension is explicitly permitted by ADR-0003's
  own "additional operational columns may be added
  additively" clause and by ADR-0039's evolution
  rule.
- **ADR-0025** commits the `mode` column as
  additive. This ADR cites it as the mode-as-primitive
  carrier on the table without amending.
- **ADR-0026** commits record-mode aggregation
  semantics. This ADR cites them as the source of
  record-mode `result` values without re-deriving.
- **ADR-0039** commits the dashboard contract. This
  ADR extends the stability tier to include
  `window_start` / `window_end` (additive per the
  contract's own evolution rule). The implementation-
  side amendment to ADR-0039's inventory ships with
  the B2 follow-up consumer slice.

---

## Consequences

1. **The unified-reporting invariant is committed as
   a single consumer-facing promise.** Five prior
   ADRs' joint posture is articulated in one place.
   A consumer reading this ADR (and only this ADR)
   understands that record-mode and set-mode results
   coexist in the same tables under the same
   identifier scheme.

2. **`window_start` and `window_end` are committed
   as additive `dq_executions` columns at the
   contract level.** Implementation deferred to a B2
   consumer slice. The contract is binding now; the
   slice ships the struct/DDL/view changes when it
   lands.

3. **Per-mode time-field semantics are committed for
   `started_at` / `completed_at`.** Dashboard
   authors have one place to look for the per-mode
   interpretation. The `dq_run_duration_seconds`
   histogram label-by-mode is the queryable
   workaround for the per-mode duration semantics.

4. **The mode-transition rule preserves historical
   observability.** An entity flipping modes
   preserves its prior rows with their original
   `mode` value; new rows carry the new `mode`.
   Cross-transition queries remain valid SQL.

5. **The cross-mode dashboard interpretation
   guidelines are committed.** Mode-agnostic queries
   work as-is; mode-semantically-different metric
   comparisons require `GROUP BY mode` or per-mode
   filtering.

6. **A new B2 row registers the implementation slice
   for window-endpoint columns.** The slice adds
   `WindowStart` / `WindowEnd` to
   `engine/internal/results/types.go`'s
   `ExecutionRow`, ships the DDL migration that adds
   the columns to `dq_executions`, updates the
   `dq_executions_current` view projection, and amends
   ADR-0039's stable-column inventory. Paced
   post-Phase-4c metric emission (same pacing as
   ADR-0039's B2-24 baseline-dashboard slice) so the
   new columns smoke-test end-to-end against actual
   metric output.

7. **B2-4 closes.** The decision-log B2-4 row moves
   to `resolved-adr`. One new B2 row registers the
   window-endpoint-column implementation follow-up.

8. **ADR-0002, ADR-0003, ADR-0025, ADR-0026, ADR-0039
   are preserved.** This ADR layers a consumer-facing
   continuity invariant on top of their joint
   commitments without amending any.

9. **ADR-0021 mode-as-primitive is preserved and
   surfaced.** The `{set, record}` enum from ADR-0021
   is what carries the mode label on every reporting
   surface (column, metric label). Renaming or
   removing values from this enum is a breaking
   change to the reporting contract.

10. **One deferred posture is registered out-of-scope:**
    whether a platform-defined cross-mode entity
    pass-rate metric (e.g., `dq_entity_health_score`)
    should ship as a consumer-facing convenience. The
    cross-mode dashboard interpretation guidelines in
    this ADR commit the mechanics (mode-aware
    queries); the *platform-defined* convenience
    metric is a separate question reserved until
    concrete consumer signal surfaces. The current
    per-mode metrics from ADR-0039 are sufficient for
    v1 consumers.
