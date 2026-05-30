---
description: Open a PR from the current feature branch against main following the PR-flow checklist authoritative in CONTRIBUTING.md Flow 5.
---

<!-- path: .claude/commands/open-pr.md -->

You are opening a pull request following the PR-flow contract
authoritative in
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5. The
[`session-governance`](../skills/session-governance/SKILL.md)
skill is the in-session reading layer for the cross-cutting
discipline (G1–G6).

No arguments — the command reads the current branch state.

---

## Pre-flight

1. **Confirm the branch is not `main`.**
   Run `git branch --show-current`. If that flag is unavailable
   on an older Git, fall back to
   `git rev-parse --abbrev-ref HEAD`. Show the operator the
   output. If the result is `main`, **stop** — surface the
   error to the operator and ask for the dedicated branch slug
   per [`CONTRIBUTING.md`](../../CONTRIBUTING.md) §"Branch
   naming for post-Wave-3 sessions". Do not invent a slug.

2. **Show the staged commits.**
   Run `git log main..HEAD --oneline`. If the output is empty,
   `/open-pr` is premature — there are no commits to PR. Stop
   and ask the operator whether to wait for commits or to
   close the session.

3. **Show the working tree state.**
   Run `git status --short`. If untracked files are present
   that the operator may have intended to commit, surface them
   before proceeding. Do not silently leave them behind.

## Gates

4. **Run the local gates applicable to the surface.** Pick
   from the inventory enumerated by
   [`session-governance`](../skills/session-governance/SKILL.md)
   reference:

   - `make lint` — always, if the change touches any source
     file.
   - `make test-engine`, `make test-tools` — if engine or
     tools sources changed.
   - `make lint-rules` — if `rules/` changed.
   - `make validate-deploy` — if `deploy/` changed.
   - `make demo-p6` — for end-to-end smoke when CI parity
     matters.

   Surfaces that have no gates (e.g., `.claude/` and
   `docs/`-only changes) declare the gate run as **vacuous**
   in the PR body's test plan — do not invent gates that do
   not exist.

   If a gate fails, **stop**. Surface the failure to the
   operator. Do not pass `--no-verify` to bypass a failing
   pre-commit hook (G4). Do not skip a failing test
   "temporarily" — that is the cost of the discipline.

## Push and open

5. **Push the branch.**
   Run `git push -u origin <current-branch>`. The `-u` flag
   sets upstream tracking so future pushes are shorter.

6. **Open the PR.**
   Run `gh pr create --base main` with a title and a body. The
   title follows the conventional-commit form used elsewhere
   in this repository (`<type>(<scope>): <subject>`). The body
   follows this template:

   ```markdown
   ## Summary

   <2–4 bullets: what this PR commits, in the operator's voice>

   ## Citation map

   - [ADR-NNNN](docs/adr/...) — what this PR implements or honors.
   - B-row reference if applicable.
   - R / P / AC labels the PR is grounded in.

   ## Critique result

   | Round | Blocking | Important | Minor | Status |
   |---|---|---|---|---|
   | 1 | N | N | N | dispositioned in <commit-hash> |
   | 2 | N | N | N | dispositioned in <commit-hash> |

   (Skip the table for code slices where critique was
   internal-only; record the skip reason here instead.)

   ## Test plan

   - [x] Local gates that ran (with results).
   - [x] Manual verification steps performed.
   - [ ] Reviewer concurrence points (load-bearing checks the
     [H] reviewer is asked to confirm).

   ## Scope explicitly NOT in this PR

   - <follow-on items deferred per R4 / G6>
   ```

## Stop

7. **The agent stops.** The PR is opened; the merge belongs to
   the `[H]` reviewer in the GitHub UI after CI is green and
   review is approved. **Never call `gh pr merge`** (G3) under
   any condition — not when CI is green, not when the
   operator says "this looks good", not when a `/critique`
   round emits zero blocking.

   If the operator says "merge it", surface the merge intent
   for confirmation in chat; do not run `gh pr merge`. The
   deny entry `Bash(gh pr merge *)` in
   [`.claude/settings.json`](../settings.json)
   enforces this mechanically.

---

## Why this command exists

The PR-opening checklist is enforced by two layers in this
repository:

- [`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5 is the
  upstream documentary contract (per
  [ADR-0051](../../docs/adr/0051-claude-tooling-postwave3.md)
  Clause 2).
- [`session-governance`](../skills/session-governance/SKILL.md)
  is the in-session reading layer.

This command is the procedural surface that runs the checklist
end-to-end. Without it, every session re-derives the steps from
the playbook or the skill by hand; with it, the steps are one
slash command away. Per ADR-0051 Clause 5.

**This command reads from
[`CONTRIBUTING.md`](../../CONTRIBUTING.md) Flow 5; updates to
the PR-flow contract land in `CONTRIBUTING.md` first.**
