<!-- path: studies/decisions/2026-05-26-b2-6-dashboard-contract.md -->

# B2-6 — Dashboard Contract

## Context

Several prior ADRs commit the surfaces a downstream
consumer (a SQL-based BI tool, a time-series
observability platform, a custom dashboard implementation,
or a forensic SQL query) reads from this platform:

- [ADR-0003](../../docs/adr/0003-result-write-model.md)
  commits the append-only `dq_executions` and
  `dq_check_results` tables, the canonical lazy view
  `dq_executions_current`, the required column inventory
  on `dq_executions`, and the `status` enum
  (`running` / `success` / `failed` / `error` / `aborted`)
  as closed-but-additive.
- [ADR-0004](../../docs/adr/0004-failure-scope.md) commits
  the per-check → per-execution status mapping (the
  failure-scope rules).
- [ADR-0026](../../docs/adr/0026-failure-scope-aggregated.md)
  commits the record-mode aggregation extension to the
  failure-scope rules.
- [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md)
  CC10 commits that every failure path emits a log +
  metric + span signal (the three-channel commitment is
  binding; the exact metric and span names are
  scaffolding details).
- [ADR-0010](../../docs/adr/0010-substrate-posture.md)
  §3.2 commits that the engine exposes a metrics
  endpoint locally — the substrate-row promise behind
  ADR-0007's emission obligation.
- The metric emission code itself is deferred per
  `engine/internal/runner/runner.go` lines 15-27 (today
  the engine emits log signals only; metrics + spans
  queued for a Phase-4c follow-up).
- `studies/foundation/05-operational-discipline.md`
  §"Metrics" + §"Dashboards" lists the standard metric
  inventory (run-level counters, check-level counters,
  duration histograms, cost gauges, queue depth, scheduler
  state) and promised a "baseline dashboard ships during
  Wave 3."

The baseline-dashboard implementation did **not** ship
during Wave 3; Wave 3 closed at the gate (per the W3-P8
phase-closing commits) with the dashboard committed as a
follow-up. B2-6 was registered for this work.

What B2-6 actually asks is **not** "what dashboard ships?"
— ADR-0003 and ADR-0007 already commit the underlying
surfaces a dashboard would read from. B2-6 asks:

> Which metrics and dimensions are guaranteed for
> downstream consumers? Avoids each consumer inventing
> its own interpretation.

The question is about the **consumer contract** that
sits on top of the table and metric surfaces. Two
distinct consumer audiences exist:

1. **SQL-based consumers** — BI tools and ad-hoc SQL
   query runners that read `dq_executions_current` and
   `dq_check_results` over the BigQuery tabular
   substrate.
2. **Time-series-metric consumers** — observability
   platforms that scrape the engine's Prometheus-
   compatible metrics endpoint for run-level /
   check-level / cost signals.

The contract this ADR commits is to the consumer
*categories*, not to any specific tool. Naming
`Prometheus`, `OpenTelemetry`, and `BigQuery` is allowed
as environment substrate per R5; no other tools are
named.

The principles bearing on the decision are **P3**
(ownership is explicit — without a committed contract,
each downstream consumer invents its own interpretation
of `status=failed` vs `status=error` and the consumer
set diverges over time) and **P5** (evolution must be
contract-driven — the consumer surface needs a documented
compatibility rule so consumer code can be written against
a *version*, not a *snapshot*). R3 (do not revisit
settled architecture) is the operating constraint:
ADR-0003, ADR-0004, ADR-0007, ADR-0026 are preserved;
this ADR commits the consumption layer on top.

What B2-6 must commit:

1. **The two consumer-surface contracts** — what tables
   + view a SQL consumer can rely on; what metrics + label
   dimensions a time-series consumer can rely on.
2. **The stability tier** — what's guaranteed within a
   minor engine-version bump vs. across a major bump.
3. **The evolution rules** — what changes are additive
   (no consumer code change required) vs. breaking
   (requires migration window per the existing
   compatibility framework).
