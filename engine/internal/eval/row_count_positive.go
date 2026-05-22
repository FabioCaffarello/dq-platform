// path: engine/internal/eval/row_count_positive.go

package eval

import (
	"context"
	"fmt"
	"regexp"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// rowCountPositiveThreshold is the inclusive lower bound a passing
// row_count_positive check must exceed (row_count > threshold ⇒
// pass). v1 hardcodes 0; configurable thresholds land in a future
// schema amendment.
//
// New contribution proposed here, requires review: the threshold
// model itself (a single inclusive lower bound) is not directly
// committed by any prior ADR. A richer threshold contract (warning
// bands per ADR-0004 ResultDegraded, time-windowed minimums, etc.)
// is a future ADR amendment.
const rowCountPositiveThreshold = int64(0)

// entityPattern restricts trigger.Entity to a safe identifier
// shape. The HTTP trigger handler's decoder (engine/internal/api)
// applies an analogous validation upstream, but the runner can be
// driven by direct Go callers (the runner integration tests, future
// internal triggers) that bypass the decoder. The evaluator runs
// the same defensive check so a malformed Entity cannot reach
// BigQuery's table reference under any code path. Pattern mirrors
// the spirit of ADR-0002 §2 input safety: ASCII identifier
// characters only, no special characters that could escape the
// backtick-quoted table reference.
var entityPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Note on EvidenceSummary visibility on error paths:
//
// Returning a non-nil error from Evaluate causes the runner's
// check loop (engine/internal/runner/runner.go, the `evalErr != nil`
// branch in Run) to overwrite the Evaluation with the minimal
// `Evaluation{Result: ResultError}`. The detailed EvidenceSummary
// built below on error paths (reason, table_ref, error message)
// is therefore visible to direct callers and unit tests but
// dropped before persistence in dq_check_results.
//
// Preserving the EvidenceSummary across the error branch is a
// runner-package concern (the overwrite happens in runner.go).
// W3-P6c documents the gap; the actual runner change is deferred
// to a follow-up session per R4 (one topic per session).

// evaluateRowCountPositive runs the row_count_positive check:
//
//	SELECT COUNT(*) AS row_count FROM `<project>.<dataset>.<entity>`
//
// Maps the count to results.CheckResult:
//
//   - row_count > 0  → ResultPass
//   - row_count == 0 → ResultFail
//   - BigQuery error → ResultError (ADR-0004 CC1)
//
// EvidenceSummary carries row_count, threshold, table_ref, and
// kind so the dq_check_results row preserves enough forensic
// information for a future read API to reconstruct the result
// without rerunning the query.
func (e *Evaluator) evaluateRowCountPositive(
	ctx context.Context,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
) (runner.Evaluation, error) {
	if e.sourceDataset == "" {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "source_dataset_not_configured",
			},
		}, errSourceDatasetMissing
	}
	if !entityPattern.MatchString(trigger.Entity) {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "invalid_entity_identifier",
				"entity": trigger.Entity,
			},
		}, fmt.Errorf("row_count_positive: entity %q does not match identifier pattern %s", trigger.Entity, entityPattern.String())
	}

	tableRef := fmt.Sprintf("%s.%s.%s", e.sourceProject, e.sourceDataset, trigger.Entity)
	sql := fmt.Sprintf("SELECT COUNT(*) AS row_count FROM `%s`", tableRef)

	q := e.client.Query(sql)
	it, err := q.Read(ctx)
	if err != nil {
		e.logger.Warn("row_count_positive query read failed",
			"table_ref", tableRef,
			"check_id", spec.CheckID,
			"entity", trigger.Entity,
			"error", err.Error(),
			"adr_reference", "ADR-0004 CC1",
		)
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":      spec.Kind,
				"table_ref": tableRef,
				"reason":    "query_read_failed",
				"error":     err.Error(),
			},
		}, fmt.Errorf("row_count_positive: query %q: %w", tableRef, err)
	}

	var row struct {
		RowCount int64 `bigquery:"row_count"`
	}
	if err := it.Next(&row); err != nil {
		e.logger.Warn("row_count_positive returned no rows",
			"table_ref", tableRef,
			"check_id", spec.CheckID,
			"entity", trigger.Entity,
			"error", err.Error(),
			"adr_reference", "ADR-0004 CC1",
		)
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":      spec.Kind,
				"table_ref": tableRef,
				"reason":    "no_rows_from_count_query",
				"error":     err.Error(),
			},
		}, fmt.Errorf("row_count_positive: %q returned no rows: %w", tableRef, err)
	}

	result := results.ResultPass
	if row.RowCount <= rowCountPositiveThreshold {
		result = results.ResultFail
	}

	return runner.Evaluation{
		Result: result,
		EvidenceSummary: map[string]any{
			"kind":      spec.Kind,
			"table_ref": tableRef,
			"row_count": row.RowCount,
			"threshold": rowCountPositiveThreshold,
		},
	}, nil
}

// Compile-time assertion that *Evaluator satisfies the runner's
// CheckEvaluator interface. Catches a signature-drift mistake at
// build time rather than at runtime.
var _ runner.CheckEvaluator = (*Evaluator)(nil)
