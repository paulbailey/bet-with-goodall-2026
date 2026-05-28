package main

import (
	"math"
	"testing"
)

// approxPtr asserts a *float64 field was set and is within tolerance. (f() — the
// float-pointer helper — lives in payout_test.go.)
func approxPtr(t *testing.T, label string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s: probability not set", label)
	}
	if math.Abs(*got-want) > 1e-6 {
		t.Errorf("%s: got %v want %v", label, *got, want)
	}
}

// testSnapshot wires up a small but representative set of markets.
func testSnapshot() OddsSnapshot {
	snap := OddsSnapshot{
		Match: []MatchOdds{
			{Home: "England", Away: "Croatia", PHome: 0.60, PDraw: 0.25, PAway: 0.15},
		},
		Winner:    []OutrightProb{{Selection: "England", Prob: 0.10}},
		TopScorer: []OutrightProb{{Selection: "Harry Kane", Prob: 0.12}},
		GroupWinners: map[string][]OutrightProb{
			"A": {{Selection: "Mexico", Prob: 0.40}},
			"B": {{Selection: "Switzerland", Prob: 0.50}},
		},
		CorrectScore: map[string]FixtureScores{},
	}
	key, _ := fixtureKey("England", "Croatia")
	snap.CorrectScore[key] = FixtureScores{
		Home: "England", Away: "Croatia",
		Scores: []ScoreProb{{HomeGoals: 3, AwayGoals: 1, Prob: 0.08}},
	}
	snap.index()
	return snap
}

func TestApplyLikelihoods_GroupAcca(t *testing.T) {
	snap := testSnapshot()
	s := StateJSON{
		Bets: []BetJSON{
			{
				ID: "ga-both-alive", Status: "alive", PotentialReturn: f(100),
				Legs: []LegJSON{
					{Group: "A", Team: "Mexico", Status: "alive"},
					{Group: "B", Team: "Switzerland", Status: "alive"},
				},
			},
			{
				ID: "ga-one-won", Status: "alive", PotentialReturn: f(100),
				Legs: []LegJSON{
					{Group: "A", Team: "Mexico", Status: "won"},
					{Group: "B", Team: "Switzerland", Status: "alive"},
				},
			},
		},
	}
	applyLikelihoods(&s, snap, nil, nil)

	approxPtr(t, "both legs alive", s.Bets[0].Probability, 0.40*0.50) // 0.20
	approxPtr(t, "expected return", s.Bets[0].ExpectedReturn, 20.0)
	approxPtr(t, "one leg won", s.Bets[1].Probability, 0.50) // won leg is a factor of 1
}

func TestApplyLikelihoods_GroupAccaFallsBackToSim(t *testing.T) {
	snap := testSnapshot()
	// Group C has no market; the simulator supplies the leg probability.
	sim := map[string]TournamentSimResult{"Brazil": {PWinGroup: 0.7}}
	s := StateJSON{
		Bets: []BetJSON{{
			ID: "ga-sim", Status: "alive", PotentialReturn: f(10),
			Legs: []LegJSON{{Group: "C", Team: "Brazil", Status: "alive"}},
		}},
	}
	applyLikelihoods(&s, snap, sim, nil)
	approxPtr(t, "sim fallback", s.Bets[0].Probability, 0.7)
}

func TestApplyLikelihoods_MatchAcca(t *testing.T) {
	snap := testSnapshot()
	s := StateJSON{
		MatchAccaBets: []MatchAccaBetJSON{{
			ID: "ma", Status: "alive", PotentialReturn: f(50),
			Legs: []MatchOutcomeLegJSON{
				{Team: "England", Opponent: "Croatia", Outcome: "win", Status: "alive"},
			},
		}},
	}
	applyLikelihoods(&s, snap, nil, nil)
	approxPtr(t, "match acca win leg", s.MatchAccaBets[0].Probability, 0.60)

	// Away perspective: Croatia to lose == England to win.
	s2 := StateJSON{
		MatchAccaBets: []MatchAccaBetJSON{{
			ID: "ma2", Status: "alive",
			Legs: []MatchOutcomeLegJSON{
				{Team: "Croatia", Opponent: "England", Outcome: "lose", Status: "alive"},
			},
		}},
	}
	applyLikelihoods(&s2, snap, nil, nil)
	approxPtr(t, "croatia-to-lose == england win", s2.MatchAccaBets[0].Probability, 0.60)
}

