<!-- path: studies/foundation/05-operational-discipline.md -->

# 05 — Operational Discipline

## Metadata

- Purpose: define how the platform behaves under load, under failure,
  and over time. Covers cost discipline, execution identity,
  idempotency, failure scoping, retry semantics, observability, and
  evidence retention.
- Audience: platform engineers, SRE, incident responders, future
  maintainers.
- Status: draft (many topics are tracked as B0/B1 in the decision log;
  this document is the consolidating frame).
- Last updated: 2026-05-20
- Promotion target: `docs/operations/discipline.md`,
  `docs/operations/runbooks.md`, and several ADRs during Wave 3.

---

## Operating Posture

The platform's operational behavior is governed by four imperatives,
in priority order:

1. **Determinism over convenience.** Behavior must be reproducible
   from inputs (rule version, window, source state).
2. **Cost containment over throughput.** BigQuery has no implicit
   spending limit; the platform must impose explicit ones.
3. **Visible failure over silent degradation.** When something is
   wrong, it is loudly wrong. Silent partial success is the most
   dangerous failure mode.
4. **Evidence retention proportional to actionability.** Keep enough
   to triage; do not keep so much that storage or privacy becomes a
   second problem.

Every operational decision below is a consequence of one or more of
these imperatives.

---

## Cost Discipline

BigQuery cost is a design concern, not an operations afterthought. The
platform imposes guardrails at three levels.

### Schema-level guardrails

Built into the DSL itself:

- **`partition_field` is mandatory** on every entity source. The
  field must be a true BigQuery partition column, not a regular
  column. Validated by the linter.
- **`primary_key` is mandatory** on every entity source so that
  failed samples are uniquely identifiable without scanning extra
  columns.
- **`window.duration` is mandatory** on every scheduled rule. There
  is no implicit unbounded window.

### Compiler-level guardrails

Every compiled BigQuery query:

- includes a `WHERE` clause filtering on `partition_field` against
  the active execution window;
- uses parameterized queries (no string interpolation of values);
- has a documented worst-case bytes-scanned estimate;
- is subject to a dry-run check at compile time that confirms the
  partition filter is present and well-formed.

### Runtime guardrails

The engine enforces:

- **Concurrency budgets per environment.** Maximum number of
  concurrent BigQuery queries, with separate budgets for `prod`,
  `qa`, and `local`.
- **Maximum execution window per check.** The default is 24 hours;
  individual checks may declare smaller windows but cannot exceed
  the environment-level maximum without explicit override.
- **Per-run bytes-scanned ceiling.** If the cumulative bytes scanned
  for a run exceeds the configured ceiling, the run aborts with a
  clear error before causing further damage.
- **Dry-run mode in CI.** Every rule change in `rules/` triggers a
  dry-run compilation in CI; the resulting estimated bytes-scanned
  is posted as a merge-request comment.

The exact values for the ceilings and budgets are environment-specific
and resolved as B1 decisions.

---

## Run Identity and Idempotency

This is the single most important operational concept in the
platform. It is referenced as B0 in the decision log; the framing
below is what the resolution must address.

### What identifies a run

A run is uniquely identified by a tuple:

```
execution_id = hash(
    ruleset_version,
    engine_version,
    entity,
    window_start,
    window_end,
    trigger_source
)
```

The exact hash inputs and format are a B0 decision. The principle is
that the same logical execution (same rules, same data window, same
trigger) produces the same `execution_id`.

### What happens on rerun

When a request arrives that would produce the same `execution_id` as
a recent run, the platform has three possible behaviors. The B0
decision picks one:

- **Append always:** every triggered execution produces a new row,
  even if logically identical. Simplest, but duplicates inflate
  dashboards and confuse alerting.
- **Upsert:** the most recent run wins; older identical executions
  are replaced. Cleanest dashboards, but loses history of operator
  reruns.
- **Hybrid:** appends with a `superseded_by` link from old to new.
  Most expressive, most complex.

The chosen approach must be consistent across the reporting schema,
the alerting deduplication, and the operator-facing UI.

