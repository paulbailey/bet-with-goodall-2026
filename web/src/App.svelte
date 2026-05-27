<script lang="ts">
  import { onMount } from 'svelte'
  import type { TournamentState } from './types'
  import Header from './components/Header.svelte'
  import BetGrid from './components/BetGrid.svelte'
  import TournamentWinnerBets from './components/TournamentWinnerBets.svelte'
  import TopScorerBets from './components/TopScorerBets.svelte'
  import TopScorers from './components/TopScorers.svelte'
  import GroupStandings from './components/GroupStandings.svelte'

  const POLL_INTERVAL_MS = 60_000

  let data = $state<TournamentState | null>(null)
  let error = $state<string | null>(null)
  let lastUpdatedAt: string | null = null

  async function fetchState() {
    try {
      const res = await fetch(`/data/state.json?_=${Date.now()}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const json: TournamentState = await res.json()
      if (json.updated_at !== lastUpdatedAt) {
        lastUpdatedAt = json.updated_at
        data = json
        error = null
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load data'
    }
  }

  onMount(() => {
    fetchState()
    const id = setInterval(fetchState, POLL_INTERVAL_MS)
    return () => clearInterval(id)
  })
</script>

<Header updatedAt={data?.updated_at ?? null} phase={data?.tournament_phase ?? null} />

{#if error}
  <p class="state-message error">Failed to load data: {error}</p>
{:else if !data}
  <p class="state-message">Loading…</p>
{:else}
  <main class="app-content">
    <BetGrid bets={data.bets} />
    {#if data.tournament_winner_bets.length > 0}
      <TournamentWinnerBets bets={data.tournament_winner_bets} />
    {/if}
    {#if data.top_scorer_bets.length > 0 || data.top_scorers.length > 0}
      <div class="scorer-panels">
        {#if data.top_scorer_bets.length > 0}
          <TopScorerBets bets={data.top_scorer_bets} />
        {/if}
        {#if data.top_scorers.length > 0}
          <TopScorers scorers={data.top_scorers} bets={data.top_scorer_bets} />
        {/if}
      </div>
    {/if}
    <GroupStandings groups={data.groups} />
  </main>
{/if}
