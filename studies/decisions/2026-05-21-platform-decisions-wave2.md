<!-- path: studies/decisions/2026-05-21-platform-decisions-wave2.md -->

# Wave 2 — Consolidated Platform Decisions

## Metadata

- **Wave reference:** Wave 2 (consolidated)
- **Status:** resolved-study (two critique rounds; round 2 cleared
  with no blocking findings)
- **Last updated:** 2026-05-21
- **Upstream resolved:** B0-1, B0-2, B0-3, B0-4, B0-5, B0-6, B0-7
- **Downstream open:** Wave 3 scaffolding of `engine/`, `rules/`, `tools/`,
  `deploy/`, `docs/`
- **Promotion target:** five per-item ADRs under `docs/adr/`
  (`0008-git-host.md`, `0009-multi-agent-contract.md`,
  `0010-substrate-posture.md`, `0011-documentation-language.md`,
  `0012-tag-conventions.md`), each promoted independently during
  Wave 3. See §Promotion target for the numbering rationale.
- **Wave 1 closure:** confirmed — `studies/foundation/06-decision-log.md`
  reports "7 of 7 resolved — gate met"

---

## Context

Foundation 02 (`studies/foundation/02-monorepo-topology.md`) and the
decision log identify five Wave 2 items that must be resolved before
any product workspace is scaffolded:

- **W2-1.** Git host choice (affects every CI artifact).
- **W2-2.** Multi-agent contract for `.claude/`, `.codex/`, and
  `AGENTS.md`.
- **W2-3.** Docker Compose local scope: which substrate is emulated,
  which requires sandbox cloud access.
- **W2-4.** Documentation language policy.
- **W2-5.** Per-workspace tag prefix conventions.

Wave 1 (B0-1 through B0-7) is closed and its commitments are
load-bearing for several Wave 2 items — especially W2-3, which
inherits substrate requirements from B0-3, B0-5, B0-6, and B0-7.

This document resolves all five Wave 2 items in a single study. The
structure mirrors the MADR shape used by B0 studies — Metadata,
Context, Decision Drivers, Considered Options, Recommendation,
Consequences, Open Questions, Promotion target — and each W2 item
appears as a numbered section with its own mini block of Drivers,
Options, Recommendation, Consequences, and Open Questions.

**Structural extension (AC-2 deviation, recorded explicitly).** The
five per-item mini-MADR sections (§§1–5) sit between the outer
"Recommendation (meta)" and the outer cross-cutting "Consequences".
This preserves the MADR contract — every outer section AC-2
requires is present, in order — while letting each W2 item carry
its own self-contained option-space analysis. Readers who only
care about one item can read its mini-MADR in isolation; readers
who care about cross-item coupling can read the outer sections.

What Wave 2 explicitly does **not** do:

- It does **not** create production code, scaffolding files, or
  workspace contents. CLAUDE.md R1 still applies; foundation 02's
  layout is descriptive of the target state, not authorization to
  build it now. Files like `docker-compose.yml`, `.github/`,
  `.codex/AGENTS.md`, and the per-workspace `go.mod` files are
  Wave 3 outputs.
- It does **not** revisit any commitment marked `resolved` in
  `studies/foundation/06-decision-log.md` (CLAUDE.md R3).
- It does **not** pick concrete emulator images or CI runner
  primitives. Wave 2 commits **capability requirements**; Wave 3
  picks artifacts that satisfy them.

---

## Decision Drivers (cross-cutting)

D1. **No erosion of Wave 1 commitments.** Every Wave 2 choice must
honour the B0 commitments cited in the relevant section.

D2. **Independent workspace evolution** is the foundation 02 default
for the release model (§"Release Model"). Wave 2 must keep this
default workable.

