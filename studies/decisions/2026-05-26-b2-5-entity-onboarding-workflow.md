<!-- path: studies/decisions/2026-05-26-b2-5-entity-onboarding-workflow.md -->

# B2-5 — Entity Onboarding Workflow

## Context

Foundation 04 §5 ("Alerting") commits **progressive
promotion from test to production channels** as a
layer-5 responsibility. The platform's posture is that a
newly-onboarded entity should not page production
on-call rotations before its rule set has stabilized;
the operator needs an audit-grade procedure for moving
an entity from "candidate" to "production-alerting"
state.

The committed surfaces today around onboarding:

- [ADR-0006](../../docs/adr/0006-alert-routing-contract.md)
  commits the `_owners.yaml` schema with per-category
  `channels` (data_quality, operational) and the §9
  invariant "no alert without owner". The
  `severity_overrides` block already permits
  per-environment severity tuning.
- [ADR-0037](../../docs/adr/0037-owner-codeowners-cross-check.md)
  commits the linter-time owner-group cross-check
  against `.github/CODEOWNERS`.
- [ADR-0034](../../docs/adr/0034-local-testing-strategy.md)
  commits the six-tier test taxonomy; `make demo-p6` is
  the end-to-end smoke for an entity's rules.
- [ADR-0018](../../docs/adr/0018-env-config.md) +
  [ADR-0019](../../docs/adr/0019-infrastructure-tooling.md)
  commit per-environment deploy overlays
  (`deploy/overlays/{local,qa,prod}/`).
- [`CONTRIBUTING.md`](../../CONTRIBUTING.md) §"Flow 1 —
  Add a rule for a new entity" commits the technical
  steps for *adding* a rule (file shape, lint, PR).

What's not committed is the **governance checklist**
that determines when an entity is "ready" — the binary,
auditable criteria a reviewer evaluates before
authorizing test-channel routing, and the criteria for
the later test→production promotion. Today an operator
adding `rules/orders.yaml` + the matching
`_owners.yaml` entry has the *technical* path (Flow 1)
but no governance contract: how long should the entity
soak in test before promotion? what signals constitute
"stable"? who signs off? Each onboarding session would
otherwise improvise these criteria, and the audit trail
on which entities are production-eligible diverges.

The B2-5 row registered this gap at the W3 backlog
numbering step:

> What exact checklist determines when a new entity is
> ready for test channel and later for production
> alerting? Governance quality depends on repeatable
> onboarding.

The principles bearing on the decision are **P3**
(ownership is explicit — "no alert without owner" is
committed by ADR-0006 §9; this study extends the same
principle to "no production alert without a documented
readiness sign-off"), **P5** (evolution must be
contract-driven — the readiness checklist must be a
documented contract operators apply consistently rather
than per-entity judgment), and **R3** (do not revisit
settled architecture — ADR-0006, ADR-0018, ADR-0019,
ADR-0034 are preserved; this ADR adds the governance
layer on top).

What B2-5 must commit:

1. **The readiness-tier model** — what discrete states
   an entity passes through during onboarding.
2. **The checklist per tier** — auditable binary
   criteria a reviewer evaluates.
3. **The promotion procedure** — operational steps to
   move an entity from one tier to the next.
4. **The mechanism posture** — does this ADR commit
   new engine code (e.g., a per-entity "onboarding"
   flag), or does it work entirely with the existing
   environment-overlay separation?
5. **The runbook** — operator-facing playbook for the
   onboarding workflow, paired with the ADR.

---

## Decision Drivers

- **DD-1 — Audit-grade checklist beats per-session
  judgment.** Today's onboarding has no documented gate;
  each session improvises. Without a checklist,
  reviewers cannot verify that an entity has met the
  same bar as the entity onboarded last quarter — and
  the audit trail (who signed off, on what evidence) is
  reconstructed from PR comments rather than committed
  artefacts.

