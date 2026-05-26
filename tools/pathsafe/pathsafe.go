// path: tools/pathsafe/pathsafe.go

// Package pathsafe implements ADR-0044 §"Clause 3 — Path
// resolution" for external-artifact references in rule YAMLs.
// Both `dq-lint` and `dq-manifest publish` invoke Resolve to
// validate a `<field>_ref` value before reading the referenced
// file.
//
// Three rules enforced (per ADR-0044 §Clause 3):
//
//  1. No upward traversal in the reference path. Any `..` segment
//     in the literal reference value is a hard error before any
//     file-system access.
//  2. Symlink canonicalization. The resolved path is run through
//     filepath.EvalSymlinks before the containment check, so a
//     symlink under `rules/schemas/` pointing at `/etc/passwd`
//     cannot bypass containment.
//  3. Rules-tree containment. After symlink canonicalization, the
//     resolved absolute path MUST be a descendant of the rules
//     workspace root. Paths resolving outside `rules/` are a hard
//     error.
package pathsafe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrUpwardTraversal is returned when a reference value contains
// any `..` segment (rule 1 above).
var ErrUpwardTraversal = errors.New("pathsafe: reference contains upward-traversal segment")

// ErrAbsolutePath is returned when a reference value is an
// absolute path. Per ADR-0044 §Clause 3, references are relative
// to the rule file's directory; absolute paths are rejected
// before any filesystem access.
var ErrAbsolutePath = errors.New("pathsafe: reference must be a relative path")

// ErrOutsideRulesRoot is returned when a reference resolves
// outside the rules workspace root after symlink canonicalization
// (rule 3 above).
var ErrOutsideRulesRoot = errors.New("pathsafe: reference resolves outside the rules workspace root")

// ErrMissingFile is returned when the resolved path does not
// exist on disk. This is distinct from ErrOutsideRulesRoot because
// the path may be syntactically inside the rules root but point
// at a nonexistent file.
var ErrMissingFile = errors.New("pathsafe: referenced file does not exist")

// Resolve validates a `<field>_ref` value relative to the
// referring rule file's directory. Returns the canonicalized
// absolute path on success.
//
// Inputs:
//
//   - rulesRoot: absolute path to the rules workspace root (the
//     directory under which all references must resolve). Caller
//     resolves this once (e.g., from CLI flag or env config) and
//     passes it on every call.
//   - rulePath: absolute path of the rule YAML carrying the
//     reference. The reference is resolved relative to
//     filepath.Dir(rulePath).
//   - ref: the literal reference value from the YAML (a relative
//     path).
//
// Returns the absolute path that has passed all three checks. The
// caller reads + parses the file at this path.
func Resolve(rulesRoot, rulePath, ref string) (string, error) {
	// Reject absolute paths before any other check. Per ADR-0044
	// §Clause 3, references are relative to the rule file's
	// directory; an absolute path is a contract violation.
	if filepath.IsAbs(ref) {
		return "", fmt.Errorf("%w: %q", ErrAbsolutePath, ref)
	}
	// Rule 1: reject `..` anywhere in the reference. We check
	// before any filesystem access so a malicious ref doesn't
	// even trigger a symlink dereference.
	for _, seg := range strings.Split(filepath.ToSlash(ref), "/") {
		if seg == ".." {
			return "", fmt.Errorf("%w: %q", ErrUpwardTraversal, ref)
		}
	}

	// Resolve relative to the rule file's directory. filepath.Join
	// + filepath.Abs gives an absolute path; the path is NOT yet
	// canonicalized for symlinks.
	abs, err := filepath.Abs(filepath.Join(filepath.Dir(rulePath), ref))
	if err != nil {
		return "", fmt.Errorf("pathsafe: resolve %q: %w", ref, err)
	}

	// Rule 2: canonicalize via EvalSymlinks. This dereferences any
	// symlinks along the way so the containment check (rule 3)
	// sees the real target. EvalSymlinks fails on missing files —
	// we convert that to ErrMissingFile for caller ergonomics.
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %q", ErrMissingFile, ref)
		}
		return "", fmt.Errorf("pathsafe: canonicalize %q: %w", ref, err)
	}

	// We also need rulesRoot's canonical form so the containment
	// comparison is apples-to-apples (e.g., on macOS where /tmp is
	// a symlink to /private/tmp). filepath.Abs first so callers
	// can pass a relative rulesRoot; EvalSymlinks then dereferences
	// any symlinks on the way.
	absRoot, err := filepath.Abs(rulesRoot)
	if err != nil {
		return "", fmt.Errorf("pathsafe: absolutize rules root %q: %w", rulesRoot, err)
	}
	rootCanonical, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", fmt.Errorf("pathsafe: canonicalize rules root %q: %w", rulesRoot, err)
	}

	// Rule 3: containment. The resolved path must be either
	// equal to rootCanonical or a descendant (separator-safe
	// comparison via filepath.Rel).
	rel, err := filepath.Rel(rootCanonical, canonical)
	if err != nil {
		return "", fmt.Errorf("pathsafe: relative path %q to %q: %w", canonical, rootCanonical, err)
	}
	// `rel == "."` means the resolved path equals rootCanonical
	// — i.e., the rules-root directory itself. That is INSIDE
	// the rules root (caller's read step would surface "is a
	// directory" if the ref points there). Only reject `..` and
	// `../*` results.
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: %q resolved to %q (rules root %q)", ErrOutsideRulesRoot, ref, canonical, rootCanonical)
	}

	return canonical, nil
}
