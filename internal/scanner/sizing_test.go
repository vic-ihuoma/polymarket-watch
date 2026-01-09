// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"math"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewSizingAnalyzer(t *testing.T) {
	t.Run("default threshold", func(t *testing.T) {
		a := NewSizingAnalyzer()
		if a.cvThreshold != DefaultCVThreshold {
			t.Errorf("expected cvThreshold %v, got %v", DefaultCVThreshold, a.cvThreshold)
		}
		if a.minTrades != DefaultMinTradesForSizing {
			t.Errorf("expected minTrades %v, got %v", DefaultMinTradesForSizing, a.minTrades)
		}
	})

	t.Run("custom threshold", func(t *testing.T) {
		a := NewSizingAnalyzer(WithCVThreshold(0.05), WithMinTradesForSizing(50))
		if a.cvThreshold != 0.05 {
			t.Errorf("expected cvThreshold 0.05, got %v", a.cvThreshold)
		}
		if a.minTrades != 50 {
			t.Errorf("expected minTrades 50, got %v", a.minTrades)
		}
	})
}

func TestSizingAnalyzer_Name(t *testing.T) {
	a := NewSizingAnalyzer()
	if a.Name() != "sizing" {
		t.Errorf("expected name 'sizing', got '%s'", a.Name())
	}
}

func TestSizingAnalyzer_Weight(t *testing.T) {
	a := NewSizingAnalyzer()
	if a.Weight() != SizingWeight {
		t.Errorf("expected weight %v, got %v", SizingWeight, a.Weight())
	}
}

func TestSizingAnalyzer_Analyze(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("empty trades", func(t *testing.T) {
		a := NewSizingAnalyzer()
		result := a.Analyze(nil, nil)
		if result.Score != 0 {
			t.Errorf("expected score 0, got %v", result.Score)
		}
		if len(result.Signals) != 0 {
			t.Errorf("expected no signals, got %d", len(result.Signals))
		}
	})

	t.Run("insufficient trades", func(t *testing.T) {
		a := NewSizingAnalyzer()
		trades := models.TradeList{
			{ID: "1", MatchTime: baseTime, Size: 100, Side: "BUY"},
			{ID: "2", MatchTime: baseTime.Add(time.Hour), Size: 100, Side: "BUY"},
		}
		result := a.Analyze(nil, trades)
		if result.Score != 0 {
			t.Errorf("expected score 0 for insufficient trades, got %v", result.Score)
		}
	})

	t.Run("uniform sizing flagged", func(t *testing.T) {
		// All trades with exactly the same size - very bot-like
		a := NewSizingAnalyzer()
		trades := createTradesWithSizes(baseTime, []float64{
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		})
		result := a.Analyze(nil, trades)
		if result.Score != 1.0 {
			t.Errorf("expected score 1.0 for uniform sizing, got %v", result.Score)
		}
		if len(result.Signals) != 1 {
			t.Errorf("expected 1 signal, got %d", len(result.Signals))
		}
		if result.Signals[0].Type != "sizing" {
			t.Errorf("expected signal type 'sizing', got '%s'", result.Signals[0].Type)
		}
	})

	t.Run("algorithmic round numbers flagged", func(t *testing.T) {
		// Round number sizes with low variance - algorithmic pattern
		a := NewSizingAnalyzer()
		trades := createTradesWithSizes(baseTime, []float64{
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			50, 50, 50, 50, 50, 50, 50, 50, 50, 50,
		})
		result := a.Analyze(nil, trades)
		// Should be flagged due to round number pattern
		if result.Score == 0 {
			t.Error("expected non-zero score for round number algorithmic sizing")
		}
	})

	t.Run("natural variance not flagged", func(t *testing.T) {
		// Human-like trading with natural variance in sizing
		a := NewSizingAnalyzer()
		trades := createTradesWithSizes(baseTime, []float64{
			47, 152, 83, 201, 67, 134, 89, 176, 45, 223,
			91, 158, 73, 189, 112, 267, 54, 199, 81, 143,
		})
		result := a.Analyze(nil, trades)
		if result.Score != 0 {
			t.Errorf("expected score 0 for natural variance, got %v", result.Score)
		}
	})

	t.Run("slightly varying algorithmic sizing flagged", func(t *testing.T) {
		// Tight variance around a value - algorithmic pattern
		a := NewSizingAnalyzer()
		trades := createTradesWithSizes(baseTime, []float64{
			99, 101, 100, 100, 99, 101, 100, 99, 100, 101,
			100, 99, 100, 101, 100, 99, 100, 100, 101, 99,
		})
		result := a.Analyze(nil, trades)
		if result.Score == 0 {
			t.Error("expected non-zero score for tight variance algorithmic sizing")
		}
	})

	t.Run("evidence includes statistics", func(t *testing.T) {
		a := NewSizingAnalyzer()
		trades := createTradesWithSizes(baseTime, []float64{
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
			100, 100, 100, 100, 100, 100, 100, 100, 100, 100,
		})
		result := a.Analyze(nil, trades)
		if len(result.Signals) == 0 {
			t.Fatal("expected at least one signal")
		}
		evidence := result.Signals[0].Evidence
		if _, ok := evidence["coefficient_of_variation"]; !ok {
			t.Error("evidence should include coefficient_of_variation")
		}
		if _, ok := evidence["mean_size"]; !ok {
			t.Error("evidence should include mean_size")
		}
		if _, ok := evidence["std_dev"]; !ok {
			t.Error("evidence should include std_dev")
		}
		if _, ok := evidence["round_number_ratio"]; !ok {
			t.Error("evidence should include round_number_ratio")
		}
	})
}

