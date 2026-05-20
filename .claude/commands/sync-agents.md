---
description: Propagate context across CLAUDE.md, AGENTS.md, and .codex/AGENTS.md.
---

<!-- path: .claude/commands/sync-agents.md -->

You are propagating context across the multi-agent contract files.

No arguments.

Read in this order:

1. `CLAUDE.md` — the primary contract.
2. `AGENTS.md` — the cross-agent entry point.
3. `.codex/AGENTS.md` if it exists (introduced by the Wave 2
   decision; may legitimately be absent).

Detect drift across the files in:

- The hard rules section (R1–R8).
- The platform principles section (P1–P6).
- The slash-command list.
- The waves narrative (Wave 1 / Wave 2 / Wave 3 framing).
- The path-header convention (R6).
- The required-reading list (foundation docs + playbooks).

For each piece of drift:

- `CLAUDE.md` wins as the source of truth.
- Update `AGENTS.md` (and `.codex/AGENTS.md` if present) in place to
  converge.
- Do not invent rules or principles that exist in neither file.

Print a summary listing each file touched and which sections were
updated. If no drift is found, say so explicitly.
