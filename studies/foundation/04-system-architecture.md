<!-- path: studies/foundation/04-system-architecture.md -->

# 04 — System Architecture

## Metadata

- Purpose: describe the technical architecture of the DQ Platform end
  to end, including the layers, the runtime flow, the internal
  patterns adopted, and the rationale behind each design choice.
- Audience: platform engineers, architects, future maintainers, AI
  agents.
- Status: draft
- Last updated: 2026-05-27
- Promotion target: `docs/architecture/overview.md`,
  `docs/architecture/components.md`,
  `docs/architecture/data-flow.md`, and several ADRs during Wave 3.

---

## Architectural Layers

The platform is organized as five layers. Each layer has a clear
responsibility and a clear set of things it must not do.

The architecture is **multi-mode**. Every check executes in one of two
modes — `set` over a BigQuery-backed bounded row set, or `record` over a
Kafka-backed stream window — and the choice of mode is declared on the
rule and the entity. These are the runtime-level names for the same two
primitives the charter defines as **set-oriented mode** and
**record-oriented mode**; the architecture uses the shorter `set-mode`
and `record-mode` forms throughout. The layers below describe behavior
common to both modes; per-mode divergence is called out explicitly where
it matters. See
[`01-charter-and-principles.md`](./01-charter-and-principles.md)
("Capability Modes") for the principle-level statement, and the **Wave-S
ADRs** (0020–0027) for the founding wave.

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

**Per-mode behavior.** Each rule declares `mode: set` or
`mode: record`; the entity declares its own mode and the linter
rejects rule/entity mismatches. Kind names carry the mode prefix
(`set.*` or `record.*`) and the source declaration must match
(BigQuery for set, Kafka topic + consumer group for record). See
ADR-0021 (mode primitive) and ADR-0023 (sources schema).

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

**Per-mode behavior.** Control is mode-agnostic. The manifest carries
mode-tagged rules but distribution, atomic publication, and refresh
semantics are identical across modes. Scheduler catch-up behavior is
unified (ADR-0033). The kind catalog
(`engine/internal/dsl/catalog/v1.yaml`, ADR-0022) is a co-versioned
artifact that travels with the schema and is validated alongside it.

### Layer 3 — Execution

**Lives in:** `engine/`

**Responsibilities:**

- HTTP API for triggering runs;
- schedule resolution;
- rule loading from the active manifest;
- scheduler lifecycle reconciliation;
- compilation of declarative rules to safe BigQuery queries
  (set-mode);
- bounded consumption from Kafka topics under a tumbling watermark
  window (record-mode);
- query / window execution and result capture;
- retry and failure semantics;
- concurrency and cost control.

**Does not:**

- contain domain-specific logic;
- accept rule data outside the manifest path;
- expose escape hatches for ad-hoc execution.

This is the operational heart of the platform. It must stay
platform-centric and refuse domain-specific shortcuts.

**Per-mode behavior.** A single runner dispatches by mode (ADR-0025).
Set-mode invokes the BigQuery handler over the rule's declared
partition window. Record-mode invokes the Kafka handler over the
declared tumbling window in `source.type: kafka` and aggregates
per-record outcomes inside the kind handler at window close
(ADR-0024, ADR-0026). The `execution_id` formula is unchanged across
modes; record-mode commits the window endpoints declared by the rule
into the formula (ADR-0024).

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

**Per-mode behavior.** Both modes write to the same `dq_executions`
and `dq_check_results` tables under the same `execution_id` formula
(ADR-0003, ADR-0041). An additive `mode` column distinguishes the
two. Window endpoints are not yet first-class columns; they are
reconstructed from the rule version pinned in `dq_executions` and
will be promoted to first-class columns in a later T1 extension
(ADR-0041). Per-mode evidence shapes differ: set-mode samples
violating rows; record-mode samples violating records with bounded
sample size per cost guardrails (ADR-0027, ADR-0031).

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

**Per-mode behavior.** The routing contract (ADR-0006) is identical
across modes; per-mode differences are payload-only (evidence shape).
The onboarding-channel override (ADR-0046) applies uniformly. Alert
egress remains Pub/Sub for both modes — Kafka is a source substrate,
not an egress substrate.

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

