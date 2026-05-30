// path: engine/internal/runner/runner.go

// Package runner orchestrates one execution attempt against the
// loaded manifest. It computes the execution_id (ADR-0002), writes
// the running row (ADR-0003), runs the pre-check (ADR-0007 CC8),
// evaluates each check (always-continue per ADR-0004 CC4), writes
// per-check result rows (ADR-0003 CC7), applies the failure-scope
// status mapping (ADR-0004 CC2), and writes the terminal row.
//
// The runner is library-shaped; the engine binary instantiates
// it and (in a future phase) hands triggers to it via an HTTP /
// gRPC handler. Phase 4c exercises the runner via Go tests
// directly.
//
// # Observability emission (ADR-0007 §12)
//
// The runner emits log + metric signals at every terminal
// transition and at every per-check evaluation. Logs go through
// slog.Info / slog.Warn at the same paths they always have; the
// metric channel went live with ADR-0055, which wires the
// engine/internal/metrics package's RunnerMetrics through
// Config and emits five of ADR-0039's eight inventory metrics
// (dq_runs_total, dq_run_duration_seconds,
// dq_checks_evaluated_total, dq_check_duration_seconds,
// dq_bytes_scanned) at the call sites named in ADR-0055
// §Clause 4. The span / tracing channel from ADR-0007 §12
// remains a separate scope (B3-4 OQ-2; lands in a follow-on
// session).
package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/Masterminds/semver/v3"

	"dq-platform/engine/internal/alerts"
	"dq-platform/engine/internal/metrics"
	"dq-platform/engine/internal/results"
)

// TriggerRequest is the input to Runner.Run. The five identity
// fields (Entity, WindowStart, WindowEnd, TriggerSource, plus the
// effective RulesetVersion) are the inputs to the execution_id
// hash per ADR-0002 CC1.
type TriggerRequest struct {
	Entity        string
	WindowStart   time.Time
	WindowEnd     time.Time
	TriggerSource results.TriggerSource

	// SupersedesExecutionID is required for operator-rerun
	// triggers per ADR-0002 CC5 and forbidden for scheduler /
	// manual. The runner enforces both halves at input
	// validation.
	SupersedesExecutionID *string

	// Checks is the list of checks to evaluate. Phase 4c accepts
	// these directly from the trigger; Phase 6 wires manifest-
	// driven check lookup so the trigger handler discovers the
	// entity's check list from the loaded manifest.
	Checks []CheckSpec

	// RulesetVersion overrides the runner's constructor-time pin
	// for this call only. The trigger handler (W3-P4e) sets this
	// to the active manifest's RulesetVersion at trigger
	// acceptance per ADR-0007 §3 (in-flight execution isolation
	// against the manifest active at plan creation). Empty
	// string falls back to the runner's constructor-time value
	// for Phase-4c backwards compatibility.
	RulesetVersion string

	// AttemptID overrides the runner's constructor-time
	// AttemptIDFunc for this call only. The trigger handler
	// (W3-P4e) mints the UUID at trigger acceptance per ADR-0003
	// §4 so the response DTO and the persisted row carry the
	// same identifier. Nil falls back to the runner's
	// configured AttemptIDFunc.
	AttemptID *string

	// Records is the per-window batch of records the
	// record-mode runner accumulated before the window closed
	// (per ADR-0024). The record handler reads this slice;
	// set-mode triggers leave it nil. Per ADR-0025 aggregation
	// happens inside the handler — the runner does not
	// inspect Records.
	Records []Record

	// LateDroppedCount is the count of records that arrived
	// after the watermark closed the window per ADR-0024. The
	// record handler surfaces this in evidence per ADR-0026 and
	// uses it to disambiguate the vacuous-case status (zero
	// records evaluated + zero late drops ⇒ pass; zero records
	// evaluated + positive late drops ⇒ degraded). Set-mode
	// triggers leave it zero.
	LateDroppedCount int
}

// Record is one Kafka message presented to a record-mode handler.
// The runner does not interpret Body; the handler decodes per its
// per-kind contract (e.g., record.schema_conformance treats Body
// as a JSON object).
//
// Offset / Partition are surfaced for evidence: the handler uses
// them as forensic linkage in sample_violations descriptors per
// ADR-0026's evidence shape.
type Record struct {
	Partition int32
	Offset    int64
	Timestamp time.Time
	Body      []byte
}

