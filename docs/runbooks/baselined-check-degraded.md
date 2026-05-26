<!-- path: docs/runbooks/baselined-check-degraded.md -->

# Runbook — Baselined check fires `degraded` with `insufficient_baseline_samples`

A baselined check (a kind whose rule carries
`params.baseline`) returned
`result: degraded` and
`evidence_summary.reason: insufficient_baseline_samples`.
The engine did **not** compare the current value against the
baseline — it found fewer historical `pass` rows than the
rule's `min_samples` threshold and stopped short
per [ADR-0032](../adr/0032-baseline-strategy.md)'s
sparse-history policy.

This runbook diagnoses **why** the history was sparse and
guides the operator to the right remediation. `degraded` is
a **soft** signal here: the check is not in failure; it is
saying "I cannot answer this question yet". Most causes
resolve themselves with time; a minority require a rule edit.

---

## 1. When to use

- An on-call alert names a baselined kind (e.g.,
  `set.row_count_within_baseline`) and the alert payload
  carries `reason: "insufficient_baseline_samples"`.
- A dashboard panel shows a baselined check in the
  `degraded` band with the same reason.
- An operator inspecting `dq_check_results` directly sees
  rows with
  `result = 'degraded'`
  and a JSON `evidence_summary.reason` of
  `insufficient_baseline_samples`.

Do **not** use this runbook for:

- A baselined check that returns `fail` — that means the
  baseline was computed and the current value exceeded the
  tolerance band. Investigate the deviation against the
  rule's `tolerance` block; this runbook does not cover
  fail-path remediation.
- A baselined check that returns `error` — the engine
  could not run the baseline query at all (auth, results-
  table missing, etc.). Follow
  [`refresh-failure-escalation.md`](refresh-failure-escalation.md)
  or the equivalent results-table runbook based on the
  `evidence_summary.error` field.
- A non-baselined check (`set.row_count_positive`, etc.)
  that fires `degraded`. Other kinds have their own
  degraded paths; this runbook is scoped to the baselined
  sparse-history reason only.

## 2. Preconditions

- Read access to engine logs and to the `dq_check_results`
  table for the affected env (qa or prod). The
  `evidence_summary` JSON is the source of truth for the
  diagnosis.
- Read access to the rule YAML in `rules/<entity>.yaml` to
  inspect the `params.baseline` block.
- Read access to the per-env `EvidenceRetention.ResultsRetention`
  constant (30d / 90d / 365d for local / qa / prod per
  ADR-0031); see `engine/internal/env/{local,qa,prod}.go`.
- Awareness that **a rule edit re-enters the manifest
  publication path** ([ADR-0005](../adr/0005-manifest-publication-semantics.md)).
  Edits land via the standard PR → lint → dry-run → publish
  flow; no out-of-band engine restart is needed.

## 3. Procedure

### 3.A Read the evidence

Pull the most recent degraded row for the affected
`(entity, check_id)`:

```sql
SELECT
  execution_id,
  attempt_id,
  executed_at,
  result,
  evidence_summary
FROM `${PROJECT}.${DATASET}.dq_check_results`
WHERE check_id = '<check_id>'
  AND result = 'degraded'
ORDER BY executed_at DESC
LIMIT 1
```

The `evidence_summary` JSON carries the four fields the
diagnosis depends on:

| Field | What it tells you |
|---|---|
| `samples_used` | How many `pass` rows the baseline query actually found in the effective reference window. |
| `min_samples` | The rule-declared threshold; the gap (`min_samples - samples_used`) is the size of the deficit. |
| `effective_reference_window` | The window the query actually covered, after the per-env `min(declared, ResultsRetention)` cap. |
| `kind` | The baselined kind (used to locate the rule). |

### 3.B Classify by cause

The four causes below are ordered by likelihood (highest
first). Walk them in order and pick the first that matches.

#### Cause 1 — New check, history hasn't accumulated yet

**Signal:** The check was recently introduced (rule landed
in the last `effective_reference_window`). `samples_used`
will be low because there simply aren't that many past
`pass` runs yet.

