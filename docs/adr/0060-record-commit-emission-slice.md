<!-- path: docs/adr/0060-record-commit-emission-slice.md -->

# ADR-0060 — Record-Commit Emission Slice

- **Status:** accepted
- **Date:** 2026-06-01

---

## Context

[ADR-0058](./0058-record-runner-commit-after-dispatch.md) (promoted
2026-05-30) committed the record-mode runner's commit-after-dispatch
posture — per-trigger commit on `dispatcher.Run` success, warning-log
+ skip + transitive-commit recovery on commit failure.
[ADR-0059](./0059-record-runner-commit-retry.md) (promoted
2026-05-31) layered a bounded retry envelope between
`closeAndDispatch` and `consumer.Commit` (`commitWithRetry` helper;
`max_attempts = 3`; `base = 100ms`; `600ms` worst-case stall; ADR-0058
§Clause 2 terminal behavior preserved verbatim after retry
exhaustion). ADR-0059 §Clause 5 additionally distinguished
context-cancellation (operator-driven shutdown) from broker-failure
exhaustion — context-cancellation returns from `commitWithRetry`
*without* warning-logging because shutdown is not a failure mode.

Three deferred observability questions surfaced in those ADRs'
§Notes and were registered in the OQ Register at PR #127 on
2026-05-31:

- **ADR-0058 OQ-4** — `dq_record_commit_failures_total` counter
- **ADR-0059 OQ-3** — `dq_record_commit_retries_total` counter
- **ADR-0059 OQ-6** — commit-RPC duration histogram for stall-budget
  calibration

The OQ Register's §"Adjacency cluster note" flagged the triple
explicitly as a co-discoverable B3-N seed: three OQs, three
commit-boundary semantics, one emission package — splitting them
into three separate B3-N entries would split a cohesive slice
across three sessions without reducing review load and would risk
inconsistent label sets across the three series.

This ADR is the **promotion of B3-8** under the post-Wave-3
evolutionary lane committed by
[ADR-0049](./0049-b3-evolutionary-launch.md). The originating study
(`studies/decisions/2026-05-31-b3-8-record-commit-emission-slice.md`)
cleared the ADR-0049 §(a) eligibility filter on all four conditions
with **no D0 borderline**: Condition 1 via
[ADR-0039](./0039-dashboard-contract.md) §"Evolution rules" rule 1
("a new metric" is additive within an engine-major) plus
[ADR-0055](./0055-metric-emission-slice-scope.md) §Consequence 6
naming `engine/internal/metrics` as the lane for any future ADR-0039
inventory addition; Condition 2 via direct precedent reuse of
ADR-0055's operator-ratified Tooling-extensions reading (no new
expansive reading proposed; same disposition mechanism as B3-3 /
ADR-0053); Condition 3 via the constraint envelope preserved
verbatim across [ADR-0020](./0020-wave-s-launch.md),
[ADR-0021](./0021-mode-as-primitive.md),
[ADR-0022](./0022-kind-catalog.md),
[ADR-0023](./0023-sources-schema.md), ADR-0039, ADR-0055, ADR-0058,
and ADR-0059; Condition 4 via material decisions (histogram bucket
boundaries, label-set cardinality posture, emission-site placement
against the `commitWithRetry` loop boundary, shutdown-exemption
discipline on the failures counter).

This ADR also lands its implementation slice in the same PR per an
**operator-authorized R4 scope collapse**, precedent
[ADR-0054](./0054-engine-image-registry-amendment.md) §Notes,
[ADR-0055](./0055-metric-emission-slice-scope.md) §Notes,
[ADR-0056](./0056-panel-5-lighting-slice.md) §Notes,
[ADR-0058](./0058-record-runner-commit-after-dispatch.md) §Notes,
and [ADR-0059](./0059-record-runner-commit-retry.md) §Notes.

The principles bearing on the decision are **P3** (ownership is
explicit — every new series carries `entity` as its load-bearing
breakdown label, mapping back to an owner per
[ADR-0006](./0006-alert-routing-contract.md)), **P4** (cost is
first-class — cardinality contribution is bounded and decomposed
explicitly below; no per-attempt / per-partition / per-broker label
is introduced), and **P5** (evolution is contract-driven — the
slice extends ADR-0039 along its own §"Evolution rules" rule 1 lane
and consumes ADR-0055's per-package emitter convention without
reshape).

---

## Decision

