// path: engine/internal/runner/attempt.go

package runner

import "github.com/google/uuid"

// AttemptIDFunc generates a fresh attempt_id per ADR-0003 CC4.
// The default is uuid.NewString (version-4 random UUID); tests
// inject a deterministic generator.
type AttemptIDFunc func() string

// DefaultAttemptID is the production attempt_id generator. It
// returns a fresh version-4 UUID per call.
func DefaultAttemptID() string { return uuid.NewString() }
