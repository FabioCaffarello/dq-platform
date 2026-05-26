// path: tools/dryrun/dryrun.go

// Package main implements `dq-dryrun` per ADR-0029
// §"Compiler layer": walks the rules directory, compiles the SQL
// template for each set-mode BigQuery-source check, issues a
// BigQuery dry-run against the configured substrate, and reports
// (or enforces) the bytes-scanned estimate per rule.
//
// Today's catalog ships one set-mode kind whose SQL is
// dry-runnable: `set.row_count_positive`. Its template (see
// engine/internal/eval/row_count_positive.go) is
//
//	SELECT COUNT(*) AS row_count FROM `<project>.<dataset>.<table>`
//
// The binary compiles the same template, so a CI PR that
// introduces a typo in a rule's source descriptor or a new rule
// kind without a compiler-side template surfaces here before the
// runtime evaluator hits the cost ceiling.
//
// Substrates:
//
//   - Real BigQuery (default when no emulator host is set): the
//     dry-run estimate is authoritative; bytes-scanned ≤
//     `MaxBytesScannedPerRun` enforces ADR-0029's per-env ceiling.
//   - Local fake-BigQuery emulator (the local Docker Compose
//     stack): the emulator's dry-run fidelity is best-effort —
//     SQL-syntax + table-existence are caught, but the
//     bytes-scanned figure should NOT be treated as authoritative.
//     The CI lane runs against the emulator today; real-BQ
//     enforcement awaits the operational PR that wires CI
//     credentials.
package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"gopkg.in/yaml.v3"
)

// ErrCostCeilingExceeded is returned (wrapped) when a rule's
// dry-run estimate exceeds the configured MaxBytesScannedPerRun
// ceiling. The CLI maps it to exit code 1 (verification failure).
var ErrCostCeilingExceeded = errors.New("dryrun: cost ceiling exceeded")

// ErrCompileFailed is returned when a rule's source descriptor
// cannot produce a compileable SQL template (missing fields,
// invalid identifiers, unknown kind). Exit code 1.
var ErrCompileFailed = errors.New("dryrun: compile failed")

// Config configures a Runner.
type Config struct {
	// RulesDir is the directory walked for rule YAMLs. Required.
	RulesDir string

	// MaxBytesScannedPerRun enforces the ADR-0029 ceiling. Zero
	// disables enforcement (dry-runs still execute and report,
	// but no rule is rejected for cost). Match the operator's
	// target environment: 1 GB for local, 100 GB for qa, 1 TB
	// for prod.
	MaxBytesScannedPerRun int64

	// BigQueryClient is the connected client. Required.
	BigQueryClient *bigquery.Client

	// Logger is optional; defaults to a discarding logger.
	Logger *slog.Logger
}

// Runner walks RulesDir, dry-runs each compileable rule's SQL,
// and reports + optionally enforces the cost ceiling.
type Runner struct {
	cfg Config
}

