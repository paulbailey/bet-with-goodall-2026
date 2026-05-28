package main

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

// fullGroups builds the 12 WC groups A–L with four teams each, named "<L>1".."<L>4".
func fullGroups() []GroupStanding {
	groups := make([]GroupStanding, 0, 12)
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
		g := GroupStanding{Group: letter}
		for n := 1; n <= 4; n++ {
			g.Teams = append(g.Teams, TeamStats{Name: fmt.Sprintf("%s%d", letter, n)})
		}
		groups = append(groups, g)
	}
	return groups
}

// ── Bracket data integrity ────────────────────────────────────────────────────

func TestBracket_Has32SlotsAndEightThirds(t *testing.T) {
	if len(wc2026Bracket) != 32 {
		t.Fatalf("bracket must have 32 slots, got %d", len(wc2026Bracket))
	}
	thirds := 0
	for _, s := range wc2026Bracket {
		if s.pos == posThird {
			thirds++
		}
	}
	if thirds != 8 {
		t.Fatalf("bracket must have 8 third-place slots, got %d", thirds)
	}
}

// Every R32 pairing must be cross-group (FIFA: same-group teams can't meet in
// the R32). For winner/runner-up slots that means different group letters; for
// third slots, the opponent's group must not be in the allowed set.
func TestBracket_NoSameGroupRound32Clash(t *testing.T) {
	for i := 0; i < len(wc2026Bracket); i += 2 {
		a, b := wc2026Bracket[i], wc2026Bracket[i+1]
		switch {
		case a.pos != posThird && b.pos != posThird:
			if a.group == b.group {
				t.Fatalf("slots %d,%d are both group %s", i, i+1, a.group)
			}
		case a.pos == posThird && b.pos != posThird:
			if containsFold(a.allowed, b.group) {
				t.Fatalf("third slot %d allows group %s which faces 1/2%s", i, b.group, b.group)
			}
		case b.pos == posThird && a.pos != posThird:
			if containsFold(b.allowed, a.group) {
				t.Fatalf("third slot %d allows group %s which faces 1/2%s", i+1, a.group, a.group)
			}
		}
	}
}

// ── Knockout engine ───────────────────────────────────────────────────────────

func TestSimulateBracket_FinalistsFromOppositeHalves(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	slots := make([]*simTeam, 32)
	for i := range slots {
		slots[i] = &simTeam{name: fmt.Sprintf("T%02d", i)}
	}
	for trial := 0; trial < 200; trial++ {
		_, finalists := simulateBracket(slots, rng)
		// Identify the slot index of each finalist by name suffix.
		idx := func(t *simTeam) int { var n int; fmt.Sscanf(t.name, "T%02d", &n); return n }
		f0, f1 := idx(finalists[0]), idx(finalists[1])
		if (f0 < 16) == (f1 < 16) {
			t.Fatalf("finalists must come from opposite halves, got slots %d and %d", f0, f1)
		}
	}
}

func TestSimulateBracket_EqualTeamsRoughlyUniform(t *testing.T) {
	slots := make([]*simTeam, 32)
	for i := range slots {
		slots[i] = &simTeam{name: fmt.Sprintf("T%02d", i)}
	}
	rng := rand.New(rand.NewSource(5))
	const trials = 32000
	champs := map[string]int{}
	for i := 0; i < trials; i++ {
		c, _ := simulateBracket(slots, rng)
		champs[c.name]++
	}
	// Equal teams → each of 32 wins ~1/32 ≈ 0.031.
	for _, s := range slots {
		p := float64(champs[s.name]) / trials
		if math.Abs(p-1.0/32) > 0.015 {
			t.Errorf("%s champion prob %.4f, expected ~0.031", s.name, p)
		}
	}
}

func TestSimulateBracket_StrongerTeamWinsMore(t *testing.T) {
	slots := make([]*simTeam, 32)
	for i := range slots {
		slots[i] = &simTeam{name: fmt.Sprintf("T%02d", i)}
	}
	slots[0].rating = 1.5 // clearly the strongest
	rng := rand.New(rand.NewSource(8))
	const trials = 20000
	wins := 0
	for i := 0; i < trials; i++ {
		c, _ := simulateBracket(slots, rng)
		if c.name == "T00" {
			wins++
		}
	}
	if p := float64(wins) / trials; p < 0.15 {
		t.Fatalf("a much stronger team should win far more than 1/32, got %.3f", p)
	}
}

// ── Bracket seeding ───────────────────────────────────────────────────────────

