/// <reference lib="webworker" />
//
// Custom service worker (vite-plugin-pwa `injectManifest` strategy). It does two
// jobs:
//   1. Offline app shell — precache the built assets so the dashboard opens
//      instantly and still renders when the network is flaky at the pub.
//   2. Push notifications — receive match-result pushes from the builder and
//      surface them, then focus/open the site when one is tapped.
//
// The live data (`/data/*.json`) is deliberately *not* precached: it changes
// every poll, so it uses a network-first runtime cache that falls back to the
// last-seen copy offline.
import { precacheAndRoute, cleanupOutdatedCaches, type PrecacheEntry } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { NetworkFirst } from 'workbox-strategies'
import { ExpirationPlugin } from 'workbox-expiration'

declare const self: ServiceWorkerGlobalScope & {
  // Injected at build time by vite-plugin-pwa with the precache manifest.
  __WB_MANIFEST: (string | PrecacheEntry)[]
}

// __WB_MANIFEST is injected at build time with the content-hashed app assets.
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()

// Live tournament data: prefer the network (so a fresh poll always wins), but
// keep the last successful response so an offline launch shows the last grid we
// saw rather than an error. Short timeout keeps the UI snappy on a dead network.
registerRoute(
  ({ url }) => url.pathname.startsWith('/data/') && url.pathname.endsWith('.json'),
  new NetworkFirst({
    cacheName: 'tournament-data',
    networkTimeoutSeconds: 5,
    plugins: [new ExpirationPlugin({ maxEntries: 8, maxAgeSeconds: 60 * 60 * 24 })],
  }),
)

// autoUpdate registration: take over as soon as a new SW is ready so users get
// the latest shell without a manual refresh.
self.skipWaiting()
self.addEventListener('activate', () => self.clients.claim())

// ── Push notifications ────────────────────────────────────────────────────────

// Payload the builder sends with each match-result push. Everything is optional
// so a malformed or empty push still produces a sensible notification.
interface PushPayload {
  title?: string
  body?: string
  url?: string
  tag?: string
}

self.addEventListener('push', (event: PushEvent) => {
  let payload: PushPayload = {}
  if (event.data) {
    try {
      payload = event.data.json() as PushPayload
    } catch {
      payload = { body: event.data.text() }
    }
  }

  const title = payload.title || 'Bet With Goodall'
  const url = payload.url || '/'
  event.waitUntil(
    self.registration.showNotification(title, {
      body: payload.body || '',
      tag: payload.tag,
      icon: '/pwa-192x192.png',
      badge: '/pwa-192x192.png',
      data: { url },
    }),
  )
})

self.addEventListener('notificationclick', (event: NotificationEvent) => {
  event.notification.close()
  const target = (event.notification.data as { url?: string } | undefined)?.url || '/'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      // Focus an existing tab if the site is already open; otherwise open one.
      for (const client of clients) {
        if ('focus' in client) {
          client.navigate(target).catch(() => {})
          return client.focus()
        }
      }
      return self.clients.openWindow(target)
    }),
  )
})
