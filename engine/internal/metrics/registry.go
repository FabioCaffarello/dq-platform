// path: engine/internal/metrics/registry.go

// Package metrics owns the engine's Prometheus registry and the
// per-package Metrics structs that satisfy ADR-0039's eight-metric
// inventory. Per ADR-0055 §Clause 3 the package is the central
// inventory matching ADR-0039 §"Metric contract" verbatim;
// consuming packages (runner, loader) take handles via constructor
// injection mirroring the prevailing Logger / Publisher / Evaluator
// wiring.
//
// Two scheduler-side metrics from ADR-0039's inventory
// (dq_queue_depth, dq_scheduler_triggers_managed) are out of
// engine scope per ADR-0033 — they describe an external scheduler
// the engine binary cannot observe and are not exposed here. See
// ADR-0055 §Consequence 4.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry owns the *prometheus.Registry the /metrics route
// serves and the per-package Metrics structs the engine binary
// injects into consuming packages.
//
// One instance lives in the engine binary for its lifetime;
// tests construct their own via New.
type Registry struct {
	reg             *prometheus.Registry
	Runner          RunnerMetrics
	Loader          LoaderMetrics
	SchedulerProxy  SchedulerProxyMetrics
}

// New constructs a Registry with every metric from ADR-0039's
// inventory pre-registered.
func New() *Registry {
	reg := prometheus.NewRegistry()
	return &Registry{
		reg:             reg,
		Runner:          newRunnerMetrics(reg),
		Loader:          newLoaderMetrics(reg),
		SchedulerProxy:  newSchedulerProxyMetrics(reg),
	}
}

// Handler returns the http.Handler the engine binary mounts at
// /metrics per ADR-0039 §"Endpoint" + ADR-0055 §Clause 2.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

// Gatherer returns the underlying prometheus.Gatherer so tests
// can collect samples via prometheus/testutil helpers.
func (r *Registry) Gatherer() prometheus.Gatherer { return r.reg }

// RunnerMetrics holds the handles emitted by the runner package
// per ADR-0055 §Clause 4. The first five handles correspond to
// five of ADR-0039's eight inventory metrics; the remaining
// engine-side metric (dq_loader_refresh_failures_total) lives on
// LoaderMetrics, and the two scheduler-side metrics are external.
//
// The three commit-path handles (RecordCommitFailures,
// RecordCommitRetries, RecordCommitDuration) are added by
// ADR-0060 §Clause 1 — record-mode commit-path observability for
// the boundaries committed by ADR-0058 (commit-after-dispatch) +
// ADR-0059 (commit retry). They extend ADR-0039 along its
// §"Evolution rules" rule 1 additive-within-engine-major lane;
// only the record-mode RecordRunner emits them.
type RunnerMetrics struct {
	RunsTotal            *prometheus.CounterVec
	ChecksEvaluatedTotal *prometheus.CounterVec
	RunDurationSeconds   *prometheus.HistogramVec
	CheckDurationSeconds *prometheus.HistogramVec
	BytesScanned         *prometheus.GaugeVec

	// RecordCommitFailures counts cycles where commitWithRetry
	// exhausts the retry budget per ADR-0059 §Clause 5 and
	// closeAndDispatch falls through to ADR-0058 §Clause 2's
	// warning-log + skip path. Per ADR-0060 §Clause 5, returns
	// caused by context.Canceled / context.DeadlineExceeded are
	// excluded — operator-driven shutdown is not a failure mode.
	RecordCommitFailures *prometheus.CounterVec

	// RecordCommitRetries counts commitWithRetry cycles that
	// consumed at least one retry, broken down by terminal
	// outcome (success_after_retry | exhausted). First-attempt
	// success is the no-op-retry path and does not increment.
	RecordCommitRetries *prometheus.CounterVec

	// RecordCommitDuration observes per-attempt consumer.Commit
	// duration per ADR-0060 §Clause 2 (one observation per
	// individual Commit call, not per cycle). Per ADR-0060
	// §Clause 4, β bucket boundaries are prometheus.DefBuckets;
	// re-tuning is ADR-0060 OQ-1.
	RecordCommitDuration *prometheus.HistogramVec
}

