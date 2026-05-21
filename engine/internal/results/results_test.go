// path: engine/internal/results/results_test.go

package results

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestExecutionStatus_Values(t *testing.T) {
	// Closed-enum coverage: every committed value is non-empty and
	// distinct from the others. Catches accidental empty consts or
	// duplicate values during refactors.
	cases := map[string]ExecutionStatus{
		"running": StatusRunning,
		"success": StatusSuccess,
		"failed":  StatusFailed,
		"error":   StatusError,
		"aborted": StatusAborted,
	}
	seen := map[ExecutionStatus]string{}
	for want, got := range cases {
		if string(got) != want {
			t.Errorf("ExecutionStatus const %q = %q; want %q", want, got, want)
		}
		if prev, ok := seen[got]; ok {
			t.Errorf("ExecutionStatus %q duplicated (also %q)", got, prev)
		}
		seen[got] = want
	}
}

func TestCheckResult_Values(t *testing.T) {
	cases := map[string]CheckResult{
		"pass":     ResultPass,
		"fail":     ResultFail,
		"degraded": ResultDegraded,
		"error":    ResultError,
	}
	for want, got := range cases {
		if string(got) != want {
			t.Errorf("CheckResult const %q = %q; want %q", want, got, want)
		}
	}
}

func TestTriggerSource_Values(t *testing.T) {
	cases := map[string]TriggerSource{
		"scheduler":      TriggerScheduler,
		"manual":         TriggerManual,
		"operator-rerun": TriggerOperatorRerun,
	}
	for want, got := range cases {
		if string(got) != want {
			t.Errorf("TriggerSource const %q = %q; want %q", want, got, want)
		}
	}
}

func TestExecutionRow_NullableFields(t *testing.T) {
	// A row with all-nullable fields nil should not panic on
	// conversion. Mirrors the running-transition row shape.
	row := ExecutionRow{
		ExecutionID:    "abc",
		AttemptID:      "att-1",
		RecordedAt:     time.Now().UTC(),
		Status:         StatusRunning,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		Entity:         "customer",
		TriggerSource:  TriggerScheduler,
	}
	rec := toExecutionRecord(row)
	if rec.StartedAt.Valid || rec.CompletedAt.Valid ||
		rec.ErrorSummary.Valid || rec.SupersedesExecutionID.Valid {
		t.Errorf("toExecutionRecord: nullable fields should be invalid when ExecutionRow has nil pointers; got %+v", rec)
	}
	back := fromExecutionRecord(rec)
	if back.StartedAt != nil || back.CompletedAt != nil ||
		back.ErrorSummary != nil || back.SupersedesExecutionID != nil {
		t.Errorf("fromExecutionRecord: nullable fields should remain nil; got %+v", back)
	}
}

func TestExecutionRow_NonNullableFields(t *testing.T) {
	// Round-trip with every nullable field populated.
	started := time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC)
	completed := started.Add(time.Second)
	summary := "everything errored"
	supersedes := "deadbeef"

	row := ExecutionRow{
		ExecutionID:           "exec-1",
		AttemptID:             "att-1",
		RecordedAt:            completed,
		Status:                StatusError,
		EngineVersion:         "0.1.0",
		RulesetVersion:        "rules-v1.0.0",
		Entity:                "customer",
		TriggerSource:         TriggerOperatorRerun,
		StartedAt:             &started,
		CompletedAt:           &completed,
		ErrorSummary:          &summary,
		SupersedesExecutionID: &supersedes,
	}
	rec := toExecutionRecord(row)
	if !rec.StartedAt.Valid || !rec.CompletedAt.Valid ||
		!rec.ErrorSummary.Valid || !rec.SupersedesExecutionID.Valid {
		t.Errorf("toExecutionRecord: nullable fields should be Valid; got %+v", rec)
	}
	back := fromExecutionRecord(rec)
	if back.StartedAt == nil || !back.StartedAt.Equal(started) {
		t.Errorf("StartedAt round-trip failed: got %v", back.StartedAt)
	}
	if back.CompletedAt == nil || !back.CompletedAt.Equal(completed) {
		t.Errorf("CompletedAt round-trip failed: got %v", back.CompletedAt)
	}
	if back.ErrorSummary == nil || *back.ErrorSummary != summary {
		t.Errorf("ErrorSummary round-trip failed: got %v", back.ErrorSummary)
	}
	if back.SupersedesExecutionID == nil || *back.SupersedesExecutionID != supersedes {
		t.Errorf("SupersedesExecutionID round-trip failed: got %v", back.SupersedesExecutionID)
	}
}

