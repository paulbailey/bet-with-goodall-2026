package main

import (
	"strings"
	"time"
)

const groupGames = 3 // each team plays 3 group-stage games

// nextPollInterval returns how long to sleep before the next fetch cycle.
// 2 minutes during a match window, up to 30 minutes otherwise.
func nextPollInterval(matches []Match) time.Duration {
	now := time.Now()

	for _, m := range matches {
		if m.Status == "IN_PLAY" || m.Status == "PAUSED" {
			return 2 * time.Minute
		}
	}

	var nextKickoff time.Time
	for _, m := range matches {
		if !notStarted(m.Status) {
			continue
		}
		if nextKickoff.IsZero() || m.UtcDate.Before(nextKickoff) {
			nextKickoff = m.UtcDate
		}
	}

	if nextKickoff.IsZero() {
		return 30 * time.Minute
	}

	// Poll every 2 min within the window [kickoff-30m, kickoff+3h]
	windowOpen := nextKickoff.Add(-30 * time.Minute)
	if now.After(windowOpen) {
		return 2 * time.Minute
	}

	// Sleep until the window opens, capped at 30 minutes so we don't drift too far.
	untilWindow := time.Until(windowOpen)
	if untilWindow > 30*time.Minute {
		return 30 * time.Minute
	}
	return untilWindow
}

// evaluateLeg determines whether the predicted group winner is still possible.
// Returns one of: alive | won | lost. Anything not yet decided is "alive".
func evaluateLeg(group, teamName string, groups []GroupStanding) string {
	for _, g := range groups {
		if g.Group != group {
			continue
		}

		var predicted *TeamStats
		for i := range g.Teams {
			if teamsMatch(g.Teams[i].Name, teamName) {
				predicted = &g.Teams[i]
				break
			}
		}
		if predicted == nil {
			// Team not found in this group — config mismatch
			return "alive"
		}

		// Group is complete when all teams have played 3 games
		groupComplete := true
		for _, t := range g.Teams {
			if t.Played < groupGames {
				groupComplete = false
				break
			}
		}

		if groupComplete {
			// Winner is the team at position 0 (already sorted by standing)
			if teamsMatch(g.Teams[0].Name, teamName) {
				return "won"
			}
			return "lost"
		}

		if predicted.Played == 0 {
			// Group hasn't kicked off yet
			return "alive"
		}

		// Mathematically impossible if another team's current points already
		// exceed the maximum our team can ever reach.
		maxPoints := predicted.Points + 3*(groupGames-predicted.Played)
		for _, other := range g.Teams {
			if teamsMatch(other.Name, teamName) {
				continue
			}
			if other.Points > maxPoints {
				return "lost"
			}
		}

		return "alive"
	}

	// No group data available yet
	return "alive"
}

// isTeamEliminated returns true once a team's group is complete and they
// finished 4th. Best-third-place advancement is not modelled here; that
// refinement can be added once the group stage is underway.
func isTeamEliminated(teamName string, groups []GroupStanding) bool {
	for _, g := range groups {
		for i, t := range g.Teams {
			if !teamsMatch(t.Name, teamName) {
				continue
			}
			groupComplete := true
			for _, tt := range g.Teams {
				if tt.Played < groupGames {
					groupComplete = false
					break
				}
			}
			// Only mark eliminated if group is done and they finished last
			return groupComplete && i == 3
		}
	}
	return false
}

// evaluateTopScorerBet returns alive | lost | won for a top-scorer bet.
// A bet is lost once the player's team is eliminated AND an active player
// already has strictly more goals.
func evaluateTopScorerBet(playerName string, scorers []TopScorerEntry, groups []GroupStanding) string {
	var predicted *TopScorerEntry
	for i := range scorers {
		if strings.EqualFold(scorers[i].Player, playerName) {
			predicted = &scorers[i]
			break
		}
	}
	if predicted == nil {
		return "alive"
	}

	if !isTeamEliminated(predicted.Team, groups) {
		return "alive"
	}

	// Player's team is out — their tally is locked in.
	// Lost if any still-active player already has more goals.
	for _, s := range scorers {
		if strings.EqualFold(s.Player, playerName) {
			continue
		}
		if !isTeamEliminated(s.Team, groups) && s.Goals > predicted.Goals {
			return "lost"
		}
	}
	return "alive"
}

// evaluateTournamentWinnerBet returns alive | lost | won for an overall-winner bet.
// Lost: the team finished 4th in their group, or lost a knockout match.
// Won:  the team won the FINAL.
func evaluateTournamentWinnerBet(teamName string, groups []GroupStanding, matches []Match) string {
	if didTeamWinFinal(teamName, matches) {
		return "won"
	}
	if isTeamEliminated(teamName, groups) || lostKnockoutMatch(teamName, matches) {
		return "lost"
	}
	return "alive"
}

// lostKnockoutMatch returns true if the team played any finished knockout
// match (R16, QF, SF, 3rd-place playoff, FINAL) as the losing side.
func lostKnockoutMatch(teamName string, matches []Match) bool {
	for _, m := range matches {
		if m.Status != "FINISHED" || strings.Contains(m.Stage, "GROUP") {
			continue
		}
		switch {
		case teamsMatch(m.HomeTeam, teamName) && m.Winner == "AWAY_TEAM":
			return true
		case teamsMatch(m.AwayTeam, teamName) && m.Winner == "HOME_TEAM":
			return true
		}
	}
	return false
}

