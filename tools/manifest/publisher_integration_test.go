// path: tools/manifest/publisher_integration_test.go

//go:build integration

// Integration test for the dq-manifest publisher against the
// local fake-gcs-server emulator. Bring the substrate up first:
//
//	make up
//	cd tools/manifest && go test -tags integration ./...
//
// The test exercises ADR-0010 §3.2 row "Object store:
// generation-conditional pointer write" — happy path only.
// fake-gcs-server has a documented CAS fidelity gap per B1-11
// (it accepts ifGenerationMatch but does not enforce it on
// stale generations). The unit-test suite covers the CAS-
// race-loser branch via the in-mem fake; this test covers the
// substrate happy path and the round-trip artifact contract.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const (
	integrationProjectID = "dq-local"
)

func gcsTestClient(t *testing.T) *storage.Client {
	t.Helper()
	if os.Getenv("STORAGE_EMULATOR_HOST") == "" {
		t.Setenv("STORAGE_EMULATOR_HOST", "localhost:4443")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := storage.NewClient(ctx,
		option.WithoutAuthentication(),
		option.WithEndpoint("http://localhost:4443/storage/v1/"),
	)
	if err != nil {
		t.Skipf("integration: cannot create storage client (is `make up` running?): %v", err)
	}
	return cli
}

// ensureBucket creates a unique bucket for this test run so
// parallel suites do not collide on the shared emulator. The
// fake-gcs-server has no rate-limit on bucket creation.
func ensureBucket(t *testing.T, cli *storage.Client) string {
	t.Helper()
	bucket := fmt.Sprintf("itest-manifest-%d", time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cli.Bucket(bucket).Create(ctx, integrationProjectID, nil); err != nil {
		t.Skipf("integration: cannot create bucket %q (is `make up` running?): %v", bucket, err)
	}
	t.Cleanup(func() {
		// Best-effort cleanup. The fake-gcs-server allows bucket
		// deletion even with objects present.
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = cli.Bucket(bucket).Delete(bg)
	})
	return bucket
}

func TestIntegration_PublishToFakeGCS_HappyPath(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()
	bucket := ensureBucket(t, cli)

	store := NewGCSStore(cli, bucket)
	now := time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC)
	pub, err := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-vintegration.1.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
		Now:                     func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	result, err := pub.Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.RulesPublished != 2 {
		t.Errorf("RulesPublished = %d; want 2", result.RulesPublished)
	}
	if result.PointerGen == 0 {
		t.Errorf("PointerGen = 0; want non-zero post-write generation")
	}

	// Round-trip: the published pointer must reference the
	// published manifest body, which must reference the published
	// YAML bodies.
	pointerBody, err := store.ReadObject(ctx, pointerPath)
	if err != nil {
		t.Fatalf("ReadObject(pointer): %v", err)
	}
	var ptr Pointer
	if err := json.Unmarshal(pointerBody, &ptr); err != nil {
		t.Fatalf("Unmarshal pointer: %v", err)
	}
	if !strings.HasPrefix(ptr.ManifestHash, "sha256:") {
		t.Errorf("ManifestHash = %q; want sha256: prefix", ptr.ManifestHash)
	}
	wantManifestHash := strings.TrimPrefix(ptr.ManifestHash, "sha256:")
	if wantManifestHash != result.ManifestHash {
		t.Errorf("pointer.manifest_hash hex = %q; want %q", wantManifestHash, result.ManifestHash)
	}

	manifestBody, err := store.ReadObject(ctx, manifestByHashPath(wantManifestHash))
	if err != nil {
		t.Fatalf("ReadObject(manifest body): %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(manifestBody, &m); err != nil {
		t.Fatalf("Unmarshal manifest body: %v", err)
	}
	if m.RulesetVersion != "rules-vintegration.1.0" {
		t.Errorf("manifest.ruleset_version = %q; want %q", m.RulesetVersion, "rules-vintegration.1.0")
	}
	if len(m.Rules) != 2 {
		t.Fatalf("manifest rules = %d; want 2", len(m.Rules))
	}
	for _, r := range m.Rules {
		body, err := store.ReadObject(ctx, r.YAMLPath)
		if err != nil {
			t.Fatalf("ReadObject(rule %s yaml_path=%s): %v", r.Entity, r.YAMLPath, err)
		}
		if got := sha256Hex(body); got != r.YAMLHash {
			t.Errorf("rule %s body hash = %q; want %q (manifest yaml_hash)", r.Entity, got, r.YAMLHash)
		}
	}
}

func TestIntegration_PublishIdempotent_NoOpOnReplay(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()
	bucket := ensureBucket(t, cli)

	store := NewGCSStore(cli, bucket)
	makePub := func() *Publisher {
		p, err := New(Config{
			Store:                   store,
			RulesetVersion:          "rules-vintegration.1.0",
			EngineCompatibility:     ">=0.1.0, <1.0.0",
			LinterUsed:              "tools-lint-v0.1.0",
			SchemaMirrorDir:         "testdata/schema",
			SupportedSchemaVersions: []int{1},
			Now:                     func() time.Time { return time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC) },
		})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		return p
	}

	ctx := context.Background()
	r1, err := makePub().Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish #1: %v", err)
	}
	r2, err := makePub().Publish(ctx, Options{RulesDir: "testdata/rules"})
	if err != nil {
		t.Fatalf("Publish #2 (idempotent re-publish): %v", err)
	}
	if r1.ManifestHash != r2.ManifestHash {
		t.Errorf("idempotent re-publish changed manifest hash: %q vs %q",
			r1.ManifestHash, r2.ManifestHash)
	}
	// The second pointer generation must be strictly greater
	// than the first (the pointer is the single mutable object).
	if r2.PointerGen <= r1.PointerGen {
		t.Errorf("PointerGen did not advance: %d -> %d", r1.PointerGen, r2.PointerGen)
	}
}

func TestIntegration_DryRun_NoBucketWrites(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()
	bucket := ensureBucket(t, cli)

	store := NewGCSStore(cli, bucket)
	pub, err := New(Config{
		Store:                   store,
		RulesetVersion:          "rules-vintegration.1.0",
		EngineCompatibility:     ">=0.1.0, <1.0.0",
		LinterUsed:              "tools-lint-v0.1.0",
		SchemaMirrorDir:         "testdata/schema",
		SupportedSchemaVersions: []int{1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := pub.Publish(context.Background(), Options{RulesDir: "testdata/rules", DryRun: true}); err != nil {
		t.Fatalf("Publish(dry-run): %v", err)
	}
	// Confirm no pointer exists.
	if _, err := store.ReadObject(context.Background(), pointerPath); !errors.Is(err, ErrObjectNotFound) {
		t.Errorf("DryRun wrote pointer; err = %v want ErrObjectNotFound", err)
	}
}
