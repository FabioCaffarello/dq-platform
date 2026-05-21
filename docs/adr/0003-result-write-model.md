<!-- path: docs/adr/0003-result-write-model.md -->

# ADR-0003 — Result Write Model

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

The platform records evaluation history in two tabular-store
tables (BigQuery in the deployed environment, a fidelity-aware
emulator locally per the substrate-posture ADR):

- `dq_executions` — run-level metadata.
- `dq_check_results` — per-check results.

The identity-and-idempotency ADR (ADR-0002) requires that a
query keyed on `execution_id` returns a single canonical row,
that `engine_version` is per-attempt metadata actively surfaced
across attempts, and that scheduler retries reuse
`execution_id` while operator reruns produce a new one with an
audit link to the prior execution.

These constraints rule out an upsert model on `dq_executions`
(an upsert collapses retry history and erases per-attempt
`engine_version`). They also rule out an eagerly-materialized
canonical projection (materialization introduces refresh-lag
and derived-state-divergence failure modes).

---

## Decision

### 1. Two tables, both append-only

Both `dq_executions` and `dq_check_results` are **append-only**.
The engine performs **only `INSERT`** against these tables. No
`UPDATE`, no `DELETE` from any engine code path.

- **`dq_executions`** — composite primary key
  (`execution_id`, `attempt_id`, `recorded_at`). One row per
  state transition.
- **`dq_check_results`** — composite primary key
  (`execution_id`, `attempt_id`, `check_id`). One row per
  (attempt × check).

"Primary key" here is the logical key for the design; how it
is expressed in the deployed schema (clustering, partition
column, no enforced constraint) is a scaffolding detail.

### 2. Canonical view: `dq_executions_current` (lazy)

A view named `dq_executions_current` projects, per
`execution_id`, the row with the latest `recorded_at`:

```sql
-- illustrative; exact dialect is a scaffolding detail
CREATE VIEW dq_executions_current AS
SELECT *
FROM (
  SELECT *,
         ROW_NUMBER() OVER (
           PARTITION BY execution_id
           ORDER BY recorded_at DESC
         ) AS rn
  FROM dq_executions
)
WHERE rn = 1
```

The view is **lazy** (computed at query time, not
materialized). Dashboards and alert systems target the view;
forensic queries target the base table.

### 3. Required columns on `dq_executions`

Every row must carry:

- `execution_id` (string, 64-char lowercase hex per ADR-0002).
- `attempt_id` (string, UUID per below).
- `recorded_at` (timestamp, microsecond precision, UTC).
- `status` (enum per below).
- `engine_version` (string) — the engine that wrote this row.
- `ruleset_version` (string) — denormalized for query
  convenience.
- `entity` (string).
- `trigger_source` (enum per ADR-0002).
- `started_at` (timestamp; nullable for the `running`
  transition row; required for terminal rows).
- `completed_at` (timestamp; nullable for `running` rows;
  required for terminal rows).
- `error_summary` (string; nullable; populated when status is
  `failed`, `error`, or `aborted`).
- `supersedes_execution_id` (string; nullable; populated for
  the first state-transition row of an `operator-rerun`
  attempt).

Additional operational columns (concurrency budget, cost
telemetry) may be added additively.

### 4. `attempt_id` derivation

`attempt_id` is a UUID (version-4 or equivalent random)
assigned by the trigger handler when an attempt begins. The
same `attempt_id` is carried across all rows of that
attempt's lifecycle (typically two rows: `running` and the
terminal-status row).

A scheduler retry of the same `execution_id` assigns a **new
`attempt_id`** at attempt start. UUID is preferred over a
monotonic counter because no read-then-write coordination is
needed at attempt start (a retry burst with multiple trigger-
handler instances cannot race on a counter) and because
`attempt_id` is opaque (consistent with `execution_id`'s
opaque posture).

Sortability across attempts of the same `execution_id` is
provided by `recorded_at` on each row, not by `attempt_id`.

### 5. `supersedes_execution_id` is a nullable column

Stored as a nullable column on `dq_executions`, populated only
on the first state-transition row of an `operator-rerun`
attempt. The value is the `execution_id` of the prior run
being superseded. Chains of operator reruns produce a
traversable graph via this column; **no separate lineage
table** is created.

### 6. `status` enum (closed, additive extension)

Initial committed values:

- `running` — the trigger handler has created the row; the
  engine has not yet emitted a terminal-status row.
- `success` — every check evaluated and the engine emitted no
  operational errors.
