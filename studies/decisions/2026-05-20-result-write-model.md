<!-- path: studies/decisions/2026-05-20-result-write-model.md -->

# B0-3 — Result Write Model

## Metadata

- B0 reference: B0-3 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 5).
- Last updated: 2026-05-20.
- Upstream resolved: B0-2
  ([`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md)).
  Indirect inputs from B0-1 and B0-5 via the manifest's `ruleset_version`.
- Downstream open: B0-4 (failure scope), B0-6 (alert routing), B0-7
  (loader / scheduler / retry failure semantics).
- Promotion target: see final section.

---

## Context

The platform records evaluation history in two BigQuery tables, named
in foundation doc
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
§"Layer 4 — Reporting" and §"Execution Flow":

- `dq_executions` — run-level metadata.
- `dq_check_results` — per-check results.

Foundation doc 04 §"Execution Flow" describes a lifecycle in which
the trigger handler creates a `dq_executions` row with status
`running`, the engine writes per-check results, then the execution
row is "updated with final status". That language hints at mutation
but does not commit a write model. Foundation doc 05 explicitly
flags the choice — append-only, upsert, or hybrid — as a B0
decision.

B0-2 (resolved 2026-05-20,
[`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md))
locked the identity layer:

- `execution_id` is reused across scheduler retries (CC3 of B0-2).
- `execution_id` is *new* for operator reruns, with a
  `supersedes_execution_id` audit link (CC5).
- Per-attempt `engine_version` is mandatory and must be **actively
  visible** in default projections (CC4 + CC14).
- I4: a query keyed on `execution_id` returns a **single canonical
  row** regardless of physical attempt count; multiple attempt rows
  must be **collapsible by query** (CC10).
- B0-2 explicitly punted three sub-decisions to B0-3: the
  append/upsert/hybrid choice (CC10), the storage shape of
  `supersedes_execution_id` (OQ-1), and whether `ci-dry-run` is
  added to the `trigger_source` enum if CI dry-runs are persisted
  (OQ-6).

B0-3 — as recorded in the decision log:

> Are `dq_executions` and `dq_check_results` append-only, upserted,
> or hybrid?

This study locks:

1. The write model for both tables.
2. Composite-key shape for each table.
3. The canonical view that satisfies B0-2 I4.
4. The storage shape of `supersedes_execution_id`.
5. Whether CI dry-runs persist (and the resulting decision on the
   `ci-dry-run` enum value).
6. The `status` enum on `dq_executions`.

What this study does **not** decide:

- Exact partition strategy, clustering keys, or column-level
  nullability (Wave 3 schema implementation; per-environment
  parameters are B1).
- Retention policies on the append-only tables (B1 — adjacent to
  B1-6 which currently covers failed-sample retention).
- Streaming vs. batch insert mechanics (operations concern).
- The exact failure-scope mapping (check error vs. entity error
  vs. run error) — that is B0-4.

The decision matters because the table schemas are a **public
contract** for downstream consumers (dashboards, alerting, ad-hoc
forensic queries, future stream-runner reporting per
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
§"Evolution Flow"). Changing the write model after consumers exist
is breaking; locking it on the first pass is cheaper than
re-resolving later.

---

## Decision Drivers

The decision must satisfy the following, in priority order.

1. **D1. B0-2 I4 — single canonical row per `execution_id`.** A
   query keyed on `execution_id` must return one row representing
   the current state of the logical execution, regardless of how
   many physical attempt records exist.

2. **D2. B0-2 CC4 + CC14 — per-attempt `engine_version` actively
   visible.** Heterogeneous `engine_version` values across attempts
   of the same `execution_id` (a scheduler retry that spanned an
   engine upgrade per B0-2 CC3) must be surfaced in default
   projections, not behind a flag.

3. **D3. Forensic integrity.** The full history of state
   transitions (trigger → running → terminal) and the full history
   of scheduler retries must be reconstructable from the storage.
   No mutation of recorded events.

