// path: tools/lint/lint_v2_test.go

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCatalog_OK(t *testing.T) {
	cat := loadCatalog(t)
	if cat.Version != 1 {
		t.Errorf("catalog version = %d; want 1", cat.Version)
	}
	if _, ok := cat.Kind("set.row_count_positive"); !ok {
		t.Errorf("expected set.row_count_positive in catalog")
	}
	if _, ok := cat.Kind("record.schema_conformance"); !ok {
		t.Errorf("expected record.schema_conformance in catalog")
	}
}

func TestLoadCatalog_MissingFile_ReturnsNil(t *testing.T) {
	cat, err := LoadCatalog("/no/such/catalog.yaml")
	if err != nil {
		t.Fatalf("LoadCatalog(missing) returned error %v; want nil", err)
	}
	if cat != nil {
		t.Errorf("LoadCatalog(missing) returned non-nil catalog: %+v", cat)
	}
}

func TestSchemaSet_DispatchesByVersion(t *testing.T) {
	set := schemaSet(t)
	if set.V1 == nil || set.V2 == nil {
		t.Fatalf("SchemaSet missing v1 or v2: v1=%v v2=%v", set.V1, set.V2)
	}
	// v1 fixture under testdata/valid/customer.yaml validates against v1.
	v1 := mustReadFile(t, "testdata/valid/customer.yaml")
	if errs := ValidateRuleBytes(set, v1); len(errs) != 0 {
		t.Errorf("v1 fixture failed: %v", errs)
	}
	// v2 fixture validates against v2.
	v2 := mustReadFile(t, "testdata/v2/valid/customer.yaml")
	if errs := ValidateRuleBytes(set, v2); len(errs) != 0 {
		t.Errorf("v2 fixture failed schema: %v", errs)
	}
}

