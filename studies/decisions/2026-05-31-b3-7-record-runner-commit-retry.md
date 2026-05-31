<!-- path: studies/decisions/2026-05-31-b3-7-record-runner-commit-retry.md -->

# B3-7 — Record Runner Commit Retry Policy

## Metadata

- Date: 2026-05-31
- Status: draft
- Decision-log row: B3-7 (capability mode extensions family)
- Promotion target: [`docs/adr/0059-record-runner-commit-retry.md`](../../docs/adr/0059-record-runner-commit-retry.md)
- Critique rounds:
  - Round 1 — [capture](../critiques/2026-05-31-b3-7-record-runner-commit-retry-critique-1.md) (0 blocking / 4 important / 5 minor); 4 important applied (parameter-math corrected via Option (c) cap-removal; two R5-spirit phrasings rewritten in own terms; DD-2 quantification weakened to qualitative); 2 minors applied (Consequence §7 verification posture; Consequence §8 wording precision); 3 minors deferred under the two-round cap (full-jitter terminology; `kerr.IsRetriable` OQ-2 mention; OQ-4 wording sharpening).

---

## Context

[ADR-0058](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
(promoted 2026-05-30) committed the record-mode runner's
commit-after-dispatch posture: per-trigger commit keyed on
`dispatcher.Run` success, with `consumer.Commit` failures
"warning-logged and do not propagate; the next successful
dispatch in the same partition commits the uncommitted records
transitively via high-water-mark monotonicity" (ADR-0058
§Clause 2).

ADR-0058 §Notes recorded OQ-3 verbatim:

> **OQ-3: Commit failure retry / back-off policy.** β posture
> on `consumer.Commit` returning non-nil is warning-log + skip;
> the records re-flow on restart or via the next successful
> dispatch's transitive commit. A bounded retry (e.g., three
> attempts with exponential back-off) might be operationally
> better when broker connectivity is flaky. **Out-of-scope for
> current cycle** — β posture is conservative-safe;
> operational signal will reveal whether retries are needed.

This study consumes OQ-3 and commits a bounded retry policy.

### Why the β posture is incomplete

Under ADR-0058's β posture, a single transient broker hiccup at
commit time (e.g., a leader-election blip, a TCP RST on an idle
connection, a 100ms broker pause for a fsync) costs the runner
**one full window's commit boundary**. The records remain
uncommitted; the next window's successful dispatch eventually
commits them transitively, *but only if the next window closes
in the same partition*. A topic with sparse traffic in the
affected partition can leave the window's records uncommitted
for an arbitrarily long stretch.

Restart-based recovery is correct (at-least-once safety per
ADR-0058 + ADR-0003 §1+§2) but operationally heavy: the operator
sees a stale committed-offset on the broker; alarm fatigue if
the consumer-lag dashboard is wired to broker-side committed-
offset rather than runner-side dispatched-offset.

The β posture optimizes for **simplicity at the cost of
transient-blip tolerance**. A bounded retry layer shifts the
trade-off: small added complexity, substantially better
tolerance for the transient case (the most common case in
practice).

### What does NOT change

- **ADR-0058 §Clause 2 terminal behavior is preserved.** After
  retries are exhausted, the runner still warning-logs + skips;
  the next successful dispatch in the same partition still
  commits transitively; restart-based recovery still works.
- **At-least-once delivery semantics are preserved.** The
  composed model (ADR-0024 deterministic windowing + ADR-0002
  deterministic `execution_id` + ADR-0003 §2 canonical-view
  collapse) absorbs any spurious second attempt at the
  consumer-visible layer; the retry layer adds latency
  resilience but does not alter the delivery guarantee.
- **The single-goroutine runner shape is preserved.** Retries
  block the consumer poll loop synchronously; the retry budget
  is the load-bearing bound on how long the loop can stall.
  This is intentional: a concurrent retry would re-open the
  ordering / serialization design surface (OQ-1 from B3-6
  remains deferred).
- **`RecordConsumer` interface signature is unchanged.** The
  retry layer lives in the runner, above `consumer.Commit`. No
  interface evolution.

### Eligibility under ADR-0049 §(a)

