// API load test against catalog-service alone: sustained concurrent
// create + list traffic, gated on latency and error-rate thresholds.
// k6 exits non-zero if thresholds aren't met, so this is a real pass/fail
// gate, not just a numbers-printer.
import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 10,
  duration: "15s",
  thresholds: {
    http_req_duration: ["p(95)<300"], // 95% of requests under 300ms
    http_req_failed: ["rate<0.01"], // less than 1% error rate
    checks: ["rate>0.99"],
  },
};

const BASE_URL = __ENV.CATALOG_URL || "http://localhost:8081";

export default function () {
  const params = { headers: { "Content-Type": "application/json" } };

  const payload = JSON.stringify({
    title: `Load Test Album ${__VU}-${__ITER}`,
    artist: "k6 Load Test",
    genre: "Test",
    year: 2024,
  });

  const createRes = http.post(`${BASE_URL}/records`, payload, params);
  check(createRes, { "create record: status 201": (r) => r.status === 201 });

  const listRes = http.get(`${BASE_URL}/records`);
  check(listRes, { "list records: status 200": (r) => r.status === 200 });

  sleep(1);
}
