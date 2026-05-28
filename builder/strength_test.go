package main

import (
	"math"
	"testing"
)

// fastCalib is a lighter fixed-point config for tests: fewer rounds and sim
// iterations than production so the suite stays quick, while still converging
// enough to recover a bracket-self-consistent market.
func fastCalib() calibConfig {
	return calibConfig{rounds: 12, iterations: 4000, learnRate: 0.4, clamp: 0.5}
}

// marketFromRatings runs the simulator with known ratings and turns the
// resulting championship probabilities into a WINNER market. Because the target
// comes straight out of the bracket, it's perfectly self-consistent — exactly
// the kind of market calibration should be able to recover.
func marketFromRatings(groups []GroupStanding, ratings map[string]float64, iters int, seed int64) ([]OutrightProb, map[string]float64) {
	res := simulateTournament(groups, nil, ratings, iters, seed)
	out := make([]OutrightProb, 0, len(res))
	target := make(map[string]float64, len(res))
	for name, r := range res {
		out = append(out, OutrightProb{Selection: name, Prob: r.PWinTournament})
		target[lowerName(name)] = r.PWinTournament
	}
	return out, target
}

func TestDeriveStrength_EmptyMarketIsFlat(t *testing.T) {
	r := deriveStrength(fullGroups(), nil, nil, 1)
	if len(r) != 0 {
		t.Fatalf("no market should yield an empty (flat) rating map, got %d entries", len(r))
	}
}

func TestDeriveStrength_FavouriteRatedHighest(t *testing.T) {
	groups := fullGroups()
	// One clear favourite per group so the bracket can actually realise the
	// market (favourites don't have to knock each other out early).
	truth := map[string]float64{"a1": 1.6, "d1": 1.1, "g1": 0.8}
	market, _ := marketFromRatings(groups, truth, 12000, 101)

	ratings := deriveStrengthWith(groups, market, nil, 42, fastCalib())
	if len(ratings) == 0 {
		t.Fatal("expected calibrated ratings")
	}
	favR := ratings["a1"]
	for key, r := range ratings {
		if key == "a1" {
			continue
		}
		if r >= favR {
			t.Fatalf("favourite A1 (%.3f) should outrate %s (%.3f)", favR, key, r)
		}
	}
}

func TestDeriveStrength_RoundTripReproducesMarket(t *testing.T) {
	groups := fullGroups()
	// Ground-truth ratings, one strong team per group so the favourites sit in
	// different bracket paths (a self-consistent market).
	truth := map[string]float64{"a1": 1.6, "d1": 1.2, "g1": 1.0, "j1": 0.7}
	market, target := marketFromRatings(groups, truth, 12000, 101)

	ratings := deriveStrengthWith(groups, market, nil, 7, fastCalib())

	// Feed the recovered ratings back through the simulator; each favourite's
	// championship probability should land near its market target (wide band for
	// MC noise + finite calibration rounds).
	res := simulateTournament(groups, nil, ratings, 12000, 7)
	for _, name := range []string{"A1", "D1", "G1"} {
		want := target[lowerName(name)]
		got := res[name].PWinTournament
		if math.Abs(got-want) > 0.05 {
			t.Fatalf("calibrated %s win prob %.3f should be near market %.3f", name, got, want)
		}
	}
}
