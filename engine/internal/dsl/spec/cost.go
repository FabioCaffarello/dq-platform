// path: engine/internal/dsl/spec/cost.go

package spec

import (
	"fmt"
	"time"
)

// CostGuardrails carries the ADR-0027 record-mode cost ceiling
// values the engine enforces at rule-load time. Field-level
// units mirror engine/internal/env.RecordModeCost; the spec
// package does not depend on env to keep the dependency graph
// clean.
type CostGuardrails struct {
	// MaxEvidenceSampleSize bounds the per-rule
	// params.aggregation.evidence_sample_size override per
	// ADR-0027 §C-B0S7.1.
	MaxEvidenceSampleSize int

	// MaxLatenessTolerance bounds the rule's
	// source.kafka.window.lateness_tolerance per ADR-0027.
	MaxLatenessTolerance time.Duration
}

// EvaluateCost enforces the ADR-0027 record-mode cost
// guardrails against a parsed RuleSpec. Returns nil for any
// set-mode rule (no record-mode cost dimensions apply) and for
// record-mode rules within the guardrails. Returns an error
// naming the offending dimension when a rule exceeds a
// ceiling — the engine binary refuses to load the manifest
// with such a rule, surfacing the violation at boot.
//
// The ADR-0026 catalog default (evidence_sample_size = 10) is
// always under any reasonable deployment ceiling, so a rule
// that omits the override block passes cleanly.
func EvaluateCost(r RuleSpec, g CostGuardrails) error {
	if r.Mode != ModeRecord {
		return nil
	}
	if r.Source != nil && r.Source.Window != nil && g.MaxLatenessTolerance > 0 {
		lt, err := time.ParseDuration(r.Source.Window.LatenessTolerance)
		if err != nil {
			return fmt.Errorf("rule %q: parse lateness_tolerance: %w", r.Entity, err)
		}
		if lt > g.MaxLatenessTolerance {
			return fmt.Errorf(
				"rule %q: source.window.lateness_tolerance %s exceeds deployment ceiling %s (ADR-0027)",
				r.Entity, lt, g.MaxLatenessTolerance)
		}
	}
	if g.MaxEvidenceSampleSize <= 0 {
		return nil
	}
	for _, c := range r.Checks {
		size, ok := evidenceSampleSize(c.Params)
		if !ok {
			continue
		}
		if size > g.MaxEvidenceSampleSize {
			return fmt.Errorf(
				"rule %q check %q: params.aggregation.evidence_sample_size %d exceeds deployment ceiling %d (ADR-0027)",
				r.Entity, c.CheckID, size, g.MaxEvidenceSampleSize)
		}
	}
	return nil
}

// evidenceSampleSize extracts the optional override from a
// check's params map. Returns (0, false) when params or the
// nested aggregation/evidence_sample_size keys are absent.
func evidenceSampleSize(params map[string]any) (int, bool) {
	if params == nil {
		return 0, false
	}
	aggRaw, ok := params["aggregation"]
	if !ok {
		return 0, false
	}
	agg, ok := aggRaw.(map[string]any)
	if !ok {
		return 0, false
	}
	v, ok := agg["evidence_sample_size"]
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	default:
		return 0, false
	}
}
