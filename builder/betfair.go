package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// betfairClient implements OddsProvider against the Betfair Exchange API. It
// reads market-implied probabilities from the betting exchange (de-vigged best
// back prices) for the FIFA World Cup competition.
//
// Auth: an interactive login mints a session token used as X-Authentication on
// every call; KeepAlive extends it. Running unattended on k8s will eventually
// want Betfair's certificate (bot) login instead — same token semantics, just a
// different login endpoint — which can slot in behind Login.
type betfairClient struct {
	appKey        string
	token         string
	competitionID string

	apiURL       string
	loginURL     string
	keepAliveURL string

	http   *http.Client
	logger *slog.Logger
}

const (
	betfairAPIURL       = "https://api.betfair.com/exchange/betting/json-rpc/v1"
	betfairLoginURL     = "https://identitysso.betfair.com/api/login"
	betfairKeepAliveURL = "https://identitysso.betfair.com/api/keepAlive"

	// The FIFA World Cup competition id on the Betfair Exchange (confirmed via
	// the market probe). Overridable, and re-resolvable by name via
	// ResolveCompetition for future editions.
	betfairWorldCupCompetitionID = "12469077"
	betfairSoccerEventTypeID     = "1"

	// Betfair sits behind Cloudflare, which resets connections from clients
	// sending the default Go user-agent; use a conventional one.
	betfairUserAgent = "bet-with-goodall/1.0"
)

func newBetfairClient(appKey string, logger *slog.Logger) *betfairClient {
	return &betfairClient{
		appKey:        appKey,
		competitionID: betfairWorldCupCompetitionID,
		apiURL:        betfairAPIURL,
		loginURL:      betfairLoginURL,
		keepAliveURL:  betfairKeepAliveURL,
		http:          &http.Client{Timeout: 20 * time.Second},
		logger:        logger,
	}
}

// UseToken sets a pre-minted session token (e.g. from an out-of-band login).
func (c *betfairClient) UseToken(token string) { c.token = token }

// Login performs an interactive username/password login and stores the session
// token. Fails for accounts with two-factor auth (which require cert login).
func (c *betfairClient) Login(ctx context.Context, username, password string) error {
	form := url.Values{"username": {username}, "password": {password}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.loginURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("X-Application", c.appKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", betfairUserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("betfair login: %w", err)
	}
	defer resp.Body.Close()

	var body struct {
		Token  string `json:"token"`
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("betfair login decode: %w", err)
	}
	if body.Status != "SUCCESS" {
		return fmt.Errorf("betfair login failed: status=%s error=%s", body.Status, body.Error)
	}
	c.token = body.Token
	return nil
}

// KeepAlive extends the current session token's lifetime.
func (c *betfairClient) KeepAlive(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.keepAliveURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Application", c.appKey)
	req.Header.Set("X-Authentication", c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", betfairUserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("betfair keepAlive: %w", err)
	}
	defer resp.Body.Close()
	var body struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("betfair keepAlive decode: %w", err)
	}
	if body.Status != "SUCCESS" {
		return fmt.Errorf("betfair keepAlive failed: status=%s error=%s", body.Status, body.Error)
	}
	return nil
}

// ── JSON-RPC plumbing ─────────────────────────────────────────────────────────

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
	ID      int    `json:"id"`
}

// rpc calls one SportsAPING method and unmarshals its result into out.
func (c *betfairClient) rpc(ctx context.Context, method string, params any, out any) error {
	payload := []rpcRequest{{JSONRPC: "2.0", Method: "SportsAPING/v1.0/" + method, Params: params, ID: 1}}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("X-Application", c.appKey)
	req.Header.Set("X-Authentication", c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", betfairUserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", method, resp.StatusCode)
	}

	var envs []struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return fmt.Errorf("%s decode: %w", method, err)
	}
	if len(envs) == 0 {
		return fmt.Errorf("%s: empty response", method)
	}
	if envs[0].Error != nil {
		return fmt.Errorf("%s: API error %d %s", method, envs[0].Error.Code, envs[0].Error.Message)
	}
	return json.Unmarshal(envs[0].Result, out)
}

type marketFilter struct {
	EventTypeIds    []string `json:"eventTypeIds,omitempty"`
	CompetitionIds  []string `json:"competitionIds,omitempty"`
	MarketTypeCodes []string `json:"marketTypeCodes,omitempty"`
	TextQuery       string   `json:"textQuery,omitempty"`
}

type catalogMarket struct {
	MarketID string `json:"marketId"`
	Event    struct {
		Name string `json:"name"`
	} `json:"event"`
	Runners []struct {
		SelectionID int64  `json:"selectionId"`
		RunnerName  string `json:"runnerName"`
	} `json:"runners"`
}