### What "idempotent" means here

A trigger is idempotent if:

- repeating the same trigger within a defined window does not produce
  duplicate alerts;
- the reporting state after N identical triggers equals the state
  after 1;
- downstream consumers can rely on `execution_id` as a stable key.

Achieving this requires coordination between the trigger handler, the
reporter, and the alerting layer — all keying off the same
`execution_id`.

---

## Failure Scope

When something goes wrong during a run, the platform must answer
clearly: **at what level did it go wrong, and what is the impact?**
This is B0 in the decision log.

The failure scope hierarchy, from narrowest to broadest:

### Check failure (the rule found a problem)

A check evaluated successfully and detected that the data does not
meet expectations. This is the **happy path of negative results** —
the platform doing its job.

- Status: `failed` (or `degraded` for warnings).
- Reporting: a row in `dq_check_results` with details.
- Alerting: emit an event; routing depends on owner and severity.
- Other checks in the same run: continue independently.

### Check error (the check could not be evaluated)

A check could not run because of an operational problem — query
syntax error, missing source table, BigQuery quota exceeded.

- Status: `error`.
- Reporting: a row with the error details.
- Alerting: emit an event tagged as operational, not as data
  quality.
- Other checks in the same run: the B0 decision must specify
  whether they continue, abort, or are conditional on entity-level
  policy.

### Entity error (one entity is broken)

All checks for an entity fail to evaluate due to entity-level issues
(source table missing, manifest corruption, schedule misconfiguration).

- Status: `error`.
- Reporting: an execution row marked errored, with no per-check rows.
- Alerting: emit a high-severity operational alert to the platform
  team plus the entity owner.
- Other entities in the same run: continue independently.

### Run error (the whole run is broken)

The engine could not load the manifest, could not connect to
BigQuery, or hit a global resource limit.

- Status: `aborted`.
- Reporting: an execution row marked aborted.
- Alerting: high-severity alert to the platform team.
- The scheduler does not automatically retry an aborted run unless
  explicitly configured.

The exact boundary between "check error" and "entity error" — that
is, whether one bad check fails the whole entity — is the B0
question. The framing above describes the surface; the resolution
picks the policy.

---

## Retry Semantics

The platform deliberately distinguishes between **scheduler-driven
retries** and **operator-driven reruns**. They behave differently.

### Scheduler-driven retries

When the scheduler triggers a run and the trigger handler is
unreachable, the scheduler may retry the trigger. This retry:

- happens at the trigger handler level, not at the check level;
- produces the **same** `execution_id` as the original trigger
  (because all inputs are identical);
- is idempotent by design — see Run Identity above;
- has a configured maximum (e.g. three attempts) before the trigger
  is marked as failed and a high-severity alert is emitted.

### Operator-driven reruns

When an operator explicitly reruns an execution (because a transient
issue was suspected, or because they want fresh evaluation), this
rerun:

- is triggered via the Admin API with an explicit "rerun" flag;
- carries the original `execution_id` in metadata but produces a new
  one;
- is auditable: who triggered, when, with what reason note.

### Check-level retries

A check that errors transiently (BigQuery quota, network blip) may
be retried by the runner up to a small bound (e.g. two retries with
exponential backoff). After that:

- the check is marked `error`;
- the run continues with other checks (per failure-scope policy);
- the underlying error is logged with enough detail for triage.

The platform does **not** auto-retry checks that fail because the
data does not meet expectations. That is the correct outcome of a
check; retrying it would mask the signal.

---

## Observability Contract

Observability is part of the platform contract, not a separate
concern. Every component emits:

### Structured logs

- Format: JSON with `slog`-style key-value pairs.
- Levels: per-package overrides via `DQ_LOG_LEVELS` (see PAT-5
  in [`04-system-architecture.md`](./04-system-architecture.md)).
- Required fields on every log line: `time`, `level`, `component`,
  `execution_id` (when applicable), `entity` (when applicable),
  `check_id` (when applicable).

### Metrics

Exposed via a Prometheus-compatible endpoint. Standard metrics
include:

