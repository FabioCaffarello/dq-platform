// path: engine/internal/results/status.go

package results

// ExecutionStatus is the closed enum for dq_executions.status per
// ADR-0003 CC6. Extension is additive — adding a new value does not
// break existing rows. Removing or renaming a value is breaking and
// requires a future ADR.
type ExecutionStatus string

const (
	// StatusRunning is the transition row written by the trigger
	// handler before the engine begins evaluating checks.
	StatusRunning ExecutionStatus = "running"

	// StatusSuccess is the terminal status when every check
	// evaluated to pass per ADR-0004 CC2 branch 3.
	StatusSuccess ExecutionStatus = "success"

	// StatusFailed is the terminal status for any mixed-result
	// execution (at least one fail / degraded / error with at
	// least one successful evaluation) per ADR-0004 CC2 branch 5.
	StatusFailed ExecutionStatus = "failed"

	// StatusError is the terminal status when the entity could
	// not be evaluated — every check errored (ADR-0004 CC2
	// branch 4) or a pre-check entity-level problem was detected
	// before any check ran (ADR-0004 CC2 branch 2 / ADR-0007 CC8).
	StatusError ExecutionStatus = "error"

	// StatusAborted is the terminal status for a halt mid-
	// execution (cost ceiling, engine restart, OOM, concurrency
	// budget, operator-issued abort) per ADR-0007 CC10. Orphan-
	// run detection (ADR-0007 CC11) also finalizes abandoned
	// running rows to aborted.
	StatusAborted ExecutionStatus = "aborted"
)

// CheckResult is the closed enum for dq_check_results.result per
// ADR-0003 CC7 and ADR-0004 CC1.
type CheckResult string

const (
	// ResultPass — the check's query executed and the data met
	// the rule's pass condition.
	ResultPass CheckResult = "pass"

	// ResultFail — the check's query executed and the data did
	// not meet the rule's pass condition (above the warn
	// threshold if one is defined).
	ResultFail CheckResult = "fail"

	// ResultDegraded — the check's query executed and the data
	// fell into a warning band (between pass and fail).
	ResultDegraded CheckResult = "degraded"

	// ResultError — the check's query did not execute
	// successfully (compilation error, missing source, quota
	// exhaustion, exceeded retry budget, evaluation-budget
	// timeout).
	ResultError CheckResult = "error"
)

// TriggerSource is the closed enum for dq_executions.trigger_source
// per ADR-0002 CC6. The trigger_source value is one of the five
// inputs to the execution_id hash; the API layer enforces that the
// trigger-handler endpoint and the Admin-API rerun endpoint each
// produce only their own value.
type TriggerSource string

const (
	// TriggerScheduler — periodic invocation from the scheduler
	// subsystem.
	TriggerScheduler TriggerSource = "scheduler"

	// TriggerManual — one-off invocation via the trigger API by
	// a human with appropriate permissions (not a rerun).
	TriggerManual TriggerSource = "manual"

	// TriggerOperatorRerun — deliberate re-evaluation of a prior
	// run via the Admin API; always paired with a non-nil
	// SupersedesExecutionID on the new row per ADR-0002 CC5.
	TriggerOperatorRerun TriggerSource = "operator-rerun"
)
