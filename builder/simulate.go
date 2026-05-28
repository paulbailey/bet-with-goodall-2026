package main

import (
	"math"
	"math/rand"
	"sort"
	"strings"
)

func lowerName(s string) string { return strings.ToLower(s) }

// Monte Carlo tournament simulator. It estimates per-bet likelihoods that the
// closed-form odds layer can't price directly — group-winner accumulators
// (correlated across the bracket) and, later, knockout-dependent bets
// (finalists, tournament winner).
//
// The model: each team has a strength rating on a log-goals scale. For a fixture
// the expected goals are avgGoals·exp(±scale·Δrating); goals are independent
// Poisson draws. Matches already finished are taken as-is, so the estimates are
// *conditional* on results to date and tighten as the tournament unfolds.
//
// Strength ratings are supplied by the caller — derived from market odds once
// the Betfair provider lands (calibrated to the winner / match-odds markets),
// with an Elo/ranking seed as a pre-tournament fallback.
//
// This first cut implements the group stage. The knockout bracket (which feeds
// finalist and tournament-winner likelihoods) is the next layer and slots onto
// the per-iteration group results produced here.

const (
	avgGoals      = 1.30 // baseline expected goals per team between equals
	strengthScale = 1.00 // how sharply a rating gap swings expected goals
	koTeams       = 32   // teams advancing to the knockout stage (WC2026)
)

// GroupSimResult holds estimated group-stage probabilities for one team.
type GroupSimResult struct {
	PWinGroup float64 // finishes 1st in its group
	PQualify  float64 // reaches the knockout stage (top 2, or a best-placed 3rd)
}

type simTeam struct {
	name           string
	group          string // group letter (upper-case), for bracket seeding
	rating         float64
	points, gf, ga int
	tiebreak       float64 // per-iteration random, splits exact ties
}

func (t *simTeam) gd() int { return t.gf - t.ga }

// matchLambdas returns the expected goals for each side given their ratings.
func matchLambdas(ratingA, ratingB float64) (lambdaA, lambdaB float64) {
	d := ratingA - ratingB
	return avgGoals * math.Exp(strengthScale*d), avgGoals * math.Exp(-strengthScale*d)
}

// poissonSample draws a non-negative integer from Poisson(lambda) (Knuth).
func poissonSample(rng *rand.Rand, lambda float64) int {
	l := math.Exp(-lambda)
	k, p := 0, 1.0
	for {
		k++
		p *= rng.Float64()
		if p <= l {
			return k - 1
		}
	}
}

// simulateGroupStage runs the group stage `iterations` times and returns each
// team's estimated probability of winning its group and of qualifying. Results
// already present in `matches` (FINISHED group fixtures) are held fixed; the
// remaining round-robin fixtures are simulated from `strength` (rating by team
// name, case-insensitive; missing teams default to 0).
func simulateGroupStage(groups []GroupStanding, matches []Match, strength map[string]float64, iterations int, seed int64) map[string]GroupSimResult {
	rng := rand.New(rand.NewSource(seed))
	results := finishedGroupResults(matches)
	ratingFor := strengthLookup(strength)
	thirdsToQualify := thirdsQualifying(len(groups))

	winCount := map[string]int{}
	qualifyCount := map[string]int{}

	for iter := 0; iter < iterations; iter++ {
		ranked, qualThirds := simulateGroupsOnce(groups, results, ratingFor, thirdsToQualify, rng)
		for _, r := range ranked {
			winCount[lowerName(r[0].name)]++
			qualifyCount[lowerName(r[0].name)]++
			if len(r) > 1 {
				qualifyCount[lowerName(r[1].name)]++
			}
		}
		for _, t := range qualThirds {
			qualifyCount[lowerName(t.name)]++
		}
	}

	out := make(map[string]GroupSimResult)
	inv := 1.0 / float64(iterations)
	for _, g := range groups {
		for _, t := range g.Teams {
			key := lowerName(t.Name)
			out[t.Name] = GroupSimResult{
				PWinGroup: float64(winCount[key]) * inv,
				PQualify:  float64(qualifyCount[key]) * inv,
			}
		}
	}
	return out
}

// strengthLookup wraps a case-insensitive rating map, defaulting to 0.
func strengthLookup(strength map[string]float64) func(string) float64 {
	return func(name string) float64 {
		if r, ok := strength[lowerName(name)]; ok {
			return r
		}
		return 0
	}
}

