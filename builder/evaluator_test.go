package main

import "testing"

func ptr(n int) *int { return &n }

func finishedScore(home, away string, hs, as int) Match {
	return Match{
		Stage: "GROUP_STAGE", Status: "FINISHED",
		HomeTeam: home, AwayTeam: away,
		HomeScore: ptr(hs), AwayScore: ptr(as),
	}
}

func tournamentTestGroups() []GroupStanding {
	return []GroupStanding{
		{
			Group: "A",
			Teams: []TeamStats{
				{Name: "Mexico", Played: 3, Points: 7},
				{Name: "South Korea", Played: 3, Points: 6},
				{Name: "Czechia", Played: 3, Points: 3},
				{Name: "South Africa", Played: 3, Points: 1},
			},
		},
		{
			Group: "B",
			Teams: []TeamStats{
				{Name: "Switzerland", Played: 2, Points: 4},
				{Name: "Canada", Played: 2, Points: 3},
				{Name: "Qatar", Played: 2, Points: 3},
				{Name: "Bosnia-Herzegovina", Played: 2, Points: 1},
			},
		},
	}
}

func TestEvaluateTournamentWinnerBet_Alive(t *testing.T) {
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), nil)
	if got != "alive" {
		t.Fatalf("Mexico should be alive (group winner so far), got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_LostInGroup(t *testing.T) {
	got := evaluateTournamentWinnerBet("South Africa", tournamentTestGroups(), nil)
	if got != "lost" {
		t.Fatalf("South Africa finished 4th in group, expected lost, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_LostInKnockout(t *testing.T) {
	matches := []Match{
		{Stage: "LAST_16", Status: "FINISHED", HomeTeam: "Mexico", AwayTeam: "Brazil", Winner: "AWAY_TEAM"},
	}
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), matches)
	if got != "lost" {
		t.Fatalf("Mexico lost their R16 match, expected lost, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_AliveAfterKnockoutWin(t *testing.T) {
	matches := []Match{
		{Stage: "LAST_16", Status: "FINISHED", HomeTeam: "Mexico", AwayTeam: "Brazil", Winner: "HOME_TEAM"},
	}
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), matches)
	if got != "alive" {
		t.Fatalf("Mexico won their R16 match, expected alive, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_WonFinal(t *testing.T) {
	matches := []Match{
		{Stage: "FINAL", Status: "FINISHED", HomeTeam: "Mexico", AwayTeam: "Brazil", Winner: "HOME_TEAM"},
	}
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), matches)
	if got != "won" {
		t.Fatalf("Mexico won the final, expected won, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_LostFinal(t *testing.T) {
	matches := []Match{
		{Stage: "FINAL", Status: "FINISHED", HomeTeam: "Mexico", AwayTeam: "Brazil", Winner: "AWAY_TEAM"},
	}
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), matches)
	if got != "lost" {
		t.Fatalf("Mexico lost the final, expected lost, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_IgnoresUnfinishedMatches(t *testing.T) {
	matches := []Match{
		{Stage: "LAST_16", Status: "SCHEDULED", HomeTeam: "Mexico", AwayTeam: "Brazil"},
	}
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), matches)
	if got != "alive" {
		t.Fatalf("Scheduled match shouldn't affect status, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_IgnoresGroupStageMatches(t *testing.T) {
	// A group-stage loss alone shouldn't knock the team out — only the final
	// group standings determine that. Mexico won their group above, so a single
	// group-stage loss shouldn't matter.
	matches := []Match{
		{Stage: "GROUP_STAGE", Status: "FINISHED", HomeTeam: "Mexico", AwayTeam: "South Korea", Winner: "AWAY_TEAM"},
	}
	got := evaluateTournamentWinnerBet("Mexico", tournamentTestGroups(), matches)
	if got != "alive" {
		t.Fatalf("Group-stage loss alone shouldn't change status, got %s", got)
	}
}

func TestEvaluateTournamentWinnerBet_CaseInsensitiveTeamMatch(t *testing.T) {
	matches := []Match{
		{Stage: "FINAL", Status: "FINISHED", HomeTeam: "MEXICO", AwayTeam: "Brazil", Winner: "HOME_TEAM"},
	}
	got := evaluateTournamentWinnerBet("mexico", tournamentTestGroups(), matches)
	if got != "won" {
		t.Fatalf("Team match should be case-insensitive, got %s", got)
	}
}

// ── Match-result (exact scoreline) bets ───────────────────────────────────────

func TestEvaluateMatchResultBet_PendingWhenNoFixtureOrNotStarted(t *testing.T) {
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, nil); got != "pending" {
		t.Fatalf("no fixture should be pending, got %s", got)
	}
	matches := []Match{{Status: "SCHEDULED", HomeTeam: "England", AwayTeam: "Ghana"}}
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, matches); got != "pending" {
		t.Fatalf("not-started fixture should be pending, got %s", got)
	}
}

func TestEvaluateMatchResultBet_WonExact(t *testing.T) {
	matches := []Match{finishedScore("England", "Ghana", 3, 1)}
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, matches); got != "won" {
		t.Fatalf("exact scoreline should be won, got %s", got)
	}
}

func TestEvaluateMatchResultBet_WonWhenFixtureReversed(t *testing.T) {
	// Actual fixture is Ghana (home) 1 - 3 England (away); bet written England 3-1 Ghana.
	matches := []Match{finishedScore("Ghana", "England", 1, 3)}
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, matches); got != "won" {
		t.Fatalf("reversed home/away should still resolve by identity, got %s", got)
	}
}