// CheckSpec is the per-check descriptor passed to the
// CheckEvaluator. The runner does not interpret Mode / Source /
// Params; the evaluator dispatches on Kind and reads the rule's
// substrate descriptor and per-check params from this struct.
//
// v1 rules surface here with Mode = "set" and Source = nil
// (legacy callers configured the substrate via evaluator-level
// fields). v2 rules carry both per ADRs 0021 / 0023.
type CheckSpec struct {
	CheckID string
	Kind    string
	Mode    string         // "set" | "record" per ADR-0021; empty for legacy callers
	Source  *RuleSource    // per-rule substrate descriptor per ADR-0023
	Params  map[string]any // per-check params per ADR-0022; nil when absent
}

// RuleSource is the runner-package mirror of the parsed rule's
// source descriptor. Field set tracks dsl/spec.Source; the
// runner does not parse YAML, so this duplicates the shape to
// keep the runner package free of dsl/spec dependencies (per
// foundation 04's package-coupling discipline).
//
// Exactly one substrate-specific field group is populated;
// validation lives upstream (linter + parser).
type RuleSource struct {
	Type string // "bigquery" | "kafka"

	// BigQuery fields (mode=set).
	ProjectID       string
	DatasetID       string
	TableID         string
	PartitionColumn string

	// Kafka fields (mode=record).
	Topic         string
	ConsumerGroup string
	Window        *RuleWindow
}

// RuleWindow mirrors dsl/spec.Window. Duration / LatenessTolerance
// are lexical strings validated by the parser; the runner parses
// them into time.Duration when the record-mode runner consumes
// the rule.
type RuleWindow struct {
	Type              string
	Duration          string
	LatenessTolerance string
}

// Config configures a Runner. Required fields: Store,
// EngineVersion, RulesetVersion. Optional fields default to safe
// values (no-op precheck, no-op evaluator, uuid-based attempt
// IDs, time.Now clock, discarding logger, no-op publisher,
// no-op metrics).
type Config struct {
	Store          results.Store
	Precheck       EntityPrecheck
	Evaluator      CheckEvaluator
	EngineVersion  string
	RulesetVersion string
	AttemptID      AttemptIDFunc
	Now            func() time.Time
	Logger         *slog.Logger
	// Publisher is the alerting emission surface per ADR-0006.
	// Optional; nil → alerts.NoopPublisher (no alerts emitted).
	// The engine binary wires a real PubSubPublisher; tests
	// inject a capturing publisher to assert emission.
	Publisher alerts.Publisher
	// Metrics is the per-package RunnerMetrics handle set per
	// ADR-0055 §Clause 3 + §Clause 4. The engine binary wires
	// the Registry's RunnerMetrics here; tests use
	// metrics.NoopRunnerMetrics() (the zero value's nil handles
	// would nil-deref). All five handles are required when
	// non-zero — the runner does not check for nil at emission
	// time.
	Metrics metrics.RunnerMetrics
}

// Runner orchestrates one execution at a time. Multiple goroutines
// may share a single Runner instance — its fields are read-only
// after construction.
type Runner struct {
	store          results.Store
	precheck       EntityPrecheck
	evaluator      CheckEvaluator
	engineVersion  string
	rulesetVersion string
	attemptID      AttemptIDFunc
	now            func() time.Time
	logger         *slog.Logger
	publisher      alerts.Publisher
	metrics        metrics.RunnerMetrics
}

