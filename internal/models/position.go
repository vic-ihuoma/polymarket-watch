// Package models defines data structures for Polymarket API responses.
package models

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Position represents a user's position from the Polymarket Data API.
type Position struct {
	// ProxyWallet is the user's proxy wallet address.
	ProxyWallet string `json:"proxyWallet"`
	// Asset is the asset/token ID.
	Asset string `json:"asset"`
	// ConditionID is the market condition ID this position belongs to.
	ConditionID string `json:"conditionId"`
	// Outcome is the outcome name (e.g., "Yes", "No").
	Outcome string `json:"outcome"`
	// OutcomeIndex is the index of the outcome (0 for Yes, 1 for No typically).
	OutcomeIndex int `json:"-"`
	// Size is the number of shares held.
	Size float64 `json:"-"`
	// AvgPrice is the average entry price.
	AvgPrice float64 `json:"-"`
	// InitialValue is the original value of the position.
	InitialValue float64 `json:"-"`
	// CurrentValue is the current value of the position.
	CurrentValue float64 `json:"-"`
	// CashBalance is the available cash balance.
	CashBalance float64 `json:"-"`
	// PnL is the profit and loss.
	PnL float64 `json:"-"`
	// RealizedPnL is the realized profit and loss.
	RealizedPnL float64 `json:"-"`
	// PercentPnL is the percentage profit and loss.
	PercentPnL float64 `json:"-"`
	// CurPrice is the current market price.
	CurPrice float64 `json:"-"`
	// Redeemable indicates if the position can be redeemed.
	Redeemable bool `json:"redeemable"`
	// Mergeable indicates if the position can be merged.
	Mergeable bool `json:"mergeable"`
}

