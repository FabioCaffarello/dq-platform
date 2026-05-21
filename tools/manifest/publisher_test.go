// path: tools/manifest/publisher_test.go

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- in-memory Store fake ---

// memStore is the in-memory Store used by every unit test. It
// tracks writes, supports CAS via per-key generation counters,
// and lets tests inject conflicts by pre-seeding values.
type memStore struct {
	mu       sync.Mutex
	objects  map[string][]byte
	gens     map[string]int64
	writes   []string // ordered audit trail: keys written, in order
	nextGen  int64
}

func newMemStore() *memStore {
	return &memStore{
		objects: map[string][]byte{},
		gens:    map[string]int64{},
		nextGen: 1,
	}
}

func (m *memStore) WriteIfNotExists(_ context.Context, key string, body []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.objects[key]; ok {
		return fmt.Errorf("%s: %w", key, ErrAlreadyExists)
	}
	m.objects[key] = append([]byte(nil), body...)
	m.gens[key] = m.nextGen
	m.nextGen++
	m.writes = append(m.writes, key)
	return nil
}

func (m *memStore) ReadObject(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	body, ok := m.objects[key]
	if !ok {
		return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
	}
	return append([]byte(nil), body...), nil
}

func (m *memStore) ReadPointerGeneration(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gens[key], nil // 0 if missing — matches GCS semantics
}

func (m *memStore) CASWritePointer(_ context.Context, key string, body []byte, expectedGen int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	currentGen := m.gens[key]
	if expectedGen != currentGen {
		return 0, fmt.Errorf("%s: %w", key, ErrPreconditionFailed)
	}
	m.objects[key] = append([]byte(nil), body...)
	newGen := m.nextGen
	m.gens[key] = newGen
	m.nextGen++
	m.writes = append(m.writes, key)
	return newGen, nil
}

// --- common test setup ---

func testConfig(t *testing.T, store Store) Config {
	t.Helper()
	return Config{
		Store:                   store,
		RulesetVersion:          "rules-v1.0.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
		Now:                     func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
	}
}

// --- New() guard tests ---

func TestNew_RequiresStore(t *testing.T) {
	cfg := testConfig(t, nil)
	cfg.Store = nil
	if _, err := New(cfg); err == nil {
		t.Fatalf("New(nil Store) returned nil; want error")
	}
}

func TestNew_RequiresRulesetVersion(t *testing.T) {
	cfg := testConfig(t, newMemStore())
	cfg.RulesetVersion = ""
	if _, err := New(cfg); err == nil {
		t.Fatalf("New(empty RulesetVersion) returned nil; want error")
	}
}

func TestNew_RejectsPipeInRulesetVersion(t *testing.T) {
	// ADR-0002 input-safety: ruleset_version is one of the five
	// pipe-separated inputs to the execution_id hash.
	cfg := testConfig(t, newMemStore())
	cfg.RulesetVersion = "rules-v|0.1.0"
	if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), "pipe") {
		t.Fatalf("New(pipe in RulesetVersion) = %v; want pipe-character error", err)
	}
}

func TestNew_RequiresSupportedSchemaVersions(t *testing.T) {
	cfg := testConfig(t, newMemStore())
	cfg.SupportedSchemaVersions = nil
	if _, err := New(cfg); err == nil {
		t.Fatalf("New(empty SupportedSchemaVersions) returned nil; want error")
	}
}

// --- Publish() happy path ---

func TestPublish_HappyPath_FirstPublish(t *testing.T) {
	store := newMemStore()
	p, err := New(testConfig(t, store))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := p.Publish(context.Background(), Options{
		RulesDir: "testdata/rules",
	})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.RulesPublished != 2 {
		t.Errorf("RulesPublished = %d; want 2", result.RulesPublished)
	}
	if result.ManifestHash == "" || len(result.ManifestHash) != 64 {
		t.Errorf("ManifestHash = %q; want 64-char hex", result.ManifestHash)
	}
	if result.PointerGen == 0 {
		t.Errorf("PointerGen = 0; want post-write generation")
	}
	// Audit: expect 2 by-hash YAMLs + 1 by-hash manifest + 1
	// pointer write = 4 keys, in that order.
	if len(store.writes) != 4 {
		t.Fatalf("write count = %d; want 4: %v", len(store.writes), store.writes)
	}
	for i, w := range store.writes[:2] {
		if !strings.HasPrefix(w, "yamls/by-hash/sha256-") {
			t.Errorf("write[%d] = %q; want a yamls/by-hash/ key", i, w)
		}
	}
	if !strings.HasPrefix(store.writes[2], "manifests/by-hash/sha256-") {
		t.Errorf("write[2] = %q; want a manifests/by-hash/ key", store.writes[2])
	}
	if store.writes[3] != "manifests/latest.json" {
		t.Errorf("write[3] = %q; want manifests/latest.json", store.writes[3])
	}
}

