// path: tools/manifest/rollback_test.go

package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// rollbackTestStore seeds a memStore with two manifest bodies +
// a pointer pointing at the second. Returns the store + the
// hex hash of the first manifest (the rollback target).
func rollbackTestStore(t *testing.T) (*memStore, string, string) {
	t.Helper()
	ctx := context.Background()
	store := newMemStore()

	// Seed manifest A (rules-v1.0.0). The publisher's
	// canonical body shape per ADR-0005 §5.
	manifestA := Manifest{
		ManifestVersion:       1,
		RulesetVersion:        "rules-v1.0.0",
		SchemaVersionsPresent: []int{1},
		EngineCompatibility:   ">=0.1.0, <1.0.0",
		LinterUsed:            "tools-lint-v0.1.0",
		GeneratedAt:           time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC),
		Rules:                 []ManifestRule{{Entity: "customer", YAMLPath: "yamls/by-hash/sha256-AAA.yaml", YAMLHash: "aaa"}},
	}
	bytesA, _ := json.Marshal(manifestA)
	hashA := sha256Hex(bytesA)
	if err := store.WriteIfNotExists(ctx, manifestByHashPath(hashA), bytesA); err != nil {
		t.Fatalf("seed manifest A: %v", err)
	}

	// Seed manifest B (rules-v1.0.1) — the "current" manifest.
	manifestB := Manifest{
		ManifestVersion:       1,
		RulesetVersion:        "rules-v1.0.1",
		SchemaVersionsPresent: []int{1},
		EngineCompatibility:   ">=0.1.0, <1.0.0",
		LinterUsed:            "tools-lint-v0.1.0",
		GeneratedAt:           time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
		Rules:                 []ManifestRule{{Entity: "customer", YAMLPath: "yamls/by-hash/sha256-BBB.yaml", YAMLHash: "bbb"}},
	}
	bytesB, _ := json.Marshal(manifestB)
	hashB := sha256Hex(bytesB)
	if err := store.WriteIfNotExists(ctx, manifestByHashPath(hashB), bytesB); err != nil {
		t.Fatalf("seed manifest B: %v", err)
	}

	// Seed the pointer at B (the "current" pointer).
	currentPointer := Pointer{
		PointerVersion: 1,
		ManifestHash:   "sha256:" + hashB,
		RulesetVersion: "rules-v1.0.1",
		PublishedAt:    time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC),
	}
	pointerBytes, _ := json.Marshal(currentPointer)
	if _, err := store.CASWritePointer(ctx, pointerPath, pointerBytes, 0); err != nil {
		t.Fatalf("seed pointer: %v", err)
	}

	return store, hashA, hashB
}

func newRollback(t *testing.T, store Store) *Rollback {
	t.Helper()
	r, err := NewRollback(RollbackConfig{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewRollback: %v", err)
	}
	return r
}

func TestNewRollback_RequiresStore(t *testing.T) {
	if _, err := NewRollback(RollbackConfig{}); err == nil {
		t.Fatal("NewRollback(nil Store) returned nil; want error")
	}
}

func TestRollback_HappyPath(t *testing.T) {
	store, hashA, hashB := rollbackTestStore(t)
	r := newRollback(t, store)

	res, err := r.Execute(context.Background(), RollbackOptions{TargetHashHex: hashA})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.TargetHash != hashA {
		t.Errorf("TargetHash = %q; want %q", res.TargetHash, hashA)
	}
	if res.TargetRulesetVer != "rules-v1.0.0" {
		t.Errorf("TargetRulesetVer = %q; want rules-v1.0.0", res.TargetRulesetVer)
	}
	if res.PriorHash != "sha256:"+hashB {
		t.Errorf("PriorHash = %q; want sha256:%s", res.PriorHash, hashB)
	}
	if res.PriorRulesetVer != "rules-v1.0.1" {
		t.Errorf("PriorRulesetVer = %q; want rules-v1.0.1", res.PriorRulesetVer)
	}
	if res.PostPointerGen == 0 {
		t.Error("PostPointerGen should be non-zero after a real write")
	}

	// Verify the pointer now points at hashA + carries A's ruleset_version.
	pointerBody, err := store.ReadObject(context.Background(), pointerPath)
	if err != nil {
		t.Fatalf("read pointer: %v", err)
	}
	var p Pointer
	if err := json.Unmarshal(pointerBody, &p); err != nil {
		t.Fatalf("parse pointer: %v", err)
	}
	if p.ManifestHash != "sha256:"+hashA {
		t.Errorf("pointer.ManifestHash = %q; want sha256:%s", p.ManifestHash, hashA)
	}
	if p.RulesetVersion != "rules-v1.0.0" {
		t.Errorf("pointer.RulesetVersion = %q; want rules-v1.0.0", p.RulesetVersion)
	}
}

func TestRollback_TargetBodyMissing_VerificationFails(t *testing.T) {
	store, _, _ := rollbackTestStore(t)
	r := newRollback(t, store)

	// 64-char hex, but no body at this key.
	nonExistentHash := "1111111111111111111111111111111111111111111111111111111111111111"
	_, err := r.Execute(context.Background(), RollbackOptions{TargetHashHex: nonExistentHash})
	if err == nil {
		t.Fatal("expected error when target body is missing")
	}
	if !errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want ErrVerificationFailed", err)
	}
	if !strings.Contains(err.Error(), "dangling pointer") {
		t.Errorf("err %v should mention dangling-pointer safety property", err)
	}
}

