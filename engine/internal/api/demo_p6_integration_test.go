// path: engine/internal/api/demo_p6_integration_test.go

//go:build integration

// W3-P6d end-to-end integration test. Closes the W2-3 C-W2-3.4
// invariant from the CI lane (the shell counterpart at
// scripts/smoke/demo-p6.sh is the human-visible artifact).
//
// Bring the Compose substrate up first:
//
//	make up
//	cd engine && go test -tags integration ./internal/api/...
//
// The test:
//
//  1. Constructs a fresh BigQuery dataset for dq_executions /
//     dq_check_results, a fresh source dataset with a `customer`
//     table populated with three rows.
//  2. Constructs a loader.GCSStore against the fake-gcs-server
//     bucket, writes a rule YAML body at yamls/by-hash/<hex>.yaml,
//     and builds a *loader.Manifest in memory pointing at it.
//  3. Wires a runner.Runner with eval.Evaluator, then a
//     api.Handler with the ResolveChecks closure (same shape as
//     the engine binary's main.go).
//  4. Spins up httptest.NewServer, POSTs a trigger for the
//     customer entity, waits for the runner goroutine to finish
//     via OnComplete, and asserts the persisted dq_executions
//     terminal row reaches StatusSuccess.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"dq-platform/engine/internal/dsl/spec"
	"dq-platform/engine/internal/eval"
	"dq-platform/engine/internal/loader"
	"dq-platform/engine/internal/results"
	"dq-platform/engine/internal/runner"
)

const (
	demoProjectID   = "dq-local"
	demoBucket      = "dq-local"
	demoGCSEndpoint = "http://localhost:4443/storage/v1/"
	demoBQEndpoint  = "http://localhost:9050"
)