// New validates the Config and returns a Runner. Returns an error
// for missing required fields or an invalid EngineVersion (must
// parse as semver).
func New(cfg Config) (*Runner, error) {
	if cfg.Store == nil {
		return nil, errors.New("runner: Store is required")
	}
	if cfg.EngineVersion == "" {
		return nil, errors.New("runner: EngineVersion is required")
	}
	if _, err := semver.NewVersion(cfg.EngineVersion); err != nil {
		return nil, fmt.Errorf("runner: EngineVersion %q is not valid semver: %w", cfg.EngineVersion, err)
	}
	if cfg.RulesetVersion == "" {
		return nil, errors.New("runner: RulesetVersion is required")
	}

	precheck := cfg.Precheck
	if precheck == nil {
		precheck = NoopPrecheck{}
	}
	evaluator := cfg.Evaluator
	if evaluator == nil {
		evaluator = NoopEvaluator{}
	}
	attemptID := cfg.AttemptID
	if attemptID == nil {
		attemptID = DefaultAttemptID
	}
	clock := cfg.Now
	if clock == nil {
		clock = time.Now
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	publisher := cfg.Publisher
	if publisher == nil {
		publisher = alerts.NoopPublisher{}
	}
	runnerMetrics := cfg.Metrics
	if runnerMetrics.RunsTotal == nil {
		runnerMetrics = metrics.NoopRunnerMetrics()
	}

	return &Runner{
		store:          cfg.Store,
		precheck:       precheck,
		evaluator:      evaluator,
		engineVersion:  cfg.EngineVersion,
		rulesetVersion: cfg.RulesetVersion,
		attemptID:      attemptID,
		now:            clock,
		logger:         logger,
		publisher:      publisher,
		metrics:        runnerMetrics,
	}, nil
}

// Run executes one trigger end-to-end. Steps:
//
//  1. Validate trigger inputs (pipe-character ban per ADR-0002
//     CC2; trigger-source / supersedes coherence per ADR-0002
//     CC5).
//  2. Compute execution_id (ADR-0002 CC1).
//  3. Generate attempt_id (ADR-0003 CC4).
//  4. Run EntityPrecheck. If "source absent", write terminal
//     `error` row with no check rows (ADR-0007 CC8 / ADR-0004
//     CC2 branch 2) and return.
//  5. Write `running` transition row (ADR-0003 CC3).
//  6. For each check in trigger.Checks: invoke evaluator, write
//     per-check row (ADR-0003 CC7); continue on per-check error
//     per ADR-0004 CC4 (always-continue).
//  7. Compute terminal status via MapStatus per ADR-0004 CC2.
//  8. Write terminal row.
//  9. Return the terminal ExecutionRow.
//
// Returns the terminal ExecutionRow on success. Returns an error
// for input-validation failures and operational failures (Store
// errors, precheck errors). Per-check evaluator errors do not
// fail Run — they are recorded as ResultError per ADR-0004 CC1.
func (r *Runner) Run(ctx context.Context, trigger TriggerRequest) (*results.ExecutionRow, error) {
	if err := r.validateTrigger(trigger); err != nil {
		return nil, fmt.Errorf("trigger validation: %w", err)
	}

	// Per-call effective values. The trigger handler (W3-P4e)
	// sets these at acceptance so the persisted row carries the
	// same execution_id and attempt_id as the response DTO.
	// Phase-4c callers that omit them get the runner's
	// constructor-time defaults.
	if trigger.RulesetVersion == "" {
		trigger.RulesetVersion = r.rulesetVersion
	}

	executionID, err := Compute(
		trigger.RulesetVersion,
		trigger.Entity,
		trigger.WindowStart,
		trigger.WindowEnd,
		trigger.TriggerSource,
	)
	if err != nil {
		return nil, fmt.Errorf("compute execution_id: %w", err)
	}
	var attemptID string
	if trigger.AttemptID != nil {
		attemptID = *trigger.AttemptID
	} else {
		attemptID = r.attemptID()
	}
	startedAt := r.now().UTC()

	// ADR-0006 CC5: engine-side dedup is per-attempt state. The
	// deduper is discarded when Run returns; retries get a fresh
	// instance, so the consumer-side dedup is the only thing that
	// collapses retries to one user-visible alert per failing
	// check.
	dedup := alerts.NewAttemptDeduper()

	r.logger.Info("execution attempt starting",
		"execution_id", executionID,
		"attempt_id", attemptID,
		"entity", trigger.Entity,
		"trigger_source", string(trigger.TriggerSource),
		"adr_reference", "ADR-0002 CC1",
	)

	// Step 4 — pre-check entity-level validation per ADR-0007 CC8.
	present, err := r.precheck.SourceExists(ctx, trigger.Entity)
	if err != nil {
		// Operational failure inside the precheck itself. Write a
		// terminal error row with a distinct summary so operators
		// can tell this apart from "source absent".
		summary := fmt.Sprintf("pre-check operational failure: %v", err)
		return r.writePreCheckErrorRow(ctx, dedup, executionID, attemptID, startedAt, trigger, &summary)
	}
	if !present {
		summary := "pre-check entity-level: source not present (ADR-0007 CC8)"
		return r.writePreCheckErrorRow(ctx, dedup, executionID, attemptID, startedAt, trigger, &summary)
	}

	// Step 5 — write the running transition row (ADR-0003 CC3).
	runningRow := r.buildRunningRow(executionID, attemptID, startedAt, trigger)
	if err := r.store.WriteExecutionRow(ctx, runningRow); err != nil {
		return nil, fmt.Errorf("write running row: %w", err)
	}

	// Step 6 — evaluate each check; always continue per ADR-0004 CC4.
	checkResults := make([]results.CheckResult, 0, len(trigger.Checks))
	for _, spec := range trigger.Checks {
		// Honor context cancellation between checks so a long
		// pass can be interrupted by graceful shutdown.
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("execution canceled mid-pass: %w", ctx.Err())
		default:
		}

		checkStarted := r.now()
		eval, evalErr := r.evaluator.Evaluate(ctx, spec, trigger)
		checkElapsed := r.now().Sub(checkStarted).Seconds()
		if evalErr != nil {
			// ADR-0004 CC1: evaluator errors map to ResultError,
			// not a Run-level failure. The runner records the
			// error result and continues with sibling checks.
			eval = Evaluation{Result: results.ResultError}
			r.logger.Warn("check evaluator returned error; recording as ResultError",
				"execution_id", executionID,
				"attempt_id", attemptID,
				"check_id", spec.CheckID,
				"error", evalErr.Error(),
				"adr_reference", "ADR-0004 CC1",
			)
		}
		checkResults = append(checkResults, eval.Result)

		row := results.CheckResultRow{
			ExecutionID:         executionID,
			AttemptID:           attemptID,
			CheckID:             spec.CheckID,
			Result:              eval.Result,
			ExecutedAt:          r.now().UTC(),
			EngineVersion:       r.engineVersion,
			EvidenceSummary:     eval.EvidenceSummary,
			SampleViolatingRows: eval.SampleViolatingRows,
		}
		if err := r.store.WriteCheckResultRow(ctx, row); err != nil {
			return nil, fmt.Errorf("write check_result row for %s: %w", spec.CheckID, err)
		}

		// ADR-0055 §Clause 4: emit dq_checks_evaluated_total +
		// dq_check_duration_seconds + dq_bytes_scanned after the
		// check_result row is durable. Mode comes from the
		// spec; falls back to triggerMode for v1 callers that
		// leave it empty.
		checkMode := spec.Mode
		if checkMode == "" {
			checkMode = string(triggerMode(trigger))
		}
		r.metrics.ChecksEvaluatedTotal.WithLabelValues(trigger.Entity, spec.CheckID, string(eval.Result), checkMode).Inc()
		r.metrics.CheckDurationSeconds.WithLabelValues(trigger.Entity, spec.CheckID, checkMode).Observe(checkElapsed)
		r.metrics.BytesScanned.WithLabelValues(trigger.Entity, spec.CheckID).Set(bytesScannedOrZero(eval.EvidenceSummary))

		// ADR-0006 CC4: emit check-level event after the row is
		// durable. Publish failures are warning-logged, not
		// returned, so an alerting-substrate outage cannot block
		// execution finalization.
		r.emitCheckEvent(ctx, dedup, executionID, attemptID, trigger.Entity, spec.CheckID, eval.Result, row.ExecutedAt)
	}

	// Step 7 — terminal status per ADR-0004 CC2.
	terminalStatus, err := MapStatus(checkResults)
	if err != nil {
		// Phase 4c: a trigger with zero checks is a configuration
		// error at the trigger level. Map to terminal `error` with
		// a clear summary.
		summary := "trigger contained zero checks (ADR-0004 CC2 branch 2)"
		return r.writeTerminalRow(ctx, dedup, executionID, attemptID, startedAt, trigger, results.StatusError, &summary)
	}

	// Step 8 — terminal row. ADR-0003 CC3 commits that
	// error_summary is populated when status is failed or error;
	// for status=success it remains nil.
	terminalSummary := terminalErrorSummary(terminalStatus, checkResults)
	return r.writeTerminalRow(ctx, dedup, executionID, attemptID, startedAt, trigger, terminalStatus, terminalSummary)
}

