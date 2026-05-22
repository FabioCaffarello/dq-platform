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

// ValidateRuleFile reads a rule YAML from disk and validates it.
func ValidateRuleFile(schema *jsonschema.Schema, path string) []ValidationError {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []ValidationError{{Message: fmt.Sprintf("read error: %v", err)}}
	}
	return ValidateRule(schema, raw)
}

// ValidateRulesDir walks the given directory for *.yaml files
// (excluding any directory named "_schema" at any depth — the
// schema mirror lives at rules/_schema/ and is not a rule). Returns:
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
func ValidateRulesDir(schema *jsonschema.Schema, dir string, verbose bool) (results map[string][]ValidationError, filesProcessed int, err error) {
	results = make(map[string][]ValidationError)

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkCallbackErr error) error {
		if walkCallbackErr != nil {
			return walkCallbackErr
		}
		// Skip any directory named "_schema". The skip is broader
		// than strictly required (it would skip e.g.
		// some-rule-dir/_schema/ too); the v1 rules workspace does
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
		if errs := ValidateRuleFile(schema, path); len(errs) > 0 {
			results[path] = errs
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
