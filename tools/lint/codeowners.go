// path: tools/lint/codeowners.go

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// CodeOwnersGroups is the set of @org/team and @user identifiers
// that appear as reviewer tokens in a parsed CODEOWNERS file.
// The linter cross-checks `_owners.yaml`'s `owner:` values against
// this set per ADR-0037.
//
// Scope is bounded behaviorally by the parse: every rule line's
// first whitespace-separated field (the path-pattern column) is
// discarded before the reviewer-token regex is applied. This drops
// `*` (the default-path token) and any other path pattern without
// testing it against the identifier regex.
//
// The linter does NOT validate path-rule semantics (which group
// owns which path) — that is reserved as a future additive
// extension per ADR-0015 §11.
type CodeOwnersGroups struct {
	set  map[string]struct{}
	Path string // for diagnostics; empty in the disable case
}

// codeOwnersGroupPattern matches the GitHub CODEOWNERS reviewer-
// identifier shape: `@<user>` or `@<org>/<team>`. The regex does
// NOT match bare emails (CODEOWNERS allows them; the linter
// ignores them — `_owners.yaml`'s `owner:` is committed to be a
// group identifier per ADR-0015 §2).
var codeOwnersGroupPattern = regexp.MustCompile(`^@[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)?$`)

// LoadCodeOwners reads a CODEOWNERS file and returns its group
// inventory. An empty path returns an empty (but non-nil) set to
// signal the cross-check is disabled — the call site does not
// need a nil guard. An I/O error is returned only when path is
// non-empty and the file cannot be read.
func LoadCodeOwners(path string) (*CodeOwnersGroups, error) {
	groups := &CodeOwnersGroups{set: map[string]struct{}{}, Path: path}
	if path == "" {
		return groups, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read codeowners %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			// A rule line without reviewers is a CODEOWNERS-syntax
			// "unset path" directive; no group identifier to harvest.
			continue
		}
		// Discard the path-pattern column (fields[0]); apply the
		// reviewer-token regex to the remainder.
		for _, tok := range fields[1:] {
			if codeOwnersGroupPattern.MatchString(tok) {
				groups.set[tok] = struct{}{}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan codeowners %s: %w", path, err)
	}
	return groups, nil
}

// Contains reports whether group appears in the inventory. A nil
// receiver or an empty set returns false; the cross-check caller
// treats those as the disable case and emits no errors.
func (g *CodeOwnersGroups) Contains(group string) bool {
	if g == nil || len(g.set) == 0 {
		return false
	}
	_, ok := g.set[group]
	return ok
}

// Slice returns the inventory as a sorted slice for diagnostic
// messages. Returns nil when the set is empty.
func (g *CodeOwnersGroups) Slice() []string {
	if g == nil || len(g.set) == 0 {
		return nil
	}
	out := make([]string, 0, len(g.set))
	for k := range g.set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
