<!-- path: studies/decisions/2026-05-25-b1-5-local-testing-strategy.md -->

# B1-5 — Local Testing Strategy

## Context

Foundation 05 §"Local Development Posture" anticipated that
operational discipline holds locally, not only in production —
"otherwise developers ship behaviors that work in their laptop
and break in the cluster." B1-5 was registered to commit:

1. **What can be tested offline** (no substrate access)?
2. **What needs sandbox cloud access** (BigQuery + GCS + Pub/Sub
   + Kafka with full production-fidelity primitives)?
3. **How is generated SQL inspected** during local
   development?

Several adjacent commitments already shipped:

- [ADR-0010](../../docs/adr/0010-substrate-posture.md) +
  [ADR-0017](../../docs/adr/0017-substrate-posture-amendment.md)
  + [ADR-0028](../../docs/adr/0028-kafka-substrate-row.md)
  committed the capability matrix — which substrate
  capabilities run locally (**Yes** rows), which require
  sandbox for full fidelity (**Partial** rows), and which
  cannot run locally at all (**No** rows). The matrix is the
  authoritative source for local-vs-sandbox classification.
- [ADR-0029](../../docs/adr/0029-bigquery-cost-ceilings.md)
  registered a B2 follow-up for a `tools/dryrun` binary
  (compiler-layer cost enforcement via BigQuery dry-run);
  v1's generated-SQL inspection therefore relies on the
  evaluator's source code + integration tests against the
  local BigQuery emulator.
- Wave-3 phases W3-P3 / W3-P4 / W3-P6 shipped the existing
  test surface: `make test-engine` (unit), `make
  test-engine-integration` (against the Compose substrate),
  `make smoke-substrate` (per-emulator smoke tests), and
  `make demo-p6` (full-flow E2E demo). The codebase uses
  the `//go:build integration` build tag to separate
  unit from integration tests (nine integration-test files
  across engine + tools workspaces).
- [ADR-0032](../../docs/adr/0032-baseline-strategy.md) +
  [ADR-0033](../../docs/adr/0033-scheduler-catchup-behavior.md)
  shipped design-only ADRs whose first-consumer slices add
  new test surface (baselined-kind integration tests; the
  reference Kubernetes CronJob smoke).

What B1-5 must commit:

1. **Test-type taxonomy.** Clear definitions of the test
   categories the platform supports, with their
   substrate / fidelity requirements.
2. **Local-vs-sandbox mapping.** Cross-reference each test
   type to the ADR-0010/0017/0028 capability rows so a
   developer knows which tests need `make up`, which need
   sandbox access, and which run anywhere.
3. **Build-tag posture confirmation.** The codebase already
   uses `//go:build integration` for integration tests;
   B1-5 commits this as the canonical pattern.
4. **Test fixture conventions.** Codify the de-facto
   `testdata/valid/`, `testdata/invalid/`, schema-mirrored
   sub-tree patterns the codebase has settled on.
5. **Generated-SQL inspection mechanism.** Today vs the
   deferred `tools/dryrun` future.
6. **Tooling inventory.** What ships, what's deferred, what
   binaries developers run locally.
7. **Operator-facing developer guide.** A new
   `docs/dev/local-testing.md` (in a new `docs/dev/`
   directory) carrying the how-to-run-tests-locally
   prose so contributors don't have to grep the Makefile
   for the test-type matrix.

The principles bearing on the decision are **P3** (ownership
is explicit — developers own local-test pass; CI owns the
gate; operators own sandbox-fidelity tests), **P4** (cost is
a first-class constraint — local testing should not require
sandbox spend for routine flows), and **P5** (evolution
must be contract-driven — the test-type taxonomy is a
documented contract authors and reviewers learn once).

---

## Decision Drivers

- **DD-1 — Local testing must reproduce CI failures.**
  Every CI lane has a local equivalent the developer can
  run on a laptop with `make up` + `make test`. The
  alternative (CI-only failures the developer can't
  reproduce) shifts debugging cost from minutes to hours.
