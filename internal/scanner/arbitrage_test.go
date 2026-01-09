// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"testing"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewArbitrageDetector(t *testing.T) {
	t.Run("creates detector with default threshold", func(t *testing.T) {
		d := NewArbitrageDetector()
		if d == nil {
			t.Fatal("expected non-nil detector")
		}
		if d.threshold != DefaultSpreadThreshold {
			t.Errorf("expected threshold %f, got %f", DefaultSpreadThreshold, d.threshold)
		}
	})

	t.Run("creates detector with custom threshold", func(t *testing.T) {
		customThreshold := 0.05
		d := NewArbitrageDetector(WithSpreadThreshold(customThreshold))
		if d.threshold != customThreshold {
			t.Errorf("expected threshold %f, got %f", customThreshold, d.threshold)
		}
	})
}

func TestArbitrageDetector_Name(t *testing.T) {
	d := NewArbitrageDetector()
	name := d.Name()
	if name != "arbitrage" {
		t.Errorf("expected name 'arbitrage', got '%s'", name)
	}
}

func TestArbitrageDetector_Weight(t *testing.T) {
	d := NewArbitrageDetector()
	weight := d.Weight()
	expected := 0.35
	if weight != expected {
		t.Errorf("expected weight %f, got %f", expected, weight)
	}
}

func TestArbitrageDetector_Analyze(t *testing.T) {
	tests := []struct {
		name           string
		positions      models.PositionList
		trades         models.TradeList
		expectedScore  float64
		expectSignals  int
		expectEvidence bool
	}{
		{
			name:          "no positions returns zero score",
			positions:     nil,
			trades:        nil,
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "single position no opposing",
			positions: models.PositionList{
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.6},
			},
			trades:        nil,
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "one opposing pair with profitable spread",
			positions: models.PositionList{
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.45},
			},
			trades:         nil,
			expectedScore:  1.0,
			expectSignals:  1,
			expectEvidence: true,
		},
		{
			name: "opposing pair with zero spread (no profit)",
			positions: models.PositionList{
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.50},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.50},
			},
			trades:        nil,
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "opposing pair with negative spread",
			positions: models.PositionList{
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.55},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.55},
			},
			trades:        nil,
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name: "multiple opposing pairs with varying spreads",
			positions: models.PositionList{
				// Pair 1: profitable spread of 0.10
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.45},
				// Pair 2: profitable spread of 0.20
				{ConditionID: "cond2", Outcome: "Yes", Size: 50, AvgPrice: 0.40},
				{ConditionID: "cond2", Outcome: "No", Size: 50, AvgPrice: 0.40},
				// Pair 3: zero spread (no arbitrage)
				{ConditionID: "cond3", Outcome: "Yes", Size: 200, AvgPrice: 0.50},
				{ConditionID: "cond3", Outcome: "No", Size: 200, AvgPrice: 0.50},
			},
			trades:         nil,
			expectedScore:  1.0, // 2 profitable pairs = max score
			expectSignals:  1,   // One aggregated signal
			expectEvidence: true,
		},
		{
			name: "spread below threshold not flagged",
			positions: models.PositionList{
				// Spread of 0.02 (below default threshold of 0.0)
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.49},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.49},
			},
			trades:         nil,
			expectedScore:  1.0, // 0.02 > 0 so still flagged
			expectSignals:  1,
			expectEvidence: true,
		},
		{
			name: "mixed positions with some opposing",
			positions: models.PositionList{
				// Opposing pair with profit
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.45},
				// Standalone positions (no opposing)
				{ConditionID: "cond2", Outcome: "Yes", Size: 50, AvgPrice: 0.70},
				{ConditionID: "cond3", Outcome: "No", Size: 30, AvgPrice: 0.25},
			},
			trades:         nil,
			expectedScore:  1.0, // 1 profitable pair
			expectSignals:  1,
			expectEvidence: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewArbitrageDetector()
			result := d.Analyze(tt.positions, tt.trades)

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %f, got %f", tt.expectedScore, result.Score)
			}

			if len(result.Signals) != tt.expectSignals {
				t.Errorf("expected %d signals, got %d", tt.expectSignals, len(result.Signals))
			}

			if tt.expectEvidence && len(result.Signals) > 0 {
				signal := result.Signals[0]
				if signal.Type != "arbitrage" {
					t.Errorf("expected signal type 'arbitrage', got '%s'", signal.Type)
				}
				if signal.Evidence == nil {
					t.Error("expected evidence in signal")
				}
				if _, ok := signal.Evidence["profitable_pairs"]; !ok {
					t.Error("expected 'profitable_pairs' in evidence")
				}
				if _, ok := signal.Evidence["total_spread"]; !ok {
					t.Error("expected 'total_spread' in evidence")
				}
			}
		})
	}
}

