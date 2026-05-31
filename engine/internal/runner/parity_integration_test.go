// path: engine/internal/runner/parity_integration_test.go

//go:build integration

// Cross-mode table-shape parity integration test. Asserts the
// consumer-facing unified-reporting invariant ADR-0041
// §"Unified-reporting invariant" commits: both runners write to
// the same tables under the same identifier scheme, with the
// per-mode distinction carried only by the `mode` column and
// `window_start` / `window_end` populated for both.
//
// Reads:
//   - ADR-0041 §"Unified-reporting invariant" — the table-level
//     promise this test guards.
//   - ADR-0002 §3 (CC1, CC7) — the execution_id formula both modes
//     share; deterministic SHA256 over (ruleset_version, entity,
//     window_start, window_end, trigger_source).
//   - ADR-0021 — the closed {set, record} mode enum carried on the
//     mode column.
//   - ADR-0003 CC3 — the required-column inventory for
//     dq_executions; this test asserts every Required=true column
//     is populated by both runners.
//   - ADR-0025 §"Result-write schema extension" — the mode column
//     itself.
//   - ADR-0041 §"Window-endpoint columns (design-only)" + B2-27
//     implementation — window_start / window_end as additive
//     Required columns populated by both runners.
//
// Bring the substrate up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/runner/...

package runner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"dq-platform/engine/internal/results"
)

