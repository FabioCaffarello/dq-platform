<!-- path: studies/decisions/2026-05-25-b1-3-scheduler-catchup-behavior.md -->

# B1-3 — Scheduler Catchup Behavior

## Context

Foundation 05 §"Run Identity and Idempotency" anticipated that
the platform processes both scheduled and manually-triggered
evaluations. The trigger-source enum committed by ADR-0002 CC6
already names the three sources: `scheduler`, `manual`,
`operator-rerun`. ADR-0007 §5 + §6 committed that the
scheduler is **external to the engine**: lifecycle operations
(create / update / delete / inspect triggers + orphan cleanup)
are attempted independently per trigger; an orphan is a
trigger present in the external scheduler but not in the
active manifest's rule set. ADR-0014 commits the HTTP
`/v1/trigger` endpoint as the engine's trigger-acceptance
surface; the engine itself runs no internal scheduler.

What B1-3 must commit, given that posture, is the **scheduling
contract** the platform expects external schedulers to honor:

1. **Cadence declaration.** Where does the per-rule cadence
   live? On the rule artefact, on a companion file, or in
   operator-level scheduler config?
2. **Catchup posture.** When the external scheduler was down
   and missed N windows, what should it do? Run all N, skip
   to the latest, or apply a configurable catchup horizon?
3. **Missed-window detection.** How does the operator know a
   scheduled trigger never fired?
4. **Manual trigger semantics.** Manual triggers can target
   any window; how do they coexist with scheduled triggers
   for the same window?
5. **Backfill posture.** When a new entity is onboarded, does
   the platform backfill historical windows automatically or
   require operator-driven manual triggers?

The trigger-acceptance surface itself is already committed:

- [ADR-0002](../../docs/adr/0002-run-identity-and-idempotency.md)
  CC1 defines `execution_id` as a hash over
  `(ruleset_version, engine_version, entity, window_start,
  window_end, trigger_source)`. Two triggers with different
  `trigger_source` for the same window produce different
  execution_ids; two triggers with the same `trigger_source`
  for the same window collide on execution_id, and ADR-0002
  CC2/CC5 commits the rerun/upsert/supersedes semantics.
- [ADR-0014](../../docs/adr/0014-trigger-handler-contract.md)
  commits the HTTP `/v1/trigger` endpoint with strict decoder,
  async runner dispatch, and `/healthz` + `/readyz`.
- [ADR-0029](../../docs/adr/0029-bigquery-cost-ceilings.md)
  commits `MaxConcurrentEvaluations` (via the semaphore the
  trigger handler acquires) and `MaxWindowDuration` (which
  rejects triggers whose window exceeds the per-env ceiling).

B1-3 must NOT reopen any of these. The study commits the
**contract** the external scheduler observes — what to send,
when, and what assumptions to make about platform behavior
under catchup / manual / backfill conditions.

The principles bearing on the decision are **P1** (rules
must remain declarative — cadence is a property of the
entity, not an engine implementation detail), **P3**
(ownership is explicit — operators own scheduler choice +
catchup tuning), and **R3** (do not revisit settled
architecture — ADR-0007's external-scheduler commitment is
preserved; ADR-0002 / ADR-0014 / ADR-0029's trigger surface
is preserved).

---

## Decision Drivers

- **DD-1 — The scheduler is external to the engine per
  ADR-0007.** B1-3 commits the contract the external
  scheduler honors, not an internal scheduler component.
  Building an internal scheduler would reopen ADR-0007 §5
  + §6 and add a substrate dependency the platform does not
  currently have. R3 raises the bar; the deferred choice is
  to keep the scheduler external.
- **DD-2 — When a rule has a regular cadence, it should
  be declared near the rule, not buried in operator
  config.** A rule that runs hourly is semantically
  different from a rule that runs daily. Rule authors +
  CODEOWNERS reviewers (per ADR-0015) should see the
  cadence at PR-review time. The natural surface is the
  rule artefact itself. v1 commits an **optional**
  `schedule` field on the rule that external schedulers
  READ — the engine does not enforce or execute the
  cadence. Rules without a scheduled cadence (purely
  HTTP-triggered) omit the field.
- **DD-3 — Catchup is operator-tunable, not platform-fixed.**
  Some entities need every missed window evaluated (catchup-
  all). Others tolerate skipping (latest-only). The platform
  commits a **catchup horizon** guidance value per env that
  external schedulers respect — but does not enforce it in
  the engine. Operators pick the catchup policy that fits
  their scheduler tool and entity's cost profile.
