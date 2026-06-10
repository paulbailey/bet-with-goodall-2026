package main

import (
	"testing"
	"time"
)

func TestBuildMatchWindow(t *testing.T) {
	now := time.Date(2026, 6, 15, 13, 30, 0, 0, time.UTC)
	one := func(n int) *int { return &n }
	mk := func(date time.Time, home, away, status string) Match {
		return Match{
			UtcDate:  date,
			Status:   status,
			Stage:    "GROUP_STAGE",
			Group:    "GROUP_A",
			HomeTeam: home,
			AwayTeam: away,
		}
	}

	matches := []Match{
		// Out of window: two days ago and two days ahead.
		mk(time.Date(2026, 6, 13, 18, 0, 0, 0, time.UTC), "Brazil", "Ghana", "FINISHED"),
		mk(time.Date(2026, 6, 17, 18, 0, 0, 0, time.UTC), "Japan", "Chile", "TIMED"),
		// In window, deliberately unsorted.
		mk(time.Date(2026, 6, 16, 2, 0, 0, 0, time.UTC), "Mexico", "Norway", "TIMED"),
		mk(time.Date(2026, 6, 14, 23, 59, 0, 0, time.UTC), "France", "Senegal", "FINISHED"),
		mk(time.Date(2026, 6, 15, 15, 0, 0, 0, time.UTC), "England", "Croatia", "IN_PLAY"),
	}
	matches[4].HomeScore, matches[4].AwayScore = one(2), one(1)

	got := buildMatchWindow(matches, now)

	wantHomes := []string{"France", "England", "Mexico"}
	if len(got) != len(wantHomes) {
		t.Fatalf("got %d matches, want %d: %+v", len(got), len(wantHomes), got)
	}
	for i, home := range wantHomes {
		if got[i].Home != home {
			t.Errorf("match %d: home = %q, want %q", i, got[i].Home, home)
		}
	}

	live := got[1]
	if live.ID != "2026-06-15-england-v-croatia" {
		t.Errorf("id = %q, want %q", live.ID, "2026-06-15-england-v-croatia")
	}
	if live.UtcDate != "2026-06-15T15:00:00Z" {
		t.Errorf("utc_date = %q, want %q", live.UtcDate, "2026-06-15T15:00:00Z")
	}
	if live.Group != "Group A" {
		t.Errorf("group = %q, want %q", live.Group, "Group A")
	}
	if live.HomeScore == nil || *live.HomeScore != 2 || live.AwayScore == nil || *live.AwayScore != 1 {
		t.Errorf("score = %v–%v, want 2–1", live.HomeScore, live.AwayScore)
	}
	if got[0].HomeScore != nil {
		t.Errorf("France match: home_score = %v, want nil (provider gave none)", got[0].HomeScore)
	}
}
