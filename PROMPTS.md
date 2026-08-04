# VinylVault — Reverse-Engineered Build Prompts

Educated reconstruction of the prompts used to build this project, ordered
as a real build would proceed. Each prompt is based on analysis of the actual
code — package names, interface shapes, comments, test patterns, and design
decisions in the source. Evidence notes explain what in the code supports
each reconstruction.

---

## Phase 1 — Architecture & Scaffolding

### Prompt 1 — System Design

> I'm building a portfolio project in Go to demonstrate microservices patterns
> for a job application targeting companies that run distributed systems on GCP.
> Design a system with three independent services — a catalog service (master
> list of vinyl records), a collection service (which records a customer owns),
> and an activity service (event log). Each service must own its own Postgres
> database. The catalog and collection services communicate synchronously over
> REST. The collection and activity services communicate asynchronously via GCP
> Pub/Sub. Show the architecture diagram and explain why each design decision
> exists.

**Evidence:** Architecture described in README and HOW_IT_WORKS.md matches
this exactly. "Two communication patterns, both real" appears verbatim in the
README — suggests it was an explicit requirement, not an emergent decision.

---

### Prompt 2 — Go Workspace Scaffolding

> Scaffold the three Go services as a Go workspace (`go.work`). Each service
> should be in `services/catalog`, `services/collection`, `services/activity`
> with its own `go.mod`. Follow standard Go project layout: `cmd/` for main,
> `internal/` for everything else, with subdirectories for `httpapi`, `store`,
> `model`, and `events` where needed.

**Evidence:** `go.work` file exists. All three services follow identical
directory layout with the same package names — strongly suggests a single
scaffolding prompt rather than three separate ones.

---

## Phase 2 — Catalog Service

### Prompt 3 — Catalog Handlers & Store

> Build the catalog service in Go. It needs four endpoints: POST /records,
> GET /records, GET /records/{id}, DELETE /records/{id}. The Record model has
> title, artist, genre (string), and year (int). Use `http.ServeMux` with Go
> 1.22+ method+path routing syntax. Write a `Store` interface with Create, Get,
> List, Delete methods. Implement it with Postgres using `database/sql`. Validate
> that title, artist, and genre are non-empty and year is greater than zero —
> return 400 with a descriptive error if not. Write centralized `writeJSON` and
> `writeError` helpers so handlers stay clean.

**Evidence:** `handlers.go` is 88 lines with exactly these four routes. The
`Server` struct, `Store` interface, and `writeJSON`/`writeError` helpers match
precisely. Go 1.22+ method+path syntax (`"GET /records/{id}"`) is used
throughout.

---

### Prompt 4 — Catalog Unit Tests

> Write table-driven unit tests for the catalog service handlers using a fake
> `Store` implementation (not a real database) so tests run with no external
> dependencies. Test the happy path for each endpoint, the 400 validation path,
> and the 404 case for GET /records/{id}.

**Evidence:** Code comment in `catalogclient/client.go`: "Interface so tests
can inject a fake instead of making real HTTP calls." Table-driven tests are
idiomatic Go and confirmed by README.

---

## Phase 3 — Collection Service

### Prompt 5 — Collection Handler + Sync Validation

> Build the collection service. It tracks which records a customer owns. The
> Entry model has customerId (string from URL path), recordId (string from
> request body), and an auto-assigned id and addedAt timestamp. Before
> persisting a new entry, synchronously call catalog-service's
> `GET /records/{id}` to confirm the record exists — if catalog-service returns
> 404, reject the request with 400. Never trust the client-supplied recordId on
> faith. Model this as a `RecordValidator` interface with a single
> `RecordExists(ctx, id) bool` method, with a real HTTP implementation and a
> fake for tests.

**Evidence:** `handlers.go` comment: "Synchronous cross-service call: don't
trust a client-supplied recordId." `RecordValidator` interface exists in
`catalogclient/client.go`. `handlers.go` is 128 lines — significantly larger
than catalog's 88, consistent with the added complexity.

---

### Prompt 6 — Non-Fatal Pub/Sub Publish

> After persisting the collection entry, publish a `collection.record_added`
> event to a GCP Pub/Sub topic called `collection-events`. The payload should
> include customerId, recordId, and addedAt. This publish must be non-fatal —
> if Pub/Sub fails, log the error but still return 201 to the client because
> the entry is already durably saved. Model the publisher as a `Publisher`
> interface so tests can inject a fake. Note in the code that a transactional
> outbox pattern would be the production-hardening of this.

