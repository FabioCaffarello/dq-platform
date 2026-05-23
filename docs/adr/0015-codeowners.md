<!-- path: docs/adr/0015-codeowners.md -->

# ADR-0015 — CODEOWNERS Review-Ownership Map

- **Status:** accepted
- **Date:** 2026-05-23

---

## Context

Three prior ADRs commit review-ownership obligations that the
repository must enforce at PR-merge time, not by convention:

- **ADR-0001** commits that the schema source
  (`engine/internal/dsl/schema/v<N>.schema.json`) and its
  byte-equal mirror (`rules/_schema/v<N>.schema.json`) are
  CODEOWNERS-protected under a single schema-owner group; the
  asymmetric review model that lets domain teams edit per-entity
  rule YAMLs without platform-team approval depends on the
  schema half staying platform-owned.
- **ADR-0006** commits that every `owner` string in
  `_owners.yaml` must resolve to a CODEOWNERS group ("no alert
  without owner"). The linter rule is the first enforcement
  point; CODEOWNERS-routed review of `_owners.yaml` is the
  second. Engine deployment config carries the additional
  per-environment review depth "platform team plus SRE".
- **ADR-0009** commits that `CLAUDE.md` is the authoritative
  source of the multi-agent contract and `AGENTS.md` /
  `.codex/AGENTS.md` are thin pointers kept aligned via
  `/sync-agents`. Splitting CODEOWNERS across the three files
  would let a pointer drift be approved without the canonical
  contract being touched.

This ADR commits the four review-ownership elements not directly
fixed by those ADRs: the CODEOWNERS file's location, the group
inventory that satisfies the three obligations jointly, the
path-rule table that maps repository paths to review groups, and
the substitution protocol that exchanges the file's placeholder
identifiers for the real GitHub-org slug.

**Out of scope of this ADR:**

- The concrete GitHub-org identifier and team slugs (e.g.,
  `@dq-platform/platform-team` vs. `@acme/dq-platform-team`).
  The chosen Git host is GitHub per ADR-0008, but the production
  org identifier is a host-primitive follow-up that resolves
  operationally. This ADR commits the shape of the file; the
  literal identifiers are exchanged mechanically in the
  operational session that creates the org.
- Whether `/sync-agents` becomes a CI gate that fails when
  pointer files diverge from `CLAUDE.md`. CODEOWNERS plus the
  same-MR-propagation convention is the enforcement surface
  until that gate exists.
- Defense-in-depth `_owners.yaml` validation at the manifest
  publisher and engine loader, beyond the linter's first-line
  check. A future ADR extends the publisher and loader
  verification sets.
- Per-entity CODEOWNERS rules (e.g., `/rules/customer.yaml
  @customer-team`). The current shape delegates per-entity
  refinement to `_owners.yaml`; per-entity CODEOWNERS lines
  are additive and do not reopen this ADR.

---

## Decision

### 1. File location — `/.github/CODEOWNERS`

The CODEOWNERS file lives at `/.github/CODEOWNERS`. Two
alternatives are rejected: `/CODEOWNERS` mixes governance
metadata with workspace content at the top level; `/docs/CODEOWNERS`
scatters the governance surface across `/.github/` (CI gates)
and `/docs/` (review rules), leaving contributors landing on
`/.github/` looking for review rules with no signal that they
exist elsewhere.

Co-locating the file under `/.github/` clusters the governance
surface — branch-protection metadata, CI workflows, and the
review-ownership map — in one directory.

### 2. Group inventory — three groups

The review-ownership map uses exactly three groups. Each
satisfies a distinct obligation from the prior ADRs.

| Group placeholder | Responsibility |
|---|---|
| `@PLACEHOLDER-org/platform-team` | Default owner for everything not explicitly delegated: engine runtime, schema source and mirror, tools, deploy base, the `local` deploy overlay, docs (ADRs and non-ADRs), CI workflows, root build configuration, the multi-agent contract surface, and historical reasoning under `/studies/`. Source-of-truth authority for the asymmetric review model and the multi-agent contract. |
| `@PLACEHOLDER-org/sre` | Co-owner of `/deploy/overlays/qa/` and `/deploy/overlays/prod/`. Implements the per-environment review depth "platform team plus SRE" committed by ADR-0006. |
| `@PLACEHOLDER-org/rules-authors` | Per-entity rule YAMLs under `/rules/` (excluding `/rules/_schema/`, `/rules/_owners.yaml`, and `/rules/_owners.schema.json`). Implements the asymmetric review model from ADR-0001: domain teams edit per-entity rules without platform-team approval, but schema and owner metadata remain platform-owned. Also the CODEOWNERS-group value that the per-entity `owner` field in `_owners.yaml` references until per-entity refinement lands as an additive change. |

A single-group inventory is rejected: it removes the asymmetric
review depth for `/rules/` and removes the SRE review depth on
production overlays. A per-domain-team enumeration is rejected
as the initial shape: only one domain (`customer`) is onboarded
today, so enumerating speculative teams would create groups
with no members and review rules with no reviewers. Per-domain
enumeration is additive when a second domain team onboards;
the additive change is a single CODEOWNERS line per entity and
does not reopen this ADR.

### 3. Path-rule table — committed contents of `/.github/CODEOWNERS`

The file is ordered general-to-specific so that
**last-match-wins** (GitHub's CODEOWNERS evaluation order) puts
overrides last. Every path uses a leading `/` to anchor at the
repository root (GitHub's `/path/` syntax means "this exact
directory from the repository root"; without the leading slash
the pattern matches anywhere in the tree).

The exact contents of `/.github/CODEOWNERS`:

```
# /.github/CODEOWNERS
#
# Authoritative review-ownership map for dq-platform.
# Format: GitHub CODEOWNERS — last matching rule wins.
# Group identifiers are placeholders pending the ADR-0008
# host-primitive follow-up; substitution lands in the
# operational session that creates the production org.

# --- Default ------------------------------------------------------
*                                       @PLACEHOLDER-org/platform-team

# --- Engine workspace ---------------------------------------------
/engine/                                @PLACEHOLDER-org/platform-team

# --- Tools workspace ----------------------------------------------
/tools/                                 @PLACEHOLDER-org/platform-team

# --- Rules workspace (asymmetric per ADR-0001) --------------------
# Default: domain teams own per-entity rule YAMLs.
/rules/                                 @PLACEHOLDER-org/rules-authors

# Override: schema mirror is platform-team only (byte-equal copy
# of engine/internal/dsl/schema/, never hand-edited per ADR-0001).
/rules/_schema/                         @PLACEHOLDER-org/platform-team

# Override: owner metadata is platform-team only (ADR-0006 second
# line of defense — "no alert without owner" enforcement).
/rules/_owners.yaml                     @PLACEHOLDER-org/platform-team
/rules/_owners.schema.json              @PLACEHOLDER-org/platform-team

# --- Deploy workspace (per-environment depth per ADR-0006) --------
/deploy/                                @PLACEHOLDER-org/platform-team
/deploy/base/                           @PLACEHOLDER-org/platform-team
/deploy/overlays/local/                 @PLACEHOLDER-org/platform-team
/deploy/overlays/qa/                    @PLACEHOLDER-org/platform-team @PLACEHOLDER-org/sre
/deploy/overlays/prod/                  @PLACEHOLDER-org/platform-team @PLACEHOLDER-org/sre

# --- Docs ---------------------------------------------------------
# /docs/adr/ kept as an explicit line (same owner as /docs/) so a
# future ADR-specific reviewer group can be slotted in without
# editing /docs/.
/docs/                                  @PLACEHOLDER-org/platform-team
/docs/adr/                              @PLACEHOLDER-org/platform-team

# --- Multi-agent contract (ADR-0009) ------------------------------
# Single authorship for CLAUDE.md (source) and the two pointers,
# so /sync-agents drift cannot be approved without the canonical
# contract being touched.
/CLAUDE.md                              @PLACEHOLDER-org/platform-team
/AGENTS.md                              @PLACEHOLDER-org/platform-team
/.claude/                               @PLACEHOLDER-org/platform-team
/.codex/                                @PLACEHOLDER-org/platform-team

# --- CI + branch protection (ADR-0001, ADR-0008) ------------------
# /.github/workflows/ and /.github/CODEOWNERS kept as explicit
# overrides (same owner as /.github/) so a future workflows-only
# or CODEOWNERS-only reviewer group can be slotted in without
# editing /.github/.
/.github/                               @PLACEHOLDER-org/platform-team
/.github/workflows/                     @PLACEHOLDER-org/platform-team
/.github/CODEOWNERS                     @PLACEHOLDER-org/platform-team

# --- Root build/runtime config ------------------------------------
/Makefile                               @PLACEHOLDER-org/platform-team
/go.work                                @PLACEHOLDER-org/platform-team
/go.work.sum                            @PLACEHOLDER-org/platform-team
/docker-compose.yml                     @PLACEHOLDER-org/platform-team
/scripts/                               @PLACEHOLDER-org/platform-team

# --- Reasoning history --------------------------------------------
/studies/                               @PLACEHOLDER-org/platform-team
```

### 4. Substitution protocol

The placeholder slug `PLACEHOLDER-org/` is unambiguous (no real
GitHub org matches it) so a partially-substituted state is loud
rather than silently broken. The operational session that
creates the production GitHub org per the ADR-0008 host-primitive
follow-up performs a mechanical search-and-replace of
`PLACEHOLDER-org/` with the real org slug across:

- `/.github/CODEOWNERS`
- engine environment-config files for `qa` and `prod`
- deploy overlays for `qa` and `prod`

The substitution is one atomic edit per file. Shape changes
during substitution are forbidden — they would reopen this ADR.

---

## Consequences

1. The schema mirror is review-protected by the same group as
   the source. A merge request that edits the schema source but
   not the schema mirror fails the byte-equality CI gate
   committed by ADR-0001 **and** requires the same review group
   on both files — there is no asymmetric-approval path through
   which one half can land without the other.

2. `_owners.yaml` review is platform-team-only. A new entity
   onboarded via a per-entity rule YAML edit by a domain team
   cannot ship without a paired `_owners.yaml` edit, which the
   linter rejects per ADR-0006 and which CODEOWNERS routes to
   platform-team review. "No alert without owner" is enforced
   twice: at lint time and at review time.

3. Per-entity ownership refinement is `_owners.yaml`'s job,
   not CODEOWNERS's. CODEOWNERS commits at the workspace-default
   level (`@rules-authors` for `/rules/`). Per-entity routing
   is resolved at runtime by the alerting consumer via
   `_owners.yaml`'s `owner` field per ADR-0006. Adding a second
   domain team does not require editing CODEOWNERS; it requires
   adding the group to the GitHub org and referencing it from
   `_owners.yaml`.

4. Until per-entity CODEOWNERS refinement lands as an additive
   change, the per-entity `owner` field in `_owners.yaml`
   references `@PLACEHOLDER-org/rules-authors`. ADR-0006 commits
   that `owner` strings match a CODEOWNERS group; until a
   second group exists at `/rules/` scope, `@rules-authors`
   is the only group the linter check can resolve. The
   operational session that publishes `/.github/CODEOWNERS`
   adjusts `_owners.yaml` in the same change so the literal
   substitution stays consistent across both files.

5. Production overlay edits cannot land without SRE review.
   `/deploy/overlays/qa/` and `/deploy/overlays/prod/` carry
   per-environment secrets and GCP-IAM annotations; their
   CODEOWNERS lines list both `@platform-team` and `@sre`, so
   GitHub branch-protection's "review from CODEOWNERS"
   requirement gates them on both groups. The `local` overlay
   does not — it carries no production secret and no production
   substrate.

6. The multi-agent contract surface drifts under platform-team
   review only. `CLAUDE.md`, `AGENTS.md`, `.codex/AGENTS.md`,
   and the playbooks under `.claude/` cannot be edited without
   platform-team review. `/sync-agents` is the mechanism that
   keeps the pointer files aligned with `CLAUDE.md` per
   ADR-0009; CODEOWNERS is the human checkpoint that ensures
   the propagation is reviewed.

7. The linter version pin is platform-team-owned.
   `/.github/workflows/` carries the CI lanes that pin the
   linter binary's digest per ADR-0001; the workflow file is
   under platform-team CODEOWNERS, so the pin cannot be
   changed without the same review depth as the schema itself.

8. `PLACEHOLDER-org/` is intentional. The literal sits in the
   committed file until the operational substitution session
   runs. The placeholder text is unambiguous (no real GitHub
   org matches `PLACEHOLDER-org`), so a partially-deployed
   state surfaces visibly rather than silently broken.

9. The CODEOWNERS file itself is platform-team-owned. Editing
   the path-rule table is a platform-team decision, gated on
   this ADR (or its successor). Reopening the shape requires a
   new ADR.

10. `/studies/` is platform-team-owned. Reasoning history is
    protected from domain-team edits even though the documents
    inside are not code; amendments to historical reasoning go
    through the same review group as the ADRs that supersede
    them.

11. The three-group inventory is the only inventory the file
    enforces. Adding a per-domain group is additive (one
    CODEOWNERS line per entity) and does not require this ADR
    to be amended. Removing a group, or changing the review
    depth of `/deploy/overlays/{qa,prod}/`, does.

---

## Notes

- The path-rule table commits a deliberate choice to keep
  `/docs/adr/`, `/.github/workflows/`, and `/.github/CODEOWNERS`
  as explicit lines that duplicate the same owner as their
  parent directory. The duplication exists so a future
  refinement (an ADR-specific reviewer group, a CI-pipeline
  reviewer group, or a CODEOWNERS-meta reviewer group) is a
  single-line edit rather than a re-shape of the parent.

- The placeholder discipline (`PLACEHOLDER-org/...` as a loud
  literal substituted operationally) mirrors the same pattern
  used elsewhere in the repository for production-environment
  identifiers that are not yet committed. The substitution
  procedure is intentionally mechanical so it can be reviewed
  as a single atomic change rather than as a contract
  amendment.

- Deferred items captured here are explicit follow-ups, not
  ambiguities: the concrete GitHub-org identifier, the
  `/sync-agents` CI-gate question, the publisher/loader-side
  defense-in-depth `_owners.yaml` validation, the per-entity
  CODEOWNERS refinement, and CODEOWNERS rules for repo-level
  governance lanes that do not yet exist (e.g.,
  `secret_scanning`, `dependency_review`).
