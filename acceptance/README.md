# Acceptance tests — Cucumber-JVM + REST Assured

BDD acceptance suite for VinylVault. This is the **Java equivalent of
[`scripts/smoke-test.sh`](../scripts/smoke-test.sh)** and
[`web/e2e/end-to-end-flow.spec.ts`](../web/e2e/end-to-end-flow.spec.ts): the same
distributed flow — synchronous catalog validation plus the asynchronous Pub/Sub
genre-enrichment path — expressed as Gherkin scenarios and driven over HTTP with
[REST Assured](https://rest-assured.io/).

It runs against a live stack (`docker compose up`); it does not start or stop the
services itself.

## Run it

```bash
# 1. bring the backend up (from the repo root)
docker compose up --build -d catalog-db collection-db activity-db \
  pubsub-emulator catalog-service collection-service activity-service

# 2. run the suite
cd acceptance && mvn test

# 3. tear down
cd .. && docker compose down -v
```

Requires **JDK 17+** and **Maven 3.9+**. First run downloads the test
dependencies. `asdf` / `mise` users can get the toolchain from
[`.tool-versions`](./.tool-versions) with `mise install` (or `asdf install`).

### Useful variations

```bash
mvn test -Dcucumber.filter.tags="@catalog"              # one feature's worth
mvn test -Dcucumber.filter.tags="@smoke"                # just the end-to-end flow
mvn test -Dcucumber.filter.tags="not @distributed"      # everything except it

CATALOG_URL=http://host:8081 \
COLLECTION_URL=http://host:8082 \
ACTIVITY_URL=http://host:8083 mvn test                  # point at another stack
```

## Reports

After a run, under `acceptance/target/cucumber-report/`:

| File             | What it is                                            |
|------------------|------------------------------------------------------|
| `cucumber.html`  | Browsable report — open it directly                  |
| `cucumber.json`  | Machine-readable, for dashboards / trend tools       |
| `cucumber.xml`   | JUnit XML, for CI test-result UIs                    |

In CI these are uploaded as the **`cucumber-report`** artifact (see the
`acceptance-test` job in [`.github/workflows/ci.yml`](../.github/workflows/ci.yml)).

## Layout

```
acceptance/
  pom.xml
  src/test/
    java/com/vinylvault/acceptance/
      RunCucumberTest.java              JUnit Platform suite — the entry point
      support/
        TestConfig.java                service URLs (env-overridable)
        ScenarioContext.java           per-scenario shared state ("the World")
      hooks/
        Hooks.java                     wait-for-healthy, fresh customer id,
                                       attach response on failure
      steps/
        CommonSteps.java               "the response status is ..." etc.
        CatalogSteps.java              catalog-service (:8081)
        CollectionSteps.java           collection-service (:8082)
        ActivitySteps.java             activity-service (:8083), polled with Awaitility
    resources/
      junit-platform.properties        Cucumber config (glue path, report plugins)
      com/vinylvault/acceptance/
        catalog.feature                @catalog
        collection.feature             @collection
        distributed_flow.feature       @distributed @smoke
```

## How the pieces fit (Cucumber-JVM concepts)

- **Feature files** (`*.feature`) hold the scenarios in Gherkin —
  `Feature` / `Background` / `Scenario` / `Scenario Outline` + `Examples` /
  `Given` / `When` / `Then` / `And`.
- **Step definitions** (`steps/*.java`) — each annotated method matches one
  Gherkin line via a Cucumber Expression (`{string}`, `{int}`, `record(s)`) and
  does the real HTTP work.
- **Glue** — the package Cucumber scans for steps and hooks
  (`cucumber.glue` in `junit-platform.properties`).
- **The World** — `TestConfig` + `ScenarioContext`, handed to every step class
  by constructor injection (`cucumber-picocontainer`). A fresh pair per
  scenario, so scenarios can't leak state into each other.
- **Hooks** — `@Before` / `@After` in `Hooks.java` for setup/teardown.
- **Tags** — `@catalog`, `@collection`, `@distributed`, `@smoke`; filter with
  `-Dcucumber.filter.tags=...`.
- **Runner** — `RunCucumberTest` is a JUnit 5 `@Suite` that includes the
  `cucumber` engine; `mvn test`, IntelliJ, and CI all go through it.
