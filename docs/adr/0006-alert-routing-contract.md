<!-- path: docs/adr/0006-alert-routing-contract.md -->

# ADR-0006 — Alert Routing Contract

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

Every check fail/degraded, every check error, every
execution-level error or abort, and every operational signal
from the engine's loader/scheduler/orphan-detector must
reach an owner without an engine code path baking in
team-specific conditionals. The platform's anti-goal is
explicit: routing decisions are **data**, not engine code.

ADR-0004 fixed the category boundary (data quality vs.
operational). This ADR specifies where ownership and channel
information lives, what the engine emits as an alerting
event, and how deduplication works across two layers.

The contract surface has three sides — author-time
(`_owners.yaml`), deployment-time (engine config), and
runtime (Pub/Sub event payload). Splitting them is what
keeps team-specific decisions out of engine code.

---

## Decision

### 1. `_owners.yaml` schema

A file in the rules workspace declares ownership and routing
references per entity. Required and optional fields:

- **`schema_version`** (integer, top-level, required) —
  initial value `1`. Aligns with the DSL schema-versioning
  pattern.
- **`entities`** (map, top-level, required) — keyed by
  entity name. Each entity entry:
  - **`owner`** (string, required) — a team identifier
    matching a CODEOWNERS group.
  - **`channels`** (map, required) — keyed by alert
    category (`data_quality`, `operational`); each value is
    a list of channel-reference strings (below). At least
    one category must have a non-empty list.
  - **`severity_overrides`** (map, optional) — keyed by
    `check_id` (or the literal `default` to override the
    per-entity default); each value maps category to
    severity (`info` / `warning` / `critical`).
  - **`description`** (string, optional) — documentation
    only, not used in routing.

### 2. Channel-reference structure

Channel references are **(type, id) pairs**.

- `type` is a destination-class identifier matching a
  registered class in engine deployment config (`slack`,
  `pagerduty`, `email`, `webhook`, etc.). The linter
  validates that every `type` in `_owners.yaml` is
  registered.
- `id` is an environment-resolvable identifier (channel
  name, service identifier, alias). The `id` is what humans
  read in code review; the actual destination (webhook URL,
  service key, email address) lives in engine deployment
  config.

The on-wire encoding of the pair (colon-separated string vs.
YAML mapping) is a scaffolding detail.

### 3. Engine deployment config — committed surface

Owned per environment by the platform team plus SRE. Holds:

- Pub/Sub topic the engine publishes to (one per
  environment).
- Channel resolution table: maps `(environment, type, id)`
  → concrete destination. Secrets follow standard
  per-environment secret-management discipline.
- Default severity per category (`data_quality`,
  `operational`).
- Default consumer-side dedup window length.
- Per-environment overrides for any of the above.

Engine deployment config does **not** contain entity → owner
mappings, per-entity routing decisions, or per-team
conditionals.

### 4. Event payload — engine emits to Pub/Sub

Every alert-relevant engine action produces a structured
event:

*Identity fields:*

- `execution_id` (string, sha256-hex, **optional** —
  absent for engine-startup loader failures).
- `attempt_id` (string, UUID, optional — absent when
  `execution_id` is absent).
- `entity` (string, **required**) — matches `_owners.yaml`
  keys.
- `check_id` (string, optional — present for check-level
  events; absent for execution-level events).

*Routing fields:*

- `category` (string, enum:
  `data_quality` | `operational`, **required**).
- `severity` (string, optional, enum:
  `info` | `warning` | `critical`) — set when
  `_owners.yaml` has a matching override; otherwise
  omitted, and the consumer applies the category default
  from engine config.
- `event_source` (string, enum:
  `runner` | `loader` | `scheduler` | `orphan_detector` |
  `trigger_handler`, **required**).

*Status / result fields:*

- `result` (string, enum:
  `pass` | `fail` | `degraded` | `error`, optional —
  check-level only).
- `status` (string, enum:
  `running` | `success` | `failed` | `error` | `aborted`,
  optional — execution-level only).

*Context fields:*

- `recorded_at` (RFC 3339 UTC timestamp, second precision,
  **required**).
- `error_summary` (string, optional).

Consumers must tolerate unknown fields; additive payload
evolution does not break them.

### 5. Deduplication — two layers with distinct responsibilities

- **Engine-side literal-duplicate suppression.** The engine
  never emits the same
  `(execution_id, attempt_id, check_id, result)` tuple
  twice in a single attempt (e.g., due to emit retries).
  This is correctness against re-emission within an
  attempt, not the primary enforcement of the "≤1
  user-visible alert per failing check" invariant.
- **Consumer-side B0-2 I3 enforcement.** Alerting
  consumers dedup on receive within a configurable window
  (default in engine deployment config). Keys:
  - Check-level events: `(execution_id, check_id)`. The
    key **excludes `result`** so a check fluctuating
    across retries collapses to one user-visible alert per
    failing check.
  - Execution-level events with `execution_id`:
    `(execution_id, event_source)`. Different engine
    components reporting failures of the same execution
    are surfaced separately.
  - Loader-failure events with no `execution_id`:
    `(event_source, time-bucketed)`. Rare enough that
    precision is not load-bearing.

