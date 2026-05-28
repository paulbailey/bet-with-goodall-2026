export type LegStatus = 'alive' | 'won' | 'lost'
export type TopScorerBetStatus = 'alive' | 'won' | 'lost'
export type TournamentWinnerBetStatus = 'alive' | 'won' | 'lost'
export type BetStatus = 'alive' | 'won' | 'lost'
export type FinalistBetStatus = 'alive' | 'won' | 'lost'
export type MatchOutcome = 'win' | 'draw' | 'lose'
export type TournamentPhase = 'pre_tournament' | 'group_stage' | 'knockout' | 'complete'

export interface TeamStanding {
  team: string
  played: number
  won: number
  drawn: number
  lost: number
  gf: number
  ga: number
  gd: number
  points: number
}

export interface GroupState {
  standings: TeamStanding[]
  complete: boolean
  winner: string | null
}

export interface BetLeg {
  group: string
  team: string
  status: LegStatus
}

export interface Bet {
  id: string
  stake?: number
  potential_return?: number
  status: BetStatus
  probability?: number
  expected_return?: number
  legs: BetLeg[]
}

export interface TopScorerBet {
  id: string
  player: string
  team: string
  stake?: number
  potential_return?: number
  status: TopScorerBetStatus
  probability?: number
  expected_return?: number
}

export interface TournamentWinnerBet {
  id: string
  team: string
  stake?: number
  potential_return?: number
  status: TournamentWinnerBetStatus
  probability?: number
  expected_return?: number
}

export interface TopScorer {
  player: string
  team: string
  goals: number
  games: number
  team_eliminated: boolean
}

export interface MatchResultBet {
  id: string
  team_a: string
  team_b: string
  score_a: number
  score_b: number
  actual_a: number | null
  actual_b: number | null
  stake?: number
  potential_return?: number
  status: BetStatus
  probability?: number
  expected_return?: number
}

export interface MatchOutcomeLeg {
  team: string
  opponent: string
  outcome: MatchOutcome
  status: LegStatus
}

export interface MatchAccaBet {
  id: string
  stake?: number
  potential_return?: number
  status: BetStatus
  probability?: number
  expected_return?: number
  legs: MatchOutcomeLeg[]
}

export interface FinalistBet {
  id: string
  team_a: string
  team_b: string
  stake?: number
  potential_return?: number
  status: FinalistBetStatus
  probability?: number
  expected_return?: number
}

export interface ConflictBet {
  id: string
  label: string
  return: number
  status: Exclude<BetStatus, 'lost'>
  chosen: boolean
}

export interface ConflictGroup {
  bets: ConflictBet[]
}

export interface MaxPayout {
  max_payout: number
  realised_winnings: number
  total_outlay: number
  max_profit: number
  conflicts: ConflictGroup[]
}

// Expected is the probability-weighted counterpart to MaxPayout: summing each
// priced bet's chance × return. Omitted when no odds source is configured.
export interface Expected {
  expected_payout: number
  expected_profit: number
}

export interface TournamentState {
  updated_at: string
  tournament_phase: TournamentPhase
  groups: Record<string, GroupState>
  bets: Bet[]
  top_scorer_bets: TopScorerBet[]
  tournament_winner_bets: TournamentWinnerBet[]
  match_result_bets: MatchResultBet[]
  match_acca_bets: MatchAccaBet[]
  finalist_bets: FinalistBet[]
  top_scorers: TopScorer[]
  max_payout?: MaxPayout | null
  expected?: Expected | null
}
