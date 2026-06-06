<script lang="ts">
  import type { DailySummaryFile, SummaryMover } from '../types'
  import { pct } from '../format'
  import { formatDay, deltaLabel, ratioLabel } from '../summary'

  type AppRoute = '/' | '/max-payout' | '/daily-summary'

  interface Props {
    file: DailySummaryFile | null
    onNavigate?: (event: MouseEvent, to: AppRoute) => void
  }

  let { file, onNavigate }: Props = $props()

  const summaries = $derived(file?.summaries ?? [])
</script>

<main class="app-content daily-page">
  <a class="back-link" href="/" onclick={(event) => onNavigate?.(event, '/')}>Back to dashboard</a>

  {#if summaries.length === 0}
    <section class="empty-panel">
      <h2>No daily summaries yet</h2>
      <p>A summary is generated after each day's matches have finished.</p>
    </section>
  {:else}
    {#snippet moverTable(title: string, movers: SummaryMover[], kind: 'up' | 'down')}
      <div class="mover-table">
        <h4 class="mover-title mover-{kind}">{title}</h4>
        {#if movers.length === 0}
          <p class="mover-empty">Nothing notable.</p>
        {:else}
          {#each movers as m (m.id)}
            <div class="mover-row">
              <div class="mover-label">
                <span>{m.label}</span>
                <span class="mover-cat">{m.category}</span>
              </div>
              <div class="mover-figs">
                <span class="mover-prob">{pct(m.prev_prob)} → {pct(m.new_prob)}</span>
                <span class="mover-change mover-{kind}">{ratioLabel(m)} · {deltaLabel(m)}</span>
              </div>
            </div>
          {/each}
        {/if}
      </div>
    {/snippet}

    {#each summaries as summary (summary.date)}
      <section class="day-card">
        <div class="day-head">
          <h2>{formatDay(summary.date)}</h2>
          <span class="day-tracked">{summary.bets_tracked} bets tracked</span>
        </div>

        <p class="day-paragraph">{summary.paragraph}</p>

        {#if summary.settled.length > 0}
          <div class="day-settled">
            {#each summary.settled.filter((s) => s.status === 'won') as s (s.id)}
              <span class="settled-chip settled-won">★ {s.label}</span>
            {/each}
            {#each summary.settled.filter((s) => s.status === 'lost') as s (s.id)}
              <span class="settled-chip settled-lost">✗ {s.label}</span>
            {/each}
          </div>
        {/if}

        <div class="day-movers">
          {@render moverTable('Biggest climbers', summary.risers, 'up')}
          {@render moverTable('Biggest fallers', summary.fallers, 'down')}
        </div>
      </section>
    {/each}
  {/if}
</main>

<style>
  .daily-page {
    gap: 1.25rem;
  }

  .back-link {
    align-self: flex-start;
    color: var(--wc-navy);
    font-size: 0.9rem;
    font-weight: 700;
    text-decoration: none;
  }

  .back-link:hover,
  .back-link:focus-visible {
    text-decoration: underline;
  }

  .day-card,
  .empty-panel {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 8px;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    padding: 1.25rem;
  }

  .empty-panel h2 {
    color: var(--wc-navy);
    font-size: 1.2rem;
    margin-bottom: 0.35rem;
  }

  .empty-panel p {
    color: var(--wc-muted);
    font-size: 0.9rem;
  }

  .day-head {
    align-items: baseline;
    display: flex;
    gap: 1rem;
    justify-content: space-between;
  }

  .day-head h2 {
    color: var(--wc-text);
    font-size: 1.35rem;
    font-weight: 800;
  }

  .day-tracked {
    color: var(--wc-muted);
    font-size: 0.78rem;
    font-weight: 700;
    white-space: nowrap;
  }

  .day-paragraph {
    color: var(--wc-text);
    font-size: 1.02rem;
    line-height: 1.55;
    margin-top: 0.6rem;
  }

  .day-settled {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: 0.8rem;
  }

  .settled-chip {
    border-radius: 999px;
    font-size: 0.76rem;
    font-weight: 800;
    padding: 0.2rem 0.55rem;
  }

  .settled-won {
    background: var(--leg-won-bg);
    color: var(--leg-won-fg);
  }
  .settled-lost {
    background: var(--leg-lost-bg);
    color: var(--leg-lost-fg);
  }

  .day-movers {
    display: grid;
    gap: 1.25rem;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    margin-top: 1rem;
  }

  .mover-title {
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    margin-bottom: 0.6rem;
    text-transform: uppercase;
  }

  .mover-up {
    color: var(--leg-alive-fg);
  }
  .mover-down {
    color: var(--leg-lost-fg);
  }

  .mover-empty {
    color: var(--wc-muted);
    font-size: 0.85rem;
  }

  .mover-row {
    align-items: center;
    border-top: 1px solid var(--wc-border);
    display: flex;
    gap: 0.75rem;
    justify-content: space-between;
    padding: 0.5rem 0;
  }

  .mover-label {
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
    min-width: 0;
  }

  .mover-label > span:first-child {
    color: var(--wc-text);
    font-size: 0.9rem;
    font-weight: 700;
    overflow-wrap: anywhere;
  }

  .mover-cat {
    color: var(--wc-muted);
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
  }

  .mover-figs {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 0.15rem;
    white-space: nowrap;
  }

  .mover-prob {
    color: var(--wc-text);
    font-size: 0.85rem;
    font-weight: 700;
  }

  .mover-change {
    font-size: 0.78rem;
    font-weight: 800;
  }

  @media (max-width: 640px) {
    .day-card,
    .empty-panel {
      padding: 1rem;
    }
  }
</style>
