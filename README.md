# VinylVault

A distributed, Go-based, GCP-native microservice system for tracking a
customer's vinyl record collection. Built as a hands-on Go + GCP portfolio
piece — three independent services, real Postgres-per-service data
ownership, synchronous REST between services, and asynchronous GCP Pub/Sub
messaging, all containerized and verified end-to-end locally. A small React
+ TypeScript UI sits on top, driving the same distributed flow a real user
would.

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
- **No auth/customer-identity service** — `customerId` is a plain string in requests. No API gateway/BFF — the UI calls all three services directly by their own ports, and CORS is wide open (`Access-Control-Allow-Origin: *`) rather than scoped to the UI's actual origin. Both are clear, stated extension points for a local-dev-only tool, not production practice.
- **Pub/Sub publish is non-fatal** on failure in collection-service (the collection entry is already durably persisted even if the event publish fails) — a production system would harden this with a transactional outbox pattern; noted here rather than implemented, to keep scope honest.

## Setup

Requires Go 1.26+, Docker, and Docker Compose. No GCP account or `gcloud` CLI needed to run everything locally — see [GCP deployment](#gcp-deployment) below for what that path adds.

```bash
docker compose up --build
```

This starts 3 Postgres instances, the official GCP Pub/Sub emulator, all three services, and the web UI at `http://localhost:5173`. The `cloud.google.com/go/pubsub` client code is identical whether it's talking to the emulator (via `PUBSUB_EMULATOR_HOST`) or real GCP Pub/Sub — no code fork between environments.

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

## Web UI

`web/` is a small React + TypeScript (Vite) app with three tabs — Catalog (browse/create records), Collection (pick a customer, add records to their collection), Activity (the genre-enriched feed, so you can watch the async Pub/Sub path resolve in real time). It talks to all three services directly over plain `fetch()` calls; no state management library, no router (tab switching is just component state) — deliberately minimal.

Two ways to run it:

```bash
# Dev server, against services running however you started them:
cd web && npm install && npm run dev   # http://localhost:5173

# Or as part of the full containerized stack:
docker compose up --build   # nginx-served build, also at :5173
```

Verified end-to-end in a real browser, not just "it builds" — created a record, added it to a collection (exercising the sync catalog-service validation call), and confirmed it showed up in the Activity tab correctly enriched with genre after the async Pub/Sub round-trip, both against the dev server and the Dockerized nginx build.

## Testing

Four layers, all run in CI on every push:

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

**UI end-to-end tests (Playwright)** — `web/e2e/` is the browser-driven equivalent of the smoke test above: real Chromium, real clicks, against the actual containerized UI and backend. Uses the Page Object Model (`web/e2e/pages/`) — `CatalogPage`, `CollectionPage`, `ActivityPage` — the same pattern named on the resume ("Page Factory design patterns"), just applied to this stack. Six tests: record creation, required-field validation, collection add/remove, per-customer collection scoping, and a flagship test that walks the *entire* distributed flow through the UI — create a record, add it to a collection, then wait (via `ActivityPage.waitForCustomerActivity`, a click-refresh-and-retry helper, since the Activity tab doesn't auto-poll) for it to show up genre-enriched via the async Pub/Sub path. Run 4 times in a row locally with zero flakiness before this was considered done.

```bash
docker compose up --build -d
cd web && npx playwright install chromium   # first time only
npm run test:e2e
cd .. && docker compose down -v
```

**Performance testing (k6)** — `scripts/k6/` has two load tests, both with real pass/fail thresholds (k6 exits non-zero if they're not met — this is a genuine CI release gate, not a numbers-printer):
- `catalog-load-test.js` — sustained concurrent create/list traffic against catalog-service alone (10 VUs, 15s, p95 < 300ms, error rate < 1%).
- `collection-flow-load-test.js` — the full distributed flow under concurrent load: create in catalog-service → add to collection-service (sync validation + Pub/Sub publish) → list (5 VUs, 15s, p95 < 500ms, error rate < 1%). This is the one most likely to reveal cross-service contention that a single-service load test can't see.

```bash
docker compose up --build -d
k6 run scripts/k6/catalog-load-test.js              # brew install k6, or:
docker run --rm --network host -v $PWD/scripts/k6:/scripts -w /scripts grafana/k6 run /scripts/catalog-load-test.js
k6 run scripts/k6/collection-flow-load-test.js
docker compose down -v
```

Both scripts export `handleSummary` (`scripts/k6/lib/report.js`, shared by both) — each run produces a `<name>-summary.json` (raw metrics) and a `<name>-summary.html` (browsable report: total/failed requests, breached thresholds, per-metric charts), in addition to the usual colored text summary on stdout. Open the `.html` file directly in a browser. These are gitignored locally — see [Reports](#reports) below for where to find them from a CI run.

## Reports

Every CI run produces downloadable reports, not just pass/fail in the logs — from the run's page on GitHub ([Actions tab](https://github.com/twobit1268/LuceroRecordCollectionApp/actions)), scroll to the bottom for the **Artifacts** section:
- **`playwright-report`** — the full interactive Playwright HTML report (which test ran, timing, and on failure, screenshots/traces). Uploaded on every run, pass or fail, so you can browse a green run's report too, not just debug red ones. Unzip and open `index.html`, or `npx playwright show-report <unzipped-folder>`.
- **`k6-reports`** — the JSON + HTML report described above for both load tests, from the actual CI run's numbers (not your local machine's).

Artifacts are retained 7 days (Playwright) / 30 days (k6) per the workflow config, then auto-deleted by GitHub.

Runs automatically in CI as the `perf-test` job, after `smoke-test` passes (functional correctness confirmed first, then load).

## GCP deployment

Built GCP-ready but **not deployed** as part of this build — no `gcloud` CLI or GCP credentials were available in the environment this was built in, and creating real cloud resources needs the account owner's own setup and billing.

What's ready to deploy:
- **Dockerfiles** — multi-stage builds, Cloud Run– and GKE-compatible as-is.
- **`k8s/`** — Deployment + Service + ConfigMap per service, targeting GKE (matching how Bet365 actually runs their Verification platform). Image paths use an Artifact Registry placeholder (`REGION-docker.pkg.dev/PROJECT_ID/vinylvault/...`); `DATABASE_URL` is deliberately left out of the ConfigMaps and wired to a Kubernetes Secret instead, since it's never something to commit.
- **Pub/Sub code** — same client, same code path; production just points `GCP_PROJECT_ID` at a real project and omits `PUBSUB_EMULATOR_HOST`.

Roughly what deploying for real would involve: `gcloud sql instances create` (or Cloud SQL per service), `gcloud pubsub topics create collection-events`, `gcloud artifacts repositories create` + `docker push` per service image, then `kubectl apply -f k8s/` against a GKE cluster with the secrets pre-created.
