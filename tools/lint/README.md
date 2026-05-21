<!-- path: tools/lint/README.md -->

# `tools/lint/` — Rules Linter

The linter validates rule YAMLs against the schema mirror
at [`rules/_schema/`](../../rules/) and enforces the
byte-equality contract between the rules schema mirror and
the engine's canonical schema source under
[`engine/internal/dsl/schema/`](../../engine/) per
[ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md).

This is a Go module (`dq-platform/tools/lint`) declared in
the top-level [`go.work`](../../go.work) per
[B1-10's resolution](../../studies/decisions/2026-05-21-b1-10-workspace-tooling.md).

## Scope (Wave 3)

- **Phase 3** lands the linter binary. Initial scope per
  ADR-0001:
  - reject rule YAMLs missing the top-level `version:`
    field;
  - validate each rule against the mirror schema at the
    declared version;
  - reject `entity` names containing the ASCII pipe
    character per
    [ADR-0002 input-safety](../../docs/adr/0002-run-identity-and-idempotency.md);
  - run as a CI step that performs the byte-equality
    check between the engine schema source and the rules
    mirror (the contract enforced by
    [`schema-mirror.yml`](../../.github/workflows/schema-mirror.yml)).
- **Phase 5** extends the linter to reject entities
  without an `_owners.yaml` entry per
  [ADR-0006](../../docs/adr/0006-alert-routing-contract.md).

## Current state (Phase 2)

This directory holds only a `go.mod` declaring the module
identity. The linter binary lands in Phase 3.
