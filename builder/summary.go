package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Daily summary: after a tournament day's matches have all finished, capture how
// every priced bet's likelihood moved versus the previous close, rank the
// biggest movers, note anything that won or went bust, and render a short
// natural-language recap (Claude when configured, a templated fallback
// otherwise). The result is written to a public JSON file the frontend reads and
// kept as a rolling archive.

const (
	// maxMovers caps each of the risers/fallers lists surfaced per day.
	maxMovers = 5
	// archiveLimit bounds how many days of summaries we retain in the public
	// file so it stays cheap to fetch over a whole tournament.
	archiveLimit = 90
	// moverFloor is the smallest move worth listing — either a ±10% relative
	// swing or a ±0.5 percentage-point absolute swing. Below this is noise from
	// the simulator/odds jitter rather than a real story.
	moverRelFloor = 0.10
	moverAbsFloor = 0.005
)

// ── Public (frontend-facing) shapes, written to data/daily-summary.json ───────

type DailySummaryFile struct {
	UpdatedAt string         `json:"updated_at"`
	Summaries []DailySummary `json:"summaries"` // newest first
}

type DailySummary struct {
	Date        string    `json:"date"`         // YYYY-MM-DD tournament day that just finished
	GeneratedAt string    `json:"generated_at"` // RFC3339
	Paragraph   string    `json:"paragraph"`
	Risers      []Mover   `json:"risers"`
	Fallers     []Mover   `json:"fallers"`
	Settled     []Settled `json:"settled"`
	BetsTracked int       `json:"bets_tracked"`
}

// Mover is one bet whose likelihood changed. Both the absolute (delta_pp) and
// relative (ratio) figures are carried so the UI can show the hybrid view; the
// lists themselves are ranked by relative change so long-shot accumulators that
// swing in relative terms aren't buried under shorter-priced bets.
type Mover struct {
	ID       string  `json:"id"`
	Label    string  `json:"label"`
	Category string  `json:"category"`
	Status   string  `json:"status"`
	PrevProb float64 `json:"prev_prob"` // 0–1 fraction
	NewProb  float64 `json:"new_prob"`  // 0–1 fraction
	DeltaPP  float64 `json:"delta_pp"`  // new - prev, as a fraction (frontend ×100)
	Ratio    float64 `json:"ratio"`     // new / prev
}

// Settled is a bet that crossed the line today — either landed or went bust.
type Settled struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Category string `json:"category"`
	Status   string `json:"status"` // "won" | "lost"
}

// ── Builder-private snapshot state, written to data/summary-state.json ────────

type betSnap struct {
	Prob   *float64 `json:"prob"`
	Status string   `json:"status"`
}

type summaryState struct {
	Baseline           map[string]betSnap `json:"baseline"`             // pre-tournament close, the day-0 reference
	Close              map[string]betSnap `json:"close"`                // most recent summarized day's close
	LastDate           string             `json:"last_date"`            // last summarized tournament day
	PrevExpectedPayout *float64           `json:"prev_expected_payout"` // expected payout at the previous close
}

// betInfo is the labelled, current-cycle view of one bet, derived entirely from
// the state file so the summary is self-contained.
type betInfo struct {
	Key      string // category + ":" + id — unique across bet types
	ID       string
	Label    string
	Category string
	Status   string
	Prob     *float64
}

// ── Generator ─────────────────────────────────────────────────────────────────

type summaryGenerator struct {
	store     blobStore
	anth      *anthropicClient // nil → templated narration only
	loc       *time.Location   // tournament-day timezone for grouping fixtures
	publicKey string
	stateKey  string
	logger    *slog.Logger

	state  *summaryState
	loaded bool
}

// maybeGenerate is called every poll cycle. It does cheap work (detect the
// latest completed day) on the hot path and only touches the network — loading
// state, calling Claude, writing files — when there's a new day to summarise or
// a baseline to capture for the first time.
func (g *summaryGenerator) maybeGenerate(ctx context.Context, s StateJSON, matches []Match) {
	if !g.loaded {
		g.state = g.loadState(ctx)
		g.loaded = true
	}

	day := latestCompletedDay(matches, g.loc)
	needBaseline := len(g.state.Baseline) == 0
	if !needBaseline && (day == "" || day <= g.state.LastDate) {
		return // nothing finished that we haven't already covered
	}

	infos := collectBets(s)
	snap := snapshot(infos)

	if needBaseline {
		g.state.Baseline = snap
		if s.Expected != nil {
			ep := s.Expected.ExpectedPayout
			g.state.PrevExpectedPayout = &ep
		}
		g.saveState(ctx)
	}

	if day == "" || day <= g.state.LastDate {
		return
	}

	prev := g.state.Close
	if len(prev) == 0 {
		prev = g.state.Baseline
	}

	risers, fallers, settled := computeMovers(prev, infos)
	para := g.narrate(ctx, day, s, risers, fallers, settled)

	ds := DailySummary{
		Date:        day,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Paragraph:   para,
		Risers:      risers,
		Fallers:     fallers,
		Settled:     settled,
		BetsTracked: countPriced(infos),
	}
	g.appendPublic(ctx, ds)

	g.state.Close = snap
	g.state.LastDate = day
	if s.Expected != nil {
		ep := s.Expected.ExpectedPayout
		g.state.PrevExpectedPayout = &ep
	}
	g.saveState(ctx)

	g.logger.Info("daily summary generated",
		"date", day, "risers", len(risers), "fallers", len(fallers), "settled", len(settled))
}

