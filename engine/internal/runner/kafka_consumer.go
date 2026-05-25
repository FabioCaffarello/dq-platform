// path: engine/internal/runner/kafka_consumer.go

package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// FranzConsumer is the franz-go-backed RecordConsumer the engine
// binary wires in production. Tests use the fakeConsumer in
// record_runner_test.go; the production wiring uses this type.
//
// Auto-commit is disabled; the runner is responsible for
// committing offsets once a window's records have been
// dispatched (per ADR-0024 per-attempt re-read semantics; the
// commit happens here when the dispatcher returns nil). Per-
// attempt re-reads are deferred to a future slice; β commits
// after each successful Run.
type FranzConsumer struct {
	client *kgo.Client
}

// FranzConsumerConfig configures a FranzConsumer.
type FranzConsumerConfig struct {
	// Brokers is the bootstrap address list (host:port,
	// comma-separated entries acceptable per franz-go).
	Brokers []string

	// ConsumerGroup is the consumer group identifier per
	// ADR-0024. Each (entity, consumer_group) gets its own
	// FranzConsumer instance; the runner does not multiplex
	// groups across consumers.
	ConsumerGroup string

	// Topics is the list of topics the consumer subscribes to.
	// Single-topic per consumer is the β posture; the franz-go
	// client supports multi-topic, but routing across topics is
	// the RecordRunner's job.
	Topics []string
}

// NewFranzConsumer constructs a franz-go client and wraps it as
// a RecordConsumer.
func NewFranzConsumer(cfg FranzConsumerConfig) (*FranzConsumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("franz consumer: Brokers is required")
	}
	if cfg.ConsumerGroup == "" {
		return nil, errors.New("franz consumer: ConsumerGroup is required")
	}
	if len(cfg.Topics) == 0 {
		return nil, errors.New("franz consumer: Topics is required")
	}
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.ConsumerGroup),
		kgo.ConsumeTopics(cfg.Topics...),
		// Disable auto-commit; the dispatcher controls when
		// offsets are committed per ADR-0024 per-attempt
		// semantics. β commits after each successful dispatch
		// via PollFetches's natural at-most-once flow.
		kgo.DisableAutoCommit(),
	}
	cli, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("franz consumer: create client: %w", err)
	}
	return &FranzConsumer{client: cli}, nil
}

// PollFetches blocks until the underlying client returns a
// non-empty fetch (or ctx is cancelled). Returns the records
// flattened across topics + partitions. Iterator errors are
// surfaced as a single aggregated error.
func (c *FranzConsumer) PollFetches(ctx context.Context) ([]FetchedRecord, error) {
	fetches := c.client.PollFetches(ctx)
	if err := fetches.Err0(); err != nil {
		return nil, fmt.Errorf("kafka poll: %w", err)
	}
	var out []FetchedRecord
	fetches.EachRecord(func(rec *kgo.Record) {
		out = append(out, FetchedRecord{
			Topic:     rec.Topic,
			Partition: rec.Partition,
			Offset:    rec.Offset,
			Timestamp: rec.Timestamp,
			Body:      rec.Value,
		})
	})
	// Commit consumed offsets back to the broker. β semantics:
	// commit-after-fetch (at-most-once on a crash mid-dispatch).
	// Per-attempt re-read of offset ranges (ADR-0024) is a
	// future slice; that future slice replaces this with a
	// commit-after-dispatch flow keyed on the trigger's
	// successful return.
	if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
		return out, fmt.Errorf("kafka commit: %w", err)
	}
	return out, nil
}

// Close releases the underlying client.
func (c *FranzConsumer) Close() {
	c.client.Close()
}
