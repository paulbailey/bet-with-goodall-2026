<script lang="ts">
  import type { TopScorer, TopScorerBet } from '../types'
  import { getCountry } from '../countries'
  import Flag from './Flag.svelte'

  interface Props {
    scorers: TopScorer[]
    bets: TopScorerBet[]
  }

  let { scorers, bets }: Props = $props()
  let open = $state(true)

  let predictedPlayers = $derived(new Set(bets.map((b) => b.player)))
  let sorted = $derived([...scorers].sort((a, b) => b.goals - a.goals))
</script>

<section class="standings-section">
  <button class="section-toggle" onclick={() => (open = !open)}>
    <h2 class="section-title">Top Scorers</h2>
    <span class="toggle-icon">{open ? '▲' : '▼'}</span>
  </button>
  {#if open}
    <div class="top-scorers-wrap">
      <table class="top-scorers-table">
        <thead>
          <tr>
            <th class="col-rank">#</th>
            <th class="col-ts-player">Player</th>
            <th class="col-ts-team">Team</th>
            <th>Goals</th>
            <th>Games</th>
          </tr>
        </thead>
        <tbody>
          {#each sorted as s, i}
            {@const { fi } = getCountry(s.team)}
            {@const isPredicted = predictedPlayers.has(s.player)}
            <tr class={[s.team_eliminated ? 'scorer-eliminated' : '', isPredicted ? 'scorer-predicted' : ''].filter(Boolean).join(' ')}>
              <td class="col-rank">{i + 1}</td>
              <td class="col-ts-player">
                {s.player}
                {#if isPredicted}
                  <span class="scorer-bet-marker" title="Bet placed on this player"> ★</span>
                {/if}
              </td>
              <td class="col-ts-team">
                <Flag {fi} class="team-flag" />
                {s.team}
              </td>
              <td class="col-goals">{s.goals}</td>
              <td>{s.games}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>
