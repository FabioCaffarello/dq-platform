<!-- path: studies/decisions/2026-05-24-b0-s7-record-mode-cost-guardrails.md -->

# B0-S7 — Record-Oriented Cost Guardrails

## Metadata

- **B-item reference:** B0-S7 (Wave-S Phase β, item 4 of 4 — **the
  full-Wave-S-gate-closing study**)
- **Status:** resolved-study (Wave-S, B0-S7; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-24
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch); foundational triplet
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) /
  [ADR-0022](../../docs/adr/0022-kind-catalog.md) /
  [ADR-0023](../../docs/adr/0023-sources-schema.md);
  [ADR-0024](../../docs/adr/0024-window-semantics.md) (windows + late-drop);
  [ADR-0025](../../docs/adr/0025-aggregation-and-runner-shape.md)
  (parallel runners + writer-coupling acknowledged);
  [ADR-0026](../../docs/adr/0026-failure-scope-aggregated.md)
  (per-record evidence sample size as cost dimension);
  [ADR-0018](../../docs/adr/0018-environment-configuration-model.md)
  (env config model — PAT-4 typed `EnvConfig`);
  [ADR-0019](../../docs/adr/0019-infrastructure-tooling.md) (Kustomize
  per-env overlays under `deploy/overlays/{local,qa,prod}/`).
- **Downstream open:** B1-2 (BigQuery cost ceilings — open at B1;
  parallel to this ADR for the set-mode side; their composition
  story is committed here from the record-mode side).
- **Promotion target:** `docs/adr/0027-record-mode-cost-guardrails.md`
  (subject to ADR-0020 §Decision (Per-item ADR numbering); `0027`
  assumes the forthcoming ADR-0010 amendment (Kafka substrate
  row, flagged by ADR-0023 §C-B0S3.3) has not interleaved).
