package main

import (
	"math"
	"math/rand"
	"strings"
)

// Knockout-stage simulator for the WC2026 bracket. It seeds the group-stage
// qualifiers into the official 32-slot bracket, then plays single-elimination
// rounds from team strengths, so finalist and tournament-winner bets get
// likelihoods that respect the real draw structure (e.g. the two finalists must
// come from opposite halves).
//
// The bracket below is the verified WC2026 layout. The 32 slots are listed in
// tree order: adjacent slots meet in the Round of 32, adjacent winners in the
// Round of 16, and so on, so slots 0–15 form the upper half and 16–31 the
// lower. Group winners (pos 1) and runners-up (pos 2) seed directly; the eight
// best third-placed teams (pos 3) are assigned to their slots by matching each
// third to a slot whose allowed-group set contains its group.

type slotPos int

const (
	posWinner   slotPos = 1
	posRunnerUp slotPos = 2
	posThird    slotPos = 3
)

type slotRef struct {
	pos     slotPos
	group   string   // group letter, for posWinner / posRunnerUp
	allowed []string // candidate group letters, for posThird
}

// wc2026Bracket lists the 32 Round-of-32 slots in bracket-tree order. Derived
// from the official R32 pairings and the R16→Final adjacency.
var wc2026Bracket = []slotRef{
	{pos: posRunnerUp, group: "A"},                              // 0  2A  ─┐M73
	{pos: posRunnerUp, group: "B"},                              // 1  2B  ─┘
	{pos: posWinner, group: "F"},                                // 2  1F  ─┐M75
	{pos: posRunnerUp, group: "C"},                              // 3  2C  ─┘
	{pos: posWinner, group: "E"},                                // 4  1E  ─┐M74
	{pos: posThird, allowed: []string{"A", "B", "C", "D", "F"}}, // 5  3ABCDF
	{pos: posWinner, group: "I"},                                // 6  1I  ─┐M77
	{pos: posThird, allowed: []string{"C", "D", "F", "G", "H"}}, // 7  3CDFGH
	{pos: posRunnerUp, group: "K"},                              // 8  2K  ─┐M83
	{pos: posRunnerUp, group: "L"},                              // 9  2L  ─┘
	{pos: posWinner, group: "H"},                                // 10 1H  ─┐M84
	{pos: posRunnerUp, group: "J"},                              // 11 2J  ─┘
	{pos: posWinner, group: "D"},                                // 12 1D  ─┐M81
	{pos: posThird, allowed: []string{"B", "E", "F", "I", "J"}}, // 13 3BEFIJ
	{pos: posWinner, group: "G"},                                // 14 1G  ─┐M82
	{pos: posThird, allowed: []string{"A", "E", "H", "I", "J"}}, // 15 3AEHIJ
	{pos: posWinner, group: "C"},                                // 16 1C  ─┐M76
	{pos: posRunnerUp, group: "F"},                              // 17 2F  ─┘
	{pos: posRunnerUp, group: "E"},                              // 18 2E  ─┐M78
	{pos: posRunnerUp, group: "I"},                              // 19 2I  ─┘
	{pos: posWinner, group: "A"},                                // 20 1A  ─┐M79
	{pos: posThird, allowed: []string{"C", "E", "F", "H", "I"}}, // 21 3CEFHI
	{pos: posWinner, group: "L"},                                // 22 1L  ─┐M80
	{pos: posThird, allowed: []string{"E", "H", "I", "J", "K"}}, // 23 3EHIJK
	{pos: posWinner, group: "J"},                                // 24 1J  ─┐M86
	{pos: posRunnerUp, group: "H"},                              // 25 2H  ─┘
	{pos: posRunnerUp, group: "D"},                              // 26 2D  ─┐M88
	{pos: posRunnerUp, group: "G"},                              // 27 2G  ─┘
	{pos: posWinner, group: "B"},                                // 28 1B  ─┐M85
	{pos: posThird, allowed: []string{"E", "F", "G", "I", "J"}}, // 29 3EFGIJ
	{pos: posWinner, group: "K"},                                // 30 1K  ─┐M87
	{pos: posThird, allowed: []string{"D", "E", "I", "J", "L"}}, // 31 3DEIJL
}

// TournamentSimResult holds a team's estimated probabilities across the whole
// tournament.
type TournamentSimResult struct {
	PWinGroup      float64
	PQualify       float64
	PReachFinal    float64
	PWinTournament float64
}

// simulateTournament runs the full tournament `iterations` times — group stage
// (conditioned on finished results) then the knockout bracket — and returns
// per-team probabilities. The knockout phase only runs when the bracket can be
// fully seeded (all groups present); otherwise only group-stage figures are
// populated.
func simulateTournament(groups []GroupStanding, matches []Match, strength map[string]float64, iterations int, seed int64) map[string]TournamentSimResult {
	rng := rand.New(rand.NewSource(seed))
	results := finishedGroupResults(matches)
	ratingFor := strengthLookup(strength)
	thirdsToQualify := thirdsQualifying(len(groups))

	win := map[string]int{}
	qual := map[string]int{}
	reachFinal := map[string]int{}
	champion := map[string]int{}

	for iter := 0; iter < iterations; iter++ {
		ranked, qualThirds := simulateGroupsOnce(groups, results, ratingFor, thirdsToQualify, rng)
		for _, r := range ranked {
			win[lowerName(r[0].name)]++
			qual[lowerName(r[0].name)]++
			if len(r) > 1 {
				qual[lowerName(r[1].name)]++
			}
		}
		for _, t := range qualThirds {
			qual[lowerName(t.name)]++
		}

		slots, ok := buildBracketSlots(ranked, qualThirds)
		if !ok {
			continue
		}
		champ, finalists := simulateBracket(slots, rng)
		champion[lowerName(champ.name)]++
		for _, f := range finalists {
			if f != nil {
				reachFinal[lowerName(f.name)]++
			}
		}
	}

	out := make(map[string]TournamentSimResult)
	inv := 1.0 / float64(iterations)
	for _, g := range groups {
		for _, t := range g.Teams {
			key := lowerName(t.Name)
			out[t.Name] = TournamentSimResult{
				PWinGroup:      float64(win[key]) * inv,
				PQualify:       float64(qual[key]) * inv,
				PReachFinal:    float64(reachFinal[key]) * inv,
				PWinTournament: float64(champion[key]) * inv,
			}
		}
	}
	return out
}

