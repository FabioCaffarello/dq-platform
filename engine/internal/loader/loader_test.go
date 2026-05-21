// path: engine/internal/loader/loader_test.go

package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// memStore is an in-memory Store implementation for unit tests. It
// records the keys passed to ReadObject so tests can assert call
// patterns (e.g., hash-short-circuit must perform 1 read, not 2).
type memStore struct {
	objects map[string][]byte
	reads   []string
}

func newMemStore() *memStore {
	return &memStore{objects: map[string][]byte{}}
}

func (s *memStore) ReadObject(ctx context.Context, key string) ([]byte, error) {
	s.reads = append(s.reads, key)
	raw, ok := s.objects[key]
	if !ok {
		return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
	}
	return raw, nil
}

// makeManifestBody returns canonical manifest body bytes plus its
// sha256 hex. The body matches a valid ADR-0005 §5 shape.
func makeManifestBody(t *testing.T) (body []byte, hashHex string) {
	t.Helper()
	body = []byte(`{
		"manifest_version": 1,
		"ruleset_version": "rules-v1.0.0",
		"schema_versions_present": [1],
		"engine_compatibility": ">=0.1.0 <2.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	sum := sha256.Sum256(body)
	return body, hex.EncodeToString(sum[:])
}

// makePointer returns a pointer JSON referencing the given hash.
func makePointer(t *testing.T, hashHex string) []byte {
	t.Helper()
	return []byte(fmt.Sprintf(`{
		"pointer_version": 1,
		"manifest_hash": "sha256:%s",
		"ruleset_version": "rules-v1.0.0",
		"published_at": "2026-05-21T00:00:00Z"
	}`, hashHex))
}

func newTestLoader(t *testing.T, store Store) *Loader {
	t.Helper()
	l, err := New(store, Config{
		EngineVersion:           "0.1.0",
		SupportedSchemaVersions: []int{1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return l
}

func TestNew_RequiresStore(t *testing.T) {
	if _, err := New(nil, Config{EngineVersion: "0.1.0", SupportedSchemaVersions: []int{1}}); err == nil {
		t.Fatalf("New(nil, ...) returned nil; want error")
	}
}

func TestNew_RequiresEngineVersion(t *testing.T) {
	if _, err := New(newMemStore(), Config{EngineVersion: "not-semver", SupportedSchemaVersions: []int{1}}); err == nil {
		t.Fatalf("New with bad version returned nil; want error")
	}
}

func TestNew_RequiresSchemaVersions(t *testing.T) {
	if _, err := New(newMemStore(), Config{EngineVersion: "0.1.0"}); err == nil {
		t.Fatalf("New with empty schema versions returned nil; want error")
	}
}

func TestLoad_HappyPath(t *testing.T) {
	store := newMemStore()
	body, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = body

	loader := newTestLoader(t, store)
	m, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.ManifestVersion != 1 {
		t.Errorf("ManifestVersion = %d; want 1", m.ManifestVersion)
	}
	if m.RulesetVersion != "rules-v1.0.0" {
		t.Errorf("RulesetVersion = %q; want rules-v1.0.0", m.RulesetVersion)
	}
	if m.Hash != hashHex {
		t.Errorf("Hash = %q; want %q", m.Hash, hashHex)
	}
}

func TestLoad_PointerNotFound(t *testing.T) {
	store := newMemStore()
	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Load with no pointer: err = %v; want ErrObjectNotFound", err)
	}
}

func TestLoad_ManifestBodyNotFound(t *testing.T) {
	store := newMemStore()
	_, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	// Body not in store.

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Load with missing body: err = %v; want ErrObjectNotFound", err)
	}
}

func TestLoad_HashMismatch(t *testing.T) {
	store := newMemStore()
	body, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	// Body stored at the right key but with content that hashes
	// differently.
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = append(body, ' ')

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("Load with hash mismatch: err = %v; want ErrHashMismatch", err)
	}
}

func TestLoad_PointerVersionUnsupported(t *testing.T) {
	store := newMemStore()
	store.objects["manifests/latest.json"] = []byte(`{
		"pointer_version": 2,
		"manifest_hash": "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		"ruleset_version": "rules-v1.0.0",
		"published_at": "2026-05-21T00:00:00Z"
	}`)

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "pointer_version") {
		t.Fatalf("Load with bad pointer_version: err = %v; want pointer_version error", err)
	}
}

func TestLoad_ManifestHashFormatInvalid(t *testing.T) {
	store := newMemStore()
	store.objects["manifests/latest.json"] = []byte(`{
		"pointer_version": 1,
		"manifest_hash": "not-sha256-prefixed",
		"ruleset_version": "rules-v1.0.0",
		"published_at": "2026-05-21T00:00:00Z"
	}`)

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if err == nil || !strings.Contains(err.Error(), "sha256:") {
		t.Fatalf("Load with bad manifest_hash format: err = %v; want sha256: prefix error", err)
	}
}

func TestLoad_EngineCompatibilityFails(t *testing.T) {
	store := newMemStore()
	// Manifest demands engine version >=2.0.0; loader is at 0.1.0.
	body := []byte(`{
		"manifest_version": 1,
		"ruleset_version": "rules-v1.0.0",
		"schema_versions_present": [1],
		"engine_compatibility": ">=2.0.0 <3.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	sum := sha256.Sum256(body)
	hashHex := hex.EncodeToString(sum[:])
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = body

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("Load with unsatisfied engine_compatibility: err = %v; want ErrContractMismatch", err)
	}
	if !strings.Contains(err.Error(), "engine_compatibility") {
		t.Errorf("error message should mention engine_compatibility: %v", err)
	}
}

func TestLoad_SchemaVersionUnsupported(t *testing.T) {
	store := newMemStore()
	body := []byte(`{
		"manifest_version": 1,
		"ruleset_version": "rules-v1.0.0",
		"schema_versions_present": [2],
		"engine_compatibility": ">=0.1.0 <2.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	sum := sha256.Sum256(body)
	hashHex := hex.EncodeToString(sum[:])
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = body

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("Load with unsupported schema_version: err = %v; want ErrContractMismatch", err)
	}
	if !strings.Contains(err.Error(), "schema_versions_present") {
		t.Errorf("error message should mention schema_versions_present: %v", err)
	}
}

func TestLoad_ManifestVersionUnsupported(t *testing.T) {
	store := newMemStore()
	body := []byte(`{
		"manifest_version": 2,
		"ruleset_version": "rules-v1.0.0",
		"schema_versions_present": [1],
		"engine_compatibility": ">=0.1.0 <2.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	sum := sha256.Sum256(body)
	hashHex := hex.EncodeToString(sum[:])
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = body

	loader := newTestLoader(t, store)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrContractMismatch) {
		t.Fatalf("Load with manifest_version=2: err = %v; want ErrContractMismatch", err)
	}
}

func TestRefresh_HashShortCircuit(t *testing.T) {
	store := newMemStore()
	body, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = body

	loader := newTestLoader(t, store)

	// First Load primes the loader's view.
	if _, err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := len(store.reads)

	// Refresh with the same hash → short-circuit: only the pointer
	// is re-read; no body fetch.
	m, swapped, err := loader.Refresh(context.Background(), hashHex)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if swapped {
		t.Errorf("Refresh swapped = true; want false (short-circuit)")
	}
	if m != nil {
		t.Errorf("Refresh manifest non-nil; want nil (short-circuit)")
	}
	after := store.reads[before:]
	if len(after) != 1 {
		t.Errorf("Refresh did %d reads; want 1 (pointer only); reads after first load: %v",
			len(after), after)
	}
	if len(after) == 1 && after[0] != "manifests/latest.json" {
		t.Errorf("Refresh read %q; want manifests/latest.json (short-circuit must not fetch body)", after[0])
	}
}

func TestRefresh_NewManifest(t *testing.T) {
	store := newMemStore()
	bodyA, hashA := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashA)
	store.objects["manifests/by-hash/sha256-"+hashA+".json"] = bodyA

	loader := newTestLoader(t, store)
	if _, err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Publish a new manifest with a different hash and flip the
	// pointer.
	bodyB := []byte(`{
		"manifest_version": 1,
		"ruleset_version": "rules-v1.0.1",
		"schema_versions_present": [1],
		"engine_compatibility": ">=0.1.0 <2.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	sumB := sha256.Sum256(bodyB)
	hashB := hex.EncodeToString(sumB[:])
	store.objects["manifests/by-hash/sha256-"+hashB+".json"] = bodyB
	store.objects["manifests/latest.json"] = makePointer(t, hashB)

	m, swapped, err := loader.Refresh(context.Background(), hashA)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if !swapped {
		t.Errorf("Refresh swapped = false; want true")
	}
	if m == nil {
		t.Fatalf("Refresh manifest = nil; want new manifest")
	}
	if m.Hash != hashB {
		t.Errorf("Refresh manifest.Hash = %q; want %q", m.Hash, hashB)
	}
	if m.RulesetVersion != "rules-v1.0.1" {
		t.Errorf("Refresh manifest.RulesetVersion = %q; want rules-v1.0.1", m.RulesetVersion)
	}
}

func TestRefresh_PointerReadFails(t *testing.T) {
	store := newMemStore() // empty: no pointer
	loader := newTestLoader(t, store)
	_, swapped, err := loader.Refresh(context.Background(), "any")
	if err == nil {
		t.Fatalf("Refresh with empty store: err = nil; want operational error")
	}
	if swapped {
		t.Errorf("Refresh swapped = true on error; want false")
	}
}
