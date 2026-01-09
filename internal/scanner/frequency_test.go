// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"math"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewTimingAnalyzer(t *testing.T) {
	t.Run("creates analyzer with default threshold", func(t *testing.T) {
		a := NewTimingAnalyzer()
		if a == nil {
			t.Fatal("expected non-nil analyzer")
		}
		if a.threshold != DefaultIntervalThreshold {
			t.Errorf("expected threshold %v, got %v", DefaultIntervalThreshold, a.threshold)
		}
	})

	t.Run("creates analyzer with custom threshold", func(t *testing.T) {
		customThreshold := 3 * time.Second
		a := NewTimingAnalyzer(WithIntervalThreshold(customThreshold))
		if a.threshold != customThreshold {
			t.Errorf("expected threshold %v, got %v", customThreshold, a.threshold)
		}
	})
}

func TestTimingAnalyzer_Name(t *testing.T) {
	a := NewTimingAnalyzer()
	name := a.Name()
	if name != "timing" {
		t.Errorf("expected name 'timing', got '%s'", name)
	}
}

func TestTimingAnalyzer_Weight(t *testing.T) {
	a := NewTimingAnalyzer()
	weight := a.Weight()
	expected := 0.25
	if weight != expected {
		t.Errorf("expected weight %f, got %f", expected, weight)
	}
}

