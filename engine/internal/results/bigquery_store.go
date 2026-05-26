// path: engine/internal/results/bigquery_store.go

package results

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

// BigQueryStore implements Store against BigQuery (production) or
// against the bigquery-emulator (local Compose stack). The choice
// of endpoint is the caller's via storage.NewClient; this package
// is endpoint-agnostic.
//
// ADR-0003 CC1 commitment: this implementation never issues
// UPDATE or DELETE. Audit by grepping this file for those tokens;
// they appear only in comments.
//
// Note on streaming-insert semantics: WriteExecutionRow and
// WriteCheckResultRow use the BigQuery streaming-insert
// (tabledata.insertAll) surface, which is at-least-once by
// design. A retried insert under network failure can produce
// duplicate rows with identical composite keys. The append-only
// commitment is honored (no UPDATE/DELETE from engine code
// paths); duplicate-tolerant consumers either query
// dq_executions_current (canonical-view, duplicate-safe by
// recorded_at ordering) or de-dupe on the base table by
// composite key.
type BigQueryStore struct {
	client    *bigquery.Client
	projectID string
	datasetID string
	logger    *slog.Logger
}

// NewBigQueryStore wraps an existing *bigquery.Client. The caller
// is responsible for client lifecycle (the engine binary creates
// one at startup and closes it at shutdown).
//
// logger is used for non-fatal warnings — specifically,
// EnsureSchema emits a warning if the dq_executions_current view
// cannot be created (commodity-emulator fidelity gap per
// ADR-0010 lazy-view Partial row). If logger is nil, warnings
// are discarded; the engine binary (W3-P4c) injects a configured
// slog.Logger that routes to its observability backend per
// ADR-0007 CC14.
func NewBigQueryStore(client *bigquery.Client, projectID, datasetID string, logger *slog.Logger) *BigQueryStore {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &BigQueryStore{
		client:    client,
		projectID: projectID,
		datasetID: datasetID,
		logger:    logger,
	}
}

// EnsureSchema creates the dataset (if absent), the two append-only
// tables (if absent), and the lazy view (best-effort per ADR-0010
// lazy-view Partial row). Idempotent.
func (s *BigQueryStore) EnsureSchema(ctx context.Context) error {
	ds := s.client.DatasetInProject(s.projectID, s.datasetID)
	if err := ds.Create(ctx, &bigquery.DatasetMetadata{}); err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create dataset %s: %w", s.datasetID, err)
	}

	if err := s.ensureTable(ctx, tableExecutions, executionsSchema()); err != nil {
		return err
	}
	if err := s.ensureTable(ctx, tableCheckResults, checkResultsSchema()); err != nil {
		return err
	}

	// View creation is best-effort. The commodity emulator may not
	// support CREATE VIEW with window functions; in production the
	// view is mandatory per ADR-0003 CC2. The Go API's
	// QueryCurrentExecution uses the inline SQL regardless, so a
	// missing view does not break engine internals.
	//
	// Non-fatal: emit a structured warning so the engine binary's
	// observability layer (per ADR-0007 CC14) can surface the gap.
	// EnsureSchema still returns nil on view-creation failure —
	// the view is best-effort by ADR-0010 contract.
	if err := s.ensureView(ctx); err != nil {
		s.logger.Warn("dq_executions_current view creation failed; falling back to inline ROW_NUMBER query",
			"project", s.projectID,
			"dataset", s.datasetID,
			"view", viewExecutionsView,
			"error", err.Error(),
			"adr_reference", "ADR-0010 lazy-view Partial row",
		)
	}
	return nil
}

func (s *BigQueryStore) ensureTable(ctx context.Context, name string, schema bigquery.Schema) error {
	t := s.client.DatasetInProject(s.projectID, s.datasetID).Table(name)
	if err := t.Create(ctx, &bigquery.TableMetadata{Schema: schema}); err != nil && !isAlreadyExists(err) {
		return fmt.Errorf("create table %s: %w", name, err)
	}
	return nil
}

func (s *BigQueryStore) ensureView(ctx context.Context) error {
	dataset := s.fullyQualifiedDataset()
	ddl := strings.ReplaceAll(currentExecutionsViewSQL, "{{dataset}}", dataset)

	q := s.client.Query(ddl)
	q.QueryConfig.UseLegacySQL = false
	job, err := q.Run(ctx)
	if err != nil {
		return fmt.Errorf("submit view-create job: %w", err)
	}
	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("wait for view-create job: %w", err)
	}
	if err := status.Err(); err != nil {
		return fmt.Errorf("view-create job error: %w", err)
	}
	return nil
}

// WriteExecutionRow appends one row to dq_executions via the
// streaming-insert (tabledata.insertAll) surface per ADR-0003 CC1.
// Append-only; no UPDATE / DELETE.
func (s *BigQueryStore) WriteExecutionRow(ctx context.Context, row ExecutionRow) error {
	saver := &bigquery.StructSaver{
		Schema: executionsSchema(),
		Struct: toExecutionRecord(row),
	}
	inserter := s.client.DatasetInProject(s.projectID, s.datasetID).Table(tableExecutions).Inserter()
	if err := inserter.Put(ctx, saver); err != nil {
		return fmt.Errorf("insert dq_executions row: %w", err)
	}
	return nil
}

