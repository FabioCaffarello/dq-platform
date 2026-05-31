<!-- path: .claude/skills/record-mode-conventions/reference/conventions.md -->

# Record-mode / stream conventions — reference

Seven patterns. Each one: the rule, the citation, a verbatim
snippet (≤6 lines), and a one-paragraph rationale. Companion
to [`SKILL.md`](../SKILL.md); committed by
[ADR-0053](../../../../docs/adr/0053-record-mode-skill.md).

---

## S1 — Substrate-agnostic consumer boundary

**Rule.** The `RecordRunner` consumes records through a
duck-typed `RecordConsumer` interface declared on the
consumer side, and over a substrate-agnostic `FetchedRecord`
struct. Substrate-specific types (`kgo.*`) are confined to
`kafka_consumer.go`, which maps `kgo.Record → FetchedRecord`
at the boundary so the runner stays free of franz-go
dependencies.

**Citation.** `engine/internal/runner/record_runner.go:31-47`
(interface) + `49-58` (`FetchedRecord`);
`engine/internal/runner/kafka_consumer.go:13-25` (provider) +
`85-92` (per-record mapping).

```go
type RecordConsumer interface {
    PollFetches(ctx context.Context) ([]FetchedRecord, error)
    Close()
}
```

**Rationale.** Declaring the interface consumer-side (per
the [`go-coding-standards`](../../go-coding-standards/SKILL.md)
C4 boundary discipline) keeps the runner reusable against a
different substrate (an in-memory fake; a future broker)
without rewriting the state machine. A substrate-agnostic
record shape means the window-close path and the trigger
dispatch never see substrate types — which is what makes the
test doubles work without standing up franz-go.

---

## S2 — Commit-after-dispatch semantics (at-least-once)

**Rule.** Auto-commit is disabled on the franz-go client;
the runner is the sole commit authority per
[ADR-0058](../../../../docs/adr/0058-record-runner-commit-after-dispatch.md)
§Clause 3. `PollFetches` returns records without committing.
After each closed window's `dispatcher.Run` returns nil, the
runner calls `consumer.Commit(ctx, fetched)` with the
window's records (per ADR-0058 §Clause 2). The Kafka-side
`Commit` implementation translates each `FetchedRecord` to a
`*kgo.Record` carrying topic/partition/offset, calls
`client.MarkCommitRecords` (the client tracks the high-water
mark per partition internally), then
`client.CommitMarkedOffsets` (flushes to the broker
synchronously). On dispatch failure, the commit is skipped;
the records remain uncommitted in the broker; on engine
restart, they re-flow per ADR-0024's deterministic windowing,
and ADR-0003 §2's canonical view (`dq_executions_current`
collapses attempts per `execution_id` to the latest
`recorded_at`) absorbs any spurious second attempt at the
consumer-visible layer. Commit failures are warning-logged
and do not propagate; the next successful dispatch in the
same partition commits the uncommitted records transitively
via high-water-mark monotonicity.

**Citation.**
`engine/internal/runner/kafka_consumer.go:60-67` (the
`kgo.Opt` block including `DisableAutoCommit`);
`engine/internal/runner/kafka_consumer.go:107-124`
(`FranzConsumer.Commit` — `MarkCommitRecords` +
`CommitMarkedOffsets`);
`engine/internal/runner/kafka_consumer.go:13-26`
(`FranzConsumer` struct doc naming the runner-as-sole-commit-
authority posture);
`engine/internal/runner/record_runner.go:350-378`
(`closeAndDispatch` — per-trigger commit keyed on
`dispatcher.Run` success).

```go
opts := []kgo.Opt{
    kgo.SeedBrokers(cfg.Brokers...),
    kgo.ConsumerGroup(cfg.ConsumerGroup),
    kgo.ConsumeTopics(cfg.Topics...),
    kgo.DisableAutoCommit(),
}
```

