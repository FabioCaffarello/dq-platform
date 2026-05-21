<!-- path: engine/README.md -->

# `engine/` — DQ Platform Runtime

The engine workspace owns the Go runtime that evaluates
data-quality rules against the configured substrate.

This workspace is a single Go module
(`dq-platform/engine`), part of the top-level Go workspace
declared in [`go.work`](../go.work) per
[B1-10's resolution](../studies/decisions/2026-05-21-b1-10-workspace-tooling.md).

## Scope (Wave 3)

Wave 3 lands the engine runtime across Phases 3–4:

- **Phase 3** — schema source of truth under
  `internal/dsl/schema/`. The schema is the canonical
  source mirrored byte-for-byte into
  [`rules/_schema/`](../rules/) per
  [ADR-0001](../docs/adr/0001-engine-rules-compatibility.md).
- **Phase 4** — loader (per
  [ADR-0007](../docs/adr/0007-loader-scheduler-retry-failure-semantics.md)),
  runner with `execution_id` computation (per
  [ADR-0002](../docs/adr/0002-run-identity-and-idempotency.md)),
  result write to `dq_executions` and `dq_check_results`
  (per [ADR-0003](../docs/adr/0003-result-write-model.md)),
  failure-scope mapping (per
  [ADR-0004](../docs/adr/0004-failure-scope.md)),
  orphan-run detection (per ADR-0007).

## Current state

- **Phase 2 (root infrastructure)** — module declaration only.
- **Phase 3 (schema layer)** — canonical schema source at
  `internal/dsl/schema/v1.schema.json`, byte-equal to the
  rules mirror.
- **W3-P4a (loader)** — `internal/loader/` package:
  - `Loader.Load(ctx)` — startup-mode manifest load per
    ADR-0007 CC1: reads `manifests/latest.json`, verifies
    the sha256 content hash of the referenced body, parses
    the manifest, runs the ADR-0001 contract checks
    (manifest_version, engine_compatibility,
    schema_versions_present), returns the parsed Manifest.
  - `Loader.Refresh(ctx, currentHash)` — refresh-mode reload
    per ADR-0007 CC9 with hash short-circuit (pointer-only
    read when the hash matches the caller's current).
  - `Store` interface with a GCS-backed implementation that
    works against production GCS and against the local
    fake-gcs-server emulator via `STORAGE_EMULATOR_HOST`.
  - Typed errors (`ErrObjectNotFound`, `ErrHashMismatch`,
    `ErrContractMismatch`) for the engine binary's
    structured-failure observability per ADR-0007 CC14.
  - Unit tests under `internal/loader/loader_test.go`;
    integration tests under `loader_integration_test.go`
    (build tag `integration`), runnable via
    `make test-engine-integration` after `make up`.

Future work (later sub-phases):

- **W3-P4b** — result-write layer (`dq_executions`,
  `dq_check_results`, `dq_executions_current`).
- **W3-P4c** — runner + failure-scope mapping; the engine
  binary entry point also lands here.
- **W3-P4d** — orphan-run detection.
