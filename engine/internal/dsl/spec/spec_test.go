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

func TestParse_WrongVersion(t *testing.T) {
	body := strings.Replace(validBody, "version: 1", "version: 2", 1)
	_, err := Parse([]byte(body))
	if err == nil {
		t.Fatal("expected error for version != 1")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Errorf("error should name the version field; got %q", err.Error())
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
