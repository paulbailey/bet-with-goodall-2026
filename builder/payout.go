package main

import (
	"fmt"
	"sort"
	"strings"
)

// MaxPayoutJSON summarises the best-case outcome across every bet. It answers
// "if everything still possible went our way, how much could we win?" — while
// respecting that some bets contradict each other (you can't have two different
// group winners, two different champions, etc.) so they can't all win at once.
type MaxPayoutJSON struct {
	MaxPayout        float64             `json:"max_payout"`        // realised + best consistent set of alive returns
	RealisedWinnings float64             `json:"realised_winnings"` // returns already locked in (status "won")
	TotalOutlay      float64             `json:"total_outlay"`      // every stake, win or lose
	MaxProfit        float64             `json:"max_profit"`        // max_payout − total_outlay
	Conflicts        []ConflictGroupJSON `json:"conflicts"`         // groups of bets that can't all win together
}

// ConflictGroupJSON is one cluster of mutually-incompatible bets. The frontend
// renders it so a reader can see why two returns weren't both counted.
type ConflictGroupJSON struct {
	Bets []ConflictBetJSON `json:"bets"`
}

type ConflictBetJSON struct {
	ID     string  `json:"id"`
	Label  string  `json:"label"`
	Return float64 `json:"return"`
	Status string  `json:"status"` // alive | won
	Chosen bool    `json:"chosen"` // true if its return is counted toward max_payout
}

// fixture-result codes, relative to the alphabetically-first team of a fixture.
const (
	resWin  = 0 // canonical-first team wins
	resDraw = 1
	resLose = 2 // canonical-first team loses
)

type fixtureClaim struct {
	key         string // canonical "teamlo\x00teamhi"
	firstResult int    // res* for the canonical-first team
	hasExact    bool
	scoreFirst  int
	scoreSecond int
}

// claim captures everything about a bet that can clash with another bet. A
// given bet populates only the fields relevant to its kind.
type claim struct {
	groupWinners map[string]string // group -> predicted winner
	champion     string            // tournament-winner team
	finalA       string            // finalist pair (order-insensitive)
	finalB       string
	topScorer    string
	fixtures     []fixtureClaim // match-result / match-acca legs
}

type payoutNode struct {
	id    string
	label string
	ret   float64
	won   bool // status == "won" (forced into the selection)
	claim claim
}

func computeMaxPayout(s StateJSON) MaxPayoutJSON {
	var nodes []payoutNode
	var outlay, realised float64

	add := func(id, label string, stake, ret *float64, status string, c claim) {
		if stake != nil {
			outlay += *stake
		}
		r := 0.0
		if ret != nil {
			r = *ret
		}
		switch status {
		case "lost":
			return // can never pay out
		case "won":
			realised += r
		}
		nodes = append(nodes, payoutNode{id: id, label: label, ret: r, won: status == "won", claim: c})
	}

	for _, b := range s.Bets {
		gw := make(map[string]string, len(b.Legs))
		for _, leg := range b.Legs {
			gw[leg.Group] = leg.Team
		}
		add(b.ID, fmt.Sprintf("Group-winner accumulator (%d legs)", len(b.Legs)), b.Stake, b.PotentialReturn, b.Status, claim{groupWinners: gw})
	}

	for _, b := range s.TopScorerBets {
		add(b.ID, fmt.Sprintf("%s top scorer", b.Player), b.Stake, b.PotentialReturn, b.Status, claim{topScorer: b.Player})
	}

	for _, b := range s.TournamentWinnerBets {
		add(b.ID, fmt.Sprintf("%s to win the tournament", b.Team), b.Stake, b.PotentialReturn, b.Status, claim{champion: b.Team})
	}

	for _, b := range s.MatchResultBets {
		add(b.ID, fmt.Sprintf("%s %d-%d %s", b.TeamA, b.ScoreA, b.ScoreB, b.TeamB), b.Stake, b.PotentialReturn, b.Status,
			claim{fixtures: []fixtureClaim{exactFixtureClaim(b.TeamA, b.TeamB, b.ScoreA, b.ScoreB)}})
	}

	for _, b := range s.MatchAccaBets {
		fx := make([]fixtureClaim, len(b.Legs))
		for i, leg := range b.Legs {
			fx[i] = outcomeFixtureClaim(leg.Team, leg.Opponent, leg.Outcome)
		}
		add(b.ID, fmt.Sprintf("Match accumulator (%d legs)", len(b.Legs)), b.Stake, b.PotentialReturn, b.Status, claim{fixtures: fx})
	}

	for _, b := range s.FinalistBets {
		add(b.ID, fmt.Sprintf("%s & %s to reach the final", b.TeamA, b.TeamB), b.Stake, b.PotentialReturn, b.Status,
			claim{finalA: b.TeamA, finalB: b.TeamB})
	}

	adj := buildConflictGraph(nodes)
	maxPayout, chosen, components := maxWeightSelection(nodes, adj)

	out := MaxPayoutJSON{
		MaxPayout:        round2(maxPayout),
		RealisedWinnings: round2(realised),
		TotalOutlay:      round2(outlay),
		MaxProfit:        round2(maxPayout - outlay),
		Conflicts:        []ConflictGroupJSON{},
	}

	for _, comp := range components {
		if len(comp) < 2 {
			continue // a lone node clashes with nothing
		}
		grp := ConflictGroupJSON{}
		for _, i := range comp {
			grp.Bets = append(grp.Bets, ConflictBetJSON{
				ID:     nodes[i].id,
				Label:  nodes[i].label,
				Return: round2(nodes[i].ret),
				Status: statusOf(nodes[i]),
				Chosen: chosen[i],
			})
		}
		out.Conflicts = append(out.Conflicts, grp)
	}

	return out
}