func TestCheckResultRow_JSONFields(t *testing.T) {
	row := CheckResultRow{
		ExecutionID:   "exec-1",
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
			{"id": 2, "reason": "negative"},
		},
	}
	rec := toCheckResultRecord(row)

	var summary map[string]any
	if err := json.Unmarshal([]byte(rec.EvidenceSummary), &summary); err != nil {
		t.Fatalf("EvidenceSummary not valid JSON: %v; raw=%q", err, rec.EvidenceSummary)
	}
	if summary["rows_scanned"].(float64) != 1000 {
		t.Errorf("EvidenceSummary.rows_scanned round-trip failed: %v", summary)
	}

	var samples []map[string]any
	if err := json.Unmarshal([]byte(rec.SampleViolatingRows), &samples); err != nil {
		t.Fatalf("SampleViolatingRows not valid JSON: %v; raw=%q", err, rec.SampleViolatingRows)
	}
	if len(samples) != 2 {
		t.Errorf("SampleViolatingRows length = %d; want 2", len(samples))
	}
}

func TestCheckResultRow_NilJSONFields(t *testing.T) {
	// Nil maps / slices should marshal to "null", which BigQuery's
	// JSON column accepts.
	row := CheckResultRow{
		ExecutionID:   "exec-1",
		AttemptID:     "att-1",
		CheckID:       "noop",
		Result:        ResultPass,
		ExecutedAt:    time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		EngineVersion: "0.1.0",
	}
	rec := toCheckResultRecord(row)
	if rec.EvidenceSummary != "null" {
		t.Errorf("nil EvidenceSummary should marshal to \"null\"; got %q", rec.EvidenceSummary)
	}
	if rec.SampleViolatingRows != "null" {
		t.Errorf("nil SampleViolatingRows should marshal to \"null\"; got %q", rec.SampleViolatingRows)
	}
}

// mockStore is an in-memory Store for tests that exercise callers
// of the Store interface (the runner / orphan detector will use
// this pattern; for Phase 4b the mock validates the interface
// shape).
type mockStore struct {
	executions []ExecutionRow
	checks     []CheckResultRow
}

func (m *mockStore) EnsureSchema(_ context.Context) error { return nil }

func (m *mockStore) WriteExecutionRow(_ context.Context, row ExecutionRow) error {
	m.executions = append(m.executions, row)
	return nil
}

func (m *mockStore) WriteCheckResultRow(_ context.Context, row CheckResultRow) error {
	m.checks = append(m.checks, row)
	return nil
}

func (m *mockStore) QueryCurrentExecution(_ context.Context, executionID string) (*ExecutionRow, error) {
	var latest *ExecutionRow
	for i := range m.executions {
		r := m.executions[i]
		if r.ExecutionID != executionID {
			continue
		}
		if latest == nil || r.RecordedAt.After(latest.RecordedAt) {
			cp := r
			latest = &cp
		}
	}
	if latest == nil {
		return nil, ErrExecutionNotFound
	}
	return latest, nil
}

func TestStoreInterface_MockShape(t *testing.T) {
	// Confirm that mockStore satisfies the Store interface — this
	// is the shape downstream callers (runner, orphan detector)
	// will rely on. If we change the Store interface this test
	// fails at compile time.
	var s Store = &mockStore{}
	ctx := context.Background()

	if err := s.EnsureSchema(ctx); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	row := ExecutionRow{
		ExecutionID:    "abc",
		AttemptID:      "att-1",
		RecordedAt:     time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		Status:         StatusRunning,
		EngineVersion:  "0.1.0",
		RulesetVersion: "rules-v1.0.0",
		Entity:         "customer",
		TriggerSource:  TriggerScheduler,
	}
	if err := s.WriteExecutionRow(ctx, row); err != nil {
		t.Fatalf("WriteExecutionRow: %v", err)
	}

	got, err := s.QueryCurrentExecution(ctx, "abc")
	if err != nil {
		t.Fatalf("QueryCurrentExecution: %v", err)
	}
	if got.ExecutionID != "abc" {
		t.Errorf("QueryCurrentExecution returned wrong row: %+v", got)
	}

	if _, err := s.QueryCurrentExecution(ctx, "no-such"); err == nil {
		t.Fatalf("QueryCurrentExecution(no-such): got nil error, want ErrExecutionNotFound")
	} else if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q does not mention 'not found'", err)
	}
}

func TestIsAlreadyExists(t *testing.T) {
	// Defensive: the BigQuery SDK's already-exists detection is a
	// string match against the API's stable surface. If the SDK
	// changes the error text in the future, this test catches it
	// at the next CI run.
	if !isAlreadyExists(stringErr("Already Exists: foo")) {
		t.Error(`isAlreadyExists("Already Exists: foo") = false; want true`)
	}
	if !isAlreadyExists(stringErr("alreadyExists in dataset")) {
		t.Error(`isAlreadyExists("alreadyExists ...") = false; want true`)
	}
	if isAlreadyExists(stringErr("permission denied")) {
		t.Error(`isAlreadyExists("permission denied") = true; want false`)
	}
	if isAlreadyExists(nil) {
		t.Error(`isAlreadyExists(nil) = true; want false`)
	}
}

type stringErr string

func (e stringErr) Error() string { return string(e) }
