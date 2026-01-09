// Package api provides HTTP client functionality for Polymarket APIs.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

const (
	// DataAPIBaseURL is the base URL for the Polymarket Data API.
	DataAPIBaseURL = "https://data-api.polymarket.com"
	// DefaultPageSize is the default number of items per API request.
	DefaultPageSize = 100
)

// DataAPI provides access to the Polymarket Data API.
type DataAPI struct {
	client  *Client
	baseURL string
}

// DataAPIOption configures a DataAPI instance.
type DataAPIOption func(*DataAPI)

// WithClient sets a custom HTTP client for the DataAPI.
func WithClient(client *Client) DataAPIOption {
	return func(d *DataAPI) {
		d.client = client
	}
}

// WithBaseURL sets a custom base URL for the DataAPI.
func WithBaseURL(baseURL string) DataAPIOption {
	return func(d *DataAPI) {
		d.baseURL = baseURL
	}
}

// NewDataAPI creates a new DataAPI client with the given options.
func NewDataAPI(opts ...DataAPIOption) *DataAPI {
	d := &DataAPI{
		client:  NewClient(),
		baseURL: DataAPIBaseURL,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// TradeQueryOptions configures trade query parameters.
type TradeQueryOptions struct {
	// Limit is the maximum number of trades to return (default 100).
	Limit int
	// Offset is the number of trades to skip for pagination.
	Offset int
	// Market filters trades by market/condition ID.
	Market string
	// Side filters trades by side ("BUY" or "SELL").
	Side string
}

// toParams converts options to URL query parameters.
func (o TradeQueryOptions) toParams() map[string]string {
	params := make(map[string]string)

	if o.Limit > 0 {
		params["limit"] = strconv.Itoa(o.Limit)
	}
	if o.Offset > 0 {
		params["offset"] = strconv.Itoa(o.Offset)
	}
	if o.Market != "" {
		params["market"] = o.Market
	}
	if o.Side != "" {
		params["side"] = o.Side
	}

	return params
}

// PositionQueryOptions configures position query parameters.
type PositionQueryOptions struct {
	// Limit is the maximum number of positions to return (default 100).
	Limit int
	// Offset is the number of positions to skip for pagination.
	Offset int
	// SizeThreshold filters positions with size greater than this value.
	SizeThreshold float64
}

// toParams converts options to URL query parameters.
func (o PositionQueryOptions) toParams() map[string]string {
	params := make(map[string]string)

	if o.Limit > 0 {
		params["limit"] = strconv.Itoa(o.Limit)
	}
	if o.Offset > 0 {
		params["offset"] = strconv.Itoa(o.Offset)
	}
	if o.SizeThreshold > 0 {
		params["sizeThreshold"] = strconv.FormatFloat(o.SizeThreshold, 'f', -1, 64)
	}

	return params
}

// HolderQueryOptions configures holder query parameters.
type HolderQueryOptions struct {
	// Markets is a list of market/condition IDs to fetch holders for.
	Markets []string
	// Limit is the maximum number of holders to return per market (default 20, max 20).
	Limit int
	// MinBalance filters holders with balance greater than or equal to this value.
	MinBalance float64
}

// toParams converts options to URL query parameters.
func (o HolderQueryOptions) toParams() map[string]string {
	params := make(map[string]string)

	if len(o.Markets) > 0 {
		params["market"] = strings.Join(o.Markets, ",")
	}
	if o.Limit > 0 {
		params["limit"] = strconv.Itoa(o.Limit)
	}
	if o.MinBalance > 0 {
		params["minBalance"] = strconv.FormatFloat(o.MinBalance, 'f', -1, 64)
	}

	return params
}

// GetTrades fetches trades for a wallet address with default options.
func (d *DataAPI) GetTrades(ctx context.Context, wallet string) (models.TradeList, error) {
	return d.GetTradesWithOptions(ctx, wallet, TradeQueryOptions{Limit: DefaultPageSize})
}

// GetTradesWithOptions fetches trades for a wallet address with custom options.
func (d *DataAPI) GetTradesWithOptions(ctx context.Context, wallet string, opts TradeQueryOptions) (models.TradeList, error) {
	url := d.baseURL + "/trades"

	params := opts.toParams()
	params["user"] = wallet

	resp, err := d.client.GetWithParams(ctx, url, params)
	if err != nil {
		return nil, fmt.Errorf("fetching trades: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var trades models.TradeList
	if err := json.NewDecoder(resp.Body).Decode(&trades); err != nil {
		return nil, fmt.Errorf("decoding trades: %w", err)
	}

	return trades, nil
}

// GetAllTrades fetches all trades for a wallet address using pagination.
// This method handles pagination automatically to retrieve all available trades.
func (d *DataAPI) GetAllTrades(ctx context.Context, wallet string) (models.TradeList, error) {
	var allTrades models.TradeList
	offset := 0

	for {
		opts := TradeQueryOptions{
			Limit:  DefaultPageSize,
			Offset: offset,
		}

		trades, err := d.GetTradesWithOptions(ctx, wallet, opts)
		if err != nil {
			return nil, err
		}

		allTrades = append(allTrades, trades...)

		// If we got fewer trades than the page size, we've reached the end
		if len(trades) < DefaultPageSize {
			break
		}

		offset += DefaultPageSize
	}

	return allTrades, nil
}

// GetPositions fetches positions for a wallet address with default options.
func (d *DataAPI) GetPositions(ctx context.Context, wallet string) (models.PositionList, error) {
	return d.GetPositionsWithOptions(ctx, wallet, PositionQueryOptions{Limit: DefaultPageSize})
}

// GetPositionsWithOptions fetches positions for a wallet address with custom options.
func (d *DataAPI) GetPositionsWithOptions(ctx context.Context, wallet string, opts PositionQueryOptions) (models.PositionList, error) {
	url := d.baseURL + "/positions"

	params := opts.toParams()
	params["user"] = wallet

	resp, err := d.client.GetWithParams(ctx, url, params)
	if err != nil {
		return nil, fmt.Errorf("fetching positions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var positions models.PositionList
	if err := json.NewDecoder(resp.Body).Decode(&positions); err != nil {
		return nil, fmt.Errorf("decoding positions: %w", err)
	}

	return positions, nil
}

// GetAllPositions fetches all positions for a wallet address using pagination.
// This method handles pagination automatically to retrieve all available positions.
func (d *DataAPI) GetAllPositions(ctx context.Context, wallet string) (models.PositionList, error) {
	var allPositions models.PositionList
	offset := 0

	for {
		opts := PositionQueryOptions{
			Limit:  DefaultPageSize,
			Offset: offset,
		}

		positions, err := d.GetPositionsWithOptions(ctx, wallet, opts)
		if err != nil {
			return nil, err
		}

		allPositions = append(allPositions, positions...)

		// If we got fewer positions than the page size, we've reached the end
		if len(positions) < DefaultPageSize {
			break
		}

		offset += DefaultPageSize
	}

	return allPositions, nil
}

// Holder represents a wallet holding a position in a market.
type Holder struct {
	// ProxyWallet is the wallet address holding the position.
	ProxyWallet string `json:"proxyWallet"`
	// Amount is the position size held by this wallet.
	Amount float64 `json:"amount"`
	// Name is the optional display name for this wallet.
	Name string `json:"name,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Holder to handle
// string-encoded numeric amounts from the Polymarket API.
func (h *Holder) UnmarshalJSON(data []byte) error {
	// Intermediate struct with Amount as string
	type holderRaw struct {
		ProxyWallet string  `json:"proxyWallet"`
		Amount      string  `json:"amount"`
		Name        *string `json:"name"`
	}

	var raw holderRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	h.ProxyWallet = raw.ProxyWallet
	if raw.Name != nil {
		h.Name = *raw.Name
	}

	if raw.Amount != "" {
		amount, err := strconv.ParseFloat(raw.Amount, 64)
		if err != nil {
			return fmt.Errorf("parsing amount: %w", err)
		}
		h.Amount = amount
	}

	return nil
}

// HolderToken represents a market token and its top holders.
// The Polymarket /holders endpoint returns data grouped by token.
type HolderToken struct {
	// Token is the token ID (asset ID) for this outcome.
	Token string `json:"token"`
	// Holders is the list of wallets holding this token.
	Holders []Holder `json:"holders"`
}

// HolderList is a slice of HolderToken with helper methods.
type HolderList []HolderToken

// GetUniqueWallets returns a deduplicated list of all wallet addresses across all tokens.
func (hl HolderList) GetUniqueWallets() []string {
	seen := make(map[string]bool)
	var wallets []string

	for _, ht := range hl {
		for _, h := range ht.Holders {
			if !seen[h.ProxyWallet] {
				seen[h.ProxyWallet] = true
				wallets = append(wallets, h.ProxyWallet)
			}
		}
	}

	return wallets
}

// TotalHolders returns the total number of holder entries across all tokens.
// Note: This may count the same wallet multiple times if it holds multiple tokens.
func (hl HolderList) TotalHolders() int {
	count := 0
	for _, ht := range hl {
		count += len(ht.Holders)
	}
	return count
}

// FindHoldersByWallet returns all Holder entries for a given wallet address.
func (hl HolderList) FindHoldersByWallet(wallet string) []Holder {
	var holders []Holder
	for _, ht := range hl {
		for _, h := range ht.Holders {
			if h.ProxyWallet == wallet {
				holders = append(holders, h)
			}
		}
	}
	return holders
}

const (
	// DefaultHoldersLimit is the default number of holders per market (max supported by API).
	DefaultHoldersLimit = 20
)

// GetHolders fetches holders for the specified markets with custom options.
// The Polymarket API returns holders grouped by token (asset ID) for each market.
func (d *DataAPI) GetHolders(ctx context.Context, opts HolderQueryOptions) (HolderList, error) {
	url := d.baseURL + "/holders"

	params := opts.toParams()

	resp, err := d.client.GetWithParams(ctx, url, params)
	if err != nil {
		return nil, fmt.Errorf("fetching holders: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var holders HolderList
	if err := json.NewDecoder(resp.Body).Decode(&holders); err != nil {
		return nil, fmt.Errorf("decoding holders: %w", err)
	}

	return holders, nil
}

// GetHoldersForMarket fetches holders for a single market (condition ID).
// This is a convenience method that wraps GetHolders with a single market.
// If limit is 0, it defaults to DefaultHoldersLimit (20).
func (d *DataAPI) GetHoldersForMarket(ctx context.Context, conditionID string, limit int) (HolderList, error) {
	if limit <= 0 {
		limit = DefaultHoldersLimit
	}

	opts := HolderQueryOptions{
		Markets: []string{conditionID},
		Limit:   limit,
	}

	return d.GetHolders(ctx, opts)
}
