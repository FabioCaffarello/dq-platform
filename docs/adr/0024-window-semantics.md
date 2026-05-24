<!-- path: docs/adr/0024-window-semantics.md -->

# ADR-0024 — Window Semantics

- **Status:** accepted
- **Date:** 2026-05-24

---

## Context

ADR-0020 launched Wave-S; the foundational triplet (ADR-0021
mode primitive, ADR-0022 kind catalog, ADR-0023 sources schema
with Kafka substrate) closed the partial-Wave-S gate in spec.
ADR-0023's protocol-fit clause for Kafka was provisional: it
named consumer-group + per-partition-offset as the natural
anchor for record-mode `execution_id` binding but explicitly
deferred the windowing and offset semantics that turn that
anchor into runtime behaviour. This ADR closes that deferral.

The set-mode `execution_id` formula committed by
[ADR-0002](./0002-run-identity-and-idempotency.md) is:

```
execution_id = sha256_hex(
    ruleset_version || entity || window_start || window_end || trigger_source
)
```

Set-mode windows are **trigger-declared**: the scheduler or
operator sends a trigger carrying `window_start` and `window_end`,
the engine evaluates a snapshot of BigQuery for that window, and
one `dq_executions` row is written under the deterministic
`execution_id`. The committed `trigger_source` enum carries
`scheduler`, `manual`, and `operator-rerun`.

Record-mode has no per-trigger payload. The engine consumes a
Kafka topic continuously via a consumer group; there is no
external scheduler sending window endpoints. The engine itself
must decide what makes one evaluation — a **batch of records over
some boundary** — and bind that batch to an `execution_id`. The
boundary is the windowing model.

Four sub-decisions live inside this scope: the windowing model
itself; where the window declaration lives in the rule YAML; the
`execution_id` shape for record-mode (reuse the set-mode formula,
or commit a new identity rule); and how late-arrival records
interact with ADR-0002's closed-window invariant.

