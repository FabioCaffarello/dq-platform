// path: engine/internal/dsl/spec/parse.go

package spec

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// maxIdentifierLen is the per-field length ceiling for the entity
// and check_id strings, matching the v1 JSON schema's maxLength: 200.
const maxIdentifierLen = 200

// identifierPattern matches the entity / check_id pattern from
// engine/internal/dsl/schema/v1.schema.json: ASCII alphanumeric
// plus underscore, dot, hyphen. Pipe is forbidden per ADR-0002
// §2 input safety (the rule's entity flows into the execution_id
// formula).
var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// Parse reads a rule YAML body and returns a validated RuleSpec.
// Unknown top-level or per-check fields are rejected (strict
// decode via KnownFields(true)), and per-field invariants are
// enforced before the spec is returned.
//
// The parser does not validate `kind` against any whitelist; the
// evaluator (engine/internal/eval) dispatches on kind and returns
// ResultError for unrecognized kinds per ADR-0004 CC1.
func Parse(body []byte) (RuleSpec, error) {
	var r RuleSpec
	dec := yaml.NewDecoder(bytes.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(&r); err != nil {
		return RuleSpec{}, fmt.Errorf("rule yaml decode: %w", err)
	}

	if err := validate(&r); err != nil {
		return RuleSpec{}, err
	}
	return r, nil
}

func validate(r *RuleSpec) error {
	if r.Version != 1 {
		return fmt.Errorf("rule yaml: version %d is not supported (expected 1 per ADR-0001)", r.Version)
	}
	if err := validateIdentifier("entity", r.Entity); err != nil {
		return err
	}
	if len(r.Checks) == 0 {
		return errors.New("rule yaml: checks must be a non-empty array")
	}
	for i, c := range r.Checks {
		if err := validateCheck(i, c); err != nil {
			return err
		}
	}
	return nil
}

func validateCheck(index int, c Check) error {
	if err := validateIdentifier(fmt.Sprintf("checks[%d].check_id", index), c.CheckID); err != nil {
		return err
	}
	if c.Kind == "" {
		return fmt.Errorf("rule yaml: checks[%d].kind must be a non-empty string", index)
	}
	// kind is not one of the five execution_id formula inputs
	// (ADR-0002 §1), so ADR-0002 §2's pipe-safety invariant does
	// not apply to it. The v1 schema treats kind as a free-form
	// string with minLength: 1; the evaluator (engine/internal/eval)
	// dispatches on kind and returns ResultError for unrecognized
	// values per ADR-0004 CC1.
	return nil
}

// validateIdentifier applies the shared invariants for entity and
// check_id strings: non-empty, length-bounded, identifier-pattern
// match, no ASCII pipe.
func validateIdentifier(field, value string) error {
	if value == "" {
		return fmt.Errorf("rule yaml: %s must be a non-empty string", field)
	}
	if len(value) > maxIdentifierLen {
		return fmt.Errorf("rule yaml: %s exceeds %d bytes", field, maxIdentifierLen)
	}
	if strings.ContainsRune(value, '|') {
		return fmt.Errorf("rule yaml: %s contains forbidden ASCII pipe character (ADR-0002 §2)", field)
	}
	if !identifierPattern.MatchString(value) {
		return fmt.Errorf("rule yaml: %s %q does not match %s", field, value, identifierPattern.String())
	}
	return nil
}
