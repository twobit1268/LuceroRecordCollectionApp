// Package events is the async consumer side of the system: activity-service
// subscribes to the same Pub/Sub topic collection-service publishes to,
// entirely decoupled from it — collection-service has no idea activity-service
// exists. Uses the official cloud.google.com/go/pubsub client, which works
// identically against the local emulator and real GCP Pub/Sub.
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

type Handler func(ctx context.Context, e RecordAddedEvent) error

type Subscriber struct {
	sub *pubsub.Subscription
}

func NewSubscriber(ctx context.Context, projectID, topicID, subscriptionID string) (*Subscriber, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("pubsub client: %w", err)
	}

	topic := client.Topic(topicID)
	if exists, err := topic.Exists(ctx); err != nil {
		return nil, fmt.Errorf("checking topic exists: %w", err)
	} else if !exists {
		if topic, err = client.CreateTopic(ctx, topicID); err != nil {
			return nil, fmt.Errorf("creating topic: %w", err)
		}
	}

	sub := client.Subscription(subscriptionID)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("checking subscription exists: %w", err)
	}
	if !exists {
		sub, err = client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{Topic: topic})
		if err != nil {
			return nil, fmt.Errorf("creating subscription: %w", err)
		}
	}

	return &Subscriber{sub: sub}, nil
}

// Listen blocks, pulling messages until ctx is canceled — meant to run in
// its own goroutine alongside the HTTP server.
func (s *Subscriber) Listen(ctx context.Context, handle Handler) error {
	return s.sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		var e RecordAddedEvent
		if err := json.Unmarshal(msg.Data, &e); err != nil {
			msg.Nack()
			return
		}
		if err := handle(ctx, e); err != nil {
			msg.Nack()
			return
		}
		msg.Ack()
	})
}