// terminalErrorSummary returns a brief one-line summary suitable
// for dq_executions.error_summary when the terminal status is
// failed or error per ADR-0003 CC3. Detailed per-check
// diagnostics live in dq_check_results; this is the execution-
// level forensic signal. Returns nil for status=success.
func terminalErrorSummary(status results.ExecutionStatus, checkResults []results.CheckResult) *string {
	switch status {
	case results.StatusFailed:
		nonPass := 0
		for _, r := range checkResults {
			if r != results.ResultPass {
				nonPass++
			}
		}
		s := fmt.Sprintf("%d of %d checks did not pass (see dq_check_results)", nonPass, len(checkResults))
		return &s
	case results.StatusError:
		s := fmt.Sprintf("all %d checks errored (see dq_check_results)", len(checkResults))
		return &s
	default:
		return nil
	}
}

// validateTrigger checks the per-trigger invariants from ADR-0002.
func (r *Runner) validateTrigger(trigger TriggerRequest) error {
	if trigger.Entity == "" {
		return errors.New("entity is required")
	}
	if trigger.TriggerSource == "" {
		return errors.New("trigger_source is required")
	}
	switch trigger.TriggerSource {
	case results.TriggerScheduler, results.TriggerManual, results.TriggerOperatorRerun:
		// ok
	default:
		return fmt.Errorf("trigger_source %q is not a recognized enum value", trigger.TriggerSource)
	}

	// ADR-0002 CC5: operator-rerun requires SupersedesExecutionID;
	// scheduler / manual forbid it. The API layer enforces the
	// mapping between endpoint and trigger_source; the runner
	// double-checks the coherence here as belt-and-suspenders.
	if trigger.TriggerSource == results.TriggerOperatorRerun {
		if trigger.SupersedesExecutionID == nil || *trigger.SupersedesExecutionID == "" {
			return errors.New("operator-rerun requires SupersedesExecutionID (ADR-0002 CC5)")
		}
	} else {
		if trigger.SupersedesExecutionID != nil {
			return fmt.Errorf("%s trigger must not set SupersedesExecutionID (ADR-0002 CC5)", trigger.TriggerSource)
		}
	}

	if !trigger.WindowEnd.After(trigger.WindowStart) {
		return errors.New("window_end must be strictly after window_start")
	}
	return nil
}

