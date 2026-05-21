// path: engine/internal/orphan/orphan.go

// Package orphan implements orphan-run detection per ADR-0007 CC11.
//
// The orphan detector is a periodic engine task that scans
// dq_executions for `running` rows whose started_at is older than
// a configurable threshold and finalizes them to `aborted` by
// writing a follow-up row carrying the detector's own
// engine_version. This is the mechanism that closes the ADR-0007
// CC10 halt conditions for engine restart / OOM / mid-execution
// crash, where the abandoned engine cannot write its own death
// certificate.
//
// The follow-up row's engine_version is the **detector's**
// version, not the abandoned engine's. Different engine_version
// values within a single attempt's lifecycle is the expected
// pattern for orphan-finalization (ADR-0007 CC11 + ADR-0002
// CC14): forensic queries on the base table see both versions;
// the canonical view returns the detector's as the effective
// evaluator.
//
// This package exposes a RunOnce function the engine binary
// invokes on its configured cadence (the cadence value is a B1
// parameter per ADR-0007 CC15; the engine binary supplies it).
// The periodic loop itself lives in the engine binary (W3-P4c).
//
// # Single-instance constraint
//
// The Detector is designed for **single-instance** operation per
// environment. If two Detector instances ran concurrently against
// the same Scanner — for example, two engine replicas in an HA
// deployment — both could pick up the same candidate between
// scan and write, producing duplicate `aborted` follow-up rows.
// The canonical view per ADR-0003 CC2 still returns a single
// row (latest recorded_at), so external consumers are not
// misled, but the base table accumulates duplicate orphan-
// finalizations and forensic queries see the same execution
// twice.
//
// The engine binary (W3-P4c) is responsible for ensuring only
// one Detector loop runs per Store at a time. Today the engine
// is single-instance per environment; if a future ADR introduces
// HA / multiple replicas, this package will need leader-election
// or per-candidate compare-and-set guards to preserve the
// orphan-finalization correctness boundary.
package orphan

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"dq-platform/engine/internal/results"
)

// Scanner is the narrow surface the orphan detector consumes from
// the result-write layer. results.Store satisfies this interface;
// tests inject a mock.
type Scanner interface {
	ListRunningOlderThan(ctx context.Context, before time.Time) ([]results.ExecutionRow, error)
	WriteExecutionRow(ctx context.Context, row results.ExecutionRow) error
}

// AbandonmentSummary is the error_summary written into the
// follow-up aborted row. ADR-0007 CC11 leaves the exact wording
// to scaffolding; this is the chosen value.
const AbandonmentSummary = "orphan-run detection: engine instance abandoned this execution"

// Config configures a Detector. EngineVersion and Threshold are
// required; Logger and Now are optional (defaults: discarding
// logger, time.Now).
type Config struct {
	// EngineVersion is the detector's own version string, written
	// into every follow-up aborted row's engine_version field per
	// ADR-0007 CC11.
	EngineVersion string

	// Threshold is the age beyond which a `running` row is
	// considered abandoned. The threshold must exceed the maximum
	// expected execution duration in the environment; ADR-0007
	// CC15 lists this as a B1 parameter, supplied by the engine
	// binary.
	Threshold time.Duration

	// Logger is used for the structured per-finalization log
	// emission per ADR-0007 CC14. Optional; nil → discarding.
	Logger *slog.Logger

	// Now is an injectable clock for tests. Optional; nil →
	// time.Now.
	Now func() time.Time
}

// Detector finalizes abandoned `running` executions per ADR-0007
// CC11. Construct with New; invoke RunOnce on the engine binary's
// configured cadence.
type Detector struct {
	scanner       Scanner
	engineVersion string
	threshold     time.Duration
	logger        *slog.Logger
	now           func() time.Time
}

// New constructs a Detector from a Scanner and Config.
func New(scanner Scanner, cfg Config) (*Detector, error) {
	if scanner == nil {
		return nil, errors.New("orphan: scanner is required")
	}
	if cfg.EngineVersion == "" {
		return nil, errors.New("orphan: EngineVersion is required")
	}
	if cfg.Threshold <= 0 {
		return nil, fmt.Errorf("orphan: Threshold must be positive; got %v", cfg.Threshold)
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	clock := cfg.Now
	if clock == nil {
		clock = time.Now
	}
	return &Detector{
		scanner:       scanner,
		engineVersion: cfg.EngineVersion,
		threshold:     cfg.Threshold,
		logger:        logger,
		now:           clock,
	}, nil
}

// RunOnce performs one scan pass. For each `running` row whose
// started_at is older than now-threshold, RunOnce writes a follow-
// up `aborted` row carrying the detector's engine_version.
// Returns the number of rows finalized and a slice of per-row
// errors (write failures do not abort the pass; sibling rows are
// still attempted).
//
// An err return value (third return) signals an operational
// failure (e.g., the scan query itself failed); the finalized
// count and errs are zero/nil in that case. A nil err with a
// non-empty errs slice means the pass completed but some
// finalizations failed; the caller can retry on the next cadence
// (the candidate row is still `running` in the canonical view).
func (d *Detector) RunOnce(ctx context.Context) (finalized int, errs []error, err error) {
	cutoff := d.now().UTC().Add(-d.threshold)

	candidates, err := d.scanner.ListRunningOlderThan(ctx, cutoff)
	if err != nil {
		return 0, nil, fmt.Errorf("scan for orphan candidates: %w", err)
	}

	for _, candidate := range candidates {
		// Honor context cancellation between candidates so the
		// engine binary's graceful-shutdown path can interrupt a
		// long-running pass cleanly. Returns whatever was
		// finalized so far plus the cancellation error.
		select {
		case <-ctx.Done():
			return finalized, errs, ctx.Err()
		default:
		}

		followup := d.buildFollowupRow(candidate)
		if writeErr := d.scanner.WriteExecutionRow(ctx, followup); writeErr != nil {
			errs = append(errs, fmt.Errorf("finalize %s/%s: %w",
				candidate.ExecutionID, candidate.AttemptID, writeErr))
			continue
		}
		finalized++
		d.logger.Info("orphan-run finalized to aborted",
			"execution_id", candidate.ExecutionID,
			"attempt_id", candidate.AttemptID,
			"entity", candidate.Entity,
			"abandoned_started_at", candidate.StartedAt,
			"detector_engine_version", d.engineVersion,
			"adr_reference", "ADR-0007 CC11",
		)
	}
	return finalized, errs, nil
}

// buildFollowupRow constructs the aborted follow-up row for one
// orphan candidate per ADR-0007 CC11. Visibility of the
// detector's engine_version vs the abandoned engine's is the
// load-bearing semantic — preserved by setting
// EngineVersion = d.engineVersion (not candidate.EngineVersion)
// and StartedAt = candidate.StartedAt (for forensic linkage).
func (d *Detector) buildFollowupRow(candidate results.ExecutionRow) results.ExecutionRow {
	now := d.now().UTC()
	summary := AbandonmentSummary
	return results.ExecutionRow{
		ExecutionID:    candidate.ExecutionID,
		AttemptID:      candidate.AttemptID,
		RecordedAt:     now,
		Status:         results.StatusAborted,
		EngineVersion:  d.engineVersion,
		RulesetVersion: candidate.RulesetVersion,
		Entity:         candidate.Entity,
		TriggerSource:  candidate.TriggerSource,
		StartedAt:      candidate.StartedAt,
		CompletedAt:    &now,
		ErrorSummary:   &summary,
		SupersedesExecutionID: candidate.SupersedesExecutionID,
	}
}
