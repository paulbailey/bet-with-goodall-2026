<script lang="ts">
  import type { MaxPayout } from '../types'
  import { money } from '../format'

  interface Props {
    maxPayout: MaxPayout
    labelByBetID?: Record<string, string>
  }

  let { maxPayout, labelByBetID = {} }: Props = $props()

  const conflicts = $derived(maxPayout.conflicts ?? [])

  function displayLabel(label: string): string {
    return label.replace(/\ba[c]c[a]\b/gi, 'accumulator')
  }

  function betLabel(id: string, fallback: string): string {
    return labelByBetID[id] ?? displayLabel(fallback)
  }
</script>

{#if conflicts.length > 0}
  <section class="conflicts-section">
    <h2 class="section-title">Mutually Exclusive Bets</h2>
    <p class="conflicts-intro">
      These bets contradict each other, so they can't all win. The max payout
      counts only the best combination from each group.
    </p>
    <div class="conflicts-grid">
      {#each conflicts as group}
        <div class="conflict-card">
          {#each group.bets as bet (bet.id)}
            <div class="conflict-bet" class:dropped={!bet.chosen}>
              <span class="conflict-mark" aria-hidden="true">{bet.chosen ? '✓' : '✕'}</span>
              <span class="conflict-label">
                {betLabel(bet.id, bet.label)}
                {#if bet.status === 'won'}<span class="conflict-won-tag">won</span>{/if}
              </span>
              <span class="conflict-return">{money(bet.return)}</span>
              <span class="conflict-tag">{bet.chosen ? 'counted' : 'not counted'}</span>
            </div>
          {/each}
        </div>
      {/each}
    </div>
  </section>
{/if}

<style>
  .conflicts-intro {
    margin-top: 0.4rem;
    color: var(--wc-muted);
    font-size: 0.85rem;
    max-width: 60ch;
  }

  .conflicts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 1rem;
    margin-top: 0.75rem;
  }

  .conflict-card {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 0.75rem;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    overflow: hidden;
  }

  .conflict-bet {
    display: grid;
    grid-template-columns: 1.25rem 1fr auto;
    grid-template-areas:
      'mark label return'
      'mark tag   tag';
    align-items: center;
    column-gap: 0.6rem;
    row-gap: 0.1rem;
    padding: 0.65rem 0.85rem;
    border-bottom: 1px solid var(--wc-border);
  }

  .conflict-bet:last-child { border-bottom: none; }

  .conflict-mark {
    grid-area: mark;
    font-weight: 800;
    font-size: 0.95rem;
    color: var(--leg-alive-fg);
  }

  .conflict-label {
    grid-area: label;
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--wc-text);
    min-width: 0;
    overflow-wrap: break-word;
  }

  .conflict-return {
    grid-area: return;
    font-weight: 700;
    font-size: 0.875rem;
    color: var(--wc-navy);
    white-space: nowrap;
  }

  .conflict-tag {
    grid-area: tag;
    font-size: 0.68rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--leg-alive-fg);
  }

  .conflict-won-tag {
    margin-left: 0.4rem;
    font-size: 0.62rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #b45309;
    background: var(--leg-won-bg);
    border-radius: 999px;
    padding: 0.05rem 0.4rem;
    vertical-align: middle;
  }

  .conflict-bet.dropped { background: var(--wc-light); }
  .conflict-bet.dropped .conflict-mark,
  .conflict-bet.dropped .conflict-tag { color: var(--leg-lost-fg); }
  .conflict-bet.dropped .conflict-label,
  .conflict-bet.dropped .conflict-return {
    color: var(--wc-muted);
    text-decoration: line-through;
  }
</style>