// New constructs a Runner. Returns an error when RulesDir is
// empty or BigQueryClient is nil.
func New(cfg Config) (*Runner, error) {
	if cfg.RulesDir == "" {
		return nil, errors.New("dryrun: RulesDir is required")
	}
	if cfg.BigQueryClient == nil {
		return nil, errors.New("dryrun: BigQueryClient is required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}
	return &Runner{cfg: cfg}, nil
}

// Report is one Runner.Run result.
type Report struct {
	// Estimates is the per-rule, per-check dry-run estimate. One
	// entry per dry-run-able check encountered.
	Estimates []Estimate

	// Skipped is the per-rule list of skip reasons (rule was a
	// known non-dryrunnable shape — v1 schema, record-mode,
	// non-bigquery source, no checks). Surfaced in the CLI
	// output so the operator can see what was inspected vs
	// skipped.
	Skipped []SkipReason

	// TotalBytesProcessed is the sum of TotalBytesProcessed
	// across all Estimates. Useful for an aggregate-PR-cost
	// signal.
	TotalBytesProcessed int64
}

// Estimate is one rule + check dry-run result.
type Estimate struct {
	RulePath            string
	Entity              string
	CheckID             string
	Kind                string
	SQL                 string
	TotalBytesProcessed int64
}

// SkipReason explains why a rule (or check) was not dry-run.
type SkipReason struct {
	RulePath string
	Entity   string
	Reason   string
}

// Run walks the rules directory and dry-runs each compileable
// check. Returns the Report (always; non-nil even on error) plus
// an error wrapping ErrCostCeilingExceeded / ErrCompileFailed
// when applicable.
func (r *Runner) Run(ctx context.Context) (*Report, error) {
	rep := &Report{}
	walkErr := filepath.WalkDir(r.cfg.RulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == "_schema" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		if strings.HasPrefix(name, "_") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		return r.handleRule(ctx, path, body, rep)
	})
	if walkErr != nil {
		if errors.Is(walkErr, fs.ErrNotExist) {
			return rep, fmt.Errorf("dryrun: rules dir %s does not exist", r.cfg.RulesDir)
		}
		return rep, walkErr
	}
	return rep, nil
}

// ruleDoc is the minimum shape this binary needs to read from
// the rule YAML. Full validation is the linter's job; the
// dry-run binary trusts a previously-lint-clean tree.
type ruleDoc struct {
	Version int            `yaml:"version"`
	Entity  string         `yaml:"entity"`
	Mode    string         `yaml:"mode"`
	Source  ruleSource     `yaml:"source"`
	Checks  []ruleCheckDoc `yaml:"checks"`
}

type ruleSource struct {
	Type      string `yaml:"type"`
	ProjectID string `yaml:"project_id"`
	DatasetID string `yaml:"dataset_id"`
	TableID   string `yaml:"table_id"`
}

type ruleCheckDoc struct {
	CheckID string `yaml:"check_id"`
	Kind    string `yaml:"kind"`
}

func (r *Runner) handleRule(ctx context.Context, path string, body []byte, rep *Report) error {
	var doc ruleDoc
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Version < 2 {
		rep.Skipped = append(rep.Skipped, SkipReason{
			RulePath: path, Entity: doc.Entity,
			Reason: fmt.Sprintf("v%d schema has no source descriptor (dryrun requires v2+)", doc.Version),
		})
		return nil
	}
	if doc.Mode != "set" {
		rep.Skipped = append(rep.Skipped, SkipReason{
			RulePath: path, Entity: doc.Entity,
			Reason: fmt.Sprintf("mode=%q has no BigQuery query (dryrun handles set-mode bigquery sources today)", doc.Mode),
		})
		return nil
	}
	if doc.Source.Type != "bigquery" {
		rep.Skipped = append(rep.Skipped, SkipReason{
			RulePath: path, Entity: doc.Entity,
			Reason: fmt.Sprintf("source.type=%q (dryrun handles bigquery sources today)", doc.Source.Type),
		})
		return nil
	}
	if doc.Source.ProjectID == "" || doc.Source.DatasetID == "" || doc.Source.TableID == "" {
		return fmt.Errorf("%s entity %q: source descriptor missing project_id/dataset_id/table_id: %w",
			path, doc.Entity, ErrCompileFailed)
	}

	for _, c := range doc.Checks {
		sql, ok := compileSQL(c.Kind, doc.Source)
		if !ok {
			rep.Skipped = append(rep.Skipped, SkipReason{
				RulePath: path, Entity: doc.Entity,
				Reason: fmt.Sprintf("check %q kind %q has no compiler-side template yet (dryrun adds one when the runtime evaluator does)", c.CheckID, c.Kind),
			})
			continue
		}
		bytes, err := r.dryRun(ctx, sql)
		if err != nil {
			return fmt.Errorf("%s entity %q check %q kind %q: dry-run %w",
				path, doc.Entity, c.CheckID, c.Kind, err)
		}
		rep.Estimates = append(rep.Estimates, Estimate{
			RulePath:            path,
			Entity:              doc.Entity,
			CheckID:             c.CheckID,
			Kind:                c.Kind,
			SQL:                 sql,
			TotalBytesProcessed: bytes,
		})
		rep.TotalBytesProcessed += bytes
		if r.cfg.MaxBytesScannedPerRun > 0 && bytes > r.cfg.MaxBytesScannedPerRun {
			return fmt.Errorf("%s entity %q check %q kind %q: estimated %d bytes exceeds MaxBytesScannedPerRun=%d: %w",
				path, doc.Entity, c.CheckID, c.Kind, bytes, r.cfg.MaxBytesScannedPerRun, ErrCostCeilingExceeded)
		}
	}
	return nil
}

// compileSQL produces the SQL template for one (kind, source)
// combination. Mirrors engine/internal/eval/row_count_positive.go
// for the only set-mode kind shipping today. Returns (sql, true)
// on success; ("", false) when no template exists for the kind
// (caller logs a skip and proceeds).
func compileSQL(kind string, src ruleSource) (string, bool) {
	switch kind {
	case "set.row_count_positive":
		tableRef := fmt.Sprintf("%s.%s.%s", src.ProjectID, src.DatasetID, src.TableID)
		return fmt.Sprintf("SELECT COUNT(*) AS row_count FROM `%s`", tableRef), true
	}
	return "", false
}

// dryRun issues the BigQuery dry-run for one SQL string. Returns
// the TotalBytesProcessed estimate. Note: the BigQuery emulator
// commonly used in the local Compose stack does not faithfully
// implement dry-run; this function still returns nil-error on
// emulator runs but the bytes figure is unreliable.
func (r *Runner) dryRun(ctx context.Context, sql string) (int64, error) {
	q := r.cfg.BigQueryClient.Query(sql)
	q.DryRun = true
	job, err := q.Run(ctx)
	if err != nil {
		// Emulator does not support DryRun? Surface the error
		// verbatim — operator decides whether to bypass with
		// `-skip-emulator` or wait for real-BQ creds.
		return 0, err
	}
	status := job.LastStatus()
	if status == nil {
		// Drain the iterator to force a job-status check.
		it, ierr := job.Read(ctx)
		if ierr != nil {
			return 0, ierr
		}
		for {
			var row []bigquery.Value
			if err := it.Next(&row); err != nil {
				if errors.Is(err, iterator.Done) {
					break
				}
				return 0, err
			}
		}
		status = job.LastStatus()
	}
	if status == nil {
		return 0, errors.New("dryrun: job status unavailable")
	}
	stats := status.Statistics
	if stats == nil {
		return 0, errors.New("dryrun: job statistics unavailable")
	}
	return stats.TotalBytesProcessed, nil
}
