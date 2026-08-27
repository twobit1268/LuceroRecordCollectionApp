package com.vinylvault.acceptance.steps;

import com.vinylvault.acceptance.support.ScenarioContext;
import com.vinylvault.acceptance.support.TestConfig;
import io.cucumber.java.en.Given;
import io.cucumber.java.en.Then;
import io.cucumber.java.en.When;
import io.restassured.RestAssured;
import io.restassured.http.ContentType;
import io.restassured.response.Response;

import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Steps for collection-service (:8082) - which records a given customer owns.
 * On add it validates synchronously against catalog-service before persisting,
 * then publishes a {@code collection.record_added} event to Pub/Sub.
 */
public class CollectionSteps {

    private final TestConfig config;
    private final ScenarioContext context;

    public CollectionSteps(TestConfig config, ScenarioContext context) {
        this.config = config;
        this.context = context;
    }

    @When("I add {string} to my collection")
    public void iAddToMyCollection(String title) {
        String recordId = context.recordIdsByTitle.get(title);
        assertThat(recordId)
                .as("record '%s' must be created earlier in the scenario", title)
                .isNotNull();
        addRecordId(recordId);
    }

    @Given("{string} is in my collection")
    public void isInMyCollection(String title) {
        iAddToMyCollection(title);
        assertThat(context.lastResponse.statusCode())
                .as("precondition: add '%s' to collection", title)
                .isEqualTo(201);
    }

    @When("I try to add record id {string} to my collection")
    public void iTryToAddRecordId(String recordId) {
        addRecordId(recordId);
    }

    @Then("my collection has {int} record(s)")
    public void myCollectionHasRecords(int expected) {
        Response response = RestAssured.given().baseUri(config.collectionUrl)
                .when().get("/collections/{customerId}/records", context.customerId)
                .then().statusCode(200)
                .extract().response();
        context.lastResponse = response;

        List<Object> entries = response.jsonPath().getList("$");
        assertThat(entries)
                .as("entries in %s's collection", context.customerId)
                .hasSize(expected);
    }

    @When("I remove that entry from my collection")
    public void iRemoveThatEntry() {
        assertThat(context.lastEntryId).as("no collection entry to remove").isNotNull();
        context.lastResponse = deleteEntry(context.lastEntryId);
    }

    @Then("removing that entry again returns {int}")
    public void removingThatEntryAgainReturns(int expected) {
        Response response = deleteEntry(context.lastEntryId);
        context.lastResponse = response;
        assertThat(response.statusCode())
                .as("second DELETE of the same entry should be idempotent")
                .isEqualTo(expected);
    }

    // --- helpers -----------------------------------------------------------

    private void addRecordId(String recordId) {
        Response response = RestAssured.given().baseUri(config.collectionUrl)
                .contentType(ContentType.JSON)
                .body(Map.of("recordId", recordId))
                .when().post("/collections/{customerId}/records", context.customerId)
                .then().extract().response();

        context.lastResponse = response;
        if (response.statusCode() == 201) {
            context.lastEntryId = response.jsonPath().getString("id");
        }
    }

    private Response deleteEntry(String entryId) {
        return RestAssured.given().baseUri(config.collectionUrl)
                .when().delete("/collections/{customerId}/records/{entryId}",
                        context.customerId, entryId)
                .then().extract().response();
    }
}