// buildBracketSlots seeds the qualifiers into the 32 bracket slots in tree
// order. Returns false if a winner/runner-up slot can't be filled (incomplete
// groups).
func buildBracketSlots(ranked map[string][]*simTeam, qualThirds []*simTeam) ([]*simTeam, bool) {
	slots := make([]*simTeam, len(wc2026Bracket))

	// Direct seeds: group winners and runners-up.
	var thirdSlotIdx []int
	for i, ref := range wc2026Bracket {
		switch ref.pos {
		case posWinner:
			r := ranked[strings.ToUpper(ref.group)]
			if len(r) < 1 {
				return nil, false
			}
			slots[i] = r[0]
		case posRunnerUp:
			r := ranked[strings.ToUpper(ref.group)]
			if len(r) < 2 {
				return nil, false
			}
			slots[i] = r[1]
		case posThird:
			thirdSlotIdx = append(thirdSlotIdx, i)
		}
	}

	// Match each qualifying third to a slot whose allowed set contains its group.
	allowed := make([][]string, len(thirdSlotIdx))
	for i, si := range thirdSlotIdx {
		allowed[i] = wc2026Bracket[si].allowed
	}
	assign := matchThirdsToSlots(qualThirds, allowed)
	used := make([]bool, len(thirdSlotIdx))
	var unplaced []*simTeam
	for ti, si := range assign {
		if si < 0 {
			unplaced = append(unplaced, qualThirds[ti])
			continue
		}
		slots[thirdSlotIdx[si]] = qualThirds[ti]
		used[si] = true
	}
	// Fallback: drop any unmatched third into a remaining empty third slot.
	for _, t := range unplaced {
		for si := range thirdSlotIdx {
			if !used[si] {
				slots[thirdSlotIdx[si]] = t
				used[si] = true
				break
			}
		}
	}

	for _, s := range slots {
		if s == nil {
			return nil, false
		}
	}
	return slots, true
}

// matchThirdsToSlots returns, for each third, the index of the third-slot it's
// assigned to (or -1), via maximum bipartite matching on allowed-group sets
// (Kuhn's algorithm). FIFA designs the chart so a perfect matching always
// exists for a valid set of qualifying thirds.
func matchThirdsToSlots(thirds []*simTeam, slotAllowed [][]string) []int {
	n, m := len(thirds), len(slotAllowed)
	canTake := make([][]bool, n)
	for ti := range thirds {
		canTake[ti] = make([]bool, m)
		for si := range slotAllowed {
			canTake[ti][si] = containsFold(slotAllowed[si], thirds[ti].group)
		}
	}

	slotOwner := make([]int, m) // slot -> third index, or -1
	for i := range slotOwner {
		slotOwner[i] = -1
	}

	var augment func(ti int, seen []bool) bool
	augment = func(ti int, seen []bool) bool {
		for si := 0; si < m; si++ {
			if !canTake[ti][si] || seen[si] {
				continue
			}
			seen[si] = true
			if slotOwner[si] == -1 || augment(slotOwner[si], seen) {
				slotOwner[si] = ti
				return true
			}
		}
		return false
	}
	for ti := 0; ti < n; ti++ {
		augment(ti, make([]bool, m))
	}

	assign := make([]int, n)
	for i := range assign {
		assign[i] = -1
	}
	for si, ti := range slotOwner {
		if ti != -1 {
			assign[ti] = si
		}
	}
	return assign
}

func containsFold(set []string, s string) bool {
	for _, v := range set {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// simulateBracket plays the single-elimination rounds over slots given in tree
// order (adjacent slots meet first). Returns the champion and the two finalists
// (one from each half of the draw).
func simulateBracket(slots []*simTeam, rng *rand.Rand) (champion *simTeam, finalists [2]*simTeam) {
	round := make([]*simTeam, len(slots))
	copy(round, slots)
	for len(round) > 1 {
		if len(round) == 2 {
			finalists = [2]*simTeam{round[0], round[1]}
		}
		next := round[:0:0]
		for i := 0; i+1 < len(round); i += 2 {
			next = append(next, koWinner(round[i], round[i+1], rng))
		}
		round = next
	}
	if len(round) == 1 {
		champion = round[0]
	}
	return champion, finalists
}

// koWinner resolves one knockout tie: simulated goals, then a strength-weighted
// shootout if level.
func koWinner(a, b *simTeam, rng *rand.Rand) *simTeam {
	la, lb := matchLambdas(a.rating, b.rating)
	ga, gb := poissonSample(rng, la), poissonSample(rng, lb)
	switch {
	case ga > gb:
		return a
	case gb > ga:
		return b
	default:
		pa := 1 / (1 + math.Exp(-strengthScale*(a.rating-b.rating)))
		if rng.Float64() < pa {
			return a
		}
		return b
	}
}
