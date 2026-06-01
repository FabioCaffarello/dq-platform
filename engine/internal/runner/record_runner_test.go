// path: engine/internal/runner/record_runner_test.go

package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"

	"dq-platform/engine/internal/metrics"
	"dq-platform/engine/internal/results"
)

// fakeConsumer is a test double for RecordConsumer: returns
// pre-seeded batches one PollFetches call at a time, then
// blocks until ctx is cancelled.
//
// committed accumulates the records the runner passed to Commit
// across the consumer's lifetime per ADR-0058 §Clause 5 test
// discipline; tests assert on this ledger to verify the
// commit-after-dispatch boundary.
//
// commitFailureSequence is the per-call retry failure pattern
// per ADR-0059 §Clause 6: when set, the first N Commit calls
// return the configured errors in order; subsequent calls
// append records to the ledger as usual. Independent from
// commitErr (which returns the same error indefinitely);
// commitFailureSequence drains as it's consumed.
type fakeConsumer struct {
	mu                    sync.Mutex
	batches               [][]FetchedRecord
	idx                   int
	closed                bool
	committed             []FetchedRecord
	commitErr             error
	commitFailureSequence []error
	commitFailureIdx      int
	commitCallCount       int
}

func (f *fakeConsumer) PollFetches(ctx context.Context) ([]FetchedRecord, error) {
	f.mu.Lock()
	if f.idx < len(f.batches) {
		b := f.batches[f.idx]
		f.idx++
		f.mu.Unlock()
		return b, nil
	}
	f.mu.Unlock()
	// Block until ctx is cancelled — simulates a real consumer
	// waiting for new records.
	<-ctx.Done()
	return nil, ctx.Err()
}

func (f *fakeConsumer) Commit(_ context.Context, records []FetchedRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commitCallCount++
	if f.commitFailureIdx < len(f.commitFailureSequence) {
		err := f.commitFailureSequence[f.commitFailureIdx]
		f.commitFailureIdx++
		return err
	}
	if f.commitErr != nil {
		return f.commitErr
	}
	f.committed = append(f.committed, records...)
	return nil
}

func (f *fakeConsumer) commitCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.commitCallCount
}

func (f *fakeConsumer) committedSnapshot() []FetchedRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]FetchedRecord, len(f.committed))
	copy(out, f.committed)
	return out
}

func (f *fakeConsumer) Close() { f.closed = true }

// captureDispatcher captures every trigger it receives so the
// test can assert window-close semantics.
type captureDispatcher struct {
	mu       sync.Mutex
	triggers []TriggerRequest
	err      error
}

func (c *captureDispatcher) Run(_ context.Context, t TriggerRequest) (*results.ExecutionRow, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.triggers = append(c.triggers, t)
	if c.err != nil {
		return nil, c.err
	}
	return &results.ExecutionRow{}, nil
}

func (c *captureDispatcher) snapshot() []TriggerRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]TriggerRequest, len(c.triggers))
	copy(out, c.triggers)
	return out
}

