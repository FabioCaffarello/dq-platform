<!-- path: docs/runbooks/manual-vs-operator-rerun.md -->

# Runbook — `trigger_source: manual` vs `trigger_source: operator-rerun`

When an operator triggers an evaluation outside the
scheduler, two `trigger_source` values exist:

- `manual` — an ad-hoc evaluation against any window. Goes
  through the public `/v1/trigger` endpoint per
  [ADR-0014](../adr/0014-trigger-handler-contract.md). Does
  **NOT** replace any prior execution.
- `operator-rerun` — an explicit re-evaluation of a PRIOR
  execution. Goes through the Admin API and carries
  `supersedes_execution_id` per
  [ADR-0002](../adr/0002-run-identity-and-idempotency.md) §4
  +
  [ADR-0003](../adr/0003-result-write-model.md) §5.
  Replaces the prior execution in the canonical view.

Picking the wrong one is rarely catastrophic — both produce
real `dq_executions` rows — but the auditability and the
downstream alert routing differ. This runbook walks the
operator to the right choice and flags the Admin API gap
that's still open at v1.

---

## 1. When to use

- An operator needs to run an evaluation outside the
  scheduler's cadence and isn't sure which `trigger_source`
  to set.
- An on-call wants to "replace" a failing or wrong result
  with a corrected one and is unsure whether to use manual
  (parallel) or operator-rerun (supersede).
- A reviewer is auditing a `dq_executions` table and finds
  multiple executions for the same `(entity, window)` and
  wants to understand whether they are reruns or parallel
  observations.

Do **not** use this runbook for:

- Scheduler-driven retries — they happen automatically per
  [ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)
  and produce additional `attempt_id` rows under the **same**
  `execution_id`. No operator action is needed.
- Backfilling a freshly-onboarded entity's historical
  windows — that is a series of `manual` triggers, one per
  window; see
  [`entity-onboarding.md`](entity-onboarding.md) §3 for the
  end-to-end onboarding flow.

## 2. Preconditions

- HTTP access to the engine's `/v1/trigger` endpoint
  (cluster-internal Service `dq-engine:8080` in the
  reference deployment per ADR-0014 §4). Operators in
  Kubernetes typically port-forward via
  `kubectl port-forward svc/dq-engine 8080:8080` or run the
  `curlimages/curl` reference pattern from
  `deploy/base/cronjob-scheduler.yaml`.
- Read access to `dq_executions` + `dq_check_results` for
  the affected env (BigQuery dataset
  `dq_results_{local,qa,prod}` per
  `engine/internal/env/{local,qa,prod}.go`).
- The prior execution's `execution_id` if the planned action
  is `operator-rerun`. The Admin API rerun endpoint
  mandatorily requires it per ADR-0002 §4 — there is no
  "rerun the latest" shorthand at v1.
- Awareness that the Admin API is **not yet implemented**
  at v1. The runbook documents the eventual surface; until
  the Admin API lands, operators wanting supersede
  semantics work around the gap per §3.C below.

## 3. Procedure

### 3.A Pick the right `trigger_source`

The decision is binary. Walk the question tree once:

1. **Is the goal to replace a prior execution's result in
   the canonical view (`dq_executions_current` per
   ADR-0003 §2)?**

   - Yes → `operator-rerun`. Continue to §3.C.
   - No → Continue.

2. **Is the goal to record a parallel, independent
   observation of the same window — for example, comparing
   what a new rule produces against what the old rule
   produced for the same window?**

   - Yes → `manual`. Continue to §3.B.

3. **Is the goal to evaluate a window the scheduler hasn't
   covered (back-dated, future, or a one-off ad-hoc range)?**

   - Yes → `manual`. Continue to §3.B.

If none of the three match, the action probably belongs in
a different runbook. Common confusions:

- **A check is failing and you want to retry it.** The
  scheduler retries automatically per ADR-0007 §3 within
  its retry budget; the retry produces a new `attempt_id`
  under the same `execution_id`. No operator action needed.
  If retries are exhausted and the underlying issue is
  fixed, then `operator-rerun` is the right call.
- **A rule's logic was wrong and prior results are now
  believed incorrect.** This is `operator-rerun` after the
  rule has been corrected via the standard PR → publish
  path per ADR-0005. The new execution carries the new
  ruleset_version + the corrected logic; the
  supersedes_execution_id pointer makes the lineage
  auditable.

### 3.B Manual trigger (`trigger_source: manual`)

