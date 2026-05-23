<!-- path: studies/decisions/2026-05-22-b1-9-codeowners.md -->

# B1-9 — CODEOWNERS Finalization

## Metadata

- **B1 reference:** B1-9 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- **Status:** resolved-study (single critique round expected; the
  decision is largely confirmatory of upstream ADRs and commits
  shapes, not concrete identifiers).
- **Last updated:** 2026-05-22
- **Upstream resolved:** ADR-0001
  ([engine ↔ rules compatibility](../../docs/adr/0001-engine-rules-compatibility.md)),
  ADR-0006
  ([alert routing contract](../../docs/adr/0006-alert-routing-contract.md)),
  ADR-0008
  ([git host — GitHub](../../docs/adr/0008-git-host.md)),
  ADR-0009
  ([multi-agent contract](../../docs/adr/0009-multi-agent-contract.md)),
  B1-4
  ([environment configuration model](./2026-05-22-b1-4-environment-configuration-model.md)),
  the sequencing study
  ([Wave 3 phases](./2026-05-21-wave3-sequencing.md)).
- **Downstream:** Wave 3 Phase 8b (Governance) — publishes the
  CODEOWNERS file with concrete identifiers; ADR-0006 §9 linter
  enforcement of `owner ↔ CODEOWNERS group` correspondence.
- **Promotion target:** `docs/adr/0015-codeowners.md`.

---

## Context

ADR-0001 commits that the schema source
(`engine/internal/dsl/schema/v<N>.schema.json`) and its byte-equal
mirror (`rules/_schema/v<N>.schema.json`) are "CODEOWNERS-protected
under the schema owner group" (Consequence 3, Decision §2). ADR-0006
commits that `owner` strings in `_owners.yaml` "match a CODEOWNERS
group" (Decision §1). ADR-0009 commits that `CLAUDE.md` is the
authoritative source and `AGENTS.md` / `.codex/AGENTS.md` are thin
pointers whose drift is caught by `/sync-agents` (Decision §1–§3),
which implies that authorship of all three surfaces must concentrate
in one review group.

Three commitments, three open questions:

1. Which groups exist, by name (shape)?
2. Which paths each group owns, and where the file lives?
3. How are the concrete GitHub-org identifiers (`@org/team`) handled
   before the production org exists?

The decision log row B1-9 has remained `open` because no study had
collected these into one document. Wave 3 Phase 8b (Governance) is
blocked on this resolution per ADR-0013 Consequence 9 (an open B1
row blocking a Wave 3 phase resolves in a separate study session
before the phase proceeds).

This study is deliberately short. The decision is **confirmatory of
upstream commitments and shape-only**: it commits the role
inventory, the path-rule table, and the file location. Concrete
GitHub identifiers are deferred to the operational session in W3-P8b
that creates the production org and substitutes the placeholders —
mirroring the pattern already used by `engine/internal/env/{qa,prod}.go`
PLACEHOLDER markers from W3-P7a.

---

## Decision Drivers

1. **D1. Asymmetric review across engine/schema vs. rules
   (ADR-0001 §C3).** The schema source and the schema mirror are
   platform-owned; per-entity rule YAMLs under `rules/` are
   editable by domain teams. CODEOWNERS must implement the
   asymmetry at path-rule granularity, not by convention.

2. **D2. Owner ↔ group correspondence (ADR-0006 §1).** Every `owner`
   value in `_owners.yaml` must resolve to a CODEOWNERS group. The
   linter rule from ADR-0006 §9 ("no alert without owner") is the
   first enforcement point; CODEOWNERS is the second — review of a
   new `_owners.yaml` entry cannot bypass platform-team approval
   because the file itself is platform-owned.

3. **D3. Single authorship for the agent-contract surface
   (ADR-0009 §1–§3).** `CLAUDE.md`, `AGENTS.md`, and
   `.codex/AGENTS.md` share authorship — rule changes flow
   `CLAUDE.md` → pointers via `/sync-agents`. Splitting CODEOWNERS
   across these three files would let a pointer drift be approved
   without the canonical contract being touched.

