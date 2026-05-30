<!-- path: docs/adr/0058-record-runner-commit-after-dispatch.md -->

# ADR-0058 — Record Runner Commit-After-Dispatch

- **Status:** accepted
- **Date:** 2026-05-30

---

## Context

[ADR-0024](./0024-window-semantics.md) committed watermark-bounded
tumbling windows for record-mode rules and bound the `execution_id`
shape to [ADR-0002](./0002-run-identity-and-idempotency.md)'s
five-input formula with `trigger_source = stream-watermark`.
Determinism is committed at the windowing layer: same input stream
+ same ruleset version + same window configuration produce the
same `execution_id` set, with offset-range replay producing
identical rows. ADR-0024 did **not** decide *when* the record-mode
runner commits consumer-group offsets back to the broker; the
question was left to implementation.

The Wave-S β implementation chose **commit-after-fetch**: the
Kafka consumer's `PollFetches` returns the batch and synchronously
commits offsets before returning to the runner. A β-marker
comment in `engine/internal/runner/kafka_consumer.go` flagged the
gap and named the target posture verbatim:

> β semantics: commit-after-fetch (at-most-once on a crash
> mid-dispatch). Per-attempt re-read of offset ranges
> (ADR-0024) is a future slice; that future slice replaces this
> with a commit-after-dispatch flow keyed on the trigger's
> successful return.

Under the β posture, the broker advances the consumer-group
committed offset the moment `PollFetches` returns. If the engine
crashes between `PollFetches` returning and `dispatcher.Run`
returning nil for a closed window, the records that fed the
window are lost: the broker treats them as consumed; the engine
never wrote a `dq_executions` row for them. The
determinism / replayability promise of ADR-0002 + ADR-0024 is
silently weakened — operators reading the ADR cannot tell from
the ADR alone that mid-dispatch crashes drop data.

