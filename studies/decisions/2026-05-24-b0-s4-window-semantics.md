<!-- path: studies/decisions/2026-05-24-b0-s4-window-semantics.md -->

# B0-S4 — Window Semantics

## Metadata

- **B-item reference:** B0-S4 (Wave-S Phase β, item 1 of 4)
- **Status:** resolved-study (Wave-S, B0-S4; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-24
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch — assigns B0-S4's scope; partial-Wave-S gate
  closed by B0-S3 promotion);
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) (mode
  primitive); [ADR-0022](../../docs/adr/0022-kind-catalog.md) (kind
  catalog — `record.schema_conformance` per-record kind);
  [ADR-0023](../../docs/adr/0023-sources-schema.md) (Kafka
  substrate, topic + consumer_group identifier shape);
  [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md)
  (set-mode `execution_id` formula — five inputs, sha256_hex,
  pipe-separated; trigger_source closed enum with `scheduler`,
  `manual`, `operator-rerun`).
- **Downstream open:** B0-S5 (aggregation & unified-vs-parallel
  runner — consumes window-shape decision committed here);
  B0-S6 (failure scope aggregated — uses windows as the
  aggregation boundary for per-record failures); B0-S7
  (record-mode cost guardrails — lag and lateness budgets bind
  to window discipline).
- **Promotion target:** `docs/adr/0024-window-semantics.md`
  (subject to ADR-0020 §Decision (Per-item ADR numbering); `0024`
  reflects the expected sequence S4 → 0024, modulo unrelated
  promotions and the ADR-0010 amendment-ADR that may land first).
