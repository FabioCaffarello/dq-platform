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

// LoadOwnersSchema compiles the _owners.yaml JSON Schema. The
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

// Owners is the in-memory representation of a loaded _owners.yaml,
// reduced to what the linter cross-checks: the set of declared
// entity names. The full owner descriptor (channels, severity
// overrides) is consumed by the alerting consumer at deploy time
// per ADR-0006 CC3, not by the linter.
type Owners struct {
	// Entities is the set of entity names declared in
	// _owners.yaml. Empty when the file is missing.
	Entities map[string]struct{}
	// Path is the path the linter loaded from, for diagnostics.
	// Empty when the file was missing.
	Path string
}

// LoadOwners reads and validates _owners.yaml against the given
// compiled schema. A missing file is NOT an error: the returned
// Owners has an empty Entities set. The caller (CheckRulesHaveOwners)
// is responsible for rejecting rules whose entity is absent.
//
// Returns a validation-error slice when the file exists but does
// not conform to the schema. The schema-validation errors are
// surfaced verbatim from the JSON-schema validator.
func LoadOwners(schema *jsonschema.Schema, ownersPath string) (*Owners, []ValidationError, error) {
	raw, err := os.ReadFile(ownersPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Owners{Entities: map[string]struct{}{}}, nil, nil
		}
		return nil, nil, fmt.Errorf("read owners: %w", err)
	}

	var doc any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, []ValidationError{{Message: fmt.Sprintf("yaml parse error: %v", err)}}, nil
	}

	if err := schema.Validate(doc); err != nil {
		return nil, []ValidationError{{Message: err.Error()}}, nil
	}

	owners := &Owners{
		Entities: map[string]struct{}{},
		Path:     ownersPath,
	}
	// Schema validation already enforced the structure; this is a
	// straightforward extraction of the `entities` map keys.
	if m, ok := doc.(map[string]any); ok {
		if entities, ok := m["entities"].(map[string]any); ok {
			for name := range entities {
				owners.Entities[name] = struct{}{}
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
