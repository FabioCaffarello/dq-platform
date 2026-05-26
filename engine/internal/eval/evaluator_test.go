// path: engine/internal/eval/evaluator_test.go

package eval

import (
	"context"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// stubClient returns a minimal *bigquery.Client suitable for unit
// tests that never reach the query-execution path. Constructed
// without authentication and pointed at a fake endpoint so an
// accidental query would fail loud rather than reaching real
// BigQuery, and so test runs don't require ADC credentials. The
// client is closed via t.Cleanup so each subtest leaves the test
// fixture clean.
func stubClient(t *testing.T) *bigquery.Client {
	t.Helper()
	cli, err := bigquery.NewClient(context.Background(), "stub-project",
		option.WithoutAuthentication(),
		option.WithEndpoint("http://127.0.0.1:0"),
	)
	if err != nil {
		t.Fatalf("stub client: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func TestNew_RequiresClient(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("New accepted nil Client")
	}
}

func TestNew_RegistersInaugurralKinds(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	kinds := e.RegisteredKinds()
	want := map[string]bool{
		KindSetRowCountPositive:       true,
		KindSetRowCountWithinBaseline: true, // B2-14 first baselined kind
		KindRecordSchemaConformance:   true,
	}
	for _, k := range kinds {
		if !want[k] {
			t.Errorf("unexpected kind %q registered", k)
		}
		delete(want, k)
	}
	if len(want) != 0 {
		t.Errorf("kinds missing from registry: %v", want)
	}
}

func TestEvaluate_UnsupportedKind_ReturnsResultError(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: "no_such_kind"},
		runner.TriggerRequest{Entity: "customer"})
	if err == nil {
		t.Errorf("expected non-nil error for unsupported kind")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
	if eval.EvidenceSummary["reason"] != "unsupported_kind" {
		t.Errorf("EvidenceSummary[reason] = %v; want unsupported_kind", eval.EvidenceSummary["reason"])
	}
	if eval.EvidenceSummary["kind"] != "no_such_kind" {
		t.Errorf("EvidenceSummary[kind] = %v; want %q", eval.EvidenceSummary["kind"], "no_such_kind")
	}
}

func TestEvaluate_SetRowCountPositive_MissingSource_ReturnsResultError(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: KindSetRowCountPositive},
		runner.TriggerRequest{Entity: "customer"})
	if err == nil {
		t.Errorf("expected non-nil error when source is missing")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
	if eval.EvidenceSummary["reason"] != "missing_or_non_bigquery_source" {
		t.Errorf("EvidenceSummary[reason] = %v; want missing_or_non_bigquery_source", eval.EvidenceSummary["reason"])
	}
}

func TestEvaluate_RecordSchemaConformance_RequiresParams(t *testing.T) {
	// The real β handler rejects missing params with a clear
	// invalid_params diagnostic. The full happy-path coverage
	// for record.schema_conformance lives in
	// record_schema_conformance_test.go.
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: KindRecordSchemaConformance},
		runner.TriggerRequest{Entity: "orders_stream"})
	if err == nil {
		t.Errorf("expected handler to reject empty params")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
	reason, _ := eval.EvidenceSummary["reason"].(string)
	if reason != "invalid_params" {
		t.Errorf("EvidenceSummary[reason] = %q; want invalid_params", reason)
	}
}

func TestEvaluator_SatisfiesRunnerInterface(t *testing.T) {
	// The compile-time assertion in row_count_positive.go already
	// enforces this; the test exists so a future refactor that
	// accidentally narrows the assertion still produces a visible
	// failure during test runs.
	var _ runner.CheckEvaluator = (*Evaluator)(nil)
}

func TestEvaluate_UnsupportedKindNamesRegisteredKinds(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: "freshness"},
		runner.TriggerRequest{Entity: "customer"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), KindSetRowCountPositive) {
		t.Errorf("error should name the registered kinds for forward compatibility; got %q", err.Error())
	}
}

func TestRegister_OverridesExistingHandler(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	called := false
	e.Register(KindSetRowCountPositive, func(ctx context.Context, _ *Evaluator, _ runner.CheckSpec, _ runner.TriggerRequest) (runner.Evaluation, error) {
		called = true
		return runner.Evaluation{Result: results.ResultPass}, nil
	})
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: KindSetRowCountPositive},
		runner.TriggerRequest{Entity: "customer"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !called {
		t.Error("Register did not install the new handler")
	}
	if eval.Result != results.ResultPass {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultPass)
	}
}
