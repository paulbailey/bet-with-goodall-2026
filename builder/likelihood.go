package main

import (
	"context"
	"log/slog"
	"strings"
)

// Per-bet likelihood combiner. It turns market odds (de-vigged, via OddsProvider)
// and the Monte Carlo simulator into a win probability for every open bet, then
// rolls those up into an expected payout.
//
// Source per bet kind:
//   - group-winner acca  → product of each leg's GROUP_x_WINNER market prob
//                          (independent groups), simulator PWinGroup as fallback
//   - match-outcome acca → product of each leg's MATCH_ODDS 1X2 prob
//   - exact score        → that scoreline's CORRECT_SCORE prob
//   - tournament winner  → WINNER market prob (simulator PWinTournament fallback)
//   - top scorer         → TOP_GOALSCORER market prob
//   - finalist pair      → simulator JOINT P(both reach the final), which depends
//                          on the bracket halves and has no direct market
//
// A bet already decided short-circuits: status "won" → 1, "lost" → 0. Within an
// acca, individually "won" legs contribute a factor of 1. When a bet can't be
// priced (no market and no simulator figure) its probability is left unset and
// it's excluded from the expected payout rather than guessed.

const (
	// simIterations is the Monte Carlo sample size for the production sim.
	simIterations = 20000
	// simSeed keeps the sim deterministic between builds (stable figures).
	simSeed = 20260611
)

// ExpectedJSON is the probability-weighted counterpart to MaxPayoutJSON: what we
// expect to win on average given current market/simulator probabilities.
type ExpectedJSON struct {
	ExpectedPayout float64 `json:"expected_payout"` // Σ P(win)·return over all priced bets
	ExpectedProfit float64 `json:"expected_profit"` // expected_payout − total_outlay
}

// OddsSnapshot is the set of de-vigged markets the combiner reads, plus a
// fixture-keyed index of match odds for quick lookup.
type OddsSnapshot struct {
	Match        []MatchOdds
	Winner       []OutrightProb
	TopScorer    []OutrightProb
	GroupWinners map[string][]OutrightProb
	CorrectScore map[string]FixtureScores

	matchByKey map[string]MatchOdds
}

// index builds the fixture-keyed match-odds map. Call after populating Match.
func (snap *OddsSnapshot) index() {
	snap.matchByKey = make(map[string]MatchOdds, len(snap.Match))
	for _, mo := range snap.Match {
		if mo.Home == "" || mo.Away == "" {
			continue
		}
		k, _ := fixtureKey(mo.Home, mo.Away)
		snap.matchByKey[k] = mo
	}
}

// gatherOdds pulls every market the combiner needs from a provider. Each market
// is best-effort: a failure on one is logged and that market is left empty, so a
// flaky correct-score endpoint doesn't sink the whole likelihood pass. Returns
// false only if nothing at all could be fetched.
func gatherOdds(ctx context.Context, p OddsProvider, logger *slog.Logger) (OddsSnapshot, bool) {
	var snap OddsSnapshot
	got := false

	if m, err := p.MatchWinOdds(ctx); err != nil {
		logger.Warn("odds: match win", "err", err)
	} else {
		snap.Match = m
		got = got || len(m) > 0
	}
	if w, err := p.Outright(ctx, MarketWinner); err != nil {
		logger.Warn("odds: tournament winner", "err", err)
	} else {
		snap.Winner = w
		got = got || len(w) > 0
	}
	if ts, err := p.Outright(ctx, MarketTopGoalscorer); err != nil {
		logger.Warn("odds: top goalscorer", "err", err)
	} else {
		snap.TopScorer = ts
		got = got || len(ts) > 0
	}
	if gw, err := p.GroupWinnerOdds(ctx); err != nil {
		logger.Warn("odds: group winners", "err", err)
	} else {
		snap.GroupWinners = gw
		got = got || len(gw) > 0
	}
	if cs, err := p.CorrectScore(ctx); err != nil {
		logger.Warn("odds: correct score", "err", err)
	} else {
		snap.CorrectScore = cs
		got = got || len(cs) > 0
	}

	snap.index()
	return snap, got
}

// applyLikelihoods sets each bet's probability and expected return in place and
// fills s.Expected. sim and joints come from runTournament (joints keyed by
// canonical fixture key for the finalist pairs).
func applyLikelihoods(s *StateJSON, snap OddsSnapshot, sim map[string]TournamentSimResult, joints map[string]float64) {
	simByName := lowerSim(sim)
	var expected float64

	for i := range s.Bets {
		b := &s.Bets[i]
		p, ok := statusProb(b.Status, func() (float64, bool) {
			return groupAccaProb(b.Legs, snap, simByName)
		})
		expected += price(p, ok, b.PotentialReturn, &b.Probability, &b.ExpectedReturn)
	}

	for i := range s.MatchAccaBets {
		b := &s.MatchAccaBets[i]
		p, ok := statusProb(b.Status, func() (float64, bool) {
			return matchAccaProb(b.Legs, snap)
		})
		expected += price(p, ok, b.PotentialReturn, &b.Probability, &b.ExpectedReturn)
	}

	for i := range s.MatchResultBets {
		b := &s.MatchResultBets[i]
		p, ok := statusProb(b.Status, func() (float64, bool) {
			return snap.exactScoreProb(b.TeamA, b.TeamB, b.ScoreA, b.ScoreB)
		})
		expected += price(p, ok, b.PotentialReturn, &b.Probability, &b.ExpectedReturn)
	}

	for i := range s.TournamentWinnerBets {
		b := &s.TournamentWinnerBets[i]
		p, ok := statusProb(b.Status, func() (float64, bool) {
			if mp, ok := outrightLookup(snap.Winner, b.Team); ok {
				return mp, true
			}
			if r, ok := simByName[lowerName(b.Team)]; ok {
				return r.PWinTournament, true
			}
			return 0, false
		})
		expected += price(p, ok, b.PotentialReturn, &b.Probability, &b.ExpectedReturn)
	}

	for i := range s.TopScorerBets {
		b := &s.TopScorerBets[i]
		p, ok := statusProb(b.Status, func() (float64, bool) {
			return outrightLookup(snap.TopScorer, b.Player)
		})
		expected += price(p, ok, b.PotentialReturn, &b.Probability, &b.ExpectedReturn)
	}

	for i := range s.FinalistBets {
		b := &s.FinalistBets[i]
		p, ok := statusProb(b.Status, func() (float64, bool) {
			k, _ := fixtureKey(b.TeamA, b.TeamB)
			jp, ok := joints[k]
			return jp, ok
		})
		expected += price(p, ok, b.PotentialReturn, &b.Probability, &b.ExpectedReturn)
	}

	s.Expected = &ExpectedJSON{
		ExpectedPayout: round2(expected),
		ExpectedProfit: round2(expected - s.MaxPayout.TotalOutlay),
	}
}

