import { expect, test } from '@playwright/test';

// page.route doesn't see requests from pages a service worker controls (WebKit), so keep it off.
test.use({ serviceWorkers: 'block' });

test('an expired Access session offers to sign in again on the same page', async ({ page }) => {
	// Cloudflare Access answers an expired session with a redirect to its login page.
	await page.route('**/api/profile', (route) =>
		route.fulfill({ status: 302, headers: { Location: 'https://example.cloudflareaccess.com/login' } })
	);
	await page.goto('/settings');

	const banner = page.getByRole('alert');
	await expect(banner).toContainText('Your session expired.');
	await expect(banner.getByRole('link', { name: 'Sign in again' })).toHaveAttribute(
		'href',
		'/settings?signin'
	);
});
