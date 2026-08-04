package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/twobit1268/vinylvault/services/activity/internal/catalogclient"
	"github.com/twobit1268/vinylvault/services/activity/internal/events"
	"github.com/twobit1268/vinylvault/services/activity/internal/store"
)

type fakeFetcher struct {
	summary catalogclient.RecordSummary
	err     error
}

func (f *fakeFetcher) GetRecord(_ context.Context, _ string) (catalogclient.RecordSummary, error) {
	return f.summary, f.err
}

type fakeStore struct {
	added []store.Entry
	err   error
}

func (f *fakeStore) RecordAdded(_ context.Context, e store.Entry) (store.Entry, error) {
	if f.err != nil {
		return store.Entry{}, f.err
	}
	f.added = append(f.added, e)
	return e, nil
}

func (f *fakeStore) RecentFeed(_ context.Context, _ int) ([]store.Entry, error) { return f.added, nil }

func (f *fakeStore) GenreCounts(_ context.Context) ([]store.GenreCount, error) { return nil, nil }

func TestHandleEvent_EnrichesWithGenreAndStores(t *testing.T) {
	fetcher := &fakeFetcher{summary: catalogclient.RecordSummary{ID: "rec-1", Genre: "Jazz"}}
	st := &fakeStore{}
	c := &Consumer{Store: st, Catalog: fetcher}

	err := c.HandleEvent(context.Background(), events.RecordAddedEvent{
		CustomerID: "cust-1",
		RecordID:   "rec-1",
		AddedAt:    time.Now(),
	})
	if err != nil {
		t.Fatalf("HandleEvent returned error: %v", err)
	}
	if len(st.added) != 1 {
		t.Fatalf("expected 1 stored entry, got %d", len(st.added))
	}
	if st.added[0].Genre != "Jazz" {
		t.Fatalf("expected genre enriched to Jazz, got %q", st.added[0].Genre)
	}
}

func TestHandleEvent_CatalogFetchFails_ReturnsErrorForRedelivery(t *testing.T) {
	fetcher := &fakeFetcher{err: errors.New("catalog-service unavailable")}
	st := &fakeStore{}
	c := &Consumer{Store: st, Catalog: fetcher}

	err := c.HandleEvent(context.Background(), events.RecordAddedEvent{CustomerID: "cust-1", RecordID: "rec-1"})
	if err == nil {
		t.Fatal("expected error when catalog-service fetch fails, got nil")
	}
	if len(st.added) != 0 {
		t.Fatalf("expected nothing stored on enrichment failure, got %d entries", len(st.added))
	}
}

func TestHandleEvent_StoreFails_ReturnsError(t *testing.T) {
	fetcher := &fakeFetcher{summary: catalogclient.RecordSummary{Genre: "Rock"}}
	st := &fakeStore{err: errors.New("db down")}
	c := &Consumer{Store: st, Catalog: fetcher}

	err := c.HandleEvent(context.Background(), events.RecordAddedEvent{CustomerID: "cust-1", RecordID: "rec-1"})
	if err == nil {
		t.Fatal("expected error when store fails, got nil")
	}
}