// buildRunningRow constructs the running transition row per
// ADR-0003 CC3. started_at and completed_at are nil per CC3
// nullable semantics (terminal rows set them); recorded_at is
// the run-start timestamp.
func (r *Runner) buildRunningRow(executionID, attemptID string, startedAt time.Time, trigger TriggerRequest) results.ExecutionRow {
	return results.ExecutionRow{
		ExecutionID:           executionID,
		AttemptID:             attemptID,
		RecordedAt:            startedAt,
		Status:                results.StatusRunning,
		Mode:                  triggerMode(trigger),
		EngineVersion:         r.engineVersion,
		RulesetVersion:        trigger.RulesetVersion,
		Entity:                trigger.Entity,
		TriggerSource:         trigger.TriggerSource,
		WindowStart:           trigger.WindowStart, // ADR-0041 + B2-27
		WindowEnd:             trigger.WindowEnd,
		StartedAt:             nil, // ADR-0003 CC3: nullable on running row
		CompletedAt:           nil, // ADR-0003 CC3: nullable on running row
		ErrorSummary:          nil,
		SupersedesExecutionID: trigger.SupersedesExecutionID,
	}
}

// triggerMode resolves the execution's mode from the trigger's
// checks. v2 rules carry mode per check (post-ADR-0021); the
// runner promotes the first check's mode to the execution row's
// mode column. Per ADR-0022 cross-check #4 (kind prefix matches
// rule mode) all checks in one rule share a mode, so the first
// check is authoritative.
//
// Backfill default: an empty per-check Mode falls through to
// "set" — matches the ADR-0021 backfill contract for pre-Wave-S
// rows. The dsl/spec parser populates Mode for v1 rules at
// ToCheckSpecs translation time, so this default is defensive.
func triggerMode(trigger TriggerRequest) results.Mode {
	for _, c := range trigger.Checks {
		if c.Mode != "" {
			return results.Mode(c.Mode)
		}
	}
	return results.ModeSet
}