func newRunnerMetrics(reg prometheus.Registerer) RunnerMetrics {
	m := RunnerMetrics{
		RunsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dq_runs_total",
			Help: "Count of terminal execution rows written by the engine. One increment per terminal row in dq_executions per ADR-0039.",
		}, []string{"entity", "status", "trigger_source", "mode"}),
		ChecksEvaluatedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dq_checks_evaluated_total",
			Help: "Count of check evaluations per ADR-0039.",
		}, []string{"entity", "check_id", "result", "mode"}),
		RunDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dq_run_duration_seconds",
			Help:    "Distribution of completed_at - started_at per terminal execution per ADR-0039.",
			Buckets: prometheus.DefBuckets,
		}, []string{"entity", "status", "mode"}),
		CheckDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dq_check_duration_seconds",
			Help:    "Per-check evaluator duration per ADR-0039.",
			Buckets: prometheus.DefBuckets,
		}, []string{"entity", "check_id", "mode"}),
		BytesScanned: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dq_bytes_scanned",
			Help: "Most-recent bytes-scanned value per (entity, check_id) per ADR-0039. Emits zero when the evidence_summary.bytes_scanned sub-field is absent per ADR-0055 §Clause 4 OQ-6 resolution (preserves time-series continuity).",
		}, []string{"entity", "check_id"}),
		RecordCommitFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dq_record_commit_failures_total",
			Help: "Count of record-mode commitWithRetry cycles that exhausted the retry budget and fell through to ADR-0058 §Clause 2's warning-log + skip path. Per ADR-0060 §Clause 5, context.Canceled / context.DeadlineExceeded returns are excluded (operator-driven shutdown is not a failure mode).",
		}, []string{"entity"}),
		RecordCommitRetries: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dq_record_commit_retries_total",
			Help: "Count of record-mode commitWithRetry cycles that consumed at least one retry per ADR-0060 §Clause 1, broken down by outcome: success_after_retry (commit succeeded on attempt > 1) or exhausted (all recordCommitMaxAttempts attempts failed).",
		}, []string{"entity", "outcome"}),
		RecordCommitDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "dq_record_commit_duration_seconds",
			Help:    "Distribution of per-attempt record-mode consumer.Commit duration per ADR-0060 §Clause 2 (one observation per individual Commit call, not per cycle). β buckets = prometheus.DefBuckets per ADR-0060 §Clause 4.",
			Buckets: prometheus.DefBuckets,
		}, []string{"entity"}),
	}
	reg.MustRegister(
		m.RunsTotal,
		m.ChecksEvaluatedTotal,
		m.RunDurationSeconds,
		m.CheckDurationSeconds,
		m.BytesScanned,
		m.RecordCommitFailures,
		m.RecordCommitRetries,
		m.RecordCommitDuration,
	)
	return m
}

// LoaderMetrics holds the loader-package emission handles per
// ADR-0055 §Clause 5.
type LoaderMetrics struct {
	RefreshFailuresTotal *prometheus.CounterVec
}

func newLoaderMetrics(reg prometheus.Registerer) LoaderMetrics {
	m := LoaderMetrics{
		RefreshFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "dq_loader_refresh_failures_total",
			Help: "Loader-refresh failures classified by ADR-0055 §Clause 5's error_class enum (pointer_read | body_fetch | hash_mismatch | parse_error | compatibility_contract), concretized from ADR-0007 §1's enumerated failure surface.",
		}, []string{"error_class"}),
	}
	reg.MustRegister(m.RefreshFailuresTotal)
	return m
}

// NoopRunnerMetrics returns a RunnerMetrics whose handles are
// registered against a throwaway registry. Safe for tests that
// do not assert emission; passing a RunnerMetrics zero value to
// the runner is a nil-deref hazard.
func NoopRunnerMetrics() RunnerMetrics {
	return newRunnerMetrics(prometheus.NewRegistry())
}

// NoopLoaderMetrics returns a LoaderMetrics whose handles are
// registered against a throwaway registry.
func NoopLoaderMetrics() LoaderMetrics {
	return newLoaderMetrics(prometheus.NewRegistry())
}

// SchedulerProxyMetrics holds the handles set by the engine's
// scheduler-proxy periodic loop per ADR-0056 §Clause 4. The two
// gauge handles map to the two scheduler-side metrics in
// ADR-0039's inventory; both carry the additive `source` label
// per ADR-0056 §Clause 2 so the engine's emission self-identifies
// as engine-derived without amending ADR-0039 §"Metric contract".
type SchedulerProxyMetrics struct {
	QueueDepth               *prometheus.GaugeVec
	SchedulerTriggersManaged *prometheus.GaugeVec
}

func newSchedulerProxyMetrics(reg prometheus.Registerer) SchedulerProxyMetrics {
	m := SchedulerProxyMetrics{
		QueueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dq_queue_depth",
			Help: "Count of runs the engine observes in the named state, scoped by source per ADR-0056 §Clause 2 (additive `source` label distinguishes engine-derived from scheduler-derived emission). The engine emits state=running from a partition-pruned dq_executions_current count per ADR-0056 §Clause 1; state=scheduled is engine-non-derivable and emits constant zero per ADR-0056 §Clause 3.",
		}, []string{"state", "source"}),
		SchedulerTriggersManaged: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dq_scheduler_triggers_managed",
			Help: "Count of triggers the engine observes in the named state, scoped by source per ADR-0056 §Clause 2. Engine-non-derivable in either state (healthy / errored) per ADR-0033 external-scheduler posture; emits constant zero per ADR-0056 §Clause 3.",
		}, []string{"state", "source"}),
	}
	reg.MustRegister(m.QueueDepth, m.SchedulerTriggersManaged)
	return m
}

// NoopSchedulerProxyMetrics returns a SchedulerProxyMetrics
// whose handles are registered against a throwaway registry.
// Safe for tests that do not assert emission.
func NoopSchedulerProxyMetrics() SchedulerProxyMetrics {
	return newSchedulerProxyMetrics(prometheus.NewRegistry())
}
