package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"
)

func fptr(f float64) *float64 { return &f }

// memBlob is an in-memory blobStore for exercising the generator end-to-end.
type memBlob struct{ m map[string][]byte }

func newMemBlob() *memBlob { return &memBlob{m: map[string][]byte{}} }

func (b *memBlob) Get(_ context.Context, key string) ([]byte, error) { return b.m[key], nil }
func (b *memBlob) Put(_ context.Context, key string, data []byte) error {
	b.m[key] = append([]byte(nil), data...)
	return nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func finishedMatch(date string) Match {
	d, _ := time.Parse(time.RFC3339, date)
	return Match{UtcDate: d, Status: "FINISHED"}
}

func TestGeneratorBaselineThenSummary(t *testing.T) {
	store := newMemBlob()
	gen := &summaryGenerator{
		store:     store,
		anth:      nil, // templated narration
		loc:       time.UTC,
		publicKey: "data/daily-summary.json",
		stateKey:  "data/summary-state.json",
		logger:    quietLogger(),
	}
	ctx := context.Background()

	// Cycle 1: pre-tournament, nothing finished. Should record a baseline but
	// write no public summary.
	pre := StateJSON{Bets: []BetJSON{{ID: "acca1", Status: "alive", Probability: fptr(0.02)}}}
	gen.maybeGenerate(ctx, pre, []Match{{UtcDate: time.Now(), Status: "SCHEDULED"}})
	if _, ok := store.m["data/summary-state.json"]; !ok {
		t.Fatal("baseline state should have been written")
	}
	if _, ok := store.m["data/daily-summary.json"]; ok {
		t.Fatal("no public summary should exist before a day completes")
	}

	// Cycle 2: day complete, probability has climbed. Expect one summary.
	day := StateJSON{Bets: []BetJSON{{ID: "acca1", Status: "alive", Probability: fptr(0.06)}}}
	gen.maybeGenerate(ctx, day, []Match{finishedMatch("2026-06-14T18:00:00Z")})

	var file DailySummaryFile
	if err := json.Unmarshal(store.m["data/daily-summary.json"], &file); err != nil {
		t.Fatalf("unmarshal public file: %v", err)
	}
	if len(file.Summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(file.Summaries))
	}
	s := file.Summaries[0]
	if s.Date != "2026-06-14" {
		t.Errorf("summary date = %q, want 2026-06-14", s.Date)
	}
	if len(s.Risers) != 1 || s.Risers[0].ID != "acca1" {
		t.Errorf("expected acca1 as a riser, got %+v", s.Risers)
	}
	if s.Paragraph == "" {
		t.Error("expected a templated paragraph")
	}

	// Cycle 3: same completed day, re-run. Must not append a duplicate.
	gen.maybeGenerate(ctx, day, []Match{finishedMatch("2026-06-14T18:00:00Z")})
	if err := json.Unmarshal(store.m["data/daily-summary.json"], &file); err != nil {
		t.Fatalf("unmarshal public file: %v", err)
	}
	if len(file.Summaries) != 1 {
		t.Fatalf("re-run should be idempotent, got %d summaries", len(file.Summaries))
	}
}

