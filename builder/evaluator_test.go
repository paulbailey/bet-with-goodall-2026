package main

import "testing"

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
