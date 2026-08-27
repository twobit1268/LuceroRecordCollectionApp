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
 * Steps for catalog-service (:8081) - the master list of records.
 */
public class CatalogSteps {

    private final TestConfig config;
    private final ScenarioContext context;

    public CatalogSteps(TestConfig config, ScenarioContext context) {
        this.config = config;
        this.context = context;
    }

    @Given("the catalog service is available")
    public void theCatalogServiceIsAvailable() {
        Response response = RestAssured.given().baseUri(config.catalogUrl)
                .when().get("/records")
                .then().extract().response();
        context.lastResponse = response;
        assertThat(response.statusCode())
                .as("GET /records should be reachable")
                .isEqualTo(200);
    }

    // Phrased as a "When" for scenarios that are testing creation itself...
    @When("I create a record {string} by {string} in genre {string} released in {int}")
    public void iCreateARecord(String title, String artist, String genre, int year) {
        Response response = RestAssured.given().baseUri(config.catalogUrl)
                .contentType(ContentType.JSON)
                .body(Map.of("title", title, "artist", artist, "genre", genre, "year", year))
                .when().post("/records")
                .then().extract().response();

        context.lastResponse = response;
        if (response.statusCode() == 201) {
            String id = response.jsonPath().getString("id");
            context.lastRecordId = id;
            context.recordIdsByTitle.put(title, id);
        }
    }

    // ...and as a "Given" for scenarios where an existing record is just a
    // precondition. Same call, but here a non-201 is a setup failure, not a
    // result under test.
    @Given("a catalog record {string} by {string} in genre {string} released in {int}")
    public void aCatalogRecordExists(String title, String artist, String genre, int year) {
        iCreateARecord(title, artist, genre, year);
        assertThat(context.lastResponse.statusCode())
                .as("precondition: create catalog record '%s'", title)
                .isEqualTo(201);
    }

    @When("I fetch that record by its id")
    public void iFetchThatRecordByItsId() {
        assertThat(context.lastRecordId).as("no record has been created yet").isNotNull();
        context.lastResponse = RestAssured.given().baseUri(config.catalogUrl)
                .when().get("/records/{id}", context.lastRecordId)
                .then().extract().response();
    }

    @Then("the catalog list contains {string}")
    public void theCatalogListContains(String title) {
        List<String> titles = RestAssured.given().baseUri(config.catalogUrl)
                .when().get("/records")
                .then().statusCode(200)
                .extract().jsonPath().getList("title", String.class);

        assertThat(titles).as("titles returned by GET /records").contains(title);
    }
}
