// Package api provides HTTP client functionality for Polymarket APIs.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	// GammaAPIBaseURL is the base URL for the Polymarket Gamma API.
	GammaAPIBaseURL = "https://gamma-api.polymarket.com"
)

// GammaAPI provides access to the Polymarket Gamma API for market metadata.
type GammaAPI struct {
	client  *Client
	baseURL string
}

// GammaAPIOption configures a GammaAPI instance.
type GammaAPIOption func(*GammaAPI)

// WithGammaClient sets a custom HTTP client for the GammaAPI.
func WithGammaClient(client *Client) GammaAPIOption {
	return func(g *GammaAPI) {
		g.client = client
	}
}

// WithGammaBaseURL sets a custom base URL for the GammaAPI.
func WithGammaBaseURL(baseURL string) GammaAPIOption {
	return func(g *GammaAPI) {
		g.baseURL = baseURL
	}
}

// NewGammaAPI creates a new GammaAPI client with the given options.
func NewGammaAPI(opts ...GammaAPIOption) *GammaAPI {
	g := &GammaAPI{
		client:  NewClient(),
		baseURL: GammaAPIBaseURL,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Market represents a Polymarket prediction market.
type Market struct {
	// ID is the unique identifier for the market.
	ID string `json:"id"`
	// Question is the market question.
	Question string `json:"question"`
	// ConditionID is the condition ID used in trades and positions.
	ConditionID string `json:"conditionId"`
	// Slug is the URL-friendly market identifier.
	Slug string `json:"slug"`
	// ResolutionSource is the source used for market resolution.
	ResolutionSource string `json:"resolutionSource"`
	// EndDate is when the market ends.
	EndDate time.Time `json:"endDate"`
	// Liquidity is the total liquidity in the market.
	Liquidity float64 `json:"liquidity,string"`
	// Volume is the total trading volume.
	Volume float64 `json:"volume,string"`
	// Volume24hr is the 24-hour trading volume.
	Volume24hr float64 `json:"volume24hr,string"`
	// Active indicates if the market is currently active.
	Active bool `json:"active"`
	// Closed indicates if the market has been closed.
	Closed bool `json:"closed"`
	// MarketMakerAddress is the address of the market maker contract.
	MarketMakerAddress string `json:"marketMakerAddress"`
	// OutcomePrices is the JSON-encoded string of outcome prices.
	OutcomePrices string `json:"outcomePrices"`
	// Outcomes is the JSON-encoded string of outcome names.
	Outcomes string `json:"outcomes"`
	// ClobTokenIds is the JSON-encoded string of CLOB token IDs.
	ClobTokenIds string `json:"clobTokenIds"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Market.
func (m *Market) UnmarshalJSON(data []byte) error {
	// Intermediate struct to handle string-encoded numerics and nullable fields
	type marketJSON struct {
		ID                 string `json:"id"`
		Question           string `json:"question"`
		ConditionID        string `json:"conditionId"`
		Slug               string `json:"slug"`
		ResolutionSource   string `json:"resolutionSource"`
		EndDate            string `json:"endDate"`
		Liquidity          string `json:"liquidity"`
		Volume             string `json:"volume"`
		Volume24hr         string `json:"volume24hr"`
		Active             bool   `json:"active"`
		Closed             bool   `json:"closed"`
		MarketMakerAddress string `json:"marketMakerAddress"`
		OutcomePrices      string `json:"outcomePrices"`
		Outcomes           string `json:"outcomes"`
		ClobTokenIds       string `json:"clobTokenIds"`
	}

	var mj marketJSON
	if err := json.Unmarshal(data, &mj); err != nil {
		return fmt.Errorf("unmarshaling market: %w", err)
	}

	m.ID = mj.ID
	m.Question = mj.Question
	m.ConditionID = mj.ConditionID
	m.Slug = mj.Slug
	m.ResolutionSource = mj.ResolutionSource
	m.MarketMakerAddress = mj.MarketMakerAddress
	m.OutcomePrices = mj.OutcomePrices
	m.Outcomes = mj.Outcomes
	m.ClobTokenIds = mj.ClobTokenIds
	m.Active = mj.Active
	m.Closed = mj.Closed

	// Parse EndDate if present
	if mj.EndDate != "" {
		endDate, err := time.Parse(time.RFC3339, mj.EndDate)
		if err != nil {
			return fmt.Errorf("parsing end date: %w", err)
		}
		m.EndDate = endDate
	}

	// Parse numeric fields (may be empty strings)
	if mj.Liquidity != "" {
		liquidity, err := strconv.ParseFloat(mj.Liquidity, 64)
		if err != nil {
			return fmt.Errorf("parsing liquidity: %w", err)
		}
		m.Liquidity = liquidity
	}

	if mj.Volume != "" {
		volume, err := strconv.ParseFloat(mj.Volume, 64)
		if err != nil {
			return fmt.Errorf("parsing volume: %w", err)
		}
		m.Volume = volume
	}

	if mj.Volume24hr != "" {
		volume24hr, err := strconv.ParseFloat(mj.Volume24hr, 64)
		if err != nil {
			return fmt.Errorf("parsing volume24hr: %w", err)
		}
		m.Volume24hr = volume24hr
	}

	return nil
}

// GetOutcomePrices parses and returns the outcome prices as a slice of floats.
func (m *Market) GetOutcomePrices() ([]float64, error) {
	if m.OutcomePrices == "" {
		return nil, nil
	}

	var priceStrings []string
	if err := json.Unmarshal([]byte(m.OutcomePrices), &priceStrings); err != nil {
		return nil, fmt.Errorf("parsing outcome prices: %w", err)
	}

	prices := make([]float64, len(priceStrings))
	for i, ps := range priceStrings {
		price, err := strconv.ParseFloat(ps, 64)
		if err != nil {
			return nil, fmt.Errorf("parsing price %d: %w", i, err)
		}
		prices[i] = price
	}

	return prices, nil
}

// GetOutcomes parses and returns the outcome names as a slice of strings.
func (m *Market) GetOutcomes() ([]string, error) {
	if m.Outcomes == "" {
		return nil, nil
	}

	var outcomes []string
	if err := json.Unmarshal([]byte(m.Outcomes), &outcomes); err != nil {
		return nil, fmt.Errorf("parsing outcomes: %w", err)
	}

	return outcomes, nil
}

// MarketList is a slice of Market with helper methods.
type MarketList []*Market

// FilterActive returns only active markets.
func (ml MarketList) FilterActive() MarketList {
	var active MarketList
	for _, m := range ml {
		if m.Active && !m.Closed {
			active = append(active, m)
		}
	}
	return active
}

// FindByConditionID returns the market with the given condition ID, or nil if not found.
func (ml MarketList) FindByConditionID(conditionID string) *Market {
	for _, m := range ml {
		if m.ConditionID == conditionID {
			return m
		}
	}
	return nil
}

// MarketQueryOptions configures market query parameters.
type MarketQueryOptions struct {
	// Active filters markets by active status.
	Active *bool
	// Closed filters markets by closed status.
	Closed *bool
	// Limit is the maximum number of markets to return.
	Limit int
	// Offset is the number of markets to skip for pagination.
	Offset int
}

// toParams converts options to URL query parameters.
func (o MarketQueryOptions) toParams() map[string]string {
	params := make(map[string]string)

	if o.Active != nil {
		params["active"] = strconv.FormatBool(*o.Active)
	}
	if o.Closed != nil {
		params["closed"] = strconv.FormatBool(*o.Closed)
	}
	if o.Limit > 0 {
		params["limit"] = strconv.Itoa(o.Limit)
	}
	if o.Offset > 0 {
		params["offset"] = strconv.Itoa(o.Offset)
	}

	return params
}

// GetMarket fetches a single market by its condition ID.
func (g *GammaAPI) GetMarket(ctx context.Context, conditionID string) (*Market, error) {
	url := g.baseURL + "/markets/" + conditionID

	resp, err := g.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching market: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var market Market
	if err := json.NewDecoder(resp.Body).Decode(&market); err != nil {
		return nil, fmt.Errorf("decoding market: %w", err)
	}

	return &market, nil
}

// GetMarkets fetches all markets with default options.
func (g *GammaAPI) GetMarkets(ctx context.Context) (MarketList, error) {
	return g.GetMarketsWithOptions(ctx, MarketQueryOptions{})
}

// GetMarketsWithOptions fetches markets with custom query options.
func (g *GammaAPI) GetMarketsWithOptions(ctx context.Context, opts MarketQueryOptions) (MarketList, error) {
	url := g.baseURL + "/markets"

	params := opts.toParams()

	var resp *http.Response
	var err error
	if len(params) > 0 {
		resp, err = g.client.GetWithParams(ctx, url, params)
	} else {
		resp, err = g.client.Get(ctx, url)
	}

	if err != nil {
		return nil, fmt.Errorf("fetching markets: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var markets MarketList
	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("decoding markets: %w", err)
	}

	return markets, nil
}
