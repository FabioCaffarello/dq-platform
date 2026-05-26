// path: tools/manifest/rollback_integration_test.go

//go:build integration

// Integration test for the dq-manifest set-pointer subcommand
// against the local fake-gcs-server emulator. Bring the
// substrate up first:
//
//	make up
//	cd tools/manifest && go test -tags integration ./...
//
// The test exercises the rollback half of ADR-0005 §3 + §4:
// publish manifest A, publish manifest B, set-pointer back to
// A, verify the pointer carries A's hash + A's ruleset_version.
//
// Per the publisher integration test, fake-gcs-server has a
// documented CAS fidelity gap (B1-11): it accepts
// ifGenerationMatch but does not enforce it on stale
// generations. The unit-test suite covers the CAS-race-loser
// branch via the in-mem fake; this test covers the substrate
// happy path.

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIntegration_SetPointer_RollbackHappyPath(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()
	bucket := ensureBucket(t, cli)

	store := NewGCSStore(cli, bucket)
	ctx := context.Background()

	// Publish manifest A (rules-vintegration.A.0).
	pubA, err := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-vintegration.A.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
		Now:                     func() time.Time { return time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("New(A): %v", err)
	}
	resA, err := pubA.Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish(A): %v", err)
	}

	// Publish manifest B (rules-vintegration.B.0). Different
	// ruleset_version → different manifest body → different hash.
	pubB, err := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-vintegration.B.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
		Now:                     func() time.Time { return time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("New(B): %v", err)
	}
	resB, err := pubB.Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish(B): %v", err)
	}
	if resA.ManifestHash == resB.ManifestHash {
		t.Fatalf("manifests A and B have the same hash %q; test setup broken", resA.ManifestHash)
	}

	// Sanity: pointer currently references B.
	pointerBeforeRollback, err := store.ReadObject(ctx, pointerPath)
	if err != nil {
		t.Fatalf("ReadObject(pointer pre-rollback): %v", err)
	}
	var ptrPre Pointer
	if err := json.Unmarshal(pointerBeforeRollback, &ptrPre); err != nil {
		t.Fatalf("Unmarshal pointer pre-rollback: %v", err)
	}
	if !strings.HasSuffix(ptrPre.ManifestHash, resB.ManifestHash) {
		t.Fatalf("pre-rollback pointer.manifest_hash = %q; want suffix %q (B's hash)", ptrPre.ManifestHash, resB.ManifestHash)
	}

	// Roll back to A via set-pointer.
	rb, err := NewRollback(RollbackConfig{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("NewRollback: %v", err)
	}
	rbRes, err := rb.Execute(ctx, RollbackOptions{TargetHashHex: resA.ManifestHash})
	if err != nil {
		t.Fatalf("set-pointer to A: %v", err)
	}
	if rbRes.TargetHash != resA.ManifestHash {
		t.Errorf("TargetHash = %q; want %q (A's hash)", rbRes.TargetHash, resA.ManifestHash)
	}
	if rbRes.TargetRulesetVer != "rules-vintegration.A.0" {
		t.Errorf("TargetRulesetVer = %q; want rules-vintegration.A.0", rbRes.TargetRulesetVer)
	}
	if rbRes.PostPointerGen <= rbRes.PriorPointerGen {
		t.Errorf("PostPointerGen (%d) must exceed PriorPointerGen (%d)", rbRes.PostPointerGen, rbRes.PriorPointerGen)
	}

	// Verify: pointer now references A + carries A's ruleset_version.
	pointerAfter, err := store.ReadObject(ctx, pointerPath)
	if err != nil {
		t.Fatalf("ReadObject(pointer post-rollback): %v", err)
	}
	var ptrPost Pointer
	if err := json.Unmarshal(pointerAfter, &ptrPost); err != nil {
		t.Fatalf("Unmarshal pointer post-rollback: %v", err)
	}
	wantHash := "sha256:" + resA.ManifestHash
	if ptrPost.ManifestHash != wantHash {
		t.Errorf("post-rollback pointer.manifest_hash = %q; want %q", ptrPost.ManifestHash, wantHash)
	}
	if ptrPost.RulesetVersion != "rules-vintegration.A.0" {
		t.Errorf("post-rollback pointer.ruleset_version = %q; want rules-vintegration.A.0", ptrPost.RulesetVersion)
	}
}

func TestIntegration_SetPointer_DryRun_DoesNotWrite(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()
	bucket := ensureBucket(t, cli)

	store := NewGCSStore(cli, bucket)
	ctx := context.Background()

	// Publish A then B.
	pubA, _ := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-vintegration.A.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
		Now:                     func() time.Time { return time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC) },
	})
	resA, err := pubA.Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish(A): %v", err)
	}
	pubB, _ := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-vintegration.B.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
		Now:                     func() time.Time { return time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC) },
	})
	resB, err := pubB.Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish(B): %v", err)
	}

	// Dry-run set-pointer to A: must NOT mutate the pointer.
	rb, _ := NewRollback(RollbackConfig{Store: store})
	rbRes, err := rb.Execute(ctx, RollbackOptions{
		TargetHashHex: resA.ManifestHash,
		DryRun:        true,
	})
	if err != nil {
		t.Fatalf("set-pointer dry-run: %v", err)
	}
	if rbRes.PostPointerGen != 0 {
		t.Errorf("DryRun PostPointerGen = %d; want 0", rbRes.PostPointerGen)
	}

	// Pointer must still reference B.
	pointerAfter, err := store.ReadObject(ctx, pointerPath)
	if err != nil {
		t.Fatalf("ReadObject(pointer post-dryrun): %v", err)
	}
	var ptrPost Pointer
	_ = json.Unmarshal(pointerAfter, &ptrPost)
	wantHash := "sha256:" + resB.ManifestHash
	if ptrPost.ManifestHash != wantHash {
		t.Errorf("DryRun changed the pointer; got %q want %q", ptrPost.ManifestHash, wantHash)
	}
}