The platform principles that bear on the design: **P1** (rules
remain declarative — windowing lives in the rule, not in engine
code or env), **P2 / foundation 01 §"Determinism"** (the same
input stream must produce the same evaluation semantics —
watermark-based closing is deterministic; wall-clock closing is
not), **P5** (evolution contract-driven — `execution_id`
extensions live under ADR-0002's contract, schema and enum
extensions live under [ADR-0001](./0001-engine-rules-compatibility.md)'s).

---

## Decision

### Tumbling, watermark-bounded windows

Record-mode rules declare a **tumbling** window: non-overlapping
batches of fixed `duration`, aligned to UTC epoch. The window
closes when the watermark advances past `window_end`. Sliding
and session windows are not supported in v1; the windowing-type
enum is open for additive extension when concrete future kinds
motivate them.

### Window declaration lives in the source block

The `source.type: kafka` shape committed by
[ADR-0023](./0023-sources-schema.md) gains a required `window`
sub-object. Fields:

| Field | Required | Type | Description |
|---|---|---|---|
| `window.type` | yes | const `tumbling` | Window type. v1 supports `tumbling` only; future enum values extend additively. |
| `window.duration` | yes | string (Go-style duration; min `1s`) | Window size — e.g. `1m`, `5m`, `1h`. Aligned via `window_start = floor(event_time / duration) * duration` (nanoseconds since epoch). Works for any duration ≥ `1s`, including non-epoch-divisor values like `7m` or `13m`. |
| `window.lateness_tolerance` | yes | string (Go-style duration; min `0s`) | Watermark lag. A `0s` tolerance means the watermark equals `max_observed_event_time` — the window closes on any forward event-time movement past `window_end`, including a single out-of-order record. `0s` is recommended only for strictly monotonic event-time streams. |

The window block is **substrate-specific**, not mode-specific:
the field lives inside `source.type: kafka`, not at the rule's
top level. Set-mode rules (`source.type: bigquery`) carry no
window field; their windows arrive from the trigger.

```yaml
# Record rule with tumbling window (illustration)
version: 2
entity: customer_events
mode: record
source:
  type: kafka
  topic: customer_events
  consumer_group: dq-engine-customer-events
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema: { type: object, required: [id, event_type] }
```

A future record-mode substrate variant (other than Kafka) that
needs windowing would carry its own `window` sub-object under
its `source.type` variant — the per-substrate redundancy is
accepted in v1 in exchange for keeping mode-specific shape
contained inside the substrate variant.

### Watermark formula and partition combination

The watermark is a function of the consumer-group state:

```
watermark = max_observed_event_time - lateness_tolerance
```

where `max_observed_event_time` is the maximum event-time
observed across all consumed records in the consumer-group. The
per-partition combination policy — how the global watermark is
derived when records arrive on multiple Kafka partitions —
defaults in v1 to **min-across-partitions** (the global watermark
is the slowest partition's watermark). This is the most
conservative and most deterministic choice: a slow partition
holds back window closing for all faster partitions, but no
record is dropped for arriving "late" relative to a faster
partition's progress.

Two alternative policies remain available as future tuning
options when concrete operational signal motivates them:
**max-across-partitions** (closes windows fastest but risks
dropping records from slow partitions) and **per-partition**
(each partition closes its own windows independently;
cross-partition aggregation happens at result-write time).
Neither is wired in v1.

### Late-arrival records are dropped

Records whose event-time falls in a window whose watermark has
already passed `window_end` are **late** and are dropped. The
count is recorded as `late_dropped_count` in the
`dq_executions` evidence per the result-write model
([ADR-0003](./0003-result-write-model.md)). Late records never
reflow into a closed execution — the closed-window invariant
ADR-0002 commits and ADR-0003's append-only model rely on is
preserved.

### Idle topics fail in v1

A topic with no traffic never advances its watermark, so windows
never close. In v1 the engine **rejects idle topics**: a
record-mode rule whose topic has no traffic for longer than its
`lateness_tolerance` fails its current window and emits an
operational alert (per [ADR-0006](./0006-alert-routing-contract.md)).
Idle-watermark-advance (periodic heartbeat records or
wall-clock-based forward progress when no events arrive) is a
future enrichment that lands when concrete operational signal
motivates it, with explicit determinism caveats in operator
documentation.

### `execution_id` reuses ADR-0002's five-input shape

Record-mode `execution_id` is computed with the same formula
structure as ADR-0002:

```
execution_id = sha256_hex(
    ruleset_version || entity || window_start || window_end || trigger_source
)
```

Inputs for record-mode runs:

| Input | Source |
|---|---|
| `ruleset_version` | Manifest version active when the window closed |
| `entity` | Rule's `entity:` field |
| `window_start` | Tumbling window start (RFC3339, UTC, floor-aligned to `duration`) |
| `window_end` | `window_start + duration` (RFC3339) |
| `trigger_source` | `stream-watermark` (new enum value) |

`window_start` and `window_end` are **watermark timestamps** —
derived from event-time, not wall-clock. Same input stream +
same ruleset version + same window configuration produces the
same `execution_id` set. Operator replay (re-evaluating a Kafka
offset range against the same configuration) produces identical
`execution_id`s, matching ADR-0002's determinism promise.

### `trigger_source` enum extended

The `trigger_source` enum gains the value `stream-watermark`.
The set-mode trigger API continues to reject this value;
record-mode produces it internally when a window closes. ADR-0002's
formula itself is not reopened — only the enum's permitted
values expand, which is additive under ADR-0001's compatibility
contract (additive-within-major). Set-mode and record-mode share
the `execution_id` namespace; collisions are not a concern
because `entity` values are distinct across modes (record-mode
entities target Kafka topics; set-mode entities target BigQuery
tables) and `trigger_source` adds further entropy.

### Lint cross-checks #9, #10, #11

The lint binary (`tools/lint/`) gains three cross-checks on top
of the eight committed by ADRs 0021, 0022, and 0023:

- **#9 — `source.window` present iff `source.type` is
  record-mode.** A `source.type: bigquery` rule that declares
  `window` fails lint. A `source.type: kafka` rule that omits
  `window` fails lint.
- **#10 — `window.type` is in the supported enum.** v1
  supports only `tumbling`. Future enum extensions augment the
  supported set additively.
- **#11 — Duration grammar and minimum values.** `window.duration`
  ≥ `1s`; `window.lateness_tolerance` ≥ `0s`. Schema-side
  validation enforces the duration regex; the minimum-value
  constraint is lint-side.

Ten lint cross-checks total at this ADR's promotion.

---

## Consequences

1. **Rule schema v2's `source.type: kafka` variant gains a
   required `window` sub-object.** Three fields — `type`,
   `duration`, `lateness_tolerance` — all required. The schema
   change is additive within the combined v2 implementation
   commit that lands ADR-0021 / ADR-0022 / ADR-0023 / ADR-0024
   artefacts together. If any prior v2 artefact ships first, the
   v3 schema bump contingency from ADR-0022 §Decision and
   ADR-0023 §Decision applies.

2. **ADR-0002's `trigger_source` enum extends to include
   `stream-watermark`.** The extension is additive under
   ADR-0001's compatibility contract (additive-within-major); no
   set-mode contract is reopened. The set-mode trigger API
   continues to reject `stream-watermark`; only the engine's
   internal record-mode runner produces it.

3. **Record-mode `execution_id` reuses ADR-0002's five-input
   formula structure.** No new identity rule. Set-mode and
   record-mode share the `execution_id` namespace without
   collision.

4. **The closed-window invariant from ADR-0002 is preserved.**
   Late records are dropped and counted in evidence; they never
   reflow into a closed `dq_executions` row. ADR-0003's
   append-only write model and ADR-0006's per-attempt deduper
   continue to assume closed-window finality.

5. **Watermark-based closing is deterministic.** Same input
   stream + same ruleset version + same window configuration
   produces the same `execution_id` set. Operator replay against
   a fixed offset range produces identical `execution_id`s.

6. **Lint binary gains three cross-checks (#9, #10, #11).** Ten
   cross-checks total at this ADR's promotion.

7. **The protocol-fit clause from ADR-0023 retires its
   provisional status.** ADR-0023's Kafka substrate pick rested
   on protocol-fit "until B0-S4 finalises offset/watermark
   semantics". This ADR finalises those semantics consistent
   with the Kafka pick — watermark-bounded windows over
   partitioned topics, with offset-range replay producing
   deterministic re-evaluation. ADR-0023's protocol-fit clause
   is now committed, not provisional.

8. **Phase β progression: B0-S5 / B0-S6 / B0-S7 inherit this
   ADR's window-shape contract.** B0-S5 (unified-vs-parallel
   runner) consumes the window as the runner's evaluation unit.
   B0-S6 (failure scope aggregated) uses the window as the
   aggregation boundary — N per-record failures within one
   window map to one entity-status outcome. B0-S7 (cost
   guardrails) binds lag and lateness budgets to window
   discipline.

9. **Engine record-mode runtime gains watermark and consumer-
   offset tracking.** The source-fetching layer (location TBD
   per ADR-0023 §C-B0S3.7) consumes the rule's `window`
   configuration to drive consumer-group offset reads and
   per-partition watermark state. Implementation lands in the
   combined implementation commit that closes ADRs 0021–0024.

10. **Idle topics fail loudly in v1, not silently.** A
    record-mode rule whose topic has no traffic for longer than
    its `lateness_tolerance` fails its current window and emits
    an operational alert per ADR-0006. Operators see the
    problem; the engine does not silently stall.

11. **Sliding and session windows remain reachable additively.**
    The `window.type` enum is open for extension; future record
    kinds that genuinely need overlapping or gap-bounded windows
    can extend the enum without reopening any prior ADR. The v1
    commitment is tumbling-only because the only shipped record
    kind (`record.schema_conformance`, per ADR-0022) is
    per-record and gains nothing from overlap or session
    grouping.

---

## Notes

- The partition-combination policy (min vs max vs per-partition)
  is the most operationally consequential implementation knob
  this ADR exposes. v1 commits min-across-partitions for
  determinism; if a topic with one slow partition produces
  unacceptable closing lag, the operator-facing tuning path is
  to switch policies, not to reopen this ADR.
- Window alignment by floor-of-event-time-by-duration means any
  rule that picks an unusual duration (e.g., `7m`) will produce
  windows whose endpoints are not human-friendly multiples of
  minutes. The operator-facing documentation for window
  configuration should call this out.
- A `lateness_tolerance` of `0s` is allowed but is strict: any
  forward event-time movement past `window_end` closes the
  window, and any record subsequently arriving with an earlier
  event-time is dropped. Authors who pick `0s` without
  understanding the implication will see surprising drops.
- An operator-rerun path for record-mode (replay of a Kafka
  offset range with a captured window range) is not committed
  here. The future ADR that defines record-mode rerun
  semantics will decide whether the `trigger_source` enum
  gains a `stream-rerun` value or whether `operator-rerun`
  itself extends to record-mode.
- `dq_executions` partition column behaviour for record-mode
  runs (partition by `window_end` date, matching set-mode) is
  expected to work identically. Whether B0-S6 surfaces a
  separate partition discriminator (e.g., a `mode` column) is
  B0-S6's question.
- A future record-mode substrate variant added under ADR-0023's
  `source.type` discriminator that needs windowing will carry
  its own `window` sub-object — per-substrate redeclaration
  redundancy is the accepted v1 trade-off. If that redundancy
  becomes operationally painful, the placement is revisited in
  the ADR that introduces the new substrate.
