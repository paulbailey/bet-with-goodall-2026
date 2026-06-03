<script lang="ts">
  import type { Bet } from '../types'
  import { getCountry } from '../countries'
  import { money, pct } from '../format'
  import {
    STATUS_CLASS,
    STATUS_LABEL,
    BET_STATUS_CLASS,
    BET_STATUS_LABEL,
    computeFavByGroup,
    isPick,
    betLabel,
    sortBetsForDisplay,
  } from '../bets'
  import Flag from './Flag.svelte'
  import BetCardList from './BetCardList.svelte'

  interface Props {
    bets: Bet[]
  }

  let { bets }: Props = $props()

  let allGroups = $derived(
    Array.from(new Set(bets.flatMap((b) => b.legs.map((l) => l.group)))).sort()
  )

  let favByGroup = $derived(computeFavByGroup(bets))

  // Live bets first (most likely leading), then won, then bust — same order as
  // the mobile list (BetCardList sorts identically).
  let sortedBets = $derived(sortBetsForDisplay(bets))
</script>

<section class="bet-grid-section">
  <h2 class="section-title">Accumulators</h2>

  <!-- Desktop table -->
  <div class="bet-grid-scroll">
    <table class="bet-grid">
      <thead>
        <tr>
          <th class="col-bet-name">Bet</th>
          {#each allGroups as g}
            <th class="col-group">Grp {g}</th>
          {/each}
          <th class="col-stake">Stake</th>
          <th class="col-return">Return</th>
          <th class="col-chance">Chance</th>
          <th class="col-status">Status</th>
        </tr>
      </thead>
      <tbody>
        {#each sortedBets as bet}
          {@const legByGroup = Object.fromEntries(bet.legs.map((l) => [l.group, l]))}
          <tr class="bet-row {BET_STATUS_CLASS[bet.status]}">
            <td class="col-bet-name">{betLabel(bet, favByGroup)}</td>
            {#each allGroups as g}
              {@const leg = legByGroup[g]}
              {#if !leg}
                <td class="leg-cell leg-empty"><span class="leg-na">—</span></td>
              {:else}
                {@const { fi, code } = getCountry(leg.team)}
                <td class="leg-cell {STATUS_CLASS[leg.status]} {isPick(leg, favByGroup) ? 'leg-pick' : ''}" title={leg.team}>
                  <Flag {fi} class="leg-flag" />
                  <span class="leg-full">{leg.team}</span>
                  <span class="leg-short">
                    <span class="leg-code">{code}</span>
                  </span>
                  <span class="leg-icon">{STATUS_LABEL[leg.status]}</span>
                </td>
              {/if}
            {/each}
            <td class="col-stake">
              {bet.stake != null ? money(bet.stake) : '—'}
            </td>
            <td class="col-return {bet.status === 'won' ? 'return-won' : ''}">
              {bet.potential_return != null ? money(bet.potential_return) : '—'}
            </td>
            <td class="col-chance" title={bet.expected_return != null ? `Expected return ${money(bet.expected_return)}` : ''}>
              {pct(bet.probability)}
            </td>
            <td class="col-status status-cell {BET_STATUS_CLASS[bet.status]}">
              {BET_STATUS_LABEL[bet.status]}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  <!-- Mobile collapsed list -->
  <BetCardList {bets} {favByGroup} />
</section>
