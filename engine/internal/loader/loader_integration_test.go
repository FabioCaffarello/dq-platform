// path: engine/internal/loader/loader_integration_test.go

//go:build integration

// Integration tests for the loader against the local Compose
// substrate from Wave 3 Phase 2. Bring the stack up first:
//
//	make up
//	cd engine && go test -tags integration ./...
//
// The tests use the fake-gcs-server emulator via the cloud.google.com/go/storage
// client by setting STORAGE_EMULATOR_HOST. The bucket dq-local is
// created if it does not already exist.

package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

const (
	integrationBucket    = "dq-local"
	integrationProjectID = "dq-local"
)

// gcsTestClient sets STORAGE_EMULATOR_HOST and returns a *storage.Client
// pointed at the local emulator. Skips the test if the emulator is
// unreachable (so the test fails cleanly with a useful message
// instead of timing out).
func gcsTestClient(t *testing.T) *storage.Client {
	t.Helper()
	if host := os.Getenv("STORAGE_EMULATOR_HOST"); host == "" {
		if err := os.Setenv("STORAGE_EMULATOR_HOST", "localhost:4443"); err != nil {
			t.Fatalf("setenv STORAGE_EMULATOR_HOST: %v", err)
		}
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
	// Probe the emulator. If unreachable, skip.
	if _, err := cli.Bucket(integrationBucket).Attrs(ctx); err != nil {
		if errors.Is(err, storage.ErrBucketNotExist) {
			if mkErr := cli.Bucket(integrationBucket).Create(ctx, integrationProjectID, nil); mkErr != nil {
				t.Skipf("integration: cannot create bucket %q (is `make up` running?): %v", integrationBucket, mkErr)
			}
		} else {
			t.Skipf("integration: emulator unreachable (is `make up` running?): %v", err)
		}
	}
	return cli
}

// publishObject writes raw bytes to the integration bucket at key.
func publishObject(t *testing.T, cli *storage.Client, key string, raw []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w := cli.Bucket(integrationBucket).Object(key).NewWriter(ctx)
	if _, err := io.Copy(w, strings.NewReader(string(raw))); err != nil {
		_ = w.Close()
		t.Fatalf("write %s: %v", key, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer for %s: %v", key, err)
	}
}

// publishValidManifest writes a pointer + manifest body pair under
// keys unique to this test (so parallel integration tests don't
// collide). Returns the hex sha256 of the body.
func publishValidManifest(t *testing.T, cli *storage.Client) (pointerKey, bodyKey, hashHex string) {
	t.Helper()
	body := []byte(`{
		"manifest_version": 1,
		"ruleset_version": "rules-vintegration",
		"schema_versions_present": [1],
		"engine_compatibility": ">=0.1.0 <2.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	sum := sha256.Sum256(body)
	hashHex = hex.EncodeToString(sum[:])
	pointer := []byte(fmt.Sprintf(`{
		"pointer_version": 1,
		"manifest_hash": "sha256:%s",
		"ruleset_version": "rules-vintegration",
		"published_at": "2026-05-21T00:00:00Z"
	}`, hashHex))
	// Use per-test prefix so concurrent test runs don't stomp on
	// each other if the bucket is shared.
	prefix := fmt.Sprintf("test/%s/", t.Name())
	pointerKey = prefix + "manifests/latest.json"
	bodyKey = prefix + "manifests/by-hash/sha256-" + hashHex + ".json"
	publishObject(t, cli, pointerKey, pointer)
	publishObject(t, cli, bodyKey, body)
	return pointerKey, bodyKey, hashHex
}

// makeLoader returns a Loader pointed at a per-test object key
// layout so each test is self-contained.
func makeLoader(t *testing.T, cli *storage.Client, pointerKey, bodyPrefix string) *Loader {
	t.Helper()
	store := NewGCSStore(cli, integrationBucket)
	l, err := New(store, Config{
		EngineVersion:           "0.1.0",
		SupportedSchemaVersions: []int{1},
		PointerKey:              pointerKey,
		BodyPrefix:              bodyPrefix,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return l
}

func TestIntegration_LoadFromObjectStore(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()

	pointerKey, _, hashHex := publishValidManifest(t, cli)
	bodyPrefix := strings.TrimSuffix(pointerKey, "manifests/latest.json") + "manifests/by-hash/"

	loader := makeLoader(t, cli, pointerKey, bodyPrefix)
	m, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if m.Hash != hashHex {
		t.Errorf("Hash = %q; want %q", m.Hash, hashHex)
	}
	if m.RulesetVersion != "rules-vintegration" {
		t.Errorf("RulesetVersion = %q; want rules-vintegration", m.RulesetVersion)
	}
}

// countingStore wraps a Store and counts ReadObject calls. Used to
// assert hash-short-circuit performs no body fetch.
type countingStore struct {
	inner Store
	reads atomic.Int64
}

func (s *countingStore) ReadObject(ctx context.Context, key string) ([]byte, error) {
	s.reads.Add(1)
	return s.inner.ReadObject(ctx, key)
}

func TestIntegration_HashShortCircuit(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()

	pointerKey, _, hashHex := publishValidManifest(t, cli)
	bodyPrefix := strings.TrimSuffix(pointerKey, "manifests/latest.json") + "manifests/by-hash/"

	inner := NewGCSStore(cli, integrationBucket)
	counter := &countingStore{inner: inner}
	loader, err := New(counter, Config{
		EngineVersion:           "0.1.0",
		SupportedSchemaVersions: []int{1},
		PointerKey:              pointerKey,
		BodyPrefix:              bodyPrefix,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := loader.Load(context.Background()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	before := counter.reads.Load()

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
	delta := counter.reads.Load() - before
	if delta != 1 {
		t.Errorf("Refresh did %d ReadObject calls; want 1 (pointer only, no body fetch)", delta)
	}
}

func TestIntegration_HashMismatchRefuseLoad(t *testing.T) {
	cli := gcsTestClient(t)
	defer cli.Close()

	pointerKey, _, hashHex := publishValidManifest(t, cli)
	bodyPrefix := strings.TrimSuffix(pointerKey, "manifests/latest.json") + "manifests/by-hash/"

	// Corrupt the body: write different bytes at the same key.
	bodyKey := bodyPrefix + "sha256-" + hashHex + ".json"
	publishObject(t, cli, bodyKey, []byte(`{"manifest_version":1,"ruleset_version":"corrupted","schema_versions_present":[1],"engine_compatibility":">=0.1.0 <2.0.0","linter_used":"tools-lint-v0.1.0","generated_at":"2026-05-21T00:00:00Z","rules":[]}`))

	loader := makeLoader(t, cli, pointerKey, bodyPrefix)
	_, err := loader.Load(context.Background())
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("Load with corrupted body: err = %v; want ErrHashMismatch", err)
	}
}
