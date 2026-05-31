<!-- path: docs/adr/0059-record-runner-commit-retry.md -->

# ADR-0059 — Record Runner Commit Retry Policy

- **Status:** accepted
- **Date:** 2026-05-31

---

## Context

[ADR-0058](./0058-record-runner-commit-after-dispatch.md) (promoted
2026-05-30) committed the record-mode runner's commit-after-dispatch
posture: per-trigger commit keyed on `dispatcher.Run` success, with
`consumer.Commit` failures "warning-logged and do not propagate;
the next successful dispatch in the same partition commits the
uncommitted records transitively via high-water-mark monotonicity"
(ADR-0058 §Clause 2).

ADR-0058 §Notes recorded OQ-3 verbatim: "β posture on
`consumer.Commit` returning non-nil is warning-log + skip … A
bounded retry (e.g., three attempts with exponential back-off)
might be operationally better when broker connectivity is flaky.
Out-of-scope for current cycle — β posture is conservative-safe;
operational signal will reveal whether retries are needed." This
ADR consumes OQ-3 and commits a bounded retry policy.

Under ADR-0058's β posture, a single transient broker hiccup at
commit time (leader-election blip, TCP RST on an idle connection,
brief fsync pause) costs the runner one full window's commit
boundary. The records remain uncommitted; transitive-commit
recovery requires the next window's successful dispatch in the
same partition. A topic with sparse traffic in the affected
partition can leave the window's records uncommitted for an
arbitrarily long stretch. Restart-based recovery remains correct
(at-least-once safety per ADR-0058 + ADR-0003 §1+§2) but
operationally heavy. The β posture optimizes for simplicity at
the cost of transient-blip tolerance; a bounded retry layer
shifts the trade-off in the common transient case.