// didTeamWinFinal returns true if the team won a finished match at the FINAL stage.
func didTeamWinFinal(teamName string, matches []Match) bool {
	for _, m := range matches {
		if m.Stage != "FINAL" || m.Status != "FINISHED" {
			continue
		}
		if teamsMatch(m.HomeTeam, teamName) && m.Winner == "HOME_TEAM" {
			return true
		}
		if teamsMatch(m.AwayTeam, teamName) && m.Winner == "AWAY_TEAM" {
			return true
		}
	}
	return false
}

// teamsMatch does a case-insensitive comparison so team names in bets.yaml
// don't need to exactly match the API's casing.
func teamsMatch(a, b string) bool {
	return strings.EqualFold(a, b)
}

// findMatch returns the fixture between two teams in either home/away
// orientation. In the group stage each pair meets once, and any knockout pair
// meets once, so the result is unambiguous. Returns nil if no such fixture.
func findMatch(teamA, teamB string, matches []Match) *Match {
	for i := range matches {
		m := &matches[i]
		if (teamsMatch(m.HomeTeam, teamA) && teamsMatch(m.AwayTeam, teamB)) ||
			(teamsMatch(m.HomeTeam, teamB) && teamsMatch(m.AwayTeam, teamA)) {
			return m
		}
	}
	return nil
}

// teamScores returns the (team, opponent) goals for a match from team's
// perspective, plus whether scores are populated yet.
func teamScores(m *Match, team string) (forGoals, againstGoals int, ok bool) {
	if m.HomeScore == nil || m.AwayScore == nil {
		return 0, 0, false
	}
	if teamsMatch(m.HomeTeam, team) {
		return *m.HomeScore, *m.AwayScore, true
	}
	return *m.AwayScore, *m.HomeScore, true
}

// evaluateMatchResultBet returns alive | won | lost for an exact scoreline.
// It flips to "lost" mid-match the moment the scoreline becomes unreachable
// (goals only ever increase). Anything not yet decided is "alive".
func evaluateMatchResultBet(teamA, teamB string, scoreA, scoreB int, matches []Match) string {
	m := findMatch(teamA, teamB, matches)
	if m == nil || notStarted(m.Status) {
		return "alive"
	}

	forA, againstA, ok := teamScores(m, teamA)
	if !ok {
		return "alive"
	}

	if m.Status == "FINISHED" {
		if forA == scoreA && againstA == scoreB {
			return "won"
		}
		return "lost"
	}

	// In play: unreachable once either side has already scored more than predicted.
	if forA > scoreA || againstA > scoreB {
		return "lost"
	}
	return "alive"
}

// evaluateMatchOutcomeLeg returns alive | won | lost for a single
// "team to win/draw/lose against opponent" prediction. A result can swing until
// the final whistle, so there is no mid-match bust here. Anything not yet
// decided is "alive".
func evaluateMatchOutcomeLeg(team, opponent, outcome string, matches []Match) string {
	m := findMatch(team, opponent, matches)
	if m == nil || notStarted(m.Status) {
		return "alive"
	}
	if m.Status != "FINISHED" {
		return "alive"
	}

	var actual string
	switch {
	case m.Winner == "DRAW":
		actual = "draw"
	case (teamsMatch(m.HomeTeam, team) && m.Winner == "HOME_TEAM") ||
		(teamsMatch(m.AwayTeam, team) && m.Winner == "AWAY_TEAM"):
		actual = "win"
	default:
		actual = "lose"
	}

	if strings.EqualFold(actual, outcome) {
		return "won"
	}
	return "lost"
}

// evaluateFinalistBet returns alive | won | lost for a predicted final pairing.
// Lost once either team is knocked out; won once both teams occupy the final.
func evaluateFinalistBet(teamA, teamB string, groups []GroupStanding, matches []Match) string {
	for _, team := range []string{teamA, teamB} {
		if isTeamEliminated(team, groups) || lostKnockoutMatch(team, matches) {
			return "lost"
		}
	}
	if finalContains(teamA, teamB, matches) {
		return "won"
	}
	return "alive"
}

// finalContains reports whether the FINAL fixture features both teams. True
// once the semi-finals populate the final, regardless of whether it's played.
func finalContains(teamA, teamB string, matches []Match) bool {
	for _, m := range matches {
		if m.Stage != "FINAL" {
			continue
		}
		if (teamsMatch(m.HomeTeam, teamA) && teamsMatch(m.AwayTeam, teamB)) ||
			(teamsMatch(m.HomeTeam, teamB) && teamsMatch(m.AwayTeam, teamA)) {
			return true
		}
	}
	return false
}

// combineLegStatuses rolls per-leg statuses up into an accumulator status:
// any lost → lost; all won → won; otherwise alive.
func combineLegStatuses(statuses []string) string {
	if len(statuses) == 0 {
		return "alive"
	}
	allWon := true
	for _, s := range statuses {
		if s == "lost" {
			return "lost"
		}
		if s != "won" {
			allWon = false
		}
	}
	if allWon {
		return "won"
	}
	return "alive"
}
