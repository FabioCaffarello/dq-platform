<!-- path: docs/adr/0056-panel-5-lighting-slice.md -->

# ADR-0056 — Panel 5 Lighting Slice

- **Status:** accepted
- **Date:** 2026-05-30

---

## Context

[ADR-0039](./0039-dashboard-contract.md) §"Metric contract"
committed an eight-metric inventory.
[ADR-0055](./0055-metric-emission-slice-scope.md) shipped six of
the eight; the remaining two — `dq_queue_depth` and
`dq_scheduler_triggers_managed` — were committed at the contract
layer but the engine binary did not emit them. As a consequence,
panel #5 ("Scheduler health") of
`deploy/dashboards/baseline.json` rendered `"no data"` against
both gauges.

B3-5 triaged three paths for lighting panel #5: (A) engine-side
emission with the §(a) classification depending on the operator's
reading of ADR-0039 Meaning column wording; (B) a
scheduler-binary instrumentation slice (Rejected per
[ADR-0049](./0049-b3-evolutionary-launch.md) §(a) — fails
Condition 2 AND outside any active wave's gate); (C) permanent
deferral (non-action).

Two coupled D0s on ADR-0049 §(a) Conditions 1 (P-B3.1
expands-not-rewrites) and 3 (P-B3.2 envelope conformance) were
**operator-ratified mid-PR-#113** per `CONTRIBUTING.md` Flow 5
§"Operator-side responsibilities" by adopting the **weak
reading** of ADR-0039 §"Metric contract" Meaning column wording
*"Count of runs the scheduler currently tracks, split by state"*
AND the **A.y sub-path** (constant zero for engine-non-derivable
series, not drop). Under that reading, the committed contract is
the gauge semantics (queue depth, label `state`); the source
identification in the Meaning column described the
ADR-0039-time-known emitter without committing the source as a
label-source rule.

Per R5 + A7 of the `adr-writing` skill, the ratified reading is
recorded here as **new contribution requiring review**: the weak
interpretation of ADR-0039's Meaning column wording, plus the
additive `source` label as the structural mechanism that
reconciles engine-as-source with the existing wording without
amending the contract. Both are reviewed in this ADR.

The principles bearing on the decision are **P3** (ownership is
explicit — the additive `source` label self-identifies the gauge
emission so operators reading the panel can tell engine-derived
from scheduler-derived values), **P5** (evolution is
contract-driven — the additive-label mechanism per ADR-0039
§"Evolution rules" satisfies P5 verbatim), and **R3** (do not
revisit settled architecture — ADR-0033's external-scheduler
posture is preserved verbatim; ADR-0039's Meaning column is
neither rewritten nor amended).

Per the operator-authorized **R4 scope collapse** at promotion
time (precedent [ADR-0054](./0054-engine-image-registry-amendment.md)
§Notes, [ADR-0055](./0055-metric-emission-slice-scope.md)
§Notes), this ADR ships in a single PR with its implementation
slice.

---

## Decision

The slice is committed in five clauses (source, label, zero-fill
for engine-non-derivable, struct convention, cadence), plus a
Notes block recording the R4 collapse and the ratified D0
carry-forward per R5 + A7.

### Clause 1 — Source for `dq_queue_depth{state="running"}`

The engine emits `dq_queue_depth{state="running",source="engine"}`
as the count returned by a new reader method on
`engine/internal/results.Reader`:

```
CountRunningExecutions(ctx context.Context) (int64, error)
```

The method's semantics match `ListRunningOlderThan`'s
canonical-view projection: it counts executions whose latest
state per `dq_executions_current` is `running` — i.e., executions
that have a `running` row written but no terminal follow-up row
(success, failed, error, aborted). The query is partition-pruned
per [ADR-0031](./0031-evidence-retention-parameters.md); scan
cost is bounded by `ResultsRetention`.

`Reader.CountRunningExecutions` is committed on the `Reader`
interface alongside `QueryCurrentExecution`,
`ListRunningOlderThan`, and `LatestExecutionPerEntityCheck` —
matching the existing reader-surface shape. The
BigQueryStore implementation runs `SELECT COUNT(*) FROM (canonical
projection) WHERE rn = 1 AND status = 'running'` — same
canonical-view ROW_NUMBER() projection as `ListRunningOlderThan`,
COUNT(*) instead of row read.

### Clause 2 — Additive `source` label

Both `dq_queue_depth` and `dq_scheduler_triggers_managed` gain
an additive `source` label per ADR-0039 §"Evolution rules"
clause 1 (a new label is additive within an engine-major-version;
any consumer reading the existing surface continues to read it
because PromQL ignores unmentioned labels). The engine emits all
four series with `source="engine"`.

The label self-identifies the gauge emission's origin. Under the
ratified weak reading of ADR-0039, this is the load-bearing
mechanism: by tagging the emission as engine-derived, the engine
no longer claims scheduler-internal knowledge the Meaning column
appears to assert. Operators reading panel #5 can distinguish
engine-observed values from any future scheduler-observed
values (when / if scheduler-binary work surfaces).

The other six metrics from ADR-0055 do not gain a `source` label;
per-metric label sets are independent in Prometheus, and the six
engine-side runtime metrics have only one possible source by
construction (they emit from runner call sites the engine binary
unambiguously owns).

### Clause 3 — Constant zero for engine-non-derivable series

Three of the four `state`-labeled series the engine cannot derive
from its own observations:

| Series | Engine derivability |
|---|---|
| `dq_queue_depth{state="running",source="engine"}` | Derivable (Clause 1) |
| `dq_queue_depth{state="scheduled",source="engine"}` | Not derivable — "scheduled but not yet triggered" is scheduler-tracked state per ADR-0033 |
| `dq_scheduler_triggers_managed{state="healthy",source="engine"}` | Not derivable — trigger lifecycle is external per ADR-0033 |
| `dq_scheduler_triggers_managed{state="errored",source="engine"}` | Not derivable — same |

The three non-derivable series emit **constant zero**. This is
the A.y sub-path the operator ratified over A.x (which would
have dropped the series — amendment-shaped under either reading
per the B3-5 study's §Option A 2×2 table). Constant zero
preserves time-series continuity (the gauge series stays
defined across scrapes) and is distinguishable from positive
values via the operator's panel.

### Clause 4 — Per-package convention: new `SchedulerProxyMetrics`

A new `SchedulerProxyMetrics` struct lands in
`engine/internal/metrics` alongside `RunnerMetrics` and
`LoaderMetrics`, mirroring the per-package convention ADR-0055
§Clause 3 committed:

```
type SchedulerProxyMetrics struct {
    QueueDepth               *prometheus.GaugeVec  // labels: state, source
    SchedulerTriggersManaged *prometheus.GaugeVec  // labels: state, source
}
```

`NoopSchedulerProxyMetrics()` returns a `SchedulerProxyMetrics`
registered against a throwaway registry, safe for tests that do
not assert emission. The `Registry` struct gains a
`SchedulerProxy SchedulerProxyMetrics` field; `Registry.New()`
constructs and registers it alongside the existing runner /
loader metrics.

This struct is the carrier surface for the scheduler-proxy
emission. The engine binary's `main()` injects it where the
periodic loop in Clause 5 sets the gauges; the metric handles
themselves stay on the central inventory in
`engine/internal/metrics`.

### Clause 5 — Cadence: 15s constant in main.go

The engine binary runs a new periodic loop
`schedulerProxyMetricsLoop` alongside the existing
`loaderRefreshLoop` and `orphanScanLoop` per the patterns in
`engine/cmd/dq-engine/main.go`. The loop ticks every **15
seconds** (a constant in main.go matching the default Prometheus
scrape interval per `deploy/observability/prometheus/prometheus.yml`).

On each tick the loop:

1. Calls `Reader.CountRunningExecutions(ctx)`. Errors are
   warning-logged; the gauge values are left at their prior set
   value (i.e., the next successful tick recovers the
   correctness). Same "best-effort, retry-on-cadence" posture as
   `loaderRefreshLoop` per ADR-0007 §2.
2. Sets the four series:
   - `QueueDepth.WithLabelValues("running","engine").Set(float64(count))`
   - `QueueDepth.WithLabelValues("scheduled","engine").Set(0)`
   - `SchedulerTriggersManaged.WithLabelValues("healthy","engine").Set(0)`
   - `SchedulerTriggersManaged.WithLabelValues("errored","engine").Set(0)`

The cadence is **not** a per-env `EnvConfig` field — it would
trigger ADR-0018's "separate decision" clause for a new
EnvConfig field and inflate the slice for no current operational
benefit. If concrete operator demand to tune per-env surfaces,
a future ADR commits the EnvConfig field; until then, 15s is
the documented value.

The cadence is decoupled from Prometheus' scrape — the gauge
values are set whenever the loop ticks, and the scrape reads the
last-set value. A scrape between ticks reads stale-by-up-to-15s
values, which matches the cadence semantics of `dq_queue_depth`
(snapshot of a counter that can shift on any execution
state-transition).

---

## Consequences

1. **B3-5 closes at `resolved-adr` via this ADR + the slice
   landing in the same PR (operator-authorized R4 scope collapse,
   precedent ADR-0054 + ADR-0055 §Notes).** The decision-log B3-5
   row updates accordingly.

2. **Panel #5 lights up on next deployment.** The four series
   appear with the labels Clause 2 + Clause 3 commit. Panel #5's
   stat-panel renderings of `dq_queue_depth` and
   `dq_scheduler_triggers_managed` resolve against real data;
   the three constant-zero series render as flat zero, the
   `dq_queue_depth{state="running"}` series carries the engine's
   in-flight count.

3. **ADR-0039 is preserved verbatim.** No amendment to ADR-0039
   §"Metric contract" Meaning column ships from this ADR. The
   additive `source` label per ADR-0039 §"Evolution rules" is
   the load-bearing mechanism that reconciles engine emission
   with the existing wording without rewriting the contract.

4. **ADR-0033 is preserved verbatim.** The engine does not gain
   any new knowledge of scheduler-internal state. The constant-
   zero series for the three engine-non-derivable label combos
   make this explicit — the engine reports zero precisely
   because it does not know.

5. **`engine/internal/results.Reader` interface gains
   `CountRunningExecutions`.** All Store implementations
   (`BigQueryStore`, `results_test.go mockStore`, `runner_test.go
   inMemStore`, and the orphan-test mock if present) implement
   the new method. Interface extension is additive within the
   engine module; no external consumer is affected.

6. **New `SchedulerProxyMetrics` struct in
   `engine/internal/metrics` extends the central inventory.** The
   metrics package remains the canonical inventory matching
   ADR-0039's metric set; `RunnerMetrics` + `LoaderMetrics` +
   `SchedulerProxyMetrics` together cover six (runner) + one
   (loader) + four (scheduler-proxy) = eleven metric series,
   matching the eight ADR-0039 inventory metrics plus the three
   constant-zero engine-non-derivable label combos.

7. **A new periodic loop `schedulerProxyMetricsLoop` ticks every
   15s in the engine binary.** Cadence is a constant in
   `engine/cmd/dq-engine/main.go`, not an `EnvConfig` field;
   ADR-0018's "separate decision" clause is not triggered until
   concrete operator demand for per-env tuning surfaces.

8. **The ratified weak reading of ADR-0039's Meaning column
   wording is now load-bearing precedent.** Future B-rows
   touching `dq_queue_depth` or `dq_scheduler_triggers_managed`
   (or any other ADR-0039 metric whose Meaning column names an
   emitter) inherit the reading: the committed contract is the
   gauge semantics, not the source identity. Source distinctions
   are surfaced via additive labels per ADR-0039 §"Evolution
   rules".

9. **The additive-label mechanism is now a precedented
   extension shape.** Combined with ADR-0055's per-package
   emitter convention, the metrics package has two extension
   patterns: per-emission-site call-out (the six ADR-0055
   runtime metrics) and periodic-snapshot via additive labels
   (the four ADR-0056 scheduler-proxy series). Future B-rows
   that add or modify metric emission can pick the pattern
   that fits the metric's update cadence.

10. **Test surface lands with the slice.**
    `engine/internal/results.CountRunningExecutions` is covered
    by mockStore + unit test. `engine/internal/metrics.SchedulerProxyMetrics`
    is covered by the existing metric-registry test pattern (the
    new struct is registered, the gauges are settable, the
    `/metrics` route serves the new series).

11. **ADR-0039 OQ-1 (from B3-4 originating) is resolved.** Panel
    #5 is no longer dark by design — it is lit by the engine's
    in-flight count + constant-zero placeholders for what the
    engine cannot observe. The B3-4 OQ-1 deferral closes;
    future operator demand for actual scheduler-tracked metrics
    (the `state="scheduled"` count, the trigger lifecycle state)
    routes through Option B (scheduler-binary instrumentation),
    which remains Rejected per ADR-0049 §(a) until concrete
    wave-scale demand surfaces.

12. **ADR-0055, ADR-0049, ADR-0039, ADR-0033, ADR-0031,
    ADR-0018, ADR-0010, ADR-0007, ADR-0003 are preserved.** This
    ADR adds emission surface; no contract is reshaped; no
    envelope ADR is reopened. ADR-0033 §"Why this does NOT
    commit specific scheduler tooling" stays the operator-scope
    guidance.

---

## Notes

**Operator-authorized R4 scope collapse.** B3-5 surfaced the
triage; the implementation slice was flagged as a separate
session per R4. The operator authorized collapsing both into a
single PR at promotion time, same precedent as ADR-0054 §Notes
and ADR-0055 §Notes. The collapse rationale: the structural
choices Clauses 1–5 commit (the Reader interface extension; the
additive `source` label; the constant-zero fill discipline; the
SchedulerProxyMetrics convention; the 15s cadence constant) are
reviewer-load-bearing precisely because they are reviewable
against working code. Splitting would force the ADR reviewer to
evaluate prose-only choices that the code would either validate
or falsify in the next session.

**Condition 1 + 3 D0 carry-forward.** Per R5 + A7, the
operator-ratified weak reading of ADR-0039 §"Metric contract"
Meaning column wording, plus the additive-`source`-label
mechanism that satisfies Condition 3 under that reading, are
recorded here as **new contribution requiring review**. The
ratification was operator-side (mid-PR-#113) per
`CONTRIBUTING.md` Flow 5; this ADR carries the reading forward
for future B-rows touching ADR-0039 wording.

**Critique rounds.** This ADR's Decision survived one
`/critique` round before promotion (round-1 disposition recorded
in the PR body's Critique result table). The originating B3-5
study survived two rounds (1 = 0 blocking / 4 important / 5
minor with the four importants applied; 2 = ratification
trailer). The implementation code in this PR is self-verified
against AC-W3-3 + AC-W3-7 per ADR-0052 §6.4 row 6 close-gates
and ADR-0048 §"Skip" path for code-only `/critique` rounds.

**No ADR-0033 reopening.** Path B (scheduler-binary
instrumentation) remains Rejected per ADR-0049 §(a) and is not
re-litigated here. Future operator demand for a platform-owned
scheduler binary is wave-scale work per ADR-0033 §Notes; this
ADR explicitly does not authorize any engine-side proxy for
truly scheduler-tracked state (only zero-emission for the
engine-non-derivable label combos).
