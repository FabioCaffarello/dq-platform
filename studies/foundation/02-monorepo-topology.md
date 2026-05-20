<!-- path: studies/foundation/02-monorepo-topology.md -->

# 02 — Monorepo Topology

## Metadata

- Purpose: define the internal structure of the monorepo, the tooling
  that supports it, the ownership rules that protect boundaries, the
  CI strategy that enforces them, and the release model that lets
  workspaces evolve at their own pace.
- Audience: platform engineers, contributors, CI maintainers, AI
  agents.
- Status: draft
- Last updated: 2026-05-20
- Promotion target: `docs/architecture/monorepo.md` and ADR set during
  Wave 3.

---

## Why Monorepo

The project consists of two strongly-related artifacts (the engine and
the rules) plus several supporting concerns (tools, deployment, docs).
A single repository is the right starting point because:

- **Compatibility is structural, not negotiated.** When the engine and
  the rules live in the same commit graph, an incompatible
  combination cannot exist as a published state — it would have
  failed CI together.
- **Refactors that span both can be atomic.** A schema change and its
  corresponding rule updates land in the same merge request.
- **Onboarding is unified.** A new contributor clones one repository,
  not two, and sees the whole picture.
- **CI complexity stays low.** No cross-repository pipelines, no
  release coordination between repos.

The monorepo choice does not weaken the **logical separation** between
the engine and the rules — see
[`03-boundary-contract.md`](./03-boundary-contract.md). The boundary
is enforced by ownership rules, path-filtered CI, and explicit
internal interfaces — not by repository boundaries.

## Anti-Goals

The monorepo must not become:

- a single blob where any change can touch any file;
- a workspace where the engine knows the shape of any specific rule;
- a workspace where the rules import engine internals;
- a release pipeline that rebuilds and republishes everything on every
  commit;
- a CI surface that runs every test on every change.

If any of these happen, the monorepo discipline has failed and the
project loses the benefits that justified it.

## Top-Level Directory Layout

```
.
├── engine/                  # Go runtime, DSL schema, compilers
├── rules/                   # YAML rule specifications by entity
├── tools/                   # Auxiliary CLIs (linter, dry-run, publisher)
├── deploy/                  # Kubernetes manifests, infra config
├── docs/                    # Cross-workspace documentation, ADRs
├── studies/                 # Reasoning artifacts (foundation, decisions)
├── .claude/                 # Claude Code workspace configuration
├── .codex/                  # Codex CLI workspace configuration (Wave 2+)
├── .gitlab/ or .github/     # CI configuration (Wave 2 decides which)
├── AGENTS.md                # Multi-agent entry point
├── CLAUDE.md                # AI agent operating contract
├── README.md                # Repository entry point
├── CONTRIBUTING.md          # How to contribute
├── CODEOWNERS               # Path-based review ownership
├── go.work                  # Go workspace declaration
├── Makefile                 # Top-level developer commands
├── docker-compose.yml       # Local development environment
└── .gitignore
```

## Workspace Tooling

### Go Workspaces (`go.work`)

The Go portions of the project — `engine/`, `tools/`, and any Go-based
infrastructure under `deploy/` — are organized as Go workspaces
declared in `go.work` at the repository root.

```
go 1.22

use (
    ./engine
    ./tools/lint
    ./tools/dryrun
    ./tools/publisher
)
```

Benefits:

- each Go module is self-contained with its own `go.mod`;
- cross-module development without `replace` directives during local
  iteration;
- CI can build and test each module independently;
- versioning happens per-module, aligning with the per-workspace tag
  strategy described below.

### YAML Workspace (`rules/`)

`rules/` is not a Go module. It is a structured collection of YAML
files validated by the linter in `tools/lint`. Its "tooling" is:

- the linter binary (versioned with `engine/`);
- a Makefile target that runs the linter locally;
- CI pipelines that validate every YAML on every change.

### Documentation Workspace (`docs/`)

`docs/` is plain markdown plus diagrams. It has no build step at
present. If the project later adopts a static site generator, that
generator's configuration lives inside `docs/`, not at the root.

### Deployment Workspace (`deploy/`)

`deploy/` contains Kubernetes manifests, Kustomize overlays, and
infrastructure-as-code declarations. Its "tooling" is the
infrastructure CLI of choice — to be decided in Wave 2 (kustomize,
helm, terraform, or a combination).

## Ownership Boundaries (CODEOWNERS)

A path-based CODEOWNERS file enforces who reviews what. The
asymmetric review model recognizes that **the engine has cross-cutting
risk, while the rules have domain-specific risk**.

Provisional pattern (to be finalized in Wave 3):

```
# Default: platform team reviews everything not claimed below.
*                                    @platform-team

# Engine internals — platform team only.
/engine/                             @platform-team

# Schema — platform team plus a designated schema owner.
/engine/internal/dsl/schema/         @platform-team @schema-owner

# Tools — platform team.
/tools/                              @platform-team

# Deployment — platform team plus SRE.
/deploy/                             @platform-team @sre-team

# Documentation — platform team for cross-cutting; per-section owners
# inside.
/docs/                               @platform-team
/docs/operations/                    @platform-team @sre-team

# Rules — central infrastructure files reviewed by platform team.
/rules/_schema/                      @platform-team
/rules/_owners.yaml                  @platform-team
/rules/_examples/                    @platform-team

# Each entity's rules are reviewed by its declared owner.
# These are populated dynamically as entities are onboarded.
# Example:
# /rules/entities/customer/          @customer-team
# /rules/entities/transactions/      @transactions-team
```

