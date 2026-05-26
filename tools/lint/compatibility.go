// path: tools/lint/compatibility.go

package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SchemaStatus is the closed enum of schema-version states from
// ADR-0035 §"Compatibility-state table".
type SchemaStatus string

const (
	// SchemaStatusCurrent is the actively-recommended version.
	// New rules should declare this version; existing rules at
	// this version need no migration.
	SchemaStatusCurrent SchemaStatus = "current"

	// SchemaStatusDeprecated is engine-supported but warned-on
	// at lint time. Operators should migrate to the current
	// version before the earliest-drop date.
	SchemaStatusDeprecated SchemaStatus = "deprecated"
)

// SchemaCompatibilityEntry mirrors one row of ADR-0035's
// compatibility-state table. The lint binary reads this map to
// emit a deprecation warning when a rule declares a deprecated
// schema version; the engine binary's `SupportedSchemaVersions`
// is the parallel runtime authority (see
// engine/cmd/dq-engine/main.go).
type SchemaCompatibilityEntry struct {
	// Status is `current` or `deprecated` per ADR-0035 §"Compatibility-state table".
	Status SchemaStatus

	// EngineSupportSince records the engine version that first
	// accepted this schema (informational; surfaced in the
	// deprecation warning so operators can audit the support
	// window).
	EngineSupportSince string

	// EarliestDrop is the earliest date the engine release
	// dropping this version may land. Computed from the 90-day
	// floor anchored at v(N+1)'s first manifest publish per
	// ADR-0035 §"Compatibility-state table". For currently-
	// current versions the value is "TBD (when v(N+1) ships)".
	EarliestDrop string
}

// SchemaCompatibility is the canonical compatibility-state map.
// MUST be kept byte-equivalent to the ADR-0035 §"Compatibility-
// state table" markdown table — when the ADR amendment lands
// (e.g., v3 ships → v2 becomes deprecated), this map updates in
// the same PR. There is no CI byte-equality gate for this map
// because the ADR table is human-prose; updating both in one PR
// is the operating convention.
var SchemaCompatibility = map[int]SchemaCompatibilityEntry{
	1: {
		Status:             SchemaStatusDeprecated,
		EngineSupportSince: "0.1.0",
		EarliestDrop:       "2026-08-23",
	},
	2: {
		Status:             SchemaStatusCurrent,
		EngineSupportSince: "0.1.0",
		EarliestDrop:       "TBD (when v3 ships)",
	},
}

// SchemaVersionStatus returns the compatibility entry for the
// given schema version. The second return value is false for
// versions absent from the table — the caller treats those as
// "unknown" (not deprecated; not current). The lint binary's
// schema validator rejects truly-unsupported versions earlier;
// this helper is consulted only after schema validation passes.
func SchemaVersionStatus(version int) (SchemaCompatibilityEntry, bool) {
	entry, ok := SchemaCompatibility[version]
	return entry, ok
}

// DeprecationWarning is one warning surfaced by
// CheckDeprecatedSchemaVersions. Distinct from `ValidationError`
// so callers can route warnings (informational) separately from
// errors (exit-non-zero).
type DeprecationWarning struct {
	Path    string
	Version int
	Message string
}

// CheckDeprecatedSchemaVersions walks the rules directory tree
// and returns one DeprecationWarning per rule whose version is
// `deprecated` in the compatibility-state table per ADR-0035.
//
// The walker mirrors ValidateRulesDir's directory-tree posture:
// skips any `_schema/` subdirectory, skips files whose basename
// starts with `_` (e.g., `_owners.yaml`), and considers only
// *.yaml / *.yml files. Schema validation is NOT performed —
// callers run ValidateRulesDir for that. Files whose YAML cannot
// be parsed silently produce no warning here; ValidateRulesDir
// surfaces the parse error.
//
// Returns:
//
//   - warnings: slice of DeprecationWarning, one per deprecated
//     rule. Empty (non-nil) slice when nothing is deprecated.
//   - err: operational error (I/O, etc.). A missing rules
//     directory is not an error — the walker returns an empty
//     slice so `make lint-rules` continues to succeed.
//
// Per ADR-0035 §"Migration support level": "Deprecation warning
// at lint time — a future `tools/lint` enhancement emits a
// warning when a rule's `version` field declares a deprecated
// schema. The warning fires on deprecated-but-supported
// versions; it becomes a hard error at the version's drop
// release." Drop releases land in B2-20; this helper covers
// the deprecated-but-supported window between B2-19 (customer
// migration) and B2-20 (v1-drop release, earliest 2026-08-23).
func CheckDeprecatedSchemaVersions(dir string) ([]DeprecationWarning, error) {
	warnings := []DeprecationWarning{}

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkCallbackErr error) error {
		if walkCallbackErr != nil {
			return walkCallbackErr
		}
		if d.IsDir() {
			if filepath.Base(path) == "_schema" {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			return nil
		}
		if strings.HasPrefix(name, "_") {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			// I/O on a rule file ValidateRulesDir is also walking
			// will produce its own error there; we silently skip
			// to avoid double-reporting.
			return nil
		}
		version := parseRuleVersion(raw)
		if version == 0 {
			return nil
		}
		entry, ok := SchemaVersionStatus(version)
		if !ok || entry.Status != SchemaStatusDeprecated {
			return nil
		}
		warnings = append(warnings, DeprecationWarning{
			Path:    path,
			Version: version,
			Message: fmt.Sprintf(
				"rule declares schema version %d which is deprecated per ADR-0035 (engine support since %s; earliest drop %s); migrate to a `current` version before the drop date",
				version, entry.EngineSupportSince, entry.EarliestDrop),
		})
		return nil
	})

	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return warnings, nil
		}
		return nil, fmt.Errorf("walk %s: %w", dir, walkErr)
	}
	return warnings, nil
}
