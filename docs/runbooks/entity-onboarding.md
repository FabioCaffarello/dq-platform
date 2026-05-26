<!-- path: docs/runbooks/entity-onboarding.md -->

# Runbook — Entity onboarding workflow

Move a new entity through the three-tier readiness
model committed by
[ADR-0040](../adr/0040-entity-onboarding-workflow.md):
**Candidate → Test-soak → Production**.

The qa environment IS the test surface — alerts during
Tier 1 route to the qa substrate (qa Slack workspace,
qa email lists, qa PagerDuty), not to a per-entity
"test channel" the engine routes specially. No engine
code paths trigger on tier transitions; the procedure
is governance-driven and gated by the per-tier
checklists in ADR-0040 §"Tier 0 → Tier 1 checklist"
and §"Tier 1 → Tier 2 checklist".

---

## 1. When to use

- A new entity needs onboarding: the rule YAML +
  `_owners.yaml` entry have either landed on `main`
  (Tier 0 — proceed to §3.A for the Tier 0 → Tier 1
  promotion) or have been running in qa for some
  period (Tier 1 — proceed to §3.B for the Tier 1 →
  Tier 2 promotion).
- An operator wants to verify whether an entity meets
  the criteria for promotion (proceed to §4
  Verification queries).

Do **not** use this runbook for:

- Repairing a broken entity in production (use
  [`alert-dedup-debugging.md`](alert-dedup-debugging.md)
  or [`orphan-run-remediation.md`](orphan-run-remediation.md)
  depending on the failure mode).
- A manifest-wide rollback that affects multiple
  entities (use
  [`manifest-rollback.md`](manifest-rollback.md)).

## 2. Preconditions

- Read access to qa BigQuery for the soak queries (§4.A).
- Write access to the manifest bucket for the qa
  overlay (Tier 0 → Tier 1) and the prod overlay
  (Tier 1 → Tier 2) — typically `dq-manifest publish`
  via a CI lane keyed on the `rules-vN.Y.Z` tag.
- `make lint-rules` clean locally (the linter is the
  cheapest enforcement layer; failures here block
  the promotion regardless of tier).
- Awareness of the deployment's substrate posture:
  whether qa and prod share a Slack workspace, email
  domain, or PagerDuty tenant. ADR-0040 §"Tier 0 →
  Tier 1 checklist" criterion 6 applies to
  shared-substrate deployments only.

## 3. Procedure

### 3.A Tier 0 → Tier 1 (candidate → test-soak)

Run through ADR-0040 §"Tier 0 → Tier 1 checklist" in
order:

1. **Schema validation.** Run `make lint-rules`. The
   linter exit code is non-zero on any failure; fix
   issues at the source and re-run.

2. **Owner declaration.** Verify
   `_owners.yaml.entities.<new-entity>` declares
   `owner`, `mode`, `description`, and at least one
   non-empty channel category. The linter enforces
   this via the `_owners.yaml` schema.

3. **CODEOWNERS cross-check.** `make lint-rules` also
   runs the ADR-0037 cross-check against
   `.github/CODEOWNERS`. A failure prints `ADR-0037:
   entity <name> owner <value> does not match any
   CODEOWNERS group ... ; valid groups: [...]`. Fix
   by updating the `owner:` value to one of the listed
   groups (or by adding the new group to CODEOWNERS,
   which requires a separate platform-team PR per
   ADR-0015).

4. **End-to-end demo.** Run `make up && make demo-p6`.
   The demo exercises the full publish → load →
   execute → report → alert flow against the local
   substrate. If the demo fails, the runbook entry
   in [`refresh-failure-escalation.md`](refresh-failure-escalation.md)
   or [`orphan-run-remediation.md`](orphan-run-remediation.md)
   may apply depending on the failure mode.

5. **Code review.** Confirm the PR has both
   approvals: `@PLACEHOLDER-org/rules-authors` for the
   rule YAML and `@PLACEHOLDER-org/platform-team` for
   the `_owners.yaml` edit. CODEOWNERS routes both
   automatically per ADR-0015.

