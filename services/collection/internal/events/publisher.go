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

	topic := client.Topic(topicID)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking topic exists: %w", err)
	}
	if !exists {
		topic, err = client.CreateTopic(ctx, topicID)
		if err != nil {
			return nil, fmt.Errorf("creating topic: %w", err)
		}
	}

	return &PubSubPublisher{topic: topic}, nil
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
