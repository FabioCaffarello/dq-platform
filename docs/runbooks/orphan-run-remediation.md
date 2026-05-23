<!-- path: docs/runbooks/orphan-run-remediation.md -->

# Runbook — Orphan-run remediation

Finalize a stuck `dq_executions` row whose terminal status
never landed because the engine attempt crashed (OOM,
container restart, network partition mid-write).

The orphan-detector
([ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)
§CC10/§CC11) handles this automatically: a periodic scan
finalizes rows whose `running` state has aged past the
detection threshold, by appending a follow-up row with the
same `(execution_id, attempt_id)`, `status = aborted`, and
the detector's own `engine_version`. This runbook covers two
cases the auto-detector cannot resolve:

- the orphan-detector itself is down or stuck;
- the operator needs to finalize a row immediately, not at
  the next detector tick.

---

## 1. When to use

- A `dq_executions` row remains at `status = running` past the
  orphan-detector threshold age (B1-2 — exact value TBD; rule
  of thumb until B1-2 lands: 10× the longest expected
  execution duration).
- A monitoring alert names "orphan_detector_lag" or "stuck
  running rows" as the trigger.
- An operator needs to free downstream consumers blocked on
  a specific stuck row immediately (faster than the detector's
  next scan tick).

Do **not** use this runbook for executions still actively
running. Confirm via engine logs (`runner: execution
<execution_id> attempt <attempt_id>` lines) before applying.

## 2. Preconditions

- BigQuery admin-SQL access to the project + dataset hosting
  `dq_executions` and `dq_check_results` (per
  [ADR-0003](../adr/0003-result-write-model.md) §CC9: engine
  code paths never UPDATE/DELETE; the correction path is
  always out-of-band).
- The stuck row's `(execution_id, attempt_id)`. Retrieve via:

  ```sql
  SELECT execution_id, attempt_id, recorded_at
  FROM `<project>.<dataset>.dq_executions_current`
  WHERE status = 'running'
    AND recorded_at < TIMESTAMP_SUB(CURRENT_TIMESTAMP(),
                                    INTERVAL <threshold-minutes> MINUTE)
  ```

- The engine version that should be recorded on the
  finalization row. For an orphan-detector-equivalent
  finalization, use the **operator's tool version**, not the
  original engine's (per ADR-0007 §CC11; this makes
  heterogeneous `engine_version` across an attempt's lifecycle
  observable per
  [ADR-0002](../adr/0002-run-identity-and-idempotency.md)
  §CC14).

## 3. Procedure

### 3.A Append a finalization row to `dq_executions`

A separate Admin API endpoint is **TBD** (no engine surface
today; per ADR-0003 §CC9 the correction path is out-of-band,
and the concrete Admin API is a Wave-3 follow-up). Until that
lands, use BigQuery admin SQL directly:

```sql
INSERT INTO `<project>.<dataset>.dq_executions` (
  execution_id, attempt_id, recorded_at, status,
  engine_version, ruleset_version, entity,
  window_start, window_end, trigger_source,
  supersedes_execution_id, error_summary
)
SELECT
  execution_id, attempt_id, CURRENT_TIMESTAMP(), 'aborted',
  '<operator-tool-version>',
  ruleset_version, entity,
  window_start, window_end, trigger_source,
  supersedes_execution_id,
  'OPERATOR_FINALIZED: stuck running row past threshold'
FROM `<project>.<dataset>.dq_executions`
WHERE execution_id = '<execution_id>'
  AND attempt_id = '<attempt_id>'
  AND status = 'running'
LIMIT 1
```

Notes:

- The `INSERT … SELECT` preserves `(execution_id, attempt_id,
  ruleset_version, entity, window_*, trigger_source)` from
  the prior `running` row so the canonical view per
  [ADR-0002](../adr/0002-run-identity-and-idempotency.md) §I4
  resolves correctly.
- The new `recorded_at` is `CURRENT_TIMESTAMP()`; this is
  what `dq_executions_current` will return as the terminal
  state for this `execution_id`.
- `error_summary` carries an operator marker — recommended so
  later forensic queries can distinguish operator-finalized
  rows from detector-finalized ones.

### 3.B (Optional) Append corresponding check-level rows

If the alerting consumer reads
`dq_check_results` directly for forensic context, also append
a row per check that was queued but never wrote a result:

```sql
-- See engine/internal/results/schema for the dq_check_results
-- column list; TBD: a single canonical INSERT template
-- shipped under tools/ is a Wave-3 follow-up.
```

Operator judgment: skip 3.B if the consumer only reads
`dq_executions_current`.

## 4. Verification

1. **Canonical view returns aborted.**

   ```sql
   SELECT status, engine_version, recorded_at
   FROM `<project>.<dataset>.dq_executions_current`
   WHERE execution_id = '<execution_id>'
   ```

   Returns one row, `status = aborted`, `engine_version =
   <operator-tool-version>`, `recorded_at` recent.

2. **Operational alert lands** (per
   [ADR-0006](../adr/0006-alert-routing-contract.md) §CC4 —
   `status = aborted` maps to category `operational`).
   Confirm in the alerting consumer's channel for the
   affected entity.

3. **Downstream consumers unblocked.** Re-query whatever
   surface was blocked (a dashboard, a scheduler, a manual
   re-trigger). The new terminal row is the canonical one
   from this `execution_id` forward.

## 5. Rollback / escape

`dq_executions` is append-only (per ADR-0003 §CC1). To
"undo" an operator finalization, append a **second**
finalization row with a fresh `recorded_at` and a status of
your choice (typically you would not re-finalize as
`running` — that confuses the canonical view; you would
re-finalize as `aborted` with a corrective `error_summary`
explaining the prior finalization was in error).

If 3.A inserted against the wrong `(execution_id,
attempt_id)`:

- Append a corrective row with `error_summary =
  'OPERATOR_CORRECTION: prior aborted finalization in error;
  see <ticket>'`.
- The canonical view still returns the latest row; the
  forensic trail in `dq_executions` retains both finalization
  attempts plus the original `running` row.

## 6. Escalation

- **`dq_executions` write IAM denies the INSERT.** Escalate
  to SRE.
- **The orphan-detector itself is down.** Find why
  (`dq-engine` pod logs; CrashLoopBackoff?); a single
  detector down is not normally an emergency since the
  detector is idempotent — once revived, it finalizes the
  backlog. Escalate to platform-team if the detector won't
  start.
- **Many stuck rows accumulating faster than this runbook
  can finalize.** The engine itself is misbehaving (crashing
  mid-execution at rate). Escalate to platform-team; a
  blanket orphan-finalization via a one-shot script
  (operator-written; not in tools/ today — Wave-3 follow-up)
  may be needed.
