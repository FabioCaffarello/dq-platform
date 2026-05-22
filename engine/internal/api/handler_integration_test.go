// path: engine/internal/api/handler_integration_test.go

//go:build integration

// Integration tests for the W3-P4e HTTP trigger handler against
// the local Compose substrate. Bring the stack up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/api/...
//
// The test wires a real BigQuery store (results package) and a
// real runner with NoopEvaluator (Phase 4c parity — Phase 6
// wires real check resolution), spins up the HTTP server via
// httptest, POSTs a trigger, and verifies both the running and
// terminal rows land in dq_executions.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/loader"
	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

const (
	integrationProjectID = "dq-local"
	integrationEndpoint  = "http://localhost:9050"
	integrationRuleset   = "rules-v1.0.0"
)

func bqTestClient(t *testing.T) *bigquery.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := bigquery.NewClient(ctx, integrationProjectID,
		option.WithoutAuthentication(),
		option.WithEndpoint(integrationEndpoint),
	)
	if err != nil {
		t.Skipf("integration: cannot create BigQuery client (is `make up` running?): %v", err)
	}
	return cli
}

func uniqueDatasetID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("itest_api_%d", time.Now().UnixNano())
}

func makeStore(t *testing.T, cli *bigquery.Client) *results.BigQueryStore {
	t.Helper()
	ds := uniqueDatasetID(t)
	store := results.NewBigQueryStore(cli, integrationProjectID, ds, nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	return store
}

func makeRunner(t *testing.T, store results.Store) *runner.Runner {
	t.Helper()
	r, err := runner.New(runner.Config{
		Store:          store,
		EngineVersion:  "0.1.0",
		RulesetVersion: integrationRuleset,
	})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}
	return r
}

// startTestServer builds the api.Handler + api.Server and serves
// it over httptest. Returns the server's URL, the underlying
// completion channel (so the test can wait on the runner
// goroutine without polling the store), and a teardown func.
func startTestServer(t *testing.T, r *runner.Runner, m *loader.Manifest) (string, <-chan struct{}, func()) {
	t.Helper()
	complete := make(chan struct{}, 8)
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     r,
		ActiveManifest: func() *loader.Manifest { return m },
		EngineCtx:      context.Background(),
		OnComplete: func(_ string, _ error) {
			complete <- struct{}{}
		},
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	srv := NewServer(":0", h, nil)
	ts := httptest.NewServer(srv.server.Handler)
	return ts.URL, complete, ts.Close
}

func waitForRunnerDone(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatal("runner goroutine did not finish within 10s")
	}
}

// waitForStatus polls QueryCurrentExecution until the canonical
// view returns the expected status or the deadline expires.
// Mirrors the helper in runner_integration_test.go (different
// package, so copied locally).
func waitForStatus(t *testing.T, store *results.BigQueryStore, executionID string, want results.ExecutionStatus) *results.ExecutionRow {
	t.Helper()
	ctx := context.Background()
	deadline := time.Now().Add(10 * time.Second)
	var last *results.ExecutionRow
	for time.Now().Before(deadline) {
		row, err := store.QueryCurrentExecution(ctx, executionID)
		if err == nil && row.Status == want {
			return row
		}
		last = row
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s to reach %s; last seen: %+v", executionID, want, last)
	return nil
}

func TestIntegration_HTTPTrigger_AcceptedAndPersisted(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store := makeStore(t, cli)
	r := makeRunner(t, store)

	manifest := &loader.Manifest{
		ManifestVersion: 1,
		RulesetVersion:  integrationRuleset,
		Hash:            "integration-test-hash",
	}

	url, complete, teardown := startTestServer(t, r, manifest)
	defer teardown()

	body := []byte(`{
        "entity": "customer",
        "window_start": "2026-05-22T14:00:00Z",
        "window_end": "2026-05-22T15:00:00Z",
        "trigger_source": "scheduler"
    }`)

	resp, err := http.Post(url+"/v1/trigger", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /v1/trigger: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d; want 200", resp.StatusCode)
	}

	var dto TriggerHTTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dto.Status != "running" {
		t.Errorf("DTO Status = %q; want running", dto.Status)
	}
	if len(dto.ExecutionID) != 64 {
		t.Errorf("DTO ExecutionID length = %d; want 64", len(dto.ExecutionID))
	}

	// Wait for the runner goroutine to finalize, then assert the
	// canonical (latest-recorded_at) row reached a terminal
	// status. With zero checks ADR-0004 CC2 branch 2 maps to
	// StatusError ("trigger contained zero checks").
	waitForRunnerDone(t, complete)
	row := waitForStatus(t, store, dto.ExecutionID, results.StatusError)
	if row.Entity != "customer" {
		t.Errorf("canonical row Entity = %q; want customer", row.Entity)
	}
	if row.AttemptID != dto.AttemptID {
		t.Errorf("canonical row AttemptID = %q; DTO AttemptID = %q (mismatch)",
			row.AttemptID, dto.AttemptID)
	}
	if row.RulesetVersion != integrationRuleset {
		t.Errorf("canonical row RulesetVersion = %q; want %q",
			row.RulesetVersion, integrationRuleset)
	}
	if row.TriggerSource != results.TriggerScheduler {
		t.Errorf("canonical row TriggerSource = %q; want scheduler",
			row.TriggerSource)
	}
}

func TestIntegration_HTTPTrigger_OperatorRerunRejected(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store := makeStore(t, cli)
	r := makeRunner(t, store)
	manifest := &loader.Manifest{
		ManifestVersion: 1,
		RulesetVersion:  integrationRuleset,
	}
	url, _, teardown := startTestServer(t, r, manifest)
	defer teardown()

	body := []byte(`{
        "entity": "customer",
        "window_start": "2026-05-22T14:00:00Z",
        "window_end": "2026-05-22T15:00:00Z",
        "trigger_source": "operator-rerun"
    }`)

	resp, err := http.Post(url+"/v1/trigger", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /v1/trigger: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400", resp.StatusCode)
	}
	var env ErrorResponse
	_ = json.NewDecoder(resp.Body).Decode(&env)
	if env.Code != ErrCodeInvalidTriggerSrc {
		t.Errorf("err code = %q; want %q", env.Code, ErrCodeInvalidTriggerSrc)
	}
}

func TestIntegration_HTTPHealthAndReady(t *testing.T) {
	cli := bqTestClient(t)
	defer cli.Close()

	store := makeStore(t, cli)
	r := makeRunner(t, store)
	manifest := &loader.Manifest{RulesetVersion: integrationRuleset}
	url, _, teardown := startTestServer(t, r, manifest)
	defer teardown()

	for _, ep := range []string{"/healthz", "/readyz"} {
		resp, err := http.Get(url + ep)
		if err != nil {
			t.Fatalf("GET %s: %v", ep, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s status = %d; want 200", ep, resp.StatusCode)
		}
		resp.Body.Close()
	}
}