- `failed` — at least one check produced a `fail` or
  `degraded` result.
- `error` — entity-level evaluation could not proceed.
- `aborted` — the run was halted by a global operational
  condition (cost ceiling, manifest load failure, etc.).

The detailed mapping from check-level results to
execution-level `status` is specified by ADR-0004.

Extension is additive (new value does not break existing
rows). Removal or rename is a breaking change.

### 7. Required columns on `dq_check_results`

- `execution_id`, `attempt_id`, `check_id` (composite key).
- `result` (enum: `pass` / `fail` / `degraded` / `error` —
  semantic mapping per ADR-0004).
- `executed_at` (timestamp).
- `engine_version` — the engine version that evaluated this
  check.
- `evidence_summary` — structured aggregate counts (rows
  scanned, rows failing, etc.).
- `sample_violating_rows` — repeated record capped at a
  configured limit.

### 8. CI dry-runs do not pollute production reporting

CI dry-runs write **no rows** to `dq_executions` or
`dq_check_results`. CI dry-runs exercise the trigger handler
and may compute `execution_id` for logging, but never persist
to the two production tables. Whether CI dry-runs persist
**elsewhere** (a separate `dq_ci_dry_runs` table, for
example) is a follow-up; this ADR commits only the
no-pollution rule for the two production tables.

---

## Consequences

1. **No DML from engine code paths.** Only `INSERT`. Any
   operator action that would mutate rows (data corrections,
   purges, retroactive corrections) is out-of-band of the
   engine's INSERT-only commitment; the path for such
   corrections (admin SQL endpoint, audited migration tool)
   is a follow-up scaffolding item.

2. **The canonical-view invariant is satisfied.** A query
   keyed on `execution_id` against `dq_executions_current`
   returns exactly one row, the current canonical state.
   This satisfies ADR-0002's reporting-consistency
   invariant.

3. **Forensic queries are first-class.** The base table
   preserves every state transition and every attempt; a
   forensic query joining `dq_executions` and
   `dq_check_results` on (`execution_id`, `attempt_id`)
   recovers the full history of any execution.

4. **`engine_version` is preserved per-attempt.** Required
   on both tables. A scheduler retry that spans an engine
   upgrade produces two attempt rows with the same
   `execution_id` and different `engine_version` values;
   the canonical view returns the latest. Reporting tools
   include `engine_version` in default projections per
   ADR-0002 (the visibility obligation is upstream; this
   ADR ensures the data is available).

5. **The view is lazy by design.** Computing the canonical
   projection at query time avoids the refresh-lag and
   derived-state-divergence failure modes that an eagerly
   materialized view introduces. At expected volumes
   (millions of rows per year per environment) the per-query
   cost is single-digit seconds; if observed traffic makes
   this unsatisfactory, materialization is a future option
   without changing the contract.

6. **Operator-rerun lineage is queryable via a single
   column.** No separate lineage table. The traversal is a
   self-join on `supersedes_execution_id`.

7. **The append-only multiplier is modest but unbounded over
   time.** Each execution produces 2 rows per attempt × 1–3
   attempts (typical scheduler-retry cycle) = 2–6 rows in
   `dq_executions`; each check produces 1 row per attempt =
   1–3 rows in `dq_check_results`. Storage cost is `O(n)`
   in evaluation volume — bounded only if retention is
   bounded. Per-environment retention parameters are a
   follow-up.

8. **The schema is a public contract.** Column names, enum
   values, and the composite-key shape are public once
   consumers exist. Removal or type-change of a column,
   removal of an enum value, or change to the composite-key
   shape requires a future ADR with a migration path.
   Additive changes (new columns, new enum values) do not.

9. **`attempt_id` is opaque.** Like `execution_id`,
   consumers must not parse, prefix-match, or attempt to
   derive metadata from the bits. Sortability across
   attempts is via `recorded_at`.

---

## Notes

- The exact deployed-environment dialect (BigQuery
  expressions, partition column choice, clustering keys) is
  a scaffolding detail. The lazy-view shape with
  `ROW_NUMBER() OVER (PARTITION BY ... ORDER BY ...)` is
  load-bearing; the syntax may be adapted to the dialect.
- The local-emulator fidelity gap on the lazy view is
  documented under the substrate-posture ADR; engine logic
  depending on view semantics must be exercisable in unit
  tests against an abstraction layer, with full-view fidelity
  verified in the sandbox lane.
- Retention parameters and an out-of-band correction path
  are follow-up scaffolding items, not constrained by this
  ADR.