**Evidence:** `publisher.go` and `handlers.go` both contain comments about
non-fatal publish and the transactional outbox pattern. The exact phrase
"transactional outbox" appears in both code comments and the README.

---

### Prompt 7 — Race-Safe Topic Setup

> Write the Pub/Sub topic setup code. The naive check-then-create pattern
> (`if !exists { create }`) is a race condition when multiple services start
> concurrently. Use create-and-tolerate-AlreadyExists instead: always call
> `CreateTopic`, and if the error code is `AlreadyExists`, return the existing
> topic handle as success. Explain the TOCTOU race in a code comment.

**Evidence:** `publisher.go` has an `ensureTopic()` function with exactly
this pattern and a comment explaining the race. The TOCTOU term appears in
`HOW_IT_WORKS.md`. This level of specificity suggests the bug was hit in
practice and a follow-up prompt asked for the fix — see the closing note.

---

## Phase 4 — Activity Service

### Prompt 8 — Async Subscriber + Event Enrichment

> Build the activity service. It subscribes to the `collection-events` Pub/Sub
> topic and knows nothing about collection-service — fully decoupled. When it
> receives a `collection.record_added` event, call catalog-service's
> `GET /records/{id}` to fetch the record's genre (enrichment), then persist
> the enriched entry to `activity_db`. If the catalog-service call fails, Nack
> the message so Pub/Sub redelivers it. Start the subscriber in a background
> goroutine in main, separate from the HTTP server.

**Evidence:** `subscriber.go` and `consumer.go` implement this exactly.
Comment in `subscriber.go`: "Collection-service and activity-service are
entirely decoupled." Nack-on-error is explicit with a comment explaining
retry semantics.

---

### Prompt 9 — Activity Feed Endpoints

> Add two HTTP endpoints to the activity service: `GET /activity/feed?limit=N`
> for recent enriched activity across all customers, and `GET /activity/genres`
> for a GROUP BY genre count. These are the observable proof that the full
> async chain worked.

**Evidence:** Both endpoints exist. `HOW_IT_WORKS.md` explicitly calls them
"proof that the async path actually worked" — that framing suggests it was
in the original prompt.

---

### Prompt 10 — Race-Safe Subscription Setup

> Apply the same create-and-tolerate-AlreadyExists pattern to activity-service
> for both the topic AND the subscription setup. Note in comments that
> activity-service's k8s deployment runs 2 replicas, so two pods of the same
> service race to create the same subscription on startup — this is a
> production GKE failure mode, not just a local dev curiosity.

**Evidence:** `subscriber.go` has `ensureSubscription()` with the same
pattern. The k8s deployment manifest sets `replicas: 2`. `HOW_IT_WORKS.md`
documents the multi-replica angle explicitly.

---

## Phase 5 — React UI

### Prompt 11 — Three-Tab Vite/React Frontend

> Build a minimal React + TypeScript (Vite) UI with three tabs: Catalog (list
> and create records), Collection (pick a customer ID, add/remove records from
> their collection), and Activity (the enriched feed with a Refresh button and
> genre counts). No router, no state management library — tab switching is
> plain component state. All service calls go through a typed `api.ts` module
> with plain `fetch()` wrappers and a shared `asJson<T>` error helper. Service
> base URLs come from Vite env vars with localhost fallbacks so the same build
> works in docker-compose and plain local dev.

**Evidence:** `api.ts` is 91 lines with this exact structure. Four TypeScript
interfaces (`AlbumRecord`, `CollectionEntry`, `ActivityEntry`, `GenreCount`)
and three API modules. README explicitly notes "no state management library,
no router — deliberately minimal."

---

## Phase 6 — Testing

### Prompt 12 — Bash Smoke Test

> Write a bash smoke test (`scripts/smoke-test.sh`) that runs against a real
> `docker compose up` stack and proves the full distributed flow end-to-end.
> Steps: wait for all services healthy, create a record, try to add a
> nonexistent recordId (expect 400), add the real record to a collection, poll
> for activity-service to consume and enrich the event, verify genre appears in
> `/activity/genres`, delete the entry, verify 404 on redelete. Exit non-zero
> on any failure.

**Evidence:** `smoke-test.sh` is 105 lines and follows this exact sequence.
A `json_field` helper function is used for JSON parsing — a detail that
points to iterative refinement of the bash script.

---

### Prompt 13 — Playwright E2E Test

