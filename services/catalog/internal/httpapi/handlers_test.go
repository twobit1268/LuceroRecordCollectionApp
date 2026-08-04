package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/twobit1268/vinylvault/services/catalog/internal/model"
	"github.com/twobit1268/vinylvault/services/catalog/internal/store"
)

// fakeStore is an in-memory store.Store used so handler tests don't need a
// real Postgres instance — the same dependency-injection pattern discussed
// in interview prep for mocking in Go (small interface, fake implementation).
type fakeStore struct {
	mu      sync.Mutex
	records map[string]model.Record
}

func newFakeStore() *fakeStore {
	return &fakeStore{records: map[string]model.Record{}}
}

func (f *fakeStore) Create(_ context.Context, r model.Record) (model.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r.ID = uuid.NewString()
	f.records[r.ID] = r
	return r, nil
}

func (f *fakeStore) Get(_ context.Context, id string) (model.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.records[id]
	if !ok {
		return model.Record{}, store.ErrNotFound
	}
	return r, nil
}

func (f *fakeStore) List(_ context.Context) ([]model.Record, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.Record
	for _, r := range f.records {
		out = append(out, r)
	}
	return out, nil
}

func testServer() (*Server, *fakeStore) {
	fs := newFakeStore()
	return &Server{Store: fs, Log: slog.New(slog.DiscardHandler)}, fs
}

func TestHandleCreate(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid record",
			body:       `{"title":"OK Computer","artist":"Radiohead","genre":"Alternative Rock","year":1997}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "missing title",
			body:       `{"artist":"Radiohead","genre":"Alternative Rock","year":1997}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `not json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero year",
			body:       `{"title":"OK Computer","artist":"Radiohead","genre":"Alternative Rock","year":0}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := testServer()
			mux := NewMux(s)

			req := httptest.NewRequest(http.MethodPost, "/records", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestHandleGet_NotFound(t *testing.T) {
	s, _ := testServer()
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodGet, "/records/does-not-exist", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleCreateThenGet(t *testing.T) {
	s, _ := testServer()
	mux := NewMux(s)

	createReq := httptest.NewRequest(http.MethodPost, "/records", bytes.NewBufferString(
		`{"title":"Kind of Blue","artist":"Miles Davis","genre":"Jazz","year":1959}`,
	))
	createRec := httptest.NewRecorder()
	mux.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", createRec.Code)
	}

	var created model.Record
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected server-assigned ID, got empty string")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/records/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200", getRec.Code)
	}

	var got model.Record
	if err := json.NewDecoder(getRec.Body).Decode(&got); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if got.Title != "Kind of Blue" || got.Artist != "Miles Davis" {
		t.Fatalf("got %+v, want title/artist to match", got)
	}
}

func TestHandleList_EmptyIsNotNull(t *testing.T) {
	s, _ := testServer()
	mux := NewMux(s)

	req := httptest.NewRequest(http.MethodGet, "/records", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Body)
	if string(bytes.TrimSpace(body)) != "[]" {
		t.Fatalf("expected empty JSON array, got %q", body)
	}
}