4. **The baseline-dashboard implementation posture** —
   does this ADR ship the baseline dashboard, or defer
   it to a consumer slice?
5. **The honest-gap acknowledgment** — metric emission
   is deferred per `runner.go`; the contract commits the
   shape, implementation lands additively.

---

## Decision Drivers

- **DD-1 — Two distinct consumer surfaces, two
  contracts.** SQL consumers read tables; metric consumers
  scrape the metrics endpoint. The two surfaces have
  different shapes (table columns vs. metric labels),
  different evolution mechanisms (DDL vs. metric-name
  changes), and different fidelity guarantees (SQL is
  exact; metric scraping is sampled). A single unified
  "data contract" that conflates them produces a worse
  contract for both audiences. Commit two contracts in
  one ADR.

- **DD-2 — The table-side contract is already mostly
  committed; this ADR closes the consumption gap.**
  ADR-0003 commits the column inventory; ADR-0004 commits
  the status mapping; ADR-0026 commits the record-mode
  aggregation. What's not yet committed is the
  *consumer*-facing posture: which view is the canonical
  read surface, which columns are guaranteed stable
  across engine minor versions, which are additive-extend
  candidates. This ADR commits those promises explicitly
  so a consumer can write `SELECT ... FROM
  dq_executions_current WHERE status IN (...)` against a
  *stable contract*, not against a snapshot.

- **DD-3 — Metric-emission code is deferred, but the
  contract shape is committable now.** Per
  `engine/internal/runner/runner.go` lines 15-27, the
  current engine emits log signals only; metric and span
  emission is queued for a Phase-4c follow-up. The
  *inventory* of metrics — what's emitted, what labels
  it carries, what type (counter/histogram/gauge) — can
  be committed now without blocking on the emission
  code. When the emission lands, it lands against the
  committed inventory. This matches the honest-gap
  pattern set by ADR-0010's lazy-view Partial row and
  B1-11 (CAS-fidelity gap) — contract surface committed,
  implementation lands additively.

