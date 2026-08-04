package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/twobit1268/vinylvault/services/catalog/internal/model"
	"github.com/twobit1268/vinylvault/services/catalog/internal/store"
)

type Server struct {
	Store store.Store
	Log   *slog.Logger
}

func NewMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /records", s.handleCreate)
	mux.HandleFunc("GET /records", s.handleList)
	mux.HandleFunc("GET /records/{id}", s.handleGet)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var rec model.Record
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := rec.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	created, err := s.Store.Create(r.Context(), rec)
	if err != nil {
		s.Log.Error("create record failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create record")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	records, err := s.Store.List(r.Context())
	if err != nil {
		s.Log.Error("list records failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list records")
		return
	}
	if records == nil {
		records = []model.Record{}
	}
	writeJSON(w, http.StatusOK, records)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, err := s.Store.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "record not found")
		return
	}
	if err != nil {
		s.Log.Error("get record failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get record")
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
