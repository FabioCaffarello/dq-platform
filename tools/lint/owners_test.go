// path: tools/lint/owners_test.go

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// ownersSchemaPath returns the absolute path to the v1 _owners.yaml
// schema mirror.
func ownersSchemaPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../rules/_schema/_owners.v1.schema.json")
	if err != nil {
		t.Fatalf("ownersSchemaPath: %v", err)
	}
	return p
}

// ownersSchemaV2Path returns the absolute path to the v2 _owners.yaml
// schema mirror.
func ownersSchemaV2Path(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../rules/_schema/_owners.v2.schema.json")
	if err != nil {
		t.Fatalf("ownersSchemaV2Path: %v", err)
	}
	return p
}

// ownersSchemaSet builds an OwnersSchemaSet with both v1 and v2 owners
// schemas loaded from the canonical mirror paths.
func ownersSchemaSet(t *testing.T) *OwnersSchemaSet {
	t.Helper()
	set, err := LoadOwnersSchemaSet(ownersSchemaPath(t), ownersSchemaV2Path(t))
	if err != nil {
		t.Fatalf("LoadOwnersSchemaSet: %v", err)
	}
	return set
}

func TestLoadOwnersSchema_OK(t *testing.T) {
	if _, err := LoadOwnersSchema(ownersSchemaPath(t)); err != nil {
		t.Fatalf("LoadOwnersSchema: %v", err)
	}
}

func TestLoadOwners_HappyPath(t *testing.T) {
	set := ownersSchemaSet(t)
	owners, valErrs, err := LoadOwners(set, "testdata/owners/valid/_owners.yaml")
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
	set := ownersSchemaSet(t)
	owners, valErrs, err := LoadOwners(set, "/no/such/_owners.yaml")
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
	set := ownersSchemaSet(t)
	_, valErrs, err := LoadOwners(set, "testdata/owners/invalid/missing-channels.yaml")
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
	set := ownersSchemaSet(t)
	_, valErrs, err := LoadOwners(set, "testdata/owners/invalid/bad-channel-format.yaml")
	if err != nil {
		t.Fatalf("LoadOwners op error = %v; want nil", err)
	}
	if len(valErrs) == 0 {
		t.Fatalf("LoadOwners(bad-channel-format) returned no validation errors; want one for pattern mismatch")
	}
}

func TestLoadOwners_InvalidSchema_WrongVersion(t *testing.T) {
	set := ownersSchemaSet(t)
	_, valErrs, err := LoadOwners(set, "testdata/owners/invalid/wrong-version.yaml")
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
	set := ownersSchemaSet(t)
	owners, valErrs, err := LoadOwners(set, "testdata/owners/cross-check/_owners.yaml")
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
	emptyOwners := &Owners{Entities: map[string]OwnerEntity{}}
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
	emptyOwners := &Owners{Entities: map[string]OwnerEntity{}}
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
	emptyOwners := &Owners{Entities: map[string]OwnerEntity{}}
	results, err := CheckRulesHaveOwners(emptyOwners, "/no/such/rules/dir")
	if err != nil {
		t.Fatalf("CheckRulesHaveOwners(missing dir): %v", err)
	}
	if len(results) != 0 {
		t.Errorf("missing rules dir returned %d errors; want 0", len(results))
	}
}

func TestLoadOwners_V2_HappyPath(t *testing.T) {
	set := ownersSchemaSet(t)
	owners, valErrs, err := LoadOwners(set, "testdata/v2/valid/_owners.yaml")
	if err != nil {
		t.Fatalf("LoadOwners(v2 valid) op error: %v", err)
	}
	if len(valErrs) != 0 {
		t.Fatalf("LoadOwners(v2 valid) validation errors: %v", valErrs)
	}
	if owners.SchemaVersion != 2 {
		t.Errorf("SchemaVersion = %d; want 2", owners.SchemaVersion)
	}
	cust, ok := owners.Entities["customer"]
	if !ok {
		t.Fatalf("customer entity missing: %v", owners.Entities)
	}
	if cust.Mode != "set" {
		t.Errorf("customer.Mode = %q; want %q", cust.Mode, "set")
	}
	stream, ok := owners.Entities["orders_stream"]
	if !ok {
		t.Fatalf("orders_stream entity missing")
	}
	if stream.Mode != "record" {
		t.Errorf("orders_stream.Mode = %q; want %q", stream.Mode, "record")
	}
}

func TestLoadOwners_V2_MissingMode_RejectsBySchema(t *testing.T) {
	// v2 schema requires `mode` per entity. An owners file declaring
	// schema_version: 2 without per-entity mode must be rejected.
	set := ownersSchemaSet(t)
	_, valErrs, err := LoadOwners(set, "testdata/owners/v2/invalid/missing-mode.yaml")
	if err != nil {
		t.Fatalf("op error: %v", err)
	}
	if len(valErrs) == 0 {
		t.Fatalf("v2 owners missing mode returned no errors; want one for required field")
	}
	combined := strings.ToLower(combineErrs(valErrs))
	if !strings.Contains(combined, "mode") {
		t.Errorf("expected validation error to mention mode; got %q", combined)
	}
}

func TestCheckOwnersGroupMembership_NilGroups_NoErrors(t *testing.T) {
	// Disable case: nil groups argument returns no errors so the
	// linter can opt out for fixture-only test harnesses.
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer": {Owner: "data-platform"},
		},
	}
	if errs := CheckOwnersGroupMembership(owners, nil); len(errs) != 0 {
		t.Errorf("nil groups produced %d errors; want 0: %v", len(errs), errs)
	}
}

