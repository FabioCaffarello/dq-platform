// path: tools/lint/lint_test.go

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// schemaPath returns the absolute path to the schema mirror. Tests
// load the canonical schema file under rules/_schema/v1.schema.json
// from the workspace root, using filepath.Abs to resolve relative
// to the package directory (tools/lint/ → repo root via "../..").
func schemaPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../rules/_schema/v1.schema.json")
	if err != nil {
		t.Fatalf("schemaPath: %v", err)
	}
	return p
}

// schemaV2Path returns the absolute path to the v2 schema mirror.
func schemaV2Path(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../rules/_schema/v2.schema.json")
	if err != nil {
		t.Fatalf("schemaV2Path: %v", err)
	}
	return p
}

// schemaSet builds a SchemaSet with both v1 and v2 schemas loaded
// from the canonical mirror paths.
func schemaSet(t *testing.T) *SchemaSet {
	t.Helper()
	set, err := LoadSchemaSet(schemaPath(t), schemaV2Path(t))
	if err != nil {
		t.Fatalf("LoadSchemaSet: %v", err)
	}
	return set
}

// catalogPath returns the absolute path to the v1 catalog mirror.
func catalogPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../rules/_schema/catalog.v1.yaml")
	if err != nil {
		t.Fatalf("catalogPath: %v", err)
	}
	return p
}

// loadCatalog loads the canonical catalog mirror for tests.
func loadCatalog(t *testing.T) *Catalog {
	t.Helper()
	cat, err := LoadCatalog(catalogPath(t))
	if err != nil {
		t.Fatalf("LoadCatalog: %v", err)
	}
	if cat == nil {
		t.Fatalf("LoadCatalog returned nil; catalog mirror missing at %s", catalogPath(t))
	}
	return cat
}

func TestLoadSchema_OK(t *testing.T) {
	if _, err := LoadSchema(schemaPath(t)); err != nil {
		t.Fatalf("LoadSchema returned %v; want nil", err)
	}
}

func TestLoadSchema_MissingFile(t *testing.T) {
	_, err := LoadSchema("/no/such/file/anywhere.json")
	if err == nil {
		t.Fatalf("LoadSchema(missing) returned nil; want operational error")
	}
}

func TestSchemaContainsPipeRejection(t *testing.T) {
	// Belt-and-suspenders: confirm the canonical schema's entity
	// pattern does not admit the pipe character. This guards
	// against a future schema edit that would silently disable
	// the ADR-0002 input-safety invariant.
	raw, err := os.ReadFile(schemaPath(t))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if err := assertEntityPipeRejected(raw); err != nil {
		t.Fatalf("entity pattern admits pipe character: %v", err)
	}
}

func TestValidateRule_ValidMinimal(t *testing.T) {
	schema, err := LoadSchema(schemaPath(t))
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	errs := ValidateRuleFile(schema, "testdata/valid/customer.yaml")
	if len(errs) > 0 {
		t.Fatalf("ValidateRuleFile(valid) returned %d errors: %v", len(errs), errs)
	}
}

func TestValidateRule_MissingVersion(t *testing.T) {
	expectInvalid(t, "testdata/invalid/no-version.yaml", "version")
}

func TestValidateRule_WrongVersion(t *testing.T) {
	// version: 2 is not yet supported; the schema's const: 1
	// rejects it.
	expectInvalid(t, "testdata/invalid/wrong-version.yaml", "version")
}

func TestValidateRule_EntityWithPipe(t *testing.T) {
	// Load-bearing test for the ADR-0002 input-safety invariant.
	expectInvalid(t, "testdata/invalid/pipe-entity.yaml", "entity")
}

func TestValidateRule_NoChecks(t *testing.T) {
	expectInvalid(t, "testdata/invalid/no-checks.yaml", "checks")
}

func TestValidateRule_AdditionalProperty(t *testing.T) {
	expectInvalid(t, "testdata/invalid/unknown-field.yaml", "additional")
}

