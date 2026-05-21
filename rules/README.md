<!-- path: rules/README.md -->

# `rules/` — Declarative Rule Specifications

The rules workspace holds declarative YAML rule
specifications by entity, owner metadata, and governance
workflow.

This workspace is **not a Go module**. It is a structured
collection of YAML files, validated in isolation by the
linter binary under
[`tools/lint/`](../tools/lint/) per
[ADR-0001](../docs/adr/0001-engine-rules-compatibility.md).

## Scope (Wave 3)

- **Phase 3** lands the schema mirror at
  `_schema/v<N>.schema.json` (byte-equal copy of the
  engine's canonical schema under
  [`engine/internal/dsl/schema/`](../engine/), enforced by
  the byte-equality CI gate per ADR-0001).
- **Phase 5** lands the `_owners.yaml` schema fragment per
  [ADR-0006](../docs/adr/0006-alert-routing-contract.md).
- **Phase 6** lands the first onboarded entity end-to-end.

## Current state (Phase 2)

This directory exists for the empty-layout commitment from
[ADR-0013](../docs/adr/0013-wave3-sequencing.md). No rules
content, no schema mirror, no `_owners.yaml` yet.
