// path: tools/lint/codeowners_test.go

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeCodeowners writes content to a temp file and returns its path.
func writeCodeowners(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "CODEOWNERS")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write codeowners: %v", err)
	}
	return path
}

func TestLoadCodeOwners_EmptyPath_DisableCase(t *testing.T) {
	groups, err := LoadCodeOwners("")
	if err != nil {
		t.Fatalf("LoadCodeOwners(\"\"): %v", err)
	}
	if groups == nil {
		t.Fatal("LoadCodeOwners(\"\") returned nil; want non-nil empty set")
	}
	if got := groups.Slice(); got != nil {
		t.Errorf("empty set Slice() = %v; want nil", got)
	}
	if groups.Contains("@anything") {
		t.Error("empty set should not contain anything")
	}
}

func TestLoadCodeOwners_MissingFile_ReturnsError(t *testing.T) {
	_, err := LoadCodeOwners("/no/such/CODEOWNERS")
	if err == nil {
		t.Fatal("LoadCodeOwners(missing) returned nil; want error so main.go exits 2")
	}
}

func TestLoadCodeOwners_CommentsAndBlankLinesIgnored(t *testing.T) {
	path := writeCodeowners(t, `
# This is a comment
#

# Another comment
`)
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	if got := groups.Slice(); got != nil {
		t.Errorf("comment-only file Slice() = %v; want nil", got)
	}
}

func TestLoadCodeOwners_DiscardsDefaultPathToken(t *testing.T) {
	// CODEOWNERS line shape: `<path-pattern> <reviewer> [<reviewer>...]`.
	// The path-pattern column is discarded; only reviewer tokens
	// matching @<org>/<team> or @<user> are harvested.
	path := writeCodeowners(t, "*    @PLACEHOLDER-org/platform-team\n")
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	got := groups.Slice()
	want := []string{"@PLACEHOLDER-org/platform-team"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slice() = %v; want %v", got, want)
	}
	// The * default-path token must NOT appear in the set.
	if groups.Contains("*") {
		t.Error("default-path token `*` must not be harvested as a group identifier")
	}
}

func TestLoadCodeOwners_MultipleReviewersPerLine(t *testing.T) {
	path := writeCodeowners(t, "/deploy/overlays/prod/ @PLACEHOLDER-org/platform-team @PLACEHOLDER-org/sre\n")
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	got := groups.Slice()
	want := []string{"@PLACEHOLDER-org/platform-team", "@PLACEHOLDER-org/sre"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slice() = %v; want %v", got, want)
	}
}

func TestLoadCodeOwners_DuplicatesDeduped(t *testing.T) {
	path := writeCodeowners(t, `
*                  @PLACEHOLDER-org/platform-team
/engine/           @PLACEHOLDER-org/platform-team
/tools/            @PLACEHOLDER-org/platform-team
`)
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	got := groups.Slice()
	want := []string{"@PLACEHOLDER-org/platform-team"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slice() = %v; want %v (deduped)", got, want)
	}
}

func TestLoadCodeOwners_BareEmailIgnored(t *testing.T) {
	// GitHub allows bare emails as CODEOWNERS reviewers; the linter
	// ignores them because `_owners.yaml`'s `owner:` is committed to
	// be a group identifier per ADR-0015 §2.
	path := writeCodeowners(t, "*    user@example.com @PLACEHOLDER-org/platform-team\n")
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	got := groups.Slice()
	want := []string{"@PLACEHOLDER-org/platform-team"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slice() = %v; want %v (email ignored)", got, want)
	}
}

func TestLoadCodeOwners_BareUserToken(t *testing.T) {
	// @username (no slash) is a valid GitHub CODEOWNERS reviewer.
	path := writeCodeowners(t, "*    @octocat @PLACEHOLDER-org/platform-team\n")
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	got := groups.Slice()
	want := []string{"@PLACEHOLDER-org/platform-team", "@octocat"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slice() = %v; want %v", got, want)
	}
}

func TestLoadCodeOwners_PathWithoutReviewer_Skipped(t *testing.T) {
	// CODEOWNERS allows "unset path" lines (just a pattern, no
	// reviewer). The linter must not blow up on them.
	path := writeCodeowners(t, "/path-with-no-reviewer\n*    @PLACEHOLDER-org/platform-team\n")
	groups, err := LoadCodeOwners(path)
	if err != nil {
		t.Fatalf("LoadCodeOwners: %v", err)
	}
	got := groups.Slice()
	want := []string{"@PLACEHOLDER-org/platform-team"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slice() = %v; want %v", got, want)
	}
}

func TestLoadCodeOwners_RealRepoCODEOWNERS(t *testing.T) {
	// Golden test against the repository's own .github/CODEOWNERS.
	// It must parse without error and contain the three groups
	// committed by ADR-0015 §2.
	repoPath, err := filepath.Abs("../../.github/CODEOWNERS")
	if err != nil {
		t.Fatalf("repo CODEOWNERS path: %v", err)
	}
	groups, err := LoadCodeOwners(repoPath)
	if err != nil {
		t.Fatalf("LoadCodeOwners(real): %v", err)
	}
	for _, wantGroup := range []string{
		"@PLACEHOLDER-org/platform-team",
		"@PLACEHOLDER-org/sre",
		"@PLACEHOLDER-org/rules-authors",
	} {
		if !groups.Contains(wantGroup) {
			t.Errorf("real CODEOWNERS missing %q; got %v", wantGroup, groups.Slice())
		}
	}
	if groups.Contains("*") {
		t.Error("real CODEOWNERS Contains(`*`) = true; default-path token must be discarded")
	}
}

func TestCodeOwnersGroups_NilSafe(t *testing.T) {
	var g *CodeOwnersGroups
	if g.Contains("@anything") {
		t.Error("nil receiver Contains() should return false")
	}
	if got := g.Slice(); got != nil {
		t.Errorf("nil receiver Slice() = %v; want nil", got)
	}
}
