<!-- path: studies/decisions/2026-05-25-b2-7-documentation-site-generator.md -->

# B2-7 — Documentation Site Generator

## Context

The repository carries a substantial documentation corpus
under `docs/`:

- 37 ADRs under `docs/adr/`
- Four operator runbooks under `docs/runbooks/`
- Two security notes under `docs/security/`
- One developer guide under `docs/dev/` (the local-testing
  guide from ADR-0034)
- A glossary, a governance document, and a top-level
  `docs/README.md` index
- Plus the repository-root `CONTRIBUTING.md` (ADR-0011 §C4
  + W3-P8c) and the foundation documents under `studies/`
  (governed by R8, not part of the published doc product)

Today the rendering surface is **GitHub's web UI**:
contributors land on a markdown file, GitHub renders it,
relative links resolve to other markdown files, and search
is the repo-search box ("/" hotkey). AI agents working in
this repository navigate via the filesystem and follow
path references from `CLAUDE.md` and the path-header
comment in each file (per R6).

B2-7 was registered at the W3 backlog-numbering step with
the question:

> Does `docs/` get a static site generator, or stay as raw
> markdown? Affects how documentation is discovered by
> non-developers.

The "non-developers" framing was speculative — at this
ADR's writing the documentation has no committed
non-developer audience. Platform-team contributors, code
reviewers, and AI agents are the documented readers; an
external operator, partner-org reviewer, or regulator
audience is hypothetical. R3 (don't revisit settled
architecture without strong cause) bites here from the
other direction: adding a site-generator substrate ahead
of a concrete audience signal is choosing infrastructure
to solve a problem the project does not yet have.

The principles bearing on the decision are **P4** (cost is
a first-class constraint — adding doc-publishing
infrastructure adds CI lanes, hosting, theme maintenance,
and a drift surface between source and rendered site;
none of those costs are free), **P6** (borrow patterns,
not baggage — adopting a doc-site framework imports its
plugin ecosystem, version-bump cadence, and theme
posture, all of which require ongoing maintenance), and
**R3** (do not revisit settled architecture — ADR-0011
committed the documentation language; this ADR is the
adjacent documentation-infrastructure follow-up and
preserves ADR-0011 without amendment).

What B2-7 must commit:

1. **The current-state deliverable** — what does `docs/`
   ship as?
2. **The current-state navigation surface** — how do the
   committed audiences (platform-team contributors, code
   reviewers, AI agents) find documents today?
3. **The auditable trigger conditions** under which the
   deferral revisits — what observable signals would
   justify adopting a site generator later?
4. **The migration posture** — if the trigger conditions
   are met, what does the migration look like, and what
   shape does the markdown source need to be in today
   so a future migration is mechanical rather than a
   rewrite?

---

## Decision Drivers

- **DD-1 — Today's audience is platform-team-internal.**
  The committed audiences for `docs/` are platform-team
  contributors editing the artefacts, code reviewers on
  PRs, and AI agents navigating via path references. No
  external audience (onboarded entity owners outside the
  platform team, partner-org reviewers, external operators
  reading runbooks during an incident, regulators
  reviewing posture documents) is documented at this ADR's
  writing.

- **DD-2 — Adding infrastructure ahead of a concrete
  consumer violates P4.** A static-site generator adds:
  (a) a CI lane (build + publish), (b) a hosting surface,
  (c) a theme + plugin maintenance burden, (d) a drift
  risk between markdown source and rendered HTML, and
  (e) a parallel addressing scheme (site URL vs. file
  path). These costs are paid up-front; the benefits
  (sidebar nav, full-text search, branded URLs) accrue
  only when an audience exists that the GitHub-rendered
  surface fails.

- **DD-3 — The current rendering surface is environment
  substrate.** GitHub renders markdown with relative-link
  resolution, fenced-code highlighting, and Mermaid
  diagram support; offers full repository search; and
  serves every file under stable `blob/<sha>/<path>` URLs
  (immutable per commit) and `blob/main/<path>` URLs
  (current). These are commodity capabilities of the
  hosting substrate, not a deferred decision the platform
  carries.

