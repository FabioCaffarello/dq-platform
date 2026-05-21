// path: engine/internal/alerts/dedup.go

package alerts

import "sync"

// AttemptDeduper implements the engine-side literal-duplicate
// suppression per ADR-0006 CC5. The dedup key is
// (execution_id, attempt_id, check_id, result); a tuple seen
// once causes subsequent calls for the same tuple to return
// false, suppressing the emit.
//
// Scope: one AttemptDeduper per attempt. The runner constructs
// one at the start of each attempt and discards it when the
// attempt finalizes. This bounds the in-memory map size to the
// attempt's check count + one execution-level entry.
//
// ADR-0006 CC5 names the engine-side dedup as **correctness
// against re-emission within an attempt**, not as the primary
// enforcement of the platform's "≤1 user-visible alert per
// failing check" invariant. That invariant lives in the
// consumer-side dedup (configured per environment per CC3 +
// CC13). Engine-side dedup is the belt; consumer-side dedup is
// the suspenders.
//
// The deduper does not store the Event itself, only the key
// tuple, so the in-memory footprint is small and bounded.
type AttemptDeduper struct {
	mu   sync.Mutex
	seen map[dedupKey]struct{}
}

type dedupKey struct {
	executionID string
	attemptID   string
	checkID     string
	result      string // result enum value; empty for execution-level events
}

// NewAttemptDeduper returns a fresh deduper. Construct one per
// attempt; do not share across attempts.
func NewAttemptDeduper() *AttemptDeduper {
	return &AttemptDeduper{seen: make(map[dedupKey]struct{})}
}

// ShouldPublish returns true the first time the event's key
// tuple is seen by this AttemptDeduper, and false on every
// subsequent call with the same tuple. The check_id and result
// fields are part of the key; events with the same
// (execution_id, attempt_id, check_id) but different results
// are NOT considered duplicates by the engine-side deduper —
// the consumer-side deduper collapses those per CC5.
//
// For events without an execution_id (engine-startup loader
// failures per ADR-0007 CC1/CC2), the key is constructed from
// the empty-string components; same source can still produce
// duplicates if the key tuple matches.
func (d *AttemptDeduper) ShouldPublish(event Event) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := dedupKey{}
	if event.ExecutionID != nil {
		key.executionID = *event.ExecutionID
	}
	if event.AttemptID != nil {
		key.attemptID = *event.AttemptID
	}
	if event.CheckID != nil {
		key.checkID = *event.CheckID
	}
	if event.Result != nil {
		key.result = string(*event.Result)
	}

	if _, exists := d.seen[key]; exists {
		return false
	}
	d.seen[key] = struct{}{}
	return true
}
