import { expect, test, type Page } from '@playwright/test';

// The config blocks service workers; these specs are about the worker.
test.use({ serviceWorkers: 'allow' });

/** Load the app and wait for the service worker to control the page. */
async function openApp(page: Page) {
	await page.goto('/');
	await page.evaluate(async () => {
		await navigator.serviceWorker.ready;
		if (!navigator.serviceWorker.controller) {
			await new Promise((r) =>
				navigator.serviceWorker.addEventListener('controllerchange', r, { once: true })
			);
		}
	});
	await expect(page.getByTestId('score-value')).toHaveText(/^\d{3,4}$/);
}

test('launches from the cached shell without the network', async ({ page, context }) => {
	await openApp(page);

	await context.setOffline(true);
	await page.reload();
	await expect(page.getByRole('link', { name: 'Profile' })).toBeVisible();

	// Every route is the same shell, so deep links open offline too.
	for (const path of ['/settings', '/trends', '/profile', '/profile?view=help', '/week/2026-01-04']) {
		const res = await page.goto(path);
		expect(res!.fromServiceWorker(), path).toBe(true);
		await expect(page.locator('h1').first(), path).toBeVisible();
	}
	await page.goto('/settings');
	await expect(page.getByTestId('profile-form')).toBeVisible();
	await context.setOffline(false);
});

test('deep links and reloads work online under the worker', async ({ page }) => {
	await openApp(page);

	const res = await page.goto('/settings');
	expect(res!.fromServiceWorker()).toBe(true);
	await expect(page.getByTestId('profile-form').getByLabel('Birth date')).toHaveValue(
		'1990-06-15'
	);
	await page.reload();
	await expect(page.getByTestId('profile-form').getByLabel('Birth date')).toHaveValue(
		'1990-06-15'
	);
});

test('the worker leaves the API alone', async ({ page }) => {
	await openApp(page);
	const [res] = await Promise.all([
		page.waitForResponse((r) => r.url().endsWith('/api/dashboard')),
		page.reload()
	]);
	expect(res.fromServiceWorker()).toBe(false);
});

test('sign-in navigations skip the cached shell and come back clean', async ({ page }) => {
	await openApp(page);
	expect((await page.goto('/'))!.fromServiceWorker()).toBe(true);

	// "Sign in again" links here, so an expired Access session reaches its login page.
	expect((await page.goto('/?signin'))!.fromServiceWorker()).toBe(false);
	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByTestId('score-value')).toHaveText(/^\d{3,4}$/);

	// From a sub-page, the sign-in comes back to that page.
	expect((await page.goto('/settings?signin'))!.fromServiceWorker()).toBe(false);
	await expect(page).toHaveURL(/\/settings$/);
	await expect(page.getByTestId('profile-form')).toBeVisible();
});
