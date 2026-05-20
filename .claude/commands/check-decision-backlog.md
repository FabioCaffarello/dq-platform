---
description: Report the current state of all platform decisions.
---

<!-- path: .claude/commands/check-decision-backlog.md -->

You are reporting the current state of the decision log.

No arguments.

Read `studies/foundation/06-decision-log.md`.

Produce a report with these sections:

**B0 (Wave 1 blocking):**
- count by status: `open` / `in-progress` / `resolved-study` /
  `resolved-adr`.
- list each B0 row with its current status and, if any, link to its
  study file.

**B1 (Important):**
- count by status only.

**B2 (Later):**
- count by status only.

**W2 (Wave 2 platform decisions):**
- whether the consolidated W2 document exists in `studies/decisions/`.

**Wave gates:**
- Wave 1 gate: `X of 7 B0 resolved` (PASS if 7/7, else BLOCK).
- Wave 2 gate: PASS if the consolidated W2 document exists, else
  BLOCK.

**Next recommended action:**
- Use the "Recommended Next Sequence" section of the log to pick the
  next B0 whose dependencies are resolved.

**Consistency check:**
- For each B0 row marked `in-progress` or `resolved-study`, verify
  the linked study file actually exists in `studies/decisions/`.
  Flag any mismatch.

Print the report and stop. Do not modify any files.
