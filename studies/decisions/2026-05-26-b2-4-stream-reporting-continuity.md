<!-- path: studies/decisions/2026-05-26-b2-4-stream-reporting-continuity.md -->

# B2-4 — Stream Reporting Continuity

## Context

Wave-S shipped record-oriented capability alongside the
pre-existing set-oriented capability: the engine binary
now carries two runners (`SetRunner` + `RecordRunner`,
ADR-0025), a closed `mode` enum on rules and entities
(ADR-0021), tumbling-window semantics over Kafka
streams (ADR-0024 + ADR-0028), and record-mode
aggregation that maps per-record violations into a
single per-window check result (ADR-0026).

The reporting layer already accommodates the second
mode in several committed ways:

- **`execution_id`** is computed from
  `(ruleset_version, entity, window_start, window_end, trigger_source)`
  — both runners share the formula. The
  `RecordRunner.closeAndDispatch` at
  `engine/internal/runner/record_runner.go:327` builds
  a `TriggerRequest` (lines 332-341) with the tumbling
  window's start and end and dispatches through the
  same `Runner.Run` that set-mode uses.
- **The `dq_executions` table gained an additive
  `mode` column** committed by ADR-0025
  §"Result-write schema extension". The Go struct
  (`engine/internal/results/types.go:35`) carries
  `Mode Mode` on every `ExecutionRow`.
- **The dashboard contract metric inventory** in
  ADR-0039 commits `mode` as a label on every emitted
  metric (`dq_runs_total`,
  `dq_checks_evaluated_total`, the duration histograms,
  `dq_check_duration_seconds`). Cross-mode aggregation
  is possible at the metric layer.
- **The failure-scope mapping** in ADR-0026 commits
  the record-mode threshold aggregation that produces
  a single `dq_check_results` row per (window × check)
  with `pass` / `fail` / `degraded` from the same enum
  set-mode uses.

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

In addition, two specific reporting gaps remain at
this ADR's writing:

1. **Window endpoints are not first-class
   `dq_executions` columns.** `window_start` and
   `window_end` are *inputs* to `execution_id`
   (ADR-0002 §3) but not stored as queryable columns.
   For record-mode this is acute — windows are
   frequent (tumbling 1m/5m/15m at typical
   configurations), and an observer querying
   "show me all windows for entity X yesterday" must
   either re-derive endpoints from
   `(started_at, ruleset_version, ...)` or query a
   separate trigger-handler audit log. The data is
   present at compute time and discarded at storage.

2. **Per-mode time-field semantics are not
   documented.** `started_at` and `completed_at`
   (ADR-0003 §3) carry different meanings under each
   runner:
   - Set-mode: trigger acceptance time → SQL
     evaluation completion time (a single roughly-
     short interval).
   - Record-mode: window close trigger time →
     aggregation completion time (an interval that
     may straddle late-arriving records within the
     lateness-tolerance per ADR-0024).
   The
   `dq_run_duration_seconds` histogram label
   committed by ADR-0039 carries different meaning
   under each mode; a dashboard panel showing
   "average run duration" mixes apples and oranges
   without mode awareness.

The B2-4 row registered this gap at the W3 backlog
numbering step:

> How will stream-runner results align with batch
> result tables and identifiers? Future migration
> should not fracture observability.

The "future migration" framing reads literally: if a
customer-entity transitions from set-mode to
record-mode (or vice versa, e.g., during a substrate
migration), the entity's historical observability
must remain coherent. A dashboard panel scoped to
`entity='customer'` must not break when half the
rows are `mode='set'` and the other half are
`mode='record'`.

The principles bearing on the decision are **P3**
(ownership is explicit — the consumer-facing
continuity invariant needs to be articulated in one
place rather than left implicit across five ADRs),
**P5** (evolution must be contract-driven — mode
transitions are an evolution the consumer surface
must handle predictably), and **P2** (deterministic
behavior — per-mode time-field semantics must be
documented or "duration" comparisons mislead).

