// path: engine/internal/alerts/alerts_integration_test.go

//go:build integration

// Integration test for the alerts publisher against the local
// Pub/Sub emulator. Bring the substrate up first:
//
//	make up
//	cd engine && PUBSUB_EMULATOR_HOST=localhost:8085 \
//	  go test -tags integration ./internal/alerts/...
//
// The `make test-engine-integration` target sets PUBSUB_EMULATOR_HOST
// automatically; running the package directly requires it.
//
// The test exercises ADR-0010 §3.3 row "Pub/Sub publish/subscribe"
// (Yes) end-to-end: the publisher used by the engine binary is the
// same one this test wires; the message format is the canonical
// Event JSON encoding consumers must tolerate per ADR-0006 §4.

package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"

	"dq-platform/engine/internal/results"
)

const (
	integrationProjectID = "dq-local"
)

func pubsubTestClient(t *testing.T) *pubsub.Client {
	t.Helper()
	// The Pub/Sub SDK keys off PUBSUB_EMULATOR_HOST to route to a
	// non-production endpoint. The local Compose stack exposes the
	// emulator at localhost:8085; default to that if the operator
	// hasn't set the env var explicitly.
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		t.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cli, err := pubsub.NewClient(ctx, integrationProjectID)
	if err != nil {
		t.Skipf("integration: cannot create Pub/Sub client (is `make up` running?): %v", err)
	}
	return cli
}

func uniqueResourceName(t *testing.T, kind string) string {
	t.Helper()
	return fmt.Sprintf("itest-alerts-%s-%d", kind, time.Now().UnixNano())
}

// ensureTopicAndSubscription provisions a unique topic and
// subscription pair via the admin gRPC clients exposed on
// *pubsub.Client and registers a cleanup that deletes both.
func ensureTopicAndSubscription(t *testing.T, cli *pubsub.Client) (topicID, subID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	topicID = uniqueResourceName(t, "topic")
	subID = uniqueResourceName(t, "sub")
	topicName := fmt.Sprintf("projects/%s/topics/%s", integrationProjectID, topicID)
	subName := fmt.Sprintf("projects/%s/subscriptions/%s", integrationProjectID, subID)

	if _, err := cli.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName}); err != nil {
		t.Fatalf("create topic %s: %v", topicID, err)
	}
	if _, err := cli.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:               subName,
		Topic:              topicName,
		AckDeadlineSeconds: 10,
	}); err != nil {
		t.Fatalf("create subscription %s: %v", subID, err)
	}

	t.Cleanup(func() {
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = cli.SubscriptionAdminClient.DeleteSubscription(bg, &pubsubpb.DeleteSubscriptionRequest{Subscription: subName})
		_ = cli.TopicAdminClient.DeleteTopic(bg, &pubsubpb.DeleteTopicRequest{Topic: topicName})
	})

	return topicID, subID
}

func TestIntegration_PubSubPublisher_RoundTrip(t *testing.T) {
	cli := pubsubTestClient(t)
	defer cli.Close()

	topicID, subID := ensureTopicAndSubscription(t, cli)

	pub := NewPubSubPublisher(cli, topicID)
	defer pub.Close()

	severity := SeverityWarning
	execID := "exec-int-1"
	attID := "att-1"
	checkID := "row_count_positive"
	want := Event{
		ExecutionID: &execID,
		AttemptID:   &attID,
		Entity:      "customer",
		CheckID:     &checkID,
		Category:    CategoryDataQuality,
		Severity:    &severity,
		EventSource: SourceRunner,
		Result:      ptrCheck(results.ResultFail),
		RecordedAt:  time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pub.Publish(ctx, want); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// Receive the message via the Subscriber. Receive blocks until
	// the ctx is canceled; cancel as soon as the first message
	// arrives so the test does not wait for the receive deadline.
	var (
		received atomic.Int32
		gotData  []byte
	)
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer recvCancel()
	sub := cli.Subscriber(subID)
	receiveErr := sub.Receive(recvCtx, func(_ context.Context, m *pubsub.Message) {
		if received.CompareAndSwap(0, 1) {
			gotData = append([]byte{}, m.Data...)
			m.Ack()
			recvCancel()
		} else {
			m.Nack()
		}
	})
	if receiveErr != nil && recvCtx.Err() == nil {
		t.Fatalf("Receive: %v", receiveErr)
	}
	if received.Load() == 0 {
		t.Fatalf("no message received within deadline")
	}

	var got Event
	if err := json.Unmarshal(gotData, &got); err != nil {
		t.Fatalf("unmarshal received payload: %v (raw: %s)", err, string(gotData))
	}
	if got.Entity != want.Entity || got.Category != want.Category {
		t.Errorf("round-trip lost fields: got=%+v want=%+v", got, want)
	}
	if got.CheckID == nil || *got.CheckID != checkID {
		t.Errorf("CheckID round-trip failed: got=%v", got.CheckID)
	}
	if got.Severity == nil || *got.Severity != severity {
		t.Errorf("Severity round-trip failed: got=%v", got.Severity)
	}
	if got.Result == nil || *got.Result != results.ResultFail {
		t.Errorf("Result round-trip failed: got=%v", got.Result)
	}
}
