import { expect, test } from '@playwright/test';

function rows(): string {
	const lines: string[] = [];
	for (let i = 6; i >= 0; i--) {
		const d = new Date(Date.now() - i * 86400000).toISOString().slice(0, 10);
		lines.push(`RHR; 50;count/min;${d}T00:03:14+01:00;${d}T23:54:13+01:00;Agustin’s Apple Watch`);
		lines.push(`SETS;12;count;${d}T00:00:00+01:00;${d}T23:59:59+01:00;gympass`);
	}
	return lines.join('\n');
}

test('refuses a cross-site browser request', async ({ request }) => {
	const res = await request.post('/api/ingest/health', {
		headers: { Origin: 'https://evil.example', 'Sec-Fetch-Site': 'cross-site' },
		data: { rows: rows() }
	});
	expect(res.status()).toBe(403);
});

test('stores an ingest payload and records the sync', async ({ request, page }) => {
	const res = await request.post('/api/ingest/health', { data: { rows: rows() } });
	expect(res.ok()).toBe(true);
	const body = await res.json();
	expect(body.rejected).toBe(0);
	expect(body.metrics.rhr).toBe(7);

	const dashboard = await (await request.get('/api/dashboard')).json();
	expect(dashboard.latest_data).toMatch(/^\d{4}-\d{2}-\d{2}$/);

	await page.goto('/');
	await expect(page.getByTestId('stale-banner')).toHaveCount(0);
});
