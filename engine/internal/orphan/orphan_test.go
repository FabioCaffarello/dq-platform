// path: engine/internal/orphan/orphan_test.go

package orphan

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"dq-platform/engine/internal/results"
)

// mockScanner is the in-memory test double. It records writes so
// tests can assert the follow-up row shape, and it can be
// configured to fail specific writes for the partial-failure
// test.
type mockScanner struct {
	candidates []results.ExecutionRow
	writes     []results.ExecutionRow
	writeFail  func(row results.ExecutionRow) error // optional per-row failure
}

func (m *mockScanner) ListRunningOlderThan(_ context.Context, _ time.Time) ([]results.ExecutionRow, error) {
	return m.candidates, nil
}

func (m *mockScanner) WriteExecutionRow(_ context.Context, row results.ExecutionRow) error {
	if m.writeFail != nil {
		if err := m.writeFail(row); err != nil {
			return err
		}
	}
	m.writes = append(m.writes, row)
	return nil
}

// fixedClock returns a deterministic now-function for tests.
func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// candidateRow is the standard test fixture: a `running` row with
// recognizable values. Tests mutate this to test specific behaviors.
func candidateRow(now time.Time) results.ExecutionRow {
	started := now.Add(-time.Hour)
	return results.ExecutionRow{
		ExecutionID:    "exec-orphan-1",
		AttemptID:      "att-1",
		RecordedAt:     started,
		Status:         results.StatusRunning,
		EngineVersion:  "0.0.9", // abandoned engine's version
		RulesetVersion: "rules-v1.0.0",
		Entity:         "customer",
		TriggerSource:  results.TriggerScheduler,
		StartedAt:      &started,
	}
}

