// path: engine/internal/runner/record_runner.go

package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"time"

	"dq-platform/engine/internal/metrics"
	"dq-platform/engine/internal/results"
)

// recordCommitRetryOutcome enumerates the two outcome label values
// on dq_record_commit_retries_total per ADR-0060 §Clause 1.
const (
	recordCommitRetryOutcomeSuccessAfterRetry = "success_after_retry"
	recordCommitRetryOutcomeExhausted         = "exhausted"
)

// Commit-retry parameters per ADR-0059 §Clause 2. Constants are
// package-level (not Config-injected) — operator-tunable knobs
// are deferred per ADR-0059 OQ-1 until production signal
// motivates promotion to env-var or runner-config field.
const (
	// recordCommitMaxAttempts is the total number of
	// consumer.Commit attempts (1 initial + 2 retries) the
	// runner makes before falling through to ADR-0058 §Clause 2's
	// warning-log + skip path.
	recordCommitMaxAttempts = 3

	// recordCommitBackoffBase is the base of the exponential
	// back-off schedule: the window upper bound after the Nth
	// failure is base × 2^N. At the chosen parameters, back-off
	// 1's window is [0, 200ms]; back-off 2's window is [0, 400ms];
	// total worst-case stall is 600ms.
	recordCommitBackoffBase = 100 * time.Millisecond
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
// ctx is cancelled. The runner — not the consumer — is the sole
// commit authority. PollFetches must not commit offsets; the
// runner calls Commit after each successful dispatcher return
// per ADR-0058 §Clause 2.
//
// Commit advances the consumer-group committed offset to cover
// the passed records (high-water mark per partition). Called by
// the runner exclusively after dispatcher.Run returns nil for a
// closed window, passing that window's records. Per ADR-0058
// §Clause 3, the production implementation translates each
// record to the underlying client's commit-records primitive.
// Commit failures are warning-logged by the runner; the next
// successful dispatch in the same partition commits the
// uncommitted records transitively.
//
// Close releases any underlying resources; the runner calls it
// once at shutdown.
type RecordConsumer interface {
	PollFetches(ctx context.Context) ([]FetchedRecord, error)
	Commit(ctx context.Context, records []FetchedRecord) error
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

	// Metrics is the per-package RunnerMetrics handle set per
	// ADR-0055 §Clause 3 + ADR-0060 §Clause 1. The engine binary
	// passes the Registry's RunnerMetrics; tests use
	// metrics.NoopRunnerMetrics() (the zero value's nil handles
	// would otherwise panic on the commit-path emission sites).
	// The record runner emits dq_record_commit_failures_total,
	// dq_record_commit_retries_total, and
	// dq_record_commit_duration_seconds from the commit path
	// per ADR-0060 §Clause 2.
	Metrics metrics.RunnerMetrics
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
	metrics        metrics.RunnerMetrics

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
//
// fetched carries the same records in their FetchedRecord shape
// (with the Topic that Record drops) for the Commit RPC per
// ADR-0058 §Clause 2. The parallel field keeps Record-shaped
// TriggerRequest.Records untouched per ADR-0026 evidence
// contract — reshaping Record would ripple into per-kind
// evaluators outside the slice's scope.
type recordWindow struct {
	start   time.Time
	end     time.Time
	records []Record
	fetched []FetchedRecord
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
		metrics:        cfg.Metrics,
		sources:        map[string]*RecordSource{},
		state:          map[string]*entityState{},
	}
	if r.now == nil {
		r.now = time.Now
	}
	if r.logger == nil {
		r.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	// Mirror the runner.go convention per ADR-0055 §Clause 3:
	// if the caller didn't pass Metrics, install Noop handles so
	// the commit-path emission sites never deref a nil counter.
	if r.metrics.RecordCommitFailures == nil {
		r.metrics = metrics.NoopRunnerMetrics()
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
	state.active.fetched = append(state.active.fetched, f)

	// Post-append close check: did this record's timestamp push
	// the watermark past active.end + lateness_tolerance?
	if state.watermark.After(state.active.end.Add(src.LatenessTolerance)) {
		r.closeAndDispatch(ctx, state)
	}
}

// closeAndDispatch finalizes the active window: emits a
// TriggerRequest with the accumulated records, invokes the
// dispatcher, and resets the per-entity state. Per ADR-0058
// §Clause 2, the consumer-group offset commit fires only after
// dispatcher.Run returns nil for this window — passing the
// window's FetchedRecord slice to consumer.Commit. On dispatch
// failure the commit is skipped; the records remain uncommitted
// in the broker; on engine restart they re-flow and re-populate
// the same window deterministically per ADR-0024, with
// canonical-view collapse on the dq_executions side per
// ADR-0003 §2 absorbing any spurious second attempt at the
// consumer-visible layer.
//
// Commit failures are warning-logged and do not propagate; the
// next successful dispatch in the same partition commits the
// uncommitted records transitively via high-water-mark
// monotonicity. Per ADR-0060 §Clause 5, the warning-log path
// also increments dq_record_commit_failures_total — excluding
// context.Canceled / context.DeadlineExceeded so the counter
// tracks broker failure, not operator-driven shutdown.
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
		state.active = nil
		state.lateDropped = 0
		return
	}
	if err := r.commitWithRetry(ctx, w.fetched); err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			r.logger.Warn("record window commit failed",
				"entity", state.source.Entity,
				"window_start", w.start.UTC(),
				"error", err.Error(),
				"commit_attempts", recordCommitMaxAttempts,
				"adr_reference", "ADR-0058+ADR-0059",
			)
			// Per ADR-0060 §Clause 5: increment the failures
			// counter alongside the warning-log line, excluding
			// context.Canceled / context.DeadlineExceeded
			// (operator-driven shutdown is not a failure mode
			// per ADR-0059 §Clause 5).
			r.metrics.RecordCommitFailures.WithLabelValues(state.source.Entity).Inc()
		}
	}
	state.active = nil
	state.lateDropped = 0
}

