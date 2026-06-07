package main

import (
	"fmt"
	"strings"
)

// Match-result notifications: when a fixture transitions to FINISHED between two
// poll cycles, the builder records how the result moved the group's bets (see
// matchresults.go) and sends a push. The push itself carries only the score and
// a call to action — tapping it deep-links to /match/<id>, the statically-served
// PWA page that renders the bet changes from data/match-results.json.

// matchKey is a stable identifier for a fixture, used to track status across
// cycles (it must not change when a score/status updates).
func matchKey(m Match) string {
	return fmt.Sprintf("%s|%s|%s", m.UtcDate.UTC().Format("2006-01-02"), m.HomeTeam, m.AwayTeam)
}

// matchSlug is the URL-safe id shared by the push deep link and the
// match-results record, e.g. "2026-06-20-spain-v-japan".
func matchSlug(m Match) string {
	return fmt.Sprintf("%s-%s-v-%s", m.UtcDate.UTC().Format("2006-01-02"), slugify(m.HomeTeam), slugify(m.AwayTeam))
}

// slugify lowercases and hyphenates a team name for use in a URL.
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// statusMap captures each fixture's current status, keyed by matchKey.
func statusMap(matches []Match) map[string]string {
	out := make(map[string]string, len(matches))
	for _, m := range matches {
		out[matchKey(m)] = m.Status
	}
	return out
}

// newlyFinished returns the fixtures that are FINISHED now but were known to be
// unfinished in the previous cycle. A nil/empty prev means we don't yet have a
// baseline (first cycle, or a fresh restart) — we return nothing so a restart
// doesn't replay every already-finished match.
func newlyFinished(prev map[string]string, matches []Match) []Match {
	if len(prev) == 0 {
		return nil
	}
	var out []Match
	for _, m := range matches {
		if m.Status != "FINISHED" {
			continue
		}
		was, known := prev[matchKey(m)]
		if known && was != "FINISHED" {
			out = append(out, m)
		}
	}
	return out
}

// matchScoreLine renders "Home 2–1 Away", falling back to the fixture name when
// the score isn't populated yet (shouldn't happen for a FINISHED match).
func matchScoreLine(m Match) string {
	if m.HomeScore != nil && m.AwayScore != nil {
		return fmt.Sprintf("%s %d–%d %s", m.HomeTeam, *m.HomeScore, *m.AwayScore, m.AwayTeam)
	}
	return fmt.Sprintf("%s vs %s", m.HomeTeam, m.AwayTeam)
}

// buildMatchNotification composes the push for one finished fixture: the score
// as the title and a CTA as the body, deep-linking to the bet-changes page. The
// detail (movers, settled bets, recap) lives on that page, not in the push.
func buildMatchNotification(m Match, id string) Notification {
	return Notification{
		Title: "Full time: " + matchScoreLine(m),
		Body:  "Tap to see how it changed the bets →",
		URL:   "/match/" + id,
		Tag:   "match-" + matchKey(m),
	}
}
