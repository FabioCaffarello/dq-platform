// path: engine/internal/dsl/catalog/catalog.go

// Package catalog exposes the engine-side view of the v1 kind
// catalog committed by ADR-0022. The catalog YAML at v1.yaml is
// the canonical source; rules/_schema/catalog.v1.yaml is the
// byte-equal mirror enforced by the schema-mirror CI gate.
//
// The engine reads the embedded YAML at boot and exposes the
// set of declared kinds for the dispatcher startup invariant
// (cmd/dq-engine cross-checks the eval handler registry against
// the catalog). Per-kind aggregation defaults and params_schema
// are not consumed by the engine here — they are consumed by the
// linter (tools/lint) and by the record-mode runner / handler
// when those land in Wave-S sub-slice β.
package catalog

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

// rawCatalog is the embedded v1.yaml body. Embedded at build
// time so the engine binary carries its own catalog without
// depending on a deploy-time mount.
//
//go:embed v1.yaml
var rawCatalog []byte

// Catalog is the parsed v1 catalog.
type Catalog struct {
	Version int
	Kinds   []Kind
}

// Kind is one entry in the catalog. Only the fields the engine
// dispatcher and (future) record-mode runner consume are typed
// here; per-kind params_schema lives in the YAML for the linter
// but is not re-parsed engine-side at sub-slice α.
type Kind struct {
	Name       string
	Mode       string // "set" | "record" per ADR-0021
	SourceMode string // "set" | "record" per ADR-0023
}

// Load returns the parsed embedded catalog. Always returns the
// same content for a given engine binary — the catalog is part
// of the binary's compiled state.
func Load() (*Catalog, error) {
	return parseCatalog(rawCatalog)
}

// parseCatalog is the byte-oriented parser Load wraps. Tests
// inject crafted byte slices to exercise the malformed-YAML and
// unsupported-version rejection paths the embedded data never
// trips in production.
func parseCatalog(raw []byte) (*Catalog, error) {
	var doc struct {
		CatalogVersion int `yaml:"catalog_version"`
		Kinds          []struct {
			Name       string `yaml:"name"`
			Mode       string `yaml:"mode"`
			SourceMode string `yaml:"source_mode"`
		} `yaml:"kinds"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse embedded catalog: %w", err)
	}
	if doc.CatalogVersion != 1 {
		return nil, fmt.Errorf("embedded catalog version %d is not supported (expected 1 per ADR-0022)", doc.CatalogVersion)
	}
	cat := &Catalog{Version: doc.CatalogVersion}
	for _, k := range doc.Kinds {
		cat.Kinds = append(cat.Kinds, Kind{
			Name:       k.Name,
			Mode:       k.Mode,
			SourceMode: k.SourceMode,
		})
	}
	return cat, nil
}
