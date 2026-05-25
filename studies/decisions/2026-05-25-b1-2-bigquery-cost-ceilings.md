<!-- path: studies/decisions/2026-05-25-b1-2-bigquery-cost-ceilings.md -->

# B1-2 — BigQuery Cost Ceilings (Set-Mode)

## Context

Foundation 05 §"Cost Discipline" commits that BigQuery cost is a
first-class design concern (P4) and identifies three guardrail
layers:

- **Schema-level** (already locked by ADR-0023 §source.bigquery:
  optional `partition_column` declares partition-pruning intent;
  ADRs 0002 + 0014 commit window_start/window_end on every trigger).
- **Compiler-level** (parameterized queries are committed by P2
  determinism + ADR-0004 always-continue; per-run bytes-scanned
  estimation is open).
- **Runtime** (concurrency budgets, per-run bytes-scanned ceiling,
  dry-run mode in CI — all deferred to this B1 row).

The remaining cost dimensions — concrete per-environment values
plus the mechanism that enforces them — were tagged as a B1
follow-up at foundation 05 §"Cost Discipline" closing paragraph:
"The exact values for the ceilings and budgets are
environment-specific and resolved as B1 decisions."

Multiple downstream artefacts already depend on B1-2's numeric
output. Specifically:

- `docs/runbooks/refresh-failure-escalation.md` defers two
  parameters to B1-2 (both marked TBD): the
  consecutive-failure threshold for severity escalation per
  ADR-0007 §2 (resolved below as
  `RefreshFailureEscalationN`), and the post-recovery
  verification-gate duration the runbook §4 currently calls
  the "alert-window" (resolved below as
  `RefreshFailureRecoveryWindow`).
- `docs/runbooks/orphan-run-remediation.md` defers the orphan-
  detector threshold-age justification to B1-2 (the value is
  set in `engine/internal/env/{local,qa,prod}.go` via
  `OrphanThreshold` but the justification is open).
- `docs/runbooks/README.md` lists "refresh-failure thresholds
  per B1-2" as an open dependency.

Per ADR-0021 the cost discipline is mode-bound: this row
addresses set-mode (BigQuery) cost; ADR-0027 already committed
the record-mode (Kafka) cost guardrails via
`EnvConfig.RecordModeCost`. B1-2's output mirrors the same
shape — per-env values carried in a new `EnvConfig.SetModeCost`
sub-struct populated in `local.go`, `qa.go`, `prod.go`, with the
enforcement points spread across the engine binary, the linter,
and CI per the three guardrail layers above.

The principles bearing on the decision are P4 (Cost as a
first-class constraint; this row IS the P4 commitment for set-
mode), P5 (Evolution must be contract-driven; the ceilings ship
under a documented per-env contract, not buried in code), and
foundation 04 §PAT-4 (Typed environment config — values flow
through the same selector the rest of EnvConfig uses).

---

## Decision Drivers

- **DD-1 — Cost ceilings must be enforced at three layers**
  (schema, compiler, runtime) per foundation 05. A study that
  only commits one layer leaves cost discipline incomplete.
- **DD-2 — Per-environment values must diverge by an order of
  magnitude** so that `local` is fast for development feedback,
  `qa` is realistic for integration, and `prod` is the
  conservative ceiling. Identical values across envs would defeat
  the multi-env isolation that ADR-0018 committed.
- **DD-3 — The mechanism must mirror ADR-0027's
  `RecordModeCost` pattern.** ADR-0027 ships record-mode cost
  ceilings via a typed sub-struct on `EnvConfig` populated in
  the per-env Go files. Diverging from that shape for set-mode
  would split the cost-guardrail surface across two patterns
  for no benefit.
- **DD-4 — The runbook TBDs must resolve.** B1-2 must commit
  both runbook parameters: the consecutive-failure escalation
  threshold (per ADR-0007 §2) and the post-recovery
  verification-gate duration the runbook §4 calls the
  "alert-window". The runbook TBDs are blocked on these
  values.
- **DD-5 — `row_count_positive` cost gap must be acknowledged
  honestly.** The inaugural set-mode kind (ADR-0022,
  `set.row_count_positive`) issues `SELECT COUNT(*) FROM <table>`
  with no partition filter. The bytes-scanned ceiling cannot
  prevent a runaway scan of a partition-less table at v1; the
  study must commit the ceiling AND name the partition-pruning
  follow-up.
