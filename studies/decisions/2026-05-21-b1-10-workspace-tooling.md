<!-- path: studies/decisions/2026-05-21-b1-10-workspace-tooling.md -->

# B1-10 — Workspace Tooling Stack

## Metadata

- **B1 reference:** B1-10 in
  [`studies/foundation/06-decision-log.md`](../foundation/06-decision-log.md).
- **Status:** resolved-study (single critique round; the
  decision is largely confirmatory of foundation 02).
- **Last updated:** 2026-05-21
- **Upstream resolved:** B0-1
  ([compatibility model](./2026-05-20-engine-rules-compatibility.md)),
  W2-5
  ([tag conventions](./2026-05-21-platform-decisions-wave2.md)),
  the sequencing study
  ([Wave 3 phases](./2026-05-21-wave3-sequencing.md)).
- **Downstream:** Wave 3 Phase 2 (root infrastructure) and
  every Wave 3 phase that adds a Go module.
- **Promotion target:** `docs/adr/0014-workspace-tooling.md`.

---

## Context

Foundation 02 (§"Top-Level Directory Layout" and §"Local
Development") proposes Go workspaces via `go.work` as the
working default for the platform, with `engine/` as one
module and `tools/<tool>/` as one module per tool. The
foundation document lists three example tool modules
(`tools/lint`, `tools/dryrun`, `tools/publisher`).

B1-10 in the decision log asks two things:

1. Confirm Go workspaces (`go.work`) as the tooling choice.
2. Finalize the per-tool module structure.

The first is settled in spirit by foundation 02's working
default; the second is settled in substance by foundation
02's `go.work` example. The B1-10 row remained `open`
because no resolution document had been written.

Wave 3 Phase 2 (root infrastructure) is the first phase to
create `go.work` and per-tool modules. Per the sequencing
ADR (ADR-0013) Consequence 9, an open B1 row blocking a
Wave 3 phase is resolved in a separate study session before
the phase proceeds. This study is that resolution. It is
deliberately short — the decision is confirmatory of
foundation 02, not a major new architectural commitment.

---

## Decision Drivers

1. **D1. Workspace independence.** Each Go module evolves
   at its own pace and is independently buildable. Cross-
   module changes still get a single commit graph, but the
   per-module isolation prevents accidental coupling.
2. **D2. Per-tool release model.** ADR-0012 commits the
   `tools-lint-v*` tag prefix and notes that `tools/` may
   grow other tools, each with its own prefix. The module
   structure must support per-tool release without
   forcing other tools into the same version.
3. **D3. Rules workspace is not a Go module.** ADR-0001
   commits that `rules/` is lintable in isolation via
   `tools/lint` reading the schema mirror from
   `rules/_schema/`; the lintability does not depend on
   importing engine packages. `rules/` is a structured
   collection of YAML files, not Go code.
4. **D4. Deploy workspace is non-Go by default.** ADR-0007
   and ADR-0010 expect `deploy/` to carry Kubernetes
   manifests and environment overlays. Future Go-based
   infrastructure tools (if any) join `go.work`
   additively.
5. **D5. Additive evolution.** Adding a new tool module
   later (e.g., `tools/dryrun`) is an additive change to
   `go.work`. The initial `go.work` should contain only
   what currently exists, not speculative future modules.

---

## Considered Options

- **(A) Single root module.** One `go.mod` at the
  repository root; engine and tools live as sub-packages.
  Rejected by D2 (no per-tool release independence) and
  D1 (cross-package changes have no module-boundary
  pushback).

- **(B) One module per workspace.** `engine/`, `rules/`,
  `tools/`, `deploy/`, `docs/` each get a `go.mod`.
  Rejected by D3 (`rules/` is not Go) and by the
  observation that `tools/` is a holder for multiple
  binaries, each of which deserves its own module per D2.

- **(C) Go workspaces with one module per Go-binary tool,
  plus the engine module.** `go.work` lists `./engine` and
  one `./tools/<tool>/` per tool binary. `rules/` and
  `deploy/` are not Go modules. **Recommended.**

---

## Recommendation

Adopt **(C)** — Go workspaces with one module per
Go-binary tool. This is the foundation-02 baseline,
recorded here as the formal resolution.

- **`engine/`** — single Go module
  (`dq-platform/engine`).
- **`tools/<tool>/`** — one Go module per tool binary
  (`dq-platform/tools/<tool>`). Tools are added to
  `go.work` additively as each tool's phase lands.
- **`rules/`** — not a Go module. Lintable by
  `tools/lint` via the schema mirror at
  `rules/_schema/`.
- **`deploy/`** — not a Go module by default. If a future
  Go-based infrastructure tool is introduced (e.g., a
  manifest generator), its module is added to `go.work`
  at that time.

The recommendation is grounded in foundation 02 (§"Top-
Level Directory Layout", §"Local Development") and in the
Wave 1 commitments cited above. The specific commitment
beyond what those documents state is:

- The **initial `go.work`** lists only the modules whose
  scaffolding is in place. Phase 2 lands `engine/` and
  `tools/lint/`. Other tools are added when their
  scaffolding phases land. **New contribution proposed
  here, requires review.**

---

## Consequences

1. **`go.work` at the repository root is the canonical
   workspace declaration.** The Go toolchain reads
   `go.work` from the root and treats each listed
   directory as a module participating in the workspace.

2. **Go version is declared once in `go.work`.** Module-
   level `go.mod` files inherit the workspace's Go version
   unless they explicitly override.

3. **Cross-module imports do not require a `replace`
   directive.** `go.work` resolves `dq-platform/tools/lint`
   importing `dq-platform/engine/internal/dsl/schema` (when
   that becomes a thing) without any per-module `replace`.

4. **Adding a new tool is a single line in `go.work` plus
   a new module directory.** No central registry, no other
   modules need to change.

5. **The rules workspace stays validatable without Go
   tooling involvement.** The linter binary (a Go tool
   under `tools/lint/`) is the only Go-touching consumer
   of `rules/`. Contributors authoring rules use only the
   linter binary; they do not run `go` directly against
   the rules workspace.

6. **The `deploy/` workspace stays language-agnostic by
   default.** Kubernetes manifests, environment overlays,
   and infrastructure-as-code files live there without
   Go module ceremony. Future Go-based infra tools are
   additive.

7. **Per-tool tags from ADR-0012 align with per-tool
   modules.** A `tools-lint-v*` tag releases the
   `tools/lint/` module independently of the engine or
   other tools.

8. **The initial `go.work` is minimal.** Foundation 02's
   example listed three tool modules; Phase 2 lands only
   what exists. `tools/dryrun` and `tools/publisher` (or
   any other future tool) are added when their scaffolding
   phases land. This avoids speculative modules that may
   never materialize in their proposed shape.

---

## Open Questions

- **OQ-B1-10.1.** Whether `tools/lint`'s `go.mod` should
  declare a stricter minimum Go version than the workspace
  default. **Out-of-scope for current cycle — defer to
  the first Phase-3 session that builds the linter binary
  in earnest.**

- **OQ-B1-10.2.** Whether the engine's internal package
  layout (`engine/internal/dsl/schema/`,
  `engine/internal/loader/`, etc.) needs its own structural
  policy beyond "internal packages stay internal". **Out-
  of-scope for current cycle — Wave 3 Phase 4 engine
  scaffolding decides as code lands.**

- **OQ-B1-10.3.** Whether a future infrastructure tool
  under `deploy/` would join `go.work` or live as a
  separate workspace. **Out-of-scope for current cycle —
  Wave 3 Phase 7 (`deploy/` scaffolding) decides.**

---

## Promotion target

This study is promoted during a future ADR-promotion
session to:

    docs/adr/0014-workspace-tooling.md

The `0014` is the next ADR number after the Phase-1 batch
(`0001–0013`). The slug (`workspace-tooling`) is the
stable part; the number adjusts at promotion time if the
ADR numbering convention shifts.

The decision-log update lands in the same session that
commits this study: B1-10 row → `resolved-study` with the
link to this file.
