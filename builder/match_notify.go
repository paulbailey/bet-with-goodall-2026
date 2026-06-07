package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
)

// Match-result notifications: when a fixture transitions to FINISHED between two
// poll cycles, send a push with the score and a one-line summary of how the
// group's pending bets moved. The bet movement is the diff between the previous
// cycle's snapshot and the current one (reusing the daily-summary machinery), so
// during a live match window — where polling is frequent and usually only one
// match finishes per cycle — it reads as "this result did X to the bets".

// matchKey is a stable identifier for a fixture, used to dedupe notifications
// (as the notification tag) and to track status across cycles.
func matchKey(m Match) string {
	return fmt.Sprintf("%s|%s|%s", m.UtcDate.UTC().Format("2006-01-02"), m.HomeTeam, m.AwayTeam)
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

// buildMatchNotification composes the push for one finished fixture from the
// cycle's bet movers and settled bets.
func buildMatchNotification(ctx context.Context, m Match, risers, fallers []Mover, settled []Settled, anth *anthropicClient, logger *slog.Logger) Notification {
	score := matchScoreLine(m)
	return Notification{
		Title: "Full time: " + score,
		Body:  matchNarrate(ctx, score, risers, fallers, settled, anth, logger),
		URL:   "/",
		Tag:   "match-" + matchKey(m),
	}
}

// matchNarrate writes the one-line body: Claude when configured, a templated
// sentence otherwise.
func matchNarrate(ctx context.Context, score string, risers, fallers []Mover, settled []Settled, anth *anthropicClient, logger *slog.Logger) string {
	if anth != nil {
		digest := matchDigest(score, risers, fallers, settled)
		if text, err := anth.complete(ctx, matchSummarySystemPrompt, digest); err != nil {
			logger.Warn("push: Claude narration failed; using fallback", "err", err)
		} else {
			return text
		}
	}
	return matchTemplateBody(risers, fallers, settled)
}

const matchSummarySystemPrompt = `You write a one-line push notification body for a website that tracks a group of friends' shared accumulator bets on the FIFA World Cup 2026. You are given a just-finished match result and how the group's bet win-probabilities moved as a result. Write a single punchy sentence (aim for under 140 characters) in British English with a wry pub tone, summarising what this result did to the bets. Call out anything that landed or went bust, or the biggest swing. No markdown, no headings, no emoji, no preamble — respond with only the sentence. Do not invent results or numbers beyond what you are given.`

// matchDigest renders the result and bet moves as a compact prompt block.
func matchDigest(score string, risers, fallers []Mover, settled []Settled) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Result: %s\n", score)
	b.WriteString("\nBiggest risers:\n")
	writeMoverLines(&b, risers)
	b.WriteString("\nBiggest fallers:\n")
	writeMoverLines(&b, fallers)
	if len(settled) > 0 {
		b.WriteString("\nSettled by this result:\n")
		for _, st := range settled {
			outcome := "WON"
			if st.Status == "lost" {
				outcome = "BUST"
			}
			fmt.Fprintf(&b, "- %s (%s): %s\n", st.Label, st.Category, outcome)
		}
	}
	return b.String()
}

// matchTemplateBody is the deterministic fallback. It leads with anything that
// settled, then the single biggest likelihood swing.
func matchTemplateBody(risers, fallers []Mover, settled []Settled) string {
	var wins, busts []string
	for _, s := range settled {
		if s.Status == "won" {
			wins = append(wins, s.Label)
		} else {
			busts = append(busts, s.Label)
		}
	}

	var parts []string
	if len(wins) > 0 {
		parts = append(parts, "Landed: "+strings.Join(wins, ", ")+".")
	}
	if len(busts) > 0 {
		parts = append(parts, "Bust: "+strings.Join(busts, ", ")+".")
	}

	if big := biggestMover(risers, fallers); big != nil {
		dir := "up"
		if big.NewProb < big.PrevProb {
			dir = "down"
		}
		parts = append(parts, fmt.Sprintf("%s %s (%s→%s).", big.Label, dir, pctStr(big.PrevProb), pctStr(big.NewProb)))
	}

	if len(parts) == 0 {
		return "No change to the group's bets."
	}
	return strings.Join(parts, " ")
}

// biggestMover returns the mover with the largest relative swing across the
// (already sorted) risers and fallers, or nil if there were none.
func biggestMover(risers, fallers []Mover) *Mover {
	var best *Mover
	bestSwing := -1.0
	consider := func(m *Mover) {
		swing := math.Abs(m.Ratio - 1)
		if swing > bestSwing {
			best, bestSwing = m, swing
		}
	}
	if len(risers) > 0 {
		consider(&risers[0])
	}
	if len(fallers) > 0 {
		consider(&fallers[0])
	}
	return best
}
