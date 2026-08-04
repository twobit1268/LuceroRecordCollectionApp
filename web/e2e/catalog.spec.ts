import { test, expect } from "@playwright/test";
import { CatalogPage } from "./pages/CatalogPage";

test.describe("Catalog", () => {
  test("creates a record and displays it in the list", async ({ page }) => {
    const catalog = new CatalogPage(page);
    await page.goto("/");

    const title = `Test Album ${Date.now()}`;
    await catalog.createRecord({ title, artist: "Test Artist", genre: "Rock", year: "2020" });

    const row = catalog.rowForTitle(title);
    await expect(row).toBeVisible();
    await expect(row).toContainText("Test Artist");
    await expect(row).toContainText("Rock");
    await expect(row).toContainText("2020");
  });

  test("does not submit when a required field is left empty", async ({ page }) => {
    const catalog = new CatalogPage(page);
    await page.goto("/");

    const title = `Incomplete Record ${Date.now()}`;
    await catalog.titleInput.fill(title);
    // artist/genre/year deliberately left blank
    await catalog.addButton.click();

    // Native HTML5 required-field validation should block the submit —
    // no row should ever appear for this title.
    await expect(catalog.rowForTitle(title)).not.toBeVisible();
  });
});
