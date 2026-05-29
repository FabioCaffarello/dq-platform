<!-- path: docs/adr/0050-v1-retirement-engine-release.md -->

# ADR-0050 — v1-Retirement Engine Release

- **Status:** accepted
- **Date:** 2026-05-29

---

## Context

[ADR-0035](./0035-compatibility-window-duration.md) committed three
artifacts on 2026-05-25: the compatibility-state table as
authoritative reference for version-support state, the 90-day
calendar-time floor as the minimum-cohabitation period for a
deprecated schema version, and the drop mechanism as an engine binary
release that removes the deprecated version from the loader's
`SupportedSchemaVersions` array. ADR-0035 §Consequence 4 explicitly
deferred two operational decisions to a follow-up: *when* the
v1-drop release lands, and the mechanical sequencing of code, test,
and fixture removal.

Both preconditions ADR-0035 placed on the follow-up are now in a
defined state:

- **Precondition 1 — the 90-day floor.** v1 was committed as
  `deprecated` on 2026-05-25 with earliest-drop **2026-08-23**.
- **Precondition 2 — `customer.yaml` migrated.** The production
  rules workspace carries no v1 consumer; the next manifest publish
  carries v2-only rules and `schema_versions_present: [2]` (per
  [ADR-0005](./0005-manifest-publication-semantics.md)).

The first precondition is a calendar gate: it bounds the earliest
possible release date but does not specify the actual release date.
The second is satisfied. What remains is to fix the operational
parameters over ADR-0035 §Consequence 4's enumerated scope so the
release itself is a mechanical execution against a committed plan,
not a fresh design exercise on the day it lands.

Four parameter classes are open:

1. **Release-date selection rule.** ADR-0035 says "any operator-
   chosen time after the 90-day floor elapses" — that permits a
   range, not a date. This ADR commits to either a specific date
   or a selection rule that turns the range into a single date when
   the floor is met.
2. **Engine version bump.** Current `EngineVersion` is `0.1.0`
   across `engine/internal/env/{local,qa,prod}.go`. The v1-drop is
   a breaking change for unpinned operators.
   [ADR-0012](./0012-tag-conventions.md) commits the semver shape
   but not the bump magnitude for a contract drop.
3. **Compatibility-state table transition.** The table currently
   carries v1 as `deprecated`. After the drop, v1 must transition
   to a terminal status. ADR-0035's table does not yet enumerate
   that status's name or row shape.
4. **Test-surface treatment.** Test sites outside
   `cmd/dq-engine/main.go` declare `SupportedSchemaVersions:
   []int{1}` (loader unit + integration; manifest publisher unit +
   integration). Most are testing the support-set mechanism
   generically (a loader that accepts version *N* — *N* happens to
   be 1) rather than v1 specifically. The drop must distinguish
   "delete because v1-specific" from "update because mechanism-
   generic with v1 chosen as example".

The principles bearing on this decision are **P4** (cost is a
first-class constraint — the dual-support code surface carries
operational cost for negligible capability gain after the customer
migration), **P5** (evolution must be contract-driven — the
compatibility-state table is the long-lived contract surface that
future v(N) retirements depend on), and **R3** in
[`CLAUDE.md`](../../CLAUDE.md) §3 (do not revisit settled
architectural decisions — ADR-0035's 90-day floor, ADR-0001's
structural compatibility contract, ADR-0005's
`schema_versions_present` field shape, and ADR-0012's `engine-v*`
tag-prefix semver shape all stand as committed and are not amended
here).

---

## Decision

### (a) Release-date selection rule

**Rule (forward-applicable to all future schema-version
retirements):** the v(N)-drop release ships on the **first weekday
on or after the v(N) floor's expiry date**, subject to the operating
constraint that the date is not covered by a declared incident-
response window or a declared merge-freeze. If either applies, the
release shifts to the next weekday for which neither applies.

The rule is self-contained: it does not depend on a separate
release-cadence ADR. If platform-wide release cadence is codified in
a future ADR, that ADR may revisit this rule; until then, "first
weekday on or after floor expiry" is the operative rule and stands
without an implicit upstream.

**Concrete for v1 drop:** floor 2026-08-23 (Sunday) → first weekday
on or after is **2026-08-24** (Monday). Day-of-week verified
2026-05-29: `date -j -f "%Y-%m-%d" "2026-08-23" "+%A"` → `Sunday`;
`2026-08-24` → `Monday`. Absent a declared incident or merge-freeze
covering that date as the floor approaches, **B2-20 release target
is 2026-08-24**.