### Why this matters

The CODEOWNERS file is the most concrete defense against
responsibility drift. Without it, anyone with merge rights could
silently change the schema, change another team's rules, or modify
deployment configuration. With it, the review graph reflects the
project's actual ownership model.

## CI Strategy

The repository uses **path-filtered pipelines** so that a change to
`rules/entities/customer/` does not trigger a full engine rebuild, and
a change to `docs/` does not trigger anything heavy.

### Pipeline shapes

The exact CI platform is a Wave 2 decision. The conceptual shape, in
abstract terms, is:

```
trigger: any merge request

stages:
  - detect-changes        # determine which workspaces changed
  - lint-engine           # only if engine/ changed
  - test-engine           # only if engine/ changed
  - build-engine          # only if engine/ changed
  - lint-tools            # only if tools/ changed
  - test-tools            # only if tools/ changed
  - lint-rules            # only if rules/ changed
  - validate-rules-schema # only if rules/ or schema changed
  - dry-run-rules         # only if rules/ changed, requires schema compat
  - lint-deploy           # only if deploy/ changed
  - lint-docs             # only if docs/ changed
  - integration           # only if engine/ or tools/ changed
  - security-scan         # always, but lightweight if only docs changed
```

### Two CI invariants

Independent of platform choice, the CI design honors two invariants:

**CI-1. Schema and rules can never diverge on `main`.**
A change to the schema that breaks existing rules must update the rules
or be rejected. A rule that does not match the current schema cannot
land. CI enforces this by always running rule validation against the
current schema, regardless of which side changed.

**CI-2. A workspace is responsible for proving its own health.**
The engine's CI proves the engine builds, lints, and tests. The
rules' CI proves every rule is schema-valid and dry-run-compilable.
The tools' CI proves each tool builds and tests. No workspace
piggybacks on another's success.

## Release Model

The project uses **per-workspace tags** rather than a single
monorepo-wide version. Each workspace evolves at its own pace, and
its tag carries a workspace-specific prefix.

### Tag conventions

```
engine-v<major>.<minor>.<patch>     # engine binary releases
rules-v<major>.<minor>.<patch>      # rules snapshot releases (manifest)
tools-lint-v<major>.<minor>.<patch> # linter releases (one per tool binary)
deploy-v<major>.<minor>.<patch>     # infrastructure manifest releases
```

The schema itself is **not** released independently. It is part of the
engine binary release; its version (`v1`, `v2`, etc.) is declared
inside the schema artifact and consumed by both engine and rules.
See [`03-boundary-contract.md`](./03-boundary-contract.md).

### Why per-workspace tags

- A change in the rules does not bump the engine version.
- A change in deployment configuration does not bump anything else.
- A bug fix in the linter does not require a synchronized
  re-release of the engine.
- The release notes for each artifact stay focused on what changed
  for **its** consumers.

### Release process (conceptual)

For each workspace, the release process is:

1. CI on `main` passes for that workspace.
2. A maintainer creates an annotated tag with the appropriate prefix.
3. CI detects the tag, builds the release artifact, and publishes it
   to the appropriate destination:
   - `engine-v*` → container registry;
   - `rules-v*` → manifest object in GCS plus a Git tag;
   - `tools-*-v*` → container registry or binary artifact store;
   - `deploy-v*` → tagged commit consumed by the deployment pipeline.
4. The release notes are generated from the commits since the previous
   tag of the same prefix.

### Versioning policy

Semantic versioning applies per workspace, with the
**workspace-relative** interpretation:

- **Major** — a change that breaks downstream consumers of this
  workspace's outputs (the schema, the rule manifest format, the
  linter CLI flags, the engine API).
- **Minor** — a change that adds capability without breaking existing
  consumers.
- **Patch** — bug fix or non-behavioral change.

The schema version (e.g. DSL `v1`) is **separate** from the engine
version. An engine release of `engine-v2.5.0` may still implement
schema `v1`. This is intentional: the schema version is the
**contract** between engine and rules; the engine version is the
**implementation**.

## Local Development

The repository root provides a Makefile with workspace-aware targets:

```
make lint              # lint everything that changed since main
make test              # test everything that changed since main
make lint-engine       # lint engine only
make test-engine       # test engine only
make lint-rules        # validate every rule YAML
make dry-run-rules     # generate SQL for every rule without executing
make up                # start docker-compose local environment
make down              # stop docker-compose local environment
```

The local environment (`docker-compose.yml`) emulates whichever
external services the Wave 2 decision allows to be emulated. Services
that cannot be emulated faithfully require a sandbox cloud project.

## Open Topics

The following are not yet resolved and are tracked in
[`06-decision-log.md`](./06-decision-log.md):

- exact CI platform (Wave 2 decision: affects every pipeline file);
- exact local-environment emulator stack (Wave 2 decision: which
  cloud services are emulated, which need sandbox);
- final CODEOWNERS team names (depends on org structure at time of
  Wave 3);
- whether documentation gets a static site generator;
- whether `deploy/` uses kustomize, helm, terraform, or a
  combination;
- exact tooling subdirectory layout inside `tools/` (one module per
  tool, or one module with multiple binaries).

These do not block the topology decision; they are downstream choices.
