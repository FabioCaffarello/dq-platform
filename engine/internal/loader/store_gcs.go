// path: engine/internal/loader/store_gcs.go

package loader

import (
	"context"
	"errors"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
)

// GCSStore is a Store implementation backed by the GCS Go SDK. It
// works against production GCS and against fake-gcs-server via the
// STORAGE_EMULATOR_HOST environment variable (commodity emulator
// posture per ADR-0010).
type GCSStore struct {
	bucket *storage.BucketHandle
}

// NewGCSStore wraps an existing *storage.Client and bucket name. The
// client is the caller's responsibility (the engine binary creates
// one at startup with the appropriate auth context and reuses it).
func NewGCSStore(client *storage.Client, bucketName string) *GCSStore {
	return &GCSStore{bucket: client.Bucket(bucketName)}
}

// ReadObject implements Store.ReadObject. Maps storage.ErrObjectNotExist
// to the loader's ErrObjectNotFound so the loader's failure-mode
// classification is portable across Store impls.
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
