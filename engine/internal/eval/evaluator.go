// path: engine/internal/eval/evaluator.go

package eval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"

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
	KindSetRowCountPositive    = "set.row_count_positive"
	KindRecordSchemaConformance = "record.schema_conformance"
)

// Handler is the per-kind evaluation function the registry
// dispatches into. The signature mirrors CheckEvaluator.Evaluate
// so that switching from a switch-on-kind to a map-on-kind is
// behavior-preserving for set-mode kinds and forward-compatible
// for record-mode kinds added in Wave-S sub-slice β.
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
}

// Evaluator dispatches CheckSpecs to per-kind handlers. The
// registry is populated at construction time with every kind
// the catalog declares; the dispatcher startup invariant
// (cmd/dq-engine) verifies catalog ⊆ registry.
//
// Satisfies runner.CheckEvaluator via the Evaluate method below.
type Evaluator struct {
	client   *bigquery.Client
	logger   *slog.Logger
	handlers map[string]Handler
}

// New validates the Config and returns an Evaluator with the
// default handler registry pre-populated. Returns an error when
// Client is missing.
//
// Handlers registered at construction time:
//
//   - set.row_count_positive — full set-mode implementation
//     (evaluateRowCountPositive in row_count_positive.go).
//   - record.schema_conformance — stub returning ResultError with
//     a "record-mode runtime not yet wired" diagnostic. Replaced
//     by the real implementation in Wave-S sub-slice β alongside
//     the record-mode runner.
//
// The stub keeps the dispatcher startup invariant satisfiable at
// sub-slice α (catalog declares record.schema_conformance; the
// engine has a registered handler for it) without shipping the
// runtime that consumes it. Record-mode rules that arrive at
// sub-slice α land on the stub and surface ResultError; in
// practice no record-mode rules exist at α (the rule migration
// stays set-only until β).
func New(cfg Config) (*Evaluator, error) {
	if cfg.Client == nil {
		return nil, errors.New("eval: Client is required")
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	e := &Evaluator{
		client:   cfg.Client,
		logger:   logger,
		handlers: map[string]Handler{},
	}
	e.Register(KindSetRowCountPositive, setRowCountPositiveHandler)
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
// implemented in record_schema_conformance.go. Wave-S sub-slice
// β wires the real handler that consumes per-window records
// from the record-mode runner via TriggerRequest.Records and
// applies the threshold aggregation per ADR-0026.
