<script lang="ts">
  import { onMount } from 'svelte'
  import type { MatchResultsFile, MatchResult, SummaryMover } from '../types'
  import { pct } from '../format'
  import { deltaLabel, ratioLabel } from '../summary'

  type AppRoute = '/' | '/max-payout' | '/daily-summary'

  interface Props {
    id: string | null
    onNavigate?: (event: MouseEvent, to: AppRoute) => void
  }

  let { id, onNavigate }: Props = $props()

  let loading = $state(true)
  let result = $state<MatchResult | null>(null)

  // The /match page is the push-notification deep-link target, so it fetches the
  // match-results file directly (the dashboard doesn't keep it loaded). A missing
  // file or unknown id just shows the not-found panel.
  onMount(async () => {
    try {
      const res = await fetch(`/data/match-results.json?_=${Date.now()}`)
      if (res.ok) {
        const file: MatchResultsFile = await res.json()
        result = file.results?.find((r) => r.id === id) ?? null
      }
    } catch {
      // Network blip — fall through to the not-found state.
    } finally {
      loading = false
    }
  })

  const hasMovers = $derived(
    (result?.risers.length ?? 0) > 0 || (result?.fallers.length ?? 0) > 0,
  )
</script>

<main class="app-content match-page">
  <a class="back-link" href="/" onclick={(event) => onNavigate?.(event, '/')}>Back to dashboard</a>

  {#if loading}
    <p class="state-message">Loading…</p>
  {:else if !result}
    <section class="empty-panel">
      <h2>Match not found</h2>
      <p>We couldn't find the bet changes for this match. It may have rolled off the archive.</p>
    </section>
  {:else}
    <section class="match-card">
      <span class="match-group">{result.group}</span>
      <h1 class="match-score">
        <span class="team">{result.home}</span>
        <span class="score">{result.home_score}&ndash;{result.away_score}</span>
        <span class="team">{result.away}</span>
      </h1>
      <p class="match-paragraph">{result.paragraph}</p>

      {#if result.settled.length > 0}
        <div class="match-settled">
          {#each result.settled.filter((s) => s.status === 'won') as s (s.id)}
            <span class="settled-chip settled-won">★ {s.label}</span>
          {/each}
          {#each result.settled.filter((s) => s.status === 'lost') as s (s.id)}
            <span class="settled-chip settled-lost">✗ {s.label}</span>
          {/each}
        </div>
      {/if}
    </section>

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

    {#if hasMovers}
      <section class="movers-card">
        <p class="section-title">How the bets moved</p>
        <div class="match-movers">
          {@render moverTable('Biggest climbers', result.risers, 'up')}
          {@render moverTable('Biggest fallers', result.fallers, 'down')}
        </div>
      </section>
    {:else if result.settled.length === 0}
      <p class="state-message">This result didn't move the group's bets.</p>
    {/if}
  {/if}
</main>

<style>
  .match-page {
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

  .match-card,
  .movers-card,
  .empty-panel {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 8px;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    padding: 1.25rem;
  }

  .match-group {
    color: var(--wc-muted);
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .match-score {
    align-items: center;
    color: var(--wc-text);
    display: flex;
    flex-wrap: wrap;
    font-size: 1.6rem;
    font-weight: 800;
    gap: 0.5rem 1rem;
    margin-top: 0.4rem;
  }

  .match-score .score {
    background: var(--wc-navy);
    border-radius: 8px;
    color: var(--wc-white);
    padding: 0.1rem 0.7rem;
  }

  .match-paragraph {
    color: var(--wc-text);
    font-size: 1.05rem;
    line-height: 1.55;
    margin-top: 0.9rem;
  }

  .match-settled {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: 0.9rem;
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

  .section-title {
    color: var(--wc-muted);
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    margin-bottom: 0.8rem;
    text-transform: uppercase;
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

  .match-movers {
    display: grid;
    gap: 1.25rem;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
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
    align-items: flex-end;
    display: flex;
    flex-direction: column;
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
    .match-card,
    .movers-card,
    .empty-panel {
      padding: 1rem;
    }
  }
</style>
