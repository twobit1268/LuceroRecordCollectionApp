package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/twobit1268/vinylvault/services/activity/internal/catalogclient"
	"github.com/twobit1268/vinylvault/services/activity/internal/consumer"
	"github.com/twobit1268/vinylvault/services/activity/internal/events"
	"github.com/twobit1268/vinylvault/services/activity/internal/httpapi"
	"github.com/twobit1268/vinylvault/services/activity/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	port := getenv("PORT", "8083")
	dsn := getenv("DATABASE_URL", "postgres://vinylvault:vinylvault@localhost:5434/activity_db?sslmode=disable")
	catalogURL := getenv("CATALOG_SERVICE_URL", "http://localhost:8081")
	gcpProject := getenv("GCP_PROJECT_ID", "vinylvault-local")
	topicID := getenv("PUBSUB_TOPIC", "collection-events")
	subID := getenv("PUBSUB_SUBSCRIPTION", "activity-service-sub")

	ctx := context.Background()

	pgStore, err := store.NewPostgresStore(ctx, dsn)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	sub, err := events.NewSubscriber(ctx, gcpProject, topicID, subID)
	if err != nil {
		log.Error("failed to set up pubsub subscriber", "error", err)
		os.Exit(1)
	}

	c := &consumer.Consumer{Store: pgStore, Catalog: catalogclient.New(catalogURL)}

	// The Pub/Sub subscription runs independently of the HTTP server — this
	// service's core job is consuming events, not answering requests; the
	// HTTP API just exposes what's been consumed so far.
	go func() {
		log.Info("starting pubsub subscriber", "topic", topicID, "subscription", subID)
		if err := sub.Listen(ctx, c.HandleEvent); err != nil {
			log.Error("pubsub subscriber stopped", "error", err)
		}
	}()

	server := &httpapi.Server{Store: pgStore, Log: log}
	mux := httpapi.NewMux(server)

	log.Info("activity-service listening", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
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
