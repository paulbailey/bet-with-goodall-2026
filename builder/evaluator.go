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
// Returns one of: pending | alive | won | lost
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
			return "pending"
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
			return "pending"
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
	return "pending"
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

// teamsMatch does a case-insensitive comparison so team names in bets.yaml
// don't need to exactly match the API's casing.
func teamsMatch(a, b string) bool {
	return strings.EqualFold(a, b)
}
