<!-- path: docs/adr/0027-record-mode-cost-guardrails.md -->

# ADR-0027 — Record-Oriented Cost Guardrails (Full Wave-S Gate Closure)

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

ADR-0020 launched Wave-S with four locked premises and split the
decision work into a foundational triplet (ADR-0021 mode primitive,
ADR-0022 kind catalog, ADR-0023 sources schema) and a Phase β
(ADR-0024 windowing, ADR-0025 runner shape, ADR-0026 failure
scope, and this ADR). All prior Wave-S ADRs have promoted; **this
ADR is the gate-closing event** per ADR-0020 §Decision (Full-Wave-S
gate criterion): once ADR-0027 merges, all seven B0-S items are at
`resolved-adr` and Wave-S has no remaining decision-side
commitments.

Three Phase β commitments flow into this ADR:

- **ADR-0025 §C-B0S5.6 — per-runner shape for cost guardrails.**
  Record-mode lag is consumer-group lag tracked by the record
  runner; set-mode cost is BigQuery slot consumption tracked by
  the set runner. The per-runner isolation lets budgets enforce
  and signal per mode independently.
- **ADR-0025 §DD-S5.8 / §OQ-B0S5.3 — writer-coupling.** The
  shared writer is a cross-mode SPOF; writer-level isolation is
  deferred. This ADR either surfaces the coupling as a signal or
  commits an isolation path.
- **ADR-0026 §C-B0S6.6 / §DD-S6.9 — late-drop as cost signal.**
  Late-dropped records are excluded from aggregation; lateness-
  rate alerting and cost guardrails live here, not in ADR-0026's
  per-check status determination.
- **ADR-0026 §C-B0S6.9 — `evidence_sample_size` as cost
  dimension.** Per-kind defaults from ADR-0026 bind to per-runner
  storage budgets that this ADR commits.

The current engine env config (per ADR-0018) is the typed
`EnvConfig` struct in `engine/internal/env/config.go`, with
per-env constants in `local.go` / `qa.go` / `prod.go`. ADR-0023
§C-B0S3.5 removed `SourceProject` / `SourceDataset` from those
files at the combined implementation commit; the env config now
carries deployment-level identifiers (log level, env name, HTTP
address, emulator overrides). This ADR **adds** record-mode cost-
ceiling fields to that typed struct, with per-env values in the
respective files, exposed to Kubernetes via Kustomize overlays
per ADR-0019.

Four sub-decisions are resolved by this ADR:

1. **The cost dimensions to bound** — which knobs matter for
   record-mode operational cost and signal.
2. **Per-env ceiling location** — env-config struct (engine-side),
   Kustomize overlays (deploy-side), rule YAML (rule-side), or
   layered.
3. **Enforcement location** — engine-side, broker-side (Kafka
   quotas), or both.
4. **Composition with B1-2** — how an entity with rules in both
   modes respects both budgets.

The platform principles bearing on the design: **foundation 01
§"Cost"** (cost is a first-class constraint — partition
discipline, query templates, dry-run visibility, concurrency
budgets, evidence-retention limits are platform design, not
later hardening); **P5** (evolution contract-driven — env-config
extensions live under ADR-0018; overlay extensions under
ADR-0019); **P2 / foundation 01 §"Determinism"** (cost
enforcement must not introduce non-determinism in evaluation —
guardrails throttle ingestion, never alter the result).

---

## Decision

### Six record-mode cost dimensions

The record-mode runtime is bounded along six dimensions, all
declared as fields on a new typed `RecordModeCost` sub-struct
attached to ADR-0018's `EnvConfig`:

| Dimension | Field | Meaning |
|---|---|---|
| Consumer lag | `MaxConsumerLag` | Kafka consumer-group lag (records); above triggers operational alert |
| Late-drop rate | `MaxLateDropRate` | Per-window rate of late-dropped records (per ADR-0024); above triggers alert |
| Dead-letter rate | `MaxDeadLetterRate` | Per-window rate of records the handler could not process; above triggers alert |
| Evidence sample size | `MaxEvidenceSampleSize` | Hard upper bound on per-rule overrides of ADR-0026's per-kind catalog default |
| Writer-queue saturation | `WriterQueueSaturationThreshold` | Ratio of writer-queue occupancy; above triggers per-runner back-off |
| Consumer throughput | `MaxConsumerThroughputRecsSec` | Steady-state throttle target |

Per-env values land in `engine/internal/env/{local,qa,prod}.go`
matching the existing log-level / HTTP-address / emulator-
override pattern. Indicative defaults (final values picked at the
combined implementation commit; tunable via operational signal):