// TestIntegration_CrossModeTableShapeParity drives one set-mode
// execution and one record-mode execution against the same
// BigQuery emulator dataset and asserts:
//
//  1. Both terminal rows are discoverable via the same canonical
//     view query (QueryCurrentExecution).
//  2. The mode column distinguishes them (set vs record); every
//     other Required column from ADR-0003 CC3 is populated by both.
//  3. window_start / window_end are populated for both and match
//     the trigger's bounds per ADR-0041 + B2-27.
//  4. The execution_id formula is honored: a record-mode row is
//     discoverable by computing Compute(ruleset, entity, ws, we,
//     trigger_source) — same formula set-mode uses.
//  5. dq_check_results rows produced by both runs share the same
//     column shape (composite key, Result, ExecutedAt,
//     EngineVersion).
//  6. A mode-agnostic count query (no WHERE mode = ..., no
//     window-endpoint filter) returns both executions — the
//     simplest consumer-facing invariant from ADR-0041
//     §"Cross-mode dashboard interpretation" Rule 1.
//
// The two executions use distinct entity names so their
// execution_ids cannot collide (entity is a hash input).
func TestIntegration_CrossModeTableShapeParity(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	// Local store + dataset so the test owns the dataset ID for
	// direct SQL queries below (the makeStore helper does not
	// surface it). Mirrors the makeStore pattern verbatim.
	datasetID := fmt.Sprintf("itest_parity_%d", time.Now().UnixNano())
	store := results.NewBigQueryStore(cli, integrationProjectID, datasetID, nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	r := makeRunner(t, store, NoopEvaluator{})

	// --- Phase 1: set-mode execution -----------------------------

	setEntity := "customer_set"
	setStart := time.Date(2026, 5, 31, 14, 0, 0, 0, time.UTC)
	setEnd := setStart.Add(time.Hour)
	setTrigger := TriggerRequest{
		Entity:        setEntity,
		WindowStart:   setStart,
		WindowEnd:     setEnd,
		TriggerSource: results.TriggerScheduler,
		Checks: []CheckSpec{{
			CheckID: "row_count_positive",
			Kind:    "set.row_count_positive",
			Mode:    "set",
		}},
	}

	setTerminal, err := r.Run(context.Background(), setTrigger)
	if err != nil {
		t.Fatalf("set-mode Run: %v", err)
	}
	if setTerminal.Status != results.StatusSuccess {
		t.Fatalf("set-mode terminal.Status = %q; want success", setTerminal.Status)
	}

	// --- Phase 2: record-mode execution --------------------------

	recEntity := "customer_record"
	recBase := time.Date(2026, 5, 31, 16, 0, 0, 0, time.UTC)
	recBatch := []FetchedRecord{
		{Topic: "orders.events.v1", Partition: 0, Offset: 1,
			Timestamp: recBase.Add(5 * time.Second), Body: []byte(`{"id":"a"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 2,
			Timestamp: recBase.Add(30 * time.Second), Body: []byte(`{"id":"b"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 3,
			Timestamp: recBase.Add(90 * time.Second), Body: []byte(`{"id":"c"}`)},
	}
	consumer := &fakeConsumer{batches: [][]FetchedRecord{recBatch}}
	recordRunner, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     r,
		RulesetVersion: "rules-v1.0.0",
		Sources: []RecordSource{{
			Entity:            recEntity,
			Topic:             "orders.events.v1",
			ConsumerGroup:     "dq-orders-stream",
			WindowDuration:    60 * time.Second,
			LatenessTolerance: 10 * time.Second,
			Checks: []CheckSpec{{
				CheckID: "schema_present",
				Kind:    "record.schema_conformance",
				Mode:    "record",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}
	// Start blocks on PollFetches between batches — once the seeded
	// batch is drained the consumer waits until ctx cancellation.
	// Run the loop in a goroutine so the test can poll the store
	// for the dispatched row, then cancel cleanly once the row
	// lands.
	recordCtx, recordCancel := context.WithCancel(context.Background())
	defer recordCancel()
	startDone := make(chan struct{})
	go func() {
		_ = recordRunner.Start(recordCtx)
		close(startDone)
	}()

	// Compute the record-mode execution_id from the same five
	// inputs Runner.Run hashes — the ADR-0002 §3 formula
	// proves identifier-scheme alignment by construction.
	recWindowStart := recBase
	recWindowEnd := recBase.Add(60 * time.Second)
	recExecutionID, err := Compute(
		"rules-v1.0.0",
		recEntity,
		recWindowStart,
		recWindowEnd,
		results.TriggerScheduler,
	)
	if err != nil {
		t.Fatalf("Compute record execution_id: %v", err)
	}

	// --- Phase 3: parity assertions ------------------------------

	setRow := waitForStatus(t, store, setTerminal.ExecutionID, results.StatusSuccess)
	recRow := waitForStatus(t, store, recExecutionID, results.StatusSuccess)

	// Record-mode row landed; release the consumer goroutine.
	recordCancel()
	<-startDone

	// (1) Mode column distinguishes them.
	if setRow.Mode != results.ModeSet {
		t.Errorf("set row Mode = %q; want %q", setRow.Mode, results.ModeSet)
	}
	if recRow.Mode != results.ModeRecord {
		t.Errorf("record row Mode = %q; want %q", recRow.Mode, results.ModeRecord)
	}

	// (2) Required-column parity — ADR-0003 CC3.
	assertRequiredColumnsPopulated(t, "set", setRow)
	assertRequiredColumnsPopulated(t, "record", recRow)

	// (3) Window endpoints populated for both per ADR-0041 + B2-27.
	if !setRow.WindowStart.Equal(setStart) {
		t.Errorf("set row WindowStart = %v; want %v", setRow.WindowStart, setStart)
	}
	if !setRow.WindowEnd.Equal(setEnd) {
		t.Errorf("set row WindowEnd = %v; want %v", setRow.WindowEnd, setEnd)
	}
	if !recRow.WindowStart.Equal(recWindowStart) {
		t.Errorf("record row WindowStart = %v; want %v", recRow.WindowStart, recWindowStart)
	}
	if !recRow.WindowEnd.Equal(recWindowEnd) {
		t.Errorf("record row WindowEnd = %v; want %v", recRow.WindowEnd, recWindowEnd)
	}

	// (4) Terminal-row nullability resolved for both.
	for _, c := range []struct {
		name string
		row  *results.ExecutionRow
	}{{"set", setRow}, {"record", recRow}} {
		if c.row.StartedAt == nil {
			t.Errorf("%s row StartedAt is nil; want set on terminal row (ADR-0003 CC3)", c.name)
		}
		if c.row.CompletedAt == nil {
			t.Errorf("%s row CompletedAt is nil; want set on terminal row (ADR-0003 CC3)", c.name)
		}
		if c.row.ErrorSummary != nil {
			t.Errorf("%s row ErrorSummary = %q; want nil on success", c.name, *c.row.ErrorSummary)
		}
		if c.row.SupersedesExecutionID != nil {
			t.Errorf("%s row SupersedesExecutionID = %q; want nil for scheduler trigger (ADR-0002 CC5)",
				c.name, *c.row.SupersedesExecutionID)
		}
	}

	// (5) execution_id formula honored. setTerminal.ExecutionID is
	// what Runner.Run computed; re-running Compute over the same
	// inputs must yield the identical value.
	setRecomputed, err := Compute(
		setTrigger.RulesetVersion,
		setTrigger.Entity,
		setTrigger.WindowStart,
		setTrigger.WindowEnd,
		setTrigger.TriggerSource,
	)
	if err != nil {
		t.Fatalf("Compute set execution_id: %v", err)
	}
	// setTrigger.RulesetVersion was empty at construction, so
	// Runner.Run fell back to the runner's constructor-time default
	// "rules-v1.0.0" (per makeRunner). Recompute with that.
	if setRecomputed == setTerminal.ExecutionID {
		// Already match; nothing further.
	} else {
		setRecomputedDefault, err := Compute(
			"rules-v1.0.0",
			setTrigger.Entity,
			setTrigger.WindowStart,
			setTrigger.WindowEnd,
			setTrigger.TriggerSource,
		)
		if err != nil {
			t.Fatalf("Compute set execution_id (default ruleset): %v", err)
		}
		if setRecomputedDefault != setTerminal.ExecutionID {
			t.Errorf("set execution_id = %q; want %q (formula divergence)",
				setTerminal.ExecutionID, setRecomputedDefault)
		}
	}
	// The record-mode lookup itself used Compute() at phase 2 — a
	// successful waitForStatus on recExecutionID confirms the
	// formula was honored on the write path. Assert the row's
	// ExecutionID matches the queried key as belt-and-braces.
	if recRow.ExecutionID != recExecutionID {
		t.Errorf("record row ExecutionID = %q; want %q (canonical-view divergence)",
			recRow.ExecutionID, recExecutionID)
	}

	// (6) dq_check_results row-shape parity.
	setChecks := readCheckResults(t, cli, datasetID, setTerminal.ExecutionID)
	recChecks := readCheckResults(t, cli, datasetID, recExecutionID)
	if len(setChecks) != 1 {
		t.Errorf("set execution check-result count = %d; want 1", len(setChecks))
	}
	if len(recChecks) != 1 {
		t.Errorf("record execution check-result count = %d; want 1", len(recChecks))
	}
	for _, c := range append(setChecks, recChecks...) {
		if c.ExecutionID == "" {
			t.Errorf("check-result row missing ExecutionID: %+v", c)
		}
		if c.AttemptID == "" {
			t.Errorf("check-result row missing AttemptID: %+v", c)
		}
		if c.CheckID == "" {
			t.Errorf("check-result row missing CheckID: %+v", c)
		}
		if c.Result != string(results.ResultPass) {
			t.Errorf("check-result Result = %q; want pass", c.Result)
		}
		if c.ExecutedAt.IsZero() {
			t.Errorf("check-result ExecutedAt is zero: %+v", c)
		}
		if c.EngineVersion == "" {
			t.Errorf("check-result row missing EngineVersion: %+v", c)
		}
	}

	// (7) Mode-agnostic count: ADR-0041 §"Cross-mode dashboard
	// interpretation" Rule 1 commits this query shape as valid by
	// construction. No WHERE on mode, no WHERE on window endpoints.
	gotCount := mustQueryCountSuccess(t, cli, datasetID, setEntity, recEntity)
	if gotCount != 2 {
		t.Errorf("mode-agnostic success count = %d; want 2 (one per mode)", gotCount)
	}
}

// assertRequiredColumnsPopulated asserts every ADR-0003 CC3
// Required=true column carries a non-zero value on a terminal
// row. Status-specific nullability (started_at, completed_at,
// error_summary, supersedes_execution_id) is asserted by the
// caller because the contract for nullable columns depends on
// the terminal status.
func assertRequiredColumnsPopulated(t *testing.T, label string, row *results.ExecutionRow) {
	t.Helper()
	if row.ExecutionID == "" {
		t.Errorf("%s row ExecutionID is empty", label)
	}
	if row.AttemptID == "" {
		t.Errorf("%s row AttemptID is empty", label)
	}
	if row.RecordedAt.IsZero() {
		t.Errorf("%s row RecordedAt is zero", label)
	}
	if row.Status == "" {
		t.Errorf("%s row Status is empty", label)
	}
	if row.Mode == "" {
		t.Errorf("%s row Mode is empty", label)
	}
	if row.EngineVersion == "" {
		t.Errorf("%s row EngineVersion is empty", label)
	}
	if row.RulesetVersion == "" {
		t.Errorf("%s row RulesetVersion is empty", label)
	}
	if row.Entity == "" {
		t.Errorf("%s row Entity is empty", label)
	}
	if row.TriggerSource == "" {
		t.Errorf("%s row TriggerSource is empty", label)
	}
	if row.WindowStart.IsZero() {
		t.Errorf("%s row WindowStart is zero", label)
	}
	if row.WindowEnd.IsZero() {
		t.Errorf("%s row WindowEnd is zero", label)
	}
	if !row.WindowEnd.After(row.WindowStart) {
		t.Errorf("%s row WindowEnd (%v) is not strictly after WindowStart (%v)",
			label, row.WindowEnd, row.WindowStart)
	}
}

// checkResultProbe is the subset of dq_check_results columns the
// parity test reads. JSON-typed columns (evidence_summary,
// sample_violating_rows) are intentionally omitted — their
// per-check-kind shape is not part of the cross-mode parity
// contract.
type checkResultProbe struct {
	ExecutionID   string    `bigquery:"execution_id"`
	AttemptID     string    `bigquery:"attempt_id"`
	CheckID       string    `bigquery:"check_id"`
	Result        string    `bigquery:"result"`
	ExecutedAt    time.Time `bigquery:"executed_at"`
	EngineVersion string    `bigquery:"engine_version"`
}

// readCheckResults pulls every dq_check_results row for the
// given execution_id. The Store interface does not expose a
// check-result reader (ADR-0033 added a latest-per-entity-check
// projection but not raw enumeration), so the test runs an
// inline query.
func readCheckResults(t *testing.T, cli *bigquery.Client, datasetID, executionID string) []checkResultProbe {
	t.Helper()
	ctx := context.Background()
	q := cli.Query(fmt.Sprintf(`
SELECT execution_id, attempt_id, check_id, result, executed_at, engine_version
FROM `+"`%s.%s.dq_check_results`"+`
WHERE execution_id = @execution_id
ORDER BY check_id
`, integrationProjectID, datasetID))
	q.Parameters = []bigquery.QueryParameter{{Name: "execution_id", Value: executionID}}

	// Streaming-insert visibility delay: retry briefly.
	deadline := time.Now().Add(5 * time.Second)
	for {
		it, err := q.Read(ctx)
		if err != nil {
			t.Fatalf("readCheckResults query: %v", err)
		}
		var rows []checkResultProbe
		for {
			var r checkResultProbe
			err := it.Next(&r)
			if err == iterator.Done {
				break
			}
			if err != nil {
				t.Fatalf("readCheckResults iterate: %v", err)
			}
			rows = append(rows, r)
		}
		if len(rows) > 0 {
			return rows
		}
		if time.Now().After(deadline) {
			return rows
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// mustQueryCountSuccess runs a mode-agnostic SELECT COUNT(*)
// over dq_executions for the two given entities — the canonical
// "consumer doesn't filter by mode" query shape ADR-0041
// commits as valid SQL by construction. Two-entity flavor is
// portable across the emulator's parameter-handling fidelity
// (IN UNNEST(@array) is not honored).
func mustQueryCountSuccess(t *testing.T, cli *bigquery.Client, datasetID, entityA, entityB string) int64 {
	t.Helper()
	ctx := context.Background()
	q := cli.Query(fmt.Sprintf(`
SELECT COUNT(*) AS n
FROM (
  SELECT execution_id, status,
         ROW_NUMBER() OVER (PARTITION BY execution_id ORDER BY recorded_at DESC) AS rn
  FROM `+"`%s.%s.dq_executions`"+`
  WHERE entity = @entity_a OR entity = @entity_b
)
WHERE rn = 1 AND status = 'success'
`, integrationProjectID, datasetID))
	q.Parameters = []bigquery.QueryParameter{
		{Name: "entity_a", Value: entityA},
		{Name: "entity_b", Value: entityB},
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		it, err := q.Read(ctx)
		if err != nil {
			t.Fatalf("mustQueryCountSuccess query: %v", err)
		}
		var row struct {
			N int64 `bigquery:"n"`
		}
		err = it.Next(&row)
		if err == nil && row.N >= 2 {
			return row.N
		}
		if time.Now().After(deadline) {
			if err != nil && err != iterator.Done {
				t.Fatalf("mustQueryCountSuccess iterate: %v", err)
			}
			return row.N
		}
		time.Sleep(200 * time.Millisecond)
	}
}