func TestEvaluateMatchResultBet_LostAtFullTime(t *testing.T) {
	matches := []Match{finishedScore("England", "Ghana", 2, 1)}
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, matches); got != "lost" {
		t.Fatalf("wrong final score should be lost, got %s", got)
	}
}

func TestEvaluateMatchResultBet_LostMidMatchWhenUnreachable(t *testing.T) {
	// Predicted 3-1 but Ghana already has 2 in play — away total can only rise.
	matches := []Match{{
		Status: "IN_PLAY", HomeTeam: "England", AwayTeam: "Ghana",
		HomeScore: ptr(1), AwayScore: ptr(2),
	}}
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, matches); got != "lost" {
		t.Fatalf("unreachable scoreline mid-match should be lost, got %s", got)
	}
}

func TestEvaluateMatchResultBet_AliveMidMatchWhenStillReachable(t *testing.T) {
	matches := []Match{{
		Status: "IN_PLAY", HomeTeam: "England", AwayTeam: "Ghana",
		HomeScore: ptr(2), AwayScore: ptr(1),
	}}
	if got := evaluateMatchResultBet("England", "Ghana", 3, 1, matches); got != "alive" {
		t.Fatalf("still-reachable scoreline mid-match should be alive, got %s", got)
	}
}

// ── Match-outcome accumulator legs ────────────────────────────────────────────

func TestEvaluateMatchOutcomeLeg(t *testing.T) {
	cases := []struct {
		name           string
		match          Match
		team, opponent string
		outcome, want  string
	}{
		{"win when team wins as home", Match{Stage: "GROUP_STAGE", Status: "FINISHED", HomeTeam: "England", AwayTeam: "Ghana", Winner: "HOME_TEAM"}, "England", "Ghana", "win", "won"},
		{"win when team wins as away", Match{Stage: "GROUP_STAGE", Status: "FINISHED", HomeTeam: "Ghana", AwayTeam: "England", Winner: "AWAY_TEAM"}, "England", "Ghana", "win", "won"},
		{"lost when team draws but predicted win", Match{Stage: "GROUP_STAGE", Status: "FINISHED", HomeTeam: "England", AwayTeam: "Ghana", Winner: "DRAW"}, "England", "Ghana", "win", "lost"},
		{"won when draw predicted and drawn", Match{Stage: "GROUP_STAGE", Status: "FINISHED", HomeTeam: "England", AwayTeam: "Ghana", Winner: "DRAW"}, "England", "Ghana", "draw", "won"},
		{"won when lose predicted and lost", Match{Stage: "GROUP_STAGE", Status: "FINISHED", HomeTeam: "England", AwayTeam: "Ghana", Winner: "AWAY_TEAM"}, "England", "Ghana", "lose", "won"},
		{"alive while in play", Match{Stage: "GROUP_STAGE", Status: "IN_PLAY", HomeTeam: "England", AwayTeam: "Ghana"}, "England", "Ghana", "win", "alive"},
		{"pending when not started", Match{Stage: "GROUP_STAGE", Status: "SCHEDULED", HomeTeam: "England", AwayTeam: "Ghana"}, "England", "Ghana", "win", "pending"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evaluateMatchOutcomeLeg(c.team, c.opponent, c.outcome, []Match{c.match})
			if got != c.want {
				t.Fatalf("%s: want %s, got %s", c.name, c.want, got)
			}
		})
	}
}

func TestCombineLegStatuses(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"won", "won", "won"}, "won"},
		{[]string{"won", "lost", "alive"}, "lost"},
		{[]string{"pending", "pending"}, "pending"},
		{[]string{"won", "pending", "alive"}, "alive"},
		{nil, "pending"},
	}
	for _, c := range cases {
		if got := combineLegStatuses(c.in); got != c.want {
			t.Fatalf("combineLegStatuses(%v): want %s, got %s", c.in, c.want, got)
		}
	}
}

// ── Finalist bets ─────────────────────────────────────────────────────────────

func TestEvaluateFinalistBet_AliveBeforeFinal(t *testing.T) {
	if got := evaluateFinalistBet("Mexico", "Switzerland", tournamentTestGroups(), nil); got != "alive" {
		t.Fatalf("both teams still in, expected alive, got %s", got)
	}
}

func TestEvaluateFinalistBet_LostWhenTeamEliminatedInGroup(t *testing.T) {
	if got := evaluateFinalistBet("South Africa", "Mexico", tournamentTestGroups(), nil); got != "lost" {
		t.Fatalf("South Africa finished 4th, expected lost, got %s", got)
	}
}

func TestEvaluateFinalistBet_LostWhenTeamLosesKnockout(t *testing.T) {
	matches := []Match{
		{Stage: "SEMI_FINALS", Status: "FINISHED", HomeTeam: "Mexico", AwayTeam: "Brazil", Winner: "AWAY_TEAM"},
	}
	if got := evaluateFinalistBet("Mexico", "Switzerland", tournamentTestGroups(), matches); got != "lost" {
		t.Fatalf("Mexico lost their semi, expected lost, got %s", got)
	}
}

func TestEvaluateFinalistBet_WonWhenBothInFinal(t *testing.T) {
	matches := []Match{
		{Stage: "FINAL", Status: "SCHEDULED", HomeTeam: "Mexico", AwayTeam: "Switzerland"},
	}
	if got := evaluateFinalistBet("Switzerland", "Mexico", tournamentTestGroups(), matches); got != "won" {
		t.Fatalf("both teams reached the final, expected won, got %s", got)
	}
}