### 6. Severity — default plus override

- Default severity per category lives in engine deployment
  config (per environment).
- Per-entity / per-check overrides live in
  `_owners.yaml`'s `severity_overrides`.
- The engine reads `_owners.yaml` at manifest load (per
  loader path) and applies overrides at event emission. If
  no override matches, `severity` is omitted from the
  event and the consumer applies the category default.

### 7. Category mapping table

The engine sets the `category` field per:

| Engine signal                                  | category     |
|------------------------------------------------|--------------|
| check `result = fail` or `degraded`            | data_quality |
| check `result = error`                         | operational  |
| execution `status = error` (pre-check)         | operational  |
| execution `status = aborted`                   | operational  |
| loader failure                                 | operational  |
| scheduler reconciliation failure               | operational  |
| trigger-handler retry exhaustion               | operational  |
| orphan finalization                            | operational  |

The engine never deviates from this mapping. Routing has no
discretion to reassign categories.

### 8. Cross-entity correlation is consumer-side

The engine emits one event per `(entity, check)` failure
regardless of correlated activity across entities. The
engine does not collapse N entities erroring in the same
window into a single event; it does not suppress events for
"downstream" entities based on upstream failure.
Cross-entity correlation, if desired, is the alerting
consumer's policy.

### 9. Owner declaration mandatory at the linter

The linter (extended from the byte-equality CI gate in
ADR-0001) checks that every entity declared in a rule file
has a corresponding entry in `_owners.yaml`. An MR
introducing an entity without an `_owners.yaml` entry fails
CI and cannot land on `main`. "No alert without owner" is
enforced at author time.

### 10. Engine never embeds team-specific conditionals

All team-relevant routing decisions are reads from
`_owners.yaml` and engine deployment config. No engine code
path conditions on team name, channel destination, or
per-team severity. The category boundary (rule 7) is
permitted as a non-team-specific routing decision; the
channel-type discriminator (rule 2) is permitted as a
non-team-specific type discriminator. Everything beyond is
data.

---

## Consequences

1. **The contract surface for alerting consumers is the
   event payload + `_owners.yaml` + engine deployment
   config.** Consumers do not read `dq_executions` or
   `dq_check_results` for routing decisions; the routing
   decision is made from the event payload alone. Consumers
   may query the tables for forensic or escalation purposes,
   but routing-time operation is event-driven.

2. **Two-layer dedup serves two purposes.** Engine-side
   prevents emit-time re-emission; consumer-side enforces
   the platform-wide "≤1 user-visible alert per failing
   check across N retries" invariant.

3. **Result-value changes across retries do not multiply
   alerts.** A check fluctuating `fail` → `error` → `fail`
   across retries collapses to one user-visible alert.

4. **Loader-startup failures alert without an
   `execution_id`.** They are bucketed by
   `(event_source, time-bucketed)` consumer-side.

5. **`_owners.yaml` is on the path to "no alert without
   owner".** The linter check is the first enforcement
   point. Defense-in-depth at the publisher and loader
   layers is a follow-up; deeper enforcement requires
   extending ADR-0005 and ADR-0007 verification sets.

6. **The schema evolves under the contract-driven protocol.**
   The `schema_version` field on `_owners.yaml` follows the
   DSL schema-versioning pattern. Additive changes do not
   bump the version; breaking changes require a new
   `schema_version` and a documented migration path.

7. **Numeric parameters are deferred.** Default severity
   per category, consumer-side dedup window length,
   engine-side dedup state retention bound, and
   per-environment channel-resolution-table entries are
   follow-up parameters with per-environment values. This
   ADR commits shapes; the values are scaffolding.

8. **`_owners.yaml` distribution to consumers is an
   integration point.** How alerting consumers obtain the
   current `_owners.yaml` and the relevant channel
   resolution from engine deployment config (published
   artifact, Admin API endpoint, embedded config) is a
   scaffolding detail. Consumers may consume the same
   pointer + content-addressed pattern used for manifests.

9. **The category boundary is non-negotiable.** This ADR
   inherits the mapping from ADR-0004. Routing cannot
   reassign a check `error` as a data-quality alert or
   vice versa.

---

## Notes

- The channel-type registry (the set of supported `type`
  values beyond the illustrative `slack` / `pagerduty` /
  `email` / `webhook`) is extended additively as
  integrations are built.
- The alerting consumer's implementation shape (separate
  service, sidecar, function, or library) is a
  scaffolding detail. The contract surface is committed;
  the consumer is downstream.
- Owner-existence enforcement at the publisher and loader
  layers strengthens the invariant against out-of-band
  edits to `_owners.yaml` after linting; that is a
  follow-up beyond this ADR.