func statusOf(n payoutNode) string {
	if n.won {
		return "won"
	}
	return "alive"
}

// buildConflictGraph returns an adjacency matrix where adj[i][j] is true when
// bets i and j cannot both win. Edges between two already-won bets are skipped:
// realised results are, by definition, mutually consistent.
func buildConflictGraph(nodes []payoutNode) [][]bool {
	n := len(nodes)
	adj := make([][]bool, n)
	for i := range adj {
		adj[i] = make([]bool, n)
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if nodes[i].won && nodes[j].won {
				continue
			}
			if claimsConflict(nodes[i].claim, nodes[j].claim) {
				adj[i][j] = true
				adj[j][i] = true
			}
		}
	}
	return adj
}

func claimsConflict(a, b claim) bool {
	// Two group-winner accas clash if they name different winners for one group.
	if len(a.groupWinners) > 0 && len(b.groupWinners) > 0 {
		for g, t := range a.groupWinners {
			if t2, ok := b.groupWinners[g]; ok && !teamsMatch(t, t2) {
				return true
			}
		}
	}

	// Only one team can win the tournament.
	if a.champion != "" && b.champion != "" && !teamsMatch(a.champion, b.champion) {
		return true
	}

	// The champion must be one of the two finalists.
	if a.champion != "" && b.finalA != "" && !teamIn(a.champion, b.finalA, b.finalB) {
		return true
	}
	if b.champion != "" && a.finalA != "" && !teamIn(b.champion, a.finalA, a.finalB) {
		return true
	}

	// The final has exactly two teams, so different finalist pairs are exclusive.
	if a.finalA != "" && b.finalA != "" && !samePair(a.finalA, a.finalB, b.finalA, b.finalB) {
		return true
	}

	// Only one top scorer (dead-heats aren't modelled).
	if a.topScorer != "" && b.topScorer != "" && !teamsMatch(a.topScorer, b.topScorer) {
		return true
	}

	// Bets on the same fixture clash if they need incompatible results.
	for _, fa := range a.fixtures {
		for _, fb := range b.fixtures {
			if fa.key == fb.key && fixturesConflict(fa, fb) {
				return true
			}
		}
	}

	return false
}

func fixturesConflict(a, b fixtureClaim) bool {
	if a.hasExact && b.hasExact {
		return a.scoreFirst != b.scoreFirst || a.scoreSecond != b.scoreSecond
	}
	return a.firstResult != b.firstResult
}

func teamIn(team, a, b string) bool {
	return teamsMatch(team, a) || teamsMatch(team, b)
}

func samePair(a1, a2, b1, b2 string) bool {
	return (teamsMatch(a1, b1) && teamsMatch(a2, b2)) || (teamsMatch(a1, b2) && teamsMatch(a2, b1))
}

// fixtureKey canonicalises a team pair so the same fixture always maps to one
// key regardless of which side is listed first, and reports whether teamA is
// the canonical-first (alphabetically smaller) team.
func fixtureKey(teamA, teamB string) (key string, aIsFirst bool) {
	la, lb := strings.ToLower(teamA), strings.ToLower(teamB)
	if la <= lb {
		return la + "\x00" + lb, true
	}
	return lb + "\x00" + la, false
}

func exactFixtureClaim(teamA, teamB string, scoreA, scoreB int) fixtureClaim {
	key, aIsFirst := fixtureKey(teamA, teamB)
	sf, ss := scoreA, scoreB
	if !aIsFirst {
		sf, ss = scoreB, scoreA
	}
	return fixtureClaim{key: key, firstResult: resultFromScore(sf, ss), hasExact: true, scoreFirst: sf, scoreSecond: ss}
}

