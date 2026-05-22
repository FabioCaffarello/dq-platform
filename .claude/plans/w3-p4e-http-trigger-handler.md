<!-- path: .claude/plans/w3-p4e-http-trigger-handler.md -->

# W3-P4e — HTTP Trigger Handler (plan)

> **Renumbered from W3-P6b → W3-P4e on 2026-05-22.** P4e closes
> **Phase 4 (engine runtime)** — ADR-0013 §"Phase 4 — Engine
> runtime scaffold" enumerates loader, runner, result write,
> failure scope, and orphan detection; the runner is unexercisable
> without a trigger surface, so the HTTP trigger handler belongs
> inside Phase 4 as its closing sub-phase. **Phase 6 remains
> "first onboarded entity end-to-end"** exactly as ADR-0013
> specifies — one real `_owners.yaml`, one real published
> manifest, exercising Phases 3–5 against real content. The
> handler is part of the runtime that Phase 6 will exercise, not
> the entity-onboarding work itself.

---

## Upstream contract

`docs/adr/0014-trigger-handler-contract.md` (`accepted`,
2026-05-22). The four sections of ADR-0014's Decision are the
load-bearing contract:

- §1 **Hydration timing** — eager-at-load; listener binds only
  after first successful manifest load.
- §2 **Strict decoder** — unknown fields rejected; per-field
  invariants (UTF-8, ASCII-pipe absence, RFC 3339 UTC with
  `Z`-only, closed `trigger_source` enum, length ceilings)
  enforced before `execution_id` is computed; `operator-rerun`
  rejected with `400` at the data-plane path.
- §3 **Separate API DTO** — `{execution_id, attempt_id, status,
  accepted_at, self}` distinct from `dq_executions` row; v1
  served at `/v1/trigger`; v1-path-versioning is the initial
  evolution channel.
- §4 **Health endpoints** — `/healthz` (liveness) + `/readyz`
  (readiness); both return 200 once the listener is bound (which
  is post-first-load per §1).

---

## Unit scope

### In scope

- New package `engine/internal/api/` implementing the HTTP
  surface (handler, strict decoder, DTOs, mux, server wrapper)
  per ADR-0014 §§1–4.
- `cmd/dq-engine/main.go` wiring: bind the HTTP listener
  **after** the initial manifest load completes (ADR-0014 §1);
  add the HTTP server to the existing graceful-shutdown
  WaitGroup; new `DQ_HTTP_ADDR` env var (default `:8080`).
- `runner.TriggerRequest` enhancements (smallest change to
  honor ADR-0014 §3 acceptance-time IDs):
  - new optional field `AttemptID *string` — override the
    runner's constructor-time `AttemptIDFunc` per call.
  - new field `RulesetVersion string` — override the runner's
    constructor-time pin per call, so a manifest refresh between
    startup and trigger acceptance doesn't desync the
    `execution_id` formula's first input. Empty string falls
    back to the runner's pin (Phase 4c backwards-compat).
- The `_ = r` line in `main.go:174` is replaced with real
  wiring: the runner is passed to the new HTTP handler.
