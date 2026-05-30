<!-- path: studies/decisions/2026-05-30-b3-6-record-runner-commit-after-dispatch.md -->

# B3-6 — Record Runner Commit-After-Dispatch

## Metadata

- Date: 2026-05-30
- Status: draft
- Decision-log row: B3-6 (capability mode extensions family)
- Promotion target: [`docs/adr/0058-record-runner-commit-after-dispatch.md`](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
- Critique rounds:
  - Round 1 — [capture](../critiques/2026-05-30-b3-6-record-runner-commit-after-dispatch-critique-1.md) (0 blocking / 4 important / 5 minor); 4 important applied in the revision; 2 minors applied (OQ-2 cross-reference + franz-go-mention generalization on the two non-load-bearing references); 3 minors deferred under the two-round cap.

---

## Context

[ADR-0024](../../docs/adr/0024-window-semantics.md) committed
watermark-bounded tumbling windows for record-mode rules and bound
the `execution_id` shape to ADR-0002's five-input formula with
`trigger_source = stream-watermark`. The ADR makes determinism
across the run: same input stream + same ruleset + same window
configuration → same `execution_id` set, with offset-range replay
producing identical rows. ADR-0024 does **not** decide *when* the
record-mode runner commits consumer-group offsets back to the
broker. That decision was left to implementation.

The implementation that shipped under Wave-S β
([`engine/internal/runner/kafka_consumer.go`](../../engine/internal/runner/kafka_consumer.go))
chose **commit-after-fetch**: `PollFetches` returns the batch and
synchronously invokes `client.CommitUncommittedOffsets` before
returning to the runner (lines 79–104). The β-marker comment is
explicit about the trade-off (lines 94–99):

> β semantics: commit-after-fetch (at-most-once on a crash
> mid-dispatch). Per-attempt re-read of offset ranges (ADR-0024) is
> a future slice; that future slice replaces this with a
> commit-after-dispatch flow keyed on the trigger's successful
> return.

That future slice is this study.

### Why at-most-once is not acceptable for v1

Under the current posture, the broker advances the consumer-group
committed offset the moment `PollFetches` returns. If the engine
crashes between `PollFetches` returning and `dispatcher.Run`
returning nil for a closed window, the records that fed that
window are lost: the broker thinks they were consumed; the
engine never wrote a `dq_executions` row for them.

The ADR-0024 closed-window invariant survives a crash mid-dispatch
(it is about *closed* windows; an interrupted dispatch never
reaches the close-and-write path). What does not survive is the
data: the window's records are gone, the window's
`dq_executions` row was never written, and ADR-0003's append-only
write model cannot reconstruct it from anywhere.

The platform's determinism promise — ADR-0002 §"Determinism" and
ADR-0024 §"Watermark-based closing is deterministic" — is a
*replayability* promise: same inputs reproduce same outputs.
Commit-after-fetch breaks that promise on every mid-dispatch
crash because the inputs are silently dropped from the consumer's
view of the stream.

### What commit-after-dispatch buys

If the runner commits **only after** `dispatcher.Run` returns nil
for a window, a crash mid-dispatch leaves the window's records
uncommitted on the broker. On engine restart, the Kafka consumer
client connects, reads the broker's committed offset, and
re-fetches the uncommitted records. The deterministic windowing
per ADR-0024 re-populates the same window with the same records,
the deterministic `execution_id` formula per ADR-0002 produces
the same identifier, and ADR-0003 §1's append-only writes +
ADR-0003 §2's canonical view (`dq_executions_current` collapses
attempts per `execution_id` to the row with the latest
`recorded_at`) make any spurious re-dispatch consumer-invisible:
the base table preserves both attempts (different `attempt_id`
per ADR-0003 §4), the canonical view that dashboards and alert
systems target shows one row per `execution_id`.

The composed semantics are **at-least-once delivery with
canonical-view collapse at the `execution_id` boundary**. The
"at-least-once" promise is what record-mode needs to be safe
under crashes; the canonical-view-collapse half is what makes
the "at-least" not produce duplicate rows for downstream
consumers.

### What `kafka_consumer.go` already does right

Three pieces of the runtime are pre-positioned for this change:

1. Auto-commit is **already disabled**
   ([`kafka_consumer.go:66`](../../engine/internal/runner/kafka_consumer.go)).
   The commit point is under runner control; only the timing is
   wrong.
2. The `RecordConsumer` interface
   ([`record_runner.go:44-47`](../../engine/internal/runner/record_runner.go))
   is already substrate-agnostic: `PollFetches(ctx)` returning
   `[]FetchedRecord`. Adding a commit method extends the interface
   in one direction.
