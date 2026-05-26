<!-- path: studies/decisions/2026-05-25-b1-7-compatibility-window-duration.md -->

# B1-7 — Compatibility Window Duration

## Context

[ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
committed the engine ↔ rules compatibility contract: the engine
ships a canonical schema at
`engine/internal/dsl/schema/v<N>.schema.json`; the rules workspace
mirrors it at `rules/_schema/v<N>.schema.json`; a byte-equality CI
gate enforces the mirror invariant; manifests declare
`schema_versions_present` so the loader can fail-closed when an
engine encounters a version it does not support. ADR-0001 also
committed the additive-within-major contract — new optional fields
ship without a version bump, breaking changes require a new major
schema version.

What ADR-0001 did NOT commit:

- **How long is each schema version supported** after a successor
  ships? When v3 lands, does v1 remain valid forever, or get
  dropped after some window?
- **What is the migration path** for rule authors moving rules
  from v(N) to v(N+1)?
- **What is the drop mechanism** — engine-version change, manifest
  signal, lint flag, or other?

The current platform state is concrete:

- The engine binary's loader carries
  `SupportedSchemaVersions: []int{1, 2}` (per the Wave-S runtime
  slice α in `cmd/dq-engine/main.go`).
- Two schema versions ship on disk: `v1.schema.json` (legacy
  shape) and `v2.schema.json` (mode + source + per-check params per
  ADRs 0021–0024).
- The production `rules/` directory carries **one v1 rule** and
  **one v2 rule**: `customer.yaml` remains at `version: 1` (the
  Wave-S β slice's "Option (b) — defer rule migration"
  operator choice explicitly left it on v1); `orders_stream.yaml`
  ships at `version: 2` (the inaugural v2 record-mode rule from
  the Wave-S β slice). Both schemas have live consumers — v1 is
  not abandoned, only deprecated by virtue of v2 being current.

What B1-7 must commit:

1. **The window-duration policy.** When a new schema version is
   released, how long does the immediately prior version stay
   supported?
2. **The drop mechanism.** Which artefact gates the
   removal — the engine binary's `SupportedSchemaVersions`
   array, a manifest signal, or something else?
3. **The migration support level.** What tooling / documentation
   helps rule authors migrate?
4. **The relationship to ADR-0001's additive-within-major
   contract.** Additive changes within a major version don't
   trigger the window; breaking changes (new major) do.

The principles bearing on the decision are **P5** (evolution
must be contract-driven — the compatibility window is a
documented contract operators rely on), **P4** (cost is a
first-class constraint — indefinite multi-version support
amplifies engine code complexity over time), and **R3** (do
not revisit settled architecture — ADR-0001's compatibility
contract is preserved; B1-7 extends it with concrete numbers,
not by reopening it).

---

## Decision Drivers

- **DD-1 — ADR-0001 committed the framework; B1-7 supplies the
  numbers.** ADR-0001 commits the structural contract
  (canonical + mirror + byte-equality + manifest version
  declarations + additive-within-major). It does not commit the
  calendar-time window during which a deprecated version stays
  supported. B1-7 is an extension, not a reopen of ADR-0001.
- **DD-2 — The window must be predictable for rule authors.**
  A migration target with no documented deadline pushes
  decisions into ad-hoc judgment ("is v1 still supported?
  let's check the latest engine commit"). A committed window
  gives rule authors a planning horizon.
- **DD-3 — The window must be bounded for engine
  maintainers.** Indefinite multi-version support means the
  engine carries every parser forever — code complexity grows
  monotonically. A bounded window limits the dual-support
  surface to (current + previous) at any point in time.
- **DD-4 — The drop mechanism must be engine-binary-bound.**
  The loader's `SupportedSchemaVersions` array is the single
  source of truth for "which versions does this engine
  binary accept". Dropping a version requires an engine release
  (incrementing the binary's `EngineVersion` per ADR-0001's
  manifest `engine_compatibility` field). Any other mechanism
  (manifest signal, lint flag, runtime config) creates a
  separate authority and risks inconsistency.
- **DD-5 — A calendar-time floor protects operators from
  fast-moving releases.** N-1 alone (always support last 2
  versions) could mean v1 is dropped immediately if v3 ships
  shortly after v2. A calendar-time floor (e.g., 90 days)
  ensures operators always have a known minimum migration
  window regardless of release cadence.
- **DD-6 — Additive-within-major doesn't trigger the window.**
  Per ADR-0001's additive contract, a new optional field on
  v2 (e.g., the `schedule` field from ADR-0033) does not
  ship a new major version. The compatibility window applies
  only to MAJOR schema-version transitions.
- **DD-7 — Migration support is documentation-grade at v1.**
  The platform does not ship an automated rule-migration
  tool; rule authors update their YAML manually per the new
  schema. A future enhancement could ship a `tools/migrate`
  binary; reserved for B2.

---

## Considered Options

### Option 1 — N-1 with 90-day calendar-time floor (recommended)

When schema version v(N+1) ships and reaches stable status,
the engine continues to support v(N-1) for at least 90 days
after v(N+1)'s first manifest publish. After the floor
elapses AND v(N+1) is stable, the engine release that drops
v(N-1) lands. At any point in time the engine supports the
current schema version + the immediately prior version
(N-1 policy), with the 90-day floor ensuring at least a
quarter of migration runway after each new major release.

Concrete example (timeline):

- T+0: v1 stable. Engine supports `{1}`.
- T+M1: v2 released. Engine supports `{1, 2}`. v1 is
  marked deprecated in ADR-0001's compatibility table.
- T+M1+90d: earliest possible v1 removal. The release that
  drops v1 from `SupportedSchemaVersions` lands at any
  point after this; operators have ≥90 days of dual-support
  runway.
- T+M2: v3 released. Engine supports `{2, 3}` (v1 already
  dropped). The 90-day floor restarts for v2.
- And so on for future major versions.

**Strengths.** Bounded engine complexity (at most two
parsers active at any time); predictable migration window
(at least 90 days regardless of release cadence); aligns
with the engine-binary-bound drop mechanism (DD-4);
preserves ADR-0001's structural contract (DD-1).

**Trade-offs.** Operators with very long migration cycles
(e.g., a quarterly release calendar) may need the 90-day
floor extended for specific contexts; this is operationally
addressable via a per-deployment policy (operator pins to
an older engine until migration completes) — the platform's
default is 90d, but operators have escape hatches.

### Option 2 — Indefinite multi-version support

The engine carries every schema parser forever. v1 stays
supported indefinitely even after v2, v3, v4 ship.

**Strengths.** Operators never have to migrate (in
principle). Maximally permissive.

**Trade-offs.** Engine code complexity grows monotonically.
Every parser carries its own validation logic, its own
`ToCheckSpecs` translation, its own test surface. The
fixture-tree convention from ADR-0034 grows
`testdata/v1/`, `testdata/v2/`, `testdata/v3/`, ... forever.
Bug-fix surface area amplifies linearly with version count.
Security review surface (PII patterns vary across schema
shapes) becomes harder. P4 (cost) is violated: engine
maintenance cost grows without a discipline ceiling.
Rejected.

### Option 3 — Fixed N (current + N-1) with no time floor

Always support the current version + the immediately prior
version, with no calendar-time floor. v(N+1) ships → v(N-1)
drops in the next engine release, regardless of how soon
after v(N+1)'s release.

**Strengths.** Maximally simple posture: "the engine always
supports the last two versions."

**Trade-offs.** A fast-moving release cadence (v2 ships
Monday; v3 ships Friday of the same week) means v1's
migration window is effectively zero. Operators get no
predictable runway. The platform's release discipline would
have to commit to a minimum release-interval to make this
operator-friendly, which is harder to enforce than a
calendar-time floor on the compatibility window. Rejected
in favor of Option 1's combined N-1 + floor.

### Option 4 — Manifest-driven sunset

When no manifest in any deployment's history references
v(N-1) for some period (e.g., 60 days), the engine release
that drops v(N-1) can ship. This requires monitoring
deployments' manifest histories from a central observability
surface.

**Strengths.** Empirical drop signal — v(N-1) is only
dropped when no one is using it.

**Trade-offs.** Requires central observability of
manifest publications across all deployments — not currently
a platform capability (each deployment manages its own
manifests; ADR-0005 commits the publication primitive
per-deployment, not across deployments). Pulls a new
multi-deployment-observation surface into the platform
that doesn't exist. The signal also doesn't account for
deployments planning to onboard v(N-1)-using rules in the
future. Rejected — the empirical signal is appealing but
the implementation cost is too high for a 90-day-floor
problem.

---

## Recommendation

**Option 1.** N-1 policy with a 90-day calendar-time floor.

### Window-duration policy

The platform commits the following compatibility contract
as an extension of ADR-0001:

- **N-1 baseline.** The engine binary supports the current
  schema version and the immediately prior version at any
  point in time.
- **90-day floor after each major release.** When schema
  version v(N+1) is released and reaches stable status,
  v(N-1) remains in `SupportedSchemaVersions` for at least
  90 days from v(N+1)'s first manifest publish.
- **Drop mechanism: engine release.** The engine release
  that drops v(N-1) from `SupportedSchemaVersions` lands at
  any operator-chosen time AFTER the 90-day floor elapses.
  The drop is an engine-binary change (the
  `SupportedSchemaVersions` array literal in
  `cmd/dq-engine/main.go` changes from `{N-1, N, N+1}` to
  `{N, N+1}`). Manifests declaring v(N-1) thereafter cause
  loader failure per ADR-0007 CC1 (startup mode) or
  refuse-swap per ADR-0007 CC2 (refresh mode).
- **Additive changes don't trigger the window.** A new
  optional field added to v(N) per ADR-0001's
  additive-within-major contract does not start the
  90-day clock. Only **major schema-version transitions**
  (v1 → v2; v2 → v3) trigger the compatibility window.

### Concrete current state and projected timeline

Current state at this ADR's acceptance:

- Engine `SupportedSchemaVersions: {1, 2}`.
- v1 is the legacy version; v2 is current. v1 was the only
  shipping version pre-Wave-S; v2 shipped with the Wave-S
  schema slice (ADR-0021–0028) and runtime slices α + β.
- v1 has no live rules in `rules/` at this ADR's acceptance
  (`customer.yaml` and `orders_stream.yaml` are both v2).
- v1's 90-day floor began with v2's first manifest publish
  (the Wave-S β slice's `rules/_owners.yaml` schema_version
  2 + `rules/orders_stream.yaml` v2 publication).

Projected v1 retirement:

- **Floor elapses:** approximately 90 days after the Wave-S
  β slice's first manifest publish (PR #40 merged
  2026-05-25), so **earliest possible v1 removal is
  2026-08-23**. The operational session that schedules the
  v1-drop engine release lands the
  `SupportedSchemaVersions: []int{2}` change at any time
  after the floor — provided no live v1 rules remain.
- **v1 retirement gates on customer.yaml migration.** The
  surviving v1 rule (`rules/customer.yaml`) must migrate
  to v2 before the v1-drop engine release ships; otherwise
  the manifest publication that the operator runs would
  fail at the loader. The customer.yaml migration is a B2
  follow-up registered below, independent of the
  v1-retirement engine release.
- **Sequencing:** customer.yaml migration → next manifest
  publish carrying v2-only rules → 90-day floor elapses
  (already met at 2026-08-23 for v2's first publish) →
  engine release dropping v1 from
  `SupportedSchemaVersions`. The two B2 follow-ups
  (customer.yaml migration; v1-drop engine release) can
  ship in either order so long as the engine release
  follows or coincides with the manifest carrying v2-only
  rules.

### ADR-0001 compatibility table extension

ADR-0001 commits the structural contract; B1-7 extends it
with the compatibility table that operators read to know
the current version-support state. The table lives at the
top of [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
or as an appendix; the close-step decides the exact home.

```
| Schema version | Status     | Engine support since | Earliest drop |
|----------------|------------|----------------------|---------------|
| v1             | deprecated | 0.1.0 (Wave-3 P3)    | 2026-08-23    |
| v2             | current    | 0.1.0 (Wave-S α)     | TBD (when v3 ships) |
```

The "Earliest drop" column is calculated from the
90-day floor + v(N+1)'s first manifest publish; operators
read it to plan migrations. The column is human-maintained
in ADR-0001; the close-step amends ADR-0001's body to
include the table (or appends an addendum section, per
R3's "don't revisit settled architecture without strong
cause" — the table is additive content, not a revision of
ADR-0001's commitments).

### Migration support level

v1 ships:

- **Documentation-grade migration guidance.** A new section
  in `docs/dev/local-testing.md` or a new
  `docs/dev/schema-migration.md` documents the v(N) → v(N+1)
  delta. The first instance — v1 → v2 — is already documented
  implicitly in ADRs 0021–0024 (the new fields v2
  introduces); a future `docs/dev/schema-migration.md`
  consolidates them as a migration playbook.
- **Deprecation warning at lint time** (deferred to B2). A
  future enhancement to `tools/lint` could emit a warning
  when a rule's `version` field declares a deprecated
  schema (per the ADR-0001 compatibility table). The
  warning would fire on v1 rules once v2 is current; it
  becomes a hard error once v1's 90-day floor elapses (at
  which point the engine release dropping v1 also drops
  lint support — symmetric posture).
- **No automated migration tool at v1.** Rule authors
  update their YAML manually per the v(N+1) schema. A
  future `tools/migrate` binary could automate
  field-renames + structural transforms; reserved for B2.

### Relationship to ADR-0001's additive-within-major contract

ADR-0001 commits that additive changes within a major
version (e.g., adding an optional `schedule` field per
ADR-0033) do not require a new schema version. The
compatibility window applies **only to major
schema-version transitions** (v1 → v2; v2 → v3, etc.) —
NOT to additive changes within v2.

Mechanism for additive changes:

- The JSON schema gains the new optional field.
- The byte-equality CI gate enforces canonical ↔ mirror
  byte-equality after the new field lands.
- Existing rules without the field continue to validate
  (the field is optional).
- No drop occurs; v2 is still v2.

Mechanism for major-version transitions (this row's
surface):

- A new schema version ships (`v(N+1).schema.json` +
  mirror).
- The engine loader's `SupportedSchemaVersions` gains
  v(N+1).
- v(N-1) enters the 90-day-floor countdown from v(N+1)'s
  first manifest publish.
- After the floor, an engine release drops v(N-1).

### Why this does NOT reopen ADR-0001

ADR-0001 commits the structural compatibility contract:
canonical + mirror + byte-equality gate + manifest
declarations + additive-within-major. B1-7 supplies the
**calendar-time numbers** the contract was missing — it
extends the contract, not amends its structure.

ADR-0001's body changes only in the compatibility-table
appendix described above; the additive-within-major
section, the byte-equality section, the manifest-version
section are all unchanged. The close-step decides whether
the compatibility table lands as a new appendix to
ADR-0001 or as ADR-0035's own table referenced by
ADR-0001 (cleaner per R8 forward-only — ADR-0035 owns
its table; ADR-0001 stays stable). I'll recommend the
latter at close-step.

---

## Consequences

1. **No engine code change ships from this ADR.** The
   90-day-floor policy and the N-1 baseline are
   **commitments** that govern future engine releases.
   The next engine release that drops v(N-1) lands per the
   B2 follow-up below (currently green-field — v1 has no
   live rules; the drop is operator-scheduled).

2. **A compatibility-state table ships in ADR-0035** (this
   row's promotion target). The table lists each schema
   version, its status (current / deprecated / dropped),
   the engine version that introduced support, and the
   earliest drop date. Operators read the table to plan
   migrations. The table is human-maintained; each new
   schema version's release also amends the table.

3. **B2 follow-up: `customer.yaml` v1 → v2 migration.** A
   new B2 row registers the rule-author work that
   migrates `rules/customer.yaml` to v2 (adds `mode: set`,
   `source: { type: bigquery, ... }`, prefixed kind
   `set.row_count_positive`). The migration ships under
   the standard rules-author PR-review path per ADR-0015.
   Gating: must land before the v1-retirement engine
   release (Consequence 4) so the post-retirement manifest
   does not carry any v1 rule.

4. **B2 follow-up: v1 retirement engine release.** A new B2
   row registers the engine release that drops v1 from
   `SupportedSchemaVersions`. The release lands at any
   operator-chosen time after the 90-day floor elapses
   (~2026-08-23 at the earliest) AND after `customer.yaml`
   has migrated to v2 (Consequence 3). The release ships:
   - `SupportedSchemaVersions: []int{2}` in
     `cmd/dq-engine/main.go`.
   - Removal of v1's parser path in
     `engine/internal/dsl/spec/parse.go` (`validateV1`,
     `validateCheckV1`).
   - Removal of `engine/internal/dsl/schema/v1.schema.json`
     and its `rules/_schema/v1.schema.json` mirror.
   - Removal of v1 fixtures under `testdata/valid/` +
     `testdata/invalid/` (the v1-specific tests in
     `lint_test.go`, `spec_test.go`).
   - Engine version bump (major or minor per the operator's
     release-engineering call).
   - Updated compatibility-state table in ADR-0035.

5. **B2 follow-up: deprecation warning at lint time.** A
   future enhancement to `tools/lint` emits a warning when
   a rule's `version` field declares a deprecated schema
   (per the ADR-0035 compatibility table). The warning
   fires on v1 rules until v1 is dropped; the warning
   becomes a hard error at the v1-drop engine release.
   B2 row registered for the implementation.

6. **B2 follow-up: `docs/dev/schema-migration.md`.** A
   new dev guide consolidating the v1 → v2 migration
   delta (and future v2 → v3 delta when v3 ships). Lands
   under `docs/dev/` (introduced by ADR-0034); operator-
   readable migration playbook. B2 row registered.

7. **B2 follow-up: `tools/migrate` binary.** A future
   automated rule-migration tool. Rule authors run
   `tools/migrate -from=v1 -to=v2 rules/customer.yaml`
   and the tool emits the migrated YAML. Out of scope
   for v1; B2 row reserved.

8. **The additive-within-major contract from ADR-0001 is
   preserved.** Additive optional fields (e.g., the
   `schedule` field from ADR-0033, the `params.baseline`
   block fragment from ADR-0032) ship within v2 without a
   new schema version; no compatibility-window clock starts
   for them.

9. **The drop mechanism is engine-binary-bound.** The
   loader's `SupportedSchemaVersions` array is the single
   source of truth for "which versions does this engine
   accept". No manifest signal, no lint flag, no runtime
   config; the engine binary's version determines the
   supported-set, and operators pin engine versions per
   their migration timeline.

10. **The platform's P4 + P5 commitments for schema
   versioning are now explicit.** P4 (cost): bounded
   engine complexity at most two parsers active at any
   time. P5 (contract-driven): the compatibility window
   is a documented contract operators rely on.

11. **The platform's posture toward backward compatibility
    is explicit.** v1 rules continue to parse for at least
    90 days after v2's first manifest publish; the
    operational session that schedules v1's drop owns the
    timing. Rule authors who fall behind the 90-day floor
    pin to an older engine until migration completes —
    this escape hatch is documented at close-step.

---

## Open Questions

None blocking.

Two deferred items surfaced during drafting and are
explicitly **out-of-scope for current cycle**:

- **OQ-1: Per-deployment override of the 90-day floor.** A
  future amendment could ship a per-env config field
  (`EnvConfig.SchemaCompatibilityWindowExtension`) that
  operators set to extend the floor for specific
  deployments (e.g., a slow-moving production env with
  quarterly migration windows). Reserved until concrete
  operational signal shows the default 90d is too short
  for a deployment; pinning to an older engine release is
  the v1 escape hatch.

- **OQ-2: Schema-version forward-compatibility.** A future
  amendment could ship "engine supports v(N+1) before
  v(N+1)'s rules ship in production" — i.e., the engine
  accepts the future schema version ahead of operator
  rollout. Reserved until a concrete need surfaces (e.g.,
  a phased rollout where engine ships first and rule
  artefacts catch up). v1 commits the simpler
  current+previous posture.

---

## Promotion target

`docs/adr/0035-compatibility-window-duration.md` — ships the
N-1 + 90-day-floor policy, the engine-binary-bound drop
mechanism, the compatibility-state table (initially
listing v1 deprecated + v2 current), and four B2 follow-ups
(v1 retirement engine release; lint deprecation warning;
`docs/dev/schema-migration.md`; `tools/migrate` binary).
