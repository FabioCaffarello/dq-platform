// path: engine/internal/eval/record_schema_conformance.go

package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// recordSchemaConformanceParams is the typed view of a
// record.schema_conformance rule's params block per the catalog
// at engine/internal/dsl/catalog/v1.yaml. The linter validates
// the params shape against the catalog; the handler re-validates
// defensively before consuming.
type recordSchemaConformanceParams struct {
	// Schema is the JSON Schema fragment the handler validates
	// each record against. Required per ADR-0026.
	Schema map[string]any

	// FailRate is the violation rate at or above which the
	// aggregated CheckResult is `fail`. Catalog default: 0.0
	// (strict — any violation fails).
	FailRate float64

	// WarnRate is the violation rate at or above which the
	// aggregated CheckResult is `degraded`. Catalog default:
	// nil (no degraded band). When non-nil, WarnRate < FailRate.
	WarnRate *float64

	// SampleSize bounds the count of per-record violation
	// descriptors retained in evidence per ADR-0026. Catalog
	// default: 10.
	SampleSize int
}

// recordSchemaConformanceCatalogDefaults mirrors the catalog
// aggregation.defaults block. The handler applies these to
// any params field the rule omits.
var recordSchemaConformanceCatalogDefaults = recordSchemaConformanceParams{
	FailRate:   0.0,
	WarnRate:   nil,
	SampleSize: 10,
}

// recordSchemaConformanceHandler is the production registry
// entry for record.schema_conformance per ADR-0026. It replaces
// the α stub. The runner's RecordRunner delivers per-window
// records via TriggerRequest.Records; the handler decodes each
// record body as JSON, validates against params.schema, and
// aggregates per-record outcomes into one CheckResult using the
// threshold function.
func recordSchemaConformanceHandler(_ context.Context, e *Evaluator, spec runner.CheckSpec, trigger runner.TriggerRequest) (runner.Evaluation, error) {
	params, err := parseRecordSchemaConformanceParams(spec.Params)
	if err != nil {
		e.logger.Warn("record.schema_conformance params invalid",
			"check_id", spec.CheckID,
			"entity", trigger.Entity,
			"error", err.Error(),
			"adr_reference", "ADR-0022",
		)
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "invalid_params",
				"error":  err.Error(),
			},
		}, err
	}

	compiledSchema, err := compileRecordSchema(spec.CheckID, params.Schema)
	if err != nil {
		e.logger.Warn("record.schema_conformance schema compile failed",
			"check_id", spec.CheckID,
			"entity", trigger.Entity,
			"error", err.Error(),
		)
		return runner.Evaluation{
			Result: results.ResultError,
			EvidenceSummary: map[string]any{
				"kind":   spec.Kind,
				"reason": "schema_compile_failed",
				"error":  err.Error(),
			},
		}, err
	}

	recordsEvaluated := len(trigger.Records)
	violations := 0
	samples := make([]map[string]any, 0, params.SampleSize)

	for _, rec := range trigger.Records {
		var body any
		if err := json.Unmarshal(rec.Body, &body); err != nil {
			violations++
			if len(samples) < params.SampleSize {
				samples = append(samples, map[string]any{
					"partition": rec.Partition,
					"offset":    rec.Offset,
					"reason":    fmt.Sprintf("json decode: %v", err),
				})
			}
			continue
		}
		if verr := compiledSchema.Validate(body); verr != nil {
			violations++
			if len(samples) < params.SampleSize {
				samples = append(samples, map[string]any{
					"partition": rec.Partition,
					"offset":    rec.Offset,
					"reason":    verr.Error(),
				})
			}
		}
	}

	status := classifyThreshold(recordsEvaluated, violations, trigger.LateDroppedCount, params)
	violationRate := 0.0
	if recordsEvaluated > 0 {
		violationRate = float64(violations) / float64(recordsEvaluated)
	}

	return runner.Evaluation{
		Result: status,
		EvidenceSummary: map[string]any{
			"kind":                spec.Kind,
			"records_evaluated":   recordsEvaluated,
			"records_passed":      recordsEvaluated - violations,
			"violations":          violations,
			"violation_rate":      violationRate,
			"late_dropped_count":  trigger.LateDroppedCount,
			"fail_if_rate":        params.FailRate,
			"warn_if_rate":        warnRateForEvidence(params.WarnRate),
			"evidence_sample_cap": params.SampleSize,
		},
		SampleViolatingRows: samples,
	}, nil
}

