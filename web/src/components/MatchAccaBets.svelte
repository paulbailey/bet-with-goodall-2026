<script lang="ts">
  import type { MatchAccaBet, BetStatus, LegStatus, MatchOutcome } from '../types'
  import { getCountry } from '../countries'
  import Flag from './Flag.svelte'

  interface Props {
    bets: MatchAccaBet[]
  }

  let { bets }: Props = $props()

  const LEG_STATUS_CLASS: Record<LegStatus, string> = {
    alive:   'leg-alive',
    won:     'leg-won',
    lost:    'leg-lost',
  }

  const LEG_STATUS_LABEL: Record<LegStatus, string> = {
    alive:   '✓',
    won:     '★',
    lost:    '✗',
  }

  const OUTCOME_LABEL: Record<MatchOutcome, string> = {
    win:  'to beat',
    draw: 'to draw',
    lose: 'to lose to',
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
</script>

<section class="acca-bets-section">
  <h2 class="section-title">Match Accumulators</h2>
  <div class="acca-cards">
    {#each bets as bet}
      <div class="acca-card {BET_STATUS_CLASS[bet.status]}">
        <div class="acca-legs">
          {#each bet.legs as leg}
            {@const team = getCountry(leg.team)}
            {@const opp = getCountry(leg.opponent)}
            <span class="acca-leg {LEG_STATUS_CLASS[leg.status]}" title="{leg.team} {OUTCOME_LABEL[leg.outcome]} {leg.opponent}">
              <Flag fi={team.fi} class="acca-flag" />
              <span class="acca-leg-text">{team.code} {OUTCOME_LABEL[leg.outcome]} {opp.code}</span>
              <span class="acca-leg-icon">{LEG_STATUS_LABEL[leg.status]}</span>
            </span>
          {/each}
        </div>
        <div class="acca-footer">
          {#if bet.stake != null}
            <span class="acca-stake">£{bet.stake.toFixed(2)}</span>
          {/if}
          {#if bet.potential_return != null}
            <span class="acca-return">→ £{bet.potential_return.toFixed(2)}</span>
          {/if}
          <span class="acca-status {BET_STATUS_CLASS[bet.status]}">
            {BET_STATUS_LABEL[bet.status]}
          </span>
        </div>
      </div>
    {/each}
  </div>
</section>

<style>
  .acca-cards { display: flex; flex-direction: column; gap: 0.75rem; margin-top: 0.75rem; }
  .acca-card {
    border: 1px solid var(--wc-border, #e5e7eb);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--wc-white, #fff);
  }
  .acca-card.bet-lost { opacity: 0.65; }
  .acca-card.bet-won  { background: #fffbeb; }
  .acca-legs { display: flex; flex-wrap: wrap; gap: 0.4rem; }
  .acca-leg {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.2rem 0.5rem;
    border-radius: 999px;
    font-size: 0.8rem;
    font-weight: 600;
  }
  .acca-flag { width: 1.4em; height: 1.05em; }
  .acca-leg-icon { font-size: 0.7rem; opacity: 0.8; }
  .acca-footer {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.6rem;
    font-size: 0.8rem;
  }
  .acca-stake  { color: var(--wc-muted); }
  .acca-return { font-weight: 700; flex: 1; }
  .acca-status { font-weight: 700; }
  .acca-status.bet-alive   { color: var(--leg-alive-fg); }
  .acca-status.bet-lost    { color: var(--leg-lost-fg); }
  .acca-status.bet-won     { color: #b45309; }
</style>
