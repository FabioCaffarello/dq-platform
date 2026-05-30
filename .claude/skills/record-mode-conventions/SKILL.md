<!-- path: .claude/skills/record-mode-conventions/SKILL.md -->
---
name: record-mode-conventions
description: Use when writing, reviewing, or extending record-mode / stream-processing code under engine/internal/runner/ (record_runner.go, kafka_consumer.go, check_evaluator.go) or the engine binary's record-mode boot translation (engine/cmd/dq-engine/main.go's buildRecordRunners). Encodes seven conventions S1–S7 the existing record-mode code carries: substrate-agnostic consumer boundary (RecordConsumer interface + FetchedRecord shape; franz-go mapping at the boundary); β commit semantics (DisableAutoCommit + manual CommitUncommittedOffsets after PollFetches; per-attempt re-read deferred per ADR-0024); translation-at-boot boundary (runner package free of dsl/spec; engine binary calls RuleSpec.ToCheckSpecs() at boot; guarded by reflect-based struct-mirror test in external test package); TriggerDispatcher interface for testability with compile-time assertion that *Runner satisfies it; watermark-driven window-close semantics per ADR-0024 with LateDroppedCount on the trigger; single-goroutine state machine with no internal locking; colocated CheckEvaluator test doubles. Apply when touching the record-mode runner, the franz-go consumer, the boot-time translation, or the inner evaluator boundary. Trigger phrases include "record runner", "record mode", "RecordRunner", "RecordSource", "kafka consumer", "FetchedRecord", "consumer group", "TriggerDispatcher", "window close", "watermark", "lateness tolerance", "β commit", "commit-after-fetch", "DisableAutoCommit", "CommitUncommittedOffsets", "boot translation", "ToCheckSpecs", "struct mirror".
---

# `record-mode-conventions`

Patterns extracted from `engine/internal/runner/` (record-mode
files: `record_runner.go`, `kafka_consumer.go`,
`check_evaluator.go`), `engine/cmd/dq-engine/main.go`'s
record-mode boot translation, and the external `runner_test`
struct-mirror test. Every rule traces to a real file:line. For
the full code snippets and the rationale at length, see the
reference file.

> Reference file:
> - `reference/conventions.md` — the seven patterns S1–S7 with
>   file:line citations and verbatim Go snippets (≤6 lines each).

Committed by [ADR-0053](../../../docs/adr/0053-record-mode-skill.md).

---

## S1. Substrate-agnostic consumer boundary

`RecordConsumer` is a duck-typed interface declared
consumer-side; `FetchedRecord` is the substrate-agnostic record
shape the runner consumes. Substrate-specific types (`kgo.*`)
are confined to `kafka_consumer.go`, which maps `kgo.Record →
FetchedRecord` at the boundary.

```go
type RecordConsumer interface {
    PollFetches(ctx context.Context) ([]FetchedRecord, error)
    Close()
}
```

`engine/internal/runner/record_runner.go:31-47` (interface +
doc comment); `engine/internal/runner/record_runner.go:49-58`
(`FetchedRecord` shape); `engine/internal/runner/kafka_consumer.go:13-25`
(franz-go-backed implementation); `engine/internal/runner/kafka_consumer.go:85-92`
(per-record mapping inside `EachRecord`).

## S2. β commit semantics

`DisableAutoCommit()` is set on the franz-go client;
`CommitUncommittedOffsets` runs after every successful
`PollFetches`. β commits after the fetch, not after the
dispatch — at-most-once on a crash mid-dispatch. Per-attempt
re-read of offset ranges per ADR-0024 is a future slice that
will replace β commit with commit-after-dispatch.

```go
opts := []kgo.Opt{
    kgo.SeedBrokers(cfg.Brokers...),
    kgo.ConsumerGroup(cfg.ConsumerGroup),
    kgo.ConsumeTopics(cfg.Topics...),
    kgo.DisableAutoCommit(),
}
```

`engine/internal/runner/kafka_consumer.go:62-67` (the
`kgo.Opt` block including `DisableAutoCommit`);
`engine/internal/runner/kafka_consumer.go:94-102` (the
`CommitUncommittedOffsets` call + β-posture comment naming the
future replacement); `engine/internal/runner/kafka_consumer.go:18-22`
(`FranzConsumer` struct doc naming β commits-after-fetch and
the future slice).

## S3. Translation-at-boot boundary

The runner package deliberately does not import `dsl/spec`.
The engine binary's `buildRecordRunners` calls
`RuleSpec.ToCheckSpecs()` (defined in `dsl/spec`) and
constructs `runner.RecordSource` values at boot. The
duplication of the `Source` shape on both sides of the
boundary is protected by a reflection sweep in the external
`runner_test` package — internal-test-package import would
close an import cycle.

```go
// runner/record_runner.go (consumer-side, no dsl/spec import):
// The struct mirrors the carrying-format from
// engine/internal/dsl/spec.RuleSpec; the runner package does
// not depend on dsl/spec so the engine binary translates at
// boot.
```

`engine/internal/runner/record_runner.go:18-21` (translation
comment on `RecordSource`); `engine/internal/dsl/spec/spec.go:114-127`
(`RuleSpec.ToCheckSpecs` translation method);
`engine/cmd/dq-engine/main.go:624-703` (`buildRecordRunners`
call site that parses + translates + wires);
`engine/internal/runner/struct_mirror_test.go:1-60` (external
test package rationale + reflection sweep).

## S4. `TriggerDispatcher` interface, declared consumer-side

