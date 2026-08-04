package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/twobit1268/vinylvault/services/collection/internal/catalogclient"
	"github.com/twobit1268/vinylvault/services/collection/internal/events"
	"github.com/twobit1268/vinylvault/services/collection/internal/httpapi"
	"github.com/twobit1268/vinylvault/services/collection/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := getenv("PORT", "8082")
	dsn := getenv("DATABASE_URL", "postgres://vinylvault:vinylvault@localhost:5433/collection_db?sslmode=disable")
	catalogURL := getenv("CATALOG_SERVICE_URL", "http://localhost:8081")
	gcpProject := getenv("GCP_PROJECT_ID", "vinylvault-local")
	topicID := getenv("PUBSUB_TOPIC", "collection-events")

	ctx := context.Background()

	pgStore, err := store.NewPostgresStore(ctx, dsn)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	publisher, err := events.NewPubSubPublisher(ctx, gcpProject, topicID)
	if err != nil {
		log.Error("failed to set up pubsub publisher", "error", err)
		os.Exit(1)
	}

	server := &httpapi.Server{
		Store:     pgStore,
		Validator: catalogclient.New(catalogURL),
		Publisher: publisher,
		Log:       log,
	}
	mux := httpapi.NewMux(server)

	log.Info("collection-service listening", "port", port, "catalogServiceURL", catalogURL)
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
