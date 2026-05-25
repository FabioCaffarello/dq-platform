// path: engine/internal/results/types.go

package results

import "time"

// ExecutionRow is one row in dq_executions. Field set mirrors
// ADR-0003 CC3 (required columns on dq_executions). The table is
// append-only per ADR-0003 CC1; one ExecutionRow is one state-
// transition row. The composite key is
// (ExecutionID, AttemptID, RecordedAt).
//
// Required (always set):
//   - ExecutionID, AttemptID, RecordedAt, Status, EngineVersion,
//     RulesetVersion, Entity, TriggerSource.
//
// Nullable (set on terminal rows only):
//   - StartedAt, CompletedAt — nullable for the running transition
//     row written before the engine begins processing; required for
//     terminal rows per ADR-0003 CC3.
//   - ErrorSummary — populated when Status is failed, error, or
//     aborted.
//   - SupersedesExecutionID — populated only on the first state-
//     transition row of an operator-rerun attempt per ADR-0003 CC5.
//
// The Hash field of the loaded Manifest (ADR-0007 CC3) is held in
// engine memory as the in-flight manifest_hash; it is NOT a column
// of dq_executions (ADR-0007 CC3). The forensic link from a
// persisted row to a manifest is RulesetVersion.
type ExecutionRow struct {
	ExecutionID    string          // 64-char lowercase hex per ADR-0002 CC7
	AttemptID      string          // UUID per ADR-0003 CC4
	RecordedAt     time.Time       // µs precision UTC per ADR-0003 CC3
	Status         ExecutionStatus // ADR-0003 CC6
	Mode           Mode            // ADR-0021: set | record (Wave-S)
	EngineVersion  string
	RulesetVersion string
	Entity         string
	TriggerSource  TriggerSource // ADR-0002 CC6

	StartedAt             *time.Time
	CompletedAt           *time.Time
	ErrorSummary          *string
	SupersedesExecutionID *string
}

// Mode is the closed-enum execution mode committed by ADR-0021.
// The column on dq_executions records the mode the rule declared
// at evaluation time; the orphan detector preserves the mode of
// the running row it finalizes.
type Mode string

const (
	ModeSet    Mode = "set"
	ModeRecord Mode = "record"
)

// CheckResultRow is one row in dq_check_results. Field set mirrors
// ADR-0003 CC7. The table is append-only per ADR-0003 CC1; the
// composite key is (ExecutionID, AttemptID, CheckID).
//
// EvidenceSummary is a structured aggregate (rows scanned, rows
// failing, etc.) — its exact shape is per-check-kind and Phase-4+
// scaffolding.
//
// SampleViolatingRows is a repeated record capped at a configured
// limit per ADR-0007 / foundation doc 05 §"Evidence Retention".
type CheckResultRow struct {
	ExecutionID         string
	AttemptID           string
	CheckID             string
	Result              CheckResult // ADR-0003 CC7 / ADR-0004 CC1
	ExecutedAt          time.Time
	EngineVersion       string
	EvidenceSummary     map[string]any
	SampleViolatingRows []map[string]any
}
