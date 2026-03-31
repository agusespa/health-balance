const { test, expect } = require("@playwright/test");
const { resetSeededDatabase } = require("../helpers/test-env");

test.beforeEach(() => {
  resetSeededDatabase();
});

test("loads the dashboard with seeded profile data", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByTestId("score-container")).toBeVisible();
  await expect(page.locator("body")).toHaveAttribute("data-has-profile", "true");
});

test("shows the completed-week entry state before any active-week save", async ({ page }) => {
  await page.goto("/");

  await page.getByTestId("health-pillar-toggle").click();
  const healthWeekState = page.getByTestId("health-week-state");

  await expect(healthWeekState.getByTestId("health-week-badge")).toBeVisible();
  await expect(healthWeekState.getByTestId("health-week-status-eyebrow")).toHaveText("No entry yet for this completed week");
  await expect(healthWeekState.getByTestId("copy-last-health")).toBeVisible();
});
