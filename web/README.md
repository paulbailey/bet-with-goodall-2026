# web

Svelte 5 + Vite frontend for the World Cup 2026 bet tracker. No SSR — the app
fetches `data/state.json` (and `data/daily-summary.json`) at runtime from the
same CloudFront origin and renders the bet grid.

## Develop

```bash
npm install
npm run dev      # vite dev server
npm run build    # production build → dist/
npm run check    # svelte-check (app) + tsc (service worker)
```

During development the live data files aren't served by Vite; the dashboard
shows a loading state until `data/state.json` exists. Point the builder at this
directory's `public/` (via `LOCAL_OUTPUT`) to populate them, or run against the
deployed JSON.

## PWA

The app is an installable PWA built with
[`vite-plugin-pwa`](https://vite-pwa-org.netlify.app/) using the
`injectManifest` strategy. The custom service worker lives in `src/sw.ts`:

- Precaches the app shell (Workbox) for offline use.
- Runtime-caches `data/*.json` network-first, so an offline launch shows the
  last grid we saw.
- Handles `push` / `notificationclick` for match-result notifications.

The service worker uses the `WebWorker` lib, which conflicts with the app's
`DOM` lib, so it's type-checked via its own `tsconfig.worker.json` and excluded
from the app `tsconfig.json`.

### App icons

`public/*.png` are rasterised from `icons/trophy.svg`. To regenerate after
editing the source:

```bash
npm i -D sharp && node scripts/gen-icons.mjs && npm uninstall sharp
```

### Push notifications

The "Match alerts" toggle (`PushOptIn.svelte`) subscribes the browser with the
VAPID public key and POSTs the subscription to `/api/subscribe`. The public key
comes from `VITE_VAPID_PUBLIC_KEY` at build time; when it's unset the toggle
hides itself and everything else works unchanged. See `builder/README.md` →
Push notifications for the end-to-end flow and key generation.
