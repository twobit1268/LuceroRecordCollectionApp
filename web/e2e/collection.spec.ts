import { test, expect } from "@playwright/test";
import { CatalogPage } from "./pages/CatalogPage";
import { CollectionPage } from "./pages/CollectionPage";

test.describe("Collection", () => {
  test("adds a catalog record to a customer's collection", async ({ page }) => {
    const catalog = new CatalogPage(page);
    const collection = new CollectionPage(page);
    await page.goto("/");

    const title = `Collection Test ${Date.now()}`;
    const artist = "Someone";
    await catalog.createRecord({ title, artist, genre: "Jazz", year: "1999" });

    await collection.goto();
    await collection.setCustomerId(`e2e-customer-${Date.now()}`);
    await collection.addRecordByLabel(`${title} — ${artist}`);

    await expect(collection.rowForRecordTitle(title)).toBeVisible();
  });

  test("removes an entry from the collection", async ({ page }) => {
    const catalog = new CatalogPage(page);
    const collection = new CollectionPage(page);
    await page.goto("/");

    const title = `Removable Record ${Date.now()}`;
    const artist = "Nobody";
    await catalog.createRecord({ title, artist, genre: "Folk", year: "2005" });

    await collection.goto();
    await collection.setCustomerId(`e2e-remove-${Date.now()}`);
    await collection.addRecordByLabel(`${title} — ${artist}`);
    await expect(collection.rowForRecordTitle(title)).toBeVisible();

    await collection.removeButtonForRecordTitle(title).click();
    await expect(collection.rowForRecordTitle(title)).not.toBeVisible();
  });

  test("switching customer ID shows a different collection", async ({ page }) => {
    const catalog = new CatalogPage(page);
    const collection = new CollectionPage(page);
    await page.goto("/");

    const title = `Customer-Scoped Record ${Date.now()}`;
    const artist = "Scoped Artist";
    await catalog.createRecord({ title, artist, genre: "Pop", year: "2015" });

    await collection.goto();
    const customerA = `e2e-a-${Date.now()}`;
    const customerB = `e2e-b-${Date.now()}`;

    await collection.setCustomerId(customerA);
    await collection.addRecordByLabel(`${title} — ${artist}`);
    await expect(collection.rowForRecordTitle(title)).toBeVisible();

    await collection.setCustomerId(customerB);
    await expect(collection.rowForRecordTitle(title)).not.toBeVisible();
  });
});
