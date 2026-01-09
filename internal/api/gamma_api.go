// Package api provides HTTP client functionality for Polymarket APIs.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
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
		ID                 string            `json:"id"`
		Question           string            `json:"question"`
		ConditionID        string            `json:"conditionId"`
		Slug               string            `json:"slug"`
		ResolutionSource   string            `json:"resolutionSource"`
		EndDate            string            `json:"endDate"`
		Liquidity          models.FlexString `json:"liquidity"`
		Volume             models.FlexString `json:"volume"`
		Volume24hr         models.FlexString `json:"volume24hr"`
		Active             bool              `json:"active"`
		Closed             bool              `json:"closed"`
		MarketMakerAddress string            `json:"marketMakerAddress"`
		OutcomePrices      string            `json:"outcomePrices"`
		Outcomes           string            `json:"outcomes"`
		ClobTokenIds       string            `json:"clobTokenIds"`
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
	if !mj.Liquidity.IsEmpty() {
		liquidity, err := strconv.ParseFloat(mj.Liquidity.String(), 64)
		if err != nil {
			return fmt.Errorf("parsing liquidity: %w", err)
		}
		m.Liquidity = liquidity
	}

	if !mj.Volume.IsEmpty() {
		volume, err := strconv.ParseFloat(mj.Volume.String(), 64)
		if err != nil {
			return fmt.Errorf("parsing volume: %w", err)
		}
		m.Volume = volume
	}

	if !mj.Volume24hr.IsEmpty() {
		volume24hr, err := strconv.ParseFloat(mj.Volume24hr.String(), 64)
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

// filterByThresholds returns markets that meet all specified minimum thresholds.
// Zero values for thresholds are ignored (no filtering applied for that field).
func (ml MarketList) filterByThresholds(minVolume, minVolume24hr, minLiquidity float64) MarketList {
	var filtered MarketList
	for _, m := range ml {
		if minVolume > 0 && m.Volume < minVolume {
			continue
		}
		if minVolume24hr > 0 && m.Volume24hr < minVolume24hr {
			continue
		}
		if minLiquidity > 0 && m.Liquidity < minLiquidity {
			continue
		}
		filtered = append(filtered, m)
	}
	return filtered
}

// sortByField sorts the market list by the specified field in descending order.
// Valid fields: "volume", "liquidity", "volume24hr". Defaults to volume if invalid.
func (ml MarketList) sortByField(field string) {
	sort.Slice(ml, func(i, j int) bool {
		switch field {
		case "liquidity":
			return ml[i].Liquidity > ml[j].Liquidity
		case "volume24hr":
			return ml[i].Volume24hr > ml[j].Volume24hr
		default: // "volume" or any other value
			return ml[i].Volume > ml[j].Volume
		}
	})
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

// TopMarketsOptions configures options for fetching top markets by various metrics.
type TopMarketsOptions struct {
	// SortBy specifies the field to sort markets by.
	// Valid values: "volume", "liquidity", "volume24hr".
	SortBy string
	// MinVolume is the minimum total volume threshold for filtering markets.
	MinVolume float64
	// MinVolume24hr is the minimum 24-hour volume threshold for filtering markets.
	MinVolume24hr float64
	// MinLiquidity is the minimum liquidity threshold for filtering markets.
	MinLiquidity float64
	// Limit is the maximum number of markets to return.
	Limit int
}

// toParams converts options to URL query parameters.
func (o TopMarketsOptions) toParams() map[string]string {
	params := make(map[string]string)

	if o.SortBy != "" {
		params["sort_by"] = o.SortBy
	}
	if o.MinVolume > 0 {
		params["min_volume"] = strconv.FormatFloat(o.MinVolume, 'f', -1, 64)
	}
	if o.MinVolume24hr > 0 {
		params["min_volume_24hr"] = strconv.FormatFloat(o.MinVolume24hr, 'f', -1, 64)
	}
	if o.MinLiquidity > 0 {
		params["min_liquidity"] = strconv.FormatFloat(o.MinLiquidity, 'f', -1, 64)
	}
	if o.Limit > 0 {
		params["limit"] = strconv.Itoa(o.Limit)
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

// GetTopMarkets fetches markets, filters by thresholds, sorts by specified field, and returns top N.
// It only returns active, non-closed markets. The sorting is always descending (highest values first).
// If SortBy is empty, defaults to sorting by volume.
func (g *GammaAPI) GetTopMarkets(ctx context.Context, opts TopMarketsOptions) (MarketList, error) {
	// Fetch active, non-closed markets directly from API
	active := true
	closed := false
	markets, err := g.GetMarketsWithOptions(ctx, MarketQueryOptions{
		Active: &active,
		Closed: &closed,
	})
	if err != nil {
		return nil, err
	}

	// Apply min thresholds
	filtered := markets.filterByThresholds(opts.MinVolume, opts.MinVolume24hr, opts.MinLiquidity)

	// Determine sort field (default to volume)
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "volume"
	}

	// Sort by specified field (descending)
	filtered.sortByField(sortBy)

	// Apply limit
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}

	return filtered, nil
}

// GetMarketBySlug fetches a market by its URL-friendly slug.
// Returns the first match if found, or an error if no market matches the slug.
func (g *GammaAPI) GetMarketBySlug(ctx context.Context, slug string) (*Market, error) {
	if slug == "" {
		return nil, fmt.Errorf("slug cannot be empty")
	}

	url := g.baseURL + "/markets"
	params := map[string]string{"slug": slug}

	resp, err := g.client.GetWithParams(ctx, url, params)
	if err != nil {
		return nil, fmt.Errorf("fetching market by slug: %w", err)
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

	if len(markets) == 0 {
		return nil, fmt.Errorf("market not found for slug: %s", slug)
	}

	return markets[0], nil
}