The `RecordRunner` needs only one method from the inner
runner. The interface is declared in `record_runner.go`
(consumer-side, duck-typed) so tests can inject a mock
dispatcher without standing up the full `*Runner`. A
compile-time assertion in the same file pins `*Runner` to the
contract.

```go
type TriggerDispatcher interface {
    Run(ctx context.Context, trigger TriggerRequest) (*results.ExecutionRow, error)
}

var _ TriggerDispatcher = (*Runner)(nil)
```

`engine/internal/runner/record_runner.go:92-97` (interface
declaration with doc comment naming the test-mock motivation);
`engine/internal/runner/record_runner.go:381` (compile-time
assertion).

## S5. Watermark-driven window-close semantics (ADR-0024)

The watermark advances monotonically as the max of record
timestamps. The active window closes when the watermark
crosses `active.end + lateness_tolerance` per
[ADR-0024](../../../docs/adr/0024-window-semantics.md); on
close, a `TriggerRequest` carries the accumulated records,
the window bounds, and `LateDroppedCount`. Strictly-later-
window records eagerly close the active window at sub-slice
β; a per-window parallel buffer is a follow-up enhancement.

```go
// Post-append close check: did this record's timestamp push
// the watermark past active.end + lateness_tolerance?
if state.watermark.After(state.active.end.Add(src.LatenessTolerance)) {
    r.closeAndDispatch(ctx, state)
}
```

`engine/internal/runner/record_runner.go:230-316`
(`handleFetched` state machine — watermark advance, window
selection, eager-close-on-later, late-drop counting, post-
append close check); `engine/internal/runner/record_runner.go:318-359`
(`closeAndDispatch` — emits `TriggerRequest` with
`LateDroppedCount`, resets state).

## S6. Single-goroutine state machine

The per-entity state map is mutated only inside `handleFetched`
and `closeAndDispatch`, both called from `Start`'s single
consumer poll loop. No internal locking. A `sync`-import
sentinel makes the absence of mutex-protected fields explicit
in the diff: a future PR that adds a mutex must re-justify the
single-goroutine invariant.

```go
// Per-entity state. The runner is single-goroutine by
// construction (Start runs one consumer poll loop); no
// internal locking is required.
sources map[string]*RecordSource // keyed by topic
state   map[string]*entityState  // keyed by entity
```

`engine/internal/runner/record_runner.go:110-115` (invariant
doc comment on the `RecordRunner` struct);
`engine/internal/runner/record_runner.go:193-227` (`Start`
poll loop — single goroutine reads, routes, closes windows);
`engine/internal/runner/record_runner.go:399-403` (`sync`-
import sentinel `var _ = sync.Mutex{}` keeping the import
explicit).

## S7. `CheckEvaluator` boundary + colocated test doubles

The `CheckEvaluator` interface is declared in
`check_evaluator.go` (consumer-side, duck-typed); three test
doubles (`NoopEvaluator`, `FixedResultEvaluator`,
`PerCheckEvaluator`) live in the same file so test wiring is
a single-import action. The production `*eval.Evaluator`
satisfies the interface implicitly per the
[`go-coding-standards`](../go-coding-standards/SKILL.md) C4
boundary discipline.

```go
type CheckEvaluator interface {
    Evaluate(ctx context.Context, spec CheckSpec, trigger TriggerRequest) (Evaluation, error)
}

type NoopEvaluator struct{}
type FixedResultEvaluator struct{ Result results.CheckResult }
type PerCheckEvaluator struct{ Results map[string]results.CheckResult }
```

`engine/internal/runner/check_evaluator.go:20-64` (interface
+ three test doubles in one file);
`engine/internal/eval/doc.go:9-13` (production `Evaluator`
satisfies `runner.CheckEvaluator` via duck typing —
boundary documented from the provider side).

---

## Anti-patterns the record-mode code consistently avoids

Do not introduce any of these — they are absent from the
record-mode surface for a reason:

- **Substrate-specific types leaking into production
  record-mode code outside `kafka_consumer.go`.** `kgo.*`
  does not appear in production Go outside
  `kafka_consumer.go`; `FetchedRecord` is the only shape the
  rest of the runner sees in code. The only reference in any
  other file today is a doc-comment mention at
  `record_runner.go:51` describing the boundary mapping —
  that is intentional documentation, not a type leak.
- **Internal locking inside the `RecordRunner` struct.** No
  `sync.Mutex` field on `RecordRunner`. The `sync` import is
  referenced only by the sentinel at
  `record_runner.go:399-403`. Test fixtures in
  `record_runner_test.go` use mutexes — that is expected and
  fine; the S6 invariant binds production runner code, not
  the consumer fakes the tests inject. Adding a mutex to the
  production struct requires re-justifying S6.
- **`dsl/spec` imported by any file under
  `engine/internal/runner/` (production code or internal
  test).** The struct-mirror test (S3) lives in the external
  `runner_test` package precisely to break the
  `dsl/spec → runner → dsl/spec` import cycle that an
  internal test would close. Production code in
  `engine/internal/runner/` is `dsl/spec`-free; comments
  mention the package conceptually, but no `import` line
  pulls it in.
- **WHAT-style comments inside record-mode code.** Comments
  explain WHY, typically citing ADR-0024 (window-close
  semantics) or naming the future-slice replacement of β
  commit. The
  [`go-coding-standards`](../go-coding-standards/SKILL.md)
  anti-patterns apply uniformly across the engine.
