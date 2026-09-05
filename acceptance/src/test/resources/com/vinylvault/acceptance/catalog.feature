@catalog
Feature: Catalog service manages the master list of records

  catalog-service owns the canonical record data (title, artist, genre, year)
  and is a plain REST CRUD service. Every other service reads from it but
  never writes to its database.

  Background:
    Given the catalog service is available

  Scenario: A created record can be read back by id
    When I create a record "Kind of Blue" by "Miles Davis" in genre "Jazz" released in 1959
    Then the response status is 201
    And the response body field "artist" equals "Miles Davis"
    When I fetch that record by its id
    Then the response status is 200
    And the response body field "genre" equals "Jazz"

  # Scenario Outline runs the same steps once per row of the Examples table -
  # data-driven testing without copy-pasting the scenario.
  Scenario Outline: Records across genres are catalogued and listed
    When I create a record "<title>" by "<artist>" in genre "<genre>" released in <year>
    Then the response status is 201
    And the catalog list contains "<title>"

    Examples:
      | title      | artist        | genre      | year |
      | Rumours    | Fleetwood Mac | Rock       | 1977 |
      | Blue Train | John Coltrane | Jazz       | 1957 |
      | Discovery  | Daft Punk     | Electronic | 2001 |
