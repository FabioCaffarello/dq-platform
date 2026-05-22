// path: engine/internal/eval/evaluator_test.go

package eval

import (
	"context"
	"errors"
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

func TestNew_SourceProjectDefaultsToClientProject(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if e.sourceProject != "stub-project" {
		t.Errorf("sourceProject = %q; want client default %q", e.sourceProject, "stub-project")
	}
}

func TestEvaluate_UnsupportedKind_ReturnsResultError(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli, SourceDataset: "ds"})
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

func TestEvaluate_RowCountPositive_MissingSourceDataset_ReturnsResultError(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli}) // SourceDataset deliberately empty
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: KindRowCountPositive},
		runner.TriggerRequest{Entity: "customer"})
	if !errors.Is(err, errSourceDatasetMissing) {
		t.Errorf("err = %v; want errSourceDatasetMissing", err)
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
	if eval.EvidenceSummary["reason"] != "source_dataset_not_configured" {
		t.Errorf("EvidenceSummary[reason] = %v; want source_dataset_not_configured", eval.EvidenceSummary["reason"])
	}
	if eval.EvidenceSummary["kind"] != KindRowCountPositive {
		t.Errorf("EvidenceSummary[kind] = %v; want %q", eval.EvidenceSummary["kind"], KindRowCountPositive)
	}
}

func TestEvaluator_SatisfiesRunnerInterface(t *testing.T) {
	// The compile-time assertion in row_count_positive.go already
	// enforces this; the test exists so a future refactor that
	// accidentally narrows the assertion still produces a visible
	// failure during test runs.
	var _ runner.CheckEvaluator = (*Evaluator)(nil)
}

func TestEvaluate_UnsupportedKindIncludesExpectedKindInError(t *testing.T) {
	cli := stubClient(t)
	e, err := New(Config{Client: cli, SourceDataset: "ds"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = e.Evaluate(context.Background(),
		runner.CheckSpec{CheckID: "c1", Kind: "freshness"},
		runner.TriggerRequest{Entity: "customer"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), KindRowCountPositive) {
		t.Errorf("error message should name the supported kind for forward compatibility; got %q", err.Error())
	}
}
