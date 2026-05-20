---
description: Adversarial review of a decision document.
argument-hint: <file>
---

<!-- path: .claude/commands/critique.md -->

You are running an adversarial critique of the markdown file at:

    $ARGUMENTS

Re-ground first:

1. Read `CLAUDE.md` §3 (R1–R8) and §4 (P1–P6).
2. Read `.claude/playbooks/acceptance-criteria.md` (AC-1–AC-10).
3. Read `.claude/playbooks/feedback-protocol.md` — the output format
   below mirrors it.

Then read the target file end-to-end.

Produce findings, each labeled with one of:

- **blocking** — must be fixed before the study can move to
  `resolved-study`. Typically a violation of R1, R2, R5, R6, or an
  unresolved Open Question with no out-of-scope marker.
- **important** — should be fixed; weakens the study but does not
  invalidate it.
- **minor** — wording, ordering, or polish.

For every finding, use the template:

    [severity] <R/P/AC label>: <section name> — <what to change>.

Example:

    [blocking] R5: "Considered Options" — option 2 cites a vendor by
    name. Rewrite in our own terms.

Cover at minimum: R1, R5, R6, P1, and every acceptance criterion. Do
**not** modify the target file. Print findings to stdout.

If you find no `blocking` findings, say so explicitly — the operator
needs that signal to advance the loop.