Required per
[`.claude/playbooks/post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
step 2. All four conditions must hold.

| # | Condition | Resolution |
|---|---|---|
| 1 | P-B3.1 — expands not rewrites | **Passes.** The retry layer is inserted **between** the runner's `closeAndDispatch` and `consumer.Commit`; ADR-0058 §Clause 2's terminal behavior (warning-log + skip + transitive-commit fallback) is preserved verbatim after the retry budget is exhausted. No clause of ADR-0058 §Decision is reshaped. **Borderline reading flagged:** one reader might argue inserting a retry loop modifies §Clause 2's decided shape ("commit failures are warning-logged" → "commit failures are retried-then-warning-logged"). My reading: ADR-0058 OQ-3 explicitly named bounded retry as "deferred-implementation framing" — that's the *open* OQ-3 decision space the new ADR fills, not the *decided* shape of §Clause 2. The retry layer is evolution against an authorized OQ deferral, parallel to how ADR-0058 itself filled the kafka_consumer.go β-marker deferral. R5 + adr-writing A7 carry-forward; precedent ADR-0055 / ADR-0056 / ADR-0058 §Notes. |
| 2 | P-B3.4 — in-scope family | **Passes.** ADR-0049 §"Per-family scope" → "Capability mode extensions" → "a refinement to how capability is dispatched at runtime". Same family as B3-6 / ADR-0058; the retry layer refines the commit-after-dispatch dispatch boundary. |
| 3 | P-B3.2 — envelope conformance | **Passes.** ADR-0020 (substrate posture), ADR-0021 (mode primitive), ADR-0022 (kind catalog), ADR-0023 (sources schema), ADR-0024 (windowing semantics), ADR-0002 (`execution_id` formula), ADR-0003 (write model), and ADR-0058 (commit-after-dispatch terminal behavior) are all preserved unchanged. |
| 4 | Additive-maintenance threshold | **Passes.** Introduces a configurable retry budget + back-off math + new operator-observable behavior (commit latency under flaky broker is bounded by retry budget instead of single-RTT failure-then-skip). New failure semantics (after retries exhausted, what happens) are decided here even though they match ADR-0058's existing terminal. |

Temporal classification: today is 2026-05-31; Wave-S full gate
closed 2026-05-25; ADR-0058 merged 2026-05-30. Post-shipping
against a closed wave and a closed ADR — B3.

---

## Decision Drivers

- **DD-1 — Transient broker hiccups are the common case; retries
  match the failure-mode distribution.** Broker-side transient
  failures (leader election, TCP RST on idle connection, brief
  fsync pause) are operationally common and resolve in
  milliseconds to a few hundred milliseconds. A retry layer
  tuned to that range smooths over them; ADR-0058's β posture
  treats them identically to permanent failure (warning-log +
  skip + restart-based recovery), which is operationally costly
  for a case that resolves itself in <1s.

- **DD-2 — Single-goroutine runner caps the acceptable stall
  budget.** Retries block the consumer poll loop. A retry policy
  must bound total wait time tightly enough that the loop's
  forward progress isn't visibly degraded. Concrete bound is the
  worst-case sum of back-offs, which must remain small relative
  to typical poll-batch processing time — a quantitative
  multiplier requires operational signal not available at this
  ADR's promotion time (OQ-6 carries the measurement-once-shipped
  framing).

- **DD-3 — Jitter prevents thundering-herd on broker recovery.**
  If a broker is unavailable and N concurrent record-mode runners
  all retry on a fixed back-off, they synchronize their retry
  attempts. Jitter de-synchronizes them so broker recovery isn't
  immediately re-overwhelmed.

- **DD-4 — Context cancellation must break the retry loop
  promptly.** Engine shutdown (ctx.Done) is an operator-driven
  signal; a retry loop that ignores it stalls shutdown by the
  retry budget. The loop must check ctx between attempts.

- **DD-5 — Terminal behavior must remain ADR-0058 §Clause 2.**
  After retries are exhausted, the runner returns to ADR-0058's
  warning-log + skip path. The retry layer is *additive* on top
  of the existing terminal; it does not replace it. This is
  what keeps the eligibility Condition 1 reading defensible.

---

## Considered Options

### Option A — No retry (status quo)

Keep ADR-0058 §Clause 2's β posture: first `consumer.Commit`
failure → warning-log + skip → rely on transitive-commit
recovery or restart-based replay.

- **Strengths.** Zero added complexity; the failure path is one
  branch; the consumer poll loop never stalls beyond a single
  Commit RPC's RTT.
- **Weaknesses.** Every transient broker hiccup costs a full
  window's commit boundary. Operationally costly for the
  common transient-failure case; alarm fatigue on
  consumer-lag dashboards wired to broker-side committed
  offset. Defers the OQ-3 deferral indefinitely without a
  trigger criterion.

### Option B — Fixed-delay retry

Retry N times with a constant delay between attempts (e.g.,
3 attempts, 200ms each).

- **Strengths.** Simpler to reason about than exponential
  back-off. Predictable maximum wait (N × delay).
- **Weaknesses.** Two pathologies. (1) Synchronized retries on
  broker recovery — N concurrent runners hammer the recovering
  broker at exactly the same intervals. (2) Wrong-shape for
  the failure-mode distribution: a 200ms-after-fail-1 retry
  isn't long enough for a 500ms broker pause; a 500ms-fixed
  retry is too long for a 50ms TCP-RST blip.

### Option C — Exponential back-off with uniform-random jitter (Recommended)

Retry up to `max_attempts` times. The delay between attempts is
computed as `random_uniform(0, base × 2^attempt)` where
`attempt` counts back-offs from 1 (first back-off uses
`attempt = 1`, second uses `attempt = 2`, …). The upper bound
of the random window grows exponentially per attempt; the
realized delay is uniform within that window — this
de-synchronizes retries from concurrent runners against a
recovering broker.

For β: **`max_attempts = 3`** (1 initial + 2 retries),
**`base = 100ms`**. Worst-case wait is the sum of the two
back-off windows' upper bounds: `(base × 2^1) + (base × 2^2) =
200ms + 400ms = 600ms`. Expected wait under uniform random
jitter is half of that: **`~300ms`**.

- **Strengths.** The exponentially-widening random window
  matches the failure-mode distribution shape (most transients
  resolve in <100ms; tail cases need progressively longer
  waits). Jitter de-synchronizes retries from concurrent
  runners against a recovering broker. Total wait is bounded
  to `600ms` worst-case — small enough to keep the
  single-goroutine runner's forward progress reasonable.
- **Weaknesses.** More complex than fixed delay. The β
  parameter values (`3 / 100ms`) are judgment calls — not
  derived from operational signal because no operational
  signal exists yet (ADR-0058 shipped 24 hours ago). Operator
  may need to tune later if production telemetry reveals
  different transient-failure shape.

---

## Recommendation

**Option C** with β parameter values **`max_attempts = 3`,
`base = 100ms`, uniform-random jitter**.

The choice is grounded in:

- **DD-1** — An exponentially-widening random window matches
  the transient-failure distribution shape better than fixed
  delay (most transients resolve fast; tail cases need
  progressively longer waits).
- **DD-2** — A `600ms` worst-case stall is the largest the
  single-goroutine runner can absorb without consuming the
  poll loop's forward progress; the bound is small enough
  that a one-off transient blip is invisible to operators
  (no observable stall), while a sustained broker failure
  surfaces predictably (every commit attempt hits the bound
  before falling through to ADR-0058 §Clause 2). A more
  precise multiplier against typical poll-batch processing
  time waits for operational signal (OQ-6).
- **DD-3** — Uniform-random jitter inside an exponentially-
  growing window de-synchronizes retries between concurrent
  runners against a recovering broker. Without jitter, N
  concurrent runners would retry at exactly the same intervals
  and re-overwhelm the broker on first recovery; the random
  draw spreads them across the back-off window.
- **DD-4** — The loop checks `ctx.Done()` between attempts via
  a `select { case <-ctx.Done() ... case <-time.After(delay) ... }`
  pattern; engine shutdown breaks the retry loop within the
  current attempt's back-off window.
- **DD-5** — After `max_attempts` failures, the runner returns
  to ADR-0058 §Clause 2's warning-log + skip path verbatim;
  the retry layer is additive, not replacing.

**Reading carried forward as new contribution proposed here,
requires review (R5) — see eligibility Condition 1 in Context.**
The "retry layer fills OQ-3's deferred-implementation space" vs
"retry layer modifies §Clause 2's decided shape" reading is
operator-ratifiable; precedent disposition lives in ADR-0055
§Notes, ADR-0056 §Notes, and ADR-0058 §Notes per
adr-writing A7.

### What this study commits

- **Decision shape**: exponential back-off with uniform-random
  jitter, β parameters `max_attempts = 3`, `base = 100ms`.
- **Implementation site**: a new `commitWithRetry` helper in
  `engine/internal/runner/record_runner.go` invoked from
  `closeAndDispatch` in place of the direct `consumer.Commit`
  call. The helper owns the retry loop, the jitter math, and
  the context-cancellation check; `consumer.Commit` is
  invoked once per attempt.
- **Configuration shape**: package-level `const` values for
  the two knobs (`recordCommitMaxAttempts`,
  `recordCommitBackoffBase`). β posture commits the values in
  code; operator-tunable knobs (env var, runner-config field)
  are OQ-1 below.
- **Error classification**: all errors are retryable in β. No
  transient-vs-permanent discrimination (OQ-3 below).
- **Terminal behavior preserved**: after retry exhaustion, the
  runner falls through to ADR-0058 §Clause 2's warning-log +
  skip path verbatim. The runner-side log line gains a count
  of retry attempts so operators can observe the retry budget
  exhausting (OQ-4 covers metric-side observability).
- **Test discipline**: extend `fakeConsumer` with a
  configurable failure sequence (fail-N-times-then-succeed);
  three new tests covering eventual-success-after-retry,
  exhaustion-falls-through-to-warning, and context-cancellation-
  during-retry.
- **Jitter source**: `math/rand/v2`'s package-level
  `rand.Float64()` — the runner does not need cryptographic-
  quality randomness for retry jitter; `math/rand/v2` is the
  modern Go standard for non-crypto random.

### What this study does NOT commit

- A change to `RecordConsumer` interface. The retry layer
  lives above `consumer.Commit`; the consumer's contract is
  one Commit RPC per call.
- A transient-vs-permanent error classification. OQ-3 below.
- Operator-tunable knobs for the three β parameters (env var
  / runner-config field). OQ-2 below.
- A retry-related observability metric. OQ-4 below; folds
  into the same emission-slice scoping question ADR-0058 OQ-4
  raised.
- A concurrent runner posture (multi-goroutine consumer).
  OQ-1 from B3-6 / ADR-0058 stands.
- An operator-rerun path for record-mode replay. ADR-0024
  §Notes deferral stands.

---

## Consequences

1. **B3-7 reaches `resolved-study` and promotes to ADR-0059 in
   the same session under operator-authorized R4 scope
   collapse** (precedent ADR-0054 / 0055 / 0056 / 0058 §Notes).
   The promotion ADR commits the decision shape, the β
   parameter values, the implementation site, and the test
   discipline.

2. **`closeAndDispatch` now invokes `commitWithRetry`** in
   place of the direct `consumer.Commit` call. The retry
   helper owns the loop, the jitter math, and the
   context-cancellation check. On final failure (retries
   exhausted) the helper returns the last error; the runner
   then warning-logs and skips per ADR-0058 §Clause 2.

3. **Worst-case runner stall bounded by 600ms on commit
   failure.** First attempt fails immediately; back-off 1
   waits up to `200ms` (`random(0, base × 2^1)`); second retry
   attempt fails; back-off 2 waits up to `400ms` (`random(0,
   base × 2^2)`); third attempt fails and the runner falls
   through to ADR-0058 §Clause 2's warning-log + skip path.
   Expected wait under uniform-random jitter is `~150ms`
   (half each window's upper bound). Typical case (transient
   resolves on first retry) waits `~100ms` expected (uniform
   random within `[0, 200ms]` window).

4. **No interface or substrate-coupling change.**
   `RecordConsumer.Commit`'s signature is unchanged; the
   `FranzConsumer.Commit` implementation is unchanged; the
   retry layer is package-internal to the runner.

5. **ADR-0058 §Clause 2 terminal behavior preserved verbatim.**
   The retry layer is *additive on top of* the existing
   terminal; the warning-log + skip + transitive-commit
   recovery path remains the runtime's fallback after retry
   exhaustion. ADR-0058 itself is not reopened.

6. **The runner-side warning log line gains a `commit_attempts`
   field.** When the retry budget exhausts, the warning
   includes the count of attempts made so an operator reading
   the log can distinguish "retried and exhausted" from
   "single-attempt failure". The field name follows the
   `engine/internal/logging/` per-package convention per
   ADR-0043.

7. **No new dependency.** `math/rand/v2` is stdlib in Go 1.22+;
   the implementation slice in this PR verifies the engine's
   Go version is ≥ 1.22 (via `engine/go.mod`'s `go` directive)
   before importing `math/rand/v2`. No third-party retry
   library is introduced.

8. **`record-mode-conventions` skill convention S2 updated**
   in the same PR to cite ADR-0059's retry envelope on top of
   ADR-0058's commit-after-dispatch posture. Light-touch update
   per ADR-0053's framing — S2's commit-after-dispatch rule
   wording stays as the spine; a retry-envelope paragraph is
   appended with new `record_runner.go` citations covering the
   retry helper and the two β constants.

9. **No B-row backlog amendment.** B3-7 reaches `resolved-adr`
   via this ADR's promotion; no other B-row's row in
   `studies/foundation/06-decision-log.md` is touched.

10. **PR-flow per CONTRIBUTING.md Flow 5 with R4 scope-collapse
    trailer.** Single PR carries study + critique capture +
    ADR + implementation + tests + decision-log update.

---

## Open Questions

- **OQ-1: Operator-tunable retry parameters.** The two β
  constants (`max_attempts = 3`, `base = 100ms`) are committed
  in code. If production operational signal reveals a
  different optimal shape (e.g., longer back-off because
  broker hiccups are typically 1–5s, or fewer attempts because
  the runner stall is concerning), promoting one or more to
  operator-tunable knobs (env var per ADR-0018 EnvConfig, or
  runner-config field) is the natural next slice.
  **Out-of-scope for current cycle** — β posture commits
  constants; tuning waits for operational signal.

- **OQ-2: Transient-vs-permanent error classification.** β
  retries all errors. A permanent error (e.g., authentication
  failure, schema mismatch) wastes the full retry budget
  before falling through. The Kafka client library's error
  types could classify some failures as non-retryable
  (e.g., `kerr.IsRetriable` from franz-go), but the
  classification is substrate-specific and would need to be
  exposed through the substrate-agnostic `RecordConsumer`
  interface to be usable in the runner. **Out-of-scope for
  current cycle** — substrate-agnostic error classification
  is a separate design surface; β accepts the wasted-budget
  cost on permanent errors.

- **OQ-3: Retry-attempt observability metric.** A
  `dq_record_commit_retries_total` counter (per
  outcome=success-after-retry vs exhausted) would let
  operators dashboard the retry rate without grepping logs.
  ADR-0058 OQ-4 raised a parallel question about
  `dq_record_commit_failures_total`; both are emission-side
  questions. **Out-of-scope for current cycle** — folds into
  a follow-up emission-slice session when ADR-0039's inventory
  is re-scoped.

- **OQ-4: Per-attempt timeout for `consumer.Commit`.** The
  current `consumer.Commit` invocation is bounded only by the
  parent context. A flaky broker that *hangs* (rather than
  errors) consumes the parent context's full deadline on a
  single attempt; subsequent retries never fire. A
  per-attempt timeout (e.g., 5s child context) would bound
  each attempt and let the retry budget be consumed even on
  hang. **Out-of-scope for current cycle** — operational
  signal will reveal whether broker-hang is a realistic
  failure mode; β trusts the parent context's deadline.

- **OQ-5: Total retry budget across a session.** The β
  policy retries per-Commit-call; a runner that fails N
  successive commits each consumes its own retry budget. A
  session-level circuit breaker (e.g., "after K consecutive
  Commit failures, stop retrying for M seconds") could
  protect a broker in extended outage. **Out-of-scope for
  current cycle** — per-call retry is the conservative shape;
  session-level state adds surface area that needs its own
  design pass.

- **OQ-6: Quantitative stall-budget calibration.** DD-2
  defends the `600ms` worst-case stall qualitatively (small
  relative to typical poll-batch processing time). A
  measurement-driven calibration — comparing the worst-case
  retry stall against observed poll-batch processing time
  under production load — would let the operator decide
  whether the bound is tight enough or needs adjustment via
  OQ-1's tunable-knob path. **Out-of-scope for current
  cycle** — requires production telemetry that does not exist
  at this ADR's promotion time (ADR-0058 shipped 24 hours
  ago; no production runner has emitted commit-RPC duration
  histograms yet). The calibration session lands when the
  emission slice (ADR-0055 / ADR-0056 / a future B3-N) wires
  a commit-RPC histogram metric and a sufficient observation
  window has accumulated.

---

## Promotion target

[`docs/adr/0059-record-runner-commit-retry.md`](../../docs/adr/0059-record-runner-commit-retry.md)
(reserved per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7; ADR-0058 merged 2026-05-30 at PR #121).
