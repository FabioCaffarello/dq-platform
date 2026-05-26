<!-- path: docs/adr/0035-compatibility-window-duration.md -->

# ADR-0035 — Compatibility Window Duration

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

[ADR-0001](./0001-engine-rules-compatibility.md) committed the
engine ↔ rules compatibility contract: the engine ships a
canonical schema at `engine/internal/dsl/schema/v<N>.schema.json`;
the rules workspace mirrors it at
`rules/_schema/v<N>.schema.json`; a byte-equality CI gate
enforces the mirror invariant; manifests declare
`schema_versions_present` so the loader can fail-closed when
an engine encounters a version it does not support. ADR-0001
also committed the additive-within-major contract — new
optional fields ship without a version bump; breaking changes
require a new major schema version.

What ADR-0001 did NOT commit:

- **How long is each schema version supported** after a
  successor ships?
- **What is the migration path** for rule authors moving
  rules from v(N) to v(N+1)?
- **What is the drop mechanism** — engine-version change,
  manifest signal, lint flag, or other?

This ADR supplies those numbers and that mechanism as an
**extension** of ADR-0001's structural contract — not as a
revision. ADR-0001's canonical + mirror + byte-equality +
manifest-version-declarations + additive-within-major
commitments are all preserved.

The current platform state at this ADR's acceptance:

- The engine binary's loader carries
  `SupportedSchemaVersions: []int{1, 2}` (per the Wave-S
  runtime slice α in `cmd/dq-engine/main.go`).
- Two schema versions ship on disk: `v1.schema.json` (legacy
  shape) and `v2.schema.json` (mode + source + per-check
  params per ADRs 0021–0024).
- The production `rules/` directory carries **one v1 rule**
  (`customer.yaml`) and **one v2 rule**
  (`orders_stream.yaml`). The Wave-S runtime slice β
  explicitly deferred `customer.yaml`'s migration to v2;
  both schemas have live consumers.

The principles bearing on the decision are **P5** (evolution
must be contract-driven — the compatibility window is a
documented contract operators rely on), **P4** (cost is a
first-class constraint — indefinite multi-version support
amplifies engine code complexity over time), and **R3** (do
not revisit settled architecture — ADR-0001's compatibility
contract is preserved; this ADR extends it with concrete
numbers).

---

## Decision

### N-1 baseline with a 90-day calendar-time floor

The engine binary supports the **current schema version
plus the immediately prior version** at any point in time
(the N-1 baseline). When a new major schema version
v(N+1) is released and reaches stable status, the engine
continues to support v(N-1) for **at least 90 days from
v(N+1)'s first manifest publish** before an engine
release can drop v(N-1) from `SupportedSchemaVersions`.

Concrete timeline:

- T+0: v(N) stable. Engine supports `{N}`.
- T+M1: v(N+1) released. Engine supports `{N, N+1}`.
  v(N) is marked deprecated in the compatibility-state
  table below.
- T+M1+90d: earliest possible v(N) removal. The release
  that drops v(N) from `SupportedSchemaVersions` lands
  at any point after this; operators have ≥90 days of
  dual-support runway from the moment v(N+1)'s first
  manifest publishes.
- T+M2: v(N+2) released. Engine supports `{N+1, N+2}`
  (v(N) already dropped). The 90-day floor restarts for
  v(N+1).

The 90-day floor is calendar-time, not release-count.
A fast-moving cadence (v(N+2) shipping shortly after
v(N+1)) does not shorten v(N)'s window — the operator
always has at least 90 days to migrate.

### Drop mechanism: engine binary release

The loader's `SupportedSchemaVersions` array in
`cmd/dq-engine/main.go` is the single source of truth
for which schema versions the engine accepts. Dropping
a version requires an **engine binary release** (a new
`EngineVersion` per ADR-0001's manifest
`engine_compatibility` field). No other artefact — not
a manifest signal, not a lint flag, not a runtime
config — can drop a version; all paths converge on the
engine binary as the authority.

Manifests declaring a dropped version cause loader
failure per [ADR-0007](./0007-loader-scheduler-retry-failure-semantics.md)
CC1 (startup mode) or refuse-swap per ADR-0007 CC2
(refresh mode). Operators planning a migration coordinate
their manifest publication with their engine release
cadence to avoid mid-window failures.

### Additive changes don't trigger the window