- **DD-6 — Dry-run enforcement requires `tools/dryrun`, which
  does not yet exist.** Foundation 05 §"Runtime guardrails"
  commits "dry-run mode in CI" but the Makefile's
  `dry-run-rules` target is a stub. The study commits the
  posture (CI must run dry-run on every `rules/` change) and
  acknowledges the tooling gap.

---

## Considered Options

### Option 1 — Per-env Go constants in `EnvConfig.SetModeCost` (recommended)

Mirror ADR-0027's `RecordModeCost` pattern. Add a typed
`SetModeCost` sub-struct to `engine/internal/env.EnvConfig`
with fields for each cost dimension. Populate per-env values in
`local.go`, `qa.go`, `prod.go`. The reflect-based
exhaustiveness test from B1-4 MD-4 catches any forgotten field.

The engine enforces the runtime-layer ceilings inside the runner
/ HTTP handler; the linter enforces the schema-layer ceilings at
rule-load time; CI enforces the compiler-layer ceilings via a
`make dry-run-rules` target that calls a future
`tools/dryrun` binary.

**Strengths.** Consistent with ADR-0018 PAT-4 + ADR-0027 pattern;
typed; CI-enforced exhaustiveness; values reviewable in code; no
deploy-time misconfiguration risk; per-env divergence committed
in one place.

**Trade-offs.** A ceiling change requires a code commit + engine
rebuild + deploy. Acceptable: cost ceilings are not a hot-path
tuning surface; the operational session that adjusts them goes
through the same review the rest of EnvConfig does.

### Option 2 — Kustomize-driven ConfigMap

Ship ceiling values in `deploy/overlays/{local,qa,prod}/`
ConfigMaps; engine reads them at startup. Cleaner separation
between code and operational tunables.

**Strengths.** Operators can adjust ceilings without rebuilding
the engine; emergency cost-overrun adjustments are deploy-time
rather than code-time.

**Trade-offs.** Departs from ADR-0027's pattern for no clear
gain; introduces a second mechanism for what is conceptually the
same surface; the reflect-based exhaustiveness test from B1-4
MD-4 does not cover ConfigMap-driven fields; misconfiguration at
deploy time produces silent ceiling regressions instead of a
fail-fast at engine startup. The overlay-extension cost is
non-trivial and would set a precedent that erodes the typed-env-
config posture B1-4 committed.

### Option 3 — Runtime auto-tuning

Engine observes actual bytes-scanned + concurrency over a
rolling window and self-adjusts ceilings.

**Strengths.** Adaptive to actual workload; no manual tuning.

**Trade-offs.** Introduces emergent behavior the platform
explicitly avoids per P2 (determinism). Same rule + same window
+ same data should produce the same cost ceiling, not a moving
target that depends on prior load. Reserved as a future
enrichment if static ceilings prove inadequate; out of scope for
this row.

---

## Recommendation

**Option 1.** Per-env Go constants on `EnvConfig.SetModeCost`.

### Cost dimensions

The following dimensions ship as fields on `SetModeCost`:

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

Field definitions:

- **`MaxBytesScannedPerRun`** — bytes-scanned ceiling per single
  execution. The evaluator (`engine/internal/eval`) issues the
  BigQuery dry-run before the real query — `row_count_positive`
  and future set-mode kinds build their SQL inside the eval
  package, so the dry-run check lives there. If the estimate
  exceeds the ceiling, the handler returns `ResultError` with
  reason `cost_ceiling_exceeded`; the runner reads the reason
  and writes the terminal row with `status = aborted` via the
  new short-circuit committed in Consequence 3. The status enum
  is unchanged — `aborted` is already one of the five committed
  states (ADR-0003 CC6); this row only extends its producer
  set.

- **`MaxWindowDuration`** — upper bound on
  `trigger.WindowEnd - trigger.WindowStart`. The HTTP trigger
  handler (ADR-0014) rejects triggers exceeding this ceiling
  before dispatch — a 90-day operator-rerun is conceptually a
  different operation than a 24-hour daily-batch check and
  gets an explicit ceiling. v2 set-mode rules do not carry a
  window field on the rule artefact (windows arrive on the
  trigger per ADR-0014), so the linter has no
  `MaxWindowDuration` surface to enforce — runtime enforcement
  via the trigger handler is the sole gate. A future schema
  amendment could add an optional per-rule
  `max_window_duration` declaration the linter could read; that
  amendment is out of scope here.

