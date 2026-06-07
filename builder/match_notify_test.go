package main

import (
	"strings"
	"testing"
	"time"
)

func TestNewlyFinished(t *testing.T) {
	matches := []Match{
		{HomeTeam: "Spain", AwayTeam: "Japan", Status: "FINISHED", HomeScore: ptr(2), AwayScore: ptr(0)},
		{HomeTeam: "Brazil", AwayTeam: "Serbia", Status: "IN_PLAY"},
		{HomeTeam: "France", AwayTeam: "Peru", Status: "FINISHED", HomeScore: ptr(1), AwayScore: ptr(0)},
	}

	// No baseline (first cycle / restart): nothing fires.
	if got := newlyFinished(nil, matches); got != nil {
		t.Fatalf("expected no notifications without a baseline, got %d", len(got))
	}

	// Spain was in play last cycle and is now finished → it fires. France was
	// already finished last cycle → it does not.
	prev := map[string]string{
		matchKey(matches[0]): "IN_PLAY",
		matchKey(matches[1]): "IN_PLAY",
		matchKey(matches[2]): "FINISHED",
	}
	got := newlyFinished(prev, matches)
	if len(got) != 1 || got[0].HomeTeam != "Spain" {
		t.Fatalf("expected only Spain to be newly finished, got %+v", got)
	}
}

func TestMatchKeyStableAcrossCycles(t *testing.T) {
	d := time.Date(2026, 6, 20, 19, 0, 0, 0, time.UTC)
	a := Match{UtcDate: d, HomeTeam: "Spain", AwayTeam: "Japan", Status: "IN_PLAY"}
	b := Match{UtcDate: d, HomeTeam: "Spain", AwayTeam: "Japan", Status: "FINISHED", HomeScore: ptr(2), AwayScore: ptr(0)}
	if matchKey(a) != matchKey(b) {
		t.Fatalf("matchKey changed across status update: %q vs %q", matchKey(a), matchKey(b))
	}
}

func TestMatchScoreLine(t *testing.T) {
	m := Match{HomeTeam: "Spain", AwayTeam: "Japan", HomeScore: ptr(2), AwayScore: ptr(0)}
	if got, want := matchScoreLine(m), "Spain 2–0 Japan"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// No score yet → fall back to the fixture name.
	if got, want := matchScoreLine(Match{HomeTeam: "Spain", AwayTeam: "Japan"}), "Spain vs Japan"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMatchTemplateBody(t *testing.T) {
	// Settled bets lead, then the biggest swing.
	settled := []Settled{
		{Label: "Spain treble", Category: "Match acca", Status: "won"},
		{Label: "Japan to win Group E", Category: "Group acca", Status: "lost"},
	}
	risers := []Mover{{Label: "Spain to win the tournament", PrevProb: 0.10, NewProb: 0.15, Ratio: 1.5}}
	fallers := []Mover{{Label: "Japan finalists", PrevProb: 0.04, NewProb: 0.01, Ratio: 0.25}}

	body := matchTemplateBody(risers, fallers, settled)
	if !strings.Contains(body, "Landed: Spain treble.") {
		t.Errorf("missing win: %q", body)
	}
	if !strings.Contains(body, "Bust: Japan to win Group E.") {
		t.Errorf("missing bust: %q", body)
	}
	// Japan finalists swung furthest in relative terms (ratio 0.25 vs 1.5).
	if !strings.Contains(body, "Japan finalists down") {
		t.Errorf("expected biggest mover to be Japan finalists: %q", body)
	}

	// Nothing moved or settled → a graceful default.
	if got := matchTemplateBody(nil, nil, nil); got != "No change to the group's bets." {
		t.Errorf("unexpected empty-state body: %q", got)
	}
}

func TestBiggestMover(t *testing.T) {
	risers := []Mover{{Label: "up a bit", Ratio: 1.2}}
	fallers := []Mover{{Label: "crashed", Ratio: 0.1}}
	if got := biggestMover(risers, fallers); got == nil || got.Label != "crashed" {
		t.Fatalf("expected the bigger relative swing (crashed), got %+v", got)
	}
	if got := biggestMover(nil, nil); got != nil {
		t.Fatalf("expected nil with no movers, got %+v", got)
	}
}
