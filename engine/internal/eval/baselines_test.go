// path: engine/internal/eval/baselines_test.go

package eval

import (
	"strings"
	"testing"
	"time"
)

func TestParseReferenceWindow_AcceptsGrammar(t *testing.T) {
	cases := []struct {
		raw  string
		want time.Duration
	}{
		{"500ms", 500 * time.Millisecond},
		{"30s", 30 * time.Second},
		{"15m", 15 * time.Minute},
		{"24h", 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
	}
	for _, tc := range cases {
		got, err := ParseReferenceWindow(tc.raw)
		if err != nil {
			t.Errorf("ParseReferenceWindow(%q): unexpected error %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseReferenceWindow(%q) = %v; want %v", tc.raw, got, tc.want)
		}
	}
}

func TestParseReferenceWindow_RejectsMalformed(t *testing.T) {
	bad := []string{
		"",       // empty
		"7",      // no suffix
		"d",      // no number
		"7w",     // unsupported suffix (week)
		"7y",     // unsupported suffix (year)
		"-7d",    // negative
		"7.5d",   // decimal
		"7 d",    // space
		"seven",  // non-numeric
		"7days",  // long suffix
	}
	for _, raw := range bad {
		if _, err := ParseReferenceWindow(raw); err == nil {
			t.Errorf("ParseReferenceWindow(%q): want error, got nil", raw)
		}
	}
}

func TestParseReferenceWindow_RejectsZero(t *testing.T) {
	// Zero is grammatically valid (`0d`) but semantically meaningless —
	// the parser rejects it so a baselined rule cannot accidentally
	// declare a zero-width history window.
	if _, err := ParseReferenceWindow("0d"); err == nil {
		t.Error("ParseReferenceWindow(\"0d\"): want error for zero-width window, got nil")
	}
}

func TestParseBaselineSpec_PlatformHistory_HappyPath(t *testing.T) {
	raw := map[string]any{
		"source":           "platform_history",
		"reference_window": "7d",
		"aggregation":      "mean",
		"min_samples":      int64(10),
		"tolerance": map[string]any{
			"type":  "percent",
			"value": 5.0,
		},
	}
	spec, err := ParseBaselineSpec(raw, "row_count")
	if err != nil {
		t.Fatalf("ParseBaselineSpec: %v", err)
	}
	if spec.Source != BaselineSourcePlatformHistory {
		t.Errorf("Source = %q; want platform_history", spec.Source)
	}
	if spec.ReferenceWindow != 7*24*time.Hour {
		t.Errorf("ReferenceWindow = %v; want 168h", spec.ReferenceWindow)
	}
	if spec.Aggregation != AggregationMean {
		t.Errorf("Aggregation = %q; want mean", spec.Aggregation)
	}
	if spec.MinSamples != 10 {
		t.Errorf("MinSamples = %d; want 10", spec.MinSamples)
	}
	if spec.Tolerance.Type != TolerancePercent || spec.Tolerance.Value != 5.0 {
		t.Errorf("Tolerance = %+v; want {percent, 5}", spec.Tolerance)
	}
	if spec.HistoryFieldName != "row_count" {
		t.Errorf("HistoryFieldName = %q; want row_count", spec.HistoryFieldName)
	}
}

func TestParseBaselineSpec_PlatformHistory_DefaultMinSamples(t *testing.T) {
	raw := map[string]any{
		"source":           "platform_history",
		"reference_window": "7d",
		"aggregation":      "median",
		"tolerance":        map[string]any{"type": "absolute", "value": 100.0},
	}
	spec, err := ParseBaselineSpec(raw, "row_count")
	if err != nil {
		t.Fatalf("ParseBaselineSpec: %v", err)
	}
	if spec.MinSamples != 5 {
		t.Errorf("MinSamples = %d; want default 5", spec.MinSamples)
	}
}

func TestParseBaselineSpec_Static_HappyPath(t *testing.T) {
	raw := map[string]any{
		"source": "static",
		"value":  1000.0,
		"tolerance": map[string]any{
			"type":  "absolute",
			"value": 50.0,
		},
	}
	spec, err := ParseBaselineSpec(raw, "row_count")
	if err != nil {
		t.Fatalf("ParseBaselineSpec: %v", err)
	}
	if spec.Source != BaselineSourceStatic {
		t.Errorf("Source = %q; want static", spec.Source)
	}
	if spec.Value != 1000.0 {
		t.Errorf("Value = %v; want 1000", spec.Value)
	}
	if spec.Tolerance.Type != ToleranceAbsolute || spec.Tolerance.Value != 50.0 {
		t.Errorf("Tolerance = %+v; want {absolute, 50}", spec.Tolerance)
	}
}

func TestParseBaselineSpec_Static_RejectsStddevTolerance(t *testing.T) {
	raw := map[string]any{
		"source": "static",
		"value":  1000.0,
		"tolerance": map[string]any{
			"type":  "stddev",
			"value": 2.0,
		},
	}
	_, err := ParseBaselineSpec(raw, "row_count")
	if err == nil {
		t.Fatal("ParseBaselineSpec(static + stddev tolerance): want error, got nil")
	}
	if !strings.Contains(err.Error(), "stddev") {
		t.Errorf("error = %q; want mention of stddev", err.Error())
	}
}

func TestParseBaselineSpec_RejectsMalformed(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]any
	}{
		{
			name: "missing source",
			raw: map[string]any{
				"reference_window": "7d",
				"aggregation":      "mean",
				"tolerance":        map[string]any{"type": "percent", "value": 5.0},
			},
		},
		{
			name: "unknown source",
			raw: map[string]any{
				"source":    "fabricated",
				"tolerance": map[string]any{"type": "percent", "value": 5.0},
			},
		},
		{
			name: "missing tolerance",
			raw: map[string]any{
				"source":           "platform_history",
				"reference_window": "7d",
				"aggregation":      "mean",
			},
		},
		{
			name: "unknown tolerance type",
			raw: map[string]any{
				"source":           "platform_history",
				"reference_window": "7d",
				"aggregation":      "mean",
				"tolerance":        map[string]any{"type": "ratio", "value": 5.0},
			},
		},
		{
			name: "platform_history missing reference_window",
			raw: map[string]any{
				"source":      "platform_history",
				"aggregation": "mean",
				"tolerance":   map[string]any{"type": "percent", "value": 5.0},
			},
		},
		{
			name: "platform_history malformed reference_window",
			raw: map[string]any{
				"source":           "platform_history",
				"reference_window": "7weeks",
				"aggregation":      "mean",
				"tolerance":        map[string]any{"type": "percent", "value": 5.0},
			},
		},
		{
			name: "platform_history missing aggregation",
			raw: map[string]any{
				"source":           "platform_history",
				"reference_window": "7d",
				"tolerance":        map[string]any{"type": "percent", "value": 5.0},
			},
		},
		{
			name: "platform_history unknown aggregation",
			raw: map[string]any{
				"source":           "platform_history",
				"reference_window": "7d",
				"aggregation":      "geomean",
				"tolerance":        map[string]any{"type": "percent", "value": 5.0},
			},
		},
		{
			name: "static missing value",
			raw: map[string]any{
				"source":    "static",
				"tolerance": map[string]any{"type": "absolute", "value": 50.0},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseBaselineSpec(tc.raw, "row_count"); err == nil {
				t.Errorf("want error for %s, got nil", tc.name)
			}
		})
	}
}

