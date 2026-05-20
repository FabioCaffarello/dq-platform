<!-- path: studies/foundation/04-system-architecture.md -->

# 04 — System Architecture

## Metadata

- Purpose: describe the technical architecture of the DQ Platform end
  to end, including the layers, the runtime flow, the internal
  patterns adopted, and the rationale behind each design choice.
- Audience: platform engineers, architects, future maintainers, AI
  agents.
- Status: draft
- Last updated: 2026-05-20
- Promotion target: `docs/architecture/overview.md`,
  `docs/architecture/components.md`,
  `docs/architecture/data-flow.md`, and several ADRs during Wave 3.

---

## Architectural Layers

The platform is organized as five layers. Each layer has a clear
responsibility and a clear set of things it must not do.

### Layer 1 — Authoring

**Lives in:** `rules/`

**Responsibilities:**

- entity YAML rule specifications;
- owner declarations and alert routing metadata;
- contributor-facing examples and onboarding material;
- review workflow for domain changes.

**Does not:**

- contain runtime execution logic;
- contain environment-specific credentials or deployment
  configuration;
- contain hidden transforms that mutate author intent after merge.

### Layer 2 — Control

**Lives across:** `rules/` CI, `tools/`, and `engine/`

**Responsibilities:**

- schema validation;
- manifest generation and publication to object storage;
- compatibility verification between rules and engine;
- safe distribution of active rule versions.

**Does not:**

- own rule content;
- own runtime execution.

This layer exists to make runtime deterministic. The engine consumes
curated, validated artifacts — never raw repository state.

### Layer 3 — Execution

**Lives in:** `engine/`

**Responsibilities:**

- HTTP API for triggering runs;
- schedule resolution;
- rule loading from the active manifest;
- scheduler lifecycle reconciliation;
- compilation of declarative rules to safe BigQuery queries;
- query execution and result capture;
- retry and failure semantics;
- concurrency and cost control.

**Does not:**

- contain domain-specific logic;
- accept rule data outside the manifest path;
- expose escape hatches for ad-hoc execution.

This is the operational heart of the platform. It must stay
platform-centric and refuse domain-specific shortcuts.

### Layer 4 — Reporting

**Lives in:** `engine/` (writers) and BigQuery (storage)

**Responsibilities:**

- run-level metadata persistence (`dq_executions`);
- per-check result persistence (`dq_check_results`);
- evidence storage boundaries (failed sample retention);
- downstream consumption contracts for dashboards and alerting.

**Does not:**

- transform or summarize results in ways that lose evidence;
- couple itself to specific dashboard implementations.

Reporting is not a side effect. It is part of the platform contract.

### Layer 5 — Alerting

**Lives in:** `engine/` (event emission) with routing data from `rules/`

**Responsibilities:**

- event emission for failed or degraded checks;
- deduplication windows;
- severity-based routing;
- progressive promotion from test to production channels.

**Does not:**

- hardcode team-specific behavior inside engine logic;
- send alerts without an owner declared in `rules/_owners.yaml`.

Routing policy is data-driven wherever possible. The engine never
embeds team-specific conditionals.

---

## End-to-End Flow

### Rule Lifecycle Flow

1. A contributor proposes a change in `rules/` via merge request.
2. CI validates the YAML against the schema, runs lint, and produces
   a dry-run SQL preview.
3. Reviewers (entity owners plus platform team for cross-cutting
   areas) approve the merge request.
4. On merge to `main`, CI re-runs validation across the whole
   ruleset to catch cross-rule conflicts.
5. When a `rules-v*` tag is created, CI generates a new manifest
   and publishes it to GCS using the write-new-then-swap pattern.
6. The engine reads the new manifest on its next refresh cycle.

### Execution Flow

1. The scheduler (or a manual operator) triggers `/runs` on the
   engine, authenticated via OIDC.
2. The trigger handler validates identity, parses the requested
   entities and windows, and creates an execution plan.