4. **D4. Visible failure over silent degradation** (foundation doc
   [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
   §"Operating Posture", imperative #3). A failure or error state
   must be queryable in its specific form, not masked by a
   subsequent successful retry.

5. **D5. Storage cost is bounded** (P4). `dq_executions` is
   low-volume (one logical run per (entity, window) trigger);
   `dq_check_results` is moderate (one per (run × check)).
   Append-only multipliers are small but non-zero; the design
   should not multiply storage gratuitously.

6. **D6. Query simplicity for dashboards.** The common-case dashboard
   query ("what is the current state of execution X?") should be
   one read, no client-side post-processing. The forensic query
   ("what was the full history of execution X?") should also be
   one read, on a different surface.

7. **D7. Schema is a public contract.** Once consumers (dashboards,
   alerting, downstream analytics) begin keying off these tables,
   the schema is locked in the sense of B0-1's schema migration
   protocol. The write model influences every downstream consumer's
   query patterns; it must be deliberate.

8. **D8. Alignment with the engine's append-friendly substrate.**
   The platform's underlying storage substrate (BigQuery) is
   optimized for append-and-query rather than row-level update;
   the canonical idiom there is "write events, query for latest".
   The chosen write model should align with the substrate's
   strengths rather than fight them.

9. **D9. Composite keys must accommodate B0-2's identity model.**
   Scheduler retries of the same `execution_id` produce new
   *attempts* (CC3 of B0-2); the storage key must distinguish
   them.

---

## Considered Options

Each option is described by **what gets written when**, the
**identifying key**, and **how a canonical-state query works**.

### Option A — Upsert both tables

Both tables hold one logical row per execution. `dq_executions` is
keyed by `execution_id`; `dq_check_results` by (`execution_id`,
`check_id`). State transitions and scheduler retries overwrite the
existing rows.

```
dq_executions(execution_id PRIMARY KEY, status, engine_version, ...)
dq_check_results((execution_id, check_id) PRIMARY KEY, result, ...)
```

Canonical query: `SELECT * FROM dq_executions WHERE execution_id = ?`
— direct, returns one row.

**Trade-offs.**

- Pro: simplest dashboards (one row, no projection logic).
- Pro: D1 trivially satisfied (one row exists by construction).
- Con: violates D3 — state transitions overwrite previous state;
  the trigger-time `running` row is lost the moment the engine
  writes the terminal state.
- Con: violates D2 + D4 — only the final attempt's `engine_version`
  is preserved; heterogeneous-engine-version attempts (B0-2 CC14)
  are invisible by construction.
- Con: violates D8 — row-level updates are expensive on the
  underlying substrate; this fights the substrate.
- Con: scheduler retry overwrites the prior attempt's metadata
  (recorded_at, engine_version, error_summary if any) — forensic
  reconstruction impossible.

Reject on D2, D3, D4, D8.

### Option B — Upsert `dq_executions`, append-only `dq_check_results`

`dq_executions` is upserted on `execution_id` (mutable through
lifecycle and across scheduler retries). `dq_check_results` is
append-only on (`execution_id`, `check_id`) — the result row exists
once per (execution, check) and is never overwritten.

This is the apparent reading of foundation doc 04 §"Execution
Flow" — the row "is updated with final status".

**Trade-offs.**

- Pro: dashboards on `dq_executions` are simple (one row per id).
- Pro: per-check results are immutable, aligning with "the
  evaluation result is what it is once computed".
- Con: same D2/D3/D4 failures as Option A for `dq_executions`.
- Con: scheduler retries on `dq_check_results` produce key
  conflicts on (`execution_id`, `check_id`) — must either reject
  the retry's check writes or define ON CONFLICT semantics that
  themselves require a design decision.
- Con: B0-2 CC14 (active visibility of heterogeneous engine_versions
  across attempts) cannot be satisfied — `dq_executions` does not
  retain per-attempt rows.

Reject on D2 + the dq_check_results retry conflict.

### Option C — Append-only on both, canonical view per `execution_id`

Both tables are append-only. Every state transition on a logical
execution writes a new row; every scheduler retry of every check
writes a new row. A view named `dq_executions_current` returns the
LATEST row per `execution_id` and serves as the canonical
projection per B0-2 I4.

```
dq_executions((execution_id, attempt_id, recorded_at) PRIMARY KEY,
              status, engine_version, ruleset_version,
              supersedes_execution_id NULLABLE,
              started_at, completed_at NULLABLE,
              error_summary NULLABLE,
              ...)

dq_executions_current AS  -- view, not a table
  SELECT * FROM dq_executions
  WHERE ROW_NUMBER() OVER (PARTITION BY execution_id
                            ORDER BY recorded_at DESC) = 1

dq_check_results((execution_id, attempt_id, check_id) PRIMARY KEY,
                 result, evidence_summary,
                 sample_violating_rows, executed_at,
                 engine_version, ...)
```

`attempt_id` is a UUID assigned by the trigger handler at attempt
start; shared across rows in that attempt's lifecycle (e.g., the
`running` row and the terminal-status row). A scheduler retry of
the same `execution_id` assigns a **new** `attempt_id`.

Canonical query: `SELECT * FROM dq_executions_current WHERE execution_id = ?`
— one row.

Forensic query: `SELECT * FROM dq_executions WHERE execution_id = ?
ORDER BY recorded_at` — full history.

**Trade-offs.**

- Pro: D1 satisfied by the view's shape.
- Pro: D2 + D4 satisfied — every attempt row carries
  `engine_version`; heterogeneous values are first-class.
- Pro: D3 + D5 satisfied — append-only preserves history; storage
  multiplier is modest (2-3 rows per attempt × 1-3 attempts per
  typical execution = 2-9 rows; vs. 1 in Option A).
- Pro: D8 — append-and-query is the substrate's strength.
- Pro: scheduler retry produces a new `attempt_id` and a new set
  of `dq_check_results` rows; no key conflicts.
- Pro: `dq_check_results` keyed by (`execution_id`, `attempt_id`,
  `check_id`) makes "which retry attempt produced which result"
  unambiguous.
- Con: dashboards must learn to query `dq_executions_current`
  (the view) rather than `dq_executions` (the base table) for the
  common case. Documentation burden, mitigated by view naming.
- Con: D6 query simplicity requires the view — without it,
  consumers would have to write the LATEST projection themselves.
  The view solves this.
- Con: forensic queries see 2-3 rows per attempt (one per state
  transition); the per-row meaning ("the state at this moment")
  must be documented.

### Option D — Append-only on both, with an **eagerly-materialized** canonical projection

Same as Option C in its event-table shape, but the canonical
single-row-per-`execution_id` projection is **materialized into
a separate physical table** rather than computed by a lazy view.
Both event tables remain append-only; the engine writes
`dq_executions` (event history) and also updates a separate
`dq_executions_current` table (projection) on every state
transition.

```
dq_executions((execution_id, attempt_id, recorded_at) PRIMARY KEY,
              status, engine_version, ruleset_version, ...)
              -- append-only event history

dq_executions_current(execution_id PRIMARY KEY,
                       <same columns as dq_executions>)
              -- materialized projection; upserted on every new
              -- dq_executions row, OR refreshed periodically by
              -- an out-of-band process

dq_check_results((execution_id, attempt_id, check_id) PRIMARY KEY,
                 ...)
              -- append-only
```

Canonical query: `SELECT * FROM dq_executions_current WHERE execution_id = ?`
— direct table read, returns one row at bounded query cost.

Forensic query: `SELECT * FROM dq_executions WHERE execution_id = ?
ORDER BY recorded_at` — full history.

The difference from Option C is **where the projection lives**:
Option C computes it lazily at query time (a view scanning the
append-only event table); Option D materializes it eagerly into
a separate table.

**Trade-offs.**

- Pro: D1 + D6 satisfied directly; canonical queries are
  constant-cost regardless of attempt-row count growth over time.
- Pro: D2 + D4 + D3 retained for event history (append-only on
  both event tables; the projection table is a derived artifact,
  not source-of-truth).
- Pro: Dashboards never pay the per-query projection scan; cost
  compounds favorably for high-frequency dashboard traffic.
- Con: The projection table requires an update path on every
  state transition. Either (a) the engine writes both
  `dq_executions` (append) and `dq_executions_current` (upsert)
  in the same transaction — this reintroduces upsert on a public
  table and risks inconsistency if the two writes diverge; or
  (b) the projection is refreshed by an out-of-band process,
  introducing refresh lag (eventual consistency) and a new
  failure mode (stalled refresh shows stale state without
  indication).
- Con: The projection table is **derived state**, not
  source-of-truth. Operators reading `dq_executions_current` must
  understand it is a projection; the canonical record is in
  `dq_executions`. Option C's view is also derived, but
  always-current by construction — no refresh-lag failure mode.
- Con: Storage cost is duplicated: every `execution_id` has one
  row in `dq_executions_current` in addition to its event rows.
  Modest at expected volumes, but real.
- Con: Failure modes are richer than Option C. A refresh-on-write
  partial failure produces a divergent projection; an out-of-band
  refresh stall produces silent staleness. Option C has neither
  failure mode — the view's projection is always consistent with
  the event table by construction.

This option is rejected. The eager-materialization gains
(bounded per-query cost) are real but small at expected volumes
(O(millions) of event rows per year per environment;
single-digit-second projection scans). The introduced failure
modes (refresh-lag staleness, divergence-on-partial-write,
derived-state-vs-source-of-truth confusion) are significant
enough to outweigh the gains at current volume estimates.

A future revisit is reasonable: if Wave 3 operation shows the
lazy view's per-query cost compounding unsatisfactorily under
observed dashboard traffic, the engine can materialize the
projection at that point — informed by real query patterns, not
speculation. The lazy-view starting posture preserves that
option.

---

## Recommendation

Adopt **Option C** — append-only on both tables, with a
`dq_executions_current` view providing the canonical
single-row-per-`execution_id` projection.

The recommendation is grounded in:

- prior decision
  [`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md)
  (B0-2) — I4, CC4, CC10, CC14 — every constraint here is directly
  satisfied by Option C;
- foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Operating Posture" — visible failure over silent degradation
  (D4), determinism over convenience (D3 forensic integrity);
- foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"Layer 4 — Reporting" — the reporting schema is a documented
  contract for downstream consumers;
- prior decision
  [`2026-05-20-manifest-publication-semantics.md`](./2026-05-20-manifest-publication-semantics.md)
  (B0-5) — the platform's append-only posture for control-plane
  data (manifests, YAMLs) extends naturally to execution data.

The specific commitments beyond what those documents state are
**new contribution proposed here, requires review**:

1. Both `dq_executions` and `dq_check_results` are
   **append-only**; no DML (`UPDATE` / `DELETE`) is performed by
   the engine on these tables. **New contribution proposed here,
   requires review.**
2. `dq_executions_current` is committed as a **lazy view** (not
   a materialized view, not a derived table). The view returns
   the row with the latest `recorded_at` per `execution_id`,
   projected at query time. The lazy-vs-eager choice is argued
   in Option D's rejection: lazy projection preserves
   always-current semantics and avoids the refresh-lag and
   derived-state-divergence failure modes that an eagerly
   materialized projection introduces. The per-query cost is
   acceptable at expected volumes (O(millions) of event rows per
   year per environment; single-digit-second projection scans).
   Wave 3 may revisit if observed dashboard traffic makes the
   per-query cost unsatisfactory; the lazy-view starting posture
   preserves the option to materialize later, informed by real
   query patterns. **New contribution proposed here, requires
   review.**
3. `attempt_id` is committed as a **UUID assigned by the trigger
   handler** when an attempt begins; shared across rows of that
   attempt's lifecycle (the `running` row and the terminal-status
   row both carry the same `attempt_id`). **New contribution
   proposed here, requires review.**
4. `supersedes_execution_id` is a **nullable column on
   `dq_executions`** — not a separate lineage table. **New
   contribution proposed here, requires review** (B0-2 OQ-1
   resolution).
5. **CI dry-runs do not persist** to `dq_executions` or
   `dq_check_results`. The `ci-dry-run` enum value is **not added**
   to B0-2's `trigger_source` enum. CI dry-runs may compute
   `execution_id` for logging per B0-2's OQ-6 framing, but no row
   is written. **New contribution proposed here, requires review**
   (B0-2 OQ-6 resolution).
6. `status` on `dq_executions` is a closed enum:
   `running` / `success` / `failed` / `aborted`. Extension is
   additive per the same policy as B0-2's `trigger_source` enum
   (CC6 of B0-2). **New contribution proposed here, requires
   review.**

---

## Consequences

Adopting this recommendation commits the platform to the following.

**CC1. Tables and write model.** Two tables, both append-only:

- `dq_executions` — composite primary key
  (`execution_id`, `attempt_id`, `recorded_at`). One row per
  state transition. The engine writes **inserts only**; no
  `UPDATE` or `DELETE` statements are issued against this table.
- `dq_check_results` — composite primary key
  (`execution_id`, `attempt_id`, `check_id`). One row per
  (attempt × check). The engine writes **inserts only**; no
  `UPDATE` or `DELETE`.

`PRIMARY KEY` here is the logical key for the design; how it is
expressed in the BigQuery schema (clustering, partition column,
no enforced constraint) is Wave 3 schema-implementation detail.

**CC2. Canonical view.** A view named `dq_executions_current` is
defined as:

```sql
-- illustrative; exact dialect/syntax is Wave 3
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

The view satisfies B0-2 I4: a query keyed on `execution_id`
against `dq_executions_current` returns exactly one row, the
current canonical state. The view is **lazy** (not materialized);
it computes at query time. Dashboards target the view; forensic
queries target the base table.

**CC3. Required columns on `dq_executions`** (every row must
carry):

- `execution_id` (string, 64-char lowercase hex per B0-2 CC7).
- `attempt_id` (string, UUID per CC4 below).
- `recorded_at` (timestamp, microsecond precision, UTC).
- `status` (enum per CC6).
- `engine_version` (string) — the engine that wrote this row.
- `ruleset_version` (string) — the value from B0-2's formula
  inputs; denormalized here for query convenience.
- `entity` (string).
- `trigger_source` (enum per B0-2 CC6 — `scheduler`, `manual`, or
  `operator-rerun`).
- `started_at` (timestamp, nullable for the `running` transition
  row written before the engine begins processing; required for
  terminal rows).
- `completed_at` (timestamp, nullable for `running` rows; required
  for terminal rows).
- `error_summary` (string, nullable; populated when status is
  `failed` or `aborted`).
- `supersedes_execution_id` (string, nullable; populated for
  operator-rerun rows per CC5 of B0-2).

Other operational fields (concurrency budget, cost telemetry, etc.)
may be added additively without breaking the contract.

**CC4. `attempt_id` derivation.** `attempt_id` is a UUID
(version-4 or equivalent random) assigned by the trigger handler
when an attempt begins. The same `attempt_id` is carried across
all rows of that attempt's lifecycle (typically two rows: the
`running` transition and the terminal-status transition). A
scheduler retry of the same `execution_id` assigns a **new
`attempt_id`** at attempt start.

UUID is preferred over a monotonic-per-`execution_id` counter
(1, 2, 3, ...) because (a) no read-then-write coordination is
needed at attempt start — a scheduler-retry burst with multiple
trigger-handler instances cannot race on a counter; and (b) it
is consistent with B0-2's opaque-identifier posture (B0-2 CC8:
`execution_id` is opaque to consumers; `attempt_id` is similarly
opaque). Sortability across attempts of the same `execution_id`
is provided by `recorded_at` on each row (`ORDER BY recorded_at`),
not by `attempt_id`; `attempt_id` is opaque and not intended for
ordering.

