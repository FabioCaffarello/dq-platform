<!-- path: docs/adr/0022-kind-catalog.md -->

# ADR-0022 — Kind Catalog

- **Status:** accepted
- **Date:** 2026-05-24

---

## Context

ADR-0020 launched Wave-S with mode as the architectural primitive
and the `set.*` / `record.*` kind-prefix discipline. ADR-0021
(B0-S1) realised the primitive in schema shape: the rule artefact
and the entity declaration each carry a typed `mode` field, the
`kind` field on every check matches `^(set|record)\..+$`, and four
lint cross-checks enforce the consistency of mode declarations
across the rule, the entity, and the kind prefix. ADR-0021
explicitly deferred the **suffix shape** of the kind grammar to
this ADR.

This ADR resolves that deferral and goes further: it commits the
**catalog** — the authoritative declaration of which kinds exist,
what mode each is in, what source shape each expects, and what
parameter schema each kind requires from a rule that uses it.

Until this ADR lands, the rule layer carries exactly one shipped
kind (`set.row_count_positive`), implemented at
`engine/internal/eval/row_count_positive.go` and dispatched by
`engine/internal/eval/evaluator.go`. The rule schema permits any
string matching `^(set|record)\..+$`; the engine silently accepts
an unknown kind at parse time and only fails at the dispatcher
when no handler matches. No artefact in the repository declares
"the set of valid kinds" — the universe of accepted kinds is
implicit in the dispatcher's switch statement.

Four interlocking sub-decisions live inside this ADR:

1. Where the catalog lives.
2. What a catalog entry declares.
3. The starting catalog content (the inaugural record kind that
   exercises Wave-S record-mode plumbing without committing to
   B0-S4, B0-S5, B0-S6, or B0-S7).
4. The catalog's own compatibility contract — when adding a kind
   is additive, when it is breaking, and who reviews each.

R5 (cite or mark) and ADR-0001 (engine ↔ rules compatibility)
both bear on the design: the catalog is a contract surface
between engine and rules, and ADR-0001's byte-equality mirror
pattern extends to cover the catalog's canonical/mirror pair.

---

## Decision

### Catalog lives in a separate file, loaded by engine and lint

The catalog is a YAML file at
`engine/internal/dsl/catalog/v1.yaml` (canonical) with a
byte-equal mirror at `rules/_schema/catalog.v1.yaml`. The
ADR-0001 byte-equality CI gate extends to cover the catalog
mirror — the same `make sync-schema`-style discipline that
governs the rule schema mirror applies here.

Both the lint binary (`tools/lint/`) and the engine binary
(`engine/internal/eval/evaluator.go`) load the catalog at
startup. Lint rejects rules that violate the catalog; the engine
dispatcher validates that every catalog entry has a registered
handler in the eval package and fails fast at startup if one is
missing. Compile-time codegen is not used; the runtime-dispatch
model already shipped by Wave 3 is preserved.

### Per-entry shape

Each catalog entry declares five fields:

- `name` — the full kind identifier, e.g. `set.row_count_positive`.
  Must match the kind-prefix grammar from ADR-0021's lint
  cross-check #2 (`^(set|record)\..+$`).
- `mode` — `set` or `record`. Must match the prefix of `name`.
- `source_mode` — `set` or `record`. The shape of source the kind
  expects (B0-S3, forthcoming, finalises the per-mode source
  declaration; the catalog's `source_mode` declares which side of
  that bifurcation a kind consumes).
- `params_schema` — a JSON Schema fragment validating the
  per-check `params:` object (see "Rule schema v2 extension" and
  "Meta vs data validation" below).
- `description` — human-readable summary, one paragraph.

### Inaugural catalog v1

Catalog v1 declares two kinds:

- **`set.row_count_positive`** — existing kind shipped in Wave 3.
  No parameters. Verifies the source (a BigQuery table or view
  per ADR-0007's loader contract) has at least one row.
- **`record.schema_conformance`** — new kind. Validates each
  record against a JSON Schema fragment passed via
  `params.schema`. Per-record evaluation; no windowing (B0-S4
  remains deferred), no aggregation across records (B0-S5
  remains deferred). The handler's entity-status aggregation
  path is bounded by B0-S6 — see Consequence 5 below.

### Rule schema v2 extension: per-check `params:`

The v2 rule schema committed by ADR-0021 — currently not yet
shipped to disk; ADR-0021's implementation is pending — is
extended in the same implementation step to add an optional
`params:` object per check. The `check` object's
`additionalProperties: false` is preserved; the only addition is
the new optional `params` property.

The implementation path is a **single combined commit** landing
ADR-0021's and this ADR's artefacts together, so the v2 schema
carries `params:` from the moment it ships to disk. If ADR-0021's
v2 schema ships to disk before this ADR's implementation lands, a
**v3 schema bump** is required, because adding a property to a
shipped schema with `additionalProperties: false` is a breaking
change under ADR-0001's compatibility contract. The combined
commit is the preferred path; v3 is the contingency.

### Meta-validation vs data-validation

Two JSON-Schema-shaped objects participate in this design, and
they serve different layers:

- The catalog's `params_schema` is **meta-validation**: it asks
  *"does this rule have the parameters this kind requires?"* The
  catalog validates rule shape.
- The rule's own `params.schema` (in the `record.schema_conformance`
  example) is **data-validation**: it asks *"do incoming records
  satisfy this contract?"* The rule validates record content.

The two layers do not overlap; lint runs meta-validation, the
engine runs data-validation at evaluation time.

### Lint cross-checks #5 and #6

On top of ADR-0021's four cross-checks (mode-typed on rule and
entity, kind prefix matches rule's mode, rule's mode matches
entity's mode), the lint binary gains two more:

- **#5 — catalog membership.** A rule's `kind` must be a
  registered name in the catalog. Unknown kinds fail lint at
  PR-review time, not at engine startup.
- **#6 — per-kind params validation.** A rule's check `params`
  (with absent `params` treated as the empty object `{}` for
  required-field checks) must satisfy the catalog entry's
  `params_schema`. A rule that omits `params` fails validation
  when the kind's `params_schema` declares required fields, so
  omitted params do not pass by absence.

### Engine dispatcher startup invariant

At boot, the engine binary validates that every catalog entry
has a registered handler in `engine/internal/eval/`. A catalog
entry without a handler fails the boot — the engine refuses to
run with a partial dispatch surface. The dispatch logic itself
is unchanged; the only addition is the startup check.

### Catalog evolution rules

The catalog versions independently of the rule schema. The
contract is shaped like ADR-0001's but scoped to the catalog
file:

| Change | Treatment | Version bump |
|---|---|---|
| Add a new kind | additive | none (within catalog major) |
| Add an optional field to a kind's `params_schema` | additive | none |
| Add a required field to a kind's `params_schema` | breaking | major bump |
| Remove a kind | breaking | major bump |
| Rename a kind | breaking | major bump |
| Change a kind's `mode` or `source_mode` | breaking | major bump |
| Change a kind's `description` | metadata-only | none |

Additive changes do not require a new ADR. Breaking changes
require a new ADR superseding this one's catalog v1 commitments.

### Governance

Adding a kind is a PR that:

1. Adds the catalog entry, the handler in
   `engine/internal/eval/`, a unit test, and (for record-mode
   kinds, until the partial-Wave-S gate closes) an explicit
   acknowledgement that the kind ships to the catalog but the
   engine loader still rejects it per ADR-0021 §Decision (Engine
   loader behaviour).
2. Receives dual CODEOWNERS review per ADR-0015:
   engine-maintainers approve the handler and the dispatcher
   integration; rules-authors approve the catalog entry and its
   impact on the rule-authoring surface.

CODEOWNERS path rules extend at this ADR's promotion commit:
`engine/internal/dsl/catalog/` falls under engine-maintainers'
ownership; `rules/_schema/catalog.v1.yaml` falls under both
engine-maintainers and rules-authors due to its rules-side
location. The extension follows ADR-0015's additive contract; no
ADR-0015 supersession.

---

## Consequences

1. **The rule schema v2 carries `params:`.** ADR-0021's v2 schema
   commitment is extended (not amended — v2 has not shipped to
   disk yet, so R3 does not fully bite). The single combined
   implementation commit landing ADR-0021's and this ADR's
   artefacts is the preferred path; if ADR-0021's v2 ships
   first, a v3 schema bump is required.

2. **Lint enforces the catalog at PR-review time.** Cross-checks
   #5 and #6 land in `tools/lint/` at this ADR's promotion. A
   rule whose `kind` is not in the catalog, or whose `params`
   fails the catalog entry's `params_schema` (treating absent
   `params` as `{}`), fails lint before it reaches the engine.

