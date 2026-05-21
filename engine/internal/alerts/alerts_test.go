// path: engine/internal/alerts/alerts_test.go

package alerts

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"dq-platform/engine/internal/results"
)

// --- MapCategory tests (ADR-0006 CC7) ---

func ptrCheck(r results.CheckResult) *results.CheckResult       { return &r }
func ptrStatus(s results.ExecutionStatus) *results.ExecutionStatus { return &s }
func ptrStr(s string) *string                                     { return &s }

func TestMapCategory_CheckPass_NoAlert(t *testing.T) {
	cat, emit := MapCategory(SourceRunner, ptrCheck(results.ResultPass), nil)
	if emit {
		t.Errorf("check=pass should not emit; got category=%q", cat)
	}
}

func TestMapCategory_CheckFail_DataQuality(t *testing.T) {
	for _, r := range []results.CheckResult{results.ResultFail, results.ResultDegraded} {
		cat, emit := MapCategory(SourceRunner, ptrCheck(r), nil)
		if !emit || cat != CategoryDataQuality {
			t.Errorf("result=%q: got (%q, %v); want (data_quality, true)", r, cat, emit)
		}
	}
}

func TestMapCategory_CheckError_Operational(t *testing.T) {
	cat, emit := MapCategory(SourceRunner, ptrCheck(results.ResultError), nil)
	if !emit || cat != CategoryOperational {
		t.Errorf("got (%q, %v); want (operational, true)", cat, emit)
	}
}

func TestMapCategory_ExecutionSuccess_NoAlert(t *testing.T) {
	cat, emit := MapCategory(SourceRunner, nil, ptrStatus(results.StatusSuccess))
	if emit {
		t.Errorf("execution=success should not emit; got category=%q", cat)
	}
}

func TestMapCategory_ExecutionFailedErrorAborted_Operational(t *testing.T) {
	for _, s := range []results.ExecutionStatus{
		results.StatusFailed,
		results.StatusError,
		results.StatusAborted,
	} {
		cat, emit := MapCategory(SourceRunner, nil, ptrStatus(s))
		if !emit || cat != CategoryOperational {
			t.Errorf("status=%q: got (%q, %v); want (operational, true)", s, cat, emit)
		}
	}
}

func TestMapCategory_NonRunnerSources_AlwaysOperational(t *testing.T) {
	for _, src := range []EventSource{
		SourceLoader,
		SourceScheduler,
		SourceOrphanDetector,
		SourceTriggerHandler,
	} {
		cat, emit := MapCategory(src, nil, nil)
		if !emit || cat != CategoryOperational {
			t.Errorf("source=%q: got (%q, %v); want (operational, true)", src, cat, emit)
		}
	}
}

func TestMapCategory_RunnerNoSignals_NoAlert(t *testing.T) {
	cat, emit := MapCategory(SourceRunner, nil, nil)
	if emit {
		t.Errorf("runner with no signals should not emit; got category=%q", cat)
	}
}

// --- Dedup tests (ADR-0006 CC5) ---

func sampleEvent(execID, attID, checkID string, result results.CheckResult) Event {
	now := time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC)
	return Event{
		ExecutionID: ptrStr(execID),
		AttemptID:   ptrStr(attID),
		Entity:      "customer",
		CheckID:     ptrStr(checkID),
		Category:    CategoryDataQuality,
		EventSource: SourceRunner,
		Result:      ptrCheck(result),
		RecordedAt:  now,
	}
}

func TestDedup_FirstCallPublishes_SecondSuppressed(t *testing.T) {
	d := NewAttemptDeduper()
	e := sampleEvent("exec1", "att1", "c1", results.ResultFail)

	if !d.ShouldPublish(e) {
		t.Errorf("first ShouldPublish = false; want true")
	}
	if d.ShouldPublish(e) {
		t.Errorf("second ShouldPublish (same key) = true; want false (dedup)")
	}
}

