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
	reg    *prometheus.Registry
	Runner RunnerMetrics
	Loader LoaderMetrics
}

// New constructs a Registry with every metric from ADR-0039's
// inventory pre-registered.
func New() *Registry {
	reg := prometheus.NewRegistry()
	return &Registry{
		reg:    reg,
		Runner: newRunnerMetrics(reg),
		Loader: newLoaderMetrics(reg),
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
// per ADR-0055 §Clause 4. The five handles correspond to five of
// ADR-0039's eight inventory metrics; the remaining engine-side
// metric (dq_loader_refresh_failures_total) lives on
// LoaderMetrics, and the two scheduler-side metrics are external.
type RunnerMetrics struct {
	RunsTotal            *prometheus.CounterVec
	ChecksEvaluatedTotal *prometheus.CounterVec
	RunDurationSeconds   *prometheus.HistogramVec
	CheckDurationSeconds *prometheus.HistogramVec
	BytesScanned         *prometheus.GaugeVec
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
	}
	reg.MustRegister(
		m.RunsTotal,
		m.ChecksEvaluatedTotal,
		m.RunDurationSeconds,
		m.CheckDurationSeconds,
		m.BytesScanned,
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
