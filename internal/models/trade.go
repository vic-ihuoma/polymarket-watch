// Package models defines data structures for Polymarket API responses.
package models

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Trade represents a single trade from the Polymarket Data API.
type Trade struct {
	// ID is the unique trade identifier.
	ID string `json:"id"`
	// TakerOrderID is the ID of the taker's order.
	TakerOrderID string `json:"taker_order_id"`
	// Market is the market/condition ID this trade belongs to.
	Market string `json:"market"`
	// AssetID is the asset/token ID being traded.
	AssetID string `json:"asset_id"`
	// Side is the trade direction: "BUY" or "SELL".
	Side string `json:"side"`
	// Size is the number of shares traded.
	Size float64 `json:"-"`
	// FeeRateBps is the fee rate in basis points.
	FeeRateBps int `json:"-"`
	// Price is the execution price (0-1 range for prediction markets).
	Price float64 `json:"-"`
	// Status is the trade status (e.g., "MATCHED").
	Status string `json:"status"`
	// MatchTime is when the trade was matched.
	MatchTime time.Time `json:"-"`
	// LastUpdate is when the trade was last updated.
	LastUpdate time.Time `json:"-"`
	// Outcome is the outcome name (e.g., "Yes", "No").
	Outcome string `json:"outcome"`
	// BucketIndex identifies which outcome bucket this trade is for.
	BucketIndex int `json:"-"`
	// Owner is the wallet address that owns this trade.
	Owner string `json:"owner"`
	// MakerAddress is the maker's wallet address (may be empty).
	MakerAddress string `json:"maker_address"`
	// TransactionHash is the blockchain transaction hash (may be empty).
	TransactionHash string `json:"transaction_hash"`
	// TraderSide indicates if the owner was "TAKER" or "MAKER".
	TraderSide string `json:"trader_side"`
	// Type is the order type (e.g., "MARKET", "LIMIT").
	Type string `json:"type"`
}

// tradeJSON is an internal struct for JSON unmarshaling with string numeric fields.
type tradeJSON struct {
	ID              string  `json:"id"`
	TakerOrderID    string  `json:"taker_order_id"`
	Market          string  `json:"market"`
	AssetID         string  `json:"asset_id"`
	Side            string  `json:"side"`
	Size            string  `json:"size"`
	FeeRateBps      string  `json:"fee_rate_bps"`
	Price           string  `json:"price"`
	Status          string  `json:"status"`
	MatchTime       string  `json:"match_time"`
	LastUpdate      string  `json:"last_update"`
	Outcome         string  `json:"outcome"`
	BucketIndex     string  `json:"bucket_index"`
	Owner           string  `json:"owner"`
	MakerAddress    *string `json:"maker_address"`
	TransactionHash *string `json:"transaction_hash"`
	TraderSide      string  `json:"trader_side"`
	Type            string  `json:"type"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Trade.
// The Polymarket API returns numeric values as strings, so we need to convert them.
func (t *Trade) UnmarshalJSON(data []byte) error {
	var raw tradeJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	t.ID = raw.ID
	t.TakerOrderID = raw.TakerOrderID
	t.Market = raw.Market
	t.AssetID = raw.AssetID
	t.Side = raw.Side
	t.Status = raw.Status
	t.Outcome = raw.Outcome
	t.Owner = raw.Owner
	t.TraderSide = raw.TraderSide
	t.Type = raw.Type

	// Handle nullable string fields
	if raw.MakerAddress != nil {
		t.MakerAddress = *raw.MakerAddress
	}
	if raw.TransactionHash != nil {
		t.TransactionHash = *raw.TransactionHash
	}

	// Parse numeric strings
	if raw.Size != "" {
		size, err := strconv.ParseFloat(raw.Size, 64)
		if err != nil {
			return err
		}
		t.Size = size
	}

	if raw.FeeRateBps != "" {
		fee, err := strconv.Atoi(raw.FeeRateBps)
		if err != nil {
			return err
		}
		t.FeeRateBps = fee
	}

	if raw.Price != "" {
		price, err := strconv.ParseFloat(raw.Price, 64)
		if err != nil {
			return err
		}
		t.Price = price
	}

	if raw.BucketIndex != "" {
		idx, err := strconv.Atoi(raw.BucketIndex)
		if err != nil {
			return err
		}
		t.BucketIndex = idx
	}

	// Parse timestamps
	if raw.MatchTime != "" {
		matchTime, err := time.Parse(time.RFC3339, raw.MatchTime)
		if err != nil {
			return err
		}
		t.MatchTime = matchTime
	}

	if raw.LastUpdate != "" {
		lastUpdate, err := time.Parse(time.RFC3339, raw.LastUpdate)
		if err != nil {
			return err
		}
		t.LastUpdate = lastUpdate
	}

	return nil
}

// IsBuy returns true if this trade is a buy order.
func (t *Trade) IsBuy() bool {
	return strings.EqualFold(t.Side, "BUY")
}

// IsSell returns true if this trade is a sell order.
func (t *Trade) IsSell() bool {
	return strings.EqualFold(t.Side, "SELL")
}

// Value returns the total value of the trade (size * price).
func (t *Trade) Value() float64 {
	return t.Size * t.Price
}

// TradeList is a slice of trades with convenience methods.
type TradeList []Trade

// SortByTime sorts trades by match time in ascending order.
func (tl TradeList) SortByTime() {
	sort.Slice(tl, func(i, j int) bool {
		return tl[i].MatchTime.Before(tl[j].MatchTime)
	})
}

// FilterByMarket returns trades that belong to the specified market.
func (tl TradeList) FilterByMarket(market string) TradeList {
	var filtered TradeList
	for _, trade := range tl {
		if trade.Market == market {
			filtered = append(filtered, trade)
		}
	}
	return filtered
}

// TotalValue returns the sum of all trade values.
func (tl TradeList) TotalValue() float64 {
	var total float64
	for _, trade := range tl {
		total += trade.Value()
	}
	return total
}
