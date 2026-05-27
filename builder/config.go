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
