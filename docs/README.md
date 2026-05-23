<!-- path: docs/README.md -->

# `docs/` — Documentation Workspace

The docs workspace holds cross-workspace documentation that
is part of the published repository product (per
`CLAUDE.md` R8):

- **[`adr/`](adr/)** — Architecture Decision Records
  (MADR-aligned). ADRs `0001–0014` cover Wave 1, Wave 2,
  the Wave 3 sequencing, and the HTTP trigger handler
  contract.
- **[`glossary.md`](glossary.md)** — canonical terminology
  for terms with codebase-specific meaning (lands in
  W3-P8a).
- **[`governance.md`](governance.md)** — review model, the
  three review groups, contribution-time flows (lands in
  W3-P8b).
- Contribution guide, runbook seeds — land in W3-P8c /
  W3-P8d.

## Current state (Phase 8)

This directory holds:

- `adr/0001–0007` — Wave 1 commitments (B0).
- `adr/0008–0012` — Wave 2 commitments (W2).
- `adr/0013` — the Wave 3 phase-sequencing ADR.
- `adr/0014` — the HTTP trigger handler contract (W3-P4e).
- `glossary.md` — codebase-specific terminology (W3-P8a).
- `governance.md` — review model and contribution-time
  flows (W3-P8b).

## Reading conventions

- All technical documents in this workspace are in
  **English** per
  [ADR-0011](adr/0011-documentation-language.md).
- ADRs are forward-only: they do not link backwards into
  `studies/` (per `CLAUDE.md` R8). The studies that
  originated each ADR remain in
  [`../studies/decisions/`](../studies/decisions/) for
  historical reasoning.