- run-level counters: triggered, completed, failed, aborted;
- check-level counters: evaluated, passed, failed, error;
- duration histograms: per-check execution time, per-run total time;
- cost gauges: bytes scanned per check, per entity, per run;
- queue depth: scheduled runs waiting for execution;
- scheduler state: triggers managed, triggers in error.

### Traces

Distributed traces via OpenTelemetry. Each run is a root span;
sub-spans cover load, plan, compile, execute, report, alert. Traces
are sampled by configurable rate and on every error.

### Dashboards

A baseline dashboard ships with the platform (during Wave 3). It
shows:

- run success rate over time;
- check pass/fail rate per entity;
- cost per entity per day;
- alerting volume per owner;
- scheduler health.

Downstream teams may build their own dashboards on top of the
reporting tables. The reporting schema is a documented contract; see
the boundary contract for how dashboards are versioned.

---

## Evidence Retention

When a check fails, the platform captures **sample violating rows**
to help triage. This evidence is valuable but expensive and
sensitive.

### What is captured

For each failed check, up to a configured number of violating rows
(default: 100). Each sample row contains:

- the primary key (always);
- the columns relevant to the check (depends on check type);
- the partition value (for context).

Columns not relevant to the check are **not** captured. This is
defense against accidentally exfiltrating PII or other sensitive data
through the DQ reporting tables.

### Where it lives

Failed samples are stored in `dq_check_results` as a structured field
(typically `repeated RECORD` in BigQuery). They are partitioned by
`execution_date` for cost-controlled queries and time-based purging.

### How long it is kept

- Default retention: 90 days for failed samples.
- The reporting tables themselves (execution rows, check rows
  without samples) are retained longer — exact value to be decided.
- After the retention window, samples are purged but the check
  results remain (with `samples_purged: true`).

Exact values are environment-specific and resolved as B1 decisions.

### Privacy posture

The platform treats failed samples as **potentially sensitive** by
default. Access controls on the reporting tables are tighter than
access controls on aggregated dashboards. The exact access matrix is
a deployment concern (see `deploy/`).

---

## Local Development Posture

Operational discipline must hold locally, not only in production.
Otherwise developers ship behaviors that work in their laptop and
break in the cluster.

The `docker-compose.yml` at the repository root provides a local
environment that emulates as much of the cloud as possible. The exact
emulator stack is a Wave 2 decision, but the principle is:

- **Object storage:** fully emulated locally.
- **Pub/Sub:** fully emulated locally.
- **BigQuery:** requires a sandbox cloud project; no faithful local
  emulator exists. Local development uses synthetic small datasets
  in the sandbox.
- **Scheduler:** triggered manually via the Admin API locally; no
  emulator needed.
- **Observability:** local Prometheus + Grafana + Jaeger via
  docker-compose, mirroring the production observability shape.

Every developer can run the full engine against the local
docker-compose plus a sandbox BigQuery project. CI runs the same
stack.

---

## Open Topics

Tracked in [`06-decision-log.md`](./06-decision-log.md):

- exact `execution_id` derivation algorithm (B0);
- exact write semantics: append vs upsert vs hybrid (B0);
- exact failure-scope policy between check error and entity status
  (B0);
- exact retry counts and backoff schedule (B1);
- exact concurrency budgets per environment (B1);
- exact bytes-scanned ceilings per run (B1);
- exact failed-sample retention windows per environment (B1);
- exact local-emulator stack (Wave 2).

The framing in this document is stable; the parameters are not.

---

## Closing Position

Operational discipline is what separates a platform that works in
demos from a platform that works at 3 AM during an incident. Every
imperative in this document exists because skipping it has a known
failure mode:

- Without determinism, incident analysis becomes guesswork.
- Without cost discipline, BigQuery bills become unbounded.
- Without visible failure, silent data quality drift accumulates.
- Without proportional retention, storage and privacy become
  secondary problems.

The decisions tracked in `06-decision-log.md` will refine the
parameters. The discipline itself is not negotiable.
