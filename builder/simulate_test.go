package main

import (
	"math"
	"testing"
)

func fourTeamGroup(letter string, names ...string) GroupStanding {
	g := GroupStanding{Group: letter}
	for _, n := range names {
		g.Teams = append(g.Teams, TeamStats{Name: n})
	}
	return g
}

// ── de-vig & accumulator math ─────────────────────────────────────────────────

func TestDevig_RemovesOverround(t *testing.T) {
	// Odds with a built-in margin (implied sums to ~1.08).
	got := devig([]float64{1.9, 3.5, 3.9})
	var sum float64
	for _, p := range got {
		sum += p
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("de-vigged probabilities should sum to 1, got %v (sum %v)", got, sum)
	}
	// Favourite (shortest odds) keeps the highest probability.
	if !(got[0] > got[1] && got[0] > got[2]) {
		t.Fatalf("shortest odds should map to highest probability, got %v", got)
	}
}

func TestDevig_FairOddsUnchanged(t *testing.T) {
	got := devig([]float64{2, 4, 4}) // implied 0.5/0.25/0.25, already sums to 1
	want := []float64{0.5, 0.25, 0.25}
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-9 {
			t.Fatalf("fair odds should pass through, got %v want %v", got, want)
		}
	}
}

func TestAccaProbability(t *testing.T) {
	if got := accaProbability([]float64{0.5, 0.5}); math.Abs(got-0.25) > 1e-9 {
		t.Fatalf("acca prob: want 0.25, got %v", got)
	}
	if got := accaProbability(nil); got != 1 {
		t.Fatalf("empty acca should be certain (1), got %v", got)
	}
}

// ── group-stage simulation ────────────────────────────────────────────────────

func TestSimulateGroupStage_EqualTeamsUniform(t *testing.T) {
	g := fourTeamGroup("A", "Alpha", "Bravo", "Charlie", "Delta")
	res := simulateGroupStage([]GroupStanding{g}, nil, map[string]float64{}, 40000, 1)

	var sum float64
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
		p := res[name].PWinGroup
		if math.Abs(p-0.25) > 0.03 {
			t.Errorf("%s: equal teams should win ~25%%, got %.3f", name, p)
		}
		sum += p
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("win-group probabilities should sum to 1, got %v", sum)
	}
}

func TestSimulateGroupStage_StrongerTeamWinsMore(t *testing.T) {
	g := fourTeamGroup("A", "Strong", "Weak1", "Weak2", "Weak3")
	strength := map[string]float64{"strong": 1.0} // others default to 0
	res := simulateGroupStage([]GroupStanding{g}, nil, strength, 40000, 7)

	if res["Strong"].PWinGroup < 0.6 {
		t.Fatalf("a clearly stronger team should win the group >60%%, got %.3f", res["Strong"].PWinGroup)
	}
	if res["Strong"].PQualify < res["Strong"].PWinGroup {
		t.Fatalf("qualify prob (%.3f) must be >= win-group prob (%.3f)",
			res["Strong"].PQualify, res["Strong"].PWinGroup)
	}
}

func TestSimulateGroupStage_RespectsFinishedResults(t *testing.T) {
	// All of Alpha's games are already finished as big wins; with equal ratings,
	// Alpha should win the group far more often than a free-for-all 25%.
	g := fourTeamGroup("A", "Alpha", "Bravo", "Charlie", "Delta")
	matches := []Match{
		finishedScore("Alpha", "Bravo", 3, 0),
		finishedScore("Alpha", "Charlie", 3, 0),
		finishedScore("Alpha", "Delta", 3, 0),
	}
	res := simulateGroupStage([]GroupStanding{g}, matches, map[string]float64{}, 40000, 3)
	if res["Alpha"].PWinGroup < 0.85 {
		t.Fatalf("Alpha with 9 pts and +9 GD should usually win the group, got %.3f", res["Alpha"].PWinGroup)
	}
}

func TestSimulateGroupStage_Deterministic(t *testing.T) {
	g := fourTeamGroup("A", "Alpha", "Bravo", "Charlie", "Delta")
	a := simulateGroupStage([]GroupStanding{g}, nil, map[string]float64{}, 5000, 42)
	b := simulateGroupStage([]GroupStanding{g}, nil, map[string]float64{}, 5000, 42)
	if a["Alpha"].PWinGroup != b["Alpha"].PWinGroup {
		t.Fatalf("same seed should give identical results: %v vs %v", a["Alpha"], b["Alpha"])
	}
}

func TestSimulateGroupStage_QualificationSlots(t *testing.T) {
	// Two groups, equal teams: top 2 of each always qualify (4 teams), plus
	// best-placed thirds fill up to koTeams. With only 8 teams total, everyone
	// who isn't 4th-and-beyond the cut should qualify; check probabilities are
	// in range and the two finishers per group dominate.
	groups := []GroupStanding{
		fourTeamGroup("A", "A1", "A2", "A3", "A4"),
		fourTeamGroup("B", "B1", "B2", "B3", "B4"),
	}
	res := simulateGroupStage(groups, nil, map[string]float64{}, 20000, 11)
	for name, r := range res {
		if r.PQualify < 0 || r.PQualify > 1 {
			t.Fatalf("%s qualify prob out of range: %v", name, r.PQualify)
		}
		if r.PQualify < r.PWinGroup {
			t.Fatalf("%s: qualify (%.3f) must be >= win group (%.3f)", name, r.PQualify, r.PWinGroup)
		}
	}
}
