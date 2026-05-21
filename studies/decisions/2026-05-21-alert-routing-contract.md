<!-- path: studies/decisions/2026-05-21-alert-routing-contract.md -->

# B0-6 — Alert Routing Contract

## Metadata

- B0 reference: B0-6 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 8 — the final B0).
- Last updated: 2026-05-21.
- Upstream resolved: B0-4
  ([`2026-05-20-failure-scope.md`](./2026-05-20-failure-scope.md))
  is the direct upstream. Indirect inputs from B0-2
  ([`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md)),
  B0-3
  ([`2026-05-20-result-write-model.md`](./2026-05-20-result-write-model.md)),
  and B0-7
  ([`2026-05-20-loader-scheduler-retry-failure-semantics.md`](./2026-05-20-loader-scheduler-retry-failure-semantics.md)).
- Downstream open: none (this is the last open B0; resolving it
  closes the Wave-1 gate at 7 of 7).
- Promotion target: see final section.
- Wave 1 closure: B0-6 is the seventh and final B0 of Wave 1.
  With B0-1, B0-2, B0-3, B0-4, B0-5, B0-7, and B0-6 all at
  `resolved-study`, the Wave 1 gate is met upon acceptance.
  Wave 2 (consolidated platform decisions) and Wave 3
  (scaffolding) follow.

---

## Context

When a check fails, an entity errors, or the engine emits any of
the operational signals B0-7 enumerates, **someone needs to know**.
Without an explicit routing contract, every alert becomes a
case-by-case decision: which team gets paged, which channel
receives it, how often, deduplicated against what. Foundation doc
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
§"Layer 5 — Alerting" already commits the high-level architecture:
the engine emits events to Pub/Sub; alerting consumers fan out to
destinations based on `_owners.yaml`. What B0-6 must lock is the
**schema** of that data flow.

Six prior B0s already constrain B0-6's design space:

- **B0-4 CC7** locks the **category boundary** (data quality vs.
  operational) as non-negotiable. B0-6 routes within categories,
  not across.
- **B0-2 I3 + CC11** locks the **dedup key** as `execution_id`.
  N scheduler retries of the same `execution_id` produce at most
  one user-visible alert per failing check.
- **B0-4 CC7** locks the **per-check fan-out** shape for execution
  `failed`: each check's `result` value produces its own alert;
  the execution-level row triggers no separate alert.
- **B0-7** produces operational alerts from seven sources
  (CC1/CC2 loader, CC4 scheduler reconciliation, CC6
  trigger-handler retry exhaustion, CC8 pre-check entity error,
  CC10 `status = aborted`, CC11 orphan finalization).
- **B0-4 OQ-4** explicitly deferred **severity within category**
  (info / warning / critical) to B0-6.
- **B0-4 OQ-6** explicitly deferred **cross-entity correlation**
  (whether N entities erroring in the same window collapse to
  one incident or fan out) to B0-6.

Foundation doc 02 §"Ownership Boundaries" locks `/rules/_owners.yaml`
as platform-team-reviewed (the schema and central infrastructure
files); individual entity entries are domain-team-reviewed per
CODEOWNERS path patterns.

Foundation doc 04 §"Layer 5" also commits an **anti-goal**: the
engine never embeds team-specific conditionals. Routing must be
data-driven; the routing data lives in `_owners.yaml`.

B0-6 — as recorded in the decision log:

> What fields live in `_owners.yaml`, what stays in engine
> config, and what is deduplicated on the data itself?

This study locks:

1. **`_owners.yaml` schema** — required and optional fields per
   entity; the owner-declaration shape.
2. **Engine config vs. `_owners.yaml` split** — which fields are
   environment-level infrastructure (Pub/Sub topic, channel-ID
   resolution, dedup defaults) and which are entity-level
   ownership (owner, channel references, severity overrides).
3. **Event payload schema** — what the engine emits to Pub/Sub
   for each failure category.
4. **Dedup mechanism** — engine-side idempotent emission, plus
   consumer-side dedup as defense in depth; the dedup key
   composition; the dedup window shape.
5. **Severity within category** (B0-4 OQ-4 resolved) — default
   severity per category + per-entity / per-check override
   mechanism.
6. **Cross-entity correlation** (B0-4 OQ-6 resolved) — fan-out
   per (entity, check) is the default; correlation/collapsing
   is consumer-side, not engine-side.

What this study does **not** decide:

- The specific alerting consumer implementation — Wave 3.
- The concrete Slack channel / PagerDuty service / email
  destinations — these live in engine deployment config, per
  environment.
- Numeric parameters for dedup windows, severity escalation
  cadences, etc. — B1.
- The Pub/Sub topic naming convention — Wave 3 infrastructure.
- Per-team operational practices (on-call rotations, escalation
  trees) — these are downstream of channel resolution.

The decision matters because every prior B0 has produced an alert
source; B0-6 is where those signals reach humans. A routing
contract that's hardcoded, ambiguous, or partial means the
platform's promise to "make data quality posture visible, owned,
and operationally actionable" (foundation doc
[`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md))
collapses at the last mile.

