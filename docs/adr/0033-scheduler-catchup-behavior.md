<!-- path: docs/adr/0033-scheduler-catchup-behavior.md -->

# ADR-0033 — Scheduler Catchup Behavior

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

The platform processes both scheduled and manually-triggered
evaluations. The trigger-source enum committed by
[ADR-0002](./0002-run-identity-and-idempotency.md) CC6 names
the three sources: `scheduler`, `manual`, `operator-rerun`.
[ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md)
§5 + §6 committed that the scheduler is **external to the
engine**: lifecycle operations are attempted per-trigger
best-effort; orphans are external triggers not in the active
manifest's rule set.
[ADR-0014](./0014-trigger-handler-contract.md) commits the
HTTP `/v1/trigger` endpoint as the engine's trigger-acceptance
surface; the engine runs no internal scheduler.

What this ADR commits, given that posture, is the
**scheduling contract** the platform expects external
schedulers to honor:

1. **Cadence declaration.** Where does the per-rule cadence
   live? On the rule artefact, on a companion file, or in
   operator-level scheduler config?
2. **Catchup posture.** When the external scheduler was
   down and missed N windows, what should it do?
3. **Missed-window detection.** How does the operator know
   a scheduled trigger never fired?
4. **Manual trigger semantics.** Manual triggers can
   target any window; how do they coexist with scheduled
   triggers for the same window?
5. **Backfill posture.** When a new entity is onboarded,
   does the platform backfill historical windows
   automatically?

Existing commitments this ADR builds on:

- ADR-0002 CC1 defines `execution_id` as a hash over
  `(ruleset_version, engine_version, entity, window_start,
  window_end, trigger_source)`. Two triggers with different
  `trigger_source` for the same window produce different
  execution_ids; two triggers with the same `trigger_source`
  for the same window collide on execution_id, and
  ADR-0002 CC2 / CC5 commit the rerun / upsert / supersedes
  semantics.
- [ADR-0029](./0029-bigquery-cost-ceilings.md) commits
  `MaxConcurrentEvaluations` (via the semaphore the trigger
  handler acquires) and `MaxWindowDuration` (rejects
  triggers whose window exceeds the per-env ceiling).
- [ADR-0031](./0031-evidence-retention-parameters.md)
  commits the partition-pruned tables that make a
  cross-execution scan cheap.

This ADR must NOT reopen any of these. The engine code
path (loader, runner, trigger handler) is unchanged. The
ADR commits the **contract** external schedulers honor and
the **read surface** external monitors use to detect
missed triggers — design only; implementation lands with
the first scheduler-consumer slice (registered as a B2
follow-up).

