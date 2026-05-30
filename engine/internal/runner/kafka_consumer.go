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
// Auto-commit is disabled; the runner is the sole commit
// authority per ADR-0058 §Clause 3. PollFetches returns records
// without committing; the runner calls Commit after each
// successful dispatcher return per ADR-0058 §Clause 2. The
// composed delivery semantic is at-least-once with canonical-
// view collapse at the execution_id boundary per ADR-0003 §1
// (append-only writes) + §2 (`dq_executions_current` collapses
// attempts per execution_id to the latest recorded_at).
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
		// Auto-commit disabled; the runner is the sole commit
		// authority per ADR-0058 §Clause 3.
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
// surfaced as a single aggregated error. Per ADR-0058 §Clause 3,
// PollFetches does NOT commit offsets; the runner calls Commit
// after each successful dispatcher return.
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
	return out, nil
}

// Commit advances the consumer-group committed offset to cover
// the passed records (high-water mark per partition) per
// ADR-0058 §Clause 3. The implementation translates each
// FetchedRecord to a *kgo.Record carrying topic/partition/offset,
// calls MarkCommitRecords (the client tracks the high-water mark
// per partition internally), then CommitMarkedOffsets (flushes
// to the broker synchronously). Empty input returns nil without
// an RPC.
func (c *FranzConsumer) Commit(ctx context.Context, records []FetchedRecord) error {
	if len(records) == 0 {
		return nil
	}
	kgoRecs := make([]*kgo.Record, 0, len(records))
	for _, r := range records {
		kgoRecs = append(kgoRecs, &kgo.Record{
			Topic:     r.Topic,
			Partition: r.Partition,
			Offset:    r.Offset,
		})
	}
	c.client.MarkCommitRecords(kgoRecs...)
	if err := c.client.CommitMarkedOffsets(ctx); err != nil {
		return fmt.Errorf("kafka commit: %w", err)
	}
	return nil
}

// Close releases the underlying client.
func (c *FranzConsumer) Close() {
	c.client.Close()
}