---

## Decision Drivers

The decision must satisfy the following, in priority order.

1. **D1. Data-driven routing (foundation doc 04 §Layer 5).** No
   team-specific conditionals in engine code. Routing decisions
   are read from data, not encoded in branches.

2. **D2. Explicit ownership (P3).** Every entity has a declared
   owner. An entity without an owner cannot be activated by the
   manifest (CI gate via the linter, B0-1 surface). No alert
   without an owner declared.

3. **D3. Category boundary unbreakable (B0-4 CC7).** B0-6 routes
   within categories (data quality, operational). The boundary
   between them is not B0-6's to renegotiate; a check `error`
   never routes as a data-quality alert.

4. **D4. Dedup on `execution_id` (B0-2 I3 + CC11).** The dedup
   key is locked; B0-6 picks the dedup window and the
   composition of additional dedup fields (per-check), not the
   primary key.

5. **D5. Per-environment infrastructure (P4 + foundation doc
   05).** Infrastructure config — Pub/Sub topic names, channel
   ID → destination mappings, deployment-time defaults — lives
   in engine deployment config per environment. Entity-level
   ownership lives in `rules/`.

6. **D6. Domain teams shouldn't manage infrastructure
   secrets** (P5 + indirection). Webhook URLs, service IDs,
   on-call rotation keys rotate frequently and are
   environment-specific; domain teams declaring them inline in
   `_owners.yaml` produces credential leakage and per-environment
   `rules/` divergence. **`_owners.yaml` carries channel
   *references*; engine deployment config resolves them.**

7. **D7. Severity-within-category default + override
   (B0-4 OQ-4).** Default severity per category should be
   stateable in engine config; per-entity or per-check overrides
   in `_owners.yaml`. B0-6 commits the shape; B1 picks numeric
   defaults.

8. **D8. No-alert anti-goal (foundation doc 04).** An entity
   missing an owner declaration produces a hard load-time
   failure, not silent default routing.

9. **D9. Cross-entity correlation is downstream (B0-4 OQ-6).**
   The engine emits per (entity, check) events; whether and how
   to collapse correlated events into incidents is an
   alerting-consumer concern, not engine logic.

10. **D10. Storage-of-record stays in `dq_executions` /
    `dq_check_results` (B0-3).** The Pub/Sub event stream is a
    notification mechanism, not a system of record. Alerting
    consumers may persist their own state (suppression history,
    delivery audit) but the **canonical** record is the BigQuery
    tables.

---

## Considered Options

Each option specifies **what lives in `_owners.yaml`** and
**what lives in engine deployment config**. The architecture
(engine emits Pub/Sub events; consumers fan out) is held
constant — it's already locked by foundation doc 04 §"Layer 5".

### Option A — `_owners.yaml` carries everything (entity + destinations)

`_owners.yaml` carries entity owner, full destination details
(webhook URLs, service IDs, channel names), dedup window per
entity, severity overrides. Engine config only carries the
Pub/Sub topic name; everything else is in `rules/`.

```yaml
# Illustrative; not committed
entities:
  customer:
    owner: customer-team
    destinations:
      - type: slack
        webhook: https://hooks.slack.com/T0/B1/xyz
      - type: pagerduty
        service_key: secret-xyz
    dedup_window_minutes: 15
```

**Trade-offs.**

- Pro: simplest mental model — one file, all routing.
- Pro: domain teams have full control over their alerts.
- Con: violates D5 — infrastructure config (webhooks, service
  keys) lives in `rules/`, which is environment-agnostic.
  Multi-environment deployments would need a `rules/_owners.qa.yaml`
  / `rules/_owners.prod.yaml` split or env-aware substitution.
- Con: violates D6 — secrets (webhook URLs, service keys) in
  `rules/` means CI logs leak them, code reviews expose them,
  rotation is a YAML edit.
- Con: violates principle of separation between
  authoring (rules workspace) and infrastructure (deployment).

Reject on D5 and D6.

### Option B — `_owners.yaml` carries owner + channel references only