3. The plan is persisted in `dq_executions` with status `running`.
4. For each entity in the plan:
   a. The relevant rule is fetched from the in-memory ruleset.
   b. The compiler generates a BigQuery query per check.
   c. The runner executes the queries concurrently (bounded).
   d. Results are captured: pass / fail / degraded, with summary
      counts and (for failures) sample violating rows up to a
      configured limit.
5. Per-check results are written to `dq_check_results`.
6. The execution row is updated with final status.
7. For each failed check, a result event is published to Pub/Sub.
8. Alerting consumers fan out the events to Slack, PagerDuty, or
   other destinations based on `_owners.yaml`.

### Evolution Flow

Checks marked `stream_compatible` in their YAML declaration become
candidates for future materialization in a Kafka-backed stream runner.
The semantic model (entity, check identity, severity, result shape)
stays the same; only the runtime substrate changes. The reporting
schema accommodates both modes through aggregated time-window rows.

---

## Internal Patterns

The platform adopts several internal patterns that recur across
components. Describing them here once avoids re-explaining the same
shape in every component description.

### Pattern P1 — Fail-fast registry loading

The loader does not behave like a passive file reader. It behaves
like a registry builder:

1. Reads the active manifest in one operation.
2. Fetches every referenced rule YAML in parallel.
3. Validates each rule against the schema during load.
4. Indexes rules by entity.
5. Fails fast on:
   - duplicate entity keys (two YAMLs claim the same entity);
   - schema validation failures;
   - checksum mismatches;
   - missing referenced YAMLs.

The failure mode is **always** a clear error message that names the
offending file and the specific problem. No partial loading. No
silent skipping.

This pattern is the foundation of runtime trust. If the ruleset
loaded, it loaded completely and correctly.

### Pattern P2 — Lifecycle-aware scheduler integration

The scheduler integration is not a single "create cron" call. It is a
lifecycle:

- **Deploy or reconcile:** create or update scheduled triggers based
  on the active ruleset. Stable trigger names so updates do not
  duplicate.
- **Delete:** remove triggers for entities that were removed from the
  ruleset.
- **Status:** inspect every trigger's current state (`enabled`,
  `paused`, `failing`), last run time, last error.
- **Orphan cleanup:** detect and remove triggers that exist in the
  scheduler but no longer correspond to any active rule.

This lifecycle is owned by a single subsystem inside `engine/`, with
admin endpoints for manual invocation and a periodic reconciliation
loop for drift correction.

### Pattern P3 — Fixture-driven compiler testing

Query compilation is one of the highest-risk parts of the platform.
The testing posture for compilers is:

- every compiler has a `testdata/` directory with one subdirectory per
  scenario;
- each scenario contains the input rule fragment, the expected SQL
  output, and (when relevant) the expected result against a synthetic
  dataset;
- tests are table-driven, comparing generated SQL to the expected
  output and computed results to the expected results;
- adding a new check type is impossible without adding fixtures, by
  convention enforced in code review.

This pattern catches regressions in SQL generation immediately and
makes it possible to review a compiler change by reading the fixture
diff.

### Pattern P4 — Typed multi-environment configuration

The engine runs in multiple environments (local, qa, prod, possibly
more). The configuration model is:

- one Go file per environment in `engine/internal/env/`;
- each file declares a typed `EnvConfig` struct with the same shape;
- selection at startup via an environment variable (e.g.
  `DQ_ENV=qa`);
- no dynamic discovery, no inheritance chains, no implicit fallbacks.

If a new field is added to one environment, the build fails until it
is added to all environments. This forces deliberate decisions and
makes drift impossible.

### Pattern P5 — Modular logging contract

Logging supports a global default level plus per-package overrides
via an environment variable:

```
DQ_LOG_LEVELS="root:INFO,engine.compilers:DEBUG,engine.scheduler:WARN"
```

This lets maintainers raise verbosity in one subsystem without
flooding logs globally — especially useful when debugging compilers,
scheduler reconciliation, manifest loading, or alert emission.

The implementation is a small layer over Go's `slog` standard library
package. It is intentionally minimal.

### Pattern P6 — Closed catalog of check types