func TestParseBaselineSpec_RejectsEmptyHistoryFieldName(t *testing.T) {
	raw := map[string]any{
		"source":           "platform_history",
		"reference_window": "7d",
		"aggregation":      "mean",
		"tolerance":        map[string]any{"type": "percent", "value": 5.0},
	}
	if _, err := ParseBaselineSpec(raw, ""); err == nil {
		t.Error("want error for empty historyFieldName, got nil")
	}
}

func TestAggregationExpr_EachKind(t *testing.T) {
	inner := "CAST(JSON_VALUE(cr.evidence_summary, '$.row_count') AS FLOAT64)"
	cases := []struct {
		agg  Aggregation
		want string
	}{
		{AggregationMean, "AVG(" + inner + ")"},
		{AggregationMin, "MIN(" + inner + ")"},
		{AggregationMax, "MAX(" + inner + ")"},
		{AggregationMedian, "APPROX_QUANTILES(" + inner + ", 100)[OFFSET(50)]"},
		{AggregationP50, "APPROX_QUANTILES(" + inner + ", 100)[OFFSET(50)]"},
		{AggregationP90, "APPROX_QUANTILES(" + inner + ", 100)[OFFSET(90)]"},
		{AggregationP95, "APPROX_QUANTILES(" + inner + ", 100)[OFFSET(95)]"},
		{AggregationP99, "APPROX_QUANTILES(" + inner + ", 100)[OFFSET(99)]"},
	}
	for _, tc := range cases {
		got := aggregationExpr(tc.agg, inner)
		if got != tc.want {
			t.Errorf("aggregationExpr(%q) = %q; want %q", tc.agg, got, tc.want)
		}
	}
}

func TestBuildBaselineSQL_Shape(t *testing.T) {
	sql := buildBaselineSQL("dq-local", "dq_results", "row_count", AggregationMean)
	for _, frag := range []string{
		"baseline_value",
		"samples_used",
		"`dq-local.dq_results.dq_check_results`",
		"`dq-local.dq_results.dq_executions`",
		"@entity",
		"@check_id",
		"@window_end",
		"@window_start_cutoff",
		"cr.result = 'pass'",
		"AVG(CAST(JSON_VALUE(cr.evidence_summary, '$.row_count') AS FLOAT64))",
	} {
		if !strings.Contains(sql, frag) {
			t.Errorf("buildBaselineSQL output missing fragment %q\nfull SQL:\n%s", frag, sql)
		}
	}
}

func TestAllowedDeviation(t *testing.T) {
	cases := []struct {
		name     string
		tol      Tolerance
		baseline float64
		want     float64
	}{
		{"percent 10%% of 1000", Tolerance{Type: TolerancePercent, Value: 10}, 1000, 100},
		{"percent 5%% of negative baseline", Tolerance{Type: TolerancePercent, Value: 5}, -200, 10},
		{"absolute 50", Tolerance{Type: ToleranceAbsolute, Value: 50}, 1000, 50},
		{"stddev placeholder", Tolerance{Type: ToleranceStddev, Value: 2}, 1000, 0},
		{"unknown returns zero", Tolerance{Type: "unknown", Value: 99}, 1000, 0},
	}
	for _, tc := range cases {
		got := allowedDeviation(tc.tol, tc.baseline)
		if got != tc.want {
			t.Errorf("%s: allowedDeviation = %v; want %v", tc.name, got, tc.want)
		}
	}
}

func TestNumericValue(t *testing.T) {
	cases := []struct {
		raw  any
		want float64
		ok   bool
	}{
		{1.5, 1.5, true},
		{int(7), 7.0, true},
		{int64(42), 42.0, true},
		{"3.14", 3.14, true},
		{" 99 ", 99.0, true},
		{"nope", 0, false},
		{nil, 0, false},
		{true, 0, false},
	}
	for _, tc := range cases {
		got, ok := numericValue(tc.raw)
		if ok != tc.ok || got != tc.want {
			t.Errorf("numericValue(%v) = (%v, %v); want (%v, %v)", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}
