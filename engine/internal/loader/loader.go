// path: engine/internal/loader/loader.go

// Package loader reads the active manifest from the object store
// per ADR-0005 and ADR-0007. It exposes a startup-mode Load and a
// refresh-mode Refresh; failure handling (process exit on startup
// failure, refuse-swap on refresh failure) lives in the engine
// binary that consumes this package.
package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// ErrHashMismatch wraps the failure mode where the fetched manifest
// body's sha256 does not equal the pointer's claimed hash. ADR-0005
// content-addressing guarantees this never happens in production; a
// mismatch signals corruption or tampering and is fatal to the
// load.
var ErrHashMismatch = errors.New("manifest body hash does not match pointer hash")

// Config configures a Loader. EngineVersion and SupportedSchemaVersions
// are required; the *Key / *Prefix fields default to ADR-0005's
// committed object-store layout if empty.
type Config struct {
	// EngineVersion is the running engine's semver string (e.g.
	// "0.1.0"). Checked against the manifest's engine_compatibility
	// field per ADR-0001.
	EngineVersion string

	// SupportedSchemaVersions is the engine-supported set of DSL
	// schema versions. Every value in manifest.schema_versions_present
	// must be in this set or the load fails closed.
	SupportedSchemaVersions []int

	// SupportedManifestVersions is the set of manifest meta-versions
	// this engine accepts. Defaults to {1} if empty. Future loaders
	// add v2 additively.
	SupportedManifestVersions []int

	// PointerKey is the object-store key for the pointer file.
	// Defaults to "manifests/latest.json" per ADR-0005 §1.
	PointerKey string

	// BodyPrefix is the object-store prefix for content-addressed
	// manifest bodies. Defaults to "manifests/by-hash/" per
	// ADR-0005 §1.
	BodyPrefix string
}

// Loader reads and verifies manifests from an object Store.
type Loader struct {
	store         Store
	engineVersion *semver.Version
	supportedMV   []int
	supportedSV   []int
	pointerKey    string
	bodyPrefix    string
}

// hashHexRE matches a lowercase 64-char hex string (sha256 output).
var hashHexRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// New constructs a Loader from a Store and Config. Returns an error
// if EngineVersion is not a valid semver string or
// SupportedSchemaVersions is empty.
func New(store Store, cfg Config) (*Loader, error) {
	if store == nil {
		return nil, errors.New("loader: store is required")
	}
	ev, err := semver.NewVersion(cfg.EngineVersion)
	if err != nil {
		return nil, fmt.Errorf("loader: engine_version %q is not a valid semver: %w", cfg.EngineVersion, err)
	}
	if len(cfg.SupportedSchemaVersions) == 0 {
		return nil, errors.New("loader: SupportedSchemaVersions must be non-empty")
	}
	supportedMV := cfg.SupportedManifestVersions
	if len(supportedMV) == 0 {
		supportedMV = []int{1}
	}
	pointerKey := cfg.PointerKey
	if pointerKey == "" {
		pointerKey = "manifests/latest.json"
	}
	bodyPrefix := cfg.BodyPrefix
	if bodyPrefix == "" {
		bodyPrefix = "manifests/by-hash/"
	}
	return &Loader{
		store:         store,
		engineVersion: ev,
		supportedMV:   supportedMV,
		supportedSV:   cfg.SupportedSchemaVersions,
		pointerKey:    pointerKey,
		bodyPrefix:    bodyPrefix,
	}, nil
}

// Load executes the startup-mode load pipeline per ADR-0007 CC1:
//
//  1. Read the pointer file (manifests/latest.json).
//  2. Parse and validate the pointer's structural fields.
//  3. Fetch the content-addressed manifest body at
//     manifests/by-hash/sha256-<hex>.json.
//  4. Verify the body's sha256 equals the pointer's declared hash.
//  5. Parse the manifest body.
//  6. Run the ADR-0001 contract checks (manifest_version,
//     engine_compatibility, schema_versions_present).
//  7. Return the parsed Manifest with Hash populated.
//
// Any error returned by Load is fatal at engine startup; the engine
// binary's main() exits non-zero with a structured log line naming
// the failure type per ADR-0007 CC1.
func (l *Loader) Load(ctx context.Context) (*Manifest, error) {
	pointer, err := l.readPointer(ctx)
	if err != nil {
		return nil, fmt.Errorf("read pointer: %w", err)
	}
	return l.fetchAndVerify(ctx, pointer)
}