The slice is committed in seven clauses (metric inventory, emission
sites, labels and cardinality posture, histogram bucket boundaries,
shutdown exemption on the failures counter, test discipline,
skill-side update), plus a Notes block that records the R4
scope-collapse rationale, the clean-eligibility note, and the
OQ Register posture (a) ratified after the originating study's
round-1 critique.

### Clause 1 — Metric inventory: three new series on `RunnerMetrics`

Three new series land as new fields on `metrics.RunnerMetrics` per
[ADR-0055](./0055-metric-emission-slice-scope.md) §Clause 3's
per-package emitter convention:

| Metric name | Type | Labels |
|---|---|---|
| `dq_record_commit_failures_total` | counter | `entity` |
| `dq_record_commit_retries_total` | counter | `entity`, `outcome` |
| `dq_record_commit_duration_seconds` | histogram | `entity` |

The `outcome` label on the retries counter takes two values:
`success_after_retry` and `exhausted`. No first-attempt-success
increment is emitted (first-attempt success is the no-op-retry
path; the counter only fires when at least one retry was consumed).

### Clause 2 — Emission sites

Each series emits from a single site to keep observability code
co-located with the control flow it observes:

- **`dq_record_commit_failures_total`** — incremented in
  `closeAndDispatch` post-`commitWithRetry` non-nil return,
  alongside the existing warning-log line. **Excludes**
  `context.Canceled` / `context.DeadlineExceeded` returns (see
  Clause 5).
- **`dq_record_commit_retries_total`** — incremented inside the
  `commitWithRetry` helper at the two terminal branches that
  consumed at least one retry: the success-after-retry branch
  (`err == nil` return where `attempt > 1`) and the exhausted
  branch (`attempt == recordCommitMaxAttempts && err != nil`).
- **`dq_record_commit_duration_seconds`** — observed around each
  individual `consumer.Commit(ctx, records)` call inside
  `commitWithRetry`. One observation per attempt (not per cycle).

The per-attempt histogram observation site honors ADR-0059 OQ-6's
calibration intent: the load-bearing latency quantity is the
per-attempt commit-RPC duration, not the per-cycle aggregate
(which would include back-off sleep time per ADR-0059 §Clause 3's
loop shape). The per-cycle aggregate remains reconstructible from
the per-attempt histogram combined with the `commit_attempts` log
field ADR-0059 §Clause 5 commits.

### Clause 3 — Labels and cardinality posture

The label sets are minimal — `entity` across all three series;
`outcome ∈ {success_after_retry, exhausted}` on the retries
counter only. No `attempt`, no `partition`, no `broker`, no
`error_class` label is introduced. Rationale:

- Per-attempt cardinality is operator-recoverable from the
  per-attempt histogram combined with ADR-0059 §Clause 5's
  `commit_attempts` log field; introducing an `attempt` label
  would multiply cardinality by `max_attempts` (currently 3,
  potentially larger if ADR-0059 OQ-1 lifts the parameter to a
  tunable knob) without buying analysis the existing surfaces
  cannot support.
- Per-partition / per-broker labels would expose substrate
  internals (Kafka-specific) into a substrate-agnostic emission
  surface and would multiply cardinality by partition count,
  which is unbounded from the metric layer's perspective.
- An `error_class` label is gated on ADR-0059 OQ-2 (substrate-
  agnostic transient-vs-permanent classification at the
  `RecordConsumer` interface layer); it becomes natural after
  that OQ closes and extends additively per ADR-0039 §"Evolution
  rules" rule 1 at that time.

ADR-0039 §"Cardinality posture" continues to govern; no numeric
ceiling is committed by this ADR.

### Clause 4 — Histogram bucket boundaries: `prometheus.DefBuckets` (β)

The β buckets are the `prometheus.DefBuckets` constant from
`github.com/prometheus/client_golang`:

```
0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 (seconds)
```

These cover ADR-0059's documented range (§Consequence 2: `~100ms`
typical, `~150ms` expected, `600ms` worst-case at β parameters) at
a resolution that supports the calibration intent without
committing measurement-grounded boundaries pre-signal. A
measurement-grounded re-tuning may surface different optimal
boundaries once production signal accumulates — that re-tuning is
this ADR's OQ-1 below.

### Clause 5 — Shutdown exemption on the failures counter

`dq_record_commit_failures_total` increments only on
`commitWithRetry` non-nil returns that are **not**
`context.Canceled` / `context.DeadlineExceeded`. ADR-0059
§Clause 5 distinguishes context-cancellation from warning-log +
skip explicitly:

