<!-- path: .claude/skills/session-governance/SKILL.md -->
---
name: session-governance
description: Use when the contributor or agent is about to take a branch / commit / PR action — verbatim trigger phrases include "create a branch", "git switch", "git checkout -b", "first commit", "open a PR", "gh pr create", and "merge the PR". Encodes the cross-cutting session-governance discipline that every post-Wave-3 session needs: create the feature branch before the first commit; confirm with `git branch --show-current` (fallback `git rev-parse --abbrev-ref HEAD`); never commit on `main`; never call `gh pr merge` (reserved for the [H] reviewer); never pass `--no-verify` to `git commit` or `git push`; cite the ADR or B-row in every produced artifact; stay inside R4 (one topic per session). Apply at every session that produces commits, opens a PR, or modifies any tracked file in this repository.
---

# `session-governance`

The cross-cutting discipline every session needs at branch / commit
/ PR time. Built on top of [`CONTRIBUTING.md`](../../../CONTRIBUTING.md)
Flow 5 (the upstream contract), [`CLAUDE.md`](../../../CLAUDE.md) §3
(R4 / R5 / R6), [`CLAUDE.md`](../../../CLAUDE.md) §"Executing
actions with care", and
[`.claude/playbooks/wave-3-session-loop.md`](../../playbooks/wave-3-session-loop.md)
step 10 (the original Wave-3 PR-flow contract Flow 5 generalizes).

> Reference file:
> - `reference/governance-checklist.md` — long-form rationale and
>   the verbatim PR-flow checklist that `/open-pr` executes.

**This skill reads from
[`CONTRIBUTING.md`](../../../CONTRIBUTING.md) Flow 5; updates to
the PR-flow contract land in `CONTRIBUTING.md` first and this skill
follows.**

