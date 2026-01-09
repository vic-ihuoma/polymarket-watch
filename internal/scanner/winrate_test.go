// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewWinRateAnalyzer(t *testing.T) {
	t.Run("default thresholds", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		if a.winRateThreshold != DefaultWinRateThreshold {
			t.Errorf("expected winRateThreshold %v, got %v", DefaultWinRateThreshold, a.winRateThreshold)
		}
		if a.minTrades != DefaultMinTradesForWinRate {
			t.Errorf("expected minTrades %v, got %v", DefaultMinTradesForWinRate, a.minTrades)
		}
	})

	t.Run("custom thresholds", func(t *testing.T) {
		a := NewWinRateAnalyzer(WithWinRateThreshold(0.90), WithMinTradesForWinRate(50))
		if a.winRateThreshold != 0.90 {
			t.Errorf("expected winRateThreshold 0.90, got %v", a.winRateThreshold)
		}
		if a.minTrades != 50 {
			t.Errorf("expected minTrades 50, got %v", a.minTrades)
		}
	})
}

func TestWinRateAnalyzer_Name(t *testing.T) {
	a := NewWinRateAnalyzer()
	if a.Name() != "winrate" {
		t.Errorf("expected name 'winrate', got '%s'", a.Name())
	}
}

func TestWinRateAnalyzer_Weight(t *testing.T) {
	a := NewWinRateAnalyzer()
	if a.Weight() != WinRateWeight {
		t.Errorf("expected weight %v, got %v", WinRateWeight, a.Weight())
	}
}

func TestWinRateAnalyzer_Analyze(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("empty trades", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		result := a.Analyze(nil, nil)
		if result.Score != 0 {
			t.Errorf("expected score 0, got %v", result.Score)
		}
		if len(result.Signals) != 0 {
			t.Errorf("expected no signals, got %d", len(result.Signals))
		}
	})

	t.Run("insufficient trades", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		trades := models.TradeList{
			{ID: "1", MatchTime: baseTime, Price: 0.50, Size: 10, Side: "BUY", Outcome: "Yes"},
			{ID: "2", MatchTime: baseTime.Add(time.Hour), Price: 0.60, Size: 10, Side: "SELL", Outcome: "Yes"},
		}
		result := a.Analyze(nil, trades)
		if result.Score != 0 {
			t.Errorf("expected score 0 for insufficient trades, got %v", result.Score)
		}
	})

	t.Run("normal win rate not flagged", func(t *testing.T) {
		// Create 150 trades with ~60% win rate (normal human trading)
		a := NewWinRateAnalyzer()
		trades := createTradesWithWinRate(baseTime, 150, 0.60)
		result := a.Analyze(nil, trades)
		if result.Score != 0 {
			t.Errorf("expected score 0 for normal win rate, got %v", result.Score)
		}
	})

	t.Run("high win rate with insufficient trades not flagged", func(t *testing.T) {
		// High win rate but only 50 trades (below threshold)
		a := NewWinRateAnalyzer()
		trades := createTradesWithWinRate(baseTime, 50, 0.98)
		result := a.Analyze(nil, trades)
		if result.Score != 0 {
			t.Errorf("expected score 0 for high win rate with insufficient trades, got %v", result.Score)
		}
	})

	t.Run("high win rate with many trades flagged", func(t *testing.T) {
		// 98% win rate with 150 trades - extremely suspicious
		a := NewWinRateAnalyzer()
		trades := createTradesWithWinRate(baseTime, 150, 0.98)
		result := a.Analyze(nil, trades)
		if result.Score != 1.0 {
			t.Errorf("expected score 1.0 for very high win rate, got %v", result.Score)
		}
		if len(result.Signals) != 1 {
			t.Errorf("expected 1 signal, got %d", len(result.Signals))
		}
		if result.Signals[0].Type != "winrate" {
			t.Errorf("expected signal type 'winrate', got '%s'", result.Signals[0].Type)
		}
		if result.Signals[0].Severity != "high" {
			t.Errorf("expected severity 'high', got '%s'", result.Signals[0].Severity)
		}
	})

	t.Run("borderline win rate above threshold", func(t *testing.T) {
		// 96% win rate with 110 trades - just above threshold
		a := NewWinRateAnalyzer()
		trades := createTradesWithWinRate(baseTime, 110, 0.96)
		result := a.Analyze(nil, trades)
		// Should be flagged but with lower score
		if result.Score == 0 {
			t.Error("expected non-zero score for borderline case above threshold")
		}
	})

	t.Run("win rate below threshold not flagged", func(t *testing.T) {
		// 92% win rate with 200 trades - below 95% threshold
		a := NewWinRateAnalyzer()
		trades := createTradesWithWinRate(baseTime, 200, 0.92)
		result := a.Analyze(nil, trades)
		// Should NOT be flagged since below threshold
		if result.Score != 0 {
			t.Errorf("expected score 0 for win rate below threshold, got %v", result.Score)
		}
	})

	t.Run("evidence includes statistics", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		trades := createTradesWithWinRate(baseTime, 150, 0.98)
		result := a.Analyze(nil, trades)
		if len(result.Signals) == 0 {
			t.Fatal("expected at least one signal")
		}
		evidence := result.Signals[0].Evidence
		if _, ok := evidence["win_rate"]; !ok {
			t.Error("evidence should include win_rate")
		}
		if _, ok := evidence["total_trades"]; !ok {
			t.Error("evidence should include total_trades")
		}
		if _, ok := evidence["profitable_trades"]; !ok {
			t.Error("evidence should include profitable_trades")
		}
	})
}