// commitWithRetry invokes consumer.Commit up to
// recordCommitMaxAttempts times per ADR-0059. Back-off between
// attempts is uniform-random within an exponentially-growing
// window (random_uniform(0, base × 2^attempt)) — the random draw
// de-synchronizes retries between concurrent runners against a
// recovering broker. Context cancellation pre-empts the back-off
// wait and returns ctx.Err() immediately so engine shutdown is
// not stalled by the retry budget.
//
// Returns nil on the first successful commit; returns the last
// commit error after the budget is exhausted; returns ctx.Err()
// on cancellation. The caller (closeAndDispatch) is responsible
// for warning-logging the non-nil non-context return per
// ADR-0058 §Clause 2's terminal path.
//
// Instrumentation per ADR-0060: each consumer.Commit call is
// observed by dq_record_commit_duration_seconds (per-attempt,
// per §Clause 2); the retries counter increments at the two
// terminal branches that consumed at least one retry —
// success-after-retry (err == nil where attempt > 1) and
// exhausted (attempt == recordCommitMaxAttempts && err != nil).
// First-attempt success is uninstrumented per §Clause 1.
func (r *RecordRunner) commitWithRetry(ctx context.Context, records []FetchedRecord) error {
	entity := r.commitEntity(records)
	var lastErr error
	for attempt := 1; attempt <= recordCommitMaxAttempts; attempt++ {
		start := time.Now()
		err := r.consumer.Commit(ctx, records)
		r.metrics.RecordCommitDuration.WithLabelValues(entity).Observe(time.Since(start).Seconds())
		if err == nil {
			if attempt > 1 {
				r.metrics.RecordCommitRetries.WithLabelValues(entity, recordCommitRetryOutcomeSuccessAfterRetry).Inc()
			}
			return nil
		}
		lastErr = err
		if attempt == recordCommitMaxAttempts {
			r.metrics.RecordCommitRetries.WithLabelValues(entity, recordCommitRetryOutcomeExhausted).Inc()
			return lastErr
		}
		// Window upper bound: base << attempt = base × 2^attempt.
		// At attempt=1: [0, base × 2 = 200ms]; attempt=2: [0,
		// base × 4 = 400ms].
		windowMax := time.Duration(int64(recordCommitBackoffBase) << attempt)
		delay := time.Duration(rand.Float64() * float64(windowMax))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return lastErr
}

// commitEntity resolves the entity label for the commit-path
// metrics. The slice's records are always from one window of one
// entity per ADR-0058 §Clause 2; the lookup is a topic →
// source.Entity translation through r.sources. Defensive fallback
// to empty string if the records slice is empty or the topic is
// unknown — the latter shouldn't happen given the Start loop's
// own topic-routing guard.
func (r *RecordRunner) commitEntity(records []FetchedRecord) string {
	if len(records) == 0 {
		return ""
	}
	if src, ok := r.sources[records[0].Topic]; ok {
		return src.Entity
	}
	return ""
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