func demoBQClient(t *testing.T) *bigquery.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := bigquery.NewClient(ctx, demoProjectID,
		option.WithoutAuthentication(),
		option.WithEndpoint(demoBQEndpoint),
	)
	if err != nil {
		t.Skipf("integration: BigQuery emulator unreachable (is `make up` running?): %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func demoGCSClient(t *testing.T) *storage.Client {
	t.Helper()
	if host := os.Getenv("STORAGE_EMULATOR_HOST"); host == "" {
		_ = os.Setenv("STORAGE_EMULATOR_HOST", "localhost:4443")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := storage.NewClient(ctx,
		option.WithoutAuthentication(),
		option.WithEndpoint(demoGCSEndpoint),
	)
	if err != nil {
		t.Skipf("integration: GCS emulator unreachable (is `make up` running?): %v", err)
	}
	if _, err := cli.Bucket(demoBucket).Attrs(ctx); err != nil {
		_ = cli.Bucket(demoBucket).Create(ctx, demoProjectID, nil)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

func demoUniqueDataset(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("itest_p6d_%s_%d", prefix, time.Now().UnixNano())
}

func demoCreateSourceTable(t *testing.T, cli *bigquery.Client, dataset string) {
	t.Helper()
	ctx := context.Background()
	if err := cli.Dataset(dataset).Create(ctx, &bigquery.DatasetMetadata{}); err != nil {
		t.Fatalf("create source dataset %q: %v", dataset, err)
	}
	tbl := cli.Dataset(dataset).Table("customer")
	if err := tbl.Create(ctx, &bigquery.TableMetadata{
		Schema: bigquery.Schema{{Name: "id", Type: bigquery.IntegerFieldType, Required: true}},
	}); err != nil {
		t.Fatalf("create source table customer: %v", err)
	}
	type row struct {
		ID int64 `bigquery:"id"`
	}
	if err := tbl.Inserter().Put(ctx, []row{{ID: 1}, {ID: 2}, {ID: 3}}); err != nil {
		t.Fatalf("insert source rows: %v", err)
	}
}

func demoWriteYAML(t *testing.T, cli *storage.Client, key string, body []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w := cli.Bucket(demoBucket).Object(key).NewWriter(ctx)
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		t.Fatalf("write yaml object %q: %v", key, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close yaml writer %q: %v", key, err)
	}
}

// TestIntegration_DemoP6_EndToEnd closes the W2-3 C-W2-3.4
// invariant from the CI lane: manifest publish → loader-style
// in-memory manifest reference → handler resolves checks from
// the YAML body in the object store → runner → evaluator →
// BigQuery → dq_executions terminal row with StatusSuccess.
func TestIntegration_DemoP6_EndToEnd(t *testing.T) {
	bqCli := demoBQClient(t)
	gcsCli := demoGCSClient(t)

	sourceDS := demoUniqueDataset(t, "src")
	resultsDS := demoUniqueDataset(t, "res")

	demoCreateSourceTable(t, bqCli, sourceDS)

	// Write the rule YAML into the GCS emulator at a content-
	// addressed path. The test stores the same body the
	// manifest publisher would have written. Per ADR-0023 the
	// BigQuery target lives on the rule's `source:` descriptor,
	// so the body interpolates the dynamically-generated source
	// dataset name.
	yamlBody := []byte(fmt.Sprintf(`version: 2
entity: customer
mode: set
description: P6d integration test.
source:
  type: bigquery
  project_id: %s
  dataset_id: %s
  table_id: customer
checks:
  - check_id: row_count_positive
    kind: set.row_count_positive
`, demoProjectID, sourceDS))
	const yamlPath = "yamls/by-hash/sha256-demo-p6-fixture.yaml"
	demoWriteYAML(t, gcsCli, yamlPath, yamlBody)

	// In-memory manifest with one rule pointing at the YAML.
	// Mirrors what the loader would hold after a successful load.
	manifest := &loader.Manifest{
		ManifestVersion: 1,
		RulesetVersion:  "rules-p6d-itest-v0.1.0",
		Hash:            "p6d-itest-manifest-hash",
		Rules: []loader.ManifestRule{
			{Entity: "customer", YamlPath: yamlPath, YamlHash: "p6d-itest-yaml-hash"},
		},
	}
	gcsStore := loader.NewGCSStore(gcsCli, demoBucket)

	// Results store + EnsureSchema.
	store := results.NewBigQueryStore(bqCli, demoProjectID, resultsDS, nil)
	if err := store.EnsureSchema(context.Background()); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// Evaluator + runner. Per ADR-0023 the per-rule source
	// descriptor (parsed from the YAML body above) carries the
	// BigQuery target; the evaluator no longer pins a
	// deployment-wide source. ResultsProject / ResultsDataset
	// locate dq_executions + dq_check_results for the baseline
	// framework (ADR-0032) — set here for parity with main.go's
	// wiring even though set.row_count_positive does not consume
	// them.
	evaluator, err := eval.New(eval.Config{
		Client:         bqCli,
		ResultsProject: demoProjectID,
		ResultsDataset: resultsDS,
	})
	if err != nil {
		t.Fatalf("eval.New: %v", err)
	}
	r, err := runner.New(runner.Config{
		Store:          store,
		Evaluator:      evaluator,
		EngineVersion:  "0.1.0",
		RulesetVersion: manifest.RulesetVersion,
	})
	if err != nil {
		t.Fatalf("runner.New: %v", err)
	}

	// ResolveChecks closure (same shape as the engine binary's
	// main.go closure under cmd/dq-engine).
	resolveChecks := func(ctx context.Context, entity string) ([]runner.CheckSpec, error) {
		var rule *loader.ManifestRule
		for i := range manifest.Rules {
			if manifest.Rules[i].Entity == entity {
				rule = &manifest.Rules[i]
				break
			}
		}
		if rule == nil {
			return nil, ErrEntityNotInManifest
		}
		body, err := gcsStore.ReadObject(ctx, rule.YamlPath)
		if err != nil {
			return nil, fmt.Errorf("read yaml body %q: %w", rule.YamlPath, err)
		}
		parsed, err := spec.Parse(body)
		if err != nil {
			return nil, fmt.Errorf("parse yaml %q: %w", rule.YamlPath, err)
		}
		return parsed.ToCheckSpecs(), nil
	}

	// Handler + httptest server.
	complete := make(chan struct{}, 4)
	h, err := NewHandler(HandlerConfig{
		Dispatcher:     r,
		ActiveManifest: func() *loader.Manifest { return manifest },
		ResolveChecks:  resolveChecks,
		EngineCtx:      context.Background(),
		OnComplete:     func(_ string, _ error) { complete <- struct{}{} },
	})
	if err != nil {
		t.Fatalf("NewHandler: %v", err)
	}
	srv := NewServer(":0", h, nil)
	ts := httptest.NewServer(srv.server.Handler)
	defer ts.Close()

	// POST trigger.
	body := []byte(`{
        "entity": "customer",
        "window_start": "2026-05-22T14:00:00Z",
        "window_end": "2026-05-22T15:00:00Z",
        "trigger_source": "manual"
    }`)
	resp, err := http.Post(ts.URL+"/v1/trigger", "application/json", bytes.NewReader(body))
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

	// Wait for the runner goroutine to finalize.
	select {
	case <-complete:
	case <-time.After(15 * time.Second):
		t.Fatal("runner goroutine did not finish within 15s")
	}

	// Assert the canonical row reached StatusSuccess.
	deadline := time.Now().Add(10 * time.Second)
	var row *results.ExecutionRow
	for time.Now().Before(deadline) {
		got, err := store.QueryCurrentExecution(context.Background(), dto.ExecutionID)
		if err == nil && got.Status == results.StatusSuccess {
			row = got
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if row == nil {
		t.Fatalf("dq_executions canonical row for %s never reached success", dto.ExecutionID)
	}
	if row.Entity != "customer" {
		t.Errorf("Entity = %q; want customer", row.Entity)
	}
	if row.RulesetVersion != manifest.RulesetVersion {
		t.Errorf("RulesetVersion = %q; want %q", row.RulesetVersion, manifest.RulesetVersion)
	}
}
