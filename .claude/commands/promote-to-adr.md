---
description: Promote a resolved study to a MADR ADR under docs/adr/.
argument-hint: <study-file>
---

<!-- path: .claude/commands/promote-to-adr.md -->

You are promoting a resolved study to a formal ADR.

Argument (study file path): `$ARGUMENTS`

**Wave gate check — refuse if not satisfied:**

1. Read `studies/foundation/06-decision-log.md`.
2. Confirm every B0 row is at `resolved-study` or `resolved-adr`.
3. Confirm the Wave 2 consolidated decisions document exists in
   `studies/decisions/`.

If either check fails, stop. Print which gate is unmet and recommend
the operator return to the appropriate wave.

If both pass, proceed:

1. Read `$ARGUMENTS`.
2. Read `CLAUDE.md` §3 (R1–R8) — R8 is the load-bearing one:
   **rewrite for the new audience; do not link backwards into
   `studies/`**.
3. Determine the next available ADR number under `docs/adr/`
   (sequential, four-digit, e.g. `0001`, `0002`).
4. Write the ADR at `docs/adr/<NNNN>-<slug>.md` in MADR format:
   - HTML path-header comment (R6).
   - Title.
   - Status (proposed → accepted).
   - Context.
   - Decision.
   - Consequences.
5. The ADR is forward-only prose. Do not include
   "see `studies/...`" references.
6. Update the matching row in
   `studies/foundation/06-decision-log.md` to `resolved-adr` and add
   the ADR filename next to the existing study link.

Print: study path, new ADR path, decision-log row updated.