// writePreCheckErrorRow writes the single terminal `error` row
// for the ADR-0007 CC8 / ADR-0004 CC2 branch 2 case — no `running`
// row, no check rows. After the row is durable, emits the
// operational alert per ADR-0006 CC7.
func (r *Runner) writePreCheckErrorRow(ctx context.Context, dedup *alerts.AttemptDeduper, executionID, attemptID string, startedAt time.Time, trigger TriggerRequest, summary *string) (*results.ExecutionRow, error) {
	now := r.now().UTC()
	row := results.ExecutionRow{
		ExecutionID:           executionID,
		AttemptID:             attemptID,
		RecordedAt:            now,
		Status:                results.StatusError,
		Mode:                  triggerMode(trigger),
		EngineVersion:         r.engineVersion,
		RulesetVersion:        trigger.RulesetVersion,
		Entity:                trigger.Entity,
		TriggerSource:         trigger.TriggerSource,
		WindowStart:           trigger.WindowStart, // ADR-0041 + B2-27
		WindowEnd:             trigger.WindowEnd,
		StartedAt:             &startedAt,
		CompletedAt:           &now,
		ErrorSummary:          summary,
		SupersedesExecutionID: trigger.SupersedesExecutionID,
	}
	if err := r.store.WriteExecutionRow(ctx, row); err != nil {
		return nil, fmt.Errorf("write pre-check error row: %w", err)
	}
	r.logger.Info("execution finalized via pre-check error",
		"execution_id", executionID,
		"attempt_id", attemptID,
		"entity", trigger.Entity,
		"adr_reference", "ADR-0007 CC8",
	)
	r.emitRunMetrics(trigger, results.StatusError, startedAt, now)
	r.emitExecutionEvent(ctx, dedup, executionID, attemptID, trigger.Entity, results.StatusError, now, summary)
	return &row, nil
}

// writeTerminalRow writes the terminal transition row after
// per-check evaluation completes. terminalStatus comes from
// MapStatus; errorSummary is nil for success and non-nil for
// failed/error paths if known. After the row is durable, emits
// the execution-level alert per ADR-0006 CC7 (no-op for
// status=success).
func (r *Runner) writeTerminalRow(ctx context.Context, dedup *alerts.AttemptDeduper, executionID, attemptID string, startedAt time.Time, trigger TriggerRequest, terminalStatus results.ExecutionStatus, errorSummary *string) (*results.ExecutionRow, error) {
	now := r.now().UTC()
	row := results.ExecutionRow{
		ExecutionID:           executionID,
		AttemptID:             attemptID,
		RecordedAt:            now,
		Status:                terminalStatus,
		Mode:                  triggerMode(trigger),
		EngineVersion:         r.engineVersion,
		RulesetVersion:        trigger.RulesetVersion,
		Entity:                trigger.Entity,
		TriggerSource:         trigger.TriggerSource,
		WindowStart:           trigger.WindowStart, // ADR-0041 + B2-27
		WindowEnd:             trigger.WindowEnd,
		StartedAt:             &startedAt,
		CompletedAt:           &now,
		ErrorSummary:          errorSummary,
		SupersedesExecutionID: trigger.SupersedesExecutionID,
	}
	if err := r.store.WriteExecutionRow(ctx, row); err != nil {
		return nil, fmt.Errorf("write terminal row: %w", err)
	}
	r.logger.Info("execution finalized",
		"execution_id", executionID,
		"attempt_id", attemptID,
		"entity", trigger.Entity,
		"status", string(terminalStatus),
		"adr_reference", "ADR-0004 CC2",
	)
	r.emitRunMetrics(trigger, terminalStatus, startedAt, now)
	r.emitExecutionEvent(ctx, dedup, executionID, attemptID, trigger.Entity, terminalStatus, now, errorSummary)
	return &row, nil
}

