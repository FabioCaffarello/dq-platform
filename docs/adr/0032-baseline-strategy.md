<!-- path: docs/adr/0032-baseline-strategy.md -->

# ADR-0032 — Baseline Strategy

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

Several anticipated check kinds compare a current value against
a historical reference: volume checks ("row count within X% of
the last 7 days"), freshness checks ("data is fresher than
typical lag"), distribution checks ("numeric distribution
within tolerance of historical"). Foundation 05 §"Check Design"
framed these as "baselined checks" and committed that the
specifics — where the baseline comes from, how sparse history
is handled, how cost is bounded — would resolve as a B1
decision.

The v1 catalog ([ADR-0022](./0022-kind-catalog.md)) carries
two kinds — `set.row_count_positive` and
`record.schema_conformance` — neither of which is baselined.
This ADR therefore commits the **framework** for baselined
kinds as design only; no engine code change ships here. The
first baselined kind lands under a future B2 follow-up
(registered in Consequences), implementing this framework
exactly as documented.

Related commitments this ADR builds on:

- [ADR-0003](./0003-result-write-model.md) commits
  `dq_executions` + `dq_check_results` as append-only — the
  platform's own history is durable and queryable.
- [ADR-0022](./0022-kind-catalog.md) commits the per-kind
  `params_schema`; baseline-related fields land inside
  `params.baseline` per kind, validated by the linter when
  the first baselined kind ships.
- [ADR-0026](./0026-failure-scope-aggregated.md) committed
  the record-mode vacuous-case split (insufficient input set
  → degraded, not fail). The sparse-history policy below
  mirrors this precedent.
- [ADR-0029](./0029-bigquery-cost-ceilings.md) committed
  `MaxBytesScannedPerRun` and the evaluator dry-run
  pre-flight. Baseline queries inherit this cost-ceiling
  mechanism without modification.
- [ADR-0031](./0031-evidence-retention-parameters.md)
  committed `ResultsRetention` (30 / 90 / 365 days per env)
  and the table-partitioning extension that makes
  partition-pruned baseline queries cheap.

The principles bearing on the decision:

- **P1 (rules must remain declarative).** Baseline shape
  lives in the rule artefact, not in engine code.
- **P2 (engine behavior must be deterministic).** "Same
  rule + same window + same source state → same outcome."
  The baseline query's window is pinned to
  `trigger.WindowEnd` so the same source state in
  `dq_check_results` produces the same baseline.
- **P4 (cost is a first-class constraint).** Baseline
  queries respect ADR-0029's `MaxBytesScannedPerRun`
  ceiling via the existing evaluator dry-run pre-flight.
- **R3 (do not revisit settled architecture).** The
  platform's own history (ADR-0003) is the baseline
  source; the engine code path stays read-only against
  the result tables; ADR-0003's append-only commitment is
  preserved.

---

## Decision

### Platform-history baselines with optional static baselines

Baselined kinds carry a `params.baseline` block. Two sub-modes
are committed:

- **`source: "platform_history"`** — baseline computed from
  prior `dq_check_results` rows for the same `(entity,
  check_id)` pair. The cheapest baseline primitive
  available; deterministic under P2's same-source-state
  qualifier; read-only against ADR-0003 tables.
- **`source: "static"`** — baseline declared by the rule
  author as a numeric `value`. Used when the expected value
  is known in advance and history would not add signal.

Data-plane baselines (re-querying the source table for
historical aggregates) and external-metric-system baselines
are explicitly rejected — the data-plane option doubles
query cost and breaks the source-state-pinned determinism
model; the external option pulls a new substrate dependency
outside ADR-0010's capability matrix.

### `params.baseline` block shape

```jsonc
// Illustration — params.baseline (platform_history sub-mode)
{
  "source": "platform_history",
  "reference_window": "7d",
  "min_samples": 5,
  "aggregation": "mean",
  "tolerance": { "type": "percent", "value": 20 }
}

// Illustration — params.baseline (static sub-mode)
{
  "source": "static",
  "value": 1000000,
  "tolerance": { "type": "percent", "value": 10 }
}
```

Field semantics:

- **`source`** — closed enum `[platform_history, static]`.
- **`reference_window`** *(required when source =
  platform_history)* — duration the baseline query covers.
  Lexical grammar: `^[0-9]+(ms|s|m|h|d)$`. **Parallel** to
  ADR-0024's record-mode window grammar
  (`^[0-9]+(ms|s|m|h)$`); the two grammars share the short
  suffixes by convention but are not coupled — ADR-0024
  governs Kafka window durations, this ADR governs set-mode
  baseline reference windows. The `d` (day) suffix supports
  the longer reference windows baselined set-mode checks
  typically need. The **effective** reference window at
  runtime is `min(declared, env's ResultsRetention)` —
  see §"Effective reference window vs declared" below.
- **`min_samples`** *(required when source =
  platform_history)* — minimum count of `pass` rows
  required to compute a fair baseline. Below this, the
  check returns `degraded` with reason
  `insufficient_baseline_samples`. Catalog-level default
  `5`; rule may override.
- **`aggregation`** *(required when source =
  platform_history)* — closed enum `[mean, median, min, max,
  p50, p90, p95, p99]`. Applied to the historical samples
  to produce a single baseline value.
- **`value`** *(required when source = static)* — the
  operator-declared baseline value. Numeric.
- **`tolerance`** *(required in both sub-modes)* — `type`
  is one of `[percent, absolute, stddev]`; `value` is the
  numeric tolerance. The check passes if
  `|current - baseline| <= tolerance applied to baseline`.
  `stddev` requires at least `min_samples` samples to
  compute and is history-mode-only.

### Baseline query (platform-history mode)

The baseline is computed from `dq_check_results` via a
helper at `engine/internal/eval/baselines.go` (lands with
the first baselined kind, not with this ADR):

```sql
SELECT
  -- the specific evidence-summary field the kind reads,
  -- e.g., row_count for a set.row_count_within_baseline kind
  <AGGREGATION>(JSON_VALUE(evidence_summary, '$.<field>'))
    AS baseline_value,
  COUNT(*) AS samples_used
FROM `<project>.<dataset>.dq_check_results` cr
JOIN `<project>.<dataset>.dq_executions` ex
  ON ex.execution_id = cr.execution_id
 AND ex.attempt_id   = cr.attempt_id
WHERE
  ex.entity = @entity
  AND cr.check_id = @check_id
  AND cr.result = 'pass'
  AND cr.executed_at < @window_end
  AND cr.executed_at >= @window_end - INTERVAL '<effective_reference_window>'
```

The helper signature:

```
ComputeBaseline(
    ctx context.Context,
    evalCtx *Evaluator,
    spec runner.CheckSpec,
    trigger runner.TriggerRequest,
) (baseline float64, samplesUsed int, err error)
```

Handlers for baselined kinds call `ComputeBaseline` and
compare the current evaluation's value against the returned
baseline within the rule's tolerance.

### Determinism (P2) — same-source-state qualifier

P2 commits "same rule + same window + same source state →
same outcome". The baseline query's `@window_end` is pinned
to `trigger.WindowEnd`. Re-running the same `execution_id`
produces the same `window_end` (per ADR-0002 CC1); given
the same source state in `dq_check_results` strictly before
that timestamp, the baseline is byte-identical. P2 is
honored.

Re-runs at different wall-clock times see **different
source states** — `dq_check_results` is append-only and
monotonically growing. A re-run that sees one or two
additional `pass` rows that landed between the original
and the re-run may produce a slightly different baseline
value. This is **expected per P2's source-state
qualifier**, not a P2 violation. The platform commits the
source-state-pinned behavior; the runbook (when the first
baselined kind lands) documents that literally-identical
outputs across re-runs require the source state to be
unchanged.

Strict literal reproducibility regardless of source-state
advance would require per-execution baseline-rowset
snapshotting to a side table. The implementation cost is
non-trivial; reserved as a deferred enhancement (see
Notes).

### Effective reference window vs declared

ADR-0031 commits per-env `ResultsRetention` values
(30 / 90 / 365 days per local / qa / prod).
[ADR-0005](./0005-manifest-publication-semantics.md)
commits that the same manifest deploys to every env
immutably. A rule declaring `reference_window: 200d`
therefore must not be rejected at local (30-day
retention) while accepted at prod (365-day retention);
rejection would break ADR-0005's
multi-env-from-one-manifest invariant.

The runtime contract is:

```
effective_reference_window = min(
    declared_reference_window,
    env's ResultsRetention,
)
```

The `ComputeBaseline` helper applies this cap at query
time. In local (30-day retention) a rule declaring 200d
reads 30 days of history; in prod (365-day retention) the
same rule reads 200 days. The sparse-history policy then
applies uniformly: if `samples_used < min_samples`,
`degraded` fires with reason
`insufficient_baseline_samples`.

This is the **silent-cap-with-degraded-signal** path. The
lint binary does **not** reject rules on the basis of
`reference_window > ResultsRetention` — the runtime
degraded signal carries the operational message when the
effective window is too short.

### Sparse-history policy

When the baseline query returns
`samples_used < min_samples`:

- **Status:** `degraded` (matches ADR-0026's vacuous-case
  precedent — insufficient input set → degraded, not
  fail).
- **Evidence summary:** `reason:
  "insufficient_baseline_samples"`, `samples_used:
  <count>`, `min_samples: <required>`,
  `effective_reference_window: <duration>`.
- **Alert routing:** degraded routes through the
  data-quality category per
  [ADR-0006](./0006-alert-routing-contract.md) CC7. The
  runbook seed for baselined-check degraded states ships
  when the first baselined kind lands (B2 follow-up).

The policy is uniform across modes: a record-mode kind
that consumes the baseline framework follows the same
path.

### Cost discipline (P4)

The baseline query reads from `dq_check_results` and
`dq_executions`. Both tables are partitioned by
`recorded_at` per ADR-0031. The partition-pruning
predicate (`executed_at >= window_end -
effective_reference_window`) makes the scan cost
proportional to `effective_reference_window × per-day
row count`. At typical usage (one execution per check per
day per entity, ~100 entities, ~10 checks each), a
365-day reference window scans ≈365,000 check-result
rows; at ~1 KB per row JSON, that's ≈350 MB —
comfortably under ADR-0029's `MaxBytesScannedPerRun`
(1 GB local / 100 GB qa / 1 TB prod).

The evaluator dry-run pre-flight committed by ADR-0029
runs against the baseline query before the real
execution. If the dry-run estimate exceeds
`MaxBytesScannedPerRun`, the handler returns
`ResultError` with reason `cost_ceiling_exceeded`, which
the runner maps to `status = aborted` per ADR-0029's
short-circuit. No new cost-ceiling mechanism is committed
here; baseline queries inherit the existing one.

### Why this does NOT reopen ADR-0003

The baseline query is a **read** against
`dq_check_results`. The engine code path issues no UPDATE
/ DELETE on the result tables. ADR-0003 CC1's
append-only commitment is preserved.

### Why this does NOT reopen ADR-0005

A rule's `reference_window` is **not** an env-scoped
field. The same manifest deploys to local + qa + prod
immutably; the runtime helper applies the
`min(declared, ResultsRetention)` cap per env. ADR-0005's
multi-env-from-one-manifest invariant is honored.

### Why this does NOT commit specific baselined kinds

The v1 catalog (ADR-0022) carries two kinds, neither
baselined. This ADR commits the **design** — the
`params.baseline` schema shape, the `ComputeBaseline`
helper signature, the baseline query template, the
sparse-history policy, the effective-window cap, the
cost-discipline integration — but does not commit any
specific baselined kind. The first baselined kind ships
under its own additive catalog extension (per ADR-0022's
open-for-extension contract) at a future date when
concrete operational signal justifies it. The
implementation of the helper, the schema fragment, and
the catalog extension all land in that same future slice.

This matches the
[ADR-0030](./0030-manifest-cryptographic-posture.md)
precedent: ADR-0030 committed a deferral with an
implementation path; no engine code shipped with that
ADR; the implementation slice would land when a trigger
condition fires. ADR-0032 follows the same posture:
design committed; implementation deferred to the
consumer slice.

---

## Consequences

1. **No engine code change ships from this ADR.** The
   `params.baseline` JSON-Schema fragment, the
   `ComputeBaseline` helper, and the baseline query template
   are **design commitments**; the implementation lands
   under the B2 follow-up that ships the first baselined
   kind (see Consequence 7). This matches the ADR-0030
   precedent — design committed in the ADR; implementation
   deferred to the consumer slice. The alternative
   (shipping unconsumed infrastructure now) would violate
   the operational guidance against designing for
   hypothetical future requirements.

2. **The design is the ADR.** Future baselined kinds
   consume the design exactly as documented:
   - `params.baseline` block shape (per the field semantics
     above) is the contract for any baselined-kind
     `params_schema` to reference.
   - `ComputeBaseline(ctx, evalCtx, spec, trigger)
     (baseline float64, samplesUsed int, err error)` is
     the helper signature the first baselined kind
     implements alongside the new helper file
     (`engine/internal/eval/baselines.go`).
   - The baseline query template (per §"Baseline query"
     above) is the SQL the helper executes.
   - The sparse-history → degraded policy is what the
     helper returns when `samples_used < min_samples`.

3. **The runtime effective-window cap is part of the
   helper's contract.** The helper applies
   `min(declared_reference_window, env's
   ResultsRetention)` at query time, so a rule that
   declares 200 days but runs in a 30-day-retention env
   reads only 30 days of history. The sparse-history
   policy handles the rest. No lint cross-check rejects
   rules on `reference_window > ResultsRetention` —
   ADR-0005's multi-env-from-one-manifest invariant
   would otherwise break.

4. **Sparse-history → degraded is uniform across modes.**
   `samples_used < min_samples` →
   `degraded` with reason `insufficient_baseline_samples`.
   The framework applies identically to set-mode and
   record-mode kinds. ADR-0026's vacuous-case precedent
   is the structural model.

5. **Static baselines as a sub-mode.** Rules can declare
   `source: static` for checks where the operator knows
   the expected value in advance. The framework supports
   both modes in the same `params.baseline` block; rule
   authors pick per check. No history read; no
   effective-window cap; sparse-history policy does not
   apply.

6. **`dq_check_results` and `dq_executions` are unchanged
   schema-wise.** The baseline helper (when it ships)
   reads existing columns (`check_id`, `result`,
   `executed_at`, `evidence_summary`, with `entity` via
   join). No new columns; no new tables.

7. **B2 follow-up: first baselined kind ships the
   framework implementation.** A new B2 row registers
   the first baselined kind — likely
   `set.row_count_within_baseline` driven by a concrete
   onboarded-entity need. That slice ships:
   - The new helper at `engine/internal/eval/baselines.go`
     with the `ComputeBaseline` signature.
   - The `params.baseline` JSON-Schema fragment at
     `rules/_schema/_baseline.fragment.json` (name
     pending the slice's convention) under the ADR-0001
     byte-equality mirror gate.
   - The new kind's handler that calls `ComputeBaseline`
     and applies the tolerance check.
   - The new kind's entry in
     `engine/internal/dsl/catalog/v1.yaml` referencing
     the fragment in `params_schema`.
   The B2 row is added at close-step assignment of a
   number.

8. **B2 follow-up: baselined-check degraded runbook.**
   When the first baselined kind lands, a runbook seed
   ships under `docs/runbooks/` for the
   degraded-on-sparse-history state — "what an operator
   does when a baselined check fires `degraded` with
   `reason: insufficient_baseline_samples`". A B2 row
   registers this; not needed until a baselined kind
   exists.

9. **No engine UPDATE/DELETE on result tables.** The
   helper (when it lands) is read-only. ADR-0003 CC1
   append-only commitment preserved.

10. **The platform's P1 + P2 + P4 commitments for
    baselined checks are now explicit.** P1
    (declarative): baseline shape lives in the rule
    artefact, not in engine code. P2 (determinism):
    window pinned at `trigger.WindowEnd`; the
    same-source-state qualifier covers re-run behavior.
    P4 (cost): baseline reads against partition-pruned
    `dq_check_results` fit inside ADR-0029's
    `MaxBytesScannedPerRun` ceiling via the existing
    evaluator dry-run.

---

## Open Questions

None blocking.

Two deferred items surfaced during the design phase and
are explicitly **out-of-scope for current cycle**:

- **OQ-1: Strict-P2 baseline snapshot.** A future
  amendment could ship per-execution baseline-row-set
  snapshotting to a side table at trigger acceptance,
  for literal byte-identical reproducibility across
  re-runs regardless of source-state advance. The
  implementation cost is non-trivial (new table, new
  write path that respects ADR-0003 CC1, retry
  semantics). Deferred until concrete operational signal
  (an incident or audit finding) shows the source-state
  delta matters in practice.

- **OQ-2: Cross-entity baselines.** A future baseline
  kind could compare an entity against sibling entities
  ("this entity's row count vs the median of all
  entities in the same dataset"). The framework's
  `params.baseline` schema would gain a
  `cross_entity_scope` field. Deferred until a concrete
  need surfaces; this ADR ships entity-internal
  baselines only.

---

## Notes

- The strict-P2 snapshot mechanism from OQ-1, if
  implemented, would write a per-execution baseline
  snapshot to a new table (e.g.,
  `dq_baseline_snapshots`) at trigger acceptance. The
  write would happen on the same code path as the
  running-row write (ADR-0003 CC3), keeping the
  append-only invariant. Re-runs would read from the
  snapshot rather than from `dq_check_results`,
  achieving literal byte-identical reproducibility. The
  cost is a 4th-table write per execution; deferred
  until operational signal justifies it.
- Cross-entity baselines (OQ-2) would extend the helper
  signature with an additional `entityScope` parameter
  and would require careful access-control review (one
  entity's check reading sibling entities' history is a
  new ownership-boundary surface; CODEOWNERS PR review
  would need to cover the cross-entity dependency
  declaration in the rule artefact).
- The framework supports any aggregation that can be
  expressed as a BigQuery aggregate function over a
  numeric field. New aggregations (e.g., MAD — median
  absolute deviation) extend the
  `params.baseline.aggregation` enum additively when
  needed; no ADR amendment required.
