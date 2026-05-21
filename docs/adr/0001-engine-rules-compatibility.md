<!-- path: docs/adr/0001-engine-rules-compatibility.md -->

# ADR-0001 — Engine ↔ Rules Compatibility Declaration

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

The DQ Platform lives in a monorepo. The `engine/` workspace owns
the runtime and the JSON Schema source of truth; the `rules/`
workspace owns declarative YAML rule specifications authored by
domain teams; the `tools/` workspace owns the linter binary. The
three workspaces have different change rates, different
reviewers, and different release artifacts (`engine-v*`,
`rules-v*` manifests published to object storage,
`tools-lint-v*`).

The engine and the published ruleset manifest are **decoupled in
time** even within a single monorepo: the manifest is built and
published from CI; the engine reads it later at startup or
refresh. "We move together in git" does not extend to runtime,
so the boundary between engine and rules is a real interface
that must be declared and verified independently in code, in
the linter, and at the manifest.

Without a declared compatibility contract, the boundary
degrades silently: rules become unreviewable in isolation,
manifests can be loaded by an engine that does not support
their schema, and migrations from schema `vN` to `vN+1` cannot
hold both versions during a transition window.

---

## Decision

The compatibility contract is **declared three times,
independently**, and verified by **three independent checks**.

### 1. Per-rule version declaration

Every rule YAML carries a mandatory top-level `version:` field
identifying the schema version it was written against.

```yaml
version: 1
entity: customer
checks:
  - ...
```

The linter rejects any rule without `version:`. The engine
refuses to index any rule without it at load time. There is no
default.

### 2. Workspace schema mirror with byte-equality CI gate

The canonical JSON Schema lives at
`engine/internal/dsl/schema/v<N>.schema.json`. A byte-identical
copy lives at `rules/_schema/v<N>.schema.json` for every
supported `N`. The mirror is **never edited by hand** — schema
changes happen at the engine source and are propagated to the
mirror mechanically (a Makefile target invoking `cp` or
equivalent) in the same merge request.

A **mandatory CI gate on `main`** runs a mechanical byte
comparison (`cmp` or `diff`) between each pair on every merge
request and every push. Any divergence blocks the merge. The
gate cannot be downgraded to advisory.

The mirror exists so the rules workspace is lintable without
depending on engine internals at lint time. The duplication is
intentional and load-bearing; deduplicating it requires
reopening this ADR.

### 3. Published manifest contract block

Every published manifest carries three semantic commitments:

- **Schema versions present** — the set of schema versions
  actually declared by rule YAMLs inside this manifest. During
  steady state this is a singleton; during a migration window
  it carries both old and new versions.
- **Engine compatibility expression** — a semver range over
  engine releases that support every version in the set above.
- **Linter reference** — the linter release that validated the
  manifest, recorded for audit and reconstruction. **Not a
  runtime gate** — the engine does not read or verify this
  field at load time.

The exact JSON field names, structure, and encoding are
specified by ADR-0005 (manifest publication semantics).

### 4. Three independent verification gates

- **Linter (CI on `rules/` merges).** Rejects rules without
  `version:`; rejects rules whose declared `version:` is outside
  the linter's supported set; validates each rule against the
  mirror schema at the declared version.
- **Manifest publisher.** Verifies that every rule's declared
  `version:` is in the engine's currently-supported set; that
  the set declared in the manifest equals the set observed
  across the packaged YAMLs; that every value in the set has a
  corresponding mirror under `rules/_schema/`.
- **Engine load.** Verifies that every rule in the manifest has
  a `version:` whose value is in the engine's supported set;
  that the manifest's declared set matches the observed set;
  that the engine's running version satisfies the manifest's
  compatibility expression. Any failure aborts the load (no
  partial loading, no silent skipping — see ADR-0007).

### 5. Linter version pinning

The linter version is pinned in the CI configuration of the
chosen Git host, not in any rule YAML or in `rules/` metadata.
The pin must be **unforgeable** — a re-tag of an existing
linter release must not silently change which artifact CI
executes. The specific mechanism (digest pinning,
content-addressable artifact store, or other) is selected by
the Wave-3 root-infrastructure scaffolding (ADR-0008 names
GitHub as the chosen host; the mechanism within that host is a
follow-up sub-decision).

---

## Consequences

1. **Schema duplication is intentional.** Two copies of every
   schema version live on disk: the source at
   `engine/internal/dsl/schema/` and the mirror at
   `rules/_schema/`. The CI byte-equality gate makes the
   duplication safe. Any proposal to deduplicate by deleting the
   mirror must reopen this ADR.

2. **The byte-equality CI gate is load-bearing.** It cannot be
   weakened to a warning or a soft-fail. If the gate breaks for
   infrastructure reasons, merges to `main` pause until the
   gate is restored.

3. **The schema source of truth is unambiguous.** All schema
   edits happen at `engine/internal/dsl/schema/`. The mirror is
   updated mechanically in the same merge request. CODEOWNERS
   protect both the source and the mirror under the schema
   owner group; the mirror is not editable by domain teams.

4. **Every rule YAML must declare its schema version.** A
   missing `version:` is a hard error at the earliest possible
   point (the linter, before merge). No default.

5. **Manifest publication is gated on contract consistency.**
   No manifest may be published until the publisher has
   verified (a) every rule's `version:` is engine-supported,
   (b) the declared schema-version set equals the observed set,
   (c) every value has a corresponding mirror file.

6. **Engine load fails closed on contract mismatch.** Unknown
   `version:` in a rule, set mismatch, or out-of-range engine
   compatibility — any of these aborts the load. The exact
   failure mode (process exit, refuse-swap, fallback) is
   specified by ADR-0007.

7. **The linter is the first line of defense; the engine is the
   second.** Each can fail closed independently — there is no
   single point of trust across the boundary.

8. **Multi-version coexistence is supported.** A manifest may
   carry rules at multiple schema versions simultaneously,
   provided every version is engine-supported and
   linter-supported. The duration of the multi-version window
   is set by a follow-up decision and is not constrained by
   this ADR.

9. **The linter version pin is unforgeable.** This keeps the
   manifest's recorded linter reference truthful for audit.
   Without unforgeability, a re-tag could falsify which linter
   actually validated a given manifest.

---

## Notes

The compatibility window duration (how long the engine
continues to support a deprecated schema version after its
successor ships) is a follow-up decision and is not constrained
by this ADR. The cryptographic posture of the manifest
(checksums vs. signatures) is also a follow-up. The specific
linter-pinning mechanism within the chosen Git host is a
Wave-3 sub-decision.
