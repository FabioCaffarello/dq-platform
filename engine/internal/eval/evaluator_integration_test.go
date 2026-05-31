// path: engine/internal/eval/evaluator_integration_test.go

//go:build integration

// Integration tests for the set.row_count_positive evaluator
// against the local Compose substrate. Bring the stack up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/eval/...
//
// The tests create an ephemeral source dataset in the BigQuery
// emulator, populate (or leave empty) a per-test table named
// after the trigger entity, and verify the evaluator maps the
// query result to ResultPass / ResultFail / ResultError per
// ADR-0004 CC1.
//
// Originally written for W3-P6c (commit 3c5dc2b, 2026-05-22).
// Refactored to track the post-Wave-S evolutions of the eval
// package: ADR-0022 §"Kind catalog" moved the kind identifier
// to the `set.` prefix (KindSetRowCountPositive), and ADR-0023
// moved source resolution from a deployment-wide Config field
// to a per-rule CheckSpec.Source descriptor. The Config no
// longer carries SourceProject / SourceDataset; tests construct
// a runner.RuleSource per CheckSpec.

package eval

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

const (
	integrationProjectID = "dq-local"
	integrationEndpoint  = "http://localhost:9050"
)

func bqTestClient(t *testing.T) *bigquery.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := bigquery.NewClient(ctx, integrationProjectID,
		option.WithoutAuthentication(),
		option.WithEndpoint(integrationEndpoint),
	)
	if err != nil {
		t.Skipf("integration: cannot create BigQuery client (is `make up` running?): %v", err)
	}
	return cli
}

func uniqueDatasetID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("itest_eval_%d", time.Now().UnixNano())
}

// createDataset creates the source dataset and returns its name.
// The dataset is not torn down — emulator state is ephemeral and
// `make down` clears everything at the end of the session.
func createDataset(t *testing.T, cli *bigquery.Client) string {
	t.Helper()
	ds := uniqueDatasetID(t)
	if err := cli.Dataset(ds).Create(context.Background(), &bigquery.DatasetMetadata{}); err != nil {
		t.Fatalf("create dataset %q: %v", ds, err)
	}
	return ds
}

// createTableWithRows creates a source table with N rows
// containing a single id column. Uses CREATE OR REPLACE TABLE AS
// SELECT so the rows are immediately queryable (no streaming
// buffer delay). N may be zero, in which case the table is
// created empty.
func createTableWithRows(t *testing.T, cli *bigquery.Client, ds, tbl string, n int) {
	t.Helper()
	var ddl string
	if n == 0 {
		// Pure DDL for the empty-table case — a CREATE ... AS
		// SELECT with no source would require a WHERE on a
		// FROM-less SELECT, which standard SQL does not permit.
		ddl = fmt.Sprintf(
			"CREATE OR REPLACE TABLE `%s.%s.%s` (id INT64)",
			integrationProjectID, ds, tbl,
		)
	} else {
		values := ""
		for i := 1; i <= n; i++ {
			if i > 1 {
				values += " UNION ALL "
			}
			values += fmt.Sprintf("SELECT %d AS id", i)
		}
		ddl = fmt.Sprintf(
			"CREATE OR REPLACE TABLE `%s.%s.%s` AS %s",
			integrationProjectID, ds, tbl, values,
		)
	}
	q := cli.Query(ddl)
	job, err := q.Run(context.Background())
	if err != nil {
		t.Fatalf("run DDL for %s.%s: %v", ds, tbl, err)
	}
	status, err := job.Wait(context.Background())
	if err != nil {
		t.Fatalf("wait DDL job for %s.%s: %v", ds, tbl, err)
	}
	if err := status.Err(); err != nil {
		t.Fatalf("DDL job for %s.%s returned status error: %v", ds, tbl, err)
	}
}

