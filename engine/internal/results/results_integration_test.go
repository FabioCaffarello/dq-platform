// path: engine/internal/results/results_integration_test.go

//go:build integration

// Integration tests for the result-write layer against the local
// Compose substrate from Wave 3 Phase 2. Bring the stack up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/results/...
//
// The tests use the cloud.google.com/go/bigquery client against
// the bigquery-emulator on localhost:9050 via an explicit endpoint
// override (the SDK does not honor an environment variable for
// custom endpoints).

package results

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"
)

const (
	integrationProjectID = "dq-local"
	integrationEndpoint  = "http://localhost:9050"
)

// bqTestClient returns a BigQuery client pointed at the local
// emulator. Skips the test if the emulator is unreachable.
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

// uniqueDatasetID returns a per-test dataset name so parallel
// integration test runs don't collide.
func uniqueDatasetID(t *testing.T) string {
	t.Helper()
	// Replace characters BigQuery does not accept in dataset IDs.
	name := fmt.Sprintf("itest_%d", time.Now().UnixNano())
	return name
}

func TestIntegration_EnsureSchemaIdempotent(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := uniqueDatasetID(t)
	store := NewBigQueryStore(cli, integrationProjectID, ds, nil)

	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("first EnsureSchema: %v", err)
	}
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("second EnsureSchema (should be a no-op): %v", err)
	}

	// Verify both tables exist.
	for _, name := range []string{tableExecutions, tableCheckResults} {
		_, err := cli.DatasetInProject(integrationProjectID, ds).Table(name).Metadata(ctx)
		if err != nil {
			t.Errorf("table %s missing after EnsureSchema: %v", name, err)
		}
	}
}

func TestIntegration_WriteExecutionRow_RoundTrip(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := uniqueDatasetID(t)
	store := NewBigQueryStore(cli, integrationProjectID, ds, nil)
	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	now := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	row := ExecutionRow{
		ExecutionID:    "exec-roundtrip-1",
		AttemptID:      "att-1",
		RecordedAt:     now,
		Status:         StatusRunning,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		Entity:         "customer",
		TriggerSource:  TriggerScheduler,
	}
	if err := store.WriteExecutionRow(ctx, row); err != nil {
		t.Fatalf("WriteExecutionRow: %v", err)
	}

	// Streaming inserts on the emulator may take a moment to be
	// queryable; small retry loop.
	var got *ExecutionRow
	var lastErr error
	for i := 0; i < 10; i++ {
		got, lastErr = store.QueryCurrentExecution(ctx, "exec-roundtrip-1")
		if lastErr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("QueryCurrentExecution after insert: %v", lastErr)
	}

	if got.ExecutionID != row.ExecutionID {
		t.Errorf("ExecutionID = %q; want %q", got.ExecutionID, row.ExecutionID)
	}
	if got.Status != row.Status {
		t.Errorf("Status = %q; want %q", got.Status, row.Status)
	}
	if got.RulesetVersion != row.RulesetVersion {
		t.Errorf("RulesetVersion = %q; want %q", got.RulesetVersion, row.RulesetVersion)
	}
}

func TestIntegration_AppendOnly_TwoAttempts(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := uniqueDatasetID(t)
	store := NewBigQueryStore(cli, integrationProjectID, ds, nil)
	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	t0 := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Minute)

	rowA := ExecutionRow{
		ExecutionID:    "exec-twin",
		AttemptID:      "att-1",
		RecordedAt:     t0,
		Status:         StatusRunning,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		Entity:         "customer",
		TriggerSource:  TriggerScheduler,
	}
	rowB := rowA
	rowB.AttemptID = "att-2"
	rowB.RecordedAt = t1
	rowB.Status = StatusSuccess

	if err := store.WriteExecutionRow(ctx, rowA); err != nil {
		t.Fatalf("WriteExecutionRow(A): %v", err)
	}
	if err := store.WriteExecutionRow(ctx, rowB); err != nil {
		t.Fatalf("WriteExecutionRow(B): %v", err)
	}

	// QueryCurrentExecution should return the latest by recorded_at.
	var got *ExecutionRow
	var lastErr error
	for i := 0; i < 10; i++ {
		got, lastErr = store.QueryCurrentExecution(ctx, "exec-twin")
		if lastErr == nil && got != nil && got.AttemptID == "att-2" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("QueryCurrentExecution: %v", lastErr)
	}
	if got.AttemptID != "att-2" {
		t.Errorf("expected latest attempt att-2; got %q (recorded_at=%v)", got.AttemptID, got.RecordedAt)
	}
	if got.Status != StatusSuccess {
		t.Errorf("expected latest status %q; got %q", StatusSuccess, got.Status)
	}
}

func TestIntegration_WriteCheckResultRow(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := uniqueDatasetID(t)
	store := NewBigQueryStore(cli, integrationProjectID, ds, nil)
	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	row := CheckResultRow{
		ExecutionID:   "exec-check-1",
		AttemptID:     "att-1",
		CheckID:       "row_count_positive",
		Result:        ResultFail,
		ExecutedAt:    time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		EngineVersion: "0.1.0",
		EvidenceSummary: map[string]any{
			"rows_scanned": 1000,
			"rows_failing": 12,
		},
		SampleViolatingRows: []map[string]any{
			{"id": 1, "reason": "null"},
		},
	}
	if err := store.WriteCheckResultRow(ctx, row); err != nil {
		t.Fatalf("WriteCheckResultRow: %v", err)
	}
	// The Reader API in W3-P4b does not expose check-result
	// queries (single-execution lookups only); the round-trip is
	// implicit via the write succeeding. The integration test
	// verifies that the composite key + JSON fields are accepted.
}

func TestIntegration_QueryNotFound(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := uniqueDatasetID(t)
	store := NewBigQueryStore(cli, integrationProjectID, ds, nil)
	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	_, err := store.QueryCurrentExecution(ctx, "no-such-execution")
	if !errors.Is(err, ErrExecutionNotFound) {
		t.Fatalf("QueryCurrentExecution(no-such): err = %v; want ErrExecutionNotFound", err)
	}
}

func TestIntegration_SchemaInitCreatesView_BestEffort(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	ds := uniqueDatasetID(t)
	store := NewBigQueryStore(cli, integrationProjectID, ds, nil)
	ctx := context.Background()
	if err := store.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// Best-effort check: if the emulator created the view, query
	// it; if not, log and pass. Honors the ADR-0010 lazy-view
	// Partial commitment.
	viewMeta, err := cli.DatasetInProject(integrationProjectID, ds).Table(viewExecutionsView).Metadata(ctx)
	if err != nil {
		t.Logf("dq_executions_current view not created (emulator fidelity gap; ADR-0010 lazy-view Partial row): %v", err)
		return
	}
	if viewMeta.Type != bigquery.ViewTable {
		t.Errorf("dq_executions_current exists but is type %q; want View", viewMeta.Type)
	}
	t.Logf("dq_executions_current view present on emulator; fidelity gap may still apply at query time")
}
