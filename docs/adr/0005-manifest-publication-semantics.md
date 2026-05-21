<!-- path: docs/adr/0005-manifest-publication-semantics.md -->

# ADR-0005 — Manifest Publication Semantics

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

The manifest is the artifact the engine consumes at runtime
to know which rules to evaluate, at which schema version, and
under which engine-compatibility constraint (ADR-0001). The
manifest is **derived**, not authored: CI builds it from
rule YAMLs, verifies the compatibility contract, and
publishes it to object storage. The engine reads it later.

The decoupling in time between publish and read makes
publication semantics safety-critical. Without a precise
contract for *what publishing means*, the system has several
silent failure modes:

- Two CI lanes publish concurrently and the later write
  overwrites the earlier with no failure surface.
- An engine reads a half-written manifest between publish
  steps and accepts inconsistent contract data.
- An operator rollback overwrites the live manifest with a
  prior one, losing the ability to re-rollback.
- A manifest published with a misvalidated rule set cannot
  be retroactively identified.

The publication primitive must close all four.

---

## Decision

### 1. Object-store layout

The publisher writes exclusively under these prefixes:

```
manifests/latest.json
manifests/by-hash/sha256-<hex>.json
yamls/by-hash/sha256-<hex>.yaml
```

The `sha256-` prefix on `by-hash/` paths encodes the hash
algorithm in the path itself, so an object's path identifies
its algorithm. A future migration to a successor algorithm
becomes a coexistence of differently-prefixed paths.

No other prefix is used by the publisher. Reading manifests
reads from these prefixes; listing prefixes for discovery is
a bug (manifests are reached via the pointer).

### 2. Manifests and YAMLs are immutable

Once a `manifests/by-hash/sha256-<hex>.json` or a
`yamls/by-hash/sha256-<hex>.yaml` exists, it is never modified
and never deleted by the publisher. Lifecycle deletion (if
any) is a separate, audited operator action governed by
retention policy declared in `deploy/`.

### 3. The pointer file is the single mutable object

`manifests/latest.json` is the single mutable control-plane
object. Every "publish a new manifest" and every "rollback"
is **exactly one generation-conditional write** (compare-and-
swap) to that pointer.

### 4. Verify-then-write-content-then-CAS-pointer

The publisher's sequence is:

1. Run the three pre-publish verifications from ADR-0001
   (every rule's declared `version:` is engine-supported;
   the manifest's declared schema-version set equals the
   observed set; every value in the set has a corresponding
   mirror under `rules/_schema/`).
2. Write the rule YAMLs to `yamls/by-hash/sha256-<hex>.yaml`.
3. Write the manifest body to
   `manifests/by-hash/sha256-<hex>.json`.
4. Issue the generation-conditional write to
   `manifests/latest.json`.

If verification fails, no `by-hash/` objects are written. If
two publishers race, both succeed at writing their content
hashes (idempotent if identical content, harmless if not
because both hashes are preserved) and **at most one
succeeds** at the pointer write. The loser receives a
precondition-failed error and surfaces it as a publication
failure.

### 5. Manifest body — committed field set

```jsonc
{
  "manifest_version": <integer>,
  "ruleset_version": "rules-v<major>.<minor>.<patch>",
  "schema_versions_present": [<integer>, ...],
  "engine_compatibility": "<semver-range>",
  "linter_used": "tools-lint-v<major>.<minor>.<patch>",
  "generated_at": "<rfc3339-timestamp>",
  "rules": [
    { "entity": "<string>", "yaml_path": "<string>", "yaml_hash": "<sha256-hex>" },
    ...
  ]
}
```

`schema_versions_present`, `engine_compatibility`, and
`linter_used` carry the three semantic commitments from
ADR-0001. `linter_used` is audit-only — the engine does not
read or verify it at load time.

### 6. Pointer file — committed field set

```jsonc
{
  "pointer_version": <integer>,
  "manifest_hash": "sha256:<hex>",
  "ruleset_version": "rules-v<major>.<minor>.<patch>",
  "published_at": "<rfc3339-timestamp>"
}
```

