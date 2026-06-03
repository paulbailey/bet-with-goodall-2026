<script lang="ts">
  import type { Bet } from '../types'
  import { getCountry } from '../countries'
  import { money, pct } from '../format'
  import {
    STATUS_CLASS,
    STATUS_LABEL,
    BET_STATUS_CLASS,
    BET_STATUS_LABEL,
    isPick,
    deviations,
    lostLegs,
  } from '../bets'
  import Flag from './Flag.svelte'

  interface Props {
    bet: Bet
    favByGroup: Record<string, string>
  }

  let { bet, favByGroup }: Props = $props()
  let open = $state(false)

  // Bust bets lead with the leg(s) that killed them — the label alone may just
  // say "Favourites". Alive/won bets lead with the picks that make them unique.
  let lost = $derived(lostLegs(bet))
  let picks = $derived(deviations(bet, favByGroup))
</script>

<div class="bet-row {BET_STATUS_CLASS[bet.status]}">
  <button class="bet-row-toggle" aria-expanded={open} onclick={() => (open = !open)}>
    <span class="bet-row-chips">
      {#if bet.status === 'lost'}
        {#each lost as leg}
          {@const { fi, code } = getCountry(leg.team)}
          <span class="bet-chip bet-chip-lost" title={leg.team}>
            <span class="bet-chip-group">{leg.group}</span>
            <Flag {fi} class="bet-chip-flag" />
            <span class="bet-chip-code">{code}</span>
            <span class="bet-chip-icon">{STATUS_LABEL.lost}</span>
          </span>
        {/each}
      {:else if picks.length > 0}
        {#each picks as leg}
          {@const { fi, code } = getCountry(leg.team)}
          <span class="bet-chip bet-chip-pick" title={leg.team}>
            <span class="bet-chip-group">{leg.group}</span>
            <Flag {fi} class="bet-chip-flag" />
            <span class="bet-chip-code">{code}</span>
          </span>
        {/each}
      {:else}
        <span class="bet-chip-label">Favourites</span>
      {/if}
    </span>

    <span class="bet-row-meta">
      {#if bet.potential_return != null}
        <span class="bet-row-return">{money(bet.potential_return)}</span>
      {/if}
      {#if bet.probability != null}
        <span class="bet-row-chance">{pct(bet.probability)}</span>
      {/if}
      <span class="bet-row-status">{BET_STATUS_LABEL[bet.status]}</span>
    </span>

    <span class="bet-row-chevron" aria-hidden="true">{open ? '▾' : '▸'}</span>
  </button>

  {#if open}
    <div class="bet-card-legs">
      {#each bet.legs as leg}
        {@const { fi, code } = getCountry(leg.team)}
        <div
          class="bet-card-leg {STATUS_CLASS[leg.status]} {isPick(leg, favByGroup) ? 'leg-pick' : ''}"
          title={leg.team}
        >
          <span class="bet-card-leg-group">Grp {leg.group}</span>
          <Flag {fi} class="bet-card-leg-flag" />
          <span class="bet-card-leg-code">{code}</span>
          <span class="bet-card-leg-icon">{STATUS_LABEL[leg.status]}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>
