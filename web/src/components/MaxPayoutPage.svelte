<script lang="ts">
  import type {
    Bet,
    BetStatus,
    FinalistBet,
    MatchAccaBet,
    MatchOutcome,
    MatchResultBet,
    TopScorerBet,
    TournamentState,
    TournamentWinnerBet,
  } from '../types'
  import { getCountry } from '../countries'
  import { money, pct } from '../format'
  import Flag from './Flag.svelte'
  import MaxPayoutBreakdown from './MaxPayoutBreakdown.svelte'

  type AppRoute = '/' | '/max-payout'
  type LiveStatus = Exclude<BetStatus, 'lost'>

  interface Props {
    data: TournamentState
    onNavigate?: (event: MouseEvent, to: AppRoute) => void
  }

  interface CoreBet {
    id: string
    status: string
    stake?: number
    potential_return?: number
    probability?: number
  }

  interface DetailItem {
    label: string
    teams?: string[]
    status?: LiveStatus
  }

  interface SelectedBet {
    id: string
    title: string
    status: LiveStatus
    stake?: number
    returnValue: number
    probability?: number
    details: DetailItem[]
  }

  interface SelectedGroup {
    title: string
    bets: SelectedBet[]
  }

  let { data, onNavigate }: Props = $props()

  const maxPayout = $derived(data.max_payout ?? null)
  const droppedBetIDs = $derived.by(() => {
    const ids = new Set<string>()
    for (const group of data.max_payout?.conflicts ?? []) {
      for (const bet of group.bets) {
        if (!bet.chosen) ids.add(bet.id)
      }
    }
    return ids
  })

  const favByGroup = $derived.by(() => {
    const counts: Record<string, Record<string, number>> = {}
    for (const bet of data.bets) {
      for (const leg of bet.legs) {
        counts[leg.group] ??= {}
        counts[leg.group][leg.team] = (counts[leg.group][leg.team] ?? 0) + 1
      }
    }

    const fav: Record<string, string> = {}
    for (const [group, teams] of Object.entries(counts)) {
      fav[group] = Object.entries(teams).sort((a, b) => b[1] - a[1])[0][0]
    }
    return fav
  })
  const selectedGroups = $derived.by(() => {
    if (!maxPayout) return []

    const groups: SelectedGroup[] = []
    addGroup(groups, 'Group Winner Accumulators', data.bets, buildGroupWinnerBet)
    addGroup(groups, 'Match Accumulators', data.match_acca_bets, buildMatchAccaBet)
    addGroup(groups, 'Match Result Bets', data.match_result_bets, buildMatchResultBet)
    addGroup(groups, 'Finalist Bets', data.finalist_bets, buildFinalistBet)
    addGroup(groups, 'Tournament Winner Bets', data.tournament_winner_bets, buildTournamentWinnerBet)
    addGroup(groups, 'Top Scorer Bets', data.top_scorer_bets, buildTopScorerBet)
    return groups
  })
  const labelByBetID = $derived.by(() => {
    const labels: Record<string, string> = {}
    for (const bet of data.bets) labels[bet.id] = groupWinnerBetTitle(bet)
    for (const bet of data.match_acca_bets) labels[bet.id] = matchAccumulatorBetTitle(bet)
    for (const bet of data.match_result_bets) labels[bet.id] = matchResultBetTitle(bet)
    for (const bet of data.finalist_bets) labels[bet.id] = finalistBetTitle(bet)
    for (const bet of data.tournament_winner_bets) labels[bet.id] = tournamentWinnerBetTitle(bet)
    for (const bet of data.top_scorer_bets) labels[bet.id] = topScorerBetTitle(bet)
    return labels
  })

  const selectedCount = $derived(
    selectedGroups.reduce((sum, group) => sum + group.bets.length, 0)
  )
  const selectedReturn = $derived(
    selectedGroups.reduce(
      (sum, group) => sum + group.bets.reduce((groupSum, bet) => groupSum + bet.returnValue, 0),
      0
    )
  )

  const STATUS_LABEL: Record<LiveStatus, string> = {
    alive: 'Still needed',
    won: 'Already won',
  }

  const OUTCOME_LABEL: Record<MatchOutcome, string> = {
    win: 'to beat',
    draw: 'to draw',
    lose: 'to lose to',
  }

  function addGroup<T extends CoreBet>(
    groups: SelectedGroup[],
    title: string,
    bets: T[],
    build: (bet: T) => SelectedBet
  ) {
    const counted = bets
      .filter((bet) => isCountedBet(bet))
      .map(build)
    if (counted.length > 0) {
      groups.push({ title, bets: counted })
    }
  }

  function isCountedBet(bet: CoreBet): boolean {
    return bet.status !== 'lost' && !droppedBetIDs.has(bet.id)
  }

  function liveStatus(status: string): LiveStatus {
    return status === 'won' ? 'won' : 'alive'
  }

  function buildBaseBet(bet: CoreBet, title: string, details: DetailItem[]): SelectedBet {
    return {
      id: bet.id,
      title,
      status: liveStatus(bet.status),
      stake: bet.stake,
      returnValue: bet.potential_return ?? 0,
      probability: bet.probability,
      details,
    }
  }

  function buildGroupWinnerBet(bet: Bet): SelectedBet {
    return buildBaseBet(
      bet,
      groupWinnerBetTitle(bet),
      bet.legs.map((leg) => ({
        label: `Group ${leg.group}: ${leg.team}`,
        teams: [leg.team],
        status: liveStatus(leg.status),
      }))
    )
  }

  function buildMatchAccaBet(bet: MatchAccaBet): SelectedBet {
    return buildBaseBet(
      bet,
      matchAccumulatorBetTitle(bet),
      bet.legs.map((leg) => ({
        label: `${leg.team} ${OUTCOME_LABEL[leg.outcome]} ${leg.opponent}`,
        teams: [leg.team, leg.opponent],
        status: liveStatus(leg.status),
      }))
    )
  }

  function buildMatchResultBet(bet: MatchResultBet): SelectedBet {
    return buildBaseBet(
      bet,
      matchResultBetTitle(bet),
      [{
        label: `${bet.team_a} ${bet.score_a}-${bet.score_b} ${bet.team_b}`,
        teams: [bet.team_a, bet.team_b],
        status: liveStatus(bet.status),
      }]
    )
  }

  function buildFinalistBet(bet: FinalistBet): SelectedBet {
    return buildBaseBet(
      bet,
      finalistBetTitle(bet),
      [{
        label: `${bet.team_a} and ${bet.team_b} reach the final`,
        teams: [bet.team_a, bet.team_b],
        status: liveStatus(bet.status),
      }]
    )
  }

  function buildTournamentWinnerBet(bet: TournamentWinnerBet): SelectedBet {
    return buildBaseBet(
      bet,
      tournamentWinnerBetTitle(bet),
      [{
        label: `${bet.team} wins the tournament`,
        teams: [bet.team],
        status: liveStatus(bet.status),
      }]
    )
  }

  function buildTopScorerBet(bet: TopScorerBet): SelectedBet {
    return buildBaseBet(
      bet,
      topScorerBetTitle(bet),
      [{
        label: `${bet.player} finishes top scorer`,
        teams: [bet.team],
        status: liveStatus(bet.status),
      }]
    )
  }

  function isGroupPick(leg: Bet['legs'][number]): boolean {
    return favByGroup[leg.group] != null && leg.team !== favByGroup[leg.group]
  }

  function groupWinnerBetTitle(bet: Bet): string {
    const picks = bet.legs.filter(isGroupPick)
    if (picks.length === 0) return 'Favourites'
    return picks.map((leg) => `${leg.group}: ${leg.team}`).join(', ')
  }

  function matchAccumulatorBetTitle(bet: MatchAccaBet): string {
    return `Match accumulator (${bet.legs.length} legs)`
  }

  function matchResultBetTitle(bet: MatchResultBet): string {
    return `${bet.team_a} ${bet.score_a}-${bet.score_b} ${bet.team_b}`
  }

  function finalistBetTitle(bet: FinalistBet): string {
    return `${bet.team_a} and ${bet.team_b} to reach the final`
  }

  function tournamentWinnerBetTitle(bet: TournamentWinnerBet): string {
    return `${bet.team} to win the tournament`
  }

  function topScorerBetTitle(bet: TopScorerBet): string {
    return `${bet.player} top scorer`
  }
