package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

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