func TestPublish_StableManifestHashAcrossRuns(t *testing.T) {
	// Re-running Publish on the same rules with the same Now()
	// must produce the same manifest hash (deterministic JSON
	// marshaling). Rule discovery order from filepath.WalkDir
	// is alphabetical on most filesystems, but the publisher
	// also sorts by entity before marshaling — so the test
	// holds regardless.
	cfg := testConfig(t, newMemStore())
	p1, _ := New(cfg)
	r1, err := p1.Publish(context.Background(), Options{RulesDir: "testdata/rules", DryRun: true})
	if err != nil {
		t.Fatalf("Publish #1: %v", err)
	}
	p2, _ := New(testConfig(t, newMemStore()))
	r2, err := p2.Publish(context.Background(), Options{RulesDir: "testdata/rules", DryRun: true})
	if err != nil {
		t.Fatalf("Publish #2: %v", err)
	}
	if r1.ManifestHash != r2.ManifestHash {
		t.Errorf("ManifestHash unstable across runs: %q vs %q", r1.ManifestHash, r2.ManifestHash)
	}
}

func TestPublish_DryRun_NoWrites(t *testing.T) {
	store := newMemStore()
	p, err := New(testConfig(t, store))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result, err := p.Publish(context.Background(), Options{
		RulesDir: "testdata/rules",
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("Publish(dry-run): %v", err)
	}
	if len(store.writes) != 0 {
		t.Errorf("DryRun wrote %d keys; want 0: %v", len(store.writes), store.writes)
	}
	if result.PointerGen != 0 {
		t.Errorf("DryRun PointerGen = %d; want 0", result.PointerGen)
	}
	if result.ManifestHash == "" {
		t.Errorf("DryRun ManifestHash empty; want populated")
	}
}

func TestPublish_IdempotentBodyWrites(t *testing.T) {
	// Pre-seed both by-hash bodies so WriteIfNotExists returns
	// ErrAlreadyExists. The publisher must tolerate this and
	// proceed to the pointer CAS write.
	store := newMemStore()
	p, err := New(testConfig(t, store))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// First call to populate the bucket.
	first, err := p.Publish(context.Background(), Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish #1: %v", err)
	}
	preWriteCount := len(store.writes)

	// Second call — same content, so by-hash writes should be
	// rejected as ErrAlreadyExists and tolerated.
	p2, _ := New(testConfig(t, store))
	// Re-use the same store. The next CAS write needs the new
	// pointer generation.
	second, err := p2.Publish(context.Background(), Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish #2 (idempotent): %v", err)
	}
	if second.ManifestHash != first.ManifestHash {
		t.Errorf("idempotent re-publish changed manifest hash: %q vs %q", second.ManifestHash, first.ManifestHash)
	}
	// Only the pointer write should have happened in run #2 —
	// audit increases by exactly 1.
	if len(store.writes) != preWriteCount+1 {
		t.Errorf("idempotent re-publish wrote %d additional keys; want 1 (pointer only). all writes: %v",
			len(store.writes)-preWriteCount, store.writes)
	}
	if store.writes[len(store.writes)-1] != "manifests/latest.json" {
		t.Errorf("last write = %q; want manifests/latest.json", store.writes[len(store.writes)-1])
	}
}

// --- verification failures ---

func TestPublish_RuleVersionNotSupported_Rejected(t *testing.T) {
	tmp := t.TempDir()
	mustWriteRule(t, filepath.Join(tmp, "bad.yaml"), 99, "bad")
	cfg := testConfig(t, newMemStore())
	p, _ := New(cfg)
	_, err := p.Publish(context.Background(), Options{RulesDir: tmp})
	if !errors.Is(err, ErrVerificationFailed) {
		t.Fatalf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "99") {
		t.Errorf("err %q should mention the offending version 99", err)
	}
}

func TestPublish_MissingSchemaMirror_Rejected(t *testing.T) {
	tmp := t.TempDir()
	mustWriteRule(t, filepath.Join(tmp, "ok.yaml"), 2, "entity")
	cfg := testConfig(t, newMemStore())
	cfg.SupportedSchemaVersions = []int{1, 2} // support v2, but mirror dir has only v1
	p, _ := New(cfg)
	_, err := p.Publish(context.Background(), Options{RulesDir: tmp})
	if !errors.Is(err, ErrVerificationFailed) {
		t.Fatalf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "v2.schema.json") {
		t.Errorf("err %q should mention the missing v2 mirror", err)
	}
}

func TestPublish_DuplicateEntities_Rejected(t *testing.T) {
	tmp := t.TempDir()
	mustWriteRule(t, filepath.Join(tmp, "a.yaml"), 1, "customer")
	mustWriteRule(t, filepath.Join(tmp, "b.yaml"), 1, "customer")
	cfg := testConfig(t, newMemStore())
	p, _ := New(cfg)
	_, err := p.Publish(context.Background(), Options{RulesDir: tmp})
	if !errors.Is(err, ErrVerificationFailed) {
		t.Fatalf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "duplicate entity") {
		t.Errorf("err %q should mention duplicate entity", err)
	}
}

func TestPublish_ZeroRules_Rejected(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig(t, newMemStore())
	p, _ := New(cfg)
	_, err := p.Publish(context.Background(), Options{RulesDir: tmp})
	if !errors.Is(err, ErrVerificationFailed) {
		t.Fatalf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "no rule YAMLs") {
		t.Errorf("err %q should mention no rule YAMLs", err)
	}
}

func TestPublish_MissingRulesDir_OperationalError(t *testing.T) {
	// A missing rules dir is an operational error (CLI exits
	// with code 2), not a verification failure (code 1). The
	// distinction matters because a typo in --rules looks like
	// "no rule YAMLs" otherwise, which is the wrong operator
	// signal.
	cfg := testConfig(t, newMemStore())
	p, _ := New(cfg)
	_, err := p.Publish(context.Background(), Options{RulesDir: "/no/such/dir"})
	if err == nil {
		t.Fatalf("Publish(missing dir) returned nil; want operational error")
	}
	if errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want operational error, not ErrVerificationFailed", err)
	}
}

