<!-- path: docs/adr/0029-bigquery-cost-ceilings.md -->

# ADR-0029 — BigQuery Cost Ceilings (Set-Mode)

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

Foundation 05 §"Cost Discipline" commits that BigQuery cost is a
first-class design concern (P4) and identifies three guardrail
layers — schema, compiler, runtime. Schema-level (`partition_column`
on the source descriptor; `window_start` / `window_end` on every
trigger) is locked by [ADR-0023](./0023-sources-schema.md) and
[ADR-0014](./0014-trigger-handler-contract.md). Compiler-level
parameterization is locked by P2 determinism and
[ADR-0004](./0004-failure-scope.md)'s always-continue contract. The
remaining surface — concrete per-environment cost ceilings, the
mechanisms that enforce them, and the loader-refresh escalation
parameters — was deferred to this row.

Set-mode (BigQuery) cost is mode-bound per
[ADR-0021](./0021-mode-as-primitive.md);
[ADR-0027](./0027-record-mode-cost-guardrails.md) already shipped
the record-mode (Kafka) cost ceilings via
`EnvConfig.RecordModeCost`. This ADR commits the parallel set-mode
surface under the same per-env-Go-constants pattern committed by
[ADR-0018](./0018-environment-configuration-model.md) (PAT-4) so
the cost-guardrail surface is uniform across modes.

Two downstream operational runbooks
(`docs/runbooks/refresh-failure-escalation.md`,
`docs/runbooks/orphan-run-remediation.md`) carried TBD markers for
parameters this ADR resolves:

- the consecutive-failure threshold for severity escalation
  deferred by [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md)
  §2;
- the post-recovery verification-gate duration the
  refresh-failure runbook §4 currently calls the "alert-window";
- the orphan-detector threshold-age justification (the values
  themselves already live in `engine/internal/env/{local,qa,prod}.go`
  as `OrphanThreshold`; this ADR provides the rationale that
  closes the runbook TBD).

The principles bearing on the decision are **P4** (Cost as a
first-class constraint; this ADR is the P4 commitment for set-mode),
**P5** (Evolution must be contract-driven — ceilings ship under a
documented per-env contract, not buried in code), and **P2**
(determinism — auto-tuning ceilings are rejected because the same
rule on the same data window must produce the same cost ceiling).

---

## Decision

### `EnvConfig.SetModeCost` sub-struct

A new typed sub-struct ships on `EnvConfig` per
[ADR-0018](./0018-environment-configuration-model.md) PAT-4 and is
populated in each per-env file
(`engine/internal/env/{local,qa,prod}.go`). The reflect-based
exhaustiveness test committed by ADR-0018 MD-4 catches any field
omitted from any per-env declaration at CI time.

```
// Annotations: "runtime" = enforced by engine code; "runbook"
// = read by operators, not enforced by engine code.
type SetModeCost struct {
    MaxBytesScannedPerRun           int64         // runtime
    MaxWindowDuration               time.Duration // runtime
    MaxConcurrentEvaluations        int           // runtime
    MaxEvidenceSampleSize           int           // runtime
    RefreshFailureEscalationN       int           // runtime
    RefreshFailureRecoveryWindow    time.Duration // runbook
}
```

#### Field semantics

- **`MaxBytesScannedPerRun`** — bytes-scanned ceiling per single
  execution. The evaluator (`engine/internal/eval`) owns SQL
  construction; the dry-run check lives there. If
  `QueryConfig.DryRun = true` returns an estimate exceeding the
  ceiling, the handler returns `ResultError` with
  `EvidenceSummary["reason"] = "cost_ceiling_exceeded"`. The runner
  reads the reason and writes the terminal row with
  `status = aborted` via the short-circuit committed below.

- **`MaxWindowDuration`** — upper bound on
  `trigger.WindowEnd - trigger.WindowStart`. The HTTP trigger
  handler ([ADR-0014](./0014-trigger-handler-contract.md)) rejects
  triggers exceeding this ceiling before dispatch — a 90-day
  operator-rerun is a different operation than a 24-hour daily-batch
  check and gets an explicit ceiling. v1 set-mode rule artefacts do
  not carry a window field on the rule
  ([ADR-0023](./0023-sources-schema.md)); windows arrive on the
  trigger. The trigger handler is therefore the sole gate. A future
  schema amendment may add an optional per-rule
  `max_window_duration` declaration that the linter would read; that
  amendment is out of scope here.

