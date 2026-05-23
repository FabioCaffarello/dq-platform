<!-- path: docs/runbooks/alert-dedup-debugging.md -->

# Runbook — Alert dedup debugging

Diagnose why an alert channel surfaces duplicates of the same
failing check, or why an expected alert is missing despite
engine logs showing event emission.

The dedup contract has two layers
([ADR-0006](../adr/0006-alert-routing-contract.md) §CC5):

- **Engine-side** suppresses literal duplicates within an
  attempt (`(execution_id, attempt_id, check_id, result)`).
  Implemented in `engine/internal/alerts/dedup.go`.
- **Consumer-side** enforces the platform-wide invariant of
  ≤1 user-visible alert per failing check across N retries,
  keyed on `(execution_id, check_id)` for check-level events
  (the `result` field is **excluded** from the key so a
  fluctuating check collapses to one alert).

Most "duplicate alert" incidents are consumer-side
misconfiguration; most "missing alert" incidents are either
the wrong dedup key or an upstream event-emission gap.

---

## 1. When to use

- An on-call reports duplicate alerts in the channel for the
  same `(execution_id, check_id)`.
- An on-call reports a missing alert despite Pub/Sub topic
  showing the event was published.
- A new entity onboarding produces unexpected alert
  cardinality (too many or too few).

## 2. Preconditions

- Pub/Sub subscription pull access for the engine's alert
  topic (per environment; see
  `engine/internal/env/{local,qa,prod}.go` for the
  topic identifiers).
- Read access to engine logs (the Pub/Sub publisher logs each
  emitted event with the structured fields per
  [ADR-0006](../adr/0006-alert-routing-contract.md) §4).
- Read access to the alerting consumer's dedup state (the
  consumer implementation is **TBD** per ADR-0006 OQ-4;
  until it lands, this runbook covers the engine-side path
  and the Pub/Sub surface; consumer-side debugging refers to
  the future consumer's own runbook).

## 3. Procedure

### 3.A Triage — is the duplicate at the engine or the consumer?

1. Pull the last N messages from the Pub/Sub topic:

   ```
   gcloud pubsub subscriptions pull \
     <subscription> \
     --auto-ack=false --limit=50 --format=json
   ```

2. For the suspected `(execution_id, check_id)`, count
   messages by `(execution_id, attempt_id, check_id, result)`
   (the engine-side dedup key).

3. **If the count is > 1 for the same 4-tuple** → engine-side
   dedup leaked. Go to 3.B.

   **If the count is 1 per 4-tuple but > 1 across different
   `attempt_id` values for the same `(execution_id,
   check_id)`** → consumer-side dedup leaked. Go to 3.C.

   **If the count is 1 total but no alert reached the
   channel** → consumer is dropping or the channel is
   misconfigured. Go to 3.D.

### 3.B Engine-side dedup leak

The `AttemptDeduper` is per-attempt
(`engine/internal/alerts/dedup.go`). Two causes:

- **Engine restart mid-attempt.** A new process starts with
  an empty deduper; a retry after restart re-emits. Confirm
  by correlating the dup timestamps with engine pod
  start/stop events (`kubectl get pods --watch` or the
  observability surface).
- **Code path emits outside the deduper.** Grep
  `engine/internal/alerts/` callsites of `Publish` for paths
  that bypass `dedup.Allow(...)`. Expected callsites:
  runner / loader / orphan-detector / trigger-handler.

Engine-restart-driven re-emission is **expected behavior**
per ADR-0006 §CC5 ("engine-side dedup is not the primary
enforcement of the ≤1-user-visible-alert invariant"); the
consumer-side dedup absorbs it. The fix is to verify the
consumer-side dedup is healthy, not to add cross-attempt
state to the engine.

### 3.C Consumer-side dedup leak

The consumer-side dedup key is `(execution_id, check_id)`
with `result` excluded (per ADR-0006 §CC5). Common bugs:

- Consumer keys on `(execution_id, check_id, result)`
  instead — `result` fluctuating across retries multiplies
  alerts.
- Consumer dedup window expired (default per the consumer's
  configured retention; B1-deferred). N retries spanning
  longer than the window means alerts pass through.

Fix in the consumer's implementation; not engine work.

### 3.D Missing alert (single event reached topic, no
channel surface)

- **Channel resolution table miss.** The engine emits a
  `(type, id)` pair per `_owners.yaml.channels`; the
  consumer's deployment config maps `(env, type, id) →
  concrete destination` (per ADR-0006 §3). A missing entry
  drops silently. Audit the resolution table for the
  affected entity's channel ids.
- **Severity threshold filter.** The consumer may filter
  events below a per-category severity threshold. Engine
  default severity per category is in the deployment config
  (per ADR-0006 §6); per-entity overrides in
  `_owners.yaml.severity_overrides`. If neither is set, the
  consumer applies its category default.

## 4. Verification

1. **Re-publish a test event** with a known
   `(execution_id, check_id)`. The simplest local check is
   `make demo-p6` (it triggers a known-good rule that emits
   one alert).
2. **Observe exactly one channel surface** for that test
   event. If 3.C was the cause, the consumer fix may need a
   replay of historical events; coordinate with the consumer
   team.
3. **No regression on production traffic.** Watch the next
   hour of alerts to confirm dup count returns to baseline.

## 5. Rollback / escape

- **Engine-side change rolled back.** If 3.B led to a code
  change in `engine/internal/alerts/`, the change ships
  through normal PR flow; rollback is a revert PR.
- **Consumer-side configuration rolled back.** Configuration
  is per the consumer; coordinate with that team.
- **Dedup state corrupted.** If consumer-side dedup state is
  in a cache (Redis, in-memory) and gets corrupted, the
  rollback is to flush it and accept a one-time burst of
  pent-up events. Coordinate timing with the on-call.

## 6. Escalation

- **Engine-side leak persists after 3.B fix attempt.**
  Escalate to platform-team — there may be a callsite the
  grep missed.
- **Consumer behavior is unspecified and the consumer team
  is the alerting-consumer-owner from ADR-0006 OQ-4.** The
  consumer implementation is **TBD**; until it lands, route
  ambiguity through platform-team and the entity's
  `_owners.yaml.owner`.
- **Pub/Sub subscription itself is offline.** Escalate to
  SRE.