func TestNew_MissingSchemaMirrorDir_FailsFast(t *testing.T) {
	// Stat the mirror dir at construction so a typo produces
	// one error instead of N per-file errors at verify time.
	cfg := testConfig(t, newMemStore())
	cfg.SchemaMirrorDir = "/no/such/dir"
	if _, err := New(cfg); err == nil {
		t.Fatalf("New(missing SchemaMirrorDir) returned nil; want error")
	}
}

// --- manifest / pointer body shape ---

func TestPublish_ManifestBodyShape(t *testing.T) {
	store := newMemStore()
	p, _ := New(testConfig(t, store))
	result, err := p.Publish(context.Background(), Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	manifestKey := manifestByHashPath(result.ManifestHash)
	body, err := store.ReadObject(context.Background(), manifestKey)
	if err != nil {
		t.Fatalf("ReadObject(%s): %v", manifestKey, err)
	}
	var got Manifest
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("Unmarshal manifest body: %v", err)
	}
	if got.ManifestVersion != 1 {
		t.Errorf("ManifestVersion = %d; want 1", got.ManifestVersion)
	}
	if got.RulesetVersion != "rules-v1.0.0" {
		t.Errorf("RulesetVersion = %q; want rules-v1.0.0", got.RulesetVersion)
	}
	if got.EngineCompatibility != ">=0.1.0, <1.0.0" {
		t.Errorf("EngineCompatibility = %q; want '>=0.1.0, <1.0.0'", got.EngineCompatibility)
	}
	if got.LinterUsed != "tools-lint-v0.1.0" {
		t.Errorf("LinterUsed = %q; want tools-lint-v0.1.0", got.LinterUsed)
	}
	if len(got.SchemaVersionsPresent) != 1 || got.SchemaVersionsPresent[0] != 1 {
		t.Errorf("SchemaVersionsPresent = %v; want [1]", got.SchemaVersionsPresent)
	}
	if len(got.Rules) != 2 {
		t.Errorf("Rules count = %d; want 2", len(got.Rules))
	}
	// Rules are sorted by entity: account < customer.
	if got.Rules[0].Entity != "account" || got.Rules[1].Entity != "customer" {
		t.Errorf("Rules order = [%s, %s]; want [account, customer]",
			got.Rules[0].Entity, got.Rules[1].Entity)
	}
	for _, r := range got.Rules {
		if !strings.HasPrefix(r.YAMLPath, "yamls/by-hash/sha256-") {
			t.Errorf("Rule %q YAMLPath = %q; want yamls/by-hash/ prefix", r.Entity, r.YAMLPath)
		}
		if len(r.YAMLHash) != 64 {
			t.Errorf("Rule %q YAMLHash = %q; want 64-char hex", r.Entity, r.YAMLHash)
		}
	}
}

