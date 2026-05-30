<!-- path: .claude/skills/session-governance/reference/governance-checklist.md -->

# Session-governance — verbatim checklist and long-form rationale

The procedural checklist `/open-pr` runs, with the rationale for
each step. Keep this file in sync with
[`../SKILL.md`](../SKILL.md) G1–G6 and with
[`../../../commands/open-pr.md`](../../../commands/open-pr.md).

---

## Pre-flight (before the first commit of any session)

1. **Confirm the branch is not `main`.**
   ```
   git branch --show-current
   # fallback: git rev-parse --abbrev-ref HEAD
   ```
   Show the operator the output. If the result is `main`,
   **stop and create the dedicated branch** with the
   operator-provided slug:
   ```
   git switch -c <operator-provided-slug>
   ```
   Slug prefixes the operator commonly passes are listed in
   [`CONTRIBUTING.md`](../../../../CONTRIBUTING.md)
   §"Branch naming for post-Wave-3 sessions" and are themselves
   provisional per R5. The agent does not invent new prefixes.

2. **Confirm the upstream is what you expect.**
   ```
   git status --short
   git log main..HEAD --oneline
   ```
   Empty `git log main..HEAD` means you have no commits to PR
   yet — `/open-pr` is premature. Diverging history (commits on
   the branch the operator didn't ask for) means investigate
   before pushing.

## Commit (during the session)

3. **Stage specific files; do not stage broadly.**
   ```
   git add <specific paths>
   ```
   Avoid `git add -A` or `git add .` — those pick up
   environment-leak files (`.env`, generated artifacts,
   editor caches) the operator did not intend to commit.

4. **Never pass `--no-verify`.**
   Pre-commit hooks exist for a reason. If a hook fails,
   investigate the failure or stop — bypassing is not a fix.
   The deny entries in
   [`.claude/settings.json`](../../../settings.json)
   enforce this for `git commit` and `git push`.

5. **Write the commit message with the citation.**
   The body cites the ADR / B-row / foundation document that
   the commit implements or extends, when load-bearing.
   Routine commits (a one-line typo fix) do not need the
   citation. The Co-Authored-By trailer matches the prior
   commits in this repository.

## PR opening (at session close)

6. **Push the branch.**
   ```
   git push -u origin <branch>
   ```
   The `-u` flag sets upstream tracking so future pushes are
   shorter.

7. **Open the PR.**
   ```
   gh pr create --base main --title "<type>(<scope>): <subject>" --body "<body>"
   ```
   The PR body lists:
   - **Summary** — what the PR commits, in 2–4 bullet points.
   - **Citation map** — every ADR / B-row / R-rule the PR
     implements or honors.
   - **Critique result** — round counts, what was addressed,
     remaining doubts as Open Questions.
   - **Test plan** — local gates run, manual verification
     steps, reviewer concurrence points.
   - **Scope explicitly NOT in this PR** — what was
     considered and deferred (R4 / G6 follow-up items).

8. **Stop. Never call `gh pr merge`.**
   The merge belongs to the `[H]` reviewer in the GitHub UI
   after CI is green and review is approved. If the operator
   says "merge it", surface the merge intent for confirmation;
   do not run `gh pr merge`.

   The deny entry
   `Bash(gh pr merge *)` in
   [`.claude/settings.json`](../../../settings.json)
   enforces this mechanically.

---

## Rationale

### Why a dedicated branch before the first commit

A commit on `main` is hard to undo (rewriting public history is
a destructive operation that requires force-push to the remote).
The cheapest defense is to never write the commit in the first
place. Creating the branch before the first commit makes the
branch state explicit at the moment when the cost of redirection
is still zero.

### Why never `gh pr merge`

The agent and the reviewer share a session identity in
single-operator workflows. A merge issued by the agent inside
the same session that produced the PR is structurally
"author-merges-own-work" — it skips the reviewer concurrence
that the PR-flow exists to require. Even when the operator says
"this looks good", the merge belongs to a separate operator
action (the GitHub UI click) so the audit trail records the
review-and-merge as a distinct human decision, not an
agent-emitted action.

### Why never `--no-verify`

Pre-commit hooks catch issues the agent cannot see locally:
secrets in staged files, lint violations the agent did not
trigger, formatting drift, conflicts with other ongoing
sessions. Bypassing the hook silences the signal but does not
fix the underlying issue; the issue surfaces later in CI or in
review, where it costs more to address. The harness denies the
two load-bearing call sites (`git commit --no-verify`,
`git push --no-verify`) mechanically; the in-session
confirmation gate is the safeguard for the cases the suffix
shape does not catch.

### Why one topic per session

A PR mixing two topics is harder to review than two PRs each
mixing zero. The review attention budget is per-PR, not
per-topic: an extra topic dilutes the attention each gets. R4
exists to keep the attention budget aligned with the topic
scope.

---

## Pointers

- Skill body: [`../SKILL.md`](../SKILL.md).
- Procedural command:
  [`../../../commands/open-pr.md`](../../../commands/open-pr.md).
- Upstream contract:
  [`CONTRIBUTING.md`](../../../../CONTRIBUTING.md) Flow 5.
- Settings enforcement:
  [`../../../settings.json`](../../../settings.json).
