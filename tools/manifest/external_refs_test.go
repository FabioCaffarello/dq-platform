// path: tools/manifest/external_refs_test.go

package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// buildExternalRefsFixture builds a temporary publisher fixture
// for ADR-0044 inlining tests. Layout:
//
//	tmp/
//	├─ rules/
//	│  ├─ <ruleFile>.yaml
//	│  ├─ _schema/
//	│  │  ├─ v2.schema.json         (permissive — body validation is the linter's job)
//	│  │  └─ catalog.v1.yaml        (declares schema as external-eligible)
//	│  └─ schemas/                  (created on demand by caller)
//	└─ (caller writes referenced artifacts under rules/schemas/)
//
// Returns rulesDir + schemaMirrorDir for Publisher Config.
func buildExternalRefsFixture(t *testing.T, ruleFile, ruleBody string) (rulesDir, schemaMirrorDir string) {
	t.Helper()
	tmp := t.TempDir()
	rulesDir = filepath.Join(tmp, "rules")
	schemaMirrorDir = filepath.Join(rulesDir, "_schema")
	if err := os.MkdirAll(schemaMirrorDir, 0o755); err != nil {
		t.Fatalf("mkdir schema mirror: %v", err)
	}
	// Permissive v2 schema — the linter validates rule shape;
	// the publisher only checks entity + version presence and
	// the ADR-0044 inlining. A pass-through schema keeps the
	// fixture compact.
	v2Schema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["version", "entity"],
		"properties": {
			"version": { "type": "integer" },
			"entity": { "type": "string" },
			"mode": { "type": "string" },
			"source": { "type": "object" },
			"checks": { "type": "array" }
		}
	}`
	if err := os.WriteFile(filepath.Join(schemaMirrorDir, "v1.schema.json"), []byte(v2Schema), 0o644); err != nil {
		t.Fatalf("write v1 schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(schemaMirrorDir, "v2.schema.json"), []byte(v2Schema), 0o644); err != nil {
		t.Fatalf("write v2 schema: %v", err)
	}
	// Catalog with record.schema_conformance declaring schema as
	// external-eligible — mirrors the real catalog v1's state.
	catalog := `catalog_version: 1
kinds:
  - name: record.schema_conformance
    mode: record
    source_mode: record
    external_eligible_fields:
      - schema
`
	if err := os.WriteFile(filepath.Join(schemaMirrorDir, "catalog.v1.yaml"), []byte(catalog), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, ruleFile), []byte(ruleBody), 0o644); err != nil {
		t.Fatalf("write rule: %v", err)
	}
	return rulesDir, schemaMirrorDir
}

// newExternalRefsPublisher builds a Publisher wired against a
// fresh memStore and the fixture's schema mirror.
func newExternalRefsPublisher(t *testing.T, schemaMirrorDir string) (*Publisher, *memStore) {
	t.Helper()
	store := newMemStore()
	p, err := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-v0.0.1",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         schemaMirrorDir,
		SupportedSchemaVersions: []int{1, 2},
		Now:                     func() time.Time { return time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("New publisher: %v", err)
	}
	return p, store
}

func TestPublish_ExternalRef_HappyPath_Inlined(t *testing.T) {
	// A rule with `schema_ref: schemas/orders.json` should
	// publish a YAML body where `params.schema` is the inlined
	// content and `schema_ref` is absent.
	rule := `version: 2
entity: orders_stream
mode: record
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema_ref: schemas/orders.json
`
	rulesDir, schemaMirrorDir := buildExternalRefsFixture(t, "orders_stream.yaml", rule)
	if err := os.MkdirAll(filepath.Join(rulesDir, "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "schemas", "orders.json"), []byte(`{"type":"object","required":["order_id"]}`), 0o644); err != nil {
		t.Fatalf("write referenced schema: %v", err)
	}

	p, store := newExternalRefsPublisher(t, schemaMirrorDir)
	_, err := p.Publish(context.Background(), Options{RulesDir: rulesDir})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Find the single published YAML body (by-hash) in the store.
	var yamlBody []byte
	for key, body := range store.objects {
		if strings.HasPrefix(key, "yamls/by-hash/") {
			yamlBody = body
			break
		}
	}
	if yamlBody == nil {
		t.Fatal("no yamls/by-hash/ entry found in store")
	}
	var doc map[string]any
	if err := yaml.Unmarshal(yamlBody, &doc); err != nil {
		t.Fatalf("parse published body: %v", err)
	}
	checks := doc["checks"].([]any)
	params := checks[0].(map[string]any)["params"].(map[string]any)
	if _, present := params["schema_ref"]; present {
		t.Errorf("published body retained `schema_ref`; want inlined-only")
	}
	schema, ok := params["schema"].(map[string]any)
	if !ok {
		t.Fatalf("params.schema not inlined as object; got %T = %v", params["schema"], params["schema"])
	}
	if schema["type"] != "object" {
		t.Errorf("inlined schema.type = %v; want object", schema["type"])
	}
}

func TestPublish_ExternalRef_BothPresent_Rejected(t *testing.T) {
	rule := `version: 2
