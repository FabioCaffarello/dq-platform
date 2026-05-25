// path: engine/internal/runner/record_runner_test.go

package runner

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"dq-platform/engine/internal/results"
)

// fakeConsumer is a test double for RecordConsumer: returns
// pre-seeded batches one PollFetches call at a time, then
// blocks until ctx is cancelled.
type fakeConsumer struct {
	mu      sync.Mutex
	batches [][]FetchedRecord
	idx     int
	closed  bool
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

func (e *errConsumer) Close() { e.closed = true }

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