Issue a POST against `/v1/trigger`:

```sh
curl -fsS -X POST \
  -H 'Content-Type: application/json' \
  -d '{
    "entity": "<entity>",
    "window_start": "<YYYY-MM-DDTHH:MM:SSZ>",
    "window_end": "<YYYY-MM-DDTHH:MM:SSZ>",
    "trigger_source": "manual"
  }' \
  http://dq-engine:8080/v1/trigger
```

The window-string format is RFC 3339 UTC with the literal
`Z` suffix; `+00:00` is rejected per ADR-0014 §2 to
preserve the execution_id formula's byte-equality.
`window_end` must be strictly after `window_start`.

The response carries the new `execution_id` (ADR-0002 §1
formula over the five inputs including
`trigger_source: manual`). This `execution_id` is
**different** from the scheduler's execution_id for the
same window (different `trigger_source` input to the hash)
— the two rows coexist in `dq_executions`. Neither
supersedes the other.

### 3.C Operator-rerun (`trigger_source: operator-rerun`)

> **Admin API not yet implemented at v1.** ADR-0002 §4 +
> ADR-0014 §"Consequences" item 3 reserve a separate Admin
> API path for this. The public `/v1/trigger` endpoint
> rejects `trigger_source: operator-rerun` with HTTP 400
> per `engine/internal/api/decoder.go`'s
> `ErrCodeInvalidTriggerSrc` check.
>
> Until the Admin API endpoint ships (B2 follow-up — no
> row registered yet at the time of this runbook's seed),
> use one of the **workarounds** below.

**Eventual call shape** (documented for when the endpoint
lands):

```sh
# Indicative only — replace with the actual Admin API path
# when it ships. The mandatory parameter is the prior
# execution's execution_id.
curl -fsS -X POST \
  -H 'Content-Type: application/json' \
  -H 'Authorization: <admin-credential>' \
  -d '{
    "execution_id": "<prior-execution-id>",
    "reason": "<operator-readable justification>"
  }' \
  http://dq-engine:8080/admin/v1/rerun
```

The Admin API resolves the prior execution's `(entity,
window_start, window_end)` from `dq_executions`, recomputes
the new `execution_id` with `trigger_source:
operator-rerun`, and writes the new `dq_executions` row's
first state-transition row with
`supersedes_execution_id = <prior-execution-id>` per
ADR-0003 §5.

**Workarounds until the Admin API lands:**

- **For a failed-then-fixed check.** Wait for the
  scheduler's next tick to produce a fresh execution under
  the new ruleset version. The original failed execution
  remains in the history; the new execution covers the next
  scheduled window. No supersede lineage is recorded, but
  the canonical view reflects the current truth from the
  next scheduled tick onward.
- **For an explicitly-failed result the operator wants to
  override.** Issue a `manual` trigger for the same window
  with the corrected ruleset deployed. The manual execution
  produces a parallel row (different execution_id);
  dashboard consumers that key on `(entity, window)` and
  pick the latest `recorded_at` see the manual row as the
  current state. The supersede lineage is missing —
  reviewers cross-referencing `dq_executions` and
  `manifest_hash` can reconstruct the intent.
- **For a chain of reruns auditability matters.** Document
  the rerun intent in an operator-readable log or ticket
  outside the platform until the Admin API lands. The
  platform's lineage column will be empty for these
  workaround rows.

### 3.D What the rows look like in `dq_executions`

Quick read-pattern for distinguishing the four trigger
sources after they've landed:

```sql
SELECT
  execution_id,
  trigger_source,
  attempt_id,
  recorded_at,
  status,
  supersedes_execution_id
FROM `${PROJECT}.${DATASET}.dq_executions`
WHERE entity = '<entity>'
  AND window_start = '<YYYY-MM-DDTHH:MM:SSZ>'