**Rationale.** Commit-after-dispatch keyed on
`dispatcher.Run` success delivers at-least-once semantics
across all crash positions, not only mid-dispatch. ADR-0024's
deterministic windowing replays the same window from the
re-flowed records; ADR-0002's deterministic `execution_id`
formula produces the same identifier; ADR-0003 §1's append-
only writes preserve both attempts in the base table under
the shared `execution_id` (with distinct `attempt_id` per
§4), and ADR-0003 §2's canonical view collapses them to one
row for downstream consumers. The runner-driven commit
boundary makes the dispatcher.Run nil return the load-bearing
event; pushing the commit responsibility into the consumer
would couple the consumer to dispatch outcomes it does not
otherwise observe.

**Retry envelope** (per
[ADR-0059](../../../../docs/adr/0059-record-runner-commit-retry.md)):
the runner wraps `consumer.Commit` in a `commitWithRetry`
helper that retries up to `recordCommitMaxAttempts = 3` times
with exponential back-off and uniform-random jitter (delay =
`random_uniform(0, recordCommitBackoffBase × 2^attempt)`
where `base = 100ms`). Worst-case stall is `(base × 2^1) +
(base × 2^2) = 600ms`; expected `~150ms` under uniform-random
jitter. The back-off `select` statement respects `ctx.Done()`
so engine shutdown pre-empts the retry loop within the
current back-off window. Jitter source: `math/rand/v2`
(stdlib in Go 1.22+; no third-party retry library). After the
retry budget is exhausted, the helper returns the last commit
error and `closeAndDispatch` falls through to the
warning-log + skip path verbatim — the retry layer is additive
on top of the existing terminal, not replacing it. The runner
warning log gains a `commit_attempts` field on exhaustion so
operators can distinguish retried-and-exhausted from
single-attempt failure.

**Retry citation.**
`engine/internal/runner/record_runner.go:18-35`
(`recordCommitMaxAttempts` + `recordCommitBackoffBase`
β constants);
`engine/internal/runner/record_runner.go:390-440`
(`commitWithRetry` helper + `closeAndDispatch` invocation
site).

---

## S3 — Translation-at-boot boundary

**Rule.** The runner package deliberately does not import
`dsl/spec`. The engine binary's `buildRecordRunners` reads
the parsed `RuleSpec`, calls `RuleSpec.ToCheckSpecs()`
(defined in `dsl/spec`) to produce the `[]runner.CheckSpec`
slice the runner consumes, and constructs
`runner.RecordSource` values at boot. The duplication of the
`Source` shape on both sides of the boundary
(`dsl/spec.Source` and `runner.RuleSource`) is protected by
a reflection sweep in the **external** `runner_test`
package — an internal-test-package would close an import
cycle (since `dsl/spec` imports `runner` for the
`runner.CheckSpec` return type).

**Citation.**
`engine/internal/runner/record_runner.go:18-21`
(translation comment on `RecordSource`);
`engine/internal/dsl/spec/spec.go:114-127` (`ToCheckSpecs`
method); `engine/cmd/dq-engine/main.go:624-703`
(`buildRecordRunners` call site that parses + translates +
wires `RecordRunnerConfig.Sources`);
`engine/internal/runner/struct_mirror_test.go:1-60`
(external test package rationale + reflection sweep
asserting field-set parity).

```go
// runner/record_runner.go: the struct mirrors the
// carrying-format from engine/internal/dsl/spec.RuleSpec;
// the runner package does not depend on dsl/spec so the
// engine binary translates at boot.
```

**Rationale.** The runner stays a pure consumer of typed
values; the engine binary owns the YAML → typed-value
translation. The translation-at-boot move keeps the runner
testable without a parser, but it costs a per-side
duplication of the `Source` shape. The reflection sweep is
the load-bearing safety net: a new field on either side that
forgets the other surfaces as a CI failure rather than a
silent runtime miss. Placing the sweep in the external
`runner_test` package is the only way to import both
packages in the same test without closing the import cycle
that `dsl/spec → runner → dsl/spec` would otherwise form.

---

## S4 — `TriggerDispatcher` interface declared consumer-side

