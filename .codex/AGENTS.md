<!-- path: .codex/AGENTS.md -->
<!-- audience: the Codex CLI agent and equivalents that look under .codex/ -->
<!-- status: pointer file maintained by /sync-agents; do not edit by hand -->

# Codex CLI — Agent Pointer

The canonical operating contract for AI coding agents in this
repository lives in [`/CLAUDE.md`](../CLAUDE.md). The contract
applies to every AI agent in this repository, not only to
Claude Code — if you are reading this file from inside the
Codex CLI, you are still bound by the rules in `CLAUDE.md`.

Read, in this order, before producing anything:

1. [`/CLAUDE.md`](../CLAUDE.md) — primary contract.
2. Every playbook in
   [`/.claude/playbooks/`](../.claude/playbooks/) — operational
   protocol. Waves 1–3 are closed; the current operational loop is
   `post-wave3-session-loop.md` (B2 follow-ups, B3 evolutionary
   entries, ADR amendments, ADR promotions). The Wave 1 and Wave 3
   loops remain as historical references.
3. [`/CONTRIBUTING.md`](../CONTRIBUTING.md) Flow 5 — upstream
   authority for PR-flow in the post-Wave-3 lane.
4. [`/studies/foundation/`](../studies/foundation/) — numbered
   foundation documents, read in order.
5. [`/studies/foundation/06-decision-log.md`](../studies/foundation/06-decision-log.md)
   — current state of every B-row and ADR, including Wave-S partial
   / full gate status and B3 lane activity.

This file introduces no new rules. It is a pointer maintained
by the `/sync-agents` skill; rule authorship flows from
`CLAUDE.md`, never the reverse.
