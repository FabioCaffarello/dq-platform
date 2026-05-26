// path: tools/dryrun/dryrun_test.go

package main

import (
	"strings"
	"testing"
)

func TestCompileSQL_RowCountPositive(t *testing.T) {
	src := ruleSource{
		Type:      "bigquery",
		ProjectID: "dq-local",
		DatasetID: "dq_fixture",
		TableID:   "customer",
	}
	got, ok := compileSQL("set.row_count_positive", src)
	if !ok {
		t.Fatal("compileSQL returned !ok for set.row_count_positive")
	}
	want := "SELECT COUNT(*) AS row_count FROM `dq-local.dq_fixture.customer`"
	if got != want {
		t.Errorf("SQL = %q; want %q", got, want)
	}
}

func TestCompileSQL_UnknownKind(t *testing.T) {
	// A kind without a compiler-side template returns !ok. The
	// caller (handleRule) records this as a "no template yet"
	// skip — not an error. The runtime evaluator may still
	// handle the kind via its own code path.
	src := ruleSource{Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t"}
	if _, ok := compileSQL("record.schema_conformance", src); ok {
		t.Error("compileSQL accepted record-mode kind; want !ok")
	}
	if _, ok := compileSQL("set.future_kind", src); ok {
		t.Error("compileSQL accepted unknown set-mode kind; want !ok")
	}
}

func TestNew_RequiresRulesDirAndClient(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Error("New({}) returned nil; want error for missing RulesDir")
	}
	if _, err := New(Config{RulesDir: "rules"}); err == nil {
		t.Error("New(RulesDir-only) returned nil; want error for missing BigQueryClient")
	}
}

// TestErrCostCeilingExceeded_Message confirms the sentinel error
// surfaces in the wrapped error message — the CI lane parses for
// this string to surface a clear cost-related failure.
func TestErrCostCeilingExceeded_Message(t *testing.T) {
	if !strings.Contains(ErrCostCeilingExceeded.Error(), "cost ceiling exceeded") {
		t.Errorf("ErrCostCeilingExceeded = %q; want 'cost ceiling exceeded' substring", ErrCostCeilingExceeded.Error())
	}
}
