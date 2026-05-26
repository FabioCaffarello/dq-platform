// path: engine/internal/eval/baselines.go

package eval

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"dq-platform/engine/internal/runner"
)

// BaselineSource is the closed enum for `params.baseline.source`
// per ADR-0032 §"Platform-history baselines with optional static
// baselines".
type BaselineSource string

const (
	BaselineSourcePlatformHistory BaselineSource = "platform_history"
	BaselineSourceStatic          BaselineSource = "static"
)

// Aggregation is the closed enum for `params.baseline.aggregation`.
// Only the SQL-expressible aggregations are committed at v1.
type Aggregation string

const (
	AggregationMean   Aggregation = "mean"
	AggregationMedian Aggregation = "median"
	AggregationMin    Aggregation = "min"
	AggregationMax    Aggregation = "max"
	AggregationP50    Aggregation = "p50"
	AggregationP90    Aggregation = "p90"
	AggregationP95    Aggregation = "p95"
	AggregationP99    Aggregation = "p99"
)

// ToleranceType is the closed enum for tolerance kind.
type ToleranceType string

const (
	TolerancePercent  ToleranceType = "percent"
	ToleranceAbsolute ToleranceType = "absolute"
	ToleranceStddev   ToleranceType = "stddev"
)

// BaselineSpec is the parsed shape of one rule's
// `params.baseline` block. Two sub-modes share fields; the
// platform-history sub-mode fills the query-related fields, the
// static sub-mode fills `Value`.
type BaselineSpec struct {
	Source           BaselineSource
	ReferenceWindow  time.Duration // platform_history only
	MinSamples       int           // platform_history only; default 5
	Aggregation      Aggregation   // platform_history only
	Value            float64       // static only
	Tolerance        Tolerance
	HistoryFieldName string // JSON field inside evidence_summary to read (e.g., "row_count")
}

// Tolerance carries the operator-declared tolerance applied to
// the baseline value. The check passes when
// `|current - baseline| <= tolerance applied to baseline`.
type Tolerance struct {
	Type  ToleranceType
	Value float64
}

// BaselineResult is the output of ComputeBaseline. The caller
// (per-kind handler) compares its evaluation's current value
// against Baseline within Tolerance to decide pass/fail.
//
// SamplesUsed counts the `pass` rows the platform_history query
// found; for static baselines, SamplesUsed is 0. Insufficient
// samples (`SamplesUsed < spec.MinSamples` in platform_history
// mode) is signaled by Status == ResultDegraded.
type BaselineResult struct {
	Baseline    float64
	SamplesUsed int
	// EffectiveReferenceWindow is the actual reference window
	// the query covered after applying the
	// `min(declared, ResultsRetention)` cap. Exposed so the
	// per-kind handler can include it in the EvidenceSummary
	// for the degraded path.
	EffectiveReferenceWindow time.Duration
}

// referenceWindowPattern matches the lexical grammar from
// ADR-0032 §"reference_window field semantics":
// `^[0-9]+(ms|s|m|h|d)$`. The `d` suffix is the extension over
// ADR-0024's record-mode window grammar.
var referenceWindowPattern = regexp.MustCompile(`^([0-9]+)(ms|s|m|h|d)$`)

// ParseReferenceWindow parses an ADR-0032 reference_window
// literal (e.g., "7d", "24h", "30m") into a Duration. Returns
// an error for any value outside the grammar.
func ParseReferenceWindow(raw string) (time.Duration, error) {
	matches := referenceWindowPattern.FindStringSubmatch(raw)
	if matches == nil {
		return 0, fmt.Errorf("reference_window %q does not match grammar %s", raw, referenceWindowPattern.String())
	}
	n, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("reference_window %q: %w", raw, err)
	}
	if n <= 0 {
		return 0, fmt.Errorf("reference_window %q must be positive", raw)
	}
	switch matches[2] {
	case "ms":
		return time.Duration(n) * time.Millisecond, nil
	case "s":
		return time.Duration(n) * time.Second, nil
	case "m":
		return time.Duration(n) * time.Minute, nil
	case "h":
		return time.Duration(n) * time.Hour, nil
	case "d":
		return time.Duration(n) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("reference_window %q: unreachable suffix", raw)
}

