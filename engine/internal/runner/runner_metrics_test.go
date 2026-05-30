// path: engine/internal/runner/runner_metrics_test.go

package runner

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"dq-platform/engine/internal/metrics"
	"dq-platform/engine/internal/results"
)

// newRunnerWithMetrics constructs a Runner wired to a real
// RunnerMetrics handle set so per-metric increments can be
// asserted via prometheus/testutil. Mirrors newTestRunner above
// but injects Metrics; deterministic Now so dq_run_duration_seconds
// observes a stable value.
func newRunnerWithMetrics(t *testing.T, store results.Store, evaluator CheckEvaluator, m metrics.RunnerMetrics) *Runner {
	t.Helper()
	tick := 0
	r, err := New(Config{
		Store:          store,
		Evaluator:      evaluator,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		AttemptID:      func() string { return "att-metrics" },
		Now: func() time.Time {
			// Each Now() call advances by one second, so the
			// per-check loop's started→finished delta is exactly
			// 1s and the run-level startedAt→completedAt is the
			// sum of in-flight call counts.
			tick++
			return time.Date(2026, 5, 30, 12, 0, tick, 0, time.UTC)
		},
		Metrics: m,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func TestRun_EmitsRunsTotal_OnHappyPath(t *testing.T) {
	store := &inMemStore{}
	r := metrics.New()
	rn := newRunnerWithMetrics(t, store, NoopEvaluator{}, r.Runner)

	if _, err := rn.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// ADR-0055 §Clause 4: one terminal-row write → one increment
	// with status=success and mode=set (the sample trigger leaves
	// CheckSpec.Mode empty, so triggerMode defaults to "set" per
	// the runner's existing backfill).
	got := testutil.ToFloat64(r.Runner.RunsTotal.WithLabelValues("customer", "success", "scheduler", "set"))
	if got != 1 {
		t.Errorf("dq_runs_total{entity=customer,status=success,trigger_source=scheduler,mode=set} = %v; want 1", got)
	}
	// dq_checks_evaluated_total: one check in sampleTrigger, result=pass.
	gotChecks := testutil.ToFloat64(r.Runner.ChecksEvaluatedTotal.WithLabelValues("customer", "row_count_positive", "pass", "set"))
	if gotChecks != 1 {
		t.Errorf("dq_checks_evaluated_total = %v; want 1", gotChecks)
	}
}

func TestRun_EmitsRunsTotal_OnPreCheckErrorPath(t *testing.T) {
	store := &inMemStore{}
	r := metrics.New()
	rn := newRunnerWithMetrics(t, store, NoopEvaluator{}, r.Runner)
	rn.precheck = absentPrecheck{}

	if _, err := rn.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Pre-check error path writes a terminal `error` row directly.
	got := testutil.ToFloat64(r.Runner.RunsTotal.WithLabelValues("customer", "error", "scheduler", "set"))
	if got != 1 {
		t.Errorf("dq_runs_total{status=error,...} = %v; want 1 (pre-check error path)", got)
	}
	// No per-check loop ran, so checks_evaluated_total stays at zero.
	gotChecks := testutil.ToFloat64(r.Runner.ChecksEvaluatedTotal.WithLabelValues("customer", "row_count_positive", "pass", "set"))
	if gotChecks != 0 {
		t.Errorf("dq_checks_evaluated_total on pre-check path = %v; want 0", gotChecks)
	}
}

func TestRun_EmitsBytesScannedFromEvidence(t *testing.T) {
	store := &inMemStore{}
	r := metrics.New()
	rn := newRunnerWithMetrics(t, store, evidenceEvaluator{bytesScanned: int64(2048)}, r.Runner)
	if _, err := rn.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := testutil.ToFloat64(r.Runner.BytesScanned.WithLabelValues("customer", "row_count_positive"))
	if got != 2048 {
		t.Errorf("dq_bytes_scanned = %v; want 2048", got)
	}
}

func TestRun_BytesScannedZero_WhenSubFieldAbsent(t *testing.T) {
	// ADR-0055 §Clause 4 OQ-6 resolution: zero, not NaN, not skip.
	store := &inMemStore{}
	r := metrics.New()
	rn := newRunnerWithMetrics(t, store, NoopEvaluator{}, r.Runner)
	if _, err := rn.Run(context.Background(), sampleTrigger()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	got := testutil.ToFloat64(r.Runner.BytesScanned.WithLabelValues("customer", "row_count_positive"))
	if got != 0 {
		t.Errorf("dq_bytes_scanned with absent sub-field = %v; want 0", got)
	}
}

func TestBytesScannedOrZero_NumericTypes(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want float64
	}{
		{"absent", nil, 0},
		{"float64", float64(1024), 1024},
		{"int", int(2048), 2048},
		{"int64", int64(4096), 4096},
		{"uint64", uint64(8192), 8192},
		{"non-numeric string", "1024", 0},
		{"nil-typed map value", any(nil), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := map[string]any{}
			if tc.in != nil || tc.name == "nil-typed map value" {
				m["bytes_scanned"] = tc.in
			}
			got := bytesScannedOrZero(m)
			if got != tc.want {
				t.Errorf("bytesScannedOrZero(%v) = %v; want %v", tc.in, got, tc.want)
			}
		})
	}
}

// absentPrecheck always reports the source is absent — drives the
// pre-check error path in writePreCheckErrorRow.
type absentPrecheck struct{}

func (absentPrecheck) SourceExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// evidenceEvaluator is a test double that returns ResultPass with
// a configurable bytes_scanned value in the evidence summary so
// the dq_bytes_scanned gauge test can assert the read-out.
type evidenceEvaluator struct {
	bytesScanned int64
}

func (e evidenceEvaluator) Evaluate(_ context.Context, _ CheckSpec, _ TriggerRequest) (Evaluation, error) {
	return Evaluation{
		Result:          results.ResultPass,
		EvidenceSummary: map[string]any{"bytes_scanned": e.bytesScanned},
	}, nil
}
