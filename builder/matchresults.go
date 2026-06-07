package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"
)

// Match-results page data: each finished fixture gets a record describing the
// result and how it moved the group's bets, written to a rolling archive in
// data/match-results.json. The PWA's /match/<id> page reads this file and
// renders the entry the push notification deep-links to.

const matchResultsLimit = 200 // plenty for a 104-match tournament

// ── Public (frontend-facing) shapes ───────────────────────────────────────────

type MatchResult struct {
	ID         string    `json:"id"`
	FinishedAt string    `json:"finished_at"` // RFC3339, when we recorded it
	Group      string    `json:"group"`       // e.g. "Group A" or "Last 16"
	Home       string    `json:"home"`
	Away       string    `json:"away"`
	HomeScore  int       `json:"home_score"`
	AwayScore  int       `json:"away_score"`
	Paragraph  string    `json:"paragraph"` // recap (Claude or templated)
	Risers     []Mover   `json:"risers"`
	Fallers    []Mover   `json:"fallers"`
	Settled    []Settled `json:"settled"`
}

type MatchResultsFile struct {
	UpdatedAt string        `json:"updated_at"`
	Results   []MatchResult `json:"results"` // newest first
}

// ── Generator ─────────────────────────────────────────────────────────────────

type matchResultGenerator struct {
	store     blobStore
	anth      *anthropicClient // nil → templated recap only
	publicKey string
	logger    *slog.Logger
}

// setupMatchResults builds the generator backed by the same store as the state
// file (local dir or S3 bucket). Returns nil to disable when the store can't be
// created. Independent of push being configured, so the page is populated for
// any caller (push CTA, a shared link, etc.).
func setupMatchResults(env Env, anth *anthropicClient, logger *slog.Logger) *matchResultGenerator {
	store, err := newBlobStore(env, logger)
	if err != nil {
		logger.Warn("match results disabled (blob store unavailable)", "err", err)
		return nil
	}
	return &matchResultGenerator{
		store:     store,
		anth:      anth,
		publicKey: env.MatchResultsKey,
		logger:    logger,
	}
}

// record builds the result entry for a finished fixture and prepends it to the
// public archive. risers/fallers/settled are the bet moves for the cycle the
// match finished in (see main loop).
func (g *matchResultGenerator) record(ctx context.Context, id string, m Match, risers, fallers []Mover, settled []Settled) {
	home, away := 0, 0
	if m.HomeScore != nil {
		home = *m.HomeScore
	}
	if m.AwayScore != nil {
		away = *m.AwayScore
	}

	mr := MatchResult{
		ID:         id,
		FinishedAt: time.Now().UTC().Format(time.RFC3339),
		Group:      matchGroupLabel(m),
		Home:       m.HomeTeam,
		Away:       m.AwayTeam,
		HomeScore:  home,
		AwayScore:  away,
		Paragraph:  g.narrate(ctx, matchScoreLine(m), risers, fallers, settled),
		Risers:     risers,
		Fallers:    fallers,
		Settled:    settled,
	}
	g.append(ctx, mr)
}

// append prepends the entry to the archive (replacing any existing entry with
// the same id so a re-run is idempotent) and writes the file.
func (g *matchResultGenerator) append(ctx context.Context, mr MatchResult) {
	file := MatchResultsFile{}
	if data, err := g.store.Get(ctx, g.publicKey); err != nil {
		g.logger.Warn("match results: load file failed; rewriting", "err", err)
	} else if len(data) > 0 {
		if err := json.Unmarshal(data, &file); err != nil {
			g.logger.Warn("match results: corrupt file; rewriting", "err", err)
			file = MatchResultsFile{}
		}
	}

	kept := file.Results[:0]
	for _, r := range file.Results {
		if r.ID != mr.ID {
			kept = append(kept, r)
		}
	}
	file.Results = append([]MatchResult{mr}, kept...)
	if len(file.Results) > matchResultsLimit {
		file.Results = file.Results[:matchResultsLimit]
	}
	file.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(file)
	if err != nil {
		g.logger.Error("match results: marshal failed", "err", err)
		return
	}
	if err := g.store.Put(ctx, g.publicKey, data); err != nil {
		g.logger.Error("match results: write failed", "err", err)
		return
	}
	g.logger.Info("match result recorded", "id", mr.ID, "risers", len(mr.Risers), "fallers", len(mr.Fallers), "settled", len(mr.Settled))
}

// matchGroupLabel renders a readable stage/group label, e.g. "GROUP_A" → "Group
// A", "LAST_16" → "Last 16".
func matchGroupLabel(m Match) string {
	if strings.HasPrefix(m.Group, "GROUP_") {
		return "Group " + strings.TrimPrefix(m.Group, "GROUP_")
	}
	return titleCase(strings.ReplaceAll(m.Stage, "_", " "))
}

// titleCase capitalises the first letter of each space-separated word and
// lowercases the rest ("LAST 16" → "Last 16").
func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		r := []rune(w)
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

// ── Recap narration ────────────────────────────────────────────────────────---

func (g *matchResultGenerator) narrate(ctx context.Context, score string, risers, fallers []Mover, settled []Settled) string {
	if g.anth != nil {
		digest := matchDigest(score, risers, fallers, settled)
		if text, err := g.anth.complete(ctx, matchSummarySystemPrompt, digest); err != nil {
			g.logger.Warn("match results: Claude narration failed; using fallback", "err", err)
		} else {
			return text
		}
	}
	return matchTemplateParagraph(risers, fallers, settled)
}

const matchSummarySystemPrompt = `You write a short recap for a website that tracks a group of friends' shared accumulator bets on the FIFA World Cup 2026. You are given a just-finished match result and how the group's bet win-probabilities moved as a result. Write a punchy, light-hearted 1-3 sentence summary of what this result did to the bets — call out anything that landed or went bust and the biggest swing. Use British English and a wry pub tone. No markdown, headings, or emoji. Respond with only the summary and nothing else. Do not invent results or numbers beyond what you are given.`

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

// matchTemplateParagraph is the deterministic fallback recap. It leads with
// anything that settled, then the single biggest likelihood swing.
func matchTemplateParagraph(risers, fallers []Mover, settled []Settled) string {
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
