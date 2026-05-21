// path: engine/internal/runner/check_evaluator.go

package runner

import (
	"context"

	"dq-platform/engine/internal/results"
)

// CheckEvaluator drives the evaluation of one check against the
// loaded manifest. ADR-0004 CC1 enumerates the four `result`
// values; the evaluator returns one of them plus the structured
// evidence-summary fields per ADR-0003 CC7.
//
// Phase-4c ships a no-op default (NoopEvaluator). Real evaluators
// are added by check-kind in Phase 6 onward. The CheckSpec.Kind
// field is the evaluator's discriminator; for the no-op default,
// Kind is ignored.
type CheckEvaluator interface {
	Evaluate(ctx context.Context, spec CheckSpec, trigger TriggerRequest) (Evaluation, error)
}

// Evaluation is the structured output of one check evaluation.
type Evaluation struct {
	Result              results.CheckResult
	EvidenceSummary     map[string]any
	SampleViolatingRows []map[string]any
}

// NoopEvaluator returns ResultPass for every check with an empty
// evidence summary. Used by Phase 4c integration tests when no
// real check kind is wired.
type NoopEvaluator struct{}

func (NoopEvaluator) Evaluate(_ context.Context, _ CheckSpec, _ TriggerRequest) (Evaluation, error) {
	return Evaluation{Result: results.ResultPass}, nil
}

// FixedResultEvaluator is a test double: every call returns the
// same configured Evaluation. Used to exercise the runner's
// mapping logic without standing up a real check kind.
type FixedResultEvaluator struct {
	Result results.CheckResult
}

func (e FixedResultEvaluator) Evaluate(_ context.Context, _ CheckSpec, _ TriggerRequest) (Evaluation, error) {
	return Evaluation{Result: e.Result}, nil
}

// PerCheckEvaluator is a test double that maps check_id → result
// so a single test can drive the runner with a heterogeneous mix
// of pass / fail / degraded / error results.
type PerCheckEvaluator struct {
	Results map[string]results.CheckResult
}

func (e PerCheckEvaluator) Evaluate(_ context.Context, spec CheckSpec, _ TriggerRequest) (Evaluation, error) {
	r, ok := e.Results[spec.CheckID]
	if !ok {
		r = results.ResultPass
	}
	return Evaluation{Result: r}, nil
}
