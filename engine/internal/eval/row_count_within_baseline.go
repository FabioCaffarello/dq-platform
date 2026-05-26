// path: engine/internal/eval/row_count_within_baseline.go

package eval

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/bigquery"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// rowCountWithinBaselineField is the evidence-summary JSON field
// the baseline query reads from historical `dq_check_results`
// rows. This kind reports `row_count` as the current value and
// reads `$.row_count` from history.
const rowCountWithinBaselineField = "row_count"

// setRowCountWithinBaselineHandler is the registry entry for the
// set.row_count_within_baseline kind. The first baselined kind
// committed by B2-14 per ADR-0032's framework.
//
// Evaluation sequence:
//
//  1. Parse `params.baseline` via ParseBaselineSpec.
//  2. Run the current-value count (same SQL as
//     set.row_count_positive, with optional partition pruning
//     per B2-12 when source.partition_column is set).
//  3. Run ComputeBaseline to get the historical value +
//     samples-used count.
//  4. If platform_history mode and samples_used < min_samples,
//     return ResultDegraded with reason
//     `insufficient_baseline_samples` per ADR-0032
//     §"Sparse-history policy".
//  5. Compute `|current - baseline|` and compare against the
//     rule's tolerance. Within tolerance → ResultPass; outside
//     → ResultFail.
//
// Maps to `runner.Evaluation`. Errors at any step that block
// the comparison map to ResultError per ADR-0004 CC1.
func setRowCountWithinBaselineHandler(ctx context.Context, e *Evaluator, spec runner.CheckSpec, trigger runner.TriggerRequest) (runner.Evaluation, error) {
	return e.evaluateRowCountWithinBaseline(ctx, spec, trigger)
}

func (e *Evaluator) evaluateRowCountWithinBaseline(
	ctx context.Context,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
) (runner.Evaluation, error) {
	// Step 1 — parse the baseline block from params.
	if spec.Params == nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "missing_params",
			},
		}, fmt.Errorf("set.row_count_within_baseline: params is required (baseline block)")
	}
	baselineRaw, ok := spec.Params["baseline"].(map[string]any)
	if !ok {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "missing_baseline_block",
			},
		}, fmt.Errorf("set.row_count_within_baseline: params.baseline is required")
	}
	baseline, err := ParseBaselineSpec(baselineRaw, rowCountWithinBaselineField)
	if err != nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "invalid_baseline_block",
				"error":  err.Error(),
			},
		}, fmt.Errorf("set.row_count_within_baseline: %w", err)
	}

	// Step 2 — current-value count (reuses the set.row_count_positive
	// SQL template via rowCountPositiveSQL, including the B2-12
	// partition predicate when source.partition_column is set).
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
	sql := rowCountPositiveSQL(tableRef, spec.Source.PartitionColumn)
	q := e.client.Query(sql)
	if spec.Source.PartitionColumn != "" {
		q.Parameters = []bigquery.QueryParameter{
			{Name: "window_start", Value: trigger.WindowStart},
			{Name: "window_end", Value: trigger.WindowEnd},
		}
	}
	it, err := q.Read(ctx)
	if err != nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":      spec.Kind,
				"table_ref": tableRef,
				"reason":    "current_value_query_failed",
				"error":     err.Error(),
			},
		}, fmt.Errorf("set.row_count_within_baseline: current-value query: %w", err)
	}
	var current struct {
		RowCount int64 `bigquery:"row_count"`
	}
	if err := it.Next(&current); err != nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":      spec.Kind,
				"table_ref": tableRef,
				"reason":    "current_value_no_rows",
				"error":     err.Error(),
			},
		}, fmt.Errorf("set.row_count_within_baseline: current-value scan: %w", err)
	}
	currentValue := float64(current.RowCount)

	// Step 3 — baseline lookup.
	br, err := ComputeBaseline(ctx, e, spec, trigger, baseline)
	if err != nil {
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":      spec.Kind,
				"table_ref": tableRef,
				"reason":    "baseline_query_failed",
				"error":     err.Error(),
			},
		}, fmt.Errorf("set.row_count_within_baseline: %w", err)
	}

	// Step 4 — sparse-history → ResultDegraded.
	if baseline.Source == BaselineSourcePlatformHistory && br.SamplesUsed < baseline.MinSamples {
		return runner.Evaluation{
			Result: results.ResultDegraded,
			EvidenceSummary: map[string]any{
				"kind":                       spec.Kind,
				"table_ref":                  tableRef,
				"reason":                     "insufficient_baseline_samples",
				"row_count":                  current.RowCount,
				"samples_used":               br.SamplesUsed,
				"min_samples":                baseline.MinSamples,
				"effective_reference_window": br.EffectiveReferenceWindow.String(),
			},
		}, nil
	}

	// Step 5 — pass/fail by tolerance.
	deviation := math.Abs(currentValue - br.Baseline)
	allowance := allowedDeviation(baseline.Tolerance, br.Baseline)
	result := results.ResultPass
	if deviation > allowance {
		result = results.ResultFail
	}
	return runner.Evaluation{
		Result: result,
		EvidenceSummary: map[string]any{
			"kind":                       spec.Kind,
			"table_ref":                  tableRef,
			"row_count":                  current.RowCount,
			"baseline":                   br.Baseline,
			"deviation":                  deviation,
			"allowed_deviation":          allowance,
			"tolerance_type":             string(baseline.Tolerance.Type),
			"tolerance_value":            baseline.Tolerance.Value,
			"samples_used":               br.SamplesUsed,
			"effective_reference_window": br.EffectiveReferenceWindow.String(),
			"baseline_source":            string(baseline.Source),
		},
	}, nil
}

// allowedDeviation translates the tolerance spec into an
// absolute numeric bound against which `|current - baseline|`
// is compared.
//
//   - percent: `tolerance.value / 100 * |baseline|`.
//   - absolute: `tolerance.value` directly.
//   - stddev: not implemented at v1 (requires the historical
//     sample set to compute, not just the aggregate); v1
//     returns 0 to make the check fail loudly if a rule
//     accidentally specifies stddev — operator amends the rule
//     to one of the other two types. ADR-0032 §"Tolerance"
//     committed stddev for completeness but the implementation
//     can land in a follow-up B2 when concrete operational
//     signal demands it. For v1 the linter could reject
//     stddev too; deferred.
func allowedDeviation(tol Tolerance, baseline float64) float64 {
	switch tol.Type {
	case TolerancePercent:
		return tol.Value / 100.0 * math.Abs(baseline)
	case ToleranceAbsolute:
		return tol.Value
	case ToleranceStddev:
		// v1 placeholder: treat as zero allowance, surfacing
		// the gap loudly via ResultFail when the operator picks
		// stddev. A future B2 implements it properly.
		return 0
	}
	return 0
}
