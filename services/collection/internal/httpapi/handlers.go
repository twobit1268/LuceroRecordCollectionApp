package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/twobit1268/vinylvault/services/collection/internal/catalogclient"
	"github.com/twobit1268/vinylvault/services/collection/internal/events"
	"github.com/twobit1268/vinylvault/services/collection/internal/model"
	"github.com/twobit1268/vinylvault/services/collection/internal/store"
)

type Server struct {
	Store     store.Store
	Validator catalogclient.RecordValidator
	Publisher events.Publisher
	Log       *slog.Logger
}

func NewMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /collections/{customerId}/records", s.handleAdd)
	mux.HandleFunc("GET /collections/{customerId}/records", s.handleList)
	mux.HandleFunc("DELETE /collections/{customerId}/records/{entryId}", s.handleRemove)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type addRequest struct {
	RecordID string `json:"recordId"`
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customerId")

	var req addRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	entry := model.Entry{CustomerID: customerID, RecordID: req.RecordID}
	if err := entry.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Synchronous cross-service call: don't trust a client-supplied
	// recordId, confirm it actually exists in catalog-service first.
	exists, err := s.Validator.RecordExists(r.Context(), req.RecordID)
	if err != nil {
		s.Log.Error("catalog validation call failed", "error", err)
		writeError(w, http.StatusBadGateway, "could not validate record against catalog-service")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "recordId does not exist in catalog")
		return
	}

	created, err := s.Store.Add(r.Context(), entry)
	if err != nil {
		s.Log.Error("add entry failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to add record to collection")
		return
	}

	// Async event publish. Deliberately non-fatal: the collection entry is
	// already durably persisted, so a Pub/Sub hiccup shouldn't fail the
	// customer's request — it's logged so it's visible, and a production
	// system would harden this further with a transactional outbox pattern
	// (noted in the README as an extension, not implemented here).
	if err := s.Publisher.PublishRecordAdded(r.Context(), events.RecordAddedEvent{
		CustomerID: created.CustomerID,
		RecordID:   created.RecordID,
		AddedAt:    created.AddedAt,
	}); err != nil {
		s.Log.Error("failed to publish record_added event", "error", err, "entryId", created.ID)
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	customerID := r.PathValue("customerId")
	entries, err := s.Store.ListByCustomer(r.Context(), customerID)
	if err != nil {
		s.Log.Error("list entries failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list collection")
		return
	}
	if entries == nil {
		entries = []model.Entry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleRemove(w http.ResponseWriter, r *http.Request) {
	entryID := r.PathValue("entryId")
	err := s.Store.Remove(r.Context(), entryID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "entry not found")
		return
	}
	if err != nil {
		s.Log.Error("remove entry failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to remove entry")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