func TestRollback_InvalidHashShape_VerificationFails(t *testing.T) {
	store, _, _ := rollbackTestStore(t)
	r := newRollback(t, store)

	for _, badHash := range []string{
		"",
		"too-short",
		"sha256:abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd", // has the prefix the CLI shouldn't accept
		"AAAA1111111111111111111111111111111111111111111111111111111111111111",    // uppercase
		"zzz1111111111111111111111111111111111111111111111111111111111111111",     // non-hex
	} {
		_, err := r.Execute(context.Background(), RollbackOptions{TargetHashHex: badHash})
		if err == nil {
			t.Errorf("Execute(%q) returned nil; want ErrVerificationFailed", badHash)
			continue
		}
		if !errors.Is(err, ErrVerificationFailed) {
			t.Errorf("Execute(%q) err = %v; want ErrVerificationFailed", badHash, err)
		}
	}
}

func TestRollback_MalformedTargetBody_VerificationFails(t *testing.T) {
	store, _, _ := rollbackTestStore(t)
	// Inject a malformed body under a fresh hex hash.
	badHash := "2222222222222222222222222222222222222222222222222222222222222222"
	store.objects[manifestByHashPath(badHash)] = []byte("not valid json")

	r := newRollback(t, store)
	_, err := r.Execute(context.Background(), RollbackOptions{TargetHashHex: badHash})
	if err == nil {
		t.Fatal("expected error when target body is malformed")
	}
	if !errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want ErrVerificationFailed", err)
	}
}

func TestRollback_TargetBodyEmptyRulesetVersion_VerificationFails(t *testing.T) {
	store, _, _ := rollbackTestStore(t)

	// Inject a body with empty ruleset_version under a fresh hash.
	emptyVerManifest := Manifest{ManifestVersion: 1, RulesetVersion: ""}
	body, _ := json.Marshal(emptyVerManifest)
	badHash := sha256Hex(body)
	store.objects[manifestByHashPath(badHash)] = body

	r := newRollback(t, store)
	_, err := r.Execute(context.Background(), RollbackOptions{TargetHashHex: badHash})
	if err == nil {
		t.Fatal("expected error when ruleset_version is empty")
	}
	if !errors.Is(err, ErrVerificationFailed) {
		t.Errorf("err = %v; want ErrVerificationFailed", err)
	}
}

func TestRollback_DryRun_EmitsButDoesNotWrite(t *testing.T) {
	store, hashA, hashB := rollbackTestStore(t)

	// Capture the current pointer generation before dry-run.
	preGen, _ := store.ReadPointerGeneration(context.Background(), pointerPath)

	r := newRollback(t, store)
	res, err := r.Execute(context.Background(), RollbackOptions{
		TargetHashHex: hashA,
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("Execute(DryRun): %v", err)
	}
	if res.PostPointerGen != 0 {
		t.Errorf("DryRun PostPointerGen = %d; want 0", res.PostPointerGen)
	}
	// Pointer body should still reference hashB (no CAS write happened).
	pointerBody, _ := store.ReadObject(context.Background(), pointerPath)
	var p Pointer
	_ = json.Unmarshal(pointerBody, &p)
	if p.ManifestHash != "sha256:"+hashB {
		t.Errorf("DryRun changed the pointer; got %q want sha256:%s", p.ManifestHash, hashB)
	}
	// Generation should be unchanged.
	postGen, _ := store.ReadPointerGeneration(context.Background(), pointerPath)
	if postGen != preGen {
		t.Errorf("DryRun changed the pointer generation; got %d want %d", postGen, preGen)
	}
}

func TestRollback_CASLost_PreconditionFailed(t *testing.T) {
	store, hashA, _ := rollbackTestStore(t)

	// Race: between the Rollback reading the prior pointer
	// generation and writing the new pointer, a concurrent
	// publisher updates the pointer. The Rollback's CAS write
	// loses with ErrPreconditionFailed.
	//
	// Simulate by wrapping the store with a racingStore that
	// bumps the pointer generation before the CAS write.
	racing := &rollbackRacingStore{inner: store, mutated: false}

	r, err := NewRollback(RollbackConfig{Store: racing})
	if err != nil {
		t.Fatalf("NewRollback: %v", err)
	}
	_, err = r.Execute(context.Background(), RollbackOptions{TargetHashHex: hashA})
	if err == nil {
		t.Fatal("expected error when CAS write loses the race")
	}
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Errorf("err = %v; want ErrPreconditionFailed", err)
	}
}

// rollbackRacingStore wraps a memStore + mutates the pointer
// between the prior-pointer read and the CAS write to simulate
// a concurrent publisher.
type rollbackRacingStore struct {
	inner   *memStore
	mutated bool
}

func (r *rollbackRacingStore) WriteIfNotExists(ctx context.Context, key string, body []byte) error {
	return r.inner.WriteIfNotExists(ctx, key, body)
}
func (r *rollbackRacingStore) ReadObject(ctx context.Context, key string) ([]byte, error) {
	return r.inner.ReadObject(ctx, key)
}
func (r *rollbackRacingStore) ReadPointerGeneration(ctx context.Context, key string) (int64, error) {
	return r.inner.ReadPointerGeneration(ctx, key)
}
func (r *rollbackRacingStore) CASWritePointer(ctx context.Context, key string, body []byte, expectedGen int64) (int64, error) {
	if !r.mutated {
		// Bump the pointer's generation before the CAS write.
		// The CAS write below will see expectedGen != current
		// and return ErrPreconditionFailed.
		r.inner.mu.Lock()
		r.inner.gens[key] = r.inner.nextGen
		r.inner.nextGen++
		r.inner.mu.Unlock()
		r.mutated = true
	}
	return r.inner.CASWritePointer(ctx, key, body, expectedGen)
}