func TestBuildBracketSlots_SeedsAllAndCrossGroup(t *testing.T) {
	ratingFor := strengthLookup(map[string]float64{})
	rng := rand.New(rand.NewSource(1))
	ranked, qualThirds := simulateGroupsOnce(fullGroups(), map[string][2]int{}, ratingFor, thirdsQualifying(12), rng)

	slots, ok := buildBracketSlots(ranked, qualThirds)
	if !ok {
		t.Fatal("bracket should fully seed with 12 complete groups")
	}
	if len(slots) != 32 {
		t.Fatalf("expected 32 seeded slots, got %d", len(slots))
	}
	seen := map[string]bool{}
	for i, s := range slots {
		if s == nil {
			t.Fatalf("slot %d unseeded", i)
		}
		if seen[s.name] {
			t.Fatalf("team %s seeded into two slots", s.name)
		}
		seen[s.name] = true
	}
	// No R32 match between two teams from the same group.
	for i := 0; i < 32; i += 2 {
		if slots[i].group == slots[i+1].group {
			t.Fatalf("R32 match %d pairs two group-%s teams (%s, %s)", i/2, slots[i].group, slots[i].name, slots[i+1].name)
		}
	}
}

// ── Full tournament ───────────────────────────────────────────────────────────

func TestSimulateTournament_ProbabilitiesConsistent(t *testing.T) {
	res := simulateTournament(fullGroups(), nil, map[string]float64{}, 8000, 2024)

	var sumWin, sumFinal, sumChamp float64
	for _, r := range res {
		for _, p := range []float64{r.PWinGroup, r.PQualify, r.PReachFinal, r.PWinTournament} {
			if p < 0 || p > 1 {
				t.Fatalf("probability out of range: %+v", r)
			}
		}
		if r.PQualify < r.PWinGroup || r.PReachFinal > r.PQualify+1e-9 || r.PWinTournament > r.PReachFinal+1e-9 {
			t.Fatalf("monotonicity violated (win⊆qualify, champion⊆final⊆qualify): %+v", r)
		}
		sumWin += r.PWinGroup
		sumFinal += r.PReachFinal
		sumChamp += r.PWinTournament
	}
	// 12 group winners, 2 finalists, 1 champion across all teams.
	if math.Abs(sumWin-12) > 0.2 {
		t.Errorf("group-winner probabilities should sum to ~12, got %.3f", sumWin)
	}
	if math.Abs(sumFinal-2) > 0.1 {
		t.Errorf("reach-final probabilities should sum to ~2, got %.3f", sumFinal)
	}
	if math.Abs(sumChamp-1) > 0.05 {
		t.Errorf("champion probabilities should sum to ~1, got %.3f", sumChamp)
	}
}

func TestSimulateTournament_StrongTeamFavoured(t *testing.T) {
	strength := map[string]float64{"a1": 1.8}
	res := simulateTournament(fullGroups(), nil, strength, 8000, 77)
	if res["A1"].PWinTournament < 0.1 {
		t.Fatalf("a dominant team should be clear favourite, got %.3f", res["A1"].PWinTournament)
	}
	if res["A1"].PWinGroup < 0.8 {
		t.Fatalf("a dominant team should usually win its group, got %.3f", res["A1"].PWinGroup)
	}
}

func TestRunTournament_FinalistJointWithinMarginals(t *testing.T) {
	pairs := [][2]string{{"A1", "B1"}}
	res, joints := runTournament(fullGroups(), nil, map[string]float64{}, 16000, 314, pairs)

	key, _ := fixtureKey("A1", "B1")
	jp, ok := joints[key]
	if !ok {
		t.Fatalf("joint probability for requested pair not returned")
	}
	if jp < 0 || jp > 1 {
		t.Fatalf("joint probability out of range: %v", jp)
	}
	// A joint event can't be likelier than either team reaching the final.
	mA, mB := res["A1"].PReachFinal, res["B1"].PReachFinal
	if jp > mA+1e-9 || jp > mB+1e-9 {
		t.Fatalf("joint (%.4f) exceeds a marginal (A1 %.4f, B1 %.4f)", jp, mA, mB)
	}
	// Two cross-group equal teams do sometimes both reach the final.
	if jp <= 0 {
		t.Fatalf("expected a positive joint finalist probability, got %.4f", jp)
	}
}

func TestRunTournament_StrongPairHigherJoint(t *testing.T) {
	// Two strong teams from opposite-looking groups should reach the final
	// together more often than two equal teams would.
	strength := map[string]float64{"a1": 1.2, "g1": 1.2}
	pairs := [][2]string{{"A1", "G1"}}
	_, joints := runTournament(fullGroups(), nil, strength, 16000, 99, pairs)
	key, _ := fixtureKey("A1", "G1")
	if joints[key] < 0.03 {
		t.Fatalf("two strong cross-group teams should pair in the final reasonably often, got %.4f", joints[key])
	}
}

func TestRunTournament_NoPairsNoJoints(t *testing.T) {
	_, joints := runTournament(fullGroups(), nil, map[string]float64{}, 2000, 1, nil)
	if len(joints) != 0 {
		t.Fatalf("no requested pairs should yield no joint entries, got %d", len(joints))
	}
}