> Write a Playwright end-to-end test that drives the same flow through the
> actual UI. Use the Page Object Model: `CatalogPage`, `CollectionPage`,
> `ActivityPage`. The flagship test should create a record, add it to a
> collection, then wait for it to appear genre-enriched in Activity. Since the
> Activity tab only fetches once on mount and doesn't auto-poll, implement a
> `waitForCustomerActivity` helper on `ActivityPage` that clicks the Refresh
> button in a retry loop.

**Evidence:** `end-to-end-flow.spec.ts` is 38 lines and matches this exactly.
The retry-click pattern is documented in `HOW_IT_WORKS.md` as a deliberate
design decision — the framing ("the Activity tab fetches once on mount and
doesn't poll on its own, so a passive waitFor would just time out") reads
like prompt language that made it into the docs.

---

### Prompt 14 — k6 Load Tests

> Write two k6 load tests. First: catalog-service alone, 10 VUs, 15s,
> p95 < 300ms, error rate < 1%. Second: full distributed flow under concurrent
> load — create in catalog, add to collection (sync validation + Pub/Sub
> publish), list collection, 5 VUs, 15s, p95 < 500ms. Both must exit non-zero
> if thresholds aren't met — real CI gates, not number printers. Write a shared
> `buildSummary()` helper that outputs JSON and HTML reports.

**Evidence:** `catalog-load-test.js` matches these thresholds exactly. "Real
CI gates, not number printers" appears in the README verbatim.

---

## Phase 7 — Containerization & Deployment

### Prompt 15 — Docker Compose

> Write a `docker-compose.yml` that starts 3 Postgres instances (one per
> service, on different ports), the official GCP Pub/Sub emulator, all three
> Go services, and the Vite web UI. Wire startup dependencies so catalog-service
> waits for its Postgres healthcheck, and collection/activity wait for the
> Pub/Sub emulator. Use `PUBSUB_EMULATOR_HOST` so the Pub/Sub client code is
> identical in local and production — no code fork between environments.

**Evidence:** `docker-compose.yml` is 112 lines with exactly three Postgres
instances on ports 5532–5534. "No code fork" language appears in the README.

---

### Prompt 16 — Kubernetes Manifests

> Write Kubernetes manifests in `k8s/` for GKE deployment: Deployment +
> Service + ConfigMap per service. Set activity-service to 2 replicas. Use
> Artifact Registry placeholder image paths. Put `DATABASE_URL` in a Secret
> reference, not in the ConfigMap. Document what `gcloud` commands would be
> needed to deploy for real.

**Evidence:** `k8s/` directory exists with per-service manifests.
`activity-deployment.yaml` has `replicas: 2`. Secret reference for
`DATABASE_URL` is in the manifests. GCP deployment section in README lists
exact `gcloud` commands.

---

## Phase 8 — Documentation

### Prompt 17 — HOW_IT_WORKS.md

> Write a `HOW_IT_WORKS.md` that walks through one complete request
> end-to-end — a customer adding a record to their collection — tied to actual
> file paths and function names. Include: the UI fetch call, the sync
> cross-service validation, the non-fatal Pub/Sub publish, the async consume
> and enrichment, and the read-back proof. Document the Pub/Sub TOCTOU race
> condition that was actually caught during development — the symptom, root
> cause, why it would also hit in production with multiple replicas, the fix,
> and how it was caught (smoke test, not unit tests).

**Evidence:** `HOW_IT_WORKS.md` follows this exact structure in this exact
order. The closing sentence of the race condition section — "This is the same
argument for why the CI smoke-test job exists as a separate stage from unit
tests" — reads like prompt language that made it into the documentation.

---

## A Note on the TOCTOU Bug

The most revealing prompt in the whole project is the one that produced the
TOCTOU race fix (Prompts 7 and 10). The level of specificity — exact symptom,
exact root cause, exact k8s replica failure mode, exact fix, and which testing
layer caught it — is more detailed than someone would write if they'd only
thought about the problem theoretically.

The most likely reconstruction is that the original scaffolding used a naive
check-then-create pattern. When `docker compose up` was run and two services
raced on startup, the crash appeared. A follow-up prompt was something like:

> "My docker compose up is crashing with: 'rpc error: code = AlreadyExists
> desc = Topic already exists'. Here is the stack trace. Explain exactly what
> is happening, why it would also be a problem with multiple Kubernetes
> replicas, and fix it properly without introducing a different race
> condition."

That kind of prompt — a real error, a real stack trace, a real environment —
produces the specific, honest documentation this project has. It's also the
most valuable thing in the portfolio: not that it was built correctly the
first time, but that a real distributed-systems bug was hit, diagnosed, fixed
correctly, and documented in a way that shows genuine understanding of why
it happened.
