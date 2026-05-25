// path: engine/internal/dsl/spec/spec_test.go

package spec

import (
	"strings"
	"testing"

	"dq-platform/engine/internal/runner"
)

const validBody = `
version: 1
entity: customer
description: First onboarded entity end-to-end.
checks:
  - check_id: row_count_positive
    kind: row_count_positive
    description: Verifies the source table has at least one row.
`

const validV2SetBody = `
version: 2
entity: customer
mode: set
source:
  type: bigquery
  project_id: dq-local
  dataset_id: dq_fixture
  table_id: customer
checks:
  - check_id: row_count_positive
    kind: set.row_count_positive
`

const validV2RecordBody = `
version: 2
entity: orders_stream
mode: record
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema:
        type: object
`

func TestParse_HappyPath(t *testing.T) {
	r, err := Parse([]byte(validBody))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Version != 1 {
		t.Errorf("Version = %d; want 1", r.Version)
	}
	if r.Entity != "customer" {
		t.Errorf("Entity = %q; want customer", r.Entity)
	}
	if len(r.Checks) != 1 {
		t.Fatalf("len(Checks) = %d; want 1", len(r.Checks))
	}
	if r.Checks[0].CheckID != "row_count_positive" {
		t.Errorf("CheckID = %q; want row_count_positive", r.Checks[0].CheckID)
	}
	if r.Checks[0].Kind != "row_count_positive" {
		t.Errorf("Kind = %q; want row_count_positive", r.Checks[0].Kind)
	}
}

func TestParse_RejectsUnknownTopLevelField(t *testing.T) {
	body := validBody + "\nunknown_field: nope\n"
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for unknown top-level field")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("error should name the unknown field; got %q", err.Error())
	}
}

func TestParse_RejectsUnknownPerCheckField(t *testing.T) {
	body := `
version: 1
entity: customer
checks:
  - check_id: c1
    kind: row_count_positive
    unknown_check_field: nope
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for unknown per-check field")
	}
}

func TestParse_UnsupportedVersion(t *testing.T) {
	body := strings.Replace(validBody, "version: 1", "version: 7", 1)
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for unsupported version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error should name the version field; got %q", err.Error())
	}
}

func TestParse_V2Set_HappyPath(t *testing.T) {
	r, err := Parse([]byte(validV2SetBody))
	if err != nil {
		t.Fatalf("Parse v2 set: %v", err)
	}
	if r.Mode != "set" {
		t.Errorf("Mode = %q; want set", r.Mode)
	}
	if r.Source == nil || r.Source.Type != "bigquery" || r.Source.ProjectID != "dq-local" {
		t.Errorf("Source = %+v; want bigquery dq-local", r.Source)
	}
	specs := r.ToCheckSpecs()
	if len(specs) != 1 {
		t.Fatalf("ToCheckSpecs len = %d; want 1", len(specs))
	}
	if specs[0].Mode != "set" || specs[0].Source == nil || specs[0].Source.TableID != "customer" {
		t.Errorf("CheckSpec[0] = %+v; want mode=set source.table_id=customer", specs[0])
	}
}

func TestParse_V2Record_HappyPath(t *testing.T) {
	r, err := Parse([]byte(validV2RecordBody))
	if err != nil {
		t.Fatalf("Parse v2 record: %v", err)
	}
	if r.Mode != "record" {
		t.Errorf("Mode = %q; want record", r.Mode)
	}
	if r.Source == nil || r.Source.Type != "kafka" || r.Source.Topic != "orders.events.v1" {
		t.Errorf("Source = %+v; want kafka orders.events.v1", r.Source)
	}
	if r.Source.Window == nil || r.Source.Window.Type != "tumbling" || r.Source.Window.Duration != "1m" {
		t.Errorf("Window = %+v; want tumbling 1m", r.Source.Window)
	}
	specs := r.ToCheckSpecs()
	if specs[0].Mode != "record" {
		t.Errorf("CheckSpec mode = %q; want record", specs[0].Mode)
	}
	if specs[0].Params == nil {
		t.Errorf("CheckSpec.Params is nil; want non-nil")
	}
}

func TestParse_V2_RequiresMode(t *testing.T) {
	body := `
version: 2
entity: customer
source:
  type: bigquery
  project_id: p
  dataset_id: d
  table_id: t
checks:
  - check_id: c
    kind: set.row_count_positive
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected v2 to require mode")
	}
	if !strings.Contains(err.Error(), "mode") {
		t.Errorf("error should mention mode; got %q", err.Error())
	}
}

