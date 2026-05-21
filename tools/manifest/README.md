<!-- path: tools/manifest/README.md -->

# `tools/manifest/` — Ruleset Manifest Publisher

`dq-manifest` publishes a DQ Platform ruleset manifest to
object storage per
[ADR-0005](../../docs/adr/0005-manifest-publication-semantics.md)
(manifest publication semantics).

This is a Go module (`dq-platform/tools/manifest`) declared in
the top-level [`go.work`](../../go.work) per
[B1-10's resolution](../../studies/decisions/2026-05-21-b1-10-workspace-tooling.md).

## What the publisher does

The publisher implements ADR-0005 §4's four-step sequence
verbatim:

1. **Pre-publish verification** (per
   [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md))
   - every rule's declared `version:` is in the engine's
     supported set;
   - the `schema_versions_present` field equals the observed
     set (structurally enforced by deriving both from the same
     walk);
   - every value in the observed set has a corresponding
     mirror file at `<schema-mirror>/v<N>.schema.json`.
2. **Write rule YAMLs** to `yamls/by-hash/sha256-<hex>.yaml`
   (immutable per ADR-0005 §2).
3. **Write the manifest body** to
   `manifests/by-hash/sha256-<hex>.json`.
4. **CAS-write the pointer** at `manifests/latest.json` with
   generation-conditional write per ADR-0005 §3.

Step ordering is load-bearing: a failed verification leaves no
`by-hash/` objects behind.

## CLI

```
dq-manifest publish \
  --rules ./rules \
  --schema-mirror ./rules/_schema \
  --bucket dq-local-manifests \
  --ruleset-version rules-v0.1.0 \
  --engine-compatibility ">=0.1.0, <1.0.0" \
  --linter-used tools-lint-v0.1.0 \
  --supported-schema-versions 1 \
  [--dry-run] \
  [--storage-emulator-host localhost:4443]
```

### Exit codes

| Code | Meaning |
|------|---------|
| `0`  | publish OK |
| `1`  | pre-publish verification failed (content problem; operator fixes rules) |
| `2`  | operational failure (bucket missing, network, missing rules dir, etc.) |
| `3`  | CAS precondition failed (pointer race lost; operator retries) |
| `64` | usage error (missing required flag, bad argv) |

Exit codes are part of the binary contract; operator wrapper
scripts and CI lanes pattern-match on them.

## Local emulator

The publisher works against the
[fake-gcs-server emulator](../../docker-compose.yml) in the
local Compose stack via the `--storage-emulator-host` flag or
the standard `STORAGE_EMULATOR_HOST` environment variable.

There is a known fidelity gap (per
[B1-11](../../studies/decisions/2026-05-21-b1-11-substrate-posture-amendment.md)):
`fake-gcs-server` accepts `ifGenerationMatch` query parameters
but does not enforce the precondition, so the CAS-race-loser
path cannot be exercised locally. Unit tests cover that branch
via an in-memory fake; integration tests cover the happy path
against the emulator.

## Idempotency

`by-hash/` objects are immutable per ADR-0005 §2. Re-publishing
the same ruleset content produces the same `yaml_hash` values
and (modulo `generated_at`) a fresh manifest hash; the
publisher tolerates `ErrAlreadyExists` on by-hash writes and
proceeds to the pointer CAS. The manifest hash itself is not
guaranteed stable across re-publishes because `generated_at`
advances each call — the durable identity is the by-hash path
plus the pointer's `manifest_hash` reference.

## Determinism

Rules are sorted by `entity` before manifest marshaling so the
rules-array bytes are stable for identical rule sets across
re-runs (within the same `generated_at`). Sort-by-entity is
this tool's chosen convention (ADR-0005 does not specify
ordering).
