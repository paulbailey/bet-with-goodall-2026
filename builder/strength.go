package main

import "math"

// Strength derivation. The Monte Carlo simulator needs a per-team rating on the
// log-goals scale (see simulate.go). Rather than hand-pick those, we calibrate
// them to the market: ratings are nudged until the simulator reproduces the
// de-vigged WINNER (tournament outright) probabilities. This keeps the bracket
// realistic — favourites are favoured by exactly as much as the market says —
// which is what the finalist joint-probability bet relies on.
//
// Calibration is a fixed-point iteration. Each round we run a (cheap) tournament
// sim with the current ratings, then for every team adjust its rating toward its
// market target on the log scale:
//
//	rating += lr · (ln target − ln simulated)
//
// because raising a rating raises that team's win probability monotonically.
// Ratings are re-centred to mean zero each round so the absolute level stays
// pinned to avgGoals. The per-step move is clamped to keep noisy long-shots from
// exploding.

const (
	calibFloorProb = 1e-4 // floor for log() of tiny probabilities
	calibSeedScale = 0.8  // closed-form seed: rating ≈ scale·ln(p / geomean)
	// Only teams the market gives a real chance are iteratively calibrated; true
	// long-shots stay at their (low) seed. This stops a large tail of ~zero-win
	// teams getting max-clamped upward pushes each round, which would inflate the
	// mean and, via re-centring, drag the genuine favourites back down.
	calibMinTarget = 0.01
)

// calibConfig holds the fixed-point iteration's tunables. Production uses
// defaultCalib (accuracy); tests can dial it down for speed.
type calibConfig struct {
	rounds     int
	iterations int
	learnRate  float64
	clamp      float64 // max |rating change| per team per round
}

func defaultCalib() calibConfig {
	return calibConfig{rounds: 20, iterations: 6000, learnRate: 0.35, clamp: 0.5}
}

// deriveStrength returns log-goals ratings keyed by lower-case team name,
// calibrated so simulateTournament reproduces the WINNER market. When the market
// is empty (e.g. Betfair unavailable) it returns an empty map, which the
// simulator reads as a flat, equal-strength field.
func deriveStrength(groups []GroupStanding, winnerProbs []OutrightProb, matches []Match, seed int64) map[string]float64 {
	return deriveStrengthWith(groups, winnerProbs, matches, seed, defaultCalib())
}

func deriveStrengthWith(groups []GroupStanding, winnerProbs []OutrightProb, matches []Match, seed int64, cfg calibConfig) map[string]float64 {
	names := teamNames(groups)
	target := map[string]float64{} // lower name -> market P(win tournament)
	for _, w := range winnerProbs {
		key := lowerName(w.Selection)
		if w.Prob > 0 && containsName(names, key) {
			target[key] = w.Prob
		}
	}
	if len(target) == 0 || len(names) == 0 {
		return map[string]float64{}
	}

	ratings := seedRatings(names, target)

	for round := 0; round < cfg.rounds; round++ {
		sim := simulateTournament(groups, matches, ratings, cfg.iterations, seed+int64(round))
		for _, name := range names {
			key := lowerName(name)
			tgt, ok := target[key]
			if !ok || tgt < calibMinTarget {
				continue // no market view, or a long-shot pinned at its seed
			}
			simP := clampProb(sim[name].PWinTournament)
			delta := cfg.learnRate * (math.Log(clampProb(tgt)) - math.Log(simP))
			ratings[key] += clampDelta(delta, cfg.clamp)
		}
		recentre(names, ratings)
	}
	return ratings
}

// seedRatings gives the calibration a sensible starting point: a closed-form
// rating proportional to the log of each team's market win probability, centred
// on the geometric mean. Teams with no market price start at the mean (0).
func seedRatings(names []string, target map[string]float64) map[string]float64 {
	var sumLog float64
	var n int
	for _, p := range target {
		sumLog += math.Log(clampProb(p))
		n++
	}
	meanLog := sumLog / float64(n)

	ratings := make(map[string]float64, len(names))
	for _, name := range names {
		key := lowerName(name)
		if p, ok := target[key]; ok {
			ratings[key] = calibSeedScale * (math.Log(clampProb(p)) - meanLog)
		} else {
			ratings[key] = 0
		}
	}
	recentre(names, ratings)
	return ratings
}

func recentre(names []string, ratings map[string]float64) {
	if len(names) == 0 {
		return
	}
	var sum float64
	for _, name := range names {
		sum += ratings[lowerName(name)]
	}
	mean := sum / float64(len(names))
	for _, name := range names {
		ratings[lowerName(name)] -= mean
	}
}

func teamNames(groups []GroupStanding) []string {
	var names []string
	for _, g := range groups {
		for _, t := range g.Teams {
			names = append(names, t.Name)
		}
	}
	return names
}

func containsName(names []string, lowerKey string) bool {
	for _, n := range names {
		if lowerName(n) == lowerKey {
			return true
		}
	}
	return false
}

func clampProb(p float64) float64 {
	if p < calibFloorProb {
		return calibFloorProb
	}
	if p > 1 {
		return 1
	}
	return p
}

func clampDelta(d, clamp float64) float64 {
	if d > clamp {
		return clamp
	}
	if d < -clamp {
		return -clamp
	}
	return d
}