func TestSizingAnalyzer_AnalyzeWithCustomThreshold(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("custom lower threshold", func(t *testing.T) {
		// With CV threshold of 0.05, sizes with CV ~0.08 should not be flagged by low CV rules
		a := NewSizingAnalyzer(WithCVThreshold(0.05))
		// Create trades with CV ~0.08 (above threshold) and low round number ratio
		// These values have CV ~0.08 and only some are round numbers
		trades := createTradesWithSizes(baseTime, []float64{
			93, 107, 98, 102, 91, 109, 96, 104, 94, 106,
			97, 103, 92, 108, 99, 101, 95, 105, 93, 107,
		})
		result := a.Analyze(nil, trades)
		// Should not be flagged since CV > threshold and round ratio is low
		if result.Score != 0 {
			t.Errorf("expected score 0 with CV above threshold, got %v", result.Score)
		}
	})

	t.Run("custom higher threshold", func(t *testing.T) {
		// With CV threshold of 0.2, more variance should still be flagged
		a := NewSizingAnalyzer(WithCVThreshold(0.2))
		trades := createTradesWithSizes(baseTime, []float64{
			100, 90, 110, 100, 90, 110, 100, 90, 110, 100,
			90, 110, 100, 90, 110, 100, 90, 110, 100, 90,
		})
		result := a.Analyze(nil, trades)
		// Should be flagged since CV < higher threshold
		if result.Score == 0 {
			t.Error("expected non-zero score with higher CV threshold")
		}
	})
}

func TestCalculateSizingStats(t *testing.T) {
	t.Run("empty trades", func(t *testing.T) {
		stats := calculateSizingStats(nil)
		if stats.Mean != 0 {
			t.Errorf("expected mean 0, got %v", stats.Mean)
		}
		if stats.TradeCount != 0 {
			t.Errorf("expected trade count 0, got %d", stats.TradeCount)
		}
	})

	t.Run("single trade", func(t *testing.T) {
		trades := models.TradeList{
			{Size: 100},
		}
		stats := calculateSizingStats(trades)
		if stats.Mean != 100 {
			t.Errorf("expected mean 100, got %v", stats.Mean)
		}
		if stats.StdDev != 0 {
			t.Errorf("expected std dev 0 for single trade, got %v", stats.StdDev)
		}
	})

	t.Run("uniform sizes", func(t *testing.T) {
		trades := models.TradeList{
			{Size: 100}, {Size: 100}, {Size: 100}, {Size: 100}, {Size: 100},
		}
		stats := calculateSizingStats(trades)
		if stats.Mean != 100 {
			t.Errorf("expected mean 100, got %v", stats.Mean)
		}
		if stats.StdDev != 0 {
			t.Errorf("expected std dev 0 for uniform sizes, got %v", stats.StdDev)
		}
		if stats.CoefficientOfVariation() != 0 {
			t.Errorf("expected CV 0 for uniform sizes, got %v", stats.CoefficientOfVariation())
		}
	})

	t.Run("varying sizes", func(t *testing.T) {
		trades := models.TradeList{
			{Size: 100}, {Size: 200}, {Size: 300}, {Size: 400}, {Size: 500},
		}
		stats := calculateSizingStats(trades)
		expectedMean := 300.0
		if stats.Mean != expectedMean {
			t.Errorf("expected mean %v, got %v", expectedMean, stats.Mean)
		}
		// StdDev for [100,200,300,400,500] is sqrt(20000) ≈ 141.42
		if stats.StdDev < 140 || stats.StdDev > 143 {
			t.Errorf("expected std dev ~141.42, got %v", stats.StdDev)
		}
	})
}

