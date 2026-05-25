// path: engine/internal/eval/row_count_positive.go

package eval

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// rowCountPositiveThreshold is the inclusive lower bound a passing
// row_count_positive check must exceed (row_count > threshold ⇒
// pass). v1 hardcodes 0; configurable thresholds land in a future
// schema amendment.
const rowCountPositiveThreshold = int64(0)

// bqIdentifierPattern restricts BigQuery project / dataset / table
// identifiers to safe characters before they are interpolated
// into the backtick-quoted table reference. The dsl/spec parser
// applies the same patterns at rule-load time; the evaluator
// repeats the check defensively for any code path that may bypass
// the parser (direct CheckSpec construction in tests).
var bqIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// errSourceMissing is returned when a set-mode CheckSpec carries
// no source descriptor. v1 rules surface here because v1 has no
// source descriptor; the operator's migration to v2 closes this
// gap.
var errSourceMissing = errors.New(
	"eval: set-mode check requires a BigQuery source descriptor (v2 rule.source per ADR-0023; v1 rules pre-migration cannot run set-mode evaluations on the post-Wave-S engine)",
)

// evaluateRowCountPositive runs the set.row_count_positive check:
//
//	SELECT COUNT(*) AS row_count FROM `<project>.<dataset>.<table>`
//
// Project / dataset / table come from the rule's source descriptor
// per ADR-0023; the evaluator no longer pins a deployment-wide
// SourceProject / SourceDataset.
//
// Maps the count to results.CheckResult:
//
//   - row_count > 0  → ResultPass
//   - row_count == 0 → ResultFail
//   - BigQuery error → ResultError (ADR-0004 CC1)
func (e *Evaluator) evaluateRowCountPositive(
	ctx context.Context,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
) (runner.Evaluation, error) {
	if spec.Source == nil || spec.Source.Type != "bigquery" {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "missing_or_non_bigquery_source",
			},
		}, errSourceMissing
	}
	if err := validateBQIdentifiers(spec.Source); err != nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "invalid_source_identifier",
				"error":  err.Error(),
			},
		}, err
	}

	tableRef := fmt.Sprintf("%s.%s.%s", spec.Source.ProjectID, spec.Source.DatasetID, spec.Source.TableID)
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

// validateBQIdentifiers re-checks the BigQuery identifiers on the
// CheckSpec.Source descriptor before they are interpolated into
// the table reference. Same pattern as the dsl/spec parser;
// belt-and-suspenders against direct CheckSpec construction.
func validateBQIdentifiers(s *runner.RuleSource) error {
	if !bqIdentifierPattern.MatchString(s.ProjectID) {
		return fmt.Errorf("source.project_id %q does not match %s", s.ProjectID, bqIdentifierPattern.String())
	}
	if !bqIdentifierPattern.MatchString(s.DatasetID) {
		return fmt.Errorf("source.dataset_id %q does not match %s", s.DatasetID, bqIdentifierPattern.String())
	}
	if !bqIdentifierPattern.MatchString(s.TableID) {
		return fmt.Errorf("source.table_id %q does not match %s", s.TableID, bqIdentifierPattern.String())
	}
	return nil
}

// Compile-time assertion that *Evaluator satisfies the runner's
// CheckEvaluator interface. Catches a signature-drift mistake at
// build time rather than at runtime.
var _ runner.CheckEvaluator = (*Evaluator)(nil)