ORDER BY recorded_at ASC
```

Interpretation:

- **Multiple rows with the same `execution_id`** =
  one logical execution with multiple state-transitions
  (running → terminal) and possibly multiple `attempt_id`s
  (scheduler retries per ADR-0007 §3). The canonical view
  picks the latest `recorded_at` per ADR-0003 §2.
- **Multiple rows with different `execution_id` for the
  same `(entity, window)`** = parallel observations from
  different trigger sources. `trigger_source` distinguishes
  them. Neither supersedes the other.
- **A row with non-null `supersedes_execution_id`** =
  operator-rerun. The pointer's value is the
  `execution_id` of the prior execution being replaced.
  Self-join on this column to walk the rerun chain.

## 4. Verification

1. **The new execution row appears in `dq_executions`.**

   ```sql
   SELECT execution_id, trigger_source, recorded_at, status
   FROM `${PROJECT}.${DATASET}.dq_executions`
   WHERE execution_id = '<new-execution-id>'
   ORDER BY recorded_at ASC
   ```

   At least one row with the expected `trigger_source`
   (`manual` or `operator-rerun`).
2. **For `operator-rerun`: the supersede lineage is
   recorded.** The first state-transition row carries
   `supersedes_execution_id = <prior-execution-id>` per
   ADR-0003 §5; the prior execution's row in
   `dq_executions` is unchanged (append-only per ADR-0003
   CC1).
3. **For `manual`: the scheduled row is undisturbed.**
   Query for the prior `(entity, window)` and confirm the
   scheduler's execution_id is present alongside the new
   manual execution_id. Two distinct
   `(execution_id, attempt_id)` tuples; both terminal-state
   rows present.
4. **Alerts route correctly.** The alerting consumer
   dedups on `(execution_id, check_id)` per ADR-0006 CC5;
   a manual trigger produces a separate alert from the
   scheduler's because the execution_ids differ. An
   operator-rerun, once the Admin API lands, will route
   through the same dedup key as the new execution
   (replacing the prior alert in canonical state).

## 5. Rollback / escape

Both trigger paths write to `dq_executions` per the
**append-only** contract (ADR-0003 CC1). There is no
DELETE / UPDATE from engine code paths; both manual and
operator-rerun executions become permanent rows in the
history.

If a manual or operator-rerun trigger produced an
unexpected or wrong result:

- **The wrong result was a manual trigger.** It is
  recorded in history and visible in dashboards; it does
  **not** override the scheduler's result for the same
  window (different execution_id). No rollback action is
  required — the scheduler's row remains canonical for
  dashboards that group by `(entity, window)` modulo
  `trigger_source`. If the manual result is misleading in
  a specific dashboard, file a bug against the dashboard's
  query: it should be filtering on
  `trigger_source = 'scheduler'` for canonical-cadence
  visualizations.
- **The wrong result was an operator-rerun (once the
  Admin API ships).** Issue **another** operator-rerun
  against the wrong rerun's execution_id, producing a new
  supersede chain link. The chain is preserved; the
  canonical view picks the latest `recorded_at`.
- **The wrong inputs were typed into the trigger payload
  (entity / window typo).** The bad execution_id is a
  permanent row but it doesn't reference any real
  scheduled run (no `(entity, window)` collision). Note
  the bad execution_id in the operator log; future
  forensic queries should filter it out by
  `execution_id`.

## 6. Escalation

- **The Admin API endpoint shape isn't documented yet
  and an operator needs supersede semantics urgently.**
  This is the v1 gap. Escalate to **platform-team** with
  a description of the use case so the Admin API B2
  follow-up is prioritised; in the meantime, use the
  §3.C workarounds.
- **A `manual` trigger appears in `dq_executions` with
  `status = error` and the operator can't tell whether
  the engine accepted the trigger or rejected it.** The
  trigger handler returns 200 only on acceptance per
  ADR-0014 §3; a 400 / 500 response means no row was
  written. If a row exists with `status = error` it
  means the engine accepted the trigger and the run
  hit an error during evaluation. Follow the standard
  error-investigation path: read `evidence_summary.error`
  on the failing check row; check engine logs for the
  matching `execution_id`. Escalate per the failing
  check's owner from `rules/_owners.yaml`.
- **A `dq_executions` row carries a
  `supersedes_execution_id` that doesn't match any
  existing `execution_id`.** This is a data-integrity
  issue (ADR-0003 §5 commits the column as a self-
  reference). Escalate to **platform-team** — the
  pointer may have been set incorrectly by the future
  Admin API or by a manual SQL write outside the
  platform.

---

## Maturity disclaimer

This is a **seed**. The largest TBD is the **Admin API
endpoint itself**: ADR-0002 §4 + ADR-0014 §"Consequences"
item 3 commit the surface but no engine endpoint yet
implements it. Once the Admin API lands as a B2 follow-up
slice, §3.C is rewritten to point at the real path; the
workaround text becomes a "before Admin API" historical
note. Ops feedback during real incidents is the source of
truth for sharpening §3.A's decision tree.
