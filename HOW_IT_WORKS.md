# How it works

A step-by-step walkthrough of one full request — a customer adding a record
to their collection — tied to the actual code.

## 0. Where the request actually comes from

The `web/` React app is a thin client — `src/api.ts` is a set of plain `fetch()` wrappers, no framework-specific data layer. `CatalogView` calls `catalogApi.create()` on submit; `CollectionView` calls `collectionApi.add()`. Neither view knows or cares that adding a record triggers a synchronous validation call and an asynchronous Pub/Sub publish under the hood — from the UI's perspective it's just a `POST` that resolves. That gap (a simple UI action, a much less simple distributed operation behind it) is the whole point of steps 1–4 below.

## 1. The record exists in catalog-service

`POST /records` on catalog-service (`services/catalog/internal/httpapi/handlers.go`) validates the body (`model.Record.Validate()` — title/artist/genre required, year > 0), assigns a UUID, and inserts it into `catalog_db` via `PostgresStore.Create` (`services/catalog/internal/store/postgres.go`). Nothing distributed happens yet — this is a plain CRUD service.

## 2. A customer adds it to their collection

`POST /collections/{customerId}/records` on collection-service (`services/collection/internal/httpapi/handlers.go`, `handleAdd`) does four things in order:

1. **Validate the request shape** — `customerId` from the path, `recordId` from the body, via `model.Entry.Validate()`.
2. **Synchronous cross-service call** — `catalogclient.RecordValidator.RecordExists(ctx, recordId)` (`services/collection/internal/catalogclient/client.go`) makes a real HTTP `GET /records/{id}` to catalog-service. If catalog-service says 404, the request is rejected with `400` before anything is persisted — collection-service never trusts a client-supplied `recordId` on faith.
3. **Persist** — `store.Store.Add` inserts into `collection_db`.
4. **Publish an event, non-fatally** — `events.Publisher.PublishRecordAdded` sends a `collection.record_added` message to the GCP Pub/Sub topic `collection-events`. If this fails, the error is logged but the HTTP response still returns `201` — the entry is already durably persisted, so a Pub/Sub hiccup shouldn't fail the customer's request. (A production system would harden this further with a transactional outbox pattern; that's a stated extension, not implemented here.)

Both the `RecordValidator` and `Publisher` are small interfaces (`services/collection/internal/{catalogclient,events}`), so `handlers_test.go` exercises all of this — including the publish-fails-but-still-201 case — with fakes, no real network calls or Pub/Sub topic required.

## 3. activity-service consumes the event, independently

activity-service has no knowledge of collection-service's existence — it only knows about the Pub/Sub topic. `events.Subscriber.Listen` (`services/activity/internal/events/subscriber.go`) pulls messages in a background goroutine (started in `cmd/main.go`, separate from the HTTP server) and hands each decoded event to `consumer.Consumer.HandleEvent` (`services/activity/internal/consumer/consumer.go`).

`HandleEvent` does its own enrichment: the event only carries `customerId`/`recordId`/`addedAt`, so it calls catalog-service itself — `catalogclient.RecordFetcher.GetRecord` — to fetch the record's `genre`, then persists the enriched entry to `activity_db`. **If that catalog-service call fails, `HandleEvent` returns an error, which causes the Pub/Sub message to be Nacked and redelivered** — a transient catalog-service outage becomes a retry, not a silently dropped event. This is tested directly in `consumer_test.go` with fakes for both the store and the catalog client.

## 4. Reading the result back

`GET /activity/feed` and `GET /activity/genres` on activity-service just read from `activity_db` — `RecentFeed` (most recent entries, `LIMIT`-bounded) and `GenreCounts` (a `GROUP BY genre` aggregate). Both are proof that the async path actually worked: if an entry a customer just added shows up here, the whole chain — sync validation, persistence, publish, subscribe, enrichment, persistence again — really executed, not just individual services in isolation.

## A real concurrency bug this caught: the Pub/Sub topic/subscription race

This isn't a hypothetical "here's a bug I could imagine" — it actually happened while building this, and it's worth documenting in detail because it's a genuine distributed-systems failure mode, not a typo.

**Symptom:** `docker compose up` would sometimes bring up `activity-service` in a crash loop with:
```
{"level":"ERROR","msg":"failed to set up pubsub subscriber","error":"creating topic: rpc error: code = AlreadyExists desc = Topic already exists"}
```

**Root cause:** both `collection-service` and `activity-service` set up the same Pub/Sub topic on startup, using the obvious-looking pattern:
```go
exists, _ := topic.Exists(ctx)
if !exists {
    topic, err = client.CreateTopic(ctx, topicID)
}
```
This is a **check-then-act race** (a TOCTOU bug — time-of-check to time-of-use). When two processes start at roughly the same time, both call `Exists()`, both get `false`, both call `CreateTopic()` — one succeeds, the other gets an `AlreadyExists` error and, in the original code, treated that as fatal and crashed.

**Why it's more than a two-service coincidence:** the same race exists on the *subscription* side inside `activity-service` alone. The `k8s/activity-deployment.yaml` manifest sets `replicas: 2` — meaning in a real GKE deployment, two pods of the *same service* would race to create the same subscription on a rolling restart or scale-up, hitting this exact bug in production, not just at the two-service boundary this demo happens to have.

**Fix** (`services/collection/internal/events/publisher.go`, `services/activity/internal/events/subscriber.go`): replace check-then-create with **create-and-tolerate-`AlreadyExists`**:
```go
topic, err := client.CreateTopic(ctx, topicID)
if err == nil {
    return topic, nil
}
if status.Code(err) == codes.AlreadyExists {
    return client.Topic(topicID), nil // someone else made it — that's fine, use it
}
return nil, fmt.Errorf("creating topic: %w", err)
```
This collapses the race window to zero: there's no gap between checking and acting, because the "check" *is* the act, and the only two outcomes (I created it / someone else already did) are both handled as success.

**How it was actually caught:** not by code review, and not by the unit tests (which use fakes and never touch a real Pub/Sub topic) — by `scripts/smoke-test.sh` running against the *real* `docker compose` stack, exactly the class of bug that only shows up when independently-starting real processes actually race against each other. This is the same argument for why the CI `smoke-test` job exists as a separate stage from unit tests: unit tests with fakes prove the logic is right in isolation, but concurrency bugs between real processes require actually running them concurrently to find.

## Why three services, not one

The two communication patterns (sync REST validation, async Pub/Sub fan-out) only mean something if they cross a real process/deployment boundary — a single monolith calling its own functions wouldn't demonstrate anything about distributed-systems failure modes (a validator call that times out, a Pub/Sub message that needs a retry). Each service also owns its own Postgres database rather than sharing one schema — the actual "microservices" pattern, and the reason `docker-compose.yml` runs three separate Postgres containers instead of one shared instance.