- **`MaxConcurrentEvaluations`** — upper bound on the count of
  `Runner.Run` invocations in flight at any moment. Enforced by a
  semaphore the HTTP trigger handler acquires before dispatching;
  exhaustion surfaces as `HTTP 503` with a `Retry-After` header,
  matching the set-runner backpressure pattern committed by
  [ADR-0027](./0027-record-mode-cost-guardrails.md).

- **`MaxEvidenceSampleSize`** — per-check upper bound on
  `sample_violating_rows` retained on `dq_check_results`. The
  evaluator truncates before writing. The cap is independent from
  ADR-0027's record-mode `MaxEvidenceSampleSize` (each mode bounds
  its own evidence shape per
  [ADR-0021](./0021-mode-as-primitive.md)).

- **`RefreshFailureEscalationN`** — resolves the **N** parameter
  [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md) §2
  deferred to this row. Every refresh failure continues to emit a
  basic operational alert (mechanism unchanged). After **N
  consecutive failures** the same alert is **promoted to high
  severity** and the loader emits a separate "manifest refresh
  persistently failing" tag alongside the next basic emission; the
  counter resets on the first successful refresh.

- **`RefreshFailureRecoveryWindow`** — duration of consecutive
  successful refreshes after a failing period that the operational
  runbook uses as the verification gate to declare the failure mode
  resolved. The field is a `time.Duration` to match the env
  package's existing duration conventions
  (`LoaderRefreshInterval`, `OrphanThreshold`, etc.). No engine
  code path enforces it; operators read it from `EnvConfig` when
  closing a refresh-failure incident. The refresh-failure runbook
  §4 is rewritten to read the duration directly (see Consequences).

#### Per-environment values

| Field | local | qa | prod |
|---|---|---|---|
| `MaxBytesScannedPerRun` | 1 GB | 100 GB | 1 TB |
| `MaxWindowDuration` | 7 days | 30 days | 90 days |
| `MaxConcurrentEvaluations` | 4 | 16 | 64 |
| `MaxEvidenceSampleSize` | 5 | 50 | 100 |
| `RefreshFailureEscalationN` | 2 | 3 | 5 |
| `RefreshFailureRecoveryWindow` | 5 m | 30 m | 1 h |

**Per-value rationale.**

- **`MaxBytesScannedPerRun`** — local 1 GB matches a clean
  small-table count without expensive partition scans; qa 100 GB
  matches an integration-window aggregate against sample data;
  prod 1 TB is the per-execution ceiling under which a single
  misconfigured rule cannot exhaust a typical daily BigQuery cost
  budget. The 1000× spread mirrors the set-mode storage-cost ratio
  between dev fixtures and production assets.
- **`MaxWindowDuration`** — local 7 days favors fast iteration on a
  fresh fixture window; qa 30 days matches a monthly integration
  cycle; prod 90 days bounds an operator-rerun to a quarter, which
  is the longest forensic re-evaluation the runbooks anticipate
  without explicit ceiling override.
- **`MaxConcurrentEvaluations`** — local 4 is comfortable
  single-developer iteration on a laptop runner; qa 16 lets the
  integration lane drive a small entity-fanout test without
  queueing; prod 64 is sized against the HTTP trigger handler's
  natural concurrency under typical worker-pool sizing.
- **`MaxEvidenceSampleSize`** — local 5 is enough to debug a
  failing rule; qa 50 surfaces a representative violation
  distribution to integration reviewers; prod 100 is the evidence
  cap that orphan-debug forensics need without unbounded
  `dq_check_results` row growth.
- **`RefreshFailureEscalationN`** — local 2 escalates quickly for
  fast dev feedback; qa 3 absorbs a transient blip without paging;
  prod 5 absorbs the longest expected object-store
  read-availability hiccup before promoting to high severity.
- **`RefreshFailureRecoveryWindow`** — aligned with `OrphanThreshold`
  (local 5 min, qa/prod 1 h); the recovery verification gate and
  the orphan-detection cutoff describe the same operational tempo
  at v1.

All values are first-draft defaults. The operational session that
provisions real GCP projects tunes them against observed workload
via the same code-review surface as the rest of `EnvConfig`. The
ratios listed are conservative starting points, not calibrated
targets.

### Enforcement points

The three guardrail layers from foundation 05 §"Cost Discipline"
map to the following ceilings at v1:

