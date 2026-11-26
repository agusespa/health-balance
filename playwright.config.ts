import { defineConfig, devices } from '@playwright/test';

const PORT = Number(process.env.E2E_PORT ?? 8181);

export default defineConfig({
	testDir: 'e2e',
	// Tests share one server database, so run them one at a time.
	workers: 1,
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: process.env.CI ? [['html', { open: 'never' }], ['list']] : 'list',
	use: {
		baseURL: `http://localhost:${PORT}`,
		...devices['iPhone 13'],
		// The iPhone preset defaults to WebKit; run it in Chromium (what CI installs).
		browserName: 'chromium',
		serviceWorkers: 'block',
		trace: 'retain-on-failure'
	},
	webServer: {
		command: './scripts/e2e-server.sh',
		url: `http://localhost:${PORT}/health`,
		env: { PORT: String(PORT) },
		reuseExistingServer: false,
		timeout: 180_000
	}
});