**Rule.** The `RecordRunner` needs only one method from the
inner runner — `Run(ctx, TriggerRequest) → (*ExecutionRow,
error)`. The interface is declared in `record_runner.go`
(consumer-side, duck-typed) so tests can inject a mock
dispatcher without standing up the full `*Runner`. A
compile-time assertion (`var _ TriggerDispatcher =
(*Runner)(nil)`) in the same file pins `*Runner` to the
contract: a future change to `Runner.Run`'s signature breaks
the assertion at compile time.

**Citation.**
`engine/internal/runner/record_runner.go:92-97` (interface
declaration with doc comment);
`engine/internal/runner/record_runner.go:381` (compile-time
assertion).

```go
type TriggerDispatcher interface {
    Run(ctx context.Context, trigger TriggerRequest) (*results.ExecutionRow, error)
}

var _ TriggerDispatcher = (*Runner)(nil)
```

**Rationale.** Without the interface, tests would need to
construct a real `*Runner` (which requires a `Store`, an
evaluator, an env config, a logger) just to exercise the
record runner's state machine. The narrow consumer-side
interface lets tests inject a single-method mock. The
compile-time assertion is the load-bearing inverse: it
prevents the production `*Runner` from drifting away from
the interface during a `Run` signature change without anyone
noticing — a runtime-only check would fail late and far from
the cause.

---

## S5 — Watermark-driven window-close semantics (ADR-0024)

**Rule.** The watermark advances monotonically as the max of
record timestamps seen so far. The active window closes when
the watermark crosses `active.end + lateness_tolerance` per
ADR-0024. On close, a `TriggerRequest` carries the
accumulated records, the window bounds, and the
`LateDroppedCount`. Strictly-later-window records eagerly
close the active window at sub-slice β; a per-window
parallel buffer is a follow-up enhancement.

**Citation.**
`engine/internal/runner/record_runner.go:230-316`
(`handleFetched` state machine — watermark advance, window
selection, eager-close-on-later, late-drop counting,
post-append close check);
`engine/internal/runner/record_runner.go:318-359`
(`closeAndDispatch` — emits `TriggerRequest` with
`LateDroppedCount`, resets `state.active` and `state.lateDropped`).

```go
// Post-append close check: did this record's timestamp push
// the watermark past active.end + lateness_tolerance?
if state.watermark.After(state.active.end.Add(src.LatenessTolerance)) {
    r.closeAndDispatch(ctx, state)
}
```

**Rationale.** Watermark-driven close is the ADR-0024
contract: closing eagerly on a strictly-later-window record
(rather than waiting for the watermark to advance) is the β
simplification that trades parallel-window throughput for
state-machine simplicity. The `LateDroppedCount` field on
the trigger surfaces the late-drop count to the dispatcher
so it can be recorded in the result; without it, late drops
would be invisible to operators. The follow-up enhancement
(per-window parallel buffer) replaces the eager-close path
without changing the watermark contract — the close fires
*at the same moment*; the difference is whether the
later-window records are buffered or eagerly dispatched.

---

## S6 — Single-goroutine state machine

**Rule.** The per-entity state map is initialized in
`NewRecordRunner` and mutated only inside `handleFetched`
and `closeAndDispatch`, both called from `Start`'s single
consumer poll loop. No internal locking. A `sync`-import
sentinel (`var _ = sync.Mutex{}`) at the bottom of
`record_runner.go` keeps the import explicit so a future PR
that adds mutex-protected fields surfaces the addition in
the diff (the import line would otherwise be silent).

**Citation.**
`engine/internal/runner/record_runner.go:110-115`
(invariant doc comment on the `RecordRunner` struct);
`engine/internal/runner/record_runner.go:193-227` (`Start`
poll loop — single goroutine reads, routes, closes windows);
`engine/internal/runner/record_runner.go:399-403`
(`sync`-import sentinel).

```go
// Per-entity state. The runner is single-goroutine by
// construction (Start runs one consumer poll loop); no
// internal locking is required.
sources map[string]*RecordSource // keyed by topic
state   map[string]*entityState  // keyed by entity
```

