<!-- path: studies/decisions/2026-05-23-b0-s1-mode-as-primitive.md -->

# B0-S1 — Mode as Primitive

## Metadata

- **B-item reference:** B0-S1 (Wave-S foundational triplet, item 1 of 3)
- **Status:** resolved-study (Wave-S, B0-S1; one critique round; round 1 cleared with no blocking findings)
- **Last updated:** 2026-05-23
- **Upstream resolved:** [ADR-0020](../../docs/adr/0020-wave-s-launch.md)
  (Wave-S launch — locks premises P1, P2, P3, P4; commits sequencing
  and gate criteria); [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
  (engine ↔ rules compatibility contract — governs schema-version bumps).
- **Downstream open:** B0-S2 (kind catalog — consumes the mode-field
  shape and the kind-prefix lint gate committed here); B0-S3 (sources
  schema).
- **Promotion target:** `docs/adr/0021-mode-as-primitive.md`
  (subject to the same numbering caveat ADR-0020 §"Per-item ADR
  numbering" carries — `0021` is descriptive of the planned sequence).
- **Loop discipline:** same as B0-1 … B0-7 — `/resolve-b0` study →
  `/critique` (≥1 round) → operator acceptance → `/promote-to-adr`.

---

## Context

ADR-0020 launched Wave-S with four locked premises: **(P1)** mode is
the architectural primitive, **(P2)** every DSL kind carries its mode
as a name prefix (`set.*` / `record.*`), **(P3)** capability is
derived from mode and not declared independently, **(P4)** execution
unified-vs-parallel is reserved for B0-S5. ADR-0020 §"The seven B0-S
items" assigns B0-S1 the responsibility of deciding **the typed
`mode` field on the rule artefact and on the entity declaration, the
yaml shape of that field and its lint-time validation, and the rule
that the kind catalog (B0-S2) and the source schema (B0-S3) carry
mode as their organising key**. B0-S1's promotion delivers the
kind-prefix lint gate that ADR-0020 §C-S.5 commits as the boundary
guard between half-built Wave-S phases.

The current state of the rule layer is set-mode-implicit:

- `rules/_schema/v1.schema.json` (mirrored from
  `engine/internal/dsl/schema/v1.schema.json` per the ADR-0001
  byte-equality contract) declares a top-level shape of `version`,
  `entity`, `checks[]`, and an optional `description`. There is no
  `mode` field. The `kind` field on each check accepts **any string**,
  per the schema's own note that "Phase-4+ schemas tighten this to an
  enumerated set of supported kinds as the DSL grammar lands".
- `rules/_schema/_owners.v1.schema.json` declares per-entity `owner`,
  `channels`, optional `severity_overrides`, and an optional
  `description`. There is no `mode` field and no `capability` field.
- The first onboarded rule, `rules/customer.yaml`, uses
  `kind: row_count_positive` (no `set.` prefix), and its
  `_owners.yaml` entry carries no mode marker. The platform's only
  shipped rule reads as set-mode by tacit understanding, not by
  declaration — exactly the drift that P1 exists to forbid.

The drift between today's tacit set-mode and tomorrow's record-mode
addition is the gap B0-S1 closes. Per ADR-0020's R3 commitment,
nothing in this study reopens the set-oriented ADRs ADR-0002 through
ADR-0017. The mode-field commitment is additive: it lands as a
schema-version bump under the ADR-0001 compatibility contract,
moves the existing rule and its `_owners.yaml` entry to the new
shape in the same promotion commit, and leaves every set-oriented
contract (identity formula, result write model, failure scope,
alert routing, loader semantics, trigger handler) unchanged.

The platform principles relevant to this decision live in
[`studies/foundation/01-charter-and-principles.md`](../foundation/01-charter-and-principles.md):
**P1** (rules remain declarative — escape hatches forbidden), **P3**
(ownership explicit everywhere — no entity without an owner, no
alert without a route), **P5** (evolution contract-driven — schema-
breaking changes require a new DSL version and a documented
migration path).

---

## Decision Drivers

- **DD-S1.1** — **Honour the ADR-0020 locked premises.** P1, P2, P3,
  and P4 are operator-locked at the Wave-S level; B0-S1's job is to
  realise them in concrete schema shape and lint enforcement, not to
  relitigate them.

- **DD-S1.2** — **Honour ADR-0001's compatibility contract.** Adding
  a required field to the rule artefact and the entity declaration is
  a schema-breaking change. Per P5 and ADR-0001 §C4, this requires a
  schema-version bump: rule schema v1 → v2 with the byte-equality
  mirror at `rules/_schema/v2.schema.json`; `_owners.yaml` schema
  v1 → v2 at `rules/_schema/_owners.v2.schema.json`. The byte-equality
  CI gate from ADR-0001 covers both sides of the new version.

- **DD-S1.3** — **Honour R3.** No set-oriented ADR (0002–0017) is
  reopened. The mode-field landing is additive: a new schema version
  lives alongside the existing engine contracts; the engine binary
  loads v2 rules without altering ADR-0002's `execution_id` formula,
  ADR-0003's `dq_executions` write model, ADR-0004's status mapping,
  ADR-0006's alert routing, or ADR-0007's loader semantics.

- **DD-S1.4** — **Migration must be atomic.** The single shipped rule
  (`rules/customer.yaml`) and its `_owners.yaml` entry move to v2 in
  the same commit that promotes B0-S1 to ADR-0021. The repository
  must not carry a mix of v1 and v2 rules at rest — schema-version
  cohabitation across rules is a B1-7 (compatibility window)
  question, and B1-7 is open. Until B1-7 resolves, the rule of "one
  version at rest" applies. *(New contribution proposed here,
  requires review.)*

- **DD-S1.5** — **Lint is the enforcement point.** Per ADR-0020 §C-S.5,
  the kind-prefix lint gate lands with B0-S1's promotion. The lint
  binary (`tools/lint/`) gains the cross-checks defined below; the
  engine loader is not the enforcement point because the loader is
  set-mode-only until the partial-Wave-S gate closes (ADR-0020
  §"Partial-Wave-S gate"). Lint catches violations at PR-review time,
  before they reach the runtime.

- **DD-S1.6** — **Rule-level (not check-level) mode.** A rule
  artefact is a single YAML file with a single `entity:` field and a
  non-empty `checks` array. The mode is a property of the
  rule-and-its-entity, not of the individual check. Per-check mode
  would imply a single rule with mixed-mode checks against a single
  entity, which contradicts P1 (mode is the primitive — the unit at
  which "set vs record" is decided is the entity, and the rule
  artefact follows). The rule's mode applies to every check it
  contains; check kinds must match the rule's mode prefix.

- **DD-S1.7** — **Redundancy is a feature, not a defect.** With
  three independent declarations of mode — the rule's `mode:` field,
  the entity's `mode:` field in `_owners.yaml`, and the kind name
  prefix — the lint cross-check catches any one of them being wrong.
  Diagnostic redundancy is cheap at the lint layer; the ergonomic cost of
  declaring mode twice is small relative to the safety it buys when
  the catalog (B0-S2) is partial and the runtime is still
  set-mode-only.

---

## Considered Options

The options below are placement shapes for the mode declaration.
Each option assumes the ADR-0020 lock on P1/P2/P3 — none of them
removes the kind prefix or the mode primitive. The variation is
*where* the `mode:` field lives and *how* the cross-checks compose.

### Option A — Mode required at rule level **and** entity level; lint cross-checks (recommended)

**Shape.**

- Rule schema v2 adds a top-level **required** field
  `mode: <"set" | "record">`. The field sits at the same level as
  `entity` and `version`.
- `_owners.yaml` schema v2 adds a per-entity **required** field
  `mode: <"set" | "record">` at the same level as `owner` and
  `channels`.
- Every check's `kind:` is constrained to a string matching
  `^(set|record)\..+$`, and the prefix must equal the rule's
  `mode`. The suffix shape is finalised by B0-S2 (kind catalog).
- Lint rules (four):
  1. Rule's `mode` is one of `set`, `record` (schema-enforced).
  2. Every check's `kind` matches the rule's `mode` prefix.
  3. Rule's `mode` equals the `_owners.yaml` entry's `mode` for the
     rule's `entity`.
  4. `_owners.yaml` entity's `mode` is one of `set`, `record`
     (schema-enforced).

**Cost.** Mode is declared three times per (rule + entity) pair: the
rule's `mode`, the entity's `mode`, the kind's prefix. Authors type
"set" twice and the prefix appears on every check. The cost is small
prose duplication; the benefit is independent failure detection.

**Verdict.** Recommended.

### Option B — Mode at rule level only; entity's mode derived from its rules

**Shape.**

- Rule schema v2 adds `mode:` required at the top level (as in A).
- `_owners.yaml` schema v2 does **not** carry `mode`; the entity's
  mode is derived from the set of rules that target it. All rules
  for one entity must agree on `mode`; lint rejects inconsistent
  rule sets.
- Check `kind` prefix constrained as in Option A; cross-check between
  rule's `mode` and kind prefix.

**Cost.** Entity declarations in `_owners.yaml` are silent on mode
until at least one rule for them exists. A newly-onboarded entity
with zero rules has no defined mode, which weakens P3 (capability
derived from mode requires mode to be defined). Governance reviews
of `_owners.yaml` cannot answer "is this entity set or record?"
without scanning every rule file.

**Verdict.** Rejected. Implicit entity mode contradicts the spirit
of P3 — capability needs a definite ground truth, and rules are an
inverted source (rules attach to entities, not the other way).

### Option C — Mode at entity level only; rule's mode inferred from its entity

**Shape.**

- `_owners.yaml` schema v2 adds `mode:` required at the entity level
  (as in A).
- Rule schema v2 does **not** carry `mode`. Each rule's mode is
  inferred from its `_owners.yaml` entry by the linter.
- Check `kind` prefix is still constrained to `set.*` / `record.*`;
  cross-check between kind prefix and the inferred entity mode.

**Cost.** Rule files are not self-describing — a reviewer reading a
rule yaml cannot answer "what mode is this?" without consulting
`_owners.yaml`. This loses a property the rule artefact has had
since v1 (a rule reads cold). The kind prefix carries the mode in
practice, but redundancy with an explicit field is the value
proposition of Option A.

**Verdict.** Rejected. The rule artefact is the authoring surface;
making it depend on a sibling file for mode classification is the
wrong trade-off.

### Option D — Per-check mode (rule contains mixed-mode checks)

**Shape.**

- Rule schema v2 does **not** carry a top-level `mode`. Each check
  carries its own `mode:` field.
- A rule could in principle declare a set-mode check and a record-
  mode check against the same entity, with the engine dispatching
  each check to the appropriate runtime.

**Cost.** Contradicts DD-S1.6 (the unit at which set-vs-record is
decided is the entity, not the check). Per-check mode forces the
B0-S5 unified-vs-parallel runner decision into B0-S1 (mixed-mode
rules can only be evaluated by a unified runner), prematurely
collapsing P4's deferral. Also breaks the symmetry between the rule
artefact and its `_owners.yaml` entry — what would the entity's
mode be if its rules contain both?

**Verdict.** Rejected. Premature unification of P4; symmetry break.

---

## Recommendation

**Pick Option A — Mode required at rule level and entity level; lint
cross-checks.**

Rationale, tied directly to drivers:

- **DD-S1.1 (premises honoured).** Option A is the most direct
  realisation of P1 (mode is a typed declaration on both surfaces)
  and P3 (entity mode is explicit, capability follows). P2 (kind
  prefix) is enforced as a third declaration; the three independent
  declarations buy independent diagnostic signal at the lint layer.
- **DD-S1.2 (compatibility contract).** Option A's schema bump
  (v1→v2 on both rule schema and owners schema) follows ADR-0001's
  C4 invariant; both schemas land their byte-equality mirrors in
  the same commit; the CI gate from ADR-0001 catches drift.
- **DD-S1.3 (R3 honoured).** No set-oriented ADR is reopened;
  the v2 schema is additive; the loader and runtime continue to
  read v2 rules under existing semantics.
- **DD-S1.4 (atomic migration).** `customer.yaml` and the `customer`
  entry in `_owners.yaml` move to v2 in the promotion commit;
  no v1-and-v2 cohabitation at rest.
- **DD-S1.5 (lint is the enforcement).** All four cross-checks land
  in `tools/lint/` at promotion; the engine loader remains
  set-mode-only.
- **DD-S1.6 (rule-level mode).** Mode lives on the rule and on the
  entity, not on the check; check kinds must match.
- **DD-S1.7 (redundancy is a feature).** Three declarations, four
  lint cross-checks, independent failure detection.

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Mode-field shape | Required, top-level, `mode: set \| record` on both rule and entity |
| Schema bump | Rule schema v1 → v2; `_owners.yaml` schema v1 → v2 |
| Kind constraint | `^(set\|record)\..+$`, prefix matches rule's `mode`; the suffix shape is finalised by B0-S2 (kind catalog) |
| Cross-check enforcement | Lint binary at `tools/lint/`; four checks |
| Engine loader behaviour | Accepts v2 rules; remains set-mode-only until partial-Wave-S gate closes |
| `_owners.yaml` `capability:` field | Does not exist today; this study confirms it remains absent (P3 satisfied by `mode:` alone) |
| Migration | `customer.yaml` and `_owners.yaml` `customer` entry move to v2 atomically at promotion |

---

## Consequences

### Cross-cutting consequences

- **C-B0S1.1** — **Schema-version bump for both schemas.** A new
  canonical `engine/internal/dsl/schema/v2.schema.json` and its
  byte-equal mirror `rules/_schema/v2.schema.json` land at B0-S1
  promotion; a new `rules/_schema/_owners.v2.schema.json` lands
  alongside (no engine-side mirror — the owners schema lives only on
  the rules side, as it does today). The byte-equality CI gate from
  ADR-0001 extends to cover v2 of the rule schema; the existing
  Wave-3 lint pipeline learns to dispatch by version field.

- **C-B0S1.2** — **The shipped rule and its owner entry migrate
  atomically.** At promotion, `rules/customer.yaml` becomes
  `version: 2`, `mode: set`, `kind: set.row_count_positive`;
  `rules/_owners.yaml`'s `customer` entry gains
  `mode: set`. No commit between this study's session and the
  promotion commit may add a new v1 rule (because the v1 schema is
  about to be retired); the next rule added at or after B0-S1
  promotion is a v2 rule.

- **C-B0S1.3** — **Kind-prefix lint gate lands at promotion.** The
  four lint cross-checks committed under Option A land in
  `tools/lint/` at B0-S1's promotion commit. This satisfies ADR-0020
  §C-S.5 — the gate is in place before B0-S2's catalog is finalised,
  so even a partial Wave-S phase cannot leak mode-mismatched rules
  into the runtime.

- **C-B0S1.4** — **Capability is derived, not declared.** P3 is
  satisfied without a `capability:` field. The decision-log Wave-S
  follow-up OQ ("does P3 mean `capability:` is removed entirely or
  kept as a redundant cross-check") is resolved as **no field, no
  cross-check** — the field does not exist today and will not be
  added.

- **C-B0S1.5** — **Engine loader stays set-mode-only.** The loader
  (ADR-0007) gains v2 schema awareness but reads only `mode: set`
  rules until the partial-Wave-S gate closes (B0-S1 + B0-S2 + B0-S3
  promoted). A `mode: record` rule that lands before the partial
  gate is rejected by the loader with a clear "record-mode not yet
  shipped" error — the lint layer should already have caught the
  attempt because B0-S2's catalog will not list any `record.*` kind
  until S2 promotes.

- **C-B0S1.6** — **Until B1-7 resolves, this study's
  one-version-at-rest default holds.** The rule of "one schema
  version at rest in `rules/`" applies as the default. When B1-7
  commits a window (e.g., "v1 supported for N days after v2
  ships"), the default is reconsidered. This study does not
  pre-decide B1-7's outcome. *(New contribution proposed here,
  requires review.)*

### Per-artefact consequences

- **`engine/internal/dsl/schema/v2.schema.json`** — new file at
  promotion. Top-level: `version` (const `2`), `entity`, `mode`
  (enum `["set", "record"]`), `checks` (non-empty array). Each check
  requires `check_id` and `kind` (regex
  `^(set|record)\..+$`; suffix shape finalised by B0-S2), with
  optional `description`.

- **`rules/_schema/v2.schema.json`** — byte-equal mirror of the
  engine source per the ADR-0001 invariant.

- **`rules/_schema/_owners.v2.schema.json`** — new file. Top-level:
  `schema_version` (const `2`), `entities` (object map). Each entity
  requires `mode` (enum `["set", "record"]`), `owner`, `channels`;
  optional `severity_overrides`, `description`.

- **`tools/lint/`** — four new cross-check rules per Option A.
  Existing lint rules (entity-without-owner from ADR-0006 CC9;
  byte-equality gate from ADR-0001) remain in place.

- **`rules/customer.yaml`** — migrates to v2 atomically with B0-S1
  promotion.

- **`rules/_owners.yaml`** — `customer` entity entry gains
  `mode: set` atomically with B0-S1 promotion.

- **Engine runtime changes are limited to the loader** (per
  C-B0S1.5): v2 schema dispatch, mode-field filter (set-only until
  the partial-Wave-S gate closes), and the "record-mode not yet
  shipped" rejection error path.
  ADR-0002/0003/0004/0006/0007/0010/0014/0017 contracts are
  untouched.

---

## Open Questions

- **OQ-B0S1.1** — **Schema-version cadence.** This study commits a
  single v1→v2 bump that adds both `mode:` and tightens `kind` to
  the prefixed form in one move. An alternative would be two
  separate bumps (v2 adds `mode:`; v3 tightens `kind:`). *Out of
  scope for current cycle.* The single bump minimises ADR-0001
  migration overhead and keeps the kind-prefix lint gate atomic with
  the mode primitive's arrival.

- **OQ-B0S1.2** — **Diagnostic-message phrasing.** The exact wording
  of the four lint rules' failure messages (e.g., "rule.mode `set`
  does not match check.kind prefix `record.`") is a `tools/lint/`
  implementation detail. *Out of scope for current cycle.* The
  promotion commit picks phrasing aligned with the existing lint
  binary's style.

- **OQ-B0S1.3** — **Multi-rule files.** The current schema treats
  each YAML file as a single rule (one `entity`, one `checks[]`).
  Whether a future schema version supports multiple rules per file
  (and how their modes would interact) is a separate evolution.
  *Out of scope for current cycle.*

- **OQ-B0S1.4** — **Engine-loader rejection message for `record:`
  rules pre-partial-gate.** Whether the loader emits a separate
  metric/log line distinguishing "record-mode rejected because
  partial gate not closed" from "record-mode rejected because the
  configured kind is not in the catalog" depends on the B0-S2
  catalog shape. *Defer to B0-S2.*

- **OQ-B0S1.5** — **`_owners.yaml` schema-version filename.** The
  existing owners schema lives at
  `rules/_schema/_owners.v1.schema.json`; the v2 file follows the
  same `_owners.v2.schema.json` pattern. Whether the owners schema
  evolves on the same version cadence as the rule schema (lock-step)
  or independently is implicit in this study's "both bump v1→v2 at
  the same promotion". *Out of scope for current cycle;* lock-step
  is the default until a future bump diverges.

---

## Promotion target

**Target:** `docs/adr/0021-mode-as-primitive.md`.

This study promotes to **ADR-0021** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. ADR-0021 is the first per-item ADR of the
Wave-S foundational triplet; per ADR-0020 §Decision (Per-item ADR numbering),
the `0021` slot is descriptive of the expected sequence (S1 → 0021,
S2 → 0022, S3 → 0023) and may shift if an unrelated promotion lands
between B0-S items.

ADR-0021's promotion commit lands the five artefacts committed in
§Consequences above:

1. The new `engine/internal/dsl/schema/v2.schema.json` and its
   byte-equal mirror `rules/_schema/v2.schema.json`.
2. The new `rules/_schema/_owners.v2.schema.json`.
3. The four lint cross-checks in `tools/lint/`.
4. The atomic v1→v2 migration of `rules/customer.yaml` and the
   `customer` entry in `rules/_owners.yaml`.
5. The engine loader gains v2 schema dispatch + mode-field filter
   (set-only until the partial-Wave-S gate closes) + the
   "record-mode not yet shipped" rejection error path
   (per C-B0S1.5).

Per R8, the future ADR-0021 will be rewritten from this study, not
linked back to it.