Per ADR-0001's additive-within-major contract, a new
optional field added to v(N) (e.g., the `schedule` field
from [ADR-0033](./0033-scheduler-catchup-behavior.md),
the `params.baseline` block fragment from
[ADR-0032](./0032-baseline-strategy.md)) does not ship
a new major schema version. The compatibility window
applies **only to major schema-version transitions** —
not to additive changes within a major version.

Mechanism for additive changes:

- The JSON schema gains the new optional field.
- The byte-equality CI gate enforces canonical ↔ mirror
  byte-equality after the new field lands.
- Existing rules without the field continue to validate.
- No drop occurs; v(N) stays v(N).

Mechanism for major-version transitions:

- A new schema version v(N+1) ships (canonical +
  mirror).
- The engine loader's `SupportedSchemaVersions` gains
  v(N+1).
- v(N-1) enters the 90-day-floor countdown from
  v(N+1)'s first manifest publish.
- After the floor, an engine release drops v(N-1).

### Compatibility-state table

| Schema version | Status | Engine support since | Earliest drop |
|---|---|---|---|
| v1 | deprecated | 0.1.0 (Wave-3 P3) | 2026-08-23 |
| v2 | current | 0.1.0 (Wave-S α) | TBD (when v3 ships) |

The "Earliest drop" column is calculated from the
90-day floor + v(N+1)'s first manifest publish (PR #40,
merged 2026-05-25 → 2026-08-23 for v1). Operators read
this table to plan migrations. The table is amended on
each new major release.

### Current state and v1 retirement sequencing

v1 has one live consumer at this ADR's acceptance:
`rules/customer.yaml` is at `version: 1` because the
Wave-S runtime slice β explicitly deferred its
migration. The v1-drop engine release therefore gates
on **two preconditions**:

1. The 90-day floor elapsed (earliest 2026-08-23).
2. `customer.yaml` migrated to v2 (or removed from the
   production rules workspace).

Sequencing:

- **Step 1** — `customer.yaml` migrates to v2 under a
  standard rules-author PR (registered as a B2
  follow-up below).
- **Step 2** — The next manifest publish carries v2-only
  rules; `schema_versions_present` becomes `[2]`.
- **Step 3** — After ~2026-08-23 (90-day floor met),
  the engine release dropping v1 lands (registered as
  a B2 follow-up below).

The two B2 rows are independent: customer.yaml can
migrate before, during, or after the floor elapses;
the engine release lands when both preconditions are
met. Until then, the engine accepts both v1 and v2 and
the dual-support code surface stays bounded at one
extra parser path.

### Migration support level at v1

- **Documentation-grade migration guidance.** A future
  `docs/dev/schema-migration.md` consolidates the
  v1 → v2 delta (and future v(N) → v(N+1) deltas) as a
  migration playbook. Lands under `docs/dev/`
  (introduced by ADR-0034). Registered as a B2
  follow-up below.
