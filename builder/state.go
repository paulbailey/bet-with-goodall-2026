package main

import (
	"strings"
	"time"
)

// StateJSON is the schema written to S3 and consumed by the React frontend.
// Field names must stay in sync with web/src/types.ts.
type StateJSON struct {
	UpdatedAt            string                    `json:"updated_at"`
	TournamentPhase      string                    `json:"tournament_phase"`
	Groups               map[string]GroupJSON      `json:"groups"`
	Bets                 []BetJSON                 `json:"bets"`
	TopScorerBets        []TopScorerBetJSON        `json:"top_scorer_bets"`
	TournamentWinnerBets []TournamentWinnerBetJSON `json:"tournament_winner_bets"`
	MatchResultBets      []MatchResultBetJSON      `json:"match_result_bets"`
	MatchAccaBets        []MatchAccaBetJSON        `json:"match_acca_bets"`
	FinalistBets         []FinalistBetJSON         `json:"finalist_bets"`
	TopScorers           []TopScorerJSON           `json:"top_scorers"`
	MaxPayout            MaxPayoutJSON             `json:"max_payout"`
}

type GroupJSON struct {
	Standings []StandingJSON `json:"standings"`
	Complete  bool           `json:"complete"`
	Winner    *string        `json:"winner"`
}

type StandingJSON struct {
	Team   string `json:"team"`
	Played int    `json:"played"`
	Won    int    `json:"won"`
	Drawn  int    `json:"drawn"`
	Lost   int    `json:"lost"`
	GF     int    `json:"gf"`
	GA     int    `json:"ga"`
	GD     int    `json:"gd"`
	Points int    `json:"points"`
}

type BetJSON struct {
	ID              string    `json:"id"`
	Stake           *float64  `json:"stake,omitempty"`
	PotentialReturn *float64  `json:"potential_return,omitempty"`
	Status          string    `json:"status"`
	Legs            []LegJSON `json:"legs"`
}

type LegJSON struct {
	Group  string `json:"group"`
	Team   string `json:"team"`
	Status string `json:"status"`
}

type TopScorerBetJSON struct {
	ID              string   `json:"id"`
	Player          string   `json:"player"`
	Team            string   `json:"team"`
	Stake           *float64 `json:"stake,omitempty"`
	PotentialReturn *float64 `json:"potential_return,omitempty"`
	Status          string   `json:"status"`
}

type TournamentWinnerBetJSON struct {
	ID              string   `json:"id"`
	Team            string   `json:"team"`
	Stake           *float64 `json:"stake,omitempty"`
	PotentialReturn *float64 `json:"potential_return,omitempty"`
	Status          string   `json:"status"`
}

type MatchResultBetJSON struct {
	ID              string   `json:"id"`
	TeamA           string   `json:"team_a"`
	TeamB           string   `json:"team_b"`
	ScoreA          int      `json:"score_a"`
	ScoreB          int      `json:"score_b"`
	ActualA         *int     `json:"actual_a"` // live/final goals for team_a, null until known
	ActualB         *int     `json:"actual_b"`
	Stake           *float64 `json:"stake,omitempty"`
	PotentialReturn *float64 `json:"potential_return,omitempty"`
	Status          string   `json:"status"`
}

type MatchAccaBetJSON struct {
	ID              string                `json:"id"`
	Stake           *float64              `json:"stake,omitempty"`
	PotentialReturn *float64              `json:"potential_return,omitempty"`
	Status          string                `json:"status"`
	Legs            []MatchOutcomeLegJSON `json:"legs"`
}

type MatchOutcomeLegJSON struct {
	Team     string `json:"team"`
	Opponent string `json:"opponent"`
	Outcome  string `json:"outcome"`
	Status   string `json:"status"`
}

type FinalistBetJSON struct {
	ID              string   `json:"id"`
	TeamA           string   `json:"team_a"`
	TeamB           string   `json:"team_b"`
	Stake           *float64 `json:"stake,omitempty"`
	PotentialReturn *float64 `json:"potential_return,omitempty"`
	Status          string   `json:"status"`
}

type TopScorerJSON struct {
	Player         string `json:"player"`
	Team           string `json:"team"`
	Goals          int    `json:"goals"`
	Games          int    `json:"games"`
	TeamEliminated bool   `json:"team_eliminated"`
}

