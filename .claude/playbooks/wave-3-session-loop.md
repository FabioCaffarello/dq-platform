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
   reason, **or** be resolved before the PR is opened. If
   neither, the unit is not ready.

10. **Land via pull request.** Create a feature branch from
    `main` using the convention `wave-3/<phase>-<topic-slug>`
    (e.g., `wave-3/p4b-result-write`,
    `wave-3/protocol-pr-flow`). Stage all session artifacts —
    including the decision-log row update for the closing
    sub-phase and any B-row updates the session resolved — and
    commit on the branch with the `Co-Authored-By` trailer used
    by prior commits.

    Push the branch to `origin` and open a PR against `main` via
    `gh pr create`. The PR body lists:
    - the citation map (B0 / W2 / W3 / B1 commitments the
      session implements);
    - the critique result (blocking / important / minor
      counts; what was addressed);
    - a test plan (the local gates from AC-W3-7, the
      integration tests if any, and a note on which CI
      workflows from Phase 2 will run on the
      `pull_request` trigger).

    CI workflows (`lint.yml`, `test.yml`, `schema-mirror.yml`
    from Phase 2) run on the PR's `pull_request` trigger.
    Once CI passes and the [H] reviewer (project lead)
    approves, the PR is merged via the GitHub UI. The squash-
    vs-merge-commit-vs-rebase choice is a host-primitive
    follow-up of ADR-0008 (recorded in the decision log
    under W2-1 sub-decisions); until that lands, the
    default is whatever the project lead picks at merge time.

    **Direct-to-main commits are reserved for the historical
    Wave-1, Wave-2, and Wave-3-Phase-0…Phase-3 commits up to
    `ee0d56f` (the Phase-4 sub-phase split).** Everything from
    W3-P4a (the loader) onward lands via PR — W3-P4a is the
    first PR (`#1`) and the playbook update committing this
    rule lands via the second PR. There is no carve-out for
    protocol changes: protocol updates also follow PR-flow so
    the playbook can be reviewed the same way as the
    scaffolding it governs.

---

## When to abort the session (no PR opened)

Stop the loop **without opening a PR** (and without committing
on the feature branch) if any of the following appears:

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
  a clear deferral than to open a PR for a half-baked surface.

If a feature branch already carries a commit when one of these
appears, either rewrite the branch (interactive rebase) before
opening the PR or delete the branch and re-scope. The PR is
the contract surface for reviewers; it must not include
known-broken state when opened.

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
