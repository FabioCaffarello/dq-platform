// path: engine/internal/eval/evaluator.go

package eval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"cloud.google.com/go/bigquery"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// Kind constants for the v2 check kinds the engine knows about.
// The catalog at engine/internal/dsl/catalog/v1.yaml is the
// authoritative list; the dispatcher invariant at engine boot
// (cmd/dq-engine/main.go) cross-checks the registry against the
// catalog.
const (
	KindSetRowCountPositive      = "set.row_count_positive"
	KindSetRowCountWithinBaseline = "set.row_count_within_baseline"
	KindRecordSchemaConformance  = "record.schema_conformance"
)

// Handler is the per-kind evaluation function the registry
// dispatches into. The signature mirrors CheckEvaluator.Evaluate
// so the same map-on-kind dispatch serves both set-mode handlers
// (e.g. set.row_count_positive) and record-mode handlers
// (e.g. record.schema_conformance, the production handler
// shipped in Wave-S sub-slice β).
type Handler func(ctx context.Context, evalCtx *Evaluator, spec runner.CheckSpec, trigger runner.TriggerRequest) (runner.Evaluation, error)

// Config configures an Evaluator. Client is required; the other
// fields default to safe values.
type Config struct {
	// Client is the BigQuery client used to execute set-mode
	// check queries. Required for set-mode handlers; record-mode
	// handlers ignore it.
	Client *bigquery.Client

	// Logger is the structured logger. Optional; defaults to
	// slog.Default().
	Logger *slog.Logger

	// ResultsProject + ResultsDataset locate `dq_check_results`
	// + `dq_executions` for the baseline framework (ADR-0032).
	// Optional at construction time; baselined kinds fail loud
	// at evaluation time when these are empty and a baseline
	// query is needed. Set from EnvConfig.BigQueryProject /
	// .BigQueryDataset in main.go.
	ResultsProject string
	ResultsDataset string

	// ResultsRetention is the per-env partition-expiration
	// duration (ADR-0031). ComputeBaseline caps the effective
	// reference window at `min(declared, ResultsRetention)` so
	// a rule declaring a long reference window doesn't silently
	// scan missing partitions on a short-retention env.
	ResultsRetention time.Duration
}

// Evaluator dispatches CheckSpecs to per-kind handlers. The
// registry is populated at construction time with every kind
// the catalog declares; the dispatcher startup invariant
// (cmd/dq-engine) verifies catalog ⊆ registry.
//
// Satisfies runner.CheckEvaluator via the Evaluate method below.
type Evaluator struct {
	client           *bigquery.Client
	logger           *slog.Logger
	handlers         map[string]Handler
	resultsProject   string
	resultsDataset   string
	resultsRetention time.Duration
}

// New validates the Config and returns an Evaluator with the
// default handler registry pre-populated. Returns an error when
// Client is missing.
//
// Handlers registered at construction time:
//
//   - set.row_count_positive — set-mode implementation
//     (evaluateRowCountPositive in row_count_positive.go).
//   - record.schema_conformance — record-mode implementation
//     (recordSchemaConformanceHandler in
//     record_schema_conformance.go) per ADR-0026: compiles each
//     rule's JSON Schema, validates per-window records delivered
//     via TriggerRequest.Records, and aggregates per-record
//     outcomes into one CheckResult using the threshold function.
//
// Registering every catalog kind at construction time satisfies
// the dispatcher startup invariant per ADR-0022 §C-B0S2.3: the
// engine binary's verifyHandlerRegistryAgainstCatalog
// (cmd/dq-engine/main.go) cross-checks that every kind declared
// in the embedded catalog has a registered handler here and
// fails the boot otherwise.
func New(cfg Config) (*Evaluator, error) {
	if cfg.Client == nil {
		return nil, errors.New("eval: Client is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	e := &Evaluator{
		client:           cfg.Client,
		logger:           logger,
		handlers:         map[string]Handler{},
		resultsProject:   cfg.ResultsProject,
		resultsDataset:   cfg.ResultsDataset,
		resultsRetention: cfg.ResultsRetention,
	}
	e.Register(KindSetRowCountPositive, setRowCountPositiveHandler)
	e.Register(KindSetRowCountWithinBaseline, setRowCountWithinBaselineHandler)
	e.Register(KindRecordSchemaConformance, recordSchemaConformanceHandler)
	return e, nil
}

// Register installs a handler for the given kind. Subsequent
// Register calls for the same kind overwrite the prior entry;
// tests use this to inject doubles. Production wiring registers
// every kind exactly once at New time.
func (e *Evaluator) Register(kind string, h Handler) {
	e.handlers[kind] = h
}

// RegisteredKinds returns the sorted set of kinds the evaluator
// has handlers for. Used by the dispatcher startup invariant
// (cmd/dq-engine) to cross-check against the catalog.
func (e *Evaluator) RegisteredKinds() []string {
	kinds := make([]string, 0, len(e.handlers))
	for k := range e.handlers {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}

// Evaluate dispatches on spec.Kind via the registry. Unrecognized
// kinds return ResultError per ADR-0004 CC1.
func (e *Evaluator) Evaluate(
	ctx context.Context,
	spec runner.CheckSpec,
	trigger runner.TriggerRequest,
) (runner.Evaluation, error) {
	handler, ok := e.handlers[spec.Kind]
	if !ok {
		err := fmt.Errorf("unsupported check kind %q (registered kinds: %v)",
			spec.Kind, e.RegisteredKinds())
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
	return handler(ctx, e, spec, trigger)
}

// setRowCountPositiveHandler is the registry entry for the
// set.row_count_positive kind. Thin adapter onto the per-kind
// method on Evaluator so the registry holds a uniform Handler
// signature.
func setRowCountPositiveHandler(ctx context.Context, e *Evaluator, spec runner.CheckSpec, trigger runner.TriggerRequest) (runner.Evaluation, error) {
	return e.evaluateRowCountPositive(ctx, spec, trigger)
}

// recordSchemaConformanceHandler is registered above and
// implemented in record_schema_conformance.go. It is the
// production handler shipped in Wave-S sub-slice β: it consumes
// per-window records from the record-mode runner via
// TriggerRequest.Records and applies the threshold aggregation
// per ADR-0026.