This ADR is the **promotion of B3-6** under the post-Wave-3
evolutionary lane committed by
[ADR-0049](./0049-b3-evolutionary-launch.md). B3-6 routes through
the **capability mode extensions** family per ADR-0049
§"Per-family scope" ("a refinement to how capability is
dispatched at runtime"). Eligibility on Conditions 2, 3, and 4
clears cleanly; Condition 1 (P-B3.1 expands not rewrites) is
borderline because one reader might argue the change is
"completing a deferred slice of ADR-0024" rather than "evolution
against ADR-0024's open decision space". The reading is borderline
because ADR-0024 itself never decided commit timing (the β-marker
is an implementation note, not an ADR clause) and the runner's
user-observable capability remains the same shape after the
change — the change is to *delivery posture*, which is the
canonical capability-mode-extension surface. Per `CLAUDE.md` R5 +
A7 of the `adr-writing` skill, the reading is recorded here as
**new contribution requiring review** and is reviewed in this
ADR. Precedent disposition for borderline B3 eligibility readings
lives in [ADR-0055](./0055-metric-emission-slice-scope.md) §Notes
and [ADR-0056](./0056-panel-5-lighting-slice.md) §Notes.

This ADR also lands its implementation slice in the same PR per
an **operator-authorized R4 scope collapse**, precedent
[ADR-0054](./0054-engine-image-registry-amendment.md) §Notes,
[ADR-0055](./0055-metric-emission-slice-scope.md) §Notes, and
[ADR-0056](./0056-panel-5-lighting-slice.md) §Notes ("operator-
authorized R4 scope collapse at promotion time").

The principles bearing on the decision are **P1** (rules remain
declarative — no DSL surface is touched; the change is to engine
runtime alone), **P2** (deterministic behavior — same input
stream + same configuration produce the same windows and the
same `execution_id` set under ADR-0024; the commit boundary
moves but the determinism contract is preserved), **P3**
(ownership is explicit — the consumer-group identity that owns
the committed offset is unchanged; the per-(entity,
consumer_group) ownership shape ADR-0024 committed continues),
**P4** (cost is first-class — the commit RPC rate is bounded by
the window-close rate; concrete bound below in Consequences),
**P5** (evolution is contract-driven — the slice extends
ADR-0003 §1's append-only writes + §2's canonical-view collapse
without changing their shape), and **P6** (borrow patterns, not
baggage — at-least-once-with-canonical-view-collapse is committed
on this platform's terms, not by importing it whole).

---

## Decision

The slice is committed in six clauses (interface signature,
runner-side discipline, Kafka-client implementation, β-marker
retirement, test discipline, skill-side update), plus a Notes
block that records the R4 scope collapse rationale and the
ratified Condition-1 reading per R5 + A7.

### Clause 1 — `RecordConsumer` interface gains a Commit method

The substrate-agnostic consumer interface in
`engine/internal/runner/record_runner.go` extends to three
methods:

```go
type RecordConsumer interface {
    PollFetches(ctx context.Context) ([]FetchedRecord, error)
    Commit(ctx context.Context, records []FetchedRecord) error
    Close()
}
```

The `records` argument carries `(Topic, Partition, Offset)` per
record. The interface keeps the substrate-agnostic shape Wave-S
β shipped: tests inject fakes that implement all three methods;
the engine binary wires `*FranzConsumer` (Clause 3) which
satisfies the interface transparently.

### Clause 2 — Runner-side discipline: per-trigger commit on dispatcher's successful return

Each closed window's `dispatcher.Run(ctx, trigger)` success is
its own commit boundary. After `closeAndDispatch` observes
`dispatcher.Run` returning nil, the runner calls
`consumer.Commit(ctx, fetched)` passing the `FetchedRecord`
slice that fed the window's `recordWindow`. On dispatch failure,
the commit is skipped; the records remain uncommitted in the
broker; on engine restart, they re-flow and re-populate the same
window deterministically per ADR-0024's watermark logic, and
ADR-0002's deterministic `execution_id` formula produces the
same identifier.

Records-in-still-open-windows are **not** committed: the active
window has not closed; no `dispatcher.Run` has been called for
it; `consumer.Commit` is not invoked. On crash, those records
are uncommitted in the broker. On restart, they re-flow and
re-populate the same active window per ADR-0024.

The runner carries the `FetchedRecord` slice alongside the
existing `Record` slice on `recordWindow` (a parallel field, not
a reshape of `TriggerRequest.Records []Record`). `Record` is
consumed by per-kind evaluators per
[ADR-0026](./0026-record-mode-evidence.md) evidence shape;
reshaping it would ripple into evaluator assertions outside this
slice's scope. The parallel-field choice is the minimal-blast-
radius shape.

Commit failures are warning-logged and do not propagate. The
records remain uncommitted in the broker; the next successful
dispatch in the same partition commits transitively via the
high-water-mark monotonicity (a partition's committed offset is
the max-per-partition of all records passed to Commit so far).
A bounded retry / back-off policy is deferred to a separate
decision (see Notes OQ-3 carry-forward).

### Clause 3 — `FranzConsumer.Commit` uses `MarkCommitRecords` + `CommitMarkedOffsets`

The franz-go-backed `FranzConsumer` in
`engine/internal/runner/kafka_consumer.go` implements
`Commit(ctx, records)` by translating each `FetchedRecord` to a
`*kgo.Record` carrying `Topic / Partition / Offset`, calling
`client.MarkCommitRecords(...)`, then
`client.CommitMarkedOffsets(ctx)`. The Kafka client library's
`MarkCommitRecords` primitive tracks the high-water mark per
partition internally; `CommitMarkedOffsets` flushes the marks
to the broker synchronously. Auto-commit remains disabled (per
the existing `kgo.DisableAutoCommit()` option); the runner is
the sole commit authority.

The `Commit` call is synchronous (one round-trip per closed
window per entity); failure surfaces as a non-nil error the
runner warning-logs and skips. The cost discipline is bounded
per Consequence 5.

### Clause 4 — β-marker comment retirement in `kafka_consumer.go`

The β-marker comments at the top of `FranzConsumer` (the
"future slice replaces this" wording) and inside `PollFetches`
(the "commit-after-fetch (at-most-once on a crash mid-dispatch)"
wording) are rewritten to reflect the post-ADR-0058 posture.
`PollFetches` no longer commits offsets; the inline
`client.CommitUncommittedOffsets` call is removed. The
type-level doc-comment on `FranzConsumer` records the new
posture (commit-after-dispatch keyed on the runner's per-trigger
Commit call) and points to this ADR.

### Clause 5 — Test discipline: extend the existing fake consumer

The Wave-S β test surface
(`engine/internal/runner/record_runner_test.go`) is the only test
surface covering the runner's poll / commit loop. The slice
extends `fakeConsumer` and `errConsumer` with a `Commit` method
that records committed records into an in-memory ledger; two new
tests cover the material paths:

- `TestRecordRunner_CommitsAfterSuccessfulDispatch` — a batch
  whose dispatch succeeds asserts the consumer's commit ledger
  contains exactly the records of the dispatched windows.
- `TestRecordRunner_DoesNotCommitOnDispatchFailure` — a batch
  whose dispatcher returns non-nil for at least one closed
  window asserts that those window's records are not in the
  commit ledger; subsequent successful dispatches in the same
  batch may still commit (per-trigger commit, not per-batch).

All existing tests continue to pass. The franz-go-side
implementation is exercised by the engine-binary integration
suite; no franz-go is brought into the unit-test layer.

### Clause 6 — `record-mode-conventions` skill S2 update

The `record-mode-conventions` skill at
`.claude/skills/record-mode-conventions/SKILL.md` is updated in
the same PR. Convention S2 (β commit semantics) drops the
β-marker wording and points to this ADR. The skill-side update
is light-touch per [ADR-0053](./0053-record-mode-skill.md)'s
framing — the convention's rule wording reflects the new
posture; the citation moves from the kafka_consumer.go doc-
comment to this ADR's Clause 2 + Clause 3.

---

## Consequences

1. **Record-mode runtime gains at-least-once delivery
   semantics.** A crash between `PollFetches` returning and
   `dispatcher.Run` returning nil for a closed window now leaves
   the window's records uncommitted in the broker. On engine
   restart, those records re-flow and re-populate the same
   window deterministically per ADR-0024; the same window's
   `execution_id` per ADR-0002 plus ADR-0003 §1's append-only
   writes + ADR-0003 §2's canonical view (`dq_executions_current`
   collapses attempts per `execution_id` to the row with the
   latest `recorded_at`) make any spurious re-dispatch
   consumer-invisible. The base table preserves both attempts
   (different `attempt_id` per ADR-0003 §4) for forensic
   queries; the canonical view dashboards and alert systems
   target shows one row.

2. **The `RecordConsumer` interface gains one method.** All
   producers of the interface (production `*FranzConsumer`,
   test `fakeConsumer` + `errConsumer`) implement `Commit`. The
   engine binary wiring at
   `engine/cmd/dq-engine/main.go:buildRecordRunners` is
   unaffected: it constructs the concrete `*FranzConsumer`
   which satisfies the extended interface transparently.

3. **`PollFetches` no longer commits offsets.** The auto-
   commit-disabled posture continues; the runner is the sole
   commit authority. `PollFetches` returns the fetched records
   and surfaces fetch errors; the commit RPC moves to
   `closeAndDispatch`'s post-success path per Clause 2.

4. **ADR-0024 / ADR-0002 / ADR-0003 are preserved.** The decided
   shape of each is unchanged; this slice consumes them as its
   substrate. ADR-0024's windowing semantics, ADR-0002's
   `execution_id` formula and `trigger_source = stream-watermark`
   enum entry, and ADR-0003 §1 + §2's write-and-collapse model
   all stand verbatim.

5. **P4 cost bound: one commit RPC per closed window per
   entity.** At ADR-0024's minimum `window.duration: 1s` with N
   concurrent record-mode rules, the upper bound is N
   RPCs/second to the broker; at typical durations (1m–5m),
   the rate is N/minute or lower. The Kafka client library's
   commit path is local-to-the-broker (no cross-partition
   coordination), so RPC latency is the round-trip-time floor.
   Cost is P4-acceptable; no new cardinality posture is
   introduced.

6. **The β-marker comments in `kafka_consumer.go` are
   retired.** Future readers see the post-ADR-0058 posture
   without needing to trace the β-era deferral history. The
   "future slice" wording is removed; the new doc-comment
   points to this ADR.

7. **Storage cost of crash-driven retries is bounded by
   retention.** A re-dispatched window writes a second
   `dq_executions` attempt row (different `attempt_id`, same
   `execution_id`). Storage cost is one additional row per
   crash-retried window — bounded by retention since at-least-
   once retries are crash-driven, not steady-state.

8. **The `record-mode-conventions` skill (ADR-0053) stays
   accurate.** Convention S2 is updated in the same PR per
   Clause 6; the rest of the skill's seven conventions
   (S1, S3–S7) are unaffected.

9. **No B-row backlog amendment.** B3-6 reaches
   `resolved-adr` via this ADR's promotion; no other B-row's
   row in `studies/foundation/06-decision-log.md` is touched.

10. **PR-flow per `CONTRIBUTING.md` Flow 5 with the R4 scope-
    collapse trailer.** The single PR carries the study, the
    round-1 critique capture, this ADR, the implementation, the
    tests, and the decision-log update. The Operator-authorized
    R4 collapse rationale is recorded in Notes below citing the
    precedent chain.

---

## Notes

- **R4 scope-collapse rationale.** This ADR's promotion +
  implementation slice land in a single PR per operator-
  authorized R4 scope collapse. Precedent:
  [ADR-0054](./0054-engine-image-registry-amendment.md) §Notes
  introduced the pattern at promotion time;
  [ADR-0055](./0055-metric-emission-slice-scope.md) §Notes and
  [ADR-0056](./0056-panel-5-lighting-slice.md) §Notes carried it
  forward. The collapse is appropriate here because the
  implementation slice is small (one interface method, one
  Kafka-client method, one runner-side commit call, two new
  tests, one β-marker comment retirement, one skill convention
  update) and the structural decisions (interface signature,
  commit-boundary placement, FetchedRecord-carrying field) are
  load-bearing in the ADR; separating them would split a single
  cohesive change across two sessions without reducing review
  load.

- **Borderline reading on ADR-0049 §(a) Condition 1, ratified
  per R5 + A7.** The reading is that this slice is "evolution
  against ADR-0024's open decision space" (capability-mode-
  extension family, per ADR-0049 §"Per-family scope") rather
  than "completing a deferred slice of ADR-0024". The reading is
  borderline because the kafka_consumer.go β-marker can be read
  either way: as an implementation deferral inside ADR-0024's
  authority, or as a placeholder for a not-yet-decided posture
  ADR-0024 explicitly did not commit. ADR-0024 §Decision does
  not contain a commit-timing clause; the β-marker is in source
  code comments only. The capability-mode-extension reading is
  supported by the user-observable shape of the change (delivery
  posture refinement, not capability addition or removal) and by
  the precedent of ADR-0055 / ADR-0056's borderline-Condition-2
  ratifications (each carried forward the operator's reading
  via §Notes per R5 + A7). The reading is operator-ratifiable;
  the disposition is recorded here. Future B3-N items in the
  capability-mode-extension family can cite this ADR's Notes for
  similar evolution-vs-deferred-slice readings.

- **Open Questions carried forward from the study (B3-6, OQ-1
  through OQ-5).** Each is out-of-scope for this ADR per
  operator-authorized framing and the study's AC-6 disposition;
  the most operationally adjacent one is OQ-3 (commit failure
  retry / back-off policy), which surfaces if broker
  connectivity proves flaky in production. None of the OQs
  introduce a blocking dependency on this ADR; they extend the
  capability-mode-extension surface in future B3-N entries when
  concrete demand surfaces.

- **`source.type: kafka` substrate reading.** This ADR's
  commit-after-dispatch posture is substrate-specific to Kafka
  (the only record-mode substrate ADR-0023 commits in v1). A
  future record-mode substrate variant introduced under
  ADR-0023's `source.type` discriminator inherits the *runner-
  side* discipline (Clause 2: per-trigger commit on dispatcher
  success) but must carry its own consumer-side `Commit`
  implementation analogous to Clause 3. The substrate-agnostic
  interface in Clause 1 is the contract surface that admits
  variants.
