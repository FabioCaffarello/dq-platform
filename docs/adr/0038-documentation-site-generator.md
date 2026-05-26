<!-- path: docs/adr/0038-documentation-site-generator.md -->

# ADR-0038 — Documentation Site Generator (Deferred with Triggers)

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

The repository ships a substantial documentation corpus
under `docs/`: 37 ADRs, four operator runbooks, two
security notes, one developer guide, the glossary, the
governance document, and the top-level `docs/README.md`
index — 48 markdown files in total at this ADR's writing,
plus the repository-root `CONTRIBUTING.md` per ADR-0011
§C4.

The rendering surface today is the hosting substrate's
markdown viewer: GitHub renders markdown with relative-
link resolution, fenced-code highlighting, and Mermaid
diagram support; offers repository-wide search; and
serves every file under stable `blob/<sha>/<path>` URLs
(immutable per commit) and `blob/main/<path>` URLs
(current). IDE viewers and AI-agent rendering surfaces
consume the same source files.

The committed audiences for `docs/` are platform-team
contributors editing the artefacts, code reviewers on
PRs, and AI coding agents navigating via the path
references that `CLAUDE.md` and the per-file path-header
comments (per R6) anchor on. No external audience —
onboarded entity owners outside the platform team,
partner-org reviewers, external operators reading
runbooks during shared incidents, regulators reviewing
posture documents — is committed at this ADR's writing.

[ADR-0011](./0011-documentation-language.md) committed
the documentation *language* (English for technical
artefacts, Portuguese permitted for onboarding guides
with a language marker). [ADR-0019](./0019-infrastructure-tooling.md)
committed the infrastructure tooling for the *deploy*
surface (Kustomize for Kubernetes overlays). What neither
ADR committed was the *publication-infrastructure*
posture for the documentation surface itself: does
`docs/` ship as raw markdown, or as a generated site
(HTML produced by a markdown-in / HTML-out static-site
generator, with sidebar navigation, full-text search,
and stable public URLs)?

The principles bearing on the decision are **P4** (cost
is a first-class constraint — a site generator adds a
CI lane, a hosting surface, a theme + plugin
maintenance burden, and a drift surface between
markdown source and rendered HTML; none of those costs
are free), **P6** (borrow patterns, not baggage —
adopting a doc-site framework imports its plugin
ecosystem, version-bump cadence, and theme posture, all
of which require ongoing maintenance), and **R3** (do
not revisit settled architecture — ADR-0011 and
ADR-0019 stay accepted without amendment; this ADR is
adjacent, not amending).

---

## Decision

**Raw markdown is the documentation deliverable. No
static-site generator is adopted at this ADR's writing.**
The deferral is auditable: four observable trigger
conditions are committed below; when any single trigger
fires, a successor ADR re-evaluates adoption.

### Current-state deliverable

`docs/` is the published deliverable. Every file is
markdown; every file carries an HTML path-header comment
per R6; every file uses repository-relative paths in
cross-document links. The rendering surface is the
substrate-provided markdown viewer (GitHub web UI in
particular, IDE markdown viewers and AI-agent markdown
rendering in general). No build step transforms markdown
into HTML for the documentation corpus.

### Current-state navigation surfaces

Three audience-by-surface combinations are committed:

| Audience | Navigation surface |
|---|---|
| Platform-team contributors editing artefacts | Filesystem (IDE) + git history + the `docs/README.md` index |
| Code reviewers on PRs | GitHub PR diff view + the `docs/README.md` index + repo search |
| AI coding agents | Filesystem path references from `CLAUDE.md` + path-header comments per R6 |

No audience currently requires URL-stable, externally-
accessible navigation. When that changes, the trigger
conditions below fire.

### Auditable trigger conditions

A successor ADR re-evaluates adoption of a static-site
generator when **any one** of the following observable
conditions is met:

1. **External-audience commitment.** A non-platform-team
   audience is committed as a documented reader of
   `docs/` — captured as a decision-log row that names
   the audience and the consumed subset (e.g., runbooks
   for external operators during shared incidents,
   security notes for partner-org reviewers, governance
   for regulators). The trigger is the *commitment*, not
   informal request — a one-off question from one
   external reader does not fire the trigger.

