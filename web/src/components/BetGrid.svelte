<script lang="ts">
  import type { Bet, BetLeg, LegStatus, BetStatus } from '../types'
  import { getCountry } from '../countries'
  import { money, pct } from '../format'
  import Flag from './Flag.svelte'

  interface Props {
    bets: Bet[]
  }

  let { bets }: Props = $props()

  const STATUS_CLASS: Record<LegStatus, string> = {
    alive:   'leg-alive',
    won:     'leg-won',
    lost:    'leg-lost',
  }

  const STATUS_LABEL: Record<LegStatus, string> = {
    alive:   '✓',
    won:     '★',
    lost:    '✗',
  }

  const BET_STATUS_CLASS: Record<BetStatus, string> = {
    alive:   'bet-alive',
    won:     'bet-won',
    lost:    'bet-lost',
  }

  const BET_STATUS_LABEL: Record<BetStatus, string> = {
    alive:   'Alive',
    won:     'Won!',
    lost:    'Bust',
  }

  let allGroups = $derived(
    Array.from(new Set(bets.flatMap((b) => b.legs.map((l) => l.group)))).sort()
  )

  // Every accumulator is a variation on the same favourites: the pick chosen by
  // the most bets in each group is the "favourite", and the leg(s) where a bet
  // departs from it are what make that bet unique. We surface those as a row
  // label and highlight the cell so 20 near-identical rows can be told apart.
  let favByGroup = $derived.by(() => {
    const counts: Record<string, Record<string, number>> = {}
    for (const b of bets) {
      for (const l of b.legs) {
        counts[l.group] ??= {}
        counts[l.group][l.team] = (counts[l.group][l.team] ?? 0) + 1
      }
    }
    const fav: Record<string, string> = {}
    for (const [g, teams] of Object.entries(counts)) {
      fav[g] = Object.entries(teams).sort((a, b) => b[1] - a[1])[0][0]
    }
    return fav
  })

  function isPick(leg: BetLeg): boolean {
    return favByGroup[leg.group] != null && leg.team !== favByGroup[leg.group]
  }

  function betLabel(bet: Bet): string {
    const dev = bet.legs.filter(isPick)
    if (dev.length === 0) return 'Favourites'
    return dev.map((l) => `${l.group}: ${l.team}`).join(', ')
  }
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
        {#each bets as bet}
          {@const legByGroup = Object.fromEntries(bet.legs.map((l) => [l.group, l]))}
          <tr class="bet-row {BET_STATUS_CLASS[bet.status]}">
            <td class="col-bet-name">{betLabel(bet)}</td>
            {#each allGroups as g}
              {@const leg = legByGroup[g]}
              {#if !leg}
                <td class="leg-cell leg-empty"><span class="leg-na">—</span></td>
              {:else}
                {@const { fi, code } = getCountry(leg.team)}
                <td class="leg-cell {STATUS_CLASS[leg.status]} {isPick(leg) ? 'leg-pick' : ''}" title={leg.team}>
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

  <!-- Mobile cards -->
  <div class="bet-cards">
    {#each bets as bet}
      <div class="bet-card {BET_STATUS_CLASS[bet.status]}">
        <div class="bet-card-title">{betLabel(bet)}</div>
        <div class="bet-card-legs">
          {#each bet.legs as leg}
            {@const { fi, code } = getCountry(leg.team)}
            <div class="bet-card-leg {STATUS_CLASS[leg.status]} {isPick(leg) ? 'leg-pick' : ''}" title={leg.team}>
              <span class="bet-card-leg-group">Grp {leg.group}</span>
              <Flag {fi} class="bet-card-leg-flag" />
              <span class="bet-card-leg-code">{code}</span>
              <span class="bet-card-leg-icon">{STATUS_LABEL[leg.status]}</span>
            </div>
          {/each}
        </div>
        <div class="bet-card-footer">
          {#if bet.stake != null}
            <span class="bet-card-stake">{money(bet.stake)}</span>
          {/if}
          {#if bet.potential_return != null}
            <span class="bet-card-return">→ {money(bet.potential_return)}</span>
          {/if}
          {#if bet.probability != null}
            <span class="bet-card-chance">{pct(bet.probability)}</span>
          {/if}
          <span class="bet-card-status {BET_STATUS_CLASS[bet.status]}">
            {BET_STATUS_LABEL[bet.status]}
          </span>
        </div>
      </div>
    {/each}
  </div>
</section>
