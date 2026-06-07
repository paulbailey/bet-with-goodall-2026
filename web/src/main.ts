import { mount } from 'svelte'
import { registerSW } from 'virtual:pwa-register'
import 'flag-icons/css/flag-icons.min.css'
import './index.css'
import App from './App.svelte'

// Register the service worker so the site is installable and works offline.
// `immediate` activates a freshly-built SW as soon as it's ready (autoUpdate).
registerSW({ immediate: true })

mount(App, { target: document.getElementById('root')! })