3. The runner is single-goroutine by construction
   ([`record_runner.go:110-112`](../../engine/internal/runner/record_runner.go)).
   No concurrent dispatch / commit interleaving to worry about.

### Eligibility under ADR-0049 §(a)

Required per
[`.claude/playbooks/post-wave3-session-loop.md`](../../.claude/playbooks/post-wave3-session-loop.md)
step 2. All four conditions must hold.

| # | Condition | Resolution |
|---|---|---|
| 1 | P-B3.1 — expands not rewrites | **Passes.** The record-mode runtime already ships; the runner shape (consume → window → dispatch) is unchanged. The change moves the consumer-group commit boundary from `PollFetches` return to `dispatcher.Run` success. No window contract, no kind catalog row, no `execution_id` input is reshaped. **Borderline reading flagged:** one reader might argue this is "completing a deferred slice of ADR-0024" rather than "evolution against ADR-0024's open decision space". The reading is borderline because ADR-0024 itself never decided commit timing (the kafka_consumer.go β-marker is an implementation note, not an ADR clause) and the runner's user-observable capability remains the same shape after the change — the change is to *delivery posture*, which is the canonical capability-mode-extension surface. Reading carried forward per R5 + adr-writing A7 (precedent: ADR-0055 §Notes, ADR-0056 §Notes). |
| 2 | P-B3.4 — in-scope family | **Passes.** ADR-0049 §"Per-family scope" → "Capability mode extensions" admits "a refinement to how capability is dispatched at runtime." Commit-boundary placement is a dispatch-time refinement. No new mode (`set` / `record` duo unchanged per ADR-0021); no extension that re-opens [ADR-0021](../../docs/adr/0021-mode-as-primitive.md). |
| 3 | P-B3.2 — envelope conformance | **Passes.** ADR-0020 substrate posture unchanged (Kafka stays Kafka, no new substrate). ADR-0021 mode primitive untouched. ADR-0022 kind catalog untouched. ADR-0023 sources schema untouched (no YAML change). ADR-0024 windowing semantics preserved unchanged. ADR-0002 `execution_id` formula preserved unchanged. ADR-0003 §1's append-only writes + ADR-0003 §2's canonical-view collapse per `execution_id` are the load-bearing substrate that *enables* at-least-once delivery to be consumer-visible-deduped — depending on them, not amending them. |
| 4 | Additive-maintenance threshold | **Passes.** Delivery-semantics change from at-most-once to at-least-once-with-dedup is operationally consequential — operators observe different post-crash behavior, and the interface change adds a method to a public package interface in `engine/internal/runner/`. Not a routine catalog PR. |

The temporal-classification test reads to B3 cleanly: Wave-S full
gate met 2026-05-25 per
[`CLAUDE.md`](../../CLAUDE.md) §2.2; today is 2026-05-30 (post-
shipping against a closed wave per ADR-0049 §(a)). Per
[`MEMORY.md`](../../../.claude/projects/-Volumes-OWC-Express-1M2-Develop-dq-platform/memory/MEMORY.md):
"record-mode items surfacing from 2026-05-25 onward are B3
(post-shipping against a closed wave), not B2-S."

The amendment-or-B3 disambiguation per ADR-0049 §(a) reads to B3:
ADR-0024's decided shape (windowing, watermark, late-drop policy,
execution_id formula reuse) is **preserved**; this study adds a
delivery-semantics layer the ADR did not decide. Not amendment.

---

## Decision Drivers

- **DD-1 — Closed-window invariant is contract; data delivery is
  silently broken.** ADR-0002 / ADR-0024 promise replayability;
  commit-after-fetch silently breaks the promise on every mid-
  dispatch crash by removing the inputs from the consumer-group's
  view. The β-marker comment acknowledges the gap explicitly.

- **DD-2 — Determinism (ADR-0024) + ADR-0003 §1 (append-only
  writes) + ADR-0003 §2 (canonical-view collapse via
  `dq_executions_current`) make at-least-once safe for
  downstream consumers.** The substrate that at-least-once
  delivery needs is the canonical-view collapse: re-dispatch
  produces a second attempt row (different `attempt_id` per
  ADR-0003 §4, same `execution_id` per ADR-0002), the base
  table preserves both, the canonical view that dashboards and
  alert systems target shows the latest `recorded_at` per
  `execution_id`. No new dedup mechanism is required; the
  `execution_id` boundary already exists and ADR-0003 §2 already
  collapses attempts at the consumer-visible layer. Storage
  cost of the second attempt is bounded by retention (acceptable
  in v1 because at-least-once retries are crash-driven, not
  steady-state).

