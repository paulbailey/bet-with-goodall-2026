package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Env struct {
	APIKey      string
	LocalOutput string // set to write to a local file instead of S3
	Bucket      string
	S3Key       string
	BetsFile    string
	Region      string
}

func loadEnv() Env {
	env := Env{
		APIKey:      mustEnv("FDB_API_KEY"),
		LocalOutput: os.Getenv("LOCAL_OUTPUT"),
		BetsFile:    getEnv("BETS_FILE", "/config/bets.yaml"),
	}
	if env.LocalOutput == "" {
		env.Bucket = mustEnv("S3_BUCKET")
		env.S3Key = getEnv("S3_KEY", "data/state.json")
		env.Region = getEnv("AWS_REGION", "eu-west-1")
	}
	return env
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "required env var %s is not set\n", key)
		os.Exit(1)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type Config struct {
	CompetitionID        string                      `yaml:"competition_id"`
	Bets                 []BetConfig                 `yaml:"bets"`
	TopScorerBets        []TopScorerBetConfig        `yaml:"top_scorer_bets"`
	TournamentWinnerBets []TournamentWinnerBetConfig `yaml:"tournament_winner_bets"`
	MatchResultBets      []MatchResultBetConfig      `yaml:"match_result_bets"`
	MatchAccaBets        []MatchAccaBetConfig        `yaml:"match_acca_bets"`
	FinalistBets         []FinalistBetConfig         `yaml:"finalist_bets"`
}

type BetConfig struct {
	ID              string      `yaml:"id"`
	Stake           float64     `yaml:"stake"`
	PotentialReturn float64     `yaml:"potential_return"`
	Legs            []LegConfig `yaml:"legs"`
}

type LegConfig struct {
	Group string `yaml:"group"`
	Team  string `yaml:"team"`
}

type TopScorerBetConfig struct {
	ID              string  `yaml:"id"`
	Player          string  `yaml:"player"`
	Team            string  `yaml:"team"`
	Stake           float64 `yaml:"stake"`
	PotentialReturn float64 `yaml:"potential_return"`
}

type TournamentWinnerBetConfig struct {
	ID              string  `yaml:"id"`
	Team            string  `yaml:"team"`
	Stake           float64 `yaml:"stake"`
	PotentialReturn float64 `yaml:"potential_return"`
}

// MatchResultBetConfig predicts an exact scoreline for a single fixture, e.g.
// "England 3-1 Ghana". Scores are matched by team identity, so the order of
// team_a/team_b here is independent of the fixture's nominal home/away side.
type MatchResultBetConfig struct {
	ID              string  `yaml:"id"`
	TeamA           string  `yaml:"team_a"`
	TeamB           string  `yaml:"team_b"`
	ScoreA          int     `yaml:"score_a"`
	ScoreB          int     `yaml:"score_b"`
	Stake           float64 `yaml:"stake"`
	PotentialReturn float64 `yaml:"potential_return"`
}

// MatchAccaBetConfig is an accumulator whose legs each predict the outcome of
// one fixture, e.g. "England to win all three group games".
type MatchAccaBetConfig struct {
	ID              string                  `yaml:"id"`
	Stake           float64                 `yaml:"stake"`
	PotentialReturn float64                 `yaml:"potential_return"`
	Legs            []MatchOutcomeLegConfig `yaml:"legs"`
}

type MatchOutcomeLegConfig struct {
	Team     string `yaml:"team"`
	Opponent string `yaml:"opponent"`
	Outcome  string `yaml:"outcome"` // win | draw | lose
}

// FinalistBetConfig predicts both teams that reach the final, e.g. "France vs
// Spain". Won once both teams are confirmed finalists, regardless of who wins.
type FinalistBetConfig struct {
	ID              string  `yaml:"id"`
	TeamA           string  `yaml:"team_a"`
	TeamB           string  `yaml:"team_b"`
	Stake           float64 `yaml:"stake"`
	PotentialReturn float64 `yaml:"potential_return"`
}

func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return &cfg, nil
}