- **Loop discipline:** same as B0-S1–S6 — `/resolve-b0` study →
  `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.
- **Significance — gate closure.** B0-S7 is the **fourth and
  final Phase β study**. Its promotion to ADR-0027 closes the
  **full Wave-S gate** per ADR-0020 §Decision (Full-Wave-S gate
  criterion): all seven B0-S items at `resolved-adr`, all seven
  ADRs merged into `docs/adr/`. After ADR-0027 promotes, Wave-S
  has no remaining decision-side work; the remaining surface is
  the combined implementation commit that lands ADRs 0021–0027
  artefacts together.

---

## Context

ADR-0020 §B0-S7 commits the scope:

> Decides the **throughput**, **backpressure**, **dead-letter**,
> and **consumer-lag** ceilings that record-mode must respect
> under each environment (per ADR-0018); how the guardrails are
> enforced (engine-side, broker-side, or both); how the
> guardrails compose with the existing BigQuery cost ceilings
> (B1-2) so that an entity with both set-mode and record-mode
> rules respects both budgets.

Three Phase β commitments flow into this study:

- **ADR-0025 §C-B0S5.6 (per-runner shape for cost guardrails)**
  committed that record-mode lag is consumer-group lag tracked
  by the record runner; set-mode cost is BigQuery slot
  consumption tracked by the set runner. The per-runner
  isolation lets budgets be enforced and signalled per mode
  independently.
- **ADR-0025 §DD-S5.8 / §OQ-B0S5.3 (writer-coupling concern)**
  acknowledged that writer-side queue saturation propagates
  across modes; resolving writer-side isolation is deferred to
  a follow-up if operational signal motivates it. B0-S7 either
  surfaces this as a cost-guardrail signal or commits an
  isolation path.
- **ADR-0026 §C-B0S6.6 / §DD-S6.9 (late-drop as cost signal)**
  committed that late-dropped records are excluded from
  aggregation; the lateness-rate alerting and cost guardrails
  live in B0-S7, not in B0-S6's per-check status determination.
  B0-S7 commits the late-drop ceiling and the alerting trigger.
- **ADR-0026 §C-B0S6.9 (evidence_sample_size as cost
  dimension)** committed that the per-kind
  `evidence_sample_size` from B0-S6 binds to per-runner storage
  budgets that B0-S7 commits.

The current engine env config (per ADR-0018) is typed
`EnvConfig` in `engine/internal/env/config.go`, with per-env
constants in `local.go` / `qa.go` / `prod.go`. ADR-0023 §C-B0S3.5
removed `SourceProject` / `SourceDataset` from those files at the
combined implementation commit; the env config now carries only
deployment-level identifiers (log level, env name, HTTP address,
emulator overrides). B0-S7 **adds** record-mode cost-ceiling
fields to that typed struct, with per-env values in the
respective files, exposed to Kubernetes via Kustomize overlays
per ADR-0019.

Four interlocking sub-decisions live inside B0-S7's scope:

1. **The cost dimensions to bound** — which knobs matter for
   record-mode operational cost and signal.
2. **Per-env ceiling location** — env-config struct (engine-side),
   Kustomize overlays (deploy-side), rule YAML (rule-side), or
   layered.
3. **Enforcement location** — engine-side, broker-side (Kafka
   quotas), or both.
4. **Composition with B1-2** — how an entity with rules in both
   modes respects both budgets.

Platform principles bearing on the design: **foundation 01
§"Cost"** (cost is a first-class constraint — partition
discipline, query templates, dry-run visibility, concurrency
budgets, evidence-retention limits are platform design, not
later hardening); **P5** (evolution contract-driven — env-config
extensions live under ADR-0018; overlay extensions under
ADR-0019); **P2 / foundation 01 §"Determinism"** (cost
enforcement must not introduce non-determinism in evaluation —
guardrails throttle ingestion, never alter the result).

---

## Decision Drivers

- **DD-S7.1** — **Honour foundation 01 §"Cost".** Record-mode
  cost is a first-class constraint, not later hardening. Every
  cost dimension that materially affects operational cost or
  blast-radius is committed at this ADR's promotion — not
  deferred to incident-response.

- **DD-S7.2** — **Honour R3 — B1-2 stays scope-noted.** The B1-2
  row (BigQuery cost ceilings, open at B1) is pinned to set-mode
  per ADR-0020 §DD-S.6 / §C-S.1 of the Wave-S launch ADR. B0-S7
  is the **record-mode parallel**, not an amendment to B1-2.
  Both rows coexist; the composition story lives here.

- **DD-S7.3** — **Honour ADR-0018 (typed env config).** Per-env
  cost ceilings live as typed fields on the `EnvConfig` struct
  in `engine/internal/env/config.go`, with per-env values in
  `engine/internal/env/{local,qa,prod}.go`. Same pattern as the
  log-level, HTTP-address, and emulator-override fields that
  already exist. *(New contribution proposed here, requires
  review — extending `EnvConfig` with cost fields.)*

- **DD-S7.4** — **Honour ADR-0019 (Kustomize overlays).** Per-env
  cost values surface to Kubernetes via the existing overlay
  pattern at `deploy/overlays/{local,qa,prod}/`. The combined
  implementation commit lands the overlay extensions alongside
  the env-config field additions.

- **DD-S7.5** — **Inherit ADR-0026 C-B0S6.9 — `evidence_sample_size`
  as a cost dimension.** Per-kind catalog defaults declare
  evidence-sample sizes that operators may override per rule;
  the env config carries a hard upper-bound ceiling per env
  (`max_evidence_sample_size`). Rule overrides that exceed the
  env ceiling are rejected by the loader at boot — same pattern
  as ADR-0024's idle-topic rejection.

- **DD-S7.6** — **Surface the writer-coupling concern from
  ADR-0025 §DD-S5.8 / §OQ-B0S5.3.** The shared writer is a
  cross-mode coupling acknowledged in ADR-0025 but not resolved
  there. B0-S7 commits a writer-queue saturation threshold
  (`writer_queue_saturation_threshold`) per env that, when
  crossed, triggers per-runner backpressure (each runner slows
  its consumption rate until the writer queue drops below the
  threshold). This **converts the cross-mode coupling from
  silent failure into observable backpressure** — both runners
  feel the writer pressure, but they feel it as a known,
  monitored signal rather than as silent latency growth.
  *(New contribution proposed here, requires review.)*

- **DD-S7.7** — **Surface late-drop as a cost-guardrail signal
  per ADR-0026 §DD-S6.9.** B0-S6 deferred lateness-rate
  alerting to B0-S7. B0-S7 commits a `max_late_drop_rate` per
  env that triggers an operational alert per ADR-0006 when
  exceeded over a sliding observation window. Late-drop in
  evidence is the per-window data; the alert is the cumulative
  signal that the data is degrading. *(New contribution
  proposed here, requires review.)*

- **DD-S7.8** — **Composition with B1-2 is independent budgets,
  not unified.** An entity with both set-mode rules (using
  BigQuery, bounded by B1-2 when resolved) and record-mode
  rules (using Kafka, bounded by B0-S7) respects **both
  budgets independently**. The platform does not (yet) carry a
  per-entity total-cost budget that spans modes. Each mode's
  runner enforces its own budgets; an entity's "total platform
  cost" is the sum of mode-specific costs. A future unified-
  per-entity-budget mechanism is a follow-up if operational
  signal motivates it. *(New contribution proposed here,
  requires review.)*

- **DD-S7.9** — **Engine-side enforcement only in v1.**
  Broker-side enforcement (Kafka quotas, broker-level rate
  limits) is a substrate-coupling that adds operational
  complexity (broker configuration, broker-engine coordination
  on quota state). v1 commits engine-side enforcement only;
  broker-side is a future enrichment when operational signal
  motivates it. Engine-side enforcement is sufficient because
  the engine controls the consumer-group read rate; backpressure
  at the engine throttles upstream demand naturally.

- **DD-S7.10** — **Cost enforcement throttles, never alters
  results.** Guardrails slow ingestion (backpressure) or reject
  configurations (loader-side ceiling rejection) — they never
  change a check's per-record evaluation outcome. This preserves
  P2 (determinism): the same input stream + same configuration
  produces the same `execution_id` set and the same per-check
  statuses, regardless of whether the engine throttled. *(New
  contribution proposed here, requires review.)*

- **DD-S7.11** — **Dead-letter v1 is "drop and count", matching
  late-drop pattern from ADR-0024.** Records the handler cannot
  process (malformed record, schema parse error before
  validation, unbounded recursion in validator) are dropped
  and counted as `dead_letter_count` in evidence. A
  `max_dead_letter_rate` per env triggers an operational alert
  per ADR-0006. Dead-letter Kafka topic routing is a future
  enrichment; v1 keeps the substrate dependency minimal.

---

## Considered Options

The four options below differ on **where the cost ceilings
live** and **how they compose with per-rule overrides**. All
four assume the cost dimensions themselves (throughput, lag,
late-drop rate, dead-letter rate, evidence sample size, writer
saturation) are locked — the variation is location.

### Option A — Layered: per-env env-config ceilings + per-rule overrides (recommended)

**Shape.** Per-env cost ceilings live on `EnvConfig` (typed
struct per ADR-0018), with values in `engine/internal/env/{local,qa,prod}.go`
and Kustomize overlay extensions in `deploy/overlays/{local,qa,prod}/`
(per ADR-0019). Per-rule overrides exist for the dimensions
already declared per-kind by ADR-0026 (`evidence_sample_size`).
**Env ceilings are hard upper bounds** on per-rule overrides —
a rule override that exceeds the env ceiling is rejected by the
loader at boot.

```go
// EnvConfig extension (R1: illustrative shape, not committed code)
type RecordModeCost struct {
    MaxConsumerLag                 int     // records; consumer-group lag above this triggers alert
    MaxLateDropRate                float64 // 0.0–1.0; per-window late-drop rate above this triggers alert
    MaxDeadLetterRate              float64 // 0.0–1.0; per-window dead-letter rate above this triggers alert
    MaxEvidenceSampleSize          int     // hard upper bound on rule overrides
    WriterQueueSaturationThreshold float64 // 0.0–1.0; writer queue ratio above this triggers per-runner backpressure
    MaxConsumerThroughputRecsSec   int     // throttle target; engine-side enforcement
}

