// path: engine/internal/logging/levels.go

// Package logging implements ADR-0043's DQ_LOG_LEVELS contract:
// per-package log-level overrides expressed as comma-separated
// PACKAGE:LEVEL pairs, resolved by longest-prefix-match at dot
// boundaries.
package logging

import (
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
)

// RootKey is the reserved package name that replaces the
// substrate-wide root level when present in DQ_LOG_LEVELS.
const RootKey = "root"

// CanonicalInventory is the closed set of officially-supported
// leaf package names per ADR-0043 §"Clause 2 — Officially-supported
// package inventory". Adding a new engine/internal/<x>/ package
// extends this list additively per ADR-0001's compatibility model;
// the operations doc is amended in the same PR.
//
// Intermediate prefixes (e.g., bare "engine") are also valid
// override keys per ADR-0043 §"Clause 2" + Clause 3's longest-
// prefix-match rule; they are NOT enumerated here because the
// inventory tracks leaf names only — intermediate-prefix matches
// fall out of the resolution algorithm.
var CanonicalInventory = []string{
	RootKey,
	"engine",
	"engine.alerts",
	"engine.api",
	"engine.dsl",
	"engine.env",
	"engine.eval",
	"engine.loader",
	"engine.orphan",
	"engine.results",
	"engine.runner",
}

// identPattern enforces the IDENT grammar from ADR-0043 §"Clause 1":
// `[A-Za-z][A-Za-z0-9_]*`. Applied to each dot-separated segment of
// a PACKAGE token.
var identPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// ParseResult is the structured output of ParseLogLevels.
type ParseResult struct {
	// Levels is the parsed map: package-name → slog.Level. Empty
	// when DQ_LOG_LEVELS is unset or empty.
	Levels map[string]slog.Level

	// Ignored lists package names that parsed successfully but
	// are not in CanonicalInventory. Per ADR-0043 §"Clause 4 —
	// Error handling", these are silently honored (kept in
	// Levels) but the caller emits one info-level startup audit
	// line listing them so operators can audit.
	Ignored []string
}

// ParseLogLevels parses a DQ_LOG_LEVELS value per ADR-0043 §"Clause 1"
// grammar. Returns a ParseResult on success, or an error wrapping
// the first syntactic problem encountered. Per ADR-0043 §"Clause 4
// — Error handling", syntactic errors are fatal at the caller's
// discretion (the engine binary's loader-strict posture exits
// non-zero); unknown package names (not in CanonicalInventory) are
// recorded in result.Ignored, NOT returned as errors.
//
// Returns an empty (but non-nil) Levels map and a nil error for an
// unset or empty input value.
func ParseLogLevels(raw string) (ParseResult, error) {
	result := ParseResult{Levels: map[string]slog.Level{}}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return result, nil
	}

	canonical := canonicalSet()

	for _, rawPair := range strings.Split(raw, ",") {
		pair := strings.TrimSpace(rawPair)
		if pair == "" {
			return result, fmt.Errorf("DQ_LOG_LEVELS: empty pair (consecutive commas?)")
		}
		colon := strings.Index(pair, ":")
		if colon < 0 {
			return result, fmt.Errorf("DQ_LOG_LEVELS: pair %q missing ':' separator", pair)
		}
		pkg := strings.TrimSpace(pair[:colon])
		lvlRaw := strings.TrimSpace(pair[colon+1:])
		if pkg == "" {
			return result, fmt.Errorf("DQ_LOG_LEVELS: pair %q has empty package name", pair)
		}
		if lvlRaw == "" {
			return result, fmt.Errorf("DQ_LOG_LEVELS: pair %q has empty level value", pair)
		}
		if err := validatePackageIdent(pkg); err != nil {
			return result, fmt.Errorf("DQ_LOG_LEVELS: %w", err)
		}
		lvl, err := parseLevel(lvlRaw)
		if err != nil {
			return result, fmt.Errorf("DQ_LOG_LEVELS: pair %q: %w", pair, err)
		}
		if _, dup := result.Levels[pkg]; dup {
			return result, fmt.Errorf("DQ_LOG_LEVELS: package %q appears more than once", pkg)
		}
		result.Levels[pkg] = lvl
		if _, known := canonical[pkg]; !known {
			result.Ignored = append(result.Ignored, pkg)
		}
	}
	sort.Strings(result.Ignored)
	return result, nil
}

// validatePackageIdent enforces ADR-0043 §"Clause 1"'s PACKAGE
// grammar: dot-separated IDENT segments. Internal whitespace inside
// the token is fatal here (whitespace adjacent to ':' or ',' is
// trimmed by the caller before reaching this point).
func validatePackageIdent(pkg string) error {
	for _, segment := range strings.Split(pkg, ".") {
		if segment == "" {
			return fmt.Errorf("package %q has empty segment (leading, trailing, or doubled '.')", pkg)
		}
		if !identPattern.MatchString(segment) {
			return fmt.Errorf("package %q segment %q does not match grammar [A-Za-z][A-Za-z0-9_]*", pkg, segment)
		}
	}
	return nil
}

// parseLevel is case-insensitive per ADR-0043 §"Clause 1": DEBUG,
// Debug, and debug all canonicalize to slog.LevelDebug.
func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("level %q is not one of debug|info|warn|error (case-insensitive)", raw)
	}
}

// canonicalSet returns CanonicalInventory as a set for O(1)
// membership testing.
func canonicalSet() map[string]struct{} {
	set := make(map[string]struct{}, len(CanonicalInventory))
	for _, name := range CanonicalInventory {
		set[name] = struct{}{}
	}
	return set
}