What B2-4 must commit:

1. **The unified-reporting invariant** as a single
   consumer-facing statement.
2. **The window-endpoint observability gap** —
   either close it (add `window_start` /
   `window_end` as additive columns) or commit the
   recompute-from-trigger-log workaround.
3. **Per-mode time-field semantics** for
   `started_at` / `completed_at`.
4. **Cross-mode dashboard interpretation guidelines**
   — what query shapes are mode-agnostic, what shapes
   require mode-awareness.
5. **The mode-transition observability rule** — how
   the consumer surface handles an entity flipping
   modes.

---

## Decision Drivers

- **DD-1 — The unified-reporting invariant is true
  in code but unwritten as a contract.** Five ADRs
  (0002 identity, 0003 tables, 0025 mode column +
  runner split, 0026 aggregation, 0039 dashboard
  contract) jointly establish that record-mode and
  set-mode results coexist in the same tables under
  the same identifier scheme. No single ADR commits
  this as a *consumer-facing promise*. A reviewer or
  downstream consumer reading any one of those ADRs
  cannot derive the joint invariant. This ADR
  articulates it once.

- **DD-2 — Window endpoints are present at compute
  time and discarded at storage.** The
  `RecordRunner` (record_runner.go:332-341)
  constructs a `TriggerRequest` with `WindowStart` /
  `WindowEnd` from the closed tumbling window; the
  `Runner.Run` consumes those endpoints to compute
  `execution_id` and then drops them — the
  `ExecutionRow` written to `dq_executions`
  (results/types.go:30-45) carries no window fields.
  For set-mode the gap is less acute (windows are
  often inferable from the trigger cadence), but for
  record-mode the same entity produces hundreds of
  windows per day, all with distinct
  `execution_id`s, none of which is human-readable
  back to its window endpoints without a recompute.

- **DD-3 — Per-mode time-field semantics differ in
  ways that matter for dashboards.** The interpretation
  rules in this DD are **new contribution proposed
  here, requires review** — no prior ADR commits the
  per-mode semantics of `started_at` / `completed_at`.
  `started_at` is captured by the runner at `Run`
  entry (runner.go:307); for set-mode this is
  effectively trigger acceptance because the trigger
  handler invokes `Run` synchronously; for record-mode
  this is the moment the runner dispatches the closed
  window (`RecordRunner.closeAndDispatch` invokes
  `r.dispatcher.Run`, which fires after the watermark
  advances past `window_end + lateness_tolerance` per
  ADR-0024). `completed_at` under set-mode is
  SQL-completion; under record-mode is aggregation-
  completion. The `dq_run_duration_seconds` histogram
  label thus measures "Run-entry-to-SQL latency" for
  set and "window-close-dispatch-to-aggregation
  latency" for record. A dashboard panel comparing
  the two without mode awareness misleads.

