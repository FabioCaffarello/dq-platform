<!-- path: tools/README.md -->

# `tools/` — Auxiliary CLIs

The tools workspace holds auxiliary command-line binaries:
the linter (validates rule YAMLs), the dry-run runner
(generates SQL without executing), and the manifest
publisher (builds and publishes manifests per
[ADR-0005](../docs/adr/0005-manifest-publication-semantics.md)).

Per
[B1-10's resolution](../studies/decisions/2026-05-21-b1-10-workspace-tooling.md),
each tool is **one Go module per binary** (e.g.,
`tools/lint/`, `tools/dryrun/`, `tools/publisher/`).
Modules are added to [`go.work`](../go.work) additively
as their scaffolding phases land.

## Current modules

- **[`lint/`](lint/)** — declared in `go.work`; scaffolded
  in Phase 3 (the load-bearing CI gate from
  [ADR-0001](../docs/adr/0001-engine-rules-compatibility.md)
  depends on this binary).

Future modules (`tools/dryrun/`, `tools/publisher/`, etc.)
are added when their respective Wave 3 phases need them.
