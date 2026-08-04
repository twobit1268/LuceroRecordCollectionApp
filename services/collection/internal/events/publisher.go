// Package events is the async side of this system's distributed-ness:
// after a record is added to a collection, an event is published to GCP
// Pub/Sub for activity-service to consume independently. The publisher
// wraps the official cloud.google.com/go/pubsub client, which works
// identically against the local Pub/Sub emulator (via PUBSUB_EMULATOR_HOST)
// and real GCP Pub/Sub — no code fork between environments.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RecordAddedEvent struct {
	CustomerID string    `json:"customerId"`
	RecordID   string    `json:"recordId"`
	AddedAt    time.Time `json:"addedAt"`
}

// Publisher is the interface handlers depend on, so tests can inject a fake
// instead of requiring a real (or emulated) Pub/Sub topic.
type Publisher interface {
	PublishRecordAdded(ctx context.Context, e RecordAddedEvent) error
}

type PubSubPublisher struct {
	topic *pubsub.Topic
}

func NewPubSubPublisher(ctx context.Context, projectID, topicID string) (*PubSubPublisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("pubsub client: %w", err)
	}

	topic, err := ensureTopic(ctx, client, topicID)
	if err != nil {
		return nil, err
	}

	return &PubSubPublisher{topic: topic}, nil
}

// ensureTopic creates the topic and tolerates AlreadyExists rather than
// check-then-create (Exists() then CreateTopic()) — that pattern races
// whenever more than one process/replica starts at the same time, which is
// exactly what happens here: collection-service and activity-service both
// try to set up the same topic on startup, and with multiple replicas (see
// k8s/ manifests, replicas: 2) so could two instances of the same service.
func ensureTopic(ctx context.Context, client *pubsub.Client, topicID string) (*pubsub.Topic, error) {
	topic, err := client.CreateTopic(ctx, topicID)
	if err == nil {
		return topic, nil
	}
	if status.Code(err) == codes.AlreadyExists {
		return client.Topic(topicID), nil
	}
	return nil, fmt.Errorf("creating topic: %w", err)
}

func (p *PubSubPublisher) PublishRecordAdded(ctx context.Context, e RecordAddedEvent) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	result := p.topic.Publish(ctx, &pubsub.Message{
		Data:       data,
		Attributes: map[string]string{"eventType": "collection.record_added"},
	})
	_, err = result.Get(ctx)
	return err
}
