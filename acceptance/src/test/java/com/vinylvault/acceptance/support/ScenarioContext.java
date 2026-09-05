package com.vinylvault.acceptance.support;

import io.restassured.response.Response;

import java.util.HashMap;
import java.util.Map;

/**
 * Mutable state shared between the step definitions <em>within a single
 * scenario</em>. PicoContainer gives each scenario its own fresh instance, so
 * scenarios can't leak state into each other.
 *
 * <p>This is how a {@code Given} step in one class hands data to a {@code When}
 * or {@code Then} step in another - e.g. "the record I just created" or "the
 * collection entry I just added".
 */
public class ScenarioContext {

    /** A unique customer id per scenario, so collection state never collides. */
    public String customerId;

    /** The most recent HTTP response, for the shared assertion steps. */
    public Response lastResponse;

    /** id of the last catalog record created (server-assigned). */
    public String lastRecordId;

    /** id of the last collection entry added (server-assigned). */
    public String lastEntryId;

    /** Catalog record ids keyed by title, so later steps can refer to a
     *  record by its human-readable name instead of a UUID. */
    public final Map<String, String> recordIdsByTitle = new HashMap<>();
}
