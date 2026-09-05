@collection
Feature: A customer's collection references catalogued records

  collection-service owns which records a given customer owns. Adding an entry
  triggers a synchronous validation call to catalog-service first; only if
  that record really exists is the entry persisted (and an event published).

  Background:
    Given the catalog service is available
    And a catalog record "Kind of Blue" by "Miles Davis" in genre "Jazz" released in 1959

  Scenario: Adding a real record succeeds
    When I add "Kind of Blue" to my collection
    Then the response status is 201
    And my collection has 1 record

  Scenario: Adding an unknown record id is rejected by the synchronous catalog check
    When I try to add record id "does-not-exist" to my collection
    Then the response status is 400
    And my collection has 0 records

  Scenario: Removing a collection entry is idempotent
    Given "Kind of Blue" is in my collection
    When I remove that entry from my collection
    Then the response status is 204
    And removing that entry again returns 404