4. **D4. Production overlay review depth (ADR-0006 §3).**
   `deploy/overlays/{qa,prod}/` carry secret references, GCP
   service-account annotations, and per-environment substrate
   overrides (substrate isolation per B1-4 MD-3). Their review
   depth must include SRE in addition to platform-team, per
   ADR-0006 §3 ("Owned per environment by the platform team plus
   SRE"). `deploy/overlays/local/` does not (no production
   secret, no production substrate).

5. **D5. Last-match-wins ordering (GitHub CODEOWNERS semantics).**
   GitHub evaluates CODEOWNERS rules top-to-bottom and the **last
   matching rule wins**. The path-rule table must order from general
   to specific so that schema-mirror / `_owners.yaml` / per-env
   overlay overrides take precedence over their parent-directory
   defaults.

6. **D6. Forward-compatibility with the production org
   (ADR-0008).** The chosen Git host is GitHub, but the production
   org identifier is a host-primitive follow-up. The CODEOWNERS
   shape committed here must survive that follow-up unchanged —
   only the `@org/...` literals change.

7. **D7. Deploy workspace split — base vs. overlays.**
   `deploy/base/` (tool-neutral Kubernetes manifests, no per-env
   secrets) is platform-owned; `deploy/overlays/qa/` and
   `deploy/overlays/prod/` add SRE per D4. `deploy/overlays/local/`
   is platform-team only (developer-facing, no production
   exposure).

---

## Considered Options

### CODEOWNERS file location

- **(A) `/CODEOWNERS` at repository root.** GitHub-accepted; visible
  at the root listing. Mixes governance metadata with workspace
  content at the top level.
- **(B) `/.github/CODEOWNERS`.** GitHub-accepted; co-located with
  `.github/workflows/` (the CI lanes from ADR-0008) and any future
  branch-protection metadata. The governance surfaces cluster in
  one directory.
- **(C) `/docs/CODEOWNERS`.** GitHub-accepted; surfaces it as
  documentation. Rejected: it would scatter the governance surface
  across `.github/` (CI gates) and `/docs/` (CODEOWNERS), and
  contributors landing on `.github/` looking for review rules would
  find none.

**Recommended: (B).** Co-locates the governance surface with the
CI lanes that enforce it (ADR-0008 branch protection, ADR-0001 §C2
byte-equality gate).

### Group inventory

- **(α) Single group — `platform-team` only.** Rejected by D1
  (no asymmetric review for domain-team-editable rules) and D4
  (no SRE review depth on prod overlays).
- **(β) Three groups — `platform-team`, `sre`, `rules-authors`.**
  Each maps to a distinct review responsibility. `rules-authors` is
  a placeholder for "any domain team" that owns rule YAMLs;
  per-entity ownership refinement is handled by `_owners.yaml`'s
  `owner` field at runtime, not by per-entity CODEOWNERS rules.
- **(γ) N groups, one per domain team.** Rejected for the initial
  shape: only one domain (`customer`) is onboarded today (W3-P6d);
  enumerating speculative domain teams in CODEOWNERS would create
  groups with no members and review rules with no reviewers.
  Per-entity refinement is additive when the second domain team
  onboards (see OQ-B1-9.4).

**Recommended: (β).** Three groups covers the asymmetric review
model from D1, the per-environment review depth from D4, and the
agent-contract single-authorship from D3, without speculative
per-domain enumeration.

### Concrete identifier handling

- **(I) Concrete `@dq-platform/...` identifiers in the study.**
  Rejected: the production GitHub org name is an ADR-0008
  host-primitive follow-up that has not yet resolved. Hard-coding a
  guess creates drift between the study, the CODEOWNERS file (when
  W3-P8b lands it), and the actual org.
- **(II) `PLACEHOLDER-org/...` literals throughout.** Mirrors the
  pattern used by `engine/internal/env/{qa,prod}.go` (W3-P7a) and
  `deploy/overlays/{qa,prod}/` ServiceAccount annotations (W3-P7c).
  The substitution is a mechanical edit in the operational session
  that creates the real org.

**Recommended: (II).** Documented placeholder; OQ-B1-9.1 tracks the
substitution session.