The principles bearing on the decision are **P1** (rules
must remain declarative — cadence is a property of the
entity, declared on the rule), **P3** (ownership is
explicit — operators own scheduler choice + catchup
tuning), and **R3** (do not revisit settled architecture —
ADR-0007's external-scheduler commitment is preserved).

---

## Decision

### External scheduler + advisory `schedule` field + per-env catchup horizon

The platform's role:

- **Acceptance** continues per ADR-0014 — `/v1/trigger`
  accepts triggers with any `trigger_source` and any
  window. No new acceptance behavior is committed here.
- **Contract:** the v2 rule schema gains an optional
  `schedule` field carrying an advisory cadence
  declaration. External schedulers READ this at
  manifest-publish time and provision their own
  triggers; the engine does not interpret or enforce it.
- **Catchup horizon guidance:** the platform exposes a
  per-env `SchedulerCatchupHorizon` value documenting
  "don't emit triggers older than X" — advisory, not
  engine-enforced.
- **Missed-window query:** a new reader method returns
  the most-recent execution per `(entity, check_id)` so
  external monitors can detect gaps.

An internal engine-side scheduler was considered and
rejected — it would reopen ADR-0007 §5 + §6 (external
scheduler is part of the committed architecture), pull
scheduler reliability into the engine, and add
leader-election complexity to multi-replica deployments
without operational benefit at v1.

A pure-HTTP-triggered posture (no cadence concept on the
rule at all) was also considered and rejected — it loses
the declarative-near-the-rule property (P1) and moves
cadence changes outside the CODEOWNERS PR-review path
(ADR-0015), weakening the platform's ownership posture
(P3). The optional `schedule` field is cheap (additive
per ADR-0001) and preserves the declarative pattern.

### v2 rule schema extension — `schedule` field

The v2 rule schema gains an optional top-level
`schedule` field declaring the advisory cadence:

```yaml
# Illustration — orders_stream.yaml (record-mode) with schedule
version: 2
entity: orders_stream
mode: record
schedule: "0 * * * *"   # hourly, advisory; external scheduler reads this
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema:
        type: object
```

Field semantics:

- **Optional.** Rules without `schedule` are pure
  HTTP-triggered (operators emit triggers manually or
  via an external system not driven by the rule's
  declared cadence).
- **Advisory.** The engine does not interpret or enforce
  the field. External schedulers READ the manifest at
  publish time and provision their own triggers
  accordingly.
- **Lexical grammar:** either a **5-field cron
  expression** containing internal whitespace
  (`"0 0 * * *"`, `"*/15 * * * *"`) or a **duration
  literal** with no whitespace and the
  `^[0-9]+(ms|s|m|h|d)$` suffix grammar (`"1h"`,
  `"15m"`, `"1d"`). The two forms are disambiguated by
  the presence of internal whitespace: a value with
  spaces is parsed as cron; a contiguous value with a
  duration suffix is parsed as duration. Cron aliases
  (`@daily`, `@hourly`, etc.) are explicitly NOT
  supported at v1 — the duration-literal form covers
  the same cases (`1d` ≡ `@daily`, `1h` ≡ `@hourly`)
  unambiguously. The lint binary will validate the
  grammar when the scheduler-consumer slice lands; the
  engine does not interpret the field at any phase.
- **Mode-independent.** Both set-mode and record-mode
  rules can carry a `schedule`. For record-mode rules,
  the schedule typically aligns with
  `source.kafka.window.duration` — but the alignment is
  the author's responsibility, not platform-enforced.

This extension is **additive** per ADR-0001's
compatibility contract (a new optional field within v2).
No schema-version bump.

### Per-env `SchedulerCatchupHorizon` (advisory)

A new typed field on `EnvConfig` exposes a per-env
catchup-horizon value:

```
type EnvConfig struct {
    // ... existing fields ...

    // SchedulerCatchupHorizon documents the maximum age of
    // a scheduled trigger the external scheduler should
    // emit. Triggers older than this are skipped by
    // convention; the platform does not enforce this
    // ceiling at the engine layer (the trigger handler
    // accepts any trigger per ADR-0014).
    SchedulerCatchupHorizon time.Duration
}
```

Per-environment values: **`SchedulerCatchupHorizon =
1 h / 6 h / 24 h`** for `local / qa / prod`.

**Per-value rationale.** Local 1h keeps the dev feedback
loop tight (a missed window older than an hour usually
isn't worth catching up during a single-developer
iteration); qa 6h matches the typical integration test
window (overnight integration runs catch up cleanly the
next morning); prod 24h tolerates a full day of scheduler
downtime without losing forensic fidelity. Values longer
than 24h amplify the cost spike at recovery (an N-hour
backlog triggers N concurrent evaluations bounded by
ADR-0029's `MaxConcurrentEvaluations`) and would benefit
from the explicit backfill posture below instead.

The value is **advisory** — the engine does not enforce
it. External schedulers READ the value (or read a
deployment-config copy that matches) and apply it to
their catchup logic. The platform's commitment is the
documented value, not a runtime check.

### Catchup posture

When the external scheduler was down and missed N
windows:

- **Default posture:** the scheduler emits triggers
  back to the catchup horizon, skipping anything older.
  Example: scheduler down for 18 hours in prod (24h
  horizon) emits all 18 missed hourly triggers when it
  recovers.
- **Operator override:** the external scheduler tool's
  own configuration is the override surface. The
  platform does not commit a per-rule catchup-override
  mechanism at v1.
- **Cost during catchup:** each catchup trigger goes
  through the standard `/v1/trigger` acceptance,
  including the ADR-0029 `MaxConcurrentEvaluations`
  semaphore. A 18-trigger burst into a 4-concurrent
  local engine queues; into a 64-concurrent prod
  engine runs almost in parallel. Operators tune
  `MaxConcurrentEvaluations` per env if catchup spikes
  are a cost concern.

### Manual trigger semantics

Per ADR-0002 CC1, manual triggers produce **distinct
execution_ids** from scheduled triggers for the same
window (different `trigger_source` input to the hash).
They coexist in `dq_executions` as separate executions;
neither supersedes the other.

The operational guidance:

- **Manual trigger** = ad-hoc evaluation for any
  window. Does NOT replace the scheduled run. Useful
  for: investigating an issue, validating a rule
  change, evaluating a back-dated window the scheduler
  didn't cover.
- **Operator-rerun** = explicit re-evaluation of a
  PRIOR execution. The new execution carries
  `SupersedesExecutionID` per ADR-0002 CC5 + ADR-0003
  CC5 so the lineage is preserved. Use operator-rerun
  when the goal is to replace a prior result; use
  manual when the goal is a parallel/ad-hoc evaluation.

A runbook seed clarifying this distinction ships when
the first scheduler-consumer slice lands (B2 follow-up).

### Missed-window detection — query surface

The platform exposes a new reader method on
`engine/internal/results.Store`:

```
LatestExecutionPerEntityCheck(
    ctx context.Context,
    asOf time.Time,
) ([]LatestExecutionRow, error)

type LatestExecutionRow struct {
    Entity        string
    CheckID       string
    LatestEnd     time.Time   // the latest window_end seen
    LatestStatus  ExecutionStatus
    Mode          Mode
}
```

The `asOf` parameter is the **snapshot point** the
query runs against — the method returns the latest
execution per `(entity, check_id)` whose `recorded_at <=
asOf`. External monitors checking "has each rule run in
the last hour?" pass `asOf = time.Now()`; retrospective
investigations pass an earlier timestamp to reconstruct
the state of the world as of that moment. The query
joins `dq_executions` + `dq_check_results`, groups by
`(entity, check_id)`, and returns the maximum
`window_end` per group with the status of that
execution. External monitors call this to detect "no
execution seen for (entity X, check Y) in the last
hour" gaps.

The method lands with the first scheduler-consumer
slice; it is not committed for implementation in this
ADR. The query template is documented here so the
consumer slice has a concrete starting point.

### Backfill posture

When a new entity is onboarded:

- **v1:** no automatic historical backfill. The
  operator's choice. Backfill via `/v1/trigger` with
  `trigger_source: manual` and the desired window
  range. Multiple manual triggers (one per historical
  window) produce multiple `dq_executions` rows; no
  upsert collision because each carries a unique
  `(window_start, window_end)` tuple.
- **Future enhancement:** an "auto-backfill on
  manifest publish" feature could emit manual triggers
  for the last N windows when a rule is added.
  Reserved as a B2 follow-up; not committed here.

### Why this does NOT reopen ADR-0007

ADR-0007 §5 + §6 commits the external-scheduler
posture: lifecycle operations are per-trigger
best-effort; orphans are external triggers not in the
manifest. This ADR confirms the posture and commits
the contract external schedulers honor:

- The advisory `schedule` field is READ by external
  schedulers, not by the engine.
- The `SchedulerCatchupHorizon` is documentation for
  external schedulers, not an engine-enforced ceiling.
- The missed-window query is a read surface for
  external monitors, not a scheduling primitive.

The engine code path (loader, runner, trigger handler)
is unchanged. ADR-0007 stays accepted without
amendment.

### Why this does NOT reopen ADR-0014

ADR-0014 commits the `/v1/trigger` endpoint with
strict decoder + async runner dispatch. This ADR
adds no new endpoint behavior. Manual triggers and
catchup triggers go through the existing endpoint
with the existing payload shape. The
`LatestExecutionPerEntityCheck` read method is a
Store-level surface, not a new endpoint.

### Why this does NOT commit specific scheduler tooling

The platform supports any external scheduler that can
emit HTTP POST `/v1/trigger`. Kubernetes CronJob is
the natural fit given the platform's substrate posture
(ADR-0010 + ADR-0028 commit Kubernetes-native
deployment); any other cron-compatible scheduler the
operator's stack provides is equally permissible. The
choice is operator-scoped; the platform commits the
**contract** the scheduler honors, not the specific
tool.

---

## Consequences

1. **No engine code change ships from this ADR.** The
   advisory `schedule` field on v2 rules, the
   `EnvConfig.SchedulerCatchupHorizon` field, and the
   `LatestExecutionPerEntityCheck` reader method are
   **design commitments**; the implementation lands
   under the B2 follow-up that ships the first
   scheduler-consumer slice. Matches the
   [ADR-0030](./0030-manifest-cryptographic-posture.md)
   / [ADR-0032](./0032-baseline-strategy.md) precedents
   — design committed; implementation deferred.

2. **The v2 rule schema gains an additive optional
   `schedule` field.** When the first
   scheduler-consumer slice ships, the v2 schema and
   its rules-side mirror gain the optional field (5-
   field cron expression or duration literal); the
   lint binary validates the grammar; the engine does
   not interpret it. ADR-0001's additive-within-major
   contract is honored; no schema-version bump.

3. **`EnvConfig.SchedulerCatchupHorizon` ships as
   advisory guidance.** When the first scheduler-
   consumer slice ships, the new field lands with
   per-env values (1h / 6h / 24h per local / qa /
   prod). The engine does not enforce it; external
   schedulers respect it as documented convention.

4. **`Store.LatestExecutionPerEntityCheck` ships as a
   read surface.** When the first scheduler-consumer
   slice ships, the method lands on the
   `engine/internal/results` Reader interface
   alongside `QueryCurrentExecution`. The
   BigQueryStore implementation issues a
   partition-pruned query (partition expiration per
   ADR-0031 bounds the scan).

5. **Manual triggers continue to produce distinct
   execution_ids.** ADR-0002 CC1 is unchanged. The
   operational expectation is documented: manual ≠
   replacement of scheduled; operator-rerun is the
   replacement mechanism.

6. **Backfill is operator-driven at v1.** Multiple
   manual triggers per historical window. No upsert
   collisions because each window is distinct. Auto-
   backfill on manifest publish is reserved as a B2
   follow-up.

7. **Catchup is the external scheduler's
   responsibility.** The platform commits the
   `SchedulerCatchupHorizon` guidance but does not
   enforce it; the ADR-0029 `MaxConcurrentEvaluations`
   semaphore in the trigger handler is the
   cost-discipline gate during catchup bursts.

8. **B2 follow-up: first scheduler-consumer slice.**
   A new B2 row registers the first slice that
   consumes this ADR's design surface. The slice
   ships:
   - The optional `schedule` field on the v2 rule
     schema (canonical + mirror; lint-validated
     grammar).
   - The `EnvConfig.SchedulerCatchupHorizon` field
     with per-env values.
   - The `LatestExecutionPerEntityCheck` reader
     method on `engine/internal/results.Store`.
   - A reference external scheduler (likely a
     Kubernetes CronJob manifest under
     `deploy/overlays/`) that reads the `schedule`
     field and emits triggers.
   The B2 row is added at close-step assignment of a
   number.

9. **B2 follow-up: manual-vs-operator-rerun runbook
   seed.** The runbook clarifying when to use
   `trigger_source: manual` vs `trigger_source:
   operator-rerun` ships when the first
   scheduler-consumer slice lands. A B2 row registers
   this seed.

10. **The platform's P1 + P3 commitments for
    scheduling are now explicit.** P1 (declarative):
    cadence lives on the rule artefact, not in engine
    code. P3 (ownership): operators own the scheduler
    choice and the per-env catchup-horizon tuning;
    rule authors + CODEOWNERS reviewers own the
    cadence declaration at PR-review time.

---

## Notes

- The `schedule` field's lexical grammar is
  intentionally narrow at v1 (5-field cron OR
  duration literal; no cron aliases). The narrow
  surface keeps the linter implementation simple and
  the rule-author error surface clear. Extensions
  (6-field cron with seconds; macros; time-zone-aware
  cron) can be added additively per ADR-0001 if
  concrete operational signal justifies them.
- The `LatestExecutionPerEntityCheck` query is
  partition-pruned by `recorded_at <= asOf`, so the
  scan cost is bounded by ADR-0031's
  `ResultsRetention` and ADR-0029's
  `MaxBytesScannedPerRun`. External monitors calling
  the method frequently incur a partition-pruned
  scan each time; if the call frequency becomes a
  cost concern, a future amendment could ship a
  materialized "latest execution per
  (entity, check_id)" view that refreshes
  periodically. Not committed at v1; reserved as a
  future enhancement.
- The engine-side-scheduler option (rejected) would
  reopen ADR-0007 §5 + §6 and require a substantial
  multi-replica coordination surface (leader
  election or distributed locks to avoid duplicate
  trigger emission). The amendment bar is
  correspondingly high; deferred indefinitely unless
  concrete operator demand surfaces (e.g., small
  deployments that don't want to provision an
  external scheduler at all).