The same flow applies to both modes. The only divergences are at
step 4: in set-mode the compiler emits a BigQuery query that the
runner executes against the declared partition window; in record-mode
the runner consumes from the rule's Kafka topic within the declared
tumbling watermark window and aggregates per-record outcomes inside
the kind handler at window close (ADR-0024, ADR-0025, ADR-0026).
Every other step — plan persistence, result write, event emission,
fan-out — is mode-agnostic.

### Evolution Flow

Stream evaluation is no longer a future direction: record-mode is a
declared capability backed by a Kafka source substrate (ADR-0023,
ADR-0028) and a unified runner with per-mode switch (ADR-0025). New
kinds and new modes evolve under the same contract-driven discipline
(P5): kinds enter through the catalog (ADR-0022) with schema
co-versioning, and any new mode would arrive as another value of the
`mode` discriminator under a schema-version bump.

---

## Internal Patterns

The platform adopts several internal patterns that recur across
components. Describing them here once avoids re-explaining the same
shape in every component description.

**Naming.** Internal patterns are prefixed `PAT-` to avoid collision
with the platform principles `P1`–`P6` defined in
[`01-charter-and-principles.md`](./01-charter-and-principles.md).
Principles constrain *what we will not compromise*; patterns describe
*how recurring shapes inside the engine are built*. The two are
distinct concepts and should not share a numbering scheme.

### PAT-1 — Fail-fast registry loading

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

The kind catalog (`engine/internal/dsl/catalog/v1.yaml`, ADR-0022) is
co-loaded with the manifest and rules; an unknown kind, a missing
catalog entry, or a kind/mode-prefix mismatch is a fail-fast load
error. Refusal-to-swap applies identically to both modes.

### PAT-2 — Lifecycle-aware scheduler integration

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

The scheduler is mode-agnostic: cadence lives on the rule YAML and
applies uniformly to set-mode and record-mode rules (ADR-0033). The
catch-up horizon and missed-trigger detection are likewise mode-
neutral.

### PAT-3 — Fixture-driven compiler testing

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

Per-mode fixtures partition the testdata tree under `set/` and
`record/` subtrees (target shape per ADR-0022 / ADR-0023). Set-mode
scenarios compare generated SQL; record-mode scenarios compare
per-record outcomes against synthetic Kafka inputs plus the
aggregated window result.

### PAT-4 — Typed multi-environment configuration

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

`EnvConfig` carries the per-mode cost guardrails: `SetModeCost`
(MaxBytesScannedPerRun, MaxWindowDuration, MaxConcurrentEvaluations,
MaxEvidenceSampleSize, RefreshFailureEscalationN — ADR-0029) and
`RecordModeCost` (consumer lag, late-drop rate, dead-letter rate,
evidence sample size, writer-queue saturation, throughput —
ADR-0027). It also carries `OnboardingChannel` for the
onboarding-channel override (ADR-0046).

### PAT-5 — Modular logging contract

Logging supports a global default level plus per-package overrides
via an environment variable:

```
DQ_LOG_LEVELS="root:INFO,engine.compilers:DEBUG,engine.scheduler:WARN"
```

This lets maintainers raise verbosity in one subsystem without
flooding logs globally — especially useful when debugging compilers,
scheduler reconciliation, manifest loading, or alert emission.

The implementation is a small layer over Go's `slog` standard library
package. It is intentionally minimal. The `DQ_LOG_LEVELS` grammar
(`PACKAGE:LEVEL`, comma-separated, case-insensitive) is fixed by
ADR-0043.

### PAT-6 — Capability discriminator

Mode (`set` or `record`) is the first-class architectural
discriminator. It is declared on the rule and the entity, it appears
as a prefix on every kind name (`set.*` / `record.*`), and it
constrains the shape of the source declaration.

The linter cross-checks mode-consistency across four artifacts:

- the rule's `mode` field;
- the entity's `mode` field;
- the kind name's prefix and the kind catalog's `mode` column;
- the source declaration (BigQuery for `set`, Kafka topic + consumer
  group for `record`).

Any mismatch fails lint before the engine ever sees the rule. This
pattern is what keeps the unified runner (PAT-7) safe: every rule
that reaches the engine has its mode locked at the YAML layer.

Cite ADR-0021 and ADR-0022.

### PAT-7 — Per-mode runner switch

A single `Runner` component dispatches by mode. There is no parallel
set-runner and record-runner — there is one runner with a per-mode
handler.

- **Set-mode handler.** Invokes the compiled BigQuery query over the
  declared partition window; result is one row per check.