func TestCheckOwnersGroupMembership_EmptyGroups_NoErrors(t *testing.T) {
	// Degenerate disable case: groups with empty set returns no
	// errors (LoadCodeOwners("") returns this shape).
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer": {Owner: "data-platform"},
		},
	}
	groups := &CodeOwnersGroups{set: map[string]struct{}{}}
	if errs := CheckOwnersGroupMembership(owners, groups); len(errs) != 0 {
		t.Errorf("empty groups produced %d errors; want 0: %v", len(errs), errs)
	}
}

func TestCheckOwnersGroupMembership_HappyPath(t *testing.T) {
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer":      {Owner: "@PLACEHOLDER-org/rules-authors"},
			"orders_stream": {Owner: "@PLACEHOLDER-org/rules-authors"},
		},
	}
	groups := &CodeOwnersGroups{
		set: map[string]struct{}{
			"@PLACEHOLDER-org/platform-team":  {},
			"@PLACEHOLDER-org/rules-authors":  {},
			"@PLACEHOLDER-org/sre":            {},
		},
		Path: ".github/CODEOWNERS",
	}
	if errs := CheckOwnersGroupMembership(owners, groups); len(errs) != 0 {
		t.Errorf("happy path produced %d errors; want 0: %v", len(errs), errs)
	}
}

func TestCheckOwnersGroupMembership_UnknownGroup_Rejected(t *testing.T) {
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"customer": {Owner: "@PLACEHOLDER-org/typo-group"},
		},
	}
	groups := &CodeOwnersGroups{
		set: map[string]struct{}{
			"@PLACEHOLDER-org/platform-team": {},
			"@PLACEHOLDER-org/rules-authors": {},
		},
		Path: ".github/CODEOWNERS",
	}
	errs := CheckOwnersGroupMembership(owners, groups)
	if len(errs) != 1 {
		t.Fatalf("unknown group produced %d errors; want 1: %v", len(errs), errs)
	}
	combined := combineErrs(errs)
	if !strings.Contains(combined, "ADR-0037") {
		t.Errorf("error %q should cite ADR-0037", combined)
	}
	if !strings.Contains(combined, "@PLACEHOLDER-org/typo-group") {
		t.Errorf("error %q should quote the offending owner value", combined)
	}
	if !strings.Contains(combined, "customer") {
		t.Errorf("error %q should name the entity", combined)
	}
	if !strings.Contains(combined, "@PLACEHOLDER-org/rules-authors") {
		t.Errorf("error %q should list valid groups", combined)
	}
}

func TestCheckOwnersGroupMembership_MultipleMisses(t *testing.T) {
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"a": {Owner: "@PLACEHOLDER-org/unknown-a"},
			"b": {Owner: "@PLACEHOLDER-org/known"},
			"c": {Owner: "@PLACEHOLDER-org/unknown-c"},
		},
	}
	groups := &CodeOwnersGroups{
		set: map[string]struct{}{
			"@PLACEHOLDER-org/known": {},
		},
		Path: ".github/CODEOWNERS",
	}
	errs := CheckOwnersGroupMembership(owners, groups)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors (a + c); got %d: %v", len(errs), errs)
	}
	combined := combineErrs(errs)
	if !strings.Contains(combined, `"a"`) || !strings.Contains(combined, `"c"`) {
		t.Errorf("expected entity names a and c in errors; got %q", combined)
	}
	if strings.Contains(combined, `"b"`) {
		t.Errorf("entity b is valid; should not appear in errors: %q", combined)
	}
}

func TestCheckOwnersGroupMembership_EmptyOwnerSkipped(t *testing.T) {
	// Schema validation (LoadOwners) already requires owner to be a
	// non-empty string; the cross-check skips empty values to avoid
	// double-reporting.
	owners := &Owners{
		Entities: map[string]OwnerEntity{
			"x": {Owner: ""},
		},
	}
	groups := &CodeOwnersGroups{
		set:  map[string]struct{}{"@PLACEHOLDER-org/known": {}},
		Path: ".github/CODEOWNERS",
	}
	if errs := CheckOwnersGroupMembership(owners, groups); len(errs) != 0 {
		t.Errorf("empty owner produced %d errors; want 0 (schema layer handles it): %v", len(errs), errs)
	}
}

func TestLoadOwners_CapturesOwnerField(t *testing.T) {
	// LoadOwners must populate OwnerEntity.Owner so the cross-check
	// (CheckOwnersGroupMembership) has a value to compare.
	set := ownersSchemaSet(t)
	owners, valErrs, err := LoadOwners(set, "testdata/owners/valid/_owners.yaml")
	if err != nil || len(valErrs) != 0 {
		t.Fatalf("LoadOwners returned err=%v valErrs=%v", err, valErrs)
	}
	cust := owners.Entities["customer"]
	if cust.Owner != "data-platform" {
		t.Errorf("customer.Owner = %q; want %q", cust.Owner, "data-platform")
	}
	acc := owners.Entities["account"]
	if acc.Owner != "finance-data" {
		t.Errorf("account.Owner = %q; want %q", acc.Owner, "finance-data")
	}
}

func combineErrs(errs []ValidationError) string {
	out := ""
	for _, e := range errs {
		out += e.Message + "\n"
	}
	return out
}
