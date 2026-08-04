# VinylVault

A distributed, Go-based, GCP-native microservice system for tracking a
customer's vinyl record collection. Built as a hands-on Go + GCP portfolio
piece — three independent services, real Postgres-per-service data
ownership, synchronous REST between services, and asynchronous GCP Pub/Sub
messaging, all containerized and verified end-to-end locally.

See [HOW_IT_WORKS.md](./HOW_IT_WORKS.md) for a step-by-step walkthrough of
the full request flow across all three services.

## Architecture

```
catalog-service (:8081) ──owns──> catalog_db

collection-service (:8082) ──owns──> collection_db
  │
  ├──(sync REST)──> GET /records/{id} on catalog-service   [validate before adding]
  │
  └──(async publish)──> Pub/Sub topic "collection-events"

activity-service (:8083) ──owns──> activity_db
  │
  ├──(async subscribe)──< Pub/Sub topic "collection-events"
  │
  └──(sync REST)──> GET /records/{id} on catalog-service   [enrich event with genre]
```

Two communication patterns, both real: collection-service validates synchronously against catalog-service before persisting; activity-service consumes asynchronously via Pub/Sub, fully decoupled from collection-service, then does its own synchronous enrichment call back to catalog-service.

- **catalog-service** — the master list of records (`title`, `artist`, `genre`, `year`). Plain REST CRUD. Owns `catalog_db`.
- **collection-service** — which records a given `customerId` owns. On add: calls catalog-service synchronously over REST to confirm the record exists, persists the entry, then publishes a `collection.record_added` event to Pub/Sub. Owns `collection_db`.
- **activity-service** — subscribes to that Pub/Sub topic independently (collection-service has no idea it exists), enriches each event with genre by calling catalog-service itself, and serves a recent-activity feed + genre-count stats. Owns `activity_db`.

Each service owns its own Postgres database — no shared schema, no direct cross-service DB access. This is deliberate: it's the actual microservices pattern (vs. three processes sharing one database), and a real interview talking point about service boundaries.

## Deliberate scope decisions

- **REST/JSON everywhere**, not gRPC — keeps this testable with the same tools already in use elsewhere (Postman/REST Assured) and keeps scope reasonable. gRPC for internal calls is a natural extension.
- **No auth/customer-identity service** — `customerId` is a plain string in requests. No API gateway/BFF — each service exposes its own port directly. Both are clear, stated extension points, not oversights.
- **No frontend** — this is the backend distributed system that was actually requested.
- **Pub/Sub publish is non-fatal** on failure in collection-service (the collection entry is already durably persisted even if the event publish fails) — a production system would harden this with a transactional outbox pattern; noted here rather than implemented, to keep scope honest.

## Setup

Requires Go 1.26+, Docker, and Docker Compose. No GCP account or `gcloud` CLI needed to run everything locally — see [GCP deployment](#gcp-deployment) below for what that path adds.

```bash
docker compose up --build
```

This starts 3 Postgres instances, the official GCP Pub/Sub emulator, and all three services. The `cloud.google.com/go/pubsub` client code is identical whether it's talking to the emulator (via `PUBSUB_EMULATOR_HOST`) or real GCP Pub/Sub — no code fork between environments.

## API reference

**catalog-service** (`:8081`)
- `POST /records` — `{"title","artist","genre","year"}` → `201` with the created record (server-assigned `id`)
- `GET /records` — list all records
- `GET /records/{id}` — get one record (also used internally by collection-service and activity-service)

**collection-service** (`:8082`)
- `POST /collections/{customerId}/records` — `{"recordId"}` → validates against catalog-service, persists, publishes event
- `GET /collections/{customerId}/records` — list a customer's collection
- `DELETE /collections/{customerId}/records/{entryId}` — remove an entry

**activity-service** (`:8083`)
- `GET /activity/feed?limit=20` — recent collection activity across all customers, enriched with genre
- `GET /activity/genres` — genre counts across all consumed events

## Testing

Two layers, both run in CI on every push:

**Unit tests** — table-driven Go tests per service, using fakes for dependencies (store, HTTP clients, Pub/Sub publisher) — no live Postgres or Pub/Sub required:

```bash
cd services/catalog && go test ./...
cd services/collection && go test ./...
cd services/activity && go test ./...
```

**End-to-end smoke test** — `scripts/smoke-test.sh` runs against a real `docker compose up` stack and proves the actual distributed flow works, not just that each service passes its own unit tests in isolation: create a record, reject an invalid `recordId` (400), add to a collection (sync validation), wait for the async Pub/Sub event to be consumed and enriched with genre, verify genre-count stats, delete + confirm idempotent 404. This caught a real bug during development — a race condition where collection-service and activity-service (and activity-service's own replicas) could crash on startup racing to create the same Pub/Sub topic/subscription; fixed with a create-and-tolerate-`AlreadyExists` pattern instead of check-then-create.

```bash
docker compose up --build -d
./scripts/smoke-test.sh
docker compose down -v
```

## GCP deployment

Built GCP-ready but **not deployed** as part of this build — no `gcloud` CLI or GCP credentials were available in the environment this was built in, and creating real cloud resources needs the account owner's own setup and billing.

What's ready to deploy:
- **Dockerfiles** — multi-stage builds, Cloud Run– and GKE-compatible as-is.
- **`k8s/`** — Deployment + Service + ConfigMap per service, targeting GKE (matching how Bet365 actually runs their Verification platform). Image paths use an Artifact Registry placeholder (`REGION-docker.pkg.dev/PROJECT_ID/vinylvault/...`); `DATABASE_URL` is deliberately left out of the ConfigMaps and wired to a Kubernetes Secret instead, since it's never something to commit.
- **Pub/Sub code** — same client, same code path; production just points `GCP_PROJECT_ID` at a real project and omits `PUBSUB_EMULATOR_HOST`.

Roughly what deploying for real would involve: `gcloud sql instances create` (or Cloud SQL per service), `gcloud pubsub topics create collection-events`, `gcloud artifacts repositories create` + `docker push` per service image, then `kubectl apply -f k8s/` against a GKE cluster with the secrets pre-created.