func TestPublish_PointerShape(t *testing.T) {
	store := newMemStore()
	p, _ := New(testConfig(t, store))
	result, err := p.Publish(context.Background(), Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	body, err := store.ReadObject(context.Background(), pointerPath)
	if err != nil {
		t.Fatalf("ReadObject(pointer): %v", err)
	}
	var ptr Pointer
	if err := json.Unmarshal(body, &ptr); err != nil {
		t.Fatalf("Unmarshal pointer: %v", err)
	}
	if ptr.PointerVersion != 1 {
		t.Errorf("PointerVersion = %d; want 1", ptr.PointerVersion)
	}
	wantPrefix := "sha256:"
	if !strings.HasPrefix(ptr.ManifestHash, wantPrefix) {
		t.Errorf("ManifestHash = %q; want %q prefix", ptr.ManifestHash, wantPrefix)
	}
	if strings.TrimPrefix(ptr.ManifestHash, wantPrefix) != result.ManifestHash {
		t.Errorf("ManifestHash hex %q does not match result %q", ptr.ManifestHash, result.ManifestHash)
	}
	if ptr.RulesetVersion != "rules-v1.0.0" {
		t.Errorf("RulesetVersion = %q; want rules-v1.0.0", ptr.RulesetVersion)
	}
}

// --- CAS-loss ---

func TestPublish_PointerCASLoss(t *testing.T) {
	// Pre-seed the pointer with a different body. The publisher
	// reads gen=G; a concurrent writer (simulated by mutating
	// the store between read and CAS) makes the CAS fail.
	store := newMemStore()
	// Pre-populate the pointer to generation 1.
	if err := store.WriteIfNotExists(context.Background(), pointerPath, []byte(`{"pointer_version":1}`)); err != nil {
		t.Fatalf("seed pointer: %v", err)
	}
	// Wrap the store so that ReadPointerGeneration returns
	// stale gen=1 but CASWritePointer sees the real gen=2.
	racing := &racingStore{Store: store, staleGen: 1, realStore: store}
	// Advance the real store's pointer generation by writing
	// once with the right expectation.
	if _, err := store.CASWritePointer(context.Background(), pointerPath, []byte(`{"pointer_version":1,"manifest_hash":"sha256:00"}`), 1); err != nil {
		t.Fatalf("advance pointer: %v", err)
	}

	p, _ := New(testConfig(t, racing))
	_, err := p.Publish(context.Background(), Options{RulesDir: "testdata/rules"})
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Fatalf("err = %v; want ErrPreconditionFailed", err)
	}
}

// racingStore returns a stale generation from
// ReadPointerGeneration while forwarding all other calls to the
// embedded Store. This simulates the CAS race-loser scenario.
type racingStore struct {
	Store
	realStore *memStore
	staleGen  int64
}

func (r *racingStore) ReadPointerGeneration(_ context.Context, _ string) (int64, error) {
	return r.staleGen, nil
}

// --- helpers ---

func mustWriteRule(t *testing.T, path string, version int, entity string) {
	t.Helper()
	body := fmt.Sprintf(`version: %d
entity: %s
checks:
  - check_id: row_count_positive
    kind: placeholder
`, version, entity)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mustWriteRule mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("mustWriteRule(%s): %v", path, err)
	}
}
