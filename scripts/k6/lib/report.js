import { textSummary } from "https://jslib.k6.io/k6-summary/0.1.0/index.js";
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

// Shared handleSummary builder so both k6 scripts produce a browsable HTML
// report and a raw JSON dump (in addition to the usual colored text summary
// on stdout, which this preserves) without duplicating the boilerplate.
// `prefix` keeps each script's report files from overwriting each other
// when both run in the same working directory in CI.
export function buildSummary(prefix, data) {
  return {
    stdout: textSummary(data, { indent: " ", enableColors: true }),
    [`${prefix}-summary.json`]: JSON.stringify(data, null, 2),
    [`${prefix}-summary.html`]: htmlReport(data),
  };
}