---

## Recommendation

### File location

The CODEOWNERS file lives at `/.github/CODEOWNERS`. Publication
is deferred to W3-P8b; this study commits the file's contents.

### Group inventory (shape; identifiers deferred)

**New contribution proposed here, requires review.** No upstream
ADR enumerates this specific three-group inventory. ADR-0001 §C3
mentions "the schema owner group" generically; ADR-0006 §3
mentions "platform team plus SRE" for engine deployment config;
nothing names `rules-authors`. The three-group shape below is the
study's commitment to satisfy D1, D3, and D4 jointly.

| Group placeholder | Responsibility | Why it exists |
|---|---|---|
| `@PLACEHOLDER-org/platform-team` | Engine runtime, schema source + mirror, tools, deploy base + local overlay, docs (ADRs + non-ADR), CI workflows, root config, multi-agent surface, studies. | Default owner for everything not explicitly delegated. Source-of-truth authority for the asymmetric review model (D1) and the multi-agent contract (D3). |
| `@PLACEHOLDER-org/sre` | Co-owner of `deploy/overlays/qa/` and `deploy/overlays/prod/`. | Per-environment review depth required by ADR-0006 §3 ("platform team plus SRE"). |
| `@PLACEHOLDER-org/rules-authors` | Per-entity rule YAMLs under `rules/` (excluding `_schema/`, `_owners.yaml`, `_owners.schema.json`). | Implements the asymmetric review model from D1 — domain teams edit their entity rules without platform-team approval, but schema and owner metadata stay under platform-team. Until per-entity refinement under OQ-B1-9.4 lands, this group is also the value that `_owners.yaml`'s `owner` field references for any onboarded entity (e.g., `customer` from W3-P6d) so that the ADR-0006 §9 linter's `owner ↔ CODEOWNERS-group` check resolves. |

### Path-rule table (committed contents of `/.github/CODEOWNERS`)

Ordered general → specific so last-match-wins (D5) puts overrides
last. Every path uses a leading `/` to anchor at the repo root
(GitHub's `/path/` syntax means "this exact directory from the
repo root"; without the leading slash the pattern matches anywhere
in the tree).

```
# /.github/CODEOWNERS
#
# Authoritative review-ownership map for dq-platform.
# Format: GitHub CODEOWNERS — last matching rule wins.
# Group identifiers are placeholders pending ADR-0008 host-primitive
# follow-up; substitution lands in the operational W3-P8b session.

# --- Default ------------------------------------------------------
*                                       @PLACEHOLDER-org/platform-team

# --- Engine workspace ---------------------------------------------
/engine/                                @PLACEHOLDER-org/platform-team

# --- Tools workspace ----------------------------------------------
/tools/                                 @PLACEHOLDER-org/platform-team

# --- Rules workspace (asymmetric per ADR-0001 §C3) ----------------
# Default: domain teams own per-entity rule YAMLs.
/rules/                                 @PLACEHOLDER-org/rules-authors

# Override: schema mirror is platform-team only (byte-equal copy of
# engine/internal/dsl/schema/, never hand-edited per ADR-0001 §2).
/rules/_schema/                         @PLACEHOLDER-org/platform-team

# Override: owner metadata is platform-team only (ADR-0006 §9 second
# line of defense — "no alert without owner" enforcement).
/rules/_owners.yaml                     @PLACEHOLDER-org/platform-team
/rules/_owners.schema.json              @PLACEHOLDER-org/platform-team

# --- Deploy workspace (per-environment depth per B1-4 MD-3) -------
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

# --- Multi-agent contract (ADR-0009 §1–§3) ------------------------
# Single authorship for CLAUDE.md (source) and the two pointers,
# so /sync-agents drift cannot be approved without the canonical
# contract being touched.
/CLAUDE.md                              @PLACEHOLDER-org/platform-team
/AGENTS.md                              @PLACEHOLDER-org/platform-team
/.claude/                               @PLACEHOLDER-org/platform-team
/.codex/                                @PLACEHOLDER-org/platform-team

# --- CI + branch protection (ADR-0001 §C9, ADR-0008) --------------
# /.github/workflows/ and /.github/CODEOWNERS kept as explicit
# overrides (same owner as /.github/) so a future workflows-only or
# CODEOWNERS-only reviewer group can be slotted in without editing
# /.github/.
/.github/                               @PLACEHOLDER-org/platform-team
/.github/workflows/                     @PLACEHOLDER-org/platform-team
/.github/CODEOWNERS                     @PLACEHOLDER-org/platform-team

# --- Root build/runtime config ------------------------------------
/Makefile                               @PLACEHOLDER-org/platform-team
/go.work                                @PLACEHOLDER-org/platform-team
/go.work.sum                            @PLACEHOLDER-org/platform-team
/docker-compose.yml                     @PLACEHOLDER-org/platform-team
/scripts/                               @PLACEHOLDER-org/platform-team

# --- Studies (reasoning history, R8) ------------------------------
/studies/                               @PLACEHOLDER-org/platform-team
```