> Context-cancellation returns from `commitWithRetry` without
> warning-logging since shutdown is operator-driven, not a
> failure mode; the existing logger.Info "record runner shutting
> down" emission in `Start` is the operator-visible signal.

The failures counter follows the warning-log path: incremented
iff the runner warning-logs. Clean shutdown that catches an
in-flight commit does not increment the failures series; an alert
on `rate(dq_record_commit_failures_total[5m])` therefore tracks
broker-failure rate and is not perturbed by engine-restart events.

### Clause 6 — Test discipline: extend `fakeConsumer` + `prometheus/testutil`

`engine/internal/runner/record_runner_test.go`'s `fakeConsumer`
gains a `commitDurations []time.Duration` capture and a
configurable failure sequence (already present per ADR-0059
§Clause 6 — extended here with metric-assertion hooks). Three new
tests cover the material paths:

- `TestRecordRunner_CommitFailuresCounterIncrementsOnExhaustion`
  — asserts `dq_record_commit_failures_total{entity=...}` is
  incremented exactly once when `commitWithRetry` exhausts the
  retry budget on a broker-failure sequence; asserts the counter
  is **not** incremented when the helper returns due to context
  cancellation (the shutdown-exemption assertion per Clause 5).
- `TestRecordRunner_CommitRetriesCounterOutcomeLabels` — asserts
  `dq_record_commit_retries_total{entity=..., outcome="success_after_retry"}`
  is incremented when commit succeeds on attempt > 1, and that
  `outcome="exhausted"` increments when commit fails all
  `recordCommitMaxAttempts` attempts. Asserts the counter is
  **not** incremented on first-attempt success.
- `TestRecordRunner_CommitDurationHistogramObservesPerAttempt` —
  asserts `dq_record_commit_duration_seconds{entity=...}` records
  one observation per individual `consumer.Commit` call (not per
  cycle), via `prometheus/testutil.CollectAndCount` against the
  histogram's `_count` series after a multi-attempt sequence.

All existing tests continue to pass. Metric assertions use
`prometheus/testutil`'s `CollectAndCount` / `ToFloat64` /
`CollectAndCompare` helpers per ADR-0055 §Consequence 10. No
franz-go is brought into the unit-test layer.

### Clause 7 — `record-mode-conventions` skill convention S2 update

The `record-mode-conventions` skill at
`.claude/skills/record-mode-conventions/SKILL.md` and its
`reference/conventions.md` are updated in the same PR.
Convention S2 (β commit semantics; commit-after-dispatch spine
per ADR-0058; retry envelope per ADR-0059) gains a third
paragraph noting the emission surface for the commit path
(`dq_record_commit_failures_total` + `dq_record_commit_retries_total`
+ `dq_record_commit_duration_seconds`), the shutdown-exemption
distinction, and the per-attempt histogram observation site.
Light-touch update per [ADR-0053](./0053-record-mode-skill.md)'s
framing — S2's commit-after-dispatch + retry-envelope spines stay
unchanged; only the third paragraph is appended.

---

## Consequences

1. **B3-8 reaches `resolved-adr` via this ADR's promotion** under
   operator-authorized R4 scope collapse. The decision-log row
   updates accordingly; no other B-row is touched.

2. **`engine/internal/metrics.RunnerMetrics` gains three fields.**
   Construction in
   `engine/cmd/dq-engine/main.go:buildRecordRunners` continues to
   compile transparently — `*FranzConsumer` stays unchanged;
   `record_runner.go`'s Config receives the extended
   `RunnerMetrics` struct via the existing wiring per ADR-0055
   §Clause 3.

3. **`commitWithRetry` (ADR-0059 §Clause 3) gains three
   instrumentation call sites:** a `time.Now()` capture before
   each `consumer.Commit`, a
   `metrics.RecordCommitDuration.WithLabelValues(entity).Observe(...)`
   after the call returns (success or failure), and the per-cycle
   retries-counter increments at the success-after-retry and
   exhausted terminal branches. The retry-loop control shape is
   unchanged; instrumentation is observation-only.

4. **`closeAndDispatch` (ADR-0058 §Clause 2 site) gains one
   instrumentation call:** a
   `metrics.RecordCommitFailures.WithLabelValues(entity).Inc()`
   alongside the existing warning-log line on `commitWithRetry`
   non-nil return, **excluding** `context.Canceled` /
   `context.DeadlineExceeded` per Clause 5. The shutdown
   distinction keeps the failures counter aligned with the
   warning-log path actually firing.