func TestTimingAnalyzer_Analyze(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		trades         models.TradeList
		expectedScore  float64
		expectSignals  int
		expectEvidence bool
		scoreRange     [2]float64 // min, max for approximate scores
	}{
		{
			name:          "no trades returns zero score",
			trades:        nil,
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "single trade returns zero score",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
			},
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "two trades returns zero score (insufficient data)",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(3 * time.Second)},
			},
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "consistent sub-5s intervals (bot-like)",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(2 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(4 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(6 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(8 * time.Second)},
			},
			expectedScore:  1.0,
			expectSignals:  1,
			expectEvidence: true,
		},
		{
			name: "exactly 5 second intervals (at threshold boundary - not flagged)",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(5 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(10 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(15 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(20 * time.Second)},
			},
			// PRD says "sub-5s intervals" - exactly 5s is not sub-5s
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "slow intervals (human-like)",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(30 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(90 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(120 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(200 * time.Second)},
			},
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "highly variable intervals (human-like)",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(2 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(60 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(61 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(180 * time.Second)},
			},
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "mixed intervals with one outlier (high variance - human-like)",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(3 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(6 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(9 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(100 * time.Second)},
			},
			// High variance due to 91s outlier indicates sporadic behavior, not systematic
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "moderately fast trades with consistent intervals",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(4 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(8 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(12 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(16 * time.Second)},
				{ID: "6", MatchTime: baseTime.Add(25 * time.Second)},
			},
			// 4 out of 5 intervals are fast, with reasonable consistency
			scoreRange:     [2]float64{0.5, 1.0},
			expectSignals:  1,
			expectEvidence: true,
		},
		{
			name: "unsorted trades should be sorted and analyzed",
			trades: models.TradeList{
				{ID: "3", MatchTime: baseTime.Add(4 * time.Second)},
				{ID: "1", MatchTime: baseTime},
				{ID: "4", MatchTime: baseTime.Add(6 * time.Second)},
				{ID: "2", MatchTime: baseTime.Add(2 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(8 * time.Second)},
			},
			expectedScore:  1.0,
			expectSignals:  1,
			expectEvidence: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewTimingAnalyzer()
			result := a.Analyze(nil, tt.trades)

			if tt.scoreRange[0] != 0 || tt.scoreRange[1] != 0 {
				// Check range
				if result.Score < tt.scoreRange[0] || result.Score > tt.scoreRange[1] {
					t.Errorf("expected score in range [%f, %f], got %f",
						tt.scoreRange[0], tt.scoreRange[1], result.Score)
				}
			} else {
				// Check exact value
				if result.Score != tt.expectedScore {
					t.Errorf("expected score %f, got %f", tt.expectedScore, result.Score)
				}
			}

			if len(result.Signals) != tt.expectSignals {
				t.Errorf("expected %d signals, got %d", tt.expectSignals, len(result.Signals))
			}

			if tt.expectEvidence && len(result.Signals) > 0 {
				signal := result.Signals[0]
				if signal.Type != "timing" {
					t.Errorf("expected signal type 'timing', got '%s'", signal.Type)
				}
				if signal.Evidence == nil {
					t.Error("expected evidence in signal")
				}
				if _, ok := signal.Evidence["mean_interval_seconds"]; !ok {
					t.Error("expected 'mean_interval_seconds' in evidence")
				}
				if _, ok := signal.Evidence["std_dev_seconds"]; !ok {
					t.Error("expected 'std_dev_seconds' in evidence")
				}
				if _, ok := signal.Evidence["fast_trade_ratio"]; !ok {
					t.Error("expected 'fast_trade_ratio' in evidence")
				}
			}
		})
	}
}

func TestTimingAnalyzer_AnalyzeWithCustomThreshold(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		threshold     time.Duration
		trades        models.TradeList
		expectedScore float64
		expectSignals int
	}{
		{
			name:      "lower threshold catches slower bots",
			threshold: 10 * time.Second,
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(8 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(16 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(24 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(32 * time.Second)},
			},
			expectedScore: 1.0,
			expectSignals: 1,
		},
		{
			name:      "higher threshold misses fast bots",
			threshold: 1 * time.Second,
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(3 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(6 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(9 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(12 * time.Second)},
			},
			expectedScore: 0.0,
			expectSignals: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewTimingAnalyzer(WithIntervalThreshold(tt.threshold))
			result := a.Analyze(nil, tt.trades)

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %f, got %f", tt.expectedScore, result.Score)
			}

			if len(result.Signals) != tt.expectSignals {
				t.Errorf("expected %d signals, got %d", tt.expectSignals, len(result.Signals))
			}
		})
	}
}

func TestTimingStats_StdDev(t *testing.T) {
	tests := []struct {
		name          string
		intervals     []time.Duration
		expectedStdMs float64 // expected std dev in milliseconds
		tolerance     float64 // tolerance for comparison
	}{
		{
			name:          "empty intervals",
			intervals:     nil,
			expectedStdMs: 0,
			tolerance:     0,
		},
		{
			name:          "single interval",
			intervals:     []time.Duration{5 * time.Second},
			expectedStdMs: 0,
			tolerance:     0,
		},
		{
			name: "identical intervals (zero std dev)",
			intervals: []time.Duration{
				2 * time.Second,
				2 * time.Second,
				2 * time.Second,
			},
			expectedStdMs: 0,
			tolerance:     0.1,
		},
		{
			name: "varying intervals",
			intervals: []time.Duration{
				1 * time.Second,
				2 * time.Second,
				3 * time.Second,
				4 * time.Second,
				5 * time.Second,
			},
			// Mean = 3s, variance = ((1-3)^2 + (2-3)^2 + (3-3)^2 + (4-3)^2 + (5-3)^2) / 5 = 2
			// Std dev = sqrt(2) ≈ 1.414s = 1414ms
			expectedStdMs: 1414,
			tolerance:     10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateTimingStats(tt.intervals)
			stdDevMs := stats.StdDev.Seconds() * 1000

			diff := math.Abs(stdDevMs - tt.expectedStdMs)
			if diff > tt.tolerance {
				t.Errorf("expected std dev %f ms (±%f), got %f ms",
					tt.expectedStdMs, tt.tolerance, stdDevMs)
			}
		})
	}
}

func TestTimingStats_Mean(t *testing.T) {
	tests := []struct {
		name           string
		intervals      []time.Duration
		expectedMeanMs float64
	}{
		{
			name:           "empty intervals",
			intervals:      nil,
			expectedMeanMs: 0,
		},
		{
			name: "single interval",
			intervals: []time.Duration{
				5 * time.Second,
			},
			expectedMeanMs: 5000,
		},
		{
			name: "multiple intervals",
			intervals: []time.Duration{
				2 * time.Second,
				4 * time.Second,
				6 * time.Second,
			},
			expectedMeanMs: 4000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateTimingStats(tt.intervals)
			meanMs := stats.Mean.Seconds() * 1000

			if meanMs != tt.expectedMeanMs {
				t.Errorf("expected mean %f ms, got %f ms", tt.expectedMeanMs, meanMs)
			}
		})
	}
}