// emitRunMetrics increments dq_runs_total and observes
// dq_run_duration_seconds per ADR-0055 §Clause 4. Called after
// every durable terminal-row write (both writeTerminalRow and
// writePreCheckErrorRow). Emit-after-write is load-bearing —
// a Store-write failure must not produce a metric without its
// backing row.
func (r *Runner) emitRunMetrics(trigger TriggerRequest, status results.ExecutionStatus, startedAt, completedAt time.Time) {
	mode := string(triggerMode(trigger))
	durSec := completedAt.Sub(startedAt).Seconds()
	r.metrics.RunsTotal.WithLabelValues(trigger.Entity, string(status), string(trigger.TriggerSource), mode).Inc()
	r.metrics.RunDurationSeconds.WithLabelValues(trigger.Entity, string(status), mode).Observe(durSec)
}

// bytesScannedOrZero extracts the evidence_summary.bytes_scanned
// sub-field as a float64, falling back to zero per ADR-0055
// §Clause 4 OQ-6 resolution. Zero preserves the time-series
// continuity that "skip emission" would break; the absent-vs-real-
// zero distinction is left to the operator's panel range.
//
// Accepts the numeric types JSON-decoded evidence summaries
// typically carry (float64 from encoding/json; int / int64 from
// direct Go construction). Non-numeric values resolve to zero.
func bytesScannedOrZero(evidence map[string]any) float64 {
	v, ok := evidence["bytes_scanned"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	default:
		return 0
	}
}

// emitCheckEvent constructs and publishes a check-level alert
// event per ADR-0006 §4. MapCategory filters out passing checks
// (no emission). The per-attempt deduper guards against literal
// duplicate emits. Publish failures are warning-logged, not
// returned: alerting-substrate outages must not block execution
// finalization (ADR-0006 CC5: engine-side dedup is the belt,
// consumer-side dedup is the suspenders; alerting is best-effort
// out-of-band signal, not part of the execution's durability
// contract).
func (r *Runner) emitCheckEvent(ctx context.Context, dedup *alerts.AttemptDeduper, executionID, attemptID, entity, checkID string, result results.CheckResult, recordedAt time.Time) {
	category, emit := alerts.MapCategory(alerts.SourceRunner, &result, nil)
	if !emit {
		return
	}
	event := alerts.Event{
		ExecutionID: &executionID,
		AttemptID:   &attemptID,
		Entity:      entity,
		CheckID:     &checkID,
		Category:    category,
		EventSource: alerts.SourceRunner,
		Result:      &result,
		RecordedAt:  recordedAt,
	}
	if !dedup.ShouldPublish(event) {
		return
	}
	if err := r.publisher.Publish(ctx, event); err != nil {
		r.logger.Warn("alert publish failed (check-level)",
			"execution_id", executionID,
			"attempt_id", attemptID,
			"check_id", checkID,
			"category", string(category),
			"error", err.Error(),
			"adr_reference", "ADR-0006 CC4",
		)
	}
}

// emitExecutionEvent constructs and publishes an execution-level
// alert event per ADR-0006 §4. MapCategory filters out
// status=success (no emission). The per-attempt deduper guards
// against literal duplicate emits. Publish failures are
// warning-logged, not returned (see emitCheckEvent).
func (r *Runner) emitExecutionEvent(ctx context.Context, dedup *alerts.AttemptDeduper, executionID, attemptID, entity string, status results.ExecutionStatus, recordedAt time.Time, errorSummary *string) {
	category, emit := alerts.MapCategory(alerts.SourceRunner, nil, &status)
	if !emit {
		return
	}
	event := alerts.Event{
		ExecutionID:  &executionID,
		AttemptID:    &attemptID,
		Entity:       entity,
		Category:     category,
		EventSource:  alerts.SourceRunner,
		Status:       &status,
		RecordedAt:   recordedAt,
		ErrorSummary: errorSummary,
	}
	if !dedup.ShouldPublish(event) {
		return
	}
	if err := r.publisher.Publish(ctx, event); err != nil {
		r.logger.Warn("alert publish failed (execution-level)",
			"execution_id", executionID,
			"attempt_id", attemptID,
			"category", string(category),
			"status", string(status),
			"error", err.Error(),
			"adr_reference", "ADR-0006 CC4",
		)
	}
}
