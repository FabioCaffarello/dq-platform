// path: engine/internal/eval/null_rate_test.go

package eval

import (
	"context"
	"strings"
	"testing"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

func TestNullRateSQL_NoPartition(t *testing.T) {
	got := nullRateSQL("dq-local.dq_fixture.customer", "email", "")
	want := "SELECT COUNTIF(`email` IS NULL) AS null_count, COUNT(*) AS total FROM `dq-local.dq_fixture.customer`"
	if got != want {
		t.Errorf("SQL = %q; want %q", got, want)
	}
}

func TestNullRateSQL_WithPartition(t *testing.T) {
	got := nullRateSQL("dq-local.dq_fixture.events", "user_id", "event_ts")
	want := "SELECT COUNTIF(`user_id` IS NULL) AS null_count, COUNT(*) AS total FROM `dq-local.dq_fixture.events` " +
		"WHERE `event_ts` >= @window_start AND `event_ts` < @window_end"
	if got != want {
		t.Errorf("partitioned SQL = %q; want %q", got, want)
	}
	if !strings.Contains(got, "@window_start") || !strings.Contains(got, "@window_end") {
		t.Error("partition SQL must reference both window-endpoint params")
	}
	if !strings.Contains(got, "`user_id`") {
		t.Error("target column must appear backtick-quoted in COUNTIF")
	}
	if !strings.Contains(got, "`event_ts`") {
		t.Error("partition column must appear backtick-quoted in WHERE")
	}
}

func TestParseNullRateParams(t *testing.T) {
	cases := []struct {
		name        string
		params      map[string]any
		wantCol     string
		wantRate    float64
		wantErrSubs string // empty = expect nil error
	}{
		{
			name:     "valid-float",
			params:   map[string]any{"column": "email", "max_null_rate": 0.05},
			wantCol:  "email",
			wantRate: 0.05,
		},
		{
			name:     "valid-int-coerced",
			params:   map[string]any{"column": "user_id", "max_null_rate": 0},
			wantCol:  "user_id",
			wantRate: 0.0,
		},
		{
			name:        "nil-params-rejected",
			params:      nil,
			wantErrSubs: "params is required",
		},
		{
			name:        "missing-column-rejected",
			params:      map[string]any{"max_null_rate": 0.1},
			wantErrSubs: "params.column is required",
		},
		{
			name:        "missing-rate-rejected",
			params:      map[string]any{"column": "email"},
			wantErrSubs: "params.max_null_rate is required",
		},
		{
			name:        "column-bad-shape-rejected",
			params:      map[string]any{"column": "no-hyphens", "max_null_rate": 0.1},
			wantErrSubs: "params.column",
		},
		{
			name:        "rate-out-of-range-rejected",
			params:      map[string]any{"column": "email", "max_null_rate": 1.5},
			wantErrSubs: "out of range",
		},
		{
			name:        "rate-wrong-type-rejected",
			params:      map[string]any{"column": "email", "max_null_rate": "0.1"},
			wantErrSubs: "must be a number",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			col, rate, err := parseNullRateParams(tc.params)
			if tc.wantErrSubs == "" {
				if err != nil {
					t.Fatalf("got err %v; want nil", err)
				}
				if col != tc.wantCol {
					t.Errorf("column = %q; want %q", col, tc.wantCol)
				}
				if rate != tc.wantRate {
					t.Errorf("rate = %v; want %v", rate, tc.wantRate)
				}
				return
			}
			if err == nil {
				t.Fatalf("got nil err; want error containing %q", tc.wantErrSubs)
			}
			if !strings.Contains(err.Error(), tc.wantErrSubs) {
				t.Errorf("err %q should contain %q", err.Error(), tc.wantErrSubs)
			}
		})
	}
}