- **Record-mode handler.** Consumes from the rule's Kafka topic
  inside the declared tumbling watermark window; per-record outcomes
  are aggregated inside the kind handler at window close per the
  rule's threshold or override (`params.aggregation`); result is one
  row per check per window.

The unified-runner shape is committed by ADR-0025 with an objective
criterion for future revisits. Aggregation lives inside the kind
handler in both cases — implicitly in set-mode (the SQL itself
aggregates), explicitly in record-mode (the handler tallies per-
record outcomes and maps the violation rate to the ADR-0004 enum
per ADR-0026).

Cite ADR-0025 and ADR-0026.

### PAT-8 — Kafka stream substrate

Record-mode sources are Kafka topics with explicit consumer groups
declared in `source.type: kafka`. Window semantics are tumbling and
watermark-bounded; window endpoints are declared by the rule and
committed into the `execution_id` formula (ADR-0024). Partition
offsets drive deterministic replay — a rerun of the same window
reads the same records.

The substrate capability matrix (ADR-0010, amended by ADR-0028)
carries Kafka rows alongside the existing BigQuery / GCS / Pub/Sub
rows: publish/subscribe and consumer groups are **Yes** under the
local emulator and the sandbox; per-partition offset tracking is
**Yes**; watermark/lateness is **Partial** (full fidelity requires
the sandbox tier).

Pub/Sub remains the alert and result-event egress substrate (Layer
5). The asymmetry is intentional and is explained in the charter
under "Capability Modes → Stream substrate".

Cite ADR-0023, ADR-0024, and ADR-0028.

---

## Component Inventory

The following components live inside `engine/`. Each has a focused
responsibility.

### `Loader`

Reads the active manifest, fetches and validates rules, indexes by
entity. Implements PAT-1.

### `Scheduler`

Owns the scheduled trigger lifecycle. Implements PAT-2.

### `Trigger API`

HTTP handler for `/v1/trigger`. Validates OIDC, parses request with a
strict decoder (UTF-8, no-pipe, RFC 3339 for timestamps), enqueues
plan. Accepts `trigger_source` distinguishing scheduler from manual
invocation. Returns a DTO distinct from the persistence shape. Owns
`/healthz` and `/readyz`. Contract per ADR-0014.

### `Coordinator`

Creates execution plans from a trigger request and an active ruleset.
Owns plan-level concurrency limits.

### `Compilers`

One per check type. Transform declarative check specs into BigQuery
queries. Implement PAT-3.

### `Runner`

Unified runner with per-mode switch (PAT-7). Set-mode handler
executes compiled queries against BigQuery and captures sample
violating rows; record-mode handler consumes from Kafka inside the
declared tumbling watermark window and aggregates per-record
outcomes at window close. Captures bounded evidence under the
per-mode cost guardrails in `EnvConfig` (PAT-4). Honors concurrency
budgets.

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

Every check type must have a named compiler in
`engine/internal/compilers/` and a documented contract in
`docs/dsl/`. No hidden "temporary" type, no "custom" type, no SQL
expression escape hatch in the DSL.

Adding a new check type is a deliberate process:

1. Propose an ADR.
2. Update the schema (`v1` if additive, `v2` if breaking).
3. Add the compiler with full fixtures.
4. Add documentation.
5. Add an example to `rules/_examples/`.

The closed-catalog rule is what makes the platform predictable to
operate and safe to evolve.

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

Stream evaluation is realized, not future: record-mode is a declared
capability backed by a Kafka source substrate (ADR-0023, ADR-0028)
and a unified runner with per-mode switch (ADR-0025). Conceptual
continuity is the constraint that keeps it safe to operate alongside
set-mode:

- the same entity concept across both modes;
- the same check identifiers (kind names differ only by mode prefix);
- the same severity semantics and ADR-0004 outcome enum, with
  per-record outcomes mapped to that enum via aggregation
  (ADR-0026);
- the same `dq_executions` / `dq_check_results` tables and the same
  `execution_id` formula, with an additive `mode` column
  (ADR-0003, ADR-0041);
- the same alert routing contract (ADR-0006).

Any future mode must enter under the same continuity rule.

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
- environment-specific behavior changes not declared in `env/`;
- treating record-mode as a parallel pipeline. The two modes share
  the same control plane, manifest semantics, kind catalog, result
  store, alert routing, and `execution_id` formula. A change that
  would only apply to one mode is a signal to rethink — it usually
  belongs inside the kind handler, not at the runner or reporting
  layer.

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