func buildState(cfg *Config, groups []GroupStanding, scorers []TopScorerEntry, matches []Match) StateJSON {
	s := StateJSON{
		UpdatedAt:            time.Now().UTC().Format(time.RFC3339),
		TournamentPhase:      tournamentPhase(matches),
		Groups:               buildGroups(groups),
		Bets:                 buildBets(cfg.Bets, groups),
		TopScorerBets:        buildTopScorerBets(cfg.TopScorerBets, scorers, groups),
		TournamentWinnerBets: buildTournamentWinnerBets(cfg.TournamentWinnerBets, groups, matches),
		MatchResultBets:      buildMatchResultBets(cfg.MatchResultBets, matches),
		MatchAccaBets:        buildMatchAccaBets(cfg.MatchAccaBets, matches),
		FinalistBets:         buildFinalistBets(cfg.FinalistBets, groups, matches),
		TopScorers:           buildTopScorers(scorers, groups),
	}
	s.MaxPayout = computeMaxPayout(s)
	return s
}

func buildGroups(groups []GroupStanding) map[string]GroupJSON {
	m := make(map[string]GroupJSON, len(groups))
	for _, g := range groups {
		complete := true
		for _, t := range g.Teams {
			if t.Played < groupGames {
				complete = false
				break
			}
		}

		var winner *string
		if complete && len(g.Teams) > 0 {
			w := g.Teams[0].Name
			winner = &w
		}

		standings := make([]StandingJSON, len(g.Teams))
		for i, t := range g.Teams {
			standings[i] = StandingJSON{
				Team: t.Name, Played: t.Played, Won: t.Won, Drawn: t.Drawn,
				Lost: t.Lost, GF: t.GF, GA: t.GA, GD: t.GD, Points: t.Points,
			}
		}
		m[g.Group] = GroupJSON{Standings: standings, Complete: complete, Winner: winner}
	}
	return m
}

func buildBets(bets []BetConfig, groups []GroupStanding) []BetJSON {
	out := make([]BetJSON, len(bets))
	for i, bc := range bets {
		b := BetJSON{
			ID:   bc.ID,
			Legs: make([]LegJSON, len(bc.Legs)),
		}
		if bc.Stake > 0 {
			s := bc.Stake
			b.Stake = &s
		}
		if bc.PotentialReturn > 0 {
			r := bc.PotentialReturn
			b.PotentialReturn = &r
		}

		statuses := make([]string, len(bc.Legs))
		for j, leg := range bc.Legs {
			status := evaluateLeg(leg.Group, leg.Team, groups)
			b.Legs[j] = LegJSON{Group: leg.Group, Team: leg.Team, Status: status}
			statuses[j] = status
		}
		b.Status = combineLegStatuses(statuses)

		out[i] = b
	}
	return out
}

func buildTopScorerBets(bets []TopScorerBetConfig, scorers []TopScorerEntry, groups []GroupStanding) []TopScorerBetJSON {
	out := make([]TopScorerBetJSON, len(bets))
	for i, bc := range bets {
		b := TopScorerBetJSON{
			ID:     bc.ID,
			Player: bc.Player,
			Team:   bc.Team,
			Status: evaluateTopScorerBet(bc.Player, scorers, groups),
		}
		if bc.Stake > 0 {
			s := bc.Stake
			b.Stake = &s
		}
		if bc.PotentialReturn > 0 {
			r := bc.PotentialReturn
			b.PotentialReturn = &r
		}
		out[i] = b
	}
	return out
}

func buildTournamentWinnerBets(bets []TournamentWinnerBetConfig, groups []GroupStanding, matches []Match) []TournamentWinnerBetJSON {
	out := make([]TournamentWinnerBetJSON, len(bets))
	for i, bc := range bets {
		b := TournamentWinnerBetJSON{
			ID:     bc.ID,
			Team:   bc.Team,
			Status: evaluateTournamentWinnerBet(bc.Team, groups, matches),
		}
		if bc.Stake > 0 {
			s := bc.Stake
			b.Stake = &s
		}
		if bc.PotentialReturn > 0 {
			r := bc.PotentialReturn
			b.PotentialReturn = &r
		}
		out[i] = b
	}
	return out
}