`_owners.yaml` carries entity owner identifier and a list of
**channel references** (opaque strings like `slack:customer-alerts`
or `pagerduty:customer-oncall`). Engine deployment config
resolves channel references to actual destinations
(webhook URLs, service IDs) per environment. Dedup window,
severity defaults, and Pub/Sub topic all live in engine
deployment config.

```yaml
# Illustrative; not committed verbatim
entities:
  customer:
    owner: customer-team           # matches CODEOWNERS group
    channels:                      # opaque channel references
      data_quality:
        - slack:customer-alerts
      operational:
        - slack:customer-alerts
        - pagerduty:customer-oncall
    severity_overrides:            # optional, per-check-id
      partition_freshness:
        data_quality: critical
```

**Trade-offs.**

- Pro: D5 + D6 satisfied — secrets and infrastructure stay in
  deployment config; `rules/` is environment-agnostic.
- Pro: D2 + D7 + D8 satisfied — owner is required; severity
  overrides are optional; missing owner fails at load (lint
  rejects, B0-1 surface).
- Pro: D9 satisfied — `_owners.yaml` doesn't try to encode
  cross-entity correlation; that's a consumer concern.
- Pro: per-environment channel resolution makes the same
  `_owners.yaml` work in `local`, `qa`, `prod` without
  divergence — only deployment config changes per environment.
- Con: indirection — a reviewer reading `_owners.yaml` sees
  channel references but not destinations. They must consult
  engine deployment config to know "who actually gets paged".
  Mitigated by an Admin-API endpoint (Wave 3) that resolves
  references for the current environment.
- Con: channel-reference namespace is a new contract surface
  (the format `<type>:<id>`); has to be documented and stable.

### Option C — `_owners.yaml` carries owner only; everything else in engine config

`_owners.yaml` carries just `entity → owner`. The
owner-to-channel mapping, severity overrides, dedup windows —
all in engine deployment config.

**Trade-offs.**

- Pro: simplest `_owners.yaml`; domain teams just declare who
  owns what.
- Pro: D5 + D6 maximally satisfied — `_owners.yaml` contains
  zero infrastructure detail.
- Con: violates D7 — severity overrides per check (which are
  inherently entity-knowledge, not infrastructure) end up in
  deployment config, where domain teams can't reasonably
  contribute. Or severity tuning becomes an Admin API endpoint
  the platform team operates on behalf of domain teams.
- Con: violates the explicit-ownership principle (P3) at the
  channel layer — `_owners.yaml` declares "this team owns this
  entity" but is silent on "which channels they want alerts
  on". The mapping from owner-team to channels lives in engine
  config, which is platform-team-owned; domain teams cannot
  declare their alerting preferences in their own workspace.
- Con (structural): severity per check is **knowledge held by
  the rule author, not the operator** — the author declared
  the check (its DSL shape, its threshold) and knows its
  operational sensitivity. Routing severity through engine
  config (operator territory) separates the knowledge from
  the configuration; severity tuning should round-trip in the
  same review process as the rule itself. Option B's
  `_owners.yaml` placement keeps severity adjacent to the
  rule-authorship surface.

Reject on D7, the workspace-ownership principle, and the
structural separation between rule-authorship knowledge
(domain) and infrastructure operation (platform).

### Option D — Minimal `_owners.yaml` + external routing service

`_owners.yaml` carries entity → owner only. An external
routing service (a separate platform component, possibly Wave 3
+ beyond) consumes the Pub/Sub event stream, looks up owner →
routing-policy in its own state store, and delivers.

**Trade-offs.**

- Pro: maximally decoupled — alerting policy is independently
  versioned, deployed, tuned.
- Pro: per-team operational preferences (on-call rotations,
  escalation trees) live in the routing service, not in
  `_owners.yaml`.
- Con: adds a new platform component (the routing service)
  that needs to be designed, deployed, operated. Massive
  scope expansion for Wave 1.
- Con: the "alerting consumer" foundation doc 04 already
  references is conceptually this, but as a thin consumer
  configured from `_owners.yaml` — not a stateful policy
  engine.
- Con: domain teams now have to coordinate two surfaces
  (`_owners.yaml` for ownership + the routing service for
  preferences) to change their alerting.

Reject as over-engineering for Wave 1. A future evolution
toward this is possible (B2 — "entity onboarding workflow")
once operational experience reveals where the seams want to
be; for now, Option B's simpler model is sufficient.

---

## Recommendation

Adopt **Option B** — `_owners.yaml` carries `owner` + channel
**references**; engine deployment config resolves references to
destinations and holds infrastructure-level config (Pub/Sub
topic, dedup window default, severity-per-category defaults).

The recommendation is grounded in:

- foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)
  §"Layer 5 — Alerting" (Pub/Sub event-based architecture;
  routing data in `_owners.yaml`; engine emits no team-specific
  conditionals) and §"Execution Flow" steps 7–8;
- foundation doc
  [`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md)
  §"Ownership Boundaries" (`/rules/_owners.yaml` platform-team
  reviewed);
- foundation doc
  [`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md)
  P3 (explicit ownership) and the workspace identity for
  `rules/`;
- prior decision
  [`2026-05-20-failure-scope.md`](./2026-05-20-failure-scope.md)
  (B0-4) — CC7 category boundary, OQ-4 severity deferral,
  OQ-6 cross-entity correlation deferral;
- prior decision
  [`2026-05-20-run-identity-and-idempotency.md`](./2026-05-20-run-identity-and-idempotency.md)
  (B0-2) — I3 alert dedup invariant, CC11 dedup-on-execution_id;
- prior decision
  [`2026-05-20-loader-scheduler-retry-failure-semantics.md`](./2026-05-20-loader-scheduler-retry-failure-semantics.md)
  (B0-7) — operational alert sources (CC1, CC2, CC4, CC6, CC8,
  CC10, CC11 of B0-7).

The specific commitments beyond what those documents state are
**new contribution proposed here, requires review**:

1. **`_owners.yaml` schema** — required fields (`owner`,
   `channels`); optional fields (`severity_overrides`,
   `description`); top-level entity map keyed by entity name.
   **New contribution proposed here, requires review.**

2. **Channel-reference namespace** — opaque strings of the form
   `<type>:<id>` where `<type>` indicates the destination class
   (e.g., `slack`, `pagerduty`, `email`) and `<id>` is an
   environment-resolvable identifier. The engine deployment
   config resolves `(environment, type, id)` to a concrete
   destination. **New contribution proposed here, requires
   review.**

3. **Engine deployment config carries the channel resolution
   table, the Pub/Sub topic, the default dedup window, and the
   default severity per category.** Per-environment config.
   **New contribution proposed here, requires review.**

4. **Dedup is two-layer: idempotent emission + consumer
   defense.** The engine emits at most one event per
   `(execution_id, check_id, result)` tuple (idempotent
   emission); alerting consumers also dedup on receive within a
   configured window (consumer defense, against engine-instance
   failover or re-emission). **New contribution proposed here,
   requires review.**

5. **Cross-entity correlation is consumer-side, not engine-side
   (B0-4 OQ-6 resolved).** The engine emits per (entity,
   check) events; alerting consumers may optionally roll up
   correlated events into incidents per their own policy. B0-6
   commits that the engine does not perform correlation. **New
   contribution proposed here, requires review.**

---

## Consequences

Adopting this recommendation commits the platform to the
following.

**CC1. `_owners.yaml` schema (required and optional fields).**
The file is structured as a top-level `entities` map keyed by
entity name. Each entity entry has:

- `owner` (string, **required**) — a team identifier matching
  a CODEOWNERS group (per foundation doc 02 §"Ownership
  Boundaries"). The owner identifier is the same string used
  in `CODEOWNERS` path rules for the entity's rule file.
- `channels` (map, **required**) — keyed by alert category
  (`data_quality`, `operational`), each value is a list of
  channel-reference strings (CC2 below). At least one category
  must have a non-empty list; an entity with no channels in
  either category cannot be activated.
- `severity_overrides` (map, **optional**) — keyed by
  `check_id` (or the literal string `default` to override
  per-category default); each value is a map from category
  (`data_quality`, `operational`) to severity
  (`info` / `warning` / `critical`).
- `description` (string, optional) — free-text human-readable
  description of the entity; for documentation only, not used
  in routing.

Top-level required field beyond `entities`:

- `schema_version` (integer, **required**) — the version of
  the `_owners.yaml` schema itself. Initial value `1`. Aligns
  with B0-1's schema-versioning approach for the DSL.

```yaml
# Structural shape committed; entity name (customer), channel
# IDs (customer-alerts, customer-oncall), and severity override
# values are illustrative placeholders.
schema_version: 1
entities:
  customer:
    owner: customer-team
    channels:
      data_quality:
        - slack:customer-alerts
      operational:
        - slack:customer-alerts
        - pagerduty:customer-oncall
    severity_overrides:
      partition_freshness:
        data_quality: critical
    description: "Customer master entity; high-priority data quality."
```

The entity name `customer`, channel IDs (`customer-alerts`,
`customer-oncall`), and `severity_overrides` entries above are
illustrative placeholders. The **structural shape** —
`entities` map keyed by entity, with `owner` and `channels`
required and `severity_overrides` + `description` optional,
plus top-level `schema_version` — is what CC1 commits. **New
contribution proposed here, requires review.**

**CC2. Channel-reference structure.** Channel references are
**(type, id) pairs**:

- `type` is a destination-class identifier matching a
  registered class in engine deployment config (e.g., `slack`,
  `pagerduty`, `email`, `webhook`). The set of supported types
  per environment is engine-config concern; the linter
  validates that every `type` referenced in `_owners.yaml` is
  registered in the active deployment config.
- `id` is an environment-resolvable identifier — typically a
  channel name (Slack), service identifier (PagerDuty), alias
  (email distribution list), or equivalent stable reference.
  The `id` is what humans read in code review; the actual
  destination (webhook URL, service key, email address) lives
  in engine deployment config.

The on-wire **encoding** of the pair in `_owners.yaml` — a
colon-separated string (`slack:customer-alerts`), a YAML
mapping (`{type: slack, id: customer-alerts}`), or another
encoding — is Wave 3 syntax detail. CC2 commits the
structural pair and the requirement that the `type` be
registered; the encoding is implementation-shaped.

**CC3. Engine deployment config (committed surface).** Engine
deployment config — owned per environment by the platform team
plus SRE per foundation doc 02 §"Ownership Boundaries"
(`/deploy/`) — holds:

- Pub/Sub topic the engine publishes to (one per environment).
- Channel resolution table: maps `(environment, type, id)` →
  concrete destination (webhook URL, service key, email
  address, etc.). Secrets within this table follow standard
  per-environment secret-management discipline (deferred to
  Wave 2 emulator scope and Wave 3 production substrate).
- Default severity per category: `data_quality` default
  severity, `operational` default severity. B1 picks per
  environment.
- Default dedup window length (CC5). B1 picks per environment.
- Per-environment overrides for any of the above, if needed.

Engine deployment config does **not** contain:

- Entity → owner mappings (those are in `_owners.yaml`).
- Per-entity routing decisions (those are in `_owners.yaml`).
- Per-team conditionals (foundation doc 04 anti-goal).

**CC4. Event payload schema (engine emits to Pub/Sub).** Every
alert-relevant engine action produces a structured event. The
following fields are committed; each field's provenance is
marked (grounded in an upstream B0, or new in B0-6 and flagged
for review).

*Identity fields:*

- `execution_id` (string, sha256-hex per B0-2 CC7,
  **optional**) — the logical execution this event belongs to.
  **Absent for engine-startup events** (B0-7 CC1/CC2 loader
  failures at startup, before any execution has been initiated)
  where no execution context exists; present for all
  check-level events and for execution-level events that occur
  after a plan has been created (B0-7 CC8 pre-check error,
  B0-7 CC10 aborted, B0-7 CC11 orphan finalization). Grounded
  in B0-2 CC1 (the formula).
- `attempt_id` (string, UUID, optional — absent when
  `execution_id` is absent) — the specific attempt within the
  execution. Grounded in B0-3 CC4.
- `entity` (string, **required**) — the entity name, matching
  `_owners.yaml` keys. Grounded in B0-3 CC3.
- `check_id` (string, optional — present for check-level
  events; absent for execution-level events: `status = error`
  from pre-check, `status = aborted`, loader failures).
  Grounded in B0-3 CC7 (check_results composite key).

*Routing fields:*

- `category` (string, enum: `data_quality` | `operational`,
  **required**) — the alerting category. Grounded in B0-4
  CC7's category-boundary mapping.
- `severity` (string, optional, enum: `info` | `warning` |
  `critical`) — set if `_owners.yaml` has a severity override
  for this `(entity, check_id)`; otherwise unset and the
  consumer applies the category default from engine config.
  **New contribution proposed here, requires review** (CC6
  mechanism).
- `event_source` (string, enum: `runner` | `loader` |
  `scheduler` | `orphan_detector` | `trigger_handler`,
  **required**) — identifies the engine component that emitted
  the event; sub-source for forensic linkage to B0-7 CC1–CC11.
  **New contribution proposed here, requires review.**

*Status / result fields:*

- `result` (string, enum: `pass` | `fail` | `degraded` |
  `error`, optional — present for check-level events; absent
  for execution-level events). Grounded in B0-3 CC7 and B0-4
  CC1.
- `status` (string, enum: `running` | `success` | `failed` |
  `error` | `aborted`, optional — present for execution-level
  events; absent for check-level events). Grounded in B0-3
  CC6 and B0-4 CC2.

*Context fields:*

- `recorded_at` (RFC 3339 UTC timestamp, second precision per
  B0-2 CC2 — fractional seconds excluded, **required**) — the
  event time. Grounded in B0-3 CC3.
- `error_summary` (string, optional — present for error /
  aborted / loader-failure events) — mirrors the
  `dq_check_results.evidence_summary` or
  `dq_executions.error_summary` field. Grounded in B0-3 CC3 /
  CC7.

Additional fields may be added additively without breaking
existing consumers; consumers MUST tolerate unknown fields per
the JSON Schema additive-evolution policy from B0-1.

**CC5. Dedup mechanism: two-layer with distinct
responsibilities.** Dedup operates at two layers, each with
its own role:

- **Engine-side literal-duplicate suppression.** The engine
  ensures that a single attempt does not emit the same event
  twice (e.g., due to retry on emit failure). The commitment
  is that the engine never emits the same
  `(execution_id, attempt_id, check_id, result)` tuple twice
  in a single attempt — the in-process suppression key is
  implementation-shaped (a literal-event identifier; Wave 3
  picks). This is **correctness against re-emission within an
  attempt**, not the primary enforcement of B0-2 I3.
- **Consumer-side B0-2 I3 enforcement.** Alerting consumers
  dedup on receive within a configurable window (default in
  engine deployment config; B1 picks per environment, see
  OQ-1). The consumer dedup is the **primary enforcement of
  B0-2 I3** ("N scheduler retries → ≤1 user-visible alert per
  failing check"). Consumer dedup keys differ by event class:
  - **Check-level events**: key is `(execution_id, check_id)`.
    The key deliberately **excludes `result`** so that a check
    fluctuating across retries (e.g., `fail` → `error` → `fail`)
    collapses to one user-visible alert per failing check per
    B0-2 I3. (An earlier draft included `result` in the key;
    that would have violated I3 by emitting separate alerts
    for each result-value change across retries.)
  - **Execution-level events with `execution_id`**: key is
    `(execution_id, event_source)`. Different engine
    components (loader, scheduler, orphan-detector) reporting
    failures of the same execution are surfaced separately —
    a refresh-failure operational alert and an orphan
    finalization alert for the same execution are distinct
    operational signals.
  - **Loader-failure events with no `execution_id`**
    (engine-startup failures from B0-7 CC1/CC2 where no
    execution has been initiated): key is `(event_source,
    recorded_at-bucketed)`. Wave 3 picks the bucket size; the
    loader-failure event class is rare enough that dedup
    precision is not load-bearing. The bucket-time approach is
    necessary because no `execution_id` is available to
    contribute to the key.

The engine's in-process literal-duplicate suppression state is
bounded by the longest expected execution duration plus a
margin (B1 picks). The consumer's dedup window is bounded by
the configurable window above (B1 / OQ-1).

B0-2 CC11 ("dedup on `execution_id`") is honored: every
check-level and execution-with-`execution_id` dedup key
includes `execution_id` as a component. Loader-failure events
without an `execution_id` are dedupped on `event_source` plus
a time bucket; they are operationally exceptional and
consumer-side windowing is sufficient.

**CC6. Severity within category — default + override
(B0-4 OQ-4 resolved).** Severity is a property of an emitted
event, conveyed in the optional `severity` field of CC4:

- **Default severity per category** lives in engine deployment
  config (CC3). Initial defaults (B1 picks per environment;
  see OQ-2): `data_quality` defaults to `warning`;
  `operational` defaults to `warning`.
- **Per-entity / per-check overrides** live in `_owners.yaml`
  under `severity_overrides` (CC1). An entity may declare
  per-check severity (`partition_freshness: data_quality: critical`)
  or a default-for-this-entity override (`default:
  data_quality: critical`).
- The engine reads `_owners.yaml` at manifest load (per
  B0-7 CC1-CC2 loader path) and applies overrides at event
  emission time. If no override matches, the `severity` field
  is omitted from the event and the consumer applies the
  category default from engine config.

This resolves B0-4 OQ-4 (severity beyond category): B0-4
locked the category; B0-6 picks the default-plus-override
shape. **New contribution proposed here, requires review.**

**CC7. Category mapping table — engine emits per B0-4 CC7.**
The engine sets the `category` field in CC4 events per the
table B0-4 CC7 committed:

| Engine signal                                     | category    |
|---------------------------------------------------|-------------|
| check `result = fail` or `degraded`               | data_quality|
| check `result = error`                            | operational |
| execution `status = error` (pre-check, B0-7 CC8)  | operational |
| execution `status = aborted` (B0-7 CC10)          | operational |
| loader failure (B0-7 CC1 / CC2)                   | operational |
| scheduler reconciliation failure (B0-7 CC4)       | operational |
| trigger-handler retry exhaustion (B0-7 CC6)       | operational |
| orphan finalization (B0-7 CC11)                   | operational |

This is the union of B0-4 CC7 (check-level + execution-level)
and B0-7's seven operational alert sources. The engine never
deviates from this mapping; B0-6 has no discretion to route a
check `error` as data-quality.

**CC8. Cross-entity correlation is consumer-side
(B0-4 OQ-6 resolved).** The engine emits one event per
`(entity, check)` failure regardless of correlated activity
across entities. The engine does not collapse N entities
erroring in the same window into a single event; it does not
suppress events for "downstream" entities based on upstream
failure. Cross-entity correlation, if desired, is the
alerting consumer's policy and uses the consumer's own state
(it can observe the Pub/Sub event stream and apply
correlation rules before delivery).

This resolves B0-4 OQ-6. The engine's stance: **fan out per
event; let the consumer correlate**. B0-6 commits that the
engine does not perform correlation. **New contribution
proposed here, requires review.**

**CC9. Owner declaration is mandatory at the linter layer.**
The linter (per B0-1's CI gate, which validates rule files
against the DSL schema) is extended to check that every
entity declared in a rule file has a corresponding entry in
`_owners.yaml`. The linter check is B0-6's scope: a merge
request introducing an entity without an `_owners.yaml`
entry fails CI and cannot land on `main`. This enforces
foundation doc 04's "no alert without owner" anti-goal at
**author time**.

**Defense-in-depth at the publisher and loader layers** —
extending B0-5 CC5 (manifest publication verifications) and
B0-7 CC1/CC2 (loader fail-closed checks) to also verify owner
existence — would strengthen the invariant against
out-of-band edits to `_owners.yaml` or the manifest after
linting. But B0-5 and B0-7 are resolved with their
verification sets already committed; extending them would
require reopening those studies. This is flagged as a
follow-up in OQ-10 (new). **B0-6 commits only the linter
check**; deeper enforcement is a separate decision.

**CC10. Engine never embeds team-specific conditionals.** All
team-relevant routing decisions are reads from `_owners.yaml`
and engine deployment config; no engine code path has a
conditional on team name, channel destination, or per-team
severity. This is foundation doc 04 §"Layer 5"'s anti-goal
made explicit. The category boundary (CC7) is permitted as a
non-team-specific routing decision; the channel-reference
namespace (CC2) is permitted as a non-team-specific type
discriminator. Everything beyond — which team, which channel,
which severity — is data.

**CC11. The contract surface for alerting consumers is the
event payload + `_owners.yaml` + engine deployment config.**
Alerting consumers read:

- Pub/Sub events with the CC4 payload schema;
- `_owners.yaml` (made available to consumers via a Wave 3
  mechanism — see OQ-5; typically a published artifact, e.g.,
  copied to a known object-storage path alongside the manifest);
- engine deployment config relevant to channel resolution
  (this requires consumers to be deployment-aware; how this is
  achieved — embedded config, an Admin API endpoint, a
  separate config-distribution mechanism — is Wave 3,
  OQ-5).

Consumers do **not** read `dq_executions` or `dq_check_results`
for routing decisions — the canonical record stays in
BigQuery (D10), but the routing decision is made from the
event payload alone. Consumers may query the tables for
their own forensic or escalation purposes, but routing-time
operation is event-driven.

**CC12. `_owners.yaml` schema evolves under the same
contract-driven protocol as B0-1.** The `schema_version` field
in CC1 follows B0-1's schema-versioning pattern. Additive
changes (new optional fields, new severity values within an
enum) do not bump the version; breaking changes (field
removal, renames, enum-value removal) require a new
`schema_version` and a documented migration path. The linter
validates `schema_version` against the engine-supported set
at lint time, mirroring B0-1's three contract checks (CC7 of
B0-1) for the owners file.

**CC13. Numeric parameters are B1.** The following are
explicitly deferred to B1 with per-environment differentiation:

- Default severity per category (CC6).
- Consumer-side dedup window length (CC5).
- Engine-side in-process dedup state retention bound (CC5).
- Per-environment channel-resolution-table entries (CC3) —
  the table shape is committed; the entries are operational.

B0-6 commits the **shapes**; B1 commits the values.

---

## Open Questions

- **OQ-1. Consumer-side dedup window length default.** The
  configurable default for consumer-side dedup window
  (CC5) is **out-of-scope for current cycle** — B1, per
  environment. Order of magnitude likely minutes; exact value
  depends on operational tolerances.

- **OQ-2. Default severity per category.** The initial
  defaults for `data_quality` and `operational` categories
  (CC6) are **out-of-scope for current cycle** — B1.

- **OQ-3. Channel-type registry.** The set of supported
  `<type>` values in the channel-reference namespace (CC2),
  beyond the illustrative `slack`/`pagerduty`/`email`/`webhook`,
  is **out-of-scope for current cycle** — Wave 3 implementation;
  added additively as integrations are built.

- **OQ-4. Alerting consumer implementation.** Whether the
  alerting consumer is a separate service, a sidecar, a
  function, or a library is **out-of-scope for current cycle**
  — Wave 3 implementation. The contract surface (CC11) is
  committed; the consumer shape is downstream.

- **OQ-5. `_owners.yaml` and engine deployment config
  distribution to consumers.** The mechanism by which
  alerting consumers obtain the current `_owners.yaml` and
  the relevant channel resolution from engine deployment
  config — published artifact, Admin API endpoint, embedded
  config, etc. — is **out-of-scope for current cycle**.
  Wave 3 implementation, informed by the manifest publication
  shape from B0-5 (alerting consumers may consume the same
  pointer + content-addressed pattern, or use a separate
  distribution).

- **OQ-6. Secret management for channel resolution.** The
  per-environment secret-management discipline for webhook
  URLs, service keys, and other credentials in the channel
  resolution table (CC3) is **out-of-scope for current cycle**
  — Wave 2 (emulator scope per W2-3) and Wave 3 (production
  substrate) own this.

- **OQ-7. Cross-entity correlation policy templates.**
  Standard correlation policies (e.g., "if N upstream entities
  fail in the same window, suppress downstream entity alerts")
  are **out-of-scope for current cycle** — alerting-consumer
  concern, downstream of this study. CC8 commits only that
  correlation is consumer-side; what specific correlation
  patterns are recommended is a future operations document.

- **OQ-8. Severity within `info` / `warning` / `critical`
  beyond the initial three values.** Whether to extend the
  severity enum (e.g., `notice`, `emergency`) is
  **out-of-scope for current cycle** — extensible per CC12's
  schema-evolution protocol; additive extensions don't break
  existing routing.

- **OQ-9. On-call rotation integration.** How the channel
  resolution table interacts with on-call rotation systems
  (e.g., PagerDuty on-call schedules, follow-the-sun rotation)
  is **out-of-scope for current cycle** — operational concern,
  Wave 3 and beyond. The channel reference (`pagerduty:customer-oncall`)
  is resolved to a PagerDuty service identifier; how that
  service routes to humans is PagerDuty configuration,
  outside the platform's scope.

- **OQ-10. Defense-in-depth owner verification at publisher
  and loader.** Whether to extend B0-5 CC5 (publisher
  pre-publish verifications) and B0-7 CC1/CC2 (loader
  fail-closed checks) to verify owner existence — providing
  defense-in-depth against out-of-band edits to the manifest
  or `_owners.yaml` after the linter check — is
  **out-of-scope for current cycle**. Implementing this would
  require reopening B0-5 and B0-7 (their verification sets are
  already committed). B0-6 commits only the linter check (CC9);
  deeper enforcement is a separate follow-up if operational
  experience reveals the linter alone is insufficient.

No open question in this list blocks the routing contract
shape. All items above are parameters, downstream
implementation choices, or operational policies on top of the
locked contract.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0006-alert-routing-contract.md

The `0006` is provisional and assigned at promotion time. If
the Wave 3 ADR numbering convention orders by promotion date
rather than by B0 sequence, the number changes; the slug
(`alert-routing-contract`) does not. This follows the same
convention adopted in
[`2026-05-20-engine-rules-compatibility.md`](./2026-05-20-engine-rules-compatibility.md)
(B0-1, Promotion target section).

The MADR ADR rewrites this study for an external-reviewer
audience (no `studies/` back-references per R8), folds in any
governance refinements from B1-9 (CODEOWNERS finalization)
that intersect with the owner-identifier surface (CC1's
`owner` field), and updates the relevant sections of foundation
doc
[`04-system-architecture.md`](../foundation/04-system-architecture.md)
§"Layer 5 — Alerting" to reference the ADR.

A companion **owners schema document** under
`docs/contracts/owners-schema.md` (per the decision log row's
"Governance doc + owners schema" expected output) is authored
during Wave 3, providing the contributor-facing reference for
the `_owners.yaml` schema; the schema itself ships in
`rules/_owners.schema.json` (analogous to the rule DSL schema
mirror in B0-1) and is enforced by the linter under B0-1's
established CI gate.