3. **The engine dispatcher gains a startup invariant.** A
   catalog entry without a registered handler fails boot. The
   dispatcher's existing dispatch logic is unchanged; ADR-0007's
   loader contracts are untouched.

4. **The catalog file pair gains byte-equality coverage.** The
   `engine/internal/dsl/catalog/v1.yaml` canonical and
   `rules/_schema/catalog.v1.yaml` mirror are covered by the
   ADR-0001 CI gate. The existing `make sync-schema` mechanism
   extends to the catalog mirror.

5. **`record.schema_conformance` ships half-shipped until B0-S6.**
   The handler emits per-record violations during evaluation;
   how N per-record failures map to an entity-level status
   (`pass` / `fail` / `error` / `degraded` per ADR-0004) is
   B0-S6's question. The handler's status-mapping path lands
   when B0-S6 promotes. Until then, the handler's per-record
   evaluator is complete and its entity-status aggregation path
   is TBD.

6. **No record-mode rule runs until the partial-Wave-S gate
   closes.** ADR-0021's engine loader filter (only `mode: set`
   accepted until B0-S1 + B0-S2 + B0-S3 are all at
   `resolved-adr`) is unchanged. `record.schema_conformance`
   ships in the catalog at this ADR's promotion but is unreachable
   from the loader until B0-S3 also promotes. The handler
   implementation file may land at any point between this ADR's
   promotion and the partial-gate close — eagerly at this ADR's
   promotion commit (marked as "ships but is unreachable"), or
   lazily in the gate-closing commit.

7. **CODEOWNERS extends per ADR-0015's additive contract.** The
   path rules for `engine/internal/dsl/catalog/` and
   `rules/_schema/catalog.v1.yaml` land at this ADR's promotion
   commit. The extension is not an ADR-0015 supersession; it is
   the additive path-rule mechanism ADR-0015's contract
   anticipates.

8. **The Wave-S foundational triplet is 2/3 closed.** With B0-S1
   at `resolved-adr` and B0-S2 closing on this ADR's promotion,
   the partial-Wave-S gate needs only B0-S3 (sources schema) to
   complete. After the gate closes, the engine loader accepts
   record-mode rules; until then, record-mode runs are still
   rejected at the loader regardless of catalog membership.

9. **Adding a kind does not require an ADR.** The catalog's
   additive contract (see "Catalog evolution rules" above) covers
   PR-based kind additions through CODEOWNERS dual review. A
   breaking change to the catalog (major bump) requires an ADR
   superseding this one.

10. **The kind catalog is the single source of truth.** The
    dispatcher's switch statement becomes a derived consumer
    rather than the source of truth. Adding a kind to the
    dispatcher without adding it to the catalog fails the
    engine's startup invariant; adding to the catalog without
    adding a handler fails the same invariant. The asymmetry
    that previously made "what kinds exist?" answerable only by
    reading code is closed.

---

## Notes

- The catalog file's format choice (YAML vs JSON) is an
  implementation detail picked at the promotion commit. YAML
  mirrors the rule-authoring surface's existing YAML convention;
  JSON would match the schema mirror's format. Either choice
  satisfies the byte-equality gate.
- Whether the engine's runtime registry could in principle
  support handlers in languages other than Go is a long-tail
  question. The catalog file is language-neutral by shape; the
  engine-side enforcement assumes Go for now.
- A separate `dq-catalog` CLI surface for operator inspection
  (e.g., `dq-catalog list`, `dq-catalog validate`) is a
  developer-experience addition under `tools/` that may grow
  when the catalog is large enough to benefit. For catalog v1
  with two kinds, the YAML file is human-readable and ad-hoc
  inspection suffices.
- Per-kind documentation (a `docs/kinds/<kind>.md` page per
  catalog entry) is a future enrichment. The per-kind
  `description` field is the minimum-viable surface; richer
  documentation lands when the catalog grows.
- The `record.schema_conformance` handler implementation file
  lands at the operator's pacing — eagerly in this ADR's
  promotion commit (the conservative path, ships unreachable
  code marked as such) or lazily in the gate-closing commit
  (alongside the loader unlock). Neither path affects this ADR's
  catalog commitments.