func TestRecordRunner_ClosesWindowOnWatermarkAdvance(t *testing.T) {
	// Window: 60s tumbling, 10s lateness tolerance.
	// Records: 3 records inside the [t0, t0+60s) window, then
	// a record at t0+90s that pushes the watermark past
	// t0+60s+10s=t0+70s, closing the active window and opening
	// a new one.
	base := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	batch := []FetchedRecord{
		{Topic: "orders.events.v1", Partition: 0, Offset: 1,
			Timestamp: base.Add(5 * time.Second),
			Body:      []byte(`{"id":"a"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 2,
			Timestamp: base.Add(15 * time.Second),
			Body:      []byte(`{"id":"b"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 3,
			Timestamp: base.Add(45 * time.Second),
			Body:      []byte(`{"id":"c"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 4,
			Timestamp: base.Add(90 * time.Second),
			Body:      []byte(`{"id":"d"}`)},
	}
	consumer := &fakeConsumer{batches: [][]FetchedRecord{batch}}
	dispatch := &captureDispatcher{}

	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources: []RecordSource{{
			Entity:            "orders_stream",
			Topic:             "orders.events.v1",
			ConsumerGroup:     "dq-orders-stream",
			WindowDuration:    60 * time.Second,
			LatenessTolerance: 10 * time.Second,
			Checks:            []CheckSpec{{CheckID: "schema_present", Kind: "record.schema_conformance"}},
		}},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	triggers := dispatch.snapshot()
	if len(triggers) != 1 {
		t.Fatalf("expected exactly 1 closed-window trigger; got %d", len(triggers))
	}
	tr := triggers[0]
	if tr.Entity != "orders_stream" {
		t.Errorf("trigger.Entity = %q; want orders_stream", tr.Entity)
	}
	if len(tr.Records) != 3 {
		t.Errorf("trigger.Records len = %d; want 3 (the in-window records)", len(tr.Records))
	}
	wantStart := base
	wantEnd := base.Add(60 * time.Second)
	if !tr.WindowStart.Equal(wantStart) || !tr.WindowEnd.Equal(wantEnd) {
		t.Errorf("trigger window = [%v, %v); want [%v, %v)", tr.WindowStart, tr.WindowEnd, wantStart, wantEnd)
	}
}

func TestRecordRunner_LateRecordIncrementsLateDropped(t *testing.T) {
	base := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	// Scenario:
	//   R1 (t0+5s): opens window [t0, t0+60s).
	//   R2 (t0+90s): watermark > t0+70s → closes window 1
	//                (with R1), opens window [t0+60s, t0+120s).
	//   R3 (t0+20s): belongs to a now-closed earlier window →
	//                late drop on the active (second) window.
	//   R4 (t0+200s): watermark > t0+130s → closes window 2
	//                 (containing R2 + the late_dropped count).
	batch := []FetchedRecord{
		{Topic: "orders.events.v1", Partition: 0, Offset: 1,
			Timestamp: base.Add(5 * time.Second),
			Body:      []byte(`{"id":"a"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 2,
			Timestamp: base.Add(90 * time.Second),
			Body:      []byte(`{"id":"b"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 3,
			Timestamp: base.Add(20 * time.Second),
			Body:      []byte(`{"id":"c"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 4,
			Timestamp: base.Add(200 * time.Second),
			Body:      []byte(`{"id":"d"}`)},
	}
	consumer := &fakeConsumer{batches: [][]FetchedRecord{batch}}
	dispatch := &captureDispatcher{}

	r, _ := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources: []RecordSource{{
			Entity:            "orders_stream",
			Topic:             "orders.events.v1",
			ConsumerGroup:     "dq-orders-stream",
			WindowDuration:    60 * time.Second,
			LatenessTolerance: 10 * time.Second,
			Checks:            []CheckSpec{{CheckID: "c1", Kind: "record.schema_conformance"}},
		}},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	triggers := dispatch.snapshot()
	if len(triggers) < 2 {
		t.Fatalf("expected at least 2 closed windows; got %d", len(triggers))
	}
	foundLate := false
	for _, tr := range triggers {
		if tr.LateDroppedCount > 0 {
			foundLate = true
			break
		}
	}
	if !foundLate {
		t.Errorf("expected a trigger with LateDroppedCount > 0; got %v", triggers)
	}
}

func TestRecordRunner_RequiresConsumer(t *testing.T) {
	_, err := NewRecordRunner(RecordRunnerConfig{
		Dispatcher:     &captureDispatcher{},
		RulesetVersion: "v",
	})
	if err == nil {
		t.Fatal("expected error when Consumer is nil")
	}
}

func TestRecordRunner_RequiresDispatcher(t *testing.T) {
	_, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       &fakeConsumer{},
		RulesetVersion: "v",
	})
	if err == nil {
		t.Fatal("expected error when Dispatcher is nil")
	}
}

func TestRecordRunner_RequiresRulesetVersion(t *testing.T) {
	_, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:   &fakeConsumer{},
		Dispatcher: &captureDispatcher{},
	})
	if err == nil {
		t.Fatal("expected error when RulesetVersion is empty")
	}
}

func TestRecordRunner_StartReturnsOnContextCancel(t *testing.T) {
	consumer := &fakeConsumer{}
	r, _ := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     &captureDispatcher{},
		RulesetVersion: "v",
		Sources: []RecordSource{{
			Entity: "x", Topic: "t", ConsumerGroup: "g",
			WindowDuration:    1 * time.Second,
			LatenessTolerance: 1 * time.Second,
		}},
	})
	ctx, cancel := context.WithCancel(context.Background())
	go cancel()
	if err := r.Start(ctx); err != nil {
		t.Errorf("Start returned error: %v", err)
	}
	if !consumer.closed {
		t.Error("consumer should have been closed on shutdown")
	}
}

