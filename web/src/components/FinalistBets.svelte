<script lang="ts">
  import type { FinalistBet, FinalistBetStatus } from '../types'
  import { getCountry } from '../countries'
  import { money, pct } from '../format'
  import Flag from './Flag.svelte'

  interface Props {
    bets: FinalistBet[]
  }

  let { bets }: Props = $props()

  const STATUS_CLASS: Record<FinalistBetStatus, string> = {
    alive: 'bet-alive',
    won:   'bet-won',
    lost:  'bet-lost',
  }

  const STATUS_LABEL: Record<FinalistBetStatus, string> = {
    alive: 'Alive',
    won:   'Won!',
    lost:  'Bust',
  }
</script>

<section class="winner-bets-section">
  <h2 class="section-title">Finalist Bets</h2>
  <div class="bet-grid-scroll">
    <table class="bet-grid">
      <thead>
        <tr>
          <th class="col-ts-team">Final</th>
          <th class="col-stake">Stake</th>
          <th class="col-return">Return</th>
          <th class="col-chance">Chance</th>
          <th class="col-status">Status</th>
        </tr>
      </thead>
      <tbody>
        {#each bets as bet}
          {@const a = getCountry(bet.team_a)}
          {@const b = getCountry(bet.team_b)}
          <tr class="bet-row {STATUS_CLASS[bet.status]}">
            <td class="col-ts-team">
              <Flag fi={a.fi} class="team-flag" />{bet.team_a}
              <span class="fin-vs">vs</span>
              <Flag fi={b.fi} class="team-flag" />{bet.team_b}
            </td>
            <td class="col-stake">
              {bet.stake != null ? money(bet.stake) : '—'}
            </td>
            <td class="col-return {bet.status === 'won' ? 'return-won' : ''}">
              {bet.potential_return != null ? money(bet.potential_return) : '—'}
            </td>
            <td class="col-chance" title={bet.expected_return != null ? `Expected return ${money(bet.expected_return)}` : ''}>
              {pct(bet.probability)}
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

<style>
  .fin-vs { color: var(--wc-muted); margin: 0 0.3rem; font-size: 0.8rem; }
</style>