func TestValidateRulesDir_V2_AllValid(t *testing.T) {
	// The v2 fixtures lint clean once owners (v2) and catalog are
	// passed in. customer.yaml (set-mode) + orders_stream.yaml
	// (record-mode) both pass schema + cross-checks #3–#8.
	set := schemaSet(t)
	cat := loadCatalog(t)
	owners := loadOwnersForV2(t, "testdata/v2/valid/_owners.yaml")
	results, processed, err := ValidateRulesDir(set, cat, owners, "testdata/v2/valid", false)
	if err != nil {
		t.Fatalf("ValidateRulesDir(v2 valid) op error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("ValidateRulesDir(v2 valid) returned errors: %v", results)
	}
	if processed < 2 {
		t.Errorf("processed = %d; want at least 2 v2 fixtures", processed)
	}
}

func TestValidateRulesDir_V2_AllInvalid(t *testing.T) {
	// Every fixture under testdata/v2/invalid/ must produce at least
	// one error. We don't lookup owners-mode for these — each fixture
	// is keyed on a different invariant so they don't need to be in
	// owners. Use the v2 valid owners as a lenient cross-check input.
	set := schemaSet(t)
	cat := loadCatalog(t)
	owners := loadOwnersForV2(t, "testdata/v2/valid/_owners.yaml")
	results, _, err := ValidateRulesDir(set, cat, owners, "testdata/v2/invalid", false)
	if err != nil {
		t.Fatalf("op error: %v", err)
	}
	expected := []string{
		"bad-kind-prefix.yaml",
		"unknown-kind.yaml",
		"source-mode-mismatch.yaml",
		"params-missing-required.yaml",
		"params-unknown-field.yaml",
		"schema-fails-no-mode.yaml",
		"schema-fails-bad-duration.yaml",
		"schedule-bad-grammar.yaml",
	}
	for _, want := range expected {
		found := false
		for path := range results {
			if strings.HasSuffix(path, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected validation error for %s; results = %v", want, results)
		}
	}
}

func TestV2CrossCheck_4_KindPrefixMismatch(t *testing.T) {
	results := lintV2File(t, "testdata/v2/invalid/bad-kind-prefix.yaml", "testdata/v2/valid/_owners.yaml")
	mustContain(t, results, "cross-check #4")
}

func TestV2CrossCheck_5_UnknownKind(t *testing.T) {
	results := lintV2File(t, "testdata/v2/invalid/unknown-kind.yaml", "testdata/v2/valid/_owners.yaml")
	mustContain(t, results, "cross-check #5")
}

func TestV2CrossCheck_7_SourceTypeMismatch(t *testing.T) {
	results := lintV2File(t, "testdata/v2/invalid/source-mode-mismatch.yaml", "testdata/v2/valid/_owners.yaml")
	mustContain(t, results, "cross-check #7")
}

func TestV2CrossCheck_12_NeitherSchemaNorRef(t *testing.T) {
	// After B2-32 + ADR-0044, the catalog no longer marks `schema`
	// as JSON-Schema-required (since `schema_ref` is an equivalent
	// alternative). Cross-check #12 enforces the "exactly one of
	// {schema, schema_ref}" obligation. This fixture was previously
	// named for the old cross-check #6 firing; it now exercises
	// cross-check #12's at-least-one branch.
	results := lintV2File(t, "testdata/v2/invalid/params-missing-required.yaml", "testdata/v2/valid/_owners.yaml")
	mustContain(t, results, "cross-check #12")
}

func TestV2CrossCheck_6_ParamsUnknownField(t *testing.T) {
	results := lintV2File(t, "testdata/v2/invalid/params-unknown-field.yaml", "testdata/v2/valid/_owners.yaml")
	mustContain(t, results, "cross-check #6")
}

func TestV2CrossCheck_3_RuleVsOwnersModeMismatch(t *testing.T) {
	// The owners file declares customer mode=record; the rule
	// declares mode=set. Cross-check #3 must fire.
	set := schemaSet(t)
	cat := loadCatalog(t)
	owners := loadOwnersForV2(t, "testdata/v2/cross-check/_owners.yaml")
	results, _, err := ValidateRulesDir(set, cat, owners, "testdata/v2/cross-check", false)
	if err != nil {
		t.Fatalf("op error: %v", err)
	}
	var combined string
	for _, errs := range results {
		for _, e := range errs {
			combined += e.Message + "\n"
		}
	}
	if !strings.Contains(combined, "cross-check #3") {
		t.Errorf("expected cross-check #3 in errors; got: %s", combined)
	}
}

func TestV2CrossCheck_CatalogNil_RejectsV2Rules(t *testing.T) {
	// When the catalog is missing, v2 rules can't resolve their
	// kinds — cross-check #5 still fires (it surfaces a clear
	// "no catalog loaded" message). The error path must not
	// silently let v2 rules through.
	set := schemaSet(t)
	owners := loadOwnersForV2(t, "testdata/v2/valid/_owners.yaml")
	results, _, err := ValidateRulesDir(set, nil, owners, "testdata/v2/valid", false)
	if err != nil {
		t.Fatalf("op error: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected v2 rules to fail when catalog is nil; got no errors")
	}
}

// TestV2Schema_ScheduleField_AcceptsBothForms asserts the
// schema's oneOf gates both ADR-0033 grammars: duration-literal
// (e.g., "1h") and 5-field cron (e.g., "0 * * * *"). The two
// valid fixtures are part of the testdata/v2/valid/ set so
// TestValidateRulesDir_V2_AllValid already exercises them
// indirectly; this targeted test confirms the schema path
// individually.
func TestV2Schema_ScheduleField_AcceptsBothForms(t *testing.T) {
	set := schemaSet(t)
	for _, fixture := range []string{
		"testdata/v2/valid/customer-with-schedule.yaml",
		"testdata/v2/valid/customer-with-cron-schedule.yaml",
	} {
		raw := mustReadFile(t, fixture)
		if errs := ValidateRuleBytes(set, raw); len(errs) != 0 {
			t.Errorf("%s: schema validation failed: %v", fixture, errs)
		}
	}
}

// TestV2Schema_ScheduleField_RejectsMalformed exercises the
// negative path: a 4-field-cron value (the only ambiguous-looking
// shape the oneOf intentionally rejects).
func TestV2Schema_ScheduleField_RejectsMalformed(t *testing.T) {
	set := schemaSet(t)
	raw := mustReadFile(t, "testdata/v2/invalid/schedule-bad-grammar.yaml")
	errs := ValidateRuleBytes(set, raw)
	if len(errs) == 0 {
		t.Fatal("expected schema validation error for malformed schedule; got none")
	}
	combined := ""
	for _, e := range errs {
		combined += e.Message + "\n"
	}
	if !strings.Contains(combined, "schedule") {
		t.Errorf("expected error to mention 'schedule'; got: %s", combined)
	}
}

// --- helpers ---

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("abs %s: %v", path, err)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read %s: %v", abs, err)
	}
	return raw
}

func loadOwnersForV2(t *testing.T, path string) *Owners {
	t.Helper()
	set := ownersSchemaSet(t)
	owners, valErrs, err := LoadOwners(set, path)
	if err != nil {
		t.Fatalf("LoadOwners(%s) op error: %v", path, err)
	}
	if len(valErrs) != 0 {
		t.Fatalf("LoadOwners(%s) validation errors: %v", path, valErrs)
	}
	return owners
}

// lintV2File runs a single-file lint pass through the directory
// walker by isolating the file in a temp dir. The cross-check fixture
// dir would not work for #4/#5/#6/#7 because those rules' entities
// are not declared in the cross-check owners file.
func lintV2File(t *testing.T, rulePath, ownersPath string) map[string][]ValidationError {
	t.Helper()
	set := schemaSet(t)
	cat := loadCatalog(t)
	owners := loadOwnersForV2(t, ownersPath)

	dir := t.TempDir()
	raw := mustReadFile(t, rulePath)
	target := filepath.Join(dir, filepath.Base(rulePath))
	mustWriteFile(t, target, raw)

	results, _, err := ValidateRulesDir(set, cat, owners, dir, false)
	if err != nil {
		t.Fatalf("ValidateRulesDir op error: %v", err)
	}
	return results
}

func mustContain(t *testing.T, results map[string][]ValidationError, substr string) {
	t.Helper()
	combined := ""
	for _, errs := range results {
		for _, e := range errs {
			combined += e.Message + "\n"
		}
	}
	if !strings.Contains(combined, substr) {
		t.Errorf("expected %q in errors; got: %s", substr, combined)
	}
}
