import { expect, test } from '@playwright/test';

test('shows the score with markers and behaviours', async ({ page }) => {
	await page.goto('/');

	await expect(page.getByTestId('score-value')).toHaveText(/^\d{3,4}$/);
	// The dashboard scores the last finished Saturday–Friday week, and says so.
	await expect(page.getByRole('heading', { level: 1 })).toContainText('Last week');
	// The seed has data up to today.
	await expect(page.getByTestId('stale-badge')).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Profile', exact: true })).toBeVisible();

	const markers = page.getByTestId('card-markers');
	const behaviours = page.getByTestId('card-behaviours');
	await expect(markers.locator('[data-testid^="metric-"]')).toHaveCount(6);
	await expect(behaviours.locator('[data-testid^="metric-"]')).toHaveCount(5);
	await expect(markers.getByTestId('metric-vo2_max')).toContainText('VO2 max');
	await expect(behaviours.getByTestId('metric-working_sets')).toContainText('Working sets');
});

test('explains the score in plain words', async ({ page }) => {
	await page.goto('/');
	const meaning = page.getByTestId('card-meaning');
	await expect(meaning.getByTestId('meaning-standing')).toContainText('starting point');
	await expect(meaning.getByTestId('meaning-heading')).toContainText('Weeks like this one');
	await expect(meaning.getByTestId('meaning-improve')).toHaveText(/Biggest drag|Biggest room to grow|Every metric/);
	await meaning.getByRole('button', { name: 'How the score works' }).click();
	await expect(meaning.getByRole('tooltip')).toContainText('started in January 2024');
});

test('opens score history and a week from the profile', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('link', { name: 'Profile' }).click();
	await expect(page.getByRole('tab', { name: 'Score history' })).toHaveAttribute('aria-selected', 'true');

	const weeks = page.locator('ul a[href^="/week/"]');
	await expect(weeks.first()).toBeVisible();
	await weeks.first().click();

	await expect(page).toHaveURL(/\/week\/\d{4}-\d{2}-\d{2}$/);
	await expect(page.getByTestId('card-markers')).toBeVisible();
	await expect(page.getByTestId('card-behaviours')).toBeVisible();
});

test('opens AI insights on their own page and explains a missing API key', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('link', { name: 'AI insights' }).click();
	await expect(page).toHaveURL(/\/insights$/);
	await expect(page.getByText('No insights yet')).toBeVisible();

	// The e2e server has no ANTHROPIC_API_KEY.
	await page.getByRole('button', { name: 'Write insights now' }).click();
	await expect(page.getByTestId('insight-error')).toContainText('ANTHROPIC_API_KEY');
});

test('trends shows the metric charts', async ({ page }) => {
	await page.goto('/');
	await page.getByRole('link', { name: 'Trends' }).click();
	await expect(page.getByTestId('trend-rhr').getByRole('img', { name: 'trend' })).toBeVisible();
	await expect(page.getByTestId('trend-vo2_max').getByRole('img', { name: 'trend' })).toBeVisible();
});

test('profile switches to help and opens settings', async ({ page }) => {
	await page.goto('/profile');
	await page.getByRole('tab', { name: 'How it works' }).click();
	await expect(page).toHaveURL(/\/profile\?view=help$/);
	await expect(page.getByRole('heading', { name: 'Biological Reserve Score' })).toBeVisible();

	await page.getByRole('link', { name: 'Settings' }).click();
	await expect(page.getByTestId('profile-form')).toBeVisible();
	await page.getByRole('link', { name: 'Back' }).click();
	await expect(page).toHaveURL(/\/profile$/);
});

test('a client-side route loads directly', async ({ page }) => {
	await page.goto('/profile?view=help');
	await expect(page.getByRole('heading', { name: 'Biological Reserve Score' })).toBeVisible();
});

test('card info icon shows a tooltip in place', async ({ page }) => {
	await page.goto('/');
	const card = page.getByTestId('card-markers');
	await card.getByRole('button', { name: 'How markers are scored' }).click();
	await expect(card.getByRole('tooltip')).toContainText('last one is reused');
	await expect(page).toHaveURL(/\/$/);
	await page.keyboard.press('Escape');
	await expect(card.getByRole('tooltip')).toBeHidden();
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

test('stale data badges the profile icon and explains on the profile page', async ({ page }) => {
	// The seed is fresh, so make the newest data date old.
	await page.route('**/api/dashboard', async (route) => {
		const res = await route.fetch();
		await route.fulfill({ response: res, json: { ...(await res.json()), latest_data: '2024-01-05' } });
	});
	await page.route('**/api/latest-data', (route) => route.fulfill({ json: { latest_data: '2024-01-05' } }));

	await page.goto('/');
	await expect(page.getByTestId('stale-badge')).toBeVisible();
	await expect(page.getByTestId('stale-banner')).toHaveCount(0);

	await page.getByRole('link', { name: 'Profile: no new health data' }).click();
	await expect(page).toHaveURL(/\/profile$/);
	await expect(page.getByTestId('stale-banner')).toContainText('No new health data since Jan 5, 2024');
});

test('the profile page says when no health data has arrived yet', async ({ page }) => {
	await page.route('**/api/latest-data', (route) => route.fulfill({ json: { latest_data: '' } }));
	await page.goto('/profile');
	await expect(page.getByTestId('stale-banner')).toContainText('No health data has arrived yet');
});
