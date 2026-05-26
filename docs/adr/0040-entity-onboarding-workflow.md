<!-- path: docs/adr/0040-entity-onboarding-workflow.md -->

# ADR-0040 — Entity Onboarding Workflow

- **Status:** accepted
- **Date:** 2026-05-26

---

## Context

Foundation 04 §5 ("Alerting") commits **progressive
promotion from test to production channels** as a
layer-5 responsibility. The platform's posture is that
a newly-onboarded entity should not page production
on-call rotations before its rule set has stabilized;
the operator needs an audit-grade procedure for moving
an entity from "candidate" through "test-soak" to
"production-eligible".

The surfaces this ADR layers on top of:

- [ADR-0006](./0006-alert-routing-contract.md) commits
  the `_owners.yaml` schema with per-category
  `channels` and the §9 invariant "no alert without
  owner".
- [ADR-0037](./0037-owner-codeowners-cross-check.md)
  commits the linter-time owner-CODEOWNERS-group
  cross-check.
- [ADR-0034](./0034-local-testing-strategy.md) commits
  the six-tier test taxonomy; `make demo-p6` is the
  end-to-end smoke for an entity's rules.
- [ADR-0018](./0018-env-config.md) +
  [ADR-0019](./0019-infrastructure-tooling.md) commit
  per-environment deploy overlays
  (`deploy/overlays/{local,qa,prod}/`).
- [ADR-0039](./0039-dashboard-contract.md) commits the
  `dq_executions_current` view + `dq_check_results`
  column inventory as stable consumer surfaces.
- [`CONTRIBUTING.md`](../../CONTRIBUTING.md) §"Flow 1"
  commits the technical steps for adding a rule.

What's not committed is the **governance checklist**
that determines when an entity is "ready" — the
binary, auditable criteria a reviewer evaluates before
the qa manifest publish, and again before the prod
manifest publish. Today an operator adding
`rules/orders.yaml` plus the matching `_owners.yaml`
entry has the technical path but no governance
contract: how long should the entity soak in qa? what
signals constitute "stable"? who signs off? Each
onboarding session would otherwise improvise these
criteria, and the audit trail diverges.

The principles bearing on the decision are **P3**
(ownership is explicit — extends the "no alert without
owner" principle to "no production alert without a
documented readiness sign-off") and **P5** (evolution
must be contract-driven — the checklist is a documented
contract operators apply consistently rather than
per-entity judgment). R3 is the operating constraint:
ADR-0006, ADR-0018, ADR-0019, ADR-0034, ADR-0037,
ADR-0039 are all preserved; this ADR adds a governance
layer on top.

---

## Decision

### Three-tier readiness model

An entity passes through three discrete tiers during
onboarding:

| Tier | Name | Definition | Alerts route to |
|---|---|---|---|
| 0 | Candidate | Rule + `_owners.yaml` entry merged on `main`; not yet published in any manifest. | Nowhere (no engine has the ruleset loaded). |
| 1 | Test-soak | Ruleset including the entity published to the qa-overlay manifest; the qa engine's loader has adopted it. | Channels declared in `_owners.yaml`, against the **qa substrate** (qa Slack workspace, qa email lists, qa PagerDuty). |
| 2 | Production | Ruleset including the entity published to the prod-overlay manifest; the prod engine's loader has adopted it. | Channels declared in `_owners.yaml`, against the **prod substrate**. |

The qa-substrate-as-test-surface posture is the v1
mechanism: the existing environment-overlay separation
(ADR-0018, ADR-0019) deploys the qa engine with a qa
substrate (qa Slack workspace, qa email, qa PagerDuty)
distinct from prod. A ruleset published to qa generates
alerts against the qa substrate's resolution of the
channel strings in `_owners.yaml`; the prod ruleset
resolves the same strings against the prod substrate.
No new engine code is needed.

For deployments where qa and prod **share** a substrate
workspace (one Slack workspace, one email domain, one
PagerDuty tenant), the channel strings in
`_owners.yaml` collide between Tier 1 and Tier 2. The
checklist closes this at the procedure layer (criteria
6 + 7 below) by committing qa-prefixed channel names
at Tier 0 → Tier 1 and a rename-back at Tier 1 → Tier
2. A future engine-level mechanism (per-entity
`onboarding: true` flag with an
`EnvConfig.OnboardingChannel` override) is deferred to
a B2 follow-up.

### Tier 0 → Tier 1 checklist

Six criteria. All six must be checked before the qa
manifest publish that includes the entity:

1. **Schema validation:** rule YAML schema-validates
   (`make lint-rules`).
2. **Owner declaration:** `_owners.yaml` entry declares
   `owner`, `mode`, `description`, and at least one
   non-empty channel category.