- **Loop discipline:** same as B0-S1–S3 — `/resolve-b0` study →
  `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.
- **Phase context:** B0-S4 is the first Phase β study after the
  partial-Wave-S gate opened. Per ADR-0020 §Decision (Sequencing
  rule), B0-S4–S7 may be drafted in parallel; promotion remains
  gated on declared dependencies.

---

## Context

ADR-0021 realised the mode primitive in schema shape; ADR-0022
committed the kind catalog with `record.schema_conformance` as the
inaugural record-mode kind; ADR-0023 committed Kafka as the
record-mode substrate with `source.type: kafka` carrying `topic`
and `consumer_group`. ADR-0023's protocol-fit clause explicitly
deferred **how Kafka offsets and watermarks bind to the ADR-0002
`execution_id` formula** to this study. ADR-0020 §B0-S4 commits
the scope:

> Decides what "window" means for record-mode — tumbling, sliding,
> session, watermark-bounded, or "no window, evaluate per-record";
> how watermarks interact with (or replace) the ADR-0002
> `execution_id` window-endpoint formula; whether record-mode
> `execution_id` reuses the ADR-0002 formula or gains a new shape
> under a `record.*` identity rule; how late-arrival records are
> handled relative to the closed-window invariant ADR-0002 currently
> encodes.

The set-mode `execution_id` formula committed by ADR-0002 is:

```
execution_id = sha256_hex(
    ruleset_version || entity || window_start || window_end || trigger_source
)
```

Set-mode windows are **trigger-declared**: the scheduler or
operator sends a trigger payload that carries `window_start` and
`window_end`. The engine evaluates a snapshot of BigQuery for that
window and writes one `dq_executions` row keyed on the
deterministic `execution_id`. Set-mode `trigger_source` enum:
`scheduler`, `manual`, `operator-rerun`.

Record-mode has no per-trigger payload — the engine consumes a
Kafka topic continuously via a consumer group, and there is no
external scheduler sending window endpoints. The engine itself
decides what makes one "evaluation": **a batch of records over
some boundary**, where the boundary is the windowing model B0-S4
commits.

Four interlocking sub-decisions live inside B0-S4's scope:

1. **The windowing model** — tumbling, sliding, session,
   watermark-bounded, or per-record. Each shapes the
   evaluation-batch boundary differently.
2. **Where the window declaration lives** — in the rule (and if
   so, where in the rule YAML), in a separate windowing artefact,
   or implicit per kind.
3. **The `execution_id` formula for record-mode** — same shape as
   ADR-0002 with `window_start`/`window_end` as watermark
   timestamps, or a new Kafka-native shape keyed on offsets +
   partitions.
4. **Late-arrival handling** — records whose event-time falls in a
   window whose watermark has already advanced past `window_end`.
   Drop, reconcile, or new-execution.

The platform principles bearing on this decision: **P1**
(declarative — windowing lives in the rule, not in engine code or
env), **P2 / foundation 01 §"Determinism"** (same input stream
must produce the same evaluation semantics — watermark-based
window closing is deterministic, wall-clock closing is not),
**P5** (evolution contract-driven — `execution_id` extensions
under ADR-0002's contract, schema extensions under ADR-0001's
contract), and the protocol-fit clause from ADR-0023 (Kafka's
consumer-group + per-partition-offset model is the substrate
anchor; B0-S4 turns that into runtime semantics).

---

## Decision Drivers

- **DD-S4.1** — **Honour the foundational triplet.** Mode
  primitive (P1), kind catalog (B0-S2), and Kafka source shape
  (B0-S3) constrain B0-S4's design. Windowing applies to
  record-mode rules only; set-mode rules keep their
  trigger-declared windows from ADR-0002 unchanged.

- **DD-S4.2** — **Honour ADR-0002's execution_id contract.**
  Per R3, the existing formula's structure (five inputs,
  pipe-separated, sha256_hex) is not reopened on the set side.
  Record-mode either reuses the same structure (with adapted
  inputs) or commits a new identity rule that lives alongside
  ADR-0002 without amending it.

- **DD-S4.3** — **Determinism (foundation 01 §"Determinism" /
  ADR-0002's stability promise).** Same input stream + same
  ruleset version must produce the same `execution_id` set. This
  forbids wall-clock-driven window closing — windows must close
  on a function of the input stream itself (watermarks derived
  from event time).

- **DD-S4.4** — **Minimum-viable shape for the inaugural record
  kind.** `record.schema_conformance` (the only record kind in
  catalog v1) is per-record by semantics. The windowing model
  must support per-record evaluation as a degenerate case while
  also producing a `dq_executions` row per evaluation batch —
  the platform's execution unit is the batch, not the record.

- **DD-S4.5** — **Watermarks are the closing trigger.** A window
  closes when the watermark advances past `window_end`. The
  watermark is a function of the consumer-group state:
  `max_observed_event_time - lateness_tolerance`, where
  `max_observed_event_time` is the maximum event-time observed
  across all consumed records in the consumer-group. The
  per-partition combination policy — how the global watermark
  is derived when records arrive on multiple Kafka partitions —
  is deferred to OQ-B0S4.1; the v1 default committed there is
  **min-across-partitions** for deterministic closing. Records
  arriving after the watermark passes their window's
  `window_end` are **late** and are dropped (counted as
  `late_dropped_count` in evidence, not reflowed into a closed
  execution).

- **DD-S4.6** — **The closed-window invariant from ADR-0002
  carries forward.** A `dq_executions` row whose `window_end` is
  in the past is closed — its result does not change on later
  arrivals. Late-arrival reflow would break this invariant, which
  ADR-0003's append-only write model and ADR-0006's per-attempt
  dedup both rely on. Drop is the only option that preserves the
  invariant without committing a separate-execution-per-late-batch
  shape (which is allowed in principle but defers complexity that
  the inaugural kind does not need).

- **DD-S4.7** — **Window declaration lives in the source block.**
  Windowing is substrate-specific — set-mode rules have no
  windowing field (their windows arrive from the trigger);
  record-mode rules need a windowing field that lives alongside
  `topic` and `consumer_group` in the `source.type: kafka`
  block committed by ADR-0023. This keeps the declaration close
  to the substrate that motivates it and avoids a top-level
  field that is meaningful for only one source type.

- **DD-S4.8** — **Tumbling windows only in v1.** Sliding windows
  multiply evaluation counts (each record belongs to N windows);
  session windows require per-key state machines. Neither is
  needed for `record.schema_conformance`. Future record kinds
  that require sliding or session windows can extend the
  windowing-type enum additively (per ADR-0001's contract). The
  v1 commitment is tumbling-only; the type field is open for
  extension.

- **DD-S4.9** — **Extend ADR-0002's enum, not its formula.** The
  `trigger_source` enum gains a new value (`stream-watermark`)
  for record-mode; the formula's five-input shape stays. This
  is additive under ADR-0001's compatibility contract
  (additive-within-major), which governs schema-and-enum
  evolution platform-wide. ADR-0002's formula itself is not
  reopened — only the enum's permitted values expand. *(New
  contribution proposed here, requires review — extending
  ADR-0002's `trigger_source` enum is the first such extension
  since ADR-0002 landed.)*

---

## Considered Options

The four options below differ on the **windowing model** —
what makes one evaluation batch. All four assume window
declaration lives in the source block per DD-S4.7 (the location
is locked).

### Option A — Tumbling watermark-bounded windows (recommended)

**Shape.** The rule's `source` block carries a `window` object
declaring `type: tumbling`, a fixed `duration` (e.g., `1m`,
`5m`), and a `lateness_tolerance` (e.g., `30s`, `5m`). The
engine groups records into non-overlapping windows of `duration`
aligned to UTC epoch boundaries. A window closes when the
watermark (= max observed event-time across partitions, minus
`lateness_tolerance`) advances past `window_end`. At close, the
engine writes one `dq_executions` row keyed on
`execution_id = hash(ruleset_version, entity, window_start, window_end, "stream-watermark")`.
Late records (event_time in a closed window) are **dropped**
and counted as `late_dropped_count` in evidence.

```yaml
# Record rule with tumbling window (R1: illustration only)
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

