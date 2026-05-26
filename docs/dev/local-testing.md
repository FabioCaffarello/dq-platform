<!-- path: docs/dev/local-testing.md -->

# Local Testing Guide

> **Status:** the authoritative test-type taxonomy is
> [ADR-0034](../adr/0034-local-testing-strategy.md); this
> guide is the operator-facing how-to.

---

## What this guide is for

You're here for one of these reasons:

- **Reproducing a CI failure locally.** Read the matching
  tier section below; run the corresponding `make` target.
- **Adding a new test.** Decide which tier the test
  belongs to; place the file under the right path;
  follow the fixture-tree convention.
- **Onboarding to the platform's test surface.** Skim the
  six-tier table; come back when you need a specific
  tier.

---

## The six tiers at a glance

| Tier | When to use | Run with |
|---|---|---|
| **unit-no-substrate** | Pure-Go logic; no substrate | `make test-engine`, `make test-tools` |
| **integration-compose** | Engine ↔ Compose substrate roundtrip | `make up && make test-engine-integration` |
| **integration-sandbox** | Partial-row / No-row capability (CAS, OIDC, etc.) | `make test-sandbox` (env: `DQ_SANDBOX_PROJECT`, `DQ_SANDBOX_BUCKET`) |
| **smoke-substrate** | Per-emulator smoke | `make up && make smoke-substrate` |
| **e2e-demo** | Full platform flow | `make up && make demo-p6` |
| **config-validation** | Deploy / schema mirror static check | `make validate-deploy`; schema-mirror runs in CI |

The `make test` umbrella target runs `smoke-substrate +
test-engine + test-tools` — useful when you've already
done `make up` and want everything that runs locally
without the sandbox tier.

---

## Tier-by-tier

### unit-no-substrate

Pure-Go tests against stubs, mocks, and test doubles.
No substrate; runs anywhere `go` is available.

Run:

```
make test-engine    # engine workspace
make test-tools     # tools/lint + tools/manifest
```

Or directly:

```
cd engine && go test ./...
cd tools/lint && go test ./...
```

Existing examples:

- `engine/internal/runner/runner_test.go` — runner state
  machine against `FixedResultEvaluator` + `mockStore`.
- `engine/internal/dsl/spec/spec_test.go` — parser
  against in-memory YAML strings.
- `tools/lint/lint_test.go` — lint binary against
  testdata fixtures.

When to write a unit-no-substrate test:

- The behavior under test does not depend on a substrate
  call.
- A stub / mock / test double exists or is cheap to
  build.
- You want CI to gate on this regardless of substrate
  availability.

### integration-compose

Engine ↔ substrate roundtrips against the local Compose
emulators (Pub/Sub + object-store + tabular-store +
event-stream per ADRs 0010 + 0017 + 0028).

Bring up the substrate first:

```
make up    # docker compose up -d --wait
```

Run:

```
make test-engine-integration
make test-tools-manifest-integration
```

Or directly:

```
cd engine && go test -tags integration ./...
cd tools/manifest && go test -tags integration ./...
```

Existing examples (nine files):

- `engine/internal/results/results_integration_test.go` —
  append-only writes + canonical view against
  bigquery-emulator.
- `engine/internal/loader/loader_integration_test.go` —
  manifest pointer + by-hash reads against
  fake-gcs-server.
- `engine/internal/alerts/alerts_integration_test.go` —
  publish/subscribe against pubsub-emulator.
- Plus six more under `runner/`, `orphan/`, `eval/`,
  `api/` (×2), and `tools/manifest/`.

When to write an integration-compose test:

- The behavior under test depends on a substrate call
  whose Compose emulator faithfully reproduces it
  (Yes-row capabilities per ADR-0010 / ADR-0017 /
  ADR-0028).
- You want CI to gate on the behavior against the local
  emulator.

What integration-compose does **not** cover:

- Partial-row fidelity (CAS enforcement; lazy view's
  `ROW_NUMBER() OVER`; broker-set-timestamp watermark).
  Those land in integration-sandbox when an operator
  provisions real GCP.

### integration-sandbox

For Partial-row / No-row capabilities that the local
emulator stack cannot faithfully reproduce:

- Object-store CAS with `ifGenerationMatch` (ADR-0017).
- Tabular-store lazy view fidelity (ADR-0010 Partial
  row).
- Event-stream broker-set-timestamp watermark fidelity
  (ADR-0028 Partial row).
- OIDC / service-identity flows (ADR-0010 No row).

**First test landed at B2-18**: the CAS-race-loser branch
at `tools/manifest/publisher_sandbox_test.go`. It exercises
the ADR-0017 Partial-row gap — fake-gcs-server accepts
`ifGenerationMatch` query parameters without rejecting
stale-generation writes; real GCS returns HTTP 412. Two
test functions:

- `TestSandbox_CASWritePointer_HappyPath` — confirms real
  GCS returns a non-zero post-write generation for both
  `DoesNotExist` and `GenerationMatch` preconditions
  (smoke signal that doesn't depend on the race-loser
  path).
