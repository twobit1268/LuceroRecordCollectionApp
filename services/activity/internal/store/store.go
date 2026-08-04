package store

import (
	"context"
	"time"
)

// Entry is one consumed collection.record_added event, enriched with the
// record's genre (fetched from catalog-service at consume time — see
// internal/catalogclient) so this service can serve genre stats without
// owning catalog data itself.
type Entry struct {
	ID         string    `json:"id"`
	CustomerID string    `json:"customerId"`
	RecordID   string    `json:"recordId"`
	Genre      string    `json:"genre"`
	AddedAt    time.Time `json:"addedAt"`
	ReceivedAt time.Time `json:"receivedAt"`
}

type GenreCount struct {
	Genre string `json:"genre"`
	Count int    `json:"count"`
}

type Store interface {
	RecordAdded(ctx context.Context, e Entry) (Entry, error)
	RecentFeed(ctx context.Context, limit int) ([]Entry, error)
	GenreCounts(ctx context.Context) ([]GenreCount, error)
}
