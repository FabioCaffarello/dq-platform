<!-- path: .claude/skills/adr-writing/SKILL.md -->
---
name: adr-writing
description: Use when authoring or amending an ADR under docs/adr/. Encodes the canonical 4-section structure (Context / Decision / Consequences / Notes), metadata + scope-note conventions, R8 forward-only discipline, three citation forms (ADR-NNNN, [ADR-NNNN](path), ADR-NNNN §Section), the new-contribution-requiring-review marker, the status-amendment idiom for layered decisions, and the provisional ADR numbering caveat. Apply when creating a new ADR, promoting a study to an ADR, or amending an existing decision.
---

# `adr-writing`

Patterns extracted from ADRs 0001–0047 in `docs/adr/`. Every rule here
traces to a real ADR; the canonical examples are cited inline. For the
literal skeleton and citation idioms, see the two reference files.

> Reference files:
> - `reference/canonical-structure.md` — the 4-section skeleton, the
>   eight anti-patterns to avoid, and the literal frontmatter form.
> - `reference/citation-conventions.md` — the three citation forms,
>   scope-note verbatim template, amendment idiom, new-contribution
>   marker.

---

## A1. Canonical 4-section structure

Every ADR uses exactly four top-level sections, in this order:

1. `## Context`
2. `## Decision`
3. `## Consequences`
4. `## Notes` (optional; omit if empty — never reorder, never substitute)

This is invariant across all 47 ADRs. Decision subsections may be
numbered (`1.`, `2.`, …) or named clauses (`Clause 1`, `Clause 2`),
but the four parents are fixed. Exemplar: `docs/adr/0001-engine-rules-compatibility.md`.

## A2. Metadata block

Every ADR file opens with this five-line prefix:

```markdown
<!-- path: docs/adr/NNNN-slug.md -->

# ADR-NNNN — Title

- **Status:** accepted
- **Date:** YYYY-MM-DD
```

The HTML path comment is required by `CLAUDE.md` R6. The status is
literally `accepted` (or an amendment variant — see A4). The date is
the date the ADR was accepted, ISO-8601. Exemplar: `docs/adr/0047-…md:1-6`.

## A3. Scope-note pattern (set-mode ADRs predating Wave-S)

If an ADR commits a position that is currently set-mode-scoped and a
parallel record-mode position is being addressed elsewhere, append a
scope note **between the metadata block and the `---` separator**, as
a single bolded paragraph:

```markdown
**Scope note (added YYYY-MM-DD):** This ADR currently applies to set-oriented capability (realized over BigQuery). Record-oriented capability is addressed separately in Wave-S (see ADR-0020 forthcoming).
```

Exemplar verbatim: `docs/adr/0002-run-identity-and-idempotency.md:8`.
Same pattern in ADRs 0003, 0004, 0006, 0007, 0010, 0014, 0017.

## A4. Status amendment idiom

Amendments are **standalone ADRs**, not footnotes inside the amended
ADR. The amended ADR records the relationship in its Status line:

- Single amender: `- **Status:** accepted (amends ADR-NNNN)`
  Exemplar: `docs/adr/0017-substrate-posture-amendment.md:5`.
- Layered amenders: see `docs/adr/0010-substrate-posture.md:5` — the
  Status line lists each amender with a parenthetical describing what
  was amended.

Never use status values other than `accepted` and the amendment
variants — no `draft`, no `superseded`, no `deferred`. Studies carry
status; published ADRs are all accepted.

## A5. R8 forward-only discipline

Published ADRs **do not link backwards into `studies/decisions/`**.
The study declares its promotion target in metadata; the ADR cites the
B-item name (e.g., `B0-S1`) and any prior ADRs by number, but never
back-links to the study file path. This is `CLAUDE.md` R8.

Promotion-target line lives in the study, not in the ADR — example:
`studies/decisions/2026-05-23-b0-s1-mode-as-primitive.md:17-19`.

## A6. Cross-citation conventions

Three forms, used distinctly:

1. **Bare narrative** — `ADR-NNNN` in prose where the citation is
   contextual, not load-bearing.
2. **Load-bearing link** — `[ADR-NNNN](./NNNN-slug.md)` when the
   reader is expected to navigate.
3. **Section precision** — `ADR-NNNN §Section` when citing a specific
   clause; the section name follows the heading verbatim.

Exemplars: `docs/adr/0024-…md:22,48` (bare); `0010-…md:5` (link with
amendment context); `0035-…md:109` (`§"Section"` quoted form).

## A7. New-contribution marker

When an ADR commits a position with no prior foundation-doc or ADR
backing, mark it in the Context section:

```markdown
No prior foundation document, charter clause, or ADR commits a position on [topic]. The [posture] this ADR commits is **new contribution requiring review** and is reviewed in this ADR.
```

Exemplar verbatim: `docs/adr/0044-external-artifact-references.md:34-38`.

Do not use this marker for incremental decisions — only for genuinely
new architectural ground. `CLAUDE.md` R5 requires the marker; AC-5
verifies it during critique.

## A8. Provisional numbering caveat

ADR numbers are not reserved in advance. When a study declares its
expected promotion target, attach the caveat from ADR-0020:

```markdown
- **Promotion target:** `docs/adr/NNNN-slug.md`
  (subject to the same numbering caveat ADR-0020 §"Per-item ADR
  numbering" carries — `NNNN` is descriptive of the planned sequence).
```

Source clause: `docs/adr/0020-wave-s-launch.md:172-175` —
> Per-item ADR numbering. B0-S1 through B0-S7 promote in order to
> `docs/adr/0021-…` through `docs/adr/0027-…` respectively, modulo
> shifts if an unrelated promotion lands between B0-S items. The
> expected sequence is descriptive, not reserved.

---

## Anti-patterns (consolidated)

Do **not**:

- cite a study from inside an ADR (R8 violation)
- pre-reserve ADR numbers in committed text
- reorder the four sections (Context / Decision / Consequences / Notes)
- skip the scope-note when an ADR is set-mode-scoped and record-mode is parallel
- use a Status value other than `accepted` (with optional `(amends ADR-NNNN)`)
- mark an obvious decision as "new contribution requiring review" (the marker is for genuinely new ground)
- put cross-ADR amendments in Consequences prose — they live in the Status line
- use footnotes or separate amendment documents inline; amendments are standalone ADRs
