<!-- path: docs/dev/schema-migration.md -->

# Rule Schema Migration Guide

> **Status:** the authoritative compatibility-state model is
> [ADR-0035](../adr/0035-compatibility-window-duration.md);
> this guide is the operator-facing how-to for migrating
> individual rule YAMLs between schema versions.

---

## What this guide is for

You're here for one of these reasons:

- **You saw a `DEPRECATED:` warning from `dq-lint`.** Jump to
  §"Reading the deprecation warning"; follow the cited
  version's migration section.
- **You're migrating a rule v1 → v2.** Read §"v1 → v2 delta"
  end-to-end; the section has both the field map and the
  concrete before/after example.
- **You're planning a multi-rule migration across an
  organization.** Skim §"Compatibility-state model" first to
  understand the per-version timeline; then read the relevant
  delta section.
- **You're authoring the next schema version (v3+).** Read
  §"Forward pattern" for the contract this guide commits to
  for future deltas.

---

## Compatibility-state model

[ADR-0035](../adr/0035-compatibility-window-duration.md)
commits the per-version lifecycle. Every schema version
passes through three states:

1. **`current`** — actively recommended; new rules SHOULD
   declare this version. v2 is the current version at the
   time of this writing.
2. **`deprecated`** — engine-supported but warned-on at lint
   time. Rule authors should migrate to the current version
   before the version's drop date. v1 is in this state.
