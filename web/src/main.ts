import { mount } from 'svelte'
import { registerSW } from 'virtual:pwa-register'
import 'flag-icons/css/flag-icons.min.css'
import './index.css'
import App from './App.svelte'

// Register the service worker so the site is installable and works offline.
// `immediate` activates a freshly-built SW as soon as it's ready (autoUpdate).
//
// Installed PWAs need extra nudging to pick up new deploys: a cold start paints
// from the old SW's precache before the update check finishes, and a resume
// from the app switcher never navigates, so the browser doesn't check at all.
// Without the two hooks below an installed app can run stale code indefinitely
// while normal browser tabs update on every visit.
if ('serviceWorker' in navigator) {
  // When a new SW takes control mid-session (skipWaiting + clients.claim in
  // sw.ts), reload so the page runs the new shell instead of finishing the
  // session on old code. Skip the very first claim of an uncontrolled page —
  // reloading there would add a pointless flash on a user's first visit.
  let hadController = Boolean(navigator.serviceWorker.controller)
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (!hadController) {
      hadController = true
      return
    }
    window.location.reload()
  })
}

registerSW({
  immediate: true,
  onRegisteredSW(_swUrl, registration) {
    if (!registration) return
    // Re-fetch sw.js (deployed with no-cache) whenever the app returns to the
    // foreground, and hourly for sessions left open across match days.
    const check = () => registration.update().catch(() => {})
    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState === 'visible') check()
    })
    setInterval(check, 60 * 60 * 1000)
  },
})

mount(App, { target: document.getElementById('root')! })
