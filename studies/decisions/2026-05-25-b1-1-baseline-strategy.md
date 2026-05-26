<!-- path: studies/decisions/2026-05-25-b1-1-baseline-strategy.md -->

# B1-1 — Baseline Strategy

## Context

Several anticipated check kinds compare a current value against a
historical reference: volume checks ("row count within X% of the
last 7 days"), freshness checks ("data is fresher than typical
lag"), distribution checks ("numeric distribution within tolerance
of historical"). Foundation 05 §"Check Design" frames these as
"baselined checks" and B1-1 was registered to resolve where the
baseline comes from and how the platform handles the
sparse-history case.

The v1 catalog (ADR-0022) carries two kinds — `set.row_count_positive`
and `record.schema_conformance` — neither of which is baselined.
B1-1 therefore commits the **framework** for baselined kinds, not
specific kinds. Future kinds that need a historical reference
consume the framework; v1 deployments do not yet exercise it.

Related ADRs that bear on the design:

- [ADR-0003](../../docs/adr/0003-result-write-model.md) commits
  `dq_executions` + `dq_check_results` as append-only — the
  platform's own history is durable and queryable.
- [ADR-0022](../../docs/adr/0022-kind-catalog.md) commits the
  per-kind `params_schema`; baseline-related fields land
  inside `params` per kind, validated by the linter.
- [ADR-0026](../../docs/adr/0026-failure-scope-aggregated.md)
  committed the record-mode vacuous-case split (zero records +
  zero late drops → pass; zero records + positive late drops →
  degraded). B1-1's sparse-history case is structurally
  similar — degraded status for "not enough samples".
- [ADR-0029](../../docs/adr/0029-bigquery-cost-ceilings.md)
  committed `MaxBytesScannedPerRun` — baseline queries must
  respect this ceiling.
- [ADR-0031](../../docs/adr/0031-evidence-retention-parameters.md)
  committed `ResultsRetention` (30 / 90 / 365 days per env);
  this bounds how far back baseline queries can read.

The principles bearing on the decision:

- **P2 (determinism).** Same rule + same data window + same
  trigger must produce the same baseline → same outcome. A
  baseline derived from monotonically-growing history can
  break P2 unless its window is pinned at the trigger.
- **P4 (cost is a first-class constraint).** Baseline queries
  scan historical rows; bytes-scanned grows with reference
  window. Must respect ADR-0029's `MaxBytesScannedPerRun`.
- **P1 (rules must remain declarative).** The baseline source
  must be a declared field on the rule, not a free-form code
  surface.
- **R3 (do not revisit settled architecture).** The platform's
  own history (ADR-0003) is the natural baseline source; a
  decision to query the data plane for baselines reopens the
  cost-discipline stack.

What B1-1 must commit:

1. **The baseline source.** Where do baselines come from?
2. **The baseline window primitive.** How is "reference period"
   declared on the rule and how is it bounded by retention?
3. **Determinism rule.** How is the baseline pinned so re-runs
   of the same execution_id produce the same outcome?
4. **Sparse-history policy.** What status fires when there
   isn't enough history to compute a fair baseline?
5. **Cost-discipline integration.** How baseline queries fit
   inside ADR-0029's ceilings.

---

## Decision Drivers

- **DD-1 — The platform's own history is the natural
  baseline source.** ADR-0003 commits `dq_check_results` as
  append-only and forensic-grade. Querying it for prior
  outcomes of the same (entity, check_id) is the cheapest,
  most deterministic baseline primitive available. Using the
  data plane directly (re-scanning the source table for a
  baseline aggregate) doubles the cost surface.
- **DD-2 — Baselines must be pinned to the trigger window
  for P2 determinism.** Re-running the same `execution_id`
  must produce the same outcome. The baseline query
  therefore filters by `executed_at < trigger.WindowEnd`,
  so the historical sample set is reproducible from the
  trigger's identity inputs (ADR-0002 CC1) alone.
- **DD-3 — Sparse history must yield degraded, not fail.**
  A new entity or a recently-onboarded check has zero
  historical samples. A check that fails the entity on
  sparse history would punish the platform's own onboarding
  flow. The structurally similar precedent is ADR-0026's
  record-mode vacuous case (zero records + zero late drops
  → pass; zero records + positive late drops → degraded);
  B1-1 mirrors this: insufficient samples → degraded with
  a reason that names the gap, not fail.
- **DD-4 — Baseline window must be bounded by retention.**
  ADR-0031 commits `ResultsRetention` (30 / 90 / 365 days).
  A baseline rule declaring "last 365 days" against a local
  env with 30-day retention has no data to read past
  30 days. The platform must commit how this mismatch
  resolves: either silently truncate to the available
  window (operator-friendly but hides the gap) or refuse
  the rule (loud but blocks deployment). The study picks
  one explicitly.
- **DD-5 — Static operator-declared baselines are also
  needed.** Some checks have a fixed expected value
  ("row count should be 1,000,000 ±10%"). The framework
  must support both history-derived and static baselines
  inside the same `params` schema, so rule authors can pick
  per check.
- **DD-6 — B1-1 commits the framework, not specific
  kinds.** v1 ships no baselined kinds. Future kinds
  (`set.row_count_within_baseline`,
  `set.freshness_within_baseline`,
  `set.distribution_within_baseline`) consume the framework
  when they land. The study commits the schema shape, the
  baseline-query helper, the sparse-history policy, and the
  cost-discipline integration — without committing the
  specific kinds.

---

## Considered Options

### Option 1 — Platform-history baselines with optional static baselines (recommended)

Baselined kinds carry a `params.baseline` block declaring:

- `source: "platform_history" | "static"` — where the
  baseline comes from
- For `source: platform_history`:
  - `reference_window: <duration>` — how far back to look
  - `min_samples: <int>` — minimum sample count to compute
    a fair baseline
  - `aggregation: "mean" | "median" | "min" | "max" |
    "p<N>"` — how to summarise the historical samples
  - `tolerance: { type: "percent" | "absolute" | "stddev",
    value: <number> }` — how far from baseline counts as
    pass
- For `source: static`:
  - `value: <number>` — the operator-declared baseline
  - `tolerance: { ... }` — same tolerance shape

The platform-history path reads from `dq_check_results`
filtered by `entity`, `check_id`, `executed_at <
trigger.WindowEnd` (P2 pin), and `executed_at >=
trigger.WindowEnd - reference_window`. Result rows are
restricted to `result = pass` so a baseline polluted by
prior failures does not normalise the failure bar.

**Strengths.** Cheapest baseline primitive (reads the
platform's own history, not the data plane); deterministic
under P2 (window is pinned to trigger.WindowEnd); fits
ADR-0029 cost ceilings (bytes-scanned bounded by
reference_window × historical row size, which is small);
honors ADR-0003 (no write-path change). Both history-derived
and static baselines available; rule authors pick per check.
Sparse-history degraded mirrors ADR-0026's vacuous-case
precedent.

**Trade-offs.** Baseline window capped at `ResultsRetention`
(30 / 90 / 365 days per env). A check that wants longer
history than retention allows is rejected at lint time —
this is a real constraint, and the security note documents
it for rule authors.

### Option 2 — Data-plane baselines

Each baselined check issues a second BigQuery query against
the source table to compute the baseline (e.g.,
`SELECT AVG(row_count) FROM <entity> WHERE date >= last_week`).

**Strengths.** Baseline is computed from the same data
plane as the check, so it tracks data evolution exactly.
No dependence on `dq_check_results` retention.

**Trade-offs.** Doubles the data-plane query cost (each
check evaluation runs two queries — main + baseline).
Bytes-scanned grows linearly with reference window applied
to the source table size, which can be very large. Breaks
the determinism model — the data plane is mutable; a
re-run with the same `execution_id` may read different
source rows. Pulls P2 (determinism) into tension with
freshness. Doubles the cost discipline surface and would
need its own ceiling separate from ADR-0029.

### Option 3 — External baseline source

Baselines come from an external metric system (e.g., a
time-series database that the data engineering team uses).
The platform queries that system via an adapter.

**Strengths.** Decouples the baseline from the platform's
own history; integrates with whatever metric system the
operator's stack uses.

**Trade-offs.** Pulls a new substrate dependency that's
not in ADR-0010's capability matrix. Introduces an adapter
surface the platform doesn't currently have. The
substrate-selection checkpoint from ADR-0010 + ADR-0017
would have to commit the external system as a substrate.
Defers the baseline question by introducing a larger
question. Mentioned only to be rejected — the platform's
own history is the natural fit.

### Option 4 — Static baselines only

Rules declare a numeric expected value and tolerance
band; no history-derived baselines.

**Strengths.** Simplest; no baseline query at all; pure
rule-author declaration.

**Trade-offs.** Limits the check vocabulary to "expected
value known in advance". Doesn't handle volume drift,
seasonal patterns, or organic growth without operator
re-declaring values periodically — which becomes
operational churn. Useful as a sub-mode of Option 1
(operator can opt into static where it fits), but
insufficient as the only option.

---

## Recommendation

**Option 1.** Platform-history baselines as the primary mode;
static baselines as a sub-mode in the same `params.baseline`
schema.

### `params.baseline` block shape

The catalog ships a reusable JSON-Schema fragment that any
baselined kind references in its `params_schema`:

```jsonc
// Illustration — baseline block (lives under params.baseline)
{
  "source": "platform_history",
  "reference_window": "7d",
  "min_samples": 5,
  "aggregation": "mean",
  "tolerance": { "type": "percent", "value": 20 }
}

// OR — static baseline
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
  Lexical grammar: `^[0-9]+(ms|s|m|h|d)$`. This is a
  **parallel** grammar to ADR-0024's record-mode window
  grammar (`^[0-9]+(ms|s|m|h)$`); the two grammars share
  the short suffixes by convention but are not coupled —
  ADR-0024 governs Kafka window durations only, B1-1
  governs set-mode baseline reference windows only. The
  `d` (day) suffix supports the longer reference windows
  baselined set-mode checks typically need. The **effective**
  reference window at runtime is
  `min(declared, env's ResultsRetention per ADR-0031)`;
  see §"Effective reference window vs declared" below.
- **`min_samples`** *(required when source =
  platform_history)* — minimum count of `pass` rows
  required to compute a fair baseline. Below this, the
  check returns `degraded` with reason
  `insufficient_baseline_samples`. Default `5` at the
  catalog level; rule may override.
- **`aggregation`** *(required when source =
  platform_history)* — closed enum `[mean, median, min, max,
  p50, p90, p95, p99]`. The aggregation applied to the
  historical samples. `p<N>` aggregations consume the same
  evidence-summary numeric fields the catalog kind emits
  (e.g., row_count for a row-count check).
- **`value`** *(required when source = static)* — the
  operator-declared baseline value. Numeric.
- **`tolerance`** *(required in both sub-modes)* — `type` is
  one of `[percent, absolute, stddev]`; `value` is the
  numeric tolerance. The check passes if `|current -
  baseline| <= tolerance applied to baseline`. For `stddev`
  the tolerance is in standard-deviation units, requires at
  least `min_samples` samples to compute, and is
  history-mode only.

### Baseline query (platform-history mode)

The baseline is computed from `dq_check_results` via the
following template query, executed by a new helper in
`engine/internal/eval/baselines.go`:

```sql
SELECT
  -- the specific evidence-summary field the kind looks at,
  -- e.g., row_count for set.row_count_within_baseline
  <AGGREGATION>(JSON_VALUE(evidence_summary, '$.<field>'))
    AS baseline_value,
  COUNT(*) AS samples_used
FROM `<project>.<dataset>.dq_check_results`
WHERE
  -- entity is implicit through the join below; the
  -- check_id filters across entities is wrong, so we
  -- join with dq_executions for the entity gate.
  check_id = @check_id
  AND result = 'pass'
  AND executed_at < @window_end
  AND executed_at >= @window_end - INTERVAL '<reference_window>'
```

Joined with `dq_executions` on `(execution_id, attempt_id)` so
the entity filter applies (entity lives on `dq_executions`,
not `dq_check_results`).

The helper returns `(baseline_value, samples_used)` to the
calling handler, which compares against the current
evaluation's value within the rule's `tolerance`.

### Determinism (P2) — same-source-state condition

P2 commits "same rule + same window + same source state →
same outcome". The baseline query's `@window_end` is
pinned to `trigger.WindowEnd`. Re-running the same
`execution_id` produces the same `window_end` (per
ADR-0002 CC1); given the same source state in
`dq_check_results` strictly before that timestamp, the
baseline is byte-identical and the outcome is reproducible.
P2 is honored.

Re-runs at different wall-clock times see
**different source states** (`dq_check_results` is
append-only and monotonically growing). A re-run that
sees one or two additional `pass` rows that landed
between the original and the re-run may produce a
slightly different baseline value — but this is
**expected per P2's source-state qualifier**, not a P2
violation. The platform commits the source-state-pinned
behavior; operators reading the runbook understand that
literally-identical outputs across re-runs require the
source state to be unchanged.

A future amendment could ship per-execution baseline-row
snapshotting to a side table for strict literal
reproducibility regardless of source-state advance. The
cost is non-trivial; reserved as a deferred enhancement
(see OQ-1).

### Sparse-history policy

When the baseline query returns `samples_used < min_samples`:

- **Status:** `degraded` (matches ADR-0026's vacuous-case
  precedent — degraded, not fail, when the input set is
  insufficient).
- **Evidence summary:**
  `reason: "insufficient_baseline_samples"`,
  `samples_used: <count>`, `min_samples: <required>`.
- **No alert escalation:** degraded routes through the
  data-quality category per ADR-0006 CC7. The runbook seed
  for baselined-check degraded states ships when the first
  baselined kind lands (deferred B2 follow-up).

This policy is uniform across modes: a record-mode kind
with insufficient history follows the same path.

### Effective reference window vs declared

ADR-0031 commits per-env `ResultsRetention` values
(30 / 90 / 365 days). ADR-0005 commits that the same
manifest deploys to every env immutably — the manifest
is content-addressed, and a rule must work across all
envs. A rule that declares
`reference_window: 200d` therefore must not be rejected
at local (30-day retention) or qa (90-day retention)
while accepted at prod (365-day retention); doing so
would break ADR-0005's multi-env-from-one-manifest
invariant.

The platform's commitment at runtime is:

```
effective_reference_window = min(declared, env's ResultsRetention)
```

The `ComputeBaseline` helper applies this cap at query
time. In local (30-day retention) a rule declaring 200d
reads 30 days of history; in prod (365d retention) the
same rule reads 200 days. The sparse-history policy then
applies uniformly: if `samples_used < min_samples` (per
the rule), `degraded` fires with reason
`insufficient_baseline_samples`.

This is the **silent-cap-with-degraded-signal** path. No
lint cross-check rejects rules on the basis of
`reference_window > ResultsRetention` — the runtime
degraded fires loudly when the effective window
doesn't yield enough samples; that's the operational
signal rule authors act on. The lint cross-check is
explicitly **not** committed here.

### Cost discipline (P4)

The baseline query reads from `dq_check_results` and
`dq_executions`. Both tables are partitioned per ADR-0031,
so the partition-pruning predicate
(`executed_at >= window_end - reference_window`) makes the
scan cost proportional to `reference_window × per-day row
count`. At typical platform usage (one execution per check
per day per entity, ~100 entities, ~10 checks each), a
365-day reference window scans ≈365,000 check-result rows;
at ~1 KB per row JSON, that's ≈350 MB. Comfortably under
ADR-0029's `MaxBytesScannedPerRun` (1 GB local / 100 GB qa
/ 1 TB prod).

The evaluator dry-run pre-flight committed by ADR-0029 §"Runtime
layer" runs against the baseline query before the real
execution. If the dry-run estimate exceeds
`MaxBytesScannedPerRun`, the handler returns `ResultError`
with reason `cost_ceiling_exceeded`, which the runner maps
to `status = aborted` per ADR-0029's short-circuit. No new
cost-ceiling mechanism is committed; baseline queries
inherit the existing one.

### Why this does NOT reopen ADR-0003

The baseline query is a **read** against `dq_check_results`.
The engine code path issues no UPDATE / DELETE on the
result tables. ADR-0003 CC1's append-only commitment is
preserved.

### Why this does NOT commit specific baselined kinds

The v1 catalog (ADR-0022) carries two kinds, neither
baselined. B1-1 commits the framework — the
`params.baseline` schema fragment, the baseline helper, the
sparse-history policy, the cost-discipline integration —
but does not commit any specific baselined kind. The first
baselined kind ships under its own additive catalog entry
(per ADR-0022's open-for-extension contract) at a
future date when concrete operational signal justifies it.
Deferred via a B2 follow-up registered in Consequences.

---

## Consequences

1. **No engine code change ships from this ADR.** The
   `params.baseline` JSON-Schema fragment, the
   `ComputeBaseline` helper, and the baseline query template
   are **design commitments**; the implementation lands
   under a future B2 follow-up that ships the first
   baselined kind (see Consequence 7). This matches the
   B1-8 / ADR-0030 precedent — design committed in the ADR;
   implementation deferred to the consumer slice. The
   alternative (shipping unconsumed infrastructure now)
   would violate the operational guidance against designing
   for hypothetical future requirements.

2. **The design is the ADR.** Future baselined kinds
   consume the design exactly as documented:
   - `params.baseline` block shape (per the field semantics
     above) is the contract for any baselined-kind
     `params_schema` to reference.
   - `ComputeBaseline(ctx, evalCtx, spec, trigger)
     (baseline float64, samplesUsed int, err error)` is
     the helper signature the first baselined kind
     implements + the new helper file
     (`engine/internal/eval/baselines.go`) lands with.
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

4. **The sparse-history policy is uniform across set-mode
   and record-mode.** `samples_used < min_samples` →
   `degraded` with `reason:
   insufficient_baseline_samples`. ADR-0006 CC7 routes
   degraded through the data-quality category;
   ADR-0026's record-mode vacuous-case precedent is
   structurally aligned.

5. **`dq_check_results` and `dq_executions` are unchanged
   schema-wise.** The baseline helper reads existing
   columns (`check_id`, `result`, `executed_at`,
   `evidence_summary`, `entity` via join). No new
   columns; no new tables.

6. **No engine UPDATE / DELETE on result tables.** The
   helper is read-only. ADR-0003 CC1 append-only commitment
   preserved.

7. **B2 follow-up: first baselined kind.** A new B2 row
   registers the first baselined kind (likely
   `set.row_count_within_baseline` driven by a concrete
   onboarded-entity need). The kind ships under ADR-0022's
   additive catalog extension; the kind's handler calls
   `ComputeBaseline`; the kind's `params_schema` references
   the `params.baseline` fragment. v1 does not commit the
   specific kind; the B2 row is added at close-step
   assignment of a number.

8. **B2 follow-up: baselined-check degraded runbook.**
   When the first baselined kind lands, a runbook seed
   ships under `docs/runbooks/` for the degraded-on-sparse-
   history state — "what an operator does when a baselined
   check fires `degraded` with `reason:
   insufficient_baseline_samples`". A B2 row registers
   this; not needed until a baselined kind exists.

9. **Static baselines as a sub-mode.** Rules can declare
   `source: static` for checks where the operator knows
   the expected value in advance. The framework supports
   both modes in the same `params.baseline` block; rule
   authors pick per check. The lint cross-check on
   reference_window does not apply to static baselines
   (no history read).

10. **Near-determinism trade-off documented.** Strict
    P2 would require snapshotting the historical rowset
    per execution to a side table — heavy. v1 accepts
    that a re-run of the same `execution_id` may include
    one or two additional historical samples that landed
    between the original run and the re-run; the
    bounded delta is within `min_samples` tolerance for
    the vast majority of cases. The strict-P2 version is
    reserved as a deferred amendment if operational
    signal shows the delta matters.

11. **The platform's P1 + P2 + P4 commitments for
    baselined checks are now explicit.** P1 (declarative):
    baseline shape lives in the rule artefact, not in
    engine code. P2 (determinism): window pinned at
    `trigger.WindowEnd` with the near-determinism
    trade-off documented. P4 (cost): baseline reads
    against partition-pruned `dq_check_results` fit inside
    ADR-0029's `MaxBytesScannedPerRun` ceiling.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting are explicitly
**out-of-scope for current cycle**:

- **OQ-1: Strict-P2 baseline snapshot.** A future amendment
  could ship a per-execution baseline-row-set snapshot
  written to a side table at trigger acceptance, so re-runs
  of the same `execution_id` see byte-identical historical
  samples. The implementation cost is non-trivial (new
  table, new write path that respects ADR-0003 CC1, retry
  semantics). Deferred until concrete operational signal
  (an incident or audit finding) shows the near-determinism
  delta matters in practice.

- **OQ-2: Cross-entity baselines.** A future baseline kind
  could compare an entity against sibling entities ("this
  entity's row count vs the median of all entities in the
  same dataset"). The framework's `params.baseline` schema
  would gain a `cross_entity_scope` field. Deferred until
  a concrete need surfaces; v1 ships entity-internal
  baselines only.

---

## Promotion target

`docs/adr/0032-baseline-strategy.md` — ships the design
commitments only (no engine code change): the
`params.baseline` JSON-Schema fragment shape, the
`ComputeBaseline(ctx, evalCtx, spec, trigger)` helper
signature, the baseline query template, the
sparse-history → degraded policy, the runtime
effective-window cap
`min(declared, env's ResultsRetention)`, P2's
same-source-state qualifier, and the two B2 follow-ups
(first baselined kind; degraded-on-sparse-history
runbook seed).
