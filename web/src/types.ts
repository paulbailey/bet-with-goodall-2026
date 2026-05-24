export type LegStatus = 'pending' | 'alive' | 'won' | 'lost'
export type TopScorerBetStatus = 'alive' | 'won' | 'lost'
export type BetStatus = 'pending' | 'alive' | 'won' | 'lost'
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
  legs: BetLeg[]
}

export interface TopScorerBet {
  id: string
  player: string
  team: string
  stake?: number
  potential_return?: number
  status: TopScorerBetStatus
}

export interface TopScorer {
  player: string
  team: string
  goals: number
  games: number
  team_eliminated: boolean
}

export interface TournamentState {
  updated_at: string
  tournament_phase: TournamentPhase
  groups: Record<string, GroupState>
  bets: Bet[]
  top_scorer_bets: TopScorerBet[]
  top_scorers: TopScorer[]
}
