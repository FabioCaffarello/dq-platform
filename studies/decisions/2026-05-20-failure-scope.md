<!-- path: studies/decisions/2026-05-20-failure-scope.md -->

# B0-4 — Failure Scope

## Metadata

- B0 reference: B0-4 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 6).
- Last updated: 2026-05-20.
- Upstream resolved: B0-2
  ([`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md)),
  B0-3
  ([`2026-05-20-result-write-model.md`](./2026-05-20-result-write-model.md)).
- Downstream open: B0-6 (alert routing contract — depends on B0-4),
  B0-7 (loader / scheduler / retry failure semantics — depends on
  B0-1, B0-5, B0-4).
- Promotion target: see final section.

---

## Context

When something goes wrong during a run, the platform must answer
two questions precisely: **at what level did it go wrong, and what
is the impact?** Without a single explicit policy, every run that
encounters a problem becomes its own ad-hoc judgement call —
dashboards become inconsistent, alerts fire to the wrong people,
incident responders can't tell whether a `failed` execution means
"the data is bad" or "the platform is bad."

Foundation doc
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
§"Failure Scope" sketches a four-level hierarchy:

- **Check failure** — the rule found a problem in the data (the
  happy path of negative results).
- **Check error** — the check could not be evaluated due to an
  operational problem (query error, missing source, quota
  exhaustion, timeout).
- **Entity error** — entity-level problems prevented evaluation
  (source table missing, manifest corruption, schedule
  misconfiguration).
- **Run error** — the whole run is broken (engine can't load the
  manifest, can't connect, hit a global resource limit).

Foundation 05 explicitly defers the policy:

> The exact boundary between "check error" and "entity error" —
> that is, whether one bad check fails the whole entity — is the
> B0 question. The framing above describes the surface; the
> resolution picks the policy.

And:

> Other checks in the same run: the B0 decision must specify
> whether they continue, abort, or are conditional on entity-level
> policy.

B0-4 — as recorded in the decision log:

> When one check errors operationally, does the entity error,
> degrade, or partially complete?

This study locks the policy across five sub-decisions:

1. **Check-result semantics**: exact meaning of each value in the
   `result` enum committed by B0-3 CC7 (`pass` / `fail` /
   `degraded` / `error`).
2. **Execution-status mapping**: how the multiset of check results
   for one execution maps to `dq_executions.status` (committed by
   B0-3 CC6: `running` / `success` / `failed` / `error` / aborted
   via B0-7).
3. **Promotion rule**: when (and only when) does a check-level
   error promote to entity-level `error`.
4. **Continuation rule**: when one check errors mid-execution, do
   other checks continue, abort, or follow entity-policy.
5. **Alerting category split**: which failures route as
   data-quality vs. operational alerts. (B0-6 will implement
   the routing; B0-4 commits the category mapping.)

What this study does **not** decide:

- The shape of `_owners.yaml` or alert dedup logic — B0-6.
- The conditions producing `status = aborted` (cost ceiling,
  manifest load failure, global resource limit) — B0-7.
- Per-environment retry counts, backoff schedules, or sample
  retention — B1.
- The runbook itself (operator response procedures) — Wave 3,
  informed by this policy.

The decision matters because every consumer of `dq_executions` and
`dq_check_results` — dashboards, alerting, incident triage,
downstream analytics — keys off the status and result enums. A
fuzzy policy means consumers each invent their own interpretation;
the platform's promise to "make data quality posture visible,
owned, and operationally actionable" (foundation doc
[`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md))
collapses into "visible to whoever can decode our specific
project."

---

## Decision Drivers

The decision must satisfy the following, in priority order.

1. **D1. Determinism (P2).** The mapping from check results to
   execution status must be a pure function — no hidden
   thresholds, no environment-dependent behavior, no operator
   discretion at evaluation time. The same multiset of check
   results must always produce the same execution status.