- **DD-3 — Single-goroutine runner removes the hardest design
  surface.** A concurrent runner would need per-partition offset
  tracking + commit-ordering invariants. The runner is
  single-goroutine by construction; the commit logic is sequential
  and trivially ordered by record-arrival.

- **DD-4 — Test surface must exist in the existing fakeConsumer.**
  The B0-S5 / Wave-S β test surface
  ([`record_runner_test.go`](../../engine/internal/runner/record_runner_test.go))
  is the only test surface that covers the runner's commit /
  poll loop. The chosen interface shape must be testable with
  fakeConsumer extensions alone — no franz-go in tests.

- **DD-5 — `Record` (TriggerRequest field per ADR-0002 / ADR-0024)
  carries no `Topic` field; commit needs `(topic, partition,
  offset)`.** The minimal-blast-radius design adds a parallel
  field on `recordWindow` rather than reshaping the
  `TriggerRequest.Records` slice surface, since `Record` is
  consumed by the per-kind evaluators per ADR-0026 evidence
  shape and reshaping it would ripple into evidence assertions.

---

## Considered Options

### Option A — Per-batch commit, all-or-nothing

Move the commit call from `PollFetches` to the runner. After the
runner has processed all records in a `PollFetches`-returned
batch, call `consumer.Commit(ctx)`. If **any** dispatch in the
batch returned non-nil, skip the commit; the records re-flow on
restart.

Interface change:

```go
type RecordConsumer interface {
    PollFetches(ctx context.Context) ([]FetchedRecord, error)
    Commit(ctx context.Context) error
    Close()
}
```

- **Strengths.** Smallest interface delta (one method, no
  arguments). Single commit RPC per batch (cheaper than
  per-trigger). Easy to implement against franz-go
  (`CommitUncommittedOffsets` already does the right thing).
- **Weaknesses.** Records-in-still-open-windows are committed
  before dispatch (they were consumed in this batch, the batch
  succeeded, commit fires — but the active window hasn't been
  closed yet). Crash mid-window-accumulation loses those
  records. This is **at-most-once for records in pending
  windows** — the failure mode the user explicitly called out
  is partially preserved. Does not match the kafka_consumer.go
  comment's intent ("keyed on the trigger's successful return"
  — singular trigger, not batch).

### Option B — Per-trigger commit, keyed on dispatcher's successful return (Recommended)

Each closed window's `dispatcher.Run(ctx, trigger)` success is
its own commit boundary. The runner calls
`consumer.Commit(ctx, records)` immediately after
`dispatcher.Run` returns nil for that window, passing the
records that fed the window. On dispatch failure, the commit is
skipped; the records remain uncommitted in the broker; on
restart, they re-flow and re-populate the same window
deterministically per ADR-0024.

Records-in-still-open-windows are **not** committed: the active
window has not closed; no `dispatcher.Run` has been called for
it; `consumer.Commit` is not invoked. On crash, those records
are uncommitted in the broker. On restart, they re-flow and
re-populate the same active window per ADR-0024's deterministic
watermark logic.

Interface change:

```go
type RecordConsumer interface {
    PollFetches(ctx context.Context) ([]FetchedRecord, error)
    Commit(ctx context.Context, records []FetchedRecord) error
    Close()
}
```

The `records` argument carries `(Topic, Partition, Offset)` per
record; the franz-go implementation calls
`client.MarkCommitRecords(...)` then
`client.CommitMarkedOffsets(ctx)`, which commits the high-water
mark per partition for the given records. Late-dropped records
(not in any trigger) are committed transitively: a partition's
high-water mark advances monotonically as later records in the
same partition are dispatched, covering late drops with smaller
offsets.