**Cost.** Each evaluation batch = one window = one
`dq_executions` row. For a 1-minute tumbling window on a
moderate-traffic topic, that's 1440 executions/day per
rule-entity — manageable for `dq_executions`'s append-only write
model. Watermark-bounded closing is deterministic. Tumbling does
not multiply evaluation counts (no overlap). Late-drop preserves
ADR-0002's closed-window invariant without complicating
ADR-0003's write model.

**Verdict.** Recommended.

### Option B — Sliding watermark-bounded windows

**Shape.** Same as Option A but windows overlap: a sliding
window of `duration: 5m`, `slide: 1m` produces one window every
minute, each covering 5 minutes — so each record belongs to 5
windows.

**Cost.** Multiplies evaluation counts by `duration/slide`. For
the example above, 5× the executions, 5× the result rows. No
benefit for `record.schema_conformance` (a per-record validation
needs no overlap). Adds complexity (per-record window-membership
computation) that the inaugural kind does not need. Late-arrival
handling is more complex — a late record may need to land in
multiple windows.

**Verdict.** Rejected for v1. Future kinds that need sliding
windows (e.g., a rate-over-window aggregate) can extend the
windowing-type enum.

### Option C — Per-record (no window — each record is its own evaluation)

**Shape.** No `window` field. Each record produces one
`dq_executions` row keyed on
`execution_id = hash(ruleset_version, entity, kafka_offset, "", "stream-record")`
or similar per-record identity. The engine processes records as
they arrive.

**Cost.** Explodes `dq_executions` — one row per record. For a
topic emitting 100 records/sec, that's 8.6M rows/day per
rule-entity. `dq_executions`'s append-only write model under
ADR-0003 is not designed for that scale; partition pruning and
result aggregation become much harder. The execution-as-batch
abstraction breaks down — every record is its own "run",
contradicting the platform's mental model of an execution as a
discrete evaluation unit. Also blocks B0-S6's failure-scope
aggregation (per-record failures already are the per-execution
result, with no aggregation seam).

**Verdict.** Rejected. The execution unit is the batch, not the
record. Per-record semantics are realised by the kind's handler
inside a windowed batch, not by collapsing windows to
size-1.

### Option D — Manual / operator-triggered windows

**Shape.** Record-mode rules accept manual triggers (like
set-mode), where the operator sends a trigger with explicit
`window_start` and `window_end`. The engine consumes the Kafka
topic for that exact time range. No automatic window closing —
operator drives every batch.

**Cost.** Defeats the purpose of stream-based validation (no
continuous monitoring; record-mode becomes a polled BigQuery
analogue with extra plumbing). Inappropriate for the
"continuous monitoring of stream data" use case Wave-S exists
to serve. May be useful as an additional trigger path
(operator-rerun for record-mode) but cannot be the default.

**Verdict.** Rejected as the default. May resurface as a
record-mode `operator-rerun` analogue in a future ADR, but the
default windowing model must support continuous, automatic
evaluation.

---

## Recommendation

**Pick Option A — Tumbling watermark-bounded windows.**

Rationale, tied directly to drivers:

- **DD-S4.1 (foundational triplet honoured).** Windowing applies
  to record-mode rules only; ADR-0002's set-mode formula is
  unchanged.
- **DD-S4.2 (execution_id contract).** Record-mode reuses the
  five-input formula structure from ADR-0002:
  `hash(ruleset_version | entity | window_start | window_end | trigger_source)`.
  `window_start`/`window_end` are watermark timestamps (RFC3339).
  `trigger_source` is the new enum value `stream-watermark`. The
  formula is identical in structure across modes; only enum
  values and time-source semantics differ.
- **DD-S4.3 (determinism).** Watermark-based closing is a
  deterministic function of the input stream (event-time +
  lateness tolerance). Same stream → same windows → same
  `execution_id`s.
- **DD-S4.4 (inaugural kind support).** Tumbling windows support
  `record.schema_conformance` cleanly — each record in the
  window is schema-validated; the window aggregates results into
  one `dq_executions` row plus N `dq_check_results` rows (per
  B0-S6's eventual failure-aggregation policy).
- **DD-S4.5 / DD-S4.6 (closing and late-arrival).** Late records
  are dropped and counted in evidence; the closed-window
  invariant ADR-0002 and ADR-0003 rely on is preserved.
- **DD-S4.7 (declaration location).** Windowing lives inside the
  `source.type: kafka` block, alongside `topic` and
  `consumer_group`. The `source.type: bigquery` block has no
  windowing field — set-mode windows arrive from the trigger.
- **DD-S4.8 (tumbling-only v1).** Sliding and session windows
  defer to future kind extensions; the windowing-type enum is
  open for additive extension.
- **DD-S4.9 (additive enum extension).** `trigger_source` enum
  gains `stream-watermark`; formula structure unchanged.

### The record-mode `execution_id` formula

Same five-input shape as ADR-0002. Inputs for record-mode:

| Input | Source | Notes |
|---|---|---|
| `ruleset_version` | Manifest version active when the window closed | Identical semantics to set-mode |
| `entity` | Rule's `entity:` field | Identical to set-mode |
| `window_start` | Tumbling window start (RFC3339, UTC, aligned to epoch) | Derived from event-time, not wall-clock |
| `window_end` | `window_start + duration` (RFC3339) | Same |
| `trigger_source` | `stream-watermark` (new enum value) | Additive extension to ADR-0002's enum |

Computation, output, and consumer impact are identical to
ADR-0002: pipe-separated UTF-8, sha256_hex, opaque to consumers
who treat `execution_id` as a stable key.

### Window field shape

The `source.type: kafka` block from ADR-0023 gains a required
`window` sub-object:

| Field | Required | Type | Description |
|---|---|---|---|
| `window.type` | yes | const `tumbling` | Discriminator. v1 supports `tumbling` only; future enum extensions allowed. |
| `window.duration` | yes | string (duration) | Window size (e.g., `1m`, `5m`, `1h`). Aligned to UTC epoch — windows start at duration boundaries. |
| `window.lateness_tolerance` | yes | string (duration) | Watermark lag. A window closes when the watermark (`max_observed_event_time - lateness_tolerance` over the consumer-group state, partition-combined per OQ-B0S4.1's policy) crosses `window_end`. |

Both `duration` and `lateness_tolerance` use the Go-style
duration format (`30s`, `5m`, `1h`). Minimum `duration` is
`1s`; minimum `lateness_tolerance` is `0s`. A `0s` tolerance
means the watermark equals `max_observed_event_time` — the
window closes on any forward event-time movement past
`window_end`, including a single out-of-order record from the
future. `0s` is recommended only for strictly monotonic
event-time streams; non-monotonic streams should use a
non-zero tolerance matched to expected out-of-order skew.

**Alignment of `duration` to UTC epoch.** Windows align via
floor-of-event-time-by-duration:
`window_start = floor(event_time / duration) * duration`
(measured in nanoseconds since epoch). This works for any
`duration` ≥ `1s`, including non-epoch-divisor values like
`7m` or `13m` — windows simply start at the floor-aligned
timestamp. Maximum values for both `duration` and
`lateness_tolerance` are out of scope here and may be enforced
by B0-S7 cost guardrails.

### Lint cross-checks added

On top of ADR-0021's four, ADR-0022's two, and ADR-0023's two
(eight total at ADR-0023's promotion), B0-S4 adds:

- **#9 — `source.window` present iff `source.type` is
  record-mode.** A `source.type: bigquery` rule that declares
  `window` fails lint (set-mode windows come from the trigger);
  a `source.type: kafka` rule that omits `window` fails lint.
- **#10 — `window.type` is in the supported enum.** v1 supports
  only `tumbling`; future enum values extend additively. Lint
  reads the supported enum from a contract — initially hardcoded
  in `tools/lint/`, eventually consumable from a shared
  declaration with the engine.
- **#11 — `window.duration` ≥ `1s` and `window.lateness_tolerance`
  ≥ `0s`.** Schema-side validation; the duration grammar is
  enforced by JSON Schema's pattern constraint.

Ten cross-checks total at this ADR's promotion.

**Schema dispatch — alternatives considered for `window` placement:**

- **Top-level `window:` on the rule.** Rejected: meaningful only
  for record-mode rules; set-mode rules would carry an empty or
  forbidden field. Asymmetry between modes leaks to the top
  level.
- **Inside `checks[].params.window`.** Rejected: per-check
  windowing contradicts the rule-as-one-window-discipline framing.
  All checks in a record-mode rule share the rule's window.
- **Inside `source.type: kafka` block** (recommended). Selected:
  windowing is substrate-specific, and the only substrate that
  needs it is the record-mode Kafka source. Future record-mode
  substrates that need windowing would gain their own `window`
  sub-object under their `source.type` variant — **this
  introduces per-substrate redeclaration redundancy if a second
  record-mode substrate variant lands**, with each variant
  carrying its own copy of the window shape. The trade-off
  (per-substrate redundancy vs. mode-asymmetric top-level
  field) is accepted in this study; if the redundancy becomes
  operationally painful when a second record-mode substrate
  exists, the placement is revisited in the ADR that
  introduces that substrate.

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Windowing model | Tumbling, watermark-bounded |
| Window declaration location | Inside `source.type: kafka` block |
| Window v1 enum | `tumbling` only (additive-extensible) |
| Window close trigger | Watermark > `window_end`; watermark = `max_observed_event_time - lateness_tolerance` over the consumer-group state (partition-combined per OQ-B0S4.1, v1 default = min-across-partitions) |
| Late-arrival policy | Dropped; counted as `late_dropped_count` in evidence |
| `execution_id` formula | Same shape as ADR-0002; `window_start`/`window_end` are watermark timestamps; `trigger_source` = `stream-watermark` (new enum value) |
| Closed-window invariant | Preserved per ADR-0002 / ADR-0003 |
| Lint cross-checks added | #9, #10, #11 (ten total at ADR-0024's promotion) |

---

## Consequences

### Cross-cutting consequences

- **C-B0S4.1** — **Rule schema v2's `source.type: kafka` variant
  gains a required `window` sub-object.** Adds to the
  ADR-0023's `source.type: kafka` shape: `{ topic,
  consumer_group, window: { type, duration, lateness_tolerance } }`.
  Lands in the combined implementation commit per ADR-0022 §C-B0S2.2
  path (a); if any prior v2 artefact ships first, the v3 schema
  bump contingency from ADR-0022 applies. *(New contribution
  proposed here, requires review.)*

- **C-B0S4.2** — **ADR-0002's `trigger_source` enum gains
  `stream-watermark`.** Additive under ADR-0002's enum-evolution
  pattern (which the ADR explicitly anticipates for new trigger
  paths). The set-mode trigger API continues to reject
  `stream-watermark` requests; record-mode produces the value
  internally when a window closes. No set-mode contract is
  reopened. *(New contribution proposed here, requires review.)*

- **C-B0S4.3** — **Record-mode `execution_id` reuses ADR-0002's
  five-input formula structure.** No new identity rule; the
  formula is the same `sha256_hex(ruleset_version | entity |
  window_start | window_end | trigger_source)`. Record-mode and
  set-mode share the `execution_id` namespace; collisions
  between modes are not a concern because `entity` values are
  distinct across modes (record-mode entities target Kafka
  topics; set-mode entities target BigQuery tables) and
  `trigger_source` adds further entropy.

- **C-B0S4.4** — **The closed-window invariant from ADR-0002 is
  preserved.** Late records (event_time in a closed window) are
  dropped and counted in evidence; they never reflow into a
  closed `dq_executions` row. ADR-0003's append-only write model
  and ADR-0006's per-attempt deduper continue to assume
  closed-window finality.

- **C-B0S4.5** — **Watermark-based closing is deterministic.**
  Same input stream + same ruleset version + same window
  configuration → same set of `execution_id`s. Operator reruns
  (replaying a Kafka offset range against the same configuration)
  produce identical `execution_id`s, matching ADR-0002's
  determinism promise.

- **C-B0S4.6** — **Lint binary gains three cross-checks (#9, #10,
  #11).** On top of ADR-0021's four, ADR-0022's two, and
  ADR-0023's two. Ten cross-checks total at ADR-0024's promotion.

- **C-B0S4.7** — **Phase β progression: B0-S5, B0-S6, B0-S7
  inherit B0-S4's window-shape contract.** B0-S5 (unified-vs-
  parallel runner) consumes windows as the runner's evaluation
  unit. B0-S6 (failure scope aggregated) uses the window as the
  aggregation boundary — N per-record failures within one window
  map to one entity-status outcome. B0-S7 (cost guardrails)
  binds lag and lateness budgets to window discipline (e.g.,
  consumer lag thresholds expressed as "watermark behind
  wall-clock by ≥ X").

- **C-B0S4.8** — **The protocol-fit clause from ADR-0023 retires
  its provisional status.** ADR-0023 §Decision (Protocol-fit
  forward-looking, until B0-S4 commits) said the Kafka pick
  rested on protocol-fit "until B0-S4 finalises offset/watermark
  semantics". B0-S4 finalises those semantics consistent with
  the Kafka pick — watermark-bounded windows over partitioned
  topics, with offset-range replay producing deterministic
  re-evaluation. ADR-0023's protocol-fit clause is now committed,
  not provisional.

### Per-artefact consequences

- **`engine/internal/dsl/schema/v2.schema.json`** — the
  `source.type: kafka` variant gains a required `window` sub-
  object with the shape committed in §Recommendation.
  `additionalProperties: false` preserved.

- **`rules/_schema/v2.schema.json`** — byte-equal mirror.

- **`engine/internal/dsl/catalog/v1.yaml`** (committed by
  ADR-0022) — unchanged. The `record.schema_conformance` entry
  already declares `source_mode: record`; the engine's
  source-fetching layer reads the rule's `source.window` to know
  the windowing discipline.

- **`tools/lint/`** — three new cross-checks (#9, #10, #11) on
  top of the eight from prior ADRs.

- **`engine/internal/eval/evaluator.go`** — dispatcher unchanged
  by this ADR. The record-mode evaluator path (lands at the
  combined implementation commit per ADR-0023 §C-B0S3.7) consumes
  the window configuration to drive the consumer-group offset
  reads and watermark tracking.

- **`engine/internal/eval/record_schema_conformance.go`** —
  handler unchanged; the handler still validates each record
  against `params.schema`. The handler runs once per window,
  iterating the records in the window's offset range. The
  per-window output (N validated records, M violations) is
  passed to the result-write layer; the entity-status
  aggregation policy is B0-S6's responsibility.

- **`docs/adr/0002-run-identity-and-idempotency.md`** —
  scope-note already in place (set-oriented). The `trigger_source`
  enum extension to `stream-watermark` is additive per ADR-0002's
  own anticipated evolution pattern; no scope-note amendment
  required.

- **No changes to ADR-0003, ADR-0004, ADR-0006 contracts on the
  set side.** Record-mode result-write semantics, status mapping,
  and dedup are B0-S5, B0-S6 territory.

---

## Open Questions

- **OQ-B0S4.1** — **Per-partition watermark combination.** Kafka
  topics are partitioned; each partition emits records with its
  own event-time progression and therefore its own
  partition-level watermark. Three combination policies are
  coherent:
  **min-across-partitions** (global watermark = the slowest
  partition's watermark — the most conservative and most
  deterministic choice; a slow partition holds back window
  closing for all faster partitions, but no record is dropped
  for arriving "late" relative to a faster partition's
  progress),
  **max-across-partitions** (global watermark = the fastest
  partition's watermark — closes windows fastest but risks
  dropping records from slower partitions when closing fires
  before those partitions' records arrive), and
  **per-partition** (each partition closes its own windows
  independently; aggregation across partitions happens at
  result-write time).
  *Out of scope for current cycle;* the v1 minimum-viable
  implementation commits **min-across-partitions** as the
  default for determinism. The other two policies remain as
  future tuning options when concrete operational signal (e.g.,
  unacceptable closing lag on a topic with one slow partition)
  motivates them.

- **OQ-B0S4.2** — **Window alignment offset.** Tumbling windows
  are aligned to UTC epoch boundaries by default. Whether the
  rule can declare a non-zero alignment offset (e.g., windows
  start at :30 past the minute instead of :00) is a future
  enrichment. *Out of scope for current cycle;* epoch-aligned
  is the v1 default.

- **OQ-B0S4.3** — **Watermark heartbeats on idle topics.** A
  topic with no traffic never advances its watermark, so windows
  never close. The standard mitigation is periodic "heartbeat"
  records or wall-clock-based watermark advancement when no
  events are received for a configured idle duration. This is a
  runtime concern that interacts with determinism (introducing
  wall-clock dependency for idle topics). *Out of scope for
  current cycle;* the v1 implementation **rejects idle topics**
  — a record-mode rule whose topic has no traffic for longer
  than its `lateness_tolerance` fails its current window and
  emits an operational alert. Idle-watermark-advance with
  documented determinism caveats is a future enrichment when
  concrete operational signal motivates it.

- **OQ-B0S4.4** — **`dq_executions` partition column for
  record-mode runs.** Set-mode runs partition `dq_executions` by
  `window_end` date per ADR-0003. Record-mode runs land their
  `window_end` (watermark timestamp) in the same column —
  partition pruning works identically. Whether B0-S6 surfaces a
  separate partition discriminator (e.g., `mode` column) is
  B0-S6's question. *Defer to B0-S6.*

- **OQ-B0S4.5** — **Sliding and session windows as future
  catalog-coupled extensions.** A future record kind that
  requires sliding windows (e.g., a rate-over-5m kind) would
  extend the `window.type` enum to `sliding` and require lint
  cross-check #10 to accept it. The catalog could carry a
  `required_window_type` field per kind to enforce
  kind-to-window-type compatibility. *Out of scope for current
  cycle;* the inaugural kind needs tumbling only, and adding
  sliding requires both a new kind motivating it and a window-
  type-per-kind cross-check.

- **OQ-B0S4.6** — **Operator-rerun for record-mode.** A
  record-mode operator-rerun replays a Kafka offset range
  against a captured window range. Whether this is exposed as a
  separate `trigger_source: operator-rerun` (matching set-mode's
  rerun path) or a new `stream-rerun` value is a future
  consequence of the operator-rerun design. *Defer to a future
  ADR-0002 follow-up on record-mode rerun semantics;* the v1
  shape commits `stream-watermark` only.

- **OQ-B0S4.7** — **Cost binding to window discipline.** B0-S7
  (cost guardrails) will bind throughput and lag budgets to
  window discipline (e.g., max consumer lag = N × duration).
  This study does not pre-commit the formula; B0-S7 owns it.
  *Defer to B0-S7.*

---

## Promotion target

**Target:** `docs/adr/0024-window-semantics.md`.

This study promotes to **ADR-0024** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0024 is the first Phase β ADR of
Wave-S — opened by the partial-Wave-S gate that ADR-0023's
promotion just closed. Per ADR-0020 §Decision (Per-item ADR
numbering), the `0024` slot is descriptive; if the ADR-0010
substrate-posture amendment (provisional from ADR-0023 §Decision
(ADR-0010 amendment)) lands first, B0-S4 promotes to ADR-0025
and the per-item slugs shift in lockstep.

ADR-0024's promotion commit lands the artefacts committed in
§Consequences above:

1. The rule schema v2's `source.type: kafka` variant extension
   (adds required `window` sub-object), folded into the combined
   v2 schema implementation alongside ADR-0021's `mode:`,
   ADR-0022's `params:`, and ADR-0023's `source:`.
2. ADR-0002's `trigger_source` enum extended to include
   `stream-watermark`.
3. Three new lint cross-checks (#9, #10, #11) in `tools/lint/`.
4. Engine record-mode evaluator path consumes window configuration
   to drive offset reads and watermark tracking (combined
   implementation commit).

Per R8, the future ADR-0024 will be rewritten from this study,
not linked back to it.