// listMarketCatalogue fetches markets (ids, event, runner names) for a filter.
// MARKET_DESCRIPTION is deliberately not requested — its rules HTML is heavy and
// trips the API's data-volume cap.
func (c *betfairClient) listMarketCatalogue(ctx context.Context, filter marketFilter, maxResults int) ([]catalogMarket, error) {
	params := map[string]any{
		"filter":           filter,
		"marketProjection": []string{"RUNNER_DESCRIPTION", "EVENT"},
		"maxResults":       maxResults,
		"sort":             "FIRST_TO_START",
	}
	var out []catalogMarket
	if err := c.rpc(ctx, "listMarketCatalogue", params, &out); err != nil {
		return nil, err
	}
	return out, nil
}

type bookRunner struct {
	SelectionID     int64   `json:"selectionId"`
	Status          string  `json:"status"`
	LastPriceTraded float64 `json:"lastPriceTraded"`
	Ex              struct {
		AvailableToBack []struct {
			Price float64 `json:"price"`
			Size  float64 `json:"size"`
		} `json:"availableToBack"`
	} `json:"ex"`
}

type bookMarket struct {
	MarketID string       `json:"marketId"`
	Status   string       `json:"status"`
	Runners  []bookRunner `json:"runners"`
}

// listMarketBook fetches best-back prices for the given market ids, chunked to
// stay under the API's per-request weight limit.
func (c *betfairClient) listMarketBook(ctx context.Context, marketIDs []string) (map[string]bookMarket, error) {
	out := make(map[string]bookMarket, len(marketIDs))
	const chunk = 25
	for i := 0; i < len(marketIDs); i += chunk {
		end := i + chunk
		if end > len(marketIDs) {
			end = len(marketIDs)
		}
		params := map[string]any{
			"marketIds":       marketIDs[i:end],
			"priceProjection": map[string]any{"priceData": []string{"EX_BEST_OFFERS"}},
		}
		var books []bookMarket
		if err := c.rpc(ctx, "listMarketBook", params, &books); err != nil {
			return nil, err
		}
		for _, b := range books {
			out[b.MarketID] = b
		}
	}
	return out, nil
}

// bestBack returns the best available back price for a runner, falling back to
// the last traded price, or 0 if neither is available.
func (r bookRunner) bestBack() float64 {
	if len(r.Ex.AvailableToBack) > 0 && r.Ex.AvailableToBack[0].Price > 0 {
		return r.Ex.AvailableToBack[0].Price
	}
	return r.LastPriceTraded
}

// devigBook turns a market's runner prices into fair probabilities keyed by
// selectionId. Only ACTIVE runners with a usable price contribute.
func devigBook(b bookMarket) map[int64]float64 {
	ids := make([]int64, 0, len(b.Runners))
	odds := make([]float64, 0, len(b.Runners))
	for _, r := range b.Runners {
		if r.Status != "" && r.Status != "ACTIVE" {
			continue
		}
		if p := r.bestBack(); p > 0 {
			ids = append(ids, r.SelectionID)
			odds = append(odds, p)
		}
	}
	fair := devig(odds)
	out := make(map[int64]float64, len(ids))
	for i, id := range ids {
		out[id] = fair[i]
	}
	return out
}

// ── OddsProvider implementation ───────────────────────────────────────────────

func (c *betfairClient) wcFilter(types ...string) marketFilter {
	return marketFilter{CompetitionIds: []string{c.competitionID}, MarketTypeCodes: types}
}

// MatchWinOdds returns de-vigged 1X2 probabilities for every MATCH_ODDS market.
func (c *betfairClient) MatchWinOdds(ctx context.Context) ([]MatchOdds, error) {
	cat, err := c.listMarketCatalogue(ctx, c.wcFilter("MATCH_ODDS"), 200)
	if err != nil {
		return nil, err
	}
	ids := marketIDs(cat)
	books, err := c.listMarketBook(ctx, ids)
	if err != nil {
		return nil, err
	}

	var out []MatchOdds
	for _, m := range cat {
		book, ok := books[m.MarketID]
		if !ok {
			continue
		}
		probs := devigBook(book)
		home, away := splitFixture(m.Event.Name)
		mo := MatchOdds{Home: home, Away: away}
		for _, r := range m.Runners {
			p := probs[r.SelectionID]
			switch {
			case isDrawRunner(r.RunnerName):
				mo.PDraw = p
			case teamsMatch(r.RunnerName, home):
				mo.PHome = p
			case teamsMatch(r.RunnerName, away):
				mo.PAway = p
			default:
				// Event name didn't parse cleanly; fall back to filling the
				// first empty team slot so probabilities aren't lost.
				if mo.Home == "" {
					mo.Home, mo.PHome = r.RunnerName, p
				} else if mo.Away == "" {
					mo.Away, mo.PAway = r.RunnerName, p
				}
			}
		}
		out = append(out, mo)
	}
	return out, nil
}