- **`MaxConcurrentEvaluations`** — upper bound on the count of
  `Runner.Run` invocations in flight at any moment. Enforced by
  a semaphore the trigger handler acquires before dispatching;
  exhaustion surfaces as HTTP 503 with a `Retry-After` header,
  matching ADR-0027's set-runner backpressure pattern.

- **`MaxEvidenceSampleSize`** — per-check upper bound on
  `sample_violating_rows` retained on `dq_check_results`. The
  evaluator truncates before writing; cap mirrors ADR-0027's
  record-mode `MaxEvidenceSampleSize` but applies independently
  to set-mode kinds.

- **`RefreshFailureEscalationN`** — resolves the **N**
  parameter ADR-0007 §2 explicitly deferred to B1. ADR-0007 §2
  commits that every refresh failure emits a basic operational
  alert; after **N consecutive failures** the same alert is
  **promoted to high severity** and the loader emits a separate
  "manifest refresh persistently failing" tag. This row commits
  the value of N. The initial-emission mechanism is unchanged;
  this row does not gate whether the alert fires, only when it
  escalates.

- **`RefreshFailureRecoveryWindow`** — duration of consecutive
  successful refreshes after a failing period that the
  operational runbook (`docs/runbooks/refresh-failure-escalation.md`
  §4) uses as the verification gate to declare the failure mode
  resolved. The runbook §4 currently phrases this gate in
  "ticks" — Consequence 4 commits that the runbook is rewritten
  to read the duration directly so operators do not consult two
  `EnvConfig` fields to do tick arithmetic. The field is a
  `time.Duration` to match the rest of the env package's
  duration conventions (`LoaderRefreshInterval`,
  `OrphanThreshold`, etc.). No engine code path enforces it;
  operators read it from `EnvConfig` when closing a
  refresh-failure incident.

### Per-environment values

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
  small-table count without expensive partition scans; qa
  100 GB matches an integration-window aggregate against
  sample data; prod 1 TB is the per-execution ceiling under
  which a single misconfigured rule cannot exhaust a typical
  daily BigQuery cost budget. The 1000× spread mirrors the
  set-mode storage-cost ratio between dev fixtures and
  production assets.
- **`MaxWindowDuration`** — local 7 days favors fast iteration
  on a fresh fixture window; qa 30 days matches a monthly
  integration cycle; prod 90 days bounds an operator-rerun
  to a quarter, which is the longest forensic re-evaluation
  the runbooks anticipate without explicit ceiling override.
- **`MaxConcurrentEvaluations`** — local 4 is comfortable
  single-developer iteration on a laptop runner; qa 16 lets
  the integration lane drive a small entity-fanout test
  without queueing; prod 64 matches the trigger-handler's
  natural concurrency under the per-env HTTP server's worker
  pool sizing (W3-P4e ADR-0014 §4 defaults).
- **`MaxEvidenceSampleSize`** — local 5 is enough to debug a
  failing rule; qa 50 surfaces a representative violation
  distribution to integration reviewers; prod 100 is the
  ADR-0003 evidence-cap that orphan-debug forensics need
  without unbounded `dq_check_results` row growth.
- **`RefreshFailureEscalationN`** — local 2 escalates quickly
  for fast dev feedback; qa 3 absorbs a transient blip
  without paging; prod 5 absorbs the longest expected GCS
  read-availability hiccup before promoting to high
  severity.
- **`RefreshFailureRecoveryWindow`** — aligned with
  `OrphanThreshold` per OQ-2 framing (local 5 min, qa/prod
  1 h); the recovery verification gate and the orphan
  cutoff describe the same operational tempo at v1.

**All values are first-draft defaults; the operational
session that provisions real GCP projects tunes them against
observed workload via the same code-review surface as the
rest of EnvConfig per ADR-0018 PAT-4.** The 1000× / 12× /
20× / 16× / per-env spreads above are conservative starting
points, not calibrated targets.

### Enforcement points

The three guardrail layers from foundation 05 §"Cost Discipline"
map to the following ceilings at v1:

- **Schema layer.** v1 set-mode rule artefacts do **not** carry
  a window or bytes-scanned declaration that the linter could
  cross-check against `MaxWindowDuration` or
  `MaxBytesScannedPerRun` — windows arrive on the trigger
  (ADR-0014), and the SQL is constructed inside the evaluator
  from `Source` + trigger window. The schema-layer guardrail
  surface foundation 05 anticipated (per-rule window
  declarations the linter validates) is therefore reserved for
  a future schema amendment that adds an optional per-rule
  `max_window_duration` field; this row does not commit that
  amendment. At v1 the schema layer is satisfied implicitly by
  the existing v2 schema's mandatory `Source` field per
  ADR-0023 (no source ⇒ no evaluation ⇒ no cost).