func TestValidateRule_BadYAML(t *testing.T) {
	schema, err := LoadSchema(schemaPath(t))
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	// Inline malformed YAML — colon without key.
	errs := ValidateRule(schema, []byte(": not yaml :"))
	if len(errs) == 0 {
		t.Fatalf("ValidateRule(malformed) returned no errors")
	}
	if !strings.Contains(errs[0].Message, "yaml") {
		t.Fatalf("expected yaml parse error, got: %v", errs[0].Message)
	}
}

func TestValidateRulesDir_OnlyValid(t *testing.T) {
	set := schemaSet(t)
	results, processed, err := ValidateRulesDir(set, nil, nil, "testdata/valid", false)
	if err != nil {
		t.Fatalf("ValidateRulesDir(valid) operational error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("ValidateRulesDir(valid) returned %d errored files; want 0: %v", len(results), results)
	}
	if processed < 1 {
		t.Fatalf("ValidateRulesDir(valid) processed %d files; want at least 1", processed)
	}
}

func TestValidateRulesDir_AllInvalid(t *testing.T) {
	set := schemaSet(t)
	results, processed, err := ValidateRulesDir(set, nil, nil, "testdata/invalid", false)
	if err != nil {
		t.Fatalf("ValidateRulesDir(invalid) operational error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("ValidateRulesDir(invalid) returned 0 errored files; want all to fail")
	}
	if processed < len(results) {
		t.Errorf("ValidateRulesDir(invalid) processed=%d but errored=%d; processed should be >= errored", processed, len(results))
	}
	// Confirm every fixture is represented.
	expected := []string{"no-version.yaml", "wrong-version.yaml", "pipe-entity.yaml", "no-checks.yaml", "unknown-field.yaml"}
	for _, want := range expected {
		found := false
		for path := range results {
			if strings.HasSuffix(path, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected validation error for %s; not found in results: %v", want, results)
		}
	}
}

func TestValidateRulesDir_SkipsSchemaSubdir(t *testing.T) {
	// Confirm that the walker skips a path whose immediate
	// directory name is "_schema" — the rules mirror should
	// never be linted as if it were a rule.
	tmp := t.TempDir()
	mustWriteFile(t, filepath.Join(tmp, "_schema", "v1.schema.json"), []byte(`{"$id":"x"}`))
	mustWriteFile(t, filepath.Join(tmp, "should-not-lint.yaml"), []byte("not a rule"))

	set := schemaSet(t)
	results, _, err := ValidateRulesDir(set, nil, nil, tmp, false)
	if err != nil {
		t.Fatalf("ValidateRulesDir: %v", err)
	}
	for path := range results {
		if strings.Contains(path, "_schema") {
			t.Errorf("walker entered _schema/ — should be skipped: %s", path)
		}
	}
}

func TestValidateRulesDir_MissingDir(t *testing.T) {
	// Operational policy: missing rules directory is not an
	// error — the linter exits 0 with nothing to lint. This
	// lets `make lint-rules` succeed before Phase 6 lands the
	// first rule YAML.
	set := schemaSet(t)
	results, processed, err := ValidateRulesDir(set, nil, nil, "no-such-dir", false)
	if err != nil {
		t.Fatalf("ValidateRulesDir(missing) returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("ValidateRulesDir(missing) returned %d results; want 0", len(results))
	}
	if processed != 0 {
		t.Fatalf("ValidateRulesDir(missing) processed %d; want 0", processed)
	}
}

// --- helpers ---

func expectInvalid(t *testing.T, path, mustContain string) {
	t.Helper()
	schema, err := LoadSchema(schemaPath(t))
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	errs := ValidateRuleFile(schema, path)
	if len(errs) == 0 {
		t.Fatalf("ValidateRuleFile(%s) returned no errors; want failure mentioning %q", path, mustContain)
	}
	combined := ""
	for _, e := range errs {
		combined += e.Message + "\n"
	}
	if !strings.Contains(strings.ToLower(combined), strings.ToLower(mustContain)) {
		t.Errorf("ValidateRuleFile(%s) errors %q do not mention %q", path, combined, mustContain)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}