// WriteCheckResultRow appends one row to dq_check_results per
// ADR-0003 CC7. Append-only; no UPDATE / DELETE.
func (s *BigQueryStore) WriteCheckResultRow(ctx context.Context, row CheckResultRow) error {
	saver := &bigquery.StructSaver{
		Schema: checkResultsSchema(),
		Struct: toCheckResultRecord(row),
	}
	inserter := s.client.DatasetInProject(s.projectID, s.datasetID).Table(tableCheckResults).Inserter()
	if err := inserter.Put(ctx, saver); err != nil {
		return fmt.Errorf("insert dq_check_results row: %w", err)
	}
	return nil
}

// QueryCurrentExecution runs the inline ROW_NUMBER() OVER query
// (equivalent to the dq_executions_current view) so engine
// internals are portable across the emulator's lazy-view fidelity
// gap.
func (s *BigQueryStore) QueryCurrentExecution(ctx context.Context, executionID string) (*ExecutionRow, error) {
	dataset := s.fullyQualifiedDataset()
	sql := strings.ReplaceAll(currentExecutionsInlineSQL, "{{dataset}}", dataset)

	q := s.client.Query(sql)
	q.QueryConfig.UseLegacySQL = false
	q.Parameters = []bigquery.QueryParameter{
		{Name: "execution_id", Value: executionID},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("submit query for execution %s: %w", executionID, err)
	}

	var rec executionRecord
	if err := it.Next(&rec); err != nil {
		if errors.Is(err, iterator.Done) {
			return nil, fmt.Errorf("%s: %w", executionID, ErrExecutionNotFound)
		}
		return nil, fmt.Errorf("read query row for execution %s: %w", executionID, err)
	}
	return fromExecutionRecord(rec), nil
}

// ListRunningOlderThan returns the canonical row of every
// execution whose latest state is `running` and whose started_at
// is strictly before the given cutoff. Used by the orphan-run
// detector (engine/internal/orphan) per ADR-0007 CC11.
func (s *BigQueryStore) ListRunningOlderThan(ctx context.Context, before time.Time) ([]ExecutionRow, error) {
	dataset := s.fullyQualifiedDataset()
	sql := strings.ReplaceAll(runningOlderThanSQL, "{{dataset}}", dataset)

	q := s.client.Query(sql)
	q.QueryConfig.UseLegacySQL = false
	q.Parameters = []bigquery.QueryParameter{
		{Name: "before", Value: before.UTC()},
	}

	it, err := q.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("submit ListRunningOlderThan query: %w", err)
	}

	var rows []ExecutionRow
	for {
		var rec executionRecord
		if err := it.Next(&rec); err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}
			return nil, fmt.Errorf("read ListRunningOlderThan row: %w", err)
		}
		rows = append(rows, *fromExecutionRecord(rec))
	}
	if rows == nil {
		// Return an empty (non-nil) slice so callers can range
		// safely without a nil-vs-empty check.
		rows = []ExecutionRow{}
	}
	return rows, nil
}

func (s *BigQueryStore) fullyQualifiedDataset() string {
	return fmt.Sprintf("%s.%s", s.projectID, s.datasetID)
}

// isAlreadyExists detects the BigQuery API's "already exists"
// error so EnsureSchema can be idempotent. The SDK does not
// expose a typed error for this case; the string-match is the
// documented stable surface. We accept several phrasings because:
//   - production BigQuery uses "Already Exists" (HTTP 409);
//   - the bigquery-emulator uses "is already created" (HTTP 500 —
//     a known emulator quirk that does not affect production).
func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "already created") ||
		strings.Contains(msg, "alreadyexists") ||
		strings.Contains(msg, "duplicate")
}

// --- BigQuery struct ↔ ExecutionRow conversions ---

// executionRecord mirrors ExecutionRow with BigQuery-friendly
// struct tags. Kept private to this package so the public
// ExecutionRow remains tag-free (other Store impls may use
// different tag conventions).
type executionRecord struct {
	ExecutionID           string                 `bigquery:"execution_id"`
	AttemptID             string                 `bigquery:"attempt_id"`
	RecordedAt            time.Time              `bigquery:"recorded_at"`
	Status                string                 `bigquery:"status"`
	Mode                  string                 `bigquery:"mode"`
	EngineVersion         string                 `bigquery:"engine_version"`
	RulesetVersion        string                 `bigquery:"ruleset_version"`
	Entity                string                 `bigquery:"entity"`
	TriggerSource         string                 `bigquery:"trigger_source"`
	WindowStart           time.Time              `bigquery:"window_start"`
	WindowEnd             time.Time              `bigquery:"window_end"`
	StartedAt             bigquery.NullTimestamp `bigquery:"started_at"`
	CompletedAt           bigquery.NullTimestamp `bigquery:"completed_at"`
	ErrorSummary          bigquery.NullString    `bigquery:"error_summary"`
	SupersedesExecutionID bigquery.NullString    `bigquery:"supersedes_execution_id"`
}

