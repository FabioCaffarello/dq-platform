<!-- path: docs/adr/0034-local-testing-strategy.md -->

# ADR-0034 — Local Testing Strategy

- **Status:** accepted
- **Date:** 2026-05-25

---

## Context

The platform's CI gate runs on every PR — engine unit tests,
engine integration tests against the Compose substrate, tools
unit tests, substrate smoke tests, Kustomize overlay
rendering, and the schema-mirror byte-equality check. Every
CI lane needs a local equivalent so a developer can reproduce
CI failures on a laptop without sandbox-cloud access for
routine flows. Foundation 05 §"Local Development Posture"
anticipated this — "operational discipline must hold locally,
not only in production".

This ADR commits the **test-type taxonomy** that codifies
the platform's existing test surface, the **build-tag posture**
that separates unit from integration tests, the
**local-vs-sandbox decision table** cross-referenced to the
ADR-0010+0017+0028 capability matrix, the **fixture-tree
convention** the codebase has settled on, the **generated-SQL
inspection mechanism** at v1 plus the deferred enhancement,
and the **tooling scope inventory**.

The ADR is design-only — no test files move; no make targets
are renamed. The commitment is the taxonomy + posture + dev
guide; the implementation is what already ships. A new
operator-facing dev guide at `docs/dev/local-testing.md`
carries the how-to-run-tests prose; the `docs/dev/`
directory is introduced as the contributor-facing
counterpart to `docs/runbooks/` (operator-facing) and
`docs/security/` (security-posture).

Existing commitments this ADR builds on:

- [ADR-0001](./0001-engine-rules-compatibility.md) commits
  the byte-equality CI gate (rules schema mirror).
- [ADR-0010](./0010-substrate-posture.md) +
  [ADR-0017](./0017-substrate-posture-amendment.md) +
  [ADR-0028](./0028-kafka-substrate-row.md) commit the
  capability matrix (Yes / Partial / No rows).
- [ADR-0019](./0019-infrastructure-tooling.md) commits
  Kustomize as the overlay tool that `make validate-deploy`
  exercises.
- [ADR-0022](./0022-kind-catalog.md) commits the catalog
  mirror byte-equality (extended from ADR-0001).
- [ADR-0029](./0029-bigquery-cost-ceilings.md) registered
  `tools/dryrun` as a B2 follow-up for the compiler-layer
  bytes-scanned estimate; this ADR names the v1 v limitation
  (read-source + emulator integration tests) as the
  generated-SQL inspection mechanism until the binary lands.

The principles bearing on the decision are **P3** (ownership
is explicit — developers own local-test pass; CI owns the
gate; operators own sandbox-fidelity tests), **P4** (cost is
a first-class constraint — local testing must not require
sandbox spend for routine flows), and **P5** (evolution must
be contract-driven — the taxonomy is a documented contract).

---

## Decision

### Six-tier test-type taxonomy

The platform's tests fall into exactly one of six tiers:

| Tier | Build tag / runner | Substrate | What it exercises |
|---|---|---|---|
| **unit-no-substrate** | (default `go test`) | none | Pure-Go logic against stubs / mocks; runs anywhere |
| **integration-compose** | `//go:build integration` | `make up` (Compose) | Engine ↔ substrate roundtrips against the Yes-row emulators |
| **integration-sandbox** | `//go:build sandbox` | Real GCP / sandbox | Partial-row + No-row capabilities (OIDC, full CAS fidelity, full lazy-view fidelity, broker-set-timestamp watermark) |
| **smoke-substrate** | (bash scripts under `scripts/smoke/`) | `make up` | Per-emulator smoke: one round-trip per Yes-row capability |
| **e2e-demo** | (bash + Go) | `make up` | Full-platform flow (manifest publish → loader refresh → execution write → alert publish) |
| **config-validation** | (renderer / mirror) | none | Static validation of deployment artefacts + schema mirror pairs (no Go test code, no substrate) |

#### `unit-no-substrate`