- **DD-4 — Mode-as-primitive (ADR-0021) is the
  carrier for cross-mode queries.** The closed enum
  `{set, record}` on every relevant surface (rule
  YAML, `_owners.yaml`, the `mode` column on
  `dq_executions`, the `mode` label on every metric)
  is what enables a consumer to either filter
  ("show me record-mode runs") or partition ("group
  by mode, then ..."). The invariant the consumer
  contract commits is: *every consumer-visible
  surface that carries the mode* — table column,
  metric label — *carries the same `{set, record}`
  enum from the same source*.

- **DD-5 — Mode transitions for a single entity are
  rare but operationally real.** An entity flipping
  from set-mode (BigQuery batch table) to record-mode
  (Kafka stream) — or the reverse — happens during
  substrate migrations. The historical rows for that
  entity in `dq_executions` carry the original mode;
  new rows carry the new mode. A
  cross-mode-transition query (`SELECT pass_rate WHERE
  entity = 'customer' GROUP BY DATE(recorded_at)`)
  must remain valid SQL across the transition. The
  semantic interpretation ("yesterday's pass rate"
  means "yesterday's set-mode batch run was OK")
  changes; the *query* doesn't break.

- **DD-6 — Add window endpoints as additive columns;
  defer the engine-code implementation to a B2 slice.**
  Per ADR-0003's own "Additional operational columns
  ... may be added additively" + ADR-0039's evolution
  rule, adding `window_start` / `window_end` to
  `dq_executions` is the natural close to DD-2. The
  *contract* lands in this ADR; the *implementation*
  (struct field additions, DDL migration, view
  update, dashboard inventory amendment) is a B2
  follow-up consumer slice paced post-Phase-4c
  metric emission so the change can be smoke-tested
  end-to-end. This matches the design-only pattern
  set by ADR-0030 / ADR-0032 / ADR-0033 / ADR-0039.

---

## Considered Options

### Option 1 — Articulate the unified-reporting invariant + commit window-endpoint columns + commit per-mode time-field semantics (recommended)

This ADR commits four contract clauses:

1. **Unified-reporting invariant** — record-mode and
   set-mode results land in the same `dq_executions`
   and `dq_check_results` tables under the same
   `execution_id` scheme. Cross-mode queries are
   valid SQL; mode-aware interpretation is the
   consumer's responsibility.

2. **Window endpoints as additive `dq_executions`
   columns** (design commitment; implementation
   deferred). `window_start` and `window_end` join
   the column inventory as stable additive columns
   under ADR-0039's evolution rule. Their
   implementation — Go struct field additions, DDL
   migration, `dq_executions_current` view update,
   dashboard-contract inventory amendment — ships as
   a B2 follow-up.

3. **Per-mode time-field semantics** (**new
   contribution proposed here, requires review**) —
   `started_at` and `completed_at` carry mode-
   dependent meanings committed explicitly here, so
   dashboard authors can either filter by mode or
   document their mixed-mode interpretation:
   - **Set-mode:** `started_at` = `Run` entry time
     (runner.go:307), which is effectively trigger-
     handler acceptance because the trigger handler
     invokes `Run` synchronously; `completed_at` =
     SQL evaluation completion (the last per-check
     evaluator returned).
   - **Record-mode:** `started_at` = `Run` entry time
     reached via `RecordRunner.closeAndDispatch`
     (record_runner.go:350), which fires after the
     watermark advances past
     `window_end + lateness_tolerance` per ADR-0024;
     `completed_at` = aggregation completion (the
     per-window threshold aggregation per ADR-0026
     finalized).

4. **Mode-transition rule** — when an entity flips
   mode, historical rows preserve their original
   `mode` value; new rows carry the new `mode`
   value. No row is rewritten. Cross-mode-transition
   queries on the entity's history remain valid SQL;
   the semantic interpretation shifts at the
   transition timestamp, which the consumer surfaces
   via the `mode` column.

**Strengths.** Articulates one consumer-facing
promise (DD-1). Closes the observability gap at the
contract level (DD-2) while honoring the
design-only / consumer-slice separation (DD-6).
Documents the per-mode time-field interpretation
once so dashboard authors don't re-derive it
(DD-3). Honors mode-as-primitive (DD-4) and the
mode-transition operational case (DD-5).

**Trade-offs.** The window-endpoint columns are
contract-only at this ADR; consumers who want to
filter by `window_start` today still need the
recompute workaround until the B2 implementation
slice lands. Acceptable — the contract is the
load-bearing artefact, and the implementation is a
small, mechanical addition (column add + view
extension) when it lands.

### Option 2 — Articulate the invariant only; defer window-endpoint columns to a separate B2

Commit clauses 1 + 3 + 4 (invariant, per-mode time
semantics, mode-transition rule). Leave the
window-endpoint column question to a separate B2
follow-up entirely.

**Strengths.** Smaller commit; the
contract-articulation work is independent of the
new-column work.

**Trade-offs.** Splits one logical decision into
two ADRs. The window-endpoint gap is the most
operationally acute part of B2-4's "stream reporting
continuity" — deferring it would close B2-4 while
leaving the actual gap open. Rejected: closing B2-4
should commit the gap's resolution, even at the
contract level only.

### Option 3 — Ship engine code today + contract

Land the contract + the implementation (Go struct
additions, DDL migration, view extension) in one
session.

**Strengths.** Closes the gap fully in one pass.

**Trade-offs.** The implementation crosses into
engine code territory (struct + DDL + view), each
of which is a real change with its own test
surface. The platform's standing pattern (ADR-0030,
ADR-0032, ADR-0033, ADR-0039) is design-only ADR
+ consumer-slice implementation; deviating without
strong cause violates R3-adjacent operating
discipline. Rejected: defer the implementation per
DD-6.