The trigger shape committed by this skill's frontmatter
(verbatim contributor phrases keyed to branch / commit / PR
moments) is **new contribution proposed here, requires review**
([ADR-0051](../../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 3) — pending verification that the phrases reliably load
the skill in practice across sessions.

---

## G1. Branch before the first commit

The agent never commits on `main`. Before the first commit of any
session that produces tracked changes, the agent creates and
switches to a dedicated branch using the slug provided by the
operator (provisional per
[`CONTRIBUTING.md`](../../../CONTRIBUTING.md) §"Branch naming for
post-Wave-3 sessions"; the agent does not invent slug prefixes):

```
git switch -c <operator-provided-slug>
```

If the agent finds itself on `main` when the operator asks for
work that will produce commits, its first action is to create
the branch — not to wait for the first edit.

Source: [`CONTRIBUTING.md`](../../../CONTRIBUTING.md) Flow 5 step
10; [`CLAUDE.md`](../../../CLAUDE.md) §"Executing actions with
care" (destructive operations need confirmation).

## G2. Confirm the branch

Before the first commit, run:

```
git branch --show-current
```

If that flag is unavailable on an older Git, fall back to:

```
git rev-parse --abbrev-ref HEAD
```

Show the output to the operator. Two reasons: it converts an
implicit assumption ("I think we're on the right branch") into
an explicit fact, and it catches the case where a branch switch
silently failed.

Source:
[`.claude/playbooks/wave-3-session-loop.md`](../../playbooks/wave-3-session-loop.md)
step 10; ADR-0051 Clause 3.

## G3. Never call `gh pr merge`

The agent opens the PR and **stops**. The merge belongs to the
`[H]` reviewer (the operator), after CI is green and review is
approved. The agent does not call `gh pr merge` under any
condition — not when CI is green, not when the operator says
"this looks good", not when a `/critique` round emits zero
blocking.

If the operator says "merge the PR", the agent treats that as a
trigger to surface the merge intent for confirmation, not as an
authorization to run `gh pr merge`. The merge is operator-side
in the GitHub UI.

Source:
[`.claude/playbooks/wave-3-session-loop.md`](../../playbooks/wave-3-session-loop.md)
step 10 ("the PR is merged via the GitHub UI"); ADR-0051 Clause 3,
Clause 6 (the deny block enforces this mechanically).

## G4. Never pass `--no-verify`

The agent never passes `--no-verify` to `git commit` or
`git push`. Pre-commit hooks exist to catch issues the agent
might not see; bypassing them is a destructive action per
[`CLAUDE.md`](../../../CLAUDE.md) §"Executing actions with
care". If a hook fails, the agent investigates the failure or
stops — it does not bypass the hook.

The deny entries
`Bash(git commit --no-verify*)` and
`Bash(git push --no-verify*)` in
[`.claude/settings.json`](../../settings.json) enforce
this for the load-bearing call sites
([ADR-0051](../../../docs/adr/0051-claude-tooling-postwave3.md)
Clause 6). Cases where `--no-verify` appears after other arguments
are **not** caught by those entries (the deny shape is pure-suffix
per the existing settings precedent); the in-session confirmation
gate is the safeguard.

Source: ADR-0051 Clause 6;
[`CLAUDE.md`](../../../CLAUDE.md) §"Executing actions with care".

## G5. Cite the ADR or B-row in every produced artifact

Every produced markdown file, every load-bearing comment in
produced code, every commit message body cites the ADR /
B-row / foundation document that the artifact implements or
extends. The citation form is either bare (`ADR-NNNN`),
linked (`[ADR-NNNN](../path)`), or section-precise
(`ADR-NNNN §Section`) per
[`adr-writing`](../adr-writing/SKILL.md) A6.

Citations are load-bearing only — routine code or routine
prose does not need a citation. The test:

> Would a future reader otherwise wonder *why* this artifact
> exists in its current shape?

If yes, cite. If no, skip.

Source: [`CLAUDE.md`](../../../CLAUDE.md) §3 R5 ("cite only this
repository, and own your patterns") and
[`.claude/playbooks/wave-3-acceptance-criteria.md`](../../playbooks/wave-3-acceptance-criteria.md)
AC-W3-3 (load-bearing citations only).

## G6. One topic per session (R4)

If during the session the agent identifies adjacent work — a
related cleanup, a follow-on improvement, a nearby refactor —
the agent **lists it for a future session**, never expands the
current one. The PR opened at session close stays scoped to
the topic that was approved in plan mode.

Source: [`CLAUDE.md`](../../../CLAUDE.md) §3 R4 ("one topic per
session");
[`.claude/playbooks/wave-3-session-loop.md`](../../playbooks/wave-3-session-loop.md)
step 3 and §"When to abort the session".

---

## Anti-patterns to avoid

Do **not** do any of these — each is a violation of one of G1–G6
above:

- Switch branches mid-session because the topic widened. R4
  violation; re-scope to the original topic instead.
- Commit on `main` "just for the path-header fix". G1 violation;
  the branch comes first, always.
- Call `gh pr merge --auto` "to save the operator a step". G3
  violation; the merge is operator-side.
- Use `git commit --amend --no-edit -n` to fix a hook failure
  silently. G4 violation; investigate the hook or stop.
- Add an "and while we're at it" commit that touches an
  unrelated file. R4 / G6 violation; list it for a future
  session.

---

## When this skill does NOT apply

- **Read-only sessions** that produce no commits and open no PR
  (research, exploration, /critique runs that emit stdout only).
  G1, G3, G4 are inapplicable; G5 still applies to any
  artifacts cited.
- **Sessions explicitly authorized to commit on `main`** (rare;
  the historical Wave-3-Phase-0 through Phase-3 commits ended
  at `ee0d56f`; no current authorization exists). The agent
  asks for explicit confirmation before commit-on-main, even
  when "the operator just said so" — G1 is invariant under
  ambiguity.
- **The `[H]` reviewer's own merge.** When the operator merges
  the PR via the GitHub UI, G3 is satisfied by the operator's
  action; the agent's role in that step is to wait.

---

## Pointers

- Upstream contract: [`CONTRIBUTING.md`](../../../CONTRIBUTING.md)
  Flow 5.
- PR-opening checklist (the procedural surface this skill's
  G3 / G4 enforce): [`.claude/commands/open-pr.md`](../../commands/open-pr.md).
- Settings enforcement layer:
  [`.claude/settings.json`](../../settings.json) (G3
  and G4 deny entries).
- Long-form rationale + verbatim checklist:
  `reference/governance-checklist.md`.
