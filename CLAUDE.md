<!-- path: CLAUDE.md -->
<!-- audience: Claude Code, Codex CLI, and any other AI coding agent operating in this workspace -->
<!-- status: living document. Update intentionally, never casually. -->

# Project Operating Contract — DQ Platform

This file is the canonical context for AI coding agents working in this
repository. It is loaded automatically by Claude Code at session start
and referenced explicitly by `AGENTS.md` for other agents.

**Read this entire file before producing any output in this repository.**

---

## 1. What this project is

The **DQ Platform** is a long-lived internal Data Quality engine for
curated data assets on BigQuery, with a planned evolution toward
stream-based validation over Kafka.

The platform exists to make data quality posture **visible, owned, and
operationally actionable** for every onboarded entity — without
coupling that visibility to custom code or tribal knowledge.

It is organized as a **single monorepo** with five logical workspaces:

- **`engine/`** — Go runtime, DSL schema source of truth, compilers,
  scheduler integration, reporting, alerting.
- **`rules/`** — YAML rule specifications by entity, owner metadata,
  governance workflow, contributor-facing guidance.
- **`tools/`** — auxiliary CLIs (linter, dry-run runner, manifest
  publisher).
- **`deploy/`** — Kubernetes manifests, infrastructure configuration,
  environment definitions.
- **`docs/`** — cross-workspace documentation: architecture, ADRs,
  glossary, governance.

The platform is **decoupled from the ingestion path**: it does not
block data delivery; it evaluates trust in delivered data, in parallel.

For the full project framing, read the foundation documents in
`studies/foundation/`, in numbered order. They are the canonical source
for everything below.

---

## 2. The three waves (current operating phase)

Work in this repository proceeds in **three sequential waves**. Each
wave has a clear gate: do not cross the gate without explicit human
approval.

### Wave 1 — Resolve blocking decisions

**Status: in progress.**

The decisions tracked in `studies/foundation/06-decision-log.md` as
priority `B0` (blocking) must be resolved before any workspace gains
real content. Each resolution becomes a dated document in
`studies/decisions/`.

The seven B0 topics, summarized:

1. Engine ↔ rules compatibility model (how rule artifacts declare which
   schema and linter contract they follow inside the monorepo).
2. Run identity and idempotency (`execution_id` semantics, rerun
   behavior).
3. Result write model (`dq_executions` and `dq_check_results` storage
   semantics).
