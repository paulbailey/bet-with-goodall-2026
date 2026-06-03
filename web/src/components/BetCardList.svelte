<script lang="ts">
  import type { Bet, BetStatus } from '../types'
  import { money } from '../format'
  import BetCardRow from './BetCardRow.svelte'

  interface Props {
    bets: Bet[]
    favByGroup: Record<string, string>
  }

  let { bets, favByGroup }: Props = $props()

  // Surface the bets that still matter: alive on top, won next, bust at the
  // bottom (also dimmed). Within the alive band, order by descending chance so
  // the most likely live bets lead; unpriced bets (no probability) sink last.
  // Won/bust bands keep their original order (their chances are ~all 1 or 0).
  const RANK: Record<BetStatus, number> = { alive: 0, won: 1, lost: 2 }
  let sortedBets = $derived(
    [...bets].sort((a, b) => {
      if (RANK[a.status] !== RANK[b.status]) return RANK[a.status] - RANK[b.status]
      if (a.status === 'alive') return (b.probability ?? -1) - (a.probability ?? -1)
      return 0
    })
  )

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