type EnvConfig struct {
    // ... existing fields (Name, LogLevel, HTTPAddr, emulator overrides) ...
    RecordModeCost RecordModeCost
}
```

Per-env defaults (illustrative; final numbers picked at the
combined implementation commit):

| Dimension | local | qa | prod |
|---|---|---|---|
| `MaxConsumerLag` | 1,000,000 | 100,000 | 10,000 |
| `MaxLateDropRate` | 0.50 | 0.10 | 0.01 |
| `MaxDeadLetterRate` | 0.50 | 0.05 | 0.01 |
| `MaxEvidenceSampleSize` | 100 | 50 | 20 |
| `WriterQueueSaturationThreshold` | 0.90 | 0.75 | 0.50 |
| `MaxConsumerThroughputRecsSec` | unlimited | 50,000 | 10,000 |

**Per-rule overrides:** the only currently-overridable dimension
is `evidence_sample_size` (per B0-S6 catalog `params.aggregation`
override). The override floor is the rule's value; the env
ceiling is the upper bound. A rule overriding to 200 in a `prod`
env where `MaxEvidenceSampleSize = 20` is rejected by the
loader at boot.

**Enforcement model:**
- **Engine-side, runner-loop:** the record runner monitors
  consumer lag, dead-letter rate, late-drop rate, writer-queue
  saturation. Exceeding any per-env threshold triggers either
  per-runner backpressure (saturation) or an operational alert
  (rates).
- **Engine-side, loader:** the loader rejects rule overrides that
  exceed env ceilings at boot, matching ADR-0024's idle-topic
  rejection pattern.

**Cost.** Adds a typed `RecordModeCost` sub-struct to
`EnvConfig`; adds per-env values to local/qa/prod constants;
adds overlay patches for qa/prod (local uses defaults). Lint
gains no new cross-checks (env-config is engine-side, not
rule-side). Loader gains a startup check for rule overrides
against env ceilings.

**Verdict.** Recommended.

### Option B — All per-rule (no env ceiling)

**Shape.** Cost ceilings live on each rule (in `params.cost` or
catalog defaults); no env-config involvement. Each rule
declares its own consumer-lag tolerance, late-drop tolerance,
etc.

**Cost.** Authors must understand operational characteristics
per env for every rule. A rule that ships to all three envs
needs three sets of values (or a single value that doesn't fit
qa or prod). Removes the operator's ability to tighten ceilings
platform-wide for a new env without editing every rule.
Operationally fragile.

**Verdict.** Rejected. Operational ceilings are properties of
the deployment, not the rule.

### Option C — All env-side (no per-rule override)

**Shape.** Cost ceilings live only on `EnvConfig`. Rules cannot
override; ADR-0026's `evidence_sample_size` per-rule override
is effectively reset to env-only.

**Cost.** Reverses ADR-0026's commitment that operators can tune
`evidence_sample_size` per rule for debugging or compliance.
For dimensions like consumer-lag and throughput, per-rule
override doesn't make sense; for `evidence_sample_size`, it
does — the cost vs. operability tradeoff varies per kind and
per entity.

**Verdict.** Rejected. Loses ADR-0026's per-rule expressiveness
for the one dimension where it matters.

### Option D — Broker-side enforcement (Kafka quotas)

**Shape.** Engine reads from Kafka without any throttling logic;
broker-level Kafka quotas (per consumer-group, per user, per
client-id) enforce the rate limits. No engine-side cost
enforcement; engine emits cost-related telemetry only.

**Cost.** Adds substrate coupling (the engine's cost story
depends on broker configuration). Broker-side quotas require
operational coordination between the engine team and broker
operators. Engine cannot back-pressure its own writer based on
broker quotas (quotas throttle inbound only). The writer-
coupling concern from DD-S7.6 is not addressable broker-side.

**Verdict.** Rejected for v1; reserved as a future enrichment
when broker-side quotas become operationally available.

---

## Recommendation

**Pick Option A — Layered: per-env env-config ceilings + per-rule
overrides bounded by env ceilings.**

Rationale, tied directly to drivers:

- **DD-S7.1 (cost first-class).** Six cost dimensions
  (consumer lag, late-drop rate, dead-letter rate, evidence
  sample size, writer-queue saturation, consumer throughput)
  are explicitly bounded at this ADR's promotion. None are
  deferred to incident-response.
- **DD-S7.2 (R3 / B1-2 stays scope-noted).** This ADR is the
  record-mode parallel to B1-2; the composition story (DD-S7.8)
  commits independent per-mode budgets without amending B1-2.
- **DD-S7.3 (ADR-0018 typed env config).** Ceilings live as a
  typed `RecordModeCost` sub-struct on `EnvConfig` — same
  pattern as existing log-level, HTTP-address, emulator-
  override fields.
- **DD-S7.4 (ADR-0019 overlays).** Per-env values surface to
  Kubernetes via the existing `deploy/overlays/{local,qa,prod}/`
  pattern; the combined implementation commit lands overlay
  extensions alongside env-config field additions.
- **DD-S7.5 (evidence_sample_size ceiling).** Per-kind catalog
  defaults from ADR-0026; per-rule overrides bounded by
  per-env `MaxEvidenceSampleSize` hard ceiling.
- **DD-S7.6 (writer-coupling surfaced).** `WriterQueueSaturationThreshold`
  converts the silent cross-mode coupling into observable
  per-runner backpressure.
- **DD-S7.7 (late-drop alerted).** `MaxLateDropRate` per env
  triggers operational alert per ADR-0006 when exceeded over
  the sliding observation window.
- **DD-S7.8 (independent composition with B1-2).** Each mode's
  budgets enforce independently; no unified-entity budget in
  v1.
- **DD-S7.9 (engine-side enforcement).** v1 keeps cost
  enforcement engine-side only; broker-side deferred.
- **DD-S7.10 (cost throttles, never alters).** Determinism
  preserved.
- **DD-S7.11 (dead-letter drop and count).** Matches late-drop
  pattern from ADR-0024; dead-letter Kafka topic deferred.

### Operational alert routing per ADR-0006

Each ceiling crossing produces an operational alert per
ADR-0006's existing category mapping (operational alerts route
to the `operational` channel category from `_owners.yaml`).
Alert dedup follows ADR-0006's per-attempt deduper. The
ceiling-crossing event carries:

- The dimension name (`max_consumer_lag`, `max_late_drop_rate`,
  etc.).
- The threshold value and the observed value.
- The runner identity (always `record` for B0-S7-sourced
  alerts).
- The window or observation period (for rate-based dimensions).

### Per-runner backpressure for writer saturation

When `writer_queue_saturation` ≥ `WriterQueueSaturationThreshold`,
each runner responds with a mode-appropriate back-off — the two
mechanisms are different because the runners' work shapes are
different:

- **Record runner — consumer-level back-off.** The record runner
  controls the Kafka consumer-group read rate. It slows
  consumption by a backoff factor. Records the runner has not
  yet consumed remain in Kafka.
- **Set runner — HTTP-level back-off.** The set runner is
  trigger-driven (per ADR-0014's HTTP trigger handler). It does
  not poll for work; it cannot "slow consumption" the way the
  record runner does. Instead, the HTTP trigger handler returns
  HTTP 503 (Service Unavailable) with a `Retry-After` header on
  incoming triggers while the writer is saturated. The trigger
  client (scheduler, manual API, operator-rerun path per
  ADR-0002) honours `Retry-After`; the back-off propagates
  upstream.
- **Backoff decay.** Both back-offs decay as the writer queue
  drains below the threshold.
- **Alerting.** An operational alert fires once per saturation
  event, deduped per ADR-0006.

This is a **per-runner observable response to a shared writer
signal**, not writer-side isolation. The shared writer remains
the v1 substrate; ADR-0025 §DD-S5.8 noted that writer-level
isolation is deferred. B0-S7 surfaces the shared resource as a
monitored signal that each runner responds to according to its
own work shape — the cross-mode coupling is still real, but it
is now observable and bounded.

### Composition with B1-2 (set-mode cost ceilings)

When B1-2 resolves (currently open at B1), set-mode cost
ceilings will live on the same `EnvConfig` struct as a sibling
`SetModeCost` (or similar) sub-struct. The two sub-structs
enforce independently:

- A record-mode rule's evaluation respects only `RecordModeCost`
  ceilings.
- A set-mode rule's evaluation respects only `SetModeCost`
  ceilings (when B1-2 resolves).
- An entity with rules in both modes consumes from both budget
  pools independently — there is no platform-wide "this entity
  costs X total per day" budget in v1. Each mode's runner
  enforces its own ceilings.

A future unified-per-entity-budget mechanism (operator-declared
per-entity cost cap that spans modes) is a follow-up if
operational signal motivates it; v1 commits independent
budgets to keep the cost model simple.

### Lint cross-checks added

This study adds **no new lint cross-checks**. Cost ceilings are
engine-side env config, not rule-authoring concerns. The
loader-side rule-override-vs-env-ceiling check is a runtime
enforcement (loader rejects at boot), not a lint check. Ten
lint cross-checks total remain from ADRs 0021–0024 (with
ADR-0026 adding none).

### One-line decision summary table

| Decision | Outcome |
|---|---|
| Cost dimensions bounded | Consumer lag, late-drop rate, dead-letter rate, evidence sample size, writer-queue saturation, consumer throughput |
| Ceiling location | `EnvConfig.RecordModeCost` typed sub-struct (per ADR-0018); per-env values in `engine/internal/env/{local,qa,prod}.go`; Kustomize overlay extensions (per ADR-0019) |
| Per-rule overrides | Only `evidence_sample_size` (per ADR-0026 pattern); bounded by env hard ceiling; loader rejects overruns at boot |
| Enforcement model | Engine-side (runner-loop monitors thresholds; loader rejects rule-override overruns); broker-side deferred |
| Composition with B1-2 | Independent budgets per mode; no unified-per-entity total in v1 |
| Dead-letter handling | Drop and count (matches late-drop pattern); `max_dead_letter_rate` alert per env; dead-letter Kafka topic deferred |
| Writer-coupling treatment | Surfaced as observable per-runner backpressure tied to `WriterQueueSaturationThreshold`; writer-level isolation still deferred |
| Operational alerts | Per ADR-0006 routing; `operational` channel category; per-attempt deduper |
| Lint cross-checks added | None |

---

## Consequences

### Cross-cutting consequences

- **C-B0S7.1** — **`EnvConfig` typed struct gains a
  `RecordModeCost` sub-struct.** Six fields per the recommended
  shape, with per-env values in `engine/internal/env/{local,qa,prod}.go`.
  This is an **extension to ADR-0018's typed env-config model**
  (PAT-4); ADR-0018's contract is not reopened — the extension
  follows the same pattern as the existing log-level / HTTP-
  address / emulator-override fields. The combined
  implementation commit lands the struct extension and the
  per-env values together. *(New contribution proposed here,
  requires review.)*

- **C-B0S7.2** — **Kustomize overlays under
  `deploy/overlays/{qa,prod}/` gain `RecordModeCost` overrides.**
  Per ADR-0019's overlay pattern, qa and prod overlays patch
  the `RecordModeCost` values onto the deployed `EnvConfig`.
  Local uses the struct's default values. The combined
  implementation commit lands the overlay extensions. *(New
  contribution proposed here, requires review.)*

- **C-B0S7.3** — **The loader rejects rule overrides that
  exceed env ceilings at boot, per-rule (not engine-wide).**
  Per B0-S6's per-rule `evidence_sample_size` override (in
  `params.aggregation`), the loader compares each override
  against `EnvConfig.RecordModeCost.MaxEvidenceSampleSize`;
  rules with overrides exceeding the env ceiling are
  **rejected per-rule** — the engine logs the rejection, emits
  an operational alert per ADR-0006, and **continues to start
  with the remaining rules**. One bad rule does not take down
  the engine for unrelated set-mode or other record-mode work.
  This matches ADR-0021's per-rule mode-record rejection
  pattern (not ADR-0022's catalog-handler-startup-invariant
  fail-fast pattern, which applies to catalog vs handler
  alignment). This is the **only loader-side cost check** in
  v1; other ceilings are runtime-monitored, not
  loader-enforced.

- **C-B0S7.4** — **`WriterQueueSaturationThreshold` converts the
  silent cross-mode writer coupling into observable per-runner
  back-off responses.** ADR-0025 §DD-S5.8 / §OQ-B0S5.3
  acknowledged the writer as a shared SPOF. B0-S7 commits a
  per-env threshold; when crossed, each runner applies a
  mode-appropriate back-off — the **record runner** slows its
  Kafka consumer-group read rate; the **set runner**'s HTTP
  trigger handler (per ADR-0014) returns HTTP 503 with
  `Retry-After` on incoming triggers. The cross-mode coupling
  remains, but it now has a known, monitored signal that each
  runner responds to according to its own work shape.
  Writer-level isolation (separate write queues per mode)
  remains a future enrichment.

- **C-B0S7.5** — **Late-drop ceiling triggers operational
  alert.** Per ADR-0026 §DD-S6.9, late-drop signalling was
  deferred to B0-S7. B0-S7 commits a per-env `MaxLateDropRate`;
  exceeding it over a sliding observation window triggers an
  operational alert per ADR-0006. The per-window late-drop
  count remains in evidence (per ADR-0026); the alert is the
  cumulative signal.

- **C-B0S7.6** — **Dead-letter routing is "drop and count" in
  v1.** Records the handler cannot process are dropped, counted
  as `dead_letter_count` in evidence (a new evidence field
  added alongside `late_dropped_count` from ADR-0024 and the
  ADR-0026 evidence shape), and surfaced via the
  `MaxDeadLetterRate` operational alert. Dead-letter Kafka
  topic routing is deferred to a future ADR if operational
  signal motivates it. This **extends ADR-0026's evidence shape**
  with a new field — same pacing-argument pattern (ADR-0026 has
  not shipped to disk yet; combined implementation commit
  lands the extension). *(New contribution proposed here,
  requires review.)*

- **C-B0S7.7** — **Composition with B1-2 is independent
  budgets, not unified.** An entity with rules in both modes
  respects both budgets independently; no platform-wide
  per-entity total-cost cap in v1. **The structural
  realisation is B1-2's call** — B1-2 may surface its
  set-mode ceilings as a sibling sub-struct on `EnvConfig`,
  as a separate file, or via another mechanism it picks. B0-S7
  commits only the **principle** of independent per-mode
  budgets, not the structural realisation. *(New contribution
  proposed here, requires review.)*

- **C-B0S7.8** — **Engine-side enforcement only in v1.**
  Broker-side enforcement (Kafka quotas) is reserved as a
  future enrichment when operational signal motivates it. The
  engine controls all v1 enforcement: runner-loop monitors
  thresholds; loader rejects rule-override overruns; alerter
  emits operational alerts on threshold crossings.

- **C-B0S7.9** — **Cost enforcement throttles, never alters
  results.** Guardrails slow ingestion (backpressure) or
  reject configurations (loader-side rejection); they never
  change a check's per-record evaluation outcome. Determinism
  per ADR-0002 / P2 mirror is preserved: same input stream +
  same configuration → same `execution_id` set + same per-check
  statuses, regardless of throttle behaviour.

- **C-B0S7.10** — **No new lint cross-checks.** Cost ceilings
  are engine-side env config, not rule-authoring concerns. Ten
  lint cross-checks total remain from ADRs 0021–0024.

- **C-B0S7.11** — **The full Wave-S gate closes at this ADR's
  promotion.** Per ADR-0020 §Decision (Full-Wave-S gate
  criterion), the gate is met when all seven B0-S items are at
  `resolved-adr` and their ADRs are merged into `docs/adr/`.
  B0-S1 through B0-S6 are already at `resolved-adr`; this
  study's promotion to ADR-0027 makes B0-S7 the seventh.
  After ADR-0027 merges, Wave-S has no remaining
  decision-side work — only the combined implementation
  commit remains.

### Per-artefact consequences

- **`engine/internal/env/config.go`** — gains a `RecordModeCost`
  typed sub-struct with the six fields per the recommended
  shape, embedded in the existing `EnvConfig` struct.

- **`engine/internal/env/local.go`** — gains `RecordModeCost`
  values per the local row of the recommended table (loose
  defaults for development).

- **`engine/internal/env/qa.go`** — gains `RecordModeCost`
  values per the qa row (moderate ceilings).

- **`engine/internal/env/prod.go`** — gains `RecordModeCost`
  values per the prod row (strict ceilings).

- **`deploy/overlays/qa/`** — Kustomize patches for qa overrides
  (matches qa.go values; surfaces them to the deployed
  ConfigMap or environment per ADR-0019's overlay model).

- **`deploy/overlays/prod/`** — Kustomize patches for prod
  overrides (matches prod.go values).

- **`deploy/overlays/local/`** — no patch needed (local uses
  the struct defaults).

- **`engine/internal/runner/RecordRunner`** (per ADR-0025) —
  gains a cost-monitoring layer that observes consumer lag,
  late-drop rate, dead-letter rate, writer-queue saturation
  per the env config; emits operational alerts on threshold
  crossings; applies per-runner backoff on writer saturation.

- **`engine/internal/runner/SetRunner`** (per ADR-0025) —
  gains the writer-queue-saturation backoff response (set
  runner also slows when the shared writer saturates). No
  other cost monitoring on the set side (B1-2 commits set-mode
  ceilings when resolved).

- **`engine/internal/eval/record_schema_conformance.go`** (per
  ADR-0022 / ADR-0026) — handler gains a `dead_letter_count`
  return field alongside the existing per-record outcomes;
  records the handler cannot process bump the count.

- **`engine/internal/eval/evaluator.go`** (or wherever loader
  validation lives) — gains a startup check that compares each
  loaded rule's `params.aggregation.evidence_sample_size`
  override against `EnvConfig.RecordModeCost.MaxEvidenceSampleSize`;
  rejects rules whose override exceeds the ceiling at boot.

- **`docs/adr/0018-environment-configuration-model.md`** —
  scope-noted as mode-neutral (per the 2026-05-23 scope-note
  pass which left ADR-0018 untouched). This ADR (ADR-0027)
  extends the env config additively under ADR-0018's pattern;
  ADR-0018 is not reopened.

- **`docs/adr/0019-infrastructure-tooling.md`** — mode-neutral.
  This ADR extends the overlay pattern additively per ADR-0019's
  Kustomize commitment; ADR-0019 is not reopened.

- **`docs/adr/0006-alert-routing-contract.md`** — set-mode
  scope-noted. The operational-alert-on-ceiling-crossing fits
  ADR-0006's existing `operational` category routing; the
  alert payload uses the existing event schema; ADR-0006 is
  not reopened. No new alert category, no new routing path —
  just new events using existing infrastructure.

- **B1-2** (open at B1) — when resolved, its set-mode ceilings
  will live as a sibling `SetModeCost` sub-struct on the same
  `EnvConfig` struct B0-S7 extends. B0-S7 pre-scopes the
  composition (independent budgets per mode) without committing
  B1-2's eventual values.

- **No new lint cross-checks.** The lint binary remains at ten
  cross-checks.

---

## Open Questions

- **OQ-B0S7.1** — **Sliding-window observation period for rate
  alerts.** `MaxLateDropRate` and `MaxDeadLetterRate` are
  rate-based; they need an observation period (e.g., "N
  consecutive windows above threshold triggers alert"). *Out
  of scope for current cycle;* the combined implementation
  commit picks the period based on the alerting cadence
  ADR-0006 expects.

- **OQ-B0S7.2** — **Per-runner backoff factor curve.** When
  `writer_queue_saturation` ≥ threshold, the runners back off
  per their mode-appropriate mechanism (record-side consumer
  rate; set-side HTTP 503 with `Retry-After`). The factor's
  shape (linear, exponential, curve-fitted to writer drain
  rate) and its decay constants are implementation-tuning
  questions. *Out of scope for current cycle.*

- **OQ-B0S7.3** — **Broker-side enforcement (Kafka quotas).**
  Reserved for the future enrichment path. Whether to pursue
  broker-side quotas depends on operational coordination with
  broker operators and on whether engine-side enforcement
  proves insufficient. *Out of scope for current cycle;* v1
  commits engine-side only.

- **OQ-B0S7.4** — **Dead-letter Kafka topic routing.** Reserved
  for the future enrichment path. v1 drops and counts; if
  operational signal motivates routing dead-letter records to a
  topic for offline analysis, a future ADR commits the topic
  shape, consumer-group, retention policy. *Out of scope for
  current cycle.*

- **OQ-B0S7.5** — **Unified per-entity budget that spans
  modes.** Reserved for the future enrichment path. v1 commits
  independent per-mode budgets; a future unified-entity-budget
  mechanism (operator-declared per-entity cost cap that spans
  modes) is a follow-up if operational signal motivates it.
  *Out of scope for current cycle.*

- **OQ-B0S7.6** — **Cost-dimension extensibility under
  ADR-0018's typed env config.** Adding a new cost dimension
  (e.g., a future `MaxPartitionCount` for record-mode topics
  with extreme partition counts) is additive on the
  `RecordModeCost` struct under ADR-0018's evolution pattern.
  Whether to surface a sub-versioning scheme for the
  `RecordModeCost` struct (similar to ADR-0001's schema-version
  governance) is a future enrichment. *Out of scope for current
  cycle;* additive growth under ADR-0018's contract suffices
  until cross-version compatibility becomes a concern.

- **OQ-B0S7.7** — **Composition mechanics when B1-2 resolves.**
  This study commits the **independence principle** (each mode's
  budgets enforce independently). The mechanical implementation
  (do both sub-structs live side-by-side on `EnvConfig`? does
  a future `EntityCostComposer` aggregate across modes?) is a
  B1-2 / implementation-commit question. *Defer to B1-2;* this
  study commits the principle, not the mechanics.

- **OQ-B0S7.8** — **Lint pre-flight against prod ceilings.**
  C-B0S7.10 commits no new lint cross-checks because env-bound
  checks cannot know env values at PR-review time. A future
  enrichment could have lint pre-flight per-rule overrides
  against the **strictest env's ceilings** (prod values),
  giving authors PR-time feedback that a rule will be rejected
  in prod before it ships through qa to prod. The v1
  enforcement remains loader-side per C-B0S7.3 (per-rule
  rejection at engine boot with operational alert); the
  pre-flight is purely additive operability. *Out of scope
  for current cycle;* surfaces as a developer-experience
  enrichment when concrete operational signal motivates it.

---

## Promotion target

**Target:** `docs/adr/0027-record-mode-cost-guardrails.md`.

This study promotes to **ADR-0027** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0027 is the **fourth and final
Phase β ADR** of Wave-S. Per ADR-0020 §Decision (Per-item ADR
numbering), the `0027` slot is descriptive; if the forthcoming
ADR-0010 amendment (Kafka substrate-matrix row, flagged by
ADR-0023 §C-B0S3.3) lands first at the `0027` slot, this study
promotes to `0028` and the per-item numbering shifts.

**ADR-0027's promotion is the full-Wave-S-gate-closing event.**
Per ADR-0020 §Decision (Full-Wave-S gate criterion):

> Met when **all seven B0-S items** are at status `resolved-adr`
> and their ADRs (provisional 0021 … 0027) are merged. At
> full-gate, the platform has a complete record-oriented
> capability, parallel in completeness to the set-oriented
> capability that Wave 3 closed.

After ADR-0027 merges, Wave-S has no remaining decision-side
commitments. All seven B0-S items are realised in concrete
schema, catalog, source, window, runner-shape, aggregation-
function, and cost-guardrail decisions. The remaining surface
is the **combined implementation commit** that lands ADRs
0021–0027 runtime artefacts together (per the pacing pattern
first set down by ADR-0022 §C-B0S2.2 and used in every Wave-S
ADR since).

The promotion commit lands the artefacts committed in
§Consequences above:

1. The `RecordModeCost` typed sub-struct on `EnvConfig` with
   per-env values in `engine/internal/env/{local,qa,prod}.go`.
2. Kustomize overlay extensions in
   `deploy/overlays/{qa,prod}/`.
3. The record runner's cost-monitoring layer + the set
   runner's writer-saturation backoff response.
4. The handler's `dead_letter_count` return field + evidence
   extension.
5. The loader's rule-override-vs-env-ceiling rejection check.
6. Operational alert wiring per ADR-0006 for ceiling crossings.

Per R8, the future ADR-0027 will be rewritten from this study,
not linked back to it.
