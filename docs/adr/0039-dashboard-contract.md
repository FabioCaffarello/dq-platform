<!-- path: docs/adr/0039-dashboard-contract.md -->

# ADR-0039 — Dashboard Contract

- **Status:** accepted
- **Date:** 2026-05-26

---

## Context

The platform exposes two surfaces that downstream
consumers (SQL-based BI tools, ad-hoc SQL query runners,
Prometheus-compatible scrape consumers, observability
platforms, custom dashboard implementations) read:

- **Tables.** [ADR-0003](./0003-result-write-model.md)
  commits the append-only `dq_executions` and
  `dq_check_results` tables, the canonical lazy view
  `dq_executions_current`, the required column
  inventories, and the `status` enum as
  closed-but-additive. [ADR-0004](./0004-failure-scope.md)
  commits the per-check → per-execution status mapping;
  [ADR-0026](./0026-failure-scope-aggregated.md) extends
  it for record-mode aggregation.
- **Metrics.** [ADR-0010](./0010-substrate-posture.md)
  §3.2 commits that the engine exposes a metrics
  endpoint locally; [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md)
  CC10 commits that every failure path emits a log +
  metric + span signal (three-channel emission is
  binding; exact metric and span names are scaffolding
  details).

What none of those ADRs commits is the **consumer
contract** that sits on top of the two surfaces: which
view is the canonical read surface, which columns and
metric names + labels are guaranteed stable across
engine versions, what evolution rules apply when a new
column or metric ships. Foundation 05 §"Metrics" +
§"Dashboards" promised a baseline dashboard "ships
during Wave 3", but Wave 3 closed with the dashboard
unimplemented. B2-6 was registered to commit the
contract that any future dashboard — the foundation-05
baseline or any external consumer — reads against.

The principles bearing on the decision are **P3**
(ownership is explicit — without a committed contract,
each downstream consumer invents its own interpretation
of `status=failed` vs `status=error` and the consumer
set diverges over time) and **P5** (evolution must be
contract-driven — the consumer surface needs a
documented compatibility rule so consumer code can be
written against a *version*, not a *snapshot*). R3 (do
not revisit settled architecture) is the operating
constraint: ADR-0003, ADR-0004, ADR-0007, ADR-0010,
ADR-0026 are preserved; this ADR commits the
consumption layer on top.

The metric-emission code itself is deferred per
`engine/internal/runner/runner.go` lines 15-27 (the
engine today emits log signals only; metrics + spans
queued for a Phase-4c follow-up). The contract can be
committed today without blocking on emission; when
emission lands, it lands against this contract.

---

## Decision

### Two consumer-surface contracts

Two distinct consumer audiences each get their own
contract:

1. **SQL-based consumers** read `dq_executions_current`
   and `dq_check_results` over the BigQuery tabular
   substrate.
2. **Time-series-metric consumers** scrape the engine's
   Prometheus-compatible metrics endpoint at
   `/metrics` on the engine's primary HTTP port.

The metrics-endpoint *path* (`/metrics`) is a new
contribution committed by this ADR; ADR-0010 §3.2
commits the existence of the endpoint without
committing the path. The path is the contract surface
operators key on.

### Table contract

**Canonical read surface.** `dq_executions_current` (the
lazy view from ADR-0003 §2 projecting the
latest-`recorded_at` row per `execution_id`) is the
canonical current-state read surface. The base table
`dq_executions` and the per-check table
`dq_check_results` are forensic / historical surfaces;
consumers querying them accept multiple rows per
execution (the retry history) and are responsible for
their own de-duplication.

**`dq_executions_current` guaranteed columns** — the
ADR-0003 §3 inventory with the consumer-facing
stability tier explicit:

| Column | Type | Stability tier |
|---|---|---|
| `execution_id` | string (64-char lowercase hex) | stable across engine minor versions |
| `attempt_id` | string (UUID) | stable across engine minor versions |
| `recorded_at` | timestamp (UTC, microsecond) | stable across engine minor versions |
| `status` | enum (closed, additive) | stable values; new enum members are additive |
| `engine_version` | string | stable across engine minor versions |
| `ruleset_version` | string | stable across engine minor versions |
| `entity` | string | stable across engine minor versions |
| `trigger_source` | enum (ADR-0002) | stable; additive |
| `started_at` / `completed_at` | timestamp (nullable) | stable across engine minor versions |
| `error_summary` | string (nullable) | stable across engine minor versions |
| `supersedes_execution_id` | string (nullable) | stable across engine minor versions |

**`dq_check_results` guaranteed columns** — the
ADR-0003 §7 inventory:

| Column | Type | Stability tier |
|---|---|---|
| `execution_id` | string | stable across engine minor versions |
| `attempt_id` | string | stable across engine minor versions |
| `check_id` | string | stable across engine minor versions |
| `result` | enum (`pass` / `fail` / `degraded` / `error` per ADR-0004) | stable values; new enum members are additive |
| `executed_at` | timestamp | stable across engine minor versions |
| `engine_version` | string | stable across engine minor versions |
| `evidence_summary` | structured aggregate counts (rows scanned, rows failing, etc.) | shape is consumer-facing; field-level additions are additive |
| `sample_violating_rows` | repeated record (capped per ADR-0031) | shape is consumer-facing; field-level additions are additive |

