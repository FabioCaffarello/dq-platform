<!-- path: docs/adr/0016-workspace-tooling.md -->

# ADR-0016 — Workspace Tooling Stack

- **Status:** accepted
- **Date:** 2026-05-23

---

## Context

The dq-platform monorepo carries five product workspaces — `engine/`,
`rules/`, `tools/`, `deploy/`, `docs/` — and a per-tool release model
already committed by other ADRs. Three prior commitments shape the
workspace-tooling question:

- **ADR-0001** commits that the rules workspace is lintable in
  isolation: `tools/lint` validates `rules/*.yaml` against the
  byte-equal schema mirror under `rules/_schema/`, without importing
  engine packages. The rules workspace must therefore be valid as a
  structured collection of YAML files; it is not required to compile
  as Go code.
- **ADR-0012** commits per-tool tag prefixes (e.g., `tools-lint-v*`)
  alongside the engine tag prefix `engine-v*`. Each Go-producing tool
  releases independently; the workspace structure must let one tool
  cut a release without forcing the engine or sibling tools to bump.
- **ADR-0007** and **ADR-0010** commit that `deploy/` carries
  Kubernetes manifests and per-environment overlays (Kustomize per
  ADR-0010). The deploy workspace is language-agnostic by default;
  no Go module ceremony is required for it to function.

ADR-0013 (Wave 3 sequencing) Phase 2 — root infrastructure — is the
phase that creates the workspace declaration on disk. Phase 2 needs a
settled answer to two questions before it can land:

1. Is the Go toolchain configured via a single root `go.mod`, a
   per-workspace `go.mod` set, or via Go workspaces (`go.work`)?
2. What is the module structure inside `tools/`?

This ADR commits the answers. The decision is confirmatory of the
working default sketched in the foundation document for monorepo
topology; the substance below is the formal commitment, restated for
review and enforcement, with one explicit refinement (the **initial**
`go.work` lists only modules whose scaffolding is in place — additive
afterwards).

**Out of scope of this ADR:**

- The specific Go minimum-version pin in `go.work`. The toolchain
  selector is decided per phase as modules are added; the ADR commits
  only the workspace shape, not the version field.
- The internal package layout inside `engine/` (e.g.,
  `engine/internal/dsl/schema/`, `engine/internal/loader/`).
  Engine-internal structure is decided by the engine scaffolding
  sessions that land the corresponding packages.
- Whether a future infrastructure tool under `deploy/` joins
  `go.work` or lives as its own workspace. The default position is
  that any Go-binary infrastructure tool joins this workspace; the
  formal answer lands when such a tool is proposed.

---

## Decision

### 1. Go workspaces via `go.work` is the canonical workspace declaration

The Go toolchain is configured via a `go.work` file at the repository
root. The workspace lists each participating Go module by its
directory path; each listed module is independently buildable and
independently releasable.

Two alternatives are rejected:

- **Single root `go.mod`.** One module covering the whole repository
  collapses the engine and every tool into a single release artifact,
  contradicting ADR-0012's per-tool tag prefixes. Cross-cutting
  changes also lose the module-boundary pushback that makes accidental
  coupling visible at compile time.
- **One module per workspace directory.** `engine/`, `rules/`,
  `tools/`, `deploy/`, `docs/` each gaining a `go.mod` is rejected on
  two grounds: `rules/` is not Go code (per ADR-0001 it is a
  structured YAML collection lintable through the schema mirror), and
  `tools/` is a holder for multiple binaries — each binary deserves
  its own release cadence per ADR-0012, so a single `tools/go.mod`
  would defeat the per-tool tag prefix model.

### 2. One Go module per Go-producing surface

The workspace lists Go modules at two granularities:

| Module path             | Module identifier              | Role                                                                                                    |
|-------------------------|--------------------------------|---------------------------------------------------------------------------------------------------------|
| `./engine`              | `dq-platform/engine`           | Engine runtime, schema source, internal packages.                                                       |
| `./tools/<tool>/`       | `dq-platform/tools/<tool>`     | One module per Go-binary tool. Each tool releases independently per ADR-0012's `tools-<tool>-v*` prefix. |

`rules/` and `deploy/` are intentionally absent from this table.
`rules/` is not Go and is validated by `tools/lint` reading the schema
mirror; `deploy/` is language-agnostic Kubernetes manifests and
Kustomize overlays per ADR-0010. Neither directory needs a `go.mod`,
and adding one would create build-graph noise without benefit.