---

## Recommendation

**Option 1.** Articulate the unified-reporting
invariant; commit window-endpoint columns at the
contract level (implementation deferred to a B2
consumer slice); commit per-mode time-field
semantics; commit the mode-transition rule.

### Unified-reporting invariant

Both runners write to the same reporting tables
under the same identifier scheme:

| Surface | Set-mode | Record-mode | Source ADR |
|---|---|---|---|
| Result tables | `dq_executions`, `dq_check_results` | same | ADR-0003 + ADR-0025 §5 |
| Canonical view | `dq_executions_current` | same | ADR-0003 §2 |
| `execution_id` | SHA256 of (ruleset_version, entity, window_start, window_end, trigger_source) | same | ADR-0002 §3 |
| `mode` column | `set` | `record` | ADR-0025 §5 + this ADR |
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
responsibility (see §"Per-mode time-field
semantics" + §"Cross-mode dashboard interpretation"
below).

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
available at compute time (engine/internal/runner/
runner.go:50-51). For set-mode the trigger handler
sets them per the dispatch convention (typically
"now - cadence" → "now" for batch); for record-mode
the `RecordRunner.closeAndDispatch` sets them from
the closed tumbling window's boundaries
(record_runner.go:325-326).

**Implementation deferred.** The Go struct
extension, DDL migration, view-projection update,
and ADR-0039 inventory amendment land in a B2
follow-up consumer slice (registered below). The
contract this ADR commits is binding at the
contract level; the implementation is mechanical
when the slice lands.

### Per-mode time-field semantics

The interpretation rules in this section are **new
contribution proposed here, requires review**. ADR-0003
§3 declares the `started_at` / `completed_at` columns
and their nullable-on-`running` posture but does not
commit per-mode semantics; this ADR commits them.

`started_at` and `completed_at` carry the following
mode-dependent meanings:

| Field | Set-mode | Record-mode |
|---|---|---|
| `started_at` | `Run` entry time (runner.go:307), which is effectively trigger-handler acceptance because the trigger handler invokes `Run` synchronously. | `Run` entry time reached via `RecordRunner.closeAndDispatch` (record_runner.go:350), which fires after the watermark advances past `window_end + lateness_tolerance` per ADR-0024. |
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
   entity) are valid as-is.** Examples:

   ```sql
   -- Mode-agnostic; total runs across both modes.
   SELECT COUNT(*) FROM dq_executions_current
   WHERE entity = 'customer' AND status = 'success'
   ```

2. **Aggregation queries that combine
   per-mode-semantically-different metrics must
   group by mode.** Examples include duration
   histograms (DD-3 — different semantics per mode)
   and pass-rate-by-mode-meaning queries (set-mode
   "pass" = one SQL OK; record-mode "pass" = window
   aggregation within threshold).

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
  surfaces the transition.** A consumer wanting to
  separate pre/post-transition behavior filters or
  groups by `mode`.

The transition itself — whether a mode flip
re-enters the entity-onboarding workflow (ADR-0040)
or follows a separate procedure — is governance not
committed by this ADR. The reporting-continuity
rule here commits only what the *consumer surface*
sees during and after the flip; the operational
procedure for executing the flip is a separate
governance question reserved for a future ADR.