func (g *summaryGenerator) loadState(ctx context.Context) *summaryState {
	st := &summaryState{}
	data, err := g.store.Get(ctx, g.stateKey)
	if err != nil {
		g.logger.Warn("daily summary: load state failed; starting fresh", "err", err)
		return st
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, st); err != nil {
			g.logger.Warn("daily summary: corrupt state; starting fresh", "err", err)
			return &summaryState{}
		}
	}
	return st
}

func (g *summaryGenerator) saveState(ctx context.Context) {
	data, err := json.Marshal(g.state)
	if err != nil {
		g.logger.Error("daily summary: marshal state failed", "err", err)
		return
	}
	if err := g.store.Put(ctx, g.stateKey, data); err != nil {
		g.logger.Error("daily summary: save state failed", "err", err)
	}
}

// appendPublic prepends the new summary to the rolling archive and writes it.
func (g *summaryGenerator) appendPublic(ctx context.Context, ds DailySummary) {
	file := DailySummaryFile{}
	if data, err := g.store.Get(ctx, g.publicKey); err != nil {
		g.logger.Warn("daily summary: load public file failed; rewriting", "err", err)
	} else if len(data) > 0 {
		if err := json.Unmarshal(data, &file); err != nil {
			g.logger.Warn("daily summary: corrupt public file; rewriting", "err", err)
			file = DailySummaryFile{}
		}
	}

	// Drop any existing entry for this day (idempotent re-run) then prepend.
	kept := file.Summaries[:0]
	for _, s := range file.Summaries {
		if s.Date != ds.Date {
			kept = append(kept, s)
		}
	}
	file.Summaries = append([]DailySummary{ds}, kept...)
	if len(file.Summaries) > archiveLimit {
		file.Summaries = file.Summaries[:archiveLimit]
	}
	file.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(file)
	if err != nil {
		g.logger.Error("daily summary: marshal public file failed", "err", err)
		return
	}
	if err := g.store.Put(ctx, g.publicKey, data); err != nil {
		g.logger.Error("daily summary: write public file failed", "err", err)
	}
}

// ── Day detection ─────────────────────────────────────────────────────────────

// latestCompletedDay returns the most recent tournament day (in loc) on which
// every scheduled fixture has finished, or "" if no day is fully complete.
// Grouping by local date keeps an evening's fixtures together rather than
// splitting them across the UTC midnight that falls mid-evening in North America.
func latestCompletedDay(matches []Match, loc *time.Location) string {
	type agg struct {
		any       bool
		allFinish bool
	}
	days := map[string]*agg{}
	for _, m := range matches {
		d := m.UtcDate.In(loc).Format("2006-01-02")
		a := days[d]
		if a == nil {
			a = &agg{allFinish: true}
			days[d] = a
		}
		a.any = true
		if m.Status != "FINISHED" {
			a.allFinish = false
		}
	}

	best := ""
	for d, a := range days {
		if a.any && a.allFinish && d > best {
			best = d
		}
	}
	return best
}

// ── Movement ──────────────────────────────────────────────────────────────────

func snapshot(infos []betInfo) map[string]betSnap {
	out := make(map[string]betSnap, len(infos))
	for _, b := range infos {
		var p *float64
		if b.Prob != nil {
			v := roundProb(*b.Prob)
			p = &v
		}
		out[b.Key] = betSnap{Prob: p, Status: b.Status}
	}
	return out
}

func countPriced(infos []betInfo) int {
	n := 0
	for _, b := range infos {
		if b.Prob != nil {
			n++
		}
	}
	return n
}