**How to confirm:**

```sql
SELECT MIN(executed_at) AS first_run
FROM `${PROJECT}.${DATASET}.dq_check_results`
WHERE check_id = '<check_id>'
```

If `first_run` is recent relative to
`effective_reference_window`, the check is in its warmup
period.

**Remediation:** **Wait.** The check will start returning
`pass` / `fail` once enough history accumulates. If the
operator needs an interim signal, switch the rule to
`source: static` with an operator-declared baseline value
and re-publish; once history accumulates, switch back to
`platform_history`.

#### Cause 2 — Declared reference window exceeds env retention

**Signal:** `effective_reference_window` in
`evidence_summary` is **shorter** than what the rule
declared. The runtime capped the query at the env's
`ResultsRetention` per ADR-0032 §"Effective reference
window vs declared".

**How to confirm:** Compare `effective_reference_window` in
`evidence_summary` against the `reference_window` field in
the rule YAML. If they differ, the cap fired.

**Remediation:** Two options, by env:

- **Local dev (30-day retention).** Shorten the rule's
  `reference_window` to a value the local dataset can
  actually accumulate (e.g., `7d`), or accept that the
  baselined check will run in degraded mode locally and
  only validate in qa / prod. This is the default posture
  per ADR-0032 — degraded is a signal, not an incident,
  in low-retention envs.
- **qa / prod.** Either shorten `reference_window` to fit
  the env's retention, or raise the env's
  `ResultsRetention` via the env-config change path
  (rare; ADR-0031 commits the per-env values and amending
  them requires a coordinated platform decision). Do not
  raise retention as a workaround for a single check —
  the cost-discipline trade-off is platform-wide.

#### Cause 3 — `min_samples` set too high for the check's cadence

**Signal:** `effective_reference_window` matches the
declared `reference_window` (no cap fired), but
`samples_used` is below `min_samples` because the check
runs infrequently (e.g., a weekly check with a 30-day
window and `min_samples: 30` will never satisfy itself —
at most ~4 runs land per window).

**How to confirm:** Count distinct `executed_at` days for
the check over the reference window:

```sql
SELECT COUNT(DISTINCT DATE(executed_at)) AS distinct_days
FROM `${PROJECT}.${DATASET}.dq_check_results`
WHERE check_id = '<check_id>'
  AND result = 'pass'
  AND executed_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(),
                                   INTERVAL <ref_window_days> DAY)
```

