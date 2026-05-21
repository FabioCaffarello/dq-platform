// path: engine/internal/orphan/orphan_integration_test.go

//go:build integration

// Integration tests for the orphan-run detector against the local
// Compose substrate. Bring the stack up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/orphan/...
//
// The tests use the same BigQuery emulator the result-write layer
// uses; each test creates its own dataset so parallel runs do not
// collide.

package orphan

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
	return fmt.Sprintf("itest_orphan_%d", time.Now().UnixNano())
}

func makeStore(t *testing.T, cli *bigquery.Client) (*results.BigQueryStore, string) {
	t.Helper()
	ds := uniqueDatasetID(t)
	store := results.NewBigQueryStore(cli, integrationProjectID, ds, nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return store, ds
}

// runningRow returns a `running` ExecutionRow with the given
// started_at. The detector treats started_at as the orphan
// criterion.
func runningRow(executionID, attemptID string, startedAt time.Time, abandonedEngine string) results.ExecutionRow {
	return results.ExecutionRow{
		ExecutionID:    executionID,
		AttemptID:      attemptID,
		RecordedAt:     startedAt,
		Status:         results.StatusRunning,
		EngineVersion:  abandonedEngine,
		RulesetVersion: "rules-v1.0.0",
		Entity:         "customer",
		TriggerSource:  results.TriggerScheduler,
		StartedAt:      &startedAt,
	}
}

// waitForFinalized polls the canonical view until the orphan
// detector's follow-up row is visible (streaming-insert delay)
// or the deadline expires. Returns the latest row.
func waitForFinalized(t *testing.T, store *results.BigQueryStore, executionID string, wantStatus results.ExecutionStatus) *results.ExecutionRow {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(5 * time.Second)
	var last *results.ExecutionRow
	for time.Now().Before(deadline) {
		row, err := store.QueryCurrentExecution(ctx, executionID)
		if err == nil && row.Status == wantStatus {
			return row
		}
		last = row
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to reach status %s; last seen: %+v", executionID, wantStatus, last)
	return nil
}

func TestIntegration_OrphanFinalization(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store, _ := makeStore(t, cli)
	ctx := context.Background()

	// Seed a running row whose started_at is well in the past.
	startedLongAgo := time.Now().Add(-2 * time.Hour).UTC()
	candidate := runningRow("exec-orphan-int-1", "att-1", startedLongAgo, "0.0.9")
	if err := store.WriteExecutionRow(ctx, candidate); err != nil {
		t.Fatalf("seed running row: %v", err)
	}

	// Give the seed time to become visible to queries.
	time.Sleep(500 * time.Millisecond)

	d, err := New(store, Config{
		EngineVersion: "1.0.0",
		Threshold:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	finalized, errs, err := d.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 1 {
		t.Errorf("finalized = %d; want 1", finalized)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v; want empty", errs)
	}

	got := waitForFinalized(t, store, "exec-orphan-int-1", results.StatusAborted)
	if got.EngineVersion != "1.0.0" {
		t.Errorf("canonical view EngineVersion = %q; want %q (detector's, not abandoned engine's)",
			got.EngineVersion, "1.0.0")
	}
	if got.ErrorSummary == nil || *got.ErrorSummary != AbandonmentSummary {
		t.Errorf("ErrorSummary = %v; want %q", got.ErrorSummary, AbandonmentSummary)
	}
}

func TestIntegration_NotOrphan_BelowThreshold(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store, _ := makeStore(t, cli)
	ctx := context.Background()

	// Started recently — within the threshold.
	startedRecently := time.Now().Add(-2 * time.Minute).UTC()
	candidate := runningRow("exec-fresh-int-1", "att-1", startedRecently, "0.0.9")
	if err := store.WriteExecutionRow(ctx, candidate); err != nil {
		t.Fatalf("seed running row: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	d, err := New(store, Config{
		EngineVersion: "1.0.0",
		Threshold:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	finalized, errs, err := d.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 0 {
		t.Errorf("finalized = %d; want 0 (row is within threshold)", finalized)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v; want empty", errs)
	}

	// Canonical view should still show running.
	row, err := store.QueryCurrentExecution(ctx, "exec-fresh-int-1")
	if err != nil {
		t.Fatalf("QueryCurrentExecution: %v", err)
	}
	if row.Status != results.StatusRunning {
		t.Errorf("Status = %q; want %q (must not be finalized)", row.Status, results.StatusRunning)
	}
}

func TestIntegration_AlreadyAborted_NoDuplicate(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store, _ := makeStore(t, cli)
	ctx := context.Background()

	// Seed running then aborted under the same execution_id.
	startedLongAgo := time.Now().Add(-2 * time.Hour).UTC()
	running := runningRow("exec-already-int-1", "att-1", startedLongAgo, "0.0.9")
	if err := store.WriteExecutionRow(ctx, running); err != nil {
		t.Fatalf("seed running row: %v", err)
	}
	completedAt := time.Now().Add(-time.Hour).UTC()
	summary := "operator-issued abort"
	aborted := running
	aborted.RecordedAt = completedAt
	aborted.Status = results.StatusAborted
	aborted.EngineVersion = "0.0.9" // some prior engine finalized it
	aborted.CompletedAt = &completedAt
	aborted.ErrorSummary = &summary
	if err := store.WriteExecutionRow(ctx, aborted); err != nil {
		t.Fatalf("seed aborted row: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	d, err := New(store, Config{
		EngineVersion: "1.0.0",
		Threshold:     30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	finalized, errs, err := d.RunOnce(ctx)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 0 {
		t.Errorf("finalized = %d; want 0 (already aborted; should be skipped)", finalized)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v; want empty", errs)
	}

	// Canonical view should still report the prior aborted row,
	// not a new orphan-detector follow-up.
	row, err := store.QueryCurrentExecution(ctx, "exec-already-int-1")
	if err != nil {
		t.Fatalf("QueryCurrentExecution: %v", err)
	}
	if row.EngineVersion == "1.0.0" {
		t.Errorf("canonical view EngineVersion = %q; the orphan detector should not have written; want the prior aborted row's %q",
			row.EngineVersion, "0.0.9")
	}
	if row.ErrorSummary == nil || *row.ErrorSummary != "operator-issued abort" {
		t.Errorf("ErrorSummary = %v; want the prior aborted row's summary", row.ErrorSummary)
	}
}
