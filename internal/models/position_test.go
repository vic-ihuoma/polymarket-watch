package models

import (
	"encoding/json"
	"math"
	"testing"
)

func TestPosition_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    Position
		wantErr bool
	}{
		{
			name: "full position",
			json: `{
				"proxyWallet": "0x1234567890abcdef",
				"asset": "12345",
				"conditionId": "0xabc123",
				"outcome": "Yes",
				"outcomeIndex": "0",
				"size": "100.5",
				"avgPrice": "0.65",
				"initialValue": "65.325",
				"currentValue": "70.35",
				"cashBalance": "1000.00",
				"pnl": "5.025",
				"realizedPnl": "2.00",
				"percentPnl": "7.69",
				"curPrice": "0.70",
				"redeemable": true,
				"mergeable": false
			}`,
			want: Position{
				ProxyWallet:  "0x1234567890abcdef",
				Asset:        "12345",
				ConditionID:  "0xabc123",
				Outcome:      "Yes",
				OutcomeIndex: 0,
				Size:         100.5,
				AvgPrice:     0.65,
				InitialValue: 65.325,
				CurrentValue: 70.35,
				CashBalance:  1000.00,
				PnL:          5.025,
				RealizedPnL:  2.00,
				PercentPnL:   7.69,
				CurPrice:     0.70,
				Redeemable:   true,
				Mergeable:    false,
			},
			wantErr: false,
		},
		{
			name: "minimal position",
			json: `{
				"proxyWallet": "0xwallet",
				"asset": "999",
				"conditionId": "0xcond",
				"outcome": "No",
				"outcomeIndex": "1",
				"size": "50",
				"avgPrice": "0.30"
			}`,
			want: Position{
				ProxyWallet:  "0xwallet",
				Asset:        "999",
				ConditionID:  "0xcond",
				Outcome:      "No",
				OutcomeIndex: 1,
				Size:         50,
				AvgPrice:     0.30,
			},
			wantErr: false,
		},
		{
			name: "zero size position",
			json: `{
				"proxyWallet": "0xwallet",
				"asset": "999",
				"conditionId": "0xcond",
				"outcome": "Yes",
				"outcomeIndex": "0",
				"size": "0",
				"avgPrice": "0.50"
			}`,
			want: Position{
				ProxyWallet:  "0xwallet",
				Asset:        "999",
				ConditionID:  "0xcond",
				Outcome:      "Yes",
				OutcomeIndex: 0,
				Size:         0,
				AvgPrice:     0.50,
			},
			wantErr: false,
		},
		{
			name:    "invalid size",
			json:    `{"proxyWallet": "0x", "asset": "1", "conditionId": "0x", "outcome": "Yes", "outcomeIndex": "0", "size": "not_a_number", "avgPrice": "0.5"}`,
			wantErr: true,
		},
		{
			name:    "invalid avgPrice",
			json:    `{"proxyWallet": "0x", "asset": "1", "conditionId": "0x", "outcome": "Yes", "outcomeIndex": "0", "size": "100", "avgPrice": "invalid"}`,
			wantErr: true,
		},
		{
			name:    "invalid outcomeIndex",
			json:    `{"proxyWallet": "0x", "asset": "1", "conditionId": "0x", "outcome": "Yes", "outcomeIndex": "abc", "size": "100", "avgPrice": "0.5"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pos Position
			err := json.Unmarshal([]byte(tt.json), &pos)
			if (err != nil) != tt.wantErr {
				t.Errorf("Position.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if pos.ProxyWallet != tt.want.ProxyWallet {
				t.Errorf("ProxyWallet = %v, want %v", pos.ProxyWallet, tt.want.ProxyWallet)
			}
			if pos.Asset != tt.want.Asset {
				t.Errorf("Asset = %v, want %v", pos.Asset, tt.want.Asset)
			}
			if pos.ConditionID != tt.want.ConditionID {
				t.Errorf("ConditionID = %v, want %v", pos.ConditionID, tt.want.ConditionID)
			}
			if pos.Outcome != tt.want.Outcome {
				t.Errorf("Outcome = %v, want %v", pos.Outcome, tt.want.Outcome)
			}
			if pos.OutcomeIndex != tt.want.OutcomeIndex {
				t.Errorf("OutcomeIndex = %v, want %v", pos.OutcomeIndex, tt.want.OutcomeIndex)
			}
			if pos.Size != tt.want.Size {
				t.Errorf("Size = %v, want %v", pos.Size, tt.want.Size)
			}
			if pos.AvgPrice != tt.want.AvgPrice {
				t.Errorf("AvgPrice = %v, want %v", pos.AvgPrice, tt.want.AvgPrice)
			}
			if pos.InitialValue != tt.want.InitialValue {
				t.Errorf("InitialValue = %v, want %v", pos.InitialValue, tt.want.InitialValue)
			}
			if pos.CurrentValue != tt.want.CurrentValue {
				t.Errorf("CurrentValue = %v, want %v", pos.CurrentValue, tt.want.CurrentValue)
			}
			if pos.CashBalance != tt.want.CashBalance {
				t.Errorf("CashBalance = %v, want %v", pos.CashBalance, tt.want.CashBalance)
			}
			if pos.PnL != tt.want.PnL {
				t.Errorf("PnL = %v, want %v", pos.PnL, tt.want.PnL)
			}
			if pos.RealizedPnL != tt.want.RealizedPnL {
				t.Errorf("RealizedPnL = %v, want %v", pos.RealizedPnL, tt.want.RealizedPnL)
			}
			if pos.PercentPnL != tt.want.PercentPnL {
				t.Errorf("PercentPnL = %v, want %v", pos.PercentPnL, tt.want.PercentPnL)
			}
			if pos.CurPrice != tt.want.CurPrice {
				t.Errorf("CurPrice = %v, want %v", pos.CurPrice, tt.want.CurPrice)
			}
			if pos.Redeemable != tt.want.Redeemable {
				t.Errorf("Redeemable = %v, want %v", pos.Redeemable, tt.want.Redeemable)
			}
			if pos.Mergeable != tt.want.Mergeable {
				t.Errorf("Mergeable = %v, want %v", pos.Mergeable, tt.want.Mergeable)
			}
		})
	}
}

func TestPosition_IsYes(t *testing.T) {
	tests := []struct {
		name    string
		outcome string
		want    bool
	}{
		{"uppercase YES", "YES", true},
		{"lowercase yes", "yes", true},
		{"mixed case Yes", "Yes", true},
		{"no outcome", "No", false},
		{"uppercase NO", "NO", false},
		{"empty", "", false},
		{"other", "Maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := Position{Outcome: tt.outcome}
			if got := pos.IsYes(); got != tt.want {
				t.Errorf("Position.IsYes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosition_IsNo(t *testing.T) {
	tests := []struct {
		name    string
		outcome string
		want    bool
	}{
		{"uppercase NO", "NO", true},
		{"lowercase no", "no", true},
		{"mixed case No", "No", true},
		{"yes outcome", "Yes", false},
		{"uppercase YES", "YES", false},
		{"empty", "", false},
		{"other", "Maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := Position{Outcome: tt.outcome}
			if got := pos.IsNo(); got != tt.want {
				t.Errorf("Position.IsNo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosition_Value(t *testing.T) {
	tests := []struct {
		name     string
		size     float64
		avgPrice float64
		want     float64
	}{
		{"normal position", 100, 0.65, 65.0},
		{"zero size", 0, 0.65, 0},
		{"zero price", 100, 0, 0},
		{"fractional", 50.5, 0.42, 21.21},
		{"full price", 100, 1.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := Position{Size: tt.size, AvgPrice: tt.avgPrice}
			got := pos.Value()
			if got != tt.want {
				t.Errorf("Position.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosition_IsProfitable(t *testing.T) {
	tests := []struct {
		name string
		pnl  float64
		want bool
	}{
		{"positive pnl", 5.0, true},
		{"zero pnl", 0, false},
		{"negative pnl", -5.0, false},
		{"small positive", 0.01, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos := Position{PnL: tt.pnl}
			if got := pos.IsProfitable(); got != tt.want {
				t.Errorf("Position.IsProfitable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPositionList_FilterByCondition(t *testing.T) {
	positions := PositionList{
		{ConditionID: "0xcond1", Outcome: "Yes", Size: 100},
		{ConditionID: "0xcond1", Outcome: "No", Size: 50},
		{ConditionID: "0xcond2", Outcome: "Yes", Size: 75},
	}

	t.Run("filter by condition", func(t *testing.T) {
		filtered := positions.FilterByCondition("0xcond1")
		if len(filtered) != 2 {
			t.Errorf("FilterByCondition() got %d positions, want 2", len(filtered))
		}
	})

	t.Run("filter no match", func(t *testing.T) {
		filtered := positions.FilterByCondition("0xcond999")
		if len(filtered) != 0 {
			t.Errorf("FilterByCondition() got %d positions, want 0", len(filtered))
		}
	})
}

func TestPositionList_TotalValue(t *testing.T) {
	tests := []struct {
		name      string
		positions PositionList
		want      float64
	}{
		{
			name: "multiple positions",
			positions: PositionList{
				{Size: 100, AvgPrice: 0.50},
				{Size: 200, AvgPrice: 0.25},
			},
			want: 100.0, // 50 + 50
		},
		{
			name:      "empty list",
			positions: PositionList{},
			want:      0,
		},
		{
			name:      "nil list",
			positions: nil,
			want:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.positions.TotalValue(); got != tt.want {
				t.Errorf("PositionList.TotalValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPositionList_FindOpposingPositions(t *testing.T) {
	t.Run("has opposing positions", func(t *testing.T) {
		positions := PositionList{
			{ConditionID: "0xcond1", Outcome: "Yes", Size: 100, AvgPrice: 0.55},
			{ConditionID: "0xcond1", Outcome: "No", Size: 100, AvgPrice: 0.40},
			{ConditionID: "0xcond2", Outcome: "Yes", Size: 50, AvgPrice: 0.60},
		}
		pairs := positions.FindOpposingPositions()
		if len(pairs) != 1 {
			t.Errorf("FindOpposingPositions() got %d pairs, want 1", len(pairs))
			return
		}
		if pairs[0].ConditionID != "0xcond1" {
			t.Errorf("FindOpposingPositions() conditionID = %v, want 0xcond1", pairs[0].ConditionID)
		}
	})

	t.Run("no opposing positions", func(t *testing.T) {
		positions := PositionList{
			{ConditionID: "0xcond1", Outcome: "Yes", Size: 100},
			{ConditionID: "0xcond2", Outcome: "No", Size: 50},
		}
		pairs := positions.FindOpposingPositions()
		if len(pairs) != 0 {
			t.Errorf("FindOpposingPositions() got %d pairs, want 0", len(pairs))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		positions := PositionList{}
		pairs := positions.FindOpposingPositions()
		if len(pairs) != 0 {
			t.Errorf("FindOpposingPositions() got %d pairs, want 0", len(pairs))
		}
	})
}

func TestOpposingPair_Spread(t *testing.T) {
	tests := []struct {
		name       string
		yesPrice   float64
		noPrice    float64
		wantSpread float64
	}{
		{"profitable spread", 0.55, 0.40, 0.05}, // 1 - 0.55 - 0.40 = 0.05
		{"no profit", 0.50, 0.50, 0.0},          // 1 - 0.50 - 0.50 = 0.0
		{"loss", 0.60, 0.50, -0.10},             // 1 - 0.60 - 0.50 = -0.10
		{"extreme spread", 0.30, 0.30, 0.40},    // 1 - 0.30 - 0.30 = 0.40
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair := OpposingPair{
				YesPosition: Position{AvgPrice: tt.yesPrice},
				NoPosition:  Position{AvgPrice: tt.noPrice},
			}
			got := pair.Spread()
			// Use epsilon comparison for floating point
			const epsilon = 1e-9
			if math.Abs(got-tt.wantSpread) > epsilon {
				t.Errorf("OpposingPair.Spread() = %v, want %v", got, tt.wantSpread)
			}
		})
	}
}

func TestOpposingPair_IsProfitable(t *testing.T) {
	tests := []struct {
		name     string
		yesPrice float64
		noPrice  float64
		want     bool
	}{
		{"profitable", 0.55, 0.40, true},
		{"break even", 0.50, 0.50, false},
		{"loss", 0.60, 0.50, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair := OpposingPair{
				YesPosition: Position{AvgPrice: tt.yesPrice},
				NoPosition:  Position{AvgPrice: tt.noPrice},
			}
			if got := pair.IsProfitable(); got != tt.want {
				t.Errorf("OpposingPair.IsProfitable() = %v, want %v", got, tt.want)
			}
		})
	}
}
