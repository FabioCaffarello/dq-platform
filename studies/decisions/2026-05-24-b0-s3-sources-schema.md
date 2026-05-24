<!-- path: studies/decisions/2026-05-24-b0-s3-sources-schema.md -->

# B0-S3 — Sources Schema

## Metadata

- **B-item reference:** B0-S3 (Wave-S foundational triplet, item 3 of 3)
- **Status:** resolved-study (Wave-S, B0-S3; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-24
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch — locks P1–P4 and assigns B0-S3's scope);
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) (B0-S1 — mode
  primitive, v2 schema, loader mode-field filter);
  [ADR-0022](../../docs/adr/0022-kind-catalog.md) (B0-S2 — kind
  catalog, per-kind `source_mode` declaration);
  [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
  (engine ↔ rules compatibility contract);
  [ADR-0007](../../docs/adr/0007-loader-scheduler-retry-failure-semantics.md)
  (set-mode loader contract);
  [ADR-0010](../../docs/adr/0010-substrate-posture.md) (substrate
  posture — currently set-mode and alerting-side only).
- **Downstream open:** B0-S4 (window semantics — consumes Kafka
  offset/watermark binding committed here at the identifier level;
  B0-S4 commits the *semantics* of watermarks and offsets).
- **Promotion target:** `docs/adr/0023-sources-schema.md` (subject to
  ADR-0020 §Decision (Per-item ADR numbering); `0023` reflects the
  expected sequence S1 → 0021, S2 → 0022, S3 → 0023).
- **Loop discipline:** same as B0-S1 and B0-S2 — `/resolve-b0` study
  → `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.
- **Significance:** B0-S3 is the final item of the foundational
  triplet. Its promotion meets the **partial-Wave-S gate** per
  [ADR-0020 §Decision (Sequencing and gates)](../../docs/adr/0020-wave-s-launch.md);
  the engine loader's `mode: record` rejection error path (per
  ADR-0021 §Decision (Engine loader behaviour)) unblocks once the
  partial gate closes.

---

## Context

ADR-0020 launched Wave-S with mode as the architectural primitive
(P1), the `set.*` / `record.*` kind-prefix discipline (P2),
capability derived from mode (P3), and the unified-vs-parallel
runner question deferred to B0-S5 (P4). ADR-0021 realised the
mode primitive in schema shape (v2 schema with typed `mode` and
prefixed `kind`); ADR-0022 realised the catalog (v1 kinds:
`set.row_count_positive` and `record.schema_conformance`, with
`source_mode` declared per kind). The foundational triplet is
**2/3 closed**; B0-S3 closes the last item and opens the
partial-Wave-S gate.

The current state of source declaration is **fully implicit**:

- The shipped rule (`rules/customer.yaml`) carries no `source:`
  field. The engine derives its BigQuery source from environment
  configuration (`engine/internal/env/{local,qa,prod}.go` —
  `SourceProject` / `SourceDataset` constants per environment,
  picked at boot via the ADR-0018 environment-selector).
- The rule's `entity:` field doubles as the BigQuery table name;
  the engine concatenates env config + entity to construct the
  fully-qualified table reference.
- Partition column for ADR-0007's pre-check is hardcoded inside
  the evaluator implementation, not declared per rule.

This implicit shape contradicts **P1** (rules remain declarative
— no escape hatches, no behaviour driven by mutable repository
state) and blocks B0-S3's scope: a record-mode rule cannot derive
its source from env config because env config carries no
record-substrate identifier. The source declaration must move
into the rule YAML, with mode-specific shapes that lint can
validate against the catalog's per-kind `source_mode`.

ADR-0020 §B0-S3 commits the scope:

> Decides how a **source** is described in rule YAML in each mode
> — a set source (a BigQuery table or view with its partition
> column, table-ref, and dataset-ref, as ADR-0007 already
> presumes) versus a record source (a stream substrate's
> topic/subscription identifier with its watermark and
> offset-binding semantics, the specific substrate to be picked
> during the B0-S3 study under mode-derived capability per P3);
> how the source declaration cross-checks against the kind
> catalog (S2) so that a `record.*` rule cannot reference a
> BigQuery table; how the ADR-0007 loader path extends to
> recognise record-mode sources without re-opening ADR-0007's
> set-oriented contract.

Four interlocking sub-decisions live inside this scope:

1. **Where the source declaration lives** in the rule YAML —
   rule-level, per-check, entity-level, or implicit.
2. **The set-source shape** — BigQuery table/view, dataset, project,
   partition column.
3. **The record-source shape** — substrate pick, topic/subscription
   identifier, consumer-group binding (watermark/offset semantics
   deferred to B0-S4).
4. **Cross-check + loader extension** — how lint enforces source ↔
   catalog consistency; how ADR-0007's loader extends to recognise
   the new `source:` field without reopening its set-oriented
   contracts.

The platform principles relevant here live in
[`studies/foundation/01-charter-and-principles.md`](../foundation/01-charter-and-principles.md):
**P1** (declarative — source declaration belongs in the rule
artefact); **P3** (ownership / declaration explicit — no
record-mode source without a substrate identifier in the rule);
**P5** (evolution contract-driven — source schema is a contract
surface between engine and rules); **P6** (borrow patterns, not
baggage — substrate pick is judged on fit, not external
provenance). The Wave-S framing in
[`studies/foundation/04-system-architecture.md`](../foundation/04-system-architecture.md)
§G5 ("Stream evolution preserves conceptual continuity") also
bears: the record substrate must integrate without breaking the
existing set-mode contracts.

---

## Decision Drivers

- **DD-S3.1** — **Honour the ADR-0020 locked premises.** P1
  (mode primitive), P2 (kind prefix), P3 (capability derived from
  mode), and P4 (runner shape deferred to B0-S5) constrain the
  source-schema design. The source declaration must be part of
  the declarative DSL; capability (which substrate the source
  points to) must derive from mode + kind, not from a separately-
  declared substrate field.

- **DD-S3.2** — **Honour ADR-0007's set-mode loader contract.**
  ADR-0007 currently presumes a BigQuery source with partition
  column for pre-check. The extension must be additive on the
  set side: a set source declared in the rule replaces today's
  implicit env-derived source, but ADR-0007's startup-mode,
  refresh-mode, retry budget, orphan-detection, and pre-check
  contracts are not reopened.

- **DD-S3.3** — **Honour ADR-0021 + ADR-0022's v2 schema and
  catalog.** The source declaration sits as a new top-level field
  in the v2 schema, alongside `version`, `entity`, `mode`, and
  `checks`. Lint cross-checks the source's mode against the
  rule's `mode` field (from ADR-0021) and against the catalog's
  per-kind `source_mode` (from ADR-0022).

- **DD-S3.4** — **Record-mode substrate is Kafka.** Three
  convergent arguments support this pick, in order of weight:
  **(1) Project principle.** Stream-validation data **always**
  comes from Kafka; Pub/Sub is reserved for alerting events
  (ADR-0006) and never serves as input for record-mode
  validation. The platform's substrate planes are deliberately
  separated — data ingress vs. alert egress live on different
  substrates, and ADR-0006 + ADR-0010 commit Pub/Sub to the
  alerting plane exclusively.
  **(2) Protocol-fit (forward-looking, until B0-S4 commits).**
  Kafka's consumer-group + per-partition-offset model gives the
  ADR-0002 `execution_id` binding a natural anchor for the
  windowing and offset semantics B0-S4 will commit (resumable
  evaluation, deterministic replay against a fixed offset range,
  watermark-bounded windows over partitioned topics). Pub/Sub's
  at-least-once subscription model would force B0-S4 to design
  around no-replay semantics, which is a poorer fit for the
  determinism principle (P2 mirror, foundation 01
  §"Determinism"). This clause is provisional until B0-S4
  finalises offset/watermark semantics; if B0-S4 surfaces a
  concrete reason the Pub/Sub model fits better, this driver is
  revisited.
  **(3) Chartering alignment.** Foundation 01 ("Controlled
  evolution toward stream-compatible checks over Kafka") and
  foundation 04 §G5 name Kafka explicitly as the stream
  substrate. This is supporting context, not the load-bearing
  argument — the project-principle and protocol-fit arguments
  above stand on their own.

- **DD-S3.5** — **Substrate matrix extension is additive, not
  reopening.** ADR-0010's substrate matrix has rows for tabular
  store, object store, Pub/Sub publish-subscribe, OIDC, and
  container registry — but no Kafka row. Adding a Kafka row is
  additive (follows ADR-0010's contract; mirrors the ADR-0017
  amendment pattern that added the CAS-fidelity note). The
  amendment lives in a separate forthcoming ADR (the next
  available number after ADR-0023 — likely ADR-0024); this study
  flags the amendment as a downstream consequence but does not
  pre-write it.

- **DD-S3.6** — **Watermark and offset semantics live in B0-S4.**
  ADR-0020 §B0-S4 commits "what 'window' means for record-mode"
  and "how watermarks interact with (or replace) the ADR-0002
  `execution_id` window-endpoint formula". B0-S3 commits the
  **identifier shape** of a record source (topic + consumer-group);
  the *semantics* of watermarks and offsets (closed-window
  invariant, late-arrival handling, identity-formula extension)
  are B0-S4's responsibility.

- **DD-S3.7** — **One source per rule (rule-level, not
  per-check).** A rule artefact has one `entity:` and one
  `mode:`; ADR-0022's catalog gives each kind a single
  `source_mode`. A rule with mixed-mode checks contradicts
  ADR-0021's cross-checks; a rule with multiple sources (even
  same-mode) would require a join semantic that doesn't exist in
  the declarative surface. The source is therefore declared once
  at the rule's top level; every check in the rule consumes the
  same source via the catalog's `source_mode` cross-check.

- **DD-S3.8** — **Schema dispatch on `source.type`, not on
  rule's `mode`.** The source object carries a `type:`
  discriminator (`type: bigquery` for set; `type: kafka` for
  record). The schema's source variant is selected by `source.type`,
  not by the rule's `mode`. This decouples the source-shape
  bifurcation from the mode-bifurcation, making future additions
  (e.g., a second set-mode source type — view-only, or a second
  record-mode source over a different log protocol —
  alternative streaming substrate that respects DD-S3.4's
  data-ingress-vs-alert-egress separation) additive within the
  same schema layer. Lint cross-checks enforce consistency
  between `source.type` and the rule's `mode` (and via the
  catalog, the kind's `source_mode`). *(New contribution
  proposed here, requires review.)*

- **DD-S3.9** — **Loader extension is schema-aware only.** The
  loader (ADR-0007) parses the new `source:` field as part of v2
  schema dispatch (already committed by ADR-0021). The
  source-fetching layer that actually reaches BigQuery or Kafka
  is mode-specific runtime code, not loader concern. The loader's
  startup-mode, refresh-mode, retry budget, and orphan-detection
  contracts (ADR-0007) are unchanged.

---

## Considered Options

The four options below differ on **where the source declaration
lives** in the rule YAML. All four assume the source-shape itself
(Kafka for record, BigQuery for set, with `type:` discriminator
per DD-S3.8) is the same — that's locked by DD-S3.4 and DD-S3.8.

### Option A — `source:` at rule top level, applies to all checks (recommended)

**Shape.** A new top-level `source:` object lives in the rule
YAML alongside `version`, `entity`, `mode`, and `checks`. The
object carries a `type:` discriminator and type-specific fields.
Every check in the rule consumes the rule's single source via the
catalog's per-kind `source_mode` cross-check.

```yaml
# Set rule shape (R1: code-shaped illustration only)
version: 2
entity: customer
mode: set
source:
  type: bigquery
  project_id: dq-prod-PLACEHOLDER
  dataset_id: dq_source_prod
  table_id: customer
  partition_column: ingested_at  # optional
checks:
  - check_id: row_count_positive
    kind: set.row_count_positive
```

```yaml
# Record rule shape (illustration)
version: 2
entity: customer_events
mode: record
source:
  type: kafka
  topic: customer_events
  consumer_group: dq-engine-customer-events
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema:
        type: object
        required: [id, event_type]
```

**Cost.** Authors declare the source once per rule. Multi-check
rules share the source — consistent with the "rule = one entity
= one mode" framing locked by ADR-0021 and ADR-0022. The
top-level `source:` field is required; v1 rules without it are
rejected by the v2 schema (which already happens because v1 is
retired at ADR-0021's migration).

**Verdict.** Recommended.

### Option B — `source:` per check (per-check source)

**Shape.** The `source:` object moves from rule top level to
inside each `check` object. Each check carries its own source,
independent of sibling checks.

**Cost.** A rule with multiple checks could declare different
sources per check. The current "rule = one entity = one mode"
framing breaks — sibling checks could point at different
entities (different tables, different topics). The catalog's
per-kind `source_mode` cross-check still works, but the
entity-level coherence that ADR-0021's lint cross-checks
guarantee is undermined. Authors face N source declarations per
N-check rule, with no clear benefit when all checks target the
same entity.

**Verdict.** Rejected. Multi-source rules are an edge case that
contradicts P1+P2's rule-as-coherent-unit framing.

### Option C — `source:` in `_owners.yaml` (entity-level)

**Shape.** The source declaration moves out of the rule YAML and
into the entity's `_owners.yaml` entry. The rule references the
entity (which it already does via the `entity:` field); the
engine looks up the entity in `_owners.yaml` to find the source.

**Cost.** Rule files become silent on source — a reviewer reading
a rule cannot answer "what BigQuery table is this rule
evaluating?" without consulting `_owners.yaml`. The rule loses
the "read cold" property that B0-S1's Option C analysis
rejected. The `_owners.yaml` schema also grows in scope: today
it carries `owner`, `channels`, `severity_overrides`, `mode`;
adding `source` mixes routing/governance concerns with
data-source concerns.

**Verdict.** Rejected on the same grounds B0-S1 rejected the
analogous mode-at-entity-only shape: the rule artefact must read
cold.

### Option D — Implicit (status quo, env-derived source)

**Shape.** Keep today's behaviour: source is in
`engine/internal/env/{local,qa,prod}.go`, derived per-environment
at engine boot. Rules carry no `source:` field.

**Cost.** Contradicts P1 (declarative). Cannot extend to record
mode (env config has no Kafka topic/consumer-group fields, and
adding them perpetuates the contradiction). Blocks B0-S3's
scope: the catalog's `record.schema_conformance` entry expects a
record source, but a record source cannot be derived from env
config without each environment carrying every record-mode rule's
topic — which is `O(rules × envs)` and untenable.

**Verdict.** Rejected on P1 grounds. Was the Wave-3 expedient;
not sustainable for Wave-S.

---

## Recommendation

**Pick Option A — `source:` at rule top level, applies to all
checks.**

Rationale, tied directly to drivers:

- **DD-S3.1 (locked premises).** Source declaration in the rule
  yaml satisfies P1 (declarative); the rule's `mode` and the
  source's `type` together determine substrate without an
  independent capability field (P3); the runner shape (P4) is
  unaffected — both unified and parallel runners consume the
  same `source:` field.
- **DD-S3.2 (ADR-0007 honoured).** The set side of the source
  declaration carries exactly the fields ADR-0007 presumes
  (project, dataset, table, partition column). ADR-0007's loader
  contracts are unchanged.
- **DD-S3.3 (v2 schema and catalog).** The new `source:` field
  is the third addition to the v2 schema (after ADR-0021's `mode`
  and ADR-0022's `params:` per check). Lint cross-checks
  `source.type` against the rule's `mode` (via DD-S3.8's
  bridge) and against the catalog's per-kind `source_mode`.
- **DD-S3.4 (Kafka substrate).** Record sources are Kafka; matches
  the chartering language and keeps Pub/Sub scoped to the
  alerting plane.
- **DD-S3.5 (substrate matrix amendment).** ADR-0010 gains a
  Kafka row via a forthcoming amendment ADR (likely 0024);
  pattern mirrors ADR-0017.
- **DD-S3.6 (watermark/offset deferred).** B0-S3 commits the
  topic + consumer-group fields; B0-S4 commits semantics. The
  record-source schema shape stays minimal — two fields — so the
  B0-S4 study can extend with optional fields without breaking
  v2.
- **DD-S3.7 (rule-level source).** One source per rule, shared by
  all checks. Multi-source rules are explicitly out of scope.
- **DD-S3.8 (type discriminator).** `source.type` decouples
  source-shape from mode-shape. Future-proofs against
  same-mode-multiple-source-types additions.
- **DD-S3.9 (loader schema-aware only).** Loader parses the new
  field; the source-fetching layer is mode-specific runtime code
  that lands at implementation, not at this ADR's promotion.

**Schema dispatch — alternatives considered.** Three designs were
considered for how the rule schema bifurcates set-source vs
record-source shapes:

- **Sibling top-level keys** (`source_bigquery:` and
  `source_kafka:` as separate optional top-level fields).
  Rejected: two of three keys are always empty, and lint must
  enforce exactly-one-non-empty across the pair. The
  mode-discriminator redundancy with `mode:` is replaced by a
  key-presence discriminator, which is harder to validate
  cleanly in JSON Schema and less ergonomic for authors.
- **JSON Schema `if/then/else` conditional on `mode`** — the
  `source` object's shape is selected by the rule's `mode`
  field via JSON Schema's conditional construct. Rejected: ties
  source-shape dispatch tightly to the mode bifurcation,
  blocking the future-proofing that DD-S3.8 buys (e.g., a
  future record-mode source over an alternative streaming
  substrate that respects DD-S3.4's separation).
- **Type discriminator** (`source.type` as a const value per
  variant; recommended). Selected: cleanly expressed in JSON
  Schema via `oneOf` with `properties.type` const constraint;
  ergonomic for authors (one field per source); supports future
  same-mode variants without reopening the mode-bifurcation;
  lint cross-checks #7 and #8 enforce the `source.type` ↔
  `mode` and `source.type` ↔ catalog `source_mode` consistency
  that the discriminator's looseness requires.

**Set-source shape** (Option A's `source.type: bigquery`):

| Field | Required | Type | Description |
|---|---|---|---|
| `type` | yes | const `bigquery` | Discriminator |
| `project_id` | yes | string | BigQuery project (matches ADR-0018 env-selector default if omitted? — see C-B0S3.4) |
| `dataset_id` | yes | string | BigQuery dataset |
| `table_id` | yes | string | BigQuery table or view name |
| `partition_column` | no | string | Partition column for ADR-0007's pre-check; absent for non-partitioned views |

**Record-source shape** (Option A's `source.type: kafka`):

| Field | Required | Type | Description |
|---|---|---|---|
| `type` | yes | const `kafka` | Discriminator |
| `topic` | yes | string | Kafka topic name |
| `consumer_group` | yes | string | Engine's consumer-group ID for this rule |

The Kafka source has no broker-list / bootstrap-server field;
that's an environment-configuration concern per ADR-0018 (the
forthcoming Kafka row in ADR-0018's env config), not a per-rule
concern. Authentication, schema-registry integration, and SSL
configuration are out of scope (see Open Questions).

**Lint cross-checks added by this study** (on top of ADR-0021's
four and ADR-0022's #5–#6):

- **#7 — source.type matches rule.mode.** A rule with `mode: set`
  must declare `source.type: bigquery`; a rule with `mode: record`
  must declare `source.type: kafka`. The lookup table is small
  and explicit; future source types extend the table.
- **#8 — source.type matches the catalog's per-kind
  source_mode.** For every check in the rule, the catalog entry's
  `source_mode` must equal the rule's `source.type` mode (via the
  same lookup table from #7).

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Source declaration location | Rule top level, single source per rule |
| Set source shape | `source.type: bigquery` with project/dataset/table/(optional partition_column) |
| Record source shape | `source.type: kafka` with topic/consumer_group; offset/watermark deferred to B0-S4 |
| Schema dispatch | `source.type` discriminator (decoupled from rule's `mode`) |
| Lint cross-checks added | #7 (source.type ↔ rule.mode) + #8 (source.type ↔ catalog source_mode per check) |
| Loader extension | Parses `source:` field via v2 schema dispatch (no behaviour change) |
| Substrate matrix impact | ADR-0010 gains a Kafka row via a forthcoming amendment ADR |
| Watermark/offset semantics | Deferred to B0-S4 |
| Migration | `rules/customer.yaml` gains `source:` at the combined implementation commit; env config retains `SourceProject`/`SourceDataset` as backstop defaults for OQ-B0S3.5 resolution |

---

## Consequences

### Cross-cutting consequences

- **C-B0S3.1** — **Rule schema v2 gains a required `source:`
  top-level field.** The v2 schema's top-level `required` array
  becomes `[version, entity, mode, source, checks]`. The
  `additionalProperties: false` constraint is preserved. The
  `source` object has a discriminated-union shape via the
  `type:` field (`bigquery` for set sources, `kafka` for record
  sources). This adds to ADR-0021's + ADR-0022's combined v2
  schema commitments; the same combined-implementation-commit
  contingency from ADR-0022 §C-B0S2.2 applies — single combined
  commit landing all three (ADR-0021 + ADR-0022 + ADR-0023)
  artefacts together is the preferred path; v3 schema bump is
  the contingency if any prior v2 artefact ships first.
  *(New contribution proposed here, requires review.)*

- **C-B0S3.2** — **Lint binary gains catalog-driven source
  cross-checks #7 and #8.** Added on top of ADR-0021's four and
  ADR-0022's #5–#6. Lint rejects rules whose `source.type` mode
  disagrees with the rule's `mode` (#7) or with the catalog's
  per-kind `source_mode` (#8). A `mode: record` rule that
  declares `source.type: bigquery` fails #7; a `mode: set` rule
  whose check uses a kind whose catalog entry says
  `source_mode: record` fails #8.

- **C-B0S3.3** — **ADR-0010 substrate matrix gains a Kafka row
  via a forthcoming amendment.** The amendment ADR (likely
  numbered 0024, subject to the same per-item-ADR-numbering
  caveat ADR-0020 carries) lands at the combined implementation
  session that wires Kafka into the local Compose stack. The
  Kafka row commits to a substrate-capability declaration
  (topic publish/subscribe; consumer-group binding; offset
  retention) following ADR-0010's matrix shape. This study does
  not pre-write the amendment; it flags it as the required
  follow-up. *(New contribution proposed here, requires review.)*

- **C-B0S3.4** — **`rules/customer.yaml` migration extends to
  carry a `source:` block, local-realised.** The atomic v1→v2
  migration committed by ADR-0021 §Decision (Atomic migration)
  and extended by ADR-0022 §Decision (Rule schema v2 extension)
  now also adds: `source: { type: bigquery, project_id: dq-local,
  dataset_id: dq_fixture, table_id: customer }`. The migration
  carries the **local-environment values** explicitly (mirroring
  `engine/internal/env/local.go`'s existing `SourceProject:
  "dq-local"`, `SourceDataset: "dq_fixture"`); the `table_id`
  matches the entity name (preserving today's implicit
  table=entity behaviour explicitly). The combined implementation
  commit lands the migration atomically. **qa / prod values are
  gated by OQ-B0S3.7's resolution** — until per-env defaults or
  template substitution is committed, the rule is realised for
  local only, and qa/prod operation requires either a per-env
  overlay mechanism (OQ-B0S3.7's future enrichment) or per-env
  rule files (a worse path, kept on the table only as a
  fallback). `project_id` and `dataset_id` remain **required**
  in the v2 schema; the local-first migration is the
  minimum-viable starting state, not a relaxation of the
  schema's required-field commitment.

- **C-B0S3.5** — **The engine env config's `SourceProject` and
  `SourceDataset` constants are removed at the combined
  implementation commit.** With ADR-0023 promoted, the rule's
  `source:` field is authoritative and the v2 schema requires
  `source.project_id` and `source.dataset_id` on every rule.
  The env constants serve no purpose at the schema level —
  every rule (including smoke-test rules) carries its own
  values. `engine/internal/env/{local,qa,prod}.go`'s
  `SourceProject` and `SourceDataset` fields are removed at the
  combined implementation commit; smoke tests are refactored to
  use rule-level values directly. *(New contribution proposed
  here, requires review.)*

- **C-B0S3.6** — **The partial-Wave-S gate opens at this ADR's
  markdown landing; loader rejection error is removed at the
  combined implementation commit.** Per ADR-0020 §Decision
  (Partial-Wave-S gate), the gate is met when B0-S1, B0-S2, and
  B0-S3 are all at `resolved-adr` and their ADRs are merged
  into `docs/adr/`. ADR-0021 is merged; ADR-0022 is merged;
  this study's promotion lands ADR-0023, which is the
  gate-closing event. The gate opens at that markdown landing
  (the spec surface). The engine loader's `mode: record`
  rejection error path (ADR-0021 §Decision (Engine loader
  behaviour)) is removed in the **same combined implementation
  commit** that lands the schema artefacts (per C-B0S3.1's
  path-(a) preference). Between the ADR-0023 markdown landing
  and the combined implementation commit, the gate is **met in
  spec but the loader still rejects record-mode in code** —
  record-mode rules cannot ship until both events have
  occurred. After the combined implementation commit, the
  Wave-S analogue of R1 from ADR-0020 §C-S.4 is fully
  satisfied: record-mode code may ship.

- **C-B0S3.7** — **Source-fetching layer is mode-specific runtime
  code, landing at implementation.** The loader (ADR-0007)
  parses `source:` via v2 schema dispatch but does not reach
  BigQuery or Kafka itself. The source-fetching layer lives
  **near the evaluator boundary; exact package layout (whether
  under `engine/internal/eval/` alongside check handlers, or in
  a new `engine/internal/source/` package separate from
  evaluation) is decided at the combined implementation
  commit**. Set-mode evaluators use the existing BigQuery query
  path; record-mode evaluators (currently only
  `record.schema_conformance`) gain a new Kafka consumer path.
  The Kafka consumer implementation, the local Compose Kafka
  emulator addition, and the ADR-0010 amendment land in the
  combined implementation session.

- **C-B0S3.8** — **B0-S4 inherits the Kafka identifier shape as
  the substrate for windowing decisions.** With Kafka committed,
  B0-S4's windowing options narrow: tumbling, sliding, session,
  watermark-bounded windows over Kafka topics with consumer-group
  offset state. The `record.schema_conformance` per-record kind
  bypasses windowing entirely; richer record kinds added in
  future catalog versions consume B0-S4's windowing semantics.

### Per-artefact consequences

- **`engine/internal/dsl/schema/v2.schema.json`** (committed by
  ADR-0021, extended by ADR-0022, further extended here) —
  top-level `properties` gains `source` (object, discriminated
  union via `type`); top-level `required` adds `source`.

- **`rules/_schema/v2.schema.json`** — byte-equal mirror.

- **`engine/internal/dsl/catalog/v1.yaml`** (committed by
  ADR-0022) — unchanged by this ADR. The existing entries'
  `source_mode` field is now consumed by lint cross-check #8.

- **`tools/lint/`** — two new cross-checks (#7 + #8) on top of
  ADR-0021's four and ADR-0022's two; eight total.

- **`engine/internal/env/{local,qa,prod}.go`** — `SourceProject` /
  `SourceDataset` constants remain (per C-B0S3.5) but lose
  authoritative status; rule-level `source.project_id` /
  `source.dataset_id` are authoritative.

- **`engine/internal/eval/evaluator.go`** — gains record-mode
  dispatch path that consumes Kafka source (alongside the
  existing BigQuery dispatch). Implementation lands at the
  combined implementation commit.

- **`engine/internal/eval/record_schema_conformance.go`** — the
  handler implementation file from ADR-0022; lands now with the
  Kafka consumer path wired up (assuming the operator opts for
  the eager-handler-landing path per ADR-0022 OQ-B0S2.6, which
  is now the natural choice since the partial-Wave-S gate
  closes here).

- **`docs/adr/0024-substrate-posture-amendment-2.md`** (or
  similar slug; numbering subject to ADR-0020's per-item caveat)
  — forthcoming amendment ADR that adds the Kafka row to
  ADR-0010's substrate matrix. Lands at the combined
  implementation commit.

- **`rules/customer.yaml`** — gains `source:` block per C-B0S3.4.

- **`docker-compose.yml`** — gains a Kafka emulator service
  (single-broker is sufficient for local smoke tests; OQ-B0S3.2
  defers multi-broker configurations).

- **No engine runtime changes** beyond the source-fetching layer
  additions per C-B0S3.7 and the loader's v2 schema dispatch
  (already committed by ADR-0021). ADR-0002/0003/0004/0006/0007/
  0010 (excluding the Kafka-row amendment)/0014/0017 contracts
  are untouched.

---

## Open Questions

- **OQ-B0S3.1** — **Watermark and offset semantics for Kafka
  sources.** The `topic` + `consumer_group` identifier is
  committed here; whether watermark binding is committed at
  rule-create time, at consumer-start time, or per-window is
  B0-S4's question. The schema accommodates additive fields
  (e.g., `source.watermark_mode: continuous | windowed`) once
  B0-S4 commits semantics. *Defer to B0-S4.*

- **OQ-B0S3.2** — **Multi-broker Kafka clusters and high-
  availability configuration.** The schema does not carry
  broker-list / bootstrap-server fields per rule; brokers live
  in env config (ADR-0018 forthcoming Kafka row). Whether the
  engine's Kafka client supports multi-broker failover, SASL
  authentication, or TLS is an operational concern decided at
  the combined implementation commit. *Out of scope for current
  cycle.*

- **OQ-B0S3.3** — **Schema Registry integration.** Kafka topics
  often carry an associated schema registered in a Schema
  Registry. The `record.schema_conformance` kind currently uses
  `params.schema` (a JSON Schema fragment in the rule); Schema
  Registry integration would shift schema sourcing from the rule
  to the registry. Whether the catalog grows a
  `schema_source: rule | registry` field is a future kind
  catalog evolution. *Out of scope for current cycle;* the
  rule-embedded schema path is the minimum-viable shape.

- **OQ-B0S3.4** — **SSL / SASL authentication for Kafka
  consumers.** Both are environment-configuration concerns per
  ADR-0018; both are deferred to the combined implementation
  commit's Kafka-row amendment to ADR-0010 and ADR-0018. *Out
  of scope for current cycle.*

- **OQ-B0S3.5** — **`SourceProject` / `SourceDataset` env-config
  cleanup timing.** Per C-B0S3.5, the env constants become
  backstop defaults at this ADR's promotion. Whether they remain
  in `engine/internal/env/` indefinitely (as smoke-test
  fallbacks) or are removed in a follow-up cleanup is decided at
  implementation. *Out of scope for current cycle;* the
  conservative path keeps them.

- **OQ-B0S3.6** — **Set source `partition_column` validation
  against BigQuery's INFORMATION_SCHEMA.** Today the engine
  silently accepts a wrong partition column; ADR-0007 mentions
  pre-check but doesn't enforce. Whether B0-S3's source
  declaration triggers an engine-side pre-check that verifies
  the column exists and is a partition column on the named table
  is an operational hardening question. *Defer to a future
  ADR-0007 follow-up;* this study does not strengthen ADR-0007.

- **OQ-B0S3.7** — **Per-environment `source.project_id` /
  `source.dataset_id` defaults.** A rule that targets the
  `customer` entity in qa and prod may want
  `dq-qa-PLACEHOLDER` / `dq-prod-PLACEHOLDER` to be derived
  from `DQ_ENV` rather than hard-coded in the rule. The schema
  could support template substitution (e.g., `project_id: dq-${DQ_ENV}-…`)
  or environment-overlay rules. *Out of scope for current
  cycle;* the minimum-viable shape carries explicit values per
  rule. The per-env overlay is a future enrichment.

---

## Promotion target

**Target:** `docs/adr/0023-sources-schema.md`.

This study promotes to **ADR-0023** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0023 is the third per-item ADR of
the Wave-S foundational triplet (S1 → ADR-0021 landed, S2 →
ADR-0022 landed, S3 → ADR-0023 this study); per ADR-0020
§Decision (Per-item ADR numbering), the `0023` slot is
descriptive of the expected sequence and may shift if an
unrelated promotion lands between B0-S items.

**ADR-0023's promotion is the gate-closing event** for the
partial-Wave-S gate per ADR-0020 §Decision (Sequencing and
gates). At this ADR's promotion:

- Foundational triplet is 3/3 closed.
- Engine loader's `mode: record` rejection error path (ADR-0021
  §Decision (Engine loader behaviour)) is scheduled for removal
  at the combined implementation commit.
- Wave-S analogue of R1 (ADR-0020 §C-S.4) is satisfied:
  record-mode code may ship.
- B0-S4 through B0-S7 (Phase β) studies may open per ADR-0020
  §Decision (Sequencing rule — gate on promotion, not on study
  opening).

ADR-0023's promotion commit lands the artefacts committed in
§Consequences above:

1. The rule schema v2's third extension (the `source:` top-level
   field), folded into the combined v2 schema implementation
   alongside ADR-0021's `mode:` and ADR-0022's `params:`.
2. Two new lint cross-checks (#7 + #8) in `tools/lint/`.
3. The atomic v1→v2 migration extension for `rules/customer.yaml`
   (adds `source:` block per C-B0S3.4).
4. A forthcoming amendment ADR to ADR-0010 (Kafka row).
5. A Kafka emulator service in `docker-compose.yml`.
6. The Kafka consumer path in `engine/internal/eval/` (alongside
   the `record.schema_conformance` handler from ADR-0022).
7. The loader's `mode: record` rejection error path removal.

Per R8, the future ADR-0023 will be rewritten from this study,
not linked back to it.