func TestApplyLikelihoods_ExactScoreAlignsTeams(t *testing.T) {
	snap := testSnapshot()
	s := StateJSON{
		MatchResultBets: []MatchResultBetJSON{
			{ID: "mr-home", TeamA: "England", ScoreA: 3, TeamB: "Croatia", ScoreB: 1, Status: "alive", PotentialReturn: f(75)},
			// Same scoreline expressed with teams reversed must align to the
			// market's home/away orientation.
			{ID: "mr-rev", TeamA: "Croatia", ScoreA: 1, TeamB: "England", ScoreB: 3, Status: "alive"},
			// A scoreline the market doesn't separately price → ~0.
			{ID: "mr-miss", TeamA: "England", ScoreA: 5, TeamB: "Croatia", ScoreB: 5, Status: "alive"},
		},
	}
	applyLikelihoods(&s, snap, nil, nil)
	approxPtr(t, "exact score home", s.MatchResultBets[0].Probability, 0.08)
	approxPtr(t, "exact score reversed", s.MatchResultBets[1].Probability, 0.08)
	approxPtr(t, "unpriced scoreline", s.MatchResultBets[2].Probability, 0.0)
}

func TestApplyLikelihoods_WinnerTopScorerFinalist(t *testing.T) {
	snap := testSnapshot()
	joints := map[string]float64{}
	fkey, _ := fixtureKey("France", "Spain")
	joints[fkey] = 0.05
	sim := map[string]TournamentSimResult{"Brazil": {PWinTournament: 0.18}}

	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw-mkt", Team: "England", Status: "alive", PotentialReturn: f(35)},
			{ID: "tw-sim", Team: "Brazil", Status: "alive"}, // not in winner market → sim
		},
		TopScorerBets: []TopScorerBetJSON{
			{ID: "ts", Player: "Harry Kane", Status: "alive", PotentialReturn: f(40)},
		},
		FinalistBets: []FinalistBetJSON{
			{ID: "fin", TeamA: "France", TeamB: "Spain", Status: "alive", PotentialReturn: f(105)},
		},
	}
	applyLikelihoods(&s, snap, sim, joints)

	approxPtr(t, "winner market", s.TournamentWinnerBets[0].Probability, 0.10)
	approxPtr(t, "winner sim fallback", s.TournamentWinnerBets[1].Probability, 0.18)
	approxPtr(t, "top scorer", s.TopScorerBets[0].Probability, 0.12)
	approxPtr(t, "finalist joint", s.FinalistBets[0].Probability, 0.05)
	approxPtr(t, "finalist expected return", s.FinalistBets[0].ExpectedReturn, round2(0.05*105))
}

func TestApplyLikelihoods_DecidedBetsShortCircuit(t *testing.T) {
	snap := testSnapshot()
	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "won", Team: "Nowhere", Status: "won", PotentialReturn: f(50)},
			{ID: "lost", Team: "Nowhere", Status: "lost", PotentialReturn: f(50)},
		},
	}
	applyLikelihoods(&s, snap, nil, nil)
	approxPtr(t, "won bet", s.TournamentWinnerBets[0].Probability, 1.0)
	approxPtr(t, "won expected return", s.TournamentWinnerBets[0].ExpectedReturn, 50.0)
	approxPtr(t, "lost bet", s.TournamentWinnerBets[1].Probability, 0.0)
}

func TestApplyLikelihoods_UnpricedBetOmitted(t *testing.T) {
	snap := testSnapshot()
	s := StateJSON{
		// No market, no sim entry → can't price → probability stays nil.
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "unknown", Team: "Atlantis", Status: "alive", PotentialReturn: f(50)},
		},
	}
	applyLikelihoods(&s, snap, nil, nil)
	if s.TournamentWinnerBets[0].Probability != nil {
		t.Errorf("unpriceable bet should have nil probability, got %v", *s.TournamentWinnerBets[0].Probability)
	}
}

func TestApplyLikelihoods_ExpectedPayoutAndProfit(t *testing.T) {
	snap := testSnapshot()
	s := StateJSON{
		MaxPayout: MaxPayoutJSON{TotalOutlay: 10},
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw", Team: "England", Status: "alive", PotentialReturn: f(35)}, // 0.10 * 35 = 3.5
		},
		TopScorerBets: []TopScorerBetJSON{
			{ID: "ts", Player: "Harry Kane", Status: "alive", PotentialReturn: f(40)}, // 0.12 * 40 = 4.8
		},
	}
	applyLikelihoods(&s, snap, nil, nil)
	if s.Expected == nil {
		t.Fatal("expected block not set")
	}
	wantPayout := round2(0.10*35 + 0.12*40) // 8.30
	if math.Abs(s.Expected.ExpectedPayout-wantPayout) > 1e-9 {
		t.Errorf("expected payout: got %v want %v", s.Expected.ExpectedPayout, wantPayout)
	}
	if math.Abs(s.Expected.ExpectedProfit-round2(wantPayout-10)) > 1e-9 {
		t.Errorf("expected profit: got %v want %v", s.Expected.ExpectedProfit, round2(wantPayout-10))
	}
}
