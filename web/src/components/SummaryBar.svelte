<script lang="ts">
  import type { TournamentState } from '../types'
  import { money } from '../format'

  interface Props {
    data: TournamentState
  }

  let { data }: Props = $props()

  const allBets = $derived([
    ...data.bets,
    ...data.top_scorer_bets,
    ...data.tournament_winner_bets,
    ...data.match_result_bets,
    ...data.match_acca_bets,
    ...data.finalist_bets,
  ])

  const totalOutlay = $derived(
    allBets.reduce((sum, b) => sum + (b.stake ?? 0), 0)
  )

  const totalWinnings = $derived(
    allBets
      .filter((b) => b.status === 'won')
      .reduce((sum, b) => sum + (b.potential_return ?? 0), 0)
  )

  const totalProfit = $derived(totalWinnings - totalOutlay)

  // The builder computes the best-case payout across all bets, respecting that
  // some bets contradict each other (only one champion, one set of finalists,
  // etc.) so they can't all win at once. Older builder output may omit it.
  const maxPayout = $derived(data.max_payout?.max_payout ?? null)
  const maxProfit = $derived(data.max_payout?.max_profit ?? null)

  // Probability-weighted figures from the odds-driven simulator. Absent when no
  // odds source is configured (older builder output / Betfair unavailable).
  const expectedPayout = $derived(data.expected?.expected_payout ?? null)
  const expectedProfit = $derived(data.expected?.expected_profit ?? null)
</script>

<section class="summary-bar">
  <div class="summary-figure">
    <span class="summary-label">Total Outlay</span>
    <span class="summary-value">{money(totalOutlay)}</span>
  </div>
  <div class="summary-figure">
    <span class="summary-label">Total Winnings</span>
    <span class="summary-value summary-winnings">{money(totalWinnings)}</span>
  </div>
  <div class="summary-figure">
    <span class="summary-label">Total Profit</span>
    <span class="summary-value" class:summary-positive={totalProfit >= 0} class:summary-negative={totalProfit < 0}>
      {money(totalProfit)}
    </span>
  </div>
  {#if maxPayout != null}
    <div class="summary-figure">
      <span class="summary-label">Max Possible Winnings</span>
      <span class="summary-value summary-max">{money(maxPayout)}</span>
    </div>
  {/if}
  {#if maxProfit != null}
    <div class="summary-figure">
      <span class="summary-label">Max Possible Profit</span>
      <span class="summary-value" class:summary-positive={maxProfit >= 0} class:summary-negative={maxProfit < 0}>
        {money(maxProfit)}
      </span>
    </div>
  {/if}
  {#if expectedPayout != null}
    <div class="summary-figure">
      <span class="summary-label" title="Probability-weighted across all priced bets">Expected Payout</span>
      <span class="summary-value summary-expected">{money(expectedPayout)}</span>
    </div>
  {/if}
  {#if expectedProfit != null}
    <div class="summary-figure">
      <span class="summary-label" title="Expected payout minus total outlay">Expected Profit</span>
      <span class="summary-value" class:summary-positive={expectedProfit >= 0} class:summary-negative={expectedProfit < 0}>
        {money(expectedProfit)}
      </span>
    </div>
  {/if}
</section>

<style>
  .summary-bar {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 1rem;
  }

  .summary-figure {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 0.75rem;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
    padding: 1rem 1.25rem;
  }

  .summary-label {
    font-size: 0.7rem;
    font-weight: 700;
    color: var(--wc-muted);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }

  .summary-value {
    font-size: 1.75rem;
    font-weight: 800;
    color: var(--wc-navy);
    letter-spacing: -0.02em;
    line-height: 1.1;
    white-space: nowrap;
  }

  .summary-winnings { color: #b45309; }
  .summary-max { color: var(--wc-navy); }
  .summary-expected { color: #6d28d9; }
  .summary-positive { color: var(--leg-alive-fg); }
  .summary-negative { color: var(--leg-lost-fg); }

  @media (max-width: 640px) {
    .summary-bar { grid-template-columns: repeat(2, 1fr); }
    .summary-figure { padding: 0.75rem 0.85rem; }
    .summary-value { font-size: 1.35rem; }
  }
</style>