// statusProb short-circuits decided bets and otherwise calls compute.
func statusProb(status string, compute func() (float64, bool)) (float64, bool) {
	switch status {
	case "won":
		return 1, true
	case "lost":
		return 0, true
	default:
		return compute()
	}
}

// price stores probability/expected-return pointers when the bet could be priced
// and returns its contribution (P·return) to the expected payout.
func price(prob float64, ok bool, ret *float64, probPtr, expPtr **float64) float64 {
	if !ok {
		return 0
	}
	p := round4(prob)
	*probPtr = &p
	if ret == nil {
		return 0
	}
	contrib := prob * *ret
	e := round2(contrib)
	*expPtr = &e
	return contrib
}

// groupAccaProb multiplies each leg's win-the-group probability: the direct
// GROUP_x_WINNER market when present, otherwise the simulator's PWinGroup. Legs
// already won contribute a factor of 1; a lost leg makes the whole acca 0.
func groupAccaProb(legs []LegJSON, snap OddsSnapshot, simByName map[string]TournamentSimResult) (float64, bool) {
	p := 1.0
	for _, leg := range legs {
		switch leg.Status {
		case "won":
			continue
		case "lost":
			return 0, true
		}
		if mp, ok := snap.groupWinnerProb(leg.Group, leg.Team); ok {
			p *= mp
		} else if r, ok := simByName[lowerName(leg.Team)]; ok {
			p *= r.PWinGroup
		} else {
			return 0, false
		}
	}
	return p, true
}

// matchAccaProb multiplies each match-outcome leg's 1X2 probability.
func matchAccaProb(legs []MatchOutcomeLegJSON, snap OddsSnapshot) (float64, bool) {
	p := 1.0
	for _, leg := range legs {
		switch leg.Status {
		case "won":
			continue
		case "lost":
			return 0, true
		}
		mp, ok := snap.matchOutcomeProb(leg.Team, leg.Opponent, leg.Outcome)
		if !ok {
			return 0, false
		}
		p *= mp
	}
	return p, true
}

func (snap OddsSnapshot) groupWinnerProb(group, team string) (float64, bool) {
	return outrightLookup(snap.GroupWinners[strings.ToUpper(group)], team)
}

// matchOutcomeProb returns P(team wins/draws/loses vs opponent) from MATCH_ODDS.
func (snap OddsSnapshot) matchOutcomeProb(team, opponent, outcome string) (float64, bool) {
	k, _ := fixtureKey(team, opponent)
	mo, ok := snap.matchByKey[k]
	if !ok {
		return 0, false
	}
	teamIsHome := teamsMatch(mo.Home, team)
	switch strings.ToLower(outcome) {
	case "win":
		if teamIsHome {
			return mo.PHome, true
		}
		return mo.PAway, true
	case "lose":
		if teamIsHome {
			return mo.PAway, true
		}
		return mo.PHome, true
	default: // draw
		return mo.PDraw, true
	}
}

// exactScoreProb returns the de-vigged probability of the exact scoreline,
// aligning the bet's team_a/team_b goals to the market's home/away. Returns
// (0, true) when the fixture market exists but doesn't separately price that
// scoreline (it's folded into an "any other" runner, i.e. ≈0).
func (snap OddsSnapshot) exactScoreProb(teamA, teamB string, scoreA, scoreB int) (float64, bool) {
	k, _ := fixtureKey(teamA, teamB)
	fs, ok := snap.CorrectScore[k]
	if !ok {
		return 0, false
	}
	var hg, ag int
	switch {
	case teamsMatch(fs.Home, teamA):
		hg, ag = scoreA, scoreB
	case teamsMatch(fs.Away, teamA):
		hg, ag = scoreB, scoreA
	default:
		return 0, false
	}
	for _, sp := range fs.Scores {
		if sp.HomeGoals == hg && sp.AwayGoals == ag {
			return sp.Prob, true
		}
	}
	return 0, true
}

func outrightLookup(probs []OutrightProb, selection string) (float64, bool) {
	for _, o := range probs {
		if teamsMatch(o.Selection, selection) {
			return o.Prob, true
		}
	}
	return 0, false
}

// lowerSim re-keys the simulator results by lower-case team name for
// case-insensitive matching against config team names.
func lowerSim(sim map[string]TournamentSimResult) map[string]TournamentSimResult {
	out := make(map[string]TournamentSimResult, len(sim))
	for name, r := range sim {
		out[lowerName(name)] = r
	}
	return out
}

func round4(f float64) float64 {
	if f < 0 {
		return -float64(int64(-f*1e4+0.5)) / 1e4
	}
	return float64(int64(f*1e4+0.5)) / 1e4
}
