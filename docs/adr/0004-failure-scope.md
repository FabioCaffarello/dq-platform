<!-- path: docs/adr/0004-failure-scope.md -->

# ADR-0004 — Failure Scope

- **Status:** accepted
- **Date:** 2026-05-21

**Scope note (added 2026-05-23):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).

---

## Context

A data quality evaluation produces per-check results. The
platform must map those per-check results to an
execution-level status and to alert categories, and must
decide whether one check's failure abandons sibling checks.

The mapping is a public contract: dashboards, alert routing,
incident response, and downstream analytics all key off the
execution `status` and the per-check `result`. Once those
semantics are committed, changing them is a breaking event
for every consumer.

Three positions on the design space, each with a real cost:

- **Abort on first error** maximizes per-attempt cost
  efficiency but loses information about sibling-check
  health, breaking the platform's "visible failure over
  silent degradation" posture.
- **Always continue** preserves information but spends `O(N)`
  in worst-case failure cost (every check errors against a
  missing source table).
- **The check-error → entity-error promotion threshold** is
  the central judgement call: too strict and a single
  operational hiccup taints the whole entity; too lax and
  "no evaluation succeeded" loses its strong-claim meaning.

---

## Decision

### 1. Check `result` enum

A check produces exactly one value:

- **`pass`** — the check's query executed and the data met
  the rule's pass condition.
- **`fail`** — the check's query executed and the data did
  not meet the rule's pass condition (above the warn
  threshold if one is defined).
- **`degraded`** — the check's query executed and the data
  fell into a warning band (between pass and fail).
- **`error`** — the check's query did not execute
  successfully (query compilation error, missing source
  table or column, quota exhaustion, transient-error retry
  budget exhausted, evaluation-budget timeout).

A check that completes with zero rows examined where the
rule says "no data is fine" is `pass`; where the rule says
"data must be present" it is `fail`. The DSL construct that
declares which interpretation a rule wants is a follow-up.

### 2. Execution `status` mapping (pure function)

Given the multiset `R` of check results for one logical
execution, the status is determined by the following
mutually-exclusive decision procedure, applied in order:

1. **Global engine halt** at any point during evaluation →
   `status = aborted`. (Halt conditions are specified by
   ADR-0007.)
2. **Pre-check entity-level problem** detected (no check
   rows written) → `status = error`.
3. `R ≠ ∅` and every element of `R` is `pass` →
   `status = success`.
4. `R ≠ ∅` and every element of `R` is `error` →
   `status = error`.
5. Otherwise (`R ≠ ∅` with any mixed-result combination) →
   `status = failed`.

The five branches are mutually exclusive by construction.
The mapping is a single pure function; no operator
discretion.

### 3. Promotion rule (threshold form)

The only path from check-level errors to execution-level
`error` (other than the pre-check entity-level problem
branch above) is **every check produced `error`**. One
errored check among ten successful ones produces
`status = failed`, not `error`. Mixed-result executions
(some `error`, some non-`error`) stay at `failed`.

The threshold is binary, not proportional. A multiset like
`{pass, error, error, ..., error}` (nine errors plus one
pass) produces `failed`. `status = error` is reserved for
the strong claim "we could not evaluate this entity at
all"; the operational signal of any errored check is
preserved through per-check alerting (below) and through
the `result` column on `dq_check_results`.

### 4. Continuation rule

The engine **always continues at the check level**. Every
check listed in the manifest for the entity is attempted;
every attempt's result is recorded. The terminal execution
row is written after every check has been evaluated (or has
errored). A check-level error never triggers an abort.

Engine-level halts that prevent the engine from completing
the evaluation (cost ceiling, manifest load failure, OOM)
are governed by ADR-0007 and produce `status = aborted`,
which is distinct from `error`.

### 5. Pre-check entity-level problems