- **Schema layer.** v1 set-mode rule artefacts do not carry a
  window or bytes-scanned declaration. The schema-layer guardrail
  surface foundation 05 anticipated is reserved for a future
  amendment that introduces an optional per-rule
  `max_window_duration` field; this ADR does not commit that
  amendment. The v1 schema layer is satisfied implicitly by the
  v2 schema's mandatory `Source` field per ADR-0023 — no source ⇒
  no evaluation ⇒ no cost.
- **Compiler layer.** A future `tools/dryrun` binary issues a
  BigQuery dry-run against each rule's query template and rejects
  rules whose worst-case bytes-scanned estimate exceeds
  `MaxBytesScannedPerRun`. CI runs this on every PR touching
  `rules/`. The binary does not yet exist; this ADR commits the
  posture and registers the B2 follow-up that builds it.
- **Runtime layer.** The HTTP trigger handler acquires a per-env
  semaphore sized to `MaxConcurrentEvaluations` before dispatching;
  it also rejects triggers whose `WindowEnd - WindowStart` exceeds
  `MaxWindowDuration`. The evaluator issues a BigQuery dry-run
  before each real query and, when the estimate exceeds
  `MaxBytesScannedPerRun`, returns `ResultError` with reason
  `cost_ceiling_exceeded`; the runner short-circuits to
  `status = aborted` (see below). The loader-refresh loop continues
  to emit a basic operational alert on every refresh failure per
  ADR-0007 §2 and additionally emits the high-severity tag after
  `RefreshFailureEscalationN` consecutive failures.

### Runner short-circuit for `status = aborted`

The runner's standard status mapping (ADR-0004 CC2) routes any
`ResultError` to `StatusError`. This ADR extends the mapping with
a single short-circuit: when any per-check evaluation surfaces
`EvidenceSummary["reason"] == "cost_ceiling_exceeded"`, the runner
writes the terminal row with `status = aborted` instead of
`status = error`. `aborted` is already one of the five committed
states (ADR-0003 CC6; previously produced by ADR-0007 CC11's orphan
path); this ADR extends its producer set to include the
cost-ceiling-exceeded path. No new status enum value is committed.

### `row_count_positive` cost gap

The inaugural set-mode kind issues `SELECT COUNT(*) FROM <table>`
without a partition filter. BigQuery counts this against metadata
cache when the table is partitioned with metadata caching enabled,
and against the full table otherwise. This ADR commits the
ceiling, not a partition-filter retrofit on `row_count_positive`.
The retrofit is registered as a B2 follow-up: either the existing
kind grows partition-pruning logic keyed on
`Source.PartitionColumn`, or a new partition-aware kind ships
alongside.

### Posture: per-env Go constants, not deploy-time tunables

Cost ceilings ship as per-env Go constants on `EnvConfig.SetModeCost`,
not as deploy-time ConfigMap values. A ceiling change requires a
code commit + engine rebuild + deploy. This matches the posture
ADR-0027 committed for record-mode cost ceilings and preserves
ADR-0018 PAT-4's typed-env-config contract. Cost ceilings are not
a hot-path tuning surface; the operational session that adjusts
them goes through the same review surface as any other
`EnvConfig` change.

Auto-tuning ceilings are explicitly rejected: a runtime that
self-adjusts based on observed load would introduce emergent
behavior the platform avoids per P2 (determinism). The same rule
on the same data window must produce the same cost ceiling.

---

## Consequences

1. **`engine/internal/env/config.go` ships a new
   `SetModeCost` sub-struct on `EnvConfig`**, with values
   populated in `local.go` / `qa.go` / `prod.go` per the table
   above. The ADR-0018 MD-4 reflect-based exhaustiveness test
   catches any per-env file that forgets a field.

2. **The HTTP trigger handler gains a semaphore sized to
   `MaxConcurrentEvaluations`.** Trigger acceptance returns
   `HTTP 503` with `Retry-After` when the semaphore is exhausted.
   The handler also rejects triggers whose
   `WindowEnd - WindowStart` exceeds `MaxWindowDuration` before
   dispatch.

3. **The evaluator gains a dry-run pre-flight before each real
   query.** `QueryConfig.DryRun = true` returns the estimated
   bytes-scanned without execution; exceeding `MaxBytesScannedPerRun`
   surfaces `ResultError` with reason `cost_ceiling_exceeded`.