- **DD-4 — Manual triggers produce distinct execution_ids
  via the trigger_source field.** ADR-0002 CC1 already
  commits this: manual + scheduler for the same window
  produce different execution_ids and coexist as separate
  executions in `dq_executions`. B1-3 confirms the
  pattern, names the operational expectation, and points
  operators at `operator-rerun` (ADR-0002 CC5) for the
  rerun-the-scheduled-run case.
- **DD-5 — Missed-window detection is the scheduler's job;
  the platform exposes a query surface.** Each external
  scheduler tool has its own miss-detection mechanism
  (cron history, queue depth, etc.). The platform's role
  is to expose a query that returns the most recent
  execution per (entity, check_id) so external monitors
  can detect gaps. The query lands as a new reader method
  via the future scheduler-consumer slice (not in this
  ADR; design-only).
- **DD-6 — Backfill is operator-driven at v1.** When a new
  entity is onboarded, no automatic historical backfill
  ships. Operators backfill via `/v1/trigger` with
  `trigger_source: manual` and a window range covering the
  desired history. Automatic backfill is reserved as a
  future enhancement.
- **DD-7 — The study commits design only; implementation
  defers to the first scheduler-consumer slice.** Matches
  the [ADR-0030](../../docs/adr/0030-manifest-cryptographic-posture.md)
  and [ADR-0032](../../docs/adr/0032-baseline-strategy.md)
  precedents. Shipping the schema field, the EnvConfig
  field, and the reader method without a consuming
  external scheduler in the platform's scope would be
  designing-for-hypothetical-future-requirements.

---

## Considered Options

### Option 1 — External scheduler + advisory `schedule` field on v2 rules + per-env catchup horizon guidance (recommended)

The platform's role:

- **Acceptance:** continues per ADR-0014 — `/v1/trigger` accepts
  triggers with any `trigger_source`, any window. No new
  acceptance behavior committed here.
- **Contract:** v2 rule schema gains an optional `schedule`
  field carrying an advisory cadence declaration (cron
  expression or duration literal). External schedulers READ
  this at manifest-publish time and provision their own
  triggers; the engine does not interpret or enforce it.
- **Catchup horizon guidance:** the platform exposes a per-env
  `CatchupHorizon` value documenting "don't emit triggers
  older than X" — but does not enforce it; external
  schedulers respect it as a convention.
- **Missed-window query:** a new reader method returns the
  most-recent execution per (entity, check_id) so external
  monitors can detect gaps.

**Strengths.** Honors ADR-0007's external-scheduler posture;
keeps cadence declarative near the rule (P1); avoids
reopening the trigger-acceptance surface (ADR-0014);
operators retain choice of scheduler tooling; backfill is
explicit and operator-driven (no surprise behavior).

**Trade-offs.** Adds an optional field to the v2 rule
schema (additive per ADR-0001's compatibility contract;
no major bump). The advisory nature of the `schedule`
field means a rule may be declared "hourly" but the
external scheduler may emit it daily — the platform
cannot detect this mismatch automatically. The
missed-window query surface is a new read API; modest
implementation cost when the consumer slice ships.

### Option 2 — Internal engine-side scheduler

Add a scheduler component inside the engine binary that
reads the `schedule` field per rule from the manifest and
emits internal triggers on the configured cadence. The
engine becomes its own scheduler.

**Strengths.** Operators do not need to provision an
external scheduler at all. Catchup behavior is fully
under the platform's control.

**Trade-offs.** Reopens ADR-0007 §5 + §6 (external
scheduler is part of the committed architecture). Pulls
the scheduler's reliability surface into the engine —
a missed schedule because the engine was down is now a
platform-correctness concern, not an operator-scheduler
concern. Multi-replica engine deployments need leader
election or distributed locks to avoid duplicate
trigger emission. The complexity is real and the
operational benefit is hypothetical — operators of
internal data-quality platforms already have scheduling
tools they prefer. Rejected.

### Option 3 — Pure HTTP-triggered, no platform-level cadence concept

Drop the cadence concept entirely. Rules carry no
schedule declaration; operators manage cadence purely
in external scheduler config, separately from the
rule artefact.

**Strengths.** Maximum simplicity. The platform's
contract is the smallest possible: accept triggers,
record executions.

