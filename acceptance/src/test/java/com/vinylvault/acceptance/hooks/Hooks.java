package com.vinylvault.acceptance.hooks;

import com.vinylvault.acceptance.support.ScenarioContext;
import com.vinylvault.acceptance.support.TestConfig;
import io.cucumber.java.After;
import io.cucumber.java.Before;
import io.cucumber.java.Scenario;
import io.restassured.RestAssured;

import java.time.Duration;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicBoolean;

import static org.awaitility.Awaitility.await;

/**
 * Lifecycle glue that runs around every scenario.
 *
 * <ul>
 *   <li>{@code @Before(order = 0)} - block until all three services report
 *       healthy, once per JVM. This is the Java equivalent of the
 *       {@code wait_for} loop at the top of {@code scripts/smoke-test.sh}.</li>
 *   <li>{@code @Before(order = 10)} - mint a fresh {@code customerId} so each
 *       scenario's collection is isolated.</li>
 *   <li>{@code @After} - on failure, attach the last HTTP response to the
 *       report so you can see <em>why</em> without re-running.</li>
 * </ul>
 */
public class Hooks {

    private static final AtomicBoolean SERVICES_READY = new AtomicBoolean(false);

    private final TestConfig config;
    private final ScenarioContext context;

    // Constructor injection - PicoContainer supplies both arguments.
    public Hooks(TestConfig config, ScenarioContext context) {
        this.config = config;
        this.context = context;
    }

    @Before(order = 0)
    public void waitForServicesOnce() {
        if (SERVICES_READY.get()) {
            return;
        }
        awaitHealthy("catalog-service", config.catalogUrl);
        awaitHealthy("collection-service", config.collectionUrl);
        awaitHealthy("activity-service", config.activityUrl);
        SERVICES_READY.set(true);
    }

    @Before(order = 10)
    public void freshCustomerPerScenario() {
        context.customerId = "cuke-" + UUID.randomUUID().toString().substring(0, 8);
    }

    @After
    public void attachResponseOnFailure(Scenario scenario) {
        if (scenario.isFailed() && context.lastResponse != null) {
            scenario.log("Last HTTP response was "
                    + context.lastResponse.statusCode() + ":\n"
                    + context.lastResponse.asPrettyString());
        }
    }

    private static void awaitHealthy(String name, String baseUrl) {
        await("%s to be healthy at %s/healthz".formatted(name, baseUrl))
                .atMost(Duration.ofSeconds(60))
                .pollInterval(Duration.ofSeconds(2))
                .ignoreExceptions()
                .until(() -> RestAssured.given().baseUri(baseUrl)
                        .get("/healthz")
                        .statusCode() == 200);
    }
}
