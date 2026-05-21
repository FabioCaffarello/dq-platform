// path: tools/lint/owners_test.go

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// ownersSchemaPath returns the absolute path to the _owners.yaml
// schema mirror.
func ownersSchemaPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../rules/_schema/_owners.v1.schema.json")
	if err != nil {
		t.Fatalf("ownersSchemaPath: %v", err)
	}
	return p
}

func TestLoadOwnersSchema_OK(t *testing.T) {
	if _, err := LoadOwnersSchema(ownersSchemaPath(t)); err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
}

func TestLoadOwners_HappyPath(t *testing.T) {
	schema, err := LoadOwnersSchema(ownersSchemaPath(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
	owners, valErrs, err := LoadOwners(schema, "testdata/owners/valid/_owners.yaml")
	if err != nil {
		t.Fatalf("LoadOwners: %v", err)
	}
	if len(valErrs) != 0 {
		t.Fatalf("LoadOwners(valid) validation errors = %v; want none", valErrs)
	}
	if _, ok := owners.Entities["customer"]; !ok {
		t.Errorf("customer entity not loaded: %v", owners.Entities)
	}
	if _, ok := owners.Entities["account"]; !ok {
		t.Errorf("account entity not loaded: %v", owners.Entities)
	}
}

func TestLoadOwners_MissingFile_NoError(t *testing.T) {
	// ADR-0006 CC9: missing _owners.yaml is not an operational
	// error; the linter still rejects ownerless rules in the
	// cross-check phase (CheckRulesHaveOwners).
	schema, err := LoadOwnersSchema(ownersSchemaPath(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
	owners, valErrs, err := LoadOwners(schema, "/no/such/_owners.yaml")
	if err != nil {
		t.Fatalf("LoadOwners(missing) op error = %v; want nil", err)
	}
	if len(valErrs) != 0 {
		t.Fatalf("LoadOwners(missing) validation errors = %v; want none", valErrs)
	}
	if len(owners.Entities) != 0 {
		t.Errorf("LoadOwners(missing) entities = %v; want empty", owners.Entities)
	}
}

func TestLoadOwners_InvalidSchema_MissingChannels(t *testing.T) {
	schema, err := LoadOwnersSchema(ownersSchemaPath(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
	_, valErrs, err := LoadOwners(schema, "testdata/owners/invalid/missing-channels.yaml")
	if err != nil {
		t.Fatalf("LoadOwners op error = %v; want nil", err)
	}
	if len(valErrs) == 0 {
		t.Fatalf("LoadOwners(missing-channels) returned no validation errors; want at least one")
	}
	combined := strings.ToLower(combineErrs(valErrs))
	if !strings.Contains(combined, "channels") {
		t.Errorf("validation errors %q should mention 'channels'", combined)
	}
}

func TestLoadOwners_InvalidSchema_BadChannelFormat(t *testing.T) {
	schema, err := LoadOwnersSchema(ownersSchemaPath(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
	_, valErrs, err := LoadOwners(schema, "testdata/owners/invalid/bad-channel-format.yaml")
	if err != nil {
		t.Fatalf("LoadOwners op error = %v; want nil", err)
	}
	if len(valErrs) == 0 {
		t.Fatalf("LoadOwners(bad-channel-format) returned no validation errors; want one for pattern mismatch")
	}
}

func TestLoadOwners_InvalidSchema_WrongVersion(t *testing.T) {
	schema, err := LoadOwnersSchema(ownersSchemaPath(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
	_, valErrs, err := LoadOwners(schema, "testdata/owners/invalid/wrong-version.yaml")
	if err != nil {
		t.Fatalf("LoadOwners op error = %v; want nil", err)
	}
	if len(valErrs) == 0 {
		t.Fatalf("LoadOwners(wrong-version) returned no validation errors; want one for const mismatch")
	}
}

func TestCheckRulesHaveOwners_HappyAndMissing(t *testing.T) {
	// The cross-check fixture has two rules: covered.yaml (entity
	// in _owners.yaml) and orphaned.yaml (entity NOT in
	// _owners.yaml). The check must reject orphaned and accept
	// covered.
	ownersSchema, err := LoadOwnersSchema(ownersSchemaPath(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
	owners, valErrs, err := LoadOwners(ownersSchema, "testdata/owners/cross-check/_owners.yaml")
	if err != nil || len(valErrs) != 0 {
		t.Fatalf("LoadOwners(cross-check) returned err=%v valErrs=%v", err, valErrs)
	}
	results, err := CheckRulesHaveOwners(owners, "testdata/owners/cross-check")
	if err != nil {
		t.Fatalf("CheckRulesHaveOwners: %v", err)
	}
	// Expect exactly one entry (orphaned.yaml). covered.yaml has
	// no errors so is not in the map.
	if len(results) != 1 {
		t.Fatalf("results = %d entries; want 1: %v", len(results), results)
	}
	var orphanedPath string
	for p := range results {
		orphanedPath = p
	}
	if !strings.HasSuffix(orphanedPath, "orphaned.yaml") {
		t.Errorf("expected orphaned.yaml in results; got %q", orphanedPath)
	}
	combined := combineErrs(results[orphanedPath])
	if !strings.Contains(combined, "ADR-0006 CC9") {
		t.Errorf("error %q should cite ADR-0006 CC9", combined)
	}
	if !strings.Contains(combined, "orphaned") {
		t.Errorf("error %q should mention the entity name 'orphaned'", combined)
	}
}

func TestCheckRulesHaveOwners_OwnersMissingWithRulesPresent(t *testing.T) {
	// ADR-0006 CC9 enforcement: if rules exist but _owners.yaml is
	// absent (empty owners set), report a single top-level error.
	emptyOwners := &Owners{Entities: map[string]struct{}{}}
	results, err := CheckRulesHaveOwners(emptyOwners, "testdata/valid")
	if err != nil {
		t.Fatalf("CheckRulesHaveOwners: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("CheckRulesHaveOwners with rules-but-no-owners returned no error; want one ADR-0006 CC9 error")
	}
	// The error key should be the rulesDir itself.
	found := false
	for p, errs := range results {
		if strings.HasSuffix(p, "testdata/valid") {
			found = true
			combined := combineErrs(errs)
			if !strings.Contains(combined, "ADR-0006 CC9") {
				t.Errorf("top-level error %q should cite ADR-0006 CC9", combined)
			}
		}
	}
	if !found {
		t.Errorf("expected top-level error keyed on rules dir; got %v", results)
	}
}

func TestCheckRulesHaveOwners_NoRulesAndNoOwners_OK(t *testing.T) {
	// Pre-Phase-6 state: no rules, no owners → no errors.
	emptyOwners := &Owners{Entities: map[string]struct{}{}}
	results, err := CheckRulesHaveOwners(emptyOwners, t.TempDir())
	if err != nil {
		t.Fatalf("CheckRulesHaveOwners: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("no-rules/no-owners returned %d errors; want 0: %v", len(results), results)
	}
}

func TestCheckRulesHaveOwners_MissingDir_OK(t *testing.T) {
	// Walking a missing rules directory is treated as no rules.
	emptyOwners := &Owners{Entities: map[string]struct{}{}}
	results, err := CheckRulesHaveOwners(emptyOwners, "/no/such/rules/dir")
	if err != nil {
		t.Fatalf("CheckRulesHaveOwners(missing dir): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("missing rules dir returned %d errors; want 0", len(results))
	}
}

func combineErrs(errs []ValidationError) string {
	out := ""
	for _, e := range errs {
		out += e.Message + "\n"
	}
	return out
}