- **DD-2 — The capability matrix is the authoritative
  source for local-vs-sandbox.** ADR-0010+0017+0028 already
  committed which capabilities run locally and which
  require sandbox. B1-5 cross-references the matrix to
  test types rather than re-deriving it.
- **DD-3 — Build-tag separation is the established
  pattern.** Nine integration-test files in the codebase
  use `//go:build integration`; the Makefile separates
  unit (`test-engine`, `test-tools`) from integration
  (`test-engine-integration`, `test-tools-manifest-integration`).
  B1-5 confirms this pattern rather than introducing a
  new one.
- **DD-4 — Test fixtures evolve with schema changes.** The
  ADR-0001 byte-equality CI gate (and the catalog mirror
  extension from ADR-0022) means fixture-vs-schema drift
  surfaces as CI failure. B1-5 codifies the fixture-tree
  convention so authors know where new fixtures live.
- **DD-5 — Generated-SQL inspection is partially deferred.**
  ADR-0029 registered `tools/dryrun` as a B2 follow-up; at
  v1 developers inspect generated SQL by reading
  `engine/internal/eval/<kind>.go` and running integration
  tests against the local BigQuery emulator. B1-5 names
  the v1 mechanism + points at the deferred enhancement.
- **DD-6 — Operator-facing dev guide is the artefact.**
  The ADR commits the design; the dev guide at
  `docs/dev/local-testing.md` is the operator-facing
  prose. Same pattern as ADR-0030 / ADR-0031 (security
  notes under `docs/security/`); B1-5 introduces
  `docs/dev/` as the contributor-facing equivalent.

---

## Considered Options

### Option 1 — Six-tier test-type taxonomy + capability-matrix mapping + new dev guide (recommended)

Commit a six-tier taxonomy and cross-reference it to the
capability matrix:

| Tier | Build tag / runner | Substrate | What it exercises |
|---|---|---|---|
| **unit-no-substrate** | (default `go test`) | none | Pure-Go logic against stubs / mocks; runs anywhere |
| **integration-compose** | `//go:build integration` | `make up` (Compose) | Engine ↔ substrate roundtrips against the Yes-row emulators |
| **integration-sandbox** | `//go:build sandbox` | Real GCP / sandbox | Partial-row + No-row capabilities (OIDC, full CAS fidelity, full lazy-view fidelity, broker-set-timestamp watermark) |
| **smoke-substrate** | (bash scripts under `scripts/smoke/`) | `make up` | Per-emulator smoke: one round-trip per Yes-row capability |
| **e2e-demo** | (bash + Go) | `make up` | Full-platform flow (manifest publish → loader refresh → execution write → alert publish) |
| **config-validation** | (renderer / mirror) | none | Static validation of deployment artefacts + schema mirror pairs (no Go test code, no substrate) |