**Aggregation rules.** Execution-level rollup from
check-level results is the failure-scope mapping
(ADR-0004 + ADR-0026): an execution is `success` iff
every check produced `pass`; `failed` if at least one
check produced `fail` or `degraded`; `error` if no
check produced a non-error; `aborted` if a global
condition halted the run.

Per-entity rollups across multiple executions are
consumer-computed (use the `entity` column with a time
window on `recorded_at`). The platform does not commit
a pre-aggregated rollup table today — reserved as a
future ADR when concrete consumer demand surfaces.

### Metric contract

**Endpoint.** The engine binary serves a
Prometheus-compatible scrape endpoint at `/metrics` on
its primary HTTP port. Content type and exposition
format follow the Prometheus exposition format the
substrate collector expects.

**Inventory:**

| Metric name | Type | Labels | Meaning |
|---|---|---|---|
| `dq_runs_total` | counter | `entity`, `status`, `trigger_source`, `mode` | One increment per terminal execution row in `dq_executions`. |
| `dq_checks_evaluated_total` | counter | `entity`, `check_id`, `result`, `mode` | One increment per check evaluation. |
| `dq_run_duration_seconds` | histogram | `entity`, `status`, `mode` | Distribution of `completed_at - started_at` per terminal execution. |
| `dq_check_duration_seconds` | histogram | `entity`, `check_id`, `mode` | Per-check evaluation duration. |
| `dq_bytes_scanned` | gauge | `entity`, `check_id` | Most-recent bytes-scanned value per (entity, check_id). Trend dashboards consume the metric's scrape history. |
| `dq_queue_depth` | gauge | `state` (= `scheduled` or `running`) | Count of runs the scheduler currently tracks, split by state. |
| `dq_scheduler_triggers_managed` | gauge | `state` (= `healthy` or `errored`) | Count of triggers the scheduler currently manages, split by state. |
| `dq_loader_refresh_failures_total` | counter | `error_class` (ADR-0007 taxonomy) | Loader-refresh failures, classified by ADR-0007's error class. |

**Label value sources.** Every label value comes from
a committed source:

- `entity` — free-string from `_owners.yaml` per
  ADR-0006.
- `check_id` — string from the rule YAML's
  `checks[].check_id` per ADR-0022.
- `status` — closed-but-additive enum from ADR-0003 §6.
- `result` — closed-but-additive enum from ADR-0004
  (`pass`, `fail`, `error`, `degraded`).
- `trigger_source` — closed-but-additive enum from
  ADR-0002 (`scheduler`, `manual`, `operator-rerun`).
- `mode` — closed enum from ADR-0021 (`set`, `record`).
- `error_class` — closed-but-additive enum from
  ADR-0007.

**Cardinality posture.** Label combinations are
bounded by the entity × check_id cross-product. At
Phase-6 onboarding scale per the foundation documents
(small N entities and small M checks per entity),
per-metric cardinality stays well below substrate-
collector stress points. The contract does not commit
a numeric cardinality ceiling; the substrate is
responsible for its own ingest limits per ADR-0010.
A future ADR commits a per-metric ceiling if
cardinality growth produces ingest failures in
production.

### Evolution rules

The same evolution rule applies to both contracts:

1. **Additive within an engine-major-version is
   allowed.** A new column on a table, a new metric, a
   new label value drawn from a closed-but-additive
   enum — any consumer reading the existing surface
   continues to read it. New consumers can opt into the
   new shape.

2. **Breaking changes require an engine-major bump.**
   Removing a column, renaming a column or metric or
   label, narrowing a type, replacing a closed-enum
   value — each is a breaking change requiring an
   engine-major bump. The migration window posture is
   [ADR-0035](./0035-compatibility-window-duration.md)'s
   N-1 + 90-day calendar-time floor — consumers have at
   least one major version OR 90 calendar days
   (whichever is longer) to migrate.

3. **Closed-but-additive enums** (`status`, `result`,
   `trigger_source`, `error_class`) extend additively
   within a major; consumers MUST handle "unknown enum
   value" gracefully (e.g., a default path or a
   one-line log) because the platform may introduce a
   new value at any minor-version bump.

4. **`evidence_summary` and `sample_violating_rows` are
   consumer-facing payloads with structured shapes.**
   Adding a field is additive; removing a *documented*
   field is breaking. Undocumented fields may appear
   and disappear at engine-minor cadence — consumers
   should not rely on undocumented fields. The
   currently-documented inventory is the union of what
   ADR-0003 §7 commits.

### Naming convention — the `dq_` prefix

All platform-emitted tables, views, and metrics carry
the `dq_` prefix so consumers in a shared namespace can
isolate platform signals from other signals without
collision. The prefix is part of the contract; renaming
it would be a breaking change requiring an engine-major
bump.

### Baseline-dashboard implementation — deferred

