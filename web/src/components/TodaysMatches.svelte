<script lang="ts">
  import type { MatchFixture } from '../types'
  import { getCountry } from '../countries'
  import Flag from './Flag.svelte'

  interface Props {
    matches: MatchFixture[]
  }

  let { matches }: Props = $props()

  // state.json carries fixtures for a UTC day either side of now; pick out the
  // ones kicking off on "today" in the browser's timezone. Recomputed every
  // poll (data is reassigned each cycle), so the list rolls over at midnight.
  const todays = $derived.by(() => {
    const now = new Date()
    return matches
      .filter((m) => {
        const d = new Date(m.utc_date)
        return (
          d.getFullYear() === now.getFullYear() &&
          d.getMonth() === now.getMonth() &&
          d.getDate() === now.getDate()
        )
      })
      .sort((a, b) => a.utc_date.localeCompare(b.utc_date))
  })

  // Kickoff in the browser's timezone and locale, e.g. "17:00" or "5:00 PM".
  function kickoff(iso: string): string {
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  type Badge = { text: string; kind: 'live' | 'ft' | 'note' } | null

  // Maps the provider's match status to a display badge; null (nothing to show
  // beyond the kickoff time) for matches that haven't started.
  function badge(status: string): Badge {
    switch (status) {
      case 'IN_PLAY':
        return { text: 'LIVE', kind: 'live' }
      case 'PAUSED':
        return { text: 'HT', kind: 'live' }
      case 'FINISHED':
        return { text: 'FT', kind: 'ft' }
      case 'SCHEDULED':
      case 'TIMED':
        return null
      default:
        // POSTPONED, SUSPENDED, CANCELLED, AWARDED, ...
        return { text: status.charAt(0) + status.slice(1).toLowerCase(), kind: 'note' }
    }
  }
</script>

{#if matches.length > 0}
  <section class="todays-matches">
    <h2 class="section-title">Today's Matches</h2>
    {#if todays.length === 0}
      <p class="no-matches">No matches today.</p>
    {:else}
      <div class="match-grid">
        {#each todays as m (m.id)}
          {@const b = badge(m.status)}
          <article class="fixture" class:fixture-live={b?.kind === 'live'}>
            <div class="fixture-meta">
              <span class="fixture-group">{m.group}</span>
              <span class="fixture-time">{kickoff(m.utc_date)}</span>
              {#if b}
                <span class="fixture-badge badge-{b.kind}">{b.text}</span>
              {/if}
            </div>
            <div class="fixture-team">
              <Flag fi={getCountry(m.home).fi} class="flag-inline" />
              <span class="team-name">{m.home}</span>
              {#if m.home_score != null}
                <span class="team-score">{m.home_score}</span>
              {/if}
            </div>
            <div class="fixture-team">
              <Flag fi={getCountry(m.away).fi} class="flag-inline" />
              <span class="team-name">{m.away}</span>
              {#if m.away_score != null}
                <span class="team-score">{m.away_score}</span>
              {/if}
            </div>
          </article>
        {/each}
      </div>
    {/if}
  </section>
{/if}

<style>
  .todays-matches {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .no-matches {
    color: var(--wc-muted);
    font-size: 0.9rem;
  }

  .match-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 0.75rem;
  }

  .fixture {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 0.75rem;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.75rem 0.9rem;
  }

  .fixture-live {
    border-color: var(--leg-alive-fg);
  }

  .fixture-meta {
    align-items: center;
    color: var(--wc-muted);
    display: flex;
    font-size: 0.7rem;
    font-weight: 700;
    gap: 0.5rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .fixture-time {
    margin-left: auto;
  }

  .fixture-badge {
    border-radius: 999px;
    font-size: 0.66rem;
    font-weight: 800;
    letter-spacing: 0.06em;
    padding: 0.1rem 0.45rem;
  }

  .badge-live {
    background: var(--leg-alive-bg);
    color: var(--leg-alive-fg);
  }

  .badge-ft {
    background: var(--wc-navy);
    color: var(--wc-white);
  }

  .badge-note {
    background: var(--leg-lost-bg);
    color: var(--leg-lost-fg);
    text-transform: none;
  }

  .fixture-team {
    align-items: center;
    display: flex;
    gap: 0.5rem;
    font-size: 0.95rem;
  }

  .team-name {
    font-weight: 700;
    min-width: 0;
    overflow-wrap: anywhere;
  }

  .team-score {
    color: var(--wc-navy);
    font-variant-numeric: tabular-nums;
    font-weight: 800;
    margin-left: auto;
  }
</style>
