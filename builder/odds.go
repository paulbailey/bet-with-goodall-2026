package main

import "context"

// This file defines the provider-agnostic odds layer that feeds the bet
// likelihood feature. Concrete implementations (Betfair Exchange first) live in
// their own files; the simulator and the per-bet probability combiner depend
// only on these neutral types so the source can be swapped or mocked in tests.
//
// NOTE: the exact method set is provisional until the Betfair market probe
// confirms which markets exist for the World Cup (Match Odds, Correct Score,
// Winner, To Reach Final, Top Goalscorer) and their data shape.

// MatchOdds holds de-vigged 1X2 (home/draw/away) probabilities for one fixture.
// Probabilities sum to ~1 (see devig).
type MatchOdds struct {
	Home  string
	Away  string
	PHome float64
	PDraw float64
	PAway float64
}

// OutrightProb is one selection's de-vigged probability within an outright
// market (e.g. tournament winner, group winner, top goalscorer). The selection
// is a team or player name.
type OutrightProb struct {
	Selection string
	Prob      float64
}

// ScoreProb is one exact-scoreline probability for a fixture's correct-score
// market, from the perspective of the fixture's Home/Away teams.
type ScoreProb struct {
	HomeGoals int
	AwayGoals int
	Prob      float64
}

// FixtureScores holds a single fixture's correct-score market: the actual
// home/away team names (so callers can align a scoreline to either team) plus
// the per-scoreline probabilities, where each ScoreProb's HomeGoals/AwayGoals
// are relative to Home/Away.
type FixtureScores struct {
	Home   string
	Away   string
	Scores []ScoreProb
}

// OutrightMarket names a market the provider can be asked for.
type OutrightMarket string

const (
	MarketWinner        OutrightMarket = "winner"         // tournament winner
	MarketToReachFinal  OutrightMarket = "to_reach_final" // per-team reach-the-final
	MarketTopGoalscorer OutrightMarket = "top_goalscorer"
)

// OddsProvider supplies market-implied probabilities. All probabilities are
// expected to be de-vigged (overround removed) by the implementation.
type OddsProvider interface {
	// MatchWinOdds returns 1X2 probabilities for fixtures the provider knows.
	MatchWinOdds(ctx context.Context) ([]MatchOdds, error)
	// Outright returns selections for a tournament-wide market.
	Outright(ctx context.Context, market OutrightMarket) ([]OutrightProb, error)
	// GroupWinnerOdds returns, per group letter, the win-the-group selections,
	// if the provider exposes group-winner markets. Empty if unsupported.
	GroupWinnerOdds(ctx context.Context) (map[string][]OutrightProb, error)
	// CorrectScore returns each fixture's exact-scoreline market keyed by
	// canonical fixture key (see fixtureKey), if the provider exposes
	// correct-score markets. Empty if unsupported.
	CorrectScore(ctx context.Context) (map[string]FixtureScores, error)
}

// devig removes a market's overround from a slice of decimal odds, returning
// fair probabilities that sum to 1. Uses the standard normalisation (basic)
// method: implied_i = 1/odds_i, fair_i = implied_i / Σ implied. Returns nil for
// empty input or non-positive odds.
func devig(decimalOdds []float64) []float64 {
	if len(decimalOdds) == 0 {
		return nil
	}
	implied := make([]float64, len(decimalOdds))
	var total float64
	for i, o := range decimalOdds {
		if o <= 0 {
			return nil
		}
		implied[i] = 1 / o
		total += implied[i]
	}
	if total == 0 {
		return nil
	}
	for i := range implied {
		implied[i] /= total
	}
	return implied
}

// accaProbability returns the probability that every leg wins, assuming the
// legs are independent. This is exact for legs on unrelated fixtures and an
// approximation where outcomes are correlated; correlated accumulators (e.g.
// group winners across the bracket) should be priced via the simulator instead.
func accaProbability(legProbs []float64) float64 {
	p := 1.0
	for _, lp := range legProbs {
		p *= lp
	}
	return p
}
