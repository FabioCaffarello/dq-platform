<!-- path: AGENTS.md -->
<!-- audience: any AI coding agent that reads AGENTS.md by convention (Codex CLI, other tools) -->
<!-- status: living document, kept in sync with CLAUDE.md via /sync-agents -->

# Agents Operating Contract — DQ Platform

`AGENTS.md` is the cross-agent convention file. Tools that look for it
(Codex CLI and others) should treat this file as the entry point.

For the **full** operating contract, including project context, hard
rules, principles, and slash commands, read:

- **[`CLAUDE.md`](./CLAUDE.md)** — primary contract, always current.

The rules in `CLAUDE.md` apply to every AI agent in this repository,
not only to Claude Code. If you are a different tool, you are still
bound by those rules.

---

## Quick summary for first-time agents

You are in a monorepo that is being bootstrapped from zero. The
project is a long-lived Data Quality platform organized as five
workspaces (`engine/`, `rules/`, `tools/`, `deploy/`, `docs/`). The
project is in **Wave 1 of 3**:

- **Wave 1 (now):** resolve seven `B0` decision-log items as
  documents in `studies/decisions/`. Do not write production code
  yet.
- **Wave 2:** lock platform decisions (Git host, multi-agent
  contract, Docker Compose scope, language policy, workspace tooling
  and tag conventions).
- **Wave 3:** scaffold every workspace.

Read, in this order, before producing anything:

1. `CLAUDE.md`
2. `studies/foundation/README.md`
3. Every numbered document under `studies/foundation/`, in order.

Then check `studies/foundation/06-decision-log.md` for the current
state of open decisions.

---

## What does not change between agents

- The hard rules R1–R8 in `CLAUDE.md`.
- The six platform principles P1–P6 in `CLAUDE.md`.
- The three-wave sequence.
- The path-header convention on every produced markdown file.
- English as the default for technical artifacts (provisional until
  Wave 2 confirms).
- The constraint that **no produced artifact references external
  systems, vendors, or prior art by name**.

If a different agent has tool-specific conventions (its own
file-format expectations, for example), those are accommodated
**inside** the rules above, not by overriding them.

---

## When agents disagree

If two agents produce conflicting outputs for the same decision, the
human resolves the conflict and a single authoritative document lives
in `studies/decisions/`. Do not merge agent outputs silently.
