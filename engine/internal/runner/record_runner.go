// path: engine/internal/runner/record_runner.go

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"dq-platform/engine/internal/results"
)

// RecordSource is one record-mode rule the record runner consumes.
// The struct mirrors the carrying-format from
// engine/internal/dsl/spec.RuleSpec; the runner package does not
// depend on dsl/spec so the engine binary translates at boot.
type RecordSource struct {
	Entity            string
	Topic             string
	ConsumerGroup     string
	WindowDuration    time.Duration
	LatenessTolerance time.Duration
	Checks            []CheckSpec
}

// RecordConsumer is the minimal interface RecordRunner needs from
// a Kafka-compatible event-stream client. The production engine
// wires a franz-go-backed implementation; tests inject a fake.
//
// PollFetches blocks until at least one fetch is available or
// ctx is cancelled. The implementation is expected to track
// consumer-group offsets internally; the runner consumes the
// surfaced records and trusts the implementation to commit
// offsets after the runner returns from PollFetches without
// error (typical franz-go pattern with disabled auto-commit).
//
// Close releases any underlying resources; the runner calls it
// once at shutdown.
type RecordConsumer interface {
	PollFetches(ctx context.Context) ([]FetchedRecord, error)
	Close()
}

// FetchedRecord is the substrate-agnostic record shape the
// RecordRunner consumes. franz-go-backed implementations map
// kgo.Record → FetchedRecord; tests construct directly.
type FetchedRecord struct {
	Topic     string
	Partition int32
	Offset    int64
	Timestamp time.Time
	Body      []byte
}

// RecordRunnerConfig configures a RecordRunner.
type RecordRunnerConfig struct {
	// Consumer is the event-stream consumer the runner reads
	// from. Required.
	Consumer RecordConsumer

	// Sources is the set of record-mode rules the runner should
	// process. Each source is keyed by topic; the runner routes
	// each fetched record to the matching source.
	Sources []RecordSource

	// Dispatcher is the inner set/record-agnostic runner the
	// record runner invokes when a window closes. The engine
	// binary wires the same *Runner the HTTP trigger handler
	// uses; the runner consults the dispatcher's evaluator and
	// store to apply the per-window CheckResult write path.
	Dispatcher TriggerDispatcher

	// RulesetVersion is the active manifest's ruleset_version per
	// ADR-0007 §3. The record runner pins it at construction
	// time and emits it on every trigger.
	RulesetVersion string

	// Now is the clock used for watermark advancement and for
	// timestamp-stamped log lines. Optional; defaults to time.Now.
	Now func() time.Time

	// Logger is the structured logger. Optional; defaults to
	// a discarding logger.
	Logger *slog.Logger
}

// TriggerDispatcher is the interface RecordRunner needs from
// the inner Runner. Defined here so tests can inject a mock
// dispatcher without standing up the full Runner.
type TriggerDispatcher interface {
	Run(ctx context.Context, trigger TriggerRequest) (*results.ExecutionRow, error)
}

// RecordRunner consumes record-mode rules from an event-stream
// substrate and emits a TriggerRequest per closed window per
// ADRs 0024/0025. The runner is a library; the engine binary
// drives its loop via Start in a dedicated goroutine.
type RecordRunner struct {
	consumer       RecordConsumer
	dispatcher     TriggerDispatcher
	rulesetVersion string
	now            func() time.Time
	logger         *slog.Logger

	// Per-entity state. The runner is single-goroutine by
	// construction (Start runs one consumer poll loop); no
	// internal locking is required.
	sources map[string]*RecordSource // keyed by topic
	state   map[string]*entityState  // keyed by entity
}

// entityState carries the active tumbling-window state for one
// record-mode rule. The window-close trigger fires when the
// per-source watermark advances past the active window's end +
// lateness tolerance.
type entityState struct {
	source      *RecordSource
	watermark   time.Time
	active      *recordWindow
	lateDropped int
}

// recordWindow holds the records accumulated for one closed-
// pending window. A single entity carries at most one active
// window at a time; the runner closes the active window before
// opening a new one when a record arrives past the watermark
// boundary.
type recordWindow struct {
	start   time.Time
	end     time.Time
	records []Record
}

