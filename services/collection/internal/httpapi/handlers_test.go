package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/twobit1268/vinylvault/services/collection/internal/events"
	"github.com/twobit1268/vinylvault/services/collection/internal/model"
	"github.com/twobit1268/vinylvault/services/collection/internal/store"
)

type fakeStore struct {
	mu      sync.Mutex
	entries map[string]model.Entry
}

func newFakeStore() *fakeStore { return &fakeStore{entries: map[string]model.Entry{}} }

func (f *fakeStore) Add(_ context.Context, e model.Entry) (model.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e.ID = uuid.NewString()
	f.entries[e.ID] = e
	return e, nil
}

func (f *fakeStore) ListByCustomer(_ context.Context, customerID string) ([]model.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.Entry
	for _, e := range f.entries {
		if e.CustomerID == customerID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (f *fakeStore) Remove(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.entries[id]; !ok {
		return store.ErrNotFound
	}
	delete(f.entries, id)
	return nil
}

// fakeValidator lets tests control whether catalog-service "sees" a record
// as existing, without making a real HTTP call.
type fakeValidator struct {
	exists bool
	err    error
}

func (f *fakeValidator) RecordExists(_ context.Context, _ string) (bool, error) {
	return f.exists, f.err
}

// fakePublisher records what would have been published, so tests can assert
// on it without a real (or emulated) Pub/Sub topic.
type fakePublisher struct {
	mu        sync.Mutex
	published []events.RecordAddedEvent
	err       error
}

func (f *fakePublisher) PublishRecordAdded(_ context.Context, e events.RecordAddedEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.published = append(f.published, e)
	return f.err
}

func testServer(exists bool) (*Server, *fakeStore, *fakePublisher) {
	fs := newFakeStore()
	fp := &fakePublisher{}
	s := &Server{
		Store:     fs,
		Validator: &fakeValidator{exists: exists},
		Publisher: fp,
		Log:       slog.New(slog.DiscardHandler),
	}
	return s, fs, fp
}

func TestHandleAdd_RecordExists_PublishesEvent(t *testing.T) {
	s, _, fp := testServer(true)
	mux := NewMux(s)

	body := `{"recordId":"rec-123"}`
	req := httptest.NewRequest(http.MethodPost, "/collections/cust-1/records", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var entry model.Entry
	if err := json.NewDecoder(rec.Body).Decode(&entry); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if entry.CustomerID != "cust-1" || entry.RecordID != "rec-123" {
		t.Fatalf("got %+v, want customerId=cust-1 recordId=rec-123", entry)
	}

	if len(fp.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(fp.published))
	}
	if fp.published[0].CustomerID != "cust-1" || fp.published[0].RecordID != "rec-123" {
		t.Fatalf("published event = %+v, want matching customer/record", fp.published[0])
	}
}

func TestHandleAdd_RecordDoesNotExist_Returns400(t *testing.T) {
	s, _, fp := testServer(false)
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodPost, "/collections/cust-1/records", bytes.NewBufferString(`{"recordId":"ghost"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if len(fp.published) != 0 {
		t.Fatalf("expected no event published when record doesn't exist, got %d", len(fp.published))
	}
}

func TestHandleAdd_ValidatorError_Returns502(t *testing.T) {
	s, _, _ := testServer(false)
	s.Validator = &fakeValidator{err: context.DeadlineExceeded}
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodPost, "/collections/cust-1/records", bytes.NewBufferString(`{"recordId":"rec-1"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 when catalog-service call fails", rec.Code)
	}
}

func TestHandleAdd_PublishFails_StillReturns201(t *testing.T) {
	// Publishing is deliberately non-fatal — the entry is already persisted.
	s, _, fp := testServer(true)
	fp.err = context.DeadlineExceeded
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodPost, "/collections/cust-1/records", bytes.NewBufferString(`{"recordId":"rec-1"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 even when publish fails", rec.Code)
	}
}

func TestHandleList_FiltersByCustomer(t *testing.T) {
	s, fs, _ := testServer(true)
	mux := NewMux(s)

	_, _ = fs.Add(context.Background(), model.Entry{CustomerID: "cust-1", RecordID: "rec-a"})
	_, _ = fs.Add(context.Background(), model.Entry{CustomerID: "cust-2", RecordID: "rec-b"})

	req := httptest.NewRequest(http.MethodGet, "/collections/cust-1/records", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var entries []model.Entry
	if err := json.NewDecoder(rec.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 1 || entries[0].CustomerID != "cust-1" {
		t.Fatalf("got %+v, want exactly 1 entry for cust-1", entries)
	}
}

func TestHandleRemove_NotFound(t *testing.T) {
	s, _, _ := testServer(true)
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodDelete, "/collections/cust-1/records/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
