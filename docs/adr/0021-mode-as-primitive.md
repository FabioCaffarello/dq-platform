<!-- path: docs/adr/0021-mode-as-primitive.md -->

# ADR-0021 — Mode as Primitive

- **Status:** accepted
- **Date:** 2026-05-24

---

## Context

ADR-0020 launched Wave-S with four locked premises: mode is the
architectural primitive (P1), every DSL kind carries its mode as a
name prefix (P2), capability is derived from mode and not declared
independently (P3), and the unified-vs-parallel runner decision is
reserved for B0-S5 (P4). ADR-0020 §Decision (Sequencing and gates)
assigned **B0-S1** the job of realising P1, P2, and P3 in concrete
schema shape and lint enforcement: the typed `mode` field on the
rule artefact and on the entity declaration, the yaml shape of that
field and its lint-time validation, and the rule that the kind
catalog and the source schema carry mode as their organising key.
B0-S1 is the first of the Wave-S foundational triplet
(B0-S1 → B0-S2 → B0-S3); its promotion delivers the kind-prefix lint
gate that ADR-0020 §C-S.5 commits as the boundary guard between
half-built Wave-S phases.

Until this ADR lands, the rule layer is set-mode-implicit. The
shipped rule schema (`engine/internal/dsl/schema/v1.schema.json`
canonical, `rules/_schema/v1.schema.json` byte-equal mirror per
ADR-0001) carries no `mode` field; the `kind` field accepts any
string. The `_owners.yaml` schema
(`rules/_schema/_owners.v1.schema.json`) carries no `mode` field
and no `capability` field. The first onboarded rule
(`rules/customer.yaml`) uses `kind: row_count_positive` (no `set.`
prefix), and its `_owners.yaml` entry carries no mode marker. The
platform's only shipped rule reads as set-mode by tacit
understanding rather than declaration — exactly the drift P1
exists to forbid.

This ADR closes that drift. It does so without reopening any
set-oriented ADR (ADR-0002 through ADR-0017); the change is
additive, lands under the ADR-0001 compatibility contract as a
schema-version bump, and migrates the single shipped rule and its
`_owners.yaml` entry atomically in the promotion commit.

---

## Decision

### Mode is declared explicitly on both the rule artefact and the entity declaration

A new required field `mode: <"set" | "record">` lives at the top
level of every rule YAML, alongside `version` and `entity`. A new
required field `mode: <"set" | "record">` lives on every entity
entry in `_owners.yaml`, alongside `owner` and `channels`. Neither
field is optional; neither is derived. The rule's mode and the
entity's mode must agree, and both must agree with the kind prefix
on every check the rule contains.

### Kind prefix grammar

Every check's `kind:` is constrained to the pattern
`^(set|record)\..+$`. The prefix (`set.` or `record.`) must equal
the rule's `mode`. The suffix shape (whether dots, hierarchical
names, or other separators are allowed) is finalised by **B0-S2**
(kind catalog); this ADR commits only the prefix grammar.

### Schema-version bump

Both schemas advance from v1 to v2 at this ADR's promotion:

- **Rule schema.** Canonical source at
  `engine/internal/dsl/schema/v2.schema.json`; byte-equal mirror at
  `rules/_schema/v2.schema.json`. The ADR-0001 byte-equality CI
  gate extends to cover v2. The v2 schema declares `version`
  (const `2`), `entity`, `mode` (enum `["set", "record"]`), and
  `checks` (non-empty array) as required. Each check requires
  `check_id` and `kind` (regex `^(set|record)\..+$`; suffix shape
  finalised by B0-S2), with optional `description`.
- **Owners schema.** New file at
  `rules/_schema/_owners.v2.schema.json`. The v2 owners schema
  declares `schema_version` (const `2`) and `entities` (object
  map) as required. Each entity declaration requires `mode` (enum
  `["set", "record"]`), `owner`, and `channels`; `severity_overrides`
  and `description` remain optional.

The owners schema lives only on the rules side (no engine-side
mirror), as it did in v1.

### Lint cross-checks

The `tools/lint/` binary gains four cross-checks at this ADR's
promotion:

1. Rule's `mode` is one of `set`, `record` (schema-enforced).
2. Every check's `kind` matches the rule's `mode` prefix.
3. Rule's `mode` equals the `_owners.yaml` entry's `mode` for the
   rule's `entity`.
4. `_owners.yaml` entity's `mode` is one of `set`, `record`
   (schema-enforced).

These four checks satisfy ADR-0020 §C-S.5 — the kind-prefix
discipline lands at the lint layer with this ADR's promotion, even
if B0-S2's catalog is still in draft.

### Engine loader behaviour

The engine loader (ADR-0007) gains three behaviours at this ADR's
promotion:

- **v2 schema dispatch.** The loader recognises the `version`
  field and dispatches to the v2 schema validator. The v1 schema
  remains readable but no v1 rules are at rest in `rules/` after
  the promotion commit's atomic migration.
