package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// oddsRefreshInterval is how often the (slow-moving) Betfair markets are
// re-fetched and team strengths re-calibrated. The simulator itself re-runs
// every cycle so finalist odds stay conditioned on the latest results.
const oddsRefreshInterval = 15 * time.Minute

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	env := loadEnv()

	// When running locally, default the bets file to the repo config directory
	// so you can run `go run .` from builder/ without any extra flags.
	betsFile := env.BetsFile
	if betsFile == "/config/bets.yaml" {
		if _, err := os.Stat(betsFile); os.IsNotExist(err) {
			betsFile = "../config/bets.yaml"
		}
	}

	cfg, err := loadConfig(betsFile)
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}
	logger.Info("config loaded",
		"bets", len(cfg.Bets),
		"top_scorer_bets", len(cfg.TopScorerBets),
		"tournament_winner_bets", len(cfg.TournamentWinnerBets),
		"match_result_bets", len(cfg.MatchResultBets),
		"match_acca_bets", len(cfg.MatchAccaBets),
		"finalist_bets", len(cfg.FinalistBets),
	)

	provider := newFootballDataClient(env.APIKey, cfg.CompetitionID, logger)

	var writer stateWriter
	if env.LocalOutput != "" {
		logger.Info("local mode — writing to file", "path", env.LocalOutput)
		writer = newFileWriter(env.LocalOutput, logger)
	} else {
		w, err := newS3Uploader(env.Region, env.Bucket, env.S3Key, logger)
		if err != nil {
			logger.Error("failed to create S3 uploader", "err", err)
			os.Exit(1)
		}
		writer = w
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	betfair := setupBetfair(ctx, env, logger)
	pairs := finalistPairs(cfg)
	var bundle *oddsBundle

	var matches []Match

	for {
		newMatches, err := provider.GetMatches(ctx)
		if err != nil {
			logger.Warn("fetch matches failed", "err", err)
		} else {
			matches = newMatches
		}

		standings, err := provider.GetStandings(ctx)
		if err != nil {
			logger.Warn("fetch standings failed", "err", err)
		}

		scorers, err := provider.GetScorers(ctx)
		if err != nil {
			logger.Warn("fetch scorers failed", "err", err)
		}

		if standings != nil {
			state := buildState(cfg, standings, scorers, matches)

			if betfair != nil {
				if bundle == nil || time.Since(bundle.fetched) > oddsRefreshInterval {
					if nb := refreshOdds(ctx, betfair, standings, matches, logger); nb != nil {
						bundle = nb
					}
				}
				if bundle != nil {
					sim, joints := runTournament(standings, matches, bundle.strength, simIterations, simSeed, pairs)
					applyLikelihoods(&state, bundle.snap, sim, joints)
					logger.Info("likelihoods applied",
						"expected_payout", state.Expected.ExpectedPayout,
						"expected_profit", state.Expected.ExpectedProfit,
					)
				}
			}

			if err := writer.Write(ctx, state); err != nil {
				logger.Error("write failed", "err", err)
			}
		}

		sleep := nextPollInterval(matches)
		logger.Info("sleeping until next poll", "duration", sleep.Round(time.Second))

		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received, exiting")
			return
		case <-time.After(sleep):
		}
	}
}

// oddsBundle caches the slow-moving market snapshot and the team strengths
// calibrated from it, refreshed on oddsRefreshInterval.
type oddsBundle struct {
	snap     OddsSnapshot
	strength map[string]float64
	fetched  time.Time
}

// setupBetfair builds a Betfair odds client when an app key is configured and a
// session can be established. Auth precedence: a pre-minted token, else
// certificate (bot) login when cert+key+credentials are present, else
// interactive login. Returns nil to run results-only; any auth failure is logged
// and degrades gracefully.
func setupBetfair(ctx context.Context, env Env, logger *slog.Logger) *betfairClient {
	if env.BetfairAppKey == "" {
		logger.Info("betfair odds disabled (no BETFAIR_APP_KEY)")
		return nil
	}
	c := newBetfairClient(env.BetfairAppKey, logger)
	haveCreds := env.BetfairUsername != "" && env.BetfairPassword != ""
	haveCert := env.BetfairCertFile != "" && env.BetfairKeyFile != ""
	switch {
	case env.BetfairToken != "":
		c.UseToken(env.BetfairToken)
		logger.Info("betfair auth: session token")
	case haveCert && haveCreds:
		if err := c.CertLogin(ctx, env.BetfairUsername, env.BetfairPassword, env.BetfairCertFile, env.BetfairKeyFile); err != nil {
			logger.Warn("betfair cert login failed; odds disabled", "err", err)
			return nil
		}
		logger.Info("betfair auth: certificate login")
	case haveCreds:
		if err := c.Login(ctx, env.BetfairUsername, env.BetfairPassword); err != nil {
			logger.Warn("betfair login failed; odds disabled", "err", err)
			return nil
		}
		logger.Info("betfair auth: interactive login")
	default:
		logger.Warn("BETFAIR_APP_KEY set but no session token or credentials; odds disabled")
		return nil
	}
	logger.Info("betfair odds enabled")
	return c
}

// refreshOdds extends the session, fetches the markets, and re-calibrates team
// strengths against the WINNER market. Returns nil if no market data could be
// gathered (keeping any previously cached bundle in use).
func refreshOdds(ctx context.Context, bf *betfairClient, groups []GroupStanding, matches []Match, logger *slog.Logger) *oddsBundle {
	if err := bf.KeepAlive(ctx); err != nil {
		logger.Warn("betfair keepAlive failed", "err", err)
	}
	snap, ok := gatherOdds(ctx, bf, logger)
	if !ok {
		logger.Warn("no betfair markets gathered; keeping previous odds")
		return nil
	}
	strength := deriveStrength(groups, snap.Winner, matches, simSeed)
	logger.Info("odds refreshed",
		"match_markets", len(snap.Match),
		"winner_selections", len(snap.Winner),
		"group_markets", len(snap.GroupWinners),
		"correct_score_fixtures", len(snap.CorrectScore),
		"rated_teams", len(strength),
	)
	return &oddsBundle{snap: snap, strength: strength, fetched: time.Now()}
}

// finalistPairs extracts the configured finalist bets' team pairs so the
// simulator can estimate their joint reach-the-final probabilities.
func finalistPairs(cfg *Config) [][2]string {
	out := make([][2]string, 0, len(cfg.FinalistBets))
	for _, b := range cfg.FinalistBets {
		out = append(out, [2]string{b.TeamA, b.TeamB})
	}
	return out
}
