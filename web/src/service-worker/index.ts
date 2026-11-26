/**
 * Instant launch: precache the app shell and build output, and answer page navigations from the
 * cached shell so the home-screen app paints without waiting on Cloudflare Access and the tunnel.
 * A release is a new build version, so it installs a new worker with a fresh shell; the next
 * launch picks it up.
 *
 * Every route is the same SPA shell, so the cached / answers deep links and reloads too.
 * Navigations with ?signin skip the cache, so "Sign in again" reaches the Access login page.
 * /api is never intercepted: scores, settings and the ingest endpoint always reach the server.
 */
import { self } from '$app/service-worker';
import { assets, immutable } from '$app/manifest';
import { dev, version } from '$app/env';

const CACHE = `health-balance-${version}`;
const SHELL = new URL('./', self.registration.scope).href;
const PRECACHE = [
	SHELL,
	...immutable.map((f) => new URL(f.path, self.registration.scope).href),
	...assets.map((f) => new URL(f.path, self.registration.scope).href)
];

self.addEventListener('install', (event) => {
	event.waitUntil(
		(async () => {
			const cache = await caches.open(CACHE);
			await cache.addAll(PRECACHE);
			await self.skipWaiting();
		})()
	);
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			for (const key of await caches.keys()) {
				if (key.startsWith('health-balance-') && key !== CACHE) await caches.delete(key);
			}
			await self.clients.claim();
		})()
	);
});

self.addEventListener('fetch', (event) => {
	// Never serve cached modules to the dev server.
	if (dev) return;
	const req = event.request;
	if (req.method !== 'GET') return;
	const url = new URL(req.url);
	if (url.origin !== self.location.origin || url.pathname.startsWith('/api/')) return;
	if (url.pathname === '/_app/version.json') return;

	if (req.mode === 'navigate') {
		// Pass sign-in navigations straight to the browser, redirects to the Access login included.
		if (url.searchParams.has('signin')) return;
		event.respondWith(
			(async () => {
				const cached = await caches.match(SHELL);
				return cached ?? fetch(req);
			})()
		);
		return;
	}

	event.respondWith(
		(async () => {
			const cached = await caches.match(req);
			if (cached) return cached;
			const res = await fetch(req);
			if (res.ok && res.type === 'basic') {
				const cache = await caches.open(CACHE);
				await cache.put(req, res.clone());
			}
			return res;
		})()
	);
});