- **DD-4 — AI agents today address documents by path, not
  by URL.** This is an empirical observation about how
  the repository's agent surface currently navigates —
  not a contract committed elsewhere, so it is **new
  contribution proposed here, requires review**. The
  evidence: `CLAUDE.md` references `studies/foundation/...`
  and `docs/adr/...` via filesystem paths; every produced
  file carries an HTML path-header comment per R6;
  relative links inside markdown files use repository-
  relative paths. Adding a site-URL addressing scheme
  layered on top would create two ways to refer to the
  same document, increasing drift risk without an
  AI-agent-facing benefit. If a future ADR commits a
  URL-addressing contract for agents, this DD revisits.

- **DD-5 — Corpus size is below the threshold where
  flat numbered navigation stops being scannable.** The
  documentation corpus is 48 markdown files (37 ADRs +
  11 other published documents). At this scale, the
  flat ADR index ordered by number plus the
  `docs/README.md` state listing remains scannable
  within a single editor screen at typical density —
  the platform team's daily navigation surface does not
  scroll past the index. The trigger condition below
  expresses the revisit threshold qualitatively as
  "the `docs/README.md` state listing no longer fits a
  single editor screen at typical density" rather than
  as a round-number ADR count, because the round number
  would be soft preference dressed as a hard threshold.
  The qualitative signal is the observable one.

- **DD-6 — The deferral must be auditable.** The standing
  pattern in the repository (ADR-0030 manifest crypto
  posture, ADR-0033 scheduler catchup) is to defer
  posture decisions with **concrete trigger conditions**
  that an observer can check independently. This ADR
  follows the same shape: deferral is the recommendation,
  the trigger conditions are explicit, and a future
  observer reading just the ADR can verify whether any
  trigger has fired without re-deriving the decision.

- **DD-7 — Markdown source must stay migration-friendly.**
  Even though the recommendation is "no site generator
  now", the markdown source on disk should remain in a
  shape that a future site generator can consume
  mechanically. Two source-level conventions support
  this: (a) every markdown file carries a path-header
  comment (already enforced by R6); (b) cross-document
  links use repository-relative paths from the source
  file (already the convention). No additional
  source-level discipline is committed by this ADR.

---

## Considered Options

### Option 1 — Raw markdown is the deliverable; no site generator (recommended)

`docs/` stays as the published deliverable. The rendering
surface is GitHub's web UI plus IDE viewers plus the
markdown rendering in AI coding agents. Navigation is the
existing `docs/README.md` index, the numbered ADR
sequence, and the directory structure (`adr/`, `runbooks/`,
`security/`, `dev/`).

Search is the GitHub repo-search box for the platform-team
audience; AI agents grep the filesystem directly.
Cross-document links are repository-relative paths.

The recommendation includes an **auditable trigger-
conditions table** (below) under which the deferral
revisits. When any single trigger fires, a successor ADR
re-evaluates Option 2.

**Strengths.** Zero infrastructure cost; zero CI surface
expansion; zero drift risk between source and render;
preserves the path-based addressing AI agents already use;
honors R3 (no settled-architecture revisit) and P4
(cost-first).

**Trade-offs.** Non-developer discovery via URL-stable
nav is not available. Today this is hypothetical (DD-1);
when the audience materializes the trigger conditions
fire and Option 2 lands as an additive change.

### Option 2 — Adopt a static-site generator now

Pick a markdown-in-HTML-out static-site generator (a
documentation-site builder that consumes the existing
`*.md` files, produces a navigable HTML site with sidebar
nav and full-text search, and publishes to a hosting
surface). Add a CI lane that builds and publishes on every
push to `main`.