func TestDecideNullRate_TableDriven(t *testing.T) {
	// Threshold semantics extracted to decideNullRate so the
	// pass-below / fail-above branches are unit-testable without a
	// BigQuery client. The handler-level "error in query" and
	// "source missing / non-bigquery" branches are covered by the
	// TestEvaluate_SetNullRate_* cases below.
	cases := []struct {
		name        string
		nullCount   int64
		total       int64
		maxRate     float64
		wantResult  results.CheckResult
		wantRate    float64
		wantVacuous bool
	}{
		{
			name:       "pass-below-threshold",
			nullCount:  5,
			total:      1000,
			maxRate:    0.05,
			wantResult: results.ResultPass,
			wantRate:   0.005,
		},
		{
			name:       "pass-exact-threshold-boundary",
			nullCount:  50,
			total:      1000,
			maxRate:    0.05,
			wantResult: results.ResultPass,
			wantRate:   0.05,
		},
		{
			name:       "fail-above-threshold",
			nullCount:  100,
			total:      1000,
			maxRate:    0.05,
			wantResult: results.ResultFail,
			wantRate:   0.1,
		},
		{
			name:       "fail-all-null",
			nullCount:  42,
			total:      42,
			maxRate:    0.5,
			wantResult: results.ResultFail,
			wantRate:   1.0,
		},
		{
			name:        "vacuous-empty-window",
			nullCount:   0,
			total:       0,
			maxRate:     0.05,
			wantResult:  results.ResultPass,
			wantRate:    0.0,
			wantVacuous: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, rate, vacuous := decideNullRate(tc.nullCount, tc.total, tc.maxRate)
			if got != tc.wantResult {
				t.Errorf("Result = %q; want %q", got, tc.wantResult)
			}
			if rate != tc.wantRate {
				t.Errorf("rate = %v; want %v", rate, tc.wantRate)
			}
			if vacuous != tc.wantVacuous {
				t.Errorf("vacuous = %v; want %v", vacuous, tc.wantVacuous)
			}
		})
	}
}

func TestEvaluate_SetNullRate_SourceCases(t *testing.T) {
	// Handler-level table covering the two non-threshold paths the
	// user-requested test matrix calls out: "source ausente /
	// não-bigquery" (here) and "error em query" (next test). The
	// pass/fail branches are exercised exhaustively by
	// TestDecideNullRate_TableDriven via the same decision helper
	// the handler calls.
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	validParams := map[string]any{"column": "email", "max_null_rate": 0.05}

	cases := []struct {
		name       string
		spec       runner.CheckSpec
		wantReason string
	}{
		{
			name: "missing-source",
			spec: runner.CheckSpec{
				CheckID: "c1",
				Kind:    KindSetNullRate,
				Params:  validParams,
				Source:  nil,
			},
			wantReason: "missing_or_non_bigquery_source",
		},
		{
			name: "non-bigquery-source",
			spec: runner.CheckSpec{
				CheckID: "c1",
				Kind:    KindSetNullRate,
				Params:  validParams,
				Source:  &runner.RuleSource{Type: "kafka", Topic: "orders"},
			},
			wantReason: "missing_or_non_bigquery_source",
		},
		{
			name: "missing-params",
			spec: runner.CheckSpec{
				CheckID: "c1",
				Kind:    KindSetNullRate,
				Params:  nil,
				Source:  &runner.RuleSource{Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t"},
			},
			wantReason: "invalid_params",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			eval, err := e.Evaluate(context.Background(), tc.spec, runner.TriggerRequest{Entity: "customer"})
			if err == nil {
				t.Fatalf("expected non-nil error for %s", tc.name)
			}
			if eval.Result != results.ResultError {
				t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
			}
			if got := eval.EvidenceSummary["reason"]; got != tc.wantReason {
				t.Errorf("reason = %v; want %q", got, tc.wantReason)
			}
		})
	}
}

func TestEvaluate_SetNullRate_QueryError(t *testing.T) {
	// The stub client is constructed with a fake endpoint, so any
	// Read attempt fails — this is the deterministic "error em
	// query" path. The handler must surface ResultError with
	// reason=query_read_failed per ADR-0004 CC1.
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{
			CheckID: "c1",
			Kind:    KindSetNullRate,
			Params:  map[string]any{"column": "email", "max_null_rate": 0.05},
			Source: &runner.RuleSource{
				Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t",
			},
		},
		runner.TriggerRequest{Entity: "customer"})
	if err == nil {
		t.Fatal("expected non-nil error from query path; stub endpoint should fail Read")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
	reason, _ := eval.EvidenceSummary["reason"].(string)
	if reason != "query_read_failed" {
		t.Errorf("reason = %q; want query_read_failed", reason)
	}
	if eval.EvidenceSummary["column"] != "email" {
		t.Errorf("evidence.column = %v; want email", eval.EvidenceSummary["column"])
	}
}
