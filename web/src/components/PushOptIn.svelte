<script lang="ts">
  import { onMount } from 'svelte'
  import {
    pushSupported,
    permissionState,
    currentSubscription,
    subscribe,
    unsubscribe,
  } from '../push'

  // Hidden entirely unless the browser supports Web Push and a VAPID key is
  // configured at build time, so an unconfigured deploy shows nothing.
  const supported = pushSupported()

  let subscribed = $state(false)
  let blocked = $state(false) // permission denied at the browser level
  let busy = $state(false)
  let error = $state<string | null>(null)

  onMount(async () => {
    if (!supported) return
    blocked = permissionState() === 'denied'
    try {
      subscribed = (await currentSubscription()) !== null
    } catch {
      // ignore — treat as not subscribed
    }
  })

  async function toggle() {
    if (busy) return
    busy = true
    error = null
    try {
      if (subscribed) {
        await unsubscribe()
        subscribed = false
      } else {
        await subscribe()
        subscribed = true
      }
      blocked = permissionState() === 'denied'
    } catch (e) {
      error = e instanceof Error ? e.message : 'Something went wrong'
      blocked = permissionState() === 'denied'
    } finally {
      busy = false
    }
  }
</script>

{#if supported}
  <section class="push-card" class:push-on={subscribed}>
    <div class="push-copy">
      <span class="push-bell" aria-hidden="true">{subscribed ? '🔔' : '🔕'}</span>
      <div>
        <p class="push-title">Match alerts</p>
        <p class="push-sub">
          {#if blocked}
            Notifications are blocked in your browser settings.
          {:else if subscribed}
            You'll get a push when each match finishes, with how the bets moved.
          {:else}
            Get a push when each match finishes, with how the bets moved.
          {/if}
        </p>
        {#if error}<p class="push-error">{error}</p>{/if}
      </div>
    </div>
    <button class="push-btn" onclick={toggle} disabled={busy || blocked}>
      {#if busy}…{:else if subscribed}Turn off{:else}Enable{/if}
    </button>
  </section>
{/if}

<style>
  .push-card {
    align-items: center;
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 0.75rem;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    display: flex;
    gap: 1rem;
    justify-content: space-between;
    padding: 0.9rem 1.25rem;
  }

  .push-on {
    border-color: var(--wc-navy);
  }

  .push-copy {
    align-items: center;
    display: flex;
    gap: 0.75rem;
    min-width: 0;
  }

  .push-bell {
    font-size: 1.5rem;
    line-height: 1;
  }

  .push-title {
    color: var(--wc-text);
    font-size: 0.95rem;
    font-weight: 800;
  }

  .push-sub {
    color: var(--wc-muted);
    font-size: 0.82rem;
    line-height: 1.35;
  }

  .push-error {
    color: var(--leg-lost-fg);
    font-size: 0.78rem;
    font-weight: 700;
    margin-top: 0.2rem;
  }

  .push-btn {
    background: var(--wc-navy);
    border: none;
    border-radius: 999px;
    color: var(--wc-white);
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 800;
    padding: 0.45rem 1.1rem;
    white-space: nowrap;
  }

  .push-btn:hover:not(:disabled),
  .push-btn:focus-visible:not(:disabled) {
    background: #001a4d;
  }

  .push-btn:disabled {
    cursor: default;
    opacity: 0.55;
  }

  @media (max-width: 640px) {
    .push-card {
      padding: 0.85rem 1rem;
    }
  }
</style>
