<!-- path: studies/decisions/2026-05-29-b2-20-v1-retirement-engine-release.md -->

# B2-20 — v1-Retirement Engine Release

## Metadata

- **Wave reference:** B2 (post-Wave-3 follow-up; sequenced against
  ADR-0035's 90-day floor).
- **Status:** resolved-study (B2-20; one critique round; round 1
  cleared with no blocking findings).
- **Last updated:** 2026-05-29.
- **Critique rounds:** 1 preserved
  ([`studies/critiques/2026-05-29-b2-20-v1-retirement-engine-release-critique-1.md`](../critiques/2026-05-29-b2-20-v1-retirement-engine-release-critique-1.md)).
- **Upstream resolved:**
  [ADR-0035](../../docs/adr/0035-compatibility-window-duration.md)
  (compatibility window duration — 90-day floor, drop mechanism,
  compatibility-state table), B2-19 (`rules/customer.yaml` v1 → v2
  migration, shipped 2026-05-26, removes the last v1 consumer from
  the production rules workspace).
- **Constraint envelope:**
  [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)
  (structural compatibility — canonical + mirror + byte-equality),
  [ADR-0005](../../docs/adr/0005-manifest-publication-semantics.md)
  (`schema_versions_present` manifest field),
  [ADR-0012](../../docs/adr/0012-tag-conventions.md) (`engine-v*`
  tag prefix + semver shape),
  [ADR-0035](../../docs/adr/0035-compatibility-window-duration.md)
  (this is the originating ADR; B2-20 ships its Consequence 4).
- **Locked premises** (operator-declared, not litigated here):
  - **P-B220.1** — *90-day floor is non-negotiable.* The earliest
    drop date is 2026-08-23 (ADR-0035 §"N-1 baseline with a 90-day
    calendar-time floor"). Any release date earlier than this is
    out of scope for B2-20.
  - **P-B220.2** — *Drop scope material is fixed by ADR-0035
    §Consequence 4.* The five-item inventory (loader
    `SupportedSchemaVersions`, v1 parser path, v1 schema +
    mirror, v1 fixtures, version bump, table amendment) is the
    binding scope. B2-20 fixes *operational parameters* over that
    scope; it does not redraw the scope.
  - **P-B220.3** — *Customer migration precondition is satisfied.*
    B2-19 shipped 2026-05-26. The production rules workspace
    carries no v1 consumer; the second of ADR-0035's two
    preconditions has been met.
  - **P-B220.4** — *Per-deployment escape hatch is engine-version
    pinning (ADR-0035 §"Per-deployment escape hatch"); B2-20 does
    not amend it.* Operators with migration cycles longer than 90
    days pin to the last pre-drop engine release. This study does
    not introduce a per-env config field.
- **Downstream open:** post-drop, `tools/lint/compatibility.go`'s
  `SchemaCompatibility` map row for v1 transitions from
  `deprecated` to `retired` (B2-21 already wired the table;
  retirement transition is in B2-20's scope per §(d) below).
- **Promotion target:**
  `docs/adr/0050-v1-retirement-engine-release.md` (provisional;
  the slot may shift if intervening ADRs land before promotion).
- **Loop discipline:** standard protocol — draft → `/critique`
  (≥1 round, preserved under `studies/critiques/` per ADR-0048)
  → operator acceptance → promotion to ADR.

---

## Context

ADR-0035 committed three artifacts on 2026-05-25: the compatibility-
state table as authoritative reference, the 90-day calendar-time
floor as the minimum-cohabitation period for a deprecated schema
version, and the drop mechanism as an engine binary release that
removes the deprecated version from `SupportedSchemaVersions`. It
explicitly deferred two operational decisions to a B2 follow-up
(Consequence 4): *when* the v1-drop release lands, and the
mechanical sequencing of code/test/fixture removal.

The two preconditions ADR-0035 placed on that follow-up are now in
a defined state:

- **Precondition 1 — the 90-day floor.** v1 was committed as
  `deprecated` on 2026-05-25 with earliest-drop 2026-08-23. As of
  this study (2026-05-29), the floor has 86 days remaining.
- **Precondition 2 — `customer.yaml` migrated.** B2-19 shipped
  2026-05-26 with the customer rule migrated atomically to v2;
  the next manifest publish carries v2-only rules and
  `schema_versions_present: [2]`.

The first precondition is a *calendar* gate: it bounds the earliest
possible release date but does not specify the actual release date.
The second is satisfied. What remains for B2-20 is to fix the
operational parameters over ADR-0035 §Consequence 4's enumerated
scope so the release itself is a mechanical execution against a
committed plan, not a fresh design exercise on the day it lands.

Four parameter classes are open:

1. **Release-date selection rule.** ADR-0035 says "any
   operator-chosen time after the 90-day floor elapses" — that
   permits a range, not a date. B2-20 must commit to either a
   specific date or a selection rule that turns the range into a
   single date when the floor is met.
2. **Engine version bump.** Current `EngineVersion` is `0.1.0`
   across `engine/internal/env/{local,qa,prod}.go`. The v1-drop is
   a breaking change for unpinned operators. ADR-0012 commits the
   semver shape but not the bump magnitude for a contract drop.
3. **Compatibility-state table transition.** The table currently
   carries v1 as `deprecated`. After the drop, v1 must transition
   to a terminal status. ADR-0035's table does not yet enumerate
   that status's name or row shape.
4. **Test-surface treatment.** Eight test sites outside
   `cmd/dq-engine/main.go` declare `SupportedSchemaVersions:
   []int{1}` (loader unit + integration; manifest publisher unit +
   integration). Most are testing the support-set mechanism
   generically (a loader that accepts version *N* — *N* happens to
   be 1) rather than v1 specifically. B2-20 must distinguish
   "delete because v1-specific" from "update because mechanism-
   generic with v1 chosen as example".

This study fixes all four. It does not amend any non-B2-20 surface
of ADR-0035 and does not reopen ADR-0001, ADR-0005, or ADR-0012.

---

## Decision Drivers

- **DD-B220.1 — Bounded engine complexity (P4).** The dual-support
  code surface (v1 + v2 parser paths, schema files, fixtures,
  tests) is bounded at one extra parser path today. Each calendar
  month it lives carries operational cost for negligible
  capability gain (B2-19 already migrated the only consumer). The
  release should land *promptly after the floor*, not drift.

- **DD-B220.2 — Operators rely on calendar-predictable drops
  (P5).** Operators reading ADR-0035 expect a default-90-day floor
  with the drop landing close to the floor's expiry. Padding the
  date beyond the floor for operator-comms reasons is justifiable
  only if a concrete operational reason surfaces; absent such a
  reason, the floor itself is the schedule.

- **DD-B220.3 — Drop mechanics must be reviewable as a single
  PR.** Per ADR-0035 §Consequence 4, the drop is five
  artifact-level removals plus a version bump plus a table
  amendment. A single PR carrying all six is reviewable; splitting
  across PRs invites a half-dropped state that the loader cannot
  represent.

- **DD-B220.4 — Test-surface preservation matters.** Tests that
  exercise the support-set *mechanism* using v1 as an example
  remain valuable post-drop; they describe loader/publisher
  behavior generically. Deleting them blindly leaves the support-
  set mechanism less-tested. Test-surface treatment is therefore
  a discriminating decision, not a mechanical cleanup.

- **DD-B220.5 — No premature 1.0 commitment.** Current engine
  version `0.1.0` carries no semver compatibility promise. A
  contract drop is the kind of change that *could* motivate
  declaring 1.0, but declaring 1.0 carries a separate promise
  (downstream consumers can depend on stability) that this drop
  does not by itself justify.

- **DD-B220.6 — The compatibility-state table is the long-lived
  contract artifact (P5).** Per P5 (evolution must be contract-
  driven), the table is the contract surface that governs how
  every future v(N) retirement is recorded. Future schema-version
  retirements repeat this exact transition. The status-name and
  row-shape chosen here become the precedent for v(N) retirement
  in the indefinite future. The choice must read cold to a future
  operator who is retiring v(N) and never saw the v1 drop.

---

## Considered Options

### Option A — Floor-minimum release (land 2026-08-23)

The release ships *on* the earliest-drop date the floor allows
(2026-08-23, a Sunday). The release-date selection rule is "the
calendar date the floor expires". Engine bump is operator-
engineering call separately; table transition uses an
operator-chosen terminal status name.

This option's force is calendar-predictability: every reader of
ADR-0035 can compute the date. Its weakness is operational
realism — landing a substantive release on a Sunday is unusual;
landing the next business day (Monday 2026-08-24) is more typical
but produces a 1-day-after-floor pattern that future retirements
might or might not honor.

### Option B — First weekday on or after floor expiry (recommended)

The release ships on the **first weekday on or after the floor's
expiry date, subject to no incident or merge-freeze covering that
day**. The rule is mechanical, calendar-predictable, and produces
a single date when the floor is met. For the v1 drop concretely:
floor expires 2026-08-23 (Sunday) → first weekday on or after is
2026-08-24 (Monday) → B2-20 release target is **2026-08-24**
(day-of-week verified 2026-05-29: `date -j -f "%Y-%m-%d"
"2026-08-23" "+%A"` → `Sunday`; `2026-08-24` → `Monday`).

The release-date rule generalizes for future retirements as a
single sentence with no implicit dependency on a release-cadence
ADR. If a future codification of platform-wide release cadence
lands, the rule can be revisited (see OQ-B220.1 below); until
then, the rule stands on its own.

Engine bump is fixed by the rule "any release that removes a
supported schema version is at minimum a minor bump pre-1.0
(`0.1.0` → `0.2.0`) and at minimum a major bump post-1.0
(`X.Y.Z` → `(X+1).0.0`)". The table transition name is **`retired`**,
mirroring the `deprecated` precedent (single past-participle verb).

### Option C — Criterion-based trigger (no committed date)

The release ships when a named trigger condition fires, e.g.,
"the next manifest publish that includes a v1 rule under
`rules/` shall fail the publisher's pre-publish check". The check
itself ships before the floor; the engine release lands on the
first weekday after the first such failure or after the floor,
whichever is later.

This option's force is *no v1 rule can sneak back in*: the trigger
is data-driven, not date-driven. Its weakness is operational
complexity (the publisher gains a check that exists only for the
duration of the v1 → v2 transition) and it does not actually
remove the date question — it merely defers it to "when the
trigger fires AND the floor has elapsed, on the first eligible
weekday".

---

## Recommendation

**Pick Option B — First weekday on or after floor expiry.**

Rationale, tied directly to drivers:

- **Bounded engine complexity (DD-B220.1)** — Option B lands the
  drop within 24 hours of the floor's expiry under the default
  weekday rule. Option A lands it on the floor date even if that
  is a Sunday, which is operationally awkward. Option C defers
  indefinitely on the trigger axis and is therefore the least
  prompt of the three.
- **Calendar predictability (DD-B220.2)** — Option B fixes a
  *single* date (2026-08-24) when the floor and the calendar are
  both known. Option A fixes a single date but ignores the
  release-window practice. Option C fixes no single date.
- **Reviewable single PR (DD-B220.3)** — all three options
  produce the same PR shape; they differ only in landing date.
  Not discriminating.
- **Test-surface preservation (DD-B220.4)** — orthogonal to the
  option choice; addressed in §(d) below.
- **No premature 1.0 (DD-B220.5)** — Option B's "minimum minor
  pre-1.0" rule preserves the current `0.1.0` line: the drop
  ships as `0.2.0`. Options A and C leave the bump magnitude
  open; this study would still need to commit a rule.
- **Long-lived table artifact (DD-B220.6)** — Option B commits
  **`retired`** as the terminal status name. Options A and C
  leave the name open.

**One-line decision summary table:**

| Decision | Outcome |
|---|---|
| Release date selection rule | First weekday on or after floor expiry, subject to no incident/merge-freeze (DD-B220.2) |
| Concrete v1 release date | **2026-08-24** (Monday after 2026-08-23 Sunday floor) |
| Engine version bump rule | Pre-1.0: minimum minor (`X.Y.Z` → `X.(Y+1).0`). Post-1.0: minimum major |
| Concrete v1 engine version | **`0.2.0`** (from current `0.1.0`) |
| Compatibility-state table transition | v1 row: `deprecated` → **`retired`**, with `dropped` field populated with release date and `engine-v0.2.0` |
| Drop PR shape | Single PR carrying all six ADR-0035 §C4 artifacts + table amendment |
| ADR-0035 amendment shape | **In-place amendment** to the compatibility-state table — committed (see §(c) rationale; ADR-0017 precedent does not apply because ADR-0017 reclassified a *taxonomy row*, whereas this is *bookkeeping over a committed structure* — different surfaces) |
| Test-surface treatment | v1-specific (fixtures, v1-tagged tests) deleted; mechanism-generic tests (loader/publisher support-set) updated to use `[]int{2}` as the example version |
| `tools/lint/compatibility.go` post-drop | v1 row marked `retired` in `SchemaCompatibility`; entry retained as historical lookup. Pre-lint engine dispatcher rejects v1; `CheckDeprecatedSchemaVersions` no-ops for retired versions |
| Operator pre-announcement | None beyond ADR-0035 publication. Manifest carries v2-only since B2-19; ADR-0035 is the formal commitment |

### (a) Release-date selection rule

**Rule (forward-applicable to all future schema-version
retirements):** the v(N)-drop release ships on the **first weekday
on or after the v(N) floor's expiry date**, subject to the
operating constraint that the date is not covered by a declared
incident-response window or a declared merge-freeze. If either
applies, the release shifts to the next weekday for which neither
applies.

The rule is self-contained: it does not depend on a separate
release-cadence ADR (none currently exists). If platform-wide
release cadence is codified in a future ADR, that ADR may revisit
this rule; until then, "first weekday on or after floor expiry"
is the operative rule and stands without an implicit upstream.

The rule produces a single date when the floor is known. For v1
concretely: floor 2026-08-23 (Sunday) → 2026-08-24 (Monday) →
**B2-20 target: 2026-08-24**, absent a declared incident or
merge-freeze covering that date as the floor approaches. Day-of-
week verified 2026-05-29 (`date -j -f "%Y-%m-%d" "2026-08-23"
"+%A"` → `Sunday`; `2026-08-24` → `Monday`).

### (b) Engine version bump

**Rule (forward-applicable):**

- **Pre-1.0 (current era).** A release that drops a previously-
  supported schema version is at minimum a minor bump
  (`X.Y.Z` → `X.(Y+1).0`). Patch bumps are reserved for
  no-contract-change releases.
- **Post-1.0 (future era, when the platform declares 1.0).** A
  release that drops a previously-supported schema version is at
  minimum a major bump (`X.Y.Z` → `(X+1).0.0`), reflecting the
  semver promise that contract-incompatible changes carry major-
  version bumps.

**Concrete for v1 drop:** current is `0.1.0`; drop ships as
**`0.2.0`**. The bump propagates atomically to
`engine/internal/env/{local,qa,prod}.go` in the drop PR.

**Why not declare 1.0 now?** Declaring 1.0 attaches a
forward-looking stability promise to the engine binary that this
drop does not by itself justify. 1.0 should be declared by a
separate, focused decision when the platform is ready to commit
post-1.0 semver semantics to downstream consumers — not as a side
effect of the first schema-version retirement.

### (c) Compatibility-state table transition

Per **P5** (evolution must be contract-driven), the compatibility-
state table is the long-lived contract surface that future v(N)
retirements depend on; its evolution mechanism must itself be a
committed contract.

**Status-name addition:** the compatibility-state table gains a
terminal status **`retired`**. The full status ladder becomes:

| Status | Semantics |
|---|---|
| `current` | Engine accepts; lint accepts without warning; default for newly-introduced versions |
| `deprecated` | Engine accepts; lint emits warning; in the 90-day floor period (or beyond) before drop |
| `retired` | Engine rejects at validation time; lint may emit historical note; entry retained for historical lookup |

**Row-shape amendment:** the v1 row gains a `dropped` field
populated with the release date and the engine version (e.g.,
`2026-08-24 / engine-v0.2.0`). The `earliest_drop` field is
retained as historical context. Future v(N) retirements follow
the same row shape.

**ADR-0035 amendment mechanism — in-place amendment, committed.**
The table-row transition (v1: `deprecated` → `retired`, plus the
new `dropped` field) lands as an in-place edit to ADR-0035's
compatibility-state table section, accompanied by a dated
"Amendment log" sub-section at the bottom of ADR-0035 recording
(a) the amendment date, (b) the row affected (v1), (c) the
rationale (B2-20 ships the retirement).

**Rationale for in-place over superseding-ADR:**

- **ADR-0035 §Consequence 2 commits the table as the
  authoritative reference for version-support state** and states
  "Each future major schema release amends this table". The table
  is structurally promised as the long-lived single source of
  truth; an in-place amendment preserves that single source,
  whereas a superseding ADR would fragment the truth across
  multiple files (operators reading ADR-0035 §Compatibility-state
  table would not see the v1-retired status without following a
  pointer).
- **The change is bookkeeping over a committed structure**, not a
  scope change. The table's columns, status ladder, and row shape
  are all committed by ADR-0035; B2-20 only writes a row
  transition into the pre-existing structure. New ADR ceremony
  for a row update is disproportionate.
- **The ADR-0017 precedent does not apply.** ADR-0017 is a
  separate follow-up ADR that amends ADR-0010
  ([`docs/adr/0017-substrate-posture-amendment.md`](../../docs/adr/0017-substrate-posture-amendment.md)
  §"Status: accepted (amends ADR-0010)"). ADR-0017 reclassified a
  *capability row in a taxonomy table* (substrate object-store
  CAS row transitioning from `Local-only` to `Partial`) — a
  scope-of-locally-testable change with downstream architectural
  prose implications. B2-20 transitions a *compatibility-state
  row* (a version's lifecycle marker) — different surface,
  different mechanism. ADR-0017's superseding-ADR shape was
  appropriate for taxonomy reclassification with prose impact;
  B2-20's in-place shape is appropriate for lifecycle bookkeeping
  with no prose impact.

**Amendment log convention — new contribution proposed here,
requires review.** No prior ADR amendment in this repository
uses an "Amendment log" sub-section. B2-20 proposes this
convention so future v(N) retirements have a uniform mechanism
to record their amendment metadata without re-amending the
amendment chain itself. The convention is proposed; reviewers
may accept it as committed by ADR-0050 acceptance or flag it
for separate decision.

### (d) Test-surface treatment

The eight `SupportedSchemaVersions: []int{1}` test sites split
into two cohorts:

**Cohort 1 — v1-specific (delete in drop PR):**

- v1 schema mirror tests covering `validateV1` /
  `validateCheckV1` in `engine/internal/dsl/spec/parse_test.go`
  (the file matched by ADR-0035 §C4's `validateV1` removal).
- v1 fixtures under `testdata/valid/` + `testdata/invalid/` and
  the v1-specific tests in `lint_test.go` / `spec_test.go` (per
  ADR-0035 §C4).

**Cohort 2 — mechanism-generic (update in drop PR):**

- `engine/internal/loader/loader_test.go` — `SupportedSchemaVersions: []int{1}`
  used as an example of "a non-empty set"; update to `[]int{2}`.
- `engine/internal/loader/loader_integration_test.go` — same
  pattern; update.
- `tools/manifest/publisher_test.go` (including
  `TestNew_RequiresSupportedSchemaVersions`) — same pattern;
  update.
- `tools/manifest/publisher_integration_test.go` — same pattern;
  update.

The distinguishing test: a site is **mechanism-generic** iff the
test would pass with any non-empty supported-version set and the
choice of `[]int{1}` is incidental. A site is **v1-specific** iff
the test asserts behavior tied to v1's parser, schema, or
fixtures. The drop PR's reviewer enforces this distinction.

### (e) `tools/lint/compatibility.go` post-drop

The `SchemaCompatibility` map keeps the v1 entry; its `Status`
flips from `deprecated` to `retired`, and a `DroppedRelease`
field is populated (`"engine-v0.2.0"`) alongside the existing
`EngineSupportSince` and `EarliestDrop` fields.

`CheckDeprecatedSchemaVersions` is unchanged in its current form —
it walks `rules/` looking for `version: <deprecated-N>` files and
emits warnings. Post-drop, no production rule carries `version: 1`
(B2-19 closed the last one), so the walker emits no warning for
the production workspace. The historical lookup behavior of
`SchemaVersionStatus(1)` is preserved: it now returns the
`retired` entry, which downstream callers can use for diagnostics
or migration tooling.

The engine's pre-lint dispatcher rejects v1 at validation time
before the manifest reaches the lint stage in any production path.
The `retired` row in the lint table therefore exists primarily as
documentation and as a safety net for tooling that walks the rules
directory directly (out-of-band of the engine).

### (f) Operator pre-announcement

**None beyond ADR-0035 publication.** The 90-day floor and the
drop mechanism are committed by ADR-0035. The compatibility-state
table is the operator-facing single source of truth. Manifest
`schema_versions_present` has carried `[2]` since B2-19's first
post-migration publish. No separate broadcast, email, or release-
notes pre-announcement is required for v1.

Future v(N) retirements may warrant pre-announcement if v(N)
carries production consumers other than the platform's own
`rules/` workspace at the time of deprecation. v1 had only the
platform's own consumer (`customer.yaml`), which migrated under
B2-19; no external consumer-comm surface exists to broadcast to.

---

## Consequences

### Cross-cutting

- **C-B220.1 — A single PR carries the entire drop.** The drop PR
  contains: `cmd/dq-engine/main.go` `SupportedSchemaVersions`
  edit; `env/{local,qa,prod}.go` `EngineVersion` bump;
  `engine/internal/dsl/spec/parse.go` `validateV1` /
  `validateCheckV1` removal; `engine/internal/dsl/schema/v1.schema.json`
  + `rules/_schema/v1.schema.json` removal; v1 fixtures + v1
  tests removal under `testdata/`; mechanism-generic test
  updates (Cohort 2); `tools/lint/compatibility.go` status flip +
  `DroppedRelease` field; ADR-0035 compatibility-state table
  amendment. Reviewers receive one PR to assess.

- **C-B220.2 — `0.1.0` → `0.2.0` is the engine bump.** The
  manifest publisher's `engine_compatibility` field range is a
  hand-maintained publisher-config field (per ADR-0005 §manifest
  body fields), so the drop PR includes the publisher-config
  update alongside the engine version bump — both ship in the
  same commit. The updated range must include `0.2.0` as a
  satisfied version; whether to also remove `0.1.0` from the
  range is an operator call at publish time (typical: extend
  upper bound; do not remove the lower bound mid-cycle). No
  ADR-0005 amendment is needed.

- **C-B220.3 — The compatibility-state table gains a `retired`
  status that future retirements reuse.** v(N) drop releases land
  the same shape: row gains `dropped: <date> / <engine-version>`
  and `Status` transitions to `retired`. No new status names are
  introduced for future retirements unless this study's precedent
  is explicitly revisited.

- **C-B220.4 — ADR-0001 / ADR-0005 / ADR-0012 are not amended.**
  Structural compatibility (canonical + mirror + byte-equality),
  the `schema_versions_present` field shape, and the `engine-v*`
  tag-prefix semver shape all stand as committed. The drop
  operates inside their envelope.

- **C-B220.5 — The 90-day floor's precedent solidifies.**
  Hitting the floor +1 weekday locks in the operational
  expectation that future deprecations follow the same cadence
  (deprecate-on-publish + 90-day calendar floor + next-weekday
  release). Future deviations from this cadence become explicit
  policy decisions. **Forward pointer:** v1-drop is *not* B3
  work — it is implementation-phase work scheduled by ADR-0035,
  not capability extension. Future v(N+1) *introductions* (the
  inverse direction of this retirement) fall under
  [ADR-0049](../../docs/adr/0049-b3-evolutionary-launch.md) §(b)
  Out-of-scope F (API evolution) and require their own
  launch — not a B3 entry.

### Per-artifact (fully-qualified paths)

- **`engine/cmd/dq-engine/main.go`** — `SupportedSchemaVersions:
  []int{1, 2}` → `[]int{2}`. One-line edit.
- **`engine/internal/env/local.go` / `qa.go` / `prod.go`** —
  `EngineVersion: "0.1.0"` → `"0.2.0"`. Three-file edit; one
  line each.
- **`engine/internal/dsl/spec/parse.go`** — remove `validateV1` /
  `validateCheckV1` plus their dispatch entry; the v2 path
  becomes the only path.
- **`engine/internal/dsl/spec/parse_test.go`** (and adjacent
  `spec_test.go` if separate) — v1-specific test cases removed
  (Cohort 1).
- **`engine/internal/dsl/schema/v1.schema.json`** +
  **`rules/_schema/v1.schema.json`** — removed. The mirror
  invariant per ADR-0001 holds because both artifacts are
  removed atomically.
- **`tools/lint/testdata/valid/`** + **`tools/lint/testdata/invalid/`**
  + **`tools/lint/lint_test.go`** (v1-specific cases) — v1
  fixtures and their tests removed (Cohort 1). Verify the cohort
  split against `tools/lint/testdata/valid/customer.yaml` (the
  v1 fixture currently used as the live exercise surface for
  the B2-21 deprecation walker).
- **`engine/internal/loader/loader_test.go` /
  `loader_integration_test.go`** — mechanism-generic test sites
  updated from `[]int{1}` to `[]int{2}` (Cohort 2).
- **`tools/manifest/publisher_test.go` /
  `publisher_integration_test.go`** — same as above (Cohort 2).
- **`tools/lint/compatibility.go`** — v1 entry `Status` flips to
  `retired`; new `DroppedRelease` field added with `engine-v0.2.0`.
  No row removal.
- **`docs/adr/0035-compatibility-window-duration.md`** — in-place
  amendment to the compatibility-state table plus dated entry in
  a new "Amendment log" subsection at the bottom of the ADR (per
  the new-contribution convention in §(c)).
- **Publisher-config range update** (per C-B220.2) — the
  publisher-config file carrying the `engine_compatibility`
  range gains `0.2.0` as a satisfied version. Concrete file path
  depends on the publisher's config-load surface; the drop PR's
  reviewer enforces alignment.

### Process

- **C-B220.6 — The drop PR is gated on a green CI sweep.** All
  per-module test sweeps (engine, tools/lint, tools/manifest,
  tools/dryrun, tools/pathsafe), `make validate-deploy`, and
  `make dry-run-rules` against the local BigQuery emulator must
  be clean before merge.

- **C-B220.7 — Post-drop verification.** The first post-merge
  manifest publish carries `schema_versions_present: [2]`
  (unchanged from B2-19's first post-migration publish). The
  first engine startup logs `engine_version: 0.2.0`. A v1 rule
  introduced after the drop (e.g., by a hand-edited test) is
  rejected by the loader with a clear error citing the dropped
  version.

- **C-B220.8 — No new B2 follow-ups are registered.** The drop
  consumes the last open item against ADR-0035's Consequence 4.
  The compatibility-window-duration ADR's open follow-ups
  inventory closes with this B2-20 row's `resolved-adr`
  transition.

---

## Open Questions

- **OQ-B220.1 — Platform-wide release-cadence codification.**
  The §(a) rule stands self-contained today. If a future ADR
  codifies platform-wide release cadence (e.g., regular release
  windows, change-freeze windows, deployment calendars), that
  ADR may revisit the §(a) rule. *Out-of-scope for current
  cycle.* For v1's drop, "first weekday on or after floor
  expiry" is unambiguous and the date (2026-08-24) is committed.

- **OQ-B220.2 — Incident/merge-freeze override at the drop
  date.** The §(a) rule allows shifting the date if a declared
  incident or merge-freeze covers 2026-08-24. *Out-of-scope for
  current cycle — defer to the operator's call closer to the
  date.* The rule is committed; its application is operational.

- **OQ-B220.3 — Pre-1.0 minor-bump vs. patch-bump distinction.**
  The §(b) rule says contract drops are "at minimum a minor
  bump pre-1.0". The boundary between "patch-eligible no-
  contract-change" and "minor-required contract drop" is sharp
  for schema-version retirement (clearly minor) but may blur
  for adjacent changes (e.g., a parser-only refactor that
  preserves behavior). *Out-of-scope for current cycle — defer
  to the first borderline release.* v1's drop is unambiguously
  minor under any reading of the rule.

---

## Promotion target

**Target:** `docs/adr/0050-v1-retirement-engine-release.md`
*(provisional)*.

This study promotes to **ADR-0050** once at least one round of
`/critique` has been accepted by the operator and any blocking
findings are addressed. The provisional slot may shift if
intervening ADRs land before promotion.

Per R8, the future ADR-0050 will be rewritten from this study,
not linked back to it. This study remains in `studies/decisions/`
as the reasoning artefact; ADR-0050 will read cold to a reviewer
who has never opened `studies/`. The load-bearing content the
ADR must carry intact is:

1. §(a) release-date selection rule + the concrete v1 date
   (2026-08-24).
2. §(b) engine version bump rule + the concrete v1 bump
   (`0.1.0` → `0.2.0`).
3. §(c) compatibility-state table transition — the `retired`
   status semantics, the `dropped` row-shape addition, the
   in-place amendment commitment, and the new "Amendment log"
   subsection convention (marked at promotion as a new
   contribution requiring review).
4. §(d) test-surface cohort split (v1-specific delete vs.
   mechanism-generic update).
5. §(e) `tools/lint/compatibility.go` post-drop behavior.
6. §Consequences C-B220.1 (single-PR shape), C-B220.2
   (publisher-config range mechanism), and C-B220.5 (90-day-
   floor precedent solidification + forward pointer to
   ADR-0049 §(b) for the v(N+1) introduction question).

The promotion ships two coupled changes: (i) ADR-0050 as the
operational decision (date, bump, cohort split, table-transition
semantics) and (ii) the in-place amendment to ADR-0035's
compatibility-state table — both land in the drop PR alongside
the code changes, so the operator-facing trail is consistent at
merge time.
