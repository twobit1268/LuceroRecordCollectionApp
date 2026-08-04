// Package consumer is the actual business logic of processing one
// collection.record_added event: enrich it with genre via catalog-service,
// then persist it. Kept separate from the Pub/Sub wiring (internal/events)
// so it's testable with plain fakes, no real subscription required.
package consumer

import (
	"context"
	"fmt"

	"github.com/twobit1268/vinylvault/services/activity/internal/catalogclient"
	"github.com/twobit1268/vinylvault/services/activity/internal/events"
	"github.com/twobit1268/vinylvault/services/activity/internal/store"
)

type Consumer struct {
	Store   store.Store
	Catalog catalogclient.RecordFetcher
}

// HandleEvent matches events.Handler's signature so it can be passed
// directly to Subscriber.Listen. Returning an error here means the caller
// (the Pub/Sub subscriber) will Nack the message for redelivery — a
// transient catalog-service outage becomes a retry, not a dropped event.
func (c *Consumer) HandleEvent(ctx context.Context, e events.RecordAddedEvent) error {
	record, err := c.Catalog.GetRecord(ctx, e.RecordID)
	if err != nil {
		return fmt.Errorf("enriching event with catalog data: %w", err)
	}

	_, err = c.Store.RecordAdded(ctx, store.Entry{
		CustomerID: e.CustomerID,
		RecordID:   e.RecordID,
		Genre:      record.Genre,
		AddedAt:    e.AddedAt,
	})
	if err != nil {
		return fmt.Errorf("persisting activity entry: %w", err)
	}
	return nil
}
