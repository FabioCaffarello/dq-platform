// path: tools/manifest/publisher_sandbox_test.go

//go:build sandbox

// First integration-sandbox tier test per ADR-0034
// §"`integration-sandbox`" + B2-18. Exercises the
// `ifGenerationMatch` CAS-race-loser branch against real GCS.
//
// Why this is sandbox-only
// ------------------------
// fake-gcs-server accepts `ifGenerationMatch` query parameters
// without rejecting them when the generation is stale (ADR-0017
// §"`Object store: generation-conditional pointer write` row
// is Partial"). The publisher's CAS race-loser semantics
// (ADR-0005 §4: "the loser receives a precondition-failed
// error") therefore cannot be exercised against the local
// emulator. The unit test in `publisher_test.go` covers the
// loser branch via the in-memory `racingStore` fake; this file
// confirms real GCS enforces the same contract.
//
// Run
// ---
//
//	export DQ_SANDBOX_PROJECT=<gcp-project>
//	export DQ_SANDBOX_BUCKET=<sandbox-bucket>
//	export GOOGLE_APPLICATION_CREDENTIALS=<service-account-json>
//	make test-tools-manifest-sandbox
//
// Tests skip cleanly when the env vars are absent, so a local
// `make test-sandbox` invocation without credentials is a
// no-op rather than a hard failure. The CI lane is gated on
// the matching repo secrets and runs only when configured.

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
)

// sandboxConfig collects the env-var inputs the sandbox tests
// depend on. Returns the values plus an "ok" boolean; when ok
// is false, callers t.Skip with a uniform message so a missing
// credential set produces a clear "skipped" outcome rather than
// a noisy auth error mid-test.
type sandboxConfig struct {
	project string
	bucket  string
}

func loadSandboxConfig(t *testing.T) (sandboxConfig, bool) {
	t.Helper()
	cfg := sandboxConfig{
		project: os.Getenv("DQ_SANDBOX_PROJECT"),
		bucket:  os.Getenv("DQ_SANDBOX_BUCKET"),
	}
	if cfg.project == "" || cfg.bucket == "" {
		return sandboxConfig{}, false
	}
	return cfg, true
}

// sandboxStorageClient creates a real-GCS storage.Client.
// Application Default Credentials are resolved by the SDK — the
// CI lane provides them via GOOGLE_APPLICATION_CREDENTIALS or
// workload-identity bindings.
func sandboxStorageClient(t *testing.T) *storage.Client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("sandbox: storage.NewClient: %v", err)
	}
	t.Cleanup(func() { _ = cli.Close() })
	return cli
}

// uniqueSandboxKey returns a per-test object key under the
// `dq-sandbox/manifests/` prefix. Each test run writes under a
// unique sub-prefix so parallel sandbox lanes do not collide.
// Cleanup runs in t.Cleanup so a failed test still removes its
// objects best-effort.
func uniqueSandboxKey(t *testing.T, bucket *storage.BucketHandle, suffix string) string {
	t.Helper()
	key := fmt.Sprintf("dq-sandbox/manifests/%d/%s", time.Now().UnixNano(), suffix)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = bucket.Object(key).Delete(ctx)
	})
	return key
}

// TestSandbox_CASWritePointer_HappyPath confirms real GCS
// returns a non-zero post-write generation for both the
// DoesNotExist-branch (first write) and the GenerationMatch
// branch (overwrite with matching gen). Gives the sandbox lane
// a smoke signal that doesn't depend on the race-loser path.
func TestSandbox_CASWritePointer_HappyPath(t *testing.T) {
	cfg, ok := loadSandboxConfig(t)
	if !ok {
		t.Skip("sandbox: DQ_SANDBOX_PROJECT + DQ_SANDBOX_BUCKET not set; skipping")
	}
	cli := sandboxStorageClient(t)
	bucket := cli.Bucket(cfg.bucket)
	store := NewGCSStore(cli, cfg.bucket)
	key := uniqueSandboxKey(t, bucket, "happy-path.json")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First write: DoesNotExist precondition. Real GCS returns a
	// fresh non-zero generation.
	gen1, err := store.CASWritePointer(ctx, key, []byte(`{"pointer_version":1}`), 0)
	if err != nil {
		t.Fatalf("first CAS write: %v", err)
	}
	if gen1 == 0 {
		t.Errorf("first write returned generation 0; real GCS must return a non-zero post-write generation")
	}

	// Second write: GenerationMatch precondition matching the
	// first write's generation. Real GCS advances the generation
	// atomically with the write.
	gen2, err := store.CASWritePointer(ctx, key, []byte(`{"pointer_version":2}`), gen1)
	if err != nil {
		t.Fatalf("second CAS write (matching gen): %v", err)
	}
	if gen2 <= gen1 {
		t.Errorf("second write generation %d did not advance from %d", gen2, gen1)
	}
}

// TestSandbox_CASWritePointer_StaleGenerationRejected is the
// load-bearing sandbox test. Real GCS must return 412
// Precondition Failed when CASWritePointer is called with a
// stale `expectedGen`. The GCSStore's googleapi-Error mapping
// then surfaces ErrPreconditionFailed to the caller.
//
// This is the ADR-0017 Partial-row gap fake-gcs-server cannot
// faithfully reproduce; without this test the CAS-race-loser
// contract was only ever exercised against the in-mem fake.
func TestSandbox_CASWritePointer_StaleGenerationRejected(t *testing.T) {
	cfg, ok := loadSandboxConfig(t)
	if !ok {
		t.Skip("sandbox: DQ_SANDBOX_PROJECT + DQ_SANDBOX_BUCKET not set; skipping")
	}
	cli := sandboxStorageClient(t)
	bucket := cli.Bucket(cfg.bucket)
	store := NewGCSStore(cli, cfg.bucket)
	key := uniqueSandboxKey(t, bucket, "stale-gen.json")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Seed the pointer at gen1.
	gen1, err := store.CASWritePointer(ctx, key, []byte(`{"pointer_version":1}`), 0)
	if err != nil {
		t.Fatalf("seed write: %v", err)
	}

	// Advance the pointer to gen2 via a matching-gen CAS write
	// — this is the "another publisher wrote between our
	// ReadPointerGeneration and CASWritePointer" scenario.
	gen2, err := store.CASWritePointer(ctx, key, []byte(`{"pointer_version":2}`), gen1)
	if err != nil {
		t.Fatalf("advance write: %v", err)
	}
	if gen2 == gen1 {
		t.Fatalf("advance did not change generation: %d", gen2)
	}

	// Now try to CAS-write with the original (now-stale) gen1.
	// Real GCS must reject with 412; the GCSStore maps that to
	// ErrPreconditionFailed.
	_, err = store.CASWritePointer(ctx, key, []byte(`{"pointer_version":3}`), gen1)
	if err == nil {
		t.Fatal("stale-generation CAS write succeeded; real GCS must reject with 412 (ADR-0017 §Partial row)")
	}
	if !errors.Is(err, ErrPreconditionFailed) {
		t.Fatalf("err = %v; want ErrPreconditionFailed (the GCSStore maps googleapi 412 to this sentinel)", err)
	}
}
