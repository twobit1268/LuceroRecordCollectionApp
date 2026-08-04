import type { Locator, Page } from "@playwright/test";

export interface RecordInput {
  title: string;
  artist: string;
  genre: string;
  year: string;
}

export class CatalogPage {
  readonly page: Page;
  readonly tab: Locator;
  readonly titleInput: Locator;
  readonly artistInput: Locator;
  readonly genreInput: Locator;
  readonly yearInput: Locator;
  readonly addButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.tab = page.getByRole("button", { name: "Catalog" });
    this.titleInput = page.getByPlaceholder("Title");
    this.artistInput = page.getByPlaceholder("Artist");
    this.genreInput = page.getByPlaceholder("Genre");
    this.yearInput = page.getByPlaceholder("Year");
    this.addButton = page.getByRole("button", { name: "Add record" });
  }

  async goto(): Promise<void> {
    await this.tab.click();
  }

  async createRecord(input: RecordInput): Promise<void> {
    await this.titleInput.fill(input.title);
    await this.artistInput.fill(input.artist);
    await this.genreInput.fill(input.genre);
    await this.yearInput.fill(input.year);
    await this.addButton.click();
  }

  rowForTitle(title: string): Locator {
    return this.page.getByRole("row").filter({ hasText: title });
  }
}