This ADR is the **promotion of B3-7** under the post-Wave-3
evolutionary lane committed by
[ADR-0049](./0049-b3-evolutionary-launch.md). B3-7 routes through
the **capability mode extensions** family per ADR-0049
§"Per-family scope" — same family as B3-6 / ADR-0058.
Eligibility on Conditions 2, 3, and 4 clears cleanly; Condition 1
(P-B3.1 expands not rewrites) is borderline because inserting a
retry loop between `closeAndDispatch` and `consumer.Commit` could
be read as modifying ADR-0058 §Clause 2's decided shape
("commit failures are warning-logged" → "commit failures are
retried-then-warning-logged"). The reading committed here is that
the retry layer is *evolution against an authorized OQ deferral*
(ADR-0058 OQ-3 explicitly named bounded retry as
deferred-implementation framing), preserving §Clause 2's
**terminal** behavior verbatim after retries are exhausted. Per
`CLAUDE.md` R5 + A7 of the `adr-writing` skill, the reading is
recorded here as **new contribution requiring review** and is
reviewed in this ADR. Precedent disposition for borderline B3
eligibility readings lives in
[ADR-0055](./0055-metric-emission-slice-scope.md) §Notes,
[ADR-0056](./0056-panel-5-lighting-slice.md) §Notes, and
[ADR-0058](./0058-record-runner-commit-after-dispatch.md) §Notes.

This ADR also lands its implementation slice in the same PR per
an **operator-authorized R4 scope collapse**, precedent
[ADR-0054](./0054-engine-image-registry-amendment.md) §Notes,
[ADR-0055](./0055-metric-emission-slice-scope.md) §Notes,
[ADR-0056](./0056-panel-5-lighting-slice.md) §Notes, and
[ADR-0058](./0058-record-runner-commit-after-dispatch.md) §Notes.

The principles bearing on the decision are **P2** (deterministic
behavior — retries are stateless from one Commit-call to the
next; the random draw is jitter, not control state), **P4** (cost
is first-class — the worst-case stall budget is the load-bearing
operator-facing knob; constants in code commit a conservative
shape that operational signal can tune via OQ-1), and **P5**
(evolution is contract-driven — the retry layer is additive on
top of ADR-0058's commit boundary; no contract reshape).

---

## Decision

The slice is committed in seven clauses (algorithm, parameters,
implementation site, jitter source, terminal behavior, test
discipline, skill-side update), plus a Notes block that records
the R4 scope-collapse rationale and the ratified Condition-1
reading per R5 + A7.

### Clause 1 — Algorithm: exponential back-off with uniform-random jitter

The runner retries `consumer.Commit` up to `max_attempts` times.
The delay between attempts is computed as
`random_uniform(0, base × 2^attempt)`, where `attempt` counts
back-offs from 1 (the first back-off, between the initial attempt
and the first retry, uses `attempt = 1`; the second back-off uses
`attempt = 2`; …). The upper bound of the random window grows
exponentially per back-off; the realized delay is uniform within
that window.

The exponentially-widening random window matches the
transient-failure distribution shape (most transients resolve
fast; tail cases need progressively longer waits). Uniform-random
jitter inside the window de-synchronizes retries between
concurrent runners against a recovering broker — without jitter,
N concurrent runners would retry at exactly the same intervals
and re-overwhelm the broker on first recovery.

### Clause 2 — β parameters: `max_attempts = 3`, `base = 100ms`

Two package-level `const` values in
`engine/internal/runner/record_runner.go`:

- `recordCommitMaxAttempts = 3` (1 initial + 2 retries)
- `recordCommitBackoffBase = 100 * time.Millisecond`

Worst-case wait: `(base × 2^1) + (base × 2^2) = 200ms + 400ms =
600ms`. Expected wait under uniform-random jitter: `~150ms`. No
cap value is committed — at the chosen parameters, the maximum
back-off window upper bound is `400ms` (back-off 2), which a cap
would not constrain.

The values are judgment calls grounded in DD-1, DD-2, DD-3 of
the originating study; they are not derived from operational
signal because no operational signal exists yet (ADR-0058 shipped
2026-05-30; no production runner has emitted commit-RPC duration
histograms yet). Operator-tunable knobs (env var per
[ADR-0018](./0018-environment-configuration-model.md) EnvConfig,
or runner-config field) are deferred to a future B3-N when
operational signal motivates tuning.

### Clause 3 — Implementation site: `commitWithRetry` helper in `record_runner.go`

A new package-private helper `commitWithRetry(ctx, consumer,
records, logger)` owns the retry loop, the jitter math, and the
context-cancellation check. `closeAndDispatch` invokes
`commitWithRetry` in place of the direct `consumer.Commit` call
committed by ADR-0058 §Clause 2.

The helper's loop shape:

```
for attempt := 1; attempt <= recordCommitMaxAttempts; attempt++ {
    err := consumer.Commit(ctx, records)
    if err == nil {
        return nil
    }
    if attempt == recordCommitMaxAttempts {
        return err   // caller warning-logs per Clause 5
    }
    delay := time.Duration(rand.Float64() *
        float64(recordCommitBackoffBase << attempt))
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(delay):
    }
}
```

(The `<< attempt` is the exponential growth: `base × 2^attempt`.
`rand.Float64()` returns `[0.0, 1.0)`; multiplying by the window
upper bound yields uniform-random within the window.)

### Clause 4 — Jitter source: `math/rand/v2`

The package-level `rand.Float64()` from `math/rand/v2` is the
jitter source. The runner does not need cryptographic-quality
randomness for retry jitter; `math/rand/v2` is the modern Go
standard for non-crypto random and is stdlib in Go 1.22+. The
implementation slice verifies the engine's Go version is ≥ 1.22
(via `engine/go.mod`'s `go` directive) before importing the
package.

No third-party retry library is introduced.

### Clause 5 — Terminal behavior: ADR-0058 §Clause 2 preserved

After `commitWithRetry` returns non-nil (retry budget exhausted
*or* context cancellation), `closeAndDispatch` falls through to
ADR-0058 §Clause 2's warning-log + skip path verbatim. The
runner-side warning log line gains a `commit_attempts` integer
field carrying the count of attempts made, so an operator
reading the log can distinguish "retried and exhausted" from
"single-attempt failure":

```
r.logger.Warn("record window commit failed",
    "entity", state.source.Entity,
    "window_start", w.start.UTC(),
    "error", err.Error(),
    "commit_attempts", recordCommitMaxAttempts,
    "adr_reference", "ADR-0058+ADR-0059",
)
```

(Context-cancellation returns from `commitWithRetry` without
warning-logging since shutdown is operator-driven, not a
failure mode; the existing logger.Info "record runner shutting
down" emission in `Start` is the operator-visible signal.)

ADR-0058 §Clause 2's transitive-commit recovery (next successful
dispatch in the same partition commits the uncommitted records
via high-water-mark monotonicity) and restart-based recovery
(re-flow on engine restart) continue to apply unchanged.

### Clause 6 — Test discipline: extend the fake consumer with a failure sequence

`fakeConsumer` in `engine/internal/runner/record_runner_test.go`
gains a `commitFailureSequence []error` field. When set, the
first N calls to `Commit` return the configured errors in
order; subsequent calls return nil and append records to the
existing ledger. The existing `commitErr` field (set globally on
the consumer) becomes a shorthand for `commitFailureSequence`
that returns the same error indefinitely; the two fields are
independent to keep existing tests (which use `commitErr`)
unchanged.

Three new tests cover the material paths:

- `TestRecordRunner_CommitRetryEventualSuccess` — Commit fails
  twice with a transient error, succeeds on the third attempt.
  Asserts the ledger contains the records and no warning is
  escalated past the helper.
- `TestRecordRunner_CommitRetryExhaustion` — Commit fails on
  all three attempts. Asserts the ledger is empty (no commit
  records) and the helper returns the final error.
- `TestRecordRunner_CommitRetryRespectsContext` — Commit fails
  on the first attempt; the test cancels the context during
  the back-off window. Asserts the helper returns promptly
  (without waiting for the back-off to elapse) and the ledger
  is empty.

### Clause 7 — `record-mode-conventions` skill convention S2 updated

The `record-mode-conventions` skill at
`.claude/skills/record-mode-conventions/SKILL.md` and its
`reference/conventions.md` are updated in the same PR. S2's
commit-after-dispatch rule wording stays as the spine; a retry-
envelope paragraph is appended with new citations to
`record_runner.go`'s `commitWithRetry` helper and the two β
constants. Light-touch update per
[ADR-0053](./0053-record-mode-skill.md)'s framing.

---

## Consequences

1. **Record-mode runtime gains bounded commit retries on
   transient broker failure.** A single transient hiccup
   (resolving in <600ms worst-case) no longer costs the runner
   a full window's commit boundary; the retry layer absorbs it
   within the same `closeAndDispatch` invocation.

2. **Worst-case runner stall bounded by 600ms on commit
   failure.** First attempt fails immediately; back-off 1 waits
   up to `200ms` (`random(0, 200ms)`); second retry attempt
   fails; back-off 2 waits up to `400ms` (`random(0, 400ms)`);
   third attempt fails and the runner falls through to
   ADR-0058 §Clause 2's warning-log + skip path. Expected wait
   under uniform-random jitter: `~150ms`. Typical case
   (transient resolves on first retry): `~100ms` expected.

3. **Context cancellation breaks the retry loop within the
   current back-off window.** Engine shutdown via `ctx.Done()`
   pre-empts the `time.After` channel in the
   `commitWithRetry` `select` statement; the helper returns
   `ctx.Err()` within the back-off interval rather than
   completing the full retry budget. Shutdown is bounded by
   one back-off window's upper bound (`400ms` maximum at the
   chosen β parameters).

4. **ADR-0058 §Clause 2 terminal behavior preserved verbatim.**
   After the retry budget is exhausted, the runner returns to
   the warning-log + skip path; transitive-commit recovery and
   restart-based recovery continue to apply unchanged. The
   retry layer is additive on top of the existing terminal,
   not replacing.

5. **`RecordConsumer` interface signature is unchanged.** The
   retry layer is package-internal to the runner; the consumer
   contract remains "one Commit RPC per call." No tests of
   `FranzConsumer` semantics need updating; the existing
   `fakeConsumer.Commit` continues to satisfy the interface.

6. **The runner-side warning log line gains a
   `commit_attempts` field.** When the retry budget exhausts,
   the warning carries the count of attempts so operators can
   distinguish retried-and-exhausted from single-attempt
   failure. Field name follows the `engine/internal/logging/`
   per-package convention per
   [ADR-0043](./0043-logging-contract-specifics.md).

7. **No new dependency.** `math/rand/v2` is stdlib in Go 1.22+;
   the implementation slice verifies the engine's Go version
   is ≥ 1.22 before importing the package. No third-party
   retry library is introduced.

8. **All errors are retryable in β.** No transient-vs-permanent
   classification at the substrate-agnostic `RecordConsumer`
   interface layer; a permanent error consumes the full retry
   budget before falling through. Operational cost is `600ms`
   per permanently-failing Commit call — acceptable in β;
   error-classification as a substrate-agnostic predicate is
   a separate design surface (Notes OQ-2 carry-forward).

9. **`record-mode-conventions` skill convention S2 stays
   accurate.** The skill's S2 rule wording is appended with a
   retry-envelope paragraph; the rest of the seven conventions
   (S1, S3–S7) are unaffected.

10. **No B-row backlog amendment.** B3-7 reaches `resolved-adr`
    via this ADR's promotion; no other B-row's row in
    `studies/foundation/06-decision-log.md` is touched.

11. **PR-flow per `CONTRIBUTING.md` Flow 5 with the R4 scope-
    collapse trailer.** The single PR carries the study, the
    round-1 critique capture, this ADR, the implementation,
    the tests, and the decision-log update.

---

## Notes

- **R4 scope-collapse rationale.** This ADR's promotion +
  implementation slice land in a single PR per operator-
  authorized R4 scope collapse. Precedent:
  [ADR-0054](./0054-engine-image-registry-amendment.md) §Notes
  introduced the pattern at promotion time;
  [ADR-0055](./0055-metric-emission-slice-scope.md) §Notes,
  [ADR-0056](./0056-panel-5-lighting-slice.md) §Notes, and
  [ADR-0058](./0058-record-runner-commit-after-dispatch.md)
  §Notes carried it forward. The collapse is appropriate here
  because the implementation slice is small (one helper
  function, two constants, three new tests with a tightly
  scoped `fakeConsumer` extension, one β-marker absent — no
  legacy posture being retired — one skill convention
  paragraph) and the structural decisions (algorithm,
  parameters, implementation site, jitter source, terminal
  behavior) are load-bearing in the ADR; separating them
  would split a single cohesive change across two sessions
  without reducing review load.

- **Borderline reading on ADR-0049 §(a) Condition 1, ratified
  per R5 + A7.** The reading committed is that this slice is
  "evolution against an authorized OQ deferral" (capability-
  mode-extension family) rather than "amendment of ADR-0058
  §Clause 2's decided shape." The reading is borderline
  because inserting a retry loop between `closeAndDispatch`
  and `consumer.Commit` could be read as modifying §Clause 2
  ("commit failures are warning-logged" → "commit failures
  are retried-then-warning-logged"). The capability-mode-
  extension reading is supported by:
  (a) ADR-0058 OQ-3 explicitly named bounded retry as
  "deferred-implementation framing" — that's the open OQ
  decision space the new ADR fills, not the decided shape of
  §Clause 2;
  (b) the **terminal** behavior of §Clause 2 (warning-log +
  skip + transitive-commit recovery) is preserved verbatim
  after the retry budget is exhausted;
  (c) Clause 5 above commits the terminal-preservation
  explicitly so a future reader can verify the additive-on-
  top-of framing.
  Precedent disposition for parallel borderline readings:
  ADR-0058 Condition 1 (kafka_consumer.go β-marker vs ADR-open-
  space), ADR-0055 Condition 2 (engine-runtime emission as
  Tooling-extensions), ADR-0056 Conditions 1+3 (panel-5
  weak-reading + A.y path). The reading is operator-ratifiable
  at merge time per `CONTRIBUTING.md` Flow 5 §"Operator-side
  responsibilities".

- **Open Questions carried forward from the study (B3-7, OQ-1
  through OQ-6).** Each is out-of-scope for this ADR per
  operator-authorized framing and the study's AC-6
  disposition; the most operationally adjacent are OQ-1
  (operator-tunable parameters) and OQ-6 (quantitative
  stall-budget calibration once production telemetry exists).
  None of the OQs introduce a blocking dependency on this
  ADR; they extend the capability-mode-extension surface in
  future B3-N entries when concrete demand surfaces.

- **DD-2's qualitative bound deliberately under-specified.**
  The originating study's first draft asserted "typical poll-
  batch processing is 10–100ms; the 1.6s worst-case is 16–160×
  that" — both numbers were author intuition without
  grounding, and the 1.6s figure was mathematically incorrect
  (the cap value the draft committed was dead code at the
  chosen parameters; the worst case is actually 600ms). The
  round-1 critique surfaced the inconsistency; the disposition
  was to (a) drop the cap (Option (c) in the critique's
  fix-set), (b) correct the math throughout, and (c) move
  quantitative comparison to OQ-6 pending operational signal.
  This Note records the math correction so a future reader
  understands the parameter shape was deliberately simplified
  rather than expanded.
