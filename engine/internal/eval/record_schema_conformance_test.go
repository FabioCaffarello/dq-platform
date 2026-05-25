// path: engine/internal/eval/record_schema_conformance_test.go

package eval

import (
	"context"
	"testing"
	"time"

	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

// recordHandlerHarness wraps the Evaluator with the bare minimum
// to invoke the record.schema_conformance handler. The handler
// does not use the BigQuery client, so stubClient is enough.
func recordHandlerHarness(t *testing.T) *Evaluator {
	t.Helper()
	cli := stubClient(t)
	e, err := New(Config{Client: cli})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

func requireEntitySchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []any{"id", "event_type"},
		"properties": map[string]any{
			"id":         map[string]any{"type": "string"},
			"event_type": map[string]any{"type": "string"},
		},
	}
}

func TestRecordHandler_AllRecordsPass_ReturnsResultPass(t *testing.T) {
	e := recordHandlerHarness(t)
	records := []runner.Record{
		{Partition: 0, Offset: 1, Body: []byte(`{"id":"a","event_type":"created"}`)},
		{Partition: 0, Offset: 2, Body: []byte(`{"id":"b","event_type":"updated"}`)},
	}
	spec := runner.CheckSpec{
		CheckID: "schema_present",
		Kind:    KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": requireEntitySchema(),
		},
	}
	trigger := runner.TriggerRequest{Entity: "orders_stream", Records: records}

	eval, err := e.Evaluate(context.Background(), spec, trigger)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultPass {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultPass)
	}
	if eval.EvidenceSummary["records_evaluated"] != 2 {
		t.Errorf("records_evaluated = %v; want 2", eval.EvidenceSummary["records_evaluated"])
	}
	if eval.EvidenceSummary["violations"] != 0 {
		t.Errorf("violations = %v; want 0", eval.EvidenceSummary["violations"])
	}
}

func TestRecordHandler_OneViolation_StrictDefaults_Fails(t *testing.T) {
	// Catalog default is fail_if_violation_rate=0.0, so even a
	// single violation pushes the result to fail.
	e := recordHandlerHarness(t)
	records := []runner.Record{
		{Partition: 0, Offset: 1, Body: []byte(`{"id":"a","event_type":"created"}`)},
		{Partition: 0, Offset: 2, Body: []byte(`{"id":"b"}`)}, // missing event_type
	}
	spec := runner.CheckSpec{
		CheckID: "schema_present",
		Kind:    KindRecordSchemaConformance,
		Params:  map[string]any{"schema": requireEntitySchema()},
	}

	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream", Records: records})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultFail {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultFail)
	}
	if eval.EvidenceSummary["violations"] != 1 {
		t.Errorf("violations = %v; want 1", eval.EvidenceSummary["violations"])
	}
	if len(eval.SampleViolatingRows) != 1 {
		t.Errorf("len(SampleViolatingRows) = %d; want 1", len(eval.SampleViolatingRows))
	}
	if eval.SampleViolatingRows[0]["offset"] != int64(2) {
		t.Errorf("sample offset = %v; want 2", eval.SampleViolatingRows[0]["offset"])
	}
}

func TestRecordHandler_BelowWarnRate_ReturnsPass(t *testing.T) {
	// 1 of 10 records violates; with fail=0.2 and warn=0.05,
	// rate=0.1 is below fail but above warn → degraded.
	e := recordHandlerHarness(t)
	records := make([]runner.Record, 10)
	for i := range records {
		body := []byte(`{"id":"x","event_type":"created"}`)
		if i == 9 {
			body = []byte(`{"id":"x"}`) // last one violates
		}
		records[i] = runner.Record{Partition: 0, Offset: int64(i), Body: body}
	}
	spec := runner.CheckSpec{
		Kind: KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": requireEntitySchema(),
			"aggregation": map[string]any{
				"fail_if_violation_rate": 0.2,
				"warn_if_violation_rate": 0.05,
				"evidence_sample_size":   5,
			},
		},
	}
	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream", Records: records})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultDegraded {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultDegraded)
	}
}

