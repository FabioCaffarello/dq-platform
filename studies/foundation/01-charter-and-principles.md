<!-- path: studies/foundation/01-charter-and-principles.md -->

# 01 — Charter and Principles

## Metadata

- Purpose: define the project's mission, the audiences it serves, the
  principles it refuses to compromise, and the identity of each
  workspace inside the monorepo.
- Audience: every contributor, reviewer, maintainer, and AI agent that
  touches this repository.
- Status: draft
- Last updated: 2026-05-27
- Promotion target: `docs/charter.md` and `docs/principles.md`
  during Wave 3.

---

## Mission

Build a long-lived Data Quality platform that evaluates trust in
curated data assets without coupling itself to the production
ingestion path.

The platform must support:

- **Batch validation** over BigQuery-backed entities;
- **Strong governance** for rule authoring across many domain teams;
- **Observable and auditable execution history**;
- **Controlled evolution** toward stream-compatible checks over Kafka.

## Product Position

The DQ Platform is an internal product, not just an internal service.

Its job is not merely to run validations. Its job is to create
**sustained trust** in curated data assets through:

- consistent rule authoring;
- reliable and interpretable execution;
- fast identification of ownership when something fails;
- safe evolution from a small initial scope to organization-wide
  coverage.

## The Promise

The platform makes one promise to the organization:

> If a curated data asset is onboarded into DQ, its quality posture
> becomes visible, owned, and operationally actionable — without
> coupling that visibility to custom code or tribal knowledge.

If the platform cannot uphold that promise, it is not ready to scale.

## Anti-Goals

The platform explicitly avoids these traps:

- becoming a generic SQL execution service;
- embedding domain-specific business logic in the engine;
- depending on the Git repository as runtime storage for active rules;
- treating alert routing as an afterthought;
- mixing experimental and production semantics in the same rule path
  without governance;
- accumulating one-off flags that change behavior for a single entity
  without changing the contract.

## Primary Audiences

### 1. Platform Maintainers

The team that owns the engine, the schema, the compilers, the
scheduler, the reporting pipeline.

**They need:**

- a controlled execution engine;
- predictable operational behavior;
- clear compatibility boundaries between schema and rule artifacts;
- low-friction release mechanics;
- meaningful test coverage where it matters most (compilers, loader).

### 2. Domain Teams

Engineers and analysts who own specific data entities and author rules
for them.

**They need:**

- a safe, declarative authoring surface;
- understandable feedback in CI when their rules are wrong;
- confidence that alerts they receive are actionable;
- low dependency on engine internals to do their job well.

### 3. Incident Responders and Data Consumers

People who act on quality signals when something goes wrong, and the
teams whose decisions depend on the data being trustworthy.

**They need:**

- trustworthy signals (low false-positive rate);
- enough evidence in an alert to triage quickly;
- clear owner mapping when escalation is required;
- historical context to distinguish transient noise from real
  degradation.

## Success Measures

Success is **not** the number of rules in production. A high count of
noisy rules is a failure mode, not an achievement.

Success is:

- onboarding a new entity with low coordination cost;
- high signal-to-noise ratio in alerts;
- clear accountability for failures;
- stable compatibility between schema versions and active rules;
- a path to scale without bespoke exceptions per team.

Directional metrics to optimize for (to be refined later):

- time to onboard a new entity from first draft to monitored test
  channel;
- percentage of active entities with explicit owner and routing
  metadata (target: 100%);
- false-positive rate per promoted check (target: <5% sustained);
- mean time to understand an alert well enough to act on it;
- percentage of rules that pass local and CI validation without
  platform-team intervention;
- BigQuery cost per scheduled entity run, tracked over time;
- percentage of check results reproducible for the same ruleset and
  window.

## Non-Negotiable Principles

These principles constrain every decision in the project. They are
referenced by `CLAUDE.md` as `P1` through `P6`. They must not be
weakened by any contribution.

### P1. Rules must remain declarative

If raw SQL snippets, embedded scripting, or custom escape hatches enter
the rule layer, the platform loses auditability, compatibility
guarantees, and safe evolution. The DSL is intentionally restrictive.

If a new expressive power is genuinely needed, that is a product
decision and should produce a **new DSL construct** with compiler
support — not a generic escape hatch.

### P2. Engine behavior must be deterministic

The same rule version, time window, and source state must produce the
same execution semantics. Hidden defaults, implicit retries, and
environment-driven behavior changes are treated as defects, not
features.

This is what makes incident analysis possible and what makes reports
trustworthy across reruns.

### P3. Ownership must be explicit everywhere

No entity without an owner. No alert route without an owner. No
repository area without a review policy.

Quality platforms fail when alerts are sent to shared voids or when
nobody knows who is allowed to change what.

### P4. Cost is a first-class constraint

BigQuery makes fast progress deceptively easy. Without explicit
guardrails, cost and latency drift surface late and under incident
pressure.

Partition discipline, query templates, dry-run visibility, concurrency
budgets, and evidence-retention limits are **platform design**, not
later hardening.

### P5. Evolution must be contract-driven

The schema, linter, examples, and rule artifacts must evolve under a
published compatibility contract.

Even though everything lives in a single monorepo, the boundary
between `engine/` and `rules/` is a real contract — see
[`03-boundary-contract.md`](./03-boundary-contract.md). Schema-breaking
changes require a new DSL version and a documented migration path.

### P6. Borrow patterns, not baggage