**Strengths.** Sidebar navigation flattens the discovery
gap when the corpus crosses the single-screen threshold
DD-5 names. Full-text search becomes meaningfully better
than the substrate's repo search when contributors query
cross-document phrases ("which ADR commits the
pointer-as-mutable-control-plane invariant?") instead of
keyword tokens. Stable public URLs unlock the linking
shape an external onboarding guide or partner-org
document needs to reference a specific paragraph of a
runbook without pinning a git SHA. The hypothetical
audiences DD-1 names — onboarded entity owners outside
the platform team, partner-org reviewers, external
operators reading runbooks during shared incidents — are
the consumers each strength addresses; when those
audiences materialize, the strengths convert from
hypothetical to load-bearing.

**Trade-offs.** Pays the infrastructure cost (DD-2)
before the strengths' consumers exist (DD-1).
Introduces a drift surface between markdown source and
rendered site (front-matter, sidebar config, or per-page
generator-specific metadata depending on the generator
chosen — the exact drift shape varies, but the surface
exists for every generator). Adds a new substrate the
platform team maintains: version pins, theme updates,
plugin compatibility, build-failure triage. Creates a
parallel addressing scheme (site URL vs. file path) that
AI agents do not benefit from per DD-4 unless and until
the agent-addressing contract changes. Rejected as
premature: the strengths' consumers are the same
hypothetical audiences whose absence justifies the
deferral; adopting now pays the costs and waits for the
audience.

### Option 3 — Hybrid: small TOC-generator script, no site

Keep markdown as the deliverable, but add a small script
(under `tools/` or `scripts/`) that mechanically
regenerates `docs/README.md`'s state listing from the
`docs/adr/` filename inventory. The script runs in CI; a
CI gate fails when the README is out of sync.

**Strengths.** Small cost, no parallel addressing scheme,
auto-refreshes the index, removes the manual `docs/README.md`
edits each ADR currently requires.

**Trade-offs.** Solves a problem that does not yet hurt
— manual `docs/README.md` updates today are one line per
ADR landing, performed in the same PR. The pre-commit
hook + reviewer attention already catches misalignment.
Adding a regenerator + a CI gate solves the symptom of a
larger problem (corpus growing past a manual-update
threshold) without solving the larger problem. The same
threshold from DD-5 (≥100 ADRs) is when manual updates
genuinely strain; adopting the regenerator at that point
ships under the same trigger conditions Option 2 also
trips on. Rejected as a half-step that does not bridge
to a useful endpoint.

---

## Recommendation

**Option 1.** Raw markdown is the deliverable; no site
generator now.

### Current-state deliverable

`docs/` is the published deliverable. Every file is
markdown; every file carries an HTML path-header comment
per R6; every file uses repository-relative paths in
cross-document links. The rendering surface is the
substrate-provided HTML viewer (GitHub web UI in
particular, IDE markdown viewers and AI-agent markdown
rendering in general).

### Current-state navigation surfaces

The three audience-by-surface combinations are:

| Audience | Navigation surface |
|---|---|
| Platform-team contributors editing artefacts | Filesystem (IDE) + git history + the `docs/README.md` index |
| Code reviewers on PRs | GitHub PR diff view + the `docs/README.md` index + repo search ("/") |
| AI coding agents | Filesystem path references from `CLAUDE.md` + path-header comments per R6 |

No audience currently requires URL-stable, externally-
accessible navigation. When that changes, the trigger
conditions below fire.

### Auditable trigger conditions

A successor ADR re-evaluates Option 2 when **any one** of
the following observable conditions is met:

1. **External-audience commitment.** A non-platform-team
   audience is committed as a documented reader of
   `docs/`. Examples: onboarded entity owners outside
   the platform team for runbook access; partner-org
   reviewers for security/governance notes; external
   operators reading runbooks during shared incidents;
   regulators reviewing posture documents. The trigger
   is the *commitment*, not informal request — i.e., a
   committed decision-log row that names the audience,
   not a one-off question from one external reader.

2. **Navigation-density threshold.** The
   `docs/README.md` state listing no longer fits a single
   editor screen at typical density — i.e., a
   contributor opening the index file must scroll to
   reach the most recent entries. This is the
   observable signal that the flat numbered ADR
   sequence has crossed the threshold where a
   categorized sidebar would discretely beat
   filesystem navigation. The signal is qualitative
   (per DD-5) but observable by anyone opening the
   file; no round-number threshold is committed.

3. **Documented search-quality friction.** A Wave
   retrospective or comparable platform-team review
   surfaces sustained complaints about repo-search
   recall for typical contributor questions (e.g.,
   "find the ADR that committed X"). The trigger is
   the retrospective surfacing, not the individual
   complaint — sustained friction registered in a
   review forum, not Slack grumbling.

4. **External link-out demand.** Another committed
   surface (an external onboarding guide, a partner-org
   doc, a public-facing engineering blog) requires
   linking to specific paragraphs of `docs/` documents
   with stable URLs. The trigger is the *commitment* of
   the linking surface, not a casual mention.

A future observer reading just this ADR can check each
condition against the current decision-log + the ADR
count + the search-friction backlog without re-deriving
the rationale.

### Migration posture (if a trigger fires)

When any trigger fires, the migration is mechanical
because the source-level conventions already support it:

1. Every markdown file carries a path-header comment per
   R6 — usable directly as a canonical-path metadata
   field by any generator that needs one.
2. Cross-document links use repository-relative paths
   from the source file — translatable to site-relative
   URLs by a one-pass rewrite during build.
3. No file relies on GitHub-specific extensions beyond
   what generic markdown supports (fenced code, tables,
   relative links, Mermaid where used — and Mermaid is
   widely supported across generators).

The successor ADR that lands when a trigger fires picks
the generator and the hosting surface, commits the new
CI lane, and addresses the front-matter migration (if
the chosen generator requires it). No source rewrite
should be needed.

### Why this does not reopen ADR-0011

ADR-0011 committed the documentation language (English
for technical artefacts, Portuguese permitted for
onboarding guides with a language marker). This ADR is
about the *publication infrastructure*, not the
language. ADR-0011 stays accepted without amendment;
any future site generator inherits the language posture
unchanged.

### Why this does not reopen ADR-0019

ADR-0019 committed the infrastructure tooling for the
*deploy* surface (Kustomize for Kubernetes overlays).
This ADR commits the infrastructure posture for the
*documentation* surface (no generator). The two ADRs
address distinct concerns; ADR-0019 stays accepted
without amendment.

---

## Consequences

1. **`docs/` ships as markdown.** The rendering surface
   stays the substrate-provided markdown viewer (GitHub
   web UI, IDE viewers, AI-agent rendering). No build
   step transforms markdown into HTML.

2. **No new CI lane.** The repository's CI surface
   (byte-equality, make lint, make test, make
   validate-deploy) is unchanged. No "docs-publish" lane
   is added; no hosting surface is provisioned.

3. **`docs/README.md` remains the index.** The
   state-listing convention continues — each ADR landing
   adds one line to the state listing in the PR that
   lands the ADR. Manual upkeep is acceptable at the
   current corpus size and is itself one of the trigger
   conditions (DD-5 + trigger #2) under which automation
   becomes warranted.

4. **AI-agent path-based addressing stays the only
   addressing scheme.** No site URL is introduced. Path
   references from `CLAUDE.md` and inside markdown files
   continue to use repository-relative paths.

5. **The trigger-conditions table is auditable.** A
   future contributor or reviewer can check each of the
   four triggers against the current state of the
   repository without consulting this ADR's authors.
   The deferral has a documented escape valve.

6. **Source-level migration-friendliness is preserved
   without new discipline.** The path-header comment per
   R6 and the repository-relative-link convention
   already in place are sufficient. No additional
   front-matter, sidebar configuration, or generator-
   specific marker is committed.

7. **ADR-0011 and ADR-0019 are preserved.** This ADR is
   adjacent to both: ADR-0011 commits the doc language,
   ADR-0019 commits the deploy-tooling posture, and this
   ADR commits the doc-publication posture. The three
   together form the documentation-infrastructure
   surface of the platform.

8. **B2-7 closes.** The decision-log B2-7 row moves to
   `resolved-adr` (→ ADR-0038). The deferral with
   triggers means the row stays closed; a future ADR
   reopens the *infrastructure choice* (Option 2) if any
   trigger fires, but does not reopen this ADR's
   "deferral was correct at 2026-05-25" judgment.

9. **Implementation note.** If a trigger fires and a
   future ADR adopts Option 2, the `docs/README.md`
   index becomes a build artefact of the site generator
   rather than a hand-maintained file. The transition
   is mechanical (regenerate from `docs/adr/` filename
   inventory) and does not require rewriting prior ADRs.

---

## Open Questions

None blocking.

One deferred posture surfaced during drafting and is
explicitly **out-of-scope for current cycle**:

- **OQ-1: ADR categorization vs. flat numbering.** The
  ADR sequence is flat-numbered (0001..NNNN); a future
  site generator (if a trigger fires, particularly the
  navigation-density threshold #2) might benefit from a
  categorization layer (e.g., "compatibility ADRs",
  "runtime ADRs", "governance ADRs") in addition to the
  numeric sequence. Reserved until that trigger fires;
  the numbering itself is the canonical identifier and
  does not change.

---

## Promotion target

`docs/adr/0038-documentation-site-generator.md` — next
free ADR number. Ships the "raw markdown is the
deliverable; no site generator now" commitment, the four
auditable trigger conditions, and the mechanical-
migration posture.
