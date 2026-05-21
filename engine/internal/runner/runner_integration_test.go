// path: engine/internal/runner/runner_integration_test.go

//go:build integration

// Integration tests for the runner against the local Compose
// substrate. Bring the stack up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/runner/...
//
// Wires a real BigQuery store (result-write) with stub
// precheck + evaluator so the runner's flow is exercised
// end-to-end without depending on Phase-4+ check kinds.

package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/results"
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
	return fmt.Sprintf("itest_runner_%d", time.Now().UnixNano())
}

func makeStore(t *testing.T, cli *bigquery.Client) *results.BigQueryStore {
	t.Helper()
	ds := uniqueDatasetID(t)
	store := results.NewBigQueryStore(cli, integrationProjectID, ds, nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return store
}

func makeRunner(t *testing.T, store results.Store, evaluator CheckEvaluator) *Runner {
	t.Helper()
	r, err := New(Config{
		Store:          store,
		Evaluator:      evaluator,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return r
}

func intTrigger(checkIDs ...string) TriggerRequest {
	specs := make([]CheckSpec, 0, len(checkIDs))
	for _, id := range checkIDs {
		specs = append(specs, CheckSpec{CheckID: id, Kind: "stub"})
	}
	return TriggerRequest{
		Entity:        "customer",
		WindowStart:   time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		WindowEnd:     time.Date(2026, 5, 21, 15, 0, 0, 0, time.UTC),
		TriggerSource: results.TriggerScheduler,
		Checks:        specs,
	}
}

// waitForStatus polls QueryCurrentExecution until the canonical
// view returns the expected status (streaming-insert delay) or
// the deadline expires.
func waitForStatus(t *testing.T, store *results.BigQueryStore, executionID string, want results.ExecutionStatus) *results.ExecutionRow {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(5 * time.Second)
	var last *results.ExecutionRow
	for time.Now().Before(deadline) {
		row, err := store.QueryCurrentExecution(ctx, executionID)
		if err == nil && row.Status == want {
			return row
		}
		last = row
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to reach %s; last seen: %+v", executionID, want, last)
	return nil
}

func TestIntegration_RunHappyPath(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store := makeStore(t, cli)
	r := makeRunner(t, store, NoopEvaluator{})

	trigger := intTrigger("row_count_positive")
	terminal, err := r.Run(context.Background(), trigger)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusSuccess {
		t.Errorf("terminal.Status = %q; want success", terminal.Status)
	}

	got := waitForStatus(t, store, terminal.ExecutionID, results.StatusSuccess)
	if got.Entity != "customer" {
		t.Errorf("canonical view Entity = %q; want customer", got.Entity)
	}
}

func TestIntegration_RunErrorPrecheck(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store := makeStore(t, cli)
	r, err := New(Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		Precheck:       fixedPrecheck{present: false},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	terminal, err := r.Run(context.Background(), intTrigger("c1"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusError {
		t.Errorf("terminal.Status = %q; want error", terminal.Status)
	}
	if terminal.ErrorSummary == nil {
		t.Errorf("ErrorSummary is nil; want the pre-check absence summary")
	}

	got := waitForStatus(t, store, terminal.ExecutionID, results.StatusError)
	if got.ErrorSummary == nil {
		t.Errorf("canonical view ErrorSummary is nil; want set")
	}
}

func TestIntegration_RunMixedResults(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store := makeStore(t, cli)
	evaluator := PerCheckEvaluator{
		Results: map[string]results.CheckResult{
			"a": results.ResultPass,
			"b": results.ResultFail,
			"c": results.ResultPass,
		},
	}
	r := makeRunner(t, store, evaluator)

	terminal, err := r.Run(context.Background(), intTrigger("a", "b", "c"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if terminal.Status != results.StatusFailed {
		t.Errorf("terminal.Status = %q; want failed", terminal.Status)
	}

	waitForStatus(t, store, terminal.ExecutionID, results.StatusFailed)
}