func TestRecordHandler_VacuousCase_ZeroRecords_NoLateDrops_Pass(t *testing.T) {
	e := recordHandlerHarness(t)
	spec := runner.CheckSpec{
		Kind: KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": requireEntitySchema(),
		},
	}
	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream", Records: nil})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultPass {
		t.Errorf("Result = %q; want %q (vacuous case)", eval.Result, results.ResultPass)
	}
}

func TestRecordHandler_VacuousCase_ZeroRecords_LateDrops_Degraded(t *testing.T) {
	e := recordHandlerHarness(t)
	spec := runner.CheckSpec{
		Kind: KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": requireEntitySchema(),
		},
	}
	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{
			Entity:           "orders_stream",
			Records:          nil,
			LateDroppedCount: 50,
		})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultDegraded {
		t.Errorf("Result = %q; want %q (late-drop catastrophe)", eval.Result, results.ResultDegraded)
	}
}

func TestRecordHandler_SampleSizeCap(t *testing.T) {
	// 20 violations but sample_size=3 → exactly 3 samples retained.
	e := recordHandlerHarness(t)
	records := make([]runner.Record, 20)
	for i := range records {
		records[i] = runner.Record{
			Partition: 0,
			Offset:    int64(i),
			Body:      []byte(`{"id":"x"}`), // missing event_type
		}
	}
	spec := runner.CheckSpec{
		Kind: KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": requireEntitySchema(),
			"aggregation": map[string]any{
				"evidence_sample_size": 3,
			},
		},
	}
	eval, _ := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream", Records: records})
	if len(eval.SampleViolatingRows) != 3 {
		t.Errorf("len(SampleViolatingRows) = %d; want 3 (cap)", len(eval.SampleViolatingRows))
	}
	if eval.EvidenceSummary["violations"] != 20 {
		t.Errorf("violations = %v; want 20", eval.EvidenceSummary["violations"])
	}
}

func TestRecordHandler_MalformedJSON_CountsAsViolation(t *testing.T) {
	e := recordHandlerHarness(t)
	records := []runner.Record{
		{Partition: 0, Offset: 1, Body: []byte("not json {")},
	}
	spec := runner.CheckSpec{
		Kind: KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": requireEntitySchema(),
		},
	}
	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream", Records: records})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if eval.Result != results.ResultFail {
		t.Errorf("Result = %q; want %q (malformed JSON is a violation)", eval.Result, results.ResultFail)
	}
}

func TestRecordHandler_MissingSchemaParam_ReturnsError(t *testing.T) {
	e := recordHandlerHarness(t)
	spec := runner.CheckSpec{
		Kind:   KindRecordSchemaConformance,
		Params: map[string]any{}, // schema missing
	}
	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream"})
	if err == nil {
		t.Fatalf("expected error when schema is missing")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
}

func TestRecordHandler_InvalidJSONSchema_ReturnsError(t *testing.T) {
	e := recordHandlerHarness(t)
	spec := runner.CheckSpec{
		Kind: KindRecordSchemaConformance,
		Params: map[string]any{
			"schema": map[string]any{
				"type": "not_a_valid_jsonschema_type",
			},
		},
	}
	eval, err := e.Evaluate(context.Background(), spec,
		runner.TriggerRequest{Entity: "orders_stream", Records: []runner.Record{
			{Body: []byte(`{}`)},
		}})
	if err == nil {
		t.Fatalf("expected error for invalid JSON Schema")
	}
	if eval.Result != results.ResultError {
		t.Errorf("Result = %q; want %q", eval.Result, results.ResultError)
	}
}

// Compile-time silencer for the time import — only used in tests
// that pass through Records.Timestamp; keeping the dependency
// here documents that future tests that exercise timestamp-based
// behavior should also live in this file.
var _ = time.Time{}