// NewRecordRunner validates the config and returns a runner.
// Returns an error for missing required fields.
func NewRecordRunner(cfg RecordRunnerConfig) (*RecordRunner, error) {
	if cfg.Consumer == nil {
		return nil, errors.New("record runner: Consumer is required")
	}
	if cfg.Dispatcher == nil {
		return nil, errors.New("record runner: Dispatcher is required")
	}
	if cfg.RulesetVersion == "" {
		return nil, errors.New("record runner: RulesetVersion is required")
	}
	r := &RecordRunner{
		consumer:       cfg.Consumer,
		dispatcher:     cfg.Dispatcher,
		rulesetVersion: cfg.RulesetVersion,
		now:            cfg.Now,
		logger:         cfg.Logger,
		sources:        map[string]*RecordSource{},
		state:          map[string]*entityState{},
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.logger == nil {
		r.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	for i := range cfg.Sources {
		src := cfg.Sources[i]
		if src.Topic == "" {
			return nil, errors.New("record runner: source.Topic is required")
		}
		if src.WindowDuration <= 0 {
			return nil, fmt.Errorf("record runner: source %q WindowDuration must be > 0", src.Entity)
		}
		r.sources[src.Topic] = &src
		r.state[src.Entity] = &entityState{source: &src}
	}
	return r, nil
}

// Start runs the consumer poll loop until ctx is cancelled.
// Each iteration:
//
//  1. Poll the consumer for a batch of fetched records.
//  2. Route each record to its entity's state.
//  3. For each entity, close the active window if the
//     watermark has advanced past window_end + lateness_tolerance.
//  4. Repeat until ctx.Done.
//
// On shutdown (ctx.Done), the runner closes the consumer and
// returns. Any active in-progress window is dropped; per ADR-0024
// the next start re-reads from the consumer-group offset and
// the dropped window becomes a late-drop on the next attempt.
func (r *RecordRunner) Start(ctx context.Context) error {
	defer r.consumer.Close()
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("record runner shutting down", "reason", ctx.Err())
			return nil
		default:
		}
		batch, err := r.consumer.PollFetches(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			r.logger.Warn("record runner poll failed",
				"error", err.Error(),
				"adr_reference", "ADR-0024",
			)
			// Brief back-off to avoid a tight error loop. The
			// consumer is expected to surface transient errors;
			// systemic failures will keep recurring and the
			// engine binary's restart policy is the recovery
			// mechanism.
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		for _, fetched := range batch {
			r.handleFetched(ctx, fetched)
		}
	}
}

// handleFetched routes one fetched record to its entity's state
// and triggers a window close if the watermark has advanced
// past the active window's boundary.
func (r *RecordRunner) handleFetched(ctx context.Context, f FetchedRecord) {
	src, ok := r.sources[f.Topic]
	if !ok {
		// Unknown topic — shouldn't happen if the consumer is
		// subscribed only to topics in r.sources; log defensively.
		r.logger.Warn("record runner received record from unknown topic",
			"topic", f.Topic,
			"partition", f.Partition,
			"offset", f.Offset,
		)
		return
	}
	state := r.state[src.Entity]
	if state == nil {
		return
	}

	// Advance the watermark: monotonic max of record timestamps.
	if f.Timestamp.After(state.watermark) {
		state.watermark = f.Timestamp
	}

	// Compute the window the record belongs to.
	windowStart := f.Timestamp.Truncate(src.WindowDuration)
	windowEnd := windowStart.Add(src.WindowDuration)

	// If we have no active window yet, open one keyed on this
	// record's window.
	if state.active == nil {
		state.active = &recordWindow{start: windowStart, end: windowEnd}
	}

	// If this record belongs to a window earlier than the
	// active one, treat it as a late drop.
	if windowStart.Before(state.active.start) {
		state.lateDropped++
		return
	}

	// If this record belongs to a window later than the active
	// one, the active window is "logically closed" — but per
	// ADR-0024 the close trigger fires once the watermark
	// advances past active.end + lateness_tolerance, not the
	// moment a later-window record arrives. We collect the
	// later record in a forward-window buffer below by rotating
	// the active window once the watermark advances. For
	// simplicity at sub-slice β, we close the active window
	// eagerly when a strictly-later-window record arrives and
	// the watermark already exceeds active.end + lateness.
	if windowStart.After(state.active.start) {
		// Close the active window if its grace period has
		// expired; otherwise buffer the late-window record on
		// the active window (rare; arises when the
		// lateness_tolerance is generous and out-of-order data
		// is small).
		if state.watermark.After(state.active.end.Add(src.LatenessTolerance)) {
			r.closeAndDispatch(ctx, state)
			state.active = &recordWindow{start: windowStart, end: windowEnd}
		} else {
			// Record belongs to a future window but the active
			// is still open; ADR-0024 commits to per-window
			// independence — accumulating the future-window
			// record into a parallel buffer is a follow-up
			// enhancement. For β we close the active window
			// eagerly and open the new one to maintain
			// progress.
			r.closeAndDispatch(ctx, state)
			state.active = &recordWindow{start: windowStart, end: windowEnd}
		}
	}

	// Record falls in the active window (or just-opened): append.
	state.active.records = append(state.active.records, Record{
		Partition: f.Partition,
		Offset:    f.Offset,
		Timestamp: f.Timestamp,
		Body:      f.Body,
	})

	// Post-append close check: did this record's timestamp push
	// the watermark past active.end + lateness_tolerance?
	if state.watermark.After(state.active.end.Add(src.LatenessTolerance)) {
		r.closeAndDispatch(ctx, state)
	}
}

// closeAndDispatch finalizes the active window: emits a
// TriggerRequest with the accumulated records, invokes the
// dispatcher, and resets the per-entity state. Per-attempt
// dispatch failures are warning-logged; the per-window outcome
// is the dispatcher's concern (it writes the dq_executions /
// dq_check_results rows).
//
// After dispatch, state.active is cleared and state.lateDropped
// is reset to 0 — the next window starts fresh.
func (r *RecordRunner) closeAndDispatch(ctx context.Context, state *entityState) {
	w := state.active
	if w == nil {
		return
	}
	trigger := TriggerRequest{
		Entity:           state.source.Entity,
		WindowStart:      w.start,
		WindowEnd:        w.end,
		TriggerSource:    results.TriggerScheduler,
		Checks:           state.source.Checks,
		RulesetVersion:   r.rulesetVersion,
		Records:          w.records,
		LateDroppedCount: state.lateDropped,
	}
	r.logger.Info("record window closing",
		"entity", state.source.Entity,
		"window_start", w.start.UTC(),
		"window_end", w.end.UTC(),
		"records", len(w.records),
		"late_dropped", state.lateDropped,
		"adr_reference", "ADR-0024",
	)
	if _, err := r.dispatcher.Run(ctx, trigger); err != nil {
		r.logger.Warn("record window dispatch failed",
			"entity", state.source.Entity,
			"window_start", w.start.UTC(),
			"error", err.Error(),
		)
	}
	state.active = nil
	state.lateDropped = 0
}

// ParseDuration parses a duration literal from the rule's
// source.window.duration / lateness_tolerance fields. Mirrors
// the grammar enforced by the linter and the parser (regex
// `^[0-9]+(ms|s|m|h)$`). Exposed for the engine binary that
// translates RuleSource → RecordSource at boot.
func ParseDuration(s string) (time.Duration, error) {
	// time.ParseDuration accepts our grammar — `ms`, `s`, `m`,
	// `h` are all standard — but also accepts richer forms
	// (`1h30m`, `1.5s`) we don't want. The parser already
	// rejected non-conforming strings; this is a thin wrapper.
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("parse duration %q: %w", s, err)
	}
	return d, nil
}

// Compile-time assertion that *Runner satisfies the
// TriggerDispatcher interface. Wave-S sub-slice β wires the
// record runner with the same Runner the HTTP handler uses.
var _ TriggerDispatcher = (*Runner)(nil)

// channel-style barrier helpers — kept in this file so the
// record runner's blocking semantics are co-located with its
// state machine.

// drainOrCancel waits until either the provided context is
// cancelled or the provided channel emits. Used in tests to
// avoid leaking goroutines past context cancellation.
//
//nolint:unused // helper retained for future tests
func drainOrCancel(ctx context.Context, ready <-chan struct{}) {
	select {
	case <-ctx.Done():
	case <-ready:
	}
}

// Compile-time sanity: sync is imported transitively via
// stdlib elsewhere, but referencing it here keeps the import
// explicit so future code that adds mutex-protected fields
// doesn't need to re-add the import.
var _ = sync.Mutex{}
