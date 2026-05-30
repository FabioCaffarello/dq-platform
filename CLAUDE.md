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

## 2. Operating phases and lanes

Work in this repository is organized into **three sequential waves**
(now closed), one **wave-shaped capability extension** (Wave-S), and
one **demand-driven evolutionary lane** (B3). Wave-S and B3 are
structural peers of the original waves — not a "Wave 4" or "Wave 5".
The taxonomy is defended on this project's own terms: Wave-S launches
a new capability axis at wave shape (ADR-0020 §"Wave semantics"); B3
is an open-ended lane explicitly framed as peer to the waves rather
than a fourth priority tier (ADR-0049 §"Launch posture").

### 2.1 Waves 1–3 (closed)

The set-mode capability is live; the five product workspaces carry
real content.

- **Wave 1** — seven `B0` blocking decisions closed 2026-05-21
  (`B0-1` → ADR-0001 through `B0-7` → ADR-0007).
- **Wave 2** — consolidated platform decisions (`W2-1` … `W2-5`)
  closed 2026-05-21 and promoted to ADRs 0008–0012 + 0015–0019.
- **Wave 3** — workspace scaffolding (`W3-P0` … `W3-P8`) closed
  2026-05-23. `engine/`, `rules/`, `tools/`, `deploy/`, and `docs/`
  now carry real content.

### 2.2 Wave-S — record-mode capability (partial gate met)

Launched 2026-05-23 via [ADR-0020](docs/adr/0020-wave-s-launch.md)
as a wave-shaped extension that opens the record-oriented evaluation
axis alongside set-mode. The partial gate (B0-S1 / B0-S2 / B0-S3 —
mode primitive, kind catalog, sources schema) was met 2026-05-24,
which unblocked record-mode code shipping. The full-gate criteria
remain in ADR-0020.

### 2.3 B3 — evolutionary lane (open)

Opened 2026-05-29 via
[ADR-0049](docs/adr/0049-b3-evolutionary-launch.md) as a
demand-driven lane for post-Wave-3 evolution, restricted to three
in-scope families (kind, capability mode, tooling extensions) and
filtered by the four-condition eligibility test in ADR-0049 §(a).
B3 has no closure gate — it stays open across the platform's
operating life.

### 2.4 Live state

The canonical live-state surface for every B-row and ADR is
`studies/foundation/06-decision-log.md`. Wave gates, Wave-S
partial / full gate status, and B3 lane activity are all
reflected there in the "Last updated" history.

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

The directory layout shown in
`studies/foundation/02-monorepo-topology.md` (including
`docker-compose.yml`, `Makefile`, `go.work`, `.gitlab/` or `.github/`,
and the five product workspaces) is **descriptive of the target state**.
It exists in that document so reviewers and contributors can reason
about the eventual shape of the repository. It is **not** an
authorization to create any of those files now. R1 continues to
forbid producing them during waves 1 and 2; they appear during
Wave 3, backed by the decisions resolved in waves 1 and 2.

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

### R5. Cite only this repository, and own your patterns

Every architectural claim in a produced document should cite either:

- a foundation document in `studies/foundation/`,
- a prior decision in `studies/decisions/`,
- a promoted ADR under `docs/adr/`,
- or be explicitly marked as **new contribution proposed here,
  requires review**.

In addition, the following naming prohibition applies to every
produced artifact:

- **Do not name internal projects** of the organization or other
  teams (sibling platforms, predecessor systems, sister tools) as
  justification, comparison, or pattern source.
- **Do not name prior art** — products, vendor designs, or
  third-party systems held up as the *source of an idea* — as
  justification for an architectural choice. If a pattern is worth
  adopting, describe it in our own terms and defend it on its own
  merits. The repository must read as if it were the only source of
  these ideas.