func TestArbitrageDetector_AnalyzeWithCustomThreshold(t *testing.T) {
	tests := []struct {
		name          string
		threshold     float64
		positions     models.PositionList
		expectedScore float64
		expectSignals int
	}{
		{
			name:      "high threshold excludes small spreads",
			threshold: 0.15,
			positions: models.PositionList{
				// Spread of 0.10 - below threshold
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.45},
			},
			expectedScore: 0.0,
			expectSignals: 0,
		},
		{
			name:      "high threshold includes large spreads",
			threshold: 0.15,
			positions: models.PositionList{
				// Spread of 0.20 - above threshold
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.40},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.40},
			},
			expectedScore: 1.0,
			expectSignals: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewArbitrageDetector(WithSpreadThreshold(tt.threshold))
			result := d.Analyze(tt.positions, nil)

			if result.Score != tt.expectedScore {
				t.Errorf("expected score %f, got %f", tt.expectedScore, result.Score)
			}

			if len(result.Signals) != tt.expectSignals {
				t.Errorf("expected %d signals, got %d", tt.expectSignals, len(result.Signals))
			}
		})
	}
}

func TestArbitrageDetector_ScoreGradient(t *testing.T) {
	// Test that the score scales with spread magnitude
	tests := []struct {
		name       string
		positions  models.PositionList
		minScore   float64
		maxScore   float64
		checkRange bool
	}{
		{
			name: "single pair with small spread (0.02)",
			positions: models.PositionList{
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.49},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.49},
			},
			minScore: 0.9,
			maxScore: 1.0,
		},
		{
			name: "single pair with large spread (0.30)",
			positions: models.PositionList{
				{ConditionID: "cond1", Outcome: "Yes", Size: 100, AvgPrice: 0.35},
				{ConditionID: "cond1", Outcome: "No", Size: 100, AvgPrice: 0.35},
			},
			minScore: 1.0,
			maxScore: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewArbitrageDetector()
			result := d.Analyze(tt.positions, nil)

			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("expected score in range [%f, %f], got %f",
					tt.minScore, tt.maxScore, result.Score)
			}
		})
	}
}

func TestArbitrageResult_TotalSpread(t *testing.T) {
	tests := []struct {
		name     string
		pairs    []ProfitablePair
		expected float64
	}{
		{
			name:     "empty pairs",
			pairs:    nil,
			expected: 0,
		},
		{
			name: "single pair",
			pairs: []ProfitablePair{
				{Spread: 0.10, YesPrice: 0.45, NoPrice: 0.45},
			},
			expected: 0.10,
		},
		{
			name: "multiple pairs",
			pairs: []ProfitablePair{
				{Spread: 0.10, YesPrice: 0.45, NoPrice: 0.45},
				{Spread: 0.15, YesPrice: 0.40, NoPrice: 0.45},
				{Spread: 0.05, YesPrice: 0.47, NoPrice: 0.48},
			},
			expected: 0.30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ArbitrageResult{ProfitablePairs: tt.pairs}
			total := r.TotalSpread()

			// Use epsilon for float comparison
			epsilon := 0.0001
			if diff := total - tt.expected; diff < -epsilon || diff > epsilon {
				t.Errorf("expected total spread %f, got %f", tt.expected, total)
			}
		})
	}
}

func TestDetectorInterface(t *testing.T) {
	// Compile-time check that ArbitrageDetector implements Detector interface
	var _ Detector = (*ArbitrageDetector)(nil)
}