If `distinct_days` is comfortably above `min_samples`
**but** `samples_used` (which counts `pass` rows only) is
below it, the gap is the next cause (#4). If
`distinct_days` itself is below `min_samples`, the check's
schedule cannot produce enough samples — this is cause #3.

**Remediation:** Lower `min_samples` in the rule's
`params.baseline` to a value the schedule can actually
satisfy (rule of thumb: ≤ 50% of the expected sample count
in the window). The default of `5` is calibrated for
daily-cadence checks against a 7-day window; weekly or
monthly checks need a higher window or a lower threshold.

#### Cause 4 — Recent `fail` / `error` history shrank the `pass` set

**Signal:** `samples_used` is well below the run-count for
the check over the window — there are runs, but not enough
`pass` runs (the baseline query reads only `pass` rows per
ADR-0032 §"Baseline query (platform-history mode)").

**How to confirm:** Break down recent runs by result:

```sql
SELECT result, COUNT(*) AS count
FROM `${PROJECT}.${DATASET}.dq_check_results`
WHERE check_id = '<check_id>'
  AND executed_at >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(),
                                   INTERVAL <ref_window_days> DAY)
GROUP BY result
```

If `fail` or `error` dominates the breakdown, the check is
itself unhealthy — fix the underlying cause first; the
baseline degraded signal will resolve as `pass` history
accumulates again.

**Remediation:** Treat this as a fail / error incident on
the same check. The `degraded` signal here is downstream
of the real problem and resolves once enough `pass` rows
exist again. Do not edit the rule to silence the degraded
signal while the check is genuinely unhealthy.

### 3.C Apply the remediation

Whichever cause matched, the remediation is one of:

- **Wait** (cause #1) — no action; the check warms up.
- **Edit the rule** (causes #2 / #3) — adjust
  `reference_window`, `min_samples`, or temporarily swap
  to `source: static`. Edits flow through the normal PR →
  lint → dry-run → publish path
  ([ADR-0005](../adr/0005-manifest-publication-semantics.md));
  no engine restart needed.
- **Fix the underlying check** (cause #4) — investigate
  the `fail` / `error` rows on their own merits.

## 4. Verification

1. **Next scheduled execution returns `pass` or `fail`** (not
   `degraded` with the same reason). Confirm by reading
   `dq_check_results` for an `executed_at` strictly after
   the rule edit or fix landed:

   ```sql
   SELECT result, evidence_summary
   FROM `${PROJECT}.${DATASET}.dq_check_results`
   WHERE check_id = '<check_id>'
     AND executed_at > '<rule-edit-timestamp>'
   ORDER BY executed_at DESC
   LIMIT 1
   ```

2. **`samples_used >= min_samples`** on the next degraded /
   pass / fail row (read from `evidence_summary`). If still
   below, the remediation didn't take effect — revisit the
   classification in step 3.B.
3. **No alert refires on the next refresh tick.** The
   degraded signal routes through the data-quality alert
   category per ADR-0006 CC7; once the check exits the
   degraded band, deduplication suppresses the next event.

## 5. Rollback / escape

The degraded signal is itself the safe state — the engine
declined to compare against an unreliable baseline rather
than producing a misleading `pass` or `fail`. There is
**nothing destructive to roll back**.

If a rule edit (cause #2 or #3 remediation) made things
worse — for example, lowering `min_samples` to a value so
low that the baseline became statistically meaningless and
the check started flagging false-fails — revert the rule
edit through the same PR → publish path. The append-only
contract on `dq_check_results` ([ADR-0003](../adr/0003-result-write-model.md))
means past degraded rows stay in the history; only the
future runs see the reverted rule.

A temporary switch to `source: static` (cause #1) is a
deliberate degraded-tolerance posture: the operator
accepts that the baseline is operator-declared, not
historically-derived, for the warmup window. Reverting the
switch is the same PR-driven path once history
accumulates.

## 6. Escalation

- **Cause #2 in qa / prod, and the rule's
  `reference_window` is already at the minimum useful
  value for the check.** This is a tension between
  per-env retention (ADR-0031) and the rule's signal
  needs. Escalate to **platform-team** for the
  retention-amendment discussion; a per-rule workaround
  (e.g., switching that one rule to `source: static`)
  may bridge until the retention decision lands.
- **Cause #4 persists past the next refresh tick.** The
  underlying check is in sustained failure and the
  degraded signal is masking the real incident. Escalate
  to the **entity owner** (per `rules/_owners.yaml`) and
  treat the check's `fail` / `error` rows as the
  primary incident.
- **The runtime reports `effective_reference_window: 0`**
  in `evidence_summary`. The engine could not determine
  the env's `ResultsRetention` — likely a
  `EnvConfig.EvidenceRetention.ResultsRetention` wiring
  gap. Escalate to **platform-team**; the engine should
  fail loudly when retention is unconfigured.
- **`samples_used` is suspicious (e.g., 0 when there is
  clearly history in `dq_check_results`).** The baseline
  query may be reading the wrong dataset
  (`eval.Config.ResultsProject`/`ResultsDataset`) or the
  per-(execution_id, attempt_id) join is excluding rows
  it should include. Escalate to **platform-team**; this
  is a baseline-query bug, not an operator issue.

---

## Maturity disclaimer

This is a **seed**. It covers the four causes the design
anticipated; ops feedback during real incidents will
sharpen the diagnosis steps + add cause categories that
emerge from practice. The `tolerance: stddev` path is
**not** covered here because the v1 implementation in
`engine/internal/eval/baselines.go` returns `0` allowance
for stddev (per the v1 placeholder note in
`allowedDeviation`); a future B2 row implements stddev
properly and amends this runbook.
