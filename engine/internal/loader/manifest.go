// path: engine/internal/loader/manifest.go

package loader

import "time"

// Manifest is the parsed ruleset manifest body. The field set mirrors
// ADR-0005 §5 (Manifest body — committed field set). New fields are
// added additively; consumers must tolerate unknown fields per JSON
// Schema additive-evolution policy.
type Manifest struct {
	// ManifestVersion is the meta-version of the manifest schema
	// itself (currently 1). Distinct from the DSL schema versions
	// listed in SchemaVersionsPresent.
	ManifestVersion int `json:"manifest_version"`

	// RulesetVersion identifies the released ruleset, e.g.
	// "rules-v2.4.7". Used as the canonical ruleset identity in
	// the manifest and as one of the five inputs to the
	// execution_id hash (ADR-0002 CC1/CC2). Must be pipe-free
	// per ADR-0002 input-safety.
	RulesetVersion string `json:"ruleset_version"`

	// SchemaVersionsPresent is the set of DSL schema versions
	// actually declared by rule YAMLs inside this manifest.
	// Single-element in steady state; multi-element during a
	// version migration window (ADR-0001).
	SchemaVersionsPresent []int `json:"schema_versions_present"`

	// EngineCompatibility is a semver range identifying which
	// engine releases accept this manifest (ADR-0001). The loader
	// verifies the running engine version against this range at
	// load time.
	EngineCompatibility string `json:"engine_compatibility"`

	// LinterUsed identifies the linter release that validated this
	// manifest, recorded for audit and reconstruction (ADR-0001).
	// **Audit-only**: the engine does not read or verify this field
	// at load time.
	LinterUsed string `json:"linter_used"`

	// GeneratedAt is the manifest's publish timestamp (RFC 3339 UTC).
	GeneratedAt time.Time `json:"generated_at"`

	// Rules is the list of active rules.
	Rules []ManifestRule `json:"rules"`

	// Hash is the sha256 hex of the manifest body bytes. Populated
	// by Loader.Load / Loader.Refresh on successful load. Not
	// stored in the JSON; used by the caller (refresh-mode
	// short-circuit, in-memory execution-context state per
	// ADR-0007 CC3).
	Hash string `json:"-"`
}

// ManifestRule is one entry in Manifest.Rules. Field naming mirrors
// the rename from foundation-doc-baseline (path → yaml_path,
// checksum → yaml_hash) committed in ADR-0005 §5.
type ManifestRule struct {
	Entity   string `json:"entity"`
	YamlPath string `json:"yaml_path"`
	YamlHash string `json:"yaml_hash"`
}

// Pointer is the parsed manifests/latest.json body. Field set mirrors
// ADR-0005 §6 (Pointer file — committed field set).
type Pointer struct {
	// PointerVersion is the meta-version of the pointer schema
	// itself (currently 1).
	PointerVersion int `json:"pointer_version"`

	// ManifestHash is the content hash of the currently-active
	// manifest, formatted as "sha256:<64-char-hex>" per ADR-0005 §7
	// (hash algorithm). The loader strips the "sha256:" prefix
	// before computing the object key.
	ManifestHash string `json:"manifest_hash"`

	// RulesetVersion is duplicated from the referenced manifest's
	// body for fast identification without fetching the body
	// (ADR-0005 §6).
	RulesetVersion string `json:"ruleset_version"`

	// PublishedAt is the pointer-write timestamp; may differ from
	// the referenced manifest's GeneratedAt on rollback.
	PublishedAt time.Time `json:"published_at"`
}