func TestParse_V2_KindPrefixMismatch(t *testing.T) {
	body := `
version: 2
entity: customer
mode: set
source:
  type: bigquery
  project_id: p
  dataset_id: d
  table_id: t
checks:
  - check_id: c
    kind: record.schema_conformance
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected kind prefix mismatch")
	}
	if !strings.Contains(err.Error(), "cross-check #4") {
		t.Errorf("error should cite cross-check #4; got %q", err.Error())
	}
}

func TestParse_V2_SourceTypeMismatch(t *testing.T) {
	body := `
version: 2
entity: customer
mode: set
source:
  type: kafka
  topic: t
  consumer_group: g
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: c
    kind: set.row_count_positive
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected source.type mismatch")
	}
	if !strings.Contains(err.Error(), "cross-check #7") {
		t.Errorf("error should cite cross-check #7; got %q", err.Error())
	}
}

func TestParse_V1_RejectsV2Fields(t *testing.T) {
	body := `
version: 1
entity: customer
mode: set
checks:
  - check_id: c
    kind: row_count_positive
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected v1 to reject v2-only mode field")
	}
}

func TestParse_EmptyChecks(t *testing.T) {
	body := `
version: 1
entity: customer
checks: []
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for empty checks")
	}
	if !strings.Contains(err.Error(), "checks") {
		t.Errorf("error should name the checks field; got %q", err.Error())
	}
}

func TestParse_PipeInEntity_Rejected(t *testing.T) {
	body := strings.Replace(validBody, "entity: customer", "entity: cust|omer", 1)
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for pipe in entity")
	}
	if !strings.Contains(err.Error(), "pipe") {
		t.Errorf("error should mention pipe; got %q", err.Error())
	}
}

func TestParse_LongEntity_Rejected(t *testing.T) {
	body := strings.Replace(validBody, "entity: customer",
		"entity: "+strings.Repeat("a", maxIdentifierLen+1), 1)
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for entity exceeding length ceiling")
	}
}

func TestParse_EntityInvalidPattern_Rejected(t *testing.T) {
	body := strings.Replace(validBody, "entity: customer", `entity: "cust omer"`, 1)
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for entity with whitespace")
	}
}

func TestParse_MissingCheckID(t *testing.T) {
	body := `
version: 1
entity: customer
checks:
  - kind: row_count_positive
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for missing check_id")
	}
}

func TestParse_MissingKind(t *testing.T) {
	body := `
version: 1
entity: customer
checks:
  - check_id: row_count_positive
`
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for missing kind")
	}
}

func TestToCheckSpecs(t *testing.T) {
	r, err := Parse([]byte(validBody))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got := r.ToCheckSpecs()
	want := []runner.CheckSpec{{CheckID: "row_count_positive", Kind: "row_count_positive"}}
	if len(got) != len(want) {
		t.Fatalf("len(ToCheckSpecs) = %d; want %d", len(got), len(want))
	}
	if got[0].CheckID != want[0].CheckID || got[0].Kind != want[0].Kind {
		t.Errorf("ToCheckSpecs[0] = %+v; want %+v", got[0], want[0])
	}
}

func TestToCheckSpecs_EmptyChecks(t *testing.T) {
	r := RuleSpec{}
	if got := r.ToCheckSpecs(); got != nil {
		t.Errorf("ToCheckSpecs on empty = %v; want nil", got)
	}
}

func TestParse_DescriptionOptional(t *testing.T) {
	body := `
version: 1
entity: customer
checks:
  - check_id: c1
    kind: k1
`
	r, err := Parse([]byte(body))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if r.Description != "" {
		t.Errorf("Description = %q; want empty", r.Description)
	}
}
