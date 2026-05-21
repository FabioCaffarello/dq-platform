<!-- path: docs/adr/0012-tag-conventions.md -->

# ADR-0012 — Per-Workspace Tag Conventions

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

The repository ships **per-workspace tags**, not a single
monorepo-wide version, because the workspaces evolve at
different rates and ship different artifacts (engine
container, rules manifest, linter binary, infrastructure
manifest). Each workspace's tag carries a workspace-specific
prefix.

Two upstream ADRs constrain the tag format:

- **ADR-0001** gates manifest publication on contract
  consistency (schema versions, engine compatibility). The
  manifest's `engine_compatibility` field is a semver range
  over `engine-v*` tag values; the linter pin recorded as
  audit information is a `tools-lint-v*` tag value.
- **ADR-0002** includes `ruleset_version` as one of the five
  inputs to the `execution_id = sha256_hex(...)` formula,
  pipe-separated with **no escaping**. Any pipe character
  in a `ruleset_version` value would break the hash
  invariant.

The platform must commit tag formats that are simultaneously
recognizable to humans, parseable by tooling, safe inputs to
the `execution_id` formula, and amenable to per-workspace
independent evolution.

---

## Decision

### 1. Per-workspace tag prefixes

- `engine-v<major>.<minor>.<patch>` — engine binary
  releases.
- `rules-v<major>.<minor>.<patch>` — rules snapshot
  releases (the value written into the manifest's
  `ruleset_version` field).
- `tools-lint-v<major>.<minor>.<patch>` — linter binary
  releases (initial scope; see Notes for the per-tool
  question).
- `deploy-v<major>.<minor>.<patch>` — infrastructure
  manifest releases.

`<major>`, `<minor>`, and `<patch>` are non-negative
integers. The leading prefix and the dots separating the
three numeric components are the **stable structural
shape** of every tag in the platform.

### 2. Pipe character is forbidden anywhere in any tag

No tag prefix or tag value may contain the ASCII pipe
character (`|`). This restates ADR-0002's no-escaping
invariant as a tag-convention rule: `ruleset_version` is
one of the five pipe-separated inputs to the `execution_id`
formula, and pipe characters in `ruleset_version` would
collide with the separator.

Hyphens (in the prefix, e.g., `tools-lint-`) and dots (in
the version components, e.g., `2.4.7`) are explicitly
permitted because they are pipe-free.

### 3. Pre-release semver suffixes are safe inputs to hashing

Pre-release suffixes such as `rules-v1.2.3-rc1` are
pipe-free and therefore safe inputs to the ADR-0002 hash.
Whether such a pre-release ruleset may be **published** as
`manifests/latest.json` (ADR-0005) is a separate question
not decided here.

### 4. Per-workspace tag prefix enforcement is a CI gate

CI must reject an `engine-v*` tag that points at a change
landing in `rules/`, or any other prefix/path mismatch.
The specific enforcement mechanism (a workflow that
validates the tag's reachable diff against the workspace
implied by its prefix) is a scaffolding detail.

---

## Consequences

1. **`engine-v<major>.<minor>.<patch>` directly feeds the
   `engine_compatibility` semver range in ADR-0005's
   manifest body.** The engine binary that loads a
   manifest must satisfy the manifest's declared range.

2. **`rules-v<major>.<minor>.<patch>` is the literal value
   written into the `ruleset_version` manifest field
   (ADR-0005) and the literal value hashed into
   `execution_id` per ADR-0002.** The tag and the
   manifest field are the same string; there is no
   transformation step between them.

3. **The pipe-character ban is non-negotiable.** Adding a
   sixth input to the `execution_id` formula in a future
   ADR requires defining an equivalent protection for that
   input. Adding a fifth tag-prefix variant requires that
   the new prefix also be pipe-free.

4. **Pre-release publication semantics are not committed
   here.** Whether `rules-v1.2.3-rc1` may write
   `manifests/latest.json` (becoming the live ruleset) or
   only a side channel is a manifest-publication question
   for a future ADR.

5. **Tag enforcement is a CI gate.** Workflow design and
   the specific check shape are scaffolding details, but
   the enforcement contract is committed: an `engine-v*`
   tag whose reachable diff includes changes outside
   `engine/` fails CI; same for the other three prefixes
   against their respective workspaces.

6. **`tools-lint-v*` is the release prefix for the linter
   binary specifically.** The `tools/` workspace may grow
   other tool binaries over time; whether they share the
   `tools-lint-v*` prefix or get their own (e.g.,
   `tools-X-v*`) is a follow-up sub-decision deferred
   until a second tool binary lands.

---

## Notes

- Whether `tools-lint-v*` covers the whole `tools/`
  directory or only the linter binary is revisited when a
  second tool binary lands; until then, the prefix is
  reserved for the linter.
- A schema-only release tag (introducing a `schema-v*`
  prefix to release the schema independently of the
  engine binary) is a deferred option. The current model
  keeps schema versioning tied to engine releases via
  ADR-0001's contract consistency.
- Pre-release publication rules are a follow-up to
  ADR-0005, not constrained here.
- Tag-prefix CI gate enforcement details (which workflow
  validates the tag-to-path mapping, how it is configured
  per prefix) are a scaffolding follow-up.
