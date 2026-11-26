import { expect, test } from '@playwright/test';

// Safari must discover icons from the document, before the app runs.
test.use({ javaScriptEnabled: false });

test('serves browser and installation icons in the initial HTML', async ({
	page,
	request,
}) => {
	await page.goto('/');
	const png = page.locator('head link[rel="icon"][type="image/png"]');
	const svg = page.locator('head link[rel="icon"][type="image/svg+xml"]');
	const touch = page.locator('head link[rel="apple-touch-icon"]');
	await expect(png).toHaveAttribute('sizes', '32x32');
	await expect(touch).toHaveAttribute('sizes', '180x180');
	const pngUrl = (await png.getAttribute('href'))!;
	const touchUrl = (await touch.getAttribute('href'))!;
	const hash = pngUrl.match(/-([a-f0-9]{12})\.png$/)![1];
	expect(touchUrl).toContain(`-${hash}.png`);
	const manifest = await (await request.get('/manifest.webmanifest')).json();
	const images = [
		{ src: pngUrl, size: 32 },
		{ src: touchUrl, size: 180 },
		...manifest.icons.map((icon: { src: string; sizes: string }) => ({
			src: icon.src,
			size: Number(icon.sizes.split('x')[0]),
		})),
	];
	for (const image of images) {
		expect(image.src).toContain(`-${hash}.png`);
		const response = await request.get(image.src);
		expect(response.ok()).toBeTruthy();
		expect(response.headers()['content-type']).toContain('image/png');
		const bytes = await response.body();
		expect(bytes.subarray(0, 8).toString('hex')).toBe('89504e470d0a1a0a');
		expect(bytes.readUInt32BE(16)).toBe(image.size);
		expect(bytes.readUInt32BE(20)).toBe(image.size);
	}
	const svgUrl = (await svg.getAttribute('href'))!;
	expect(svgUrl).toContain(`-${hash}.svg`);
	const response = await request.get(svgUrl);
	expect(response.headers()['content-type']).toContain('image/svg+xml');
	expect(await response.text()).toContain('<svg');
});
