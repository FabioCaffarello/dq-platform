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
	if containsWindowParams(got) {
		t.Errorf("non-partitioned SQL %q must not reference @window_start/@window_end", got)
	}
}

func TestCompileSQL_RowCountPositive_PartitionPredicate(t *testing.T) {
	// B2-12: when source.partition_column is set, the SQL gains
	// a half-open-interval predicate `>= @window_start AND
	// < @window_end`. The predicate uses the partition column
	// directly (no wrapping function) so BigQuery's partition
	// pruning fires.
	src := ruleSource{
		Type:            "bigquery",
		ProjectID:       "dq-local",
		DatasetID:       "dq_fixture",
		TableID:         "customer",
		PartitionColumn: "event_ts",
	}
	got, ok := compileSQL("set.row_count_positive", src)
	if !ok {
		t.Fatal("compileSQL returned !ok")
	}
	want := "SELECT COUNT(*) AS row_count FROM `dq-local.dq_fixture.customer` " +
		"WHERE `event_ts` >= @window_start AND `event_ts` < @window_end"
	if got != want {
		t.Errorf("partitioned SQL = %q; want %q", got, want)
	}
	if !containsWindowParams(got) {
		t.Error("partitioned SQL must reference @window_start AND @window_end")
	}
}

func TestContainsWindowParams(t *testing.T) {
	if !containsWindowParams("WHERE x >= @window_start AND x < @window_end") {
		t.Error("predicate with both params should return true")
	}
	if containsWindowParams("SELECT 1") {
		t.Error("trivial SELECT should return false")
	}
	// One-sided: not a valid combo from compileSQL, but the
	// function is conservative — requires both.
	if containsWindowParams("WHERE x >= @window_start") {
		t.Error("only @window_start present should return false")
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

func TestCompileSQL_RowCountWithinBaseline(t *testing.T) {
	// B2-14: the baselined kind's dry-run estimates the
	// **current-value** count (same SQL as the unbaselined kind);
	// the baseline query itself is NOT dry-run-checked.
	src := ruleSource{
		Type:      "bigquery",
		ProjectID: "dq-local",
		DatasetID: "dq_fixture",
		TableID:   "customer",
	}
	got, ok := compileSQL("set.row_count_within_baseline", src)
	if !ok {
		t.Fatal("compileSQL returned !ok for set.row_count_within_baseline")
	}
	want := "SELECT COUNT(*) AS row_count FROM `dq-local.dq_fixture.customer`"
	if got != want {
		t.Errorf("baseline kind SQL = %q; want %q", got, want)
	}

	// Partition retrofit applies equally.
	srcPart := src
	srcPart.PartitionColumn = "event_ts"
	got, _ = compileSQL("set.row_count_within_baseline", srcPart)
	wantPart := "SELECT COUNT(*) AS row_count FROM `dq-local.dq_fixture.customer` " +
		"WHERE `event_ts` >= @window_start AND `event_ts` < @window_end"
	if got != wantPart {
		t.Errorf("baseline+partition SQL = %q; want %q", got, wantPart)
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
