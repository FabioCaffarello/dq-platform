// path: tools/manifest/manifest.go

package main

import "time"

// Manifest is the parsed ruleset manifest body. The committed
// field set mirrors ADR-0005 §5 byte-for-byte. The field shape
// is the contract; the Go type is duplicated between this
// publisher and the engine loader (engine/internal/loader/
// manifest.go) intentionally — the publisher is a separate Go
// module per the workspace topology (B1-10 / ADR-0009), so
// importing engine internals from a tool would invert the
// dependency direction. Both halves evolve under the JSON
// shape commitment in ADR-0005 §5.
type Manifest struct {
	// ManifestVersion is the meta-version of the manifest schema
	// itself (currently 1). Distinct from the DSL schema versions
	// listed in SchemaVersionsPresent.
	ManifestVersion int `json:"manifest_version"`

	// RulesetVersion identifies the released ruleset, e.g.
	// "rules-v2.4.7". Must be pipe-free per ADR-0002 input-safety
	// (ruleset_version is one of the five inputs to the
	// execution_id hash).
	RulesetVersion string `json:"ruleset_version"`

	// SchemaVersionsPresent is the set of DSL schema versions
	// actually declared by rule YAMLs inside this manifest.
	// Single-element in steady state; multi-element during a
	// version migration window per ADR-0001.
	SchemaVersionsPresent []int `json:"schema_versions_present"`

	// EngineCompatibility is a semver range identifying which
	// engine releases accept this manifest (ADR-0001). The
	// publisher stores the string verbatim; the loader is the
	// one that evaluates the range at engine startup.
	EngineCompatibility string `json:"engine_compatibility"`

	// LinterUsed identifies the linter release that validated
	// this manifest (ADR-0001). Audit-only — the engine does not
	// read or verify this field at load time.
	LinterUsed string `json:"linter_used"`

	// GeneratedAt is the manifest's publish timestamp (RFC 3339
	// UTC).
	GeneratedAt time.Time `json:"generated_at"`

	// Rules is the list of active rules. The publisher sorts
	// this slice by Entity for deterministic JSON marshaling of
	// the rules array. ADR-0005 does not specify ordering;
	// sort-by-entity is the chosen convention here so the rules
	// array bytes are stable across re-runs on identical input.
	// Note: the full manifest hash is NOT stable across
	// re-publishes because GeneratedAt is part of the body —
	// idempotency means "by-hash objects are not re-written"
	// (ADR-0005 §2 immutability), not "same hash". **New
	// contribution proposed here, requires review.**
	Rules []ManifestRule `json:"rules"`
}

// ManifestRule is one entry in Manifest.Rules. Field naming
// matches ADR-0005 §5 (yaml_path, yaml_hash). YAMLHash is the
// lowercase sha256 hex of the rule YAML bytes — no algorithm
// prefix here; the algorithm is encoded in the by-hash storage
// path per ADR-0005 §1.
type ManifestRule struct {
	Entity   string `json:"entity"`
	YAMLPath string `json:"yaml_path"`
	YAMLHash string `json:"yaml_hash"`
}

// Pointer is the parsed manifests/latest.json body. Committed
// field set mirrors ADR-0005 §6.
type Pointer struct {
	// PointerVersion is the meta-version of the pointer schema
	// itself (currently 1).
	PointerVersion int `json:"pointer_version"`

	// ManifestHash references the content-addressed manifest
	// object, formatted as "sha256:<64-char-hex>" per ADR-0005
	// §7 (hash algorithm).
	ManifestHash string `json:"manifest_hash"`

	// RulesetVersion is duplicated from the referenced
	// manifest's body so the pointer alone identifies the live
	// ruleset without fetching the manifest (ADR-0005 §6).
	RulesetVersion string `json:"ruleset_version"`

	// PublishedAt is the pointer-write timestamp; differs from
	// the referenced manifest's GeneratedAt on rollback.
	PublishedAt time.Time `json:"published_at"`
}

// Schema-meta version constants. The publisher writes these
// values; the loader reads them. A breaking change to either
// schema requires an ADR-0005 update + a coordinated bump.
const (
	manifestSchemaVersion = 1
	pointerSchemaVersion  = 1
)
