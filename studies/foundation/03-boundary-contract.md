<!-- path: studies/foundation/03-boundary-contract.md -->

# 03 — Boundary Contract Between `engine/` and `rules/`

## Metadata

- Purpose: define the logical contract between the engine and the
  rules workspaces, so that even though both live in a single
  repository, the boundary between them stays clean, versioned, and
  enforceable.
- Audience: platform engineers, rule authors, reviewers, CI
  maintainers.
- Status: draft (subject to refinement when the corresponding B0
  decision is resolved in `studies/decisions/`).
- Last updated: 2026-05-20
- Promotion target: `docs/contracts/boundary.md` and an ADR during
  Wave 3.

---

## Why a Boundary Contract Exists in a Monorepo

A common misunderstanding about monorepos is that they eliminate the
need for contracts between components, because everything moves
together. In this project that conclusion is false.

The engine and the rules have **different change rates, different
reviewers, different risk profiles, and different consumers** even
though they live side by side. Without a contract:

- the engine starts to "know" the shape of specific rules, eroding
  the closed-DSL principle;
- the rules start to depend on undocumented engine behaviors,
  creating invisible coupling;
- compatibility becomes implicit, which means it only surfaces when
  it breaks;
- review pressure on schema changes weakens, because "we can fix the
  rules in the same MR" becomes the easy answer.

The boundary contract makes the interface **explicit** even when
physical separation does not enforce it.

## What the Contract Covers

The contract has four surfaces:

1. **The DSL schema** — what is a valid rule.
2. **The linter** — how rules are validated before they reach the
   engine.
3. **The manifest** — how the engine knows which rules are active.
4. **Compatibility** — how the contract evolves over time without
   surprising either side.

Each surface has a versioning model, a publishing mechanism, and a
consumption mechanism.

---

## Surface 1 — The DSL Schema

### What it is

A JSON Schema document (Draft 2020-12) that defines every valid shape
of a rule YAML. It is the formal contract for what the engine accepts
and what the rules workspace must produce.

### Where it lives

```
engine/internal/dsl/schema/v1.schema.json     # source of truth
rules/_schema/v1.schema.json                  # mirror, kept in sync by CI
```

The schema **source of truth** lives in `engine/`. The mirror in
`rules/` exists so that the rules workspace can be validated without
depending on engine internals at lint time.

### How it is kept in sync

CI enforces that the two copies are byte-identical. A change to one
without the other fails CI on `main`. The synchronization mechanism is
intentionally mechanical (a copy command in CI) — not a complex
extraction tool, because the simpler mechanism is harder to break.

### How it is versioned

The schema declares its own version inside the document, as a literal
constant required at the top of every rule YAML:

```yaml
version: 1
```

Schema versions advance only when a **breaking change** is needed.
Additive, non-breaking changes are made under the same version
number.

A breaking change is one of:

- a field is removed or renamed;
- a field's type changes incompatibly;
- a constraint becomes stricter (a previously-valid rule becomes
  invalid);
- a default changes in a way that alters runtime behavior;
- an enum value is removed.

A non-breaking change is one of:

- a new optional field is added;
- a new enum value is added (only if existing consumers tolerate
  unknown values);
- a constraint is relaxed (a previously-invalid rule becomes valid);
- documentation strings are updated.

### How new versions are introduced

When `v2` is introduced:

- `engine/internal/dsl/schema/v2.schema.json` is added alongside `v1`;
- the engine supports both schemas simultaneously for a documented
  compatibility window;
- existing rules continue to declare `version: 1` until they migrate;
- a migration guide is published describing what changed and how to
  rewrite affected rules;
- after the compatibility window, support for `v1` is removed in a
  major engine release (not before).

The exact length of the compatibility window is a Wave 1 decision
(see [`06-decision-log.md`](./06-decision-log.md)).

---

## Surface 2 — The Linter

### What it is

A CLI binary that takes a rule YAML (or a directory of them) and:

1. validates against the JSON Schema;
2. applies project-specific lint rules that go beyond the schema
   (naming conventions, partition discipline, baseline sanity);
3. optionally generates a dry-run SQL representation of each check
   (without executing it);
4. exits non-zero on any failure with structured error output.

### Where it lives

```
tools/lint/                # the linter source
```

### How it is versioned

The linter is released independently with its own tag prefix:
`tools-lint-v<major>.<minor>.<patch>`.

The linter declares which schema versions it supports in its `--help`
and in its release notes. A linter that supports schema `v1` and `v2`
is more useful during transition windows than one tied to a single
version.

### How it is consumed

- **Local development:** developers run the linter via `make
  lint-rules` from the repository root, which invokes the linter
  binary from `tools/lint/`.
- **CI:** the rules pipeline runs the linter on every YAML on every
  merge request.
- **Engine startup:** the engine **also** revalidates rules at load
  time, against the same schema. The linter and the engine share the
  schema artifact; they do not share validation code paths beyond the
  shared schema. This is intentional — defense in depth.

### Why the linter and the engine validate independently

Because the rules and the engine ship through different release
artifacts and at different cadences. The linter catches problems in
CI before the engine ever sees them. The engine catches problems that
slipped through (out-of-band edits, race conditions in publishing,
manifest corruption). Both layers are cheap; both layers protect
trust.

---

## Surface 3 — The Manifest

### What it is