Each tier maps to specific `make` targets and corresponds
to one or more ADR-0010+0017+0028 capability rows. The
existing build-tag pattern (`//go:build integration`) is
confirmed as canonical; a new `//go:build sandbox` tag is
reserved for the integration-sandbox tier (no tests carry
it at v1 since no Partial-row sandbox lane exists yet). The
sixth tier (**config-validation**) is the static-check lane
that runs without any substrate — `make validate-deploy`
renders the Kustomize overlays per ADR-0019; the
`schema-mirror` GitHub workflow enforces ADR-0001's
byte-equality CI gate (and ADR-0022's catalog mirror).
Neither uses substrate or Go test code; both are CI gates.

The new dev guide at `docs/dev/local-testing.md` carries
operator-facing prose: how to run each tier, what each
tier proves, when to add a test to which tier, and the
fixture-tree convention.

**Strengths.** Confirms the existing pattern (build tags,
separate `make` targets) rather than churning it; gives
contributors a clear map from "I want to test X" to
"run target Y"; carries the local-vs-sandbox decision
into the test layer where it actually matters. The
six-tier taxonomy is exhaustive — every test the
platform writes falls into exactly one tier.

**Trade-offs.** Codifying five tiers adds modest
documentation overhead (the dev guide is one new file).
A future test type that doesn't fit any tier requires
either a seventh tier (additive) or a stretch of an
existing tier; the taxonomy is a v1 commitment, not a
permanent ceiling.

### Option 2 — Two-tier taxonomy (unit + everything-else)

Commit only two tiers: pure-unit tests and "anything that
needs `make up`". Drop the smoke / e2e / sandbox
distinctions; everything substrate-touching is grouped.

**Strengths.** Simpler taxonomy; less documentation; fewer
make targets.

**Trade-offs.** Loses the smoke-vs-integration-vs-demo
distinction the codebase already has. Loses the sandbox
distinction that ADR-0010+0017+0028 commits at the
capability layer. Operators reading "all integration
tests" don't know whether a failure is the substrate's
fault (smoke) or the engine's fault (integration) or
the manifest-flow's fault (demo). The current granularity
is operationally meaningful; collapsing it loses signal.
Rejected — the existing structure is the right structure.

### Option 3 — Document-only (no design commitment)

Just write the dev guide; don't ship an ADR. The
test-type taxonomy is de-facto, not committed; future
contributors can introduce seventh/eighth tiers without
review.

**Strengths.** Zero design overhead; just prose.

**Trade-offs.** No durable commitment on the test-type
taxonomy means it drifts. A future contributor adds a
"performance-benchmark" tier without thinking about
substrate cost; another adds a "fuzz-test" tier without
thinking about the test-shape contract. The dev guide
becomes a wiki rather than a contract. Rejected — B1-5's
expected output is "Dev guide + tooling scope", and
"tooling scope" implies a committed contract, not a
wiki.

---

## Recommendation

**Option 1.** Six-tier test-type taxonomy committed in
ADR-0034; operator-facing dev guide at
`docs/dev/local-testing.md`; new `docs/dev/` directory
introduced.

### Test-type taxonomy

The five tiers and their requirements:

#### `unit-no-substrate`

- **Build tag:** none (default `go test` build).
- **Substrate:** none. Runs anywhere `go` is available
  (laptop, CI runner, etc.).
- **What it tests:** pure-Go logic against stubs, mocks,
  and test doubles. Examples in the codebase:
  - `engine/internal/runner/runner_test.go` (runner
    state machine against `FixedResultEvaluator` +
    `mockStore`).
  - `engine/internal/dsl/spec/spec_test.go` (parser
    against in-memory YAML strings).
  - `tools/lint/lint_test.go` (lint binary against
    testdata fixtures).
- **`make` target:** `make test-engine`, `make test-tools`.
- **CI lane:** runs on every PR; gate.

#### `integration-compose`

- **Build tag:** `//go:build integration`.
- **Substrate:** `make up` brings up the
  `docker-compose.yml` stack (pubsub + object-store +
  tabular-store + event-stream per ADRs 0010 + 0017 +
  0028).
- **What it tests:** engine ↔ substrate roundtrips
  against the **Yes-row** capabilities. Examples in the
  codebase (nine files):
  - `engine/internal/results/results_integration_test.go`
    (append-only writes + canonical view against
    bigquery-emulator).
  - `engine/internal/loader/loader_integration_test.go`
    (manifest pointer + by-hash reads against
    fake-gcs-server).
  - `engine/internal/alerts/alerts_integration_test.go`
    (publish/subscribe against pubsub-emulator).
  - `engine/internal/runner/runner_integration_test.go`,
    `engine/internal/orphan/orphan_integration_test.go`,
    `engine/internal/eval/evaluator_integration_test.go`,
    `engine/internal/api/handler_integration_test.go`,
    `engine/internal/api/demo_p6_integration_test.go`,
    `tools/manifest/publisher_integration_test.go`.
- **`make` target:** `make test-engine-integration`,
  `make test-tools-manifest-integration`. Requires
  `make up` first.
- **CI lane:** runs on every PR; gate.
- **Partial-row coverage:** integration-compose exercises
  the engine logic against the local emulator but does
  **not** validate Partial-row fidelity (CAS, lazy view,
  broker-set-timestamp watermark). Those gaps are
  documented in ADR-0017 and ADR-0028; the
  integration-sandbox tier covers them.

#### `integration-sandbox`

- **Build tag:** `//go:build sandbox` (reserved; no test
  carries this tag at v1).
- **Substrate:** real GCP / Kafka sandbox project.
- **What it tests:** **Partial-row** and **No-row**
  capabilities that the local emulator stack cannot
  faithfully reproduce:
  - Object-store CAS with `ifGenerationMatch` (ADR-0017:
    fake-gcs-server accepts stale-generation writes;
    real GCS enforces them).
  - Tabular-store lazy view with `ROW_NUMBER() OVER`
    fidelity (ADR-0010 §"Tabular store: lazy view"
    Partial row).
  - Event-stream broker-set-timestamp watermark fidelity
    (ADR-0028: commodity Kafka-compatible emulators may
    not faithfully reproduce broker-set timestamps).
  - OIDC / service-identity flows (ADR-0010: No row;
    sandbox-required).
- **`make` target:** no `make` target ships at v1 — the
  tag is reserved. The first sandbox test that lands
  adds a new `test-engine-sandbox` make target alongside
  the existing `test-engine-integration`.
- **CI lane:** runs only when sandbox credentials are
  available; explicitly NOT gating on the default PR
  lane.

#### `smoke-substrate`

- **Build tag:** N/A (bash scripts).
- **Substrate:** `make up`.
- **What it tests:** per-emulator smoke — one
  round-trip per Yes-row capability. Examples in the
  codebase (four scripts under `scripts/smoke/`):
  - `pubsub-smoke.sh` — create topic / sub /
    publish / pull / ack.
  - `object-store-smoke.sh` — create bucket / write /
    read / by-hash path.
  - `tabular-store-smoke.sh` — append rows / query.
  - `event-stream-smoke.sh` — produce / consume /
    consumer-group offset.
- **`make` target:** `make smoke-substrate`. Requires
  `make up` first.
- **CI lane:** runs as part of `make test` on every PR.
- **Purpose:** sanity-check the emulator stack is alive
  before running heavier integration tests.
  `smoke-substrate` failing means the substrate broke;
  `integration-compose` failing while smoke passes
  means engine logic broke. The distinction is
  operationally load-bearing.

#### `e2e-demo`

- **Build tag:** N/A (bash + Go integration test).
- **Substrate:** `make up`.
- **What it tests:** the full platform flow against the
  local substrate — manifest publish → loader refresh
  → execution write → alert publish. One demo at v1
  (`scripts/smoke/demo-p6.sh` + the matching Go
  integration test `engine/internal/api/demo_p6_integration_test.go`).
- **`make` target:** `make demo-p6`. Requires `make up`
  first.
- **CI lane:** the Go-side `demo_p6_integration_test.go`
  runs in `make test-engine-integration`; the bash-side
  script is operator-runnable but not in CI.
- **Future demos:** as the platform onboards additional
  entities or kinds, new `demo-pN` targets land
  additively. The pattern is established by W3-P6d;
  future B2 follow-ups add demos per consumer slice
  (e.g., the first scheduler-consumer slice from
  ADR-0033 adds a demo for the cron-driven flow).

#### `config-validation`

- **Build tag:** N/A (no Go test code).
- **Substrate:** none.
- **What it tests:** static validation of deployment
  artefacts and schema mirror pairs.
  - `make validate-deploy` renders the three Kustomize
    overlays under `deploy/overlays/{local,qa,prod}/` per
    [ADR-0019](../../docs/adr/0019-infrastructure-tooling.md);
    invokes `kubectl kustomize` against each overlay so
    template / patch / strategic-merge errors surface
    without a live cluster.
  - The `schema-mirror` GitHub workflow enforces
    [ADR-0001](../../docs/adr/0001-engine-rules-compatibility.md)'s
    byte-equality CI gate (rule schemas
    `engine/internal/dsl/schema/v<N>.schema.json` ↔
    `rules/_schema/v<N>.schema.json`) and
    [ADR-0022](../../docs/adr/0022-kind-catalog.md)'s
    catalog mirror pair
    (`engine/internal/dsl/catalog/v<N>.yaml` ↔
    `rules/_schema/catalog.v<N>.yaml`). The workflow
    runs `cmp -s` on each pair; any drift fails CI.
- **`make` target:** `make validate-deploy` for the
  Kustomize lane; the schema-mirror workflow runs in
  GitHub Actions on every PR.
- **CI lane:** runs on every PR; gate.
- **Purpose:** these are **the lanes that catch
  before-the-substrate failures** — broken overlay
  patches, schema-mirror drift — without paying the
  `make up` cost. They run faster than
  integration-compose (no docker pull, no emulator
  warm-up) and catch a distinct failure class.

### Local-vs-sandbox decision table

| Test scope | Local | Sandbox |
|---|---|---|
| Engine internal logic (parsing, status mapping, etc.) | Yes (unit-no-substrate) | — |
| Tabular store: append-only writes | Yes (integration-compose; ADR-0010 Yes row) | — |
| Tabular store: lazy view (`ROW_NUMBER() OVER`) | Yes for engine logic (integration-compose) | Recommended for full fidelity (ADR-0010 Partial row) |
| Object store: by-hash sha256 immutability | Yes (integration-compose; ADR-0010 Yes row) | — |
| Object store: generation-conditional CAS (`ifGenerationMatch`) | Yes for engine logic (integration-compose) | Required for production-shape enforcement (ADR-0017 Partial row; local emulator accepts stale generations) |
| Pub/Sub publish + subscribe | Yes (integration-compose; ADR-0010 Yes row) | — |
| Event stream: publish + subscribe + consumer-group offset | Yes (integration-compose; ADR-0028 Yes rows) | — |
| Event stream: broker-set-timestamp watermark fidelity | Yes for engine math (integration-compose) | Recommended for full fidelity (ADR-0028 Partial row) |
| OIDC / service-identity flows | No — cannot emulate (ADR-0010 No row) | Required (integration-sandbox) |
| Deployment-artefact rendering (Kustomize overlays) | Yes (config-validation; `make validate-deploy`) | — |
| Schema-mirror byte-equality | Yes (config-validation; `schema-mirror` workflow) | — |
| Full E2E demo flow | Yes (`make demo-p6`) | — |
| Generated-SQL bytes-scanned estimate | Read evaluator source + run integration tests against local BigQuery emulator | Required for production-fidelity dry-run estimate (integration-sandbox + future `tools/dryrun` per ADR-0029 B2) |

The table is the authoritative cross-reference for
"where do I test X?" The dev guide carries the same
table in operator-readable prose.

### Test fixture conventions

The codebase has settled on this layout; B1-5 codifies
it:

```
<workspace>/<package>/testdata/
├── valid/              ← happy-path fixtures (one or more)
├── invalid/            ← failure-path fixtures, one per error class
├── <feature>/          ← feature-specific sub-trees (e.g., v2/)
│   ├── valid/
│   ├── invalid/
│   └── cross-check/    ← cross-file fixture sets
└── owners/             ← when the feature involves _owners.yaml
    ├── valid/
    ├── invalid/
    └── cross-check/
```

Rules of thumb:

- **Each invalid fixture exercises one error class.** A
  fixture named `no-version.yaml` tests the
  missing-version error; a fixture named
  `pipe-entity.yaml` tests the pipe-character rejection.
  Mixing two error classes into one fixture obscures
  which gate failed at lint time.
- **Schema-mirrored sub-trees track schema versions.**
  When v2 ships, `testdata/v2/valid/` + `testdata/v2/invalid/`
  ship alongside the existing `testdata/valid/` +
  `testdata/invalid/` (which retain the v1 fixtures).
- **Cross-check fixtures live in `cross-check/`
  subdirs.** Tests that exercise multi-file invariants
  (rule + owners; rule + catalog) use `cross-check/`
  rather than `valid/` or `invalid/` so the
  multi-file intent is visible at the directory level.

### Generated-SQL inspection

At v1, generated SQL is inspected via three paths:

1. **Read the evaluator source.** The SQL is constructed
   in `engine/internal/eval/<kind>.go` (today only
   `row_count_positive.go`). Reading the source shows
   the exact template — which is small at v1 and grows
   as new kinds ship.
2. **Run `make test-engine-integration` against the
   local BigQuery emulator.** Integration tests exercise
   the full query path; failed queries surface in test
   output. Useful for catching syntax errors before
   sandbox / prod exposure.
3. **Future: `tools/dryrun` binary.** ADR-0029
   registered the binary as a B2 follow-up — it would
   issue a BigQuery dry-run against each rule's query
   template and post the bytes-scanned estimate per PR.
   When the binary lands, it becomes the primary
   inspection mechanism; the read-source + emulator
   paths remain available for ad-hoc inspection.

The v1 limitation: **bytes-scanned estimates are not
available locally** (the bigquery-emulator does not
implement dry-run with bytes-scanned reporting at
production fidelity). Operators needing the estimate
run the rule against a sandbox project before
production deployment.

### Tooling scope inventory

The platform's local-development tooling at v1:

| Tool | Status | Purpose |
|---|---|---|
| `dq-lint` (`tools/lint`) | Ships | Rule + owners schema + catalog validation |
| `dq-manifest` (`tools/manifest`) | Ships | Manifest publish (verify-write-CAS sequence per ADR-0005) |
| `dq-engine` (`engine/cmd/dq-engine`) | Ships | Engine binary; `make build-engine` |
| `tools/dryrun` | Deferred (B2 from ADR-0029) | Per-PR bytes-scanned estimate |
| `tools/retention` | Deferred (ADR-0031 OQ-2) | Two-tier sample-vs-row retention enforcement |
| `dq-manifest set-pointer` | Deferred (B2-10) | Rollback runbook's CLI surface |

The three shipping tools are buildable via the
top-level `make build-lint` / `make build-manifest` /
`make build-engine` targets and runnable from the repo
root (or from any `cd`-d directory once built).

### Why this does NOT reopen ADR-0010, ADR-0017, or ADR-0028

The capability matrix from those three ADRs is the
**input** to B1-5's test-type taxonomy. B1-5
cross-references the matrix at the test layer; it does
not amend the matrix itself. The Partial-row gaps
documented by ADR-0017 (object-store CAS) and ADR-0028
(broker-set-timestamp watermark) are honored by
routing the corresponding tests to the
`integration-sandbox` tier.

### Why this does NOT commit specific sandbox infrastructure

The integration-sandbox tier is **reserved** at v1 — no
tests carry the `//go:build sandbox` tag because no
operator has provisioned a real GCP sandbox yet (qa /
prod are `PLACEHOLDER`-named). When the operational
session that provisions real GCP projects lands, the
first integration-sandbox test ships alongside, and the
`test-engine-sandbox` make target lands. B1-5 commits
the tier's existence as a contract; the implementation
defers to the consumer slice (matches the design-only
posture of ADR-0030 / ADR-0032 / ADR-0033).

---

## Consequences

1. **No engine code change ships from this ADR.** The
   six-tier test-type taxonomy, the build-tag posture,
   the fixture-tree convention, and the tooling scope
   are all **documentation commitments** describing the
   platform's existing state and reserving the
   `//go:build sandbox` tag for a future consumer slice.
   No new tests are added; no existing tests are moved.

2. **A new dev guide ships at
   `docs/dev/local-testing.md`** carrying the operator-
   facing prose: how to run each tier, what each tier
   proves, when to add a test to which tier, the
   fixture-tree convention, and the local-vs-sandbox
   decision table. The `docs/dev/` directory is new;
   the guide is the first entry. Future dev-facing
   documents (release engineering guide, debugging
   guide, etc.) ship under the same directory following
   the same forward-only-prose pattern.

3. **`docs/README.md` is updated to advertise
   `docs/dev/`** alongside the existing `docs/adr/`,
   `docs/runbooks/`, `docs/security/`. Same pattern as
   the introduction of `docs/security/` with ADR-0030.

4. **`/CONTRIBUTING.md` is updated** with a brief
   pointer to `docs/dev/local-testing.md` under the
   existing "run `make demo-p6`" flow — the test-running
   surface is a sub-activity of the existing flows, not
   a new top-level flow. The four W3-P8c flows (add a
   rule, run `make demo-p6`, open a B-item, close a
   Wave 3 session loop) are unchanged in structure;
   only the cross-reference is added.

5. **The six-tier taxonomy is the contract.** Future
   tests fall into exactly one tier:
   - Pure-Go logic against stubs → unit-no-substrate.
   - Engine ↔ substrate roundtrip → integration-compose.
   - Partial-row / No-row capability validation →
     integration-sandbox.
   - Per-emulator smoke → smoke-substrate.
   - Full-platform flow → e2e-demo.
   - Static deploy / schema validation →
     config-validation.
   A future test that doesn't fit any tier either
   extends a tier additively (and the dev guide is
   amended) or argues for a seventh tier in a new ADR.

6. **The build-tag posture is committed.**
   `//go:build integration` separates unit from
   integration tests at the package level; the Makefile
   targets respect the convention. `//go:build sandbox`
   is reserved for the integration-sandbox tier — no
   test carries it at v1.

7. **The fixture-tree convention is committed.** New
   tests adopt the `testdata/valid/`,
   `testdata/invalid/`, `testdata/<feature>/...` layout.
   The convention is enforced via PR review (per
   ADR-0015 CODEOWNERS); no lint check is committed
   here.

8. **Generated-SQL inspection at v1 is read-source +
   emulator integration tests.** Bytes-scanned
   estimates require sandbox + the deferred
   `tools/dryrun` binary; ADR-0029's B2 follow-up
   covers the gap. The dev guide names the v1
   mechanism + the deferred enhancement so contributors
   know what to expect.

9. **Tooling inventory is the v1 baseline.** Three
   binaries ship (`dq-lint`, `dq-manifest`,
   `dq-engine`); three are deferred (`tools/dryrun`,
   `tools/retention`, `dq-manifest set-pointer`). The
   inventory in the dev guide is the canonical list;
   future tools land additively.

10. **B2 follow-up: integration-sandbox tier
    implementation.** A new B2 row registers the first
    sandbox test that lands once an operator
    provisions real GCP. The slice ships:
    - A new `test-engine-sandbox` make target.
    - At least one test exercising a Partial-row
      capability (CAS fidelity or lazy-view fidelity
      or broker-set-timestamp watermark).
    - CI configuration that runs the sandbox lane when
      credentials are available; explicitly NOT gating
      on the default PR lane.
    The B2 row is added at close-step assignment of a
    number.

11. **The platform's P3 + P4 + P5 commitments for
    local testing are now explicit.** P3 (ownership):
    developers own local-test pass; CI owns the gate;
    operators own sandbox-fidelity tests. P4 (cost):
    local testing doesn't require sandbox spend for
    routine flows; sandbox is reserved for Partial-row
    / No-row capabilities. P5 (contract-driven): the
    six-tier taxonomy is a documented contract;
    extensions land via ADR amendment.

---

## Open Questions

None blocking.

Two deferred items surfaced during the design phase and
are explicitly **out-of-scope for current cycle**:

- **OQ-1: Performance / benchmark tier.** A future
  amendment could ship a seventh tier for performance
  benchmarks (`go test -bench`, with substrate
  bring-up + teardown). v1 has no benchmarks; reserved
  until concrete operational signal (a regression that
  benchmarks would have caught) justifies the tier's
  build / test / CI cost.

- **OQ-2: Fuzz test tier.** A future amendment could
  ship an eighth tier for `go test -fuzz` against the
  parsers (`dsl/spec.Parse`, `tools/lint`'s YAML
  decoder, the trigger handler's strict JSON decoder).
  v1 has no fuzz tests; reserved until a parser-shaped
  vulnerability surfaces in a security review.

---

## Promotion target

`docs/adr/0034-local-testing-strategy.md` — ships the
six-tier test-type taxonomy, the build-tag posture
confirmation, the local-vs-sandbox decision table, the
fixture-tree convention, the v1 generated-SQL
inspection mechanism + deferred enhancement pointer,
the tooling scope inventory, and the B2 follow-up for
the integration-sandbox tier implementation. The
operator-facing dev guide at `docs/dev/local-testing.md`
ships alongside (introduces the new `docs/dev/`
directory; second cross-functional artefact pattern
after `docs/security/` from ADR-0030).
