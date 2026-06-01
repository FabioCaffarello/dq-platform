<!-- path: .claude/playbooks/post-wave3-session-loop.md -->

# Post-Wave-3 — Session Loop

The canonical 10-step loop for resolving one post-Wave-3 entry —
B2 follow-up, B3 evolutionary extension per
[ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md), ADR
amendment, or ADR promotion. Each step is either an **agent
action** or a **human decision point [H]**.

Mirrors
[`wave-1-session-loop.md`](./wave-1-session-loop.md) in shape with
two deltas:

- **Step 2 adds an [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
  §(a) eligibility check** when the entry is a B3-N candidate
  (committed by [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 4 as **new contribution proposed here, requires review** —
  no prior loop carries an eligibility-check step in its 10-step
  shape; pending reviewer concurrence the step stays here as
  provisional).
- **Step 10 lands via PR** following the contract authoritative in
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5; the
  [`session-governance`](../skills/session-governance/SKILL.md)
  skill is the in-session reading layer.

---

## The loop

1. **Open** a fresh Claude Code session in the repository root.
   Run `/clear` to drop any prior context.

2. **Ground.** Run `/check-decision-backlog` to see the current
   status of every B-row.

   **Additional grounding for B3-N entries:** confirm the
   proposed work clears the
   [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a)
   eligibility filter — all four conditions must hold (expands
   not rewrites; in-scope family; conforms to the ADR-0020 /
   0021 / 0022 / 0023 envelope; crosses the
   additive-maintenance threshold). If any condition fails, the
   entry is not B3 — re-triage as B2, amendment, or rejected
   per [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md)
   §(a). If the family fit relies on an expansive reading of
   "adjacent tooling" or a similar clause, surface the reading
   explicitly so the operator can ratify it later (per
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
   §"Operator-side responsibilities").

   **For B2 entries**, additionally confirm the originating
   wave's gate is still met.

3. **[H] Choose one entry.** Pick exactly one B-row whose
   dependencies are resolved. Stay inside one topic per session
   (R4). If no unblocked entry exists, stop — work on
   dependencies first.

4. **Draft / plan.** For a study, run the applicable resolution
   command (currently `/resolve-b0` reused; `/resolve-b2` and
   `/resolve-b3` are deferred per
   [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
   Notes OQ-1). For a code slice, enter plan mode and list
   every B-row / ADR commitment cited, every file to create or
   modify, every applicable acceptance criterion, and explicit
   out-of-scope deferrals.

5. **[H] Approve.** For a study draft, read end-to-end. For a
   code plan, confirm the scope matches the topic. If the
   framing is wrong, ask the agent to re-frame and rewrite —
   **do not edit the draft / plan yourself**; let the agent
   regenerate so the reasoning stays consistent.

6. **Critique.** Run `/critique` on the produced artifact.
   Read the findings.

   **Preserve the round** per
   [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md):
   capture the `/critique` stdout to
   `studies/critiques/<today>-<slug>-critique-<N>.md` and commit
   as an intermediate commit before the revision lands. The
   capture is operator-side (the agent emits stdout only).

   For code slices the preservation may be skipped if the
   operator agrees the critique is internal-only; the trade-off
   is that the audit trail loses the round. Record the skip
   reason in the close commit body when preservation is
   skipped.

7. **Iterate.** Ask the agent to revise the **original
   artifact** (not the critique) to address `blocking`
   findings. Commit the revision as its own commit. Re-run
   `/critique` if needed — **maximum two critique-revise
   rounds.** After that, accept the artifact as the best the
   current session can produce and let remaining doubts surface
   in Open Questions (for studies) or TODOs with explicit
   deferral markers (for code).

8. **[H] Check completeness.** Two gates must pass:

   - **Open Questions / TODOs** — every item explicitly marked
     "out-of-scope for current cycle" with a one-line reason
     (per `acceptance-criteria.md` AC-6 / AC-W3-6).
   - **Critique rounds bullet** — the artifact's Metadata
     block has a `Critique rounds:` bullet listing each
     round's disposition per
     [ADR-0048](../../docs/adr/0048-critique-rounds-preservation.md)
     §"Skip" grammar. Absence of the bullet is itself a
     `blocking` finding.

   If either gate fails, the artifact is not ready.

9. **Update the log.** Edit
   `studies/foundation/06-decision-log.md` — change the row's
   status to `resolved-study` (for studies; the close commit
   that follows takes it to `resolved-adr` if and only if this
   session also promotes; otherwise promotion is a separate
   session). For code slices closing an implementation
   deferral, update the relevant ADR's Consequence row or add
   a "Last updated" entry summarizing what shipped.

   **OQ Register hunk (one-line extension).** If the promoted
   ADR labels new OQs (any `OQ-N` / `OQ-G3.1`-style identifier
   in its §Notes / §"Open Questions" / §Consequences), add a
   row per OQ to the
   `06-decision-log.md` §"Open Questions Register" in the same
   PR — source ADR / OQ id / one-line description / named
   trigger condition / `open`. If the promoted ADR consumes an
   open OQ (B3-N or amendment), flip the consumed OQ's row to
   `resolved-adr` and link the consuming ADR in the description
   column. The register's lane committed by the
   classification D0 ratified at PR #126 is
   [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 6, which
   admits this maintenance hunk inside the same promotion PR
   without opening a B-row. See the register's §"Scope and
   conventions" for the description-sourcing rule and the
   reused status vocabulary.

10. **Open the PR.** This is the **close action** — the
    intermediate commits already landed in steps 6 / 7 / 9.

    The PR-flow contract is authoritative in
    [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5; this
    playbook does not re-state it. In summary:

    - confirm the branch via
      `git branch --show-current` (fallback
      `git rev-parse --abbrev-ref HEAD`);
    - run the gates applicable to the surface (`make lint`,
      `make test-engine`, `make test-tools`, `make lint-rules`,
      `make validate-deploy` if `deploy/` is touched);
    - run `/open-pr` (or `gh pr create --base main` directly)
      with the PR body template — citation map, critique
      result, test plan;
    - **stop**. The merge belongs to the `[H]` reviewer in the
      GitHub UI after CI is green and review is approved.

    The [`session-governance`](../skills/session-governance/SKILL.md)
    skill is the in-session reading layer for the rules above
    and triggers on the branch / commit / PR phrases the
    contributor or agent uses at each load-bearing moment.

---

## When to abort the session (no PR opened)

Stop the loop **without opening a PR** (and without committing
on the feature branch) if any of the following appears:

- **R1 violation.** The agent wrote production code in a
  documentation-only step (e.g., a study session that
  produced Go files). Roll back, re-prompt.
- **R4 violation.** The scaffold introduced changes outside
  the declared unit scope. Roll back, re-scope, restart.
- **R5 violation.** A produced file names a sibling-team or
  prior-art system by name as justification. Roll back,
  rewrite in our own terms.
- **R6 violation.** A produced markdown file is missing its
  path-header comment. Fix the artifact.
- **Eligibility check at step 2 failed for a B3-N entry.**
  Re-triage as B2 / amendment / rejected; do not silently
  proceed as if eligibility passed.
- **Upstream not at `resolved-study` / `resolved-adr`** for a
  citation the artifact depends on. Resolve the upstream
  first.
- **Acceptance criteria fail and cannot be resolved in two
  critique rounds.** Better to leave the entry partially
  resolved with explicit deferrals than to open a PR for a
  half-baked artifact.

If a feature branch already carries a commit when one of
these appears, either rewrite the branch (interactive
rebase before pushing) or delete the branch and re-scope. The
PR is the contract surface for reviewers; it must not include
known-broken state when opened.

---

## Why the [H] decision points exist

Steps 3, 5, and 8 are the points where the human's judgment
is load-bearing. Letting the agent pick the next entry,
approve its own draft / plan, or self-approve a TODO / OQ
list defeats the purpose of the loop discipline. These three
checks are how the post-Wave-3 lane stays honest — they are
the direct analog of the `[H]` points in the Wave-1 and
Wave-3 loops.

The eligibility check at step 2 is **not** an `[H]` point in
itself — it is an agent action — but the **eligibility
ratification** for borderline B3-N readings is an operator
responsibility per
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5
§"Operator-side responsibilities". When the eligibility
reading is borderline, the operator ratifies explicitly
rather than absorbing the reading into the `/critique`
output silently (the author-equals-reviewer circularity).

---

## Pointers

- Upstream PR-flow contract:
  [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5.
- Acceptance criteria (shared with Wave-1):
  [`acceptance-criteria.md`](./acceptance-criteria.md).
- Feedback protocol (shared with Wave-1 and Wave-3):
  [`feedback-protocol.md`](./feedback-protocol.md).
- Session-governance skill (in-session reading layer):
  [`../skills/session-governance/SKILL.md`](../skills/session-governance/SKILL.md).
- PR-opening command:
  [`../commands/open-pr.md`](../commands/open-pr.md).
- Decision log — upstream status:
  [`../../studies/foundation/06-decision-log.md`](../../studies/foundation/06-decision-log.md).
- Eligibility filter for B3-N entries:
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(a).
- ADR that committed this playbook:
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 4.
