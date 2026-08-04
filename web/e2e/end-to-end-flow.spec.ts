import { test, expect } from "@playwright/test";
import { CatalogPage } from "./pages/CatalogPage";
import { CollectionPage } from "./pages/CollectionPage";
import { ActivityPage } from "./pages/ActivityPage";

// The UI equivalent of scripts/smoke-test.sh: proves the full distributed
// flow works end-to-end through what a real user would actually click,
// not just through direct API calls. Exercises every hop in the chain —
// catalog create, collection-service's synchronous validation call back to
// catalog-service, the async Pub/Sub publish, and activity-service's own
// synchronous enrichment call back to catalog-service — and only passes if
// all of them genuinely worked together.
test("full distributed flow: catalog create -> collection add -> activity feed shows it genre-enriched", async ({
  page,
}) => {
  const catalog = new CatalogPage(page);
  const collection = new CollectionPage(page);
  const activity = new ActivityPage(page);

  await page.goto("/");

  const title = `E2E Flow Album ${Date.now()}`;
  const artist = "E2E Flow Artist";
  const genre = "Blues";
  await catalog.createRecord({ title, artist, genre, year: "2010" });
  await expect(catalog.rowForTitle(title)).toBeVisible();

  await collection.goto();
  const customerId = `e2e-flow-${Date.now()}`;
  await collection.setCustomerId(customerId);
  await collection.addRecordByLabel(`${title} — ${artist}`);
  await expect(collection.rowForRecordTitle(title)).toBeVisible();

  await activity.goto();
  await activity.waitForCustomerActivity(customerId);
  await expect(activity.rowForCustomer(customerId)).toContainText(genre);
});
