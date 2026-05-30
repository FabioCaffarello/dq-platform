// path: engine/internal/loader/loader_metrics_test.go

package loader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"dq-platform/engine/internal/metrics"
)

// newLoaderWithMetrics returns a Loader wired to a real
// LoaderMetrics handle set so per-class increments can be
// asserted via prometheus/testutil. Mirrors newTestLoader but
// injects Metrics.
func newLoaderWithMetrics(t *testing.T, store Store, m metrics.LoaderMetrics) *Loader {
	t.Helper()
	l, err := New(store, Config{
		EngineVersion:           "0.1.0",
		SupportedSchemaVersions: []int{1},
		Metrics:                 m,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return l
}

func TestRefresh_EmitsPointerRead_OnReadObjectFailure(t *testing.T) {
	r := metrics.New()
	store := newMemStore() // pointer key absent → ReadObject returns ErrObjectNotFound
	l := newLoaderWithMetrics(t, store, r.Loader)

	if _, _, err := l.Refresh(context.Background(), "deadbeef"); err == nil {
		t.Fatalf("Refresh: want error; got nil")
	}
	got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues("pointer_read"))
	if got != 1 {
		t.Errorf("dq_loader_refresh_failures_total{error_class=pointer_read} = %v; want 1", got)
	}
}

func TestRefresh_EmitsPointerRead_OnInvalidPointerJSON(t *testing.T) {
	r := metrics.New()
	store := newMemStore()
	store.objects["manifests/latest.json"] = []byte("not json")
	l := newLoaderWithMetrics(t, store, r.Loader)

	if _, _, err := l.Refresh(context.Background(), "deadbeef"); err == nil {
		t.Fatalf("Refresh: want error; got nil")
	}
	got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues("pointer_read"))
	if got != 1 {
		t.Errorf("dq_loader_refresh_failures_total{error_class=pointer_read} = %v; want 1", got)
	}
}

func TestRefresh_EmitsBodyFetch_OnMissingBody(t *testing.T) {
	r := metrics.New()
	store := newMemStore()
	_, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	// Pointer references the body hash, but the body object is
	// absent — ReadObject fails on the by-hash path.
	l := newLoaderWithMetrics(t, store, r.Loader)

	if _, _, err := l.Refresh(context.Background(), "different-hash-so-no-shortcircuit"); err == nil {
		t.Fatalf("Refresh: want error; got nil")
	}
	got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues("body_fetch"))
	if got != 1 {
		t.Errorf("dq_loader_refresh_failures_total{error_class=body_fetch} = %v; want 1", got)
	}
}

func TestRefresh_EmitsHashMismatch_OnTamperedBody(t *testing.T) {
	r := metrics.New()
	store := newMemStore()
	body, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	// Stash the body at the pointer-named key, then tamper —
	// fetchAndVerify's sha256 check fires ErrHashMismatch.
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = append(body, []byte(" tampered")...)
	l := newLoaderWithMetrics(t, store, r.Loader)

	if _, _, err := l.Refresh(context.Background(), "different-hash-so-no-shortcircuit"); err == nil {
		t.Fatalf("Refresh: want error; got nil")
	}
	got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues("hash_mismatch"))
	if got != 1 {
		t.Errorf("dq_loader_refresh_failures_total{error_class=hash_mismatch} = %v; want 1", got)
	}
}

func TestRefresh_EmitsParseError_OnNonJSONBody(t *testing.T) {
	r := metrics.New()
	store := newMemStore()
	// A body that hashes to a valid digest but is not JSON: build
	// the body deterministically and reference it by its own hash.
	bogus := []byte("not a manifest")
	hashHex := sha256Hex(bogus)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = bogus
	l := newLoaderWithMetrics(t, store, r.Loader)

	if _, _, err := l.Refresh(context.Background(), "different-hash-so-no-shortcircuit"); err == nil {
		t.Fatalf("Refresh: want error; got nil")
	}
	got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues("parse_error"))
	if got != 1 {
		t.Errorf("dq_loader_refresh_failures_total{error_class=parse_error} = %v; want 1", got)
	}
}

func TestRefresh_EmitsCompatibilityContract_OnEngineMismatch(t *testing.T) {
	r := metrics.New()
	store := newMemStore()
	// Manifest whose engine_compatibility constraint excludes the
	// running engine version (0.1.0).
	body := []byte(`{
		"manifest_version": 1,
		"ruleset_version": "rules-v1.0.0",
		"schema_versions_present": [1],
		"engine_compatibility": ">=99.0.0",
		"linter_used": "tools-lint-v0.1.0",
		"generated_at": "2026-05-21T00:00:00Z",
		"rules": []
	}`)
	hashHex := sha256Hex(body)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	store.objects["manifests/by-hash/sha256-"+hashHex+".json"] = body
	l := newLoaderWithMetrics(t, store, r.Loader)

	if _, _, err := l.Refresh(context.Background(), "different-hash-so-no-shortcircuit"); err == nil {
		t.Fatalf("Refresh: want error; got nil")
	}
	got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues("compatibility_contract"))
	if got != 1 {
		t.Errorf("dq_loader_refresh_failures_total{error_class=compatibility_contract} = %v; want 1", got)
	}
}

func TestRefresh_NoEmission_OnHashShortCircuit(t *testing.T) {
	r := metrics.New()
	store := newMemStore()
	_, hashHex := makeManifestBody(t)
	store.objects["manifests/latest.json"] = makePointer(t, hashHex)
	l := newLoaderWithMetrics(t, store, r.Loader)

	// currentHash equals pointer's hash → short-circuit → success → no emission.
	if _, swapped, err := l.Refresh(context.Background(), hashHex); err != nil || swapped {
		t.Fatalf("Refresh hash-shortcircuit: err=%v swapped=%v", err, swapped)
	}
	// All five error classes stay at zero.
	for _, class := range []string{"pointer_read", "body_fetch", "hash_mismatch", "parse_error", "compatibility_contract"} {
		got := testutil.ToFloat64(r.Loader.RefreshFailuresTotal.WithLabelValues(class))
		if got != 0 {
			t.Errorf("dq_loader_refresh_failures_total{error_class=%s} on short-circuit = %v; want 0", class, got)
		}
	}
}

// sha256Hex returns the lowercase hex sha256 of b — used by the
// metric-emission tests to construct synthetic pointer + body
// pairs that hit each ADR-0055 §Clause 5 error_class branch.
func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
