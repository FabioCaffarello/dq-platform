// path: engine/internal/api/doc.go

// Package api implements the HTTP trigger handler scaffolded by
// W3-P4e. The package is the engine's external data-plane surface:
// it accepts triggers at POST /v1/trigger, dispatches them to the
// runner, and exposes the liveness (/healthz) and readiness
// (/readyz) probes that orchestration substrates consume.
//
// The full behavior contract is committed by ADR-0014:
//
//   - §1 (eager-at-load) — the listener binds only after the first
//     successful manifest load completes. Handler reads the active
//     manifest reference via a closure accessor at trigger
//     acceptance per ADR-0007 §3 (in-flight execution isolation).
//   - §2 (strict decoder) — unknown fields rejected; per-field
//     invariants (UTF-8, ASCII-pipe absence, Z-only RFC 3339 UTC,
//     closed `trigger_source` enum, length ceilings) enforced
//     before the execution_id formula is computed. `operator-rerun`
//     is rejected with 400 at the data-plane path (operator-rerun
//     is the Admin API path's exclusive source per ADR-0002 §4).
//   - §3 (separate API DTO) — the response is a v1 DTO distinct
//     from the dq_executions persistence row; the route is mounted
//     at /v1/trigger and v1-path-versioning is the initial
//     evolution channel.
//   - §4 (health endpoints) — /healthz returns 200 OK while the
//     process is up; /readyz returns 200 OK while the listener is
//     reachable (the listener exists only post-first-load per §1,
//     so /readyz is trivially always 200 OK in-band).
package api
