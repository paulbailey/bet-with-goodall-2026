package main

import (
	"math"
	"testing"
)

func f(v float64) *float64 { return &v }

func conflictIDs(mp MaxPayoutJSON) map[string]ConflictBetJSON {
	out := map[string]ConflictBetJSON{}
	for _, g := range mp.Conflicts {
		for _, b := range g.Bets {
			out[b.ID] = b
		}
	}
	return out
}

func approx(a, b float64) bool { return math.Abs(a-b) < 0.005 }

// All independent alive bets contribute their full return, and outlay/profit
// are summed across every bet.
func TestComputeMaxPayout_NoConflicts(t *testing.T) {
	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw-1", Team: "England", Stake: f(5), PotentialReturn: f(35), Status: "alive"},
		},
		TopScorerBets: []TopScorerBetJSON{
			{ID: "ts-1", Player: "Harry Kane", Stake: f(5), PotentialReturn: f(40), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 75) {
		t.Fatalf("max payout: want 75, got %v", mp.MaxPayout)
	}
	if !approx(mp.TotalOutlay, 10) {
		t.Fatalf("outlay: want 10, got %v", mp.TotalOutlay)
	}
	if !approx(mp.MaxProfit, 65) {
		t.Fatalf("profit: want 65, got %v", mp.MaxProfit)
	}
	if len(mp.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", mp.Conflicts)
	}
}