Pure-Go tests against stubs, mocks, and test doubles.
Examples in the codebase: `engine/internal/runner/runner_test.go`,
`engine/internal/dsl/spec/spec_test.go`, `tools/lint/lint_test.go`.
Make targets: `make test-engine`, `make test-tools`. Runs
anywhere `go` is available; CI gate.

#### `integration-compose`

Engine ↔ substrate roundtrips against the Yes-row
emulators. Nine files in the codebase carry the
`//go:build integration` tag:

- `engine/internal/results/results_integration_test.go`
- `engine/internal/loader/loader_integration_test.go`
- `engine/internal/alerts/alerts_integration_test.go`
- `engine/internal/runner/runner_integration_test.go`
- `engine/internal/orphan/orphan_integration_test.go`
- `engine/internal/eval/evaluator_integration_test.go`
- `engine/internal/api/handler_integration_test.go`
- `engine/internal/api/demo_p6_integration_test.go`
- `tools/manifest/publisher_integration_test.go`

Make targets: `make test-engine-integration`,
`make test-tools-manifest-integration`. Requires `make
up` first. CI gate.

This tier exercises the engine logic against the local
emulator but does **not** validate Partial-row fidelity
(CAS, lazy view, broker-set-timestamp watermark) — those
gaps are documented in ADR-0017 and ADR-0028 and
covered by the integration-sandbox tier.

#### `integration-sandbox`

Reserved tier for Partial-row / No-row capability
validation:

- Object-store CAS with `ifGenerationMatch` (ADR-0017:
  fake-gcs-server accepts stale-generation writes;
  production GCS enforces them).
- Tabular-store lazy view with `ROW_NUMBER() OVER`
  fidelity (ADR-0010 Partial row).
- Event-stream broker-set-timestamp watermark fidelity
  (ADR-0028: commodity Kafka-compatible emulators may
  not faithfully reproduce broker-set timestamps).
- OIDC / service-identity flows (ADR-0010: No row).

No test carries the `//go:build sandbox` tag at v1. The
tier is reserved for the operational session that
provisions real GCP — the first sandbox test lands
with a new `test-engine-sandbox` make target. CI lane
runs only when sandbox credentials are available;
explicitly NOT gating on the default PR lane.

#### `smoke-substrate`

Per-emulator round-trip smoke. Four bash scripts
under `scripts/smoke/`:

- `pubsub-smoke.sh` — create topic / sub / publish /
  pull / ack.
- `object-store-smoke.sh` — create bucket / write /
  read / by-hash path.
- `tabular-store-smoke.sh` — append rows / query.
- `event-stream-smoke.sh` — produce / consume /
  consumer-group offset.

Make target: `make smoke-substrate`. Runs as part of
`make test` on every PR. CI gate.

The smoke tier exists to **distinguish substrate
failures from engine failures**. If smoke fails, the
substrate is broken; if integration-compose fails while
smoke passes, engine logic broke. The operational
distinction is load-bearing.

#### `e2e-demo`

Full-platform flow against the local substrate.
`scripts/smoke/demo-p6.sh` + the matching Go
integration test
`engine/internal/api/demo_p6_integration_test.go`
exercise: manifest publish → loader refresh →
execution write → alert publish. Make target:
`make demo-p6`. The Go-side test runs in
`make test-engine-integration` as a CI gate; the
bash-side script is operator-runnable but not in CI.

As the platform onboards additional kinds or
consumer slices, new `demo-pN` targets land
additively. Future B2 follow-ups (the first
scheduler-consumer slice from ADR-0033; the first
baselined kind from ADR-0032) ship demos under this
tier.

#### `config-validation`

Static validation that runs without any substrate:

- `make validate-deploy` renders the three Kustomize
  overlays under `deploy/overlays/{local,qa,prod}/`
  per ADR-0019; invokes `kubectl kustomize` against
  each overlay so template / patch / strategic-merge
  errors surface without a live cluster.