### 3. The initial `go.work` is minimal — additive afterwards

The `go.work` committed in Phase 2 (root infrastructure) lists only
the modules whose scaffolding exists at that point: `./engine` and
`./tools/lint`. Subsequent phases add their modules to `go.work` in
the same change that introduces the module directory; no separate
registration step is required.

Speculative future tools (e.g., a dry-run runner, a manifest
publisher prototype) are not listed in `go.work` until their
scaffolding lands in earnest. The single-line addition keeps
`go.work` honest about what currently exists in the workspace.

### 4. Cross-module imports use the workspace, not `replace` directives

Module-to-module imports across the workspace are resolved by
`go.work` directly. A `tools/<tool>/go.mod` importing
`dq-platform/engine/internal/...` (when an exported boundary exists)
does so without per-module `replace` directives.

The absence of `replace` directives is load-bearing: it is the
mechanism by which a tool module released independently of the engine
remains buildable in isolation outside the workspace (via `go get` of
a tagged version), while still resolving against the current engine
sources when built inside the workspace.

---

## Consequences

1. The `go.work` file at the repository root is the single source of
   truth for the workspace's module membership. Phase 2 commits the
   initial file with `./engine` and `./tools/lint`; every phase that
   lands a new module appends to `go.work` in the same change.

2. The Go toolchain version is declared in `go.work`. Module-level
   `go.mod` files inherit the workspace's toolchain selector unless
   they explicitly override — overrides are reserved for tools that
   need a stricter floor than the workspace default and are recorded
   in the tool module's own `go.mod`.

3. The `rules/` workspace stays validatable without any Go-tooling
   dependency at the contributor's machine beyond running the linter
   binary. Domain teams editing rule YAMLs do not invoke `go` against
   `rules/`; the linter binary under `tools/lint/` is the only
   Go-touching consumer of the rules workspace.

4. The `deploy/` workspace stays language-agnostic by default. Future
   Go-based infrastructure tools (if introduced) join `go.work`
   additively; they do not change the deploy workspace's primary
   shape as a Kubernetes manifests + overlays directory.

5. Per-tool tags from ADR-0012 align with per-tool modules.
   `tools-lint-v1.2.0` releases the `./tools/lint` module without
   touching `./engine` or any sibling tool. The engine releases under
   `engine-v*` without bumping any tool. Independent release cadence
   is enforced by the workspace structure itself, not by convention.

6. Adding a new tool is a two-step additive change: create the
   module directory under `tools/<new-tool>/` with its own `go.mod`,
   and append `./tools/<new-tool>` to `go.work`. No central registry,
   no other modules edited.

7. Sibling tools do not import each other through unstable internal
   packages. Each tool's `internal/` is invisible to siblings per
   Go's standard `internal` rule; cross-tool sharing — when it
   becomes necessary — happens through an exported package in
   `./engine` or via a deliberately introduced shared module. The
   per-tool tag model from ADR-0012 depends on this isolation: a
   tool that quietly imports another tool's `internal/` package
   couples their release cadences and breaks the independent-release
   contract.

8. The workspace remains buildable for contributors with only the
   Go toolchain installed; no `direnv`, no environment-variable
   gymnastics, no `replace` directives across modules. `go build ./...`
   from the repository root resolves through `go.work` and builds
   every listed module.

9. The decision to keep `go.work` minimal and additive prevents
   speculative modules from polluting the workspace. A directory
   containing only a `go.mod` and no real content drains
   contributors' attention with no compensating value; the discipline
   is to land the scaffolding and the workspace entry in the same
   change.

10. Reopening this ADR is required to change the workspace model
    itself — for example, splitting `engine/` into multiple modules,
    introducing a shared utility module imported by both engine and
    tools, or moving to a non-Go-workspaces approach. Adding a new
    tool module is additive and does not reopen the ADR.

---

## Notes

- The `engine/` module is intentionally a single Go module. Splitting
  it into multiple modules (e.g., a separate `engine/dsl/` module
  importable by tools without dragging in the runtime) is a
  refactor-time question reopened only when a concrete cross-module
  consumer needs it. The default position is one engine module.

- The workspace declaration commits the **shape** of the build graph,
  not its Go-version pin. The toolchain version listed in `go.work`
  is adjusted as needed during Phase 2 and at any point afterwards;
  bumping the version is a routine maintenance edit and does not
  require an ADR.

- Per-tool `go.mod` files declare their own Go version only when
  they need a stricter floor than the workspace default. The common
  case is no per-module override.