The example consumer dashboard from foundation 05
§"Dashboards" — a single page showing run success rate,
per-entity check pass/fail rates, cost per entity per
day, alerting volume per owner, scheduler health — is
**out of scope for this ADR**. It is a *consumer* of
the contract this ADR commits, not part of the contract
itself.

A new B2 row is registered for the baseline-dashboard
slice: implementation of the example dashboard against
this ADR's contract, paced post-Phase-4c metric
emission so the dashboard can smoke-test end-to-end.
The slice's ADR commits the substrate choice and the
workspace placement (likely `deploy/dashboards/` or a
new `dashboards/` workspace per ADR-0019's tooling
posture).

### Why this does not reopen ADR-0003 / ADR-0004 / ADR-0007 / ADR-0010 / ADR-0026

Each prior ADR commits a piece of the underlying
surface this ADR commits the consumer contract over:

- **ADR-0003** commits the table columns and the
  `status` enum. This ADR adds the consumer-facing
  stability tier (which columns are guaranteed stable
  across minor versions). ADR-0003 is preserved.
- **ADR-0004 / ADR-0026** commit the aggregation
  rules. This ADR cites them as the canonical
  aggregation reference; it does not re-derive them.
- **ADR-0007** commits the log + metric + span
  emission obligation (three-channel; exact names are
  scaffolding). This ADR commits the metric *shape*
  (names + labels) the emission produces.
- **ADR-0010** commits the metrics-endpoint existence.
  This ADR commits the endpoint *path* (`/metrics`).

### Why this does not reopen ADR-0001 / ADR-0035

ADR-0001 commits the rule-schema compatibility model;
ADR-0035 commits the schema-version-retirement window.
This ADR borrows the same posture (additive within
major; breaking requires major bump; N-1 + 90-day
window) for the dashboard contract surfaces. The
dashboard surfaces are engine-version-bound (not
rule-schema-version-bound), so the version axis is
"engine semver" — but the discipline is identical.

---

## Consequences

1. **A formal consumer contract is committed for both
   reporting surfaces.** SQL consumers can write
   `SELECT ... FROM dq_executions_current WHERE
   status IN (...)` against a stable column list and a
   committed status enum. Metric consumers can scrape
   `/metrics` and write panels against a stable metric
   name + label inventory.

2. **The metric inventory is committed at contract
   level even though emission code is deferred.** Per
   `engine/internal/runner/runner.go` lines 15-27, the
   current engine emits log signals only; metric
   emission is queued for a Phase-4c follow-up. When
   emission lands, it lands against this ADR's
   inventory — no contract redesign happens at
   emission time.

3. **The metrics-endpoint path `/metrics` is the
   contract surface.** ADR-0010 §3.2 commits the
   endpoint's existence; this ADR commits the path.
   Operators wiring scrape jobs and substrate
   collectors key on `/metrics`.

4. **The baseline dashboard is explicitly deferred to a
   B2 consumer slice.** The demonstrative dashboard
   from foundation 05 §"Dashboards" is a consumer of
   this ADR's contract, not part of the contract
   itself. The slice's ADR will commit the substrate
   choice and the workspace placement.

5. **Evolution rules align with the existing
   compatibility framework.** Additive within an
   engine-major-version is allowed; breaking changes
   require an engine-major bump and the ADR-0035
   migration-window posture. Consumers writing against
   a documented engine-major-version can reason about
   their compatibility window the same way rule
   authors do for rule schemas.

6. **The `dq_` prefix is part of the contract.** The
   tables and metrics share the prefix so consumers in
   a shared namespace can isolate platform signals.
   Renaming the prefix is a breaking change.

7. **Closed-but-additive enums become part of the
   consumer surface.** Consumers MUST handle "unknown
   enum value" gracefully — additive extensions to
   `status`, `result`, `trigger_source`, `error_class`
   are not breaking and may appear within a minor-
   version bump.

8. **`evidence_summary` and `sample_violating_rows` are
   consumer-facing payloads.** Field-level additions
   are additive; documented field-level removals are
   breaking. Undocumented fields may appear and
   disappear at engine-minor cadence — consumers
   should not rely on undocumented fields.

9. **No `dq_entity_rollup` pre-aggregated table is
   committed.** Consumers compute their own per-entity
   rollups against `dq_executions_current` + a time
   window on `recorded_at`. A future ADR may commit a
   pre-aggregated rollup table when concrete consumer
   demand surfaces.

10. **B2-6 closes.** The decision-log B2-6 row moves
    to `resolved-adr`. One new B2 row registers the
    baseline-dashboard consumer slice.

11. **ADR-0003, ADR-0004, ADR-0007, ADR-0010,
    ADR-0026, ADR-0001, ADR-0035 are preserved.** This
    ADR layers a consumer contract on top of their
    commitments without amending them.

12. **Three deferred items are registered out-of-scope:**
    pre-aggregated `dq_entity_rollup` table (waits for
    concrete consumer demand); per-metric cardinality
    ceiling (waits for concrete cardinality-pressure
    signal); documented field-level inventory for
    `evidence_summary` and `sample_violating_rows`
    (waits for the baseline-dashboard consumer slice
    which is the first programmatic consumer).
