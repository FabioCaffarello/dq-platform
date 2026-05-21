// path: engine/internal/loader/store.go

package loader

import (
	"context"
	"errors"
)

// Store is the object-store abstraction the loader reads from. The
// production implementation wraps the GCS Go SDK (see store_gcs.go);
// unit tests use an in-memory fake (see loader_test.go).
//
// Errors returned by Store implementations should signal the
// "object not found" case via ErrObjectNotFound so the loader can
// distinguish missing-pointer from operational-failure modes — both
// flow through ADR-0007 CC1 startup-mode process exit, but typed
// errors let the engine binary emit the right structured failure
// classification per ADR-0007 CC14 (observability emission).
type Store interface {
	// ReadObject returns the bytes of the object at the given key.
	// Returns ErrObjectNotFound (or an error wrapping it) when the
	// object does not exist.
	ReadObject(ctx context.Context, key string) ([]byte, error)
}

// ErrObjectNotFound is returned by Store implementations when the
// requested object does not exist.
var ErrObjectNotFound = errors.New("object not found")