- **DD-2 — The existing environment-overlay separation
  is the natural "test channel" mechanism.**
  ADR-0018/0019 commit per-environment engine deployments
  (`deploy/overlays/qa/`, `deploy/overlays/prod/`). A
  ruleset published to the qa-overlay engine generates
  alerts against the same `_owners.yaml` channels as
  prod, but the *substrate* is different: qa Slack
  channels, qa email lists, qa PagerDuty rotations. The
  qa environment IS the test channel. No new
  per-entity routing mechanism is needed to give a new
  entity a test surface.

- **DD-3 — Engine-level routing-during-onboarding is
  out of scope.** A full "the entity exists in
  `_owners.yaml` but its alerts route to a fixed
  onboarding channel until promoted" mechanism would
  require either an `_owners.yaml` v3 schema amendment
  (per-entity `onboarding: true` flag) or a new
  `EnvConfig.OnboardingEntities []string` field. Both
  are real changes that reopen ADR-0006 or require a
  new env-config ADR. Defer to a follow-up B2 row —
  the readiness checklist itself does not require this
  mechanism, and the qa-substrate-as-test-surface
  posture (DD-2) is sufficient for the v1 onboarding
  workflow.

- **DD-4 — Readiness criteria must be observable, not
  subjective.** "Pass-rate stable" without a metric is
  per-reviewer judgment; "≥ 50 successful runs over ≥ 7
  calendar days in qa with zero `error`-class
  executions" is observable. The committed criteria
  must be queryable against the `dq_executions_current`
  view + `dq_check_results.result` column from ADR-0039
  (both committed in that ADR's stability tier) so a
  reviewer can mechanically check them.

- **DD-5 — Numeric thresholds are committed defaults
  with documented reasoning, not pulled from thin air.**
  The thresholds (≥ 50 successful runs, ≥ 7 calendar
  days, ≥ 95% check-level pass-rate) are
  **new contribution proposed here, requires review**.
  Reasoning: 7 calendar days covers one weekly business
  cycle so that any cycle-bounded drift (weekend
  refresh anomalies, end-of-week batch surges) appears
  in the soak window before promotion; 50 runs at the
  Phase-6 onboarding cadence (hourly to daily checks)
  gives meaningful redundancy past the 7-day floor for
  hourly entities and matches the floor for daily
  entities; 95% pass-rate is the floor at which the
  alerting layer's dedup and severity-overrides
  remain operationally meaningful — below 95%, the
  signal-to-noise ratio degrades enough that the
  entity's alerts mostly indicate steady-state
  problems rather than incidents. Per-entity overrides
  (above the floor) are recorded in the promotion PR
  description; *below* the floor requires an ADR
  amendment.

- **DD-6 — Three readiness tiers is the right
  granularity.** Two tiers (test/prod) is too coarse
  — there's a gap between "entity has rules but never
  ran" and "entity has rules running in qa." Four
  tiers would be over-engineering for the platform's
  current onboarding cadence (small N entities).
  Three tiers — **candidate**, **test-soak**,
  **production** — match the natural workflow:
  (a) PR lands with rule + `_owners.yaml` entry; (b)
  ruleset is published to qa, runs accumulate, soak
  completes; (c) ruleset is published to prod, alerts
  route to production channels.

- **DD-7 — Channel-reachability verification is
  outside ADR-0037's scope.** ADR-0037 commits the
  owner-group ↔ CODEOWNERS-group cross-check. It does
  NOT verify that the `channels` references in
  `_owners.yaml` resolve to reachable Slack channels,
  email addresses, or PagerDuty services. Reachability
  is a *manual* check at onboarding time and one of
  the readiness-checklist items here; it is not a lint
  enforcement (would require Slack-API / SMTP / etc.
  access from the linter, out of scope).

---

## Considered Options

### Option 1 — Three-tier readiness model + checklist per tier + operator runbook (recommended)

Commit the readiness-tier model and the auditable
checklist for each tier. Ship a paired operator runbook
`docs/runbooks/entity-onboarding.md` that documents the
procedure. No engine code change; the qa-substrate-as-
test-surface posture (DD-2) is sufficient.

The three tiers:

- **Tier 0 — Candidate.** Entity's rule YAML +
  `_owners.yaml` entry have landed on `main` (the PR
  merged). The rule has not been published to any
  environment yet. No alerts route anywhere.
- **Tier 1 — Test-soak.** Entity's ruleset is included
  in the manifest published to the qa-overlay engine.
  Alerts route to qa-substrate channels (the same
  channel names in `_owners.yaml`; the *substrate* is
  the qa Slack workspace + qa email lists). Runs
  accumulate; the soak window applies.
- **Tier 2 — Production.** Entity's ruleset is included
  in the manifest published to the prod-overlay engine.
  Alerts route to prod-substrate channels.

The checklist for each promotion:

**Tier 0 → Tier 1 (candidate → test-soak).**

- [ ] Rule YAML schema-validates (`make lint-rules`).
- [ ] `_owners.yaml` entry declares `owner`, `mode`,
      `description`, and at least one channel category
      with at least one channel reference.
- [ ] Owner-CODEOWNERS-group cross-check passes
      (ADR-0037).
- [ ] `make demo-p6` runs end-to-end with the new
      entity's rules merged into `rules/`.
- [ ] PR has approval from `@PLACEHOLDER-org/rules-authors`
      for the rule YAML and from
      `@PLACEHOLDER-org/platform-team` for the
      `_owners.yaml` edit (CODEOWNERS-routed per
      ADR-0015).

When all five boxes pass, a manifest publish under the
next `rules-vN.Y.Z` tag includes the entity. The qa
overlay's loader refreshes; Tier 1 begins on the first
run.

**Tier 1 → Tier 2 (test-soak → production).**

- [ ] **Soak window completed:** ≥ 50 successful
      executions accumulated in `dq_executions_current`
      with `entity = <new-entity>` AND `status =
      success`, over ≥ 7 calendar days
      (`recorded_at` spans ≥ 7 days). Numbers committed
      here are floor values; the operator may increase
      them for entities with low cadence (e.g., a
      weekly check requires fewer runs but the
      calendar-day floor still applies).
- [ ] **No unresolved `error`-class executions in the
      soak window:** zero rows with `status = error` in
      the qa `dq_executions` table for the entity
      within the soak window, OR every such row has a
      documented root-cause + fix that has shipped to
      `main`. The `error_summary` column from ADR-0003
      §3 is the lookup key.
- [ ] **Pass-rate within expected envelope:** the
      entity's check pass-rate over the soak window is
      ≥ 95% (i.e., `dq_check_results.result = pass`
      across all evaluations divided by total
      evaluations). The 95% floor is the committed
      default; entities with declared lower pass-rate
      expectations record the override in the PR
      description.
- [ ] **Channel-reachability verified:** operator
      manually confirms each channel reference in
      `_owners.yaml.entities.<entity>.channels`
      resolves to a real destination (Slack channel
      exists and the engine's webhook poster can write
      to it; email address accepts mail; PagerDuty
      service exists). The verification is a one-time
      check at onboarding; channels that disappear
      later trip the existing operational-alert flow.
- [ ] **Owner sign-off:** at least one member of the
      group declared in `_owners.yaml.entities.<entity>.owner`
      has approved a one-line "production-ready"
      comment on the promotion PR (the PR that adds
      the entity to the prod overlay's published
      ruleset).
- [ ] **Platform-team sign-off:** platform-team has
      approved the prod-overlay change. The joint
      platform-team + SRE review on
      `/deploy/overlays/prod/` is committed by the
      CODEOWNERS path-rule table in ADR-0015 §3.

When all six boxes pass, the prod-overlay manifest
publication includes the entity. The prod overlay's
loader refreshes; Tier 2 begins on the first prod run.

**The runbook.**

`docs/runbooks/entity-onboarding.md` is the operator-
facing playbook. It documents the procedure for each
tier promotion: which commands to run, which queries to
execute against `dq_executions_current` to verify the
soak criteria, what to look for in PR review, how to
escalate when criteria are unclear. It follows the
existing runbook anatomy (When to use → Preconditions
→ Procedure → Verification → Rollback / escape →
Escalation) from `docs/runbooks/README.md`.

**Strengths.** Commits an auditable governance contract
without inventing new engine code (no ADR-0006
amendment, no new env-config field). Leverages the
existing environment-overlay separation (DD-2) as the
test-surface substrate. Criteria are queryable against
ADR-0039's dashboard contract — reviewers verify
mechanically rather than by judgment. The numeric
thresholds (50 runs / 7 days / 95% pass-rate) are
committed defaults that can be increased per-entity for
edge cases, not soft preferences.

**Trade-offs.** No engine-level "this entity is
onboarding, route its alerts to a fixed onboarding
channel" mechanism — the qa substrate IS the test
surface, so the onboarding channels are whatever
`_owners.yaml.channels` resolves to in the qa overlay's
deployment substrate. Operators using the same Slack
workspace for qa and prod will see qa-routed onboarding
alerts in production-named channels; the workaround is
to use a qa-prefixed channel name (e.g.,
`slack:#qa-dq-orders`) in `_owners.yaml` for
onboarding-period entities and switch to the production
name at promotion. A future B2 row registers the
engine-level mechanism that would close this workaround
permanently.

### Option 2 — Three-tier model + engine-level onboarding flag

Same as Option 1, but additionally amend ADR-0006 to add
an optional `onboarding: true` per-entity flag in
`_owners.yaml` v3. When the flag is set, the engine
routes alerts to an `EnvConfig.OnboardingChannel`
override regardless of `channels`. At promotion, the
flag flips to `false` (removed or set to false in
`_owners.yaml`) in the same PR that bumps the
production ruleset version.

**Strengths.** Closes the qa-prod-channel-collision
workaround from Option 1's trade-off section. Engine-
level mechanism means the routing decision is
deterministic and queryable from the engine's logs.

**Trade-offs.** Requires `_owners.yaml` v3 schema bump
(touches ADR-0001's compatibility commitment),
requires `EnvConfig` extension (touches ADR-0018), and
introduces a new code path in the alerting layer that
must be tested separately. Out of scope for B2-5 — the
checklist itself does not require this mechanism. The
workaround in Option 1 (qa-prefixed channel names) is
adequate for v1 onboarding cadence and the engine-level
mechanism is reserved as a follow-up B2 row when
concrete operational signal demonstrates the workaround
is insufficient.

### Option 3 — Checklist only, no runbook

Commit the checklist in the ADR but skip the paired
runbook.

**Strengths.** Smaller PR, faster to land.

**Trade-offs.** The ADR commits *what* must be true at
promotion; the runbook commits *how* the operator
verifies it. Splitting them means each operator
reconstructs the procedure (which BigQuery queries
verify the soak criteria, which gh-pr commands check
the approvals) from the ADR's prose. The runbook is
the operational complement; shipping the checklist
without it leaves the operator without a procedure.
Rejected — pair them in one PR.

---

## Recommendation

**Option 1.** Three-tier readiness model + auditable
per-tier checklist + paired operator runbook. No
engine code change. The qa-substrate-as-test-surface
posture (DD-2) is the v1 mechanism.

### Three-tier model

| Tier | Name | Definition | Alerts route to |
|---|---|---|---|
| 0 | Candidate | Rule + `_owners.yaml` entry merged on `main`; not yet published in any manifest. | Nowhere (no engine has the ruleset loaded). |
| 1 | Test-soak | Ruleset including the entity published to the qa-overlay manifest; the qa engine's loader has adopted it. | Channels declared in `_owners.yaml`, against the **qa substrate** (qa Slack workspace, qa email lists, qa PagerDuty). |
| 2 | Production | Ruleset including the entity published to the prod-overlay manifest; the prod engine's loader has adopted it. | Channels declared in `_owners.yaml`, against the **prod substrate**. |

### Tier 0 → Tier 1 checklist

Six criteria. All six must be checked before the qa
manifest publish that includes the entity:

1. Rule YAML schema-validates (`make lint-rules`).
2. `_owners.yaml` entry declares `owner`, `mode`,
   `description`, and at least one non-empty channel
   category.
3. Owner-CODEOWNERS-group cross-check passes per
   ADR-0037 (verified by `make lint-rules`).
4. `make demo-p6` runs end-to-end with the new
   entity's rules.
5. PR has both approvals: `@PLACEHOLDER-org/rules-authors`
   for the rule YAML and `@PLACEHOLDER-org/platform-team`
   for the `_owners.yaml` edit (CODEOWNERS-routed).
6. **Shared-substrate channel-collision posture**: if
   qa and prod share the same substrate workspace (one
   Slack workspace, one email domain, one PagerDuty
   tenant), channel references in
   `_owners.yaml.entities.<entity>.channels` carry a
   qa-prefix (e.g., `slack:#qa-dq-orders`,
   `email:qa-oncall@example.com`) so Tier 1 alerts do
   not land in production-named channels. Deployments
   with separate per-environment substrates skip this
   criterion with a recorded justification in the PR
   description.

### Tier 1 → Tier 2 checklist

Seven criteria. All seven must be checked before the
prod manifest publish that includes the entity:

1. **Soak window:** ≥ 50 successful executions in
   qa's `dq_executions_current` for the entity, spanning
   ≥ 7 calendar days. Floor values; operator may
   increase for low-cadence entities. Verifiable query
   shape (against ADR-0039's contract):

   ```sql
   SELECT COUNT(*), MIN(recorded_at), MAX(recorded_at)
   FROM dq_executions_current
   WHERE entity = '<new-entity>'
     AND status = 'success'
   ```

2. **No unresolved errors:** zero rows with `status =
   error` for the entity in the soak window, OR every
   such row has a documented root-cause + fix on
   `main`. `error_summary` is the lookup key.

3. **Pass-rate ≥ 95%:** check pass-rate over the soak
   window (`dq_check_results.result = pass` / total
   evaluations) is at least 95%. Floor value; declared
   overrides recorded in the promotion PR description.

4. **Channel reachability:** each channel reference in
   `_owners.yaml.entities.<entity>.channels` manually
   verified — Slack channel exists and is writable
   from the engine's poster, email address accepts
   mail, PagerDuty service exists.

5. **Owner sign-off:** at least one member of the
   `_owners.yaml.entities.<entity>.owner` group has
   approved with a "production-ready" comment on the
   promotion PR.

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
   The edit is CODEOWNERS-routed to platform-team for
   review. Deployments that skipped criterion 6 (Tier
   0 → Tier 1) also skip this one.

### Paired runbook

`docs/runbooks/entity-onboarding.md` lands with this
ADR. It follows the runbook anatomy from
`docs/runbooks/README.md` §"Anatomy of a runbook" and
documents the procedure for both promotions, the SQL
queries to verify each checklist item, the rollback
mechanism if a Tier 2 promotion produces unexpected
behavior, and escalation paths.

### Why this does not reopen ADR-0006

ADR-0006 §9 commits "no alert without owner" and the
`_owners.yaml` schema. This study commits a *governance
layer* on top of the schema (when is an entity ready to
have its declared channels actually route?). The
schema, the cross-check, and the channel structure are
all preserved. The future engine-level
routing-during-onboarding mechanism (Option 2)
*would* reopen ADR-0006 — but it's deferred as a B2
follow-up.

### Why this does not reopen ADR-0018 / ADR-0019

ADR-0018 commits the env-config struct; ADR-0019
commits the deploy-overlay structure. The
three-tier model uses the existing overlays
(qa = Tier 1 substrate, prod = Tier 2 substrate)
without adding new env-config fields. The
engine-level onboarding mechanism (Option 2 deferred)
would add a field; B2-5 does not.

### Why this does not reopen ADR-0034 / ADR-0039

ADR-0034 commits the local-testing taxonomy
(`make demo-p6` is Tier 5 smoke-substrate). This ADR
cites it as the Tier 0 → Tier 1 criterion #4 without
amending it. ADR-0039 commits the dashboard contract;
this ADR uses `dq_executions_current` as a queryable
source for the Tier 1 → Tier 2 criteria without
amending the contract.

---

## Consequences

1. **A three-tier readiness model is committed.**
   Operators have a documented progression from
   "rule merged" through "soak-stable in qa" to
   "production-eligible". Each tier has a
   well-defined entry condition.

2. **An auditable checklist per promotion is
   committed.** Five criteria for Tier 0 → Tier 1; six
   criteria for Tier 1 → Tier 2. Each criterion is
   binary and verifiable — reviewers tick boxes rather
   than apply judgment.

3. **The qa environment IS the test surface.** The
   qa-overlay engine's deployment substrate (qa Slack
   workspace, qa email, qa PagerDuty) is the routing
   destination during Tier 1. No new engine code is
   shipped; the existing environment-overlay separation
   is the mechanism.

4. **`docs/runbooks/entity-onboarding.md` ships
   alongside the ADR.** The runbook documents the
   procedure for each promotion, the SQL queries that
   verify the soak criteria against
   `dq_executions_current`, rollback for unexpected
   prod behavior, and escalation paths.

5. **Numeric thresholds are committed defaults.** 50
   runs / 7 days / 95% pass-rate are floor values for
   the Tier 1 → Tier 2 promotion. Operators may
   increase per-entity (e.g., a weekly-cadence check
   keeps the 7-day floor but the 50-run floor adjusts
   downward with a recorded justification in the PR).
   Decreasing below the committed floor requires an
   ADR amendment.

6. **The shared-workspace channel-collision workaround
   is procedurally enforced.** `_owners.yaml` is
   environment-agnostic; if qa and prod share the same
   substrate workspace (one Slack workspace, one email
   domain), Tier 1 alerts otherwise land in
   production-named channels. To close this at the
   onboarding-procedure layer, the **Tier 0 → Tier 1
   checklist gains a sixth criterion** (criterion 6
   below) committing that operators in shared-substrate
   deployments declare qa-prefixed channel names
   (`slack:#qa-dq-orders`) at Tier 0, and that the
   **Tier 1 → Tier 2 checklist gains a seventh
   criterion** (criterion 7 below) committing the
   channel-rename `_owners.yaml` edit at promotion.
   Operators in non-shared-substrate deployments
   (separate Slack workspaces per environment) skip
   both criteria with a recorded justification in the
   PR description. A future B2 row registers the
   engine-level mechanism that would close this
   permanently and eliminate the procedural enforcement
   (OQ-1 below).

7. **Channel reachability is a manual verification.**
   The linter's owner-group cross-check from ADR-0037
   does not verify channel destinations. Reachability
   is part of the Tier 1 → Tier 2 checklist (criterion
   4). A future linter extension may automate this
   when access to the channel substrates is
   architecturally feasible (registered as a future
   B2 row).

8. **B2-5 closes.** The decision-log B2-5 row moves
   to `resolved-adr` (→ ADR-0040). Two new B2 rows
   register the follow-ups: engine-level
   onboarding-channel mechanism (the Option 2
   deferral) and channel-reachability lint extension.

9. **ADR-0006, ADR-0018, ADR-0019, ADR-0034,
   ADR-0037, ADR-0039 are preserved.** This ADR
   layers a governance contract on top of their
   commitments without amending them.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: Engine-level onboarding-channel mechanism.**
  An `_owners.yaml` v3 amendment adding per-entity
  `onboarding: true` + an `EnvConfig.OnboardingChannel`
  override would close the qa-prod-channel-collision
  workaround from Consequence #6. Reserved as a B2
  follow-up; the workaround is adequate for v1
  onboarding cadence.

- **OQ-2: Channel-reachability linter extension.** The
  Tier 1 → Tier 2 channel-reachability check
  (criterion 4) is manual today. A linter extension
  that pings Slack-API / SMTP / PagerDuty-API to
  verify destinations could automate the check.
  Reserved as a B2 follow-up; the manual check is
  adequate for v1 onboarding cadence and automating it
  requires substrate access from the linter (out of
  scope per ADR-0034's local-testing posture).

---

## Promotion target

`docs/adr/0040-entity-onboarding-workflow.md` — next
free ADR number. Ships the three-tier model, the
per-tier checklist, and the runbook reference. The
paired runbook lands as `docs/runbooks/entity-onboarding.md`
in the same PR.
