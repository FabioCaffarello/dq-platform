// path: tools/lint/owners.go

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// LoadOwnersSchema compiles an _owners.yaml JSON Schema. The
// schemaPath must refer to the rules mirror per ADR-0006 CC12 (the
// schema is the workspace surface against which rule YAMLs and
// _owners.yaml declare conformance).
func LoadOwnersSchema(schemaPath string) (*jsonschema.Schema, error) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read owners schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("add owners schema resource: %w", err)
	}
	sch, err := compiler.Compile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("compile owners schema: %w", err)
	}
	return sch, nil
}

// OwnersSchemaSet holds the v1 and v2 _owners schemas the linter
// dispatches between, keyed by the `schema_version` field in the
// _owners.yaml file. Either may be nil; the dispatcher reports a
// clear error if the loaded file requires a missing schema.
type OwnersSchemaSet struct {
	V1 *jsonschema.Schema
	V2 *jsonschema.Schema
}

// LoadOwnersSchemaSet loads both v1 and v2 owners schemas. Either
// path may be empty.
func LoadOwnersSchemaSet(v1Path, v2Path string) (*OwnersSchemaSet, error) {
	set := &OwnersSchemaSet{}
	if v1Path != "" {
		sch, err := LoadOwnersSchema(v1Path)
		if err != nil {
			return nil, fmt.Errorf("v1 owners schema: %w", err)
		}
		set.V1 = sch
	}
	if v2Path != "" {
		sch, err := LoadOwnersSchema(v2Path)
		if err != nil {
			return nil, fmt.Errorf("v2 owners schema: %w", err)
		}
		set.V2 = sch
	}
	return set, nil
}

// OwnerEntity is the reduced in-memory descriptor for one entity
// in _owners.yaml. The linter retains only what is needed for
// cross-checks: the entity is declared (presence) and its mode
// (for v2 cross-check #3). Channels and severity overrides are
// consumed by the alerting layer per ADR-0006 CC3, not by the
// linter, so they are intentionally not retained here.
type OwnerEntity struct {
	// Mode is "set" or "record" for v2 owners; empty string for
	// v1 owners (which had no mode field). Cross-check #3 only
	// fires when both the rule and the owners entry carry a mode.
	Mode string
}

// Owners is the in-memory representation of a loaded _owners.yaml.
type Owners struct {
	// SchemaVersion records which owners schema this file claims
	// against (1 or 2). 0 means the file was missing.
	SchemaVersion int

	// Entities is keyed by entity identifier. Empty when the file
	// is missing.
	Entities map[string]OwnerEntity

	// Path is the path the linter loaded from, for diagnostics.
	// Empty when the file was missing.
	Path string
}

// parseOwnersVersion extracts the `schema_version` field without
// triggering schema validation. Missing returns 0; the caller
// treats 0 as "v1 dispatch" so the v1 schema can issue the
// canonical error about a missing field.
func parseOwnersVersion(raw []byte) int {
	var h struct {
		SchemaVersion int `yaml:"schema_version"`
	}
	_ = yaml.Unmarshal(raw, &h)
	return h.SchemaVersion
}

