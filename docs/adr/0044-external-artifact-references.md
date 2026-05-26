<!-- path: docs/adr/0044-external-artifact-references.md -->

# ADR-0044 — External Artifact References in DSL

- **Status:** accepted
- **Date:** 2026-05-26

---

## Context

The DSL today carries every check parameter inline
in the rule YAML. The inaugural record-mode rule
`rules/orders_stream.yaml` illustrates the shape:
the `record.schema_conformance` check's
`params.schema` holds a JSON Schema fragment
directly. For a small schema this is readable; a
production schema for a complex event commonly
grows to 100-300 lines, at which point the rule
YAML becomes mostly schema and only marginally
rule-shaped.

The same gravitational pull applies to other
future structured-data parameters: reference value
lists for `set.value_in_allowed_set`-style kinds,
lookup tables for referential-integrity kinds, and
similar.

What none of these are: SQL fragments, code,
expressions. The DSL principle P1 ("rules must
remain declarative; no raw SQL, no embedded
expressions, no escape hatches") still binds.

No prior foundation document, charter clause, or
ADR commits a position on external references in
the DSL. The bounded external-reference posture
this ADR commits is **new contribution requiring
review** and is reviewed in this ADR.

The principles bearing on the decision are **P1**
(declarative-only rules), **P4** (cost is a
first-class constraint — operator and review cost
on unreadable inline schemas is real), **P5**
(evolution must be contract-driven), and **R3**
(do not revisit settled architecture — ADR-0001,
ADR-0005, ADR-0022 all bind).

---

## Decision

The DSL allows **bounded external references** for
structured-data parameters declared
external-eligible by the kind catalog. The publisher
and linter both resolve and inline references at
their respective times; the engine sees only
inlined content; the manifest body stays
self-contained per ADR-0005.

### Clause 1 — Per-field `_ref` suffix

For any param field `<field>` declared
external-eligible by the kind catalog, the
per-kind `params_schema` in the catalog gains a
sibling key `<field>_ref` of type string. The
top-level rule schema (`v2`) is not amended;
the change lives entirely in the catalog because
rule-schema v2 already delegates per-kind param
validation to `catalog.kinds[].params_schema`.
ADR-0001's compatibility model classifies this
as additive within v1-catalog.

Mutual-exclusion of `<field>` and `<field>_ref`
is enforced as a lint cross-check (cross-check
#12 in ADR-0022's numbering) rather than a pure
JSON Schema constraint. Both present is a lint
error with a clear "choose one" message; neither
present is the schema's existing "field absent"
behavior.

### Clause 2 — Catalog `external_eligible_fields`

Each `kinds[]` entry in BOTH catalog locations
gains an optional `external_eligible_fields` list:

- **Canonical:** `engine/internal/dsl/catalog/v<N>.yaml`
- **Mirror:** `rules/_schema/catalog.v<N>.yaml`

The byte-equality CI gate from ADR-0001 §C3
ensures the two stay in sync. Example catalog
entry:

```yaml
- name: record.schema_conformance
  external_eligible_fields:
    - schema
```

A field NOT listed cannot be externalized. The
catalog is platform-team-owned per ADR-0015 §3;
adding to this list is a platform-team decision.

For the inaugural entry, `record.schema_conformance`
declares only `schema` as external-eligible. The
nested `aggregation` block on the same kind's
params is **not** external-eligible —
`aggregation` is small (a fixed handful of
numeric fields per ADR-0026), per-rule rather
than shared across rules, and ergonomically fine
inline.

### Clause 3 — Path resolution

The `_ref` value is a relative path resolved from
the rule file's directory. Three rules:

1. **No upward traversal in the reference path.**
   Any `..` segment in the literal reference
   value is a hard error before any file-system
   access.
2. **Symlink canonicalization.** The resolved
   path is run through `filepath.EvalSymlinks`
   (or equivalent realpath resolution) before
   the containment check, so a symlink under
   `rules/schemas/` pointing at `/etc/passwd`
   cannot bypass containment.
3. **Rules-tree containment.** After symlink
   canonicalization, the resolved absolute path
   MUST be a descendant of the `rules/`
   workspace root. Paths resolving outside
   `rules/` are a hard error.

### Clause 4 — Content inlining at publish time

The publisher reads the referenced file (UTF-8
text), parses it as JSON or YAML
(extension-driven), validates the parsed content
against the param's `params_schema`, and
substitutes the parsed content for the `_ref`
key. The manifest body carries
`params.<field>: <parsed content>`; no `_ref`
strings reach the manifest.

Manifest determinism (ADR-0005) is preserved
because the same source inputs (rule file +
referenced file contents) produce the same
inlined manifest body, hence the same manifest
hash. Deferring resolution to engine-load-time
would either require a new content-addressed
prefix (ADR-0005 amendment) or break determinism
across substrates; publish-time inlining avoids
both.

### Clause 5 — Permanent SQL and expression prohibition

The catalog's `external_eligible_fields` list is
bounded to *structured-data* param types
(declarative JSON Schema fragments at v1;
reference value lists and lookup tables permitted
in future additions per OQ-2). Standalone
expression-bearing artefacts — SQL fragments,
standalone regex catalogs, code fragments — are
NOT eligible.

A JSON Schema fragment is
declarative-data-with-bounded-expressivity: most
keywords are pure data (`type`, `required`,
`properties`, `enum`, `format`); the `pattern`
keyword embeds a regex which is an expression,
but it rides on the schema's declarative wrapper
(the schema controls where `pattern` applies and
how its result feeds back). A *standalone* regex
catalog without that containing-schema would be
expression-bearing data without a declarative
wrapper.

The escape-hatch slope is closed by three
independent brakes. Future externalization of a
SQL fragment param would require:

1. An ADR amendment to ADR-0022 (kind catalog).
2. A kind catalog schema bump.
3. Platform-team CODEOWNERS approval on the
   catalog edit.

### Clause 6 — Implementation deferred

The catalog extension, the publisher resolution
path, the linter resolution path, and the test
surface ship as a separate B2 follow-up slice.
The contract is binding at this ADR; the
implementation is mechanical when the slice
lands. Explicit slice scope:

- Catalog v1 extension: add
  `external_eligible_fields` field to the
  catalog YAML schema; update both canonical
  and mirror.
- Update the inaugural `record.schema_conformance`
  kind to declare
  `external_eligible_fields: [schema]` and to
  permit `schema_ref: { type: string }` in its
  `params_schema`.
- Publisher resolution: extend `dq-manifest
  publish` to walk each check's params, detect
  `<field>_ref` keys, run the three-step
  path-safety check (Clause 3), read + parse
  the file, validate against the catalog's
  `params_schema`, and inline.
- Linter resolution: extend `dq-lint` with the
  same resolution and add lint cross-check #12
  (mutual-exclusion).
- Path-safety helper: shared `pathsafe`
  package (or equivalent) used by both
  publisher and linter for the three-step check
  including symlink canonicalization.
- Test fixtures: valid `_ref`; `_ref` with
  `..` (rejected); `_ref` pointing via symlink
  outside `rules/` (rejected after
  `EvalSymlinks`); `_ref` for non-eligible
  field (rejected); both `<field>` and
  `<field>_ref` present (rejected as cross-check
  #12); referenced file missing; referenced
  file malformed JSON/YAML; referenced content
  fails param-schema validation.

### Why this does not reopen P1

P1 forbids raw SQL, embedded expressions, and
escape hatches. External JSON Schema files are
declarative data, not expressions. Clause 5
closes the escape-hatch slope through three
independent brakes — permitting SQL or standalone
expression-catalog externalization in the future
requires ADR amendment + catalog schema bump +
platform-team approval. The mechanism is the
closure, not promise.

### Why this does not reopen ADR-0005

ADR-0005 §3 commits content-addressed manifest
bodies; §4 step 2 commits content-addressed YAML
bodies. Clause 4's publish-time inlining
preserves both: references are resolved before
the manifest body is computed and hashed; the
manifest body never carries a `_ref`. No new
content-addressed prefix is introduced; the
engine reads self-contained manifest bodies per
ADR-0005 unchanged.

### Why this does not reopen ADR-0022 or ADR-0001

ADR-0022 commits the kind catalog as the contract
surface for per-kind shapes. This ADR extends the
catalog's schema with an optional
`external_eligible_fields` list and an additive
`<field>_ref` sibling on per-kind `params_schema`
— additive within ADR-0001's compatibility model.
Existing catalog entries without the new fields
are unchanged in semantics. The byte-equality CI
gate applies to both catalog locations.

---

## Consequences

1. **Bounded external references are committed at
   the contract level.** Per-field `_ref` suffix
   for catalog-declared external-eligible
   fields; publisher + linter inline at their
   respective times; engine sees only inlined
   content.

2. **The escape-hatch slope is closed by
   mechanism, not promise.** Three independent
   brakes (ADR amendment + catalog schema bump +
   platform-team CODEOWNERS) gate any future
   attempt to permit SQL or expression
   externalization.

3. **Manifest determinism (ADR-0005) is
   preserved.** References resolve before
   manifest body computation; the manifest body
   stays self-contained; content addressing
   continues to work; no new content-addressed
   prefix is introduced.

4. **Path safety is structurally enforced.**
   Relative paths only; no `..` traversal;
   symlink canonicalization before the
   containment check; resolved paths must stay
   under `rules/`. Substrate filesystem
   ambiguities (symlinks, home-directory
   references, absolute paths) cannot bypass
   the constraint.

5. **The kind catalog gains an optional
   `external_eligible_fields` per-kind list.**
   The first entry naming this list is
   `record.schema_conformance` (`schema` only;
   `aggregation` is intentionally inline-only).
   Future catalog additions go through ADR-0022's
   amendment path.

6. **No rule-schema bump is needed.** The change
   lives entirely in the catalog: the per-kind
   `params_schema` gains the optional
   `<field>_ref` sibling for external-eligible
   fields. ADR-0001's compatibility model
   classifies this as additive within v1-catalog;
   rule schema v2 stays the active version, and
   rule schema v3 remains TBD per ADR-0035's
   compatibility-state table.

7. **Implementation is deferred to a B2 follow-up
   slice with explicit scope:** catalog v1
   extension; inaugural-kind update to declare
   `external_eligible_fields: [schema]`;
   publisher resolution; linter resolution + new
   cross-check #12 (mutual-exclusion); shared
   path-safety helper; nine documented test
   fixtures.

8. **B2-1 closes.** The decision-log B2-1 row
   moves to `resolved-adr`. One new B2 row
   registers the implementation slice.

9. **P1 is preserved with explicit
   fortification.** The "no escape hatches"
   clause is mechanically reinforced by Clause 5's
   three brakes.

10. **ADR-0005, ADR-0022, ADR-0001, ADR-0015,
    ADR-0026 are preserved.** This ADR is
    additive across all without amending any.

11. **Two deferred items registered out-of-scope:**

    - **OQ-1: Cross-rule reference
      deduplication.** Two rules referencing the
      same `rules/schemas/orders.json` produce
      two inlined copies. Deduplication via a
      new content-addressed prefix would reopen
      ADR-0005 §1's exclusive prefix layout.
      Reserved until manifest size becomes
      operationally significant.

    - **OQ-2: External reference for additional
      param types.** Reference value lists and
      lookup tables can become external-eligible
      via straightforward catalog additions
      (same declarative-data posture as JSON
      Schema). Standalone regex catalogs are
      expression-bearing per Clause 5 and require
      separate ADR review before being added to
      any kind's `external_eligible_fields`.
