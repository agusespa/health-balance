import { expect, test } from '@playwright/test';

test('saves the profile', async ({ page }) => {
	await page.goto('/settings');
	const form = page.getByTestId('profile-form');
	await expect(form.getByLabel('Birth date')).toHaveValue('1990-06-15');

	await form.getByLabel('Height (cm)').fill('181');
	await form.getByRole('button', { name: 'Save profile' }).click();
	await expect(form.getByTestId('profile-saved')).toHaveText('Saved');

	await page.reload();
	await expect(form.getByLabel('Height (cm)')).toHaveValue('181');

	// Put it back for the other tests.
	await form.getByLabel('Height (cm)').fill('180');
	await form.getByRole('button', { name: 'Save profile' }).click();
	await expect(form.getByTestId('profile-saved')).toHaveText('Saved');
});