// LoadOwners reads and validates _owners.yaml. The schema is
// dispatched by the `schema_version` field. A missing file is NOT
// an error: the returned Owners has SchemaVersion 0 and empty
// Entities. The caller (CheckRulesHaveOwners) rejects rules whose
// entity is absent.
func LoadOwners(set *OwnersSchemaSet, ownersPath string) (*Owners, []ValidationError, error) {
	raw, err := os.ReadFile(ownersPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Owners{Entities: map[string]OwnerEntity{}}, nil, nil
		}
		return nil, nil, fmt.Errorf("read owners: %w", err)
	}

	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, []ValidationError{{Message: fmt.Sprintf("yaml parse error: %v", err)}}, nil
	}

	version := parseOwnersVersion(raw)
	var schema *jsonschema.Schema
	switch version {
	case 0, 1:
		schema = set.V1
		if schema == nil {
			return nil, []ValidationError{{Message: "no v1 owners schema loaded; cannot validate _owners.yaml"}}, nil
		}
	case 2:
		schema = set.V2
		if schema == nil {
			return nil, []ValidationError{{Message: "_owners.yaml declares schema_version 2 but no v2 owners schema is loaded"}}, nil
		}
	default:
		return nil, []ValidationError{{Message: fmt.Sprintf("_owners.yaml declares unsupported schema_version %d", version)}}, nil
	}

	if err := schema.Validate(doc); err != nil {
		return nil, []ValidationError{{Message: err.Error()}}, nil
	}

	owners := &Owners{
		SchemaVersion: version,
		Entities:      map[string]OwnerEntity{},
		Path:          ownersPath,
	}
	if version == 0 {
		owners.SchemaVersion = 1
	}
	if m, ok := doc.(map[string]any); ok {
		if entities, ok := m["entities"].(map[string]any); ok {
			for name, entAny := range entities {
				ent := OwnerEntity{}
				if entMap, ok := entAny.(map[string]any); ok {
					if mode, ok := entMap["mode"].(string); ok {
						ent.Mode = mode
					}
				}
				owners.Entities[name] = ent
			}
		}
	}
	return owners, nil, nil
}

// CheckRulesHaveOwners walks the rules directory and verifies that
// each rule YAML's `entity` field is declared in the owners set
// per ADR-0006 CC9.
//
// The function reads each rule file again (rule-shape validation
// happens in ValidateRulesDir; this pass only cares about the
// entity field). For files whose entity is not in the owners set,
// it returns a ValidationError keyed on the rule's path.
//
// A missing _owners.yaml is reported as a single top-level error
// when any rule file exists. This is the engine-side enforcement
// of ADR-0006 CC9; the linter rejects an entity-without-owner
// configuration rather than producing alerts that have no route.
func CheckRulesHaveOwners(owners *Owners, rulesDir string) (map[string][]ValidationError, error) {
	results := map[string][]ValidationError{}
	rulePaths, err := listRuleYAMLs(rulesDir)
	if err != nil {
		return nil, err
	}
	if len(rulePaths) == 0 {
		// No rules to cross-check; ownerless platform is OK at
		// the pre-Phase-6 state.
		return results, nil
	}
	if len(owners.Entities) == 0 {
		// Rules exist but no owners declared. Report a single
		// top-level error so CI fails loudly per ADR-0006 CC9.
		results[rulesDir] = []ValidationError{{
			Message: fmt.Sprintf("ADR-0006 CC9: rules exist under %s but no _owners.yaml entities are declared; every entity with checks must have an owner", rulesDir),
		}}
		return results, nil
	}

	for _, path := range rulePaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			results[path] = []ValidationError{{Message: fmt.Sprintf("read for owners check: %v", err)}}
			continue
		}
		var doc struct {
			Entity string `yaml:"entity"`
		}
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			// Schema validation in ValidateRulesDir already flagged
			// this; skip here so the same error isn't double-counted.
			continue
		}
		if doc.Entity == "" {
			continue
		}
		if _, ok := owners.Entities[doc.Entity]; !ok {
			results[path] = []ValidationError{{
				Message: fmt.Sprintf("ADR-0006 CC9: entity %q has no entry in _owners.yaml; rules without owners cannot route alerts", doc.Entity),
			}}
		}
	}
	return results, nil
}

// listRuleYAMLs returns the *.yaml / *.yml files under rulesDir,
// excluding any path under a "_schema" subdirectory. Returns an
// empty slice if the directory does not exist — the linter treats
// a missing rules/ directory as the pre-Phase-6 state, not an
// error. Mirrors the walking convention in ValidateRulesDir.
func listRuleYAMLs(rulesDir string) ([]string, error) {
	var paths []string
	walkErr := filepath.WalkDir(rulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Base(path) == "_schema" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		// Exclude the _owners.yaml itself from the rule list — it
		// is the contract document, not a rule.
		if name == "_owners.yaml" || name == "_owners.yml" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("walk %s: %w", rulesDir, walkErr)
	}
	return paths, nil
}
