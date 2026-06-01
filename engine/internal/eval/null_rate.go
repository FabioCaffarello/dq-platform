// path: engine/internal/eval/null_rate.go

package eval

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// setNullRateHandler is the registry entry for the set.null_rate
// kind. Thin adapter onto the per-kind method on Evaluator so the
// registry holds a uniform Handler signature (same shape as
// setRowCountPositiveHandler).
func setNullRateHandler(ctx context.Context, e *Evaluator, spec runner.CheckSpec, trigger runner.TriggerRequest) (runner.Evaluation, error) {
	return e.evaluateNullRate(ctx, spec, trigger)
}

// evaluateNullRate runs the set.null_rate check:
//
//	SELECT COUNTIF(`<column>` IS NULL) AS null_count,
//	       COUNT(*)                    AS total
//	FROM `<project>.<dataset>.<table>`
//	[ WHERE `<partition_column>` >= @window_start
//	    AND `<partition_column>` <  @window_end ]
//
// Project / dataset / table come from the rule's source descriptor
// per ADR-0023 — no deployment-wide source pin.
//
// When `source.partition_column` is set, the half-open-interval
// predicate uses the partition column directly so BigQuery's
// partition pruning fires (same shape as the B2-12 retrofit on
// row_count_positive). Window endpoints bind as `@window_start` /
// `@window_end` parameterized values so the SQL is injection-safe.
//
// The `column` and `max_null_rate` parameters come from
// `spec.Params` per ADR-0022's per-check params block; the catalog
// declares both as required with `additionalProperties: false`.
// Per P1 (rules remain declarative — no escape hatch) the operator
// declares only a column name plus a threshold, never raw SQL; the
// engine builds the query.
//
// Identifier validation is defense-in-depth: the catalog schema +
// the dsl/spec parser already constrain `column` to the BigQuery
// column-name grammar, but direct CheckSpec construction (tests,
// future callers) bypasses both. The handler re-checks via
// bqColumnPattern (shared with row_count_positive's partition-column
// path) before the column is interpolated into the SQL.
//
// Cost posture (ADR-0029):
//
//   - With `source.partition_column` set, the WHERE predicate
//     references the partition column directly so BigQuery's
//     partition pruning fires; bytes scanned scale with the window,
//     matching row_count_positive's partitioned form.
//   - Without `source.partition_column`, the query has no metadata
//     parity with row_count_positive's `COUNT(*)`: `COUNT(*)` reads
//     the table-row-count metadata, but `COUNTIF(<column> IS NULL)`
//     requires reading every value of `column` across every
//     partition. Bytes scanned scale with the column footprint
//     across the table, not the window. Rule authors should declare
//     `source.partition_column` whenever the source table is
//     partitioned — leaving it unset is supported, but the cost
//     ceiling at ADR-0029 may bite on large tables.
//
// Mapping to results.CheckResult:
//
//   - 0 ≤ null_rate ≤ max_null_rate → ResultPass
//   - null_rate > max_null_rate     → ResultFail
//   - total == 0                    → ResultPass with `vacuous: true`
//     (no records → no null violations; the operator pairs this
//     check with set.row_count_positive for liveness coverage)
//   - BigQuery error                → ResultError (ADR-0004 CC1)
//   - missing / invalid params      → ResultError (ADR-0004 CC1)
//   - missing / non-bigquery source → ResultError (ADR-0004 CC1)
func (e *Evaluator) evaluateNullRate(
	ctx context.Context,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
) (runner.Evaluation, error) {
	column, maxNullRate, err := parseNullRateParams(spec.Params)
	if err != nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "invalid_params",
				"error":  err.Error(),
			},
		}, fmt.Errorf("set.null_rate: %w", err)
	}

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
	sql := nullRateSQL(tableRef, column, spec.Source.PartitionColumn)

	q := e.client.Query(sql)
	if spec.Source.PartitionColumn != "" {
		q.Parameters = []bigquery.QueryParameter{
			{Name: "window_start", Value: trigger.WindowStart},
			{Name: "window_end", Value: trigger.WindowEnd},
		}
	}
	it, err := q.Read(ctx)
	if err != nil {
		e.logger.Warn("null_rate query read failed",
			"table_ref", tableRef,
			"column", column,
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
				"column":    column,
				"reason":    "query_read_failed",
				"error":     err.Error(),
			},
		}, fmt.Errorf("null_rate: query %q col %q: %w", tableRef, column, err)
	}

	var row struct {
		NullCount int64 `bigquery:"null_count"`
		Total     int64 `bigquery:"total"`
	}
	if err := it.Next(&row); err != nil {
		e.logger.Warn("null_rate returned no rows",
			"table_ref", tableRef,
			"column", column,
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
				"column":    column,
				"reason":    "no_rows_from_count_query",
				"error":     err.Error(),
			},
		}, fmt.Errorf("null_rate: %q col %q returned no rows: %w", tableRef, column, err)
	}

	result, nullRate, vacuous := decideNullRate(row.NullCount, row.Total, maxNullRate)
	evidence := map[string]any{
		"kind":          spec.Kind,
		"table_ref":     tableRef,
		"column":        column,
		"null_count":    row.NullCount,
		"total":         row.Total,
		"null_rate":     nullRate,
		"max_null_rate": maxNullRate,
	}
	if vacuous {
		evidence["vacuous"] = true
	}
	return runner.Evaluation{
		Result:          result,
		EvidenceSummary: evidence,
	}, nil
}