Patterns adopted by this project are described in our own terms and
judged on their fit to our context. External provenance is not a
justification. If a pattern is worth adopting, it is worth describing
on its merits — and it must survive our own scrutiny.

## Workspace Identities

The monorepo holds five workspaces, each with a distinct identity. The
detailed topology (paths, ownership rules, CI behavior) lives in
[`02-monorepo-topology.md`](./02-monorepo-topology.md). What follows
is each workspace's **purpose and boundaries** — the part that
belongs to the charter.

### `engine/` — the runtime

**Owns:**

- DSL schema source of truth;
- engine binaries and runtime services;
- query compilers for every check type;
- scheduler integration and trigger API;
- result persistence semantics;
- alert event emission.

**Success criteria:** predictable to operate, easy to test locally and
in CI, explicit about compatibility promises, hard to misuse from the
rule layer, observable under failure.

**Must reject:** domain-specific rule content, exceptions that bypass
the DSL, undocumented compatibility breaks, runtime behavior that
depends on mutable repository state.

### `rules/` — the authoring surface

**Owns:**

- rule specifications by entity;
- ownership and alert routing metadata;
- contributor-facing examples and tutorials;
- governance workflow for promotion from test to production alerting.

**Success criteria:** a domain team can add or evolve rules without
needing engine internals, reviewers can understand the impact of a
change from the YAML itself, CI failures are understandable and
actionable, every active rule traces to a known owner and a compatible
schema version.

**Must reject:** executable logic beyond the declarative contract,
local conventions not represented in the schema or docs, entity-specific
hacks that require engine conditionals.

### `tools/` — auxiliary CLIs

**Owns:** the linter binary, the dry-run runner, the manifest
publisher, and any other developer-facing CLI that supports the engine
or the rules workflow.

**Success criteria:** small, focused, reusable from both local
development and CI pipelines. Each tool has a single responsibility.

**Must reject:** drift toward becoming a second engine. Tools observe
and validate; they do not execute production checks.

### `deploy/` — infrastructure

**Owns:** Kubernetes manifests, infrastructure configuration,
environment definitions, OIDC and Workload Identity configuration.

**Success criteria:** reproducible deployments, clear separation
between environments, no hidden environment-specific logic that
diverges from what `engine/` declares.

**Must reject:** environment-specific business logic, secrets in
plaintext, untracked configuration drift between environments.

### `docs/` — cross-workspace documentation

**Owns:** architecture overview, ADRs, glossary, governance, and any
documentation that spans more than one workspace.

**Success criteria:** a new contributor can navigate from `docs/` to
the right workspace within minutes, ADRs are discoverable and
cross-linked, no architectural claim lives in only one place when it
spans workspaces.

**Must reject:** workspace-internal documentation that should live
inside that workspace, duplicated content that creates drift.

## Capability Modes

The platform evaluates data quality through two architectural
primitives, each tied to the shape of the source it consumes:

- **Set-oriented mode** — a check operates over a bounded set of rows
  read from a partitioned BigQuery table or view. Aggregation is
  implicit in the set semantics; one execution produces one result
  per check over the declared window.
- **Record-oriented mode** — a check operates on individual records
  consumed from a Kafka topic, bounded by a tumbling watermark
  window. Per-record outcomes are aggregated inside the kind handler
  at window close and produce a single result per check per window.

Mode is declared explicitly on both the rule and the entity
(`mode: set` or `mode: record`). Kind names carry the mode as a
prefix (`set.*` / `record.*`) and the linter enforces consistency
across rule, entity, kind catalog, and source declaration. This is
what P6 ("Borrow patterns, not baggage") looks like in the
data-quality domain: the modes capture the real shape of the data,
not the lineage of any prior tool.

### Stream substrate

Record-oriented checks consume from **Kafka** — not Pub/Sub. Pub/Sub
remains the engine's alert and result-event egress; it is not a
source substrate. The asymmetry is deliberate: Kafka's consumer-group
and partition-offset model gives the engine deterministic replay and
window-bounded reads that a fan-out broker cannot offer. Alert egress
has different invariants (at-least-once fan-out to subscribers) and
is served well by Pub/Sub.

### Origin in the ADR record

The mode primitive and its execution shape are defined by the Wave-S
ADRs:

- ADR-0020 — Wave-S launch
- ADR-0021 — mode as primitive
- ADR-0022 — kind catalog
- ADR-0023 — sources schema (Kafka selected as record substrate)
- ADR-0024 — window semantics
- ADR-0025 — aggregation and unified runner shape
- ADR-0026 — failure scope aggregated
- ADR-0027 — record-mode cost guardrails

These ADRs supersede earlier wording that described stream evolution
as a future direction (see `Mission`, fourth bullet) — that future
has arrived as a declared mode, not as a retrofit of the set-oriented
runtime.

## Closing Position

The project has the ingredients to become a reference-quality internal
platform. The condition is discipline:

- strict workspace boundaries even within a single repository;
- strict DSL governance;
- strict release compatibility rules;
- strict operational semantics;
- strict documentation culture.

If those constraints hold, the platform scales healthily over years.
If they do not, it drifts into a rule zoo with expensive operations and
low trust.

The single most consequential threat to this project is not
technological. It is **uncontrolled responsibility spread** — the
slow, invisible drift of responsibilities across workspace boundaries
until nobody can describe what the engine owns and what the rules own.

The principles above exist primarily to make that drift visible early.
