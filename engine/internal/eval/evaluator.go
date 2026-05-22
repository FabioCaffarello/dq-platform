// path: engine/internal/eval/evaluator.go

package eval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"cloud.google.com/go/bigquery"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// KindRowCountPositive is the first check kind shipped by P6c.
// See row_count_positive.go for the evaluation flow.
const KindRowCountPositive = "row_count_positive"

// Config configures an Evaluator. Client is required; the other
// fields default to safe values.
type Config struct {
	// Client is the BigQuery client used to execute check
	// queries. Required.
	Client *bigquery.Client

	// SourceProject is the BigQuery project that owns the
	// data-source tables checks read from. Optional; defaults to
	// Client.Project().
	SourceProject string

	// SourceDataset is the BigQuery dataset that owns the
	// data-source tables. Optional; when empty the evaluator
	// returns ResultError on any data-plane check with a
	// "source dataset not configured" diagnostic. Operators see
	// this as an operational failure per ADR-0004 CC1, not a
	// data-quality fail.
	SourceDataset string

	// Logger is the structured logger. Optional; defaults to
	// slog.Default() (mirrors the api package's logger-default
	// posture).
	Logger *slog.Logger
}

// Evaluator dispatches CheckSpecs to per-kind BigQuery
// evaluations. It satisfies runner.CheckEvaluator via duck typing:
// the engine binary passes a *Evaluator into runner.Config.Evaluator
// in place of the Phase-4c NoopEvaluator.
type Evaluator struct {
	client        *bigquery.Client
	sourceProject string
	sourceDataset string
	logger        *slog.Logger
}

// New validates the Config and returns an Evaluator. Returns an
// error when Client is missing.
func New(cfg Config) (*Evaluator, error) {
	if cfg.Client == nil {
		return nil, errors.New("eval: Client is required")
	}
	e := &Evaluator{
		client:        cfg.Client,
		sourceProject: cfg.SourceProject,
		sourceDataset: cfg.SourceDataset,
		logger:        cfg.Logger,
	}
	if e.sourceProject == "" {
		e.sourceProject = cfg.Client.Project()
	}
	if e.logger == nil {
		e.logger = slog.Default()
	}
	return e, nil
}

// Evaluate dispatches on spec.Kind. Recognized kinds run their
// per-kind evaluation; unrecognized kinds return ResultError per
// ADR-0004 CC1 (evaluator errors map to ResultError; the runner
// applies the execution-status mapping per ADR-0004 CC2).
//
// Returning a non-nil error alongside ResultError preserves the
// runner's existing error-handling contract: the runner logs the
// error at WARN level and records a check_results row with
// ResultError.
func (e *Evaluator) Evaluate(
	ctx context.Context,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
) (runner.Evaluation, error) {
	switch spec.Kind {
	case KindRowCountPositive:
		return e.evaluateRowCountPositive(ctx, spec, trigger)
	default:
		err := fmt.Errorf(
			"unsupported check kind %q (P6c ships only %q; further kinds land additively)",
			spec.Kind, KindRowCountPositive,
		)
		e.logger.Warn("evaluator received unsupported check kind",
			"kind", spec.Kind,
			"check_id", spec.CheckID,
			"entity", trigger.Entity,
			"adr_reference", "ADR-0004 CC1",
		)
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "unsupported_kind",
			},
		}, err
	}
}

// errSourceDatasetMissing is returned by per-kind evaluations
// that require a configured SourceDataset when none is set. It is
// surfaced as ResultError per ADR-0004 CC1 (operational failure).
var errSourceDatasetMissing = errors.New(
	"eval: source dataset not configured (set DQ_SOURCE_DATASET to enable data-plane checks)",
)

