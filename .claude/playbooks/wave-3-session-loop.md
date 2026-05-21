<!-- path: .claude/playbooks/wave-3-session-loop.md -->

# Wave 3 — Session Loop

The canonical 10-step loop for one Wave 3 scaffolding unit.
Mirrors `.claude/playbooks/wave-1-session-loop.md` in shape; the
operational moves are different because the outputs are real
code, real config, and real CI lanes — not documents.

Each step is either an **agent action** or a **human decision
point [H]**.

---

## The loop

1. **Open** a fresh Claude Code session at the repository root.
   Run `/clear` to drop any prior context.

2. **Ground.** Run `/check-decision-backlog`. Confirm that every
   B0 or W2 commitment the scaffolding unit will cite is at
   `resolved-study` or `resolved-adr`. If any upstream is `open`
   or `in-progress`, **stop** — work on the upstream first.

3. **[H] Choose one scaffolding unit** from
   `studies/decisions/2026-05-21-wave3-sequencing.md`. Stay
   inside **one phase, one workspace, one slice**. R4 ("one
   topic per session") still applies in Wave 3 and matters more
   here than in Wave 1, because incidental refactors of unrelated
   workspaces are much easier to write in code than in prose.

4. **Plan.** Enter plan mode. The plan must list:
   - every B0 / W2 commitment the scaffold implements, by exact
     label (e.g., `B0-3 CC1`, `W2-3 §3.3 row 4`);
   - every file to create or modify, with its full path;
   - every test or local check the scaffold will satisfy from
     `.claude/playbooks/wave-3-acceptance-criteria.md`;
   - the deferred items (TODOs, capability rows left for a later
     phase) with an explicit "out-of-scope for current cycle"
     reason for each.
   Do not implement until the plan is approved.

5. **[H] Approve the plan.** Treat this the way Wave 1 treated
   step 5: if the plan frames the wrong question or breaches the
   phase boundary, **ask the agent to re-scope** — do not edit
   the plan yourself, let the agent regenerate so the reasoning
   stays consistent.

6. **Implement.** During implementation:
   - Path header (R6) on every produced markdown file.
   - English-only identifiers and comments (R7).
   - No prior-art or sibling-team names anywhere in code or
     comments (R5). Commodity infrastructure (BigQuery, Pub/Sub,
     GCS, OIDC, etc.) is exempt — those are environment, not
     borrowed ideas.
   - Cite B0 / W2 commitment labels in code comments **only
     where the citation is load-bearing for a future reader**
     (P5 contract-driven evolution). Routine code does not need
     citations.

7. **Self-verify** against
   `.claude/playbooks/wave-3-acceptance-criteria.md`. Resolve
   every AC-W3 row to **pass**, **fail**, or **deferred with a
   marker** before requesting critique. Failing rows with no
   plan to resolve them block the commit.

8. **Critique.** Run `/critique` on the scaffolding unit (point
   it at the plan file or at a session-summary markdown if the
   scaffold spans many files). Read the findings. Address every
   `blocking` finding in the **original artifact** (the scaffold
   itself), not in the critique. **Maximum two critique-revise
   rounds.** After that, accept the unit as the best the current
   session can produce and let remaining doubts surface as
   explicit TODOs with deferral markers.

9. **[H] Read end-to-end.** Walk the diff. Every TODO / FIXME /
   `_TBD` marker must carry an "out-of-scope for current cycle"
   reason, **or** be resolved before the commit. If neither, the
   unit is not ready.

10. **Update the decision log if applicable.** If the
    scaffolding unit resolved a B1 row (e.g., scaffolding the
    config model resolves B1-4), update the row to point at the
    produced scaffold. Then **commit** per the project's commit
    style — see `git log` for precedent. Include the
    `Co-Authored-By` trailer used by prior decision and sync
    commits.

---

## When to abort the session (no commit)

Stop the loop **without committing** if any of the following
appears:

- **Out-of-scope creep.** The scaffold introduces production
  code outside the declared Wave 3 unit (e.g., a Phase-4 session
  starts modifying Phase-5 alerting). R4 violation — roll back,
  re-scope, restart.
- **R5 violation.** A produced file names a sibling-team or
  prior-art system by name as justification. Roll back, rewrite
  in our own terms.
- **Upstream not at `resolved-study` / `resolved-adr`.** The
  session tried to cite an open decision. Resolve the upstream
  first.
- **AC-W3 rows fail and cannot be resolved in two critique
  rounds.** Better to leave the unit partially scaffolded with
  a clear deferral than to commit a half-baked surface.

---

## Why the [H] decision points exist

Steps 3, 5, and 9 are the points where the human's judgment is
load-bearing. Letting the agent pick the next scaffolding unit,
approve its own plan, or self-approve a TODO list defeats the
purpose of the loop discipline. These three checks are how
Wave 3 stays honest — and they are the direct analog of the
[H] points in the Wave 1 loop.

---

## Pointers

- Acceptance criteria:
  `.claude/playbooks/wave-3-acceptance-criteria.md`.
- Feedback protocol (shared with Wave 1):
  `.claude/playbooks/feedback-protocol.md`.
- Sequencing — current phase and next unit:
  `studies/decisions/2026-05-21-wave3-sequencing.md`. Treat
  that study as the canonical ordering; do not pick a unit
  that violates its phase boundaries.
- Decision log — upstream status:
  `studies/foundation/06-decision-log.md`.
