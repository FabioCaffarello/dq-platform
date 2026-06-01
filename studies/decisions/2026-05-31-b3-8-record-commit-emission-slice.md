<!-- path: studies/decisions/2026-05-31-b3-8-record-commit-emission-slice.md -->

# B3-8 — Record-Commit Emission Slice

## Metadata

- Date: 2026-05-31
- Status: draft
- Decision-log row: B3-8 (tooling extensions family — inherited
  precedent reuse of [ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md)'s
  Condition-2 ratification)
- Promotion target: [`docs/adr/0060-record-commit-emission-slice.md`](../../docs/adr/0060-record-commit-emission-slice.md)
  (reserved per
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 7; ADR-0059 merged 2026-05-31)
- Consumes (OQ Register):
  - [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
    OQ-4 (`dq_record_commit_failures_total`)
  - [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md)
    OQ-3 (`dq_record_commit_retries_total`)
  - [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md)
    OQ-6 (commit-RPC duration histogram — wiring half; the
    quantitative-calibration half remains open as a new OQ
    in the promotion ADR, pending production signal)
- Critique rounds:
  - Round 1 — [capture](../critiques/2026-06-01-b3-8-record-commit-emission-slice-critique-1.md)
    (0 blocking / 5 important / 5 minor); 5 important applied
    in this revision — (I-1) R2 cite corrected to ADR-0059
    §Consequence 2 with its actual numbers; (I-2) DD-3
    multiplication formula dropped in favor of an explicit
    back-off derivation; (I-3) failures-counter shutdown
    exemption (`context.Canceled` / `context.DeadlineExceeded`)
    applied at all three sites (Option A table row,
    §Recommendation Implementation-site bullet, §Consequences
    #4); (I-4) OQ Register posture (a) — ADR-0059 OQ-6 stays
    `open` with ADR-0060 linked as enabler, calibration carried
    by ADR-0060's OQ-2; (I-5) cardinality math corrected to
    `entity × 17` with the histogram decomposition shown
    (`_bucket` + `_count` + `_sum`). 5 minor deferred under the
    two-round cap per [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md).

---

## Context

[ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
(promoted 2026-05-30) committed the record-mode runner's
commit-after-dispatch posture — per-trigger commit on
`dispatcher.Run` success, warning-log + skip + transitive-commit
recovery on failure.
[ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md)
(promoted 2026-05-31) committed an exponential-back-off retry
envelope between `closeAndDispatch` and `consumer.Commit`
(`commitWithRetry` helper; `max_attempts = 3`; `base = 100ms`;
600ms worst-case stall; terminal behavior of ADR-0058 §Clause 2
preserved verbatim after retry exhaustion).

Three deferred observability questions surfaced in those ADRs'
§Notes, and the OQ Register's §"Adjacency cluster note"
(landed via Flow 6 at PR #127 on 2026-05-31) flagged them
explicitly as a co-discoverable B3-N seed:

> Three OQs converge on the same emission-slice trigger and
> would plausibly be consumed by the same future B3-N when
> production telemetry lands:
>
> - ADR-0058 OQ-4 (`dq_record_commit_failures_total`)
> - ADR-0059 OQ-3 (`dq_record_commit_retries_total`)
> - ADR-0059 OQ-6 (commit-RPC histogram for stall-budget calibration)

This study opens that B3-N. The scope is the **emission-slice
shape** — metric inventory, label sets, emission sites
against the runner's commit / retry boundaries, and the cardinality
posture each series introduces — not the quantitative
stall-budget calibration itself, which still requires production
signal that does not exist at this study's drafting time.

### Why this slice now

Three converging reasons:

1. **ADR-0058 §Clause 2 and ADR-0059 §Clause 5 commit a
   terminal warning-log + skip path that is operationally
   invisible without a metric.** Operators reading the
   `engine/internal/runner/runner.go` log stream see one line
   per commit failure; there is no aggregate signal — no
   alarm, no rate, no per-entity breakdown without grepping
   logs. ADR-0059 §Clause 5 explicitly added a
   `commit_attempts` log field to distinguish retried-and-
   exhausted from single-attempt-failure, but the operator-
   facing surface is still log-level; ADR-0039 §"Metric
   contract" is the contract a dashboard reads against.

2. **ADR-0059 OQ-6 is gated on having a histogram emitter to
   measure against.** OQ-6's two-fold description ("Emission
   slice wires commit-RPC histogram (adjacent to ADR-0058
   OQ-4 and ADR-0059 OQ-3); sufficient observation window
   accumulated") splits between the *wiring* half (this slice)
   and the *calibration* half (a future analysis once
   observation accumulates). Without the wiring, the
   calibration cannot begin and the `max_attempts = 3` /
   `base = 100ms` β parameters in ADR-0059 §Clause 2 remain
   unverifiable judgment calls.

3. **The OQ Register's adjacency cluster note made the
   convergence explicit.** Three OQs, three commit-boundary
   semantics, one emission package — splitting them into
   three separate B3-N entries would split a cohesive slice
   across three sessions without reducing review load and
   would risk inconsistent label sets across the three series.

### What ADR-0055 already committed

The slice consumes [ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md)'s
substrate verbatim:

- **Clause 1 (library).** `github.com/prometheus/client_golang`
  is the direct module dependency. No new library is introduced.
- **Clause 2 (route).** `GET /metrics` on the shared
  `http.ServeMux`. No new route.
- **Clause 3 (per-package emitter convention).**
  `engine/internal/metrics` owns the `prometheus.Registry` and
  the per-package typed Metrics structs (currently
  `RunnerMetrics` and `LoaderMetrics`). New series in this
  slice land as new fields on `RunnerMetrics` (the consumer is
  the `runner` package, which already takes its Metrics struct
  via `Config` per ADR-0055 §Clause 3).
- **Clause 4 (runner-side emissions).** Five of the six
  ADR-0039 metrics already emit from
  `engine/internal/runner/runner.go`. This slice adds three
  new series to the same runner package; the consumer site
  is `engine/internal/runner/record_runner.go` (the
  `commitWithRetry` helper and `closeAndDispatch` per
  ADR-0059 §Clause 3 + ADR-0058 §Clause 2).
- **Clause 6 (cardinality posture).** ADR-0039 §"Cardinality
  posture" continues to govern; no numeric ceiling is
  committed; this slice preserves the posture and adds label
  decisions explicitly so the cardinality contribution is
  auditable.

### Eligibility under ADR-0049 §(a)

Required per
[`.claude/playbooks/post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
step 2. All four conditions must hold.

| # | Condition | Resolution |
|---|---|---|
| 1 | P-B3.1 — expands not rewrites | **Passes cleanly.** [ADR-0039](../../docs/adr/0039-dashboard-contract.md) §"Evolution rules" rule 1 explicitly authorizes "a new metric" as additive within an engine-major version. ADR-0055 §Consequence 6 names this lane verbatim: "Any future addition to ADR-0039's metric set lands in this package first; consuming packages take handles via their Config." ADR-0058 §Notes (OQ-4) and ADR-0059 §Notes (OQ-3, OQ-6) routed all three series to "the next emission-slice session re-scopes ADR-0039 inventory" — that is the open OQ decision space this study fills. ADR-0058 §Clause 2's terminal warning-log + skip behavior and ADR-0059 §Clause 5's retry-then-warning-log behavior remain verbatim; emission observes the commit / retry boundary without changing its control. No D0 borderline. |
| 2 | P-B3.4 — in-scope family | **Passes via inherited precedent — no new borderline.** Tooling extensions family by direct reuse of [ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md)'s operator-ratified Condition-2 reading ("engine-runtime emission as Tooling-extensions stretching ADR-0049 §"Per-family scope" → "engine dispatcher, and adjacent tooling" past the lint-extension canonical example"). This study sits inside ADR-0055's already-ratified envelope — same `engine/internal/metrics` package, same `/metrics` route, same `github.com/prometheus/client_golang` library, same per-package emitter convention, same runner-side emission discipline. No new expansive reading is proposed. Same disposition shape as B3-3 / ADR-0053 ("Condition 2 via direct precedent reuse"). |
| 3 | P-B3.2 — envelope conformance | **Passes cleanly.** [ADR-0020](../../docs/adr/0020-wave-s-launch.md) substrate posture, [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) mode primitive, [ADR-0022](../../docs/adr/0022-kind-catalog.md) kind catalog, [ADR-0023](../../docs/adr/0023-sources-schema.md) sources schema all unchanged. [ADR-0024](../../docs/adr/0024-window-semantics.md) windowing, [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md) `execution_id`, [ADR-0003](../../docs/adr/0003-result-write-model.md) write model, [ADR-0039](../../docs/adr/0039-dashboard-contract.md) metric contract, [ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md) emitter convention, [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md) commit boundary, and [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md) retry envelope all preserved verbatim. The slice extends ADR-0039's inventory along its own §"Evolution rules" rule 1 lane (additive metric within engine-major). |
| 4 | Additive-maintenance threshold | **Passes.** Three new metric series including a histogram. Material decisions to commit: (a) bucket boundaries for the duration histogram; (b) label-set cardinality posture across the three series (per-entity vs. per-(entity, outcome) vs. richer); (c) emission-site placement against the `commitWithRetry` helper's loop boundary (per-attempt vs. per-cycle); (d) whether failures and retries collapse into one counter with an `outcome` label or stay separate. Not a routine catalog-PR shape. |

**Temporal classification.** Today 2026-05-31; Wave-S full gate
closed 2026-05-25 per [`CLAUDE.md`](../../CLAUDE.md) §2.2. The
three OQs come from ADR-0058 (accepted 2026-05-30) and
ADR-0059 (accepted 2026-05-31). Both ADRs are post-shipping
against a closed wave; per `MEMORY.md`
`code-comment-deferrals-vs-adr-decisions.md` (OQ-in-§Notes
deferral shape admits B3). Post-shipping against a closed
wave and two closed ADRs → **B3**.

**Amendment-vs-B3 disambiguation.** This slice does not modify
ADR-0058 §Clause 2's terminal behavior (warning-log + skip +
transitive-commit recovery still fires verbatim after retry
exhaustion), does not modify ADR-0059 §Clause 5's terminal
behavior (the same warning log emits, additionally accompanied
by metric increments), and does not modify ADR-0055's emitter
convention (the slice consumes it). The three new metric names
extend ADR-0039's inventory via its own §"Evolution rules"
additive-within-major lane — not amendment.

---

## Decision Drivers

- **DD-1 — Three distinct semantics warrant three distinct
  series.** Failures (terminal-after-retry-exhaustion),
  retries (algorithm-internal back-off), and commit-RPC
  duration (per-attempt latency) carry three distinct
  operational meanings. Collapsing them into one counter with
  an `outcome` label costs the operator query clarity
  (a `rate(...{outcome="exhausted"}[5m])` per ADR-0058 §Clause 2
  failure rate vs. a `rate(...{outcome="retry"}[5m])` per
  ADR-0059 §Clause 3 retry rate read as cleanly with separate
  series). Three names match three contract surfaces.

- **DD-2 — Cardinality is bounded by recycling existing
  labels.** Every existing runner-side metric per ADR-0055
  carries `entity` as its load-bearing breakdown label. The
  new series follow the same shape: `entity` + a small
  outcome enum where applicable. No new high-cardinality
  label (per-attempt, per-partition, per-broker) is
  introduced; ADR-0039 §"Cardinality posture" is preserved
  without re-litigation.

- **DD-3 — Histogram observation site is per-attempt, not
  per-cycle.** ADR-0059 OQ-6's calibration question is
  "observed poll-batch processing time vs. retry stall" —
  the load-bearing quantity is the *per-attempt* commit-RPC
  latency (one `consumer.Commit` call). The per-cycle
  aggregate is `Σ per-attempt durations + Σ back-off durations`
  per ADR-0059 §Clause 3's loop shape; it includes back-off
  sleep time that the calibration is not asking about, and
  the per-attempt portion is anyway reconstructible from the
  per-attempt histogram combined with the `commit_attempts`
  log field ADR-0059 §Clause 5 commits. Per-attempt
  observation is the primitive; per-cycle is a derived view.

- **DD-4 — Failures counter increments once per cycle, not
  once per attempt.** The operational signal an operator
  wants from `dq_record_commit_failures_total` is "how often
  does the retry budget exhaust" (the ADR-0058 §Clause 2
  warning-log path actually fires) — not "how often does an
  individual attempt fail" (the latter conflates retried-
  recovered cases with terminal failures). One increment per
  `commitWithRetry` returning non-nil keeps the counter
  semantics aligned with ADR-0058 §Clause 2's terminal
  boundary.

- **DD-5 — Retries counter records `outcome` so the success-
  after-retry case is observable.** `commitWithRetry` has
  three terminal states from the runner's perspective:
  succeeded-on-first-attempt (the no-op-retry path,
  uninstrumented), succeeded-after-N-retries
  (`outcome=success_after_retry`), and exhausted
  (`outcome=exhausted`). The success-after-retry case is
  what tells the operator the retry layer is *working* (the
  primary motivation of ADR-0059); without an `outcome`
  label the operator cannot distinguish "retries are
  absorbing transients" from "retries are exhausting."

- **DD-6 — Bucket boundaries are deferred to OQ-pending-
  signal.** Picking histogram buckets without production
  signal is the same posture mistake ADR-0059's original
  draft made with the cap parameter (its 1.6s figure was
  unsupported; the round-1 critique exposed it). The β
  buckets should cover ADR-0059's documented range
  (§Consequence 2: `~100ms` typical, `~150ms` expected,
  `600ms` worst-case), but a measurement-grounded bucket-
  boundary commit waits for production data — recorded as
  the promotion ADR's OQ.

---

## Considered Options

### Option A — Three series, minimal labels (Recommended)

Three new metric series added to `RunnerMetrics`:

| Metric name | Type | Labels | Emission site (in `engine/internal/runner/record_runner.go`) | Increment / observe semantics |
|---|---|---|---|---|
| `dq_record_commit_failures_total` | counter | `entity` | `closeAndDispatch` post-`commitWithRetry`, on non-nil return that is **not** `context.Canceled` / `context.DeadlineExceeded` (terminal warning-log path per ADR-0058 §Clause 2 + ADR-0059 §Clause 5 — the shutdown exemption keeps the counter aligned with the warning-log path actually firing per §Clause 5's "shutdown is operator-driven, not a failure mode" distinction) | One increment per `commitWithRetry` broker-failure cycle; clean shutdown does not increment |
| `dq_record_commit_retries_total` | counter | `entity`, `outcome` (∈ `success_after_retry`, `exhausted`) | Inside `commitWithRetry`, on the success-after-retry path (where `attempt > 1`) and on the exhausted path (where `attempt == recordCommitMaxAttempts` and `err != nil`) | One increment per `commitWithRetry` cycle that consumed at least one retry; not incremented on first-attempt success |
| `dq_record_commit_duration_seconds` | histogram | `entity` | Inside `commitWithRetry`, around each `consumer.Commit(ctx, records)` call (per-attempt observation per DD-3) | One observation per attempt; histogram bucket boundaries deferred to OQ (DD-6) |

**Strengths.** Three distinct semantics → three distinct series
(DD-1). Bounded cardinality — only `entity` is new across all
three; `outcome` adds two values × N entities for the retries
counter (DD-2). Per-attempt histogram observation matches
ADR-0059 OQ-6's calibration intent (DD-3). Failures counter
increments at the ADR-0058 §Clause 2 terminal boundary (DD-4);
retries counter distinguishes the success case from exhaustion
(DD-5).

**Weaknesses.** Three series cost three handles on
`RunnerMetrics`; emission sites are spread between
`commitWithRetry` (retries + duration) and `closeAndDispatch`
(failures); operator dashboards need three panels rather than
one to read the cluster.

### Option B — Three series, richer labels (per-attempt cardinality)

Same three series as Option A, but the `dq_record_commit_duration_seconds`
histogram and the `dq_record_commit_failures_total` counter
each add an `attempt` label (`1`, `2`, `3` at β parameters per
ADR-0059 §Clause 2). The intent is richer analysis: per-attempt
latency distribution (does attempt 3 typically take longer
than attempt 1?) and per-attempt failure attribution (which
attempt is the most failure-prone?).

**Strengths.** Richer analysis surface — the per-attempt
breakdown is what a deep operational investigation would want
when broker behavior is misbehaving in unexpected ways.

**Weaknesses.** Cardinality grows as `max_attempts × entities`
(currently 3×N; would grow if ADR-0059 OQ-1 lifts
`max_attempts` to a tunable knob with a larger value).
ADR-0039 §"Cardinality posture" deferred a numeric ceiling but
the spirit of the deferral is "stay conservative until signal
demands more." Per-attempt observation is operator-recoverable
from the per-cycle histogram if combined with the
`commit_attempts` field already in the warning log (ADR-0059
§Clause 5) — the analysis is possible without the label-
cardinality cost. The `attempt` label is also potentially
unstable: ADR-0059 OQ-1 may promote `max_attempts` to a
tunable knob, at which point the `attempt` label's value range
expands without an enum-evolution mechanism for histograms.

### Option C — One combined counter + one histogram

A single counter `dq_record_commit_attempts_total` with
`outcome ∈ (success_first_attempt, success_after_retry, retry_failed, exhausted)`
captures both failures and retries; one histogram
`dq_record_commit_duration_seconds` covers duration. The
emission cluster collapses to two metric names.

**Strengths.** Two new handles instead of three; one counter
to dashboard rather than two; operator-facing inventory
expands by two rather than three.

**Weaknesses.** Loses the DD-1 separation — a
`rate(...{outcome="exhausted"}[5m])` query is identical to
the per-failures-counter case in Option A, but a parallel
query for the retry rate must filter out
`outcome=success_first_attempt` to avoid drowning the retry
signal in baseline success volume. The aggregate
`rate(dq_record_commit_attempts_total[5m])` answers "how busy
is the commit path overall" — but that question is already
answerable by the dispatcher's existing `dq_runs_total`
(divided by mode) and is not a load-bearing operational
signal. The cardinality contribution is identical to Option
A (`entity × 4 outcomes` vs. `entity × 1 + entity × 2 + entity × 1`
≈ same order). The conceptual collapse buys nothing
operationally and costs query clarity.

### Option D — Status quo (do not consume the cluster)

Keep ADR-0058 OQ-4, ADR-0059 OQ-3, and ADR-0059 OQ-6 open.
Operators read commit failures from logs (existing warning-log
line per ADR-0058 §Clause 2 + ADR-0059 §Clause 5 includes the
`commit_attempts` field). The OQ-6 calibration question
remains permanently un-addressable because no histogram is
wired.

**Strengths.** Zero added emission code; no new dashboard
panels needed; the existing log surface continues to carry
the information.

**Weaknesses.** Three OQs whose `Trigger condition (per P-B3.3)`
column in the OQ Register explicitly names "next emission-slice
session re-scopes ADR-0039 inventory" remain unaddressed;
operators have no aggregate rate signal for the commit-failure
path (logs are per-event, not rate-shaped); ADR-0059's β
parameters remain permanently un-calibrated (OQ-6 cannot
proceed without a histogram). Defers OQ Register adjacency
cluster note's explicit B3-N seed indefinitely without a
demand-driven counter-signal.

---

## Recommendation

**Option A.** Three distinct semantics warrant three distinct
series (DD-1) at bounded cardinality (DD-2). The per-attempt
histogram observation site (DD-3) is what ADR-0059 OQ-6's
calibration intent loads on; the per-cycle failures counter
(DD-4) matches ADR-0058 §Clause 2's terminal boundary; the
retries counter's `outcome` label (DD-5) makes the success-
after-retry case observable, which is what tells the operator
the retry layer is working.

Option B is rejected because the per-attempt cardinality buys
analysis the per-cycle histogram already supports (via
`commit_attempts` in the log) at a cardinality cost the
ADR-0039 deferral spirit asks us to delay; the `attempt`
label is also fragile under ADR-0059 OQ-1's potential
parameter-tunability lift.

Option C is rejected because the conceptual collapse costs
query clarity (DD-1) without compensating operational benefit.

Option D is rejected because the OQ Register adjacency cluster
note explicitly seeded this work and the three OQs' trigger
conditions are met (an emission-slice session re-scoping
ADR-0039's inventory is literally this study); deferring leaves
ADR-0059's parameters un-calibrable.

**Reading carried forward as new contribution proposed here,
requires review (R5) — none.** Eligibility passes all four
conditions cleanly via direct precedent reuse. ADR-0055's
Condition-2 ratification covers the engine-runtime-emission
reading; no new D0 borderline is introduced.

### What this study commits

- **Decision shape.** Option A: three series
  (`dq_record_commit_failures_total` counter,
  `dq_record_commit_retries_total` counter with
  `outcome ∈ (success_after_retry, exhausted)`,
  `dq_record_commit_duration_seconds` histogram).
- **Implementation site.** Three new fields on
  `metrics.RunnerMetrics` (per ADR-0055 §Clause 3). Emission
  sites: failures counter at `closeAndDispatch` post-
  `commitWithRetry` non-nil return, **excluding**
  `context.Canceled` / `context.DeadlineExceeded` (clean
  shutdown is operator-driven, not a failure mode, per
  ADR-0059 §Clause 5); retries counter at `commitWithRetry`
  `success_after_retry` and `exhausted` branches; duration
  histogram around each `consumer.Commit` call inside
  `commitWithRetry`.
- **Labels.** Minimal — `entity` across all three;
  `outcome ∈ (success_after_retry, exhausted)` on the
  retries counter only. No `attempt`, no `partition`, no
  `error_class` label.
- **Histogram bucket boundaries.** Two-fold disposition:
  - **β commits Prometheus-library default buckets**
    (`prometheus.DefBuckets` → `5ms, 10ms, 25ms, 50ms, 100ms,
    250ms, 500ms, 1s, 2.5s, 5s, 10s`). These cover ADR-0059's
    documented range (§Consequence 2: `~100ms` typical,
    `~150ms` expected, `600ms` worst-case) at a resolution
    that supports the calibration intent.
  - **Bucket-boundary calibration is OQ-1** below — a
    measurement-grounded re-tuning waits for production
    signal (parallel to ADR-0059 OQ-6's two-fold structure;
    the wiring half is this slice's commit, the calibration
    half remains open as OQ-1 here).
- **Test discipline.** Extend `engine/internal/runner/record_runner_test.go`'s
  `fakeConsumer` to assert metric increments via
  `prometheus/testutil` (parallel to ADR-0055 §Consequence 10).
  Three new tests: one asserts the failures counter
  increments on `commitWithRetry` exhaustion;
  one asserts the retries counter increments with the right
  `outcome` label on both the success-after-retry and
  exhausted paths; one asserts the duration histogram
  records observations on every `consumer.Commit` attempt.
- **Skill-side update.** The `record-mode-conventions` skill
  at `.claude/skills/record-mode-conventions/SKILL.md`
  convention S2 receives a third paragraph noting the
  emission surface for the commit path. Light-touch update
  per ADR-0053's framing.

### What this study does NOT commit

- A change to ADR-0058 §Clause 2 or ADR-0059 §Clause 5
  terminal behavior. The warning-log + skip + transitive-
  commit recovery path fires verbatim after retry
  exhaustion; the metric increment is added alongside, not
  in place of, the warning log.
- A change to ADR-0055's library / route / per-package
  emitter convention. The three new series instantiate new
  handles on `metrics.RunnerMetrics` within the existing
  package shape.
- A change to ADR-0039's `/metrics` endpoint or its
  Cardinality posture. The slice adds three names additively
  per §"Evolution rules" rule 1; no cardinality ceiling is
  proposed (the deferral continues).
- A quantitative calibration of ADR-0059's β parameters.
  ADR-0059 OQ-6's calibration half remains open (this slice
  wires the histogram so the calibration becomes possible;
  the calibration analysis itself is a separate B3-N when
  observation window accumulates per the OQ Register's
  trigger condition).
- A per-attempt or per-partition label. Cardinality posture
  per DD-2.
- A transient-vs-permanent error classification on the
  failures counter. ADR-0059 OQ-2 stands as a separate
  deferred surface.
- An alert rule against the new series. Alert routing per
  ADR-0006 is consumer-side; the metric contract makes the
  alert possible, not mandatory.
- A `RecordConsumer` interface evolution. The retry layer
  per ADR-0059 stays runner-package-internal; the consumer
  contract remains one Commit RPC per call.

---

## Consequences

1. **B3-8 reaches `resolved-study` on operator approval of
   this draft and `/critique` round disposition;
   promotion to ADR-0060 in the same session is the
   recommended path per operator-authorized R4 scope
   collapse** (precedent
   [ADR-0054](../../docs/adr/0054-engine-image-registry-amendment.md)
   §Notes,
   [ADR-0055](../../docs/adr/0055-metric-emission-slice-scope.md)
   §Notes,
   [ADR-0056](../../docs/adr/0056-panel-5-lighting-slice.md)
   §Notes,
   [ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
   §Notes,
   [ADR-0059](../../docs/adr/0059-record-runner-commit-retry.md)
   §Notes). The collapse rationale: the slice is small
   (three handles on `RunnerMetrics`; three emission-site
   call additions; three new tests; one S2 skill paragraph)
   and the structural decisions (label sets, emission sites,
   bucket-boundary β) are load-bearing in the ADR.

2. **`engine/internal/metrics.RunnerMetrics` gains three
   fields.** Construction in
   `engine/cmd/dq-engine/main.go:buildRecordRunners`
   continues to compile transparently — `*FranzConsumer`
   stays unchanged; `record_runner.go`'s Config receives
   the extended `RunnerMetrics` struct via the existing
   wiring.

3. **`commitWithRetry` (ADR-0059 §Clause 3) gains three
   instrumentation call sites:** a `time.Now()` capture
   before each `consumer.Commit`, a
   `metrics.RecordCommitDuration.WithLabelValues(entity).Observe(...)`
   after the call returns (success or failure), and the
   per-cycle counter increments at the success-after-retry
   and exhausted terminal branches. The retry-loop control
   shape is unchanged; instrumentation is observation-only.

4. **`closeAndDispatch` (ADR-0058 §Clause 2 site) gains
   one instrumentation call:** a
   `metrics.RecordCommitFailures.WithLabelValues(entity).Inc()`
   alongside the existing warning-log line on
   `commitWithRetry` non-nil return, **excluding**
   `context.Canceled` / `context.DeadlineExceeded`. ADR-0059
   §Clause 5 distinguishes those from warning-log + skip
   ("Context-cancellation returns from `commitWithRetry`
   without warning-logging since shutdown is operator-
   driven, not a failure mode"); the counter follows the
   warning-log path so clean shutdown does not increment
   the failures series.

5. **ADR-0058 / ADR-0059 / ADR-0055 / ADR-0039 / ADR-0021 /
   ADR-0023 / ADR-0024 / ADR-0002 / ADR-0003 / ADR-0049
   are preserved.** The slice extends ADR-0039 along its
   own §"Evolution rules" #1 lane and adds handles to
   ADR-0055's emitter convention; no committed contract is
   reshaped.

6. **The OQ Register flips two rows to `resolved-adr` and
   extends one description at promotion-PR time** per the
   playbook step 9 OQ Register hunk rule:
   - ADR-0058 OQ-4 → `resolved-adr ([ADR-0060](...))`
   - ADR-0059 OQ-3 → `resolved-adr ([ADR-0060](...))`
   - ADR-0059 OQ-6 **stays at `open`**; the description
     column is extended to link ADR-0060 as the *enabling*
     slice that wires the commit-RPC histogram. The
     calibration analysis OQ-6 literally names
     ("Quantitative stall-budget calibration — observed
     poll-batch processing time vs. retry stall") is not
     performed by ADR-0060 and is carried forward as
     ADR-0060's OQ-2 below. Posture rationale: the OQ
     Register §"Scope and conventions" defines
     `resolved-adr` as "consumed by a subsequent ADR or
     amendment" — ADR-0060 *enables* the calibration
     without performing it, so OQ-6 is not yet consumed.
     This posture (a) preserves the register's existing
     `resolved-adr` semantic without amending its
     conventions in the same PR.

7. **ADR-0060 adds new OQ rows to the OQ Register at
   promotion-PR time** for each labeled OQ this study
   defers below (OQ-1 / OQ-2 / OQ-3 / OQ-4 — see §"Open
   Questions"). Per the description-sourcing rule for
   ADRs that label OQs in §Notes, the register row carries
   the ADR §Notes restatement (the typical case).

8. **Cardinality posture continues to be governed by
   ADR-0039 §"Cardinality posture".** No numeric ceiling
   is committed; the three new series' time-series
   contribution decomposes as:
   - `dq_record_commit_failures_total` (counter, `entity`
     label only): `entity × 1` series.
   - `dq_record_commit_retries_total` (counter, `entity`
     + `outcome`): `entity × 2` series (one labelset per
     `outcome` value).
   - `dq_record_commit_duration_seconds` (histogram,
     `entity` label): `entity × 14` series per labelset
     — 12 cumulative `_bucket` series (the 11 explicit
     `prometheus.DefBuckets` boundaries plus the implicit
     `+Inf` bucket) plus `_count` and `_sum`.

   Total: `entity × (1 + 2 + 14) = entity × 17`.
   Operationally bounded by entity-count, which is
   bounded by the loader's manifest.

9. **`record-mode-conventions` skill convention S2 is
   updated** to mention the emission surface for the
   commit path in the same PR. Light-touch update per
   ADR-0053's framing.

10. **No B-row backlog amendment.** B3-8 reaches
    `resolved-adr` via this ADR's promotion; no other
    B-row's row in `studies/foundation/06-decision-log.md`
    is touched. The OQ Register's three flipped rows are a
    register-side update per the playbook step 9 hunk
    rule, not a B-row amendment.

11. **PR-flow per CONTRIBUTING.md Flow 5 with the R4
    scope-collapse trailer.** The single PR carries the
    study + the round-1 critique capture + ADR-0060 +
    the implementation slice + the three new tests + the
    decision-log row update + the OQ Register hunk + the
    skill S2 update.

---

## Open Questions

- **OQ-1: Histogram bucket boundary calibration.** This
  slice commits Prometheus-library default buckets
  (`prometheus.DefBuckets`). A measurement-grounded re-
  tuning may surface different optimal boundaries once
  production signal accumulates (e.g., concentrating
  resolution in the `100–600ms` range to support
  ADR-0059 OQ-6's calibration). **Out-of-scope for
  current cycle** — pre-signal bucket-boundary tuning
  repeats the parameter-math posture the round-1
  critique of B3-7 corrected (judgment without grounding
  is worse than a default with an explicit deferral).

- **OQ-2: ADR-0059 OQ-6 calibration analysis.** This
  slice wires the commit-RPC histogram so that the
  calibration becomes performable; the calibration
  itself — comparing `dq_record_commit_duration_seconds`
  percentiles against observed poll-batch processing
  time under production load — remains a future study
  when an observation window accumulates. ADR-0059 OQ-6
  in the OQ Register stays at `open` with this slice
  linked in its description column as the *enabling*
  step (per Consequences #6 above); the calibration is
  not consumed until a future B3-N performs the
  analysis. **Out-of-scope for current cycle** —
  requires production telemetry that does not exist at
  this slice's promotion time.

- **OQ-3: Per-attempt label dimension.** Option B's
  richer-cardinality reading remains available if
  operational investigation surfaces a need for per-
  attempt analysis the per-cycle aggregate cannot
  support. **Out-of-scope for current cycle** —
  cardinality is bounded per DD-2; the warning-log
  `commit_attempts` field (ADR-0059 §Clause 5) carries
  the per-cycle attempt count for log-side analysis.

- **OQ-4: Error-class label on the failures counter.**
  ADR-0059 OQ-2 stands as a separate deferred surface
  (substrate-agnostic transient-vs-permanent
  classification at the `RecordConsumer` interface
  layer). If that OQ closes, an `error_class` label on
  `dq_record_commit_failures_total` becomes natural and
  the failures counter's label cardinality extends
  additively per ADR-0039 §"Evolution rules" rule 1.
  **Out-of-scope for current cycle** — bounded by
  ADR-0059 OQ-2's prior closure.

---

## Promotion target

[`docs/adr/0060-record-commit-emission-slice.md`](../../docs/adr/0060-record-commit-emission-slice.md)
(reserved per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7; ADR-0059 merged 2026-05-31 as the highest
preceding reserved number).
