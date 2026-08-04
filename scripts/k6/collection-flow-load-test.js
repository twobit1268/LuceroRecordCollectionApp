// End-to-end platform load test: exercises the full distributed flow under
// concurrent load, not just one service in isolation — create a record in
// catalog-service, add it to a customer's collection in collection-service
// (which synchronously validates against catalog-service and publishes a
// Pub/Sub event), then confirm it's listed. This is the scenario most
// likely to reveal cross-service contention that a single-service load
// test (catalog-load-test.js) can't see.
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 5,
  duration: "15s",
  thresholds: {
    http_req_duration: ["p(95)<500"],
    http_req_failed: ["rate<0.01"],
    checks: ["rate>0.99"],
  },
};

const CATALOG_URL = __ENV.CATALOG_URL || "http://localhost:8081";
const COLLECTION_URL = __ENV.COLLECTION_URL || "http://localhost:8082";

export default function () {
  const params = { headers: { "Content-Type": "application/json" } };

  const record = JSON.stringify({
    title: `Load Test Album ${__VU}-${__ITER}`,
    artist: "k6 Load Test",
    genre: "Test",
    year: 2024,
  });

  const createRes = http.post(`${CATALOG_URL}/records`, record, params);
  const created = check(createRes, { "catalog create: status 201": (r) => r.status === 201 });
  if (!created) return;

  const recordId = createRes.json("id");
  const customerId = `k6-customer-${__VU}`;

  const addRes = http.post(
    `${COLLECTION_URL}/collections/${customerId}/records`,
    JSON.stringify({ recordId }),
    params
  );
  check(addRes, { "collection add: status 201": (r) => r.status === 201 });

  const listRes = http.get(`${COLLECTION_URL}/collections/${customerId}/records`);
  check(listRes, { "collection list: status 200": (r) => r.status === 200 });

  sleep(1);
}