</script>

<main class="app-content max-payout-page">
  <a class="back-link" href="/" onclick={(event) => onNavigate?.(event, '/')}>Back to dashboard</a>

  {#if !maxPayout}
    <section class="empty-panel">
      <h2>Max payout data unavailable</h2>
      <p>The current state file does not include max payout data yet.</p>
    </section>
  {:else}
    <section class="max-page-header">
      <div>
        <p class="section-title">Max Payout</p>
        <h2>Winning Bets Required</h2>
      </div>
      <div class="max-metrics" aria-label="Max payout summary">
        <div class="max-metric primary">
          <span class="metric-label">Max Possible Winnings</span>
          <span class="metric-value">{money(maxPayout.max_payout)}</span>
        </div>
        <div class="max-metric">
          <span class="metric-label">Max Possible Profit</span>
          <span class="metric-value" class:positive={maxPayout.max_profit >= 0} class:negative={maxPayout.max_profit < 0}>
            {money(maxPayout.max_profit)}
          </span>
        </div>
        <div class="max-metric">
          <span class="metric-label">Already Won</span>
          <span class="metric-value">{money(maxPayout.realised_winnings)}</span>
        </div>
        <div class="max-metric">
          <span class="metric-label">Total Outlay</span>
          <span class="metric-value">{money(maxPayout.total_outlay)}</span>
        </div>
      </div>
    </section>

    <section class="required-section">
      <div class="required-heading">
        <div>
          <h2 class="section-title">Counted Bets</h2>
          <p>{selectedCount} bets currently count toward {money(maxPayout.max_payout)}.</p>
        </div>
        <div class="selected-return" title="Sum of the counted bet returns">
          {money(selectedReturn)}
        </div>
      </div>

      {#if selectedGroups.length === 0}
        <div class="empty-panel">No live or won bets currently count toward the maximum.</div>
      {:else}
        <div class="selected-groups">
          {#each selectedGroups as group (group.title)}
            <section class="selected-group">
              <div class="selected-group-heading">
                <h3>{group.title}</h3>
                <span>{group.bets.length}</span>
              </div>
              <div class="selected-bets">
                {#each group.bets as bet (bet.id)}
                  <article class="selected-bet-card" class:bet-won={bet.status === 'won'}>
                    <div class="selected-bet-head">
                      <div class="selected-bet-title-wrap">
                        <h4>{bet.title}</h4>
                      </div>
                      <strong>{money(bet.returnValue)}</strong>
                    </div>

                    <div class="selected-bet-meta">
                      {#if bet.stake != null}
                        <span>Stake {money(bet.stake)}</span>
                      {/if}
                      <span>Chance {pct(bet.probability)}</span>
                      <span class="status-pill" class:status-won={bet.status === 'won'}>
                        {STATUS_LABEL[bet.status]}
                      </span>
                    </div>

                    <div class="detail-list">
                      {#each bet.details as detail}
                        <span class="detail-chip" class:detail-won={detail.status === 'won'}>
                          {#if detail.teams}
                            <span class="detail-flags">
                              {#each detail.teams as team (team)}
                                {@const { fi } = getCountry(team)}
                                <Flag {fi} class="detail-flag" />
                              {/each}
                            </span>
                          {/if}
                          <span>{detail.label}</span>
                        </span>
                      {/each}
                    </div>
                  </article>
                {/each}
              </div>
            </section>
          {/each}
        </div>
      {/if}
    </section>

    <MaxPayoutBreakdown maxPayout={maxPayout} {labelByBetID} />
  {/if}
</main>

<style>
  .max-payout-page {
    gap: 1.5rem;
  }

  .back-link {
    align-self: flex-start;
    color: var(--wc-navy);
    font-size: 0.9rem;
    font-weight: 700;
    text-decoration: none;
  }

  .back-link:hover,
  .back-link:focus-visible {
    text-decoration: underline;
  }

  .max-page-header,
  .required-section,
  .empty-panel {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 8px;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.08);
  }

  .max-page-header {
    display: grid;
    grid-template-columns: minmax(0, 0.6fr) minmax(280px, 1.4fr);
    gap: 1.25rem;
    align-items: start;
    padding: 1.25rem;
  }

  .max-page-header h2 {
    color: var(--wc-text);
    font-size: 1.6rem;
    font-weight: 800;
    line-height: 1.1;
    margin-top: 0.4rem;
  }

  .max-metrics {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 0.75rem;
  }

  .max-metric {
    border-left: 3px solid var(--wc-border);
    min-width: 0;
    padding-left: 0.75rem;
  }

  .max-metric.primary {
    border-left-color: var(--wc-navy);
  }

  .metric-label {
    color: var(--wc-muted);
    display: block;
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    line-height: 1.2;
    text-transform: uppercase;
  }

  .metric-value {
    color: var(--wc-navy);
    display: block;
    font-size: 1.35rem;
    font-weight: 800;
    line-height: 1.15;
    margin-top: 0.35rem;
    white-space: nowrap;
  }

  .positive { color: var(--leg-alive-fg); }
  .negative { color: var(--leg-lost-fg); }

  .required-section,
  .empty-panel {
    padding: 1.25rem;
  }

  .empty-panel h2 {
    color: var(--wc-navy);
    font-size: 1.2rem;
    margin-bottom: 0.35rem;
  }

  .empty-panel p,
  .required-heading p {
    color: var(--wc-muted);
    font-size: 0.9rem;
    margin-top: 0.35rem;
  }

  .required-heading {
    align-items: start;
    display: flex;
    gap: 1rem;
    justify-content: space-between;
  }

  .selected-return {
    color: var(--wc-navy);
    font-size: 1.1rem;
    font-weight: 800;
    white-space: nowrap;
  }

  .selected-groups {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    margin-top: 1rem;
  }

  .selected-group-heading {
    align-items: center;
    display: flex;
    gap: 0.75rem;
    justify-content: space-between;
    margin-bottom: 0.75rem;
  }

  .selected-group-heading h3 {
    color: var(--wc-navy);
    font-size: 0.95rem;
    font-weight: 800;
  }

  .selected-group-heading span {
    color: var(--wc-muted);
    font-size: 0.78rem;
    font-weight: 700;
  }

  .selected-bets {
    display: grid;
    gap: 0.75rem;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  }

  .selected-bet-card {
    background: var(--wc-white);
    border: 1px solid var(--wc-border);
    border-radius: 8px;
    min-width: 0;
    padding: 0.85rem;
  }

  .selected-bet-card.bet-won {
    background: #fffbeb;
  }

  .selected-bet-head {
    align-items: start;
    display: flex;
    gap: 0.75rem;
    justify-content: space-between;
  }

  .selected-bet-head strong {
    color: var(--wc-navy);
    font-size: 0.95rem;
    white-space: nowrap;
  }

  .selected-bet-title-wrap {
    min-width: 0;
  }

  .selected-bet-title-wrap h4 {
    color: var(--wc-text);
    font-size: 0.92rem;
    font-weight: 800;
    line-height: 1.25;
  }

  .selected-bet-meta {
    align-items: center;
    color: var(--wc-muted);
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    font-size: 0.78rem;
    margin-top: 0.65rem;
  }

  .status-pill {
    background: var(--leg-alive-bg);
    border-radius: 999px;
    color: var(--leg-alive-fg);
    font-size: 0.68rem;
    font-weight: 800;
    padding: 0.16rem 0.45rem;
    text-transform: uppercase;
  }

  .status-pill.status-won {
    background: var(--leg-won-bg);
    color: var(--leg-won-fg);
  }

  .detail-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-top: 0.75rem;
  }

  .detail-chip {
    align-items: center;
    background: var(--leg-alive-bg);
    border-radius: 8px;
    color: var(--leg-alive-fg);
    display: inline-flex;
    gap: 0.35rem;
    max-width: 100%;
    overflow-wrap: anywhere;
    padding: 0.28rem 0.45rem;
    font-size: 0.76rem;
    font-weight: 700;
    line-height: 1.25;
  }

  .detail-chip.detail-won {
    background: var(--leg-won-bg);
    color: var(--leg-won-fg);
  }

  .detail-flags {
    display: inline-flex;
    flex: 0 0 auto;
    gap: 0.16rem;
  }

  :global(.detail-flag) {
    height: 0.9rem;
    width: 1.2rem;
  }

  @media (max-width: 900px) {
    .max-page-header {
      grid-template-columns: 1fr;
    }

    .max-metrics {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 640px) {
    .max-page-header,
    .required-section,
    .empty-panel {
      padding: 1rem;
    }

    .max-metrics,
    .selected-bets {
      grid-template-columns: 1fr;
    }

    .required-heading,
    .selected-bet-head {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