- **Compiler layer.** A future `tools/dryrun` binary issues a
  BigQuery dry-run against each rule's query template and
  rejects rules whose worst-case bytes-scanned estimate exceeds
  `MaxBytesScannedPerRun`. CI runs this on every PR touching
  `rules/`. The binary does not yet exist; this row commits the
  posture and Consequence 7 registers the B2 follow-up that
  builds it.
- **Runtime layer.** The HTTP trigger handler acquires a
  per-env semaphore sized to `MaxConcurrentEvaluations` before
  dispatching to the runner; the trigger handler also rejects
  triggers whose `WindowEnd - WindowStart` exceeds
  `MaxWindowDuration`. The evaluator issues a BigQuery dry-run
  before each real query and, when the estimate exceeds
  `MaxBytesScannedPerRun`, returns `ResultError` with reason
  `cost_ceiling_exceeded` plus a new short-circuit signal the
  runner reads to write `status = aborted` (Consequence 3
  commits that signal). The loader-refresh loop continues to
  emit a basic operational alert on every refresh failure per
  ADR-0007 §2 and additionally emits the high-severity
  "manifest refresh persistently failing" tag after
  `RefreshFailureEscalationN` consecutive failures. The
  `RefreshFailureRecoveryWindow` value is a runbook parameter
  consumed by operators; no engine code path enforces it.

### `row_count_positive` cost gap (DD-5)

The inaugural set-mode kind issues `SELECT COUNT(*) FROM <table>`
without a partition filter. BigQuery counts this against
metadata cache when the table is partitioned with metadata
caching enabled, and against the full table otherwise.

This row commits the ceiling, not a partition-filter retrofit
on `row_count_positive`. The deferred follow-up: a future kind
(`set.row_count_positive_in_window` or a `partition_column`-
aware enrichment of the existing kind) issues a partition-
filtered query so the ceiling bites at the query-template level,
not just at the cumulative-cost guardrail. The orders_stream
record-mode rule does not have this gap because Kafka cost is
governed by ADR-0027 instead.

### Dry-run tooling gap (DD-6)

`tools/dryrun` does not yet exist. The Makefile's
`dry-run-rules` target prints a stub message. The runtime-layer
dry-run check IS implementable today (the BigQuery client
supports `QueryConfig.DryRun = true`); the CI-layer dry-run
requires a binary that loads each rule's query template,
substitutes window placeholders against the maximum-window-
duration ceiling, and reports the estimate.

This row commits the posture; the binary lands as a B2 follow-
up when set-mode adoption justifies the investment.

---

## Consequences

1. **A new `EnvConfig.SetModeCost` sub-struct ships in
   `engine/internal/env/config.go`**, populated in
   `local.go`/`qa.go`/`prod.go` with the values committed above.
   The reflect-based exhaustiveness test from B1-4 MD-4
   automatically catches any per-env file that forgets a field.

2. **The HTTP trigger handler gains a semaphore** sized to
   `MaxConcurrentEvaluations`. Trigger acceptance returns
   HTTP 503 with `Retry-After` when the semaphore is exhausted;
   the trigger handler's existing strict-decoder pattern
   (ADR-0014) gains a backpressure path. Matches ADR-0027's set-
   runner backpressure mechanism committed there.

3. **The evaluator gains a dry-run pre-flight before each real
   query** (the evaluator owns SQL construction per
   `engine/internal/eval`). `QueryConfig.DryRun = true` returns
   the estimated bytes-scanned without execution; if the
   estimate exceeds `MaxBytesScannedPerRun` the evaluator
   returns `ResultError` with `EvidenceSummary["reason"] =
   "cost_ceiling_exceeded"`. The runner gains a new short-
   circuit: when any per-check evaluation surfaces this exact
   reason, the runner writes the terminal row with
   `status = aborted` instead of running the standard
   ResultError → StatusError mapping per ADR-0004 CC2.
   `aborted` is already one of the five committed states (per
   ADR-0003 CC6 + ADR-0007 CC11 orphan path); this row extends
   its producer set to include the cost-ceiling-exceeded
   path. No new status enum value is committed.