// ParseBaselineSpec parses one rule's `params.baseline` block
// (typed-any from the YAML loader) into a BaselineSpec. Returns
// an error when the block is malformed or required fields are
// missing for the declared source mode.
//
// historyFieldName is the JSON-path field inside `evidence_summary`
// the per-kind handler reads to extract the historical numeric
// value (e.g., "row_count" for the row_count_within_baseline
// kind). Passed in so this helper stays kind-agnostic.
func ParseBaselineSpec(raw map[string]any, historyFieldName string) (BaselineSpec, error) {
	if historyFieldName == "" {
		return BaselineSpec{}, errors.New("ParseBaselineSpec: historyFieldName is required (kind-specific)")
	}
	src, ok := raw["source"].(string)
	if !ok || src == "" {
		return BaselineSpec{}, errors.New("params.baseline.source is required (one of platform_history|static)")
	}

	spec := BaselineSpec{HistoryFieldName: historyFieldName}
	switch BaselineSource(src) {
	case BaselineSourcePlatformHistory:
		spec.Source = BaselineSourcePlatformHistory
	case BaselineSourceStatic:
		spec.Source = BaselineSourceStatic
	default:
		return BaselineSpec{}, fmt.Errorf("params.baseline.source %q is not one of [platform_history, static]", src)
	}

	// Tolerance is required in both sub-modes.
	tolRaw, ok := raw["tolerance"].(map[string]any)
	if !ok {
		return BaselineSpec{}, errors.New("params.baseline.tolerance is required")
	}
	tolType, _ := tolRaw["type"].(string)
	tolVal, _ := numericValue(tolRaw["value"])
	switch ToleranceType(tolType) {
	case TolerancePercent, ToleranceAbsolute, ToleranceStddev:
		spec.Tolerance = Tolerance{Type: ToleranceType(tolType), Value: tolVal}
	default:
		return BaselineSpec{}, fmt.Errorf("params.baseline.tolerance.type %q is not one of [percent, absolute, stddev]", tolType)
	}

	if spec.Source == BaselineSourceStatic {
		v, ok := numericValue(raw["value"])
		if !ok {
			return BaselineSpec{}, errors.New("params.baseline.value (numeric) is required for source=static")
		}
		spec.Value = v
		// stddev tolerance is platform_history-only per ADR-0032.
		if spec.Tolerance.Type == ToleranceStddev {
			return BaselineSpec{}, errors.New("tolerance.type=stddev is only valid for source=platform_history")
		}
		return spec, nil
	}

	// platform_history fields.
	refWindow, _ := raw["reference_window"].(string)
	if refWindow == "" {
		return BaselineSpec{}, errors.New("params.baseline.reference_window is required for source=platform_history")
	}
	d, err := ParseReferenceWindow(refWindow)
	if err != nil {
		return BaselineSpec{}, err
	}
	spec.ReferenceWindow = d

	agg, _ := raw["aggregation"].(string)
	if agg == "" {
		return BaselineSpec{}, errors.New("params.baseline.aggregation is required for source=platform_history")
	}
	switch Aggregation(agg) {
	case AggregationMean, AggregationMedian, AggregationMin, AggregationMax,
		AggregationP50, AggregationP90, AggregationP95, AggregationP99:
		spec.Aggregation = Aggregation(agg)
	default:
		return BaselineSpec{}, fmt.Errorf("params.baseline.aggregation %q is not one of [mean, median, min, max, p50, p90, p95, p99]", agg)
	}

	// min_samples: optional, default 5.
	if v, ok := numericValue(raw["min_samples"]); ok && v > 0 {
		spec.MinSamples = int(v)
	} else {
		spec.MinSamples = 5
	}

	return spec, nil
}

// ComputeBaseline returns the baseline value for one
// (entity, check_id) pair per ADR-0032's framework. Two sub-modes:
//
//   - source=static: returns (spec.Value, 0, effectiveWindow=0, nil).
//   - source=platform_history: runs the parameterized SQL against
//     `dq_check_results` + `dq_executions`, applying the
//     effective-reference-window cap.
//
// The caller (per-kind handler) compares its current evaluation's
// value against the returned baseline within the rule's tolerance
// and decides pass/fail. When SamplesUsed < spec.MinSamples in
// platform_history mode, the caller emits ResultDegraded with the
// `insufficient_baseline_samples` reason per ADR-0032's
// sparse-history policy.
func ComputeBaseline(
	ctx context.Context,
	e *Evaluator,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
	baseline BaselineSpec,
) (BaselineResult, error) {
	if baseline.Source == BaselineSourceStatic {
		return BaselineResult{
			Baseline:                 baseline.Value,
			SamplesUsed:              0,
			EffectiveReferenceWindow: 0,
		}, nil
	}

	// platform_history path requires the engine's results-table
	// location + per-env retention.
	if e.resultsProject == "" || e.resultsDataset == "" {
		return BaselineResult{}, errors.New("ComputeBaseline: ResultsProject + ResultsDataset are required for platform_history baselines (wire from EnvConfig in main.go)")
	}

	// Effective reference window cap per ADR-0032 §"Effective
	// reference window vs declared".
	effective := baseline.ReferenceWindow
	if e.resultsRetention > 0 && effective > e.resultsRetention {
		effective = e.resultsRetention
	}

	sql := buildBaselineSQL(
		e.resultsProject, e.resultsDataset,
		baseline.HistoryFieldName, baseline.Aggregation,
	)
	q := e.client.Query(sql)
	q.Parameters = []bigquery.QueryParameter{
		{Name: "entity", Value: trigger.Entity},
		{Name: "check_id", Value: spec.CheckID},
		{Name: "window_end", Value: trigger.WindowEnd},
		{Name: "window_start_cutoff", Value: trigger.WindowEnd.Add(-effective)},
	}
	it, err := q.Read(ctx)
	if err != nil {
		return BaselineResult{}, fmt.Errorf("ComputeBaseline: query: %w", err)
	}
	var row struct {
		Baseline    bigquery.NullFloat64 `bigquery:"baseline_value"`
		SamplesUsed int64                `bigquery:"samples_used"`
	}
	if err := it.Next(&row); err != nil {
		if errors.Is(err, iterator.Done) {
			return BaselineResult{
				Baseline:                 0,
				SamplesUsed:              0,
				EffectiveReferenceWindow: effective,
			}, nil
		}
		return BaselineResult{}, fmt.Errorf("ComputeBaseline: scan: %w", err)
	}
	result := BaselineResult{
		SamplesUsed:              int(row.SamplesUsed),
		EffectiveReferenceWindow: effective,
	}
	if row.Baseline.Valid {
		result.Baseline = row.Baseline.Float64
	}
	return result, nil
}