// Outright returns de-vigged selections for a tournament-wide market.
func (c *betfairClient) Outright(ctx context.Context, market OutrightMarket) ([]OutrightProb, error) {
	mt, ok := betfairMarketType(market)
	if !ok {
		return nil, fmt.Errorf("unsupported outright market %q", market)
	}
	cat, err := c.listMarketCatalogue(ctx, c.wcFilter(mt), 5)
	if err != nil {
		return nil, err
	}
	if len(cat) == 0 {
		return nil, nil
	}
	// Some market types (e.g. TOP_GOALSCORER) have several variants; use the one
	// with the most runners, which is the main outright.
	main := cat[0]
	for _, m := range cat[1:] {
		if len(m.Runners) > len(main.Runners) {
			main = m
		}
	}
	books, err := c.listMarketBook(ctx, []string{main.MarketID})
	if err != nil {
		return nil, err
	}
	return outrightProbs(main, books[main.MarketID]), nil
}

// GroupWinnerOdds returns de-vigged win-the-group probabilities per group letter.
func (c *betfairClient) GroupWinnerOdds(ctx context.Context) (map[string][]OutrightProb, error) {
	out := make(map[string][]OutrightProb)
	for _, letter := range []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L"} {
		mt := "GROUP_" + letter + "_WINNER"
		cat, err := c.listMarketCatalogue(ctx, c.wcFilter(mt), 1)
		if err != nil {
			return nil, err
		}
		if len(cat) == 0 {
			continue
		}
		books, err := c.listMarketBook(ctx, []string{cat[0].MarketID})
		if err != nil {
			return nil, err
		}
		out[letter] = outrightProbs(cat[0], books[cat[0].MarketID])
	}
	return out, nil
}

// CorrectScore returns exact-scoreline probabilities keyed by canonical fixture
// key. Non-numeric runners (e.g. "Any Other Home Win") are kept in the de-vig
// denominator but dropped from the output.
func (c *betfairClient) CorrectScore(ctx context.Context) (map[string][]ScoreProb, error) {
	cat, err := c.listMarketCatalogue(ctx, c.wcFilter("CORRECT_SCORE"), 200)
	if err != nil {
		return nil, err
	}
	books, err := c.listMarketBook(ctx, marketIDs(cat))
	if err != nil {
		return nil, err
	}

	out := make(map[string][]ScoreProb)
	for _, m := range cat {
		book, ok := books[m.MarketID]
		if !ok {
			continue
		}
		home, away := splitFixture(m.Event.Name)
		key, _ := fixtureKey(home, away)
		probs := devigBook(book)
		var scores []ScoreProb
		for _, r := range m.Runners {
			hg, ag, ok := parseScoreRunner(r.RunnerName)
			if !ok {
				continue
			}
			scores = append(scores, ScoreProb{HomeGoals: hg, AwayGoals: ag, Prob: probs[r.SelectionID]})
		}
		if len(scores) > 0 {
			out[key] = scores
		}
	}
	return out, nil
}

// ResolveCompetition looks up the World Cup competition id by name, in case the
// hardcoded default goes stale for a future edition.
func (c *betfairClient) ResolveCompetition(ctx context.Context) error {
	params := map[string]any{"filter": marketFilter{EventTypeIds: []string{betfairSoccerEventTypeID}, TextQuery: "FIFA World Cup"}}
	var out []struct {
		Competition struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"competition"`
	}
	if err := c.rpc(ctx, "listCompetitions", params, &out); err != nil {
		return err
	}
	for _, comp := range out {
		if strings.EqualFold(comp.Competition.Name, "FIFA World Cup") {
			c.competitionID = comp.Competition.ID
			return nil
		}
	}
	return fmt.Errorf("FIFA World Cup competition not found")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func betfairMarketType(m OutrightMarket) (string, bool) {
	switch m {
	case MarketWinner:
		return "WINNER", true
	case MarketToReachFinal:
		return "TO_REACH_FINAL", true
	case MarketTopGoalscorer:
		return "TOP_GOALSCORER", true
	}
	return "", false
}

func marketIDs(cat []catalogMarket) []string {
	ids := make([]string, len(cat))
	for i, m := range cat {
		ids[i] = m.MarketID
	}
	return ids
}

func outrightProbs(m catalogMarket, book bookMarket) []OutrightProb {
	probs := devigBook(book)
	out := make([]OutrightProb, 0, len(m.Runners))
	for _, r := range m.Runners {
		if p, ok := probs[r.SelectionID]; ok {
			out = append(out, OutrightProb{Selection: r.RunnerName, Prob: p})
		}
	}
	return out
}

// splitFixture parses a Betfair event name like "Mexico v South Africa".
func splitFixture(eventName string) (home, away string) {
	parts := strings.SplitN(eventName, " v ", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(eventName), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func isDrawRunner(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "The Draw") || strings.EqualFold(strings.TrimSpace(name), "Draw")
}

// parseScoreRunner parses a correct-score runner name like "3 - 1". Returns
// false for catch-all selections such as "Any Other Home Win".
func parseScoreRunner(name string) (home, away int, ok bool) {
	parts := strings.Split(name, " - ")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	a, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return h, a, true
}