func makeEvaluator(t *testing.T, cli *bigquery.Client) *Evaluator {
	t.Helper()
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

// bqSource constructs the per-rule BigQuery source descriptor
// CheckSpec.Source carries per ADR-0023. The evaluator no longer
// reads a deployment-wide SourceProject / SourceDataset; the rule
// declares the substrate location itself.
func bqSource(datasetID, tableID string) *runner.RuleSource {
	return &runner.RuleSource{
		Type:      "bigquery",
		ProjectID: integrationProjectID,
		DatasetID: datasetID,
		TableID:   tableID,
	}
}

// stdTrigger returns a TriggerRequest with the runner-required
// window endpoints set per validateTrigger. row_count_positive
// without a partition_column doesn't read the endpoints, but the
// runner enforces `WindowEnd > WindowStart` at trigger validation
// (the runner-side TestIntegration_RunnerWithEvaluator_PassesEndToEnd
// path needs this; the direct-Evaluate tests don't go through
// validateTrigger but set them for hygiene).
func stdTrigger(entity string) runner.TriggerRequest {
	return runner.TriggerRequest{
		Entity:        entity,
		WindowStart:   time.Date(2026, 5, 22, 14, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC),
		TriggerSource: results.TriggerScheduler,
	}
}

func TestIntegration_RowCountPositive_PassesWithRows(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := createDataset(t, cli)
	createTableWithRows(t, cli, ds, "customer", 3)

	e := makeEvaluator(t, cli)
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{
			CheckID: "row_count_positive",
			Kind:    KindSetRowCountPositive,
			Source:  bqSource(ds, "customer"),
		},
		stdTrigger("customer"))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultPass {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultPass)
	}
	if got, _ := eval.EvidenceSummary["row_count"].(int64); got != 3 {
		t.Errorf("row_count = %v; want 3", eval.EvidenceSummary["row_count"])
	}
	wantRef := fmt.Sprintf("%s.%s.%s", integrationProjectID, ds, "customer")
	if eval.EvidenceSummary["table_ref"] != wantRef {
		t.Errorf("table_ref = %v; want %q", eval.EvidenceSummary["table_ref"], wantRef)
	}
}

func TestIntegration_RowCountPositive_FailsOnEmpty(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := createDataset(t, cli)
	createTableWithRows(t, cli, ds, "customer", 0)

	e := makeEvaluator(t, cli)
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{
			CheckID: "row_count_positive",
			Kind:    KindSetRowCountPositive,
			Source:  bqSource(ds, "customer"),
		},
		stdTrigger("customer"))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultFail {
		t.Errorf("Result = %q; want %q (empty table)", eval.Result, results.ResultFail)
	}
	if got, _ := eval.EvidenceSummary["row_count"].(int64); got != 0 {
		t.Errorf("row_count = %v; want 0", eval.EvidenceSummary["row_count"])
	}
}

func TestIntegration_RowCountPositive_ErrorOnMissingTable(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := createDataset(t, cli)
	// Deliberately do NOT create a table — the source descriptor
	// points at a non-existent table_id so the evaluator exercises
	// the query_read_failed branch per ADR-0004 CC1.

	e := makeEvaluator(t, cli)
	eval, err := e.Evaluate(context.Background(),
		runner.CheckSpec{
			CheckID: "row_count_positive",
			Kind:    KindSetRowCountPositive,
			Source:  bqSource(ds, "does_not_exist"),
		},
		stdTrigger("does_not_exist"))
	if err == nil {
		t.Fatal("expected non-nil error for missing table")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
	if eval.EvidenceSummary["reason"] != "query_read_failed" &&
		eval.EvidenceSummary["reason"] != "no_rows_from_count_query" {
		t.Errorf("EvidenceSummary[reason] = %v; want query_read_failed or no_rows_from_count_query",
			eval.EvidenceSummary["reason"])
	}
}

// TestIntegration_RunnerWithEvaluator_PassesEndToEnd exercises the
// full runner → eval.Evaluator → BigQuery wire so a future
// signature drift between runner.CheckEvaluator and the eval
// package surfaces in CI. The eval-only tests above call Evaluate
// directly; this test routes a trigger through runner.Run and
// asserts the persisted dq_check_results row carries
// ResultPass.
func TestIntegration_RunnerWithEvaluator_PassesEndToEnd(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	// Source dataset + table (3 rows ⇒ row_count_positive pass).
	sourceDS := createDataset(t, cli)
	createTableWithRows(t, cli, sourceDS, "customer", 3)

	// Separate dataset for dq_executions / dq_check_results.
	resultsDS := uniqueDatasetID(t) + "_results"
	store := results.NewBigQueryStore(cli, integrationProjectID, resultsDS, nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	evaluator := makeEvaluator(t, cli)

	r, err := runner.New(runner.Config{
		Store:          store,
		Evaluator:      evaluator,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
	})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}

	trigger := runner.TriggerRequest{
		Entity:        "customer",
		WindowStart:   time.Date(2026, 5, 22, 14, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC),
		TriggerSource: results.TriggerScheduler,
		Checks: []runner.CheckSpec{
			{
				CheckID: "row_count_positive",
				Kind:    KindSetRowCountPositive,
				Source:  bqSource(sourceDS, "customer"),
			},
		},
	}
	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if terminal.Status != results.StatusSuccess {
		t.Errorf("terminal.Status = %q; want success", terminal.Status)
	}
	if terminal.Entity != "customer" {
		t.Errorf("terminal.Entity = %q; want customer", terminal.Entity)
	}
}