func TestSizingStats_RoundNumberRatio(t *testing.T) {
	t.Run("all round numbers", func(t *testing.T) {
		stats := SizingStats{
			Sizes: []float64{100, 50, 200, 25, 500},
		}
		ratio := stats.RoundNumberRatio()
		if ratio != 1.0 {
			t.Errorf("expected round number ratio 1.0, got %v", ratio)
		}
	})

	t.Run("no round numbers", func(t *testing.T) {
		stats := SizingStats{
			Sizes: []float64{47.3, 152.7, 83.1, 201.9, 67.8},
		}
		ratio := stats.RoundNumberRatio()
		if ratio != 0.0 {
			t.Errorf("expected round number ratio 0.0, got %v", ratio)
		}
	})

	t.Run("mixed numbers", func(t *testing.T) {
		stats := SizingStats{
			Sizes: []float64{100, 47.3, 50, 152.7, 200},
		}
		ratio := stats.RoundNumberRatio()
		// 3 out of 5 are round numbers
		expectedRatio := 3.0 / 5.0
		if math.Abs(ratio-expectedRatio) > 0.001 {
			t.Errorf("expected round number ratio %v, got %v", expectedRatio, ratio)
		}
	})

	t.Run("empty sizes", func(t *testing.T) {
		stats := SizingStats{
			Sizes: []float64{},
		}
		ratio := stats.RoundNumberRatio()
		if ratio != 0.0 {
			t.Errorf("expected round number ratio 0.0 for empty, got %v", ratio)
		}
	})
}

func TestSizingStats_CoefficientOfVariation(t *testing.T) {
	t.Run("uniform sizes", func(t *testing.T) {
		stats := SizingStats{
			Mean:   100,
			StdDev: 0,
		}
		cv := stats.CoefficientOfVariation()
		if cv != 0 {
			t.Errorf("expected CV 0, got %v", cv)
		}
	})

	t.Run("zero mean", func(t *testing.T) {
		stats := SizingStats{
			Mean:   0,
			StdDev: 10,
		}
		cv := stats.CoefficientOfVariation()
		if cv != 0 {
			t.Errorf("expected CV 0 for zero mean, got %v", cv)
		}
	})

	t.Run("normal variance", func(t *testing.T) {
		stats := SizingStats{
			Mean:   100,
			StdDev: 10,
		}
		cv := stats.CoefficientOfVariation()
		if cv != 0.1 {
			t.Errorf("expected CV 0.1, got %v", cv)
		}
	})
}

func TestSizingAnalyzer_Severity(t *testing.T) {
	t.Run("high severity for very low CV with round numbers", func(t *testing.T) {
		a := NewSizingAnalyzer()
		severity := a.calculateSeverity(0.01, 0.95)
		if severity != "high" {
			t.Errorf("expected 'high' severity for CV 0.01 with 95%% round numbers, got '%s'", severity)
		}
	})

	t.Run("medium severity for low CV", func(t *testing.T) {
		a := NewSizingAnalyzer()
		severity := a.calculateSeverity(0.05, 0.3)
		if severity != "medium" {
			t.Errorf("expected 'medium' severity for CV 0.05, got '%s'", severity)
		}
	})

	t.Run("low severity for borderline", func(t *testing.T) {
		a := NewSizingAnalyzer()
		severity := a.calculateSeverity(0.09, 0.1)
		if severity != "low" {
			t.Errorf("expected 'low' severity for borderline CV, got '%s'", severity)
		}
	})
}

func TestSizingAnalyzer_Interface(t *testing.T) {
	// Compile-time check that SizingAnalyzer implements Detector
	var _ Detector = (*SizingAnalyzer)(nil)
}

// createTradesWithSizes creates a list of trades with the specified sizes.
func createTradesWithSizes(baseTime time.Time, sizes []float64) models.TradeList {
	trades := make(models.TradeList, len(sizes))
	for i, size := range sizes {
		trades[i] = models.Trade{
			ID:        "trade-" + string(rune('0'+i)),
			MatchTime: baseTime.Add(time.Duration(i) * time.Hour),
			Size:      size,
			Side:      "BUY",
			Market:    "test-market",
		}
	}
	return trades
}