| Dimension | local | qa | prod |
|---|---|---|---|
| `MaxConsumerLag` | 1,000,000 | 100,000 | 10,000 |
| `MaxLateDropRate` | 0.50 | 0.10 | 0.01 |
| `MaxDeadLetterRate` | 0.50 | 0.05 | 0.01 |
| `MaxEvidenceSampleSize` | 100 | 50 | 20 |
| `WriterQueueSaturationThreshold` | 0.90 | 0.75 | 0.50 |
| `MaxConsumerThroughputRecsSec` | unlimited | 50,000 | 10,000 |

```go
// EnvConfig extension (illustrative shape)
type RecordModeCost struct {
    MaxConsumerLag                 int
    MaxLateDropRate                float64
    MaxDeadLetterRate              float64
    MaxEvidenceSampleSize          int
    WriterQueueSaturationThreshold float64
    MaxConsumerThroughputRecsSec   int
}

type EnvConfig struct {
    // ... existing fields ...
    RecordModeCost RecordModeCost
}
```

Per-env values surface to Kubernetes via Kustomize overlay
extensions under `deploy/overlays/{qa,prod}/` per ADR-0019; the
local overlay uses the struct's default values.

### Per-rule overrides bounded by env hard ceilings

The only currently-overridable cost dimension is
`evidence_sample_size` — per ADR-0026's per-kind catalog default
+ per-rule override via `params.aggregation`. The env's
`MaxEvidenceSampleSize` is the **hard upper bound** on per-rule
overrides; the loader enforces this at boot (see below).

### Per-runner backpressure split (mode-appropriate)

When `writer_queue_saturation` ≥ `WriterQueueSaturationThreshold`,
each runner responds with a back-off mechanism appropriate to
its work shape — the two mechanisms are different because the
runners' work shapes are different:

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

Both back-offs decay as the writer-queue saturation drops below
the threshold. An operational alert fires once per saturation
event, deduped per ADR-0006.

This is **per-runner observable response to a shared signal**,
not writer-side isolation. The shared writer remains the v1
substrate; ADR-0025 §DD-S5.8 noted that writer-level isolation
is deferred. This ADR surfaces the shared resource as a
monitored signal that each runner responds to according to its
own work shape.

### Per-rule loader rejection on evidence-sample overruns

Per ADR-0026's per-rule `evidence_sample_size` override (in
`params.aggregation`), the loader compares each override against
`EnvConfig.RecordModeCost.MaxEvidenceSampleSize`. Rules with
overrides exceeding the env ceiling are **rejected per-rule** —
the engine logs the rejection, emits an operational alert per
ADR-0006, and **continues to start with the remaining rules**.
One bad rule does not take down the engine for unrelated set-mode
or other record-mode work. This matches ADR-0021's per-rule
mode-record rejection pattern (not ADR-0022's catalog-handler-
startup-invariant fail-fast pattern, which applies to catalog vs
handler alignment).

This is the **only loader-side cost check** at this ADR's
promotion; other ceilings are runtime-monitored, not
loader-enforced.

### Late-drop and dead-letter as operational alerts

`MaxLateDropRate` per env — exceeding it over a sliding
observation window triggers an operational alert per ADR-0006.
The per-window late-drop count remains in evidence (per
ADR-0026); the alert is the cumulative signal that lateness is
degrading data quality. Per ADR-0026 §DD-S6.9 / §C-B0S6.6, the
late-drop signal does not affect per-check status determination
— B0-S6 owns status; this ADR owns lateness alerting.

`MaxDeadLetterRate` per env — records the handler cannot
process (malformed record, schema parse failure before
validation, unbounded recursion in the validator) are dropped
and counted as `dead_letter_count` in evidence (a new evidence
field added alongside `late_dropped_count` from ADR-0024 per
the ADR-0026 evidence shape). Exceeding the rate triggers an
operational alert per ADR-0006.

Dead-letter Kafka topic routing — routing dropped records to a
dedicated topic for offline analysis — is a future enrichment
when operational signal motivates it. v1 keeps the substrate
dependency minimal: drop and count, matching the late-drop
pattern from ADR-0024.

### Composition with B1-2 — independence principle

An entity with rules in both modes respects both budgets
independently; there is no platform-wide per-entity total-cost
cap in v1. Each mode's runner enforces its own ceilings:

- A record-mode rule's evaluation respects only `RecordModeCost`.
- A set-mode rule's evaluation respects only the set-mode
  ceilings B1-2 commits when resolved.

The **structural realisation** of the set-mode ceilings is
B1-2's call — B1-2 may surface its ceilings as a sibling
sub-struct on `EnvConfig`, as a separate file, or via another
mechanism it picks. This ADR commits only the **principle** of
independent per-mode budgets, not the structural realisation.

A future unified-per-entity budget that spans modes is a
follow-up enrichment if operational signal motivates it; v1
commits independence to keep the cost model simple.

### Engine-side enforcement only

v1 commits engine-side enforcement only:

- Runner-loop monitors thresholds and emits alerts on crossings.
- Loader rejects rule-override overruns at boot.
- Alerter routes operational alerts per ADR-0006.

Broker-side enforcement (Kafka consumer-group quotas, broker-
level rate limits) is reserved as a future enrichment when
operational coordination with broker operators is in place and
engine-side enforcement proves insufficient.

### Cost enforcement throttles, never alters results

Guardrails slow ingestion (back-off responses) or reject
configurations (loader-side per-rule rejection); they never
change a check's per-record evaluation outcome. Determinism per
ADR-0002 / P2-mirror is preserved: the same input stream + same
configuration → same `execution_id` set + same per-check
statuses, regardless of whether the engine throttled.

### No new lint cross-checks

Cost ceilings are engine-side env config, not rule-authoring
concerns. The loader-side rule-override-vs-env-ceiling check is
a runtime enforcement (per-rule rejection at engine boot), not
a lint check. The ten lint cross-checks from ADRs 0021–0024
remain the lint surface; ADR-0026 added none; this ADR adds none.

A future enrichment could have lint pre-flight per-rule
overrides against the strictest env's ceilings (prod values),
giving authors PR-time feedback that a rule will be rejected in
prod before it ships. v1 enforcement is loader-side per the
preceding subsection; the pre-flight is purely additive
operability and is deferred.

---

## Consequences

1. **The full Wave-S gate closes at this ADR's promotion.** Per
   ADR-0020 §Decision (Full-Wave-S gate criterion), the gate is
   met when all seven B0-S items are at `resolved-adr` and their
   ADRs are merged into `docs/adr/`. ADR-0021 through ADR-0026
   are merged; this ADR's promotion is the seventh and final
   event. After this ADR merges, Wave-S has no remaining
   decision-side commitments — all four locked premises
   (P1–P4) plus the foundational triplet plus all four Phase β
   items are realised in concrete commitments.

2. **`EnvConfig` typed struct gains a `RecordModeCost`
   sub-struct.** Six fields per the table above, with per-env
   values in `engine/internal/env/{local,qa,prod}.go`. This is
   an **extension to ADR-0018's typed env-config model**
   (PAT-4); ADR-0018's contract is not reopened — the extension
   follows the same pattern as the existing log-level / HTTP-
   address / emulator-override fields. The combined
   implementation commit lands the struct extension and the
   per-env values together.

3. **Kustomize overlays under `deploy/overlays/{qa,prod}/` gain
   `RecordModeCost` overrides.** Per ADR-0019's overlay pattern.
   The local overlay uses struct defaults; qa and prod patch
   the values onto the deployed `EnvConfig`. The combined
   implementation commit lands the overlay extensions.

4. **`WriterQueueSaturationThreshold` converts the silent cross-
   mode writer coupling into observable per-runner back-off
   responses.** ADR-0025 §DD-S5.8 acknowledged the writer as a
   shared SPOF. This ADR commits a per-env threshold; when
   crossed, each runner applies a mode-appropriate back-off —
   the record runner slows its Kafka consumer-group read rate;
   the set runner's HTTP trigger handler returns HTTP 503 with
   `Retry-After`. The cross-mode coupling remains, but it now
   has a known, monitored signal that each runner responds to
   according to its own work shape. Writer-level isolation
   (separate write queues per mode) remains a future enrichment.

5. **Late-drop and dead-letter ceilings trigger operational
   alerts.** `MaxLateDropRate` and `MaxDeadLetterRate` per env
   surface lateness and dead-letter degradation as operational
   alerts per ADR-0006. The per-window counts remain in
   evidence; the alerts are the cumulative signals.

