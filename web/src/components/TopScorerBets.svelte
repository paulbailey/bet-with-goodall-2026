<script lang="ts">
  import type { TopScorerBet, TopScorerBetStatus } from '../types'
  import { getCountry } from '../countries'
  import Flag from './Flag.svelte'

  interface Props {
    bets: TopScorerBet[]
  }

  let { bets }: Props = $props()

  const STATUS_CLASS: Record<TopScorerBetStatus, string> = {
    alive: 'bet-alive',
    won:   'bet-won',
    lost:  'bet-lost',
  }

  const STATUS_LABEL: Record<TopScorerBetStatus, string> = {
    alive: 'Alive',
    won:   'Won!',
    lost:  'Bust',
  }
</script>

<section class="scorer-bets-section">
  <h2 class="section-title">Top Scorer Bets</h2>
  <div class="bet-grid-scroll">
    <table class="bet-grid">
      <thead>
        <tr>
          <th class="col-ts-player">Player</th>
          <th class="col-ts-team">Team</th>
          <th class="col-stake">Stake</th>
          <th class="col-return">Return</th>
          <th class="col-status">Status</th>
        </tr>
      </thead>
      <tbody>
        {#each bets as bet}
          {@const { fi } = getCountry(bet.team)}
          <tr class="bet-row {STATUS_CLASS[bet.status]}">
            <td class="col-ts-player">{bet.player}</td>
            <td class="col-ts-team">
              <Flag {fi} class="team-flag" />
              {bet.team}
            </td>
            <td class="col-stake">
              {bet.stake != null ? `£${bet.stake.toFixed(2)}` : '—'}
            </td>
            <td class="col-return {bet.status === 'won' ? 'return-won' : ''}">
              {bet.potential_return != null ? `£${bet.potential_return.toFixed(2)}` : '—'}
            </td>
            <td class="col-status status-cell {STATUS_CLASS[bet.status]}">
              {STATUS_LABEL[bet.status]}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</section>
