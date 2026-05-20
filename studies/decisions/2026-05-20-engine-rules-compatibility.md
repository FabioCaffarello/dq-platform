<!-- path: studies/decisions/2026-05-20-engine-rules-compatibility.md -->

# B0-1 — Engine ↔ Rules Compatibility Declaration

## Metadata

- B0 reference: B0-1 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- Status: draft (Wave 1, session 2).
- Last updated: 2026-05-20.
- Promotion target: see final section.

---

## Context

The platform lives in a monorepo where `engine/` (runtime, schema
source of truth) and `rules/` (declarative YAML rule specifications)
evolve side by side. Per
[`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md),
the workspaces have different change rates, different reviewers, and
different release artifacts: the engine ships as a container under
`engine-v*`; the rules ship as a manifest under `rules-v*` published
to object storage; the linter ships under `tools-lint-v*`. Even
though everything moves through the same commit graph, the boundary
between engine and rules is a real interface — see
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md).

The question for B0-1, as recorded in the decision log:

> How does the rules workspace declare which schema and linter
> contract it follows, given both live in the same repository?

The foundation contract document proposes a triple-layer shape
(per-rule `version`, mirrored schema under `rules/_schema/`,
manifest with `schema_version` and `engine_compatibility`). That
shape is descriptive — B0-1 must either lock it as the canonical
declaration model or replace it. This study locks it, names every
declaration point precisely, and specifies the CI gate that keeps
the layers consistent.

The decision matters because, per platform principle P5 (evolution
must be contract-driven), every degradation of the project over time
will trace back to a weakening of one of the four boundary surfaces
(schema, linter, manifest, compatibility) — usually through
expedient shortcuts that "just this once" bypass the contract. B0-1
defines what those shortcuts would look like, so review pressure can
catch them.

This study covers **declaration only**. The mechanics of manifest
publication (write-then-swap, lease semantics) are B0-5. The
mechanics of loader failure on contract mismatch are B0-7.

---

## Decision Drivers

The decision must satisfy the following constraints, in priority
order.

1. **D1. Each rule artifact must be self-describing.** A reader of
   one YAML file must be able to determine, without external lookup,
   which schema version that rule was written against. This protects
   reviewability and supports a multi-version migration window —
   foundation doc
   [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
   §"How new versions are introduced".

2. **D2. The rules workspace must be lintable without depending on
   engine internals at lint time.** Domain teams should be able to
   validate their YAMLs against a schema artifact that lives inside
   the rules workspace, not by importing engine packages. This is
   one of the success criteria for the `rules/` workspace per
   [`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md)
   §"`rules/` — the authoring surface": *"a domain team can add or
   evolve rules without needing engine internals."*

3. **D3. The engine must refuse to load any manifest whose contract
   declarations are inconsistent or unsupported.** This is required
   by PAT-1 (fail-fast registry loading) in
   [`04-system-architecture.md`](../foundation/04-system-architecture.md):
   no partial loading, no silent skipping.

