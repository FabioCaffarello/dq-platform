// path: tools/manifest/store.go

package main

import (
	"context"
	"errors"
)

// Store is the object-store abstraction the publisher writes
// to. The production implementation wraps the GCS Go SDK
// (store_gcs.go); unit tests use the in-memory fake in
// publisher_test.go.
//
// Each method maps directly to one of the three write steps in
// ADR-0005 §4:
//
//   - WriteIfNotExists implements the immutable by-hash writes
//     (yamls/by-hash/ and manifests/by-hash/) per ADR-0005 §2.
//     The publisher tolerates ErrAlreadyExists since hash
//     collision implies identical content (§2 immutability).
//
//   - ReadPointerGeneration captures the current pointer state
//     before the CAS write per ADR-0005 §3 (single mutable
//     control-plane object).
//
//   - CASWritePointer is the generation-conditional write per
//     ADR-0005 §4 step 4. expectedGen=0 ⇒ DoesNotExist
//     precondition (first publish to an empty bucket).
//
// ReadObject is also exposed so the integration test can
// verify written bodies; the publisher itself does not invoke
// it.
type Store interface {
	// WriteIfNotExists writes body at key only when no object
	// exists there. Returns ErrAlreadyExists if an object is
	// already present (the publisher checks for this sentinel
	// and tolerates it for by-hash paths).
	WriteIfNotExists(ctx context.Context, key string, body []byte) error

	// ReadObject returns the bytes at key. Returns
	// ErrObjectNotFound if the object does not exist.
	ReadObject(ctx context.Context, key string) ([]byte, error)

	// ReadPointerGeneration returns the current GCS generation
	// of the object at key, or 0 if the object does not exist.
	// Used by the publisher to compute expectedGen for the CAS
	// write.
	ReadPointerGeneration(ctx context.Context, key string) (int64, error)

	// CASWritePointer is the generation-conditional write per
	// ADR-0005 §3. expectedGen=0 means "write only if the object
	// does not exist" (first publish). expectedGen>0 means
	// "write only if the current generation matches".
	// Returns ErrPreconditionFailed on CAS loss.
	//
	// On success returns the post-write generation of the
	// pointer object. Returned atomically with the write so a
	// concurrent winner cannot overwrite our generation between
	// the CAS and a follow-up read (a separate Attrs RPC would
	// race). The GCS SDK populates Writer.Attrs().Generation on
	// Close, which is what the production impl returns.
	CASWritePointer(ctx context.Context, key string, body []byte, expectedGen int64) (int64, error)
}

// Sentinel errors. The publisher and CLI both pattern-match
// these via errors.Is, so the underlying type is the contract
// surface.
var (
	// ErrAlreadyExists signals an immutable-write conflict. The
	// publisher tolerates this for by-hash paths because
	// ADR-0005 §2 commits by-hash objects as immutable: a hash
	// collision means identical bytes, so the existing object
	// is the publisher's intended write.
	ErrAlreadyExists = errors.New("object already exists")

	// ErrObjectNotFound signals a missing object on read.
	ErrObjectNotFound = errors.New("object not found")

	// ErrPreconditionFailed signals a CAS write that lost the
	// race per ADR-0005 §4 ("the loser receives a precondition-
	// failed error and surfaces it as a publication failure").
	// The CLI maps this to a dedicated exit code so operator
	// wrapper scripts can retry without confusing it with an
	// operational failure.
	ErrPreconditionFailed = errors.New("CAS precondition failed (pointer generation mismatch)")

	// ErrVerificationFailed signals one of the three ADR-0001
	// pre-publish verifications rejected the input. The CLI
	// maps this to a dedicated exit code so CI surfaces the
	// content problem distinctly from operational failures.
	ErrVerificationFailed = errors.New("pre-publish verification failed")
)
