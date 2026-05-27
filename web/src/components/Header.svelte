<script lang="ts">
  interface Props {
    updatedAt: string | null
    phase: string | null
  }

  let { updatedAt, phase }: Props = $props()

  const PHASE_LABELS: Record<string, string> = {
    pre_tournament: 'Pre-Tournament',
    group_stage: 'Group Stage',
    knockout: 'Knockout Rounds',
    complete: 'Tournament Complete',
  }

  function formatTimestamp(iso: string): string {
    return new Date(iso).toLocaleString(undefined, {
      dateStyle: 'medium',
      timeStyle: 'short',
    })
  }
</script>

<header class="site-header">
  <div class="header-inner">
    <div class="header-brand">
      <div class="header-trophy">⚽</div>
      <div>
        <h1 class="header-title">Bet With Goodall</h1>
        <p class="header-subtitle">FIFA World Cup 2026</p>
      </div>
    </div>
    <div class="header-meta">
      {#if phase}
        <span class="phase-badge">{PHASE_LABELS[phase] ?? phase}</span>
      {/if}
      {#if updatedAt}
        <span class="updated-at">Updated {formatTimestamp(updatedAt)}</span>
      {/if}
    </div>
  </div>
</header>