func toExecutionRecord(r ExecutionRow) executionRecord {
	// Backfill default: rows that arrive without a mode value
	// default to set per the ADR-0021 backfill contract. The
	// runner always populates Mode for new rows, so this default
	// only fires for direct-callers / tests that omit the field.
	mode := string(r.Mode)
	if mode == "" {
		mode = string(ModeSet)
	}
	rec := executionRecord{
		ExecutionID:    r.ExecutionID,
		AttemptID:      r.AttemptID,
		RecordedAt:     r.RecordedAt,
		Status:         string(r.Status),
		Mode:           mode,
		EngineVersion:  r.EngineVersion,
		RulesetVersion: r.RulesetVersion,
		Entity:         r.Entity,
		TriggerSource:  string(r.TriggerSource),
		WindowStart:    r.WindowStart,
		WindowEnd:      r.WindowEnd,
	}
	if r.StartedAt != nil {
		rec.StartedAt = bigquery.NullTimestamp{Timestamp: *r.StartedAt, Valid: true}
	}
	if r.CompletedAt != nil {
		rec.CompletedAt = bigquery.NullTimestamp{Timestamp: *r.CompletedAt, Valid: true}
	}
	if r.ErrorSummary != nil {
		rec.ErrorSummary = bigquery.NullString{StringVal: *r.ErrorSummary, Valid: true}
	}
	if r.SupersedesExecutionID != nil {
		rec.SupersedesExecutionID = bigquery.NullString{StringVal: *r.SupersedesExecutionID, Valid: true}
	}
	return rec
}

func fromExecutionRecord(rec executionRecord) *ExecutionRow {
	mode := Mode(rec.Mode)
	if mode == "" {
		mode = ModeSet
	}
	row := &ExecutionRow{
		ExecutionID:    rec.ExecutionID,
		AttemptID:      rec.AttemptID,
		RecordedAt:     rec.RecordedAt,
		Status:         ExecutionStatus(rec.Status),
		Mode:           mode,
		EngineVersion:  rec.EngineVersion,
		RulesetVersion: rec.RulesetVersion,
		Entity:         rec.Entity,
		TriggerSource:  TriggerSource(rec.TriggerSource),
		WindowStart:    rec.WindowStart,
		WindowEnd:      rec.WindowEnd,
	}
	if rec.StartedAt.Valid {
		t := rec.StartedAt.Timestamp
		row.StartedAt = &t
	}
	if rec.CompletedAt.Valid {
		t := rec.CompletedAt.Timestamp
		row.CompletedAt = &t
	}
	if rec.ErrorSummary.Valid {
		s := rec.ErrorSummary.StringVal
		row.ErrorSummary = &s
	}
	if rec.SupersedesExecutionID.Valid {
		s := rec.SupersedesExecutionID.StringVal
		row.SupersedesExecutionID = &s
	}
	return row
}

// --- BigQuery struct ↔ CheckResultRow conversions ---

type checkResultRecord struct {
	ExecutionID         string    `bigquery:"execution_id"`
	AttemptID           string    `bigquery:"attempt_id"`
	CheckID             string    `bigquery:"check_id"`
	Result              string    `bigquery:"result"`
	ExecutedAt          time.Time `bigquery:"executed_at"`
	EngineVersion       string    `bigquery:"engine_version"`
	EvidenceSummary     string    `bigquery:"evidence_summary"`     // JSON-encoded
	SampleViolatingRows string    `bigquery:"sample_violating_rows"` // JSON-encoded array
}

func toCheckResultRecord(r CheckResultRow) checkResultRecord {
	return checkResultRecord{
		ExecutionID:         r.ExecutionID,
		AttemptID:           r.AttemptID,
		CheckID:             r.CheckID,
		Result:              string(r.Result),
		ExecutedAt:          r.ExecutedAt,
		EngineVersion:       r.EngineVersion,
		EvidenceSummary:     mustJSON(r.EvidenceSummary),
		SampleViolatingRows: mustJSONArray(r.SampleViolatingRows),
	}
}

// mustJSON encodes the value as JSON; nil maps marshal to "null"
// which BigQuery's JSON column accepts (null vs empty object are
// distinct; nil maps choose null). Returns the empty string for a
// truly nil pointer (vs nil map) only — callers can rely on
// non-empty output for non-nil inputs.
func mustJSON(v map[string]any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		// Map[string]any with json-friendly leaf types should never
		// fail. If it does, the row is unwriteable — log via the
		// resulting error at insert time rather than panicking here.
		return fmt.Sprintf(`{"_marshal_error":%q}`, err.Error())
	}
	return string(b)
}

// mustJSONArray is the map[string]any-array variant for
// SampleViolatingRows.
func mustJSONArray(v []map[string]any) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`[{"_marshal_error":%q}]`, err.Error())
	}
	return string(b)
}