// buildBaselineSQL emits the parameterized baseline query
// template per ADR-0032 §"Baseline query (platform-history
// mode)". The historyField is a JSON path extracted from
// `evidence_summary` (`$.<field>`); the aggregation wraps the
// extracted-and-cast expression.
//
// The query reads only `pass` results (per the ADR commitment)
// and filters by `executed_at` in a half-open interval
// `[window_end - effective_reference_window, window_end)`.
// `executed_at` is the per-check timestamp on dq_check_results
// per ADR-0003 §7 — partitioning-pruning-friendly per ADR-0031.
func buildBaselineSQL(project, dataset, historyField string, agg Aggregation) string {
	innerExpr := fmt.Sprintf("CAST(JSON_VALUE(cr.evidence_summary, '$.%s') AS FLOAT64)", historyField)
	aggExpr := aggregationExpr(agg, innerExpr)
	tableCR := fmt.Sprintf("%s.%s.dq_check_results", project, dataset)
	tableEX := fmt.Sprintf("%s.%s.dq_executions", project, dataset)
	return fmt.Sprintf(`SELECT
  %s AS baseline_value,
  COUNT(*) AS samples_used
FROM `+"`%s`"+` cr
JOIN `+"`%s`"+` ex
  ON ex.execution_id = cr.execution_id
 AND ex.attempt_id   = cr.attempt_id
WHERE
  ex.entity = @entity
  AND cr.check_id = @check_id
  AND cr.result = 'pass'
  AND cr.executed_at <  @window_end
  AND cr.executed_at >= @window_start_cutoff`,
		aggExpr, tableCR, tableEX,
	)
}

// aggregationExpr wraps innerExpr in the appropriate aggregation
// function. Percentiles use BigQuery's `APPROX_QUANTILES(col,
// 100)[OFFSET(N)]` idiom; mean/min/max use the bare aggregation.
func aggregationExpr(agg Aggregation, innerExpr string) string {
	switch agg {
	case AggregationMean:
		return fmt.Sprintf("AVG(%s)", innerExpr)
	case AggregationMin:
		return fmt.Sprintf("MIN(%s)", innerExpr)
	case AggregationMax:
		return fmt.Sprintf("MAX(%s)", innerExpr)
	case AggregationMedian, AggregationP50:
		return fmt.Sprintf("APPROX_QUANTILES(%s, 100)[OFFSET(50)]", innerExpr)
	case AggregationP90:
		return fmt.Sprintf("APPROX_QUANTILES(%s, 100)[OFFSET(90)]", innerExpr)
	case AggregationP95:
		return fmt.Sprintf("APPROX_QUANTILES(%s, 100)[OFFSET(95)]", innerExpr)
	case AggregationP99:
		return fmt.Sprintf("APPROX_QUANTILES(%s, 100)[OFFSET(99)]", innerExpr)
	}
	return fmt.Sprintf("AVG(%s)", innerExpr)
}

// numericValue coerces a YAML-loaded any to a float64. The YAML
// loader produces int64 for whole numbers and float64 for
// fractional numbers; the JSON-tagged params loader produces
// float64 for both. Accept both shapes.
func numericValue(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		// Numbers serialized as strings (e.g., YAML quoting
		// quirk) — accept if parseable.
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	}
	return 0, false
}
