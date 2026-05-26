<!-- path: docs/runbooks/README.md -->

# Runbooks

Operator-facing playbooks for the common DQ Platform incidents.
Each runbook follows a fixed shape — **when to use →
preconditions → procedure → verification → rollback →
escalation** — so operators can pattern-match across
incidents without re-reading the structure each time.

These runbooks are **seeds**: they cover the four incident
classes named in the W3-P8d phase row, with enough concrete
detail that an on-call operator can execute them, and explicit
TBD markers wherever ops feedback is needed before they are
production-ready.

---

## When to use which runbook

| Signal | Runbook |
|---|---|
| A published manifest causes incorrect / over-firing checks; the prior manifest was known good. | [`manifest-rollback.md`](manifest-rollback.md) |
| `dq_executions` rows stuck in `status = running` past the orphan-detector threshold, or the orphan-detector itself crashed. | [`orphan-run-remediation.md`](orphan-run-remediation.md) |
| Duplicate alerts in the channel for the same failing check, or missing alerts despite engine logs showing event emission. | [`alert-dedup-debugging.md`](alert-dedup-debugging.md) |
| Loader refuse-swap fires repeatedly; refresh fails N times consecutively (N per B1-2). | [`refresh-failure-escalation.md`](refresh-failure-escalation.md) |
| A new entity needs onboarding through the three-tier readiness model (Candidate → Test-soak → Production) per ADR-0040. | [`entity-onboarding.md`](entity-onboarding.md) |
| An existing deployment has accumulated non-partitioned `dq_executions` / `dq_check_results` tables from before ADR-0031's partitioning posture took effect. | [`results-partition-migration.md`](results-partition-migration.md) |
| A baselined check (e.g. `set.row_count_within_baseline`) returns `result: degraded` with `evidence_summary.reason: insufficient_baseline_samples`. | [`baselined-check-degraded.md`](baselined-check-degraded.md) |
| An operator needs to trigger an evaluation outside the scheduler and is unsure whether to use `trigger_source: manual` (ad-hoc; parallel observation) or `trigger_source: operator-rerun` (replace a prior result; carries `supersedes_execution_id`). | [`manual-vs-operator-rerun.md`](manual-vs-operator-rerun.md) |

## Anatomy of a runbook

Every runbook in this directory uses the same six sections:

1. **When to use** — the signal that should trigger this
   runbook. If your signal doesn't match the description, use
   the table above to pick a different one before proceeding.
2. **Preconditions** — accesses, tools, and current-state
   assumptions the procedure depends on. If any are absent,
   stop and acquire them rather than improvising.
3. **Procedure** — numbered steps with concrete commands. Each
   step is self-contained; do not skip steps even if the
   intermediate state looks fine.
4. **Verification** — how to confirm the remediation succeeded.
   Do not close the incident before verification passes.
5. **Rollback / escape** — how to undo the procedure if it
   makes the incident worse. Append-only contracts (per
   [ADR-0003](../adr/0003-result-write-model.md)) mean
   "rollback" usually means "write a corrective row", not "undo
   the prior row".
6. **Escalation** — when to involve a human, and which team.
   Group identifiers reference the CODEOWNERS placeholders in
   [`/.github/CODEOWNERS`](../../.github/CODEOWNERS); see
   [`../governance.md`](../governance.md) §2.

## Maturity disclaimer

These are seeds, not production-ready playbooks. Each runbook
flags TBD markers where:

- a B1 numeric parameter is unresolved (e.g., refresh-failure
  thresholds per B1-2);
- a CLI subcommand the procedure would prefer does not exist
  yet (and a workaround is recorded);
- an integration with an external system (PagerDuty, on-call
  rotation registry) is deferred until the integration lands.

Ops feedback during real incidents is the source of truth for
maturing these runbooks. Open a PR with revisions; CODEOWNERS
routes the review to platform-team.
