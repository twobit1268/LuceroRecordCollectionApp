package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/twobit1268/vinylvault/services/activity/internal/store"
)

type fakeStore struct {
	feed   []store.Entry
	counts []store.GenreCount
}

func (f *fakeStore) RecordAdded(_ context.Context, e store.Entry) (store.Entry, error) {
	f.feed = append(f.feed, e)
	return e, nil
}

func (f *fakeStore) RecentFeed(_ context.Context, limit int) ([]store.Entry, error) {
	if len(f.feed) > limit {
		return f.feed[:limit], nil
	}
	return f.feed, nil
}

func (f *fakeStore) GenreCounts(_ context.Context) ([]store.GenreCount, error) { return f.counts, nil }

func TestHandleFeed_ReturnsEntries(t *testing.T) {
	fs := &fakeStore{feed: []store.Entry{
		{ID: "1", CustomerID: "c1", RecordID: "r1", Genre: "Jazz", AddedAt: time.Now()},
	}}
	s := &Server{Store: fs, Log: slog.New(slog.DiscardHandler)}
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodGet, "/activity/feed", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var entries []store.Entry
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 || entries[0].Genre != "Jazz" {
		t.Fatalf("got %+v, want 1 entry with genre Jazz", entries)
	}
}

func TestHandleFeed_RespectsLimit(t *testing.T) {
	fs := &fakeStore{feed: []store.Entry{{ID: "1"}, {ID: "2"}, {ID: "3"}}}
	s := &Server{Store: fs, Log: slog.New(slog.DiscardHandler)}
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodGet, "/activity/feed?limit=2", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var entries []store.Entry
	_ = json.NewDecoder(rec.Body).Decode(&entries)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 respecting limit param", len(entries))
	}
}

func TestHandleGenres_ReturnsCounts(t *testing.T) {
	fs := &fakeStore{counts: []store.GenreCount{{Genre: "Jazz", Count: 3}, {Genre: "Rock", Count: 1}}}
	s := &Server{Store: fs, Log: slog.New(slog.DiscardHandler)}
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodGet, "/activity/genres", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var counts []store.GenreCount
	if err := json.NewDecoder(rec.Body).Decode(&counts); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(counts) != 2 || counts[0].Genre != "Jazz" || counts[0].Count != 3 {
		t.Fatalf("got %+v, want Jazz:3, Rock:1", counts)
	}
}