Every check type has a named compiler in `engine/internal/compilers/`
and a documented contract in `docs/dsl/`. There is no "temporary"
check type, no "custom" check type, no SQL expression escape hatch in
the DSL.

Adding a new check type is a deliberate process:

1. Propose an ADR.
2. Update the schema (`v1` if additive, `v2` if breaking).
3. Add the compiler with full fixtures.
4. Add documentation.
5. Add an example to `rules/_examples/`.

The closed-catalog rule is what makes the platform predictable to
operate and safe to evolve.

---

## Component Inventory

The following components live inside `engine/`. Each has a focused
responsibility.

### `Loader`

Reads the active manifest, fetches and validates rules, indexes by
entity. Implements pattern P1.

### `Scheduler`

Owns the scheduled trigger lifecycle. Implements pattern P2.

### `Trigger API`

HTTP handler for `/runs`. Validates OIDC, parses request, enqueues
plan.

### `Coordinator`

Creates execution plans from a trigger request and an active ruleset.
Owns plan-level concurrency limits.

### `Compilers`

One per check type. Transform declarative check specs into BigQuery
queries. Implement pattern P3.

### `Runner`

Executes compiled queries against BigQuery. Captures results,
including sample violating rows up to configured limits. Honors
concurrency budgets.

### `Reporter`

Writes execution metadata and per-check results to BigQuery tables.
Owns the reporting schema and its evolution.

### `Alerting`

Emits result events to Pub/Sub. Handles deduplication windows and
severity routing based on owner metadata.

### `Admin API`

Operational endpoints: scheduler status, manifest reload, dry-run a
specific rule. Separate from the trigger API, with stricter access
controls.

---

## Hard Architectural Guardrails

These are non-negotiable. They are referenced by `CLAUDE.md` as
constraints on every implementation choice.

### G1. Closed check catalog

Every check type must have a named compiler in the engine and a
documented contract. No hidden "temporary" type, no "custom" type,
no SQL expression escape hatch.

### G2. No free-form execution language in rules

No raw SQL, no embedded scripting, no Python expressions. If a new
expressive power is needed, that is a new DSL construct with compiler
support — not an escape hatch.

### G3. Runtime reads atomic rule sets

The engine consumes manifest-backed versioned sets from object
storage. Never partial rule snapshots. Never repository folders
directly. Never half-published states.

### G4. Every execution must be reconstructable

For every run, the platform can answer:

- which ruleset version was used;
- which engine version was used;
- which time window was evaluated;
- which checks ran, failed, skipped, or degraded;
- with what configuration overrides (if any).

This is what makes incident analysis possible.

### G5. Stream evolution preserves conceptual continuity

When the stream runner is built, it reuses:

- the same entity concept;
- the same check identifiers;
- the same severity semantics;
- a reporting shape compatible with batch results (aggregated by
  time window).

### G6. Internal patterns are described in our own terms

Every pattern adopted here is described and justified on its own
merits. External provenance is not a justification.

---

## Architecture Smells to Reject Early

These patterns indicate the architecture is drifting and should be
caught in code review:

- one-off flags that change behavior for a single entity without
  changing the DSL;
- bypasses that let CI publish rules the engine cannot actually run;
- alerting logic that depends on package-level conditionals per team;
- undocumented defaults that materially change generated queries;
- local-only workflows that cannot be reproduced in CI;
- "temporary" check types implemented as a special case in one
  compiler;
- direct engine-to-rule-file access bypassing the manifest;
- environment-specific behavior changes not declared in `env/`.

Catching these in review is significantly cheaper than removing them
later.

## Open Topics

Tracked in [`06-decision-log.md`](./06-decision-log.md). Architecture
details whose precise shape depends on B0 or B1 decisions are flagged
here:

- exact format of compiled BigQuery queries (templating approach,
  parameter handling);
- exact reporting table schemas;
- exact alerting event shape;
- exact retention policy for failed samples;
- exact concurrency budgets per environment.

These do not change the architecture's shape; they refine its
parameters.