5. **ADR-0058 / ADR-0059 / ADR-0055 / ADR-0039 / ADR-0021 /
   ADR-0023 / ADR-0024 / ADR-0002 / ADR-0003 / ADR-0049 are
   preserved.** The slice extends ADR-0039 along its own
   §"Evolution rules" rule 1 lane and adds handles to ADR-0055's
   emitter convention; no committed contract is reshaped.

6. **The OQ Register flips two rows to `resolved-adr` and extends
   one description** per the post-Wave-3 session-loop step 9
   OQ Register hunk rule:
   - ADR-0058 OQ-4 → `resolved-adr` ([ADR-0060](./0060-record-commit-emission-slice.md))
   - ADR-0059 OQ-3 → `resolved-adr` ([ADR-0060](./0060-record-commit-emission-slice.md))
   - ADR-0059 OQ-6 **stays at `open`**; the description column is
     extended to link ADR-0060 as the *enabling* slice that wires
     the commit-RPC histogram. The calibration analysis OQ-6
     literally names ("Quantitative stall-budget calibration —
     observed poll-batch processing time vs. retry stall") is not
     performed by ADR-0060 and is carried forward as ADR-0060's
     OQ-2 below. **Posture rationale**: the OQ Register §"Scope
     and conventions" defines `resolved-adr` as "consumed by a
     subsequent ADR or amendment" — ADR-0060 *enables* the
     calibration without performing it, so OQ-6 is not yet
     consumed. This posture (a) preserves the register's existing
     `resolved-adr` semantic without amending its conventions in
     the same PR.

7. **ADR-0060 adds four new OQ rows to the OQ Register** at
   promotion-PR time per the same step 9 hunk rule — one row per
   OQ enumerated below (OQ-1 through OQ-4).

8. **Cardinality posture continues to be governed by ADR-0039
   §"Cardinality posture".** No numeric ceiling is committed. The
   three new series' time-series contribution decomposes as:
   - `dq_record_commit_failures_total`: `entity × 1` series.
   - `dq_record_commit_retries_total`: `entity × 2` series (one
     labelset per `outcome` value).
   - `dq_record_commit_duration_seconds`: `entity × 14` series
     per labelset — 12 cumulative `_bucket` series (the 11
     explicit `prometheus.DefBuckets` boundaries plus the
     implicit `+Inf` bucket) plus `_count` and `_sum`.

   Total: `entity × (1 + 2 + 14) = entity × 17`. Operationally
   bounded by entity-count, which is bounded by the loader's
   manifest.

9. **`record-mode-conventions` skill convention S2 is updated**
   to mention the emission surface for the commit path in the
   same PR per Clause 7. Light-touch update per
   [ADR-0053](./0053-record-mode-skill.md)'s framing; the
   skill's seven conventions S1, S3–S7 are unaffected.

10. **No B-row backlog amendment.** B3-8 reaches `resolved-adr`
    via this ADR's promotion; no other B-row's row in
    `studies/foundation/06-decision-log.md` is touched. The
    OQ Register's two flipped rows + one description-extended
    row are register-side updates per the playbook step 9 hunk
    rule, not B-row amendments.

11. **PR-flow per `CONTRIBUTING.md` Flow 5 with the R4
    scope-collapse trailer.** The single PR carries the study,
    the round-1 critique capture, this ADR, the implementation,
    the three new tests, the skill S2 update, the decision-log
    row update, and the OQ Register hunk.

---

## Notes

- **R4 scope-collapse rationale.** This ADR's promotion +
  implementation slice land in a single PR per operator-
  authorized R4 scope collapse. Precedent:
  [ADR-0054](./0054-engine-image-registry-amendment.md) §Notes
  introduced the pattern at promotion time;
  [ADR-0055](./0055-metric-emission-slice-scope.md) §Notes,
  [ADR-0056](./0056-panel-5-lighting-slice.md) §Notes,
  [ADR-0058](./0058-record-runner-commit-after-dispatch.md)
  §Notes, and
  [ADR-0059](./0059-record-runner-commit-retry.md) §Notes
  carried it forward. The collapse is appropriate here because
  the implementation slice is small (three handles on
  `RunnerMetrics`; three instrumentation call sites in
  `commitWithRetry`; one instrumentation call site in
  `closeAndDispatch`; three new tests with extended
  `fakeConsumer` capture; one S2 skill paragraph) and the
  structural decisions (label sets, emission sites,
  shutdown-exemption discipline, bucket-boundary β) are
  load-bearing in the ADR. Separating them would split a
  single cohesive change across two sessions without reducing
  review load.

- **Clean eligibility — no D0 borderline.** The originating
  study cleared all four ADR-0049 §(a) conditions cleanly:
  Condition 1 via ADR-0039 §"Evolution rules" rule 1's
  explicit authorization of additive metrics within an
  engine-major (+ ADR-0055 §Consequence 6 naming the lane);
  Condition 2 via direct precedent reuse of ADR-0055's
  operator-ratified Tooling-extensions reading (no new
  expansive reading proposed — same disposition mechanism as
  B3-3 / ADR-0053 used for ADR-0051 Clause 1's adjacent-
  tooling reading); Condition 3 via the constraint envelope
  preserved verbatim; Condition 4 via material decisions
  (histogram bucket boundaries, label-set cardinality,
  emission-site placement, shutdown-exemption discipline).
  No `new contribution requiring review` reading is recorded
  here — eligibility is direct precedent reuse end-to-end.

- **OQ Register posture (a) ratified at round-1 critique
  disposition.** The originating study's round-1 critique
  surfaced a `resolved-adr` semantic question on ADR-0059
  OQ-6 (does wiring the histogram *consume* the OQ, or only
  *enable* its consumption?). Posture (a) — keep OQ-6 `open`
  with ADR-0060 linked as the enabling slice — was committed
  on the reading that the OQ Register §"Scope and conventions"
  defines `resolved-adr` as "consumed by a subsequent ADR or
  amendment", and ADR-0060 enables the calibration without
  performing it. The alternative posture (b) — flip OQ-6 to
  `resolved-adr` with a one-line register-convention note
  that "wiring closes the OQ; downstream calibration is a new
  OQ" — was rejected because it would have amended the
  register's conventions in the same PR, expanding R4 scope
  unnecessarily. The disposition is recorded here so a future
  reader encountering similar "enabling-not-consuming"
  patterns has the precedent.

- **Open Questions carried forward from the study (B3-8,
  OQ-1 through OQ-4).** Each is out-of-scope for this ADR per
  the originating study's AC-6 disposition; none introduce a
  blocking dependency on this ADR. They extend the
  observability surface in future B3-N entries when concrete
  demand surfaces.

  - **OQ-1: Histogram bucket boundary calibration.** This
    ADR commits `prometheus.DefBuckets` as β. A
    measurement-grounded re-tuning may surface different
    optimal boundaries once production signal accumulates
    (e.g., concentrating resolution in the `100–600ms` range
    to support OQ-2's calibration of ADR-0059 §Clause 2's β
    parameters). Out-of-scope for current cycle — pre-signal
    bucket-boundary tuning repeats the parameter-math posture
    the round-1 critique of B3-7 corrected.

  - **OQ-2: ADR-0059 OQ-6 calibration analysis.** This ADR
    wires the commit-RPC histogram so the calibration becomes
    performable; the calibration itself — comparing
    `dq_record_commit_duration_seconds` percentiles against
    observed poll-batch processing time under production load
    — remains a future B3-N when observation window
    accumulates. ADR-0059 OQ-6 in the OQ Register stays at
    `open` with this ADR linked as the enabling step (per
    §Consequences #6). Out-of-scope for current cycle —
    requires production telemetry that does not exist at this
    ADR's promotion time.

  - **OQ-3: Per-attempt label dimension.** Option B from the
    originating study (richer-cardinality reading with an
    `attempt` label on the duration histogram and the
    failures counter) remains available if operational
    investigation surfaces a need for per-attempt analysis
    the per-cycle aggregate cannot support. Out-of-scope for
    current cycle — cardinality is bounded per Clause 3; the
    warning-log `commit_attempts` field (ADR-0059 §Clause 5)
    carries the per-cycle attempt count for log-side analysis.

  - **OQ-4: Error-class label on the failures counter.**
    ADR-0059 OQ-2 stands as a separate deferred surface
    (substrate-agnostic transient-vs-permanent classification
    at the `RecordConsumer` interface layer). If that OQ
    closes, an `error_class` label on
    `dq_record_commit_failures_total` becomes natural and
    extends additively per ADR-0039 §"Evolution rules" rule 1.
    Out-of-scope for current cycle — bounded by ADR-0059
    OQ-2's prior closure.
