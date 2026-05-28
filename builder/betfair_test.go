package main

import (
	"math"
	"testing"
)

func TestParseScoreRunner(t *testing.T) {
	cases := []struct {
		name string
		h, a int
		ok   bool
	}{
		{"3 - 1", 3, 1, true},
		{"0 - 0", 0, 0, true},
		{"10 - 2", 10, 2, true},
		{"Any Other Home Win", 0, 0, false},
		{"Any Unquoted", 0, 0, false},
		{"3-1", 0, 0, false}, // Betfair uses " - " with spaces
	}
	for _, c := range cases {
		h, a, ok := parseScoreRunner(c.name)
		if ok != c.ok || (ok && (h != c.h || a != c.a)) {
			t.Errorf("parseScoreRunner(%q) = (%d,%d,%v), want (%d,%d,%v)", c.name, h, a, ok, c.h, c.a, c.ok)
		}
	}
}

func TestSplitFixture(t *testing.T) {
	h, a := splitFixture("Mexico v South Africa")
	if h != "Mexico" || a != "South Africa" {
		t.Fatalf("splitFixture: got (%q,%q)", h, a)
	}
	if h, a := splitFixture("Weird Name"); h != "Weird Name" || a != "" {
		t.Fatalf("unparseable fixture should return whole name as home, got (%q,%q)", h, a)
	}
}

func TestIsDrawRunner(t *testing.T) {
	for _, n := range []string{"The Draw", "Draw", "the draw"} {
		if !isDrawRunner(n) {
			t.Errorf("%q should be a draw runner", n)
		}
	}
	if isDrawRunner("Mexico") {
		t.Error("team name should not be a draw runner")
	}
}

func TestBetfairMarketType(t *testing.T) {
	cases := map[OutrightMarket]string{
		MarketWinner:        "WINNER",
		MarketToReachFinal:  "TO_REACH_FINAL",
		MarketTopGoalscorer: "TOP_GOALSCORER",
	}
	for m, want := range cases {
		if got, ok := betfairMarketType(m); !ok || got != want {
			t.Errorf("betfairMarketType(%q) = (%q,%v), want %q", m, got, ok, want)
		}
	}
	if _, ok := betfairMarketType(OutrightMarket("bogus")); ok {
		t.Error("unknown market should not map")
	}
}

func mkRunner(id int64, price float64) bookRunner {
	r := bookRunner{SelectionID: id, Status: "ACTIVE"}
	r.Ex.AvailableToBack = []struct {
		Price float64 `json:"price"`
		Size  float64 `json:"size"`
	}{{Price: price}}
	return r
}

func TestDevigBook(t *testing.T) {
	// Implied 1/2 + 1/4 + 1/4 = 1.0 already fair.
	book := bookMarket{Runners: []bookRunner{mkRunner(1, 2), mkRunner(2, 4), mkRunner(3, 4)}}
	got := devigBook(book)
	want := map[int64]float64{1: 0.5, 2: 0.25, 3: 0.25}
	var sum float64
	for id, w := range want {
		if math.Abs(got[id]-w) > 1e-9 {
			t.Errorf("selection %d: got %v want %v", id, got[id], w)
		}
		sum += got[id]
	}
	if math.Abs(sum-1) > 1e-9 {
		t.Fatalf("probabilities should sum to 1, got %v", sum)
	}
}

func TestDevigBook_SkipsRemovedRunners(t *testing.T) {
	removed := mkRunner(3, 0)
	removed.Status = "REMOVED"
	book := bookMarket{Runners: []bookRunner{mkRunner(1, 2), mkRunner(2, 2), removed}}
	got := devigBook(book)
	if _, ok := got[3]; ok {
		t.Error("removed runner should be excluded")
	}
	if math.Abs(got[1]-0.5) > 1e-9 || math.Abs(got[2]-0.5) > 1e-9 {
		t.Fatalf("two equal active runners should each be 0.5, got %v", got)
	}
}

func TestBestBack_FallsBackToLastTraded(t *testing.T) {
	r := bookRunner{SelectionID: 1, Status: "ACTIVE", LastPriceTraded: 3.5}
	if got := r.bestBack(); got != 3.5 {
		t.Fatalf("expected fallback to last traded 3.5, got %v", got)
	}
}

func TestOutrightProbs_MapsRunnerNames(t *testing.T) {
	cat := catalogMarket{MarketID: "1.1"}
	cat.Runners = []struct {
		SelectionID int64  `json:"selectionId"`
		RunnerName  string `json:"runnerName"`
	}{{SelectionID: 22, RunnerName: "Spain"}, {SelectionID: 24, RunnerName: "France"}}
	book := bookMarket{MarketID: "1.1", Runners: []bookRunner{mkRunner(22, 2), mkRunner(24, 2)}}
	got := outrightProbs(cat, book)
	if len(got) != 2 {
		t.Fatalf("expected 2 selections, got %d", len(got))
	}
	byName := map[string]float64{}
	for _, p := range got {
		byName[p.Selection] = p.Prob
	}
	if math.Abs(byName["Spain"]-0.5) > 1e-9 || math.Abs(byName["France"]-0.5) > 1e-9 {
		t.Fatalf("expected 0.5 each, got %v", byName)
	}
}
