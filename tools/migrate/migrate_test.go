// path: tools/migrate/migrate_test.go

package main

import (
	"os"
	"strings"
	"testing"
)

func TestBigQuerySource_Validate(t *testing.T) {
	cases := []struct {
		name    string
		src     BigQuerySource
		wantErr bool
	}{
		{"happy", BigQuerySource{ProjectID: "p", DatasetID: "d", TableID: "t"}, false},
		{"missing project", BigQuerySource{DatasetID: "d", TableID: "t"}, true},
		{"missing dataset", BigQuerySource{ProjectID: "p", TableID: "t"}, true},
		{"missing table", BigQuerySource{ProjectID: "p", DatasetID: "d"}, true},
		{"all empty", BigQuerySource{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.src.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("want error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestV1ToV2_CustomerExample feeds the canonical v1 fixture
// through V1ToV2 and verifies the output carries every v2
// invariant the production rules/customer.yaml shape commits
// to. The exact byte output is not asserted because YAML
// emit ordering can drift between encoder versions; we test
// structural invariants instead.
func TestV1ToV2_CustomerExample(t *testing.T) {
	raw, err := os.ReadFile("testdata/v1/customer.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	src := BigQuerySource{
		ProjectID: "dq-local",
		DatasetID: "dq_fixture",
		TableID:   "customer",
	}
	out, err := V1ToV2(raw, src)
	if err != nil {
		t.Fatalf("V1ToV2: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"version: 2",
		"entity: customer",
		"mode: set",
		"description: First onboarded entity end-to-end (W3-P6d).",
		"type: bigquery",
		"project_id: dq-local",
		"dataset_id: dq_fixture",
		"table_id: customer",
		"check_id: row_count_positive",
		"kind: set.row_count_positive",
		"description: Verifies the source table has at least one row.",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing fragment %q\nfull output:\n%s", want, s)
		}
	}
	// The v1 `kind: row_count_positive` (unprefixed) must NOT
	// appear in the v2 output — the prefix transform must have
	// fired.
	if strings.Contains(s, "kind: row_count_positive\n") {
		t.Errorf("output still carries unprefixed v1 kind:\n%s", s)
	}
}

func TestV1ToV2_RejectsV2Input(t *testing.T) {
	v2 := []byte(`version: 2
entity: customer
mode: set
source:
  type: bigquery
  project_id: p
  dataset_id: d
  table_id: t
checks:
  - check_id: c
    kind: set.row_count_positive
`)
	_, err := V1ToV2(v2, BigQuerySource{ProjectID: "p", DatasetID: "d", TableID: "t"})
	if err == nil {
		t.Fatal("want error for v2 input, got nil")
	}
	// Either error path is correct: the strict YAML decoder
	// rejects unknown fields (mode, source) before the version
	// check fires, OR the version check fires explicitly. Both
	// produce a non-nil error; we just confirm rejection.
}

func TestV1ToV2_RejectsMissingChecks(t *testing.T) {
	v1NoChecks := []byte(`version: 1
entity: customer
checks: []
`)
	_, err := V1ToV2(v1NoChecks, BigQuerySource{ProjectID: "p", DatasetID: "d", TableID: "t"})
	if err == nil {
		t.Fatal("want error for empty checks, got nil")
	}
}

func TestV1ToV2_RejectsMissingEntity(t *testing.T) {
	v1NoEntity := []byte(`version: 1
checks:
  - check_id: c
    kind: row_count_positive
`)
	_, err := V1ToV2(v1NoEntity, BigQuerySource{ProjectID: "p", DatasetID: "d", TableID: "t"})
	if err == nil {
		t.Fatal("want error for missing entity, got nil")
	}
}

func TestV1ToV2_RejectsMissingBigQueryFlags(t *testing.T) {
	raw, err := os.ReadFile("testdata/v1/customer.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	// Empty source — should be rejected before YAML parse.
	if _, err := V1ToV2(raw, BigQuerySource{}); err == nil {
		t.Fatal("want error for empty source, got nil")
	}
}

func TestPromoteKindToSetPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"row_count_positive", "set.row_count_positive"},
		{"set.row_count_positive", "set.row_count_positive"}, // idempotent
		{"record.schema_conformance", "record.schema_conformance"},
		{"unknown_kind", "set.unknown_kind"},
	}
	for _, tc := range cases {
		if got := promoteKindToSetPrefix(tc.in); got != tc.want {
			t.Errorf("promoteKindToSetPrefix(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestV1ToV2_PreservesParams(t *testing.T) {
	// A v1 check carrying params should round-trip them into v2.
	// The catalog cross-check would reject a kind without
	// params it requires; this test exercises the migrator's
	// preservation, not the downstream catalog conformance.
	v1WithParams := []byte(`version: 1
entity: example
checks:
  - check_id: future_kind
    kind: future_kind
    params:
      threshold: 100
      mode: strict
`)
	out, err := V1ToV2(v1WithParams, BigQuerySource{ProjectID: "p", DatasetID: "d", TableID: "t"})
	if err != nil {
		t.Fatalf("V1ToV2: %v", err)
	}
	s := string(out)
	for _, want := range []string{
		"threshold: 100",
		"mode: strict",
		"kind: set.future_kind",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q\nfull:\n%s", want, s)
		}
	}
}
