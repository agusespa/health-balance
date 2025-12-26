// This is a minimal service worker to make the app installable.
// It doesn't provide offline capabilities.

self.addEventListener('install', (event) => {
  // console.log('Service Worker installing.');
});

self.addEventListener('activate', (event) => {
  // console.log('Service Worker activating.');
});

self.addEventListener('fetch', (event) => {
  // console.log('Fetching:', event.request.url);
  // This service worker doesn't intercept any requests.
  // It just lets the browser handle them as usual.
  return;
});