// positionJSON is an internal struct for JSON unmarshaling with string numeric fields.
type positionJSON struct {
	ProxyWallet  string `json:"proxyWallet"`
	Asset        string `json:"asset"`
	ConditionID  string `json:"conditionId"`
	Outcome      string `json:"outcome"`
	OutcomeIndex string `json:"outcomeIndex"`
	Size         string `json:"size"`
	AvgPrice     string `json:"avgPrice"`
	InitialValue string `json:"initialValue"`
	CurrentValue string `json:"currentValue"`
	CashBalance  string `json:"cashBalance"`
	PnL          string `json:"pnl"`
	RealizedPnL  string `json:"realizedPnl"`
	PercentPnL   string `json:"percentPnl"`
	CurPrice     string `json:"curPrice"`
	Redeemable   bool   `json:"redeemable"`
	Mergeable    bool   `json:"mergeable"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Position.
// The Polymarket API returns numeric values as strings, so we need to convert them.
func (p *Position) UnmarshalJSON(data []byte) error {
	var raw positionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	p.ProxyWallet = raw.ProxyWallet
	p.Asset = raw.Asset
	p.ConditionID = raw.ConditionID
	p.Outcome = raw.Outcome
	p.Redeemable = raw.Redeemable
	p.Mergeable = raw.Mergeable

	// Parse outcomeIndex
	if raw.OutcomeIndex != "" {
		idx, err := strconv.Atoi(raw.OutcomeIndex)
		if err != nil {
			return err
		}
		p.OutcomeIndex = idx
	}

	// Parse numeric string fields
	if raw.Size != "" {
		size, err := strconv.ParseFloat(raw.Size, 64)
		if err != nil {
			return err
		}
		p.Size = size
	}

	if raw.AvgPrice != "" {
		avgPrice, err := strconv.ParseFloat(raw.AvgPrice, 64)
		if err != nil {
			return err
		}
		p.AvgPrice = avgPrice
	}

	if raw.InitialValue != "" {
		initialValue, err := strconv.ParseFloat(raw.InitialValue, 64)
		if err != nil {
			return err
		}
		p.InitialValue = initialValue
	}

	if raw.CurrentValue != "" {
		currentValue, err := strconv.ParseFloat(raw.CurrentValue, 64)
		if err != nil {
			return err
		}
		p.CurrentValue = currentValue
	}

	if raw.CashBalance != "" {
		cashBalance, err := strconv.ParseFloat(raw.CashBalance, 64)
		if err != nil {
			return err
		}
		p.CashBalance = cashBalance
	}

	if raw.PnL != "" {
		pnl, err := strconv.ParseFloat(raw.PnL, 64)
		if err != nil {
			return err
		}
		p.PnL = pnl
	}

	if raw.RealizedPnL != "" {
		realizedPnl, err := strconv.ParseFloat(raw.RealizedPnL, 64)
		if err != nil {
			return err
		}
		p.RealizedPnL = realizedPnl
	}

	if raw.PercentPnL != "" {
		percentPnl, err := strconv.ParseFloat(raw.PercentPnL, 64)
		if err != nil {
			return err
		}
		p.PercentPnL = percentPnl
	}

	if raw.CurPrice != "" {
		curPrice, err := strconv.ParseFloat(raw.CurPrice, 64)
		if err != nil {
			return err
		}
		p.CurPrice = curPrice
	}

	return nil
}

// IsYes returns true if this position is for a "Yes" outcome.
func (p *Position) IsYes() bool {
	return strings.EqualFold(p.Outcome, "Yes")
}

// IsNo returns true if this position is for a "No" outcome.
func (p *Position) IsNo() bool {
	return strings.EqualFold(p.Outcome, "No")
}

// Value returns the total value of the position (size * avgPrice).
func (p *Position) Value() float64 {
	return p.Size * p.AvgPrice
}

// IsProfitable returns true if the position has positive PnL.
func (p *Position) IsProfitable() bool {
	return p.PnL > 0
}

// PositionList is a slice of positions with convenience methods.
type PositionList []Position

// FilterByCondition returns positions that belong to the specified condition.
func (pl PositionList) FilterByCondition(conditionID string) PositionList {
	var filtered PositionList
	for _, pos := range pl {
		if pos.ConditionID == conditionID {
			filtered = append(filtered, pos)
		}
	}
	return filtered
}

// TotalValue returns the sum of all position values.
func (pl PositionList) TotalValue() float64 {
	var total float64
	for _, pos := range pl {
		total += pos.Value()
	}
	return total
}

// OpposingPair represents a Yes and No position on the same market.
// This is critical for detecting arbitrage strategies.
type OpposingPair struct {
	// ConditionID is the market condition ID.
	ConditionID string
	// YesPosition is the Yes outcome position.
	YesPosition Position
	// NoPosition is the No outcome position.
	NoPosition Position
}

// Spread returns the profit margin (1 - yesPrice - noPrice).
// Positive spread indicates potential arbitrage profit.
func (op *OpposingPair) Spread() float64 {
	return 1.0 - op.YesPosition.AvgPrice - op.NoPosition.AvgPrice
}

// IsProfitable returns true if the spread is positive.
func (op *OpposingPair) IsProfitable() bool {
	return op.Spread() > 0
}

// FindOpposingPositions identifies all Yes/No pairs on the same market.
// This is essential for detecting spread arbitrage strategies like Account88888.
func (pl PositionList) FindOpposingPositions() []OpposingPair {
	// Group positions by conditionID
	byCondition := make(map[string][]Position)
	for _, pos := range pl {
		byCondition[pos.ConditionID] = append(byCondition[pos.ConditionID], pos)
	}

	var pairs []OpposingPair
	for conditionID, positions := range byCondition {
		var yesPos, noPos *Position
		for i := range positions {
			if positions[i].IsYes() && positions[i].Size > 0 {
				yesPos = &positions[i]
			}
			if positions[i].IsNo() && positions[i].Size > 0 {
				noPos = &positions[i]
			}
		}
		// Only include if we have both sides
		if yesPos != nil && noPos != nil {
			pairs = append(pairs, OpposingPair{
				ConditionID: conditionID,
				YesPosition: *yesPos,
				NoPosition:  *noPos,
			})
		}
	}

	return pairs
}