4. **D4. The schema source of truth must be unambiguous and live in
   exactly one place.** Any mirror or copy must be derived by a
   mechanical, CI-enforced process — never edited by hand. This is
   what prevents the slow divergence that monorepo discipline is
   supposed to forbid (foundation doc
   [`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md)
   §"Anti-Goals").

5. **D5. Linter, engine, and rules must each declare their slice of
   the contract independently, so any pair can fail closed.** This
   is the "defense in depth" point of
   [`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
   §"Why the linter and the engine validate independently".
   Centralizing the declaration in one place creates a single point
   of failure.

6. **D6. The declaration must support a multi-version compatibility
   window.** During migration from schema `vN` to `vN+1`, the rules
   workspace must be able to hold rules at both versions
   simultaneously, and the engine must be able to load both. The
   exact window duration is B1-7; the shape of the declaration must
   not preclude it.

7. **D7. The declaration overhead per rule must be small.** Domain
   teams are not platform engineers. A rule's contract declaration
   should be one line, not a header block — foundation doc
   [`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md)
   §"`rules/` — the authoring surface": authoring must stay
   declarative and low-friction.

---

## Considered Options

### Option A — Implicit alignment (no declarations)

The monorepo guarantees that engine, rules, and linter all move
together. A rule YAML carries no `version:` field. There is no
schema mirror in `rules/`. The manifest does not carry a
`schema_version` field. CI checks consistency on every merge by
running the linter (which uses the current schema from `engine/`)
across every YAML.

**Trade-offs.**

- Pro: zero per-rule ceremony; nothing to keep in sync.
- Pro: simplest CI surface.
- Con: violates D1 — a rule YAML is no longer self-describing; you
  have to look at the commit it was written in to know its schema
  version.
- Con: violates D6 — a multi-version migration window is impossible
  without per-rule version declaration.
- Con: violates D5 — the published manifest carries no compatibility
  data, so the engine cannot fail closed if it is handed a manifest
  built for a different schema major version (which becomes possible
  the moment the engine and the manifest object storage are decoupled
  in time, which they always are: the engine reads a manifest that
  was published earlier).
- Con: violates D2 only weakly — the linter could still mirror the
  schema internally, but with no per-rule declaration the workspace
  itself becomes opaque.
- Con: violates the spirit of P5 — the boundary becomes a convention,
  not a contract.

This option is rejected because the engine and the manifest are
decoupled in time even within a single monorepo (the manifest is
published from CI; the engine reads it later). "We move together in
git" does not extend to runtime.

### Option B — Per-rule version only

Each rule YAML declares its schema version as a top-level field:

```yaml
version: 1
entity: customer
checks:
  - ...
```

No schema mirror in `rules/`. The linter and the engine both read
the canonical schema from `engine/internal/dsl/schema/`. The
manifest does not carry a `schema_version`.

**Trade-offs.**

- Pro: satisfies D1 (per-rule self-describing).
- Pro: supports D6 (multi-version coexistence in the workspace).
- Pro: low D7 cost — one line.
- Con: violates D2 — the linter binary lives in `tools/` and reads
  the schema from `engine/internal/dsl/schema/`. That makes
  validating any YAML in `rules/` a transitive dependency:
  `rules/` → `tools/` (linter) → `engine/` (schema). The rules
  workspace cannot be linted by reading `rules/` alone; removing
  or restructuring `engine/internal/dsl/schema/` breaks the rules
  pipeline. `rules/` stops being self-contained for validation
  even though it remains self-contained for authoring.
- Con: violates D5 — manifest carries no compatibility data, same
  failure mode as Option A at runtime.
- Con: pushes the "did the engine version that built this manifest
  actually support every rule's declared version" check entirely to
  the engine load step, with no earlier guard.

This option is rejected because the manifest is the artifact the
engine consumes at runtime. Anything not declared in the manifest
cannot be checked at load time. The manifest must carry the
contract.

### Option C — Workspace-level pinning only

A single declaration file at the root of `rules/` (e.g.
`rules/_schema/CONTRACT.yaml`) declares the workspace's schema
version and required linter version range. Individual rule YAMLs
carry no version field. The manifest reads the workspace contract
and propagates it.

```yaml
# rules/_schema/CONTRACT.yaml (illustrative, not part of this study)
schema_version: 1
linter: ">=2.3.0 <3.0.0"
```

**Trade-offs.**

- Pro: one declaration point for the whole workspace.
- Pro: lowest per-rule ceremony.
- Con: violates D1 — a rule YAML in isolation no longer carries its
  schema version. Reviewing a single rule file requires opening the
  workspace contract file.
- Con: violates D6 — the workspace can only declare one schema
  version. The whole workspace must migrate at once, which is the
  exact failure mode the compatibility window is meant to prevent.
- Con: large mass migrations create review-time risk: many YAMLs
  change at once, the diff is dominated by mechanical churn, and
  unrelated breaking changes can ride along.

This option is rejected because it makes the compatibility window
unusable. A workspace-level pin forces "big-bang" migrations.

### Option D — Triple-layer declaration with CI-gated mirror

Three declaration points, each independent, each verified by a
separate check.

**D-1. Per-rule (in every rule YAML).** Required top-level field:

```yaml
version: 1
entity: customer
checks:
  - ...
```

The field is mandatory. The linter rejects any rule without it. The
engine refuses to index any rule without it at load time.

**D-2. Workspace schema mirror.** The rules workspace contains a
byte-identical copy of every supported schema version at:

```
rules/_schema/v1.schema.json
rules/_schema/v2.schema.json     # added when v2 is introduced
```

The source of truth lives at `engine/internal/dsl/schema/v<N>.schema.json`.
The mirror is **never edited by hand**. CI runs a single
byte-equality gate on every merge request and on every push to
`main`: `cmp engine/internal/dsl/schema/v<N>.schema.json rules/_schema/v<N>.schema.json`
for every `N`. If any pair diverges, the merge is blocked. The gate
is mechanical (a `diff` or `cmp` invocation, not a parser) because
mechanical comparisons are harder to break than semantic ones.

**D-3. Published manifest contract.** The manifest carries three
contract fields:

```jsonc
{
  "manifest_version": 1,
  "ruleset_version": "rules-v2.4.7",
  "schema_versions_present": [1],         // which schemas appear in this ruleset
  "engine_compatibility": ">=2.0.0 <3.0.0",
  "linter_used": "tools-lint-v2.3.1",
  ...
}
```

- `schema_versions_present` is the set of schema versions actually
  declared by rule YAMLs inside this manifest. During a single-version
  steady state it is `[1]`. During a v1→v2 migration window it is
  `[1, 2]`.
- `engine_compatibility` is a semver range over engine releases that
  support every version in `schema_versions_present`.
- `linter_used` is the linter release that validated this manifest,
  recorded for audit and reconstruction (G4 in foundation doc
  [`04-system-architecture.md`](../foundation/04-system-architecture.md)).
  **Not a runtime gate; consumed only for audit/reconstruction.**
  The engine does not read or verify this field at load time.

The engine, at load time, verifies that:

1. Every rule in the manifest has a `version` field whose value is
   in the engine's set of supported schema versions.
2. `schema_versions_present` matches the set of `version` values
   actually observed across the manifest's rules.
3. The engine's running version satisfies `engine_compatibility`.

Any failure aborts load (PAT-1, no partial loading).

**Linter version pinning.** The rules workspace's CI pipeline pins a
specific linter release in its CI configuration file (mechanism per
C10). The pin is **not** declared in any rule YAML or in `rules/`
metadata. Rationale: the pinned linter version is a build concern,
not a contract concern — the contract concern is *which schemas the
chosen linter supports*, and that is declared in the linter's own
release notes (foundation doc
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
§"How it is versioned"). Mixing build pins into rule artifacts would
violate D7 by adding ceremony that domain teams cannot meaningfully
maintain.

**Trade-offs.**

- Pro: satisfies D1, D2, D3, D5, D6, D7 directly.
- Pro: satisfies D4 — schema source of truth is unambiguous
  (`engine/internal/dsl/schema/`), mirror is mechanically derived.
- Pro: every declaration point can fail closed independently
  (linter, CI gate, manifest publication, engine load).
- Con: schema appears twice in the repository. This is the cost of
  D2 + D5. The CI byte-equality gate makes the duplication safe; the
  alternative (`rules/` reaches into `engine/`) violates the
  workspace boundary that the rest of the topology document is
  trying to defend.
- Con: four moving parts to keep aligned (per-rule version, mirror,
  manifest contract block, linter release notes). Mitigated by:
  the gate is mechanical; the mirror is auto-updatable by a
  Makefile target that copies from the source.

### Option E — Triple-layer declaration with derived mirror (no checked-in copy)

Same as Option D, but `rules/_schema/v<N>.schema.json` is **not**
checked into the repository. Instead, a CI step copies the schema
from `engine/internal/dsl/schema/` into `rules/_schema/` at the
start of the rules pipeline, and the linter consumes that
just-copied file.

**Trade-offs.**

- Pro: schema lives in only one place on disk; no on-disk
  duplication.
- Con: shifts the equality invariant from a **single observable
  gate** (Option D's CI byte-equality check, which either passes
  or fails loudly on every MR) to a **distributed invariant** —
  every environment that runs the copy step (each CI runner, each
  developer Makefile target, each devcontainer, each CI cache
  layer) must produce a byte-identical copy. Distributed
  invariants drift silently; single observable gates fail loudly.
- Con: a developer who edits `rules/_schema/` locally (since the
  file exists on disk after the copy step runs) sees their edit
  overwritten on the next pipeline run with no error surface —
  the failure is invisible by design.
- Con: a clean clone followed by `make lint-rules` without first
  running the copy step yields a confusing "schema not found"
  error. This is a consequence of the distributed invariant, not
  an independent failure mode.

**On D4 specifically.** D4 forbids *uncontrolled* duplication —
divergent copies edited by hand. Option D has two files on disk,
but only one is editable (`engine/internal/dsl/schema/`); the
other is enforced equal by CI and owned by the schema owner per
C8. That is not the divergence D4 was written to prevent. Option
E removes the second file, but in doing so it scatters the
equality invariant across N environments — converting a single,
observable, fail-loud check into N invisible ones. D4's goal is
preserved better by Option D's single gate than by Option E's
distributed copy step.

This option is rejected on the failure-mode argument above. The
clean-clone friction is a downstream consequence of the same
issue, not the primary objection.

---

## Recommendation

Adopt **Option D** as the canonical compatibility declaration model.
This locks the contract shape proposed in
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
and adds three specific commitments beyond what that document
states:

1. The schema mirror under `rules/_schema/` is **checked in**, not
   generated at CI time.
2. The byte-equality check between `engine/internal/dsl/schema/v<N>.schema.json`
   and `rules/_schema/v<N>.schema.json` is a **mandatory CI gate on
   `main`** — not an advisory check.
3. The published manifest carries **three semantic commitments**:
   (a) the set of schema versions present in the ruleset; (b) an
   expression of which engine versions are compatible with that
   ruleset; (c) a reference to the linter release that validated
   the manifest, used only for audit/reconstruction (G4). The JSON
   field names, structure, and encoding are B0-5's call; this
   study commits only that the three values exist in the manifest
   and that B0-5's chosen shape carries all three. The field names
   used in Option D above (`schema_versions_present`,
   `engine_compatibility`, `linter_used`) are illustrative.

The recommendation is grounded in foundation documents
[`01-charter-and-principles.md`](../foundation/01-charter-and-principles.md)
(P5),
[`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md)
(workspace boundaries, anti-goals), and
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
(four-surface contract). The specific commitments above
("checked-in mirror", "byte-equality gate is mandatory", "three
manifest contract fields with the names listed") are
**new contribution proposed here, requires review**.

---

## Consequences

Adopting this recommendation commits the platform to the following.

**C1. Schema duplication is intentional, not technical debt.** Any
future contributor proposing to "deduplicate" the schema by
deleting `rules/_schema/` and pointing `rules/` at `engine/` must
re-open this decision. The duplication exists to keep `rules/`
lintable in isolation.

**C2. The byte-equality CI gate is load-bearing.** It cannot be
weakened to a "warning" or a "soft-fail". If the gate breaks (CI
infrastructure outage, runner misconfiguration), merging to `main`
must be paused until it is restored. The cost of letting one
mismatch through is the entire premise of the boundary contract.
The gate's ongoing cost — one byte-comparison per schema file per
MR — is trivial relative to the failure mode it prevents (P4).

**C3. The schema source of truth is `engine/internal/dsl/schema/`.**
All schema edits happen there. The mirror is updated mechanically
(by a Makefile target invoking `cp` or equivalent) in the same MR
as the schema change. Reviewers reject MRs that edit
`rules/_schema/` without an accompanying source change in `engine/`.

**C4. Every rule YAML must declare its schema version.** The linter
rejects rules without a `version:` field. The engine refuses to
load rules without one. There is no default. A missing `version:`
is a hard error at the earliest possible point.

**C5. Manifest publication is gated on contract consistency.** Any
manifest publication path B0-5 specifies must, before producing a
manifest, verify (a) every rule's declared `version:` is in the
engine's currently-supported set, (b) the set of schema versions
present in the manifest equals the set of `version:` values
observed across the packaged YAMLs, and (c) every value in that
set has a corresponding mirror under `rules/_schema/`. The
mechanism — how the publisher is structured, where it runs, how it
fails — is B0-5's call. This study commits only that the three
verifications above must happen at publish time.

**C6. Engine load fails closed on contract mismatch.** The engine's
load behavior must fail closed on any of the three contract checks:
unknown `version:` in a rule, mismatch between the manifest's
declared schema-version set and the set observed across its rules,
engine runtime version outside the manifest's compatibility
expression. The exact failure mode — process exit, refuse-swap,
fallback to a prior manifest, hot-reload disable — is B0-7's call.
This study commits only that none of the three checks is allowed to
fail open.

**C7. The linter is the first line of defense.** The linter
validates against `rules/_schema/v<N>.schema.json` (the mirror), so
a rule that declares an unsupported `version:` value fails CI
before merge. The engine's load-time check is the second line of
defense for cases the linter cannot catch (out-of-band manifest
publication, manifest tampering, manifest published with a linter
version that has since been retired).

**C8. CODEOWNERS protect the schema source.** The CODEOWNERS file
(foundation doc
[`02-monorepo-topology.md`](../foundation/02-monorepo-topology.md))
must list `engine/internal/dsl/schema/` under the schema owner
group. The mirror at `rules/_schema/` is **also** owned by the
schema owner — not by domain teams — because edits to it are only
valid as the mechanical consequence of an engine-side schema edit.

**C9. Multi-version coexistence is supported but not free.** A
manifest declaring schema versions `{1, 2}` requires that both
schemas exist on disk in `rules/_schema/`, both schemas are
supported by the engine version receiving the manifest, and both
schemas are supported by the linter that built the manifest. The
non-trivial obligation is on the publisher (C5): the manifest's
declared schema-version set must equal the set actually observed
across the packaged YAMLs — fabrication or omission is a
publish-time failure, not a runtime one. The B1-7 decision
(compatibility window duration) sets how long the multi-version
state is allowed to persist.

**C10. The linter version is pinned in CI, not in the workspace.**
The pin's purpose is to keep the manifest's `linter_used` value
truthful for audit — without an unforgeable pin, the recorded value
can be falsified retroactively by re-tagging the linter release.
The pin mechanism W2-1 selects must therefore be unforgeable: a
re-tag of an existing linter release must not silently change which
artifact CI executes. Whether that is achieved by digest pinning,
content-addressable artifact stores, or another mechanism is W2-1's
call.

---

## Open Questions

The following items are not resolved by this study. Each is marked
with whether it is in scope for the current cycle.

- **OQ-1. JSON shape of the manifest contract block.** The three
  semantic commitments (schema-version set, engine-compatibility
  expression, linter reference for audit) are committed here; the
  JSON field names, structure, and encoding are
  **out-of-scope for current cycle** — they are resolved by B0-5
  (manifest publication semantics), which must respect the three
  semantic commitments above.

- **OQ-2. Semver range syntax for `engine_compatibility`.** Whether
  `engine_compatibility` uses standard semver range syntax (e.g.
  `>=2.0.0 <3.0.0`) or a custom syntax is **out-of-scope for current
  cycle** — it is part of B1-7 (compatibility window) and the
  manifest schema work in B0-5.

- **OQ-3. Compatibility window duration.** How long the engine
  supports a deprecated schema version is **out-of-scope for current
  cycle** — it is B1-7. This study commits only to *supporting* a
  window, not to its length.

- **OQ-4. Cryptographic posture of the manifest.** Whether
  `linter_used` and the schema-version declarations are signed (vs.
  checksummed) is **out-of-scope for current cycle** — it is B1-8
  (manifest cryptographic posture).

- **OQ-5. Final CODEOWNERS team names for the schema owner.** Which
  named team owns `engine/internal/dsl/schema/` and the mirror is
  **out-of-scope for current cycle** — it is B1-9 (CODEOWNERS
  finalization). C8 commits only to the path-level rule.

- **OQ-6. Linter version pin location and mechanism.** The exact CI
  file in which the linter version is pinned, and the specific
  unforgeability mechanism used (digest pinning,
  content-addressable artifact store, or other), is
  **out-of-scope for current cycle** — it is W2-1 (Wave 2 CI host
  choice). C10 commits only that the pin must be unforgeable.

No open question in this list is required to lock the declaration
model itself. All items above are operational refinements of the
locked shape.

---

## Promotion target

This study is promoted during Wave 3 to:

    docs/adr/0001-engine-rules-compatibility.md

The `0001` is provisional and assigned at promotion time. If the
Wave 3 ADR numbering convention orders by promotion date rather
than by B0 sequence, the number changes; the slug
(`engine-rules-compatibility`) does not.

The MADR ADR rewrites this study for an external-reviewer audience
(no `studies/` back-references per R8), folds in any updates from
B0-5 (manifest semantics) and B0-7 (loader failures) that intersect
with the contract block, and updates
[`03-boundary-contract.md`](../foundation/03-boundary-contract.md)
to reference the ADR.