**CC5. `supersedes_execution_id` storage shape.** A nullable
column on `dq_executions`, populated only on the first
state-transition row of an `operator-rerun` attempt. The value
is the `execution_id` of the prior run being superseded (B0-2
CC5). Chains of operator reruns produce a traversable graph via
this column; no separate lineage table is created. This resolves
B0-2's OQ-1. **New contribution proposed here, requires review.**

**CC6. `status` enum.** Closed enum with initial values:

- `running` — the trigger handler has created the row; the engine
  has not yet emitted a terminal-status row.
- `success` — every check evaluated and the engine emitted no
  operational errors. (Detailed mapping is B0-4.)
- `failed` — at least one check produced a `fail` or `degraded`
  result. (Detailed mapping is B0-4.)
- `error` — entity-level evaluation could not proceed (source
  table missing, manifest corruption, schedule misconfiguration
  per foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Failure Scope" — "Entity error"). The execution row is
  marked errored; per-check rows may be absent. The exact
  boundary between `failed` and `error` (specifically, whether
  one check that errors operationally promotes the entity to
  `error` or keeps it at `failed`) is **B0-4's decision** —
  B0-3 commits only that both values exist in the enum.
- `aborted` — the run was halted by a global operational
  condition (cost ceiling exceeded, manifest load failure, etc.
  per B0-7).

Extension is additive (matching B0-2 CC6 policy for
`trigger_source`); a new value's introduction does not break
existing rows. Removing or renaming a value is a breaking change
requiring a future ADR.

**CC7. Required columns on `dq_check_results`** (every row must
carry):

- `execution_id`, `attempt_id`, `check_id` (the composite key).
- `result` (enum: `pass` / `fail` / `degraded` / `error` — exact
  semantic mapping is B0-4).
- `executed_at` (timestamp).
- `engine_version` — the engine version that evaluated this
  check.
- `evidence_summary` (structured, exact shape Wave 3) —
  aggregate counts (e.g., rows scanned, rows failing).
- `sample_violating_rows` (repeated record, capped per
  configured limit — foundation doc 05 §"Evidence Retention").

A check_id corresponds to a specific check within a rule; its
shape (path-based, hash-based, declared name) is governed by the
rules workspace and the DSL — Wave 3 onboarding contract.

**CC8. CI dry-runs do not pollute production reporting state.**
B0-2 OQ-6's restriction is the no-pollution rule for the
production reporting tables. This study tracks that restriction
exactly: CI dry-runs write **no rows** to `dq_executions` or
`dq_check_results`. CI dry-runs exercise the trigger handler
and compute `execution_id` for logging (B0-2 OQ-6 framing) but
never produce a row in the two production tables.

Whether CI dry-runs persist **elsewhere** (a separate
`dq_ci_dry_runs` table, a flagged partition with `is_dry_run`,
or nowhere at all) is **deferred** — see OQ-7. This study
commits only to the no-pollution rule for the two production
tables; persistence to a distinct surface is not foreclosed and
does not affect this study's contracts.

The `ci-dry-run` `trigger_source` value addition to B0-2's enum
is therefore **conditional**: if Wave 3 / B1 elects to persist
CI dry-runs to a distinct surface, the enum gains `ci-dry-run`
additively per B0-2 CC6's extension policy; if it does not, the
enum remains as B0-2 committed it. **New contribution proposed
here, requires review** — the softening from "do not persist
anywhere" to "do not pollute production reporting state" tracks
B0-2 OQ-6's actual wording.

**CC9. No DML on these tables from engine code paths.** The
engine performs only `INSERT` operations against `dq_executions`
and `dq_check_results`. Any operator action that would mutate
rows (data corrections, purges, retroactive corrections) is
**out-of-band of the engine's INSERT-only commitment**. The
specific path for out-of-band corrections (admin SQL endpoint,
audited migration tool, etc.), the authorization model, and the
audit-log shape are Wave 3 admin tooling. This study commits
only that engine code paths never `UPDATE` or `DELETE` these
tables; the existence of an out-of-band correction path is not
foreclosed. Lifecycle policies that delete old rows for
retention enforcement are B1 — see OQ-2.

**CC10. B0-2 I4 satisfied by the view.** The canonical
single-row-per-`execution_id` projection is `dq_executions_current`
(CC2). Dashboards, alert systems, and external consumers that need
"the current state of execution X" query the view; queries
against the base table return all attempt rows and are
forensic-grade. This resolves B0-2's CC10 punt.

**CC11. B0-2 CC4 + CC14 — storage-layer precondition committed
here; satisfaction is shared.** B0-3 commits the **precondition**
for B0-2 CC14's visibility obligation: `engine_version` is a
required column on every row of both `dq_executions` and
`dq_check_results` (per CC3 and CC7). The view
`dq_executions_current` returns the latest attempt's
`engine_version` as the "effective evaluator" per B0-2 CC14.

The visibility obligation itself — that reporting tools include
`engine_version` in default projections, and that admin tools
surfacing attempts of an `execution_id` include it without
requiring a flag — is **B0-2 CC14's commitment**, enforced
through review per B0-2 CC14's "must be flagged in review"
clause. B0-3 cannot enforce reporting-tool behavior; B0-3
ensures the data is available. The division of responsibility
is explicit: B0-3 commits the storage precondition; B0-2 commits
the reporting obligation; the visibility requirement is
satisfied when both are honored.

**CC12. Schema is a public contract.** Once
`dq_executions`, `dq_check_results`, and `dq_executions_current`
exist with consumers, the column names, the enum values
(`status`, `trigger_source`, `result`), and the composite-key
shape are public. Changes to these surfaces follow the same
breaking-change protocol as B0-1's schema migration: removal or
type-change of a column, removal of an enum value, or change to
the composite-key shape requires a future ADR with a migration
path and a compatibility window. Additive changes (new columns,
new enum values) do not.

**CC13. Append-only multiplier is bounded under bounded
retention.** A typical execution produces 2 rows in
`dq_executions` (running + terminal) per attempt × 1-3 attempts
per scheduler-retry cycle = 2-6 rows. A typical check produces
1 row per attempt × 1-3 attempts = 1-3 rows in
`dq_check_results`. This is a modest multiplier on the naive
"1 row per execution / 1 row per check" baseline; storage cost
remains O(n) in evaluation volume — **bounded over time only if
retention is bounded**. Under unbounded retention the tables
grow linearly with evaluation history. Per-environment retention
parameters are B1 (OQ-2).

---

## Open Questions

- **OQ-1. Partition and clustering strategy.** Whether
  `dq_executions` is partitioned by `recorded_at` (daily) or by
  `started_at`, and which columns serve as clustering keys, is
  **out-of-scope for current cycle** — it is a B1 cost/performance
  decision. The composite-key shape committed in CC1 does not
  preclude any partition strategy.

- **OQ-2. Retention policy on the append-only tables.** How long
  rows are retained, and whether retention is uniform across
  environments or differentiated, is **out-of-scope for current
  cycle** — it is a B1 decision adjacent to B1-6 (failed-sample
  retention). This study commits only to append-only writes; how
  long the appends accumulate is B1.

- **OQ-3. Whether `dq_check_results_current` view exists.** A
  parallel "latest attempt's check results per `execution_id`"
  view would simplify dashboards that show per-check results
  alongside the canonical execution row. Whether to define such
  a view is **out-of-scope for current cycle** — it is a Wave 3
  schema-implementation refinement. CC2 commits only the
  `dq_executions_current` view; downstream views are additive.

- **OQ-4. Operational data-correction path.** When a row is
  written incorrectly (data quality bug in the engine, schema
  migration artifact), the operator path for correcting the table
  is **out-of-scope for current cycle** — it is Wave 3 admin
  tooling, audited and out-of-band per CC9. This study commits
  only that the engine itself does not perform such corrections.

- **OQ-5. Stream-runner reporting alignment.** Foundation doc 04
  §"Evolution Flow" anticipates a stream runner that reuses the
  reporting schema. How streaming results interact with these
  tables (additional `trigger_source` value, separate
  `dq_stream_results` table, or aggregated time-window rows in
  `dq_check_results`) is **out-of-scope for current cycle** — it
  is B2-4 (Stream reporting continuity) in the decision log.
  Append-only is forward-compatible with any of those choices.

- **OQ-6. Column-level nullability and type details.** The exact
  BigQuery types (STRING vs. BYTES for hashes, TIMESTAMP vs.
  DATETIME for windows, REPEATED RECORD shape for evidence) and
  per-column nullability rules are **out-of-scope for current
  cycle** — Wave 3 schema implementation. The fields enumerated
  in CC3 and CC7 are the contract; the BigQuery rendering of
  each is downstream.

- **OQ-7. CI dry-run persistence surface (if any).** Whether CI
  dry-runs persist to a distinct surface (a separate
  `dq_ci_dry_runs` table, a flagged partition with `is_dry_run`,
  or nowhere at all) is **out-of-scope for current cycle** —
  Wave 3 / B1 follow-up. This study commits only to the
  no-pollution rule for `dq_executions` and `dq_check_results`
  (CC8). The decision affects whether B0-2's `trigger_source`
  enum is extended with `ci-dry-run` (additive per B0-2 CC6
  extension policy).

No open question in this list blocks the write-model shape. All
items above are parameters, downstream view definitions, or
schema-implementation details on top of the locked shape.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0003-result-write-model.md

The `0003` is provisional and assigned at promotion time. If the
Wave 3 ADR numbering convention orders by promotion date rather
than by B0 sequence, the number changes; the slug
(`result-write-model`) does not. This follows the same convention
adopted in
[`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
(B0-1, Promotion target section).

The MADR ADR rewrites this study for an external-reviewer
audience (no `studies/` back-references per R8), folds in any
updates from B0-4 (failure scope) and B0-6 (alert routing) that
intersect with the table schemas or the canonical view, and
updates the relevant sections of foundation doc
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
§"Layer 4 — Reporting" and §"Execution Flow" to reflect the
committed write model.
