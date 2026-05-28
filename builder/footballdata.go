package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://api.football-data.org/v4"

type footballDataClient struct {
	apiKey        string
	competitionID string
	http          *http.Client
	logger        *slog.Logger
}

func newFootballDataClient(apiKey, competitionID string, logger *slog.Logger) *footballDataClient {
	return &footballDataClient{
		apiKey:        apiKey,
		competitionID: competitionID,
		http:          &http.Client{Timeout: 15 * time.Second},
		logger:        logger,
	}
}

func (c *footballDataClient) get(ctx context.Context, path string, out any) error {
	url := baseURL + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("GET %s: rate limited (HTTP 429)", url)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// ── API response shapes ───────────────────────────────────────────────────────

type fdStandingsResp struct {
	Standings []struct {
		Type  string `json:"type"`
		Group string `json:"group"`
		Table []struct {
			PlayedGames    int `json:"playedGames"`
			Won            int `json:"won"`
			Draw           int `json:"draw"`
			Lost           int `json:"lost"`
			GoalsFor       int `json:"goalsFor"`
			GoalsAgainst   int `json:"goalsAgainst"`
			GoalDifference int `json:"goalDifference"`
			Points         int `json:"points"`
			Team           struct {
				Name string `json:"name"`
			} `json:"team"`
		} `json:"table"`
	} `json:"standings"`
}

type fdMatchesResp struct {
	Matches []struct {
		UtcDate  string `json:"utcDate"`
		Status   string `json:"status"`
		Stage    string `json:"stage"`
		Group    string `json:"group"`
		HomeTeam struct {
			Name string `json:"name"`
		} `json:"homeTeam"`
		AwayTeam struct {
			Name string `json:"name"`
		} `json:"awayTeam"`
		Score struct {
			Winner   string `json:"winner"` // HOME_TEAM | AWAY_TEAM | DRAW | null
			FullTime struct {
				Home *int `json:"home"` // null until a score is published
				Away *int `json:"away"`
			} `json:"fullTime"`
		} `json:"score"`
	} `json:"matches"`
}

type fdScorersResp struct {
	Scorers []struct {
		Player struct {
			Name string `json:"name"`
		} `json:"player"`
		Team struct {
			Name string `json:"name"`
		} `json:"team"`
		Goals         int `json:"goals"`
		PlayedMatches int `json:"playedMatches"`
	} `json:"scorers"`
}

// ── Public methods ────────────────────────────────────────────────────────────

func (c *footballDataClient) GetStandings(ctx context.Context) ([]GroupStanding, error) {
	var resp fdStandingsResp
	if err := c.get(ctx, fmt.Sprintf("/competitions/%s/standings", c.competitionID), &resp); err != nil {
		return nil, err
	}

	var groups []GroupStanding
	for _, s := range resp.Standings {
		if s.Type != "TOTAL" {
			continue
		}
		// "Group A" (2026 API format) or legacy "GROUP_A" → "A"
		letter := strings.TrimPrefix(s.Group, "Group ")
		if len(letter) != 1 {
			letter = strings.TrimPrefix(s.Group, "GROUP_")
		}
		if len(letter) != 1 {
			continue
		}

		teams := make([]TeamStats, 0, len(s.Table))
		for _, row := range s.Table {
			teams = append(teams, TeamStats{
				Name:   row.Team.Name,
				Played: row.PlayedGames,
				Won:    row.Won,
				Drawn:  row.Draw,
				Lost:   row.Lost,
				GF:     row.GoalsFor,
				GA:     row.GoalsAgainst,
				GD:     row.GoalDifference,
				Points: row.Points,
			})
		}
		groups = append(groups, GroupStanding{Group: letter, Teams: teams})
	}
	return groups, nil
}

func (c *footballDataClient) GetMatches(ctx context.Context) ([]Match, error) {
	var resp fdMatchesResp
	if err := c.get(ctx, fmt.Sprintf("/competitions/%s/matches", c.competitionID), &resp); err != nil {
		return nil, err
	}

	matches := make([]Match, 0, len(resp.Matches))
	for _, m := range resp.Matches {
		t, err := time.Parse(time.RFC3339, m.UtcDate)
		if err != nil {
			c.logger.Warn("unparseable match date", "date", m.UtcDate)
			continue
		}
		matches = append(matches, Match{
			UtcDate:   t,
			Status:    m.Status,
			Stage:     m.Stage,
			Group:     m.Group,
			HomeTeam:  m.HomeTeam.Name,
			AwayTeam:  m.AwayTeam.Name,
			Winner:    m.Score.Winner,
			HomeScore: m.Score.FullTime.Home,
			AwayScore: m.Score.FullTime.Away,
		})
	}
	return matches, nil
}

func (c *footballDataClient) GetScorers(ctx context.Context) ([]TopScorerEntry, error) {
	var resp fdScorersResp
	path := fmt.Sprintf("/competitions/%s/scorers?limit=50", c.competitionID)
	if err := c.get(ctx, path, &resp); err != nil {
		return nil, err
	}

	scorers := make([]TopScorerEntry, 0, len(resp.Scorers))
	for _, s := range resp.Scorers {
		scorers = append(scorers, TopScorerEntry{
			Player: s.Player.Name,
			Team:   s.Team.Name,
			Goals:  s.Goals,
			Games:  s.PlayedMatches,
		})
	}
	return scorers, nil
}