2. **Navigation-density threshold.** The
   `docs/README.md` state listing no longer fits a
   single editor screen at typical density — a
   contributor opening the index file must scroll to
   reach the most recent entries. This is the
   observable signal that the flat numbered ADR
   sequence has crossed the threshold where a
   categorized sidebar would discretely beat filesystem
   navigation. The signal is qualitative but observable
   by anyone opening the file; no round-number ADR
   count is committed.

3. **Documented search-quality friction.** A Wave
   retrospective or comparable platform-team review
   surfaces sustained complaints about substrate
   repo-search recall for typical contributor questions
   (e.g., "find the ADR that committed X"). The
   trigger is the retrospective surfacing, not isolated
   Slack/PR-comment complaints — sustained friction
   registered in a review forum.

4. **External link-out demand.** Another committed
   surface (an external onboarding guide, a partner-org
   document, a public-facing engineering blog)
   requires linking to specific paragraphs of `docs/`
   documents with stable URLs that are not git-SHA-
   pinned. The trigger is the *commitment* of the
   linking surface, not a casual mention.

A future observer reading just this ADR can check each
condition against the current decision-log + the
`docs/README.md` index + the retrospective backlog
without re-deriving the rationale.

### Migration posture if a trigger fires

When any trigger fires, the migration is mechanical
because the source-level conventions already support it:

1. Every markdown file carries a path-header comment per
   R6 — usable directly as a canonical-path metadata
   field by any generator that needs one.
2. Cross-document links use repository-relative paths
   from the source file — translatable to site-relative
   URLs by a one-pass rewrite during build.
3. No file relies on substrate-specific extensions
   beyond what generic markdown supports (fenced code,
   tables, relative links, Mermaid where used — Mermaid
   is widely supported across generators).

The successor ADR that lands when a trigger fires picks
the generator and the hosting surface, commits the new
CI lane, and addresses any front-matter migration (if
the chosen generator requires it). No source rewrite is
expected.

### Why this does not reopen ADR-0011

ADR-0011 committed the documentation language. This ADR
is about the publication infrastructure, not the
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
   validate-deploy) is unchanged. No "docs-publish"
   lane is added; no hosting surface is provisioned.

3. **`docs/README.md` remains the index.** The
   state-listing convention continues — each ADR
   landing adds one line in the PR that lands the ADR.
   Manual upkeep is acceptable at the current corpus
   size; the navigation-density trigger above is the
   point at which automation becomes warranted.

4. **AI-agent path-based addressing stays the only
   addressing scheme.** No site URL is introduced. Path
   references from `CLAUDE.md` and inside markdown
   files continue to use repository-relative paths.

5. **The trigger-conditions table is auditable.** A
   future contributor or reviewer can check each of
   the four triggers against the current state of the
   repository without consulting this ADR's authors.
   The deferral has a documented escape valve.

6. **Source-level migration-friendliness is preserved
   without new discipline.** The path-header comment
   per R6 and the repository-relative-link convention
   already in place are sufficient. No additional
   front-matter, sidebar configuration, or generator-
   specific marker is committed.

7. **ADR-0011 and ADR-0019 are preserved.** This ADR is
   adjacent to both: ADR-0011 commits the doc language,
   ADR-0019 commits the deploy-tooling posture, and
   this ADR commits the doc-publication posture. The
   three together form the documentation-infrastructure
   surface of the platform.

8. **B2-7 closes.** The decision-log B2-7 row moves to
   `resolved-adr`. The deferral with triggers means the
   row stays closed; a future ADR reopens the
   *infrastructure choice* if any trigger fires, but
   does not reopen this ADR's "deferral was correct at
   2026-05-25" judgment.

9. **Implementation note.** If a trigger fires and a
   future ADR adopts a site generator, the
   `docs/README.md` index becomes a build artefact of
   that generator rather than a hand-maintained file.
   The transition is mechanical (regenerate from
   `docs/adr/` filename inventory) and does not require
   rewriting prior ADRs.

10. **One deferred posture is registered out-of-scope:**
    ADR categorization vs. flat numbering. The ADR
    sequence is flat-numbered (0001..NNNN); a future
    site generator might benefit from a categorization
    layer (e.g., "compatibility ADRs", "runtime ADRs",
    "governance ADRs") in addition to the numeric
    sequence. Reserved until the navigation-density
    trigger fires; the numbering itself is the
    canonical identifier and does not change.
