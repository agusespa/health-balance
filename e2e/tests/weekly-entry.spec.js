const { test, expect } = require("@playwright/test");
const { resetSeededDatabase } = require("../helpers/test-env");

test.beforeEach(() => {
  resetSeededDatabase();
});

test("copies last week's health values into the active form", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("health-pillar-toggle").click();

  await page.getByTestId("copy-last-health").click();

  await expect(page.locator("#body_weight_kg")).not.toHaveValue("");
  await expect(page.locator("#waist_cm")).not.toHaveValue("");
  await expect(page.getByTestId("toast")).toContainText("Last week's health values copied");
});

test("saves health data and refreshes the completed-week state", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("health-pillar-toggle").click();

  await page.locator("#body_weight_kg").fill("76.4");
  await page.locator("#waist_cm").fill("83.2");
  await page.locator("#systolic_bp").fill("118");
  await page.locator("#diastolic_bp").fill("77");
  await page.locator("#rhr").fill("59");
  await page.locator("#sleep_score").fill("82");
  await page.locator("#nutrition_score").fill("7.8");

  await page.getByTestId("save-health-button").click();

  await expect(page.getByTestId("toast")).toContainText("Health data saved successfully");
  await expect(page.getByTestId("health-week-status-eyebrow")).toHaveText("Saved for this completed week");
});
