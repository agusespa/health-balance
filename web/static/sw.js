// This is a minimal service worker to make the app installable.
// It doesn't provide offline capabilities.

self.addEventListener('install', (event) => {
  // console.log('Service Worker installing.');
});

self.addEventListener('activate', (event) => {
  // console.log('Service Worker activating.');
});

self.addEventListener('fetch', (event) => {
  // This service worker doesn't intercept any requests.
  return;
});

self.addEventListener('push', (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch (e) {
    console.log('Push received without JSON payload, using defaults.');
  }

  const title = data.title || 'Health Balance Reminder';
  const options = {
    body: data.body || "It's time to record your weekly metrics!",
    icon: '/static/icon.svg',
    badge: '/static/icon.svg',
    data: {
      url: data.url || '/'
    }
  };

  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const urlToOpen = new URL(event.notification.data.url, self.location.origin).href;

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
      for (let i = 0; i < windowClients.length; i++) {
        const client = windowClients[i];
        if (client.url === urlToOpen && 'focus' in client) {
          return client.focus();
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(urlToOpen);
      }
    })
  );
});