3. **dropped** — removed from the engine's
   `SupportedSchemaVersions`. The schema-version dispatcher
   rejects the version with `unsupported schema version` at
   lint and loader time. The earliest possible drop date for
   v1 is **2026-08-23** (90-day floor anchored at v2's first
   manifest publish, PR #40 merged 2026-05-25).

The table is enforced in two places:

- **Operator-facing source of truth** — the
  [ADR-0035 §"Compatibility-state table"](../adr/0035-compatibility-window-duration.md)
  markdown table. Human-prose; updated by ADR amendment
  when a new major version ships.
- **Linter-side code** — the `SchemaCompatibility` Go map
  in
  [`tools/lint/compatibility.go`](../../tools/lint/compatibility.go).
  Byte-equivalent to the ADR table per operating
  convention; the table and the map update in the same
  PR when an amendment lands.

### Reading the deprecation warning

A run of `dq-lint -rules rules/` against a rule declaring a
deprecated version emits a line like:

```
rules/example.yaml: DEPRECATED: rule declares schema version
1 which is deprecated per ADR-0035 (engine support since
0.1.0; earliest drop 2026-08-23); migrate to a `current`
version before the drop date
```

The line carries three actionable facts:

- **The rule's path** — the file to edit.
- **The deprecated version + its earliest drop date** — the
  deadline by which the rule must be migrated to avoid
  rejection by a future engine release.
- **The "engine support since" string** — the engine
  version that first accepted this schema. Useful when
  reviewing whether an old historical rule on a
  long-deprecated version is still being maintained
  intentionally or has fallen behind.

The warning does **NOT** cause `dq-lint` to exit non-zero
— it's informational, not a validation failure. Suppress
the surface entirely with `dq-lint -no-deprecation-warnings`
when running lint inside scripts that aren't yet ready to
surface the warning.

---

## v1 → v2 delta

v2 was committed by the Wave-S decision arc and is the
**current** version. Three coupled additions distinguish
it from v1:

1. **`mode`** — declares whether the entity is **set-mode**
   (a bounded set of rows evaluated over a closed window)
   or **record-mode** (each incoming record evaluated
   independently). Per
   [ADR-0021](../adr/0021-mode-as-primitive.md). The
   linter cross-checks the value against `_owners.yaml`'s
   entry for the same entity (cross-check #3) and against
   the rule's kind prefix (cross-check #4).
2. **`source`** — declares the per-entity substrate
   binding (project / dataset / table for BigQuery;
   topic / consumer-group / window for Kafka). Per
   [ADR-0023](../adr/0023-source-descriptor.md). Replaces
   the v1-era engine-wide `SourceProject` /
   `SourceDataset` fields (which were removed during the
   Wave-S refactor).
3. **Kind prefix** — every check's `kind` value now
   carries a mode-prefix: `set.row_count_positive`,
   `record.schema_conformance`. Per
   [ADR-0022](../adr/0022-kind-catalog.md). The linter
   cross-checks the prefix matches the rule's `mode`
   (cross-check #4) and that the kind exists in the
   catalog (`engine/internal/dsl/catalog/v1.yaml`;
   cross-check #5).

### Field map (set-mode example)

| v1 field | v2 equivalent | Notes |
|---|---|---|
| `version: 1` | `version: 2` | The lexical bump itself. |
| `entity: <name>` | `entity: <name>` | Unchanged. |
| (implicit set-mode) | `mode: set` | Now explicit per ADR-0021. |
| (not present) | `source.type: bigquery` | Required per ADR-0023. |
| (not present) | `source.project_id / dataset_id / table_id` | Required for BigQuery sources. |
| (optional, runtime-resolved) | `source.partition_column` | Optional partition-pruning signal per ADR-0029 / B2-12. |
| `checks[].kind: row_count_positive` | `checks[].kind: set.row_count_positive` | Mode-prefixed per ADR-0022. |
| `checks[].check_id: <id>` | `checks[].check_id: <id>` | Unchanged. |
| `checks[].params: {...}` | `checks[].params: {...}` | Unchanged at the wire level; the linter now validates against the catalog's `params_schema` for each kind (cross-check #6). |
| `description` (top-level + per-check) | `description` (top-level + per-check) | Unchanged. |

### Concrete before / after

The v1 → v2 migration of `rules/customer.yaml` (B2-19,
PR #72) is the canonical example.

**Before (v1):**

```yaml
version: 1
entity: customer
description: First onboarded entity end-to-end (W3-P6d).
checks:
  - check_id: row_count_positive
    kind: row_count_positive
    description: Verifies the source table has at least one row.
```

**After (v2):**

```yaml
version: 2
entity: customer
mode: set
description: First onboarded entity end-to-end (W3-P6d).
source:
  type: bigquery
  project_id: dq-local
  dataset_id: dq_fixture
  table_id: customer
checks:
  - check_id: row_count_positive
    kind: set.row_count_positive
    description: Verifies the source table has at least one row.
```

### What you also have to check before migrating

The `_owners.yaml` file's entry for the entity must declare
the same `mode` value the rule does. If `_owners.yaml` is
behind, the linter fires cross-check #3
(`rule mode … does not match owners entry mode …`).
`_owners.yaml` migrated from v1 to v2 in its own pass
during the Wave-S β slice — at the time of this writing
both production entities (`customer`, `orders_stream`)
already carry their `mode` fields.

The manifest publisher's `-supported-schema-versions` flag
controls which versions the manifest's
`schema_versions_present` is allowed to carry. Until the
v1-drop engine release lands, the publisher accepts both
(`-supported-schema-versions "1,2"`). The
`scripts/smoke/demo-p6.sh` invocation already passes this;
operators publishing manually with `dq-manifest publish`
must include the flag explicitly. The engine's running
binary checks `SupportedSchemaVersions` against the
manifest's `schema_versions_present`; mismatch fails closed
at loader time per ADR-0001 §4.

### What changes at runtime

- The engine continues to accept v1 and v2 rules in
  parallel until the v1-drop engine release lands (gated
  on the 90-day floor + the migration of any production
  v1 rules). No behaviour change for the migrated rule
  beyond the new `source` descriptor now being honored at
  evaluation time (the v1 path used the engine-wide
  source binding).
- The `dq-dryrun` binary, which skips v1 rules with a
  "v1 schema has no source descriptor" reason, will now
  exercise the migrated rule as a real dry-run target
  (the source descriptor is what dryrun reads to compile
  the BigQuery dry-run query). Bytes-scanned figures from
  the local emulator are not authoritative per
  [ADR-0029](../adr/0029-bigquery-cost-ceilings.md); the
  sandbox / real-BQ-cred lane is the production-fidelity
  surface.
- Existing `dq_executions` rows for the entity are
  unchanged — the migration affects future evaluations
  only. The append-only contract on `dq_executions` per
  [ADR-0003](../adr/0003-result-write-model.md) §CC1
  means historical rows stay valid evidence forever.

---

## Operator workflow

The end-to-end sequence for migrating one rule:

1. **Update the YAML** following the v1 → v2 field map.
2. **Update `_owners.yaml`** if the entity's `mode` entry
   is absent or wrong. The linter cross-check #3 will
   catch this; correcting it locally before pushing avoids
   a PR-time round-trip.
3. **`dq-lint -rules rules/`** — confirm zero validation
   errors. The DEPRECATED warning for the rule disappears
   the moment the rule's `version` field becomes `2`.
4. **`dq-manifest publish`** — re-publish the manifest with
   the new rule body. The publisher's
   `-supported-schema-versions` must include `2` (or
   `1,2` until the v1-drop engine release lands).
   Successful publish writes a new manifest body whose
   `schema_versions_present` includes `2` and atomically
   advances the pointer file via the
   `ifGenerationMatch` CAS path
   ([ADR-0005](../adr/0005-manifest-publication-semantics.md)).
5. **Engine loader refreshes** within the next
   `LoaderRefreshInterval` (2s local / 30s qa+prod per
   `engine/internal/env/{local,qa,prod}.go`). New
   evaluations use the migrated rule; in-flight
   evaluations under the prior manifest finish per
   [ADR-0007](../adr/0007-loader-scheduler-retry-failure-semantics.md)
   §3.

The migration is **online** — no engine restart needed.
Existing `dq_executions` rows from before the migration
stay intact; the rule's new execution_ids reflect the new
ruleset version per ADR-0002's five-input hash.

---

## Engine-pinning escape hatch

Operators whose rule inventory cannot complete migration
before the deprecated version's drop release **pin to an
older engine** until they finish. Per ADR-0035 §"Per-
deployment escape hatch":

> Operators with migration cycles longer than 90 days pin
> to an older engine (i.e., the engine release before the
> v(N-1)-drop release) until their migration completes.

The pinning surface is the engine container's image tag in
the deployment overlay. `deploy/overlays/{qa,prod}/`
references `dq-engine:<tag>`; pin to the last release that
declared the deprecated version in its
`SupportedSchemaVersions`.

The platform commits **only** the 90-day floor as the
default migration window. Per-deployment extensions are
not surfaced as a config field — they land via
engine-version pinning, not via env config. This keeps the
loader's authority-of-truth single-sourced (the manifest
contract per ADR-0001 §4).

---

## Forward pattern (future v(N) → v(N+1) deltas)

When v3 ships, this guide gains a §"v2 → v3 delta" section
written in the same shape as §"v1 → v2 delta":

- A field map (old field → new field; "removed" /
  "unchanged" rows for the boundary cases).
- A concrete before / after example.
- "What you also have to check" — companion-file
  consistency requirements (owners; catalog; manifest
  flags).
- "What changes at runtime" — performance / cost /
  semantic deltas the operator should anticipate.

The §"Compatibility-state model" section is amended to
reflect the new lifecycle state: v2 transitions from
`current` to `deprecated`, gaining an `Earliest drop`
date computed from the 90-day floor anchored at v3's first
manifest publish. The
[`tools/lint/compatibility.go`](../../tools/lint/compatibility.go)
`SchemaCompatibility` map updates byte-equivalently in the
same PR as the ADR amendment.

A future `tools/migrate` binary (registered as B2-23) may
land to automate field renames + structural transforms,
emitting the migrated YAML for `tools/migrate -from=v(N)
-to=v(N+1) rules/<path>.yaml`. Until then, rule authors
update YAML manually per this guide's field map. The
escape hatch via engine pinning makes the manual approach
operationally workable.

---

## Maturity disclaimer

This is a **seed**. The current document covers the v1 → v2
delta in concrete detail because that's the only migration
the platform has executed; the §"Forward pattern" section
codifies the convention for future deltas but is otherwise
untested. Ops feedback during the next major-version
migration is the source of truth for sharpening this
guide.