4. **The runner gains a `status = aborted` short-circuit.** When
   any per-check evaluation surfaces
   `EvidenceSummary["reason"] == "cost_ceiling_exceeded"`, the
   runner writes the terminal row with `status = aborted` instead
   of running the standard `ResultError → StatusError` mapping.
   This extends `aborted`'s producer set (previously only ADR-0007
   CC11's orphan path) to include the cost-ceiling-exceeded path.

5. **The loader-refresh loop gains consecutive-failure counting
   for severity escalation per ADR-0007 §2.** After
   `RefreshFailureEscalationN` consecutive failures the loop emits
   the high-severity "manifest refresh persistently failing" tag
   alongside the next basic emission; the counter resets on the
   first successful refresh.

6. **`docs/runbooks/refresh-failure-escalation.md` §4 verification
   step is rewritten** to phrase the recovery gate as "wait
   `RefreshFailureRecoveryWindow` of clean refreshes" instead of
   the prior "the next N ticks" wording, eliminating the tick
   arithmetic operators currently have to perform.

7. **`docs/runbooks/orphan-run-remediation.md` gains a citation
   to this ADR** for the orphan-detector threshold age. The
   existing `OrphanThreshold` values in
   `engine/internal/env/{local,qa,prod}.go` (5 min / 1 h / 1 h)
   align with `RefreshFailureRecoveryWindow` by intent — the
   recovery verification gate and the orphan-detection cutoff
   describe the same operational tempo at v1. No code change is
   needed; the runbook TBD reduces to a citation update.

8. **No `tools/lint` cross-check ships from this ADR.** v1
   set-mode rule artefacts do not carry a window or bytes-scanned
   declaration the linter could read; the schema-layer guardrail
   surface is reserved for a future schema amendment that
   introduces an optional per-rule `max_window_duration` field.
   The lint binary is unchanged by this ADR; the HTTP trigger
   handler is the sole window-ceiling enforcement point.

9. **B2 follow-up: `tools/dryrun` binary.** A new B2 row
   registers the deferred compiler-layer enforcement. The binary
   loads each rule's compiled query template, substitutes window
   placeholders against `MaxWindowDuration`, issues a BigQuery
   dry-run, and reports the estimate per PR. Until the binary
   exists, the runtime-layer evaluator dry-run is the sole
   enforcement point.

10. **B2 follow-up: `row_count_positive` partition-filter
    retrofit.** A separate B2 row registers the cost gap. Either
    the existing kind grows partition-pruning logic keyed on
    `Source.PartitionColumn`, or a new partition-aware kind ships
    alongside; the choice is the follow-up's scope.

11. **Cost ceilings are committed at engine-version pinning
    resolution.** Future updates to the per-env values flow
    through the same code-review surface as other `EnvConfig`
    changes per ADR-0018 PAT-4; the values are not deploy-time
    tunable absent an explicit re-opening of this ADR.

12. **The platform's P4 commitment for set-mode is now
    explicit.** This ADR plus ADR-0027 (record-mode) together
    cover both mode halves of the cost-discipline surface
    foundation 05 anticipated.

13. **Per-rule cost-override surface is deferred.** Some
    set-mode rules legitimately need a higher ceiling than the
    per-env default (e.g., a quarterly aggregate over historical
    data). ADR-0027 ships a record-mode override mechanism via
    `params.aggregation`; the set-mode analog would extend the v2
    rule schema with a `params.cost.max_bytes_scanned_override`
    field. The override surface is reserved for a future
    amendment once operational signal justifies it — v1 ceilings
    are not tested against real workload yet.

---

## Notes

- The 1 GB / 100 GB / 1 TB and 5 min / 30 min / 1 h spreads are
  conservative starting points. The operational session that
  provisions real GCP projects has the authority to tune them via
  PR review against `engine/internal/env/{local,qa,prod}.go`.
- The runner short-circuit treats
  `EvidenceSummary["reason"] == "cost_ceiling_exceeded"` as a
  reserved sentinel. Future handlers MUST NOT reuse the reason
  string for any non-cost-ceiling outcome; a richer signal
  (typed Evaluation field) is a future enrichment if the
  sentinel collision surface grows.
- The `RefreshFailureRecoveryWindow` field is the first
  `EnvConfig.SetModeCost` field marked `// runbook` (read by
  operators, not enforced by engine code). Future fields in this
  sub-struct may add additional runbook parameters; the
  annotation legend in the struct comment documents the
  asymmetry.
