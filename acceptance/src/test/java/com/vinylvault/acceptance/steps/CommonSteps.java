package com.vinylvault.acceptance.steps;

import com.vinylvault.acceptance.support.ScenarioContext;
import io.cucumber.java.en.Then;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Assertions that apply to "the last response" regardless of which service
 * produced it. Keeping them in one class means every {@code Then the response
 * status is ...} line in every feature maps to a single step definition.
 */
public class CommonSteps {

    private final ScenarioContext context;

    public CommonSteps(ScenarioContext context) {
        this.context = context;
    }

    @Then("the response status is {int}")
    public void theResponseStatusIs(int expected) {
        assertThat(context.lastResponse).as("no request has been made yet").isNotNull();
        assertThat(context.lastResponse.statusCode())
                .as("HTTP status of the last response")
                .isEqualTo(expected);
    }

    @Then("the response body field {string} equals {string}")
    public void theResponseBodyFieldEquals(String jsonPath, String expected) {
        String actual = context.lastResponse.jsonPath().getString(jsonPath);
        assertThat(actual)
                .as("response body field '%s'", jsonPath)
                .isEqualTo(expected);
    }
}
