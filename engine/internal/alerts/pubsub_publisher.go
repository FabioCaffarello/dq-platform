// path: engine/internal/alerts/pubsub_publisher.go

package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub/v2"
)

// PubSubPublisher implements Publisher against the Pub/Sub v2 Go
// client. It works against production Pub/Sub and against the
// Pub/Sub emulator (the Phase 2 local Compose stack exposes one)
// via the standard PUBSUB_EMULATOR_HOST env var honored by the
// SDK.
//
// One PubSubPublisher wraps one topic; the engine binary
// creates exactly one per process (the engine emits to one
// topic per environment per ADR-0006 CC3). The underlying
// *pubsub.Publisher batches messages internally; callers
// should invoke Close at process shutdown to flush.
//
// # Lifecycle
//
// Construct exactly once with NewPubSubPublisher; call Publish
// concurrently from any goroutine; call Close exactly once at
// shutdown. Calls to Publish after Close return an error from
// the underlying SDK — callers must order shutdown so that
// emits drain before Close. The engine binary's main loop does
// this via deferred close ordering.
//
// Close is idempotent (sync.Once-guarded) so a redundant defer
// in a future caller cannot trigger the underlying SDK's
// double-Stop panic.
type PubSubPublisher struct {
	publisher *pubsub.Publisher
	closeOnce sync.Once
}

// NewPubSubPublisher wraps an existing *pubsub.Client + topic
// ID. The caller owns the client's lifecycle (the engine binary
// creates one at startup and closes it at shutdown).
//
// topicNameOrID accepts either the bare topic ID (preferred) or
// the fully-qualified `projects/<project>/topics/<id>` form, per
// pubsub v2's Publisher constructor.
func NewPubSubPublisher(client *pubsub.Client, topicNameOrID string) *PubSubPublisher {
	return &PubSubPublisher{publisher: client.Publisher(topicNameOrID)}
}

// Publish serializes the event as JSON and emits it as one
// Pub/Sub message. Blocks until the underlying client confirms
// the publish.
//
// The underlying SDK batches internally; Publish returns once
// the batch carrying this event has been acked by the broker.
// Per ADR-0006 §4, consumers must tolerate unknown fields; this
// publisher emits the canonical Event JSON encoding and adds no
// envelope, no message attributes.
func (p *PubSubPublisher) Publish(ctx context.Context, event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal alert event: %w", err)
	}

	result := p.publisher.Publish(ctx, &pubsub.Message{Data: data})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publish to Pub/Sub topic %s: %w", p.publisher.ID(), err)
	}
	return nil
}

// Close flushes the publisher's internal batch and releases its
// resources. The engine binary calls Close at shutdown so
// in-flight emits land before the process exits.
//
// Note: the underlying *pubsub.Publisher.Stop() blocks until all
// in-flight batches drain and accepts no context. If the engine
// shutdown budget is tight, callers must enforce their own
// deadline around the deferred Close. The Phase 5 binary uses a
// 10-second outer wait that bounds the entire shutdown
// (loaderRefreshLoop + orphanScanLoop + publisher flush) so a
// stuck Stop cannot pin the process forever.
//
// Idempotent: safe to call any number of times. Subsequent calls
// after the first are no-ops.
func (p *PubSubPublisher) Close() {
	p.closeOnce.Do(func() { p.publisher.Stop() })
}