- **Mode-field filter.** The loader accepts only `mode: set` rules
  until the partial-Wave-S gate closes (B0-S1, B0-S2, B0-S3 all at
  `resolved-adr`). The filter is set-mode-only as the engine
  runtime is itself set-mode-only at this point.
- **Rejection error.** A `mode: record` rule that reaches the
  loader is rejected with a clear "record-mode not yet shipped"
  error. The lint layer should already have caught the attempt
  because B0-S2's catalog will not list any `record.*` kind until
  S2 promotes; the loader-side rejection is defense at the runtime
  boundary.

### Atomic migration

In the promotion commit, the single shipped rule and its owners
entry move to v2 atomically. `rules/customer.yaml` becomes
`version: 2`, `mode: set`, `kind: set.row_count_positive`. The
`customer` entry in `rules/_owners.yaml` gains `mode: set`. No
intermediate state cohabits v1 and v2.

### Capability is not declared

`rules/_owners.yaml` carries no `capability:` field today, and this
ADR confirms it will not be added. Capability is fully derived from
mode per P3. The Wave-S launch ADR's OQ on this question is
redeemed: no field, no cross-check.

### One schema version at rest

Until B1-7 (compatibility window) resolves, the rule of **one
schema version at rest in `rules/`** applies as the default. No
commit may carry rules at multiple schema versions in `rules/`
simultaneously. When B1-7 commits a window, the default is
reconsidered.

---

## Consequences

1. **Set-oriented ADRs are not reopened.** ADR-0002, ADR-0003,
   ADR-0004, ADR-0006, ADR-0007, ADR-0010, ADR-0014, and ADR-0017
   stay set-mode-scoped per their 2026-05-23 scope notes. The mode-
   field landing is additive; no set-mode contract is amended by
   this ADR.

2. **Kind-prefix discipline is in force at the lint layer.** The
   four cross-checks above ship with this ADR's promotion and are
   in effect on every subsequent rule commit. ADR-0020 §C-S.5 is
   satisfied — the gate is in place before B0-S2's catalog is
   finalised, so half-built Wave-S state cannot leak mode-mismatched
   rules into the runtime.

3. **The engine loader is no longer mode-agnostic.** The loader
   gains the v2 dispatch + mode-field filter + record-mode
   rejection error described above. ADR-0007's existing contracts
   (startup-mode behaviour, refresh-mode behaviour, retry budget,
   orphan-run detection) are untouched. The loader changes are
   set-mode-additive: they make the existing set-mode path explicit
   about its mode and reject everything else until the partial-
   Wave-S gate closes.

4. **The partial-Wave-S gate is one item closer.** With B0-S1 at
   `resolved-adr`, the partial-Wave-S gate (B0-S1 + B0-S2 + B0-S3
   all at `resolved-adr`) is 1/3 closed. B0-S2 (kind catalog) opens
   as the next item per ADR-0020 §Decision (Sequencing and gates).

5. **Capability declaration is settled.** No `capability:` field
   on `_owners.yaml`. The Wave-S launch ADR's OQ on this question
   is closed by this ADR's §Decision (Capability is not declared).

6. **B1-7 inherits a default rule.** Until B1-7 commits a
   compatibility window, "one schema version at rest in `rules/`"
   holds. B1-7's eventual study may relax this default; this ADR
   does not pre-decide that outcome.

7. **No record-mode rule may ship before the partial-Wave-S gate
   closes.** The Wave-S analogue of R1 (ADR-0020 §C-S.4) is
   enforced both by lint (rule 2 above — kind prefix must match
   rule mode, and only `set.*` kinds exist in the catalog until
   B0-S2) and by the loader (the `mode: record` rejection error
   path). The kind-prefix lint gate is the single exception to the
   Wave-S R1-analogue, and it ships with this ADR.

8. **Schema-version cadence is one bump for two concerns.** The
   v1→v2 transition carries both the new `mode:` field and the
   `kind:` prefix tightening in a single bump. A two-step transition
   (v2 adds `mode:`; v3 tightens `kind:`) was considered and
   declined; the single bump minimises migration overhead and keeps
   the kind-prefix lint gate atomic with the mode primitive's
   arrival.

---

## Notes

- The exact wording of the four lint rules' diagnostic messages
  (e.g., "rule.mode `set` does not match check.kind prefix
  `record.`") is a `tools/lint/` implementation detail picked at
  the promotion commit, aligned with the existing lint binary's
  diagnostic style.
- Whether a future schema version supports multiple rules per file
  (and how their modes interact) is a separate evolution.
- Whether the loader emits a distinct metric or log line for
  "record-mode rejected because partial gate not closed" versus
  "record-mode rejected because the configured kind is not in the
  catalog" depends on the B0-S2 catalog shape and is decided in
  B0-S2.
- The owners schema evolves on the same version cadence as the
  rule schema by default (both bumped v1→v2 at this ADR's
  promotion). Future divergence is allowed if a future schema-side
  decision requires it.
- The unified-vs-parallel runner question (P4) remains deferred to
  B0-S5; this ADR's mode-field shape is neutral on runner shape.
  Both the unified-runner branch and the parallel-runner branch
  consume the same `mode:` field as the dispatch key.
