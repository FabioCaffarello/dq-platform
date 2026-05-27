// path: tools/lint/owners.go

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

// OwnersSchemaSet holds the v1, v2, and v3 _owners schemas the
// linter dispatches between, keyed by the `schema_version` field
// in the _owners.yaml file. Each may be nil; the dispatcher reports
// a clear error if the loaded file requires a missing schema.
type OwnersSchemaSet struct {
	V1 *jsonschema.Schema
	V2 *jsonschema.Schema
	V3 *jsonschema.Schema
}

// LoadOwnersSchemaSet loads v1, v2, and v3 owners schemas. Any path
// may be empty (the corresponding schema field stays nil; the
// dispatcher reports a clear error if a file requires it).
func LoadOwnersSchemaSet(v1Path, v2Path, v3Path string) (*OwnersSchemaSet, error) {
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
	if v3Path != "" {
		sch, err := LoadOwnersSchema(v3Path)
		if err != nil {
			return nil, fmt.Errorf("v3 owners schema: %w", err)
		}
		set.V3 = sch
	}
	return set, nil
}

// OwnerEntity is the reduced in-memory descriptor for one entity
// in _owners.yaml. The linter retains only what is needed for
// cross-checks: the entity is declared (presence), its mode
// (for v2 cross-check #3), its owner identifier (for the
// CODEOWNERS-group membership check per ADR-0037), and the v3
// onboarding flag (for downstream consumer-side routing per
// ADR-0046; not enforced by the linter itself but retained so
// consumers reading the parsed Owners can resolve overrides).
// Channels and severity overrides are consumed by the alerting
// layer per ADR-0006 CC3, not by the linter, so they are
// intentionally not retained here.
type OwnerEntity struct {
	// Mode is "set" or "record" for v2/v3 owners; empty string
	// for v1 owners (which had no mode field). Cross-check #3
	// only fires when both the rule and the owners entry carry
	// a mode.
	Mode string

	// Owner is the literal `owner:` value from _owners.yaml. The
	// CODEOWNERS-group cross-check (CheckOwnersGroupMembership)
	// compares this against the loaded CodeOwnersGroups inventory.
	Owner string

	// Onboarding mirrors the v3 `onboarding` flag per ADR-0046.
	// Always false for v1/v2 owners (the field is v3-only).
	// Consumer-side routing reads this and substitutes
	// env.OnboardingChannel when true AND OnboardingChannel is
	// non-empty.
	Onboarding bool
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
	case 3:
		schema = set.V3
		if schema == nil {
			return nil, []ValidationError{{Message: "_owners.yaml declares schema_version 3 but no v3 owners schema is loaded"}}, nil
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
					if owner, ok := entMap["owner"].(string); ok {
						ent.Owner = owner
					}
					if onboarding, ok := entMap["onboarding"].(bool); ok {
						ent.Onboarding = onboarding
					}
				}
				owners.Entities[name] = ent
			}
		}
	}
	return owners, nil, nil
}

// CheckOwnersGroupMembership verifies every `owner:` value in the
// loaded owners set resolves to a CodeOwnersGroups entry. A nil or
// empty groups argument disables the check (returns nil); the
// caller does not need to guard. Returns one ValidationError per
// miss, keyed by the entity's name in the message.
//
// The check implements ADR-0037: lint-time enforcement of "the
// owner field references a real review group." It is the cheaper
// first line of defense complementing ADR-0006 §9's CODEOWNERS-
// routed review (the second line of defense).
func CheckOwnersGroupMembership(owners *Owners, groups *CodeOwnersGroups) []ValidationError {
	if owners == nil || groups == nil || len(groups.set) == 0 {
		return nil
	}
	var errs []ValidationError
	entityNames := make([]string, 0, len(owners.Entities))
	for name := range owners.Entities {
		entityNames = append(entityNames, name)
	}
	sort.Strings(entityNames)
	inventory := groups.Slice()
	for _, name := range entityNames {
		ent := owners.Entities[name]
		if ent.Owner == "" {
			// Schema validation (LoadOwners) already requires
			// `owner` to be a non-empty string; skip rather than
			// double-report.
			continue
		}
		if groups.Contains(ent.Owner) {
			continue
		}
		errs = append(errs, ValidationError{
			Message: fmt.Sprintf(
				"ADR-0037: entity %q owner %q does not match any CODEOWNERS group in %s; valid groups: %v",
				name, ent.Owner, groups.Path, inventory,
			),
		})
	}
	return errs
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