// A champion bet and a finalist bet that excludes that champion can't both win;
// the higher-return bet is counted and the other is dropped.
func TestComputeMaxPayout_ChampionVsFinalist(t *testing.T) {
	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw-1", Team: "England", Stake: f(5), PotentialReturn: f(35), Status: "alive"},
		},
		FinalistBets: []FinalistBetJSON{
			{ID: "fin-1", TeamA: "France", TeamB: "Spain", Stake: f(5), PotentialReturn: f(105), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	// Drop the smaller (tw-1, 35); keep fin-1 (105).
	if !approx(mp.MaxPayout, 105) {
		t.Fatalf("max payout: want 105, got %v", mp.MaxPayout)
	}
	c := conflictIDs(mp)
	if len(c) != 2 {
		t.Fatalf("expected 2 bets in conflict, got %d", len(c))
	}
	if !c["fin-1"].Chosen || c["tw-1"].Chosen {
		t.Fatalf("expected fin-1 chosen and tw-1 dropped, got %+v", c)
	}
}

// A finalist bet that includes the champion is compatible — both win together.
func TestComputeMaxPayout_ChampionWithinFinalistPair(t *testing.T) {
	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw-1", Team: "France", PotentialReturn: f(35), Status: "alive"},
		},
		FinalistBets: []FinalistBetJSON{
			{ID: "fin-1", TeamA: "France", TeamB: "Spain", PotentialReturn: f(105), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 140) {
		t.Fatalf("max payout: want 140, got %v", mp.MaxPayout)
	}
	if len(mp.Conflicts) != 0 {
		t.Fatalf("expected no conflicts, got %v", mp.Conflicts)
	}
}

// Two group-winner accas that disagree on one group's winner are exclusive.
func TestComputeMaxPayout_GroupWinnerClash(t *testing.T) {
	s := StateJSON{
		Bets: []BetJSON{
			{ID: "acca-1", PotentialReturn: f(200), Status: "alive", Legs: []LegJSON{
				{Group: "A", Team: "Mexico"}, {Group: "B", Team: "Switzerland"},
			}},
			{ID: "acca-2", PotentialReturn: f(150), Status: "alive", Legs: []LegJSON{
				{Group: "A", Team: "South Korea"}, {Group: "C", Team: "Brazil"},
			}},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 200) {
		t.Fatalf("max payout: want 200, got %v", mp.MaxPayout)
	}
}

// Accas that agree wherever they overlap (and otherwise differ) can co-exist.
func TestComputeMaxPayout_GroupWinnerCompatible(t *testing.T) {
	s := StateJSON{
		Bets: []BetJSON{
			{ID: "acca-1", PotentialReturn: f(200), Status: "alive", Legs: []LegJSON{
				{Group: "A", Team: "Mexico"},
			}},
			{ID: "acca-2", PotentialReturn: f(150), Status: "alive", Legs: []LegJSON{
				{Group: "A", Team: "Mexico"}, {Group: "C", Team: "Brazil"},
			}},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 350) {
		t.Fatalf("max payout: want 350, got %v", mp.MaxPayout)
	}
}

// An exact-score bet and a match-acca leg on the same fixture that imply
// different results are exclusive.
func TestComputeMaxPayout_FixtureResultClash(t *testing.T) {
	s := StateJSON{
		MatchResultBets: []MatchResultBetJSON{
			{ID: "mr-1", TeamA: "England", TeamB: "Croatia", ScoreA: 1, ScoreB: 3, PotentialReturn: f(75), Status: "alive"},
		},
		MatchAccaBets: []MatchAccaBetJSON{
			{ID: "ma-1", PotentialReturn: f(14), Status: "alive", Legs: []MatchOutcomeLegJSON{
				{Team: "England", Opponent: "Croatia", Outcome: "win"},
			}},
		},
	}
	mp := computeMaxPayout(s)
	// mr-1 says England lose 1-3; ma-1 says England win — exclusive, keep mr-1.
	if !approx(mp.MaxPayout, 75) {
		t.Fatalf("max payout: want 75, got %v", mp.MaxPayout)
	}
}

// An exact-score bet and an acca leg implying the same result co-exist.
func TestComputeMaxPayout_FixtureResultCompatible(t *testing.T) {
	s := StateJSON{
		MatchResultBets: []MatchResultBetJSON{
			{ID: "mr-1", TeamA: "England", TeamB: "Croatia", ScoreA: 3, ScoreB: 1, PotentialReturn: f(75), Status: "alive"},
		},
		MatchAccaBets: []MatchAccaBetJSON{
			{ID: "ma-1", PotentialReturn: f(14), Status: "alive", Legs: []MatchOutcomeLegJSON{
				{Team: "England", Opponent: "Croatia", Outcome: "win"},
			}},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 89) {
		t.Fatalf("max payout: want 89, got %v", mp.MaxPayout)
	}
}

// Two exact-score bets on the same fixture with different scores are exclusive,
// even though both are England wins.
func TestComputeMaxPayout_ExactScoreClash(t *testing.T) {
	s := StateJSON{
		MatchResultBets: []MatchResultBetJSON{
			{ID: "mr-1", TeamA: "England", TeamB: "Ghana", ScoreA: 3, ScoreB: 1, PotentialReturn: f(60), Status: "alive"},
			{ID: "mr-2", TeamA: "England", TeamB: "Ghana", ScoreA: 2, ScoreB: 0, PotentialReturn: f(50), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 60) {
		t.Fatalf("max payout: want 60, got %v", mp.MaxPayout)
	}
}

// Distinct top-scorer players are exclusive (one winner).
func TestComputeMaxPayout_TopScorerClash(t *testing.T) {
	s := StateJSON{
		TopScorerBets: []TopScorerBetJSON{
			{ID: "ts-1", Player: "Harry Kane", PotentialReturn: f(40), Status: "alive"},
			{ID: "ts-2", Player: "Kylian Mbappe", PotentialReturn: f(30), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 40) {
		t.Fatalf("max payout: want 40, got %v", mp.MaxPayout)
	}
}

// A won bet is realised and forced into the total; an alive bet that contradicts
// it is dropped even though the alive bet has the higher return.
func TestComputeMaxPayout_WonForcedInDropsConflictingAlive(t *testing.T) {
	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw-1", Team: "England", PotentialReturn: f(35), Status: "won"},
		},
		FinalistBets: []FinalistBetJSON{
			{ID: "fin-1", TeamA: "France", TeamB: "Spain", PotentialReturn: f(105), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.RealisedWinnings, 35) {
		t.Fatalf("realised: want 35, got %v", mp.RealisedWinnings)
	}
	// England already champion ⇒ France/Spain final impossible ⇒ fin-1 dropped.
	if !approx(mp.MaxPayout, 35) {
		t.Fatalf("max payout: want 35, got %v", mp.MaxPayout)
	}
	c := conflictIDs(mp)
	if !c["tw-1"].Chosen || c["fin-1"].Chosen {
		t.Fatalf("expected won tw-1 chosen and fin-1 dropped, got %+v", c)
	}
}

// Lost bets contribute their stake to outlay but never to the payout.
func TestComputeMaxPayout_LostExcludedFromPayout(t *testing.T) {
	s := StateJSON{
		TournamentWinnerBets: []TournamentWinnerBetJSON{
			{ID: "tw-1", Team: "England", Stake: f(5), PotentialReturn: f(35), Status: "lost"},
			{ID: "tw-2", Team: "Brazil", Stake: f(5), PotentialReturn: f(20), Status: "alive"},
		},
	}
	mp := computeMaxPayout(s)
	if !approx(mp.MaxPayout, 20) {
		t.Fatalf("max payout: want 20, got %v", mp.MaxPayout)
	}
	if !approx(mp.TotalOutlay, 10) {
		t.Fatalf("outlay: want 10, got %v", mp.TotalOutlay)
	}
	if len(mp.Conflicts) != 0 {
		t.Fatalf("lost bet shouldn't create a conflict group, got %v", mp.Conflicts)
	}
}

// A chain A-B-C where A and C don't conflict should keep both ends.
func TestComputeMaxPayout_NonClique(t *testing.T) {
	s := StateJSON{
		MatchResultBets: []MatchResultBetJSON{
			{ID: "a", TeamA: "England", TeamB: "Ghana", ScoreA: 3, ScoreB: 1, PotentialReturn: f(60), Status: "alive"},
		},
		MatchAccaBets: []MatchAccaBetJSON{
			// b conflicts with a (Ghana fixture) and with c (Panama fixture).
			{ID: "b", PotentialReturn: f(10), Status: "alive", Legs: []MatchOutcomeLegJSON{
				{Team: "England", Opponent: "Ghana", Outcome: "lose"},
				{Team: "England", Opponent: "Panama", Outcome: "lose"},
			}},
		},
		// c shares only the Panama fixture with b and requires a win.
	}
	s.MatchResultBets = append(s.MatchResultBets, MatchResultBetJSON{
		ID: "c", TeamA: "England", TeamB: "Panama", ScoreA: 2, ScoreB: 0, PotentialReturn: f(50), Status: "alive",
	})
	mp := computeMaxPayout(s)
	// a + c (60 + 50) beats b (10); both ends kept.
	if !approx(mp.MaxPayout, 110) {
		t.Fatalf("max payout: want 110, got %v", mp.MaxPayout)
	}
	c := conflictIDs(mp)
	if len(c) != 3 || !c["a"].Chosen || c["b"].Chosen || !c["c"].Chosen {
		t.Fatalf("expected a & c chosen, b dropped, got %+v", c)
	}
}
