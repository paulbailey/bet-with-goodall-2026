import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    svelte(),
    VitePWA({
      // A new build ships a new SW that takes over automatically (see
      // skipWaiting/clients.claim in src/sw.ts); the page picks up the update on
      // its next poll without a manual refresh.
      registerType: 'autoUpdate',
      // Hand-written SW (src/sw.ts) so we can add push + notificationclick
      // handlers on top of Workbox precaching.
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      injectManifest: {
        // Precache the app shell; the live data files are runtime-cached in sw.ts.
        globPatterns: ['**/*.{js,css,html,svg,png,woff2}'],
      },
      manifest: {
        name: 'Bet With Goodall — World Cup 2026',
        short_name: 'Bet With Goodall',
        description:
          "Live tracker for the group's shared FIFA World Cup 2026 accumulator bets.",
        theme_color: '#003087',
        background_color: '#003087',
        display: 'standalone',
        orientation: 'portrait',
        start_url: '/',
        scope: '/',
        icons: [
          { src: '/pwa-192x192.png', sizes: '192x192', type: 'image/png' },
          { src: '/pwa-512x512.png', sizes: '512x512', type: 'image/png' },
          {
            src: '/pwa-maskable-512x512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },
    }),
  ],
})
