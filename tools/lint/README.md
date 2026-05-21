<!-- path: tools/lint/README.md -->

# `tools/lint/` — Rules Linter

`dq-lint` validates rule YAMLs against the workspace schema
mirror at [`rules/_schema/`](../../rules/) and enforces the
byte-equality contract between the rules mirror and the engine's
canonical schema source under
[`engine/internal/dsl/schema/`](../../engine/) per
[ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md).

This is a Go module (`dq-platform/tools/lint`) declared in the
top-level [`go.work`](../../go.work) per
[B1-10's resolution](../../studies/decisions/2026-05-21-b1-10-workspace-tooling.md).

## What the linter enforces

- **`version:` field is mandatory** (ADR-0001 C4). Rules
  without it are rejected.
- **`version:` must equal a supported value** (currently
  `1`). Unknown versions are rejected.
- **`entity` must not contain the ASCII pipe character**
  (ADR-0002 input-safety). Entity is one of the five
  pipe-separated inputs to the `execution_id` hash with no
  escaping; the linter enforces this twice — once via the
  schema's `pattern`, and once via a belt-and-suspenders
  in-code check at schema load (so a schema edit that
  weakens the pattern fails loudly at linter startup, not
  silently at runtime).
- **`checks` must be non-empty** — a rule with no checks
  would never execute.
- **No unknown top-level or per-check fields**
  (`additionalProperties: false`) — keeps the rules surface
  forward-compatible (extensions land in v2+ schemas, not as
  additive top-level fields in v1).

## CLI

```sh
dq-lint [-schema <path>] [-rules <dir>] [-v]
```

- `-schema <path>` (default `rules/_schema/v1.schema.json`) —
  path to the rules schema mirror.
- `-rules <dir>` (default `rules`) — directory tree to walk
  for `*.yaml` files. The `_schema/` subdirectory is
  skipped automatically.
- `-v` — verbose; print each file as it is processed.

### Exit codes

- `0` — all rules validated (or no rules to lint).
- `1` — at least one rule failed validation.
- `2` — operational error (schema missing, malformed
  schema, I/O failure).

## Updating the schema

The schema at `engine/internal/dsl/schema/v1.schema.json` is
the canonical source. The mirror at
`rules/_schema/v1.schema.json` is **never edited by hand**;
it is mechanically derived. To update both halves
atomically:

```sh
# 1. Edit engine/internal/dsl/schema/v<N>.schema.json
# 2. Run:
make sync-schema
# 3. Commit both files in the same MR.
```

The schema-mirror CI workflow
([`schema-mirror.yml`](../../.github/workflows/schema-mirror.yml))
runs `cmp` between the two on every PR and push to `main`.
A mismatch fails the gate per ADR-0001 C2.

## Phase status

- **Phase 3 (this commit)** — minimum linter binary with
  the constraints above. The schema accepts `kind` as a
  free-form string discriminator (placeholder for the
  Phase-4+ DSL grammar).
- **Phase 5** extends the linter to reject entities without
  an `_owners.yaml` entry per
  [ADR-0006](../../docs/adr/0006-alert-routing-contract.md).

## Library dependencies

- [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3)
  — YAML parsing.
- [`github.com/santhosh-tekuri/jsonschema/v5`](https://pkg.go.dev/github.com/santhosh-tekuri/jsonschema/v5)
  — JSON Schema Draft 2020-12 validation.

Both are commodity Go libraries (R5-exempt as environment,
not borrowed patterns).
