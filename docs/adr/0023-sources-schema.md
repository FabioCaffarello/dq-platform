<!-- path: docs/adr/0023-sources-schema.md -->

# ADR-0023 — Sources Schema

- **Status:** accepted
- **Date:** 2026-05-24

---

## Context

ADR-0020 launched Wave-S with mode as the architectural primitive
(P1), kind-prefix discipline (P2), capability derived from mode
(P3), and the unified-vs-parallel runner question reserved for
B0-S5 (P4). ADR-0021 realised the mode primitive in schema shape
(v2 with typed `mode` and prefixed `kind`). ADR-0022 realised the
catalog (v1 kinds: `set.row_count_positive` and
`record.schema_conformance`, with `source_mode` declared per kind).
This ADR closes the third item of the Wave-S foundational triplet
by committing how a **source** is declared per mode, picking the
**record-mode substrate**, and extending lint to cross-check rule
sources against the catalog.

The current state of source declaration is fully implicit. The
shipped rule (`rules/customer.yaml`) carries no `source:` field;
the engine derives its BigQuery source from environment
configuration (`engine/internal/env/{local,qa,prod}.go` —
`SourceProject` / `SourceDataset` constants per env, picked at
boot via the ADR-0018 env-selector). The rule's `entity:` field
doubles as the BigQuery table name. Partition column for
ADR-0007's pre-check is hardcoded inside the evaluator
implementation. This shape contradicts P1 — rules carry no
authoritative source declaration; source choice is "behaviour
driven by mutable repository state" the platform principles
forbid — and blocks Wave-S: a record-mode rule cannot derive its
source from env config because env config carries no
record-substrate identifier.

The four interlocking sub-decisions inside this ADR are: where
the source declaration lives in rule YAML; the set-source shape;
the record-source shape (substrate pick + identifier);
cross-check and loader-extension mechanics.

**Significance.** This ADR is the **gate-closing event** for the
Wave-S foundational triplet per ADR-0020 §Decision (Partial-Wave-S
gate). With B0-S1 / B0-S2 / B0-S3 all at `resolved-adr` and their
ADRs merged into `docs/adr/`, the partial-Wave-S gate is met in
spec; Phase β studies (B0-S4 through B0-S7) become draftable in
parallel per ADR-0020 §Decision (Sequencing rule).

---

## Decision

### Source declaration lives at the rule's top level

A new required top-level field `source:` is added to the rule
schema v2, alongside `version`, `entity`, `mode`, and `checks`.
The source applies to every check in the rule — multi-source
rules are out of scope; the "rule is one entity, one mode, one
source" framing follows the coherence framing locked by ADR-0021
and ADR-0022.

Per-check sources, entity-level sources (moving the declaration
into `_owners.yaml`), and the status-quo implicit env-derived
source are not adopted. Per-check sources break entity coherence
and force authors to repeat declarations; entity-level sources
remove the rule's "read cold" property (a reviewer would consult
a sibling file to know what the rule evaluates); the implicit
status quo contradicts P1.

### The record-mode substrate is Kafka

Three convergent arguments support this pick, in order of
weight:

- **Project principle.** Stream-validation data **always** comes
  from Kafka; Pub/Sub is reserved for alerting events (per
  ADR-0006) and never serves as input for record-mode validation.
  The platform's substrate planes are deliberately separated —
  data ingress and alert egress live on different substrates,
  and ADR-0006 + ADR-0010 commit Pub/Sub to the alerting plane
  exclusively.
- **Protocol-fit (forward-looking, until B0-S4 commits).** Kafka's
  consumer-group + per-partition-offset model gives the ADR-0002
  `execution_id` binding a natural anchor for the windowing and
  offset semantics B0-S4 will commit — resumable evaluation,
  deterministic replay against a fixed offset range, watermark-
  bounded windows over partitioned topics. Pub/Sub's
  at-least-once subscription model would force B0-S4 to design
  around no-replay semantics, which fits the determinism
  principle (foundation 01 §"Determinism") less cleanly. This
  clause is provisional until B0-S4 finalises offset/watermark
  semantics; if B0-S4 surfaces a concrete reason the Pub/Sub
  model fits better, this commitment is revisited under a new
  ADR.
