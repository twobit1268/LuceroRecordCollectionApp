package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/twobit1268/vinylvault/services/activity/internal/store"
)

type Server struct {
	Store store.Store
	Log   *slog.Logger
}

func NewMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /activity/feed", s.handleFeed)
	mux.HandleFunc("GET /activity/genres", s.handleGenres)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, err := s.Store.RecentFeed(r.Context(), limit)
	if err != nil {
		s.Log.Error("recent feed failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load activity feed")
		return
	}
	if entries == nil {
		entries = []store.Entry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleGenres(w http.ResponseWriter, r *http.Request) {
	counts, err := s.Store.GenreCounts(r.Context())
	if err != nil {
		s.Log.Error("genre counts failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to load genre counts")
		return
	}
	if counts == nil {
		counts = []store.GenreCount{}
	}
	writeJSON(w, http.StatusOK, counts)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
