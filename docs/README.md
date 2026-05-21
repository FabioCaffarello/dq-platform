<!-- path: docs/README.md -->

# `docs/` — Documentation Workspace

The docs workspace holds cross-workspace documentation that
is part of the published repository product (per
`CLAUDE.md` R8):

- **[`adr/`](adr/)** — Architecture Decision Records
  (MADR-aligned). ADRs `0001–0013` cover Wave 1, Wave 2,
  and the Wave 3 sequencing.
- Glossary, governance, contribution guide, runbook seeds
  (lands in Phase 8 of Wave 3).

## Current state (Phase 2)

This directory holds the ADRs promoted in Wave 3 Phase 1
([Sessions A, B, C](../studies/foundation/06-decision-log.md)):

- `adr/0001–0007` — Wave 1 commitments (B0).
- `adr/0008–0012` — Wave 2 commitments (W2).
- `adr/0013` — the Wave 3 phase-sequencing ADR.

Future content (glossary, governance, runbooks) lands in
Phase 8.

## Reading conventions

- All technical documents in this workspace are in
  **English** per
  [ADR-0011](adr/0011-documentation-language.md).
- ADRs are forward-only: they do not link backwards into
  `studies/` (per `CLAUDE.md` R8). The studies that
  originated each ADR remain in
  [`../studies/decisions/`](../studies/decisions/) for
  historical reasoning.
