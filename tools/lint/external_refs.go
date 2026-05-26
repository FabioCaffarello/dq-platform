// path: tools/lint/external_refs.go

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"dq-platform/tools/pathsafe"

	"gopkg.in/yaml.v3"
)

// checkExternalReferences implements ADR-0044's lint cross-check
// #12. For each external-eligible field in the kind's catalog
// entry, the cross-check enforces:
//
//   - **Exactly one of `<field>` or `<field>_ref` must be
//     present.** Both present → mutual-exclusion error. Neither
//     present → at-least-one error. (The catalog's params_schema
//     no longer marks `<field>` as required, since either side
//     can satisfy the contract; this cross-check is what carries
//     the "must specify one" obligation.)
//
//   - **Non-eligible `_ref` keys are rejected.** A `_ref` key
//     for a field NOT declared external-eligible (e.g.,
//     `aggregation_ref` when only `schema` is eligible) is a
//     catalog-contract violation.
//
//   - **If `<field>_ref` is present, it is resolved via the
//     shared `pathsafe` helper.** The referenced file is read,
//     parsed as JSON or YAML (extension-driven), and validated
//     against the field's sub-schema from the catalog.
//
// The cross-check fires once per check per rule. errors include
// the check_id so multiple failing checks in one rule remain
// distinguishable.
func checkExternalReferences(rulesRoot, rulePath string, rule ruleDocV2, catalog *Catalog) []ValidationError {
	if catalog == nil {
		return nil
	}
	var errs []ValidationError
	for _, c := range rule.Checks {
		entry, ok := catalog.Kind(c.Kind)
		if !ok {
			// Cross-check #5 already reports unknown kinds.
			continue
		}
		errs = append(errs, validateCheckExternalRefs(rulesRoot, rulePath, c, entry)...)
	}
	return errs
}

// validateCheckExternalRefs runs the per-check half of cross-check
// #12 for one check inside a rule.
func validateCheckExternalRefs(rulesRoot, rulePath string, check ruleCheckV2, entry *CatalogKind) []ValidationError {
	var errs []ValidationError
	params := check.Params
	if params == nil {
		params = map[string]any{}
	}

	// Build a quick set of which `_ref` keys are present in params
	// so we can detect non-eligible `_ref` usage in one pass.
	refKeys := map[string]bool{}
	for k := range params {
		if strings.HasSuffix(k, "_ref") {
			refKeys[k] = true
		}
	}

	// For each declared external-eligible field, enforce the
	// mutual-exclusion + at-least-one rule.
	for fieldName, fieldSchema := range entry.ExternalEligibleFields {
		refKey := fieldName + "_ref"
		_, hasField := params[fieldName]
		refRaw, hasRef := params[refKey]

		switch {
		case hasField && hasRef:
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0044 cross-check #12: check %q kind %q has both %q and %q present; exactly one is permitted",
				check.CheckID, check.Kind, fieldName, refKey)})
		case !hasField && !hasRef:
			errs = append(errs, ValidationError{Message: fmt.Sprintf(
				"ADR-0044 cross-check #12: check %q kind %q has neither %q nor %q present; exactly one is required",
				check.CheckID, check.Kind, fieldName, refKey)})
		case hasRef:
			// Resolve, read, parse, validate.
			if resolveErr := resolveExternalRef(rulesRoot, rulePath, check, fieldName, refRaw, fieldSchema, &errs); resolveErr {
				// resolveExternalRef already appended a descriptive
				// error; nothing more to add here.
				_ = resolveErr
			}
		}

		// Mark this field's _ref key as known so it's not flagged
		// in the non-eligible loop below.
		delete(refKeys, refKey)
	}

	// Any remaining _ref keys are not declared external-eligible.
	for refKey := range refKeys {
		fieldName := strings.TrimSuffix(refKey, "_ref")
		errs = append(errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0044 cross-check #12: check %q kind %q uses %q but %q is not declared external-eligible in the catalog",
			check.CheckID, check.Kind, refKey, fieldName)})
	}

	return errs
}

// resolveExternalRef handles the read+parse+validate pipeline for
// one `_ref` value. Returns true if it appended an error to errs.
func resolveExternalRef(rulesRoot, rulePath string, check ruleCheckV2, fieldName string, refRaw any, fieldSchema interface {
	Validate(any) error
}, errs *[]ValidationError) bool {
	ref, ok := refRaw.(string)
	if !ok {
		*errs = append(*errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0044 cross-check #12: check %q has %s_ref of non-string type (%T)",
			check.CheckID, fieldName, refRaw)})
		return true
	}
	resolved, err := pathsafe.Resolve(rulesRoot, rulePath, ref)
	if err != nil {
		*errs = append(*errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0044 cross-check #12: check %q %s_ref %q: %v",
			check.CheckID, fieldName, ref, err)})
		return true
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		*errs = append(*errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0044 cross-check #12: check %q %s_ref %q: read %s: %v",
			check.CheckID, fieldName, ref, resolved, err)})
		return true
	}
	parsed, parseErr := parseExternalArtifact(resolved, raw)
	if parseErr != nil {
		*errs = append(*errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0044 cross-check #12: check %q %s_ref %q: parse %s: %v",
			check.CheckID, fieldName, ref, resolved, parseErr)})
		return true
	}
	if err := fieldSchema.Validate(parsed); err != nil {
		*errs = append(*errs, ValidationError{Message: fmt.Sprintf(
			"ADR-0044 cross-check #12: check %q %s_ref %q content does not match the field's catalog sub-schema: %v",
			check.CheckID, fieldName, ref, err)})
		return true
	}
	return false
}

// parseExternalArtifact decodes the referenced file as JSON or
// YAML based on extension. Returns the decoded any-tree (suitable
// for jsonschema validation).
func parseExternalArtifact(path string, raw []byte) (any, error) {
	switch {
	case strings.HasSuffix(path, ".json"):
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("json: %w", err)
		}
		return v, nil
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		var v any
		if err := yaml.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("yaml: %w", err)
		}
		// Convert YAML's map[interface{}]interface{} to
		// map[string]interface{} so jsonschema accepts it.
		return normalizeYAML(v), nil
	default:
		return nil, fmt.Errorf("unsupported extension (want .json, .yaml, or .yml)")
	}
}

// normalizeYAML converts YAML's map[interface{}]interface{} into
// map[string]interface{} recursively. yaml.v3 already produces
// map[string]interface{} in most cases, but defensive coverage
// here keeps jsonschema happy.
func normalizeYAML(v any) any {
	switch x := v.(type) {
	case map[any]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[fmt.Sprint(k)] = normalizeYAML(val)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = normalizeYAML(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, val := range x {
			out[i] = normalizeYAML(val)
		}
		return out
	default:
		return v
	}
}
