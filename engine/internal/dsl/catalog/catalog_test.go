// path: engine/internal/dsl/catalog/catalog_test.go

package catalog

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLoad_HappyPath_EmbeddedV1Parses(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error against the embedded v1.yaml: %v", err)
	}
	if cat == nil {
		t.Fatalf("Load() returned nil catalog with nil error")
	}
	if cat.Version != 1 {
		t.Errorf("cat.Version = %d; want 1 per ADR-0022", cat.Version)
	}
	if len(cat.Kinds) == 0 {
		t.Errorf("cat.Kinds is empty; v1.yaml declares at least one kind")
	}
	for i, k := range cat.Kinds {
		if k.Name == "" {
			t.Errorf("cat.Kinds[%d].Name is empty", i)
		}
		if k.Mode == "" {
			t.Errorf("cat.Kinds[%d].Mode is empty (kind=%q)", i, k.Name)
		}
		if k.SourceMode == "" {
			t.Errorf("cat.Kinds[%d].SourceMode is empty (kind=%q)", i, k.Name)
		}
	}
}

func TestLoad_CatalogShape_KnownKindsPresent(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	got := map[string]Kind{}
	for _, k := range cat.Kinds {
		got[k.Name] = k
	}

	// Table-driven per go-coding-standards C6: each row asserts
	// one currently-shipping kind's Mode / SourceMode pair. A
	// regression that renames a kind, flips a mode, or drops a
	// kind entirely surfaces here as a named test failure rather
	// than as a downstream dispatcher panic at engine boot.
	cases := []struct {
		name       string
		mode       string
		sourceMode string
	}{
		{name: "set.row_count_positive", mode: "set", sourceMode: "set"},
		{name: "set.row_count_within_baseline", mode: "set", sourceMode: "set"},
		{name: "record.schema_conformance", mode: "record", sourceMode: "record"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, ok := got[c.name]
			if !ok {
				t.Fatalf("kind %q missing from catalog", c.name)
			}
			if k.Mode != c.mode {
				t.Errorf("kind %q Mode = %q; want %q (ADR-0021)", c.name, k.Mode, c.mode)
			}
			if k.SourceMode != c.sourceMode {
				t.Errorf("kind %q SourceMode = %q; want %q (ADR-0023)", c.name, k.SourceMode, c.sourceMode)
			}
		})
	}
}

func TestParseCatalog_RejectsMalformedYAML(t *testing.T) {
	// Unbalanced bracket: yaml.Unmarshal returns a syntax error
	// before the parser ever inspects catalog_version.
	bad := []byte("catalog_version: 1\nkinds: [malformed")
	cat, err := parseCatalog(bad)
	if err == nil {
		t.Fatalf("parseCatalog accepted malformed YAML; got cat=%+v", cat)
	}
	if !strings.Contains(err.Error(), "parse embedded catalog") {
		t.Errorf("error %q lacks the 'parse embedded catalog' package-prefix context (go-coding-standards C2)", err.Error())
	}
}

func TestParseCatalog_RejectsUnsupportedVersion(t *testing.T) {
	// catalog_version: 2 is a future schema; today the engine
	// only consumes v1 per ADR-0022. The parser surfaces the
	// rejected version in the error so an operator running the
	// wrong binary against a newer catalog sees the mismatch.
	bad := []byte("catalog_version: 2\nkinds: []\n")
	cat, err := parseCatalog(bad)
	if err == nil {
		t.Fatalf("parseCatalog accepted catalog_version=2; got cat=%+v", cat)
	}
	if !strings.Contains(err.Error(), "version 2 is not supported") {
		t.Errorf("error %q does not name the rejected version 2 (ADR-0022 expectation)", err.Error())
	}
}

func TestCatalogV1Yaml_ByteEqualWithRulesMirror(t *testing.T) {
	// ADR-0001 §"byte-equality CI gate" + ADR-0022 §C-B0S2.1: the
	// catalog mirror at rules/_schema/catalog.v1.yaml MUST be the
	// byte-identical copy of the canonical engine source. The CI
	// schema-mirror workflow enforces this on every PR; this test
	// catches divergence locally before push.
	//
	// Path: tests run with cwd = the package directory, so
	// engine/internal/dsl/catalog/ → ../../../../rules/_schema/.
	mirrorPath := "../../../../rules/_schema/catalog.v1.yaml"
	mirror, err := os.ReadFile(mirrorPath)
	if err != nil {
		t.Fatalf("read mirror at %s: %v", mirrorPath, err)
	}
	if !bytes.Equal(rawCatalog, mirror) {
		t.Errorf("engine/internal/dsl/catalog/v1.yaml and %s diverged; "+
			"run `make sync-schema` (rules/_schema/ is the mirror; never edited by hand)",
			mirrorPath)
	}
}
