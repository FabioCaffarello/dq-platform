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

### 2.2 Wave-S — record-mode capability (full gate met)

Launched 2026-05-23 via [ADR-0020](docs/adr/0020-wave-s-launch.md)
as a wave-shaped extension that opens the record-oriented evaluation
axis alongside set-mode. The partial gate (B0-S1 / B0-S2 / B0-S3 —
mode primitive, kind catalog, sources schema) was met 2026-05-24,
which unblocked record-mode code shipping. **The full gate (all
seven B0-S items at `resolved-adr` per ADR-0020 §"Full Wave-S
gate") was met as of 2026-05-25** with ADR-0027 — B0-S7 promotion —
the last to land. The platform now carries a complete
record-oriented capability parallel in completeness to the
set-oriented capability Wave 3 closed. Per
[ADR-0049](docs/adr/0049-b3-evolutionary-launch.md) §(a),
record-mode items surfacing from 2026-05-25 onward are **B3**
(post-shipping against a closed wave), not B2-S.

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

The required-reading set is **session-type-aware**. The canonical
contract is committed by
[ADR-0052](docs/adr/0052-session-reading-router.md); this section
is the operative instruction. A session reads the always-on floor
(§6.1) plus the minimal playbook subset declared for its session
type in the router table (§6.2).

The router governs **only the playbook layer**. R1–R8 in §3 and
P1–P6 in §4 are read by every session **unconditionally** — §1's
"read this entire file before producing any output" mandate covers
them before §6 even applies. The router can never drop a rule or
a principle. A session that drops a rule citing the router is in
violation of §1.

### 6.1 Always-on floor

Every session reads, regardless of type:

- This file (`CLAUDE.md`) §1–8 in full — including this router
  as an intentional self-reference: every session reads §6 to
  find its own row.
- `AGENTS.md` — cross-agent convention file; rebinds the rules
  to non-Claude agents.
- `CONTRIBUTING.md` — PR-flow contract, authoritative per
  [ADR-0051](docs/adr/0051-claude-tooling-postwave3.md) Clause 2.
  Flow 5 is the post-Wave-3 PR-flow; Flow 6 is the
  operator-authorized direct-edit lane.
- `studies/foundation/06-decision-log.md` — live state surface
  for every B-row, ADR status, and Wave-S gate status.

### 6.2 The router — session type → minimal playbook reading

Six session types operate in the post-Wave-3 lane. Each row
names the minimal playbook subset for sessions of that type,
**beyond the always-on floor**. Three boundary cases (B-row
triage, study revival, ADR supersession) collapse into the six
rows per [ADR-0052](docs/adr/0052-session-reading-router.md)
Clause 2 — they do not get their own rows.

| # | Session type | When this applies (trigger) | Minimal playbook reading (beyond §6.1) |
|---|---|---|---|
| 1 | **B2 follow-up** | A B-row marked `B2` in the decision log; resolves an implementation-phase decision against an in-flight wave. | `post-wave3-session-loop.md` (step 2's wave-gate confirmation); `acceptance-criteria.md` (AC-1…AC-10; B2 studies inherit B0 study shape per [ADR-0051](docs/adr/0051-claude-tooling-postwave3.md) Notes OQ-1); `feedback-protocol.md`. |
| 2 | **B3 entry** | A B-row marked `B3-N` in the decision log; [ADR-0049](docs/adr/0049-b3-evolutionary-launch.md) §(a) eligibility filter must clear before drafting. | `post-wave3-session-loop.md` (step 2's eligibility-check sub-step); `acceptance-criteria.md`; `feedback-protocol.md`; [ADR-0049](docs/adr/0049-b3-evolutionary-launch.md) §(a) and §(b). |
| 3 | **ADR amendment** | In-place edit to an existing ADR — structured-data row amendment or Amendment-log subsection per [ADR-0050](docs/adr/0050-v1-retirement-engine-release.md) §Consequence 4. No decision rewrite. | `post-wave3-session-loop.md` step 10 (PR-flow close); `feedback-protocol.md`; the originating ADR; [ADR-0050](docs/adr/0050-v1-retirement-engine-release.md) §Consequence 4. `acceptance-criteria.md` optional — only if the amendment produces a study. |
| 4 | **ADR promotion** | A `resolved-study` is being promoted via `/promote-to-adr`. | `post-wave3-session-loop.md` step 10; `acceptance-criteria.md` (source study must have cleared AC-1…AC-10); `feedback-protocol.md` (the promotion may surface critique-style feedback on the proposed ADR text); the `/promote-to-adr` command spec. |
| 5 | **Flow 6 process edit** | Operator-authorized direct edit to `CLAUDE.md` / `AGENTS.md` / `.codex/AGENTS.md` per `CONTRIBUTING.md` Flow 6. | `CONTRIBUTING.md` Flow 6 (scope-and-gate; Flow 6 explicitly inherits Flow 5 PR-flow); `post-wave3-session-loop.md` step 10 (load-bearing playbook content for PR-flow close). `feedback-protocol.md` optional — **load-bearing if `/critique` is run** (the `/critique` command grounds on it). |
| 6 | **Implementation slice landing under a closed B-row** | A code or scaffold slice that lands the artifacts committed by a closed B-row's ADR (e.g., an [ADR-0051](docs/adr/0051-claude-tooling-postwave3.md) follow-on slice; a Wave-3 capability-matrix row landing under a Wave-3 closure). | `post-wave3-session-loop.md` (close-discipline); `wave-3-acceptance-criteria.md` (AC-W3-3 — citation discipline; AC-W3-7 — local build / lint / test gates; both scaffold-shaped semantics apply to any post-Wave-3 slice); `feedback-protocol.md`. `acceptance-criteria.md` optional — only if the slice produces a follow-up study. |

The trigger column is the concrete classification test. The
minimal-reading column is the load-bearing subset for that type
beyond §6.1.

### 6.3 Default-up safety rule

If a session does not cleanly match one row — e.g., a borderline
materiality call between Flow 6 and B3 — **read up, not down**.
Pick the next-larger reading set. Reading extra is never a
violation; reading too little may under-read a load-bearing
playbook. The router narrows confidently-classified sessions;
uncertain classification reverts to the universal-prescription
behavior for that one session.

### 6.4 Output-artifact tie-breaker

When two rows could plausibly apply (canonical case: a B2
follow-up that ships an implementation slice), disambiguate by
the session's **output artifact**:

- **Study or ADR document** → rows 1 / 2 / 3 / 4 (B2 follow-up,
  B3 entry, ADR amendment, ADR promotion). Close-gate is
  AC-1…AC-10 from `acceptance-criteria.md`.
- **Code or scaffold** → row 6 (Implementation slice). Close-gates
  are AC-W3-3 and AC-W3-7 from `wave-3-acceptance-criteria.md`.
- **Edit to a process document** (`CLAUDE.md` / `AGENTS.md` /
  `.codex/AGENTS.md`) → row 5 (Flow 6 process edit). Close-gate
  is the Flow 6 scope clause in `CONTRIBUTING.md`.

A B2 follow-up that both produces a study *and* lands an
implementation slice runs as **two separate sessions** per R4
(one topic per session). The tie-breaker resolves to one of the
six §6.2 rows in every case — it is a disambiguation between
rows, not an independent behavior axis.

### 6.5 Historical-skim set

Two playbooks are explicitly historical. They are preserved
verbatim on disk and are **not load-bearing** for any session
type in §6.2. Skim only when explicitly reading prior sessions
for shape-reference context:

- `wave-1-session-loop.md` — Wave 1 closed 2026-05-21. Shape
  reference for `post-wave3-session-loop.md`'s ten-step
  structure.
- `wave-3-session-loop.md` — Wave 3 closed 2026-05-23. Shape
  reference for the PR-flow discipline now authoritative in
  `CONTRIBUTING.md` Flow 5.

`wave-3-acceptance-criteria.md` is **not** in the
historical-skim set — its AC-W3-3 and AC-W3-7 rows remain
load-bearing for the Implementation slice row (§6.2 row 6).

### 6.6 Playbook inventory

One-line descriptions for every playbook under
`.claude/playbooks/`. The router (§6.2) declares which subset a
given session reads; this inventory exists so a contributor
scanning §6 knows what each playbook is, without opening it.

- `post-wave3-session-loop.md` — the 10-step loop for one B2
  follow-up, B3 evolutionary entry, ADR amendment, or ADR
  promotion in the post-Wave-3 lane. The current operational
  loop.
- `acceptance-criteria.md` — the binary, verifiable criteria
  (AC-1…AC-10) a decision study must meet before it can be
  marked `resolved-study`.
- `wave-3-acceptance-criteria.md` — the analogous criteria
  (AC-W3-1…AC-W3-10) for Wave 3-era scaffolding artifacts;
  still load-bearing for any post-Wave-3 implementation slice
  per §6.2 row 6.
- `feedback-protocol.md` — how feedback on a draft is given
  (always citing R/P/AC labels, never personal).
- `wave-1-session-loop.md` — historical (Wave 1 closed); shape
  reference for the post-Wave-3 loop.
- `wave-3-session-loop.md` — historical (Wave 3 closed); shape
  reference for the PR-flow discipline now in `CONTRIBUTING.md`
  Flow 5.

When a playbook turns out to be wrong, update the playbook — do
not silently diverge.

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