**Trade-offs.** Loses the declarative-near-the-rule
property (P1). Rule authors who set up a new entity
don't see what cadence the operator chose; cadence
changes happen outside the CODEOWNERS PR-review path
(ADR-0015), which weakens the platform's ownership
posture (P3). The advisory `schedule` field in Option 1
is cheap to add (additive schema extension) and
preserves the declarative pattern — Option 3's
operational simplicity does not justify the loss.
Rejected.

---

## Recommendation

**Option 1.** External scheduler + advisory `schedule` field
on v2 rules + per-env `CatchupHorizon` guidance + missed-window
query surface. Design only; implementation defers to the first
scheduler-consumer slice.

### v2 rule schema extension — `schedule` field

The v2 rule schema gains an optional top-level `schedule`
field declaring the advisory cadence:

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
  HTTP-triggered (operator emits triggers manually or via
  an external system not driven by the rule's declared
  cadence).
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
  supported at v1 — the duration-literal form covers the
  same cases (`1d` ≡ `@daily`, `1h` ≡ `@hourly`)
  unambiguously. The lint binary will validate the
  grammar when the scheduler-consumer slice lands; the
  engine does not interpret the field at any phase.
- **Mode-independent.** Both set-mode and record-mode
  rules can carry a `schedule`. For record-mode rules,
  the schedule typically aligns with the
  `source.kafka.window.duration` — but the alignment is
  the author's responsibility, not platform-enforced.

This extension is **additive** per ADR-0001's
compatibility contract (a new optional field within v2).
No schema-version bump.

### Per-env `CatchupHorizon` (advisory)

A new typed field on `EnvConfig` exposes a per-env
catchup-horizon value:

