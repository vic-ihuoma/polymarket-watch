package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTrade_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    Trade
		wantErr bool
	}{
		{
			name: "valid trade",
			json: `{
				"id": "trade-123",
				"taker_order_id": "order-456",
				"market": "0xmarket123",
				"asset_id": "asset-789",
				"side": "BUY",
				"size": "100.5",
				"fee_rate_bps": "50",
				"price": "0.65",
				"status": "MATCHED",
				"match_time": "2026-01-09T10:30:00Z",
				"last_update": "2026-01-09T10:30:01Z",
				"outcome": "Yes",
				"bucket_index": "1",
				"owner": "0xwallet123",
				"maker_address": "0xmaker456",
				"transaction_hash": "0xtx789",
				"trader_side": "TAKER",
				"type": "MARKET"
			}`,
			want: Trade{
				ID:              "trade-123",
				TakerOrderID:    "order-456",
				Market:          "0xmarket123",
				AssetID:         "asset-789",
				Side:            "BUY",
				Size:            100.5,
				FeeRateBps:      50,
				Price:           0.65,
				Status:          "MATCHED",
				MatchTime:       time.Date(2026, 1, 9, 10, 30, 0, 0, time.UTC),
				LastUpdate:      time.Date(2026, 1, 9, 10, 30, 1, 0, time.UTC),
				Outcome:         "Yes",
				BucketIndex:     1,
				Owner:           "0xwallet123",
				MakerAddress:    "0xmaker456",
				TransactionHash: "0xtx789",
				TraderSide:      "TAKER",
				Type:            "MARKET",
			},
			wantErr: false,
		},
		{
			name: "trade with null optional fields",
			json: `{
				"id": "trade-123",
				"taker_order_id": "order-456",
				"market": "0xmarket123",
				"asset_id": "asset-789",
				"side": "SELL",
				"size": "50",
				"fee_rate_bps": "0",
				"price": "0.35",
				"status": "MATCHED",
				"match_time": "2026-01-09T10:30:00Z",
				"last_update": "2026-01-09T10:30:01Z",
				"outcome": "No",
				"bucket_index": "0",
				"owner": "0xwallet123",
				"maker_address": null,
				"transaction_hash": null,
				"trader_side": "MAKER",
				"type": "LIMIT"
			}`,
			want: Trade{
				ID:           "trade-123",
				TakerOrderID: "order-456",
				Market:       "0xmarket123",
				AssetID:      "asset-789",
				Side:         "SELL",
				Size:         50,
				FeeRateBps:   0,
				Price:        0.35,
				Status:       "MATCHED",
				MatchTime:    time.Date(2026, 1, 9, 10, 30, 0, 0, time.UTC),
				LastUpdate:   time.Date(2026, 1, 9, 10, 30, 1, 0, time.UTC),
				Outcome:      "No",
				BucketIndex:  0,
				Owner:        "0xwallet123",
				TraderSide:   "MAKER",
				Type:         "LIMIT",
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Trade
			err := json.Unmarshal([]byte(tt.json), &got)

			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if got.ID != tt.want.ID {
				t.Errorf("ID = %v, want %v", got.ID, tt.want.ID)
			}
			if got.TakerOrderID != tt.want.TakerOrderID {
				t.Errorf("TakerOrderID = %v, want %v", got.TakerOrderID, tt.want.TakerOrderID)
			}
			if got.Market != tt.want.Market {
				t.Errorf("Market = %v, want %v", got.Market, tt.want.Market)
			}
			if got.AssetID != tt.want.AssetID {
				t.Errorf("AssetID = %v, want %v", got.AssetID, tt.want.AssetID)
			}
			if got.Side != tt.want.Side {
				t.Errorf("Side = %v, want %v", got.Side, tt.want.Side)
			}
			if got.Size != tt.want.Size {
				t.Errorf("Size = %v, want %v", got.Size, tt.want.Size)
			}
			if got.FeeRateBps != tt.want.FeeRateBps {
				t.Errorf("FeeRateBps = %v, want %v", got.FeeRateBps, tt.want.FeeRateBps)
			}
			if got.Price != tt.want.Price {
				t.Errorf("Price = %v, want %v", got.Price, tt.want.Price)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %v, want %v", got.Status, tt.want.Status)
			}
			if !got.MatchTime.Equal(tt.want.MatchTime) {
				t.Errorf("MatchTime = %v, want %v", got.MatchTime, tt.want.MatchTime)
			}
			if !got.LastUpdate.Equal(tt.want.LastUpdate) {
				t.Errorf("LastUpdate = %v, want %v", got.LastUpdate, tt.want.LastUpdate)
			}
			if got.Outcome != tt.want.Outcome {
				t.Errorf("Outcome = %v, want %v", got.Outcome, tt.want.Outcome)
			}
			if got.BucketIndex != tt.want.BucketIndex {
				t.Errorf("BucketIndex = %v, want %v", got.BucketIndex, tt.want.BucketIndex)
			}
			if got.Owner != tt.want.Owner {
				t.Errorf("Owner = %v, want %v", got.Owner, tt.want.Owner)
			}
			if got.MakerAddress != tt.want.MakerAddress {
				t.Errorf("MakerAddress = %v, want %v", got.MakerAddress, tt.want.MakerAddress)
			}
			if got.TransactionHash != tt.want.TransactionHash {
				t.Errorf("TransactionHash = %v, want %v", got.TransactionHash, tt.want.TransactionHash)
			}
			if got.TraderSide != tt.want.TraderSide {
				t.Errorf("TraderSide = %v, want %v", got.TraderSide, tt.want.TraderSide)
			}
			if got.Type != tt.want.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.want.Type)
			}
		})
	}
}

