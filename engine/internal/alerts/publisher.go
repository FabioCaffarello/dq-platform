// path: engine/internal/alerts/publisher.go

// Package alerts implements the alerting emission surface per
// ADR-0006. The engine emits structured Events to a Pub/Sub
// topic; consumers route them per `_owners.yaml` + engine
// deployment config.
//
// ADR-0006 CC5 commits two-layer dedup:
//   - Engine-side: AttemptDeduper suppresses literal duplicate
//     emits within a single attempt.
//   - Consumer-side: per-environment dedup window collapses
//     retries to one user-visible alert per failing check (the
//     primary enforcement of B0-2 I3). Consumer-side dedup is
//     out of scope for this package; it lives in the alerting
//     consumer (Phase 7+ deploy work).
//
// Publishers are interface-typed so the engine can wire a
// real Pub/Sub publisher (production / emulator) or a
// NoopPublisher (tests, or engine started without an alerts
// topic configured).
package alerts

import "context"

// Publisher is the emission surface for one Event. Implementations
// are expected to be safe for concurrent use; the engine binary
// may emit from multiple goroutines (runner attempts + orphan
// scans + future loader/scheduler hooks).
type Publisher interface {
	// Publish emits one Event. Returns an error on operational
	// failure (network, serialization). The caller is
	// responsible for engine-side dedup via AttemptDeduper
	// before invoking Publish.
	Publish(ctx context.Context, event Event) error
}
