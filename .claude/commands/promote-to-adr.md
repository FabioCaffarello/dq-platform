---
description: Promote a resolved study to a MADR ADR under docs/adr/.
argument-hint: <study-file>
---

<!-- path: .claude/commands/promote-to-adr.md -->

You are promoting a resolved study to a formal ADR.

Argument (study file path): `$ARGUMENTS`

**Post-Wave-3 recognition step:**

1. Read `$ARGUMENTS`.
2. Confirm the study carries `Status: resolved-study` in its
   Metadata block. If the study is still `in-progress` or `open`,
   stop and tell the operator to close the study via
   `.claude/playbooks/post-wave3-session-loop.md`
   (or `wave-1-session-loop.md` for legacy B0 studies) before
   promotion.
3. Confirm the study's `Promotion target` line names a concrete
   `docs/adr/<NNNN>-<slug>.md` filename. If the line is missing
   or carries `<NNNN>` as a placeholder rather than a number,
   stop and ask the operator for the intended number.
4. Confirm the promotion is happening under post-Wave-3 PR-flow
   per [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 — the
   current branch is not `main`, the
   [`session-governance`](../skills/session-governance/SKILL.md)
   skill is loaded, and the PR will be opened against `main`
   without merge.

> Audit note — the original gate this command carried was the
> Wave-1 gate: "every B0 row at `resolved-study` or `resolved-adr`
> + Wave-2 consolidated document exists". That gate was met on
> 2026-05-21 and has always passed since. It is dropped here per
> [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
> Clause 7 as a noise-only check.

**ADR-number reservation step:**

5. Determine the next available ADR number under `docs/adr/`
   (sequential, four-digit, e.g. `0001`, `0051`). Cross-reference
   against the study's `Promotion target` — if the study reserved
   a specific number, that is the proposed number.
6. **Post a message to the operator naming the proposed `<NNNN>`
   and pause for acknowledgement.** The reservation is
   operator-side (the operator tracks reserved numbers across
   parallel sessions); the command makes the step explicit so
   it cannot be skipped silently. The reservation closes the
   parallel-PR collision risk where two sessions both compute
   the same `<NNNN>` and collide at merge time. See ADR-0051
   Clause 7.
7. If the operator confirms a different `<NNNN>` than the
   computed next-available (because a parallel session already
   reserved it), use the operator-supplied number. The study's
   `Promotion target` reservation is descriptive, not binding;
   the operator is the authority at promotion time.

**Write the ADR:**

8. Read [`CLAUDE.md`](../../CLAUDE.md) §3 (R1–R8) — R8 is the
   load-bearing one: **rewrite for the new audience; do not link
   backwards into `studies/`**.
9. Read [`.claude/skills/adr-writing/SKILL.md`](../skills/adr-writing/SKILL.md)
   for the canonical 4-section structure (A1), the metadata block
   (A2), the new-contribution marker (A7), and the citation
   conventions (A6). Apply A7 for any decision in the study that
   carries a "new contribution proposed here, requires review"
   marker — the ADR must propagate the marker.
10. Write the ADR at `docs/adr/<NNNN>-<slug>.md` using MADR shape:
    - HTML path-header comment (R6).
    - Title — `# ADR-NNNN — Title`.
    - Status: `accepted`. Date: ISO-8601.
    - Context.
    - Decision (numbered subsections or named clauses; pick one
      form per ADR — do not mix).
    - Consequences (plain numbering).
    - Notes (optional; omit if empty).
11. The ADR is forward-only prose. Do not include
    "see `studies/...`" references. Cite prior ADRs, B-rows, and
    foundation documents (in `studies/foundation/`); never back-link
    to `studies/decisions/` or `studies/critiques/`.

**Close the loop:**

12. Update the matching row in
    `studies/foundation/06-decision-log.md` to `resolved-adr` and
    add the ADR filename next to the existing study link.
13. Add a new "Last updated" entry at the top of the decision-log
    Metadata section summarizing what landed (the prior "Last
    updated" entry is demoted to "Earlier update"). Match the
    shape of the existing entries.

Print: study path, new ADR path, decision-log row updated.

**Stop.** The PR is opened by [`open-pr`](./open-pr.md) (or by
the operator running `gh pr create --base main`); the agent does
not merge.
