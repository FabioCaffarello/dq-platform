// path: tools/lint/lint.go

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

// LoadSchema reads a JSON Schema document and compiles it for repeated
// validation. The schemaPath must refer to the rules mirror, not the
// engine source — the linter validates against the workspace surface
// per ADR-0001 (the schema mirror is what the rules workspace claims
// to be lintable against).
//
// The function also performs a belt-and-suspenders check that the
// loaded schema's entity pattern does not admit the ASCII pipe
// character. The pipe-character ban is load-bearing per ADR-0002
// input-safety (entity is one of the five pipe-separated inputs to
// the execution_id hash, with no escaping). If a future schema edit
// weakens the pattern, this check fails loudly at linter startup
// instead of allowing unsafe entity names through.
func LoadSchema(schemaPath string) (*jsonschema.Schema, error) {
	raw, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("read schema: %w", err)
	}

	if err := assertEntityPipeRejected(raw); err != nil {
		return nil, fmt.Errorf("schema integrity check: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	sch, err := compiler.Compile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return sch, nil
}

// SchemaSet holds the compiled rule schemas the linter dispatches
// between, keyed by the rule's `version` field. v1 is mandatory at
// the current contract level (it underwrites every existing rule);
// v2 is optional and may be nil when the v2 schema is absent from
// disk — the linter then rejects any v2 rule it encounters with a
// helpful error.
type SchemaSet struct {
	V1 *jsonschema.Schema
	V2 *jsonschema.Schema
}

// LoadSchemaSet loads both v1 and v2 rule schemas. Either path may
// be empty, in which case the corresponding schema is omitted from
// the set; the per-file dispatcher then reports an error if a rule
// of the missing version is encountered.
func LoadSchemaSet(v1Path, v2Path string) (*SchemaSet, error) {
	set := &SchemaSet{}
	if v1Path != "" {
		sch, err := LoadSchema(v1Path)
		if err != nil {
			return nil, fmt.Errorf("v1 schema: %w", err)
		}
		set.V1 = sch
	}
	if v2Path != "" {
		sch, err := LoadSchema(v2Path)
		if err != nil {
			return nil, fmt.Errorf("v2 schema: %w", err)
		}
		set.V2 = sch
	}
	return set, nil
}

// assertEntityPipeRejected inspects the raw schema bytes and verifies
// that the entity property's pattern explicitly forbids the ASCII
// pipe character. Belt-and-suspenders against schema weakening.
func assertEntityPipeRejected(raw []byte) error {
	var root struct {
		Properties struct {
			Entity struct {
				Pattern string `json:"pattern"`
			} `json:"entity"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return fmt.Errorf("parse schema JSON: %w", err)
	}
	pattern := root.Properties.Entity.Pattern
	if pattern == "" {
		return errors.New("schema does not declare a pattern for the entity property")
	}
	if strings.ContainsRune(pattern, '|') {
		return fmt.Errorf("entity pattern %q contains an unescaped pipe character; ADR-0002 input-safety requires the pipe to be forbidden", pattern)
	}
	return nil
}

// ValidationError describes a single problem with a single rule file.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }

// ValidateRule validates a single rule YAML (already read into memory)
// against the compiled schema. The YAML is parsed into a generic
// structure and then handed to the JSON Schema validator. Returns an
// empty slice for valid rules.
func ValidateRule(schema *jsonschema.Schema, ruleBytes []byte) []ValidationError {
	var doc any
	if err := yaml.Unmarshal(ruleBytes, &doc); err != nil {
		return []ValidationError{{Message: fmt.Sprintf("yaml parse error: %v", err)}}
	}

	// jsonschema expects map[string]any with JSON-shaped types. The yaml
	// library returns map[string]any for v3, so the structures align.
	if err := schema.Validate(doc); err != nil {
		// jsonschema returns a *ValidationError tree; we surface it as
		// a single multi-line string per source file. Future Phase-4+
		// work can split this into per-path errors if structured
		// output is required by tooling.
		return []ValidationError{{Message: err.Error()}}
	}
	return nil
}

// ValidateRuleFile reads a rule YAML from disk and validates it
// against a single compiled schema. Used by tests and by v1-only
// call sites; the production linter walks via ValidateRulesDir,
// which dispatches between v1 and v2 via the SchemaSet.
func ValidateRuleFile(schema *jsonschema.Schema, path string) []ValidationError {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []ValidationError{{Message: fmt.Sprintf("read error: %v", err)}}
	}
	return ValidateRule(schema, raw)
}

// parseRuleVersion extracts the rule's `version` field without
// triggering schema validation. A missing or unparseable version is
// reported as 0; the caller treats 0 as "v1 dispatch" so the v1
// schema can issue the canonical error about a missing version.
func parseRuleVersion(ruleBytes []byte) int {
	var h struct {
		Version int `yaml:"version"`
	}
	_ = yaml.Unmarshal(ruleBytes, &h)
	return h.Version
}

// ValidateRuleBytes dispatches a rule's schema validation by its
// version field. v1 rules (or rules with no version field) are
// validated against the v1 schema; v2 rules are validated against
// the v2 schema. A version this linter has no schema for surfaces
// as a single ValidationError.
func ValidateRuleBytes(set *SchemaSet, ruleBytes []byte) []ValidationError {
	switch v := parseRuleVersion(ruleBytes); v {
	case 0, 1:
		if set.V1 == nil {
			return []ValidationError{{Message: "no v1 schema loaded; cannot validate rule"}}
		}
		return ValidateRule(set.V1, ruleBytes)
	case 2:
		if set.V2 == nil {
			return []ValidationError{{Message: "rule declares version 2 but no v2 schema is loaded"}}
		}
		return ValidateRule(set.V2, ruleBytes)
	default:
		return []ValidationError{{Message: fmt.Sprintf("rule declares unsupported schema version %d", v)}}
	}
}

// ValidateRulesDir walks the given directory for *.yaml files
// (excluding any directory named "_schema" at any depth — the
// schema mirror lives at rules/_schema/ and is not a rule).
//
// Each file is dispatched by its `version` field to the matching
// schema in the SchemaSet. v2 rules also get the v2 cross-check
// pass (cross-checks #3–#8 per ADRs 0021–0024) when the catalog
// and owners are loaded; the cross-checks are skipped if the
// schema validation failed (those checks would otherwise panic
// on missing required fields).
//
// Returns:
//
//   - results: a map of relative path → validation errors; only
//     files with errors are present.
//   - filesProcessed: the count of YAML files actually validated
//     (success + failure). Returned explicitly so the function
//     is reentrant and parallel-safe (no package-level state).
//   - err: operational failure (I/O, etc.), not a validation
//     failure. A missing rules directory is not an error — the
//     walker returns empty results so `make lint-rules` succeeds
//     before Phase 6 lands the first rule YAML.
//
// The catalog and owners arguments may be nil; v2 cross-checks
// that require them surface a single top-level error when a v2
// rule is encountered without the corresponding context.
func ValidateRulesDir(set *SchemaSet, catalog *Catalog, owners *Owners, dir string, verbose bool) (results map[string][]ValidationError, filesProcessed int, err error) {
	results = make(map[string][]ValidationError)

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkCallbackErr error) error {
		if walkCallbackErr != nil {
			return walkCallbackErr
		}
		// Skip any directory named "_schema". The skip is broader
		// than strictly required (it would skip e.g.
		// some-rule-dir/_schema/ too); the rules workspace does
		// not use nested _schema/ directories so the conservatism
		// is safe.
		if d.IsDir() {
			if filepath.Base(path) == "_schema" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		// Only consider *.yaml or *.yml files.
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		// Skip files whose basename begins with an underscore.
		// The manifest publisher (tools/manifest) uses the same
		// convention: `_owners.yaml`, `_schema/`, and any future
		// metadata files prefixed with `_` are not rule YAMLs.
		// The dq-lint binary validates `_owners.yaml` separately
		// via the `--owners` flag; the rule-schema walker must
		// not re-validate it against the rule schema.
		if strings.HasPrefix(name, "_") {
			return nil
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "dq-lint: checking %s\n", path)
		}
		filesProcessed++

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			results[path] = []ValidationError{{Message: fmt.Sprintf("read error: %v", readErr)}}
			return nil
		}
		schemaErrs := ValidateRuleBytes(set, raw)
		if len(schemaErrs) > 0 {
			results[path] = schemaErrs
			return nil
		}
		// Schema-valid; for v2 rules, run cross-checks #3–#8.
		if parseRuleVersion(raw) == 2 {
			if cErrs := validateV2CrossChecks(raw, catalog, owners); len(cErrs) > 0 {
				results[path] = append(results[path], cErrs...)
			}
		}
		return nil
	})

	if walkErr != nil {
		// fs.ErrNotExist is operational, not a validation failure.
		if errors.Is(walkErr, fs.ErrNotExist) {
			return results, filesProcessed, nil
		}
		return nil, 0, fmt.Errorf("walk %s: %w", dir, walkErr)
	}
	return results, filesProcessed, nil
}

// ruleDocV2 is the typed shape of a v2 rule for cross-checks.
// Schema validation runs first; this typed parse is best-effort
// (the schema has already enforced required-field presence).
type ruleDocV2 struct {
	Version int          `yaml:"version"`
	Entity  string       `yaml:"entity"`
	Mode    string       `yaml:"mode"`
	Source  ruleSourceV2 `yaml:"source"`
	Checks  []ruleCheckV2 `yaml:"checks"`
}

type ruleSourceV2 struct {
	Type string `yaml:"type"`
}

type ruleCheckV2 struct {
	CheckID string         `yaml:"check_id"`
	Kind    string         `yaml:"kind"`
	Params  map[string]any `yaml:"params,omitempty"`
}

// validateV2CrossChecks runs lint cross-checks #3–#8 on a
// schema-valid v2 rule. Cross-checks #1, #2, #4, #7 first-half,
// #9, #10, #11 are enforced by the v2 schema itself (mode field
// required, kind regex, source oneOf, window structure, duration
// pattern). The cross-checks below cover the cross-file and
// semantic-cross-field cases that pure JSON Schema cannot express:
//
//   #3 rule.mode == owners.entities[rule.entity].mode
//   #4 (semantic half) kind prefix matches rule.mode
//   #5 every kind is declared in the catalog
//   #6 per-rule check params validate against catalog.params_schema
//   #7 source.type matches rule.mode (bigquery↔set, kafka↔record)
//   #8 source.type matches catalog.kind.source_mode for every check
func validateV2CrossChecks(ruleBytes []byte, catalog *Catalog, owners *Owners) []ValidationError {
	var rule ruleDocV2
	if err := yaml.Unmarshal(ruleBytes, &rule); err != nil {
		// Schema validation has already passed, so a parse error
		// here is unexpected. Surface it but do not panic.
		return []ValidationError{{Message: fmt.Sprintf("v2 cross-check: yaml parse: %v", err)}}
	}

	var errs []ValidationError

	// #7 source.type matches rule.mode.
	if want, ok := expectedSourceTypeFor(rule.Mode); ok && rule.Source.Type != want {
		errs = append(errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0023 cross-check #7: rule mode %q expects source.type %q but found %q",
			rule.Mode, want, rule.Source.Type)})
	}

	// #3 rule.mode == owners[entity].mode (cross-file).
	if owners != nil && rule.Entity != "" {
		if ent, ok := owners.Entities[rule.Entity]; ok && ent.Mode != "" && ent.Mode != rule.Mode {
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0021 cross-check #3: rule mode %q for entity %q does not match owners entry mode %q",
				rule.Mode, rule.Entity, ent.Mode)})
		}
	}

	// #4 (semantic half) and #5/#6/#8 for each check.
	for _, c := range rule.Checks {
		prefix := kindPrefix(c.Kind)
		// #4 kind prefix must match rule.mode.
		if prefix != "" && prefix != rule.Mode {
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0022 cross-check #4: check %q kind %q has prefix %q which does not match rule mode %q",
				c.CheckID, c.Kind, prefix, rule.Mode)})
			// Subsequent kind-keyed checks are skipped for this
			// check: the catalog lookup would either miss or
			// produce a confusing secondary error.
			continue
		}

		// #5 kind exists in catalog.
		if catalog == nil {
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0022 cross-check #5: check %q kind %q cannot be resolved (no catalog loaded)",
				c.CheckID, c.Kind)})
			continue
		}
		entry, ok := catalog.Kind(c.Kind)
		if !ok {
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0022 cross-check #5: check %q kind %q is not declared in the catalog at %s",
				c.CheckID, c.Kind, catalog.Path)})
			continue
		}

		// #8 source.type matches catalog source_mode for this kind.
		if want, ok := expectedSourceTypeFor(entry.SourceMode); ok && rule.Source.Type != want {
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0023 cross-check #8: check %q kind %q has catalog source_mode %q (requires source.type %q) but rule source.type is %q",
				c.CheckID, c.Kind, entry.SourceMode, want, rule.Source.Type)})
		}

		// #6 params validates against the catalog kind's params_schema.
		// Absent params are validated as the empty object so that
		// required-field checks in the catalog params_schema still fire.
		params := c.Params
		if params == nil {
			params = map[string]any{}
		}
		if err := entry.ParamsSchema.Validate(any(params)); err != nil {
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0022 cross-check #6: check %q kind %q params validation: %v",
				c.CheckID, c.Kind, err)})
		}
	}

	return errs
}

// kindPrefix returns "set" or "record" from a kind string of the
// form "<prefix>.<name>". Returns the empty string if the kind has
// no dot or the prefix is unrecognised — the v2 schema would have
// rejected such a kind earlier.
func kindPrefix(kind string) string {
	idx := strings.IndexByte(kind, '.')
	if idx <= 0 {
		return ""
	}
	prefix := kind[:idx]
	switch prefix {
	case "set", "record":
		return prefix
	default:
		return ""
	}
}

// expectedSourceTypeFor maps a mode (or source_mode) value to the
// substrate descriptor type expected by ADR-0023. Returns ("", false)
// for unknown modes so the caller can decline to emit an error.
func expectedSourceTypeFor(mode string) (string, bool) {
	switch mode {
	case "set":
		return "bigquery", true
	case "record":
		return "kafka", true
	default:
		return "", false
	}
}