### (b) Engine version bump

**Rule (forward-applicable):**

- **Pre-1.0 era.** A release that drops a previously-supported
  schema version is at minimum a **minor bump**
  (`X.Y.Z` → `X.(Y+1).0`). Patch bumps are reserved for no-
  contract-change releases.
- **Post-1.0 era** (when the platform declares 1.0). A release
  that drops a previously-supported schema version is at minimum
  a **major bump** (`X.Y.Z` → `(X+1).0.0`), reflecting the semver
  promise that contract-incompatible changes carry major-version
  bumps.

**Concrete for v1 drop:** current is `0.1.0`; drop ships as
**`0.2.0`**. The bump propagates atomically to
`engine/internal/env/{local,qa,prod}.go` in the drop PR.

**No premature 1.0 declaration.** Declaring 1.0 attaches a forward-
looking stability promise to the engine binary that this drop does
not by itself justify. 1.0 is reserved for a separate, focused
decision when the platform is ready to commit post-1.0 semver
semantics to downstream consumers — not as a side effect of the
first schema-version retirement.

### (c) Compatibility-state table transition

Per **P5** (evolution must be contract-driven), the compatibility-
state table committed by ADR-0035 §Consequence 2 is the long-lived
contract surface that future v(N) retirements depend on; its
evolution mechanism must itself be a committed contract.

**Status-name addition.** The compatibility-state table gains a
terminal status **`retired`**. The full status ladder becomes:

| Status | Semantics |
|---|---|
| `current` | Engine accepts; lint accepts without warning; default for newly-introduced versions |
| `deprecated` | Engine accepts; lint emits warning; in the 90-day floor period (or beyond) before drop |
| `retired` | Engine rejects at validation time; lint may emit historical note; entry retained for historical lookup |

**Row-shape addition.** The retired row gains a `dropped` field
populated with the release date and the engine version (e.g.,
`2026-08-24 / engine-v0.2.0`). The `earliest_drop` field is
retained as historical context. Future v(N) retirements follow the
same row shape.

**ADR-0035 amendment mechanism — in-place amendment, committed.**
The table-row transition (v1: `deprecated` → `retired`, plus the
new `dropped` field) lands as an in-place edit to ADR-0035's
compatibility-state table section, accompanied by a dated "Amendment
log" sub-section at the bottom of ADR-0035 recording (a) the
amendment date, (b) the row affected (v1), (c) the rationale
(ADR-0050 ships the retirement).

**Rationale for in-place over a superseding ADR:**

- ADR-0035 §Consequence 2 commits the table as the authoritative
  reference for version-support state and states "Each future major
  schema release amends this table". The table is structurally
  promised as the long-lived single source of truth; an in-place
  amendment preserves that single source, whereas a superseding
  ADR would fragment the truth across multiple files (operators
  reading ADR-0035's compatibility-state table would not see the
  v1-retired status without following a pointer).
- The change is bookkeeping over a committed structure, not a
  scope change. The table's columns, status ladder, and row shape
  are all committed by ADR-0035; this ADR only writes a row
  transition into the pre-existing structure. New ADR ceremony
  for a row update is disproportionate.