6. **Shared-substrate channel-collision check.** If
   qa and prod share a substrate workspace, verify
   the channel references in
   `_owners.yaml.entities.<new-entity>.channels` carry
   a qa-prefix (e.g., `slack:#qa-dq-<entity>`,
   `email:qa-oncall@example.com`). Deployments with
   separate substrates skip this check with a
   one-line justification in the PR description ("qa
   and prod use separate Slack workspaces — no
   collision possible").

When all six boxes pass, merge the PR. The next
`rules-vN.Y.Z` tag's qa-overlay manifest publish
includes the entity. The qa overlay's loader
refreshes (refresh cadence per
`DQ_LOADER_REFRESH_INTERVAL` in the qa env-config);
Tier 1 begins on the first qa run.

### 3.B Tier 1 → Tier 2 (test-soak → production)

Run through ADR-0040 §"Tier 1 → Tier 2 checklist" in
order. Use the SQL queries in §4 to verify each
criterion mechanically.

1. **Soak window completed.** Run the §4.A query.
   `soak_runs` must be ≥ 50 (or the per-entity floor
   declared in the promotion PR description), and the
   span (`soak_end - soak_start`) must be ≥ 7
   calendar days.

2. **No unresolved errors.** Run the §4.B query. Each
   row returned represents an `error`-class
   execution; for each row, confirm a root-cause +
   fix has shipped to `main` (link the commit hash in
   the PR description). If any unresolved error rows
   remain, address before promotion.

3. **Pass-rate ≥ 95%.** Run the §4.C query. The
   `pass_rate` value must be ≥ 0.95 (or the
   per-entity floor declared in the promotion PR
   description).

4. **Channel reachability.** Manually verify each
   channel reference in
   `_owners.yaml.entities.<entity>.channels` resolves
   to a real destination:
   - Slack: post a test message to the channel from
     the engine's webhook poster (or a manual
     curl-equivalent). Confirm receipt.
   - Email: send a test message to the address.
     Confirm receipt or non-bounce.
   - PagerDuty: trigger a low-severity test event
     against the service. Confirm receipt + dismiss.
   Record verification results (timestamp + reviewer
   handle) in the promotion PR description.

5. **Owner sign-off.** Confirm at least one member of
   the group declared in
   `_owners.yaml.entities.<entity>.owner` has approved
   the promotion PR with a "production-ready" comment.
   The comment is searchable; CI may grep for it
   automatically in a future B2 follow-up.

6. **Platform-team sign-off.** Confirm the
   prod-overlay change has platform-team + SRE
   approval. CODEOWNERS routes
   `/deploy/overlays/prod/` to both groups per
   ADR-0015 §3; the PR cannot merge without both
   approvals.

7. **Channel-rename to production names.** Only if
   Tier 0 → Tier 1 criterion 6 applied (shared
   substrate): confirm the promotion PR includes the
   `_owners.yaml` edit renaming qa-prefixed channel
   names back to production names. The edit is
   platform-team-reviewed per ADR-0015.

When all seven boxes pass, merge the prod promotion
PR. The next `rules-vN.Y.Z` tag's prod-overlay
manifest publish includes the entity. The prod
overlay's loader refreshes; Tier 2 begins on the
first prod run.

## 4. Verification queries

The queries below target the
`dq_executions_current` view and `dq_check_results`
table committed by
[ADR-0039](../adr/0039-dashboard-contract.md). Replace
`<new-entity>` with the entity identifier; replace
`<soak-start-utc>` with the qa-publish UTC timestamp
(e.g., `2026-05-19T10:00:00Z`).

### 4.A Soak window completed

