// path: engine/internal/alerts/noop_publisher.go

package alerts

import "context"

// NoopPublisher discards every Event. Used by the engine when
// no Pub/Sub topic is configured (e.g., a local-dev binary that
// doesn't want to depend on the Pub/Sub emulator) and by tests
// that don't care about asserting emission.
//
// Returning nil from Publish is intentional — Noop is, by
// definition, never "failing".
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ Event) error { return nil }