func TestDedup_DifferentChecks_BothPublish(t *testing.T) {
	d := NewAttemptDeduper()
	a := sampleEvent("exec1", "att1", "c1", results.ResultFail)
	b := sampleEvent("exec1", "att1", "c2", results.ResultFail)

	if !d.ShouldPublish(a) || !d.ShouldPublish(b) {
		t.Errorf("different check_ids should both publish")
	}
}

func TestDedup_ResultChange_BothPublish(t *testing.T) {
	// Per ADR-0006 CC5: engine-side dedup keys on result; the
	// consumer-side dedup collapses result-value changes. So
	// the engine-side deduper allows both emissions.
	d := NewAttemptDeduper()
	a := sampleEvent("exec1", "att1", "c1", results.ResultFail)
	b := sampleEvent("exec1", "att1", "c1", results.ResultError)

	if !d.ShouldPublish(a) || !d.ShouldPublish(b) {
		t.Errorf("same key + different result should both publish (consumer-side collapses)")
	}
}

func TestDedup_DifferentAttempts_BothPublish(t *testing.T) {
	d := NewAttemptDeduper()
	a := sampleEvent("exec1", "att1", "c1", results.ResultFail)
	b := sampleEvent("exec1", "att2", "c1", results.ResultFail)

	if !d.ShouldPublish(a) || !d.ShouldPublish(b) {
		t.Errorf("different attempt_ids should both publish")
	}
}

func TestDedup_ConcurrentSafe(t *testing.T) {
	// AttemptDeduper is consumed across goroutines (the runner
	// may emit from multiple worker routines in a future
	// extension). Test that ShouldPublish is race-free.
	d := NewAttemptDeduper()
	e := sampleEvent("exec1", "att1", "c1", results.ResultFail)

	const N = 100
	var wg sync.WaitGroup
	var publishedCount int
	var mu sync.Mutex
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			if d.ShouldPublish(e) {
				mu.Lock()
				publishedCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if publishedCount != 1 {
		t.Errorf("concurrent ShouldPublish: published=%d; want exactly 1", publishedCount)
	}
}

// --- Event JSON tests (ADR-0006 §4) ---

func TestEvent_JSONRoundTrip(t *testing.T) {
	now := time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC)
	severity := SeverityWarning
	original := Event{
		ExecutionID:  ptrStr("exec1"),
		AttemptID:    ptrStr("att1"),
		Entity:       "customer",
		CheckID:      ptrStr("c1"),
		Category:     CategoryDataQuality,
		Severity:     &severity,
		EventSource:  SourceRunner,
		Result:       ptrCheck(results.ResultFail),
		RecordedAt:   now,
		ErrorSummary: ptrStr("some summary"),
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got Event
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Entity != original.Entity || got.Category != original.Category {
		t.Errorf("round-trip lost fields; got %+v", got)
	}
	if got.Severity == nil || *got.Severity != severity {
		t.Errorf("severity round-trip failed: %v", got.Severity)
	}
}

func TestEvent_JSON_OmitsNilFields(t *testing.T) {
	// Execution-startup event: no execution_id, no attempt_id,
	// no check_id, no result, no severity, no error_summary.
	now := time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC)
	e := Event{
		Entity:      "customer",
		Category:    CategoryOperational,
		EventSource: SourceLoader,
		RecordedAt:  now,
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(raw)
	for _, forbidden := range []string{"execution_id", "attempt_id", "check_id", "result", "severity", "error_summary"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("JSON should omit %q for nil pointer; raw: %s", forbidden, s)
		}
	}
	if !strings.Contains(s, `"entity":"customer"`) {
		t.Errorf("entity must be present: %s", s)
	}
}

// --- NoopPublisher ---

func TestNoopPublisher_NeverFails(t *testing.T) {
	if err := (NoopPublisher{}).Publish(nil, Event{}); err != nil {
		t.Errorf("NoopPublisher.Publish returned %v; want nil", err)
	}
}