func TestTrade_IsBuy(t *testing.T) {
	tests := []struct {
		name string
		side string
		want bool
	}{
		{"BUY side", "BUY", true},
		{"buy lowercase", "buy", true},
		{"Buy mixed", "Buy", true},
		{"SELL side", "SELL", false},
		{"sell lowercase", "sell", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trade := Trade{Side: tt.side}
			if got := trade.IsBuy(); got != tt.want {
				t.Errorf("Trade.IsBuy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrade_IsSell(t *testing.T) {
	tests := []struct {
		name string
		side string
		want bool
	}{
		{"SELL side", "SELL", true},
		{"sell lowercase", "sell", true},
		{"Sell mixed", "Sell", true},
		{"BUY side", "BUY", false},
		{"buy lowercase", "buy", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trade := Trade{Side: tt.side}
			if got := trade.IsSell(); got != tt.want {
				t.Errorf("Trade.IsSell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrade_Value(t *testing.T) {
	tests := []struct {
		name  string
		size  float64
		price float64
		want  float64
	}{
		{"standard trade", 100, 0.65, 65.0},
		{"small trade", 10.5, 0.50, 5.25},
		{"full price", 100, 1.0, 100.0},
		{"zero price", 100, 0, 0},
		{"zero size", 0, 0.65, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trade := Trade{Size: tt.size, Price: tt.price}
			if got := trade.Value(); got != tt.want {
				t.Errorf("Trade.Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTradeList_SortByTime(t *testing.T) {
	now := time.Now()
	trades := []Trade{
		{ID: "3", MatchTime: now.Add(2 * time.Hour)},
		{ID: "1", MatchTime: now},
		{ID: "2", MatchTime: now.Add(1 * time.Hour)},
	}

	list := TradeList(trades)
	list.SortByTime()

	expected := []string{"1", "2", "3"}
	for i, id := range expected {
		if list[i].ID != id {
			t.Errorf("list[%d].ID = %v, want %v", i, list[i].ID, id)
		}
	}
}

func TestTradeList_FilterByMarket(t *testing.T) {
	trades := TradeList{
		{ID: "1", Market: "market-a"},
		{ID: "2", Market: "market-b"},
		{ID: "3", Market: "market-a"},
	}

	filtered := trades.FilterByMarket("market-a")

	if len(filtered) != 2 {
		t.Errorf("len(filtered) = %v, want 2", len(filtered))
	}
	if filtered[0].ID != "1" || filtered[1].ID != "3" {
		t.Errorf("filtered IDs = [%v, %v], want [1, 3]", filtered[0].ID, filtered[1].ID)
	}
}

func TestTradeList_TotalValue(t *testing.T) {
	trades := TradeList{
		{Size: 100, Price: 0.5},  // 50
		{Size: 200, Price: 0.25}, // 50
		{Size: 50, Price: 1.0},   // 50
	}

	if got := trades.TotalValue(); got != 150 {
		t.Errorf("TotalValue() = %v, want 150", got)
	}
}

func TestTradeList_Empty(t *testing.T) {
	var trades TradeList

	if trades.TotalValue() != 0 {
		t.Error("empty list TotalValue should be 0")
	}

	trades.SortByTime() // should not panic

	filtered := trades.FilterByMarket("any")
	if len(filtered) != 0 {
		t.Error("filtering empty list should return empty")
	}
}
