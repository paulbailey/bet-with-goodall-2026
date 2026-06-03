<script lang="ts">
  import type { Bet } from '../types'
  import { money } from '../format'
  import { sortBetsForDisplay } from '../bets'
  import BetCardRow from './BetCardRow.svelte'

  interface Props {
    bets: Bet[]
    favByGroup: Record<string, string>
  }

  let { bets, favByGroup }: Props = $props()

  // Live bets first (most likely leading), then won, then bust — shared with the
  // desktop table so both views agree.
  let sortedBets = $derived(sortBetsForDisplay(bets))

  let counts = $derived.by(() => {
    let alive = 0, won = 0, lost = 0
    for (const b of bets) {
      if (b.status === 'alive') alive++
      else if (b.status === 'won') won++
      else lost++
    }
    return { alive, won, lost }
  })

  let totalStake = $derived(
    bets.reduce((sum, b) => sum + (b.stake ?? 0), 0)
  )
  // Total return only counts bets that can still pay out — alive + won.
  let liveReturn = $derived(
    bets.reduce((sum, b) => sum + (b.status !== 'lost' ? b.potential_return ?? 0 : 0), 0)
  )

  // Index of the first bust bet in the sorted list, so we can slot a divider
  // before the dead band. -1 when nothing is bust.
  let firstLostIndex = $derived(sortedBets.findIndex((b) => b.status === 'lost'))
</script>

<div class="bet-list">
  <div class="bet-list-summary">
    <span class="bet-list-counts">
      <span class="count-alive">{counts.alive} alive</span>
      {#if counts.won > 0}
        <span class="count-won">{counts.won} won</span>
      {/if}
      <span class="count-lost">{counts.lost} bust</span>
      <span class="count-total">of {bets.length}</span>
    </span>
    <span class="bet-list-totals">
      {money(totalStake)} → {money(liveReturn)}
    </span>
  </div>

  {#each sortedBets as bet, i (bet.id)}
    {#if i === firstLostIndex && firstLostIndex > 0}
      <div class="bet-list-divider">Bust ({counts.lost})</div>
    {/if}
    <BetCardRow {bet} {favByGroup} />
  {/each}
</div>