func TestTimingStats_FastTradeRatio(t *testing.T) {
	threshold := 5 * time.Second
	tests := []struct {
		name          string
		intervals     []time.Duration
		expectedRatio float64
	}{
		{
			name:          "empty intervals",
			intervals:     nil,
			expectedRatio: 0,
		},
		{
			name: "all fast",
			intervals: []time.Duration{
				2 * time.Second,
				3 * time.Second,
				4 * time.Second,
			},
			expectedRatio: 1.0,
		},
		{
			name: "none fast",
			intervals: []time.Duration{
				10 * time.Second,
				20 * time.Second,
				30 * time.Second,
			},
			expectedRatio: 0.0,
		},
		{
			name: "half fast",
			intervals: []time.Duration{
				3 * time.Second,
				10 * time.Second,
				4 * time.Second,
				15 * time.Second,
			},
			expectedRatio: 0.5,
		},
		{
			name: "exactly at threshold (not counted as fast)",
			intervals: []time.Duration{
				5 * time.Second,
				5 * time.Second,
			},
			expectedRatio: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := calculateTimingStats(tt.intervals)
			ratio := stats.FastTradeRatio(threshold)

			epsilon := 0.0001
			if math.Abs(ratio-tt.expectedRatio) > epsilon {
				t.Errorf("expected ratio %f, got %f", tt.expectedRatio, ratio)
			}
		})
	}
}

func TestTimingAnalyzer_Severity(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	a := NewTimingAnalyzer()

	tests := []struct {
		name             string
		trades           models.TradeList
		expectedSeverity string
	}{
		{
			name: "very consistent fast trades - high severity",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(2 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(4 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(6 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(8 * time.Second)},
				{ID: "6", MatchTime: baseTime.Add(10 * time.Second)},
				{ID: "7", MatchTime: baseTime.Add(12 * time.Second)},
				{ID: "8", MatchTime: baseTime.Add(14 * time.Second)},
				{ID: "9", MatchTime: baseTime.Add(16 * time.Second)},
				{ID: "10", MatchTime: baseTime.Add(18 * time.Second)},
			},
			expectedSeverity: "high",
		},
		{
			name: "some fast trades - medium severity",
			trades: models.TradeList{
				{ID: "1", MatchTime: baseTime},
				{ID: "2", MatchTime: baseTime.Add(3 * time.Second)},
				{ID: "3", MatchTime: baseTime.Add(6 * time.Second)},
				{ID: "4", MatchTime: baseTime.Add(20 * time.Second)},
				{ID: "5", MatchTime: baseTime.Add(40 * time.Second)},
			},
			expectedSeverity: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.Analyze(nil, tt.trades)

			if len(result.Signals) == 0 {
				t.Fatal("expected at least one signal")
			}

			if result.Signals[0].Severity != tt.expectedSeverity {
				t.Errorf("expected severity '%s', got '%s'",
					tt.expectedSeverity, result.Signals[0].Severity)
			}
		})
	}
}

func TestTimingAnalyzer_Interface(t *testing.T) {
	// Compile-time check that TimingAnalyzer implements Detector interface
	var _ Detector = (*TimingAnalyzer)(nil)
}
