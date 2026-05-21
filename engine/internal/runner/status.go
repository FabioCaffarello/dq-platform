// path: engine/internal/runner/status.go

package runner

import (
	"errors"

	"dq-platform/engine/internal/results"
)

// ErrEmptyResultMultiset is returned by MapStatus when the input
// slice is empty. An empty check-result multiset means no checks
// ran for this attempt — which only happens via the pre-check
// entity-level error path per ADR-0004 CC2 branch 2 / ADR-0007
// CC8. That path is the caller's responsibility (write a terminal
// error row directly, without invoking MapStatus). MapStatus
// itself does not write the error row.
var ErrEmptyResultMultiset = errors.New("empty check-result multiset; caller must use the pre-check error path")

// MapStatus implements the ADR-0004 CC2 execution-status mapping
// as a pure function of the check-result multiset. Branches are
// applied in the order committed by the ADR:
//
//  1. (Global engine halt → aborted; the caller handles this
//     branch externally — the orphan detector or the cost-ceiling
//     enforcement path. MapStatus does not produce aborted.)
//  2. (Pre-check entity-level problem → error with no check rows;
//     the caller handles this branch externally. MapStatus
//     returns ErrEmptyResultMultiset if called with no results.)
//  3. Every element is pass → success.
//  4. Every element is error → error.
//  5. Otherwise (mixed) → failed.
//
// The five branches are mutually exclusive by construction.
func MapStatus(checkResults []results.CheckResult) (results.ExecutionStatus, error) {
	if len(checkResults) == 0 {
		return "", ErrEmptyResultMultiset
	}

	allPass := true
	allError := true
	for _, r := range checkResults {
		if r != results.ResultPass {
			allPass = false
		}
		if r != results.ResultError {
			allError = false
		}
	}

	switch {
	case allPass:
		return results.StatusSuccess, nil
	case allError:
		return results.StatusError, nil
	default:
		return results.StatusFailed, nil
	}
}
