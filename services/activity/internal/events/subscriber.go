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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	topic, err := ensureTopic(ctx, client, topicID)
	if err != nil {
		return nil, err
	}

	sub, err := ensureSubscription(ctx, client, topic, subscriptionID)
	if err != nil {
		return nil, err
	}

	return &Subscriber{sub: sub}, nil
}

// ensureTopic and ensureSubscription create-and-tolerate-AlreadyExists
// rather than check-then-create — check-then-create races whenever more
// than one process starts at once, which is exactly the case here:
// collection-service and activity-service both set up the same topic on
// startup, and activity-service itself runs multiple replicas (see k8s/
// manifests, replicas: 2), so two of its own instances also race on the
// same subscription.
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

func ensureSubscription(ctx context.Context, client *pubsub.Client, topic *pubsub.Topic, subscriptionID string) (*pubsub.Subscription, error) {
	sub, err := client.CreateSubscription(ctx, subscriptionID, pubsub.SubscriptionConfig{Topic: topic})
	if err == nil {
		return sub, nil
	}
	if status.Code(err) == codes.AlreadyExists {
		return client.Subscription(subscriptionID), nil
	}
	return nil, fmt.Errorf("creating subscription: %w", err)
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