```sql
SELECT COUNT(*) AS soak_runs,
       MIN(recorded_at) AS soak_start,
       MAX(recorded_at) AS soak_end,
       TIMESTAMP_DIFF(MAX(recorded_at),
                      MIN(recorded_at), DAY) AS soak_days
FROM dq_executions_current
WHERE entity = '<new-entity>'
  AND status = 'success'
  AND recorded_at >= TIMESTAMP('<soak-start-utc>')
```

Expected: `soak_runs >= 50` AND `soak_days >= 7`. The
per-entity floor (recorded in the PR description)
overrides 50 only above the floor; the 7-day floor is
unconditional.

### 4.B No unresolved errors

```sql
SELECT execution_id,
       attempt_id,
       recorded_at,
       error_summary,
       supersedes_execution_id
FROM dq_executions_current
WHERE entity = '<new-entity>'
  AND status = 'error'
  AND recorded_at >= TIMESTAMP('<soak-start-utc>')
ORDER BY recorded_at DESC
```

Expected: zero rows, OR every row has a documented
root-cause + fix shipped to `main` (link in PR).

### 4.C Pass-rate ≥ 95%

```sql
SELECT COUNTIF(result = 'pass') / COUNT(*) AS pass_rate,
       COUNT(*) AS total_evaluations
FROM dq_check_results
WHERE execution_id IN (
  SELECT execution_id
  FROM dq_executions_current
  WHERE entity = '<new-entity>'
    AND recorded_at >= TIMESTAMP('<soak-start-utc>')
)
```

Expected: `pass_rate >= 0.95` (or the per-entity floor
declared in the PR description).

## 5. Rollback / escape

A Tier 2 promotion that produces unexpected behavior
(prod-substrate alerts misfiring, dashboard panels
showing unexpected pass-rate drift, owner reporting
the entity behaves differently in prod than it did in
qa) escapes via the manifest-rollback procedure in
[`manifest-rollback.md`](manifest-rollback.md). The
short version: re-point the prod overlay's manifest
pointer back to the prior `rules-vN.Y.Z` that did
*not* include the entity, using `dq-manifest
set-pointer`. The entity is removed from Tier 2; it
remains at Tier 1 in qa for further soak.

If the unexpected behavior is the channel-rename
(Tier 1 → Tier 2 criterion 7) producing the wrong
post-rename channel name (e.g., the prod Slack
channel doesn't exist), revert the `_owners.yaml`
edit in a follow-up PR (CODEOWNERS-routed to
platform-team) and re-run criterion 4 (channel
reachability) before retrying promotion.

The append-only contract from
[ADR-0003](../adr/0003-result-write-model.md) means
"rollback" here is "publish a corrective manifest",
not "delete prior rows". The `dq_executions` rows
from the Tier 2 attempt remain in the table; the
`supersedes_execution_id` column on subsequent rows
preserves the lineage.

## 6. Escalation

- **Soak criteria ambiguous.** A per-entity threshold
  override above the committed floor is at the
  operator's discretion (record in PR); a threshold
  below the floor is an ADR amendment. Escalate to
  `@PLACEHOLDER-org/platform-team` if uncertain
  whether an override is appropriate.
- **Channel reachability fails.** The
  Slack/email/PagerDuty destination does not respond
  or returns errors. Escalate to the owner team to
  resolve the destination before promotion; do not
  proceed.
- **`dq_executions_current` view unavailable or
  stale.** The view is the canonical Tier 1 → Tier 2
  verification surface. If queries fail, escalate to
  `@PLACEHOLDER-org/platform-team` (likely a BigQuery
  or env-config issue, not an onboarding issue).
- **Owner sign-off blocker.** No member of the
  owner group approves within an operationally
  reasonable window. Escalate to the owner team's
  manager via the channels declared in
  `_owners.yaml.entities.<entity>.channels.operational`.
- **Platform-team review blocker.** The prod-overlay
  PR review stalls. Escalate via `@platform-team`
  Slack or the platform-team-on-call rotation. The
  prod-overlay CODEOWNERS requirement is binding
  (ADR-0015 §3); no workaround.
