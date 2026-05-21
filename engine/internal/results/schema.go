// path: engine/internal/results/schema.go

package results

import (
	"cloud.google.com/go/bigquery"
)

// Table names match the canonical contract committed by ADR-0003
// CC1. Changing them is a breaking change per ADR-0003 CC12.
const (
	tableExecutions    = "dq_executions"
	tableCheckResults  = "dq_check_results"
	viewExecutionsView = "dq_executions_current"
)

// executionsSchema mirrors ADR-0003 CC3. Nullable columns
// (started_at, completed_at, error_summary, supersedes_execution_id)
// are explicitly marked Required=false.
func executionsSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "execution_id", Type: bigquery.StringFieldType, Required: true,
			Description: "64-char lowercase hex per ADR-0002 CC7"},
		{Name: "attempt_id", Type: bigquery.StringFieldType, Required: true,
			Description: "UUID assigned by the trigger handler per ADR-0003 CC4"},
		{Name: "recorded_at", Type: bigquery.TimestampFieldType, Required: true,
			Description: "µs precision UTC per ADR-0003 CC3"},
		{Name: "status", Type: bigquery.StringFieldType, Required: true,
			Description: "ADR-0003 CC6: running|success|failed|error|aborted"},
		{Name: "engine_version", Type: bigquery.StringFieldType, Required: true,
			Description: "engine that wrote this row; visibility per ADR-0002 CC14"},
		{Name: "ruleset_version", Type: bigquery.StringFieldType, Required: true,
			Description: "manifest ruleset_version per ADR-0005 CC11; pipe-free per ADR-0002 CC2"},
		{Name: "entity", Type: bigquery.StringFieldType, Required: true},
		{Name: "trigger_source", Type: bigquery.StringFieldType, Required: true,
			Description: "ADR-0002 CC6: scheduler|manual|operator-rerun"},
		{Name: "started_at", Type: bigquery.TimestampFieldType, Required: false,
			Description: "nullable for the running transition row; required for terminal rows"},
		{Name: "completed_at", Type: bigquery.TimestampFieldType, Required: false,
			Description: "nullable for the running transition row; required for terminal rows"},
		{Name: "error_summary", Type: bigquery.StringFieldType, Required: false,
			Description: "populated when status is failed, error, or aborted"},
		{Name: "supersedes_execution_id", Type: bigquery.StringFieldType, Required: false,
			Description: "populated on operator-rerun first row per ADR-0003 CC5"},
	}
}

// checkResultsSchema mirrors ADR-0003 CC7. EvidenceSummary and
// SampleViolatingRows are JSON-typed because their per-check-kind
// shape evolves with the DSL grammar (Phase 4+ work).
func checkResultsSchema() bigquery.Schema {
	return bigquery.Schema{
		{Name: "execution_id", Type: bigquery.StringFieldType, Required: true},
		{Name: "attempt_id", Type: bigquery.StringFieldType, Required: true},
		{Name: "check_id", Type: bigquery.StringFieldType, Required: true},
		{Name: "result", Type: bigquery.StringFieldType, Required: true,
			Description: "ADR-0003 CC7 / ADR-0004 CC1: pass|fail|degraded|error"},
		{Name: "executed_at", Type: bigquery.TimestampFieldType, Required: true},
		{Name: "engine_version", Type: bigquery.StringFieldType, Required: true},
		{Name: "evidence_summary", Type: bigquery.JSONFieldType, Required: false,
			Description: "aggregate counts; per-check-kind shape lands in Phase 4+"},
		{Name: "sample_violating_rows", Type: bigquery.JSONFieldType, Required: false,
			Description: "repeated record capped per evidence-retention policy"},
	}
}

// currentExecutionsViewSQL is the DDL for dq_executions_current
// per ADR-0003 CC2. The view returns the row with the latest
// recorded_at per execution_id; it is lazy (computed at query
// time) so it always reflects the current state of the base table.
//
// The {{dataset}} placeholder is replaced by the BigQuery store
// at view-creation time with the configured project + dataset
// identifier.
//
// ADR-0010's "Tabular store: lazy view" row is Partial — the
// commodity emulator may not support window functions in views
// faithfully. EnsureSchema treats view creation as best-effort
// and the Go API's QueryCurrentExecution runs an inline
// equivalent so engine internals do not depend on view existence.
const currentExecutionsViewSQL = `
CREATE VIEW IF NOT EXISTS ` + "`{{dataset}}." + viewExecutionsView + "`" + ` AS
SELECT * EXCEPT (rn)
FROM (
  SELECT *,
         ROW_NUMBER() OVER (
           PARTITION BY execution_id
           ORDER BY recorded_at DESC
         ) AS rn
  FROM ` + "`{{dataset}}." + tableExecutions + "`" + `
)
WHERE rn = 1
`

// runningOlderThanSQL is the query the Go API uses for
// ListRunningOlderThan (orphan-run detection per ADR-0007 CC11).
// It returns the canonical row of every execution whose latest
// state is `running` and whose started_at is strictly before
// @before — same canonical-view semantics as
// currentExecutionsInlineSQL but multi-row, filtered.
//
// Critically, the outer WHERE clause runs **after** the
// ROW_NUMBER() OVER projection, so an execution whose latest
// row is a terminal status (aborted, success, etc.) is excluded
// even if an earlier `running` row has a stale started_at. This
// is the load-bearing semantic that prevents the orphan detector
// from re-finalizing already-finalized rows.
//
// The {{dataset}} placeholder is replaced with the configured
// project + dataset identifier; @before is bound at query time.
const runningOlderThanSQL = `
SELECT execution_id, attempt_id, recorded_at, status,
       engine_version, ruleset_version, entity, trigger_source,
       started_at, completed_at, error_summary,
       supersedes_execution_id
FROM (
  SELECT *,
         ROW_NUMBER() OVER (
           PARTITION BY execution_id
           ORDER BY recorded_at DESC
         ) AS rn
  FROM ` + "`{{dataset}}." + tableExecutions + "`" + `
)
WHERE rn = 1
  AND status = 'running'
  AND started_at IS NOT NULL
  AND started_at < @before
`

// currentExecutionsInlineSQL is the query the Go API uses for
// QueryCurrentExecution. Same semantics as the view; portable
// across the emulator's view fidelity gap.
//
// The {{dataset}} placeholder is replaced with the configured
// project + dataset identifier; the @execution_id parameter is
// bound at query time.
const currentExecutionsInlineSQL = `
SELECT execution_id, attempt_id, recorded_at, status,
       engine_version, ruleset_version, entity, trigger_source,
       started_at, completed_at, error_summary,
       supersedes_execution_id
FROM (
  SELECT *,
         ROW_NUMBER() OVER (
           PARTITION BY execution_id
           ORDER BY recorded_at DESC
         ) AS rn
  FROM ` + "`{{dataset}}." + tableExecutions + "`" + `
  WHERE execution_id = @execution_id
)
WHERE rn = 1
LIMIT 1
`
