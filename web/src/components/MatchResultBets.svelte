<script lang="ts">
  import type { MatchResultBet, BetStatus } from '../types'
  import { getCountry } from '../countries'
  import { money, pct } from '../format'
  import Flag from './Flag.svelte'

  interface Props {
    bets: MatchResultBet[]
  }

  let { bets }: Props = $props()

  const STATUS_CLASS: Record<BetStatus, string> = {
    alive:   'bet-alive',
    won:     'bet-won',
    lost:    'bet-lost',
  }

  const STATUS_LABEL: Record<BetStatus, string> = {
    alive:   'Alive',
    won:     'Won!',
    lost:    'Bust',
  }
</script>

<section class="match-result-section">
  <h2 class="section-title">Match Result Bets</h2>
  <div class="bet-grid-scroll">
    <table class="bet-grid">
      <thead>
        <tr>
          <th class="col-ts-team">Match</th>
          <th class="col-score">Predicted</th>
          <th class="col-score">Actual</th>
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
          {@const hasActual = bet.actual_a != null && bet.actual_b != null}
          <tr class="bet-row {STATUS_CLASS[bet.status]}">
            <td class="col-ts-team">
              <Flag fi={a.fi} class="team-flag" />{a.code}
              <span class="mr-vs">v</span>
              <Flag fi={b.fi} class="team-flag" />{b.code}
            </td>
            <td class="col-score">{bet.score_a}–{bet.score_b}</td>
            <td class="col-score">
              {hasActual ? `${bet.actual_a}–${bet.actual_b}` : '—'}
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
  .mr-vs { color: var(--wc-muted); margin: 0 0.15rem; font-size: 0.75rem; }
</style>
