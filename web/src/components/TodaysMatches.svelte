<script lang="ts">
  import type { MatchFixture } from '../types'
  import { getCountry } from '../countries'
  import Flag from './Flag.svelte'

  interface Props {
    matches: MatchFixture[]
  }

  let { matches }: Props = $props()

  // Matches are grouped into days by the tournament's local time, mirroring the
  // builder's SUMMARY_TZ default, so "today" means the same set of fixtures for
  // every visitor regardless of their timezone.
  const TOURNAMENT_TZ = 'America/New_York'

  // YYYY-MM-DD for a date in the given timezone (browser's own if omitted).
  function dayKey(date: Date, timeZone?: string): string {
    return new Intl.DateTimeFormat('en-CA', {
      timeZone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
    }).format(date)
  }

  // state.json carries fixtures for a UTC day either side of now (wide enough
  // to cover any tournament-local day); pick out the ones kicking off on
  // "today" at the tournament. Recomputed every poll (data is reassigned each
  // cycle), so the list rolls over at midnight.
  const todays = $derived.by(() => {
    const today = dayKey(new Date(), TOURNAMENT_TZ)
    return matches
      .filter((m) => dayKey(new Date(m.utc_date), TOURNAMENT_TZ) === today)
      .sort((a, b) => a.utc_date.localeCompare(b.utc_date))
  })

  // Kickoff in the browser's timezone and locale, e.g. "17:00" or "5:00 PM".
  function kickoff(iso: string): string {
    return new Date(iso).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  // A tournament-day evening match can land on a different calendar day in the
  // browser's timezone; flag it so the local kickoff time isn't misleading.
  function dayNote(iso: string): string | null {
    const matchDay = dayKey(new Date(iso))
    const now = new Date()
    if (matchDay === dayKey(now)) return null
    const DAY_MS = 24 * 60 * 60 * 1000
    if (matchDay === dayKey(new Date(now.getTime() + DAY_MS))) return 'tomorrow'
    if (matchDay === dayKey(new Date(now.getTime() - DAY_MS))) return 'yesterday'
    return new Date(iso).toLocaleDateString([], { weekday: 'short' })
  }

  type Badge = { text: string; kind: 'live' | 'ft' | 'note' } | null

  // Maps the provider's match status to a display badge; null (nothing to show
  // beyond the kickoff time) for matches that haven't started. In-play matches
  // show the game clock (e.g. 56') when the provider supplies it.
  function badge(m: MatchFixture): Badge {
    switch (m.status) {
      case 'IN_PLAY':
        return { text: m.minute != null ? `${m.minute}'` : 'LIVE', kind: 'live' }
      case 'PAUSED':
        return { text: 'HT', kind: 'live' }
      case 'FINISHED':
        return { text: 'FT', kind: 'ft' }
      case 'SCHEDULED':
      case 'TIMED':
        return null
      default:
        // POSTPONED, SUSPENDED, CANCELLED, AWARDED, ...
        return { text: m.status.charAt(0) + m.status.slice(1).toLowerCase(), kind: 'note' }
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
          {@const b = badge(m)}
          {@const note = dayNote(m.utc_date)}
          <article class="fixture" class:fixture-live={b?.kind === 'live'}>
            <div class="fixture-meta">
              <span class="fixture-group">{m.group}</span>
              {#if m.status !== 'FINISHED'}
                <span class="fixture-time">
                  {#if note}<span class="fixture-day-note">{note}</span>{/if}{kickoff(m.utc_date)}
                </span>
              {/if}
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

  /* Push everything after the group label (time, badge) to the right, even
     when the kickoff time is omitted for finished matches. */
  .fixture-group {
    margin-right: auto;
  }

  .fixture-day-note {
    color: var(--wc-muted);
    font-weight: 600;
    margin-right: 0.3rem;
    text-transform: none;
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