6. **Dead-letter routing is "drop and count" in v1.** Records
   the handler cannot process are dropped, counted as
   `dead_letter_count` in evidence (a new field alongside
   ADR-0024's `late_dropped_count`), and surfaced via the
   `MaxDeadLetterRate` operational alert. Dead-letter Kafka
   topic routing is deferred to a future ADR if operational
   signal motivates it. This **extends ADR-0026's evidence
   shape** with a new field — same pacing-argument pattern
   (ADR-0026 has not shipped to disk yet; the combined
   implementation commit lands the extension).

7. **Composition with B1-2 is independent budgets, not unified.**
   Each mode's budgets enforce independently; no platform-wide
   per-entity total-cost cap in v1. When B1-2 resolves, the
   structural realisation of set-mode ceilings is B1-2's call;
   this ADR commits only the principle of independent per-mode
   budgets.

8. **The loader rejects rule overrides per-rule, not engine-
   wide.** Rules with `evidence_sample_size` overrides
   exceeding env ceilings are rejected at boot with an
   operational alert; the engine continues to start with the
   remaining rules. One bad rule does not take down the engine
   for unrelated work. Matches ADR-0021's per-rule mode-record
   rejection pattern.

9. **Engine-side enforcement only in v1.** Broker-side
   enforcement (Kafka quotas) is reserved as a future
   enrichment.

10. **Cost enforcement throttles, never alters results.**
    Guardrails slow ingestion or reject configurations; they
    never change a check's per-record evaluation outcome.
    Determinism per ADR-0002 / P2-mirror is preserved.

11. **No new lint cross-checks.** Ten lint cross-checks total
    remain from ADRs 0021–0024 (with ADR-0026 and this ADR each
    adding none).

12. **Combined-commit pacing extends across ADRs 0021–0027.**
    The runtime artefacts committed by ADRs 0021–0027 — schemas
    v2 with mode/params/source/window, catalog v1 with the
    aggregation block, ten lint cross-checks, dispatcher
    startup invariant, two parallel runners, the `mode` column
    on `dq_executions`, env-constants removal, Kafka emulator
    + ADR-0010 amendment, loader rejection-path removal,
    aggregation function + evidence-shape extensions, the
    `RecordModeCost` struct + Kustomize overlay extensions, the
    cost-monitoring layer on the record runner, the HTTP 503
    back-off on the set runner, the loader's per-rule
    cost-override rejection check — land together in a single
    combined implementation commit. If any artefact ships
    incrementally first, the schema-bump and catalog-bump
    contingencies from prior ADRs apply.

---

## Notes

- The illustrative per-env ceiling values in the table above
  are the **starting point**, not the final values. The
  combined implementation commit picks values informed by
  operational signal (smoke tests, qa load patterns, prod
  expectations). Future tuning re-fits the values without
  reopening this ADR; the **shape** (which six dimensions, which
  layered location, which enforcement mode) is what this ADR
  commits.
- The sliding-window observation period for rate alerts
  (`MaxLateDropRate`, `MaxDeadLetterRate`) is picked at the
  combined implementation commit based on the alerting cadence
  ADR-0006 expects. The choice does not affect this ADR's
  commitments.
- The per-runner backoff factor's shape (linear, exponential,
  curve-fitted to writer drain rate) and its decay constants
  are implementation-tuning questions picked at the combined
  implementation commit.
- A future unified-per-entity budget that spans modes (operator-
  declared per-entity cost cap aggregating across set and
  record modes) is a follow-up enrichment when operational
  signal motivates it. v1 commits independent per-mode budgets.
- A future enrichment could pre-flight per-rule
  `evidence_sample_size` overrides against the strictest env's
  ceiling (prod) at lint time, giving authors PR-time feedback
  before the rule reaches the loader. v1 enforcement remains
  loader-side; the pre-flight is purely additive operability.
- Broker-side enforcement (Kafka consumer-group quotas) is
  reserved for a future ADR when operational coordination with
  broker operators is in place and engine-side enforcement
  proves insufficient.
- Dead-letter Kafka topic routing — routing dropped records to
  a dedicated topic for offline analysis — is reserved for a
  future ADR when operational signal motivates it.
- Cost-dimension extensibility under ADR-0018's typed env
  config follows ADR-0001's additive-within-major contract. A
  future cost dimension (e.g., `MaxPartitionCount` for
  record-mode topics with extreme partition counts) extends the
  `RecordModeCost` struct additively without reopening this
  ADR.

---

## The state of Wave-S after this ADR

With ADR-0027 merged:

- **Foundational triplet:** ADR-0021 (mode primitive),
  ADR-0022 (kind catalog), ADR-0023 (sources schema). Realises
  P1, P2, P3.
- **Phase β:** ADR-0024 (windowing), ADR-0025 (runner shape;
  P4 retired), ADR-0026 (failure scope aggregated), ADR-0027
  (cost guardrails — this ADR).
- **Wave-S launch:** ADR-0020 (the integration point that the
  2026-05-23 scope-note pass on ADRs 0002, 0003, 0004, 0006,
  0007, 0010, 0014, 0017 pointed forward to).

The **decision-side surface of Wave-S is closed**. The
**combined implementation commit** lands the runtime artefacts
of all seven per-item ADRs plus this one together — the
implementation arrives as a single coherent set, matching the
pacing argument first set down by ADR-0022 §C-B0S2.2 and
followed by every Wave-S ADR since. After the combined
implementation commit ships, record-mode capability is
operationally complete: a record-mode rule may declare its
Kafka source and tumbling window, the catalog dispatches to
its handler, the handler aggregates per-record outcomes into
ADR-0004 statuses, the writer carries the rows under the
`mode` column, the alerter emits operational signals on
ceiling crossings, and the operator monitors cost dimensions
via the per-env `RecordModeCost`. The platform's record-
oriented capability is parallel in completeness to the set-
oriented capability that Wave 3 closed.