- The `schema-mirror` GitHub workflow enforces
  ADR-0001's byte-equality CI gate (rule schemas) and
  ADR-0022's catalog mirror pair; runs `cmp -s` on
  each pair; any drift fails CI.

Make target: `make validate-deploy` for the
Kustomize lane; the schema-mirror workflow runs in
GitHub Actions on every PR. CI gate.

This tier catches **before-the-substrate failures** —
broken overlay patches, schema-mirror drift — without
paying the `make up` cost. Faster than
integration-compose (no docker pull, no emulator
warm-up) and catches a distinct failure class.

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

### Build-tag posture

The codebase uses `//go:build integration` to separate
unit from integration tests at the package level. Files
without the tag compile + run by default; files with
the tag compile + run only when `-tags integration` is
passed to `go test`. Make targets honor the convention:

- `make test-engine` runs default-build tests only
  (unit-no-substrate).
- `make test-engine-integration` adds `-tags integration`
  (covers integration-compose + the e2e-demo Go test).
- `make test-tools-manifest-integration` does the same
  for `tools/manifest`.

This ADR reserves `//go:build sandbox` as the third
build tag for the integration-sandbox tier. The
operational session that provisions real GCP ships
the first sandbox test alongside a new
`test-engine-sandbox` make target.

### Test fixture conventions

The codebase has settled on this layout:

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
  missing-version error; mixing two error classes into
  one fixture obscures which gate failed at lint time.
- **Schema-mirrored sub-trees track schema versions.**
  When v2 ships, `testdata/v2/valid/` +
  `testdata/v2/invalid/` ship alongside the existing
  `testdata/valid/` + `testdata/invalid/` (which retain
  the v1 fixtures).
- **Cross-check fixtures live in `cross-check/` subdirs.**
  Tests that exercise multi-file invariants (rule +
  owners; rule + catalog) use `cross-check/` rather
  than `valid/` or `invalid/`.

Enforcement is by PR review (per ADR-0015 CODEOWNERS);
no lint check is committed here.

### Generated-SQL inspection

At v1, generated SQL is inspected via three paths:

1. **Read the evaluator source.** The SQL is constructed
   in `engine/internal/eval/<kind>.go` (today only
   `row_count_positive.go`). Reading the source shows
   the exact template.
2. **Run `make test-engine-integration` against the
   local BigQuery emulator.** Integration tests
   exercise the full query path; failed queries surface
   in test output. Useful for catching syntax errors
   before sandbox / prod exposure.
3. **Future: `tools/dryrun` binary.** ADR-0029
   registered the binary as a B2 follow-up — it would
   issue a BigQuery dry-run against each rule's query
   template and post the bytes-scanned estimate per PR.
   When the binary lands, it becomes the primary
   inspection mechanism.

The v1 limitation: **bytes-scanned estimates are not
available locally** (the bigquery-emulator does not
implement dry-run with production-fidelity
bytes-scanned reporting). Operators needing the
estimate run the rule against a sandbox project.

### Tooling scope inventory

| Tool | Status | Purpose |
|---|---|---|
| `dq-lint` (`tools/lint`) | Ships | Rule + owners schema + catalog validation |
| `dq-manifest` (`tools/manifest`) | Ships | Manifest publish (verify-write-CAS sequence per ADR-0005) |
| `dq-engine` (`engine/cmd/dq-engine`) | Ships | Engine binary; `make build-engine` |
| `tools/dryrun` | Deferred (B2 from ADR-0029) | Per-PR bytes-scanned estimate |
| `tools/retention` | Deferred (ADR-0031 OQ-2) | Two-tier sample-vs-row retention enforcement |
| `dq-manifest set-pointer` | Deferred (B2-10) | Rollback runbook's CLI surface |

The three shipping tools are buildable via the top-level
`make build-lint` / `make build-manifest` /
`make build-engine` targets.

### Why this does NOT reopen ADR-0010, ADR-0017, or ADR-0028

The capability matrix is the **input** to this ADR's
taxonomy. The Partial-row gaps from ADR-0017 (object-
store CAS) and ADR-0028 (broker-set-timestamp
watermark) are honored by routing the corresponding
tests to the integration-sandbox tier. The matrix is
unchanged.