```
type EnvConfig struct {
    // ... existing fields ...

    // CatchupHorizon documents the maximum age of a
    // scheduled trigger the external scheduler should
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
  E.g., scheduler down for 18 hours in prod (24h
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

Per ADR-0002 CC1, manual triggers produce
**distinct execution_ids** from scheduled triggers for
the same window (different `trigger_source` input to
the hash). They coexist in `dq_executions` as separate
executions; neither supersedes the other.

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
  manual when the goal is a parallel/ad-hoc
  evaluation.

The runbook seed for this distinction ships when the
first scheduler-consumer slice lands (B2 follow-up).

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
  window) produce multiple `dq_executions` rows;
  no upsert collision because each carries a unique
  `(window_start, window_end)` tuple.
- **Future enhancement:** an "auto-backfill on
  manifest publish" feature could emit manual triggers
  for the last N windows when a rule is added. Reserved
  as a B2 follow-up; not committed here.

### Why this does NOT reopen ADR-0007

ADR-0007 §5 + §6 commits the external-scheduler
posture: lifecycle operations are per-trigger
best-effort; orphans are external triggers not in the
manifest. B1-3 confirms the posture and commits the
contract external schedulers honor:

- The advisory `schedule` field is READ by external
  schedulers, not by the engine.
- The `CatchupHorizon` is documentation for external
  schedulers, not an engine-enforced ceiling.
- The missed-window query is a read surface for
  external monitors, not a scheduling primitive.

The engine code path (loader, runner, trigger handler)
is unchanged. ADR-0007 stays accepted without
amendment.

### Why this does NOT reopen ADR-0014

ADR-0014 commits the `/v1/trigger` endpoint with
strict decoder + async runner dispatch. B1-3 does not
add new endpoint behavior. Manual triggers and catchup
triggers go through the existing endpoint with the
existing payload shape. The `LatestExecutionPerEntityCheck`
read method is a Store-level surface, not a new
endpoint.

### Why this does NOT commit specific scheduler tooling

The platform supports any external scheduler that can
emit HTTP POST `/v1/trigger`. Kubernetes CronJob is the
natural fit given the platform's substrate posture
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
   **design commitments**; the implementation lands under
   the B2 follow-up that ships the first scheduler-consumer
   slice. Matches the ADR-0030 / ADR-0032 precedents —
   design committed; implementation deferred.

2. **The v2 rule schema gains an additive optional
   `schedule` field.** When the first scheduler-consumer
   slice ships, the v2 schema and its rules-side mirror
   gain the optional field (cron expression or duration
   literal); the lint binary validates the grammar; the
   engine does not interpret it. ADR-0001's
   additive-within-major contract is honored; no
   schema-version bump.

3. **`EnvConfig.SchedulerCatchupHorizon` ships as advisory
   guidance.** When the first scheduler-consumer slice
   ships, the new field lands with per-env values
   (1h / 6h / 24h per local / qa / prod). The engine does
   not enforce it; external schedulers respect it as
   documented convention.

4. **`Store.LatestExecutionPerEntityCheck` ships as a
   read surface.** When the first scheduler-consumer slice
   ships, the method lands on the `engine/internal/results`
   Reader interface alongside `QueryCurrentExecution`. The
   BigQueryStore implementation issues a partition-pruned
   query (partition expiration per ADR-0031 bounds the
   scan).

5. **Manual triggers continue to produce distinct
   execution_ids.** ADR-0002 CC1 is unchanged. The
   operational expectation is documented: manual ≠
   replacement of scheduled; operator-rerun is the
   replacement mechanism.

6. **Backfill is operator-driven at v1.** Multiple manual
   triggers per historical window. No upsert collisions
   because each window is distinct. Auto-backfill on
   manifest publish is reserved as a B2 follow-up.

7. **Catchup is the external scheduler's responsibility.**
   The platform commits the `CatchupHorizon` guidance but
   does not enforce it; the ADR-0029 `MaxConcurrentEvaluations`
   semaphore in the trigger handler is the cost-discipline
   gate during catchup bursts.

8. **B2 follow-up: first scheduler-consumer slice.** A new
   B2 row registers the first slice that consumes B1-3's
   design surface. The slice ships:
   - The optional `schedule` field on the v2 rule schema
     (canonical + mirror; lint-validated grammar).
   - The `EnvConfig.SchedulerCatchupHorizon` field with
     per-env values.
   - The `LatestExecutionPerEntityCheck` reader method on
     `engine/internal/results.Store`.
   - A reference external scheduler (likely a k8s CronJob
     manifest under `deploy/overlays/`) that reads the
     `schedule` field and emits triggers.
   The B2 row is added at close-step assignment of a
   number.

9. **B2 follow-up: manual-vs-operator-rerun runbook seed.**
   The runbook clarifying when to use `trigger_source:
   manual` vs `trigger_source: operator-rerun` ships when
   the first scheduler-consumer slice lands. A B2 row
   registers this seed.

10. **The platform's P1 + P3 commitments for scheduling
    are now explicit.** P1 (declarative): cadence lives
    on the rule artefact, not in engine code. P3
    (ownership): operators own the scheduler choice and
    the per-env catchup-horizon tuning; rule authors +
    CODEOWNERS reviewers own the cadence declaration at
    PR-review time.

---

## Open Questions

None blocking.

Three deferred items surfaced during the design phase and
are explicitly **out-of-scope for current cycle**:

- **OQ-1: Auto-backfill on manifest publish.** When a new
  rule is added (rule was not in the prior manifest), the
  platform could automatically emit manual triggers for
  the last N windows so the rule has immediate forensic
  history. Useful for the baseline-strategy framework from
  ADR-0032 (a new baselined rule with no history
  immediately fires `degraded`; auto-backfill would seed
  the history). Deferred until concrete operational signal
  shows the manual-backfill flow is too cumbersome.

- **OQ-2: Per-rule catchup-policy override.** A future
  amendment could ship `params.catchup_policy: "all" |
  "latest_only" | "skip_if_older_than: <duration>"` on the
  rule artefact, so rule authors override the per-env
  default. The override is operationally useful but adds
  schema surface; deferred until concrete signal justifies
  it.

- **OQ-3: Engine-side scheduler.** Option 2 above. Reserved
  as a future amendment if operator demand surfaces (e.g.,
  small deployments that don't want to provision external
  scheduling). The amendment would reopen ADR-0007 §5 + §6;
  the bar is correspondingly high.

---

## Promotion target

`docs/adr/0033-scheduler-catchup-behavior.md` — ships the
design commitments only (no engine code change): the
optional `schedule` field shape on v2 rules, the
`EnvConfig.SchedulerCatchupHorizon` advisory values, the
`Store.LatestExecutionPerEntityCheck` reader method
signature, the manual-vs-operator-rerun semantics, the
operator-driven backfill posture, and the two B2 follow-ups
(first scheduler-consumer slice; manual-vs-operator-rerun
runbook seed).
