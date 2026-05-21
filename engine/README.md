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

## Current state (Phase 2)

This directory holds only a `go.mod` declaring the module
identity. Real engine code lands in Phases 3–4.