// Refresh executes the refresh-mode reload per ADR-0007 CC9:
//
//   - Read the pointer file.
//   - If the pointer's manifest_hash equals the caller's currentHash,
//     return (nil, false, nil) — hash short-circuit, no body fetch
//     occurred; the caller continues with its current manifest.
//   - Otherwise fetch and verify the new manifest body and return
//     (newManifest, true, nil).
//   - On any error, return (nil, false, err); per ADR-0007 CC2 the
//     caller honors refuse-swap by retaining its current manifest.
//
// currentHash is the 64-char lowercase hex (no "sha256:" prefix) of
// the manifest currently held by the caller. The Manifest.Hash field
// populated by Load matches this format.
func (l *Loader) Refresh(ctx context.Context, currentHash string) (*Manifest, bool, error) {
	pointer, err := l.readPointer(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("read pointer: %w", err)
	}
	pointerHex, err := stripSha256Prefix(pointer.ManifestHash)
	if err != nil {
		return nil, false, fmt.Errorf("validate pointer manifest_hash: %w", err)
	}
	if pointerHex == currentHash {
		// Hash short-circuit per ADR-0007 CC9: no body fetch.
		return nil, false, nil
	}
	m, err := l.fetchAndVerify(ctx, pointer)
	if err != nil {
		return nil, false, err
	}
	return m, true, nil
}

func (l *Loader) readPointer(ctx context.Context) (*Pointer, error) {
	raw, err := l.store.ReadObject(ctx, l.pointerKey)
	if err != nil {
		return nil, err
	}
	var p Pointer
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("parse pointer JSON: %w", err)
	}
	if p.PointerVersion != 1 {
		return nil, fmt.Errorf("unsupported pointer_version %d (loader supports 1)", p.PointerVersion)
	}
	if _, err := stripSha256Prefix(p.ManifestHash); err != nil {
		return nil, err
	}
	return &p, nil
}

func (l *Loader) fetchAndVerify(ctx context.Context, pointer *Pointer) (*Manifest, error) {
	hexStr, err := stripSha256Prefix(pointer.ManifestHash)
	if err != nil {
		return nil, err
	}
	key := l.bodyPrefix + "sha256-" + hexStr + ".json"
	raw, err := l.store.ReadObject(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest body %s: %w", key, err)
	}

	sum := sha256.Sum256(raw)
	actual := hex.EncodeToString(sum[:])
	if actual != hexStr {
		return nil, fmt.Errorf("%w: body sha256 %s does not match pointer hash %s",
			ErrHashMismatch, actual, hexStr)
	}

	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse manifest JSON: %w", err)
	}

	if err := runContractChecks(&m, l.engineVersion, l.supportedMV, l.supportedSV); err != nil {
		return nil, err
	}

	m.Hash = actual
	return &m, nil
}

// stripSha256Prefix validates that hashStr has the form
// "sha256:<64-char-hex>" and returns the hex portion. The
// "sha256:" prefix is the ADR-0005 §7 commitment for hash algorithm
// encoding in the pointer's manifest_hash field.
func stripSha256Prefix(hashStr string) (string, error) {
	const prefix = "sha256:"
	if !strings.HasPrefix(hashStr, prefix) {
		return "", fmt.Errorf("manifest_hash %q does not start with %q (ADR-0005 §7)", hashStr, prefix)
	}
	hexStr := strings.TrimPrefix(hashStr, prefix)
	if !hashHexRE.MatchString(hexStr) {
		return "", fmt.Errorf("manifest_hash hex portion %q is not a 64-char lowercase hex string", hexStr)
	}
	return hexStr, nil
}