When the engine detects an entity-level problem **before**
any check has been attempted — the kind that would make
every check error — the execution row is written directly
with `status = error` and no `dq_check_results` rows are
produced. This is the only path to `status = error` without
all checks producing `error` results.

The specific set of pre-check validations (manifest
contract per ADR-0001, source-table existence, partition
column presence, others) is implementation-side; this ADR
commits only the state-transition outcome.

### 6. Alerting category boundaries

| Trigger                     | Category          |
|-----------------------------|-------------------|
| check `pass`                | no alert          |
| check `fail` / `degraded`   | data quality      |
| check `error`               | operational       |
| execution `success`         | no alert          |
| execution `failed`          | per-check fan-out |
| execution `error`           | operational       |
| execution `aborted`         | operational       |

The category boundary is non-negotiable. ADR-0006 specifies
routing targets, severity within each category, and
deduplication; it cannot route a check `error` as a
data-quality alert or vice versa. On execution `failed`,
each check's `result` produces its own alert per the rows
above; the execution-level row does not trigger a separate
alert.

---

## Consequences

1. **The mapping is a pure function of the result multiset.**
   No operator discretion, no per-environment override. The
   implementation is a small enum reducer inside the
   engine's reporter.

2. **`status = error` retains its strong-claim meaning.**
   Only "every check errored" or "pre-check entity-level
   problem" reaches it. Mixed evaluation outcomes — even
   highly-skewed ones — stay at `failed`. Consumers wanting
   "any operational signal triggered this execution" query
   the per-check `result` column, not just the
   execution-level `status`.

3. **The platform spends `O(N)` cost in worst-case failure.**
   A missing source table causes every check to error in
   the fast-fail path. The platform does not impose a hard
   cap on N per entity at this policy layer; if a future
   entity exceeds practical tolerance, a per-environment
   ceiling is a governance follow-up.

4. **Check-level error never abandons sibling checks.** The
   information value of sibling outcomes (some succeed
   despite one erroring) outweighs the cost-efficiency
   gain of aborting early.

5. **The category boundary is a contract for ADR-0006.**
   ADR-0006 implements routing under these category labels;
   it cannot reassign categories. Severity within a
   category and dedup windows are ADR-0006's call.

6. **Idempotency interaction.** A scheduler retry under the
   same `execution_id` recomputes failure scope per attempt;
   the canonical view per ADR-0002 returns the latest
   attempt's status. A retry that turns `error` into
   `success` (the source table came back) reaches dashboards
   via the latest-attempt projection without losing the
   prior attempt's history in the base table.

7. **Operator-rerun interaction.** An operator rerun
   produces a new `execution_id` with `supersedes_execution_id`
   pointing at the original. The rerun's status is computed
   independently; the audit link makes the relationship
   visible without merging the two outcomes.

8. **No new enum values are introduced by this ADR.** The
   `result` and `status` enums are as committed by ADR-0003.
   The contribution here is the *mapping*. Extension is
   additive: adding a new `result` value (`warn`,
   `inconclusive`, etc.) requires updating the mapping in
   the same ADR that adds the value.

9. **The mapping is a public contract.** Changes to the
   mapping function are breaking and require a future ADR
   with a migration path and compatibility window.

10. **Operator-response documentation is downstream.** Each
    `(status, result)` combination maps to documented
    operator actions in a follow-up artifact (runbook,
    operations doc, or equivalent). The format is not
    foreclosed here; the policy committed in this ADR
    informs the artifact's content.

---

## Notes

- The exact set of pre-check validations is implementation-
  side and may evolve as the platform learns which classes of
  entity-level problems are common. The state-transition
  outcome committed here does not change as that set evolves.
- The DSL construct for the empty-window pass-vs-fail
  declaration is a follow-up in the rules workspace's
  onboarding contract.
- A future per-environment "halt at N consecutive errors"
  ceiling is governance, not policy; it would live as a
  rules-workspace policy or an operations parameter, not as
  a change to the mapping function.