- **Strengths.** Matches the kafka_consumer.go comment's intent
  literally ("commit-after-dispatch flow keyed on the trigger's
  successful return"). Records-in-still-open-windows are safe:
  not committed until their window closes and dispatches.
  At-least-once is preserved across all crash positions, not
  only mid-dispatch.
- **Weaknesses.** More commit RPCs (one per closed window per
  entity vs one per batch). Concrete bound: at ADR-0024's
  minimum `window.duration: 1s` with N concurrent record-mode
  rules, the upper bound is N RPCs/second to the broker; at
  typical durations (1m–5m), the rate is N/minute or lower —
  cost is P4-acceptable. Slightly larger interface (one arg
  added). Records-only-in-partitions-with-only-late-drops never
  commit (see OQ-2).

### Option C — Status quo + defer

Keep commit-after-fetch; document the at-most-once limitation in
operator-facing docs; revisit when concrete data loss is observed.

- **Strengths.** Zero code change. No interface evolution.
- **Weaknesses.** Leaves a known, documented data-loss path in
  v1. The β-marker comment already acknowledged this is wrong;
  not fixing it converts a deferred-implementation marker into a
  permanent posture. The platform's determinism / replayability
  promise (ADR-0002 §"Determinism", ADR-0024 §"Watermark-based
  closing is deterministic") is silently weakened — operators
  reading the ADR cannot tell from the ADR alone that mid-dispatch
  crashes drop data.

---

## Recommendation

**Option B.** It is the single posture that satisfies all of:

- The kafka_consumer.go β-marker's literal intent ("keyed on the
  trigger's successful return");
- The determinism / replayability promise of ADR-0002 + ADR-0024
  across all crash positions, not just mid-dispatch;
- ADR-0003 §1's append-only writes + ADR-0003 §2's canonical-view
  collapse, which together make a re-dispatched window
  consumer-invisible-as-a-duplicate (Option B depends on the
  composed substrate; it does not introduce its own dedup);
- The single-goroutine runner shape (per-trigger sequencing is
  free).

Option A is rejected because it partially preserves the failure
mode the comment explicitly calls out (records-in-pending-
windows lost on crash). Option C is rejected because it converts
a deferred-implementation marker into a permanent at-most-once
posture, weakening the determinism promise silently.

**Reading carried forward as new contribution proposed here,
requires review (R5) — see eligibility Condition 1 in Context.**
The "evolution against ADR-0024's open decision space" reading
(as opposed to "deferred slice of ADR-0024") is operator-
ratifiable; precedent disposition lives in ADR-0055 §Notes and
ADR-0056 §Notes per adr-writing A7.

### What this study commits

- The interface change: `RecordConsumer` adds
  `Commit(ctx context.Context, records []FetchedRecord) error`.
- The semantic change: at-least-once delivery with canonical-
  view collapse at the `execution_id` boundary (per ADR-0003 §2),
  supplanting at-most-once.
- The runner-side discipline: `closeAndDispatch` commits the
  trigger's records iff `dispatcher.Run` returned nil; commit
  failures are warning-logged and do not propagate (the records
  re-flow on restart; the next successful dispatch in the
  partition commits transitively).
- The minimal-blast-radius implementation choice: add
  `fetched []FetchedRecord` as a parallel field on
  `recordWindow` rather than reshaping
  `TriggerRequest.Records []Record`. The `Record` surface is
  consumed by per-kind evaluators per ADR-0026 evidence shape;
  reshaping it would ripple into evidence assertions across
  evaluators that are not in this slice's scope.
- The test discipline: extend `fakeConsumer` with a Commit
  method that records committed records into a ledger; add
  `TestRecordRunner_CommitsAfterSuccessfulDispatch` and
  `TestRecordRunner_DoesNotCommitOnDispatchFailure` covering
  the two material paths.

### What this study does NOT commit

- A change to ADR-0002's `execution_id` formula, ADR-0024's
  windowing semantics, ADR-0003's append-only-plus-canonical-view
  write model, or ADR-0023's sources schema. None are reopened.
- A per-partition or per-(topic, partition) explicit offset
  map on the Commit signature. The Kafka client library's
  `MarkCommitRecords` primitive computes the high-water mark
  per partition from the passed records; the substrate-agnostic
  interface stays records-shaped.
- A retry / back-off policy for commit failures. β posture is
  warning-log + skip; the records re-flow on restart or via the
  next successful dispatch's transitive commit. A bounded
  commit-retry policy is OQ-3 below.
- Handling for partitions that produce only late-dropped records
  for an extended period. β posture is "the partition's offset
  stays uncommitted; broker retention is the bound." Recorded as
  OQ-2 below.

---

## Consequences

1. **B3-6 reaches `resolved-study` and promotes to ADR-0058 in
   the same session under operator-authorized R4 scope
   collapse** (precedent: ADR-0054 §Notes, ADR-0055 §Notes,
   ADR-0056 §Notes). The promotion ADR commits the interface
   change, the semantic change, and the implementation discipline.

2. **The `RecordConsumer` interface gains one method.** The
   addition is at the engine-internal `runner` package's
   public-to-the-binary surface. The engine binary
   (`engine/cmd/dq-engine/main.go:buildRecordRunners`) wires
   `*FranzConsumer` and is unaffected by the interface change
   (it constructs the concrete type; the new method satisfies
   the interface transparently). Tests inject fakes that need
   to implement the new method.

3. **The β-marker comment in
   [`kafka_consumer.go`](../../engine/internal/runner/kafka_consumer.go)
   is removed.** The "commit-after-fetch (at-most-once on a
   crash mid-dispatch)" wording and the "future slice replaces
   this" wording are rewritten to reflect the new posture in the
   same PR.

4. **Record-mode runtime gains at-least-once delivery
   semantics.** Operator-facing behavior changes on crash:
   uncommitted records re-flow after restart and re-populate the
   same window deterministically; ADR-0003 §1's append-only
   writes preserve both the original (interrupted-or-failed) and
   re-dispatched attempts in the base table under a shared
   `execution_id`, and ADR-0003 §2's canonical view
   (`dq_executions_current`) collapses them to the latest
   `recorded_at` for consumer queries. Operator documentation
   describing record-mode delivery semantics (if any references
   exist in the docs surface today) needs a follow-up cross-
   reference; none are load-bearing for this slice's correctness.

5. **No ADR-0024 / ADR-0002 / ADR-0003 reshape.** The decided
   shape of each is preserved verbatim; this slice consumes
   them as its substrate.

6. **`record-mode-conventions` skill stays accurate.** The skill
   ([`.claude/skills/record-mode-conventions/SKILL.md`](../../.claude/skills/record-mode-conventions/SKILL.md))
   committed seven conventions S1–S7 grounded in the existing
   runner code; the relevant convention (S2 — β commit
   semantics) is updated in the same PR to drop the β-marker
   and point to ADR-0058. Skill-side update is light-touch per
   the originating ADR-0053's framing.

7. **PR-flow per CONTRIBUTING.md Flow 5 with R4 scope-collapse
   trailer.** Single PR carries study + critique capture + ADR
   + implementation + tests + decision-log update. Operator
   ratification of the R4 collapse is part of the merge
   approval; the rationale is recorded in ADR-0058 §Notes citing
   the precedent chain.

---

## Open Questions

- **OQ-1: Concurrent runner posture (multi-goroutine consumer).**
  The single-goroutine runner is the only shape Wave-S β
  shipped. A future operational signal might motivate a
  per-partition fan-out, which would re-open the commit
  ordering / serialization design (per-partition commit marks
  + barrier semantics). **Out-of-scope for current cycle** —
  the single-goroutine posture is committed by the existing
  runner shape; concurrent fan-out is a separate decision when
  concrete demand surfaces.

- **OQ-2: Partitions that produce only late-dropped records.**
  Late drops are not in any trigger; their offsets are
  committed transitively by later in-window dispatches in the
  same partition (high-water mark monotonic). A partition that
  produces *only* late drops over a long window leaves its
  offset uncommitted indefinitely, bounded by broker retention.
  **Out-of-scope for current cycle** — operationally rare; the
  broker-retention bound is acceptable. If concrete signal
  surfaces, a periodic "advance committed offset to consumed
  position when all consumed records are late drops" policy
  closes it.

- **OQ-3: Commit failure retry / back-off policy.** β posture
  on `consumer.Commit` returning non-nil is warning-log + skip;
  the records re-flow on restart or via the next successful
  dispatch's transitive commit. A bounded retry (e.g., three
  attempts with exponential back-off) might be operationally
  better when broker connectivity is flaky. **Out-of-scope for
  current cycle** — β posture is conservative-safe;
  operational signal will reveal whether retries are needed.