- **Chartering alignment.** Foundation 01 ("Controlled evolution
  toward stream-compatible checks over Kafka") and foundation 04
  §G5 name Kafka explicitly as the stream substrate. This is
  supporting context, not the load-bearing argument — the
  project-principle and protocol-fit arguments stand on their own.

### Schema dispatch uses a `source.type` discriminator

The `source` object carries a required `type:` field as the
discriminator:

- `source.type: bigquery` — set-mode source. Fields:
  `project_id` (required), `dataset_id` (required), `table_id`
  (required), `partition_column` (optional — absent for
  non-partitioned views; used by ADR-0007's pre-check when
  present).
- `source.type: kafka` — record-mode source. Fields: `topic`
  (required), `consumer_group` (required). Watermark and
  offset-binding semantics are deferred to B0-S4 and may add
  optional fields to this shape additively.

Three schema-dispatch designs were considered:

- **Sibling top-level keys** (`source_bigquery:` and
  `source_kafka:` as separate optional fields). Rejected: two of
  three keys are always empty, and lint must enforce
  exactly-one-non-empty across the pair. Less ergonomic for
  authors; harder to validate cleanly in JSON Schema.
- **JSON Schema `if/then/else` conditional on rule's `mode`** —
  source shape selected via JSON Schema conditional construct
  on the rule's `mode` field. Rejected: ties source-shape
  dispatch tightly to the mode bifurcation, blocking
  future-proofing for same-mode source variants (e.g., a future
  record-mode source over a different log protocol that
  respects the substrate-plane separation).
- **Type discriminator** (`source.type` as a const value per
  variant; adopted). Cleanly expressed via JSON Schema `oneOf`
  with `properties.type` const constraint; ergonomic for authors;
  supports future same-mode variants without reopening the
  mode-bifurcation.

### Lint cross-checks #7 and #8

The lint binary (`tools/lint/`) gains two cross-checks on top of
ADR-0021's four (mode-typed on rule and entity, kind prefix
matches rule's mode, rule's mode matches entity's mode) and
ADR-0022's two (catalog membership, per-kind params validation):

- **#7 — `source.type` matches rule's `mode`.** A rule with
  `mode: set` must declare `source.type: bigquery`; a rule with
  `mode: record` must declare `source.type: kafka`. The lookup
  table is small and explicit; future source types extend the
  table.
- **#8 — `source.type` matches the catalog's per-kind
  `source_mode`.** For every check in the rule, the catalog
  entry's `source_mode` (from ADR-0022) must equal the rule's
  `source.type` mode (via the same lookup table from #7). A
  `mode: set` rule whose check uses a kind whose catalog entry
  declares `source_mode: record` fails #8.

### Loader extension is schema-aware only

The loader (per ADR-0007) parses the new `source:` field as part
of v2 schema dispatch (already committed by ADR-0021). The
loader does not reach BigQuery or Kafka itself; the
source-fetching layer is mode-specific runtime code that lands
at implementation. ADR-0007's startup-mode, refresh-mode, retry
budget, orphan-detection, and pre-check contracts are unchanged.

### Source-fetching layer location is deferred to implementation

The source-fetching layer lives near the evaluator boundary;
exact package layout — whether under `engine/internal/eval/`
alongside check handlers, or in a new `engine/internal/source/`
package separate from evaluation — is decided at the combined
implementation commit. Set-mode evaluators use the existing
BigQuery query path; record-mode evaluators (currently only
`record.schema_conformance` per ADR-0022) gain a new Kafka
consumer path.

### Migration is local-realised

The atomic v1→v2 migration committed by ADR-0021 and extended by
ADR-0022 is further extended here. `rules/customer.yaml` gains:

```yaml
source:
  type: bigquery
  project_id: dq-local
  dataset_id: dq_fixture
  table_id: customer
```

The migration carries **local-environment values** explicitly
(mirroring `engine/internal/env/local.go`'s existing
`SourceProject: "dq-local"`, `SourceDataset: "dq_fixture"`); the
`table_id` matches the entity name (preserving today's implicit
table=entity behaviour explicitly). The combined implementation
commit lands the migration atomically.

`project_id` and `dataset_id` are **required** in the v2 schema;
the local-first migration is the minimum-viable starting state,
not a relaxation of the schema's required-field commitment.
Qa / prod values are gated by future resolution of per-env
defaults or template substitution (a follow-up enrichment); until
that resolution lands, the rule is realised for local only, and
qa/prod operation requires either a per-env overlay mechanism or
per-env rule files (the latter is a fallback kept on the table
only as a contingency).

### Engine env constants are removed

`engine/internal/env/{local,qa,prod}.go`'s `SourceProject` and
`SourceDataset` fields are removed at the combined implementation
commit. With v2 requiring `source.project_id` and
`source.dataset_id` on every rule, the env constants serve no
purpose at the schema level — every rule (including smoke-test
rules) carries its own values. Smoke tests are refactored to use
rule-level values directly.

### ADR-0010 gains a Kafka row via a forthcoming amendment

ADR-0010's substrate matrix has rows for tabular store, object
store, Pub/Sub publish-subscribe, OIDC, and container registry —
but no Kafka row. Adding a Kafka row is additive under ADR-0010's
contract (mirrors the ADR-0017 amendment pattern that added the
CAS-fidelity note). The amendment lives in a separate
forthcoming ADR (the next available number after ADR-0023 —
subject to the per-item ADR numbering caveat from ADR-0020) and
lands at the combined implementation session that wires Kafka
into the local Compose stack.

---

## Consequences

1. **Rule schema v2 gains a required `source:` top-level field.**
   The v2 schema's top-level `required` array becomes
   `[version, entity, mode, source, checks]`. The
   `additionalProperties: false` constraint is preserved. The
   `source` object is a discriminated union on `type` — `bigquery`
   variant carries the set-mode fields; `kafka` variant carries
   the record-mode fields. The combined-implementation-commit
   contingency from ADR-0022 extends to ADR-0023: a single
   combined commit landing ADR-0021 + ADR-0022 + ADR-0023
   artefacts together is the preferred path; a v3 schema bump is
   the contingency if any earlier v2 artefact ships first.

2. **Lint binary gains source-side cross-checks #7 and #8.** On
   top of ADR-0021's four and ADR-0022's two, lint now enforces
   `source.type` ↔ rule's `mode` and `source.type` ↔ catalog's
   per-kind `source_mode`. Eight cross-checks total at this
   ADR's promotion.

3. **The partial-Wave-S gate opens at this ADR's markdown
   landing; loader rejection error is removed at the combined
   implementation commit.** Per ADR-0020 §Decision
   (Partial-Wave-S gate), the gate is met when B0-S1, B0-S2, and
   B0-S3 are all at `resolved-adr` and their ADRs are merged
   into `docs/adr/`. ADR-0021 is merged; ADR-0022 is merged;
   ADR-0023's promotion lands the gate-closing event. The gate
   opens at that markdown landing (the spec surface). The engine
   loader's `mode: record` rejection error path (ADR-0021
   §Decision (Engine loader behaviour)) is removed in the same
   combined implementation commit that lands the schema
   artefacts. Between ADR-0023's markdown landing and the
   combined implementation commit, the gate is met in spec but
   the loader still rejects record-mode in code — record-mode
   rules cannot ship until both events have occurred. After the
   combined implementation commit, the Wave-S analogue of R1
   from ADR-0020 §C-S.4 is fully satisfied: record-mode code may
   ship.

4. **Phase β studies become draftable in parallel.** Per ADR-0020
   §Decision (Sequencing rule), B0-S4 (window semantics), B0-S5
   (unified-vs-parallel runner — resolves P4), B0-S6 (failure
   scope aggregated), and B0-S7 (record-oriented cost guardrails)
   may be drafted in parallel after the partial-Wave-S gate
   opens. Promotion of any Phase β ADR remains gated on its
   declared dependencies being at `resolved-adr`; Phase β studies
   that depend on each other (e.g., B0-S6 depending on B0-S4
   windowing) cannot ship in arbitrary order, but the studies
   themselves may overlap in time.

5. **`rules/customer.yaml` migration extends with a
   local-realised `source:` block.** The combined implementation
   commit lands the migration atomically alongside ADR-0021's
   `mode:` / `kind:` migration and ADR-0022's catalog-driven
   `params:` extension.

6. **The engine env config's `SourceProject` and `SourceDataset`
   constants are removed.** With v2 requiring per-rule sources,
   the env constants serve no purpose. Removal happens at the
   combined implementation commit; smoke tests are refactored to
   rule-level values directly.

7. **ADR-0010 gains a Kafka row via a forthcoming amendment ADR.**
   The amendment commits a substrate-capability declaration for
   Kafka (topic publish/subscribe; consumer-group binding;
   offset retention) following ADR-0010's matrix shape. Lands at
   the combined implementation session that wires Kafka into the
   local Compose stack. The amendment is additive under
   ADR-0010's contract; no ADR-0010 supersession.

8. **The local Compose stack gains a Kafka emulator service.**
   Single-broker is sufficient for local smoke tests; multi-
   broker configurations are out of scope. The emulator addition
   lands at the combined implementation commit.

9. **B0-S4 inherits the Kafka identifier shape as the substrate
   for windowing decisions.** With Kafka committed, B0-S4's
   windowing options narrow: tumbling, sliding, session,
   watermark-bounded windows over Kafka topics with
   consumer-group offset state. The `record.schema_conformance`
   per-record kind bypasses windowing entirely; richer record
   kinds added in future catalog versions will consume B0-S4's
   windowing semantics.

10. **`source.type` discriminator decouples future source-shape
    additions from the mode-bifurcation.** A future record-mode
    source over an alternative streaming substrate (one that
    respects the data-ingress-vs-alert-egress separation) can be
    added as a third `source.type` variant without revisiting
    the mode discriminator. Lint cross-checks #7 and #8 extend
    by augmenting the small lookup table.

11. **The Wave-S foundational triplet is closed (3/3).** With
    ADR-0021, ADR-0022, and ADR-0023 all merged, the Wave-S
    decision backlog moves to Phase β. The Wave-S launch ADR
    (ADR-0020) is the integration point that the 2026-05-23
    scope notes on the eight set-oriented ADRs pointed to; the
    foundational triplet realises ADR-0020's locked premises
    (P1, P2, P3, P4) in concrete schema, catalog, and source-
    declaration shapes. The remaining Wave-S work is
    runtime-shape decisions (B0-S4 windowing, B0-S5
    unified-vs-parallel runner, B0-S6 failure aggregation, B0-S7
    cost guardrails) and the implementation that wires the
    triplet's commitments into running code.

---

## Notes

- Watermark and offset semantics for Kafka sources (continuous
  vs windowed binding, late-arrival handling, the relationship
  between Kafka consumer-group offsets and ADR-0002's
  `execution_id` formula) are decided in B0-S4. The
  `source.type: kafka` shape committed here is additive-ready:
  optional fields (e.g., `watermark_mode`) can be appended
  without breaking v2.
- Multi-broker Kafka configurations, SASL authentication, TLS,
  and Schema Registry integration are operational concerns
  decided at the combined implementation commit; bootstrap
  servers and authentication live in env config (per ADR-0018
  forthcoming Kafka row), not in per-rule source declarations.
- Per-environment `source.project_id` / `source.dataset_id`
  defaults — template substitution or per-env overlay — are a
  future enrichment that resolves the local-only realisation
  committed under §Decision (Migration is local-realised).
- Engine-side validation that the declared `partition_column`
  exists on the named BigQuery table (a strengthening of
  ADR-0007's pre-check) is a future ADR-0007 follow-up; this
  ADR does not strengthen ADR-0007's pre-check beyond what
  ADR-0007 already commits.