### Why this does NOT commit specific sandbox infrastructure

The integration-sandbox tier is reserved at v1. The
operational session that provisions real GCP ships the
first sandbox test alongside the new
`test-engine-sandbox` make target. This ADR commits
the tier's existence as a contract; the implementation
defers to the consumer slice (matches the design-only
posture of ADR-0030 / ADR-0032 / ADR-0033).

---

## Consequences

1. **No engine code change ships from this ADR.** The
   six-tier taxonomy, the build-tag posture, the
   fixture-tree convention, and the tooling scope are
   all documentation commitments describing the
   platform's existing state and reserving the
   `//go:build sandbox` tag for a future consumer
   slice.

2. **A new dev guide ships at
   `docs/dev/local-testing.md`** carrying the operator-
   facing prose. The `docs/dev/` directory is new; the
   guide is the first entry. Future dev-facing
   documents (release engineering guide, debugging
   guide, etc.) ship under the same directory following
   the same forward-only-prose pattern.

3. **`docs/README.md` is updated to advertise
   `docs/dev/`** alongside the existing `docs/adr/`,
   `docs/runbooks/`, `docs/security/`. Same pattern as
   the introduction of `docs/security/` with ADR-0030.

4. **`/CONTRIBUTING.md` is updated** with a brief
   pointer to `docs/dev/local-testing.md` under the
   existing "run `make demo-p6`" flow — the
   test-running surface is a sub-activity of the
   existing flows, not a new top-level flow.

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
   integration tests; `//go:build sandbox` is reserved
   for the integration-sandbox tier — no test carries
   it at v1.

7. **The fixture-tree convention is committed.** New
   tests adopt the `testdata/valid/`,
   `testdata/invalid/`, `testdata/<feature>/...`
   layout. Enforcement is by PR review per ADR-0015;
   no lint check is committed here.

8. **Generated-SQL inspection at v1 is read-source +
   emulator integration tests.** Bytes-scanned
   estimates require sandbox + the deferred
   `tools/dryrun` binary; ADR-0029's B2 follow-up
   covers the gap. The dev guide names the v1
   mechanism + the deferred enhancement.

9. **Tooling inventory is the v1 baseline.** Three
   binaries ship; three are deferred. The inventory in
   the dev guide is the canonical list; future tools
   land additively.

10. **B2 follow-up: integration-sandbox tier
    implementation.** A new B2 row registers the first
    sandbox test once an operator provisions real GCP.
    The slice ships: a new `test-engine-sandbox` make
    target; at least one test exercising a Partial-row
    capability; CI configuration that runs the sandbox
    lane when credentials are available (NOT gating
    the default PR lane). The B2 row is added at
    close-step assignment of a number.

11. **The platform's P3 + P4 + P5 commitments for
    local testing are now explicit.** P3 (ownership):
    developers own local-test pass; CI owns the gate;
    operators own sandbox-fidelity tests. P4 (cost):
    local testing doesn't require sandbox spend for
    routine flows. P5 (contract-driven): the six-tier
    taxonomy is a documented contract; extensions land
    via ADR amendment.

---

## Notes

- A future amendment could ship a seventh tier for
  performance benchmarks (`go test -bench`), with
  substrate bring-up + teardown. v1 has no benchmarks;
  reserved until concrete operational signal (a
  regression that benchmarks would have caught)
  justifies the cost.
- A future amendment could ship an eighth tier for
  `go test -fuzz` against the parsers (`dsl/spec.Parse`,
  `tools/lint`'s YAML decoder, the trigger handler's
  strict JSON decoder). v1 has no fuzz tests; reserved
  until a parser-shaped vulnerability surfaces in a
  security review.
- The integration-sandbox tier's first consumer slice
  ships alongside the operational session that
  provisions real GCP. The `//go:build sandbox` tag is
  reserved at this ADR's acceptance; the first test
  carrying it lands additively without amendment.