func outcomeFixtureClaim(team, opponent, outcome string) fixtureClaim {
	key, teamIsFirst := fixtureKey(team, opponent)
	r := resultFromOutcome(outcome) // relative to `team`
	if !teamIsFirst {
		r = invertResult(r)
	}
	return fixtureClaim{key: key, firstResult: r}
}

func resultFromScore(forGoals, againstGoals int) int {
	switch {
	case forGoals > againstGoals:
		return resWin
	case forGoals < againstGoals:
		return resLose
	default:
		return resDraw
	}
}

func resultFromOutcome(outcome string) int {
	switch strings.ToLower(outcome) {
	case "win":
		return resWin
	case "lose":
		return resLose
	default:
		return resDraw
	}
}

func invertResult(r int) int {
	switch r {
	case resWin:
		return resLose
	case resLose:
		return resWin
	default:
		return resDraw
	}
}

// maxWeightSelection finds the highest-value set of bets that can all win at
// once. It splits the conflict graph into connected components and solves each
// exactly (won bets are forced in). Returns the total, a per-node "counted"
// flag, and the components for building the conflict breakdown.
func maxWeightSelection(nodes []payoutNode, adj [][]bool) (total float64, chosen []bool, components [][]int) {
	n := len(nodes)
	chosen = make([]bool, n)
	visited := make([]bool, n)

	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		comp := []int{}
		stack := []int{i}
		visited[i] = true
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			comp = append(comp, v)
			for w := 0; w < n; w++ {
				if adj[v][w] && !visited[w] {
					visited[w] = true
					stack = append(stack, w)
				}
			}
		}
		w, sel := bestIndependentSet(comp, nodes, adj)
		total += w
		for idx, c := range sel {
			if c {
				chosen[comp[idx]] = true
			}
		}
		components = append(components, comp)
	}
	return total, chosen, components
}

// bestIndependentSet returns the max-weight subset of a component with no two
// chosen bets in conflict, with all won bets forced in. Components are tiny in
// practice; a greedy fallback guards against a pathological blow-up.
func bestIndependentSet(comp []int, nodes []payoutNode, adj [][]bool) (float64, []bool) {
	k := len(comp)
	var free []int // indices into comp that are still alive (toggleable)
	for ci, gi := range comp {
		if !nodes[gi].won {
			free = append(free, ci)
		}
	}

	if len(free) > 20 {
		return greedyIndependentSet(comp, nodes, adj)
	}

	independent := func(sel []bool) bool {
		for x := 0; x < k; x++ {
			if !sel[x] {
				continue
			}
			for y := x + 1; y < k; y++ {
				if sel[y] && adj[comp[x]][comp[y]] {
					return false
				}
			}
		}
		return true
	}

	best := -1.0
	var bestSel []bool
	for mask := 0; mask < (1 << len(free)); mask++ {
		sel := make([]bool, k)
		for ci, gi := range comp {
			if nodes[gi].won {
				sel[ci] = true
			}
		}
		for b := 0; b < len(free); b++ {
			if mask&(1<<b) != 0 {
				sel[free[b]] = true
			}
		}
		if !independent(sel) {
			continue
		}
		w := 0.0
		for x := 0; x < k; x++ {
			if sel[x] {
				w += nodes[comp[x]].ret
			}
		}
		if w > best {
			best = w
			bestSel = sel
		}
	}
	return best, bestSel
}

// greedyIndependentSet is an approximate fallback for components too large to
// brute force: force in won bets, then add alive bets by descending return
// whenever they don't conflict with an already-chosen bet.
func greedyIndependentSet(comp []int, nodes []payoutNode, adj [][]bool) (float64, []bool) {
	k := len(comp)
	sel := make([]bool, k)
	total := 0.0
	for ci, gi := range comp {
		if nodes[gi].won {
			sel[ci] = true
			total += nodes[gi].ret
		}
	}

	order := make([]int, 0, k)
	for ci, gi := range comp {
		if !nodes[gi].won {
			order = append(order, ci)
		}
	}
	sort.Slice(order, func(a, b int) bool {
		return nodes[comp[order[a]]].ret > nodes[comp[order[b]]].ret
	})

	for _, ci := range order {
		ok := true
		for x := 0; x < k; x++ {
			if sel[x] && adj[comp[ci]][comp[x]] {
				ok = false
				break
			}
		}
		if ok {
			sel[ci] = true
			total += nodes[comp[ci]].ret
		}
	}
	return total, sel
}

func round2(f float64) float64 {
	if f < 0 {
		return -float64(int64(-f*100+0.5)) / 100
	}
	return float64(int64(f*100+0.5)) / 100
}
