// Package models defines data structures for Polymarket API responses.
package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScanReport_NewScanReport(t *testing.T) {
	wallet := "0x7f69983eb28245bba0d5083502a78744a8f66162"
	report := NewScanReport(wallet)

	if report.WalletAddress != wallet {
		t.Errorf("WalletAddress = %q, want %q", report.WalletAddress, wallet)
	}
	if report.ScanTime.IsZero() {
		t.Error("ScanTime should not be zero")
	}
	if report.Scores == nil {
		t.Error("Scores map should be initialized")
	}
	if report.Signals == nil {
		t.Error("Signals slice should be initialized")
	}
}

func TestScanReport_AddScore(t *testing.T) {
	report := NewScanReport("0x123")
	report.AddScore("arbitrage", 0.85)
	report.AddScore("timing", 0.72)

	if got := report.Scores["arbitrage"]; got != 0.85 {
		t.Errorf("Scores[arbitrage] = %v, want 0.85", got)
	}
	if got := report.Scores["timing"]; got != 0.72 {
		t.Errorf("Scores[timing] = %v, want 0.72", got)
	}
}

func TestScanReport_AddSignal(t *testing.T) {
	report := NewScanReport("0x123")
	report.AddSignal(DetectionSignal{
		Type:        "arbitrage",
		Description: "Found 5 opposing position pairs",
		Severity:    "high",
		Evidence:    map[string]interface{}{"pairs": 5},
	})

	if len(report.Signals) != 1 {
		t.Errorf("len(Signals) = %d, want 1", len(report.Signals))
	}
	if report.Signals[0].Type != "arbitrage" {
		t.Errorf("Signal.Type = %q, want %q", report.Signals[0].Type, "arbitrage")
	}
}

func TestScanReport_ComputeScore(t *testing.T) {
	tests := []struct {
		name    string
		scores  map[string]float64
		weights map[string]float64
		want    int
	}{
		{
			name:    "all zeros",
			scores:  map[string]float64{"arbitrage": 0, "timing": 0},
			weights: map[string]float64{"arbitrage": 0.5, "timing": 0.5},
			want:    0,
		},
		{
			name:    "all ones",
			scores:  map[string]float64{"arbitrage": 1.0, "timing": 1.0},
			weights: map[string]float64{"arbitrage": 0.5, "timing": 0.5},
			want:    100,
		},
		{
			name:    "mixed scores",
			scores:  map[string]float64{"arbitrage": 0.8, "timing": 0.6, "winrate": 0.4, "sizing": 0.2},
			weights: map[string]float64{"arbitrage": 0.35, "timing": 0.25, "winrate": 0.25, "sizing": 0.15},
			want:    56, // (0.8*0.35 + 0.6*0.25 + 0.4*0.25 + 0.2*0.15) * 100 = 0.56 * 100
		},
		{
			name:    "missing weight uses zero",
			scores:  map[string]float64{"arbitrage": 1.0, "unknown": 1.0},
			weights: map[string]float64{"arbitrage": 0.5},
			want:    50, // only arbitrage contributes
		},
		{
			name:    "empty scores",
			scores:  map[string]float64{},
			weights: map[string]float64{"arbitrage": 0.5},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewScanReport("0x123")
			for k, v := range tt.scores {
				report.AddScore(k, v)
			}
			got := report.ComputeScore(tt.weights)
			// Allow +/- 1 tolerance for floating-point rounding differences
			// that can occur due to map iteration order
			diff := got - tt.want
			if diff < -1 || diff > 1 {
				t.Errorf("ComputeScore() = %v, want %v (tolerance ±1)", got, tt.want)
			}
		})
	}
}

func TestScanReport_IsProbableBot(t *testing.T) {
	tests := []struct {
		name      string
		botScore  int
		threshold int
		want      bool
	}{
		{"below threshold", 50, 80, false},
		{"at threshold", 80, 80, true},
		{"above threshold", 95, 80, true},
		{"zero threshold", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewScanReport("0x123")
			report.BotScore = tt.botScore
			if got := report.IsProbableBot(tt.threshold); got != tt.want {
				t.Errorf("IsProbableBot(%d) = %v, want %v", tt.threshold, got, tt.want)
			}
		})
	}
}