D3. **Capability framing over vendor framing** for substrate
questions. Wave 2 commits to capabilities (e.g., "object store with
generation-conditional writes"); the specific emulator or cloud
target is Wave 3.

D4. **Multi-agent ergonomics without contract drift.** The three
agent-facing surfaces (`CLAUDE.md`, `AGENTS.md`, `.codex/AGENTS.md`)
must have one source of truth.

D5. **Contributor accessibility.** Local dev parity and language
policy must lower onboarding friction without compromising the
quality bar.

D6. **R5 hygiene.** No prior-art or sibling-platform names appear as
justification. Commodity infrastructure (BigQuery, Pub/Sub, GCS,
OIDC, GitHub, Docker, Go, Kafka, Prometheus, OpenTelemetry) is
named only as environment, not as borrowed pattern.

---

## Considered Options (meta-shape)

Three ways to structure Wave 2's output:

- **(A) One consolidated study with five mini-MADR blocks.**
  Cross-item dependencies (especially W2-1↔W2-3 substrate, W2-5↔B0-2
  encoding) are visible in one read. **Recommended.**
- **(B) Five independent dated studies.** Higher per-item ceremony,
  but cross-item links are harder to keep coherent.
- **(C) Defer everything to Wave 3 scaffolding PRs.** Violates the
  three-wave operating model; Wave 3 may not begin until Wave 2 is
  closed.

---

## Recommendation (meta)

Adopt **(A)**. Per-item summary:

| Item | Decision (one line) |
|------|---------------------|
| W2-1 Git host             | **GitHub.** CI runner choice deferred. |
| W2-2 Multi-agent contract | `CLAUDE.md` source of truth; `AGENTS.md` and `.codex/AGENTS.md` are pointers. |
| W2-3 Docker Compose scope | Hybrid; capability matrix names what is emulated locally and what requires sandbox cloud access. |
| W2-4 Documentation language | Confirm CLAUDE.md R7 — English for technical artifacts; Portuguese allowed for onboarding when marked. |
| W2-5 Tag conventions      | Confirm foundation 02 (`engine-v*`, `rules-v*`, `tools-lint-v*`, `deploy-v*`) with an explicit B0-2 input-safety constraint. |

Details follow in §§1–5.

---

## 1. W2-1 — Git host

### 1.1 Drivers

- **Committed input.** GitHub has been chosen by the project owner
  and is not under debate in this study. The study records the
  choice and its downstream implications.
- **B0-1 C2** requires a mandatory CI gate on `main` that enforces
  byte-equality between `engine/internal/dsl/schema/v<N>.schema.json`
  and `rules/_schema/v<N>.schema.json`. The Git host must support a
  branch-protection model strong enough to make this gate
  non-bypassable.
- **B0-1 C8** requires `CODEOWNERS` protection on the schema source
  path.
- **B0-1 C10** requires the linter version pin to be **unforgeable**
  and explicitly defers the mechanism to W2-1.
- **B0-5 CC1/CC3** require generation-conditional writes for the
  manifest pointer. This is a substrate-layer requirement (W2-3) but
  the CI lane that publishes the manifest lives on the chosen Git
  host.

### 1.2 Options

- **(A) GitHub.**
- **(B) Other hosted Git platforms.** Listed for record only; no
  option-space analysis because (A) is committed input.
- **(C) Self-hosted Git.** Listed for record only; same reason.

### 1.3 Recommendation

**(A) GitHub.** Provenance: user directive to the Wave 2 session on
2026-05-21 — "GitHub is the chosen host — committed input, not
under debate." This is a fourth provenance class beyond AC-4's
three categories (foundation doc / prior decision / "new
contribution proposed here, requires review"): a project-owner
directive that fixes an input the study records but does not
re-derive. The Drivers in §1.1 inherit from this directive; the
Consequences in §1.4 are the only material content.

### 1.4 Consequences

- **C-W2-1.1.** Every reference to `.gitlab/` in foundation 02 —
  currently the line `.gitlab/ or .github/` in the Top-Level
  Directory Layout — collapses to `.github/` from Wave 3 onward.
- **C-W2-1.2.** `CODEOWNERS` uses the GitHub-native syntax to
  satisfy B0-1 C8.
- **C-W2-1.3.** **GitHub Actions is the default CI runner**;
  alternative runners (self-hosted, external) are a future
  sub-decision (W2-1-sub-1, deferred to Wave 3 or later).
- **C-W2-1.4.** The unforgeable linter pin required by B0-1 C10
  must be implemented on the chosen host's primitives — digest
  pinning of a registry image, or a content-addressed artifact
  store reference. The **specific mechanism is a Wave 3 design
  item**, not a Wave 2 commitment.
- **C-W2-1.5.** The container/artifact registry choice (GHCR vs.
  alternative) is deferred (W2-1-sub-2).

### 1.5 Open Questions

- **OQ-W2-1.1.** CI runner alternative beyond GitHub Actions
  default. **Out-of-scope for current cycle — Wave 3 sub-decision
  (W2-1-sub-1).** (new contribution proposed here, requires review)
- **OQ-W2-1.2.** Artifact / container registry choice.
  **Out-of-scope for current cycle — Wave 3 sub-decision
  (W2-1-sub-2); choice is independent of Wave 2 substrate posture.**
- **OQ-W2-1.3.** Branch-protection and required-check specifics
  satisfying B0-1 C2. **Out-of-scope for current cycle — lands with
  Wave 3 CI scaffolding.**

---

## 2. W2-2 — Multi-agent contract

### 2.1 Drivers

- `CLAUDE.md` §1 already names itself "the canonical context for
  AI coding agents working in this repository". Foundation 02
  already lists `.claude/`, `.codex/`, and `AGENTS.md` as
  top-level artifacts, with `.codex/` annotated "Wave 2+".
- A slash command at `.claude/commands/sync-agents.md`
  (`/sync-agents`) already exists, presuming a propagation
  contract. Wave 2 must commit what it propagates and from where.
- **B0-6 CC4** defines the Pub/Sub event payload consumed by
  downstream alerting automation. This is an operational contract,
  **not** a session-agent contract. The multi-agent decision must
  call out the scope boundary to prevent later confusion.
- No prior study under `studies/foundation/` resolves this; W2-2
  is the first decision on the shape.

### 2.2 Options

- **(A) `CLAUDE.md` is authoritative; `AGENTS.md` and
  `.codex/AGENTS.md` are thin pointers.** One source of truth;
  pointer files name the authoritative file and the playbooks
  directory. **Recommended.**
- **(B) Three sibling files, all maintained in parallel.** Higher
  drift risk; the existing `/sync-agents` skill implies a
  pointer model, not a sibling model.
- **(C) `AGENTS.md` authoritative; `CLAUDE.md` and
  `.codex/AGENTS.md` are pointers.** Inverts the model already
  established by `CLAUDE.md` §1.

### 2.3 Recommendation

**(A).** `CLAUDE.md` remains the source of truth. `AGENTS.md` is a
neutral pointer that names the canonical context file and the
playbooks directory in two or three lines. `.codex/AGENTS.md` is
committed here with the same pointer shape; its actual file
creation is Wave 3 (R1 forbids it now).

### 2.4 Consequences

- **C-W2-2.1.** The `/sync-agents` skill is the **operative
  enforcement mechanism** for this contract. Rule changes are
  authored in `CLAUDE.md` first and propagated to the two pointer
  files; pointer files do not introduce new rules.
- **C-W2-2.2.** `.codex/AGENTS.md` is created during Wave 3
  scaffolding (CLAUDE.md R1). The shape committed here: a pointer
  file naming `CLAUDE.md` as canonical and `.claude/playbooks/`
  as the operational reading list.
- **C-W2-2.3.** Future agent surfaces (other CLI agents, editor
  integrations) adopt the same pointer model. **New contribution
  proposed here, requires review.**
- **C-W2-2.4.** The Pub/Sub event-payload contract from **B0-6
  CC4** is the contract for *downstream alerting automation*. It is
  orthogonal to the session-agent contract resolved here and is
  not part of W2-2 scope.

### 2.5 Open Questions

- **OQ-W2-2.1.** Whether `.codex/` needs its own playbooks
  directory. Default position: no — reuse `.claude/playbooks/` by
  reference from `.codex/AGENTS.md`. **Out-of-scope for current
  cycle — settled as default unless a future agent surface
  demands otherwise.** (new contribution proposed here, requires
  review)
- **OQ-W2-2.2.** Permission / approval model for the Codex CLI
  agent in this repository. **Out-of-scope for current cycle —
  depends on whether `.codex/` is adopted at all in Wave 3.**
- **OQ-W2-2.3.** Drift-detection: should `/sync-agents` be
  enforced by CI (e.g., a job that fails when the pointer files
  diverge from `CLAUDE.md` summaries), or remain advisory?
  **Out-of-scope for current cycle — Wave 3 CI design item.**
  (new contribution proposed here, requires review)

---

## 3. W2-3 — Docker Compose local scope

### 3.1 Drivers

This is the most B0-coupled Wave 2 item. The substrate that
Compose must stand up — or that Compose must explicitly **not**
stand up — is determined by Wave 1 commitments:

- **B0-3 CC1**: two BigQuery-shaped tables, `dq_executions` and
  `dq_check_results`, both append-only.
- **B0-3 CC2**: `dq_executions_current` is a lazy view computed at
  query time via `ROW_NUMBER() OVER (PARTITION BY execution_id
  ORDER BY recorded_at DESC)`.
- **B0-5 CC1**: object-store layout with `manifests/latest.json`
  (mutable pointer), `manifests/by-hash/sha256-<hex>.json`
  (immutable bodies), `yamls/by-hash/sha256-<hex>.yaml`
  (immutable rule YAMLs).
- **B0-5 CC3**: the pointer is rewritten only via a
  generation-conditional write (compare-and-swap).
- **B0-5 CC11**: sha256 is the hash algorithm throughout.
- **B0-6 CC3/CC4**: engine emits to a Pub/Sub topic with the
  payload defined in CC4 (identity, routing, status, context).
- **B0-7 CC1**: loader at startup exits the process on any
  failure.
- **B0-7 CC2**: loader at refresh refuses the swap, retains the
  prior manifest, escalates after N failures.
- **B0-7 CC9**: refresh cadence is periodic with hash short-circuit
  on the pointer.
- **B0-7 CC11**: orphan-run detection polls `dq_executions` for
  `status = running` rows older than a threshold.
- **B0-1 C2 / C10**: the byte-equality CI gate and the
  unforgeable linter pin both live in the build substrate.

Foundation 02 §"Local Development" already states that
`docker-compose.yml` "emulates whichever external services the
Wave 2 decision allows to be emulated". This study makes that
decision in capability terms.

Cost-as-first-class (P4) is also a driver: pulling a full cloud
sandbox for every contributor flow is not acceptable for common
tasks like running a rule lint or a single check evaluation.

### 3.2 Options

- **(A) Pure emulation.** Every substrate emulated locally in
  Compose. Maximizes local parity but assumes high-fidelity
  emulators for every capability — not all of these exist.
- **(B) Hybrid capability matrix.** Emulate locally what has
  fidelity-true local options; require sandbox cloud access for
  the rest. **Recommended.**
- **(C) Pure cloud sandbox.** Compose orchestrates only the engine
  binary; all substrate is in a sandbox cloud project. Highest
  fidelity but worst contributor friction.

### 3.3 Recommendation

**(B).** The substrate posture is expressed as a capability
matrix. The matrix commits **what** must be available locally and
**what** must be sandbox-cloud; it does not name specific emulator
images or cloud projects (Wave 3).

| Capability | Local emulation? | Notes |
|---|---|---|
| Pub/Sub publish/subscribe | **Yes** | B0-6 CC3/CC4 payload must be exercisable end-to-end without a sandbox. |
| Object store: generation-conditional pointer write | **Yes** | B0-5 CC3 compare-and-swap on `manifests/latest.json` must be testable locally. |
| Object store: `by-hash/` immutability with sha256 | **Yes** | B0-5 CC1, CC11. |
| Tabular store: append-only writes | **Yes** | B0-3 CC1 invariant must be verifiable locally. |
| Tabular store: lazy view with `ROW_NUMBER() OVER (PARTITION BY … ORDER BY …)` | **Partial** | B0-3 CC2 fidelity gap is known; sandbox-recommended for full end-to-end validation. |
| OIDC / service identity for cross-substrate auth | **No (sandbox required)** | Production-shape identity flows cannot be emulated with fidelity. Wave 3 Compose **may** stand up a permissive development stub for local-only auth, but the stub does not satisfy production identity semantics and is never the path exercised in the sandbox lane. |
| Unforgeable linter image pin (B0-1 C10) | **Partial** | Local digest pinning works for development; production unforgeability lives in the host's registry primitives (W2-1). |
| Structured logs / metrics endpoint (B0-7 CC14) | **Yes** | Engine exposes a metrics endpoint locally; collector wiring is Wave 3. |
| Orphan-run detection polling (B0-7 CC11) | **Yes** | Logic is engine-side; only requires the tabular store, which is **Yes** above. |
| Cost-ceiling enforcement (B0-7 CC10 `aborted`) | **Yes** | Engine-side budget logic; can be exercised against either local or sandbox tabular store. |

Concrete artifacts that satisfy each row — emulator images,
collector binaries, sandbox project bootstraps — are **not**
decided here. That is Wave 3, and it can change without
re-opening Wave 2 so long as it preserves the capability rows.

### 3.4 Consequences

- **C-W2-3.1.** Wave 3 `docker-compose.yml` covers every **Yes**
  row. Contributors must be able to exercise the B0-3, B0-5, B0-6,
  and B0-7 happy paths without sandbox access.
- **C-W2-3.2.** **Sandbox cloud access is a documented contributor
  onboarding artifact** (Wave 3 doc, location TBD) for the
  **No** and **Partial** rows.
- **C-W2-3.3.** Integration tests split into two CI lanes:
  `local-runnable` and `sandbox-required`. The split lives in test
  labels (or build tags), not in directory layout. **New
  contribution proposed here, requires review.**
- **C-W2-3.4.** The end-to-end flow "manifest publish → loader
  hash-short-circuit refresh (B0-7 CC9) → execution write
  (B0-3 CC1) → operational alert publish (B0-6 CC4)" can be
  exercised locally. The "OIDC-bound publish" variant requires
  sandbox.
- **C-W2-3.5.** This capability matrix is the **substrate posture
  contract**. Future emulator or sandbox changes update this
  matrix; workspace-level docs do not redefine substrate
  expectations. **New contribution proposed here, requires
  review.**
- **C-W2-3.6.** The B0-3 CC2 lazy-view fidelity gap is **not** a
  blocker for local development; engine logic that depends on view
  semantics must be exercisable in unit tests against an
  abstraction layer, with full-view fidelity verified in the
  sandbox lane. **New contribution proposed here, requires
  review.**

### 3.5 Open Questions

- **OQ-W2-3.1.** Specific emulator image and version choices for
  each **Yes** row. **Out-of-scope for current cycle — Wave 3
  picks artifacts that satisfy the capability matrix.**
- **OQ-W2-3.2.** One-command sandbox bootstrap (e.g., a make
  target). **Out-of-scope for current cycle — Wave 3 onboarding
  artifact.**
- **OQ-W2-3.3.** Exact mechanism for documenting the B0-3 CC2
  view-fidelity gap to contributors writing B0-3- or B0-4-touching
  code. **Out-of-scope for current cycle — Wave 3 contributor doc,
  written alongside the engine workspace README.**
- **OQ-W2-3.4.** Whether the structured-logs pipeline (B0-7 CC14)
  needs a local collector image in Compose, or only a metrics
  endpoint that contributors curl directly. **Out-of-scope for
  current cycle — Wave 3 observability sub-decision; both options
  satisfy the capability row.**
- **OQ-W2-3.5.** Whether the operator-issued abort path in
  B0-7 CC10 needs a local admin-API mock, or whether it is
  exercised only in the sandbox lane. **Out-of-scope for current
  cycle — Wave 3 admin-API design item.**

---

## 4. W2-4 — Documentation language

### 4.1 Drivers

- `CLAUDE.md` R7 currently states (provisional): "English for
  technical artifacts; Portuguese optional for onboarding".
- Every existing foundation document and every B0 decision study
  is in English; promotion would not require translation today.
- `CLAUDE.md` R8 forbids back-links from published artifacts into
  `studies/`, which implies promotion is a real rewrite — language
  stability across the rewrite reduces friction.

### 4.2 Options

- **(A) English-only for every artifact in the repository.**
  Strictest; rules out Portuguese onboarding guides.
- **(B) Confirm `CLAUDE.md` R7.** English for technical artifacts
  (ADRs, schemas, READMEs, code comments, contract documents);
  Portuguese permitted for onboarding and internal guides when
  the file opens with a one-line language marker. **Recommended.**
- **(C) Portuguese primary; English on promotion.** Higher local
  ergonomics but adds translation cost to every Wave 3 promotion.

### 4.3 Recommendation

**(B).** `CLAUDE.md` R7 graduates from provisional to confirmed.

### 4.4 Consequences

- **C-W2-4.1.** ADRs (`docs/adr/`), schemas
  (`engine/internal/dsl/schema/`, `rules/_schema/`), READMEs in
  every workspace, code comments, and contract documents are
  written in English.
- **C-W2-4.2.** Onboarding guides may be in Portuguese; each such
  file must open with a one-line language marker (e.g.,
  `> Language: Portuguese (Brazilian)`). **New contribution
  proposed here, requires review.**
- **C-W2-4.3.** Promotion of a study to an ADR (R8) is also a
  language-normalization step when the study contains Portuguese
  text. The `/promote-to-adr` skill should call this out.

### 4.5 Open Questions

- **OQ-W2-4.1.** Whether the language-marker convention is
  enforced by a CI lint or remains advisory. **Out-of-scope for
  current cycle — Wave 3 CI design item; advisory is the default
  until a Portuguese onboarding guide actually exists to lint.**
  (new contribution proposed here, requires review)

---

## 5. W2-5 — Per-workspace tag conventions

### 5.1 Drivers

- Foundation 02 §"Release Model" proposes per-workspace tags:
  `engine-v<major>.<minor>.<patch>`,
  `rules-v<major>.<minor>.<patch>`,
  `tools-lint-v<major>.<minor>.<patch>`,
  `deploy-v<major>.<minor>.<patch>`.
- **B0-1 C5** gates manifest publication on contract consistency
  — schema versions and engine compatibility must match.
- **B0-5 CC11** commits `ruleset_version` as a manifest field,
  shown with the example value `rules-v2.4.7`. The tag is
  effectively the manifest's identity.
- **B0-2 CC1/CC2** include `ruleset_version` as one of the five
  pipe-separated inputs to `execution_id = sha256_hex(...)`, with
  the explicit constraint of **no escaping**. Any pipe character
  in `ruleset_version` would break the hash invariant.

### 5.2 Options

- **(A) Confirm foundation 02 as-is**, with an explicit input-safety
  constraint linking the tag format to B0-2 CC2. **Recommended.**
- **(B) Add a `schema-v*` tag** for schema-only releases. Keeps
  schema independent from the engine binary; adds a fifth prefix
  to manage.
- **(C) Single monorepo-wide tag.** Conflicts with D2 (independent
  workspace evolution) and with B0-5 CC11's `ruleset_version`
  shape.

### 5.3 Recommendation

**(A) with input-safety clarification.** Tags are
`engine-v*`, `rules-v*`, `tools-lint-v*`, `deploy-v*`. The hyphen
and dot characters are permitted because they are pipe-free; the
pipe character is forbidden anywhere in any tag-prefix variant.

### 5.4 Consequences

- **C-W2-5.1.** `engine-v<major>.<minor>.<patch>` directly feeds
  the `engine_compatibility` semver range in **B0-5 CC11**. The
  engine binary that loads a manifest must satisfy the manifest's
  declared range.
- **C-W2-5.2.** `rules-v<major>.<minor>.<patch>` is the literal
  value written into the `ruleset_version` manifest field (B0-5
  CC11) and the literal value hashed into `execution_id` per
  **B0-2 CC1/CC2**.
- **C-W2-5.3.** **Pipe character (`|`) is forbidden** in any
  future tag-prefix variant. This is B0-2 CC2's no-escaping
  invariant restated as a tag-convention rule. **New contribution
  proposed here, requires review** — foundation 02 does not state
  this explicitly.
- **C-W2-5.4.** Pre-release semver suffixes (e.g.,
  `rules-v1.2.3-rc1`) are pipe-free and therefore safe inputs to
  the **B0-2 CC1** hash. Whether such a pre-release ruleset may
  be **published** as `manifests/latest.json` is a separate
  B0-5 question and is **not** decided here.
- **C-W2-5.5.** Per-workspace tag prefix enforcement (a CI gate
  rejecting an `engine-v*` tag pointing to a `rules/` change, or
  vice versa) is a Wave 3 CI design item. **New contribution
  proposed here, requires review.**
- **C-W2-5.6.** `tools-lint-v*` is the release prefix for the
  linter binary. Whether other tools under `tools/` share this
  prefix or get their own is OQ-W2-5.1.

### 5.5 Open Questions

- **OQ-W2-5.1.** Whether `tools-lint-v*` covers the whole `tools/`
  directory or only the linter binary, with other tools getting
  their own prefixes. **Out-of-scope for current cycle — revisit
  when a second tool binary lands in `tools/`; until then, the
  prefix is reserved for the linter.**
- **OQ-W2-5.2.** Schema-only release tag (Option B). **Out-of-scope
  for current cycle — current model keeps schema versioning tied
  to engine releases via B0-1 C5 contract consistency.**
- **OQ-W2-5.3.** Pre-release publication rules — whether
  `rules-v1.2.3-rc1` may write `manifests/latest.json` or only a
  side channel. **Out-of-scope for current cycle — belongs to a
  B0-5 follow-up, not to W2-5.**

---

## Consequences (cross-cutting)

- The foundation-02 placeholder `.gitlab/ or .github/` collapses to
  `.github/` from Wave 3 onward (W2-1).
- The `/sync-agents` skill becomes the operative enforcement
  mechanism; `CLAUDE.md` is the source of truth and pointer files
  do not introduce new rules (W2-2).
- The substrate posture is committed in capability terms; a future
  emulator or sandbox change updates the W2-3 capability matrix,
  not every workspace doc (W2-3).
- English-default with explicit Portuguese exception (file-opening
  language marker) holds (W2-4).
- Per-workspace tags are direct inputs to manifest and
  `execution_id` contracts; pipe-character forbidden anywhere in a
  tag prefix (W2-5).
- Wave 3 may begin once this study is `resolved-study` and the
  W2-1…W2-5 rows in `studies/foundation/06-decision-log.md` are
  updated. **Updating the decision log is a separate session.**

## Open Questions (cross-cutting)

This section is a **summary index** of the per-item Open Questions
declared in §§1.5, 2.5, 3.5, 4.5, 5.5. Every item is already
marked out-of-scope for the current cycle in its per-item block;
the bullets below restate the categories for cross-item visibility,
not new questions.

- **Out-of-scope for current cycle:** CI runner, branch-protection,
  and registry sub-decisions on the chosen host (OQ-W2-1.1, .2,
  .3). Rationale: Wave 3 CI scaffolding.
- **Out-of-scope for current cycle:** Drift-detection enforcement
  for the multi-agent pointer contract (OQ-W2-2.3). Rationale:
  Wave 3 CI design item.
- **Out-of-scope for current cycle:** Specific emulator images,
  sandbox bootstrap, contributor-doc mechanism, observability
  collector, and admin-API mock (OQ-W2-3.1…3.5). Rationale: Wave 3
  picks artifacts; capability matrix is the binding contract.
- **Out-of-scope for current cycle:** Language-marker CI
  enforcement (OQ-W2-4.1). Rationale: advisory default until a
  Portuguese onboarding guide exists.
- **Out-of-scope for current cycle:** Tag-prefix CI gate, per-tool
  tag scope, schema-only tag, pre-release publication rules
  (OQ-W2-5.1…5.3, plus C-W2-5.5 enforcement). Rationale: Wave 3 CI
  and a separate B0-5 follow-up.

## Promotion target

This study promotes to **five per-item ADRs** under `docs/adr/`,
each carrying one Wave 2 item:

| Promotes from | Target ADR filename |
|---|---|
| §1 W2-1 (Git host)               | `docs/adr/0008-git-host.md` |
| §2 W2-2 (Multi-agent contract)   | `docs/adr/0009-multi-agent-contract.md` |
| §3 W2-3 (Substrate posture)      | `docs/adr/0010-substrate-posture.md` |
| §4 W2-4 (Documentation language) | `docs/adr/0011-documentation-language.md` |
| §5 W2-5 (Tag conventions)        | `docs/adr/0012-tag-conventions.md` |

**Numbering rationale.** `docs/adr/` does not yet exist
(scaffolding is Wave 3). The numbering above assumes that the
seven Wave 1 B0 studies, when promoted, occupy 0001–0007 in the
B0-row order recorded in `studies/foundation/06-decision-log.md`,
so the Wave 2 ADRs take 0008–0012. If the Wave 1 promotion picks a
different ordering, these five filenames shift accordingly — the
ADR slugs (`git-host`, `multi-agent-contract`, etc.) are the
stable part of the contract.

**Promotion shape.** Five small ADRs, not one consolidated ADR.
Rationale: the items have independent lifecycles — a W2-1
sub-decision or a W2-3 emulator change should move without
disturbing the other four ADRs. The alternative (one consolidated
ADR mirroring this study) was considered and rejected on that
ground. **New contribution proposed here, requires review.**

**Decision-log update.** The W2-1…W2-5 rows in
`studies/foundation/06-decision-log.md` are updated to
`resolved-study` with a link to this file as part of the same
session that produced the study (per the Wave 1 session loop,
steps 9–10).