- **Deprecation warning at lint time** — a future
  `tools/lint` enhancement emits a warning when a
  rule's `version` field declares a deprecated schema
  (per this ADR's compatibility-state table). The
  warning fires on deprecated-but-supported versions;
  it becomes a hard error at the version's drop
  release. Registered as a B2 follow-up below.
- **No automated migration tool at v1.** Rule authors
  update YAML manually per the new schema. A future
  `tools/migrate` binary could automate field renames
  + structural transforms; out of scope for v1;
  registered as a B2 follow-up below.

### Why this does NOT reopen ADR-0001

ADR-0001 commits the structural compatibility contract:
canonical + mirror + byte-equality + manifest version
declarations + additive-within-major. This ADR supplies
the **calendar-time numbers** ADR-0001 was missing
(90-day floor) and the **drop mechanism** (engine
binary's `SupportedSchemaVersions`). ADR-0001's body is
unchanged; this ADR's compatibility-state table is the
new authoritative reference for current version-support
state.

### Per-deployment escape hatch

Operators with migration cycles longer than 90 days
**pin to an older engine** (i.e., the engine release
before the v(N-1)-drop release) until their migration
completes. The platform commits the 90-day floor as
the default; per-deployment extensions land via
engine-version pinning, not via a per-env config
field. This keeps the loader's authority-of-truth
single-sourced.

A future amendment could ship a per-env config field
that extends the floor for specific deployments (see
OQ-1 below). Reserved until concrete operational signal
shows the default 90d is insufficient.

---

## Consequences

1. **No engine code change ships from this ADR.** The
   90-day-floor policy, the N-1 baseline, and the
   engine-binary-bound drop mechanism are
   **commitments** that govern future engine releases.

2. **The compatibility-state table is the authoritative
   reference for version-support state.** Each future
   major schema release amends this table to mark the
   newly-deprecated version + its earliest-drop date.
   The amendment lands in the release's ADR (e.g., a
   future ADR introducing v3 amends the table to mark
   v2 deprecated).

3. **B2 follow-up: `customer.yaml` v1 → v2 migration.**
   A new B2 row registers the rule-author work that
   migrates `rules/customer.yaml` to v2 (adds
   `mode: set`, `source: { type: bigquery, ... }`,
   prefixed kind `set.row_count_positive`). Standard
   rules-author PR-review path per
   [ADR-0015](./0015-codeowners.md). Gating: must land
   before the v1-retirement engine release (Consequence
   4) so the post-retirement manifest carries no v1
   rule.

4. **B2 follow-up: v1 retirement engine release.** A
   new B2 row registers the engine release that drops
   v1 from `SupportedSchemaVersions`. Lands at any
   operator-chosen time after the 90-day floor elapses
   (~2026-08-23 at the earliest) AND after
   `customer.yaml` has migrated. The release ships:
   - `SupportedSchemaVersions: []int{2}` in
     `cmd/dq-engine/main.go`.
   - Removal of v1's parser path in
     `engine/internal/dsl/spec/parse.go` (`validateV1`,
     `validateCheckV1`).
   - Removal of `engine/internal/dsl/schema/v1.schema.json`
     and its `rules/_schema/v1.schema.json` mirror.
   - Removal of v1 fixtures under `testdata/valid/` +
     `testdata/invalid/` (and the v1-specific tests in
     `lint_test.go`, `spec_test.go`).
   - Engine version bump (operator-engineering call).
   - Updated compatibility-state table.

5. **B2 follow-up: deprecation warning at lint time.**
   A future `tools/lint` enhancement emits a warning
   when a rule's `version` field declares a deprecated
   schema (per the compatibility-state table). Fires
   on deprecated-but-supported versions; becomes a
   hard error at the version's drop release. B2 row
   registered.

6. **B2 follow-up: `docs/dev/schema-migration.md`.**
   A new dev guide consolidating the v1 → v2 migration
   delta (and future deltas as they arise). Lands
   under `docs/dev/` (introduced by ADR-0034).
   B2 row registered.

7. **B2 follow-up: `tools/migrate` binary.** A future
   automated rule-migration tool. Rule authors run
   `tools/migrate -from=v1 -to=v2 rules/customer.yaml`
   and the tool emits the migrated YAML. Out of scope
   for v1; B2 row reserved.

8. **The additive-within-major contract from ADR-0001
   is preserved.** Additive optional fields (e.g., the
   `schedule` field from ADR-0033, the
   `params.baseline` block fragment from ADR-0032)
   ship within v(N) without a new major version. No
   compatibility-window clock starts for them.

9. **The drop mechanism is engine-binary-bound.** The
   loader's `SupportedSchemaVersions` array is the
   single source of truth. No manifest signal, no lint
   flag, no runtime config can change which versions
   the engine accepts; the engine binary's version
   determines the supported-set.

10. **The platform's P4 + P5 commitments for schema
    versioning are now explicit.** P4 (cost): bounded
    engine complexity — at most two parsers active at
    any time. P5 (contract-driven): the compatibility
    window is a documented contract operators rely on
    for migration planning.

11. **The platform's posture toward backward
    compatibility is explicit.** Deprecated versions
    continue to parse for at least 90 days after the
    successor's first manifest publish; operators with
    longer migration cycles pin to an older engine
    release until migration completes — this escape
    hatch is documented above.

---

## Notes

- The 90-day floor is a starting commitment. Concrete
  operational signal (an incident, audit finding, or
  consistent operator complaint that 90 days is too
  short) would justify amending the floor — under the
  same study → critique → promotion protocol that
  governed this ADR. Until then, 90d is the platform's
  contract.
- The compatibility-state table is human-maintained.
  A future enhancement could auto-generate it from the
  engine binary's `SupportedSchemaVersions` + manifest
  metadata; reserved until concrete drift signal shows
  manual maintenance is failing.
- Per-deployment overrides of the 90-day floor are not
  shipped at v1. Operators with longer migration
  cycles pin to older engine releases. A future
  amendment may ship a per-env config field if the
  pinning escape hatch proves cumbersome (see OQ-1).