4. **The loader-refresh loop gains consecutive-failure counting
   for severity escalation per ADR-0007 §2 + the runbook is
   rewritten to read the recovery duration directly.** Every
   refresh failure continues to emit a basic operational alert
   (the per-failure emission is unchanged from ADR-0007 §2).
   After `RefreshFailureEscalationN` consecutive failures the
   loop emits the high-severity "manifest refresh persistently
   failing" tag alongside the next basic emission; the counter
   resets on the first successful refresh.
   `docs/runbooks/refresh-failure-escalation.md` §4
   verification step 3 is rewritten to phrase the recovery
   gate as "wait `RefreshFailureRecoveryWindow` of clean
   refreshes" instead of the current "the next N ticks"
   wording, eliminating the tick-arithmetic operators
   currently have to perform. Together these resolve both
   runbook TBDs without changing ADR-0007's committed
   alert-firing posture.

5. **`docs/runbooks/orphan-run-remediation.md` gains a B1-2
   citation** for the orphan-detector threshold age. The
   existing `OrphanThreshold` values in
   `engine/internal/env/{local,qa,prod}.go` (5 min / 1 h / 1 h)
   are justified post-hoc by this row — they were committed
   ahead of B1-2 as engine scaffolding and the values match
   `RefreshFailureRecoveryWindow` by intent (the recovery
   verification gate and the orphan-detection cutoff describe
   the same operational tempo). No code change needed; the
   runbook TBD reduces to a citation update.

6. **No `tools/lint` cross-check ships from this row.** v1
   set-mode rule artefacts do not carry a window or bytes-
   scanned declaration the linter could read; the schema-layer
   guardrail surface is reserved for a future schema amendment
   that introduces an optional per-rule
   `max_window_duration` field. Until that amendment lands,
   the lint binary is unchanged by this row; the trigger
   handler (runtime layer) is the sole window-ceiling
   enforcement point.

7. **B2 follow-up: `tools/dryrun` binary.** A new B2 row
   (close-step assigns the number) registers this row's
   deferred compiler-layer enforcement. The binary loads each
   rule's compiled query template, substitutes window
   placeholders against the max-window-duration ceiling, issues
   a BigQuery dry-run, and reports the estimate per PR. Until
   the binary exists, the runtime-layer evaluator dry-run is
   the sole enforcement point.

8. **B2 follow-up: `row_count_positive` partition-filter
   retrofit.** A separate B2 row (close-step assigns the
   number) registers the cost gap from DD-5. Either the
   existing kind grows partition-pruning logic keyed on
   `Source.PartitionColumn`, or a new partition-aware kind
   ships alongside; the choice is the follow-up's scope.

9. **Cost ceilings are committed at engine-version pinning
   resolution.** Future updates to the per-env values flow
   through the same code-review surface as other EnvConfig
   changes per ADR-0018 PAT-4; the values are not deploy-time
   tunable absent an explicit re-opening of this row.

10. **The platform's P4 commitment for set-mode is now
    explicit.** This row plus ADR-0027 (record-mode) together
    cover both mode halves of the cost-discipline surface
    committed by foundation 05.

---

## Open Questions

None blocking.

One open dimension surfaced during drafting is explicitly
**out-of-scope for current cycle**:

- **OQ-1: Per-rule `bytes_scanned` budget override.** Some
  rules legitimately need a higher ceiling than the per-env
  default (e.g., a quarterly aggregate over historical data).
  ADR-0027 ships a record-mode override mechanism via
  `params.aggregation`; the set-mode analog would extend the v2
  rule schema with `params.cost.max_bytes_scanned_override`.
  Deferred until concrete operational signal justifies the
  override surface — the v1 ceilings are not tested against
  real workload yet.

The alignment of `RefreshFailureRecoveryWindow` with
`OrphanThreshold` (both at 5 min / 1 h / 1 h) is a v1 posture
committed in the per-value rationale above; the two values
describe the same operational tempo at v1. A future amendment
could split them if operational signal shows the coupling is
wrong, but the alignment is not itself an open question — it
is the recommendation.

---

## Promotion target

`docs/adr/0029-bigquery-cost-ceilings.md` — ships the
`EnvConfig.SetModeCost` sub-struct + per-env defaults, the
evaluator dry-run + runner short-circuit for
`status = aborted`, the trigger-handler semaphore + window
ceiling, the loader-refresh consecutive-failure escalation per
ADR-0007 §2, the runbook §4 rewrite from "ticks" to duration,
and the two B2 follow-up registrations (`tools/dryrun` binary +
`row_count_positive` partition-filter retrofit).
