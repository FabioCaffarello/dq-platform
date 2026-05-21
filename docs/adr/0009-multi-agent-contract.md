<!-- path: docs/adr/0009-multi-agent-contract.md -->

# ADR-0009 — Multi-Agent Contract

- **Status:** accepted
- **Date:** 2026-05-21

---

## Context

Multiple AI coding agents (Claude Code, the Codex CLI, and
future agents that read `AGENTS.md` by convention) operate
against this repository. Without a single source of truth for
the rules they all follow, the three agent-facing surfaces
(`CLAUDE.md`, `AGENTS.md`, and `.codex/AGENTS.md`) drift, and
a contributor reading any one of them cannot tell whether it
is current.

The downstream operational-alerting contract (the Pub/Sub
event payload from ADR-0006) is a different concern: it is a
**runtime** contract consumed by downstream automation, not a
session-time contract for an interactive agent. Conflating
the two surfaces creates scope confusion. This ADR scopes
itself to session-time agent operation only.

---

## Decision

### 1. `CLAUDE.md` is the source of truth

The repository's canonical operating contract for AI coding
agents lives in `CLAUDE.md` at the repository root. Hard
rules (R1–R8), platform principles (P1–P6), wave structure,
slash-command catalogue, and required-reading list are
authored here and only here.

### 2. `AGENTS.md` and `.codex/AGENTS.md` are pointer files

`AGENTS.md` (at the repository root) and `.codex/AGENTS.md`
(under the Codex CLI workspace directory) are **thin pointer
files**. Each:

- names `CLAUDE.md` as the canonical contract;
- points at the playbooks directory (`.claude/playbooks/`)
  as the operational protocol;
- introduces **no new rules** beyond what `CLAUDE.md` and
  the playbooks establish.

A reader landing on either pointer file is one hop away from
the authoritative content.

### 3. `/sync-agents` is the operative enforcement mechanism

The `/sync-agents` slash command propagates `CLAUDE.md`
changes into the pointer files. Rule authorship flows
`CLAUDE.md` → pointer files, never the reverse. A contributor
proposing a rule change writes it in `CLAUDE.md`, runs
`/sync-agents`, and commits the propagation in the same MR.

### 4. `.codex/AGENTS.md` is scaffolded with this pointer shape

The `.codex/AGENTS.md` file is created during root-
infrastructure scaffolding as a pointer file with the same
shape as `AGENTS.md`: it names `CLAUDE.md` as canonical and
`.claude/playbooks/` as the operational reading list.
`.codex/` may carry future Codex-CLI-specific configuration,
but the agent-contract surface remains a thin pointer.

### 5. Future agent surfaces follow the pointer model

Any additional agent-facing entry point added later (a
different CLI agent, an editor integration with its own
configuration file) adopts the same pointer model:
`CLAUDE.md` is the source of truth; the new file is a thin
pointer.

### 6. The operational-alerting event contract is out of scope

The Pub/Sub event-payload contract from ADR-0006 is the
contract for **downstream operational-alerting consumers**.
It is orthogonal to this ADR's session-time agent contract
and does not interact with `CLAUDE.md`, `AGENTS.md`, or
`.codex/AGENTS.md`.

---

## Consequences

1. **A reader can find the current rules in one place.**
   Anyone landing on `AGENTS.md` or `.codex/AGENTS.md` is
   directed to `CLAUDE.md`; anyone landing on `CLAUDE.md`
   is reading the authoritative version.

2. **Pointer drift is caught by `/sync-agents`.** The skill
   reads `CLAUDE.md`, compares the salient extracts in the
   pointer files against it, and reports drift. The
   propagation is mechanical, not author-judgement.

3. **Pointer files do not duplicate rule prose.** They
   summarize and link, they do not restate. A future
   pointer reader who needs the full rule reads
   `CLAUDE.md`; the pointer's job is to direct, not to
   re-author.

4. **Whether `/sync-agents` becomes a CI-enforced gate is a
   follow-up.** Today it is advisory (a contributor runs
   it; the propagation lands in the same MR). Promoting it
   to a CI gate that fails when pointer files diverge from
   `CLAUDE.md` summaries is a future option.

5. **`.codex/` may grow Codex-CLI-specific configuration in
   the future**, but its agent-contract surface remains the
   pointer file. Tool-specific configuration (permissions,
   approval models, environment-specific knobs) lives in
   tool-specific files alongside the pointer, never inside
   it.

6. **The operational alerting contract is documented
   elsewhere.** ADR-0006 carries it. This ADR's scope
   boundary prevents future contributors from looking for
   the event payload in agent-facing files and finding the
   wrong contract.

---

## Notes

- Whether `.codex/` needs its own playbooks directory is
  not foreclosed; the default is to reuse
  `.claude/playbooks/` by reference from
  `.codex/AGENTS.md`. A future Codex-CLI-specific playbook
  set is a separate decision.
- The permission / approval model for the Codex CLI agent
  in this repository is a follow-up.
- Drift-detection enforcement (whether `/sync-agents`
  failures block merges) is a follow-up CI design item.
