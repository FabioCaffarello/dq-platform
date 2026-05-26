// path: tools/lint/external_refs_test.go

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// lintExternalRefsDir runs the v2 cross-check pass against an
// isolated per-fixture directory inside testdata/external_refs/.
// Each fixture directory contains exactly one rule YAML + its
// `_owners.yaml` so the walker's error output is unambiguous.
func lintExternalRefsDir(t *testing.T, fixture string) string {
	t.Helper()
	dir := filepath.Join("testdata", "external_refs", fixture)
	set := schemaSet(t)
	cat := loadCatalog(t)
	owners := loadOwnersForV2(t, filepath.Join(dir, "_owners.yaml"))
	results, _, err := ValidateRulesDir(set, cat, owners, dir, false)
	if err != nil {
		t.Fatalf("ValidateRulesDir: %v", err)
	}
	var msgs []string
	for _, errs := range results {
		for _, e := range errs {
			msgs = append(msgs, e.Message)
		}
	}
	return strings.Join(msgs, "\n")
}

func TestCrossCheck12_ValidRef(t *testing.T) {
	got := lintExternalRefsDir(t, "valid_ref")
	if got != "" {
		t.Errorf("valid_ref produced errors; want none: %s", got)
	}
}

func TestCrossCheck12_BothPresent(t *testing.T) {
	got := lintExternalRefsDir(t, "both_present")
	if !strings.Contains(got, "cross-check #12") || !strings.Contains(got, "exactly one is permitted") {
		t.Errorf("both_present did not surface mutual-exclusion error; got: %s", got)
	}
}

func TestCrossCheck12_UpwardTraversal(t *testing.T) {
	got := lintExternalRefsDir(t, "upward_traversal")
	if !strings.Contains(got, "cross-check #12") || !strings.Contains(got, "upward-traversal") {
		t.Errorf("upward_traversal did not surface pathsafe error; got: %s", got)
	}
}

func TestCrossCheck12_MissingFile(t *testing.T) {
	got := lintExternalRefsDir(t, "missing_file")
	if !strings.Contains(got, "cross-check #12") || !strings.Contains(got, "does not exist") {
		t.Errorf("missing_file did not surface ErrMissingFile; got: %s", got)
	}
}

func TestCrossCheck12_NonEligible(t *testing.T) {
	got := lintExternalRefsDir(t, "non_eligible")
	if !strings.Contains(got, "cross-check #12") || !strings.Contains(got, "not declared external-eligible") {
		t.Errorf("non_eligible did not surface eligibility error; got: %s", got)
	}
}

func TestCrossCheck12_MalformedJSON(t *testing.T) {
	got := lintExternalRefsDir(t, "malformed_json")
	if !strings.Contains(got, "cross-check #12") || !strings.Contains(got, "parse") {
		t.Errorf("malformed_json did not surface parse error; got: %s", got)
	}
}

func TestCrossCheck12_ValidationFailure(t *testing.T) {
	got := lintExternalRefsDir(t, "validation_failure")
	if !strings.Contains(got, "cross-check #12") || !strings.Contains(got, "does not match") {
		t.Errorf("validation_failure did not surface sub-schema mismatch; got: %s", got)
	}
}