entity: orders_stream
mode: record
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema:
        type: object
      schema_ref: schemas/orders.json
`
	rulesDir, schemaMirrorDir := buildExternalRefsFixture(t, "orders_stream.yaml", rule)
	if err := os.MkdirAll(filepath.Join(rulesDir, "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir schemas: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "schemas", "orders.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write referenced schema: %v", err)
	}

	p, _ := newExternalRefsPublisher(t, schemaMirrorDir)
	_, err := p.Publish(context.Background(), Options{RulesDir: rulesDir})
	if err == nil {
		t.Fatal("expected verification failure when both schema and schema_ref present")
	}
	if !errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "exactly one is permitted") {
		t.Errorf("err %v should mention mutual-exclusion", err)
	}
}

func TestPublish_ExternalRef_NonEligible_Rejected(t *testing.T) {
	rule := `version: 2
entity: orders_stream
mode: record
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema:
        type: object
      aggregation_ref: schemas/agg.json
`
	rulesDir, schemaMirrorDir := buildExternalRefsFixture(t, "orders_stream.yaml", rule)
	p, _ := newExternalRefsPublisher(t, schemaMirrorDir)
	_, err := p.Publish(context.Background(), Options{RulesDir: rulesDir})
	if err == nil {
		t.Fatal("expected verification failure for non-eligible aggregation_ref")
	}
	if !errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "not declared external-eligible") {
		t.Errorf("err %v should mention non-eligibility", err)
	}
}

func TestPublish_ExternalRef_UpwardTraversal_Rejected(t *testing.T) {
	rule := `version: 2
entity: orders_stream
mode: record
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema_ref: ../../../etc/passwd
`
	rulesDir, schemaMirrorDir := buildExternalRefsFixture(t, "orders_stream.yaml", rule)
	p, _ := newExternalRefsPublisher(t, schemaMirrorDir)
	_, err := p.Publish(context.Background(), Options{RulesDir: rulesDir})
	if err == nil {
		t.Fatal("expected verification failure for `..` reference")
	}
	if !errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "upward-traversal") {
		t.Errorf("err %v should mention upward-traversal", err)
	}
}

func TestPublish_ExternalRef_DeterministicAcrossPublishes(t *testing.T) {
	// ADR-0005's manifest-determinism contract: same source
	// inputs (rule file + referenced file contents) → same
	// manifest hash. The publish-time inlining preserves this
	// because the inlining function is deterministic — same
	// inputs produce identical inlined YAML bytes.
	rule := `version: 2
entity: orders_stream
mode: record
source:
  type: kafka
  topic: orders.events.v1
  consumer_group: dq-orders-stream
  window:
    type: tumbling
    duration: 1m
    lateness_tolerance: 30s
checks:
  - check_id: schema_present
    kind: record.schema_conformance
    params:
      schema_ref: schemas/orders.json
`
	schemaContent := map[string]any{
		"type":     "object",
		"required": []string{"order_id"},
	}
	schemaJSON, _ := json.Marshal(schemaContent)

	hashes := make([]string, 0, 2)
	for i := 0; i < 2; i++ {
		rulesDir, schemaMirrorDir := buildExternalRefsFixture(t, "orders_stream.yaml", rule)
		if err := os.MkdirAll(filepath.Join(rulesDir, "schemas"), 0o755); err != nil {
			t.Fatalf("mkdir schemas: %v", err)
		}
		if err := os.WriteFile(filepath.Join(rulesDir, "schemas", "orders.json"), schemaJSON, 0o644); err != nil {
			t.Fatalf("write referenced schema: %v", err)
		}
		p, _ := newExternalRefsPublisher(t, schemaMirrorDir)
		r, err := p.Publish(context.Background(), Options{RulesDir: rulesDir})
		if err != nil {
			t.Fatalf("Publish #%d: %v", i, err)
		}
		hashes = append(hashes, r.ManifestHash)
	}
	if hashes[0] != hashes[1] {
		t.Errorf("manifest hash not deterministic across publishes: %s vs %s\n"+
			"ADR-0005 + ADR-0044 require same inputs → same manifest hash.",
			hashes[0], hashes[1])
	}
}