// classifyThreshold maps per-window record counts to a
// CheckResult per ADR-0026:
//
//   - records_evaluated == 0 AND late_dropped_count == 0 → pass
//     (vacuous: no data arrived, nothing to evaluate)
//   - records_evaluated == 0 AND late_dropped_count > 0  → degraded
//     (late-drop catastrophe — all in-window data arrived late)
//   - violations == 0                                     → pass
//     (short-circuit; the literal-pseudocode interpretation of
//     "rate >= fail_rate" with FailRate=0 would otherwise fail a
//     clean window, which contradicts ADR-0026's "strict by
//     default — any violation fails" intent)
//   - violation_rate >= fail_rate                         → fail
//   - warn_rate != nil AND violation_rate >= warn_rate    → degraded
//   - else                                                → pass
func classifyThreshold(recordsEvaluated, violations, lateDropped int, params recordSchemaConformanceParams) results.CheckResult {
	if recordsEvaluated == 0 {
		if lateDropped > 0 {
			return results.ResultDegraded
		}
		return results.ResultPass
	}
	if violations == 0 {
		return results.ResultPass
	}
	rate := float64(violations) / float64(recordsEvaluated)
	if rate >= params.FailRate {
		return results.ResultFail
	}
	if params.WarnRate != nil && rate >= *params.WarnRate {
		return results.ResultDegraded
	}
	return results.ResultPass
}

// parseRecordSchemaConformanceParams reads the runner's typed
// params map and produces the per-rule effective params. The
// catalog's aggregation.defaults are applied as fallbacks; the
// rule's params.aggregation block (if present) overrides them
// per ADR-0026.
func parseRecordSchemaConformanceParams(raw map[string]any) (recordSchemaConformanceParams, error) {
	p := recordSchemaConformanceCatalogDefaults
	if raw == nil {
		return p, errors.New("params is required (params.schema must be set)")
	}
	schemaRaw, ok := raw["schema"]
	if !ok {
		return p, errors.New("params.schema is required")
	}
	schemaMap, ok := schemaRaw.(map[string]any)
	if !ok {
		return p, errors.New("params.schema must be a JSON Schema object")
	}
	p.Schema = schemaMap

	if aggRaw, ok := raw["aggregation"]; ok {
		aggMap, ok := aggRaw.(map[string]any)
		if !ok {
			return p, errors.New("params.aggregation must be an object")
		}
		if v, ok := aggMap["fail_if_violation_rate"]; ok {
			f, err := asFloat(v, "params.aggregation.fail_if_violation_rate")
			if err != nil {
				return p, err
			}
			p.FailRate = f
		}
		if v, ok := aggMap["warn_if_violation_rate"]; ok {
			if v == nil {
				p.WarnRate = nil
			} else {
				f, err := asFloat(v, "params.aggregation.warn_if_violation_rate")
				if err != nil {
					return p, err
				}
				p.WarnRate = &f
			}
		}
		if v, ok := aggMap["evidence_sample_size"]; ok {
			i, err := asInt(v, "params.aggregation.evidence_sample_size")
			if err != nil {
				return p, err
			}
			p.SampleSize = i
		}
	}
	if p.FailRate < 0 || p.FailRate > 1 {
		return p, fmt.Errorf("params.aggregation.fail_if_violation_rate %v is not in [0, 1]", p.FailRate)
	}
	if p.WarnRate != nil && (*p.WarnRate < 0 || *p.WarnRate > 1) {
		return p, fmt.Errorf("params.aggregation.warn_if_violation_rate %v is not in [0, 1]", *p.WarnRate)
	}
	if p.SampleSize < 0 {
		return p, fmt.Errorf("params.aggregation.evidence_sample_size %d is negative", p.SampleSize)
	}
	return p, nil
}

// compileRecordSchema marshals the YAML-parsed schema map back to
// JSON and feeds it to jsonschema. The compiler validates the
// schema itself (rejecting malformed JSON Schemas at handler
// invocation time).
func compileRecordSchema(checkID string, schema map[string]any) (*jsonschema.Schema, error) {
	raw, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("marshal params.schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	id := fmt.Sprintf("rule://%s/params_schema", checkID)
	if err := compiler.AddResource(id, bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("add params.schema resource: %w", err)
	}
	sch, err := compiler.Compile(id)
	if err != nil {
		return nil, fmt.Errorf("compile params.schema: %w", err)
	}
	return sch, nil
}

func asFloat(v any, field string) (float64, error) {
	switch t := v.(type) {
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case int64:
		return float64(t), nil
	default:
		return 0, fmt.Errorf("%s must be a number; got %T", field, v)
	}
}

func asInt(v any, field string) (int, error) {
	switch t := v.(type) {
	case int:
		return t, nil
	case int64:
		return int(t), nil
	case float64:
		if t < 0 || t > float64(^uint(0)>>1) {
			return 0, fmt.Errorf("%s %v out of int range", field, t)
		}
		return int(t), nil
	default:
		return 0, fmt.Errorf("%s must be an integer; got %T", field, v)
	}
}

func warnRateForEvidence(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
