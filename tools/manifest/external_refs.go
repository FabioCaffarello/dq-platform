// path: tools/manifest/external_refs.go

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dq-platform/tools/pathsafe"

	"gopkg.in/yaml.v3"
)

// catalogKindRefs maps a kind name to its external-eligible field
// inventory. The publisher uses this to validate that every
// `<field>_ref` in a rule's params targets a field the catalog
// permits to be externalized; otherwise the rule fails publish
// verification per ADR-0044 §"Clause 2".
//
// The publisher does NOT re-validate inlined content against the
// field's sub-schema — that's the linter's job (cross-check #12).
// The publisher trusts that a lint-clean rule's inlined content
// is valid, but still enforces eligibility because:
//
//  1. The eligibility list is the contract surface for which
//     fields the publisher may inline (defense in depth against
//     a lint-skipping operator).
//  2. Inlining a non-eligible field would silently expand the
//     ADR-0044 contract beyond what was authorized.
type catalogKindRefs map[string]map[string]struct{}

// loadCatalogKindRefs reads the catalog YAML and extracts the
// per-kind external_eligible_fields list. Returns an empty (but
// non-nil) map if the catalog file is missing or has no kinds
// with external-eligible fields — neither case is a failure.
func loadCatalogKindRefs(catalogPath string) (catalogKindRefs, error) {
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return catalogKindRefs{}, nil
		}
		return nil, fmt.Errorf("read catalog %s: %w", catalogPath, err)
	}
	var doc struct {
		Kinds []struct {
			Name                   string   `yaml:"name"`
			ExternalEligibleFields []string `yaml:"external_eligible_fields"`
		} `yaml:"kinds"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse catalog yaml %s: %w", catalogPath, err)
	}
	refs := catalogKindRefs{}
	for _, k := range doc.Kinds {
		if len(k.ExternalEligibleFields) == 0 {
			continue
		}
		set := make(map[string]struct{}, len(k.ExternalEligibleFields))
		for _, fieldName := range k.ExternalEligibleFields {
			set[fieldName] = struct{}{}
		}
		refs[k.Name] = set
	}
	return refs, nil
}

// inlineExternalRefs walks the rule YAML body looking for
// `<field>_ref` keys under each check's params. For each found
// reference, it resolves the path via pathsafe, reads + parses
// the referenced file (JSON or YAML by extension), and substitutes
// the parsed content for the `_ref` key. The returned bytes are
// the re-serialized rule YAML; the publisher uses them in place
// of the operator's original body for content addressing per
// ADR-0044 §Clause 4 + ADR-0005 §3.
//
// Returns the original body unchanged when:
//
//   - The rule is v1 (no record-mode params, no external refs).
//   - The rule has no checks with external-eligible kinds.
//   - The rule has no `_ref` keys.
//
// Returns an error wrapping ErrVerificationFailed when:
//
//   - A `_ref` key names a non-eligible field (catalog enforces).
//   - pathsafe rejects the reference (`..`, symlink-escape,
//     missing file, absolute path).
//   - The referenced file fails to parse.
//   - Both `<field>` and `<field>_ref` are present (the linter's
//     cross-check #12 should have caught this; the publisher
//     defends against lint-skipping operators).
func inlineExternalRefs(body []byte, rulePath, rulesRoot string, refs catalogKindRefs) ([]byte, error) {
	if len(refs) == 0 {
		return body, nil
	}

	// Decode into a generic map so we can mutate params subtrees
	// without re-modeling every rule field. yaml.Node would
	// preserve formatting but is heavier; for content-addressed
	// publish, re-serializing into canonical YAML is fine — the
	// content hash is over the parsed semantic content, not the
	// operator's exact whitespace.
	var doc map[string]any
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("inlineExternalRefs: parse %s: %w", rulePath, err)
	}

	checksRaw, ok := doc["checks"].([]any)
	if !ok {
		// v1 rules or any rule without a `checks` list — nothing
		// to inline.
		return body, nil
	}

	mutated := false
	for i, ckRaw := range checksRaw {
		ck, ok := ckRaw.(map[string]any)
		if !ok {
			continue
		}
		kindStr, _ := ck["kind"].(string)
		eligible, ok := refs[kindStr]
		if !ok || len(eligible) == 0 {
			continue
		}
		paramsRaw, ok := ck["params"].(map[string]any)
		if !ok {
			continue
		}
		newParams, changed, err := resolveCheckParams(paramsRaw, eligible, rulePath, rulesRoot, ck["check_id"])
		if err != nil {
			return nil, err
		}
		if changed {
			ck["params"] = newParams
			checksRaw[i] = ck
			mutated = true
		}
	}

	if !mutated {
		return body, nil
	}
	doc["checks"] = checksRaw
	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("inlineExternalRefs: re-marshal %s: %w", rulePath, err)
	}
	return out, nil
}

// resolveCheckParams inlines references in one check's params
// map. Returns the new params, whether anything changed, and an
// error wrapping ErrVerificationFailed on contract violations.
func resolveCheckParams(params map[string]any, eligible map[string]struct{}, rulePath, rulesRoot string, checkID any) (map[string]any, bool, error) {
	checkIDStr := fmt.Sprint(checkID)

	// Build a list of `_ref` keys present in params, so we can
	// reject non-eligible ones in one pass.
	var refKeys []string
	for k := range params {
		if strings.HasSuffix(k, "_ref") {
			refKeys = append(refKeys, k)
		}
	}

	// Reject non-eligible refs first (the catalog says they
	// cannot be externalized).
	for _, refKey := range refKeys {
		fieldName := strings.TrimSuffix(refKey, "_ref")
		if _, ok := eligible[fieldName]; !ok {
			return nil, false, fmt.Errorf("publisher: check %q uses %q but %q is not declared external-eligible in the catalog: %w",
				checkIDStr, refKey, fieldName, ErrVerificationFailed)
		}
	}

	// Resolve each eligible ref.
	changed := false
	for fieldName := range eligible {
		refKey := fieldName + "_ref"
		_, hasField := params[fieldName]
		refRaw, hasRef := params[refKey]
		if !hasRef {
			continue
		}
		if hasField {
			return nil, false, fmt.Errorf("publisher: check %q has both %q and %q present; exactly one is permitted per ADR-0044 cross-check #12: %w",
				checkIDStr, fieldName, refKey, ErrVerificationFailed)
		}
		ref, ok := refRaw.(string)
		if !ok {
			return nil, false, fmt.Errorf("publisher: check %q %q has non-string value (%T): %w",
				checkIDStr, refKey, refRaw, ErrVerificationFailed)
		}
		resolved, err := pathsafe.Resolve(rulesRoot, rulePath, ref)
		if err != nil {
			return nil, false, fmt.Errorf("publisher: check %q %s %q: %w: %w",
				checkIDStr, refKey, ref, err, ErrVerificationFailed)
		}
		raw, err := os.ReadFile(resolved)
		if err != nil {
			return nil, false, fmt.Errorf("publisher: check %q %s %q: read %s: %w: %w",
				checkIDStr, refKey, ref, resolved, err, ErrVerificationFailed)
		}
		parsed, parseErr := parseExternalArtifact(resolved, raw)
		if parseErr != nil {
			return nil, false, fmt.Errorf("publisher: check %q %s %q: parse %s: %w: %w",
				checkIDStr, refKey, ref, resolved, parseErr, ErrVerificationFailed)
		}
		params[fieldName] = parsed
		delete(params, refKey)
		changed = true
	}

	return params, changed, nil
}

// parseExternalArtifact decodes a referenced file as JSON or YAML
// based on extension. Returns the decoded any-tree, normalized
// for round-trip serialization (yaml.v3 already prefers
// map[string]interface{}, so the conversion is a no-op in the
// common path).
func parseExternalArtifact(path string, raw []byte) (any, error) {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".json":
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("json: %w", err)
		}
		return v, nil
	case ".yaml", ".yml":
		var v any
		if err := yaml.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("yaml: %w", err)
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported extension %q (want .json, .yaml, or .yml)", ext)
	}
}
