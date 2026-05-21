// path: engine/internal/results/results.go

// Package results is the result-write layer per ADR-0003. It owns
// the dq_executions and dq_check_results tables and the lazy
// dq_executions_current view. The Store interface is consumed by
// the runner (W3-P4c) and the orphan detector (W3-P4d). The
// BigQuery-backed implementation works against production BigQuery
// and against the bigquery-emulator (ADR-0010 substrate posture).
//
// Per ADR-0003 CC1 the tables are append-only: no UPDATE, no
// DELETE from engine code paths. Writers issue inserts only. The
// canonical single-row-per-execution projection is
// dq_executions_current (ADR-0003 CC2), implemented as a lazy view
// with ROW_NUMBER() OVER (PARTITION BY execution_id ORDER BY
// recorded_at DESC). The Go API's QueryCurrentExecution runs an
// inline equivalent so engine internals are portable across the
// emulator's lazy-view fidelity gap (ADR-0010 lazy-view Partial
// row).
package results

import (
	"context"
	"errors"
	"time"
)

// ErrExecutionNotFound is returned by Reader.QueryCurrentExecution
// when no rows exist in dq_executions for the requested
// execution_id.
var ErrExecutionNotFound = errors.New("execution not found")

// Writer is the append-only write surface for the result-write
// layer per ADR-0003 CC1. Implementations must never issue UPDATE
// or DELETE; only inserts.
type Writer interface {
	// WriteExecutionRow appends one row to dq_executions. The row
	// is one state-transition (running, success, failed, error,
	// aborted) for a single (execution_id, attempt_id) pair.
	WriteExecutionRow(ctx context.Context, row ExecutionRow) error

	// WriteCheckResultRow appends one row to dq_check_results. One
	// row per (execution_id, attempt_id, check_id) per ADR-0003
	// CC7.
	WriteCheckResultRow(ctx context.Context, row CheckResultRow) error
}

// Reader is the engine-internal read surface. Multi-row forensic
// queries are out of scope; dashboards / reporting tools query the
// view directly via SQL.
type Reader interface {
	// QueryCurrentExecution returns the row with the latest
	// recorded_at for the given executionID per ADR-0002 I4 and
	// ADR-0003 CC2 canonical-projection semantics. Returns
	// ErrExecutionNotFound (or an error wrapping it) if no rows
	// exist for that execution_id.
	QueryCurrentExecution(ctx context.Context, executionID string) (*ExecutionRow, error)

	// ListRunningOlderThan returns the canonical row of every
	// execution whose latest state is `running` and whose
	// `started_at` is strictly before the given cutoff. Consumed by
	// the orphan-run detector per ADR-0007 CC11 to identify
	// abandoned executions (engine restart / OOM / crash mid-
	// execution). Returns an empty slice (not nil) when no rows
	// match.
	//
	// The query targets the canonical-view semantics from ADR-0003
	// CC2: an execution that already has a terminal follow-up row
	// (e.g., aborted, success) is **not** returned even if its
	// initial `running` row's started_at is old — the canonical
	// view's latest-by-recorded_at projection filters it out.
	ListRunningOlderThan(ctx context.Context, before time.Time) ([]ExecutionRow, error)
}

// Store is the full result-write surface: Writer + Reader +
// schema initialization.
type Store interface {
	Writer
	Reader

	// EnsureSchema creates the dataset (if absent), the two
	// append-only tables (if absent), and the lazy
	// dq_executions_current view (best-effort against the emulator
	// per ADR-0010 lazy-view Partial row; always succeeds against
	// production). Idempotent: a second call is a no-op.
	EnsureSchema(ctx context.Context) error
}
