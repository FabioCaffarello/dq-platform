// path: tools/lint/compatibility_test.go

package main

import (
	"strings"
	"testing"
)

func TestSchemaVersionStatus_KnownVersions(t *testing.T) {
	v1, ok := SchemaVersionStatus(1)
	if !ok {
		t.Fatal("SchemaVersionStatus(1): want ok=true (v1 is in the table)")
	}
	if v1.Status != SchemaStatusDeprecated {
		t.Errorf("v1 status = %q; want deprecated", v1.Status)
	}
	if v1.EarliestDrop == "" || v1.EarliestDrop == "TBD" {
		t.Errorf("v1 EarliestDrop = %q; want a concrete date per ADR-0035", v1.EarliestDrop)
	}

	v2, ok := SchemaVersionStatus(2)
	if !ok {
		t.Fatal("SchemaVersionStatus(2): want ok=true (v2 is in the table)")
	}
	if v2.Status != SchemaStatusCurrent {
		t.Errorf("v2 status = %q; want current", v2.Status)
	}
}

func TestSchemaVersionStatus_UnknownVersion(t *testing.T) {
	// A future version not yet in the table returns ok=false.
	// The lint binary handles unknown versions as "no warning"
	// (the schema validator will already have rejected the rule
	// at an earlier stage for being unsupported).
	if _, ok := SchemaVersionStatus(99); ok {
		t.Error("SchemaVersionStatus(99): want ok=false for unknown version")
	}
	if _, ok := SchemaVersionStatus(0); ok {
		t.Error("SchemaVersionStatus(0): want ok=false (zero is not a valid version)")
	}
}

func TestCheckDeprecatedSchemaVersions_FiresOnV1(t *testing.T) {
	// testdata/valid/customer.yaml is at version 1 (the v1
	// fixture kept around for v1-path testing). The deprecation
	// warning must fire.
	warnings, err := CheckDeprecatedSchemaVersions("testdata/valid")
	if err != nil {
		t.Fatalf("CheckDeprecatedSchemaVersions: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected at least one deprecation warning for testdata/valid/customer.yaml (v1); got none")
	}
	found := false
	for _, w := range warnings {
		if !strings.HasSuffix(w.Path, "customer.yaml") {
			continue
		}
		found = true
		if w.Version != 1 {
			t.Errorf("warning.Version = %d; want 1", w.Version)
		}
		if !strings.Contains(w.Message, "deprecated") {
			t.Errorf("warning.Message does not mention 'deprecated': %s", w.Message)
		}
		if !strings.Contains(w.Message, "2026-08-23") {
			t.Errorf("warning.Message does not include the earliest-drop date 2026-08-23: %s", w.Message)
		}
		if !strings.Contains(w.Message, "ADR-0035") {
			t.Errorf("warning.Message does not cite ADR-0035: %s", w.Message)
		}
	}
	if !found {
		t.Errorf("did not find warning for customer.yaml in: %v", warnings)
	}
}

func TestCheckDeprecatedSchemaVersions_NoWarningOnV2(t *testing.T) {
	// testdata/v2/valid/ contains only v2 rules. No warning
	// should fire.
	warnings, err := CheckDeprecatedSchemaVersions("testdata/v2/valid")
	if err != nil {
		t.Fatalf("CheckDeprecatedSchemaVersions: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for v2 fixtures; got %d: %v", len(warnings), warnings)
	}
}

func TestCheckDeprecatedSchemaVersions_SkipsSchemaDir(t *testing.T) {
	// The _schema/ subdirectory must be skipped by the walker
	// — it carries schemas, not rules. The mirror schema files
	// have no `version:` field, so even without the skip the
	// walker would emit no warnings; this test guards the
	// directory-skip behavior explicitly.
	warnings, err := CheckDeprecatedSchemaVersions("testdata")
	if err != nil {
		t.Fatalf("CheckDeprecatedSchemaVersions: %v", err)
	}
	for _, w := range warnings {
		if strings.Contains(w.Path, "/_schema/") {
			t.Errorf("walker descended into _schema/: %s", w.Path)
		}
	}
}

func TestCheckDeprecatedSchemaVersions_SkipsUnderscoreFiles(t *testing.T) {
	// Files whose basename starts with `_` (e.g., `_owners.yaml`)
	// are not rule YAMLs and must be skipped by the walker.
	warnings, err := CheckDeprecatedSchemaVersions("testdata/v2/valid")
	if err != nil {
		t.Fatalf("CheckDeprecatedSchemaVersions: %v", err)
	}
	for _, w := range warnings {
		base := w.Path
		if idx := strings.LastIndex(base, "/"); idx >= 0 {
			base = base[idx+1:]
		}
		if strings.HasPrefix(base, "_") {
			t.Errorf("walker emitted warning for underscore-prefixed file: %s", w.Path)
		}
	}
}

func TestCheckDeprecatedSchemaVersions_MissingDirReturnsEmpty(t *testing.T) {
	// A missing rules directory is not an error — the walker
	// returns an empty slice so `make lint-rules` continues to
	// succeed before phase scaffolding lands.
	warnings, err := CheckDeprecatedSchemaVersions("testdata/does-not-exist")
	if err != nil {
		t.Errorf("CheckDeprecatedSchemaVersions(missing): want nil error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("CheckDeprecatedSchemaVersions(missing): want empty slice, got %d entries", len(warnings))
	}
}