func buildMatchResultBets(bets []MatchResultBetConfig, matches []Match) []MatchResultBetJSON {
	out := make([]MatchResultBetJSON, len(bets))
	for i, bc := range bets {
		b := MatchResultBetJSON{
			ID:     bc.ID,
			TeamA:  bc.TeamA,
			TeamB:  bc.TeamB,
			ScoreA: bc.ScoreA,
			ScoreB: bc.ScoreB,
			Status: evaluateMatchResultBet(bc.TeamA, bc.TeamB, bc.ScoreA, bc.ScoreB, matches),
		}
		if m := findMatch(bc.TeamA, bc.TeamB, matches); m != nil {
			if forA, againstA, ok := teamScores(m, bc.TeamA); ok {
				a, agA := forA, againstA
				b.ActualA = &a
				b.ActualB = &agA
			}
		}
		if bc.Stake > 0 {
			s := bc.Stake
			b.Stake = &s
		}
		if bc.PotentialReturn > 0 {
			r := bc.PotentialReturn
			b.PotentialReturn = &r
		}
		out[i] = b
	}
	return out
}

func buildMatchAccaBets(bets []MatchAccaBetConfig, matches []Match) []MatchAccaBetJSON {
	out := make([]MatchAccaBetJSON, len(bets))
	for i, bc := range bets {
		b := MatchAccaBetJSON{
			ID:   bc.ID,
			Legs: make([]MatchOutcomeLegJSON, len(bc.Legs)),
		}
		statuses := make([]string, len(bc.Legs))
		for j, leg := range bc.Legs {
			status := evaluateMatchOutcomeLeg(leg.Team, leg.Opponent, leg.Outcome, matches)
			b.Legs[j] = MatchOutcomeLegJSON{
				Team: leg.Team, Opponent: leg.Opponent, Outcome: leg.Outcome, Status: status,
			}
			statuses[j] = status
		}
		b.Status = combineLegStatuses(statuses)
		if bc.Stake > 0 {
			s := bc.Stake
			b.Stake = &s
		}
		if bc.PotentialReturn > 0 {
			r := bc.PotentialReturn
			b.PotentialReturn = &r
		}
		out[i] = b
	}
	return out
}

func buildFinalistBets(bets []FinalistBetConfig, groups []GroupStanding, matches []Match) []FinalistBetJSON {
	out := make([]FinalistBetJSON, len(bets))
	for i, bc := range bets {
		b := FinalistBetJSON{
			ID:     bc.ID,
			TeamA:  bc.TeamA,
			TeamB:  bc.TeamB,
			Status: evaluateFinalistBet(bc.TeamA, bc.TeamB, groups, matches),
		}
		if bc.Stake > 0 {
			s := bc.Stake
			b.Stake = &s
		}
		if bc.PotentialReturn > 0 {
			r := bc.PotentialReturn
			b.PotentialReturn = &r
		}
		out[i] = b
	}
	return out
}

func buildTopScorers(scorers []TopScorerEntry, groups []GroupStanding) []TopScorerJSON {
	out := make([]TopScorerJSON, len(scorers))
	for i, s := range scorers {
		out[i] = TopScorerJSON{
			Player:         s.Player,
			Team:           s.Team,
			Goals:          s.Goals,
			Games:          s.Games,
			TeamEliminated: isTeamEliminated(s.Team, groups),
		}
	}
	return out
}

// notStarted returns true for statuses that mean a match hasn't kicked off yet.
// football-data.org uses "TIMED" (kickoff confirmed) and "SCHEDULED" (date TBC);
// both are "not started" for our purposes.
func notStarted(status string) bool {
	return status == "SCHEDULED" || status == "TIMED"
}

func tournamentPhase(matches []Match) string {
	if len(matches) == 0 {
		return "pre_tournament"
	}

	anyStarted, allFinished, hasKnockout := false, true, false
	for _, m := range matches {
		if !notStarted(m.Status) {
			anyStarted = true
		}
		if m.Status != "FINISHED" {
			allFinished = false
		}
		if !strings.Contains(m.Stage, "GROUP") && !notStarted(m.Status) {
			hasKnockout = true
		}
	}

	switch {
	case !anyStarted:
		return "pre_tournament"
	case allFinished:
		return "complete"
	case hasKnockout:
		return "knockout"
	default:
		return "group_stage"
	}
}
