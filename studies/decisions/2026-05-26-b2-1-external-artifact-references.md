<!-- path: studies/decisions/2026-05-26-b2-1-external-artifact-references.md -->

# B2-1 — External Artifact References in DSL

## Context

The DSL today carries every check parameter inline in
the rule YAML. The inaugural record-mode rule
`rules/orders_stream.yaml` illustrates the shape: the
`record.schema_conformance` check's `params.schema`
holds a JSON Schema fragment directly:

```yaml
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema:
        type: object
        required: [order_id, event_type]
        properties:
          order_id: { type: string }
          event_type: { type: string }
```

For a small schema this is readable. A production
schema for a complex event commonly grows to 100-300
lines (nested objects, oneOf branches, format
constraints, anchor references). The rule YAML
becomes mostly schema and only marginally
rule-shaped; the *kind* the rule is exercising
disappears under the schema body.

The same gravitational pull applies to other future
parameters that carry structured payloads:

- A "reference list" param (e.g., the set of
  allowed currency codes for a `set.value_in_allowed_set`
  check) could be 200+ items.
- A "regex pattern catalog" for a future
  `record.regex_match` check.
- A "lookup table" for a `record.referential_integrity`
  check.

What none of these are: SQL fragments, code,
expressions. The DSL principle P1 ("rules must remain
declarative; no raw SQL, no embedded expressions, no
escape hatches") still binds.

B2-1 was registered at the W3 backlog-numbering step
with the question:

> Will the DSL ever allow helper files (auxiliary
> SQL, reference payloads) resolved relative to the
> rule origin? The capability is useful but
> dangerous if it becomes an escape hatch.

The "useful" side is rule-author ergonomics (the
schema externalization above). The "dangerous" side
is the slippery slope: once any external reference is
allowed, the next request becomes "external SQL", and
P1 erodes silently.

No prior foundation document, charter clause, or
ADR commits a position on external references in
the DSL; the external-reference posture this study
commits is **new contribution proposed here,
requires review**.

The principles bearing on the decision are **P1**
(rules must remain declarative — no SQL, no
expressions, no escape hatches), **P4** (cost is a
first-class constraint — operator and review cost on
unreadable inline schemas is real), **P5** (evolution
must be contract-driven — any DSL extension needs a
documented contract preventing scope creep), and
**R3** (do not revisit settled architecture —
ADR-0001's compatibility model, ADR-0005's
content-addressed manifest posture, and ADR-0022's
kind catalog all bind).

What B2-1 must commit:

1. **The categorical posture** — does the DSL allow
   *any* external references, or none?
2. **If allowed, the bounded surface** — which
   parameters can be externalized; what kinds of
   artifacts are eligible; which are explicitly
   forbidden?
3. **The resolution mechanism** — when (lint-time,
   publish-time, engine-load-time) and how
   (file-path resolution, content inlining, hash
   verification) the reference becomes content.
4. **The escape-hatch closures** — what mechanisms
   prevent the door from opening to SQL or
   expression escape hatches in the future?
5. **The implementation posture** — design-only here
   or implementation shipped?

---

## Decision Drivers

- **DD-1 — Inline schemas hurt rule-file
  ergonomics at the scale they grow to.** A
  100-line JSON Schema embedded in a 110-line rule
  YAML buries the rule shape under the schema body.
  Reviewers reading the rule file see 90% schema
  and 10% rule. Code-search for "what kind does
  entity X exercise" is harder when the kind is
  one line surrounded by 100 lines of schema.

- **DD-2 — P1's prohibition is about *expressions*
  and *escape hatches*, not about file
  organization.** A JSON Schema fragment is
  declarative-data-with-bounded-expressivity:
  `type`, `required`, `properties`, `enum`,
  `format`, and similar keywords are data; the
  `pattern` keyword embeds a regex which IS an
  expression evaluated against record content.
  This ADR's bounded-scope commitment (Clause 5)
  excludes standalone regex-pattern catalogs from
  v1 external-eligibility precisely because a list
  of regexes is a list of expressions. Regex
  patterns embedded *inside* a declared JSON
  Schema slot ride on the schema's existing
  declarative posture (the schema controls where
  the regex applies and how its result feeds back);
  a standalone regex catalog without that
  containing-schema would be expression-bearing
  data without a declarative wrapper. The
  distinction is: catalogued JSON Schema fragments
  are external-eligible at v1; standalone
  expression catalogs (regex, SQL, etc.) require
  separate ADR review under OQ-2 below.

  Allowing JSON Schema externalization does not
  introduce code, SQL, or unbounded expressions.
  It permits source-organization without
  expanding the DSL's evaluation surface.

- **DD-3 — The escape-hatch slippery slope is real
  and must be closed by mechanism, not by
  promise.** "We won't allow external SQL" as a
  rule is unreliable across operator turnover. The
  closure must be structural: only the params
  fields declared external-eligible **in the kind
  catalog** can use external references; the kind
  catalog is platform-team-owned (CODEOWNERS-routed
  per ADR-0015); adding "SQL fragment" as
  external-eligible would require an ADR amendment
  AND a kind-catalog edit AND platform-team
  approval. The slippery slope has three independent
  brakes.

- **DD-4 — The manifest must stay self-contained
  per ADR-0005's content-addressing posture.**
  ADR-0005 §3 commits content-addressed manifest
  bodies (`manifests/by-hash/sha256-<hex>.json`);
  §4 step 2 commits content-addressed YAML
  bodies (`yamls/by-hash/sha256-<hex>.yaml`). The
  current posture stores every artefact the
  manifest references under a content-addressed
  prefix. Deferring resolution to engine-load-
  time would either require a third
  content-addressed prefix for external artefacts
  (an ADR-0005 amendment) OR break determinism
  when substrate filesystems diverge. Resolution
  must happen **at publish time**: `dq-manifest
  publish` reads the referenced file, validates
  it against the param's schema, and inlines the
  content into the manifest body before computing
  the manifest hash. The engine never sees a
  `_ref`; it only sees inlined content. Manifest
  determinism is preserved; no new
  content-addressed prefix is needed.

- **DD-5 — Lint-time resolution mirrors publish-
  time so authors see local errors.** `dq-lint`
  performs the same resolution as `dq-manifest
  publish`: read the file, validate against the
  param schema, surface errors at the source. The
  rule author runs `make lint-rules` and learns
  about missing files, malformed schemas, or
  forbidden references locally before pushing.

- **DD-6 — File-path safety: relative-only,
  rules-tree-only, no upward traversal, symlink-
  canonicalized before containment check.** A
  rule at `rules/orders_stream.yaml` can reference
  `./schemas/orders.schema.json` (resolving to
  `rules/schemas/orders.schema.json`). It cannot
  reference `../foo.yaml`, `/etc/passwd`,
  `https://example.com/schema.json`, or
  `~/private.json`. The linter and publisher both
  enforce three rules:

  1. **No upward traversal in the reference path**
     — any `..` segment is a hard error before
     any file-system access.
  2. **Symlink canonicalization** — the resolved
     path is run through `filepath.EvalSymlinks`
     (or equivalent realpath resolution) so a
     symlink under `rules/schemas/` pointing at
     `/etc/passwd` cannot bypass the containment
     check.
  3. **Rules-tree containment** — after symlink
     canonicalization, the resolved absolute path
     MUST be a descendant of the `rules/`
     workspace root. Symlinks pointing outside
     `rules/` fail at this check.

- **DD-7 — Only the kind catalog declares
  external-eligible fields.** Today's catalog
  format (`rules/_schema/catalog.v1.yaml`) has
  per-kind `params_schema`. The catalog gains an
  optional `external_eligible_fields` list per
  kind, naming the param fields that may carry a
  `_ref` suffix instead of an inline value. A
  field not declared external-eligible cannot be
  externalized; an attempt to do so is a lint /
  publish error.

- **DD-8 — SQL and expression externalization is
  permanently forbidden by catalog mechanism, not
  by promise.** The catalog's
  `external_eligible_fields` list is bounded to
  parameter types the DSL already accepts —
  structured data (JSON Schema, lookup tables,
  regex patterns as strings). SQL is not a
  catalog-permitted param type today; making it
  one would require an ADR amendment to ADR-0022
  (kind catalog) AND a kind catalog schema bump
  AND platform-team approval. Per-kind
  authorization is the closure.

- **DD-9 — Implementation is deferred to a B2
  consumer slice.** Following the design-only
  pattern from ADR-0030 / ADR-0032 / ADR-0033 /
  ADR-0039 / ADR-0041 / ADR-0042 / ADR-0043, this
  ADR commits the contract; the implementation
  (rule-schema v3 with optional `_ref` suffixes,
  catalog extension, publisher resolution,
  linter resolution, tests) ships as a separate
  session.

---

## Considered Options

### Option 1 — Allow bounded external references; publisher inlines at publish time (recommended)

A rule YAML may carry `params.<field>_ref:
<relative-path>` for any field declared
external-eligible by the kind catalog. The
publisher and linter resolve the path, read the
referenced file, validate the content against the
param's schema, and inline the result into the
manifest body. The engine sees only inlined
content.

Mechanics:

- **Catalog-only extension (additive within v1).**
  The per-kind `params_schema` in the catalog
  expresses the existing inline shape today; that
  schema gains a new optional sibling
  `<field>_ref: { type: string }` for each
  external-eligible field. Because the change
  lives in the catalog's per-kind shape and not
  in the top-level rule schema, **no rule-schema
  bump is needed** — the rule schema (`v2`)
  already delegates per-kind param validation to
  the catalog's `params_schema`. ADR-0001's
  compatibility model classifies this as
  additive-within-v1-catalog.
- **Catalog inventory extension.** Each `kinds[]`
  entry gains an optional
  `external_eligible_fields: [<field>, ...]`
  list naming which param fields accept the
  `_ref` suffix. This list is what governs which
  fields the linter and publisher will resolve;
  a field not in the list cannot be
  externalized.
- **Mutual-exclusion of `<field>` and
  `<field>_ref`** is enforced as a lint
  cross-check (the existing lint cross-check
  numbering from ADR-0022 §"Lint cross-checks" —
  this is cross-check #12), not a pure JSON
  Schema constraint. Pure JSON Schema would
  require a `oneOf` block per external-eligible
  field per kind; lint cross-check is a tighter
  surface that produces clearer error messages.
- **Path resolution** is relative to the rule
  file's directory. `_ref` values MUST NOT
  contain `..` segments; the resolved absolute
  path MUST be under the `rules/` workspace
  root. Both checks are enforced at lint and
  publish.
- **Content inlining** at publish time: the
  publisher reads the file, parses it (JSON or
  YAML), validates against the param's schema,
  and substitutes the parsed content for the
  `_ref` key. The manifest body carries
  `params.<field>: <parsed content>`, never
  `params.<field>_ref: <path>`. Manifest
  determinism (ADR-0005) is preserved — the
  same source inputs produce the same manifest
  hash.
- **Linter resolution** at lint time: `dq-lint`
  performs the same resolution as the publisher
  and surfaces errors locally. A missing
  referenced file, a malformed referenced file,
  a path traversal, a `_ref` for a non-eligible
  field — all are lint errors with file +
  line + column.

Bounded scope:

- **Only structured-data params are
  external-eligible.** JSON Schema, lookup
  tables, regex catalogs, reference value lists.
- **SQL fragments are NOT eligible.** No catalog
  entry today permits a SQL-fragment param;
  permitting one would require an ADR amendment
  to ADR-0022 + a kind catalog schema bump.
- **The kind catalog is platform-team-owned.**
  Adding a new external-eligible field to a kind
  requires platform-team CODEOWNERS approval
  per ADR-0015 §3.

**Strengths.** Closes the rule-file ergonomic
gap (DD-1) without violating P1 (DD-2). Preserves
manifest determinism (DD-4). Closes the
escape-hatch slope through mechanism not promise
(DD-3 + DD-7 + DD-8). Honors the design-only
pattern (DD-9).

**Trade-offs.** Adds a new schema version (v3)
and a catalog field. The publisher and linter
both gain resolution logic. The implementation
slice is substantial (rule schema, owners
schema, catalog, publisher, lint, tests).
Acceptable — the implementation cost is paid
once, the contract is binding.

### Option 2 — No external references; inline everything

Status quo. Every parameter, including
large JSON Schemas, lives inline in the rule
YAML.

**Strengths.** Smallest implementation (zero
new code). Strictest interpretation of P1.

**Trade-offs.** Rule-file ergonomics degrade
as schemas grow. A 200-line schema buried in a
210-line rule YAML is operator-hostile.
Reviewers scrolling through schemas to find the
rule shape is a cost that compounds with every
new record-mode entity. Rejected — the
ergonomic cost is real and the bounded option
above closes the escape-hatch concern through
mechanism, not promise.

### Option 3 — Allow URL-based references (HTTPS to a schema registry)

The rule YAML carries a URL pointing to an
external schema registry (e.g.,
`https://schemas.internal/orders.v1.json`).
The publisher fetches and inlines.

**Strengths.** Schemas could be versioned
independently of rules in a registry.

**Trade-offs.** Introduces an HTTP dependency
at publish time; substrate-dependent (which
registry? what auth?). Manifest reproducibility
depends on the URL's content being immutable
(no version pinning at the URL level). The
linter would need network access at lint time
to validate, breaking the air-gapped local
test posture from ADR-0034. Rejected — the
in-tree relative-path option (Option 1)
delivers the same ergonomic benefit without
network surface.

### Option 4 — Inline-with-YAML-anchors (no file resolution)

Use YAML's built-in `&anchor` / `*alias`
mechanism plus block-folded scalars to make
inline schemas readable without externalizing.
A 200-line schema can be defined once at the
top of the rule file and aliased into the
check's `params.schema`.

**Strengths.** Zero new code; YAML parsers
already handle anchors. No file-resolution
surface; no path safety questions; no
escape-hatch concerns. Compatible with current
v2 schema.

**Trade-offs.** Anchors are *intra-file* only —
two rules cannot share an anchored schema
without duplicating the schema content. The
ergonomic ceiling is the same per-rule
boundary: a single rule with a large schema
still has the schema body inline (just under
an anchor), and cross-rule schema reuse (e.g.,
a common money-type fragment used by orders +
invoices + payments) remains inline-duplicated.
The cross-rule reuse case is the principal
ergonomic driver from DD-1; anchors don't
address it. Rejected — anchors solve a fraction
of DD-1's problem but leave the cross-rule
duplication open. The bounded external-reference
option closes both per-rule and cross-rule.

---

## Recommendation

**Option 1.** Allow bounded external references
via per-field `_ref` suffix; the publisher and
linter both resolve and inline at their
respective times; the engine sees only inlined
content.

### Contract clauses

**Clause 1 — Per-field `_ref` suffix.**

For any param field `<field>` declared
external-eligible by the kind catalog, the
per-kind `params_schema` in the catalog gains a
sibling key `<field>_ref` of type string. The
top-level rule schema (`v2`) is not amended;
the change lives entirely in the catalog
because rule-schema v2 already delegates
per-kind validation to
`catalog.kinds[].params_schema`. ADR-0001
classifies the catalog change as additive
within v1-catalog.

Mutual-exclusion of `<field>` and
`<field>_ref` is enforced as a lint
cross-check (cross-check #12 in ADR-0022's
numbering — see "lint cross-checks" §). Both
present is a lint error with a clear
"choose one" message; neither present is the
schema's existing "field absent" behavior
(legitimate when the field is optional).

**Clause 2 — Kind catalog `external_eligible_fields`.**

Each `kinds[]` entry in BOTH catalog locations
gains an optional list:

- Canonical: `engine/internal/dsl/catalog/v<N>.yaml`
- Mirror: `rules/_schema/catalog.v<N>.yaml`

The byte-equality CI gate from ADR-0001 §C3
ensures the two stay in sync. Example
catalog entry:

```yaml
- name: record.schema_conformance
  external_eligible_fields:
    - schema
```

A field NOT listed cannot be externalized. The
catalog is platform-team-owned per ADR-0015 §3;
adding to this list is a platform-team decision.

For the inaugural entry: `record.schema_conformance`
declares only `schema` as external-eligible. The
nested `aggregation` block on the same kind's
params is **not** external-eligible — `aggregation`
is small (a fixed handful of numeric fields per
ADR-0026), per-rule rather than shared across
rules, and ergonomically fine inline.

**Clause 3 — Path resolution.**

The `_ref` value is a relative path resolved
from the rule file's directory. Three rules:

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

**Clause 4 — Content inlining.**

The publisher reads the referenced file (UTF-8
text), parses it as JSON or YAML (extension-
driven), validates the parsed content against
the param's `params_schema`, and substitutes the
parsed content for the `_ref` key. The manifest
body carries `params.<field>: <parsed content>`;
no `_ref` strings reach the manifest.

Manifest determinism (ADR-0005) is preserved
because the same source inputs (rule file +
referenced file contents) produce the same
inlined manifest body, hence the same
manifest hash.

**Clause 5 — Permanent SQL and expression
prohibition.**

The catalog's `external_eligible_fields` list is
bounded to *structured-data* param types. SQL
fragments are not a catalog-permitted param
type today and adding one would require:

1. An ADR amendment to ADR-0022 (kind catalog).
2. A kind catalog schema bump.
3. Platform-team CODEOWNERS approval on the
   catalog edit.

Three independent brakes close P1's
escape-hatch slope. Any future ADR proposing
SQL-fragment params triggers all three.

**Clause 6 — Implementation deferred.**

The rule schema v3, the catalog extension,
the publisher resolution path, the linter
resolution path, and the test surface ship as a
separate B2 follow-up slice. The contract is
binding at this ADR; the implementation is
mechanical when the slice lands.

### Why this does not reopen P1

P1 forbids raw SQL, embedded expressions, and
escape hatches. External JSON Schema files are
declarative data, not expressions. The catalog
mechanism (Clause 2 + Clause 5) closes the
escape-hatch slope at the contract level —
permitting SQL externalization in the future
would require three independent steps each
gated by platform-team review.

### Why this does not reopen ADR-0005

ADR-0005's content-addressing posture binds the
*manifest body* to be self-contained. Clause 4's
publisher-side inlining preserves that:
references are resolved before the manifest
body is computed and hashed; the manifest body
never carries a `_ref`. The engine reads
self-contained manifest bodies per ADR-0005
unchanged.

### Why this does not reopen ADR-0022

ADR-0022 commits the kind catalog as the
contract surface for per-kind shapes. This ADR
extends the catalog's schema with an optional
`external_eligible_fields` list — additive
within ADR-0001's compatibility model. Existing
catalog entries without the new field are
unchanged in semantics. No amendment.

### Why this does not reopen ADR-0001

ADR-0001 commits the rule-schema compatibility
model and the byte-equality CI gate. Rule schema
v3 (additive `_ref` keys) follows the same
versioning posture as v1 → v2 — additive within
v2 if v2 had supported it, or a new v3 file with
the additive shape. The byte-equality gate
applies to the v3 mirror identically.

---

## Consequences

1. **Bounded external references are committed at
   the contract level.** Per-field `_ref` suffix
   for catalog-declared external-eligible fields;
   publisher + linter inline at their respective
   times; engine sees only inlined content.

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
   continues to work.

4. **Path safety is structurally enforced.**
   Relative paths only; no `..` traversal;
   resolved paths must stay under `rules/`.
   Substrate filesystem ambiguities (symlinks,
   home-directory references, absolute paths)
   cannot bypass the constraint.

5. **The kind catalog gains an optional
   `external_eligible_fields` per-kind list.**
   The first entry naming this list is
   `record.schema_conformance` (`schema`).
   Future catalog additions go through ADR-0022's
   amendment path.

6. **No rule-schema bump is needed.** The change
   lives entirely in the catalog: the per-kind
   `params_schema` gains the optional
   `<field>_ref` sibling for external-eligible
   fields. ADR-0001's compatibility model classifies
   this as additive within v1-catalog; rule schema
   v2 stays the active version, and rule schema v3
   remains TBD per ADR-0035's compatibility-state
   table (it lands when a real breaking change
   requires it, not in this slice). The byte-
   equality CI gate from ADR-0001 §C3 ensures the
   catalog mirror at `rules/_schema/catalog.v<N>.yaml`
   stays in sync with the canonical at
   `engine/internal/dsl/catalog/v<N>.yaml`.

7. **Implementation is deferred to a B2 follow-up
   slice with explicit scope:**
   - Catalog v1 extension: add
     `external_eligible_fields` field to the
     catalog YAML schema (one new optional
     field); update both canonical
     (`engine/internal/dsl/catalog/v1.yaml`) and
     mirror (`rules/_schema/catalog.v1.yaml`).
   - Update the inaugural `record.schema_conformance`
     kind to declare
     `external_eligible_fields: [schema]` and
     to permit `schema_ref: { type: string }` in
     its `params_schema`.
   - Publisher resolution: extend
     `dq-manifest publish` to walk each check's
     params, detect `<field>_ref` keys, run the
     three-step path-safety check (Clause 3),
     read + parse the file, validate against
     the catalog's `params_schema`, and inline
     the parsed content into the manifest body.
   - Linter resolution: extend `dq-lint` with
     the same resolution and add lint
     cross-check #12 (mutual-exclusion).
   - Path-safety helper: shared `pathsafe`
     package (or equivalent) used by both
     publisher and linter for the three-step
     check including symlink canonicalization.
   - Test fixtures: valid `_ref`; `_ref` with
     `..` (rejected); `_ref` pointing via
     symlink outside `rules/` (rejected after
     `EvalSymlinks`); `_ref` for non-eligible
     field (rejected); both `<field>` and
     `<field>_ref` present (rejected as lint
     cross-check #12); referenced file missing
     (rejected with clear error); referenced
     file malformed JSON/YAML (rejected);
     referenced content fails param-schema
     validation (rejected).

8. **B2-1 closes.** The decision-log B2-1 row
   moves to `resolved-adr`. One new B2 row
   registers the implementation slice.

9. **P1 is preserved with explicit fortification.**
   "No raw SQL, no embedded expressions, no
   escape hatches" — the catalog mechanism
   formally fortifies the "no escape hatches"
   clause against future erosion.

10. **ADR-0005, ADR-0022, ADR-0001, ADR-0015 are
    preserved.** This ADR is additive across all
    four contracts without amending any.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and
are explicitly **out-of-scope for current cycle**:

- **OQ-1: Cross-rule reference deduplication.**
  Two rules referencing the same
  `rules/schemas/orders.json` produce two
  inlined copies in the manifest body.
  Deduplication via a new content-addressed
  prefix (e.g., `manifests/by-hash/artefact-<hex>.json`
  with `_ref` pointers in the manifest body
  resolving to the new prefix) would
  **reopen ADR-0005 §1**, which commits the
  exclusive prefix layout. The current approach
  inlines redundantly; this is acceptable while
  the rule count is small. Reserved until the
  manifest size becomes operationally
  significant; revisiting requires an ADR-0005
  amendment.

- **OQ-2: External reference for additional
  param types.** Reference value lists,
  lookup tables, and standalone regex catalogs
  could become external-eligible in future
  catalog additions. The contract permits the
  former two without further DSL change (they
  are declarative-data, same posture as JSON
  Schema fragments); the catalog does not name
  them yet. **Standalone regex catalogs are
  expression-bearing** per DD-2 and require
  separate ADR review before being added to any
  kind's `external_eligible_fields` — the
  threshold is higher than for declarative-data
  externalization. Reserved until concrete
  per-kind demand surfaces.

---

## Promotion target

`docs/adr/0044-external-artifact-references.md` —
next free ADR number. Ships the bounded external-
reference contract (per-field `_ref` suffix;
publisher + linter inlining; catalog-gated
external-eligible fields; path safety;
permanent SQL/expression prohibition).
