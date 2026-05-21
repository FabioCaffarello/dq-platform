// path: engine/internal/alerts/category.go

package alerts

import "dq-platform/engine/internal/results"

// Category is the alert-routing category committed by ADR-0006
// CC7 (and originally by ADR-0004 CC7). The category boundary is
// non-negotiable: routing code may not reassign a check `error`
// as a data_quality alert or vice versa.
type Category string

const (
	CategoryDataQuality Category = "data_quality"
	CategoryOperational Category = "operational"
)

// MapCategory implements the category-boundary table from
// ADR-0006 CC7. Returns ("", false) for inputs that do not
// warrant an alert (e.g., check result=pass or execution
// status=success).
//
// Inputs:
//   - source: who is emitting the event.
//   - result: the check result, for check-level events.
//   - status: the execution status, for execution-level events.
//
// Exactly one of `result` or `status` should be non-nil for
// runner-sourced events; non-runner sources (loader / scheduler
// / orphan_detector / trigger_handler) always map to
// operational regardless of either input.
func MapCategory(source EventSource, result *results.CheckResult, status *results.ExecutionStatus) (Category, bool) {
	// Non-runner sources are operational by definition per
	// ADR-0006 CC7 last four rows (loader failure, scheduler
	// reconciliation failure, trigger-handler retry exhaustion,
	// orphan finalization).
	if source != SourceRunner {
		return CategoryOperational, true
	}

	// Runner-sourced check-level event.
	if result != nil {
		switch *result {
		case results.ResultFail, results.ResultDegraded:
			return CategoryDataQuality, true
		case results.ResultError:
			return CategoryOperational, true
		case results.ResultPass:
			// No alert for passing checks (ADR-0006 CC7 first row).
			return "", false
		default:
			// Unknown enum value — fail loud by treating as
			// operational so the gap shows up downstream.
			return CategoryOperational, true
		}
	}

	// Runner-sourced execution-level event.
	if status != nil {
		switch *status {
		case results.StatusFailed:
			// ADR-0006 CC7 row 5: execution failed → per-check
			// fan-out. The execution-level event itself is
			// operational so consumers see "this execution had
			// failures" alongside the per-check data_quality
			// alerts.
			return CategoryOperational, true
		case results.StatusError, results.StatusAborted:
			return CategoryOperational, true
		case results.StatusSuccess, results.StatusRunning:
			return "", false
		default:
			return CategoryOperational, true
		}
	}

	// Neither result nor status set; nothing to emit.
	return "", false
}
