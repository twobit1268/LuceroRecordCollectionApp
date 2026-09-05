package com.vinylvault.acceptance.steps;

import com.vinylvault.acceptance.support.ScenarioContext;
import com.vinylvault.acceptance.support.TestConfig;
import io.cucumber.java.en.Then;
import io.restassured.RestAssured;
import io.restassured.response.Response;

import java.time.Duration;
import java.util.List;
import java.util.Map;

import static org.assertj.core.api.Assertions.assertThat;
import static org.awaitility.Awaitility.await;

/**
 * Steps for activity-service (:8083). It subscribes to the Pub/Sub topic
 * independently of collection-service, enriches each event with genre by
 * calling catalog-service itself, and serves a recent-activity feed plus
 * genre-count stats.
 *
 * <p>Because that path is asynchronous, these steps <em>poll</em> with a
 * timeout (Awaitility) rather than asserting once - the feed becomes
 * consistent a second or two after the collection add, not instantly.
 */
public class ActivitySteps {

    private final TestConfig config;
    private final ScenarioContext context;

    public ActivitySteps(TestConfig config, ScenarioContext context) {
        this.config = config;
        this.context = context;
    }

    @Then("the activity feed shows {string} enriched with genre {string} within {int} seconds")
    public void theActivityFeedShowsEnrichedGenre(String title, String expectedGenre, int seconds) {
        String recordId = context.recordIdsByTitle.get(title);
        assertThat(recordId).as("record '%s' must exist", title).isNotNull();

        await("activity feed to contain " + recordId + " enriched with genre")
                .atMost(Duration.ofSeconds(seconds))
                .pollInterval(Duration.ofSeconds(2))
                .untilAsserted(() -> {
                    Response response = RestAssured.given().baseUri(config.activityUrl)
                            .when().get("/activity/feed?limit=50")
                            .then().statusCode(200)
                            .extract().response();
                    context.lastResponse = response;

                    List<Map<String, Object>> feed = response.jsonPath().getList("$");
                    Map<String, Object> entry = feed.stream()
                            .filter(e -> recordId.equals(e.get("recordId")))
                            .findFirst()
                            .orElse(null);

                    assertThat(entry).as("feed entry for record " + recordId).isNotNull();
                    assertThat(entry.get("genre"))
                            .as("genre enriched onto the event via catalog-service")
                            .isEqualTo(expectedGenre);
                });
    }

    @Then("the genre counts include {string} with count at least {int}")
    public void theGenreCountsInclude(String genre, int minimum) {
        Response response = RestAssured.given().baseUri(config.activityUrl)
                .when().get("/activity/genres")
                .then().statusCode(200)
                .extract().response();
        context.lastResponse = response;

        List<Map<String, Object>> counts = response.jsonPath().getList("$");
        Map<String, Object> match = counts.stream()
                .filter(c -> genre.equals(c.get("genre")))
                .findFirst()
                .orElse(null);

        assertThat(match).as("genre-count row for '%s'", genre).isNotNull();
        assertThat(((Number) match.get("count")).intValue())
                .as("count for genre '%s'", genre)
                .isGreaterThanOrEqualTo(minimum);
    }
}