func TestScanReport_Severity(t *testing.T) {
	tests := []struct {
		name     string
		botScore int
		want     string
	}{
		{"low score", 25, "low"},
		{"medium score at 50", 50, "medium"},
		{"medium score at 79", 79, "medium"},
		{"high score at 80", 80, "high"},
		{"high score at 100", 100, "high"},
		{"zero score", 0, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := NewScanReport("0x123")
			report.BotScore = tt.botScore
			if got := report.Severity(); got != tt.want {
				t.Errorf("Severity() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanReport_MarshalJSON(t *testing.T) {
	report := NewScanReport("0x7f69983eb28245bba0d5083502a78744a8f66162")
	report.BotScore = 85
	report.AddScore("arbitrage", 0.9)
	report.AddSignal(DetectionSignal{
		Type:        "arbitrage",
		Description: "Found opposing positions",
		Severity:    "high",
	})

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify it contains expected fields
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded["wallet_address"] != "0x7f69983eb28245bba0d5083502a78744a8f66162" {
		t.Errorf("wallet_address = %v, want 0x7f69983eb28245bba0d5083502a78744a8f66162", decoded["wallet_address"])
	}
	if decoded["bot_score"] != float64(85) {
		t.Errorf("bot_score = %v, want 85", decoded["bot_score"])
	}
}

func TestDetectionSignal(t *testing.T) {
	signal := DetectionSignal{
		Type:        "timing",
		Description: "Trade intervals highly consistent",
		Severity:    "high",
		Evidence: map[string]interface{}{
			"avg_interval_ms":    2500.0,
			"std_dev_ms":         150.0,
			"trade_count":        500,
			"consistent_percent": 98.5,
		},
	}

	if signal.Type != "timing" {
		t.Errorf("Type = %q, want %q", signal.Type, "timing")
	}
	if signal.Severity != "high" {
		t.Errorf("Severity = %q, want %q", signal.Severity, "high")
	}
	if signal.Evidence["trade_count"] != 500 {
		t.Errorf("Evidence[trade_count] = %v, want 500", signal.Evidence["trade_count"])
	}
}

func TestScanStats(t *testing.T) {
	stats := ScanStats{
		TotalTrades:       500,
		TotalPositions:    25,
		TotalVolume:       150000.50,
		UniqueMarkets:     10,
		FirstTradeTime:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		LastTradeTime:     time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC),
		WinRate:           0.87,
		AvgPositionSize:   300.50,
		OpposingPairCount: 5,
	}

	if stats.TotalTrades != 500 {
		t.Errorf("TotalTrades = %d, want 500", stats.TotalTrades)
	}
	if stats.WinRate != 0.87 {
		t.Errorf("WinRate = %v, want 0.87", stats.WinRate)
	}
}

func TestScanStats_TradingDuration(t *testing.T) {
	tests := []struct {
		name  string
		first time.Time
		last  time.Time
		want  time.Duration
	}{
		{
			name:  "30 days",
			first: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			last:  time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
			want:  30 * 24 * time.Hour,
		},
		{
			name:  "same time",
			first: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			last:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			want:  0,
		},
		{
			name:  "zero times",
			first: time.Time{},
			last:  time.Time{},
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := ScanStats{
				FirstTradeTime: tt.first,
				LastTradeTime:  tt.last,
			}
			if got := stats.TradingDuration(); got != tt.want {
				t.Errorf("TradingDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScanStats_TradesPerDay(t *testing.T) {
	tests := []struct {
		name   string
		trades int
		first  time.Time
		last   time.Time
		want   float64
	}{
		{
			name:   "10 trades over 5 days",
			trades: 10,
			first:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			last:   time.Date(2025, 1, 6, 0, 0, 0, 0, time.UTC),
			want:   2.0, // 10 / 5 days
		},
		{
			name:   "same day returns total trades",
			trades: 100,
			first:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			last:   time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			want:   100, // less than a day, return total
		},
		{
			name:   "zero trades",
			trades: 0,
			first:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			last:   time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := ScanStats{
				TotalTrades:    tt.trades,
				FirstTradeTime: tt.first,
				LastTradeTime:  tt.last,
			}
			if got := stats.TradesPerDay(); got != tt.want {
				t.Errorf("TradesPerDay() = %v, want %v", got, tt.want)
			}
		})
	}
}
