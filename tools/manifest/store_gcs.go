// path: tools/manifest/store_gcs.go

package main

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/googleapi"
)

// GCSStore is a Store implementation backed by the GCS Go SDK.
// Works against production GCS and against the fake-gcs-server
// emulator via STORAGE_EMULATOR_HOST per ADR-0010 §3.2.
//
// # Substrate fidelity gap (B1-11)
//
// fake-gcs-server accepts ifGenerationMatch query parameters
// without rejecting them but does not enforce the precondition:
// stale-generation writes still succeed. The publisher's CAS
// race-loser semantics (ADR-0005 §4 "the loser receives a
// precondition-failed error") therefore cannot be exercised
// locally — strict CAS enforcement runs in the sandbox-required
// CI lane. The publish happy path is fully exercised locally;
// only the race-loser branch is impacted.
//
// This Store reports the gap through its error mapping: it maps
// the SDK's googleapi 412 Precondition Failed to
// ErrPreconditionFailed (the failure surface the production
// substrate uses). When fake-gcs-server doesn't generate that
// 412, the unit tests cover the loser branch via the in-mem
// fake; the integration test covers the winner branch only.
type GCSStore struct {
	bucket *storage.BucketHandle
}

// NewGCSStore wraps an existing *storage.Client and bucket
// name. The caller owns the client lifecycle (the CLI creates
// one per invocation and closes it on exit).
func NewGCSStore(client *storage.Client, bucketName string) *GCSStore {
	return &GCSStore{bucket: client.Bucket(bucketName)}
}

// WriteIfNotExists writes body at key with a DoesNotExist
// precondition. Maps the SDK's googleapi 412 Precondition
// Failed to ErrAlreadyExists so the publisher can pattern-match
// the immutable-conflict case.
func (s *GCSStore) WriteIfNotExists(ctx context.Context, key string, body []byte) error {
	obj := s.bucket.Object(key).If(storage.Conditions{DoesNotExist: true})
	w := obj.NewWriter(ctx)
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return fmt.Errorf("write body to %s: %w", key, err)
	}
	if err := w.Close(); err != nil {
		if isPreconditionFailed(err) {
			return fmt.Errorf("%s: %w", key, ErrAlreadyExists)
		}
		return fmt.Errorf("close writer for %s: %w", key, err)
	}
	return nil
}

// ReadObject returns the bytes at key. Maps the SDK's
// ErrObjectNotExist to the publisher's ErrObjectNotFound.
func (s *GCSStore) ReadObject(ctx context.Context, key string) ([]byte, error) {
	r, err := s.bucket.Object(key).NewReader(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return nil, fmt.Errorf("%s: %w", key, ErrObjectNotFound)
		}
		return nil, fmt.Errorf("open reader for %s: %w", key, err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", key, err)
	}
	return data, nil
}

// ReadPointerGeneration returns the current GCS generation of
// the object at key, or 0 if the object does not exist. The
// publisher uses this to compute expectedGen for the CAS write.
func (s *GCSStore) ReadPointerGeneration(ctx context.Context, key string) (int64, error) {
	attrs, err := s.bucket.Object(key).Attrs(ctx)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("attrs for %s: %w", key, err)
	}
	return attrs.Generation, nil
}

// CASWritePointer is the generation-conditional write per
// ADR-0005 §3.
//   - expectedGen=0 → DoesNotExist precondition (first publish
//     to an empty bucket; the pointer must not pre-exist).
//   - expectedGen>0 → GenerationMatch precondition (the pointer
//     must still have the generation the caller observed).
//
// Maps googleapi 412 Precondition Failed to
// ErrPreconditionFailed so the CLI can pattern-match the CAS-
// loss case for its dedicated exit code.
func (s *GCSStore) CASWritePointer(ctx context.Context, key string, body []byte, expectedGen int64) (int64, error) {
	var conds storage.Conditions
	if expectedGen == 0 {
		conds.DoesNotExist = true
	} else {
		conds.GenerationMatch = expectedGen
	}
	obj := s.bucket.Object(key).If(conds)
	w := obj.NewWriter(ctx)
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return 0, fmt.Errorf("write pointer body to %s: %w", key, err)
	}
	if err := w.Close(); err != nil {
		if isPreconditionFailed(err) {
			return 0, fmt.Errorf("CAS write of %s (expectedGen=%d): %w", key, expectedGen, ErrPreconditionFailed)
		}
		return 0, fmt.Errorf("close pointer writer for %s: %w", key, err)
	}
	// Writer.Attrs is populated on Close; the Generation it
	// returns is the new pointer generation, atomic with the
	// CAS write that produced it.
	return w.Attrs().Generation, nil
}

// isPreconditionFailed identifies the googleapi error the GCS
// SDK returns when an ifGenerationMatch or DoesNotExist
// precondition rejects the write. The SDK does not export a
// typed sentinel for this; the HTTP status code 412 is the
// canonical signal.
func isPreconditionFailed(err error) bool {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return apiErr.Code == 412
	}
	return false
}
