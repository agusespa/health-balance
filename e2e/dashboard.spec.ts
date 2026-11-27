import { expect, test } from '@playwright/test';

test('shows the score with markers and behaviours', async ({ page }) => {
	await page.goto('/');

	await expect(page.getByTestId('score-value')).toHaveText(/^\d{3,4}$/);
	// The seed has data up to today.
	await expect(page.getByTestId('stale-banner')).toHaveCount(0);

	const markers = page.getByTestId('card-markers');
	const behaviours = page.getByTestId('card-behaviours');
	await expect(markers.locator('[data-testid^="metric-"]')).toHaveCount(6);
	await expect(behaviours.locator('[data-testid^="metric-"]')).toHaveCount(5);
	await expect(markers.getByTestId('metric-vo2_max')).toContainText('VO2 max');
	await expect(behaviours.getByTestId('metric-working_sets')).toContainText('Working sets');
});

test('opens score history and a week', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('button', { name: 'Score history' }).click();

	const weeks = page.locator('ul a[href^="/week/"]');
	await expect(weeks).toHaveCount(10);
	await weeks.first().click();

	await expect(page).toHaveURL(/\/week\/\d{4}-\d{2}-\d{2}$/);
	await expect(page.getByTestId('card-markers')).toBeVisible();
	await expect(page.getByTestId('card-behaviours')).toBeVisible();
});

test('shows an empty insights state and explains a missing API key', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('button', { name: 'AI insights' }).click();
	await expect(page.getByText('No insights yet')).toBeVisible();

	// The e2e server has no ANTHROPIC_API_KEY.
	await page.getByRole('button', { name: 'Write insights now' }).click();
	await expect(page.getByTestId('insight-error')).toContainText('ANTHROPIC_API_KEY');
});

test('trends shows the metric charts', async ({ page }) => {
	await page.goto('/trends');
	await expect(page.getByTestId('trend-rhr').getByRole('img', { name: 'trend' })).toBeVisible();
	await expect(page.getByTestId('trend-vo2_max').getByRole('img', { name: 'trend' })).toBeVisible();
});

test('a client-side route loads directly', async ({ page }) => {
	await page.goto('/rationale');
	await expect(page.getByRole('heading', { name: 'Biological Reserve Score' })).toBeVisible();
});

test('shows nutrition for the last 7 days, unscored', async ({ page }) => {
	await page.goto('/');
	const card = page.getByTestId('card-nutrition');
	await expect(card).toContainText('Not part of the score');
	await expect(card.getByTestId('nutrition-logged')).toHaveText(/^\d\/7 logged$/);
	await expect(card.getByTestId('nutrition-balance')).toContainText(/kcal\/day/);
	await expect(card.getByTestId('nutrition-protein')).toContainText(/g\/kg/);
	await expect(card.getByTestId('nutrition-fiber')).toContainText(/\d+ g/);
	await expect(card.getByTestId('nutrition-water')).toContainText(/\d\.\d L/);
});
