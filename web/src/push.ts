// Web Push opt-in helpers. The site is static, so subscriptions are POSTed to a
// small API (DynamoDB-backed Lambda) exposed same-origin under /api/* via
// CloudFront. The builder reads that table and sends a push after each match.
//
// The VAPID public key is a build-time, non-secret value injected from the
// VITE_VAPID_PUBLIC_KEY env var (set in the deploy workflow). When it's absent
// the feature is treated as unconfigured and the UI hides itself.

const VAPID_PUBLIC_KEY = import.meta.env.VITE_VAPID_PUBLIC_KEY as string | undefined

// True only when the browser can do Web Push *and* a server key is configured.
export function pushSupported(): boolean {
  return (
    typeof navigator !== 'undefined' &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window &&
    Boolean(VAPID_PUBLIC_KEY)
  )
}

export function permissionState(): NotificationPermission {
  return Notification.permission
}

export async function currentSubscription(): Promise<PushSubscription | null> {
  const reg = await navigator.serviceWorker.ready
  return reg.pushManager.getSubscription()
}

// Request permission, subscribe with the VAPID key, and register the
// subscription with the backend. Throws if permission is denied.
export async function subscribe(): Promise<void> {
  const permission = await Notification.requestPermission()
  if (permission !== 'granted') {
    throw new Error('Notification permission was not granted')
  }
  const reg = await navigator.serviceWorker.ready
  const sub = await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(VAPID_PUBLIC_KEY as string),
  })
  await postJSON('/api/subscribe', sub.toJSON())
}

// Unregister from the backend, then drop the browser subscription. We tell the
// backend first so a failed local unsubscribe doesn't leave a stale row that
// keeps getting pushed to.
export async function unsubscribe(): Promise<void> {
  const sub = await currentSubscription()
  if (!sub) return
  await postJSON('/api/unsubscribe', { endpoint: sub.endpoint })
  await sub.unsubscribe()
}

async function postJSON(url: string, body: unknown): Promise<void> {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
}

// VAPID keys are URL-safe base64; the Push API wants the raw bytes. Backed by an
// explicit ArrayBuffer so the result satisfies BufferSource under TS's strict
// typed-array generics.
function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(base64)
  const output = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) output[i] = raw.charCodeAt(i)
  return output
}
