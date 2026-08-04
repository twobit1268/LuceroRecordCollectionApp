import { defineConfig, devices } from "@playwright/test";

// These are real browser tests against the full distributed system, not
// just the frontend — they need catalog/collection/activity-service (and
// Postgres, and the Pub/Sub emulator) actually running, not just a static
// UI build. Start everything with `docker compose up --build -d` from the
// repo root before running `npm run test:e2e`; there's no Playwright
// webServer config here because it only manages one process, and this
// suite depends on the whole stack.
export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  retries: process.env.CI ? 1 : 0,
  // In CI: "line" keeps the log readable, but "html" (default open: never)
  // still needs to run alongside it so playwright-report/ actually gets
  // written for the CI artifact upload — with only "line", no report
  // directory is created at all, so there's nothing to upload.
  reporter: process.env.CI ? [["line"], ["html", { open: "never" }]] : "html",
  use: {
    baseURL: process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:5173",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