3. **CODEOWNERS cross-check:** owner-group ↔
   `.github/CODEOWNERS` cross-check passes per
   ADR-0037 (verified by `make lint-rules`).
4. **End-to-end demo:** `make demo-p6` runs end-to-end
   with the new entity's rules.
5. **Code review:** PR has both approvals —
   `@PLACEHOLDER-org/rules-authors` for the rule YAML
   and `@PLACEHOLDER-org/platform-team` for the
   `_owners.yaml` edit (CODEOWNERS-routed per
   ADR-0015).
6. **Shared-substrate channel-collision posture**: if
   qa and prod share the same substrate workspace
   (one Slack workspace, one email domain, one
   PagerDuty tenant), channel references in
   `_owners.yaml.entities.<entity>.channels` carry a
   qa-prefix (e.g., `slack:#qa-dq-orders`,
   `email:qa-oncall@example.com`) so Tier 1 alerts do
   not land in production-named channels. Deployments
   with separate per-environment substrates skip this
   criterion with a recorded justification in the PR
   description.

When all six boxes pass, the next `rules-vN.Y.Z` tag's
qa-overlay manifest publish includes the entity. The
qa overlay's loader refreshes; Tier 1 begins on the
first run.

### Tier 1 → Tier 2 checklist

Seven criteria. All seven must be checked before the
prod manifest publish that includes the entity:

1. **Soak window completed:** ≥ 50 successful
   executions accumulated in qa's
   `dq_executions_current` for the entity, spanning ≥
   7 calendar days. The committed floors; per-entity
   overrides recorded in the promotion PR description
   may increase them (e.g., for low-cadence weekly
   checks the 50-run floor adjusts downward with a
   justification, but the 7-day floor still applies).
   Verifiable query shape:

   ```sql
   SELECT COUNT(*) AS soak_runs,
          MIN(recorded_at) AS soak_start,
          MAX(recorded_at) AS soak_end
   FROM dq_executions_current
   WHERE entity = '<new-entity>'
     AND status = 'success'
   ```

2. **No unresolved errors:** zero rows with
   `status = error` for the entity in the soak window,
   OR every such row has a documented root-cause + fix
   that has shipped to `main`. `error_summary` column
   (ADR-0003 §3) is the lookup key.

3. **Pass-rate ≥ 95%:** check-level pass-rate over the
   soak window
   (`dq_check_results.result = 'pass'` evaluations
   divided by total) is at least 95%. Committed floor;
   declared per-entity overrides above the floor are
   recorded in the promotion PR. Decreasing the floor
   requires an amendment to this ADR.

4. **Channel reachability:** each channel reference
   in `_owners.yaml.entities.<entity>.channels`
   manually verified — Slack channel exists and is
   writable from the engine's poster, email address
   accepts mail, PagerDuty service exists.

5. **Owner sign-off:** at least one member of the
   group declared in
   `_owners.yaml.entities.<entity>.owner` has approved
   with a "production-ready" comment on the promotion
   PR.

6. **Platform-team sign-off:** platform-team approves
   the prod-overlay change. The joint platform-team +
   SRE review on `/deploy/overlays/prod/` is committed
   by the CODEOWNERS path-rule table in ADR-0015 §3
   (line `/deploy/overlays/prod/ @PLACEHOLDER-org/platform-team @PLACEHOLDER-org/sre`).

7. **Channel-rename to production names** (only when
   Tier 0 → Tier 1 criterion 6 applied): the promotion
   PR includes the `_owners.yaml` edit renaming
   qa-prefixed channel names back to production names
   (e.g., `slack:#qa-dq-orders` → `slack:#dq-orders`).
   The edit is CODEOWNERS-routed to platform-team.
   Deployments that skipped criterion 6 also skip this
   one.

When all seven boxes pass, the next `rules-vN.Y.Z`
tag's prod-overlay manifest publish includes the
entity. The prod overlay's loader refreshes; Tier 2
begins on the first prod run.

### Numeric thresholds — reasoning

The thresholds (≥ 50 successful runs, ≥ 7 calendar
days, ≥ 95% check-level pass-rate) are committed
defaults with the following reasoning:

- **7 calendar days** covers one weekly business cycle
  so that any cycle-bounded drift (weekend refresh
  anomalies, end-of-week batch surges, Monday-morning
  data-source delays) appears in the soak window
  before promotion.
- **50 runs** at the Phase-6 onboarding cadence
  (hourly to daily checks) gives meaningful redundancy
  past the 7-day floor for hourly entities and matches
  the floor for daily entities.
- **95% pass-rate** is the floor at which the alerting
  layer's dedup and severity-overrides remain
  operationally meaningful — below 95%, the
  signal-to-noise ratio degrades enough that the
  entity's alerts mostly indicate steady-state problems
  rather than incidents.

