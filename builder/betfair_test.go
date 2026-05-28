package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCertLoginWith_ParsesSessionToken(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Application"); got != "appkey" {
			t.Errorf("X-Application header = %q, want appkey", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if u, p := r.PostForm.Get("username"), r.PostForm.Get("password"); u != "user" || p != "pass" {
			t.Errorf("credentials = (%q,%q), want (user,pass)", u, p)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"sessionToken":"TOKEN123","loginStatus":"SUCCESS"}`)
	}))
	defer srv.Close()

	c := newBetfairClient("appkey", discardLogger())
	c.certLoginURL = srv.URL
	if err := c.certLoginWith(context.Background(), srv.Client(), "user", "pass"); err != nil {
		t.Fatalf("certLoginWith: %v", err)
	}
	if c.token != "TOKEN123" {
		t.Fatalf("token = %q, want TOKEN123", c.token)
	}
}

func TestCertLoginWith_NonSuccessIsError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"loginStatus":"INVALID_USERNAME_OR_PASSWORD"}`)
	}))
	defer srv.Close()

	c := newBetfairClient("appkey", discardLogger())
	c.certLoginURL = srv.URL
	if err := c.certLoginWith(context.Background(), srv.Client(), "user", "bad"); err == nil {
		t.Fatal("expected error on non-SUCCESS loginStatus")
	}
	if c.token != "" {
		t.Fatalf("token should stay empty on failure, got %q", c.token)
	}
}

func TestCertLogin_MissingKeypairError(t *testing.T) {
	c := newBetfairClient("appkey", discardLogger())
	err := c.CertLogin(context.Background(), "user", "pass", "/no/such.crt", "/no/such.key")
	if err == nil {
		t.Fatal("expected error when the keypair files are missing")
	}
}

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
