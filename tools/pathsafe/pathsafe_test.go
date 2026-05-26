// path: tools/pathsafe/pathsafe_test.go

package pathsafe

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setup builds a temporary rules tree:
//
//	rulesRoot/
//	├─ entity.yaml      (the simulated rule file)
//	├─ schemas/
//	│  └─ inside.json   (a valid in-tree reference target)
//	└─ outside-target/  (sibling dir outside `rules/`)
//	   └─ secret.json
//
// rulesRoot is canonicalized so EvalSymlinks-style /tmp ↔
// /private/tmp differences on macOS don't break tests.
func setup(t *testing.T) (rulesRoot, rulePath, outsideTarget string) {
	t.Helper()
	tmp := t.TempDir()
	canonical, err := filepath.EvalSymlinks(tmp)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", tmp, err)
	}
	rulesRoot = filepath.Join(canonical, "rules")
	if err := os.MkdirAll(filepath.Join(rulesRoot, "schemas"), 0o755); err != nil {
		t.Fatalf("mkdir rules/schemas: %v", err)
	}
	rulePath = filepath.Join(rulesRoot, "entity.yaml")
	if err := os.WriteFile(rulePath, []byte("# rule"), 0o644); err != nil {
		t.Fatalf("write rule: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesRoot, "schemas", "inside.json"), []byte(`{"ok": true}`), 0o644); err != nil {
		t.Fatalf("write inside.json: %v", err)
	}
	outsideTarget = filepath.Join(canonical, "outside-target")
	if err := os.MkdirAll(outsideTarget, 0o755); err != nil {
		t.Fatalf("mkdir outside-target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outsideTarget, "secret.json"), []byte(`{"secret": true}`), 0o644); err != nil {
		t.Fatalf("write secret.json: %v", err)
	}
	return rulesRoot, rulePath, outsideTarget
}

func TestResolve_HappyPath(t *testing.T) {
	rulesRoot, rulePath, _ := setup(t)
	got, err := Resolve(rulesRoot, rulePath, "schemas/inside.json")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := filepath.Join(rulesRoot, "schemas", "inside.json")
	if got != want {
		t.Errorf("Resolve = %q; want %q", got, want)
	}
}

func TestResolve_UpwardTraversalRejected(t *testing.T) {
	rulesRoot, rulePath, _ := setup(t)
	cases := []string{
		"../secret.json",
		"schemas/../../escape.json",
		"a/b/../../../out.json",
	}
	for _, ref := range cases {
		t.Run(ref, func(t *testing.T) {
			_, err := Resolve(rulesRoot, rulePath, ref)
			if !errors.Is(err, ErrUpwardTraversal) {
				t.Errorf("Resolve(%q) err = %v; want ErrUpwardTraversal", ref, err)
			}
		})
	}
}

func TestResolve_SymlinkEscapeRejected(t *testing.T) {
	rulesRoot, rulePath, outsideTarget := setup(t)
	// schemas/escape.json → ../outside-target/secret.json
	linkPath := filepath.Join(rulesRoot, "schemas", "escape.json")
	target := filepath.Join(outsideTarget, "secret.json")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink creation not permitted on this filesystem: %v", err)
	}
	_, err := Resolve(rulesRoot, rulePath, "schemas/escape.json")
	if !errors.Is(err, ErrOutsideRulesRoot) {
		t.Errorf("Resolve(symlink-escape) err = %v; want ErrOutsideRulesRoot", err)
	}
}

func TestResolve_SymlinkToInTreeAccepted(t *testing.T) {
	// A symlink that resolves *inside* rules/ should pass.
	rulesRoot, rulePath, _ := setup(t)
	linkPath := filepath.Join(rulesRoot, "schemas", "alias.json")
	target := filepath.Join(rulesRoot, "schemas", "inside.json")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Skipf("symlink creation not permitted: %v", err)
	}
	got, err := Resolve(rulesRoot, rulePath, "schemas/alias.json")
	if err != nil {
		t.Fatalf("Resolve(in-tree symlink): %v", err)
	}
	// EvalSymlinks resolves the alias to the original target.
	if !strings.HasSuffix(got, filepath.Join("schemas", "inside.json")) {
		t.Errorf("Resolve(symlink) = %q; want suffix schemas/inside.json", got)
	}
}

func TestResolve_MissingFileRejected(t *testing.T) {
	rulesRoot, rulePath, _ := setup(t)
	_, err := Resolve(rulesRoot, rulePath, "schemas/nope.json")
	if !errors.Is(err, ErrMissingFile) {
		t.Errorf("Resolve(missing) err = %v; want ErrMissingFile", err)
	}
}

func TestResolve_AbsolutePathRejected(t *testing.T) {
	rulesRoot, rulePath, outsideTarget := setup(t)
	// Absolute references are rejected before any filesystem
	// access. The check fires whether the target is inside or
	// outside rules/ — references MUST be relative per ADR-0044
	// §Clause 3.
	absRef := filepath.Join(outsideTarget, "secret.json")
	_, err := Resolve(rulesRoot, rulePath, absRef)
	if !errors.Is(err, ErrAbsolutePath) {
		t.Errorf("Resolve(absolute outside) err = %v; want ErrAbsolutePath", err)
	}
	// Also reject absolute paths that point inside rules/ — the
	// shape of the reference is what's wrong, not the target.
	absInTree := filepath.Join(rulesRoot, "schemas", "inside.json")
	if _, err := Resolve(rulesRoot, rulePath, absInTree); !errors.Is(err, ErrAbsolutePath) {
		t.Errorf("Resolve(absolute in-tree) err = %v; want ErrAbsolutePath", err)
	}
}

func TestResolve_RulesRootSelfRejected(t *testing.T) {
	// `.` refers to the rule file's directory itself, which IS
	// inside rulesRoot but is not a file. The containment check
	// allows it; the caller's read step would surface the
	// "is a directory" error. Test that pathsafe doesn't itself
	// reject this — caller is responsible for file-type checks.
	rulesRoot, rulePath, _ := setup(t)
	got, err := Resolve(rulesRoot, rulePath, ".")
	if err != nil {
		t.Fatalf("Resolve(\".\"): %v", err)
	}
	if !strings.HasPrefix(got, rulesRoot) {
		t.Errorf("Resolve(\".\") = %q; want prefix %q", got, rulesRoot)
	}
}