- **OQ-4: Per-window or per-batch commit-failure metric.** The
  emission slice committed in ADR-0055 / ADR-0056 does not
  list a `dq_record_commit_failures_total` counter. Adding one
  is appropriate when the commit path moves under runner
  control; whether it lands in this slice or a follow-up
  emission slice is a separate scoping question against
  ADR-0039's contract. **Out-of-scope for current cycle** — a
  commit-failure counter is not load-bearing for the at-least-
  once correctness this slice commits (the broker-retention-
  bounded re-flow path is the recovery mechanism, not the
  metric); the metric is observability, not correctness, and
  lands in the next emission-slice session when ADR-0039's
  inventory is re-scoped. Recorded so the next emission-slice
  session catches it.

- **OQ-5: Operator-rerun path for record-mode (replay of a
  Kafka offset range).** ADR-0024 §Notes deferred this. The
  commit-after-dispatch posture is the natural substrate for
  such a replay (uncommitted records re-flow; operator-driven
  offset-range re-read is a similar shape). **Out-of-scope for
  current cycle** — ADR-0024's deferral stands; this slice
  does not commit a rerun path.

---

## Promotion target

[`docs/adr/0058-record-runner-commit-after-dispatch.md`](../../docs/adr/0058-record-runner-commit-after-dispatch.md)
(reserved per
[ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 7; ADR-0057 already merged at `b40ce4c`).