// thirdsQualifying is how many best-third-placed teams advance given the group
// count (knockout field minus the two automatic qualifiers per group).
func thirdsQualifying(numGroups int) int {
	n := koTeams - 2*numGroups
	if n < 0 {
		return 0
	}
	return n
}

// simulateGroupsOnce plays one full group stage: every group's round-robin
// (finished fixtures held fixed, the rest simulated), returning each group's
// teams ranked 1st→4th and the best third-placed teams that qualify.
func simulateGroupsOnce(groups []GroupStanding, results map[string][2]int, ratingFor func(string) float64, thirdsToQualify int, rng *rand.Rand) (map[string][]*simTeam, []*simTeam) {
	ranked := make(map[string][]*simTeam, len(groups))
	var thirds []*simTeam

	for _, g := range groups {
		letter := strings.ToUpper(g.Group)
		teams := make([]*simTeam, len(g.Teams))
		for i := range g.Teams {
			teams[i] = &simTeam{
				name:     g.Teams[i].Name,
				group:    letter,
				rating:   ratingFor(g.Teams[i].Name),
				tiebreak: rng.Float64(),
			}
		}

		// Round-robin: every distinct pair meets once.
		for i := 0; i < len(teams); i++ {
			for j := i + 1; j < len(teams); j++ {
				a, b := teams[i], teams[j]
				ga, gb := scoreFor(results, a.name, b.name, rng, a.rating, b.rating)
				applyResult(a, b, ga, gb)
			}
		}

		r := rankGroup(teams)
		ranked[letter] = r
		if len(r) > 2 {
			thirds = append(thirds, r[2])
		}
	}

	// Best third-placed teams across all groups fill the remaining slots.
	sort.SliceStable(thirds, func(i, j int) bool { return lessForRank(thirds[i], thirds[j]) })
	if thirdsToQualify > len(thirds) {
		thirdsToQualify = len(thirds)
	}
	return ranked, thirds[:thirdsToQualify]
}

func applyResult(a, b *simTeam, ga, gb int) {
	a.gf += ga
	a.ga += gb
	b.gf += gb
	b.ga += ga
	switch {
	case ga > gb:
		a.points += 3
	case gb > ga:
		b.points += 3
	default:
		a.points++
		b.points++
	}
}

// scoreFor returns the goals for (a, b): the real result if this fixture has
// finished, otherwise a simulated scoreline from the two ratings.
func scoreFor(results map[string][2]int, a, b string, rng *rand.Rand, ra, rb float64) (int, int) {
	key, aIsFirst := fixtureKey(a, b)
	if res, ok := results[key]; ok {
		if aIsFirst {
			return res[0], res[1]
		}
		return res[1], res[0]
	}
	la, lb := matchLambdas(ra, rb)
	return poissonSample(rng, la), poissonSample(rng, lb)
}

// finishedGroupResults indexes finished group-stage scores by canonical fixture
// key (goals stored for canonical-first, canonical-second team).
func finishedGroupResults(matches []Match) map[string][2]int {
	out := map[string][2]int{}
	for i := range matches {
		m := &matches[i]
		if m.Status != "FINISHED" || m.HomeScore == nil || m.AwayScore == nil {
			continue
		}
		if m.Group == "" && !isGroupStage(m.Stage) {
			continue
		}
		key, homeIsFirst := fixtureKey(m.HomeTeam, m.AwayTeam)
		if homeIsFirst {
			out[key] = [2]int{*m.HomeScore, *m.AwayScore}
		} else {
			out[key] = [2]int{*m.AwayScore, *m.HomeScore}
		}
	}
	return out
}

func isGroupStage(stage string) bool {
	return stage == "GROUP_STAGE" || stage == "GROUP" || stage == ""
}

// rankGroup orders teams 1st→last by the FIFA group criteria implemented here:
// points, goal difference, goals scored, then a per-iteration random tiebreak.
// Head-to-head tiebreakers are not modelled (a minor simplification that rarely
// changes the group-winner distribution).
func rankGroup(teams []*simTeam) []*simTeam {
	ranked := make([]*simTeam, len(teams))
	copy(ranked, teams)
	sort.SliceStable(ranked, func(i, j int) bool { return lessForRank(ranked[i], ranked[j]) })
	return ranked
}

// lessForRank reports whether a ranks ABOVE b (better standing first).
func lessForRank(a, b *simTeam) bool {
	if a.points != b.points {
		return a.points > b.points
	}
	if a.gd() != b.gd() {
		return a.gd() > b.gd()
	}
	if a.gf != b.gf {
		return a.gf > b.gf
	}
	return a.tiebreak > b.tiebreak
}