`ruleset_version` is duplicated from the manifest body so
the pointer alone identifies the live ruleset without
fetching the manifest. `published_at` is the pointer-write
timestamp; it differs from the referenced manifest's
`generated_at` if a prior manifest is re-pointed (rollback).

### 7. Hash algorithm

The hash algorithm for content addressing of manifests and
YAMLs is **sha256** throughout. Aligned with the identity
ADR (ADR-0002) for a single hash-algorithm posture across
the platform.

### 8. Local development uses the same publication primitive

The publisher uses the same code path against the local
object-store emulator as against the deployed object store.
No mode flag, no skipped verifications, no alternate
primitive in dev. The local emulator must therefore provide
generation-conditional writes — committed in the substrate-
posture ADR (ADR-0010).

### 9. Substrate-selection checkpoint

Any decision selecting or changing the platform's
object-storage substrate must verify the substrate provides
(a) single-object atomic writes and (b) generation-or-etag-
conditional writes. If a substrate without (b) is selected,
this ADR is reopened; the publication primitive may need a
lease-based replacement or an auxiliary coordinator outside
object storage.

---

## Consequences

1. **Concurrent publishers fail loudly.** No silent
   overwrite. The losing publisher surfaces a precondition-
   failed error.

2. **Rollback is a single pointer write.** An operator with
   rollback authority reverts to any prior manifest still
   within the lifecycle retention window by issuing one
   generation-conditional write referencing the chosen
   historical `manifest_hash`. No CI rerun, no intermediate
   state. A manifest whose `by-hash/` object has been
   purged is not rollback-eligible; pointing at a purged
   hash produces a dangling pointer that the engine fails
   closed on.

3. **Engine reads are two-step.** Pointer → manifest body,
   verified by hash. Caching, refresh cadence, short-circuit
   logic (skip the manifest read when the pointer hash is
   unchanged), and the failure response on hash mismatch are
   specified by ADR-0007.

4. **Hash verification at read time is load-bearing.** Any
   engine implementation that elides hash verification on
   the manifest object or on a referenced YAML breaks the
   integrity guarantee content addressing provides.

5. **Pre-publish verification cannot be skipped.** The
   sequence verify-then-write enforces this structurally;
   if verification fails, no `by-hash/` objects are written
   at all.

6. **The manifest body is one read away from the pointer.**
   The compatibility-contract fields
   (`schema_versions_present`, `engine_compatibility`,
   `linter_used`) live in the manifest body, not behind
   YAML fetches. The engine's contract checks run before
   any YAML is fetched.

7. **Orphaned hashes are recoverable, not corrupting.** A
   publisher that writes content-hash objects and then
   loses its pointer CAS race leaves orphan hashes. They are
   unreachable from any pointer and cost only their storage
   footprint until lifecycle policy purges them.

8. **Lifecycle policy is owned by the platform team.**
   Content-addressed objects accumulate. A declared
   retention policy on the `by-hash/` prefixes lives as
   configuration under `deploy/` and is owned by the
   platform-team CODEOWNERS group. Specific retention
   parameters are a follow-up.

9. **The pointer is the single mutable surface.** Every
   integrity property follows from this: rollback is one
   write; concurrent publishers fail loudly because the
   pointer CAS is single-target; the engine has exactly one
   thing to refresh.

10. **Local-prod parity is structural, not asserted.** The
    same publication primitive runs in both environments.
    "Works on my laptop, fails in prod" cannot arise from
    publication semantics; it can only arise from substrate
    behavior, and the substrate-selection checkpoint guards
    that.

---

## Notes

- The exact lifecycle retention duration, the deletion
  trigger (age, reference count, explicit operator action),
  and per-environment overrides are follow-up parameter
  decisions.
- The operator path for rollback (admin endpoint, CLI
  subcommand) is a scaffolding detail not foreclosed here.
- Cryptographic posture beyond integrity (signatures vs.
  checksums on the pointer) is a follow-up.
- A future migration to a successor hash algorithm
  coexists by adding a new path prefix (e.g.,
  `sha512-<hex>`), not by rewriting paths.