- Update package comments in `runner.go` that name "Phase 6
  wires triggers" to instead name "W3-P4e wires triggers"
  (one-line lexical fix; same drift class as the engine README
  fixes in PR #8).

### Out of scope (deferred to future sessions per R4)

| Topic | Where deferred | Reason |
|---|---|---|
| Manifest-driven check resolution | P6 (per runner.go package doc) | `ManifestRule` carries `{Entity, YamlPath, YamlHash}` only — YAML body not loaded. P4e passes an empty `Checks` list to the runner; the running + terminal rows still land. |
| Error-code taxonomy | ADR-0014 OQ-MD-2.1 follow-up | Taxonomy is a public API contract; lands as an ADR amendment, not in implementation. |
| Authentication / authorisation | ADR-0014 OQ-CC.1 / CC6 | Substrate-coupled (W2-3 capability matrix); the no-auth interim posture is the committed P4e default. |
| Rate limiting / concurrency budgets | ADR-0014 OQ-CC.2 | Cost-discipline ADR; out of trigger-handler scope. |
| gRPC variant | ADR-0014 OQ-CC.3 | Additive surface; separate ADR if and when an external scheduler demands it. |
| TLS / observability headers / tracing | ADR-0014 Context | Operational, not contract. |
| Max payload size at the listener | ADR-0014 OQ-MD-2.2 | Substrate-coupled (ingress configuration); per-field length ceilings cover the formula safety. |
| `self` URL fragment format | ADR-0014 OQ-MD-3.2 | Read API itself is out of scope; the handler emits the relative path `/v1/executions/{execution_id}`. |
| v2 path-bump trigger criteria | ADR-0014 OQ-MD-3.1 | Future amendment when accumulated breaking changes warrant it. |
| `/manifestz` optional endpoint | ADR-0014 OQ-MD-4.1 | Useful for operators but not contract-bearing. |
| OpenTelemetry metric + span signals | Phase 4c gap (ADR-0007 CC14 carry-forward) | Honest-gap pattern; log signal already delivered, metric + span land additively. |

---

## Files

### New files

| Path | Purpose |
|---|---|
| `engine/internal/api/doc.go` | Package doc citing ADR-0014 §§1–4 and the package boundary (data-plane surface for the runner). |
| `engine/internal/api/dto.go` | JSON DTOs: `TriggerRequest` (wire layer), `TriggerResponse` (ADR-0014 §3 v1 fields), `ErrorResponse` envelope `{code, message, field}`. |
| `engine/internal/api/decoder.go` | Strict decoder (`json.Decoder.DisallowUnknownFields`) + per-field validators (`Z`-only RFC 3339, no-pipe, length ceiling, closed enum) per ADR-0014 §2. |
| `engine/internal/api/handler.go` | `Handler` with `POST /v1/trigger`, `GET /healthz`, `GET /readyz`. Constructor takes a runner reference + a manifest-accessor closure + clock + logger + engine version. |
| `engine/internal/api/server.go` | `Server` wrapper around `*http.Server` with the mux assembly and a `Shutdown(ctx)` method. |
| `engine/internal/api/handler_test.go` | Unit tests for the handler and decoder against a fake runner + a synthetic loader manifest. |
| `engine/internal/api/handler_integration_test.go` | Integration test (`//go:build integration`) — real BigQuery store + GCS-backed loader; POSTs a trigger via `httptest`-style server; reads back from `dq_executions`. |

### Modified files

| Path | Change |
|---|---|
| `engine/internal/runner/runner.go` | Add `TriggerRequest.AttemptID *string` and `TriggerRequest.RulesetVersion string`; thread per-call effective values through `Run` and the three private row builders; update the package doc comment to remove the "Phase 4c held but not exercised" historical note in favor of W3-P4e wiring. |
| `engine/cmd/dq-engine/main.go` | Construct and start the HTTP server after `current.set(initial)`; add it to the `sync.WaitGroup`; call `server.Shutdown` during graceful drain; read `DQ_HTTP_ADDR` env var (default `:8080`); fix two lexical-drift comments (`Phase 6 → W3-P4e`). |
| `studies/foundation/06-decision-log.md` | Update the W3-P4e row's status cell from `resolved-adr` to `closed (lands via PR; …)` per the wave-3-session-loop step 10 convention (the row keeps both prior links). |

---

## ADR-0014 §-to-file map

| ADR-0014 § | Commitment | Implementing file(s) |
|---|---|---|
| §1 (eager-at-load) | Listener binds only after first manifest load | `cmd/dq-engine/main.go` (post-`current.set(initial)`) |
| §1 (in-flight isolation) | Handler captures manifest reference at acceptance | `internal/api/handler.go` (closure accessor) |
| §2 (`DisallowUnknownFields`) | Unknown fields → 400 | `internal/api/decoder.go` |
| §2 (per-field invariants) | UTF-8, no-pipe, `Z`-only RFC 3339, closed enum, length ceilings | `internal/api/decoder.go` |
| §2 (`operator-rerun` reject) | 400 at the data-plane path | `internal/api/decoder.go` (enum check) |
| §2 (`400` envelope) | Structured `{code, message, field}` | `internal/api/dto.go` |
| §3 (separate API DTO) | Distinct from `results.ExecutionRow` | `internal/api/dto.go` |
| §3 (v1-path-versioning) | Route mounted at `/v1/trigger` | `internal/api/server.go` |
| §3 (`accepted_at` distinct) | Handler-side timestamp, distinct from `started_at` | `internal/api/handler.go` |
| §4 (`/healthz`) | 200 OK while process up | `internal/api/handler.go` |
| §4 (`/readyz`) | 200 OK once first load completes (vacuous post-§1) | `internal/api/handler.go` |

---

## AC-W3 mapping

| # | Status | Note |
|---|---|---|
| AC-W3-1 | pass | This plan file + project plan file carry path headers. |
| AC-W3-2 | pass | Go code identifiers and comments in English. |
| AC-W3-3 | pass | Load-bearing citations: ADR-0002 §1 (formula), ADR-0003 §3–§4 (running row + attempt_id), ADR-0007 §1 (startup-mode), ADR-0014 §§1–4 (full contract). Placed at the formula/decode/IDs sites. |
| AC-W3-4 | pass | `git diff --stat` matches the file list above. |
| AC-W3-5 | gated | `/critique` runs after first commit; max two rounds per the loop. |
| AC-W3-6 | pass | No TODO / FIXME / `_TBD` planned. Deferrals are tabulated in §"Out of scope" above. |
| AC-W3-7 | pass | `make lint`, `make test`, `make test-integration` all green. |
| AC-W3-8 | N/A | No W2 §3.3 capability row directly maps to this surface (the HTTP listener is part of the engine runtime, exercised by Phase 6). |
| AC-W3-9 | pass | Decision-log W3-P4e row gets the `closed (lands via PR; ...)` suffix appended in the same PR. |
| AC-W3-10 | pass | R5 hygiene: no prior-art names. HTTP / RFC 3339 / sha256 / UUID are commodity environment. |