**Rationale.** The single-goroutine invariant is a
deliberate simplification: avoiding a per-entity mutex
means the state machine reads as straight-line Go without
defer-unlock noise. The engine binary's restart-policy
recovery (per the `kafka_consumer.go` β-posture comment) is
the systemic-failure response; the absence of internal
locking does not need a separate recovery path. The
`sync`-import sentinel makes the invariant **structurally
load-bearing** rather than just documented: a PR that
introduces a `sync.Mutex` field has to re-justify the
invariant; a PR that removes the sentinel removes the
visible reminder.

---

## S7 — `CheckEvaluator` boundary + colocated test doubles

**Rule.** The `CheckEvaluator` interface is declared in
`check_evaluator.go` (consumer-side, duck-typed). Three test
doubles — `NoopEvaluator`, `FixedResultEvaluator`,
`PerCheckEvaluator` — live in the **same file** so test
wiring is a single-import action. The production
`*eval.Evaluator` satisfies the interface implicitly via
duck typing, per the
[`go-coding-standards`](../../go-coding-standards/SKILL.md)
C4 boundary discipline.

**Citation.**
`engine/internal/runner/check_evaluator.go:20-64` (interface
+ three test doubles in one file);
`engine/internal/eval/doc.go:9-13` (production satisfier
named from the provider side).

```go
type CheckEvaluator interface {
    Evaluate(ctx context.Context, spec CheckSpec, trigger TriggerRequest) (Evaluation, error)
}

type NoopEvaluator struct{}
type FixedResultEvaluator struct{ Result results.CheckResult }
type PerCheckEvaluator struct{ Results map[string]results.CheckResult }
```

**Rationale.** Test wiring is a frequent operation, and an
extra package import per test adds cost without adding
value. Colocating the doubles with the interface means a
test that needs a `Noop`, a `Fixed`, or a per-check map
imports `runner` and is done. The provider-side doc.go
declaration (`eval/doc.go:9-13`) closes the boundary
description: the runner says "I need this interface"; the
eval package says "I satisfy it via duck typing." The
contract reads from both sides without either side
importing the other.

---

## Anti-patterns the record-mode code consistently avoids

These are absent from the record-mode surface for a reason.
Do not introduce them.

- **Substrate-specific types leaking into production
  record-mode code outside `kafka_consumer.go`.** `kgo.*`
  does not appear in production Go outside
  `engine/internal/runner/kafka_consumer.go`. The only
  reference to `kgo.*` anywhere else in the runner package
  today is a doc-comment mention at `record_runner.go:51`
  describing the `kgo.Record → FetchedRecord` mapping —
  intentional documentation of the boundary, not a type
  leak. A PR that introduces `kgo.*` in production code
  outside `kafka_consumer.go` breaks S1's substrate-agnostic
  boundary; the right fix is to extend `FetchedRecord` or
  to add a new mapper inside `kafka_consumer.go`.

- **Internal locking inside the `RecordRunner` struct.** No
  `sync.Mutex` field on the production `RecordRunner` type.
  The `sync` import in `record_runner.go` is referenced only
  by the sentinel at line 403 (`var _ = sync.Mutex{}`).
  Test fixtures in `record_runner_test.go` use mutexes —
  that is expected and fine; the S6 invariant binds
  production runner code, not the consumer fakes the tests
  inject. Adding a mutex to the production struct requires
  re-justifying S6 — and the sentinel exists precisely so
  that addition is visible in the diff.

- **`dsl/spec` imported by any file under
  `engine/internal/runner/` (production or internal test).**
  The struct-mirror test lives in the external `runner_test`
  package precisely to break the
  `dsl/spec → runner → dsl/spec` import cycle that an
  internal test would close. A PR that adds
  `import ".../dsl/spec"` to any file inside the runner
  package — production or internal-test — breaks the
  invariant and the test. Doc-comment mentions of `dsl/spec`
  (the conventions describing the boundary) are not
  imports and are expected.

- **WHAT-style comments inside record-mode code.** Comments
  explain WHY, typically citing ADR-0024 (window-close
  semantics), ADR-0021 (mode primitive), or ADR-0058 (the
  runner-as-sole-commit-authority posture). The
  [`go-coding-standards`](../../go-coding-standards/SKILL.md)
  anti-patterns apply uniformly across the engine.
