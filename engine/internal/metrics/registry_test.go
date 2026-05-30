// path: engine/internal/metrics/registry_test.go

package metrics_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"dq-platform/engine/internal/metrics"
)

func TestNew_RegistersEveryAdr0039EngineSideMetric(t *testing.T) {
	r := metrics.New()

	r.Runner.RunsTotal.WithLabelValues("orders", "success", "scheduler", "set").Inc()
	r.Runner.ChecksEvaluatedTotal.WithLabelValues("orders", "row_count_positive", "pass", "set").Inc()
	r.Runner.RunDurationSeconds.WithLabelValues("orders", "success", "set").Observe(0.42)
	r.Runner.CheckDurationSeconds.WithLabelValues("orders", "row_count_positive", "set").Observe(0.12)
	r.Runner.BytesScanned.WithLabelValues("orders", "row_count_positive").Set(1024)
	r.Loader.RefreshFailuresTotal.WithLabelValues("pointer_read").Inc()

	cases := []struct {
		name      string
		collector prometheus.Collector
		want      int
	}{
		{"dq_runs_total", r.Runner.RunsTotal, 1},
		{"dq_checks_evaluated_total", r.Runner.ChecksEvaluatedTotal, 1},
		{"dq_run_duration_seconds", r.Runner.RunDurationSeconds, 1},
		{"dq_check_duration_seconds", r.Runner.CheckDurationSeconds, 1},
		{"dq_bytes_scanned", r.Runner.BytesScanned, 1},
		{"dq_loader_refresh_failures_total", r.Loader.RefreshFailuresTotal, 1},
	}
	for _, tc := range cases {
		if got := testutil.CollectAndCount(tc.collector, tc.name); got != tc.want {
			t.Errorf("metric %q: collected %d samples, want %d", tc.name, got, tc.want)
		}
	}
}

func TestHandler_ServesPrometheusExpositionFormat(t *testing.T) {
	r := metrics.New()
	r.Runner.RunsTotal.WithLabelValues("orders", "success", "scheduler", "set").Add(3)

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") && !strings.HasPrefix(ct, "application/openmetrics-text") {
		t.Errorf("Content-Type: %q does not look like Prometheus exposition format", ct)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), `dq_runs_total{entity="orders",mode="set",status="success",trigger_source="scheduler"} 3`) {
		t.Errorf("body missing expected dq_runs_total sample:\n%s", body)
	}
}

func TestNoopRunnerMetrics_DoesNotPanic(t *testing.T) {
	noop := metrics.NoopRunnerMetrics()
	// All five handles are usable — Inc / Observe / Set must not
	// panic even though the registry is throwaway.
	noop.RunsTotal.WithLabelValues("e", "success", "scheduler", "set").Inc()
	noop.ChecksEvaluatedTotal.WithLabelValues("e", "c", "pass", "set").Inc()
	noop.RunDurationSeconds.WithLabelValues("e", "success", "set").Observe(1)
	noop.CheckDurationSeconds.WithLabelValues("e", "c", "set").Observe(1)
	noop.BytesScanned.WithLabelValues("e", "c").Set(0)
}

func TestNoopLoaderMetrics_DoesNotPanic(t *testing.T) {
	noop := metrics.NoopLoaderMetrics()
	noop.RefreshFailuresTotal.WithLabelValues("pointer_read").Inc()
}

func TestSchedulerProxyMetrics_RegisteredAndSettable(t *testing.T) {
	r := metrics.New()

	// Engine-derivable: in-flight running count.
	r.SchedulerProxy.QueueDepth.WithLabelValues("running", "engine").Set(3)
	// Engine-non-derivable: constant zero per ADR-0056 §Clause 3.
	r.SchedulerProxy.QueueDepth.WithLabelValues("scheduled", "engine").Set(0)
	r.SchedulerProxy.SchedulerTriggersManaged.WithLabelValues("healthy", "engine").Set(0)
	r.SchedulerProxy.SchedulerTriggersManaged.WithLabelValues("errored", "engine").Set(0)

	if got := testutil.CollectAndCount(r.SchedulerProxy.QueueDepth, "dq_queue_depth"); got != 2 {
		t.Errorf("dq_queue_depth series count = %d; want 2", got)
	}
	if got := testutil.CollectAndCount(r.SchedulerProxy.SchedulerTriggersManaged, "dq_scheduler_triggers_managed"); got != 2 {
		t.Errorf("dq_scheduler_triggers_managed series count = %d; want 2", got)
	}
	if got := testutil.ToFloat64(r.SchedulerProxy.QueueDepth.WithLabelValues("running", "engine")); got != 3 {
		t.Errorf("dq_queue_depth{state=running,source=engine} = %v; want 3", got)
	}
	if got := testutil.ToFloat64(r.SchedulerProxy.QueueDepth.WithLabelValues("scheduled", "engine")); got != 0 {
		t.Errorf("dq_queue_depth{state=scheduled,source=engine} = %v; want 0", got)
	}
}

func TestSchedulerProxyMetrics_HandlerRendersWithSourceLabel(t *testing.T) {
	r := metrics.New()
	r.SchedulerProxy.QueueDepth.WithLabelValues("running", "engine").Set(5)

	srv := httptest.NewServer(r.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	want := `dq_queue_depth{source="engine",state="running"} 5`
	if !strings.Contains(string(body), want) {
		t.Errorf("body missing expected sample %q:\n%s", want, body)
	}
}

func TestNoopSchedulerProxyMetrics_DoesNotPanic(t *testing.T) {
	noop := metrics.NoopSchedulerProxyMetrics()
	noop.QueueDepth.WithLabelValues("running", "engine").Set(0)
	noop.SchedulerTriggersManaged.WithLabelValues("healthy", "engine").Set(0)
}