// computeMovers diffs the current cycle against the previous close. Risers and
// fallers are ranked by relative change (most extreme first) and capped; settled
// bets are those that were alive at the previous close and are now decided.
func computeMovers(prev map[string]betSnap, infos []betInfo) (risers, fallers []Mover, settled []Settled) {
	for _, b := range infos {
		p, hadPrev := prev[b.Key]

		if hadPrev && p.Status == "alive" && (b.Status == "won" || b.Status == "lost") {
			settled = append(settled, Settled{ID: b.ID, Label: b.Label, Category: b.Category, Status: b.Status})
		}

		if !hadPrev || b.Prob == nil || p.Prob == nil {
			continue
		}
		prevP, newP := *p.Prob, roundProb(*b.Prob)
		if prevP <= 0 && newP <= 0 {
			continue
		}

		delta := newP - prevP
		ratio := math.Inf(1)
		if prevP > 0 {
			ratio = newP / prevP
		}
		if !significantMove(delta, ratio) {
			continue
		}

		m := Mover{
			ID: b.ID, Label: b.Label, Category: b.Category, Status: b.Status,
			PrevProb: prevP, NewProb: newP, DeltaPP: roundProb(delta), Ratio: ratio,
		}
		if newP > prevP {
			risers = append(risers, m)
		} else {
			fallers = append(fallers, m)
		}
	}

	// Risers: largest relative gain first. Fallers: largest relative drop first
	// (smallest ratio), so a bet that went bust — ratio 0 — leads the list.
	sort.SliceStable(risers, func(i, j int) bool { return risers[i].Ratio > risers[j].Ratio })
	sort.SliceStable(fallers, func(i, j int) bool { return fallers[i].Ratio < fallers[j].Ratio })

	return cap5(risers), cap5(fallers), settled
}

func significantMove(delta, ratio float64) bool {
	return math.Abs(delta) >= moverAbsFloor || math.Abs(ratio-1) >= moverRelFloor
}

func cap5(m []Mover) []Mover {
	if len(m) > maxMovers {
		return m[:maxMovers]
	}
	return m
}

func roundProb(p float64) float64 {
	return math.Round(p*1e5) / 1e5
}

// ── Bet labelling (derived from the state file) ───────────────────────────────

func collectBets(s StateJSON) []betInfo {
	var out []betInfo
	fav := favByGroup(s.Bets)

	for _, b := range s.Bets {
		out = append(out, betInfo{
			Key: key("Group acca", b.ID), ID: b.ID, Label: groupAccaLabel(b, fav),
			Category: "Group acca", Status: b.Status, Prob: b.Probability,
		})
	}
	for _, b := range s.MatchAccaBets {
		out = append(out, betInfo{
			Key: key("Match acca", b.ID), ID: b.ID, Label: matchAccaLabel(b),
			Category: "Match acca", Status: b.Status, Prob: b.Probability,
		})
	}
	for _, b := range s.MatchResultBets {
		out = append(out, betInfo{
			Key: key("Exact score", b.ID), ID: b.ID,
			Label:    fmt.Sprintf("%s %d-%d %s", b.TeamA, b.ScoreA, b.ScoreB, b.TeamB),
			Category: "Exact score", Status: b.Status, Prob: b.Probability,
		})
	}
	for _, b := range s.TournamentWinnerBets {
		out = append(out, betInfo{
			Key: key("Tournament winner", b.ID), ID: b.ID,
			Label:    fmt.Sprintf("%s to win the tournament", b.Team),
			Category: "Tournament winner", Status: b.Status, Prob: b.Probability,
		})
	}
	for _, b := range s.TopScorerBets {
		out = append(out, betInfo{
			Key: key("Top scorer", b.ID), ID: b.ID,
			Label:    fmt.Sprintf("%s top scorer", b.Player),
			Category: "Top scorer", Status: b.Status, Prob: b.Probability,
		})
	}
	for _, b := range s.FinalistBets {
		out = append(out, betInfo{
			Key: key("Finalists", b.ID), ID: b.ID,
			Label:    fmt.Sprintf("%s & %s to reach the final", b.TeamA, b.TeamB),
			Category: "Finalists", Status: b.Status, Prob: b.Probability,
		})
	}
	return out
}

func key(category, id string) string { return category + ":" + id }

// favByGroup mirrors the frontend: the team picked by the most bets in each
// group is the "favourite", and a bet is labelled by where it departs from them.
func favByGroup(bets []BetJSON) map[string]string {
	counts := map[string]map[string]int{}
	for _, b := range bets {
		for _, l := range b.Legs {
			if counts[l.Group] == nil {
				counts[l.Group] = map[string]int{}
			}
			counts[l.Group][l.Team]++
		}
	}
	fav := make(map[string]string, len(counts))
	for g, teams := range counts {
		best, bestN := "", -1
		for team, n := range teams {
			if n > bestN {
				best, bestN = team, n
			}
		}
		fav[g] = best
	}
	return fav
}

func groupAccaLabel(b BetJSON, fav map[string]string) string {
	var dev []string
	for _, l := range b.Legs {
		if f, ok := fav[l.Group]; ok && l.Team != f {
			dev = append(dev, fmt.Sprintf("%s: %s", l.Group, l.Team))
		}
	}
	if len(dev) == 0 {
		return "Favourites"
	}
	return strings.Join(dev, ", ")
}