2. **D2. Visible failure over silent degradation** (foundation
   doc
   [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
   §"Operating Posture", imperative #3). Every failure must
   surface in the appropriate state; nothing is masked by
   another's success. A single check error must not be hidden
   inside a `success` execution status.

3. **D3. Explicit ownership** (P3). Every failure category must
   route to a clear owner — data team for data-quality issues,
   platform team for operational issues. The status enum's job
   is to make the routing decision unambiguous.

4. **D4. Information preservation.** When 9 checks succeed and 1
   errors, the execution status must convey "9 succeeded, 1
   errored" — not "everything is broken" (which loses the 9
   passes) and not "everything is fine" (which loses the 1
   error).

5. **D5. Logical-execution identity (B0-2).** Failure scope
   reasons at the logical-execution level (per `execution_id`),
   not per attempt. A scheduler retry of an `error` execution
   can produce `success` on the next attempt; the canonical view
   per B0-2 I4 returns the latest attempt's status.

6. **D6. Storage contract (B0-3).** This study uses only the
   enum values B0-3 already committed (`status`: `running` /
   `success` / `failed` / `error`; `result`: `pass` / `fail` /
   `degraded` / `error`). No new values are introduced by B0-4.

7. **D7. Cost (P4).** The continuation rule must not waste
   evaluation cost unboundedly. Aborting on the first check error
   saves cost; continuing maximizes information; the choice must
   acknowledge the trade-off explicitly.

8. **D8. Scope hygiene.** Alerting routing details are B0-6's
   territory; engine-halt conditions (cost ceiling exceeded,
   manifest load failure) are B0-7's. B0-4 commits the failure
   *category mapping*; downstream B0s implement the response
   shape consistent with the category.

---

## Considered Options

The decision has two coupled sub-policies: the **promotion rule**
(when does a check-level error become an entity-level error) and
the **continuation rule** (when one check errors, do others
continue). Each option below states both.

For all options, the check-result `result` enum semantics are
held constant (defined in the Recommendation; identical across
options):
- `pass` — check evaluated; data met expectations.
- `fail` — check evaluated; data did not meet expectations.
- `degraded` — check evaluated; data within a warning band.
- `error` — check could not be evaluated due to an operational
  problem.

The differences between options are about how the *multiset of
results* maps to `dq_executions.status`, and what the engine does
mid-execution.

### Option A — Strict promotion, abort on first error

**Promotion**: any check producing `error` immediately promotes
the entity to `status = error`.

**Continuation**: as soon as one check errors, the engine
abandons remaining checks for that entity; no more
`dq_check_results` rows are written for that execution.

```
result multiset                        status
{pass, pass, pass}                     success
{pass, fail, pass}                     failed
{pass, error} → engine stops           error  (only 2 check rows
                                              of N expected)
```

**Trade-offs.**

- Pro: simplest mental model — any operational problem fails the
  whole entity.
- Pro: D7 (cost) — minimum wasted evaluation when the entity is
  broken.
- Con: violates D4 (information preservation) hard — N-1 checks
  are never evaluated; the run's data-quality picture is missing
  by construction.
- Con: violates D2 partially — `error` masks the data-quality
  status of unevaluated checks (we don't know if they would have
  passed, failed, or also errored).
- Con: a single flaky check (e.g., a transient query timeout)
  marks the entity as broken even though the underlying entity
  is fine. Noisy.
- Con: difficult to distinguish "the entity is genuinely broken"
  (source missing) from "one check is buggy". Both produce
  identical execution rows.
- Con: abandoned checks produce no `dq_check_results` row (per
  B0-3 CC1, the table is keyed by `(execution_id, attempt_id, check_id)`
  and only attempted checks produce rows). Consumers cannot
  distinguish "check was not run because of abort" from "check
  was never defined" without consulting the manifest. The
  information gap is silent.

### Option B — Threshold promotion (all-checks-error), always continue

**Promotion**: the entity is promoted to `status = error` if and
only if **every check produced `error`** (or a pre-check
entity-level problem prevented any check from being attempted —
see Recommendation for the pre-check carve-out). Any
configuration where at least one check evaluated successfully
(produced `pass`, `fail`, or `degraded`) keeps the execution at
`failed` or `success` per the result mix.

**Continuation**: every check is evaluated regardless of others'
results. The engine writes all per-check rows before writing the
terminal execution row.

```
result multiset                        status
{pass, pass, pass}                     success
{pass, fail, pass}                     failed
{pass, error}                          failed   (mixed — some
                                                evaluation
                                                succeeded)
{fail, error}                          failed
{error, error, error}                  error    (no evaluation
                                                succeeded)
(pre-check entity problem detected)    error    (no check rows)
```

**Trade-offs.**

- Pro: D2 (visible failure) — every error surfaces. A single
  errored check produces `status = failed` (errors are a kind of
  failure), which routes operationally; the data-quality picture
  for the other checks is preserved.
- Pro: D3 — `failed` with mixed results routes both data-quality
  alerts (for the `fail`/`degraded` checks) and operational
  alerts (for the `error` checks); B0-6 has the information to
  fan out correctly.
- Pro: D4 — every check's result is recorded; no information is
  lost.
- Pro: D1 — the rule is a deterministic function of the result
  multiset.
- Pro: distinguishes "entity is genuinely broken" (`status = error`
  — every check failed to evaluate, indicating an entity-level
  problem) from "some operational hiccups during otherwise fine
  evaluation" (`status = failed` with mixed errors).
- Con: D7 — wastes evaluation cost when the entity is actually
  broken (all N checks try to query a missing source). Bounded:
  N queries against a missing table fail fast (BigQuery returns
  "table not found" quickly); the cost is N × ~1 second, not
  unbounded.
- Con: requires consumers to learn that `status = failed` with
  errors in the results is an operationally-relevant failure
  (not just data quality). Documentation burden, mitigated by
  B0-6's routing rules.

### Option C — Per-check independent, always continue

**Promotion**: check errors **never** promote to entity error.
The entity status is computed solely from the data-quality
results; check errors are recorded as `error` rows in
`dq_check_results` and routed as operational alerts, but the
execution status reflects only the data-quality verdicts.

**Continuation**: every check is evaluated regardless of others'
results.

```
result multiset                        status
{pass, pass, pass}                     success
{pass, fail, pass}                     failed
{pass, error}                          success  (the pass survives;
                                                error is per-check)
{fail, error}                          failed
{error, error, error}                  success  (no data-quality
                                                verdict; no failure)
```

**Trade-offs.**

- Pro: D4 maximally preserved — data-quality and operational
  signals are fully decoupled at the execution level.
- Pro: D7 same as Option B.
- Con: violates D2 hard — `{error, error, error}` becomes
  `success`, which is the worst possible failure mode (a
  broken entity reports as successful). A consumer querying
  "did this entity pass?" gets a false positive.
- Con: violates D3 — execution status no longer conveys whether
  operational attention is needed; alert routing must inspect
  the result multiset rather than read the status.
- Con: D5 — incident responders looking at `dq_executions` see
  `success` for a broken entity; the canonical view becomes
  misleading rather than informative.

Reject on D2/D3/D5.

### Option D — Entity-declared policy

**Promotion**: each entity declares its own promotion rule in
its rule YAML. Possible declared rules include "strict" (Option
A's promotion), "threshold" (Option B's promotion), "per-check"
(Option C's promotion), or a custom predicate.

**Continuation**: same as the declared rule (or "always
continue" if not declared).

**Trade-offs.**

- Pro: flexibility — domain teams can pick the policy that
  matches their entity's nature.
- Con: violates D1 (determinism) at the platform level — two
  executions of two different entities with the same result
  multiset can produce different execution status; consumers
  reading dashboards across entities see inconsistent semantics.
- Con: violates D3 — alert routing per entity becomes
  unpredictable without inspecting each entity's declared rule.
- Con: violates platform principle P1 indirectly — adds a new
  expressive surface to the DSL (the promotion-rule declaration)
  without compiler support for verifying it. New axis of "what
  does this rule actually mean for failure scope" each domain
  team must reason about.
- Con: dashboards across entities cannot use a uniform query
  pattern (`WHERE status = 'failed'`) — each entity may interpret
  `failed` differently.

Reject on D1, D3, and the indirect P1 cost.

---

## Recommendation

Adopt **Option B** — threshold promotion (all-checks-error
promotes to entity-level `error`; any successful evaluation keeps
the execution at `failed` or `success` per the result mix) +
always-continue at the check level.

The recommendation is grounded in:

- foundation doc
  [`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
  §"Failure Scope" — the four-level hierarchy this study locks
  policy for; §"Operating Posture" imperative #3 (visible
  failure);
- foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"Layer 5 — Alerting" — the data-quality vs. operational alert
  split this study commits the category boundary for;
- prior decision
  [`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md)
  (B0-2) — I3 alert deduplication and D5 logical-execution
  reasoning;
- prior decision
  [`2026-05-20-result-write-model.md`](./2026-05-20-result-write-model.md)
  (B0-3) — CC6 status enum and CC7 result enum are used as-is;
  this study resolves B0-3 CC6's "B0-4's decision" note about the
  `failed`-vs-`error` boundary.

The specific commitments beyond what those documents state are
**new contribution proposed here, requires review**:

1. **Check `result` enum semantics** locked: `pass` (evaluated,
   met expectations), `fail` (evaluated, did not meet),
   `degraded` (evaluated, within warning band), `error` (could
   not be evaluated). **New contribution proposed here,
   requires review.**

2. **Execution-status mapping** is a pure function of the result
   multiset (deterministic per D1). The mapping is:
   - all `pass` → `success`
   - any `fail` or `degraded`, **and** at least one check
     evaluated successfully (produced `pass`/`fail`/`degraded`)
     → `failed`
   - all checks produced `error` (zero successful evaluations)
     → `error`
   - pre-check entity-level problem detected (no check rows
     written) → `error`
   - global engine halt → `aborted` (B0-7's domain)

   **New contribution proposed here, requires review.**

3. **Promotion rule (threshold form)**: check-level `error`
   promotes to entity-level `error` **only when every check
   produced `error`** (or no check ran due to a pre-check
   entity-level problem). Mixed-result executions (some `error`,
   some non-`error`) stay at `failed`. **New contribution
   proposed here, requires review.**

4. **Continuation rule**: every check is evaluated regardless of
   others' results. The engine writes all per-check rows before
   writing the terminal execution row. Engine-level halts
   (manifest load failure, cost ceiling, global resource limit)
   are owned by B0-7 and produce `status = aborted` rather than
   `error`; B0-4 commits only that no check-level error abandons
   sibling checks. **New contribution proposed here, requires
   review.**

5. **Alerting-category split** (commits the mapping; B0-6
   implements the routing):
   - check `pass` → no alert.
   - check `fail` / `degraded` → **data-quality alert** (routed
     per `_owners.yaml` semantics, B0-6).
   - check `error` → **operational alert** (routed to platform
     team plus the entity's declared owner).
   - execution `failed` (mixed) → data-quality alerts fan out
     per check, plus operational alerts per errored check.
   - execution `error` → **operational** alert (category only;
     severity, routing, and dedup are B0-6's call per OQ-4).
   - execution `aborted` → operational alert (category only;
     severity, routing, and dedup are B0-6's call per OQ-4).

   **New contribution proposed here, requires review.**

---

## Consequences

Adopting this recommendation commits the platform to the following.

**CC1. Check `result` semantics are deterministic.** A check
produces exactly one value from `pass` / `fail` / `degraded` /
`error`. The mapping:
- `pass`: the check's query executed and the data met the rule's
  pass condition.
- `fail`: the check's query executed and the data did not meet
  the rule's pass condition (above the warn threshold if one is
  defined).
- `degraded`: the check's query executed and the data fell into
  a warning band — between pass and fail. (The warning band is a
  per-check rule construct; declaring one is optional, governed
  by the DSL.)
- `error`: the check's query did not execute successfully —
  query compilation error, missing source table or column, quota
  exhaustion, exceeded retry budget for transient errors, or
  timeout exceeding the check's evaluation budget.

A check that completes evaluation with no rows examined (e.g.,
the window is empty and the rule says "no data is fine") is
`pass`; a check completing with no rows examined where the rule
says "data must be present" is `fail`. The empty-window
interpretation is governed by a DSL construct (to be defined;
see OQ-2) that lets a rule declare whether absence of data is
a pass or a fail; the specific identifier and shape of that
construct are out-of-scope here.

**CC2. Execution status is a pure function of the result
multiset.** Given the multiset `R` of check results for one
logical execution, the status is determined by the following
**mutually-exclusive** decision procedure, applied in order:

1. **Global engine halt** at any point during evaluation →
   `status = aborted` (B0-7 owns the exact halt conditions).
   This branch dominates all others if it applies.
2. **Pre-check entity-level problem** detected (no check rows
   written, see CC5) → `status = error`.
3. **R ≠ ∅** and every element of `R` is `pass` →
   `status = success`.
4. **R ≠ ∅** and every element of `R` is `error` →
   `status = error`.
5. **Otherwise** (R ≠ ∅ with any mixed-result combination) →
   `status = failed`.

The five branches are mutually exclusive by construction; for
any input the procedure halts at the first matching branch. The
mapping is a single pure function; no operator discretion. The
implementation is Wave 3 — a small enum-reducer inside the
engine's reporter.

**CC3. Promotion rule: all-checks-error.** The only path from
check-level errors to execution-level `error` (other than a
pre-check entity-level problem) is **every check produced
`error`**. One errored check among ten successful ones produces
`status = failed`, not `error`. The rule is deliberately
restrictive: `status = error` means "we could not evaluate this
entity at all", a strong claim with operational consequences.

The threshold is deliberately binary, not proportional. A
multiset like `{pass, error, error, ..., error}` (nine errors
plus one pass) produces `status = failed`, not `error` — even
though "9 of 10 errored" is intuitively severe. The operational
signal is **not lost**: each errored check produces an
operational alert per CC7's category mapping, and the
check-results table preserves every error row. A consumer
wanting "any operational signal triggered this execution"
queries the `result` column on `dq_check_results`, not just the
`status` column on `dq_executions`. `status = failed` is the
catch-all for mixed evaluation; `status = error` is reserved
for the "no evaluation succeeded" strong-claim outcome.
Operators alerting on "any execution `status = error` OR any
check `result = error`" get full visibility regardless of the
mix.

**CC4. Continuation rule: always continue at the check level.**
The engine does not abandon sibling checks when one check errors
mid-execution. Every check listed in the manifest for the entity
is attempted; every attempt's result is recorded in
`dq_check_results` (per B0-3 CC1). The terminal `dq_executions`
row is written after every check's evaluation has completed (or
errored).

Engine-level halts that prevent the engine from completing the
evaluation (cost ceiling exceeded, manifest load failure during
refresh, container OOM, etc.) are **not** governed by this rule
— they are B0-7's domain and produce `status = aborted`, which
is distinct from `error`. B0-4 commits only that a check-level
error never triggers an abort.

The wasted-cost concern is O(N) in per-check fast-fail latency.
A missing source table causes every check to error in BigQuery's
"table not found" fast-fail path (typically sub-second per check
in steady state). The platform does not impose a hard cap on N
per entity at this policy layer; if a future entity exceeds
practical operational tolerance for the wasted-cost trade-off
(very-large-N entities hitting a missing source), an explicit
cap is a governance / B1 concern (rules-workspace policy or
per-environment ceiling) rather than a B0-4 commitment. B0-4
commits only the always-continue policy and accepts O(N) cost
as the price of D4 information preservation.

**CC5. Pre-check entity-level problems produce `status = error`
with no check rows.** When the engine detects an entity-level
problem before any check has been attempted — the kind of
problem that would make every individual check error — the
execution row is written directly with `status = error` and no
`dq_check_results` rows are produced for that execution. This
state-transition path is the only way `status = error` is
reached without all checks producing `error` results.

The **mechanism** — when in the engine lifecycle these
validations run (load-time, plan-time, lazy on first check),
which specific validations are performed (manifest contract per
B0-1, source-table existence, partition column presence,
others), and how they relate to B0-1's load-time contract
checks — is **out-of-scope for current cycle** (OQ-1 above);
Wave 3 / B0-7 implementation. B0-4 commits only the
state-transition outcome: when an entity-level problem is
detected before any check produces a result, the execution row
is `status = error` with empty result multiset.

**CC6. `result` enum extension.** Adding a new check `result`
value (e.g., `warn`, `inconclusive`) is additive and does not
break existing rows; CC2 above must be updated in the same ADR
that adds the value, specifying how the new value participates
in the status mapping. Removing or renaming an existing
`result` value is breaking and requires a future ADR.

The same policy applies to the `status` enum on `dq_executions`
(already committed by B0-3 CC6); CC2 above is the authoritative
mapping until extended.

**CC7. Alerting category boundaries are committed.** The
mapping from `(result, status)` to alert category:

| Trigger                     | Category               |
|-----------------------------|------------------------|
| check `pass`                | no alert               |
| check `fail` / `degraded`   | data quality           |
| check `error`               | operational            |
| execution `success`         | no alert               |
| execution `failed`          | mixed (per-check)      |
| execution `error`           | operational            |
| execution `aborted`         | operational            |

Within each category, routing targets (which team, which owner
in `_owners.yaml`), severity within the category (info /
warning / critical), and deduplication windows are **B0-6's
call** (per OQ-4). B0-4 commits only the category boundary —
which is non-negotiable: B0-6 cannot route a check `error` as
a data-quality alert, or vice versa. The "mixed (per-check)"
category on execution `failed` is shorthand for "per-check
fan-out": each check's `result` value produces its own alert
per the rows above, and the execution-level row does not
trigger a separate alert on its own.

**CC8. Idempotency interaction with B0-2.** A scheduler retry of
an execution that errored produces a new attempt under the same
`execution_id` (B0-2 CC3). If the retry's result multiset
produces a different status (e.g., `error` → `success` because
the source table came back), the canonical view per B0-2 I4
returns the latest attempt's status. Alert deduplication per
B0-2 I3 applies: a single user-visible alert per failing check
across N retries. CC8 commits that failure scope is recomputed
per attempt, with the canonical projection winning.

**CC9. Operator-rerun interaction.** An operator rerun produces
a **new** `execution_id` (B0-2 CC5) with `supersedes_execution_id`
pointing at the original. The new execution's status is computed
independently of the original — a rerun can produce `success`
where the original produced `error`; the audit link makes the
relationship visible without merging the two status outcomes.

**CC10. No new enum values introduced.** This study uses only
the `result` values (`pass` / `fail` / `degraded` / `error`) and
the `status` values (`running` / `success` / `failed` / `error` /
`aborted`) committed by B0-3. The contribution here is the
*mapping*, not new vocabulary.

**CC11. Operator-response documentation is downstream.** The
B0-4 row in the decision log says "Failure-semantics ADR plus
runbook". The policy committed in this study informs Wave 3
operator-response documentation, mapping each `(status, result)`
combination to documented operator actions (escalation paths,
expected investigation steps, common root causes). Whether
that documentation takes the form of a runbook, an operations
doc, a wiki page, or another shape is **Wave 3's call** —
B0-4 does not foreclose the format. The artifact is not
authored here.

**CC12. The mapping is a public contract.** Dashboards,
alerting consumers, downstream analytics, and incident-response
tooling all key off the status-and-result enums and the mapping
in CC2. Once published, changes to CC2 (the mapping function)
are breaking and require a future ADR with a migration path and
compatibility window — analogous to the schema migration
protocol in B0-1.

---

## Open Questions

- **OQ-1. Exact set of pre-check validations.** Which validations
  run at plan creation (CC5) — manifest contract checks per B0-1,
  source-table existence, partition column presence, others — is
  **out-of-scope for current cycle**. The set is Wave 3
  implementation; this study commits only that some pre-check
  validation exists and that its failure produces `status = error`.

- **OQ-2. Empty-window sentinel for the empty-result case.** The
  DSL construct that lets a rule declare "no data is fine" vs.
  "data must be present" (referenced in CC1's empty-window
  paragraph) — including its identifier, shape, and default
  behavior — is **out-of-scope for current cycle**. It is a DSL
  detail governed by the rules workspace and the schema (B0-1
  surface).

- **OQ-3. Warning-band semantics for `degraded`.** The precise
  threshold mechanism for `degraded` (single threshold, range,
  per-check warn fraction, etc.) is **out-of-scope for current
  cycle** — DSL detail. CC1 commits that `degraded` exists as a
  result value; the DSL governs how a rule declares it.

- **OQ-4. Alert severity beyond category.** Within the
  data-quality category, fine-grained severity (info / warning /
  critical) and the mapping from check declarations to severity
  is **out-of-scope for current cycle** — B0-6 owns the
  per-check severity mapping. CC7 commits only the category
  boundary; severity within a category is downstream.

- **OQ-5. Retry budget exhaustion classification.** When a
  check's retry budget is exhausted (per foundation 05
  §"Retry Semantics" — typically two retries with exponential
  backoff), the result is `error` (per CC1). Whether the
  per-check retry mechanism itself is implemented in the runner
  or coordinated globally is **out-of-scope for current cycle**
  — B0-7 owns retry semantics. CC1 commits only the classification
  outcome.

- **OQ-6. Cross-entity correlation in alerting.** When N
  entities error in the same window (e.g., a shared upstream
  source goes down), whether B0-6's alert routing collapses them
  into a single incident or fans them out is **out-of-scope for
  current cycle** — B0-6 detail. CC7 commits per-entity routing;
  cross-entity collapsing is B0-6's call.

No open question in this list blocks the failure-scope policy
itself. All items above are parameters, downstream routing
details, or DSL refinements on top of the locked mapping.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0004-failure-scope.md

The `0004` is provisional and assigned at promotion time. If the
Wave 3 ADR numbering convention orders by promotion date rather
than by B0 sequence, the number changes; the slug
(`failure-scope`) does not. This follows the same convention
adopted in
[`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
(B0-1, Promotion target section).

The MADR ADR rewrites this study for an external-reviewer
audience (no `studies/` back-references per R8), folds in any
updates from B0-6 (alert routing) and B0-7 (loader / scheduler /
retry failure semantics) that intersect with the failure
boundary, and updates foundation doc
[`05-operational-discipline.md`](../foundation/05-operational-discipline.md)
§"Failure Scope" to reference the ADR's policy (specifically
resolving the "exact boundary between 'check error' and 'entity
error' is the B0 question" pointer).

A separate **operator runbook** under `docs/operations/runbooks/`
is authored during Wave 3 (per CC11), mapping each
`(status, result)` combination to documented operator response
procedures. The runbook references this ADR for the underlying
policy.