- `TestSandbox_CASWritePointer_StaleGenerationRejected` —
  the load-bearing case. Two CAS writes advance the
  pointer from gen=G1 to gen=G2; a third write
  parameterised with the now-stale gen=G1 must reject
  with `ErrPreconditionFailed`. fake-gcs-server fails to
  produce that 412; real GCS does. Without this test the
  race-loser contract was only ever exercised against the
  in-mem fake in `publisher_test.go`.

Run the sandbox tier:

```
export DQ_SANDBOX_PROJECT=<gcp-project>
export DQ_SANDBOX_BUCKET=<sandbox-bucket>
export GOOGLE_APPLICATION_CREDENTIALS=<service-account-json>
make test-sandbox
```

The two env vars (`DQ_SANDBOX_PROJECT`, `DQ_SANDBOX_BUCKET`)
gate the tests' execution. Without them set, every
sandbox test issues `t.Skip(…)` cleanly, so a local
`make test-sandbox` invocation without credentials is a
no-op rather than a hard failure. The
`tools/manifest/publisher_sandbox_test.go` writes under a
unique `dq-sandbox/manifests/<timestamp>/` prefix and
cleans up via `t.Cleanup`.

Make targets:

- `make test-tools-manifest-sandbox` — runs the
  `tools/manifest` sandbox tests directly.
- `make test-engine-sandbox` — runs the `engine` sandbox
  tests; no engine-side sandbox test exists at the time
  of B2-18, so the target is a no-op until a later slice
  adds engine-side sandbox tests (e.g., lazy-view
  fidelity per ADR-0010 Partial row, broker-set-
  timestamp watermark per ADR-0028).
- `make test-sandbox` — umbrella: every `//go:build
  sandbox` test across all modules.

CI lane:
[`.github/workflows/sandbox.yml`](../../.github/workflows/sandbox.yml).
Workflow trigger is **`workflow_dispatch` only** at v1
— operators trigger from the Actions UI when reviewing
a PR that touches a sandbox-relevant surface. The job
is gated on `vars.SANDBOX_ENABLED == 'true'` or the
presence of the `DQ_SANDBOX_PROJECT` secret, so the
workflow definition stays green while the platform
waits for the operational session that provisions real
GCP. A weekly `schedule` trigger is committed but
commented out; uncomment once sandbox credentials are
stable.

**The default PR lane is explicitly NOT gated on the
sandbox tier** per ADR-0034 §"`integration-sandbox`".
Sandbox spend is a P4 cost concern; gating every PR on
real GCP traffic would multiply the substrate bill
without operational benefit.

### smoke-substrate

Per-emulator round-trip smoke; one test per Yes-row
capability. Four bash scripts under `scripts/smoke/`:

- `pubsub-smoke.sh`
- `object-store-smoke.sh`
- `tabular-store-smoke.sh`
- `event-stream-smoke.sh`

Run:

```
make up
make smoke-substrate
```

The smoke tier exists to **distinguish substrate
failures from engine failures**. If smoke fails, the
substrate broke; if integration-compose fails while
smoke passes, engine logic broke. Don't merge a PR with
red smoke — it means CI's substrate setup is broken,
not the platform.

### e2e-demo

Full-platform flow: manifest publish → loader refresh →
execution write → alert publish. One demo at v1
(`make demo-p6`).

Run:

```
make up
make demo-p6
```

The Go-side test (`engine/internal/api/demo_p6_integration_test.go`)
runs as part of `make test-engine-integration` and is a
CI gate. The bash-side script
(`scripts/smoke/demo-p6.sh`) is operator-runnable but
not in CI — useful for "watch the demo with logs and
intermediate state visible".

Future demos land under this tier additively (e.g.,
the first scheduler-consumer slice from ADR-0033 adds
a demo for the cron-driven flow).

### config-validation

Static validation that runs without any substrate:

- `make validate-deploy` renders the three Kustomize
  overlays (`deploy/overlays/{local,qa,prod}/`) via
  `kubectl kustomize`. Template / patch / strategic-
  merge errors surface without a live cluster.
- The `schema-mirror` GitHub workflow enforces
  byte-equality on the rule-schema mirror pairs +
  catalog mirror pairs (`engine/internal/dsl/schema/v<N>.schema.json`
  ↔ `rules/_schema/v<N>.schema.json`;
  `engine/internal/dsl/catalog/v<N>.yaml` ↔
  `rules/_schema/catalog.v<N>.yaml`). Runs `cmp -s` on
  each pair; any drift fails CI.

Run the Kustomize lane locally:

```
make validate-deploy
```

The schema-mirror lane runs in GitHub Actions only;
locally you can mimic it with:

```
diff engine/internal/dsl/schema/v1.schema.json \
     rules/_schema/v1.schema.json
diff engine/internal/dsl/schema/v2.schema.json \
     rules/_schema/v2.schema.json
diff engine/internal/dsl/catalog/v1.yaml \
     rules/_schema/catalog.v1.yaml
```

If you've edited any of the engine-side artefacts and
forgot to update the rules-side mirror, run
`make sync-schema` (per ADR-0001 §C3) to refresh the
mirror.

---

## Fixture-tree convention

Each Go package's `testdata/` directory follows this
layout:

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

