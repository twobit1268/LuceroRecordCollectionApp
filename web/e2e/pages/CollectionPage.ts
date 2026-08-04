import type { Locator, Page } from "@playwright/test";

export class CollectionPage {
  readonly page: Page;
  readonly tab: Locator;
  readonly customerIdInput: Locator;
  readonly recordSelect: Locator;
  readonly addButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.tab = page.getByRole("button", { name: "Collection", exact: true });
    this.customerIdInput = page.getByLabel("Customer ID");
    this.recordSelect = page.getByRole("combobox");
    this.addButton = page.getByRole("button", { name: "Add to collection" });
  }

  async goto(): Promise<void> {
    await this.tab.click();
  }

  async setCustomerId(customerId: string): Promise<void> {
    await this.customerIdInput.fill(customerId);
  }

  async addRecordByLabel(label: string): Promise<void> {
    await this.recordSelect.selectOption({ label });
    await this.addButton.click();
  }

  rowForRecordTitle(title: string): Locator {
    return this.page.getByRole("row").filter({ hasText: title });
  }

  removeButtonForRecordTitle(title: string): Locator {
    return this.rowForRecordTitle(title).getByRole("button", { name: "Remove" });
  }
}
