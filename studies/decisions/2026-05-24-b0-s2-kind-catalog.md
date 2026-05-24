<!-- path: studies/decisions/2026-05-24-b0-s2-kind-catalog.md -->

# B0-S2 — Kind Catalog

## Metadata

- **B-item reference:** B0-S2 (Wave-S foundational triplet, item 2 of 3)
- **Status:** resolved-study (Wave-S, B0-S2; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-24
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch — locks P1–P4 and assigns B0-S2's scope);
  [ADR-0021](../../docs/adr/0021-mode-as-primitive.md) (B0-S1 — mode
  primitive, kind prefix grammar, lint cross-checks, schema-version
  bump v1 → v2); [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
  (engine ↔ rules compatibility contract).
- **Downstream open:** B0-S3 (sources schema — consumes the catalog's
  per-kind source-mode declaration to cross-check rule sources against
  expected shapes).
- **Promotion target:** `docs/adr/0022-kind-catalog.md` (subject to
  ADR-0020 §Decision (Per-item ADR numbering); `0022` reflects the
  expected sequence S1 → 0021, S2 → 0022, S3 → 0023, modulo
  unrelated promotions).
- **Loop discipline:** same as B0-S1 — `/resolve-b0` study →
  `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.

---

## Context

ADR-0020 launched Wave-S with mode as the architectural primitive
(P1), the `set.*` / `record.*` kind-prefix discipline (P2), and
capability derived from mode (P3). ADR-0021 (B0-S1) realised P1, P2,
and P3 in the rule and owners schemas — v1 → v2 across both, with
the kind constraint at the schema grammar level being
`^(set|record)\..+$`, **suffix shape explicitly deferred to this
study**. B0-S2 picks up that deferral.

The rule layer today has exactly one shipped kind: `set.row_count_positive`,
implemented at `engine/internal/eval/row_count_positive.go` and
dispatched from `engine/internal/eval/evaluator.go` at runtime. The
schema permits any string matching `^(set|record)\..+$`; the engine
silently accepts an unknown kind at parse time and only fails at the
dispatcher when no handler matches. No artefact in the repository
declares "the set of valid kinds" — the universe of accepted kinds
is implicit in the dispatcher's switch statement.

This is the gap B0-S2 closes. The Wave-S launch ADR §"The seven
B0-S items" assigns B0-S2 the following scope:

> Decides the registry of supported kinds, starting from the existing
> `set.row_count_positive` and adding one or more inaugural `record.*`
> kinds whose shape is chosen to exercise the record-mode plumbing
> minimally; the governance process for adding kinds; the
> schema-version bump rule under the ADR-0001 compatibility contract;
> and the way a kind declares the source shape it expects.

Four sub-decisions inside this scope:

1. **Where the catalog lives.** Schema-internal enum, separate file
   loaded by engine and lint, build-time-generated registry, or some
   hybrid.
2. **What a catalog entry declares.** At minimum: the kind's name, its
   mode, its expected source mode, and the shape of the per-check
   parameters it requires.
3. **The starting catalog content.** The existing `set.row_count_positive`
   stays; one inaugural `record.*` kind is added to exercise the
   record-mode plumbing without prematurely committing to windowing
   (B0-S4), aggregation (B0-S5), failure-scope shape (B0-S6), or
   cost guardrails (B0-S7).
4. **The catalog's own compatibility contract.** Adding a kind,
   changing a kind's params shape, removing a kind — each has a
   different impact and must be governed.

The platform principles relevant here are documented in
[`studies/foundation/01-charter-and-principles.md`](../foundation/01-charter-and-principles.md):
**P1** (rules remain declarative — kinds are the declarative DSL's
vocabulary; widening it through implicit means is forbidden),
**P5** (evolution contract-driven — the catalog needs its own
versioning rules), and **P3** (ownership explicit — adding a kind
crosses the engine ↔ rules boundary and must be reviewed by both
sides, governed under [ADR-0015](../../docs/adr/0015-codeowners.md)).

---

## Decision Drivers

- **DD-S2.1** — **Single source of truth for "what is a valid
  kind?"** Today, the answer lives implicitly in the dispatcher's
  switch statement; lint cannot consult it. After B0-S2, both lint
  and engine must consume the same authoritative declaration.

- **DD-S2.2** — **Lint must reject unknown kinds at PR-review
  time, not only at runtime.** Without this, a typo in a kind name
  reaches the engine before review catches it — the partial-Wave-S
  gate's runtime is still set-mode-only and rejection-by-handler-
  missing is a poor error signal.

- **DD-S2.3** — **Adding a kind must not require a schema-version
  bump.** Schema bumps cascade through ADR-0001's migration
  contract; if every new kind triggers a bump, the migration tax
  on adding common-case kinds becomes prohibitive. The catalog
  carries its own additive-vs-breaking contract, independent of the
  rule schema.

- **DD-S2.4** — **The catalog must declare the per-check
  parameter shape per kind.** Without this, `set.row_count_positive`
  (no params) and `record.schema_conformance` (requires a
  `schema` parameter) cannot be lint-validated uniformly. The rule
  schema (v2 per ADR-0021) gains an optional `params:` object per
  check; the catalog declares the JSON-Schema fragment that
  validates `params` for each kind.

- **DD-S2.5** — **Catalog evolution is contract-driven (P5).**
  Adding a kind: additive within a catalog major. Removing a kind,
  changing its mode, changing its name, breaking its params shape:
  major bump (catalog v2). Adding optional params to an existing
  kind: additive.

- **DD-S2.6** — **The first record kind exercises minimal
  Wave-S surface.** It must run per-record (no windowing — defers
  B0-S4); it must not require aggregation across records (defers
  B0-S5); its failure semantics must be definable per-record
  without committing to B0-S6's aggregated-failure shape. A
  per-record schema-conformance check satisfies all three deferrals.

- **DD-S2.7** — **The runtime dispatch model already in place is
  preserved.** `engine/internal/eval/evaluator.go` dispatches by
  string. The catalog enriches this model with a declarative
  surface that lint can consume; it does not force a rewrite of
  the dispatcher.

- **DD-S2.8** — **Adding a kind crosses the engine ↔ rules
  boundary; CODEOWNERS governance applies.** Per ADR-0015, the
  catalog file lives under engine-owned paths; a kind addition
  touches both the catalog and (if record-mode) potentially
  affects rule authors' choices. Joint review is the default.

---

## Considered Options

The four options below differ on **where the catalog lives** and
**how lint and engine consume it**. All four assume the per-kind
catalog entry declares name, mode, source mode, and a JSON-Schema
fragment for `params` validation — that shape is in scope for the
study but not the variation axis.

### Option A — Catalog enumerated inside the rule schema (kind as enum)

**Shape.** The rule schema's `kind` field becomes an enum of
allowed kind names rather than the `^(set|record)\..+$` regex
committed by ADR-0021. Adding a kind = bumping the schema. The
schema is the catalog.

**Cost.** Every kind addition is a schema-version bump (v2 → v3 →
v4 …) under ADR-0001's migration contract. The migration tax on
common-case additive operations is high. Schema versioning becomes
catalog versioning — two distinct concerns conflated.

**Verdict.** Rejected on DD-S2.3 (adding a kind must not require a
schema bump).

### Option B — Catalog as a separate file loaded at runtime by engine and lint (recommended)

**Shape.** A catalog file lives at
`engine/internal/dsl/catalog/v1.yaml` (engine-canonical) with a
byte-equal mirror at `rules/_schema/catalog.v1.yaml` (rules-side,
for offline reference and lint loading, governed by the same
ADR-0001 byte-equality CI gate that covers the rule schema mirror).

The catalog declares a list of kinds; each kind carries `name`,
`mode`, `source_mode`, `params_schema` (JSON-Schema fragment), and
`description`. The lint binary loads the catalog at startup and
rejects any rule whose `kind` is not in the catalog, whose `kind`
mode does not match the rule's `mode` (per ADR-0021's lint
cross-check #2 — re-enforced through the catalog), or whose
`params` does not satisfy the catalog entry's `params_schema` —
absent `params` is treated as the empty object `{}` so that a
kind whose `params_schema` declares required fields fails
validation when the rule omits `params`.

The engine binary's dispatcher
(`engine/internal/eval/evaluator.go`) loads the catalog at
startup, validates that every catalog entry has a registered
handler in the eval package, and dispatches incoming rules by
kind name as it does today. Compile-time safety is not required
— the engine fails fast at startup if a catalog entry has no
handler.

**Cost.** One extra file (and its mirror) in the repository.
Lint and engine each load a small YAML on startup — negligible
overhead.

**Verdict.** Recommended.

### Option C — Build-time codegen registry (catalog drives generated Go code)

**Shape.** Same catalog source file as Option B, but the engine's
runtime registry is **generated from the catalog at build time**.
`go generate` consumes the catalog and emits a typed map in
`engine/internal/eval/registry.go`; the dispatcher fails to
compile if a catalog entry has no matching handler in the same
package.

**Cost.** Adds codegen complexity to the build. The engine binary
gains an unfamiliar layer (generated code) that contributors must
recognise. The compile-time-vs-startup-time safety differential is
marginal — Option B's startup check catches the same missing-handler
case at the same effective moment (the engine binary is
short-lived per restart and fails fast).

**Verdict.** Rejected on simplicity grounds; the safety win is
small relative to the codegen tax.

### Option D — Per-kind plugin model (catalog implicit, kinds register themselves)

**Shape.** Each kind ships as a separate Go package whose `init()`
registers the kind with a global registry. No catalog file; the
catalog is the union of imported packages. Lint imports the same
packages to discover kinds.

**Cost.** Init-time side effects are an established anti-pattern.
Cross-binary catalog visibility (lint sees what engine sees) is
brittle — a lint build that drifts from the engine build can
silently disagree on the kind universe. Catalog versioning has no
home (kinds version themselves; the union has no version).

**Verdict.** Rejected on cross-binary visibility and versioning
grounds.

---

## Recommendation

**Pick Option B — Catalog as a separate file loaded at runtime by
engine and lint.**

Rationale, tied directly to drivers:

- **DD-S2.1 (single source).** The catalog file is the single
  declaration; both lint and engine load it. The dispatcher
  switch statement becomes a derived consumer rather than the
  source of truth.
- **DD-S2.2 (lint rejects unknown kinds).** Lint loads the
  catalog and rejects any rule whose kind is not in it. The
  rejection lands at PR-review time, not at engine startup.
- **DD-S2.3 (no schema bump for adding a kind).** Adding a kind
  is a catalog-additive operation; the rule schema (v2) is
  unchanged.
- **DD-S2.4 (per-kind params validation).** Each catalog entry
  carries a JSON-Schema fragment that validates the per-check
  `params:` object. Lint dispatches by kind to the right
  validation schema; the engine relies on lint having already
  validated when a rule reaches it (defense in depth: the engine
  may re-validate, but the contract surface is lint).
- **DD-S2.5 (contract-driven evolution).** The catalog carries
  its own version (`catalog_version: 1` at file head). Additive
  vs breaking changes follow the same shape as ADR-0001's
  contract for the rule schema, scoped to the catalog file.
- **DD-S2.6 (minimal first record kind).** The inaugural
  `record.*` kind committed by this study is
  `record.schema_conformance` — validates each record against a
  JSON-Schema fragment passed via `params.schema`. Per-record,
  no windowing, no aggregation, failure semantics definable per
  record without B0-S6.
- **DD-S2.7 (preserve dispatcher).** The existing
  `engine/internal/eval/evaluator.go` dispatcher gains a startup
  check that every catalog entry has a registered handler; the
  dispatch logic itself is unchanged.
- **DD-S2.8 (CODEOWNERS governance).** The catalog file lives
  under `engine/internal/dsl/catalog/`; the mirror lives under
  `rules/_schema/`. Per ADR-0015's path-rule table, changes to
  either path require both engine-maintainers and rules-authors
  review. The first new kind added under this contract goes
  through that joint review.

**Catalog entry shape** (committed by this study; landed in v1 at
ADR-0022's promotion):

```yaml
# engine/internal/dsl/catalog/v1.yaml — illustrative shape, not a
# committed file in this study (R1: code-shaped illustration only)
catalog_version: 1
kinds:
  - name: set.row_count_positive
    mode: set
    source_mode: set
    params_schema:
      type: object
      additionalProperties: false
      properties: {}
    description: >
      Verifies the source has at least one row. No parameters.
  - name: record.schema_conformance
    mode: record
    source_mode: record
    params_schema:
      type: object
      required: [schema]
      additionalProperties: false
      properties:
        schema:
          description: JSON Schema fragment to validate each record against.
          type: object
    description: >
      Validates each record against a JSON Schema fragment passed
      via params.schema. Per-record evaluation; no windowing.
```

The two JSON-Schema-shaped objects in this design serve different
layers: the catalog's `params_schema` is **meta-validation** — it
asks *"does this rule have the parameters this kind requires?"*
and lives in the catalog. The rule's own `params.schema` (in the
`record.schema_conformance` example above) is **data-validation**
— it asks *"do incoming records satisfy this contract?"* and
lives in the rule. The catalog validates rule shape; the rule
validates record content.

**Rule schema v2 amendment** (already committed by ADR-0021 to v2;
this study adds the `params` field at v2): each `check` in v2
gains an optional `params:` object whose validation schema comes
from the catalog entry matching the check's `kind`. The schema-side
constraint stays at `additionalProperties: false` for the check
object; only `description` and `params` are optional alongside the
required `check_id` and `kind`.

**Catalog evolution rules:**

| Change | Treatment | Version bump |
|---|---|---|
| Add a new kind | additive | none (within catalog major) |
| Add an optional field to a kind's `params_schema` | additive | none |
| Add a required field to a kind's `params_schema` | breaking (existing rules with that kind become invalid) | major bump |
| Remove a kind | breaking | major bump |
| Rename a kind | breaking (existing rules break) | major bump |
| Change a kind's `mode` or `source_mode` | breaking | major bump |
| Change a kind's `description` | metadata-only | none |

**Governance for adding a kind:**

1. PR adds the catalog entry, the handler in `engine/internal/eval/`,
   a unit test, and (for record-mode kinds, until the partial-
   Wave-S gate closes) explicit acknowledgement that the kind ships
   to the catalog but the engine loader still rejects it per
   ADR-0021 §Decision (Engine loader behaviour).
2. CODEOWNERS review per ADR-0015: engine-maintainers approve the
   handler and the dispatcher integration; rules-authors approve
   the catalog entry and its impact on the rule-authoring surface.
3. No new ADR is required for an additive kind — the catalog's
   own additive contract suffices. A breaking change to the catalog
   (major bump) does require an ADR (under ADR-0001's contract).

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Catalog location | Separate file at `engine/internal/dsl/catalog/v1.yaml` (canonical) + byte-equal mirror at `rules/_schema/catalog.v1.yaml` |
| Catalog format | YAML with top-level `catalog_version` + `kinds[]` |
| Per-kind fields | `name`, `mode`, `source_mode`, `params_schema` (JSON Schema), `description` |
| Inaugural catalog v1 kinds | `set.row_count_positive` (existing), `record.schema_conformance` (new) |
| Per-check `params:` field | Added to rule schema v2 as optional; validated per kind via catalog `params_schema` |
| Lint enforcement | Loads catalog at startup; rejects unknown kind, mode mismatch (cross-check #2 from ADR-0021), or params schema failure |
| Engine enforcement | Dispatcher checks at startup that every catalog entry has a handler; fails fast on mismatch |
| Catalog versioning | Independent of rule schema; additive changes do not bump; breaking changes require major bump + ADR |
| Governance | PR + CODEOWNERS dual review (engine-maintainers + rules-authors); no per-kind ADR |

---

## Consequences

### Cross-cutting consequences

- **C-B0S2.1** — **Catalog file lands at ADR-0022 promotion.** Two
  new files: `engine/internal/dsl/catalog/v1.yaml` (canonical) and
  `rules/_schema/catalog.v1.yaml` (byte-equal mirror). The
  ADR-0001 byte-equality CI gate extends to cover the catalog
  mirror.

- **C-B0S2.2** — **Rule schema v2 gains optional `params:` per
  check.** The ADR-0021 commitment to v2 is **extended** (not
  amended — R3 does not fully bite, because v2 has not shipped to
  disk yet at this study's promotion time): the `check` object's
  `additionalProperties: false` and the property list now include
  `params` (optional object). **The implementation path is a
  single combined commit landing ADR-0021 and ADR-0022 artefacts
  together**, so the v2 schema carries `params:` from the moment
  it ships to disk. **Contingency:** if ADR-0021's v2 schema
  ships to disk before this study's implementation lands, a v3
  schema bump is required to add `params:`, because adding a
  property to a shipped schema with `additionalProperties: false`
  is a breaking change under ADR-0001's compatibility contract.
  *(New contribution proposed here, requires review.)*

- **C-B0S2.3** — **Lint binary gains catalog-driven cross-checks.**
  In addition to ADR-0021's four cross-checks, lint adds:
  (5) rule's `kind` must be a registered catalog entry name;
  (6) rule's check `params` (with absent `params` treated as the
  empty object `{}` for required-field checks) must satisfy the
  catalog entry's `params_schema`, so a rule that omits `params`
  fails validation when the kind's `params_schema` declares
  required fields. Cross-check #6 is what the catalog's per-kind
  `params_schema` is for.

- **C-B0S2.4** — **Engine dispatcher gains a startup invariant.**
  At engine boot, every catalog entry must have a registered
  handler in `engine/internal/eval/`. A catalog entry without a
  handler fails the boot — the engine refuses to run with a
  partial dispatch surface.

- **C-B0S2.5** — **`record.schema_conformance` is committed but
  not runnable until the partial-Wave-S gate closes.** The kind
  ships in the catalog at ADR-0022's promotion; the engine
  loader's mode-field filter (per ADR-0021 §Decision (Engine
  loader behaviour)) rejects `mode: record` rules until B0-S1 +
  B0-S2 + B0-S3 are all at `resolved-adr`. After B0-S3 promotes,
  record-mode rules can flow through; the implementation of the
  `record.schema_conformance` handler (a Go file under
  `engine/internal/eval/`) lands when record-mode is wireable —
  either at partial-Wave-S gate close (if a single combined
  implementation commit lands all three foundational-triplet
  artefacts together) or earlier per the operator's pacing.

- **C-B0S2.6** — **No new ADR required for additive catalog
  changes.** The catalog's own additive contract (DD-S2.5 / the
  evolution table above) governs. Removing or renaming a kind, or
  any breaking change, requires a new ADR superseding ADR-0022's
  catalog v1 commitments.

- **C-B0S2.7** — **CODEOWNERS path rules extend.** Per ADR-0015,
  `engine/internal/dsl/catalog/` falls under engine-maintainers'
  ownership; `rules/_schema/catalog.v1.yaml` (and its successors)
  falls under both engine-maintainers and rules-authors due to its
  rules-side location. The CODEOWNERS file lands the extended path
  rules at ADR-0022's promotion commit. *(New path rule under
  ADR-0015's additive contract, requires review.)*

- **C-B0S2.8** — **`record.schema_conformance` handler ships
  per-record-evaluator only; entity-status aggregation is bounded
  by B0-S6.** The handler emits per-record violations during
  evaluation; how N per-record failures map to an entity-level
  status (`pass` / `fail` / `error` / `degraded` per ADR-0004) is
  B0-S6's question. The handler's status-mapping path lands when
  B0-S6 promotes; until then, the handler is half-shipped — its
  per-record evaluator is complete, and its entity-status
  aggregation path is TBD. *(New contribution proposed here,
  requires review.)*

### Per-artefact consequences

- **`engine/internal/dsl/catalog/v1.yaml`** — new file. Top-level:
  `catalog_version` (const `1`), `kinds[]` (non-empty array). Each
  kind entry: `name`, `mode`, `source_mode`, `params_schema`
  (object — a valid JSON Schema), `description`.

- **`rules/_schema/catalog.v1.yaml`** — byte-equal mirror of the
  engine source per the ADR-0001 invariant.

- **`engine/internal/dsl/schema/v2.schema.json`** (committed by
  ADR-0021; amended here) — the `check` object's `properties` map
  gains an optional `params: object` entry; `additionalProperties:
  false` is preserved.

- **`rules/_schema/v2.schema.json`** — byte-equal mirror of the
  above.

- **`tools/lint/`** — two new cross-checks (catalog membership;
  per-kind params validation) on top of ADR-0021's four.

- **`engine/internal/eval/evaluator.go`** — startup check that
  every catalog entry has a registered handler; dispatch logic
  unchanged.

- **`engine/internal/eval/record_schema_conformance.go`** — new
  handler file (lands when record-mode is wireable per C-B0S2.5).
  Per-record evaluator: pulls the `params.schema` JSON-Schema
  fragment from the rule and validates each record against it.

- **`/.github/CODEOWNERS`** — extended path rules per C-B0S2.7
  (engine-maintainers for catalog; joint review for the rules-side
  mirror).

- **No engine runtime changes** beyond the dispatcher startup
  check and the new handler file. ADR-0002/0003/0004/0006/0007/
  0010/0014/0017 contracts are untouched. ADR-0021's loader changes
  are extended with no additional behaviour from this study.

---

## Open Questions

- **OQ-B0S2.1** — **Mirror format choice (YAML vs JSON).** The
  catalog is illustrated as YAML to mirror the rule-authoring
  surface's existing YAML choice. JSON would also work and is the
  format the schema mirrors use today. The implementation commit
  picks one; the choice is detail-level and does not affect the
  decisions here. *Out of scope for current cycle.*

- **OQ-B0S2.2** — **Catalog mirror byte-equality enforcement.**
  ADR-0001's byte-equality CI gate covers the rule schema mirror;
  extending it to the catalog mirror is the natural move. Whether
  the existing CI workflow auto-picks up the new mirror pair or
  needs an explicit configuration update is an implementation
  detail. *Out of scope for current cycle.*

- **OQ-B0S2.3** — **Handler-language coupling.** The engine
  dispatcher is in Go; handlers live in
  `engine/internal/eval/`. Whether the catalog could in principle
  support handlers in other languages (e.g., a future record-mode
  handler in a stream-processing language) is a long-tail
  question. *Out of scope for current cycle.* The catalog is
  language-neutral by shape; the engine-side enforcement
  assumes Go for now.

- **OQ-B0S2.4** — **Catalog tooling.** Whether a separate CLI
  surface (e.g., `dq-catalog list`, `dq-catalog validate`) is
  worth adding under `tools/` for operator inspection is a
  developer-experience question. *Out of scope for current
  cycle.* The catalog file is human-readable YAML; ad-hoc
  inspection suffices until the catalog grows large.

- **OQ-B0S2.5** — **Per-kind documentation surface.** Whether
  each catalog entry's `description` field is sufficient, or each
  kind eventually grows a `docs/kinds/<kind>.md` page, is a
  documentation question. *Out of scope for current cycle.* The
  `description` field is the minimum-viable surface; docs pages
  are a future enrichment.

- **OQ-B0S2.6** — **Pre-partial-gate handler ordering.** The
  `record.schema_conformance` handler can land at any point
  between this ADR's promotion and the partial-Wave-S gate close,
  but it cannot be invoked by the engine loader until the gate
  closes. Whether the handler lands eagerly (in ADR-0022's
  promotion commit, marked as "ships but is unreachable") or
  lazily (in the gate-closing commit, alongside the loader
  unlock) is the operator's pacing call. *Defer to operator
  pacing.*

---

## Promotion target

**Target:** `docs/adr/0022-kind-catalog.md`.

This study promotes to **ADR-0022** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0022 is the second per-item ADR of the
Wave-S foundational triplet (S1 → 0021 already landed; S3 → 0023
forthcoming); per ADR-0020 §Decision (Per-item ADR numbering), the
`0022` slot is descriptive of the expected sequence and may shift
if an unrelated promotion lands between B0-S items.

ADR-0022's promotion commit lands the artefacts committed in
§Consequences above:

1. The new `engine/internal/dsl/catalog/v1.yaml` (canonical) and
   its byte-equal mirror `rules/_schema/catalog.v1.yaml`, declaring
   the inaugural two kinds (`set.row_count_positive`,
   `record.schema_conformance`).
2. The rule schema v2 amendment: optional `params:` per check, with
   schema-side validation via the catalog's per-kind `params_schema`.
3. Two new lint cross-checks in `tools/lint/`: catalog membership
   and per-kind params validation.
4. The engine dispatcher startup check at
   `engine/internal/eval/evaluator.go`.
5. The new handler file `engine/internal/eval/record_schema_conformance.go`
   (pacing per C-B0S2.5 and OQ-B0S2.6 — may land in the
   gate-closing commit instead).
6. CODEOWNERS path rules extended per C-B0S2.7.

Per R8, the future ADR-0022 will be rewritten from this study, not
linked back to it.