### Substitution protocol (W3-P8b operational task)

**New contribution proposed here, requires review.** The
substitution procedure below is a workflow commitment that no
upstream ADR enumerates; it generalizes the
`PLACEHOLDER`-marker discipline used by W3-P7a / W3-P7c.

The W3-P8b session creates the production GitHub org (per ADR-0008
host-primitive follow-up), then performs a mechanical
search-and-replace of `PLACEHOLDER-org/` with the real org slug
across `/.github/CODEOWNERS`, `engine/internal/env/qa.go`,
`engine/internal/env/prod.go`, `deploy/overlays/qa/`, and
`deploy/overlays/prod/`. The substitution is a single atomic edit
per file; no shape changes are permitted during substitution
(those would reopen this study).

---

## Consequences

1. **The schema mirror is review-protected by the same group as
   the source.** A merge request that edits
   `engine/internal/dsl/schema/v1.schema.json` but not
   `rules/_schema/v1.schema.json` fails CI on the byte-equality
   gate (ADR-0001 §C2) **and** requires the same review group on
   both files — there is no asymmetric-approval path through which
   one half could land without the other.

2. **`_owners.yaml` review is platform-team-only.** A new entity
   onboarded via a `rules/<entity>.yaml` edit by a domain team
   cannot ship without a paired `_owners.yaml` edit, which the
   linter rejects (ADR-0006 §9) and which CODEOWNERS routes to
   platform-team review. "No alert without owner" is enforced
   twice: at lint time and at review time.

3. **Per-entity ownership refinement is `_owners.yaml`'s job, not
   CODEOWNERS's.** CODEOWNERS commits at the workspace-default
   level (`@rules-authors` for `rules/`). Per-entity routing is
   `_owners.yaml`'s `owner` field, resolved at runtime by the
   alerting consumer (ADR-0006 §1). Adding a second domain team
   does not require editing CODEOWNERS; it requires adding the
   group to the GitHub org and referencing it from
   `_owners.yaml`.

4. **Existing `customer` entity's `_owners.yaml` `owner` field
   resolves to `@PLACEHOLDER-org/rules-authors`.** ADR-0006 §1
   commits that `owner` strings match a CODEOWNERS group; the
   ADR-0006 §9 linter check ("no alert without owner") implicitly
   requires that group to exist in CODEOWNERS. Until OQ-B1-9.4
   resolves per-entity refinement, `@rules-authors` is the only
   `rules/`-scoped group that exists, so it is the value the
   `customer` entity's `owner` field references. Concretely:
   the W3-P8b session that publishes the CODEOWNERS file must
   also adjust `rules/_owners.yaml` so the `customer` entry's
   `owner: ...` value matches the placeholder substitution. The
   shape of this is committed here; the literal edit lands in
   W3-P8b.

5. **Production overlay edits cannot land without SRE review.**
   `deploy/overlays/qa/` and `deploy/overlays/prod/` carry the
   per-environment secrets and GCP-IAM annotations (W3-P7c).
   Their CODEOWNERS line lists both `@platform-team` and `@sre`,
   so GitHub branch-protection's "review from CODEOWNERS"
   requirement gates them on both groups.

