package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/twobit1268/vinylvault/services/catalog/internal/httpapi"
	"github.com/twobit1268/vinylvault/services/catalog/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := getenv("PORT", "8081")
	dsn := getenv("DATABASE_URL", "postgres://vinylvault:vinylvault@localhost:5432/catalog_db?sslmode=disable")

	ctx := context.Background()
	pgStore, err := store.NewPostgresStore(ctx, dsn)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	server := &httpapi.Server{Store: pgStore, Log: log}
	mux := httpapi.NewMux(server)

	log.Info("catalog-service listening", "port", port)
	if err := http.ListenAndServe(":"+port, httpapi.WithCORS(mux)); err != nil {
		log.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