- The [ADR-0017](./0017-substrate-posture-amendment.md) precedent
  does not apply. ADR-0017 is a separate follow-up ADR that amends
  [ADR-0010](./0010-substrate-posture.md), but it reclassified a
  *capability row in a taxonomy table* (substrate object-store CAS
  row transitioning from `Local-only` to `Partial`) — a scope-of-
  locally-testable change with downstream architectural prose
  implications. ADR-0050 transitions a *compatibility-state row*
  (a version's lifecycle marker) — different surface, different
  mechanism. ADR-0017's superseding-ADR shape was appropriate for
  taxonomy reclassification with prose impact; ADR-0050's in-place
  shape is appropriate for lifecycle bookkeeping with no prose
  impact.

**Amendment log convention — new contribution.** No prior ADR
amendment in this repository uses an "Amendment log" sub-section.
This ADR commits the convention so future v(N) retirements have a
uniform mechanism to record their amendment metadata without re-
amending the amendment chain itself. Acceptance of this ADR commits
the convention.

### (d) Test-surface cohort split

The `SupportedSchemaVersions: []int{1}` test sites split into two
cohorts. The distinguishing test: a site is **mechanism-generic**
iff the test would pass with any non-empty supported-version set
and the choice of `[]int{1}` is incidental. A site is **v1-specific**
iff the test asserts behavior tied to v1's parser, schema, or
fixtures. The drop PR's reviewer enforces this distinction.

**Cohort 1 — v1-specific (delete in drop PR):**

- v1 schema mirror tests covering `validateV1` / `validateCheckV1`
  in `engine/internal/dsl/spec/parse_test.go` (the file matched by
  ADR-0035 §Consequence 4's `validateV1` removal).
- v1 fixtures under `tools/lint/testdata/valid/` +
  `tools/lint/testdata/invalid/` and the v1-specific tests in
  `tools/lint/lint_test.go`.

**Cohort 2 — mechanism-generic (update in drop PR):**

- `engine/internal/loader/loader_test.go` — `SupportedSchemaVersions:
  []int{1}` used as an example of "a non-empty set"; update to
  `[]int{2}`.
- `engine/internal/loader/loader_integration_test.go` — same
  pattern; update.
- `tools/manifest/publisher_test.go` (including
  `TestNew_RequiresSupportedSchemaVersions`) — same pattern;
  update.
- `tools/manifest/publisher_integration_test.go` — same pattern;
  update.

### (e) `tools/lint/compatibility.go` post-drop

The `SchemaCompatibility` map keeps the v1 entry; its `Status`
flips from `deprecated` to `retired`, and a `DroppedRelease` field
is populated (`"engine-v0.2.0"`) alongside the existing
`EngineSupportSince` and `EarliestDrop` fields.

`CheckDeprecatedSchemaVersions` is unchanged in its current form —
it walks `rules/` looking for `version: <deprecated-N>` files and
emits warnings. Post-drop, no production rule carries `version: 1`,
so the walker emits no warning for the production workspace. The
historical lookup behavior of `SchemaVersionStatus(1)` is preserved:
it now returns the `retired` entry, which downstream callers can
use for diagnostics or migration tooling.

The engine's pre-lint dispatcher rejects v1 at validation time
before the manifest reaches the lint stage in any production path.
The `retired` row in the lint table therefore exists primarily as
documentation and as a safety net for tooling that walks the rules
directory directly (out-of-band of the engine).

### (f) Operator pre-announcement

**None beyond ADR-0035 publication.** The 90-day floor and the drop
mechanism are committed by ADR-0035. The compatibility-state table
is the operator-facing single source of truth. Manifest
`schema_versions_present` carries `[2]` since the post-migration
publish. No separate broadcast, email, or release-notes pre-
announcement is required for v1.

Future v(N) retirements may warrant pre-announcement if v(N)
carries production consumers outside the platform's own `rules/`
workspace at the time of deprecation. v1 had only the platform's
own consumer, which migrated before this ADR was accepted; no
external consumer-comm surface exists to broadcast to.

### Drop PR shape

The drop ships as a **single PR** carrying all six ADR-0035
§Consequence 4 artifacts plus the table amendment plus the
publisher-config update:

- `engine/cmd/dq-engine/main.go` — `SupportedSchemaVersions: []int{1, 2}`
  → `[]int{2}`. One-line edit.
- `engine/internal/env/local.go` / `qa.go` / `prod.go` —
  `EngineVersion: "0.1.0"` → `"0.2.0"`. Three-file edit; one line
  each.
- `engine/internal/dsl/spec/parse.go` — remove `validateV1` /
  `validateCheckV1` plus their dispatch entry; the v2 path becomes
  the only path.
- `engine/internal/dsl/spec/parse_test.go` (and adjacent
  `spec_test.go` if separate) — v1-specific test cases removed
  (Cohort 1).
- `engine/internal/dsl/schema/v1.schema.json` +
  `rules/_schema/v1.schema.json` — removed. The mirror invariant
  per [ADR-0001](./0001-engine-rules-compatibility.md) holds
  because both artifacts are removed atomically.
- `tools/lint/testdata/valid/` + `tools/lint/testdata/invalid/` +
  `tools/lint/lint_test.go` (v1-specific cases) — v1 fixtures and
  their tests removed (Cohort 1).
- `engine/internal/loader/loader_test.go` /
  `loader_integration_test.go` — Cohort 2 update.
- `tools/manifest/publisher_test.go` /
  `publisher_integration_test.go` — Cohort 2 update.
- `tools/lint/compatibility.go` — v1 entry `Status` flips to
  `retired`; new `DroppedRelease` field added with `engine-v0.2.0`.
  No row removal.
- `docs/adr/0035-compatibility-window-duration.md` — in-place
  amendment to the compatibility-state table plus dated entry in
  the new "Amendment log" subsection at the bottom of the ADR.
- Publisher-config range update — the publisher-config file
  carrying the `engine_compatibility` range gains `0.2.0` as a
  satisfied version. Concrete file path depends on the publisher's
  config-load surface; the drop PR's reviewer enforces alignment.

The drop PR is gated on a green CI sweep: all per-module test
sweeps (engine, tools/lint, tools/manifest, tools/dryrun,
tools/pathsafe), `make validate-deploy`, and `make dry-run-rules`
against the local BigQuery emulator must be clean before merge.

---

## Consequences

1. **The drop ships as a single reviewable PR.** All eleven
   artifact changes — engine main, env triplet, parser, schema
   pair, fixtures, two test cohorts, lint compatibility table,
   ADR-0035 amendment, publisher-config — land together. Reviewers
   receive one PR to assess; splitting across PRs invites a half-
   dropped state that the loader cannot represent.

2. **`0.1.0` → `0.2.0` is the engine bump.** The manifest
   publisher's `engine_compatibility` field range is a hand-
   maintained publisher-config field (per
   [ADR-0005](./0005-manifest-publication-semantics.md) §manifest
   body fields), so the drop PR includes the publisher-config
   update alongside the engine version bump — both ship in the
   same commit. The updated range must include `0.2.0` as a
   satisfied version; whether to also remove `0.1.0` from the
   range is an operator call at publish time (typical: extend
   upper bound; do not remove the lower bound mid-cycle). No
   ADR-0005 amendment is needed.

3. **The compatibility-state table gains a `retired` status that
   future retirements reuse.** Every future v(N) drop release
   lands the same shape: row gains `dropped: <date> /
   <engine-version>` and `Status` transitions to `retired`. No
   new status names are introduced for future retirements unless
   this precedent is explicitly revisited.

4. **The "Amendment log" subsection convention is committed.**
   Future ADR amendments that touch only structured data
   (compatibility-state rows, taxonomy rows, configuration
   matrices) may follow this convention to record amendment
   metadata in-place without spawning a new ADR. Amendments with
   architectural-prose impact follow the
   [ADR-0017](./0017-substrate-posture-amendment.md) superseding-
   ADR pattern instead.

5. **ADR-0001 / ADR-0005 / ADR-0012 are not amended.** Structural
   compatibility (canonical + mirror + byte-equality), the
   `schema_versions_present` field shape, and the `engine-v*`
   tag-prefix semver shape all stand as committed. The drop
   operates inside their envelope.

6. **The 90-day floor's precedent solidifies.** Hitting the floor
   +1 weekday locks in the operational expectation that future
   deprecations follow the same cadence (deprecate-on-publish +
   90-day calendar floor + next-weekday release). Future
   deviations from this cadence become explicit policy decisions
   that supersede this ADR's §(a).

7. **Post-drop verification.** The first post-merge manifest
   publish carries `schema_versions_present: [2]` (unchanged from
   the post-migration publish that closed the customer-rule
   migration). The first engine startup logs `engine_version:
   0.2.0`. A v1 rule introduced after the drop (e.g., by a hand-
   edited test) is rejected by the loader with a clear error
   citing the dropped version.

8. **Forward pointer on the inverse direction.** v1-drop is not B3
   evolutionary work — it is implementation-phase work scheduled
   by ADR-0035. Future v(N+1) *introductions* (the inverse of this
   retirement) fall under
   [ADR-0049](./0049-b3-evolutionary-launch.md) §(b) Out-of-scope
   F (API evolution) and require their own launch — not a B3
   entry. The retirement-side precedent is committed by this ADR;
   the introduction-side mechanism is reserved.

9. **Operational deferrals.** Three operational parameters remain
   open beyond this ADR: (i) whether a future platform-wide
   release-cadence ADR revisits the §(a) selection rule; (ii)
   whether a declared incident or merge-freeze shifts the
   2026-08-24 date — operator call closer to the date; (iii) the
   pre-1.0 minor-vs-patch boundary for adjacent changes (e.g., a
   parser-only refactor preserving behavior). None of these
   affect the v1 drop itself, which is unambiguously minor under
   any reading of §(b).