func TestLatestCompletedDay(t *testing.T) {
	utc := time.UTC
	mk := func(date, status string) Match {
		t.Helper()
		d, err := time.Parse(time.RFC3339, date)
		if err != nil {
			t.Fatalf("bad date %q: %v", date, err)
		}
		return Match{UtcDate: d, Status: status}
	}

	tests := []struct {
		name    string
		matches []Match
		want    string
	}{
		{
			name: "all of a day finished",
			matches: []Match{
				mk("2026-06-14T16:00:00Z", "FINISHED"),
				mk("2026-06-14T19:00:00Z", "FINISHED"),
				mk("2026-06-15T16:00:00Z", "SCHEDULED"),
			},
			want: "2026-06-14",
		},
		{
			name: "latest fully-finished day wins, partial day ignored",
			matches: []Match{
				mk("2026-06-13T16:00:00Z", "FINISHED"),
				mk("2026-06-14T16:00:00Z", "FINISHED"),
				mk("2026-06-14T19:00:00Z", "IN_PLAY"),
			},
			want: "2026-06-13",
		},
		{
			name: "no day complete",
			matches: []Match{
				mk("2026-06-14T16:00:00Z", "SCHEDULED"),
			},
			want: "",
		},
		{
			name:    "no matches",
			matches: nil,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := latestCompletedDay(tt.matches, utc); got != tt.want {
				t.Errorf("latestCompletedDay() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLatestCompletedDayTimezone(t *testing.T) {
	// A 01:00 UTC kickoff is still the previous evening in New York. Both
	// fixtures should group under 2026-06-14 local, and the day is complete.
	ny, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("tzdata unavailable")
	}
	mk := func(date string) Match {
		d, _ := time.Parse(time.RFC3339, date)
		return Match{UtcDate: d, Status: "FINISHED"}
	}
	matches := []Match{
		mk("2026-06-14T23:00:00Z"), // 19:00 ET, 14th
		mk("2026-06-15T01:00:00Z"), // 21:00 ET, still the 14th
	}
	if got := latestCompletedDay(matches, ny); got != "2026-06-14" {
		t.Errorf("latestCompletedDay() = %q, want 2026-06-14", got)
	}
}

func TestComputeMoversRankingAndSettlement(t *testing.T) {
	prev := map[string]betSnap{
		"Group acca:longshot":      {Prob: fptr(0.01), Status: "alive"}, // doubles in relative terms
		"Group acca:steady":        {Prob: fptr(0.40), Status: "alive"}, // small absolute rise
		"Tournament winner:brazil": {Prob: fptr(0.20), Status: "alive"}, // drifts down
		"Match acca:england":       {Prob: fptr(0.05), Status: "alive"}, // goes bust
		"Exact score:flat":         {Prob: fptr(0.03), Status: "alive"}, // unchanged → excluded
		"Finalists:france-spain":   {Prob: fptr(0.02), Status: "alive"}, // lands
	}
	infos := []betInfo{
		{Key: "Group acca:longshot", ID: "longshot", Label: "A: Wales", Category: "Group acca", Status: "alive", Prob: fptr(0.02)},
		{Key: "Group acca:steady", ID: "steady", Label: "Favourites", Category: "Group acca", Status: "alive", Prob: fptr(0.44)},
		{Key: "Tournament winner:brazil", ID: "brazil", Label: "Brazil to win the tournament", Category: "Tournament winner", Status: "alive", Prob: fptr(0.14)},
		{Key: "Match acca:england", ID: "england", Label: "England match acca (3 legs)", Category: "Match acca", Status: "lost", Prob: fptr(0.0)},
		{Key: "Exact score:flat", ID: "flat", Label: "X 1-0 Y", Category: "Exact score", Status: "alive", Prob: fptr(0.03)},
		{Key: "Finalists:france-spain", ID: "france-spain", Label: "France & Spain to reach the final", Category: "Finalists", Status: "won", Prob: fptr(1.0)},
	}

	risers, fallers, settled := computeMovers(prev, infos)

	// The bust bet leads the fallers (ratio 0); the long-shot leads the risers
	// on relative change even though its absolute move is tiny.
	if len(risers) == 0 || risers[0].ID != "france-spain" {
		// france-spain went 0.02 -> 1.0, a 50x rise, the biggest relative gain.
		t.Fatalf("expected france-spain as top riser, got %+v", risers)
	}
	if len(fallers) == 0 || fallers[0].ID != "england" {
		t.Fatalf("expected england (bust) as top faller, got %+v", fallers)
	}

	// The unchanged exact-score bet must not appear in either list.
	for _, m := range append(append([]Mover{}, risers...), fallers...) {
		if m.ID == "flat" {
			t.Errorf("unchanged bet should be excluded, got %+v", m)
		}
	}

	// Both decided bets were alive at the previous close → settled.
	wantSettled := map[string]string{"england": "lost", "france-spain": "won"}
	if len(settled) != len(wantSettled) {
		t.Fatalf("expected %d settled, got %d: %+v", len(wantSettled), len(settled), settled)
	}
	for _, s := range settled {
		if wantSettled[s.ID] != s.Status {
			t.Errorf("settled %s: got %q want %q", s.ID, s.Status, wantSettled[s.ID])
		}
	}
}

func TestComputeMoversCapsAtFive(t *testing.T) {
	prev := map[string]betSnap{}
	var infos []betInfo
	for i := 0; i < 8; i++ {
		k := "Group acca:r" + string(rune('a'+i))
		prev[k] = betSnap{Prob: fptr(0.10), Status: "alive"}
		infos = append(infos, betInfo{Key: k, ID: k, Label: k, Category: "Group acca", Status: "alive", Prob: fptr(0.30)})
	}
	risers, _, _ := computeMovers(prev, infos)
	if len(risers) != maxMovers {
		t.Errorf("expected risers capped at %d, got %d", maxMovers, len(risers))
	}
}

func TestGroupAccaLabel(t *testing.T) {
	fav := map[string]string{"A": "Brazil", "B": "France"}
	favourites := BetJSON{Legs: []LegJSON{{Group: "A", Team: "Brazil"}, {Group: "B", Team: "France"}}}
	if got := groupAccaLabel(favourites, fav); got != "Favourites" {
		t.Errorf("all-favourites label = %q, want Favourites", got)
	}
	deviating := BetJSON{Legs: []LegJSON{{Group: "A", Team: "Brazil"}, {Group: "B", Team: "Spain"}}}
	if got := groupAccaLabel(deviating, fav); got != "B: Spain" {
		t.Errorf("deviating label = %q, want \"B: Spain\"", got)
	}
}
