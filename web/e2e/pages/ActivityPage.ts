import type { Locator, Page } from "@playwright/test";
import { expect } from "@playwright/test";

export class ActivityPage {
  readonly page: Page;
  readonly tab: Locator;
  readonly refreshButton: Locator;

  constructor(page: Page) {
    this.page = page;
    this.tab = page.getByRole("button", { name: "Activity" });
    this.refreshButton = page.getByRole("button", { name: "Refresh" });
  }

  async goto(): Promise<void> {
    await this.tab.click();
  }

  rowForCustomer(customerId: string): Locator {
    return this.page.getByRole("row").filter({ hasText: customerId });
  }

  /**
   * The Activity tab is fed asynchronously via Pub/Sub — it only fetches
   * once on mount/refresh, it doesn't poll on its own. This clicks Refresh
   * and re-checks until the expected row shows up (or the timeout elapses),
   * which is the real-world equivalent of a user hitting refresh a few
   * times while waiting for the async path to catch up.
   */
  async waitForCustomerActivity(customerId: string, timeout = 15000): Promise<void> {
    await expect(async () => {
      await this.refreshButton.click();
      await expect(this.rowForCustomer(customerId)).toBeVisible({ timeout: 1000 });
    }).toPass({ timeout, intervals: [1000] });
  }
}
