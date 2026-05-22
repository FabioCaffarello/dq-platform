<!-- path: engine/README.md -->

# `engine/` — DQ Platform Runtime

The engine workspace owns the Go runtime that evaluates
data-quality rules against the configured substrate.

This workspace is a single Go module
(`dq-platform/engine`), part of the top-level Go workspace
declared in [`go.work`](../go.work) per
[B1-10's resolution](../studies/decisions/2026-05-21-b1-10-workspace-tooling.md).

## Scope (Wave 3)

Wave 3 lands the engine runtime across Phases 3–4:

- **Phase 3** — schema source of truth under
  `internal/dsl/schema/`. The schema is the canonical
  source mirrored byte-for-byte into
  [`rules/_schema/`](../rules/) per
  [ADR-0001](../docs/adr/0001-engine-rules-compatibility.md).
- **Phase 4** — loader (per
  [ADR-0007](../docs/adr/0007-loader-scheduler-retry-failure-semantics.md)),
  runner with `execution_id` computation (per
  [ADR-0002](../docs/adr/0002-run-identity-and-idempotency.md)),
  result write to `dq_executions` and `dq_check_results`
  (per [ADR-0003](../docs/adr/0003-result-write-model.md)),
  failure-scope mapping (per
  [ADR-0004](../docs/adr/0004-failure-scope.md)),
  orphan-run detection (per ADR-0007).

## Current state

- **Phase 2 (root infrastructure)** — module declaration only.
- **Phase 3 (schema layer)** — canonical schema source at
  `internal/dsl/schema/v1.schema.json`, byte-equal to the
  rules mirror.
- **W3-P4a (loader)** — `internal/loader/` package:
  - `Loader.Load(ctx)` — startup-mode manifest load per
    ADR-0007 CC1: reads `manifests/latest.json`, verifies
    the sha256 content hash of the referenced body, parses
    the manifest, runs the ADR-0001 contract checks
    (manifest_version, engine_compatibility,
    schema_versions_present), returns the parsed Manifest.
  - `Loader.Refresh(ctx, currentHash)` — refresh-mode reload
    per ADR-0007 CC9 with hash short-circuit (pointer-only
    read when the hash matches the caller's current).
  - `Store` interface with a GCS-backed implementation that
    works against production GCS and against the local
    fake-gcs-server emulator via `STORAGE_EMULATOR_HOST`.
  - Typed errors (`ErrObjectNotFound`, `ErrHashMismatch`,
    `ErrContractMismatch`) for the engine binary's
    structured-failure observability per ADR-0007 CC14.
  - Unit tests under `internal/loader/loader_test.go`;
    integration tests under `loader_integration_test.go`
    (build tag `integration`), runnable via
    `make test-engine-integration` after `make up`.
- **W3-P4b (result-write layer)** — `internal/results/`
  package:
  - `ExecutionRow` / `CheckResultRow` types matching
    ADR-0003 CC3 / CC7 columns; closed enums for
    `ExecutionStatus`, `CheckResult`, `TriggerSource`.
  - `Store` interface (Writer + Reader + EnsureSchema).
  - `BigQueryStore` impl backed by
    `cloud.google.com/go/bigquery`. Append-only writes via
    streaming insert per ADR-0003 CC1; no UPDATE or DELETE
    from engine code paths.
  - `EnsureSchema` is idempotent — creates dataset +
    tables + the `dq_executions_current` view (best-
    effort against the emulator per ADR-0010 lazy-view
    Partial row).
  - `QueryCurrentExecution` uses an inline `ROW_NUMBER()
    OVER (PARTITION BY execution_id ORDER BY recorded_at
    DESC)` so engine internals are portable across the
    emulator's view fidelity gap.
  - Unit tests under `internal/results/results_test.go`;
    integration tests under
    `results_integration_test.go` (build tag
    `integration`), runnable via
    `make test-engine-integration` after `make up`.

- **W3-P4d (orphan-run detection)** — `internal/orphan/`
  package:
  - `Detector.RunOnce(ctx)` — one scan pass per ADR-0007
    CC11: lists `running` rows whose `started_at` is older
    than the configured threshold and writes a follow-up
    `aborted` row carrying the **detector's own**
    `engine_version` (load-bearing per CC11).
  - Tolerates per-row write failures: continues finalizing
    siblings, returns the count + per-row errors.
  - Emits a structured `slog.Info` per finalized row for
    ADR-0007 CC14 observability.
  - Consumes a narrow `Scanner` interface that the
    `results.Store` already satisfies (extended in this
    sub-phase with `ListRunningOlderThan`).
  - Unit tests under `internal/orphan/orphan_test.go`;
    integration tests under
    `orphan_integration_test.go` (build tag `integration`).

- **W3-P4c (runner + failure-scope mapping)** —
  `internal/runner/` package plus the `cmd/dq-engine/`
  binary entry point:
  - `Runner.Run(ctx, trigger)` — end-to-end execution
    attempt: validates inputs (ADR-0002 CC2 pipe safety;
    CC5 trigger-source / supersedes coherence); computes
    `execution_id` (ADR-0002 CC1); generates `attempt_id`
    (ADR-0003 CC4); runs pre-check (ADR-0007 CC8); writes
    running row; evaluates each check with
    always-continue (ADR-0004 CC4); writes per-check
    rows; computes terminal status via `MapStatus` per
    ADR-0004 CC2; writes terminal row.
  - `EntityPrecheck` + `CheckEvaluator` interfaces with
    no-op defaults (real implementations land in Phase 6).
  - `cmd/dq-engine/main.go` — minimal binary: loads
    config from env, creates GCS + BigQuery clients, runs
    `EnsureSchema`, performs initial manifest load
    (process-exit on failure per ADR-0007 CC1), starts
    two periodic loops (loader refresh + orphan
    detection), waits for SIGTERM/SIGINT.
  - HTTP trigger handler is deferred to W3-P4e
    (resolved-study 2026-05-22; provisional ADR-0014); the
    binary holds a Runner but does not exercise it at
    runtime. P4e implementation wires triggers. gRPC is
    deferred (OQ-CC.3 in the contract study).
  - Unit tests under `internal/runner/runner_test.go`;
    integration tests under
    `runner_integration_test.go` (build tag
    `integration`).

With Phase 4c, **Phase 4 closes**. Phase 5 follows:

- **W3-P5 (alerting)** — `internal/alerts/` package:
  - `Event` payload struct + JSON serialization per
    ADR-0006 §4 (`event.go`).
  - `MapCategory(source, *result, *status)` implements
    the non-negotiable category boundary per ADR-0006 CC7
    (`category.go`).
  - `AttemptDeduper` — per-attempt engine-side dedup per
    ADR-0006 CC5; bounded in-memory state discarded when
    the attempt finalizes (`dedup.go`).
  - `Publisher` interface + `NoopPublisher` + `PubSubPublisher`
    (Pub/Sub v2 client; honors `PUBSUB_EMULATOR_HOST` for the
    local Compose stack).
  - Integrated into the runner (per-check + execution-level
    emissions) and orphan detector (per-finalization
    operational emission). Publish failures are
    warning-logged, not propagated — alerting is best-effort
    out-of-band signal.
  - Engine binary creates the publisher when
    `DQ_PUBSUB_TOPIC` is set; defaults to `NoopPublisher`
    otherwise so local-dev binaries don't have to depend on
    the emulator.
  - Unit tests + integration test under
    `alerts_integration_test.go` (build tag `integration`)
    round-trip an Event through the local emulator.

Phase 5 also adds the `_owners.v1.schema.json` to
`rules/_schema/` and extends `dq-lint` with the
ADR-0006 CC9 enforcement: every rule's entity must be
declared in `_owners.yaml`, or the linter rejects.

Configuration: see `engine/internal/env/` — typed
multi-environment package per foundation 04 §PAT-4 and
B1-4 MD-4. The binary's runtime selector is `DQ_ENV`
(closed-enum `local` / `qa` / `prod`); two emulator-host
overrides (`STORAGE_EMULATOR_HOST`,
`BIGQUERY_EMULATOR_HOST`) remain env-var-driven as
substrate concerns honored directly by the GCP SDKs.

Future work:

- **W3-P4e** — HTTP trigger handler (POST `/v1/trigger`,
  dispatches to runner per ADR-0002; contract per the
  W3-P4e trigger-handler-contract study, provisional ADR-0014
  slot).
- **W3-P6** — first onboarded entity end-to-end.
- **W3-P7** — `deploy/` (Kubernetes manifests, env overlays).
- **W3-P8** — `docs/` content beyond ADRs.
