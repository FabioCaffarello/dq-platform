<!-- path: docs/governance.md -->

# Governance

The DQ Platform's review-time governance: which teams own which
parts of the repository, which contribution flows are mechanical,
and where to find each rule.

This document is **referential**: it summarizes commitments made
elsewhere and points at the canonical sources. The "how to
contribute as a domain team" walkthrough lives separately in the
contribution guide (W3-P8c).

---

## 1. Review model

The repository implements an **asymmetric review model** per
[ADR-0001](adr/0001-engine-rules-compatibility.md) §C3 (schema
source + mirror under one review group; per-entity rule YAMLs
editable by domain teams) and per-environment review depth per
[ADR-0006](adr/0006-alert-routing-contract.md) §3 (production
substrate config also reviewed by SRE).

CODEOWNERS at [`/.github/CODEOWNERS`](../.github/CODEOWNERS) is
the enforcement surface. GitHub's branch-protection rule "require
review from CODEOWNERS" gates every merge into `main` on the
groups listed for each touched path.

---

## 2. Review groups

| Group placeholder | Approves |
|---|---|
| `@PLACEHOLDER-org/platform-team` | Engine runtime, schema source and mirror, tools, deploy base, the `local` overlay, all of `docs/`, CI workflows, root build/runtime config, the multi-agent contract surface (`CLAUDE.md`, `AGENTS.md`, `.codex/AGENTS.md`, `.claude/`), and the `studies/` history. Default for everything not explicitly delegated below. |
| `@PLACEHOLDER-org/sre` | Co-owner with platform-team on `deploy/overlays/qa/` and `deploy/overlays/prod/`. Per [ADR-0006](adr/0006-alert-routing-contract.md) §3, production substrate configuration requires SRE review depth in addition to platform-team. |
| `@PLACEHOLDER-org/rules-authors` | Per-entity rule YAMLs under `rules/` (excluding `_schema/`, `_owners.yaml`, `_owners.schema.json`, which remain platform-team-only). Implements the asymmetric review model from ADR-0001 §C3. |

**Owner-identifier convention.** Per
[ADR-0006](adr/0006-alert-routing-contract.md) §1, a
`_owners.yaml` entity's `owner` field is "a team identifier
matching a CODEOWNERS group". The literal syntax this repository
uses is the same `@org/team` form CODEOWNERS itself uses, so the
two surfaces can be cross-checked by exact string comparison. The
syntax choice is a repository convention; the linter check that
enforces the correspondence mechanically is a future extension
beyond the ADR-0006 §9 entity-existence check (defense-in-depth
at the manifest publisher and engine loader is tracked under
ADR-0006 Notes as a follow-up).

---

## 3. Contribution-time flows

### 3.1 Adding or modifying a study → ADR

Studies live in `studies/decisions/` and are not part of the
published repository (per [`CLAUDE.md`](../CLAUDE.md) R8). When a
study stabilizes, the `/promote-to-adr` slash command rewrites it
as a MADR-style ADR under [`docs/adr/`](adr/). The mechanism is
captured in
[`.claude/commands/promote-to-adr.md`](../.claude/commands/promote-to-adr.md);
the decision log at `studies/foundation/06-decision-log.md` keeps
both the original study link and the ADR link.

ADRs are forward-only: an ADR never back-links into `studies/`.

### 3.2 Editing the agent contract

`CLAUDE.md` is the canonical contract for AI coding agents (per
[ADR-0009](adr/0009-multi-agent-contract.md) §1). `AGENTS.md` and
`.codex/AGENTS.md` are thin pointer files; they introduce no new
rules. The `/sync-agents` slash command propagates `CLAUDE.md`
changes into the pointer files; the mechanism is captured in
[`.claude/commands/sync-agents.md`](../.claude/commands/sync-agents.md).

Rule changes flow `CLAUDE.md` → pointers; the reverse direction
is rejected by convention and gated by CODEOWNERS (all three
files are platform-team-only).

### 3.3 Resolving a backlog decision

`B0` / `B1` / `B2` decisions are tracked in the decision log.
`/resolve-b0` drafts a study for one B-item per
[`.claude/commands/resolve-b0.md`](../.claude/commands/resolve-b0.md);
`/check-decision-backlog` reports current state per
[`.claude/commands/check-decision-backlog.md`](../.claude/commands/check-decision-backlog.md).
Studies are critiqued via `/critique` per
[`.claude/commands/critique.md`](../.claude/commands/critique.md)
before they reach `resolved-study`.

---

## 4. CODEOWNERS path-rule reference

The canonical path-rule table is the file
[`/.github/CODEOWNERS`](../.github/CODEOWNERS) itself. Reproducing
it here would create a drift target; read the file directly. The
file is ordered general → specific so GitHub's last-match-wins
evaluation puts overrides last — workspace defaults appear first
(e.g., `rules/` → `@rules-authors`) and per-path overrides follow
(e.g., `rules/_schema/` and `rules/_owners.yaml` → `@platform-team`).
Per-environment depth (SRE on `qa` and `prod` overlays) and the
multi-agent contract single-authorship rule from §1 are encoded
the same way.

---

## 5. Substitution status (placeholders)

All `@org/team` identifiers in `/.github/CODEOWNERS`,
`engine/internal/env/qa.go`, `engine/internal/env/prod.go`, and
`deploy/overlays/{qa,prod}/` are committed with the literal
`PLACEHOLDER-org/` prefix. They are substituted to the production
GitHub org slug in a separate operational session once the org
exists ([ADR-0008](adr/0008-git-host.md) host-primitive
follow-up).

A partially-deployed state is loud rather than silently broken:
no real GitHub org or team matches `PLACEHOLDER-org/`, so any
review request against a `PLACEHOLDER`-marked path fails to
resolve a reviewer until substitution lands.
