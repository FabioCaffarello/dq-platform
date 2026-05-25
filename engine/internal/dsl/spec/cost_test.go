// path: engine/internal/dsl/spec/cost_test.go

package spec

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateCost_SetModeRule_AlwaysAllowed(t *testing.T) {
	r := RuleSpec{
		Version: SchemaVersionV2,
		Entity:  "customer",
		Mode:    ModeSet,
		Source: &Source{
			Type:      SourceTypeBigQuery,
			ProjectID: "p", DatasetID: "d", TableID: "t",
		},
		Checks: []Check{{CheckID: "c", Kind: "set.row_count_positive"}},
	}
	guardrails := CostGuardrails{
		MaxEvidenceSampleSize: 100,
		MaxLatenessTolerance:  5 * time.Minute,
	}
	if err := EvaluateCost(r, guardrails); err != nil {
		t.Errorf("set-mode rule must always pass cost; got %v", err)
	}
}

func TestEvaluateCost_RecordMode_WithinCeiling_Allowed(t *testing.T) {
	r := RuleSpec{
		Version: SchemaVersionV2,
		Entity:  "orders_stream",
		Mode:    ModeRecord,
		Source: &Source{
			Type: SourceTypeKafka, Topic: "t", ConsumerGroup: "g",
			Window: &Window{Type: WindowTypeTumbling, Duration: "1m", LatenessTolerance: "30s"},
		},
		Checks: []Check{{
			CheckID: "schema_present",
			Kind:    "record.schema_conformance",
			Params: map[string]any{
				"schema":      map[string]any{"type": "object"},
				"aggregation": map[string]any{"evidence_sample_size": 50},
			},
		}},
	}
	guardrails := CostGuardrails{
		MaxEvidenceSampleSize: 100,
		MaxLatenessTolerance:  5 * time.Minute,
	}
	if err := EvaluateCost(r, guardrails); err != nil {
		t.Errorf("rule within ceiling should pass; got %v", err)
	}
}

func TestEvaluateCost_RecordMode_SampleSizeExceedsCeiling_Rejected(t *testing.T) {
	r := RuleSpec{
		Version: SchemaVersionV2,
		Entity:  "orders_stream",
		Mode:    ModeRecord,
		Source: &Source{
			Type: SourceTypeKafka, Topic: "t", ConsumerGroup: "g",
			Window: &Window{Type: WindowTypeTumbling, Duration: "1m", LatenessTolerance: "30s"},
		},
		Checks: []Check{{
			CheckID: "schema_present",
			Kind:    "record.schema_conformance",
			Params: map[string]any{
				"schema":      map[string]any{"type": "object"},
				"aggregation": map[string]any{"evidence_sample_size": 500},
			},
		}},
	}
	guardrails := CostGuardrails{MaxEvidenceSampleSize: 100}
	err := EvaluateCost(r, guardrails)
	if err == nil {
		t.Fatal("expected rejection for evidence_sample_size > ceiling")
	}
	if !strings.Contains(err.Error(), "evidence_sample_size") {
		t.Errorf("error should name the field; got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "ADR-0027") {
		t.Errorf("error should cite ADR-0027; got %q", err.Error())
	}
}

func TestEvaluateCost_RecordMode_LatenessExceedsCeiling_Rejected(t *testing.T) {
	r := RuleSpec{
		Version: SchemaVersionV2,
		Entity:  "orders_stream",
		Mode:    ModeRecord,
		Source: &Source{
			Type: SourceTypeKafka, Topic: "t", ConsumerGroup: "g",
			Window: &Window{Type: WindowTypeTumbling, Duration: "1m", LatenessTolerance: "1h"},
		},
		Checks: []Check{{CheckID: "c", Kind: "record.schema_conformance"}},
	}
	guardrails := CostGuardrails{MaxLatenessTolerance: 5 * time.Minute}
	err := EvaluateCost(r, guardrails)
	if err == nil {
		t.Fatal("expected rejection for lateness_tolerance > ceiling")
	}
	if !strings.Contains(err.Error(), "lateness_tolerance") {
		t.Errorf("error should name the field; got %q", err.Error())
	}
}

func TestEvaluateCost_RecordMode_NoParams_AllowedByDefault(t *testing.T) {
	r := RuleSpec{
		Version: SchemaVersionV2,
		Entity:  "orders_stream",
		Mode:    ModeRecord,
		Source: &Source{
			Type: SourceTypeKafka, Topic: "t", ConsumerGroup: "g",
			Window: &Window{Type: WindowTypeTumbling, Duration: "1m", LatenessTolerance: "30s"},
		},
		Checks: []Check{{CheckID: "c", Kind: "record.schema_conformance"}},
	}
	guardrails := CostGuardrails{
		MaxEvidenceSampleSize: 100,
		MaxLatenessTolerance:  5 * time.Minute,
	}
	if err := EvaluateCost(r, guardrails); err != nil {
		t.Errorf("rule without override should pass; got %v", err)
	}
}

func TestEvaluateCost_ZeroGuardrails_AlwaysAllowed(t *testing.T) {
	// Zero MaxEvidenceSampleSize means "no ceiling enforced"
	// (the engine wouldn't ship with that posture, but guard
	// against silently rejecting everything).
	r := RuleSpec{
		Version: SchemaVersionV2,
		Entity:  "orders_stream",
		Mode:    ModeRecord,
		Source: &Source{
			Type: SourceTypeKafka, Topic: "t", ConsumerGroup: "g",
			Window: &Window{Type: WindowTypeTumbling, Duration: "1m", LatenessTolerance: "1h"},
		},
		Checks: []Check{{
			CheckID: "c",
			Kind:    "record.schema_conformance",
			Params: map[string]any{
				"aggregation": map[string]any{"evidence_sample_size": 999999},
			},
		}},
	}
	if err := EvaluateCost(r, CostGuardrails{}); err != nil {
		t.Errorf("zero guardrails should allow everything; got %v", err)
	}
}