func TestWinRateAnalyzer_AnalyzeWithCustomThreshold(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("custom lower threshold", func(t *testing.T) {
		// Use 80% win rate threshold and 50 min trades
		a := NewWinRateAnalyzer(WithWinRateThreshold(0.80), WithMinTradesForWinRate(50))
		trades := createTradesWithWinRate(baseTime, 60, 0.85)
		result := a.Analyze(nil, trades)
		if result.Score == 0 {
			t.Error("expected non-zero score with custom threshold")
		}
	})

	t.Run("custom higher threshold", func(t *testing.T) {
		// Use 99% win rate threshold - 95% should not be flagged
		a := NewWinRateAnalyzer(WithWinRateThreshold(0.99))
		trades := createTradesWithWinRate(baseTime, 150, 0.95)
		result := a.Analyze(nil, trades)
		if result.Score != 0 {
			t.Errorf("expected score 0 with higher threshold, got %v", result.Score)
		}
	})
}

func TestCalculateWinRate(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("empty trades", func(t *testing.T) {
		stats := calculateWinRateStats(nil)
		if stats.WinRate != 0 {
			t.Errorf("expected win rate 0, got %v", stats.WinRate)
		}
		if stats.TotalTrades != 0 {
			t.Errorf("expected total trades 0, got %d", stats.TotalTrades)
		}
	})

	t.Run("all winning trades", func(t *testing.T) {
		trades := createTradesWithWinRate(baseTime, 10, 1.0)
		stats := calculateWinRateStats(trades)
		if stats.WinRate != 1.0 {
			t.Errorf("expected win rate 1.0, got %v", stats.WinRate)
		}
	})

	t.Run("all losing trades", func(t *testing.T) {
		trades := createTradesWithWinRate(baseTime, 10, 0.0)
		stats := calculateWinRateStats(trades)
		if stats.WinRate != 0.0 {
			t.Errorf("expected win rate 0.0, got %v", stats.WinRate)
		}
	})

	t.Run("mixed trades", func(t *testing.T) {
		trades := createTradesWithWinRate(baseTime, 100, 0.75)
		stats := calculateWinRateStats(trades)
		// Allow some tolerance for rounding
		if stats.WinRate < 0.70 || stats.WinRate > 0.80 {
			t.Errorf("expected win rate ~0.75, got %v", stats.WinRate)
		}
	})
}

func TestWinRateStats_ImpossibleRate(t *testing.T) {
	t.Run("improbable rate detection", func(t *testing.T) {
		// With 100 trades, 95%+ win rate is statistically improbable for random trading
		stats := WinRateStats{
			WinRate:          0.97,
			TotalTrades:      100,
			ProfitableTrades: 97,
		}
		if !stats.IsImprobable(0.95) {
			t.Error("expected 97% win rate with 100 trades to be flagged as improbable")
		}
	})

	t.Run("normal rate not flagged", func(t *testing.T) {
		stats := WinRateStats{
			WinRate:          0.60,
			TotalTrades:      100,
			ProfitableTrades: 60,
		}
		if stats.IsImprobable(0.95) {
			t.Error("expected 60% win rate to not be flagged as improbable")
		}
	})
}

func TestWinRateAnalyzer_Severity(t *testing.T) {
	t.Run("high severity for extreme win rate", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		severity := a.calculateSeverity(0.99, 500)
		if severity != "high" {
			t.Errorf("expected 'high' severity for 99%% win rate with 500 trades, got '%s'", severity)
		}
	})

	t.Run("medium severity for high win rate", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		severity := a.calculateSeverity(0.90, 150)
		if severity != "medium" {
			t.Errorf("expected 'medium' severity for 90%% win rate with 150 trades, got '%s'", severity)
		}
	})

	t.Run("low severity for borderline", func(t *testing.T) {
		a := NewWinRateAnalyzer()
		severity := a.calculateSeverity(0.85, 110)
		if severity != "low" {
			t.Errorf("expected 'low' severity for borderline case, got '%s'", severity)
		}
	})
}

func TestWinRateAnalyzer_Interface(t *testing.T) {
	// Compile-time check that WinRateAnalyzer implements Detector
	var _ Detector = (*WinRateAnalyzer)(nil)
}

// createTradesWithWinRate creates a list of trades with the specified win rate.
// Winning trades are BUY trades that exit at higher price (simulated by SELL trades).
// For simplicity, we create round-trip trades: BUY then SELL with profit/loss.
func createTradesWithWinRate(baseTime time.Time, count int, winRate float64) models.TradeList {
	trades := make(models.TradeList, 0, count*2)
	winningCount := int(float64(count) * winRate)

	for i := 0; i < count; i++ {
		buyTime := baseTime.Add(time.Duration(i*2) * time.Hour)
		sellTime := buyTime.Add(30 * time.Minute)
		market := "market-" + string(rune('A'+i%5))
		outcome := "Yes"
		if i%3 == 0 {
			outcome = "No"
		}

		buyPrice := 0.50
		var sellPrice float64

		// First 'winningCount' trades are winners
		if i < winningCount {
			sellPrice = 0.70 // Profit: sold higher than bought
		} else {
			sellPrice = 0.30 // Loss: sold lower than bought
		}

		// BUY trade
		trades = append(trades, models.Trade{
			ID:        "buy-" + string(rune('0'+i)),
			MatchTime: buyTime,
			Price:     buyPrice,
			Size:      10,
			Side:      "BUY",
			Outcome:   outcome,
			Market:    market,
		})

		// SELL trade (close position)
		trades = append(trades, models.Trade{
			ID:        "sell-" + string(rune('0'+i)),
			MatchTime: sellTime,
			Price:     sellPrice,
			Size:      10,
			Side:      "SELL",
			Outcome:   outcome,
			Market:    market,
		})
	}

	return trades
}