4. Failure scope (when a check fails, what is the entity's status).
5. Manifest publication atomicity (how a ruleset becomes "live").
6. Alert routing contract (shape of owner metadata, deduplication,
   severity).
7. Loader / scheduler / retry failure semantics.

### Wave 2 — Lock platform decisions

A single consolidated decision document in `studies/decisions/`
resolves:

- Git host choice (affects every CI artifact).
- Multi-agent contract for `.claude/`, `.codex/`, and `AGENTS.md`.
- Docker Compose local scope: which cloud services are emulated, which
  require sandbox access.
- Documentation language policy.
- Workspace tooling choice (Go workspaces via `go.work` is the working
  default; confirm or revise).
- Per-workspace tag prefix conventions (e.g. `engine-v1.2.0`,
  `rules-v0.5.0`).

### Wave 3 — Scaffold every workspace

Only after Wave 1 and Wave 2 are closed. Wave 3 populates `engine/`,
`rules/`, `tools/`, `deploy/`, and `docs/` with real content backed by
the decisions made in waves 1 and 2 — never placeholders.

---

## 3. Hard rules for AI agents in this repository

These are non-negotiable. They protect the quality bar that the
foundation documents establish.

### R1. Do not write production code during waves 1 and 2

The output of waves 1 and 2 is **documents only**. No Go files, no YAML
rule definitions, no Dockerfile, no CI pipeline. If a wave-1 or wave-2
output would benefit from showing a code shape, use a fenced block
inside a markdown file — never a real file in `engine/`, `rules/`,
`tools/`, or `deploy/`.

### R2. Do not invent requirements

If a decision depends on information not available in this repository,
record the gap explicitly as a `TBD` with scope. Do not guess and do
not fabricate consensus that does not exist.

### R3. Do not revisit settled architectural decisions without strong cause

Decisions marked `resolved` in `studies/foundation/06-decision-log.md`
or already promoted to an ADR are final unless you have identified a
genuine inconsistency. If you believe one is wrong, raise it
explicitly — do not quietly design around it.

### R4. One topic per session

When a session is focused on a specific B0 item or a specific
scaffolding area, stay inside that scope. If you identify adjacent
work, list it for a future session — do not expand the current one.

### R5. Cite only this repository

Every architectural claim in a produced document should cite either:

- a foundation document in `studies/foundation/`,
- a prior decision in `studies/decisions/`,
- a promoted ADR under `docs/adr/`,
- or be explicitly marked as **new contribution proposed here,
  requires review**.

**Do not reference external systems, prior art, vendor patterns, or
internal projects by name** in any produced artifact. If a pattern is
worth using, describe it in our own terms and own it. The repository
must read as if it were the only source of these ideas.

### R6. Path header on every produced file

Every markdown file produced in this repository must start with an
HTML comment header declaring its path:

```markdown
<!-- path: studies/decisions/2026-05-21-example.md -->
```

This makes outputs reconstructible if extracted, zipped, or moved.

### R7. Output language: English for technical artifacts

ADRs, schemas, technical READMEs, code comments, and contract
documents are in English. Internal onboarding guides may be in
Portuguese, clearly marked. Default to English unless instructed
otherwise.

(This rule is provisional until Wave 2 confirms it.)

### R8. Reasoning artifacts in `studies/` are not part of the published repository

The `studies/` directory captures reasoning, drafts, and decision
history. It informs the repository but is not the repository's product.
When promoting a study to an ADR or a doc, **rewrite for the new
audience** — do not link backwards into `studies/` from a published
artifact. Studies are scaffolding; ADRs are the building.

---

## 4. Non-negotiable platform principles

These are promoted from `studies/foundation/01-charter-and-principles.md`.
They must not be eroded by any output produced here.

- **P1. Rules must remain declarative.** No raw SQL, no embedded
  expressions, no escape hatches in the DSL.
- **P2. Engine behavior must be deterministic.** Same rule version,
  same window, same source state → same execution semantics.
- **P3. Ownership must be explicit everywhere.** No entity without
  owner, no alert without route, no repository area without policy.
- **P4. Cost is a first-class constraint.** Partition discipline, query
  templates, dry-run visibility, and concurrency budgets are platform
  design, not afterthought hardening.
- **P5. Evolution must be contract-driven.** Schema, linter,
  examples, and rule artifacts evolve under a published compatibility
  contract — even inside a single repository.
- **P6. Borrow patterns, not baggage.** Patterns adopted by this
  project are described in our own terms and judged on their fit to
  our context. External provenance is not a justification.

---

## 5. How to behave inside a session

- **Plan before you touch files.** For anything beyond a one-file
  edit, produce a short plan first and wait for approval before
  executing. Use Claude Code's plan mode when available.
- **Re-read the foundation documents at the start of each new session.**
  They are short. Re-reading them is cheap; drifting from them is
  expensive.
- **When in doubt, ask.** It is always better to surface ambiguity
  than to choose silently and create an architecture decision by
  accident.
- **Default to short, structured outputs.** Long prose hides
  trade-offs. Use the document templates referenced by the slash
  commands in `.claude/commands/`.
- **End each substantial change with a one-line summary** the human
  can copy into a git commit.

---

## 6. Slash commands available

Defined under `.claude/commands/`. Use them — they encode the
workflow.

- `/resolve-b0 <topic>` — produce a draft decision document for one
  B0 item.
- `/critique <file>` — run an adversarial critique on a document.
- `/promote-to-adr <study-file>` — convert a stable study into a
  formal MADR ADR in the correct destination.
- `/sync-agents` — propagate context changes across `CLAUDE.md`,
  `AGENTS.md`, and `.codex/AGENTS.md`.
- `/check-decision-backlog` — report which decisions are open,
  in-progress, or closed.

---

## 7. What success in Wave 1 looks like

By the end of Wave 1:

- `studies/decisions/` contains a dated document for every B0 topic.
- `studies/foundation/06-decision-log.md` has its B0 rows marked
  resolved, each with a link to its decision document.
- `engine/`, `rules/`, `tools/`, `deploy/`, and `docs/` contain no
  content yet (they may not even exist as directories — that is
  fine).

That last clause is the test. If any workspace gained content during
Wave 1, the wave was not respected.