- **One error class per invalid fixture.** A fixture
  named `no-version.yaml` tests the missing-version
  error; mixing two error classes into one fixture
  obscures which gate failed.
- **Schema-mirrored sub-trees track schema versions.**
  When v2 ships, `testdata/v2/valid/` +
  `testdata/v2/invalid/` ship alongside the v1
  fixtures.
- **Cross-check fixtures live in `cross-check/`
  subdirs.** Multi-file invariants (rule + owners; rule
  + catalog) use `cross-check/` rather than `valid/`
  or `invalid/`.

Enforcement is by PR review per ADR-0015 CODEOWNERS;
no lint check enforces the tree structure.

---

## Generated-SQL inspection

At v1, generated SQL is inspected via three paths:

1. **Read the evaluator source.** SQL is constructed in
   `engine/internal/eval/<kind>.go`. Today only
   `row_count_positive.go` ships a real handler;
   future baselined / partition-aware kinds extend the
   set.
2. **Run integration-compose against the local
   BigQuery emulator.** Failed queries surface in test
   output; useful for catching syntax errors before
   sandbox / prod exposure.
3. **Future: `tools/dryrun` binary.** Per ADR-0029's
   B2 follow-up, a binary will issue a BigQuery
   dry-run against each rule's query template and post
   the bytes-scanned estimate per PR. When the binary
   lands it becomes the primary inspection mechanism.

The v1 limitation: **bytes-scanned estimates are not
available locally**. The bigquery-emulator does not
implement dry-run with production-fidelity
bytes-scanned reporting. Operators needing the estimate
run the rule against a sandbox project.

---

## Tooling inventory

| Tool | Status | Build |
|---|---|---|
| `dq-lint` | Ships | `make build-lint` → `bin/dq-lint` |
| `dq-manifest` | Ships | `make build-manifest` → `bin/dq-manifest` |
| `dq-engine` | Ships | `make build-engine` → `bin/dq-engine` |
| `tools/dryrun` | Deferred (ADR-0029 B2) | — |
| `tools/retention` | Deferred (ADR-0031 OQ-2) | — |
| `dq-manifest set-pointer` subcommand | Deferred (B2-10) | — |

Run a shipping tool from the repo root after building:

```
make build-lint
./bin/dq-lint -rules rules
```

The lint binary defaults to `rules/_schema/v1.schema.json`,
`rules/_schema/v2.schema.json`, `rules/_schema/_owners.v1.schema.json`,
`rules/_schema/_owners.v2.schema.json`, and
`rules/_schema/catalog.v1.yaml`. Override paths with
the `-schema*` / `-catalog` / `-owners*` flags if
needed.

---

## Common workflows

### "I broke the schema mirror"

You edited `engine/internal/dsl/schema/v2.schema.json`
(canonical) but didn't update
`rules/_schema/v2.schema.json` (mirror). The
`schema-mirror` CI workflow fails with `cmp -s`
mismatch.

Fix:

```
make sync-schema
git diff rules/_schema/        # verify the mirror updated
git commit -am "sync schema mirror"
```

The catalog mirror has the same fix path (the
`sync-schema` target covers both rule schemas and the
catalog per ADR-0022).

### "I broke a Kustomize overlay"

`make validate-deploy` fails locally with a
strategic-merge or patch error. The CI lane
(`make validate-deploy`) also fails.

Fix: inspect the overlay (`deploy/overlays/<env>/`)
the renderer pointed at; re-run
`make validate-deploy` after each edit until clean.

### "I broke an integration test"

Integration tests against the Compose substrate fail
locally. First confirm the substrate is up:

```
docker compose ps    # all four services should be "running"
make smoke-substrate # round-trip each capability
```

If smoke passes but integration fails, the engine
logic broke — read the test output and the engine
code under `engine/internal/<package>/`.

If smoke also fails, the substrate broke — try
`docker compose down && make up` to recycle.

### "I want to onboard a new rule and demo it locally"

See the `make demo-p6` flow in
[`/CONTRIBUTING.md`](../../CONTRIBUTING.md). The
demo script + Go integration test exercise the full
manifest-publish → loader-refresh → execution-write →
alert-publish flow against the Compose substrate.

---

## Cross-references

- [ADR-0034](../adr/0034-local-testing-strategy.md) —
  the authoritative taxonomy + posture.
- [ADR-0010](../adr/0010-substrate-posture.md) +
  [ADR-0017](../adr/0017-substrate-posture-amendment.md)
  + [ADR-0028](../adr/0028-kafka-substrate-row.md) —
  the capability matrix this guide cross-references.
- [ADR-0019](../adr/0019-infrastructure-tooling.md) —
  Kustomize as the overlay tool that `make
  validate-deploy` exercises.
- [ADR-0029](../adr/0029-bigquery-cost-ceilings.md) —
  the deferred `tools/dryrun` binary for production-
  fidelity bytes-scanned estimates.
- [`/CONTRIBUTING.md`](../../CONTRIBUTING.md) — the
  four practical contributor flows (add a rule, run
  `make demo-p6`, open a B-item, close a Wave 3
  session loop).