Per-entity overrides *above* the floors are recorded
in the promotion PR description (e.g., a critical
financial-reporting entity may require ≥ 100 runs and
≥ 99% pass-rate). Decreasing *below* the committed
floors requires an amendment to this ADR.

### Paired runbook

`docs/runbooks/entity-onboarding.md` ships with this
ADR. It documents the operator procedure for each
promotion: which commands to run, which SQL queries
verify the soak criteria against
`dq_executions_current`, what to look for in PR
review, channel-reachability verification steps,
rollback for unexpected prod behavior, and escalation
paths. It follows the standard six-section runbook
anatomy from `docs/runbooks/README.md`.

### Why this does not reopen prior ADRs

- **ADR-0006** §9 commits "no alert without owner" and
  the `_owners.yaml` schema. This ADR layers a
  governance contract on top (when may declared channels
  route?). Schema, cross-check, channel structure are
  preserved.
- **ADR-0018 / ADR-0019** commit the env-config struct
  and overlay structure. This ADR uses the existing
  overlays (qa = Tier 1, prod = Tier 2) without adding
  new env-config fields.
- **ADR-0034 / ADR-0039** commit the test taxonomy and
  the dashboard contract. This ADR cites
  `make demo-p6` and `dq_executions_current` +
  `dq_check_results.result` (both in ADR-0039's stable
  inventory) without amending either.
- **ADR-0037** commits the owner-CODEOWNERS-group
  cross-check. This ADR cites it as Tier 0 → Tier 1
  criterion #3 without amending.

---

## Consequences

1. **A three-tier readiness model is committed.**
   Operators have a documented progression from "rule
   merged" through "soak-stable in qa" to
   "production-eligible". Each tier has a well-defined
   entry condition.

2. **Auditable checklists per promotion are
   committed.** Six criteria for Tier 0 → Tier 1;
   seven criteria for Tier 1 → Tier 2. Each criterion
   is binary and verifiable — reviewers tick boxes
   rather than apply judgment.

3. **The qa environment IS the test surface.** The
   qa-overlay engine's deployment substrate is the
   routing destination during Tier 1. No new engine
   code is shipped; the existing environment-overlay
   separation is the mechanism.

4. **`docs/runbooks/entity-onboarding.md` ships
   alongside this ADR.** The runbook documents the
   procedure, the SQL queries that verify the soak
   criteria, the rollback mechanism if a Tier 2
   promotion produces unexpected behavior, and
   escalation paths.

5. **Numeric thresholds are committed defaults with
   documented reasoning.** 50 runs / 7 days / 95%
   pass-rate are floor values with one-sentence
   reasoning per the §"Numeric thresholds — reasoning"
   block. Per-entity overrides above the floor recorded
   in the promotion PR; below-floor decreases require
   an ADR amendment.

6. **The shared-substrate channel-collision workaround
   is procedurally enforced.** Tier 0 → Tier 1
   criterion 6 + Tier 1 → Tier 2 criterion 7 commit
   the qa-prefix + rename-back posture for deployments
   sharing a substrate workspace. Deployments with
   separate per-environment substrates skip both
   criteria with a recorded justification.

7. **Channel reachability is a manual verification.**
   The linter's owner-group cross-check from ADR-0037
   does not verify channel destinations. Reachability
   is Tier 1 → Tier 2 criterion 4. A future linter
   extension may automate this when access to channel
   substrates is architecturally feasible (registered
   as a B2 follow-up).

8. **B2-5 closes.** The decision-log B2-5 row moves to
   `resolved-adr`. Two new B2 rows register the
   follow-ups: engine-level onboarding-channel
   mechanism (the procedural-workaround replacement)
   and channel-reachability linter extension.

9. **ADR-0006, ADR-0018, ADR-0019, ADR-0034, ADR-0037,
   ADR-0039 are preserved.** This ADR layers a
   governance contract on top of their commitments
   without amending them.

10. **Two deferred items are registered out-of-scope:**

    - **OQ-1: Engine-level onboarding-channel mechanism.**
      An `_owners.yaml` v3 amendment adding per-entity
      `onboarding: true` + an
      `EnvConfig.OnboardingChannel` override would close
      the shared-substrate workaround permanently.
      Reserved as a B2 follow-up.

    - **OQ-2: Channel-reachability linter extension.**
      A linter extension pinging Slack-API / SMTP /
      PagerDuty-API to verify destinations could
      automate Tier 1 → Tier 2 criterion 4. Reserved
      as a B2 follow-up; substrate access from the
      linter is out of scope per ADR-0034's
      local-testing posture.