func newTestDetector(t *testing.T, scanner Scanner, now time.Time) *Detector {
	t.Helper()
	d, err := New(scanner, Config{
		EngineVersion: "1.0.0", // detector's version, distinct from candidate's
		Threshold:     30 * time.Minute,
		Now:           fixedClock(now),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return d
}

func TestNew_RequiresScanner(t *testing.T) {
	_, err := New(nil, Config{EngineVersion: "1.0.0", Threshold: time.Minute})
	if err == nil {
		t.Fatalf("New(nil scanner) returned nil; want error")
	}
}

func TestNew_RequiresEngineVersion(t *testing.T) {
	_, err := New(&mockScanner{}, Config{Threshold: time.Minute})
	if err == nil {
		t.Fatalf("New with empty EngineVersion returned nil; want error")
	}
}

func TestNew_RequiresPositiveThreshold(t *testing.T) {
	for _, bad := range []time.Duration{0, -time.Minute} {
		_, err := New(&mockScanner{}, Config{EngineVersion: "1.0.0", Threshold: bad})
		if err == nil {
			t.Fatalf("New with Threshold=%v returned nil; want error", bad)
		}
	}
}

func TestRunOnce_NoCandidates(t *testing.T) {
	scanner := &mockScanner{}
	d := newTestDetector(t, scanner, time.Now())
	finalized, errs, err := d.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 0 {
		t.Errorf("finalized = %d; want 0", finalized)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v; want empty", errs)
	}
	if len(scanner.writes) != 0 {
		t.Errorf("writes = %v; want none", scanner.writes)
	}
}

func TestRunOnce_OneCandidate_HappyPath(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	scanner := &mockScanner{
		candidates: []results.ExecutionRow{candidateRow(now)},
	}
	d := newTestDetector(t, scanner, now)
	finalized, errs, err := d.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 1 {
		t.Errorf("finalized = %d; want 1", finalized)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v; want empty", errs)
	}
	if len(scanner.writes) != 1 {
		t.Fatalf("writes = %d; want 1", len(scanner.writes))
	}

	row := scanner.writes[0]
	if row.Status != results.StatusAborted {
		t.Errorf("Status = %q; want aborted", row.Status)
	}
	if row.ExecutionID != "exec-orphan-1" || row.AttemptID != "att-1" {
		t.Errorf("identity preserved badly: %+v", row)
	}
	if row.RecordedAt != now {
		t.Errorf("RecordedAt = %v; want %v (detector's now)", row.RecordedAt, now)
	}
	if row.ErrorSummary == nil || *row.ErrorSummary != AbandonmentSummary {
		t.Errorf("ErrorSummary = %v; want %q", row.ErrorSummary, AbandonmentSummary)
	}
}

func TestRunOnce_DetectorEngineVersionWins(t *testing.T) {
	// ADR-0007 CC11 load-bearing invariant: the follow-up row
	// carries the detector's engine_version, NOT the abandoned
	// engine's. This test would fail noisily if buildFollowupRow
	// were ever refactored to propagate candidate.EngineVersion.
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	scanner := &mockScanner{
		candidates: []results.ExecutionRow{candidateRow(now)},
	}
	d := newTestDetector(t, scanner, now)
	if _, _, err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	row := scanner.writes[0]
	if row.EngineVersion != "1.0.0" {
		t.Errorf("EngineVersion = %q; want %q (detector's, not abandoned engine's %q)",
			row.EngineVersion, "1.0.0", "0.0.9")
	}
}

func TestRunOnce_PreservesStartedAtAndSetsCompletedAt(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	candidate := candidateRow(now)
	originalStarted := *candidate.StartedAt

	scanner := &mockScanner{candidates: []results.ExecutionRow{candidate}}
	d := newTestDetector(t, scanner, now)
	if _, _, err := d.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	row := scanner.writes[0]

	if row.StartedAt == nil || !row.StartedAt.Equal(originalStarted) {
		t.Errorf("StartedAt = %v; want %v (preserved from candidate)",
			row.StartedAt, originalStarted)
	}
	if row.CompletedAt == nil || !row.CompletedAt.Equal(now) {
		t.Errorf("CompletedAt = %v; want %v (detector's now)", row.CompletedAt, now)
	}
}

func TestRunOnce_MultipleCandidates_AllFinalized(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	var candidates []results.ExecutionRow
	for i := 0; i < 5; i++ {
		c := candidateRow(now)
		c.ExecutionID = "exec-" + string(rune('A'+i))
		candidates = append(candidates, c)
	}
	scanner := &mockScanner{candidates: candidates}
	d := newTestDetector(t, scanner, now)
	finalized, errs, err := d.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 5 {
		t.Errorf("finalized = %d; want 5", finalized)
	}
	if len(errs) != 0 {
		t.Errorf("errs = %v; want empty", errs)
	}
	if len(scanner.writes) != 5 {
		t.Errorf("writes = %d; want 5", len(scanner.writes))
	}
}

func TestRunOnce_PartialFailure(t *testing.T) {
	now := time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
	var candidates []results.ExecutionRow
	for i, id := range []string{"A", "B", "C"} {
		c := candidateRow(now)
		c.ExecutionID = "exec-" + id
		c.AttemptID = "att-" + string(rune('1'+i))
		candidates = append(candidates, c)
	}
	// Configure the scanner to fail the write whose execution_id
	// is "exec-B". RunOnce must continue and finalize A and C.
	scanner := &mockScanner{
		candidates: candidates,
		writeFail: func(row results.ExecutionRow) error {
			if row.ExecutionID == "exec-B" {
				return errors.New("synthetic write failure")
			}
			return nil
		},
	}
	d := newTestDetector(t, scanner, now)
	finalized, errs, err := d.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if finalized != 2 {
		t.Errorf("finalized = %d; want 2 (A and C; B failed)", finalized)
	}
	if len(errs) != 1 {
		t.Fatalf("errs = %d; want 1", len(errs))
	}
	if !strings.Contains(errs[0].Error(), "exec-B") {
		t.Errorf("error %q should mention exec-B", errs[0])
	}
	// Confirm the writes that DID succeed are for A and C.
	if len(scanner.writes) != 2 {
		t.Fatalf("writes recorded = %d; want 2", len(scanner.writes))
	}
	seen := map[string]bool{}
	for _, w := range scanner.writes {
		seen[w.ExecutionID] = true
	}
	if !seen["exec-A"] || !seen["exec-C"] || seen["exec-B"] {
		t.Errorf("writes had unexpected IDs: %v", seen)
	}
}

// scanErrScanner returns the given error from
// ListRunningOlderThan to exercise the operational-failure
// branch of RunOnce.
type scanErrScanner struct{ err error }

func (s scanErrScanner) ListRunningOlderThan(_ context.Context, _ time.Time) ([]results.ExecutionRow, error) {
	return nil, s.err
}
func (s scanErrScanner) WriteExecutionRow(_ context.Context, _ results.ExecutionRow) error {
	return nil
}

func TestRunOnce_ScanFailure(t *testing.T) {
	scanner := scanErrScanner{err: errors.New("scan failed")}
	d, err := New(scanner, Config{EngineVersion: "1.0.0", Threshold: time.Minute})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	finalized, errs, runErr := d.RunOnce(context.Background())
	if runErr == nil {
		t.Fatalf("RunOnce: got nil err; want operational failure")
	}
	if finalized != 0 || len(errs) != 0 {
		t.Errorf("on scan failure: finalized=%d errs=%v; want 0 and empty", finalized, errs)
	}
}