func TestRecordRunner_ConsumerErrorIsLogged(t *testing.T) {
	consumer := &errConsumer{errs: []error{errors.New("transient")}}
	r, _ := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     &captureDispatcher{},
		RulesetVersion: "v",
		Sources: []RecordSource{{
			Entity: "x", Topic: "t", ConsumerGroup: "g",
			WindowDuration:    1 * time.Second,
			LatenessTolerance: 1 * time.Second,
		}},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = r.Start(ctx)
	// The runner should still close the consumer on shutdown
	// after a poll error.
	if !consumer.closed {
		t.Error("consumer should have been closed on shutdown")
	}
}

// errConsumer returns the configured errors on successive
// PollFetches calls, then blocks until ctx is cancelled.
type errConsumer struct {
	errs   []error
	idx    int
	closed bool
}

func (e *errConsumer) PollFetches(ctx context.Context) ([]FetchedRecord, error) {
	if e.idx < len(e.errs) {
		err := e.errs[e.idx]
		e.idx++
		return nil, err
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func (e *errConsumer) Commit(_ context.Context, _ []FetchedRecord) error { return nil }

func (e *errConsumer) Close() { e.closed = true }

// TestRecordRunner_CommitsAfterSuccessfulDispatch verifies that
// when dispatcher.Run returns nil for a closed window, the
// runner calls consumer.Commit with that window's records per
// ADR-0058 §Clause 2. Mirrors the
// TestRecordRunner_ClosesWindowOnWatermarkAdvance scenario;
// adds a commit-ledger assertion.
func TestRecordRunner_CommitsAfterSuccessfulDispatch(t *testing.T) {
	base := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	inWindow := []FetchedRecord{
		{Topic: "orders.events.v1", Partition: 0, Offset: 1,
			Timestamp: base.Add(5 * time.Second),
			Body:      []byte(`{"id":"a"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 2,
			Timestamp: base.Add(15 * time.Second),
			Body:      []byte(`{"id":"b"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 3,
			Timestamp: base.Add(45 * time.Second),
			Body:      []byte(`{"id":"c"}`)},
	}
	closingRecord := FetchedRecord{Topic: "orders.events.v1", Partition: 0, Offset: 4,
		Timestamp: base.Add(90 * time.Second),
		Body:      []byte(`{"id":"d"}`)}
	batch := append(append([]FetchedRecord{}, inWindow...), closingRecord)
	consumer := &fakeConsumer{batches: [][]FetchedRecord{batch}}
	dispatch := &captureDispatcher{}

	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources: []RecordSource{{
			Entity:            "orders_stream",
			Topic:             "orders.events.v1",
			ConsumerGroup:     "dq-orders-stream",
			WindowDuration:    60 * time.Second,
			LatenessTolerance: 10 * time.Second,
			Checks:            []CheckSpec{{CheckID: "c1", Kind: "record.schema_conformance"}},
		}},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	committed := consumer.committedSnapshot()
	if len(committed) != len(inWindow) {
		t.Fatalf("expected %d committed records (the closed window's records); got %d (%v)",
			len(inWindow), len(committed), committed)
	}
	for i, want := range inWindow {
		if committed[i].Offset != want.Offset || committed[i].Partition != want.Partition {
			t.Errorf("committed[%d] = (p=%d, o=%d); want (p=%d, o=%d)",
				i, committed[i].Partition, committed[i].Offset, want.Partition, want.Offset)
		}
	}
	// The closing record opened a new window but has not closed
	// yet — it must NOT be in the ledger.
	for _, c := range committed {
		if c.Offset == closingRecord.Offset {
			t.Errorf("closing record (offset=%d, still in active window) was committed; should not be", closingRecord.Offset)
		}
	}
}

// TestRecordRunner_DoesNotCommitOnDispatchFailure verifies that
// when dispatcher.Run returns non-nil, the runner does NOT call
// consumer.Commit for that window per ADR-0058 §Clause 2.
func TestRecordRunner_DoesNotCommitOnDispatchFailure(t *testing.T) {
	base := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	batch := []FetchedRecord{
		{Topic: "orders.events.v1", Partition: 0, Offset: 1,
			Timestamp: base.Add(5 * time.Second),
			Body:      []byte(`{"id":"a"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 2,
			Timestamp: base.Add(15 * time.Second),
			Body:      []byte(`{"id":"b"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 3,
			Timestamp: base.Add(90 * time.Second),
			Body:      []byte(`{"id":"c"}`)},
	}
	consumer := &fakeConsumer{batches: [][]FetchedRecord{batch}}
	dispatch := &captureDispatcher{err: errors.New("downstream store unavailable")}

	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources: []RecordSource{{
			Entity:            "orders_stream",
			Topic:             "orders.events.v1",
			ConsumerGroup:     "dq-orders-stream",
			WindowDuration:    60 * time.Second,
			LatenessTolerance: 10 * time.Second,
			Checks:            []CheckSpec{{CheckID: "c1", Kind: "record.schema_conformance"}},
		}},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	// At least one window closed (the dispatcher saw the trigger
	// and returned its error); the commit ledger must remain
	// empty because the dispatch failed.
	if got := len(dispatch.snapshot()); got == 0 {
		t.Fatal("expected at least one dispatch attempt; got 0")
	}
	if committed := consumer.committedSnapshot(); len(committed) != 0 {
		t.Errorf("expected no commits on dispatch failure; got %d records committed (%v)",
			len(committed), committed)
	}
}

// recordCommitTestBatch is the canonical batch used by the
// commit-retry tests: three in-window records (offsets 1–3 with
// timestamps t0+5s/15s/45s, all inside the [t0, t0+60s) window)
// plus one closing record at t0+90s that pushes the watermark
// past t0+70s and triggers closeAndDispatch on the first window.
// Shared between the three commit-retry tests so the assertion
// "first window's 3 records committed (or not)" is uniform.
func recordCommitTestBatch() (base time.Time, batch []FetchedRecord, inWindow []FetchedRecord) {
	base = time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC)
	inWindow = []FetchedRecord{
		{Topic: "orders.events.v1", Partition: 0, Offset: 1,
			Timestamp: base.Add(5 * time.Second),
			Body:      []byte(`{"id":"a"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 2,
			Timestamp: base.Add(15 * time.Second),
			Body:      []byte(`{"id":"b"}`)},
		{Topic: "orders.events.v1", Partition: 0, Offset: 3,
			Timestamp: base.Add(45 * time.Second),
			Body:      []byte(`{"id":"c"}`)},
	}
	closing := FetchedRecord{Topic: "orders.events.v1", Partition: 0, Offset: 4,
		Timestamp: base.Add(90 * time.Second),
		Body:      []byte(`{"id":"d"}`)}
	batch = append(append([]FetchedRecord{}, inWindow...), closing)
	return base, batch, inWindow
}

func recordCommitTestSource() RecordSource {
	return RecordSource{
		Entity:            "orders_stream",
		Topic:             "orders.events.v1",
		ConsumerGroup:     "dq-orders-stream",
		WindowDuration:    60 * time.Second,
		LatenessTolerance: 10 * time.Second,
		Checks:            []CheckSpec{{CheckID: "c1", Kind: "record.schema_conformance"}},
	}
}

// TestRecordRunner_CommitRetryEventualSuccess verifies that when
// the first two consumer.Commit attempts return transient errors
// and the third succeeds, the runner commits the records on the
// successful attempt and the ledger contains exactly that
// window's records per ADR-0059 §Clause 1.
func TestRecordRunner_CommitRetryEventualSuccess(t *testing.T) {
	_, batch, inWindow := recordCommitTestBatch()
	transient := errors.New("kafka commit: broker connection reset")
	consumer := &fakeConsumer{
		batches:               [][]FetchedRecord{batch},
		commitFailureSequence: []error{transient, transient},
	}
	dispatch := &captureDispatcher{}

	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources:        []RecordSource{recordCommitTestSource()},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	if got := consumer.commitCalls(); got != recordCommitMaxAttempts {
		t.Errorf("commit calls = %d; want %d (1 initial + 2 retries)", got, recordCommitMaxAttempts)
	}
	committed := consumer.committedSnapshot()
	if len(committed) != len(inWindow) {
		t.Fatalf("expected %d committed records (the closed window's records); got %d (%v)",
			len(inWindow), len(committed), committed)
	}
}

// TestRecordRunner_CommitRetryExhaustion verifies that when all
// recordCommitMaxAttempts attempts fail, the runner falls
// through to ADR-0058 §Clause 2's warning-log + skip path: no
// records appear in the ledger.
func TestRecordRunner_CommitRetryExhaustion(t *testing.T) {
	_, batch, _ := recordCommitTestBatch()
	transient := errors.New("kafka commit: persistent broker failure")
	// One error per attempt — all failing.
	failureSeq := make([]error, recordCommitMaxAttempts)
	for i := range failureSeq {
		failureSeq[i] = transient
	}
	consumer := &fakeConsumer{
		batches:               [][]FetchedRecord{batch},
		commitFailureSequence: failureSeq,
	}
	dispatch := &captureDispatcher{}

	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources:        []RecordSource{recordCommitTestSource()},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	if got := consumer.commitCalls(); got != recordCommitMaxAttempts {
		t.Errorf("commit calls = %d; want exactly %d (budget exhausted; no further calls)",
			got, recordCommitMaxAttempts)
	}
	if committed := consumer.committedSnapshot(); len(committed) != 0 {
		t.Errorf("expected empty commit ledger on retry exhaustion; got %d records (%v)",
			len(committed), committed)
	}
}

// TestRecordRunner_CommitRetryRespectsContext verifies that
// context cancellation during a back-off wait pre-empts the
// retry loop per ADR-0059 §Clause 3 (the select on ctx.Done in
// commitWithRetry). Uses a long-running consumer (commit fails
// indefinitely) and cancels the context immediately so the test
// can assert the retry loop did not exhaust its budget.
func TestRecordRunner_CommitRetryRespectsContext(t *testing.T) {
	_, batch, _ := recordCommitTestBatch()
	persistent := errors.New("kafka commit: broker unreachable")
	consumer := &fakeConsumer{
		batches:   [][]FetchedRecord{batch},
		commitErr: persistent, // returns the error indefinitely
	}
	dispatch := &captureDispatcher{}

	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     dispatch,
		RulesetVersion: "rules-v0.1.0-beta",
		Sources:        []RecordSource{recordCommitTestSource()},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}

	// Tight context: the runner has time to invoke the first
	// closeAndDispatch (which calls commitWithRetry), but the
	// retry-window stall should be pre-empted by ctx cancellation
	// rather than wait the full budget. Total worst-case retry
	// stall at the chosen β parameters is 600ms; a 100ms context
	// deadline is well below that.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_ = r.Start(ctx)
	elapsed := time.Since(start)

	// The runner returned within the context deadline plus a
	// small margin (the deadline triggers ctx.Done; the select
	// in commitWithRetry returns; closeAndDispatch returns;
	// Start's loop catches ctx.Err and exits). The retry budget
	// (≤600ms worst case) was NOT consumed.
	if elapsed > 400*time.Millisecond {
		t.Errorf("Start took %v after ctx cancellation; expected return within context deadline + margin (retry budget should have been pre-empted)", elapsed)
	}
	// Commit was attempted at least once (the initial attempt);
	// the budget should not have been exhausted before
	// cancellation. The exact count depends on timing of the
	// back-off windows vs the 100ms deadline, so the assertion
	// is bounded: at least 1 (first attempt fired), at most
	// recordCommitMaxAttempts (budget cap).
	calls := consumer.commitCalls()
	if calls < 1 {
		t.Errorf("expected at least 1 commit attempt before cancellation; got %d", calls)
	}
	if calls > recordCommitMaxAttempts {
		t.Errorf("commit calls = %d; exceeds budget cap %d", calls, recordCommitMaxAttempts)
	}
}

// histogramSampleCount extracts the cumulative sample count from
// a HistogramVec's child for the given label values. Used by the
// commit-RPC histogram tests per ADR-0060 §Clause 6.
func histogramSampleCount(t *testing.T, vec *prometheus.HistogramVec, labels ...string) uint64 {
	t.Helper()
	obs, err := vec.GetMetricWithLabelValues(labels...)
	if err != nil {
		t.Fatalf("HistogramVec.GetMetricWithLabelValues(%v): %v", labels, err)
	}
	hist, ok := obs.(prometheus.Histogram)
	if !ok {
		t.Fatalf("expected HistogramVec child to implement prometheus.Histogram; got %T", obs)
	}
	var pb dto.Metric
	if err := hist.Write(&pb); err != nil {
		t.Fatalf("histogram.Write: %v", err)
	}
	return pb.GetHistogram().GetSampleCount()
}

// TestRecordRunner_CommitFailuresCounterIncrementsOnExhaustion verifies
// the two-part invariant ADR-0060 §Clause 5 commits on
// dq_record_commit_failures_total: the counter increments exactly
// once per commitWithRetry cycle that exhausts the retry budget on
// broker failure, AND it is not incremented when the helper returns
// because of context.Canceled / context.DeadlineExceeded (clean
// shutdown is operator-driven, not a failure mode per ADR-0059
// §Clause 5).
func TestRecordRunner_CommitFailuresCounterIncrementsOnExhaustion(t *testing.T) {
	t.Run("ExhaustionIncrements", func(t *testing.T) {
		_, batch, _ := recordCommitTestBatch()
		transient := errors.New("kafka commit: persistent broker failure")
		failureSeq := make([]error, recordCommitMaxAttempts)
		for i := range failureSeq {
			failureSeq[i] = transient
		}
		consumer := &fakeConsumer{
			batches:               [][]FetchedRecord{batch},
			commitFailureSequence: failureSeq,
		}
		reg := metrics.New()
		r, err := NewRecordRunner(RecordRunnerConfig{
			Consumer:       consumer,
			Dispatcher:     &captureDispatcher{},
			RulesetVersion: "rules-v0.1.0-beta",
			Metrics:        reg.Runner,
			Sources:        []RecordSource{recordCommitTestSource()},
		})
		if err != nil {
			t.Fatalf("NewRecordRunner: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.Start(ctx)

		got := testutil.ToFloat64(reg.Runner.RecordCommitFailures.WithLabelValues("orders_stream"))
		if got != 1 {
			t.Errorf("dq_record_commit_failures_total{entity=orders_stream} = %v; want 1 (one cycle exhausted)", got)
		}
	})

	t.Run("ContextCancellationExempt", func(t *testing.T) {
		// fakeConsumer returns context.Canceled on every Commit
		// call. commitWithRetry's terminal return is context.Canceled;
		// closeAndDispatch's errors.Is(err, context.Canceled) branch
		// fires and skips both the warning-log and the failures-counter
		// increment per ADR-0060 §Clause 5 + ADR-0059 §Clause 5.
		_, batch, _ := recordCommitTestBatch()
		consumer := &fakeConsumer{
			batches:   [][]FetchedRecord{batch},
			commitErr: context.Canceled,
		}
		reg := metrics.New()
		r, err := NewRecordRunner(RecordRunnerConfig{
			Consumer:       consumer,
			Dispatcher:     &captureDispatcher{},
			RulesetVersion: "rules-v0.1.0-beta",
			Metrics:        reg.Runner,
			Sources:        []RecordSource{recordCommitTestSource()},
		})
		if err != nil {
			t.Fatalf("NewRecordRunner: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.Start(ctx)

		got := testutil.ToFloat64(reg.Runner.RecordCommitFailures.WithLabelValues("orders_stream"))
		if got != 0 {
			t.Errorf("dq_record_commit_failures_total{entity=orders_stream} = %v; want 0 (shutdown-exemption invariant per ADR-0060 §Clause 5: context.Canceled is not a failure mode)", got)
		}
	})
}

// TestRecordRunner_CommitRetriesCounterOutcomeLabels verifies
// ADR-0060 §Clause 1 + Clause 2: dq_record_commit_retries_total
// increments at the two terminal branches that consumed at least
// one retry (success_after_retry, exhausted) and is NOT incremented
// on the first-attempt-success path.
func TestRecordRunner_CommitRetriesCounterOutcomeLabels(t *testing.T) {
	t.Run("SuccessAfterRetryIncrements", func(t *testing.T) {
		_, batch, _ := recordCommitTestBatch()
		transient := errors.New("kafka commit: broker connection reset")
		consumer := &fakeConsumer{
			batches:               [][]FetchedRecord{batch},
			commitFailureSequence: []error{transient, transient},
		}
		reg := metrics.New()
		r, err := NewRecordRunner(RecordRunnerConfig{
			Consumer:       consumer,
			Dispatcher:     &captureDispatcher{},
			RulesetVersion: "rules-v0.1.0-beta",
			Metrics:        reg.Runner,
			Sources:        []RecordSource{recordCommitTestSource()},
		})
		if err != nil {
			t.Fatalf("NewRecordRunner: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.Start(ctx)

		got := testutil.ToFloat64(reg.Runner.RecordCommitRetries.WithLabelValues("orders_stream", "success_after_retry"))
		if got != 1 {
			t.Errorf("dq_record_commit_retries_total{entity=orders_stream,outcome=success_after_retry} = %v; want 1", got)
		}
		exh := testutil.ToFloat64(reg.Runner.RecordCommitRetries.WithLabelValues("orders_stream", "exhausted"))
		if exh != 0 {
			t.Errorf("dq_record_commit_retries_total{entity=orders_stream,outcome=exhausted} = %v; want 0 (success-after-retry branch fired, not exhausted)", exh)
		}
	})

	t.Run("ExhaustedIncrements", func(t *testing.T) {
		_, batch, _ := recordCommitTestBatch()
		transient := errors.New("kafka commit: persistent broker failure")
		failureSeq := make([]error, recordCommitMaxAttempts)
		for i := range failureSeq {
			failureSeq[i] = transient
		}
		consumer := &fakeConsumer{
			batches:               [][]FetchedRecord{batch},
			commitFailureSequence: failureSeq,
		}
		reg := metrics.New()
		r, err := NewRecordRunner(RecordRunnerConfig{
			Consumer:       consumer,
			Dispatcher:     &captureDispatcher{},
			RulesetVersion: "rules-v0.1.0-beta",
			Metrics:        reg.Runner,
			Sources:        []RecordSource{recordCommitTestSource()},
		})
		if err != nil {
			t.Fatalf("NewRecordRunner: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = r.Start(ctx)

		got := testutil.ToFloat64(reg.Runner.RecordCommitRetries.WithLabelValues("orders_stream", "exhausted"))
		if got != 1 {
			t.Errorf("dq_record_commit_retries_total{entity=orders_stream,outcome=exhausted} = %v; want 1", got)
		}
		sar := testutil.ToFloat64(reg.Runner.RecordCommitRetries.WithLabelValues("orders_stream", "success_after_retry"))
		if sar != 0 {
			t.Errorf("dq_record_commit_retries_total{entity=orders_stream,outcome=success_after_retry} = %v; want 0 (exhausted branch fired)", sar)
		}
	})

	t.Run("FirstAttemptSuccessUninstrumented", func(t *testing.T) {
		_, batch, _ := recordCommitTestBatch()
		consumer := &fakeConsumer{batches: [][]FetchedRecord{batch}} // no failures
		reg := metrics.New()
		r, err := NewRecordRunner(RecordRunnerConfig{
			Consumer:       consumer,
			Dispatcher:     &captureDispatcher{},
			RulesetVersion: "rules-v0.1.0-beta",
			Metrics:        reg.Runner,
			Sources:        []RecordSource{recordCommitTestSource()},
		})
		if err != nil {
			t.Fatalf("NewRecordRunner: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = r.Start(ctx)

		sar := testutil.ToFloat64(reg.Runner.RecordCommitRetries.WithLabelValues("orders_stream", "success_after_retry"))
		if sar != 0 {
			t.Errorf("dq_record_commit_retries_total{outcome=success_after_retry} = %v; want 0 (first-attempt success is the no-op-retry path per ADR-0060 §Clause 1)", sar)
		}
		exh := testutil.ToFloat64(reg.Runner.RecordCommitRetries.WithLabelValues("orders_stream", "exhausted"))
		if exh != 0 {
			t.Errorf("dq_record_commit_retries_total{outcome=exhausted} = %v; want 0", exh)
		}
	})
}

// TestRecordRunner_CommitDurationHistogramObservesPerAttempt verifies
// ADR-0060 §Clause 2: dq_record_commit_duration_seconds records one
// observation per individual consumer.Commit attempt (per-attempt,
// not per-cycle), via the HistogramVec's `_count` series.
func TestRecordRunner_CommitDurationHistogramObservesPerAttempt(t *testing.T) {
	_, batch, _ := recordCommitTestBatch()
	transient := errors.New("kafka commit: broker connection reset")
	// 2 transient failures + 1 success = 3 attempts → 3 observations.
	consumer := &fakeConsumer{
		batches:               [][]FetchedRecord{batch},
		commitFailureSequence: []error{transient, transient},
	}
	reg := metrics.New()
	r, err := NewRecordRunner(RecordRunnerConfig{
		Consumer:       consumer,
		Dispatcher:     &captureDispatcher{},
		RulesetVersion: "rules-v0.1.0-beta",
		Metrics:        reg.Runner,
		Sources:        []RecordSource{recordCommitTestSource()},
	})
	if err != nil {
		t.Fatalf("NewRecordRunner: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = r.Start(ctx)

	got := histogramSampleCount(t, reg.Runner.RecordCommitDuration, "orders_stream")
	if got != uint64(recordCommitMaxAttempts) {
		t.Errorf("dq_record_commit_duration_seconds{entity=orders_stream} sample count = %d; want %d (one observation per consumer.Commit attempt per ADR-0060 §Clause 2)", got, recordCommitMaxAttempts)
	}
}

func TestParseDuration_GrammarSubset(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"1ms", time.Millisecond},
		{"30s", 30 * time.Second},
		{"1m", time.Minute},
		{"2h", 2 * time.Hour},
	}
	for _, c := range cases {
		got, err := ParseDuration(c.in)
		if err != nil {
			t.Errorf("ParseDuration(%q): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParseDuration(%q) = %v; want %v", c.in, got, c.want)
		}
	}
}