### Why this does not reopen ADR-0002 / ADR-0003 / ADR-0025 / ADR-0026 / ADR-0039

Each prior ADR commits a piece of the reporting
surface this ADR articulates the consumer
continuity invariant over:

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
  additive. This ADR cites it as the
  mode-as-primitive carrier on the table without
  amending.
- **ADR-0026** commits record-mode aggregation
  semantics. This ADR cites them as the source of
  record-mode `result` values without re-deriving.
- **ADR-0039** commits the dashboard contract. This
  ADR extends the stability tier to include
  `window_start` / `window_end` (additive per the
  contract's own evolution rule). The
  implementation-side amendment to ADR-0039's
  inventory ships with the B2 follow-up consumer
  slice.

---

## Consequences

1. **The unified-reporting invariant is committed
   as a single consumer-facing promise.** Five
   prior ADRs' joint posture is articulated in one
   place. A consumer reading this ADR (and only
   this ADR) understands that record-mode and
   set-mode results coexist in the same tables under
   the same identifier scheme.

2. **`window_start` and `window_end` are committed
   as additive `dq_executions` columns at the
   contract level.** Implementation deferred to a B2
   consumer slice. The contract is binding now; the
   slice ships the struct/DDL/view changes when it
   lands.

3. **Per-mode time-field semantics are committed
   for `started_at` / `completed_at`.** Dashboard
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
   guidelines** are committed. Mode-agnostic
   queries work as-is; mode-semantically-different
   metric comparisons require `GROUP BY mode` or
   per-mode filtering.

6. **A new B2 row registers the implementation
   slice for window-endpoint columns.** The slice
   adds `WindowStart` / `WindowEnd` to
   `engine/internal/results/types.go`'s
   `ExecutionRow`, ships the DDL migration that adds
   the columns to `dq_executions`, updates the
   `dq_executions_current` view projection, and
   amends ADR-0039's stable-column inventory.
   Paced post-Phase-4c metric emission (same
   pacing as ADR-0039's B2-24 baseline-dashboard
   slice) so the new columns smoke-test end-to-end
   against actual metric output.

7. **B2-4 closes.** The decision-log B2-4 row moves
   to `resolved-adr`. One new B2 row registers the
   window-endpoint-column implementation follow-up.

8. **ADR-0002, ADR-0003, ADR-0025, ADR-0026,
   ADR-0039 are preserved.** This ADR layers a
   consumer-facing continuity invariant on top of
   their joint commitments without amending any.

9. **ADR-0021 mode-as-primitive is preserved and
   surfaced.** The `{set, record}` enum from
   ADR-0021 is what carries the mode label on every
   reporting surface (column, metric label).
   Renaming or removing values from this enum is a
   breaking change to the reporting contract.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: Window-endpoint column implementation
  slice.** The struct/DDL/view changes that
  implement `window_start` / `window_end` on
  `dq_executions`. Reserved as a B2 follow-up
  paced post-Phase-4c metric emission. Registered
  in the decision-log update accompanying this
  ADR's promotion.

- **OQ-2: Cross-mode entity-pass-rate definition.**
  When a dashboard asks "what's the entity's
  overall pass rate", the semantic answer differs
  by mode (set-mode "pass" = single SQL OK;
  record-mode "pass" = window aggregation within
  threshold). The cross-mode interpretation
  guidelines in this ADR commit the mechanics
  (mode-aware queries); whether a *platform-defined*
  cross-mode pass-rate metric (e.g.,
  `dq_entity_health_score`) should ship as a
  consumer-facing convenience is a separate
  question. Reserved until concrete consumer signal
  surfaces; the current per-mode metrics from
  ADR-0039 are sufficient for v1 consumers.

---

## Promotion target

`docs/adr/0041-stream-reporting-continuity.md` —
next free ADR number. Ships the unified-reporting
invariant, the window-endpoint additive-column
commitment (design-only), the per-mode time-field
semantics, the cross-mode dashboard interpretation
guidelines, and the mode-transition rule.
