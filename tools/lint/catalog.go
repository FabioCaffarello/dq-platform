// path: tools/lint/catalog.go

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// Catalog is the in-memory representation of the kind catalog per
// ADR-0022. The catalog declares the universe of supported check
// kinds; the linter cross-checks every rule's kind against this set
// (cross-check #5), validates per-rule params against the matching
// kind's params_schema (cross-check #6), and confirms the source
// substrate alignment via the per-kind source_mode (cross-check #8).
type Catalog struct {
	// Version is the catalog_version field from the YAML. v1 at the
	// current contract level; future schema-incompatible changes
	// require a new file (catalog.v2.yaml).
	Version int

	// Kinds is the in-memory index keyed by the kind name (e.g.,
	// "set.row_count_positive"). The compiled per-kind params
	// schema is held inside CatalogKind.
	Kinds map[string]*CatalogKind

	// Path is the file the catalog was loaded from, for diagnostics.
	Path string
}

// CatalogKind is one entry in the catalog. Aggregation defaults
// (per ADR-0026) live in the catalog YAML but are consumed by the
// engine handler at runtime, not by the linter; the linter does not
// retain them.
type CatalogKind struct {
	Name         string
	Mode         string // "set" | "record" (per ADR-0021)
	SourceMode   string // "set" | "record" (per ADR-0023)
	ParamsSchema *jsonschema.Schema
}

// LoadCatalog reads the catalog YAML, validates the surface shape,
// and compiles each kind's params_schema. A missing catalog file is
// NOT an operational error: the returned Catalog is nil and the
// caller (the v2 cross-check pass) treats the absence as "v2 rules
// cannot be lint-resolved" — which surfaces as a top-level error
// when any v2 rule is encountered.
func LoadCatalog(path string) (*Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read catalog: %w", err)
	}

	var doc struct {
		CatalogVersion int `yaml:"catalog_version"`
		Kinds          []struct {
			Name         string         `yaml:"name"`
			Description  string         `yaml:"description"`
			Mode         string         `yaml:"mode"`
			SourceMode   string         `yaml:"source_mode"`
			ParamsSchema map[string]any `yaml:"params_schema"`
			// Aggregation is loaded but not retained — the engine
			// consumes it at runtime.
			Aggregation map[string]any `yaml:"aggregation"`
		} `yaml:"kinds"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse catalog yaml: %w", err)
	}
	if doc.CatalogVersion != 1 {
		return nil, fmt.Errorf("catalog version %d is not supported (expected 1 per ADR-0022)", doc.CatalogVersion)
	}

	cat := &Catalog{
		Version: doc.CatalogVersion,
		Kinds:   map[string]*CatalogKind{},
		Path:    path,
	}

	for i, k := range doc.Kinds {
		if k.Name == "" {
			return nil, fmt.Errorf("catalog kind at index %d has empty name", i)
		}
		if _, exists := cat.Kinds[k.Name]; exists {
			return nil, fmt.Errorf("catalog kind %q declared more than once", k.Name)
		}
		switch k.Mode {
		case "set", "record":
		default:
			return nil, fmt.Errorf("catalog kind %q: mode %q is not one of [set, record]", k.Name, k.Mode)
		}
		switch k.SourceMode {
		case "set", "record":
		default:
			return nil, fmt.Errorf("catalog kind %q: source_mode %q is not one of [set, record]", k.Name, k.SourceMode)
		}

		// Compile params_schema. The YAML-parsed map serializes to
		// JSON; the compiler treats the bytes as a JSON Schema
		// document. An empty/missing params_schema compiles into a
		// permissive object schema that accepts any params shape.
		ps, err := compileSubSchema(k.Name, k.ParamsSchema)
		if err != nil {
			return nil, fmt.Errorf("catalog kind %q: %w", k.Name, err)
		}

		cat.Kinds[k.Name] = &CatalogKind{
			Name:         k.Name,
			Mode:         k.Mode,
			SourceMode:   k.SourceMode,
			ParamsSchema: ps,
		}
	}

	return cat, nil
}

// Kind returns the catalog entry for name, or (nil, false) when
// the kind is not declared.
func (c *Catalog) Kind(name string) (*CatalogKind, bool) {
	if c == nil {
		return nil, false
	}
	k, ok := c.Kinds[name]
	return k, ok
}

// compileSubSchema serializes a YAML-parsed schema fragment to JSON
// and compiles it via jsonschema. An empty fragment yields a
// permissive schema (accepts any object) so that a kind with no
// declared params still has a usable compiled schema.
func compileSubSchema(kindName string, fragment map[string]any) (*jsonschema.Schema, error) {
	var payload []byte
	if len(fragment) == 0 {
		payload = []byte(`{"type":"object"}`)
	} else {
		b, err := json.Marshal(fragment)
		if err != nil {
			return nil, fmt.Errorf("marshal params_schema: %w", err)
		}
		payload = b
	}
	compiler := jsonschema.NewCompiler()
	id := fmt.Sprintf("catalog://%s/params_schema", kindName)
	if err := compiler.AddResource(id, bytes.NewReader(payload)); err != nil {
		return nil, fmt.Errorf("add params_schema resource: %w", err)
	}
	sch, err := compiler.Compile(id)
	if err != nil {
		return nil, fmt.Errorf("compile params_schema: %w", err)
	}
	return sch, nil
}
