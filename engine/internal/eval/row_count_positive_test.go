// path: engine/internal/eval/row_count_positive_test.go

package eval

import (
	"strings"
	"testing"

	"dq-platform/engine/internal/runner"
)

func TestRowCountPositiveSQL_NoPartition(t *testing.T) {
	got := rowCountPositiveSQL("dq-local.dq_fixture.customer", "")
	want := "SELECT COUNT(*) AS row_count FROM `dq-local.dq_fixture.customer`"
	if got != want {
		t.Errorf("SQL = %q; want %q", got, want)
	}
}

func TestRowCountPositiveSQL_WithPartition(t *testing.T) {
	// B2-12 retrofit: partition_column adds the half-open
	// interval predicate. The partition column references the
	// column directly (no wrapping function) so BigQuery's
	// partition pruning fires.
	got := rowCountPositiveSQL("dq-local.dq_fixture.events", "event_ts")
	want := "SELECT COUNT(*) AS row_count FROM `dq-local.dq_fixture.events` " +
		"WHERE `event_ts` >= @window_start AND `event_ts` < @window_end"
	if got != want {
		t.Errorf("partitioned SQL = %q; want %q", got, want)
	}
	if !strings.Contains(got, "@window_start") || !strings.Contains(got, "@window_end") {
		t.Error("partition SQL must reference both window-endpoint params")
	}
	// The partition column should appear as a backtick-quoted
	// identifier (not bare; not wrapped in DATE() or similar).
	if !strings.Contains(got, "`event_ts`") {
		t.Error("partition column must be backtick-quoted")
	}
}

func TestValidateBQIdentifiers_PartitionColumn(t *testing.T) {
	cases := []struct {
		name        string
		src         *runner.RuleSource
		wantErrSubs string // empty = expect nil error
	}{
		{
			name: "valid-no-partition",
			src: &runner.RuleSource{
				Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t",
			},
		},
		{
			name: "valid-with-partition",
			src: &runner.RuleSource{
				Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t",
				PartitionColumn: "event_ts",
			},
		},
		{
			name: "hyphen-in-partition-rejected",
			src: &runner.RuleSource{
				Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t",
				PartitionColumn: "event-ts",
			},
			wantErrSubs: "partition_column",
		},
		{
			name: "leading-digit-rejected",
			src: &runner.RuleSource{
				Type: "bigquery", ProjectID: "p", DatasetID: "d", TableID: "t",
				PartitionColumn: "1col",
			},
			wantErrSubs: "partition_column",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBQIdentifiers(tc.src)
			if tc.wantErrSubs == "" {
				if err != nil {
					t.Errorf("got err %v; want nil", err)
				}
				return
			}
			if err == nil {
				t.Errorf("got nil err; want error containing %q", tc.wantErrSubs)
				return
			}
			if !strings.Contains(err.Error(), tc.wantErrSubs) {
				t.Errorf("err %q should contain %q", err.Error(), tc.wantErrSubs)
			}
		})
	}
}