6. **The agent-contract surface drifts under platform-team review
   only.** `CLAUDE.md`, `AGENTS.md`, `.codex/AGENTS.md`, and the
   playbooks under `.claude/playbooks/` cannot be edited without
   platform-team review. `/sync-agents` is the mechanism that
   keeps the pointer files aligned with `CLAUDE.md` (ADR-0009 §3);
   CODEOWNERS is the human checkpoint that ensures the propagation
   is reviewed.

7. **The linter version pin is platform-team-owned.**
   `.github/workflows/` carries the CI lanes that pin the linter
   binary's digest (ADR-0001 §C9, Decision §5); the workflow file
   is under platform-team CODEOWNERS, so the pin cannot be
   changed without the same review depth as the schema itself.

8. **`PLACEHOLDER-org/` is intentional.** The literal sits in the
   committed file until the W3-P8b operational session substitutes
   it. The placeholder text is unambiguous (no real GitHub org
   matches `PLACEHOLDER-org`), so a partially-deployed state is
   loud rather than silently broken.

9. **The CODEOWNERS file itself is platform-team-owned.** Editing
   the path-rule table is itself a platform-team decision, gated
   on this study's promotion-target ADR (or its successor).
   Reopening the shape requires reopening this study.

10. **`studies/` is platform-team-owned.** Reasoning history is
    protected from domain-team edits even though the documents
    inside are not "code"; R8 ("studies inform but are not the
    product") combined with platform-team CODEOWNERS means that
    amendments to historical reasoning go through the same review
    group as the ADRs that supersede them.

---

## Open Questions

- **OQ-B1-9.1.** Concrete GitHub org identifier and team slugs
  (e.g., `@dq-platform/platform-team` vs `@acme/dq-platform-team`).
  **Out-of-scope for current cycle — substituted mechanically in
  the W3-P8b operational session once the production org exists
  per ADR-0008 host-primitive follow-up.**

- **OQ-B1-9.2.** Whether `/sync-agents` becomes a CI gate that
  fails when pointer files diverge from `CLAUDE.md` summaries.
  **Out-of-scope for current cycle — follow-up to ADR-0009
  Consequence 4. Until promoted, CODEOWNERS plus the
  same-MR-propagation convention is the enforcement surface.**

- **OQ-B1-9.3.** Defense-in-depth `_owners.yaml` validation at
  the manifest publisher and engine loader, beyond the linter's
  first-line check. **Out-of-scope for current cycle — ADR-0006
  OQ-10; reopen requires extending ADR-0005 and ADR-0007
  verification sets.**

- **OQ-B1-9.4.** Per-entity CODEOWNERS rules (e.g.,
  `/rules/customer.yaml @customer-team`) versus the current
  workspace-default-only shape that delegates per-entity
  refinement to `_owners.yaml`. **Out-of-scope for current cycle
  — defer until the second domain team onboards (today only
  `customer` exists per W3-P6d). The additive change is a
  single CODEOWNERS line per entity and does not require
  reopening this study.**

- **OQ-B1-9.5.** Whether GitHub `secret_scanning` /
  `dependency_review` / other repo-level governance lanes need
  their own CODEOWNERS rules. **Out-of-scope for current cycle —
  none of those lanes exist yet; add their CODEOWNERS line in
  the same MR that introduces the lane.**

- **OQ-B1-9.6.** Per-tool CODEOWNERS refinement within
  `tools/<tool>/` once additional tools land beyond `lint` and
  `manifest`. **Out-of-scope for current cycle — `tools/` is
  platform-team default; refinement is additive when a tool with
  a distinct review group materializes.**

---

## Promotion target

This study is promoted during a future ADR-promotion session to:

    docs/adr/0015-codeowners.md

The `0015` is the next ADR number after the most recent ADR in
`docs/adr/` (`0014-trigger-handler-contract.md`). The slug
(`codeowners`) is the stable part; the number adjusts at
promotion time if the ADR numbering convention shifts.

The decision-log update lands in the same session that commits
this study: B1-9 row → `resolved-study` with the link to this
file.