// decideNullRate maps the (null_count, total, max_null_rate) triple
// to a CheckResult plus the computed rate. Extracted so the
// threshold semantics are unit-testable without a BigQuery client.
//
//   - total == 0           → ResultPass, rate=0.0, vacuous=true.
//     set.row_count_positive owns the liveness contract; an empty
//     window carries no null violations to assert against.
//   - rate ≤ max_null_rate → ResultPass.
//   - rate >  max_null_rate → ResultFail.
func decideNullRate(nullCount, total int64, maxNullRate float64) (results.CheckResult, float64, bool) {
	if total == 0 {
		return results.ResultPass, 0.0, true
	}
	rate := float64(nullCount) / float64(total)
	if rate > maxNullRate {
		return results.ResultFail, rate, false
	}
	return results.ResultPass, rate, false
}

// parseNullRateParams reads the `column` (string) and
// `max_null_rate` (float64 in [0.0, 1.0]) entries from the
// per-check params block per ADR-0022. The catalog schema enforces
// the same constraints at rule-load time; the handler re-validates
// defensively against direct CheckSpec construction (tests, future
// callers) — same belt-and-suspenders posture row_count_positive
// applies to the source identifier checks.
//
// The column name is constrained to bqColumnPattern so the value
// interpolated into the SQL cannot widen the column expression
// (P1 — declarative, no escape hatch).
func parseNullRateParams(params map[string]any) (column string, maxNullRate float64, err error) {
	if params == nil {
		return "", 0, fmt.Errorf("params is required (column, max_null_rate)")
	}

	colAny, ok := params["column"]
	if !ok {
		return "", 0, fmt.Errorf("params.column is required")
	}
	column, ok = colAny.(string)
	if !ok {
		return "", 0, fmt.Errorf("params.column must be a string; got %T", colAny)
	}
	if !bqColumnPattern.MatchString(column) {
		return "", 0, fmt.Errorf("params.column %q does not match %s", column, bqColumnPattern.String())
	}

	rateAny, ok := params["max_null_rate"]
	if !ok {
		return "", 0, fmt.Errorf("params.max_null_rate is required")
	}
	switch v := rateAny.(type) {
	case float64:
		maxNullRate = v
	case float32:
		maxNullRate = float64(v)
	case int:
		maxNullRate = float64(v)
	case int64:
		maxNullRate = float64(v)
	default:
		return "", 0, fmt.Errorf("params.max_null_rate must be a number; got %T", rateAny)
	}
	if maxNullRate < 0.0 || maxNullRate > 1.0 {
		return "", 0, fmt.Errorf("params.max_null_rate %v out of range [0.0, 1.0]", maxNullRate)
	}

	return column, maxNullRate, nil
}

// nullRateSQL emits the SQL template for one (tableRef, column,
// partitionColumn) combination. Exported (package-internal) so the
// unit tests can keep the template under regression coverage
// without duplicating string formatting.
//
// Identifiers are backtick-quoted; window endpoints bind via
// parameterized `@window_start` / `@window_end` rather than string
// interpolation. The column appears once in the COUNTIF expression
// and (when set) the partition column appears in the WHERE clause —
// both gated by bqColumnPattern at the caller's validation step.
func nullRateSQL(tableRef, column, partitionColumn string) string {
	if partitionColumn == "" {
		return fmt.Sprintf(
			"SELECT COUNTIF(`%s` IS NULL) AS null_count, COUNT(*) AS total FROM `%s`",
			column, tableRef,
		)
	}
	return fmt.Sprintf(
		"SELECT COUNTIF(`%s` IS NULL) AS null_count, COUNT(*) AS total FROM `%s` "+
			"WHERE `%s` >= @window_start AND `%s` < @window_end",
		column, tableRef, partitionColumn, partitionColumn,
	)
}