- **DD-4 — Evolution rules must align with the existing
  compatibility framework.** [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
  commits the additive-within-major contract for the
  rule schema, and [ADR-0035](../../docs/adr/0035-compatibility-window-duration.md)
  commits the N-1 + 90-day calendar-time floor for
  schema-version retirement. The consumer contract this
  ADR commits inherits the same posture: additive within
  an engine-major-version; breaking changes carry an
  engine-major bump and the same migration-window
  discipline.

- **DD-5 — The baseline dashboard is a consumer of the
  contract, not part of the contract itself.** Foundation
  05 named "a baseline dashboard ships with the platform";
  what that meant was "the platform ships an example
  consumer demonstrating the contract." That demonstrative
  artefact is independent of the contract surface and is
  itself a consumer (it would query
  `dq_executions_current`, scrape the metrics endpoint,
  and visualize). This ADR commits the contract; the
  baseline dashboard ships as a separate consumer slice
  under B2 — matching the design-only pattern set by
  ADR-0032 (baseline strategy; first baselined kind
  ships its consumer slice).

- **DD-6 — Naming convention: `dq_` prefix is the
  existing convention.** The tables (`dq_executions`,
  `dq_check_results`, `dq_executions_current`) already
  carry the `dq_` prefix per ADR-0003. The same prefix
  applies to all platform-emitted metrics
  (`dq_runs_total`, `dq_checks_evaluated_total`, etc.)
  so consumers can isolate platform signals from other
  metrics in a shared observability namespace. The
  prefix is part of the contract; renaming it would be a
  breaking change.

---

## Considered Options

### Option 1 — Commit both contracts; defer baseline dashboard to a B2 consumer slice (recommended)

This ADR commits two contracts:

**Table contract.**

- Canonical read surface: `dq_executions_current` for
  the current-state view; `dq_executions` +
  `dq_check_results` for forensic / historical queries.
- Required column inventory (already committed by
  ADR-0003 §3): `execution_id`, `attempt_id`,
  `recorded_at`, `status`, `engine_version`,
  `ruleset_version`, `entity`, `trigger_source`,
  `started_at`, `completed_at`, `error_summary`,
  `supersedes_execution_id`.
- `dq_check_results` required columns (ADR-0003 §7):
  `execution_id`, `attempt_id`, `check_id`, `result`
  (`pass` / `fail` / `degraded` / `error` per
  ADR-0004), `executed_at`, `engine_version`,
  `evidence_summary` (structured aggregate counts),
  `sample_violating_rows` (repeated record capped at
  a configured limit).
- Status enum: closed-but-additive (ADR-0003 §6).
- Evolution: new columns are additive within an
  engine-major-version; column removal or type-narrowing
  requires an engine-major bump under ADR-0035's
  migration-window posture.

**Metric contract.**

- Endpoint: HTTP-served Prometheus-format scrape
  endpoint on the engine binary (path: `/metrics`).
- Inventory (the foundation-05 list, formalized):
  - `dq_runs_total` (counter; labels: `entity`,
    `status`, `trigger_source`, `mode`).
  - `dq_checks_evaluated_total` (counter; labels:
    `entity`, `check_id`, `result`, `mode`).
  - `dq_run_duration_seconds` (histogram; labels:
    `entity`, `status`, `mode`).
  - `dq_check_duration_seconds` (histogram; labels:
    `entity`, `check_id`, `mode`).
  - `dq_bytes_scanned` (gauge; labels: `entity`,
    `check_id`).
  - `dq_queue_depth` (gauge; labels: `state` =
    `scheduled` / `running`).
  - `dq_scheduler_triggers_managed` (gauge; labels:
    `state` = `healthy` / `errored`).
  - `dq_loader_refresh_failures_total` (counter;
    labels: `error_class` per ADR-0007).
- Label dimensions: `entity`, `check_id`, `status`,
  `result`, `mode`, `trigger_source`, `error_class` —
  each drawn from a committed enum or a free-string
  identifier the contract already commits.
- Evolution: adding a new metric or a new label value
  is additive; removing a metric or renaming a label is
  breaking and requires an engine-major bump.

**Baseline-dashboard implementation deferral.**

The example consumer dashboard (a single page showing
run success rate, per-entity check pass/fail rates,
cost per entity per day, alerting volume per owner,
scheduler health — foundation 05's promised content)
ships as a B2 consumer slice in a separate session. The
slice picks a substrate (an example serialized dashboard
file for a commodity dashboarding tool), commits the
queries it issues, and ships into `deploy/` or a new
`dashboards/` workspace depending on the substrate
chosen. The slice's pacing is post-emission: it lands
after Phase-4c metric-emission code so the dashboard
can be smoke-tested end-to-end.

**Strengths.** Commits the contract today (closing the
"each consumer invents its own interpretation" gap that
B2-6's "Why It Matters" cites). Honors the honest-gap
pattern for metric emission (contract committed,
implementation lands additively). Preserves the
design-only / consumer-slice separation that ADR-0030,
ADR-0032, ADR-0033 follow. Decouples the contract
landing date from the dashboard-implementation landing
date.

**Trade-offs.** The baseline dashboard remains a future
deliverable — operators using the platform locally
today still rely on direct SQL queries or hand-rolled
panels until the consumer slice lands. Acceptable: the
contract is the load-bearing artefact; the dashboard is
a demonstration of consumption.

### Option 2 — Commit only the table contract; defer metrics entirely

Drop the metric inventory from this ADR; commit only
the SQL table + view contract. Metric inventory waits
for a future ADR once emission lands.

**Strengths.** Smaller scope; fewer implementation
gaps acknowledged.

**Trade-offs.** Splits one logical decision into two.
Today's consumers building against the metrics endpoint
(if any — currently none, but the endpoint is committed
by ADR-0007 CC14 + foundation 05) have no stable
contract to write against. When emission lands, the
choice of metric names + labels becomes a *forced*
decision under implementation pressure rather than a
*designed* contract. Rejected: the contract is exactly
what should land before emission, not after.

### Option 3 — Ship baseline dashboard now

Commit the table contract + metric contract AND ship
the baseline dashboard implementation in this ADR.

**Strengths.** Completes the foundation-05 promise in
one cycle.

**Trade-offs.** The baseline dashboard requires either:
(a) the metric emission code to be in place (deferred
per `runner.go`), so the dashboard can read actual
data; or (b) committing the dashboard against a contract
that has no live emission yet — meaning the dashboard
ships dead and "demonstrates" nothing. Option (a) makes
this ADR's scope balloon to include Phase-4c emission
work + dashboard impl. Option (b) violates the
honest-gap pattern (no dead artefacts in `deploy/`).
Rejected: the contract is independent of the dashboard;
ship them in two slices.

---

## Recommendation

**Option 1.** Commit both contracts in this ADR; defer
the baseline-dashboard consumer slice to a B2 follow-up.

### Table contract — formalized

The canonical consumer read surface is
`dq_executions_current` (per ADR-0003 §2 — a lazy view
projecting the latest-`recorded_at` row per
`execution_id`). The base table `dq_executions` and the
per-check table `dq_check_results` are forensic /
historical surfaces; consumers querying them accept
multiple rows per execution (the retry history) and are
responsible for any de-duplication.

**`dq_executions_current` guaranteed columns** (the
ADR-0003 §3 inventory, with the consumer-facing
stability tier explicit):

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

**`dq_check_results` guaranteed columns** (ADR-0003 §7):

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

**Aggregation rules:**

- Execution-level rollup from check-level results is
  the failure-scope mapping (ADR-0004 + ADR-0026). The
  rule's prose: an execution is `success` iff every
  check produced `pass`; `failed` if at least one check
  produced `fail` or `degraded`; `error` if no check
  produced a non-error; `aborted` if a global condition
  halted the run.
- Consumers wanting per-entity rollups across multiple
  executions use the `entity` column with a time window
  on `recorded_at`. The platform does not commit a
  pre-aggregated `dq_entity_rollup` table at this ADR's
  writing — consumers compute their own rollups against
  the canonical view.

### Metric contract — formalized

**Endpoint.** Each engine binary serves a
Prometheus-compatible scrape endpoint at `/metrics` on
its primary HTTP port (per ADR-0010 §3.2's
"structured logs / metrics endpoint" commitment). The
endpoint's content type and exposition format follow
the Prometheus exposition format the substrate
collector expects.

**Inventory.** The committed metric inventory:

| Metric name | Type | Labels | Meaning |
|---|---|---|---|
| `dq_runs_total` | counter | `entity`, `status`, `trigger_source`, `mode` | One increment per terminal execution row in `dq_executions`. |
| `dq_checks_evaluated_total` | counter | `entity`, `check_id`, `result`, `mode` | One increment per check evaluation. |
| `dq_run_duration_seconds` | histogram | `entity`, `status`, `mode` | Distribution of `completed_at - started_at` per terminal execution. |
| `dq_check_duration_seconds` | histogram | `entity`, `check_id`, `mode` | Per-check evaluation duration. |
| `dq_bytes_scanned` | gauge | `entity`, `check_id` | Most-recent bytes-scanned value per (entity, check_id). Trend dashboards consume the metric's scrape history rather than a `previous` label. |
| `dq_queue_depth` | gauge | `state` (= `scheduled` or `running`) | Count of runs the scheduler currently tracks, split by state. |
| `dq_scheduler_triggers_managed` | gauge | `state` (= `healthy` or `errored`) | Count of triggers the scheduler currently manages, split by state. |
| `dq_loader_refresh_failures_total` | counter | `error_class` (ADR-0007 taxonomy) | Loader-refresh failures, classified by ADR-0007's error class. |

**Label value sources.** Every label value comes from
a committed source:

- `entity` — free-string from `_owners.yaml` per
  ADR-0006.
- `check_id` — string from the rule YAML's
  `checks[].check_id` per ADR-0022.
- `status` — closed-but-additive enum from
  ADR-0003 §6.
- `result` — closed-but-additive enum from ADR-0004
  (`pass`, `fail`, `error`, `degraded`).
- `trigger_source` — closed-but-additive enum from
  ADR-0002 (`scheduler`, `manual`, `operator-rerun`).
- `mode` — closed enum from ADR-0021 (`set`,
  `record`).
- `error_class` — closed-but-additive enum from
  ADR-0007.

**Cardinality posture.** The label combinations are
bounded by the entity × check_id cross-product. For a
platform onboarded with N entities each with M checks,
the maximum series cardinality per metric is bounded by
N × M × (label-value cardinality of the remaining
labels). At Phase-6 onboarding scale (≤10 entities, ≤5
checks per entity), per-metric cardinality stays under
a few hundred series — well below substrate-collector
stress points. The contract does not commit a numeric
cardinality ceiling; the substrate is responsible for
its own ingest limits per ADR-0010.

### Evolution rules — unified

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
   ADR-0035's N-1 + 90-day calendar-time floor —
   consumers have at least one major version OR 90
   calendar days (whichever is longer) to migrate.

3. **The closed-but-additive enums** (`status`,
   `result`, `trigger_source`, `error_class`) extend
   additively within a major; consumers MUST handle
   "unknown enum value" gracefully (e.g., a default
   path or a one-line log) because the platform may
   introduce a new value at any minor-version bump.

4. **`evidence_summary` and `sample_violating_rows`
   are consumer-facing payloads with structured
   shapes.** Adding a field to either is additive;
   removing a *documented* field is breaking.
   Undocumented fields may appear and disappear at
   engine-minor cadence — consumers should not rely on
   undocumented fields. The currently-documented field
   inventory is the union of what ADR-0003 §7 commits
   ("aggregate counts (rows scanned, rows failing,
   etc.)" for `evidence_summary`; "repeated record" for
   `sample_violating_rows`); a successor ADR commits a
   more precise field-level inventory when the first
   consumer slice surfaces concrete dependencies.

### Baseline-dashboard deferral

The example consumer dashboard from foundation 05 §
"Dashboards" is **out of scope for this ADR**.
Registering one B2 row to track the slice:

- **B2-NEW: Baseline dashboard consumer slice.**
  Implementation of a single-page example dashboard
  (run success rate, check pass/fail rate per entity,
  cost per entity per day, alerting volume per owner,
  scheduler health) that reads the contract this ADR
  commits. Pacing: post-Phase-4c metric emission.
  Substrate choice + workspace placement (likely
  `deploy/dashboards/` or a new `dashboards/` workspace
  per ADR-0019's tooling posture) commits with the
  slice's ADR.

### Why this does not reopen ADR-0003 / ADR-0004 / ADR-0007 / ADR-0026

Each prior ADR commits a piece of the underlying
surface this ADR commits the *consumer contract* over:

- **ADR-0003** commits the table columns and the
  `status` enum. This ADR adds the consumer-facing
  stability tier (which columns are guaranteed stable
  across minor versions). ADR-0003's commitments are
  preserved.
- **ADR-0004 / ADR-0026** commit the aggregation
  rules. This ADR cites them as the canonical
  aggregation reference; it does not re-derive them.
- **ADR-0007** commits the log + metric + span
  emission obligation. This ADR commits the *metric
  shape* the emission produces. ADR-0007's commitment
  to emit-on-every-failure-path is preserved.

### Why this does not reopen ADR-0001 / ADR-0035

ADR-0001 commits the rule-schema compatibility model;
ADR-0035 commits the schema-version-retirement window.
This ADR borrows the same posture (additive within
major; breaking requires major bump; N-1 + 90-day
window) for the dashboard contract surfaces, without
re-deriving the posture. The dashboard surfaces are
engine-version-bound (not rule-schema-version-bound),
so the version axis is "engine semver" — but the
discipline is identical.

---

## Consequences

1. **A formal consumer contract is committed for both
   reporting surfaces.** SQL consumers can write
   `SELECT ... FROM dq_executions_current WHERE
   status IN (...)` against a stable column list and a
   committed status enum. Metric consumers can scrape
   the `/metrics` endpoint and write panels against a
   stable metric name + label inventory.

2. **The metric inventory is committed at contract
   level even though emission code is deferred.** Per
   `engine/internal/runner/runner.go` lines 15-27, the
   current engine emits log signals only; metric
   emission is queued for a Phase-4c follow-up. When
   emission lands, it lands against this ADR's
   inventory. Implementation lands additively; no
   contract redesign happens at emission time.

3. **The baseline dashboard implementation is
   explicitly deferred to a B2 consumer slice.** The
   demonstrative dashboard from foundation 05 §
   "Dashboards" is a consumer of this ADR's contract,
   not part of the contract itself. The slice's ADR
   will commit the substrate choice and the workspace
   placement; this ADR commits only what the slice
   will consume.

4. **Evolution rules align with the existing
   compatibility framework.** Additive within an
   engine-major-version is allowed; breaking changes
   require an engine-major bump and the ADR-0035
   migration-window posture. Consumers writing against
   a documented major-version of the contract can
   reason about their compatibility window the same
   way rule authors do for rule schemas.

5. **The `dq_` prefix is part of the contract.** The
   tables and metrics share the prefix so consumers in
   a shared namespace can isolate platform signals
   without collision. Renaming the prefix is a breaking
   change.

6. **The closed-but-additive enums become part of the
   consumer surface.** Consumers MUST handle "unknown
   enum value" gracefully — additive extensions to
   `status`, `result`, `trigger_source`, `error_class`
   are not breaking changes and may appear within a
   minor-version bump.

7. **No `dq_entity_rollup` pre-aggregated table is
   committed.** Consumers compute their own per-entity
   rollups against `dq_executions_current` + a time
   window on `recorded_at`. A future ADR may commit a
   pre-aggregated rollup table when concrete consumer
   demand surfaces; reserved as out-of-scope below.

8. **The `evidence_summary` and `sample_violating_rows`
   structured columns are consumer-facing payloads.**
   Field-level additions are additive; removal of a
   *documented* field is breaking. Undocumented fields
   may appear and disappear at engine-minor cadence —
   consumers should not rely on undocumented fields.

9. **B2-6 closes.** The decision-log B2-6 row moves to
   `resolved-adr`. One new B2 row registers the
   baseline-dashboard consumer slice.

10. **ADR-0003, ADR-0004, ADR-0007, ADR-0026, ADR-0001,
    ADR-0035 are preserved.** This ADR layers a
    consumer contract on top of their commitments
    without amending them.

---

## Open Questions

None blocking.

Three deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: Pre-aggregated `dq_entity_rollup` table.**
  Consumers wanting per-entity rollups compute them
  themselves today. A future ADR may commit a
  pre-aggregated rollup table (`dq_entity_rollup_daily`,
  partitioned by date) when concrete consumer demand
  surfaces. Reserved until concrete signal.

- **OQ-2: Cardinality ceiling for metric labels.** The
  contract does not commit a numeric cardinality
  ceiling; the substrate (Prometheus, OTel collector)
  is responsible for its own ingest limits per
  ADR-0010. If cardinality growth produces ingest
  failures in production, a future ADR commits a
  per-metric ceiling. Reserved until concrete
  cardinality-pressure signal surfaces.

- **OQ-3: Documented field-level inventory for
  `evidence_summary` and `sample_violating_rows`.**
  This ADR commits that field-level additions are
  additive and removals of *documented* fields are
  breaking. The current ADR does not enumerate the
  documented fields beyond what ADR-0003 §7 names; a
  future ADR (likely the same slice that lands the
  baseline dashboard) commits the documented field
  inventory because the dashboard is the first consumer
  that reads these payloads programmatically. Reserved
  until the consumer slice lands.

---

## Promotion target

`docs/adr/0039-dashboard-contract.md` — next free ADR
number. Ships the table contract, the metric contract,
the evolution rules, and the baseline-dashboard
deferral with the registered consumer-slice B2 row.
