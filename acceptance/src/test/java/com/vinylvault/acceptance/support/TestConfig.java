package com.vinylvault.acceptance.support;

/**
 * Immutable per-run configuration: the base URL of each service.
 *
 * <p>Defaults match {@code docker compose up} on localhost; override with
 * environment variables when the stack lives elsewhere (a remote host, a
 * Testcontainers network, a k8s port-forward), e.g.
 *
 * <pre>
 *   CATALOG_URL=http://catalog.staging:8081 mvn test
 * </pre>
 *
 * <p>PicoContainer creates one instance of this per scenario and injects it
 * into every step-definition class and hook that asks for it in its
 * constructor - the "World" pattern, minus the god-object.
 */
public class TestConfig {

    public final String catalogUrl = env("CATALOG_URL", "http://localhost:8081");
    public final String collectionUrl = env("COLLECTION_URL", "http://localhost:8082");
    public final String activityUrl = env("ACTIVITY_URL", "http://localhost:8083");

    /** How long to wait for the async Pub/Sub path to become consistent. */
    public final int activityTimeoutSeconds =
            Integer.parseInt(env("ACTIVITY_TIMEOUT_SECONDS", "30"));

    private static String env(String key, String fallback) {
        String value = System.getenv(key);
        return (value == null || value.isBlank()) ? fallback : value;
    }
}
