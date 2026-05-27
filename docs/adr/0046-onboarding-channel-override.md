<!-- path: docs/adr/0046-onboarding-channel-override.md -->

# ADR-0046 — Onboarding-Channel Override

- **Status:** accepted
- **Date:** 2026-05-27

---

## Context

[ADR-0040](./0040-entity-onboarding-workflow.md) commits
the three-tier entity-onboarding model (Candidate →
Test-soak → Production) and procedurally enforces a
**shared-substrate channel-collision workaround**:
deployments where qa and prod share the same alerting
substrate (one Slack workspace, one email domain, one
PagerDuty tenant) must use qa-prefixed channel
references at Tier 1
(`slack:#qa-dq-orders`, `email:qa-oncall@example.com`)
and rename them back at Tier 2. This works but requires
a **two-step `_owners.yaml` edit per onboarding**: one
at Tier 0 → Tier 1 (add prefix), another at Tier 1 →
Tier 2 (remove prefix).

ADR-0040 §"Open Questions OQ-1" registered the
engine-level mechanism that would close this with one
flip of a boolean — and that's what this ADR commits.
The contract pieces are:

1. A per-entity `onboarding: bool` field on
   `_owners.yaml` v3.
2. An env-level fixed
   `EnvConfig.OnboardingChannel` string.
3. A consumer-side routing override — when
   `entity.onboarding == true` AND
   `env.OnboardingChannel` is non-empty, the alerting
   consumer routes the alert to `OnboardingChannel`
   instead of `entity.channels`.

Two structural constraints shape the decision:

- **`_owners.yaml` is rules-side only.** Per ADR-0022
  carrying convention (also reaffirmed by ADR-0006
  §"Engine deployment config does not contain entity →
  owner mappings"), `_owners.yaml` lives under
  `rules/_schema/` with no engine-side mirror. The
  engine never reads the file at runtime; the
  alerting consumer does. This ADR honors that
  posture — the routing override lives **consumer-side**,
  not engine-side.
- **No alerting consumer code exists in this repo
  today.** The engine emits alert events to a
  Pub/Sub topic per ADR-0006 CC3; the routing code
  is operator-supplied (an external consumer). The
  consumer side of this ADR's contract is therefore
  a *commitment* operators implement in their
  consumer — the platform commits the schema fields
  and the env-config surface; the runtime routing
  is operator-owned per ADR-0006's existing posture.

Existing commitments this ADR builds on:

- [ADR-0006](./0006-alert-routing-contract.md) §C7
  commits `_owners.yaml.entities.<entity>.channels`
  as the canonical routing map and the consumer
  reads it. This ADR adds one optional sibling field
  and one consumer-side override rule without
  amending CC7.
- [ADR-0021](./0021-mode-as-primitive.md) committed
  `_owners.yaml` v2 (added `mode: set|record` per
  entity); this ADR commits v3 by extending v2 with
  the optional `onboarding` field.
- [ADR-0040](./0040-entity-onboarding-workflow.md)
  Tier 0 → Tier 1 criterion 6 + Tier 1 → Tier 2
  criterion 7 commit the qa-prefix procedural
  workaround. After this ADR ships, deployments
  using the override can elect to skip those
  criteria — the override is the
  procedural-workaround replacement.

The principles bearing on the decision are **P3**
(ownership is explicit — operators own substrate-
collision posture; the platform owns the schema
field), **P4** (cost is a first-class constraint —
one boolean flip vs two channel-list edits halves
the onboarding edit cost), and **P5** (evolution
must be contract-driven — the schema bump is
additive per ADR-0001's compatibility model).

---

## Decision

### `_owners.yaml` v3 — optional per-entity `onboarding` flag

`_owners.yaml` v3 extends v2 with one additive optional
field on each entity descriptor:

```yaml
schema_version: 3
entities:
  customer:
    owner: "@example-org/rules-authors"
    mode: set
    onboarding: false   # default; explicit for clarity
    description: First onboarded entity end-to-end (W3-P6d).
    channels:
      data_quality:
        - slack:#dq-customer
      operational:
        - email:oncall@example.com
  orders_stream:
    owner: "@example-org/rules-authors"
    mode: record
    onboarding: true    # Tier 1 — routes to env.OnboardingChannel
    channels:
      data_quality:
        - slack:#dq-orders
      operational:
        - email:oncall@example.com
```

Field semantics:

- **Optional.** Default value when absent is `false`.
  v2 owners files validate against v3 by adding the
  field at parse time with the default; the lint
  binary's v3 dispatcher accepts both shapes.
- **Boolean.** No `partial` / `tier-1` / etc. — the
  flag is a binary signal "this entity is in
  onboarding". The three-tier model from ADR-0040
  collapses to "yes / no" at the channel-override
  level because the procedural workaround only
  applies at Tier 1.
- **Operator-toggled.** Flipping the flag is a
  standard rules-side PR
  (`@example-org/platform-team` reviews per ADR-0015
  CODEOWNERS-routing); the next `_owners.yaml`
  publish makes the routing change effective on the
  next refresh tick.

The v3 schema lives at
`rules/_schema/_owners.v3.schema.json` (rules-side
only; no engine-side mirror per ADR-0022's owners-
carrying convention).

### `EnvConfig.OnboardingChannel` — env-level fixed channel

A new field on `EnvConfig` exposes the per-env
onboarding-channel target:

```
type EnvConfig struct {
    // ... existing fields ...

    // OnboardingChannel is the env-level fixed channel
    // alerts route to when the source entity has
    // onboarding: true on its _owners.yaml v3 entry.
    // Empty disables the override (entity.channels
    // routes normally). Per-env values are operator-
    // chosen; the platform commits the contract, not
    // a specific channel string. Per ADR-0046.
    OnboardingChannel string
}
```

Per-env recommendation (operators customise per their
substrate):

- **local:** `"slack:#dq-onboarding-local"` — a local
  channel used during development against the
  emulator stack.
- **qa:** `"slack:#dq-onboarding-qa"` — the qa
  substrate's onboarding channel where Tier 1
  alerts land while soaking.
- **prod:** `"slack:#dq-onboarding-prod"` — the prod
  substrate's onboarding channel.

The field is consumer-readable, not engine-readable.
The engine binary holds the value (so it surfaces
via the standard env-config inspection surface);
the alerting consumer reads it from a shared
configuration source (the engine's deployment
ConfigMap, an env var, or operator-provided
config). The platform commits the field's
existence + the routing contract; the consumer
implements the override.

### Consumer-side override rule

When the alerting consumer receives an alert event
for entity E:

1. Look up `entities[E].onboarding` in the loaded
   `_owners.yaml`. Default to `false` if absent.
2. Read `env.OnboardingChannel` from the consumer's
   shared config source.
3. **If both conditions hold** —
   `entities[E].onboarding == true` AND
   `env.OnboardingChannel` is non-empty — route the
   alert to `env.OnboardingChannel`. The
   `entities[E].channels[category]` list is NOT
   consulted.
4. **Otherwise** — route per ADR-0006 CC7 against
   `entities[E].channels[category]`.

The override applies to **all categories**
(`data_quality` and `operational`) uniformly. The
onboarding-channel target is a single substrate-
identifier string; rules with different per-
category preferences during onboarding are out of
scope at v1 (operator unflag the onboarding boolean
and use per-category channels when finer routing
is needed).

The override is **opt-in twice**: the operator must
both set `entity.onboarding: true` AND populate
`env.OnboardingChannel` for the override to fire.
This makes per-env opt-in clean: a deployment with
separate per-substrate qa and prod (no collision
risk) leaves `OnboardingChannel` empty for qa+prod
and never routes through this path.

### Why this does not amend ADR-0006

ADR-0006 CC7 commits the per-category channel-list
contract: alerts route to `entities[entity].channels[category]`.
This ADR layers an **override** on top:
when `entity.onboarding == true` AND
`env.OnboardingChannel` is non-empty, the override
fires; otherwise CC7 applies unchanged. ADR-0006's
contract surface is preserved — the override is a
sibling rule, not a CC7 amendment.

ADR-0006 §"Engine deployment config does not contain
entity → owner mappings" is also preserved — the
new `EnvConfig.OnboardingChannel` field is a
**channel identifier**, not an entity → owner
mapping. The engine holds the value but does not
consume it; consumer-side routing reads it
externally.

### Why this does not amend ADR-0040

ADR-0040's Tier 0 → Tier 1 criterion 6 and Tier 1 →
Tier 2 criterion 7 commit the procedural workaround
(qa-prefix + rename-back). After this ADR ships,
deployments using the override **may** skip those
criteria with a recorded justification in the
promotion PR. ADR-0040's criteria themselves stay
in place as the v1 fallback for deployments
without the override configured.

### Versioning

`_owners.yaml` v3 follows ADR-0006 CC12's evolution
rule: additive within a major; breaking changes
require a future v(N+1) file. The v3 → v2 path is
NOT supported by the lint binary — operators on v3
who need to roll back commit a manual edit removing
the `onboarding` field and changing `schema_version`
back to 2.

The `tools/migrate` binary (B2-23) does NOT cover
owners-schema migrations at v1 — the existing
binary scope is rule-schema (v1 → v2) only. A
future `tools/migrate` extension or a separate
`tools/owners-migrate` may automate
`_owners.yaml` migrations when concrete operator
demand surfaces.

### Why this does NOT commit engine-side resolution

The engine binary does not consume `_owners.yaml`
at any code path. The engine emits the alert event
to Pub/Sub per ADR-0006 CC3 with `entity` +
category + severity; the consumer reads
`_owners.yaml` and decides routing.

Engine-side resolution would reopen ADR-0006's
"engine doesn't know channel destinations" posture
and pull the alerting-consumer's substrate
dependencies (Slack API, SMTP, PagerDuty API) into
the engine binary. The cost is non-trivial; the
benefit is marginal (one fewer config read in the
consumer). Reserved indefinitely unless concrete
operator demand surfaces.

The slice's "engine routing logic" wording in the
decision-log row was speculative scope addition;
the actual mechanism per ADR-0006 lives entirely
in the consumer.

---

## Consequences

1. **`_owners.yaml` v3 ships with an optional
   `onboarding: bool` field per entity.** Operators
   on v2 stay on v2; operators migrating to v3 add
   the field as needed.

2. **`EnvConfig.OnboardingChannel string` ships with
   per-env values.** Recommended default placeholders
   are `slack:#dq-onboarding-{local,qa,prod}`;
   operators substitute their substrate-specific
   identifiers. Empty disables the override.

3. **No engine code change ships from this ADR.**
   The override fires consumer-side; the engine
   continues to emit alert events per ADR-0006 CC3
   unchanged. The `EnvConfig.OnboardingChannel`
   field exists on the env struct so downstream
   consumers can read it from a shared config
   source.

4. **No new B2 row for engine-side resolution.** The
   engine never consumes the field; consumer-side
   resolution is the committed pattern. Operators
   building alerting consumers implement the
   override per §"Consumer-side override rule".

5. **ADR-0040 criteria 6 + 7 become optional for
   deployments using the override.** Operators
   record a justification in the promotion PR
   description ("override configured;
   shared-substrate workaround not needed"); the
   criteria themselves stay in place as the v1
   fallback path.

6. **B2-25 closes.** The decision-log B2-25 row
   moves to `resolved-adr`. No new follow-up rows
   register; the v3 → v2 migration path is
   operator-manual at v1.

7. **ADR-0006, ADR-0040, ADR-0021, ADR-0022,
   ADR-0023 are preserved.** This ADR layers an
   override rule on top of ADR-0006 CC7 and adds
   one optional schema field per ADR-0021's v2
   foundation without amending them.

8. **`tools/lint` accepts `_owners.yaml` v3
   alongside v2.** The lint binary's
   `OwnersSchemaSet` gains a V3 field; the
   dispatcher reads the file's `schema_version`
   and validates against the matching schema. v2
   files continue to validate against the v2
   schema; v3 files validate against v3.

9. **`tools/migrate` does NOT cover owners-schema
   migrations.** The existing binary's scope is
   rule-schema (v1 → v2) only. Owners migrations
   are manual at v1; a future tool may automate
   them when demand surfaces.

10. **Two deferred items are flagged out-of-scope:**
    engine-side channel resolution (reopens
    ADR-0006; deferred indefinitely); per-category
    onboarding-channel overrides (Slack vs email vs
    PagerDuty during onboarding); reserved if the
    binary onboarding-flag turns out to be too
    coarse in practice. The v1 contract treats the
    onboarding channel as a single substrate-
    identifier string.

---

## Notes

- The recommended `slack:#dq-onboarding-{env}`
  channel-naming convention is exactly that —
  recommended. The platform commits no per-substrate
  channel-naming rule; operators choose names per
  their substrate's conventions.
- Operators with separate per-env substrates (no
  collision risk) may still find the override useful
  for non-collision reasons — e.g., routing every
  Tier 1 alert through a single onboarding-aware
  consumer that adds extra logging or routes to a
  ticketing system. The override is operationally
  useful beyond just the collision-workaround
  scenario.
- A future enhancement worth flagging if signal
  emerges: a per-category map at
  `entities[E].onboarding_channels.{data_quality,operational}`
  for fine-grained per-category routing during
  onboarding. The v1 commits the single-channel
  shape; the per-category extension is an additive
  change within v3 that lands by ADR amendment.
