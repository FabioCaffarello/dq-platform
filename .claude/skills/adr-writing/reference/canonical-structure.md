<!-- path: .claude/skills/adr-writing/reference/canonical-structure.md -->

# ADR — canonical structure

The literal skeleton, derived from ADRs 0001–0047. Every section is
required except `Notes`, which is appended only if it carries content.

```markdown
<!-- path: docs/adr/NNNN-slug.md -->

# ADR-NNNN — Title

- **Status:** accepted
- **Date:** YYYY-MM-DD

[**Scope note (added YYYY-MM-DD):** appended here only when the ADR
commits a set-mode-scoped position and a parallel record-mode
position is addressed elsewhere — see A3 in the parent SKILL.md.]

---

## Context

Exposition of the problem, the prior commitments it sits inside, the
principles and rules it must honor, and the design tensions it
balances. This section establishes why a decision is needed. It is
the only section that may quote foundation documents and prior ADRs
densely; the other three sections refer back to Context.

---

## Decision

The committed position. May use:

- numbered subsections (`1.`, `2.`, …) — exemplar ADR-0001
- named clauses (`Clause 1`, `Clause 2`, …) — exemplar ADR-0042
- descriptive headings without numbers — exemplar ADR-0020

For amendments, include a "Why this does not reopen ADR-NNNN"
subsection — exemplar ADR-0046.

---

## Consequences

The decision's effects. Numbered. Covers:

- what becomes true once this ADR lands;
- which artifacts must change (manifest fields, schema fields,
  CI lanes, env config, etc.);
- which B-items / follow-up rows are registered;
- which prior ADRs gain a scope-note redemption (if any).

Consequences flow **forward**. Do not put cross-ADR amendments in
Consequences prose — that relationship lives in the Status line.

---

## Notes

Optional. Use for:

- deferred items with explicit out-of-scope markers
- Open Questions (`OQ-N`) reserved for a future cycle
- any reserved design space the ADR does not commit on

Omit this section entirely if it would be empty. Do not write
`Notes — none` or similar placeholder text.
```

---

## Section ordering is invariant

Across all 47 ADRs, no document reorders these four. If a section
would be empty, drop it (only `Notes` is optional); do not rearrange.

## Section-naming variants observed

- Decision subsections vary widely: `1./2./3.`, `Clause 1/2/3`, or
  pure descriptive headings. Pick one form per ADR; don't mix.
- Consequences subsections always use plain numbering (`1.`, `2.`, …)
  in the canonical exemplars.

## Anti-patterns checklist

If any of these is true, fix it before merge:

1. The `<!-- path: -->` HTML comment is missing on line 1.
2. The Status line is something other than `accepted` (or an
   amendment variant naming the amender by `ADR-NNNN`).
3. The Date is missing or not ISO-8601 (`YYYY-MM-DD`).
4. A scope-note is missing on an ADR that is set-mode-scoped and has
   a parallel record-mode counterpart elsewhere.
5. The four-section order has been reordered or a section renamed.
6. Cross-ADR amendments appear in Consequences prose (they belong in
   the Status line, with the amender ADR carrying its own full body).
7. The ADR back-links into `studies/decisions/` (R8 violation).
8. A "new contribution requiring review" marker is used for a
   decision that is incremental, not genuinely new ground.