**What is exempt — environment, not borrowed ideas.** Naming the
commodity technologies and public substrates the platform runs on or
against is allowed and expected: BigQuery, Kafka, GCS, Pub/Sub, OIDC,
Prometheus, OpenTelemetry, Kubernetes, Go, Docker, slog, JSON Schema,
and equivalents. These are the environment we operate in, not
patterns we are borrowing. Describing them clearly is necessary to
keep the documents accurate.

The line is: **"we use X" is fine. "we are doing Y because X does Y"
is not.**

### R6. Path header on every produced file

Every markdown file produced in this repository must start with an
HTML comment header declaring its path:

```markdown
<!-- path: studies/decisions/2026-05-21-example.md -->
```

This makes outputs reconstructible if extracted, zipped, or moved.

R6 applies to markdown files only. Non-markdown files (JSON, YAML,
code) are out of scope regardless of whether they support comment
syntax.

### R7. Output language: English for technical artifacts

ADRs, schemas, technical READMEs, code comments, and contract
documents are in English. Internal onboarding guides may be in
Portuguese; each such file must open with a one-line language
marker (e.g., `> Language: Portuguese (Brazilian)`). Default to
English unless instructed otherwise.

(Confirmed in Wave 2 — see
[`studies/decisions/2026-05-21-platform-decisions-wave2.md`](studies/decisions/2026-05-21-platform-decisions-wave2.md)
§4 W2-4.)

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
- **P6. Borrow patterns, not baggage.** Named design patterns
  adopted by this project are described in our own terms and judged
  on their fit to our context; external provenance is not a
  justification. This principle governs *borrowed ideas*, not the
  commodity infrastructure the platform runs on (see R5 for the
  exact scope and exemptions).

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

## 6. Required reading at session start

Two reading layers govern session-start grounding. Both are short.
Read them at the start of every session, alongside the foundation
documents.

**Playbooks (operational protocol)** under `.claude/playbooks/`:

- **`wave-1-session-loop.md`** — the 10-step loop for resolving one
  B0 decision. Includes explicit human decision points. Historical
  reference now that Wave 1 has closed.
- **`wave-3-session-loop.md`** — the loop for one Wave 3 scaffolding
  unit. Historical reference now that Wave 3 has closed.
- **`post-wave3-session-loop.md`** — the 10-step loop for one B2
  follow-up, B3 evolutionary entry, ADR amendment, or ADR promotion
  in the post-Wave-3 lane. The current operational loop.
- **`acceptance-criteria.md`** — the binary, verifiable criteria a
  decision study must meet before it can be marked `resolved-study`.
- **`wave-3-acceptance-criteria.md`** — the analogous criteria for
  Wave 3 scaffolding artifacts.
- **`feedback-protocol.md`** — how feedback on a draft is given
  (always citing R/P/AC labels, never personal).

**Contributor contract:**

- **`CONTRIBUTING.md`** Flow 5 — the upstream authority for PR-flow
  in the post-Wave-3 lane (committed by
  [ADR-0051](docs/adr/0051-claude-tooling-postwave3.md) Clause 2).
  The `.claude/` playbooks and skills defer to it; read it directly
  to find the canonical branch / commit / PR discipline.

These playbooks are operational, not architectural. When a playbook
turns out to be wrong, update the playbook — do not silently
diverge.

---

## 7. Slash commands available

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
- `/open-pr` — open a PR from the current feature branch against
  `main` following the PR-flow checklist authoritative in
  `CONTRIBUTING.md` Flow 5.

---

## 8. Where success is measured

Waves 1, 2, and 3 closed. Success is no longer measured by a
phase-specific closure test inside this document; it is measured by
B-row and ADR closure recorded in
`studies/foundation/06-decision-log.md`. The closure criteria for
the open phases live in their governing ADRs:
[ADR-0020](docs/adr/0020-wave-s-launch.md) for the Wave-S
partial / full gate; [ADR-0049](docs/adr/0049-b3-evolutionary-launch.md)
§(c) for B3 (the lane has no closure gate — by design).
