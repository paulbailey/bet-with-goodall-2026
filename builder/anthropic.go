package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"

// anthropicClient is a hand-rolled client for the Messages API, in the same
// shape as the football-data and Betfair clients — one POST, no SDK. The daily
// summary is the only thing that calls it: it turns the structured day's moves
// into a short, readable paragraph. When no API key is configured the generator
// skips this entirely and falls back to a templated sentence.
type anthropicClient struct {
	apiKey string
	model  string
	http   *http.Client
	logger *slog.Logger
}

func newAnthropicClient(apiKey, model string, logger *slog.Logger) *anthropicClient {
	if model == "" {
		model = "claude-opus-4-8"
	}
	return &anthropicClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: 30 * time.Second},
		logger: logger,
	}
}

type anthropicReq struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system,omitempty"`
	Messages  []anthropicMsg `json:"messages"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// complete sends a single-turn request and returns the concatenated text blocks.
// Thinking is left off (the default): a two-or-three-sentence recap doesn't need
// it, and keeping it off makes the call cheap and fast — the system prompt tells
// the model to return only the paragraph so no stray reasoning leaks in.
func (c *anthropicClient) complete(ctx context.Context, system, user string) (string, error) {
	payload, err := json.Marshal(anthropicReq{
		Model:     c.model,
		MaxTokens: 600,
		System:    system,
		Messages:  []anthropicMsg{{Role: "user", Content: user}},
	})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", anthropicURL, err)
	}
	defer resp.Body.Close()

	var out anthropicResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		if out.Error != nil {
			return "", fmt.Errorf("anthropic %s: %s", out.Error.Type, out.Error.Message)
		}
		return "", fmt.Errorf("anthropic HTTP %d", resp.StatusCode)
	}

	var b strings.Builder
	for _, block := range out.Content {
		if block.Type == "text" {
			b.WriteString(block.Text)
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", fmt.Errorf("anthropic returned no text")
	}
	return text, nil
}
