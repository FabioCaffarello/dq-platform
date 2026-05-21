// path: engine/internal/runner/precheck.go

package runner

import "context"

// EntityPrecheck performs the lightweight pre-check operation
// from ADR-0007 CC8: the trigger handler determines source-table
// existence for the entity before any check is evaluated, using
// an operation substantially cheaper than evaluating any
// individual check.
//
// Returning false (the source is absent) causes the runner to
// write a terminal `error` row directly with no check rows per
// ADR-0007 CC8 / ADR-0004 CC2 branch 2.
//
// Returning an error signals an operational failure (the
// pre-check itself could not complete); the runner treats this
// as fatal for the attempt — the terminal row is still `error`
// but with a different error_summary so the operational failure
// is distinguishable from "source is absent".
//
// Phase-4c ships a no-op default (NoopPrecheck) that always
// returns true. Phase 6 wires a real BigQuery-metadata precheck
// (e.g., `tables.get` against the entity's source table).
type EntityPrecheck interface {
	SourceExists(ctx context.Context, entity string) (bool, error)
}

// NoopPrecheck always returns true. Used by Phase 4c when no
// real precheck is wired and by tests that want to exercise the
// happy path.
type NoopPrecheck struct{}

func (NoopPrecheck) SourceExists(_ context.Context, _ string) (bool, error) { return true, nil }
