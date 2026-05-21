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
// # Observability emission (ADR-0007 CC14)
//
// Phase 4c emits the log signal via slog.Info / slog.Warn at
// every failure path (pre-check absent, evaluator error,
// terminal finalization). The metric and span signals
// committed by ADR-0007 CC14 are deferred to a Phase-4c
// follow-up that wires an otelslog-style slog handler mirroring
// attributes to OpenTelemetry counters and spans. The same
// handler will pick up emissions from the loader (W3-P4a) and
// orphan detector (W3-P4d) without source changes here. This
// gap follows the honest-gap pattern set by B1-11 and the
// ADR-0010 lazy-view Partial row: the contract surface is
// committed; the underlying signals land additively.
package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/Masterminds/semver/v3"

	"dq-platform/engine/internal/results"
)

// TriggerRequest is the input to Runner.Run. The five identity
// fields (Entity, WindowStart, WindowEnd, TriggerSource, plus the
// engine's RulesetVersion) are the inputs to the execution_id
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
}

// CheckSpec is the per-check descriptor passed to the
// CheckEvaluator. Kind is the discriminator for the DSL grammar;
// Phase 4c does not interpret it.
type CheckSpec struct {
	CheckID string
	Kind    string
}

// Config configures a Runner. Required fields: Store,
// EngineVersion, RulesetVersion. Optional fields default to safe
// values (no-op precheck, no-op evaluator, uuid-based attempt
// IDs, time.Now clock, discarding logger).
type Config struct {
	Store          results.Store
	Precheck       EntityPrecheck
	Evaluator      CheckEvaluator
	EngineVersion  string
	RulesetVersion string
	AttemptID      AttemptIDFunc
	Now            func() time.Time
	Logger         *slog.Logger
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

	return &Runner{
		store:          cfg.Store,
		precheck:       precheck,
		evaluator:      evaluator,
		engineVersion:  cfg.EngineVersion,
		rulesetVersion: cfg.RulesetVersion,
		attemptID:      attemptID,
		now:            clock,
		logger:         logger,
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

	executionID, err := Compute(
		r.rulesetVersion,
		trigger.Entity,
		trigger.WindowStart,
		trigger.WindowEnd,
		trigger.TriggerSource,
	)
	if err != nil {
		return nil, fmt.Errorf("compute execution_id: %w", err)
	}
	attemptID := r.attemptID()
	startedAt := r.now().UTC()

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
		return r.writePreCheckErrorRow(ctx, executionID, attemptID, startedAt, trigger, &summary)
	}
	if !present {
		summary := "pre-check entity-level: source not present (ADR-0007 CC8)"
		return r.writePreCheckErrorRow(ctx, executionID, attemptID, startedAt, trigger, &summary)
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

		eval, evalErr := r.evaluator.Evaluate(ctx, spec, trigger)
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
	}

	// Step 7 — terminal status per ADR-0004 CC2.
	terminalStatus, err := MapStatus(checkResults)
	if err != nil {
		// Phase 4c: a trigger with zero checks is a configuration
		// error at the trigger level. Map to terminal `error` with
		// a clear summary.
		summary := "trigger contained zero checks (ADR-0004 CC2 branch 2)"
		return r.writeTerminalRow(ctx, executionID, attemptID, startedAt, trigger, results.StatusError, &summary)
	}

	// Step 8 — terminal row. ADR-0003 CC3 commits that
	// error_summary is populated when status is failed or error;
	// for status=success it remains nil.
	terminalSummary := terminalErrorSummary(terminalStatus, checkResults)
	return r.writeTerminalRow(ctx, executionID, attemptID, startedAt, trigger, terminalStatus, terminalSummary)
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
		EngineVersion:         r.engineVersion,
		RulesetVersion:        r.rulesetVersion,
		Entity:                trigger.Entity,
		TriggerSource:         trigger.TriggerSource,
		StartedAt:             nil, // ADR-0003 CC3: nullable on running row
		CompletedAt:           nil, // ADR-0003 CC3: nullable on running row
		ErrorSummary:          nil,
		SupersedesExecutionID: trigger.SupersedesExecutionID,
	}
}

// writePreCheckErrorRow writes the single terminal `error` row
// for the ADR-0007 CC8 / ADR-0004 CC2 branch 2 case — no `running`
// row, no check rows.
func (r *Runner) writePreCheckErrorRow(ctx context.Context, executionID, attemptID string, startedAt time.Time, trigger TriggerRequest, summary *string) (*results.ExecutionRow, error) {
	now := r.now().UTC()
	row := results.ExecutionRow{
		ExecutionID:           executionID,
		AttemptID:             attemptID,
		RecordedAt:            now,
		Status:                results.StatusError,
		EngineVersion:         r.engineVersion,
		RulesetVersion:        r.rulesetVersion,
		Entity:                trigger.Entity,
		TriggerSource:         trigger.TriggerSource,
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
	return &row, nil
}

// writeTerminalRow writes the terminal transition row after
// per-check evaluation completes. terminalStatus comes from
// MapStatus; errorSummary is nil for success and non-nil for
// failed/error paths if known.
func (r *Runner) writeTerminalRow(ctx context.Context, executionID, attemptID string, startedAt time.Time, trigger TriggerRequest, terminalStatus results.ExecutionStatus, errorSummary *string) (*results.ExecutionRow, error) {
	now := r.now().UTC()
	row := results.ExecutionRow{
		ExecutionID:           executionID,
		AttemptID:             attemptID,
		RecordedAt:            now,
		Status:                terminalStatus,
		EngineVersion:         r.engineVersion,
		RulesetVersion:        r.rulesetVersion,
		Entity:                trigger.Entity,
		TriggerSource:         trigger.TriggerSource,
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
	return &row, nil
}
