@distributed @smoke
Feature: The full distributed flow across all three services

  This is the Cucumber/Java counterpart of scripts/smoke-test.sh and
  web/e2e/end-to-end-flow.spec.ts. One scenario walks the entire system:

    catalog create
      -> collection add        (synchronous validation against catalog-service)
      -> Pub/Sub event         (collection-service publishes, async)
      -> activity-service       (consumes independently, re-calls catalog-service
                                 to enrich the event with genre)

  Because the last hop is asynchronous, the final steps poll with a timeout
  rather than asserting immediately.

  Background:
    Given the catalog service is available

  Scenario: A newly collected record appears genre-enriched in the activity feed
    When I create a record "Kind of Blue" by "Miles Davis" in genre "Jazz" released in 1959
    Then the response status is 201
    When I add "Kind of Blue" to my collection
    Then the response status is 201
    And my collection has 1 record
    Then the activity feed shows "Kind of Blue" enriched with genre "Jazz" within 30 seconds
    And the genre counts include "Jazz" with count at least 1