A small JSON document that declares the **active ruleset** at any
given moment. It is the only object the engine reads when deciding
which rules to load.

```jsonc
{
  "manifest_version": 1,
  "ruleset_version": "rules-v2.4.7",
  "schema_version": 1,
  "engine_compatibility": ">=2.0.0 <3.0.0",
  "generated_at": "2026-05-20T14:32:11Z",
  "rules": [
    {
      "entity": "customer",
      "path": "current/entities/customer.yaml",
      "checksum": "sha256:..."
    },
    {
      "entity": "transactions",
      "path": "current/entities/transactions.yaml",
      "checksum": "sha256:..."
    }
  ]
}
```

### Where it lives

The **source manifest** is generated by CI and lives in object
storage (GCS), not in the repository:

```
gs://<bucket>/rules/manifests/latest.json   # currently active manifest
gs://<bucket>/rules/manifests/history/      # every prior manifest
gs://<bucket>/rules/current/entities/       # YAMLs referenced by latest
gs://<bucket>/rules/history/                # YAMLs referenced by prior
```

The repository contains the **source YAMLs**. The manifest is a
**derived artifact** published by CI on every `rules-v*` tag.

### Why a manifest exists

Without a manifest, the engine would have to discover rules by
listing a GCS prefix and reading every file. That has three problems:

- **Atomicity:** mid-publish state can be observed by the engine.
- **Reproducibility:** "what rules ran at 14:30 UTC" cannot be
  answered.
- **Performance:** a list operation per startup, plus N reads, is
  wasteful.

A manifest solves all three: it is one read, it declares exactly which
YAMLs are active, and it can be versioned and audited.

### How it is published atomically

Manifest publication follows a **write-new-then-swap** pattern:

1. CI generates a new manifest with a unique timestamped name.
2. The new manifest is uploaded to GCS at a temporary path.
3. After upload completes and is verified, the manifest is **copied**
   to `manifests/latest.json`. GCS provides atomic single-object
   writes; the engine never observes a partial swap.
4. The old `latest.json` is moved into `history/` with its timestamp.

The engine reads `latest.json` exactly once per run startup. Between
startups it can be re-fetched on a configured interval. The engine
**never lists prefixes** to discover rules.

The exact mechanism (versioned objects, generation conditions, lease
semantics) is a Wave 1 B0 decision.

### How the engine consumes it

On startup or refresh:

1. Read `latest.json`.
2. Verify the engine's runtime version satisfies
   `engine_compatibility`.
3. Verify `schema_version` is supported by the engine.
4. For each rule, fetch the referenced YAML and verify the checksum.
5. Parse and validate every rule against the schema.
6. Index rules by entity, failing fast on duplicates.

If any step fails, the engine refuses to start with that manifest and
falls back to the previously-loaded manifest if one is available in
memory, or fails the startup if not. Manifest load failure **never**
results in silent partial loading.

---

## Surface 4 — Compatibility

### The compatibility promise

The engine commits to:

- supporting each schema version for a documented window after its
  successor is introduced;
- accepting any manifest whose `engine_compatibility` range includes
  the running engine version;
- never silently changing the semantics of an existing rule type
  within a schema version;
- documenting every release that affects the boundary contract in
  its release notes.

The rules workspace commits to:

- declaring a single schema version per rule (`version: 1`);
- updating rules when CI signals a schema-version migration is
  available;
- not depending on undocumented engine behaviors.

### The compatibility window

The default window for supporting a deprecated schema version is a
**fixed period after the successor is released** — exact duration
under discussion in Wave 1. The window exists to give domain teams
time to migrate without panic, and to give the platform team
confidence that the next version is stable before retiring the old
one.

### Breaking-change protocol

When a breaking schema change is required:

1. The change is proposed as an ADR and reviewed by the platform
   team and the schema owner.
2. If accepted, a new schema version is introduced as described in
   Surface 1.
3. The engine release that adds the new version is announced with a
   migration guide.
4. CI flags rules still on the old version with a non-blocking warning
   for the duration of the compatibility window.
5. Near the end of the window, the warning becomes blocking.
6. After the window closes, support for the old version is removed in
   the next major engine release.

### Non-breaking-change protocol

Most schema evolution is additive and does not break existing rules.
Adding an optional field, an enum value (where consumers tolerate
unknown values), or a relaxed constraint:

1. ships as an ADR if the field affects behavior in non-trivial ways;
2. ships in a minor engine release;
3. is documented in the schema's inline `description` and in the
   release notes;
4. does not require any change to existing rules.

---

## Open Topics

Tracked in [`06-decision-log.md`](./06-decision-log.md):

- exact duration of the schema compatibility window;
- exact format of `engine_compatibility` (semver range syntax, custom
  syntax, etc.);
- whether the manifest carries a cryptographic signature beyond
  checksums;
- whether the engine caches loaded rulesets across restarts or always
  re-fetches;
- how the manifest interacts with multi-environment isolation (one
  manifest per environment, or one manifest with environment-aware
  routing).

These do not block the contract's shape; they refine its operational
details.

## Closing Position

The boundary contract is the single most important interface in the
project. It is what allows the engine and the rules to evolve
independently while remaining safely interoperable, even though both
live in the same repository.

Every degradation of the project over time will trace back to a
weakening of one of the four surfaces above — usually through
expedient shortcuts that "just this once" bypass the contract.

The defense is documentation, CI enforcement, and review pressure —
not trust.