// matchAccaLabel names the accumulator by its common team when every leg is the
// same side (the usual "England to win all three" shape), else generically.
func matchAccaLabel(b MatchAccaBetJSON) string {
	team := ""
	for i, l := range b.Legs {
		if i == 0 {
			team = l.Team
		} else if l.Team != team {
			team = ""
			break
		}
	}
	if team != "" {
		return fmt.Sprintf("%s match acca (%d legs)", team, len(b.Legs))
	}
	return fmt.Sprintf("Match accumulator (%d legs)", len(b.Legs))
}

// ── Narration ─────────────────────────────────────────────────────────────────

const summarySystemPrompt = `You write the daily recap for a website that tracks a group of friends' shared accumulator bets on the FIFA World Cup 2026. The bets are mostly long-odds group-winner accumulators plus a few tournament-winner, top-scorer, exact-score and finalist bets. The figures you are given are model-estimated win probabilities.

Write a punchy, light-hearted 2-4 sentence summary of how the group's bets moved on the day. Call out the single biggest mover and anything that won or went bust. Use British English and a wry pub tone. Do not use markdown, bullet points, or headings. Respond with only the summary paragraph and nothing else. Do not invent results or numbers beyond what you are given.`

func (g *summaryGenerator) narrate(ctx context.Context, day string, s StateJSON, risers, fallers []Mover, settled []Settled) string {
	digest := buildDigest(day, s, risers, fallers, settled, g.state.PrevExpectedPayout)
	if g.anth != nil {
		if text, err := g.anth.complete(ctx, summarySystemPrompt, digest); err != nil {
			g.logger.Warn("daily summary: Claude narration failed; using fallback", "err", err)
		} else {
			return text
		}
	}
	return templateParagraph(risers, fallers, settled)
}

// buildDigest renders the day's moves as a compact text block for the model.
func buildDigest(day string, s StateJSON, risers, fallers []Mover, settled []Settled, prevEP *float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Day finished: %s\n", day)
	if s.Expected != nil {
		fmt.Fprintf(&b, "Expected total payout now: %.2f", s.Expected.ExpectedPayout)
		if prevEP != nil {
			fmt.Fprintf(&b, " (was %.2f at previous close)", *prevEP)
		}
		b.WriteByte('\n')
	}

	b.WriteString("\nBiggest risers:\n")
	writeMoverLines(&b, risers)
	b.WriteString("\nBiggest fallers:\n")
	writeMoverLines(&b, fallers)

	if len(settled) > 0 {
		b.WriteString("\nSettled today:\n")
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

func writeMoverLines(b *strings.Builder, movers []Mover) {
	if len(movers) == 0 {
		b.WriteString("- (none)\n")
		return
	}
	for _, m := range movers {
		fmt.Fprintf(b, "- %s (%s): %s -> %s (%.1fx)\n",
			m.Label, m.Category, pctStr(m.PrevProb), pctStr(m.NewProb), m.Ratio)
	}
}

// templateParagraph is the deterministic fallback when Claude isn't configured
// or the call fails.
func templateParagraph(risers, fallers []Mover, settled []Settled) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%d bets drifted up and %d drifted down.", len(risers), len(fallers)))
	if len(risers) > 0 {
		r := risers[0]
		parts = append(parts, fmt.Sprintf("Biggest climber was %s, up from %s to %s.", r.Label, pctStr(r.PrevProb), pctStr(r.NewProb)))
	}
	if len(fallers) > 0 {
		f := fallers[0]
		parts = append(parts, fmt.Sprintf("Biggest faller was %s, down from %s to %s.", f.Label, pctStr(f.PrevProb), pctStr(f.NewProb)))
	}
	var wins, busts []string
	for _, st := range settled {
		if st.Status == "won" {
			wins = append(wins, st.Label)
		} else {
			busts = append(busts, st.Label)
		}
	}
	if len(wins) > 0 {
		parts = append(parts, fmt.Sprintf("Landed: %s.", strings.Join(wins, ", ")))
	}
	if len(busts) > 0 {
		parts = append(parts, fmt.Sprintf("Bust: %s.", strings.Join(busts, ", ")))
	}
	return strings.Join(parts, " ")
}

// pctStr renders a 0–1 probability for the digest/fallback text, mirroring the
// frontend's significant-figure formatting closely enough to read naturally.
func pctStr(p float64) string {
	switch {
	case p <= 0:
		return "0%"
	case p >= 1:
		return "100%"
	}
	percent := p * 100
	if percent < 0.0001 {
		return "<0.0001%"
	}
	// Two significant figures, trailing zeros trimmed.
	prec := 1 - int(math.Floor(math.Log10(percent)))
	if prec < 0 {
		prec = 0
	}
	s := strconv.FormatFloat(percent, 'f', prec, 64)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	return s + "%"
}
