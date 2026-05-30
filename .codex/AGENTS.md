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

1. [`/CLAUDE.md`](../CLAUDE.md) — primary contract, including
   §6's session reading router (committed by
   [ADR-0052](../docs/adr/0052-session-reading-router.md)) that
   maps your session type to the minimal playbook subset.
2. The playbook subset declared by
   [`/CLAUDE.md`](../CLAUDE.md) §6.2 for your session type —
   six types: B2 follow-up, B3 entry, ADR amendment, ADR
   promotion, Flow 6 process edit, implementation slice
   landing under a closed B-row. The current operational loop
   is `post-wave3-session-loop.md`. Apply §6.3 (default-up) if
   the classification is borderline; apply §6.4
   (output-artifact tie-breaker) if two rows could apply; skim
   §6.5 (historical-skim set: `wave-1-session-loop.md`,
   `wave-3-session-loop.md`) only when explicitly reading
   prior sessions for shape-reference context. The playbook
   inventory is in §6.6.
3. [`/CONTRIBUTING.md`](../CONTRIBUTING.md) Flow 5 — upstream
   authority for PR-flow in the post-Wave-3 lane.
   (`CONTRIBUTING.md` as a whole sits in `CLAUDE.md` §6.1's
   always-on floor; Flow 6 is the operator-authorized
   direct-edit lane.)
4. [`/studies/foundation/`](../studies/foundation/) — numbered
   foundation documents, read in order.
5. [`/studies/foundation/06-decision-log.md`](../studies/foundation/06-decision-log.md)
   — current state of every B-row and ADR, including Wave-S partial
   / full gate status and B3 lane activity.

R1–R8 (CLAUDE.md §3) and P1–P6 (CLAUDE.md §4) are read by every
session unconditionally — the router (CLAUDE.md §6) governs only
the playbook layer and can never narrow rule or principle
reading.

This file introduces no new rules. It is a pointer maintained
by the `/sync-agents` skill; rule authorship flows from
`CLAUDE.md`, never the reverse.
