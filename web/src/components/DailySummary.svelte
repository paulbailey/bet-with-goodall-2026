<script lang="ts">
  import type { DailySummary } from '../types'
  import { formatDay } from '../summary'

  type AppRoute = '/' | '/max-payout' | '/daily-summary'

  interface Props {
    summary: DailySummary
    onNavigate?: (event: MouseEvent, to: AppRoute) => void
  }

  let { summary, onNavigate }: Props = $props()

  // A compact taste of the day's biggest swings; the full lists live on the
  // archive page.
  const topRisers = $derived(summary.risers.slice(0, 3))
  const topFallers = $derived(summary.fallers.slice(0, 3))
  const wins = $derived(summary.settled.filter((s) => s.status === 'won'))
  const busts = $derived(summary.settled.filter((s) => s.status === 'lost'))
</script>

<section class="daily-card">
  <div class="daily-head">
    <div>
      <p class="section-title">Daily Summary</p>
      <h2>{formatDay(summary.date)}</h2>
    </div>
    <a
      class="daily-link"
      href="/daily-summary"
      onclick={(event) => onNavigate?.(event, '/daily-summary')}
    >
      All summaries →
    </a>
  </div>

  <p class="daily-paragraph">{summary.paragraph}</p>

  {#if topRisers.length > 0 || topFallers.length > 0}
    <div class="daily-movers">
      {#if topRisers.length > 0}
        <div class="mover-col">
          <span class="mover-heading mover-up">Climbing</span>
          {#each topRisers as m (m.id)}
            <span class="mover-pill mover-up-pill">{m.label}</span>
          {/each}
        </div>
      {/if}
      {#if topFallers.length > 0}
        <div class="mover-col">
          <span class="mover-heading mover-down">Slipping</span>
          {#each topFallers as m (m.id)}
            <span class="mover-pill mover-down-pill">{m.label}</span>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  {#if wins.length > 0 || busts.length > 0}
    <div class="daily-settled">
      {#each wins as s (s.id)}
        <span class="settled-chip settled-won">★ {s.label}</span>
      {/each}
      {#each busts as s (s.id)}
        <span class="settled-chip settled-lost">✗ {s.label}</span>
      {/each}
    </div>
  {/if}
</section>

<style>
  .daily-card {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 0.75rem;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    display: flex;
    flex-direction: column;
    gap: 0.9rem;
    padding: 1.25rem;
  }

  .daily-head {
    align-items: start;
    display: flex;
    gap: 1rem;
    justify-content: space-between;
  }

  .section-title {
    color: var(--wc-muted);
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .daily-head h2 {
    color: var(--wc-text);
    font-size: 1.35rem;
    font-weight: 800;
    line-height: 1.1;
    margin-top: 0.3rem;
  }

  .daily-link {
    color: var(--wc-navy);
    font-size: 0.85rem;
    font-weight: 700;
    text-decoration: none;
    white-space: nowrap;
  }

  .daily-link:hover,
  .daily-link:focus-visible {
    text-decoration: underline;
  }

  .daily-paragraph {
    color: var(--wc-text);
    font-size: 1.02rem;
    line-height: 1.55;
  }

  .daily-movers {
    display: grid;
    gap: 1rem;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }

  .mover-col {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  .mover-heading {
    font-size: 0.68rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .mover-up {
    color: var(--leg-alive-fg);
  }
  .mover-down {
    color: var(--leg-lost-fg);
  }

  .mover-pill {
    border-radius: 8px;
    font-size: 0.8rem;
    font-weight: 700;
    line-height: 1.3;
    padding: 0.28rem 0.5rem;
  }

  .mover-up-pill {
    background: var(--leg-alive-bg);
    color: var(--leg-alive-fg);
  }
  .mover-down-pill {
    background: var(--leg-lost-bg);
    color: var(--leg-lost-fg);
  }

  .daily-settled {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
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

  @media (max-width: 640px) {
    .daily-card {
      padding: 1rem;
    }
    .daily-head {
      flex-direction: column;
      gap: 0.4rem;
    }
  }
</style>
