// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"math"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// DefaultCVThreshold is the coefficient of variation below which sizing is flagged.
// A CV of 0.1 (10%) or less indicates very consistent algorithmic position sizing.
const DefaultCVThreshold = 0.1

// DefaultMinTradesForSizing is the minimum number of trades required for sizing analysis.
// Sizing patterns are only meaningful with sufficient sample size.
const DefaultMinTradesForSizing = 10

// SizingWeight is the weight of the sizing analyzer in the composite score.
// This weight matches the PRD detection_algorithms specification.
const SizingWeight = 0.15

// SizingAnalyzer detects algorithmic position sizing patterns.
// Bots typically use consistent position sizes (same amount each trade) or
// programmatic sizing rules that result in low variance and round numbers.
type SizingAnalyzer struct {
	cvThreshold float64
	minTrades   int
}

// SizingOption is a functional option for configuring SizingAnalyzer.
type SizingOption func(*SizingAnalyzer)

// WithCVThreshold sets the coefficient of variation threshold for flagging.
func WithCVThreshold(threshold float64) SizingOption {
	return func(a *SizingAnalyzer) {
		a.cvThreshold = threshold
	}
}

// WithMinTradesForSizing sets the minimum number of trades required for analysis.
func WithMinTradesForSizing(minTrades int) SizingOption {
	return func(a *SizingAnalyzer) {
		a.minTrades = minTrades
	}
}

// NewSizingAnalyzer creates a new SizingAnalyzer with optional configuration.
func NewSizingAnalyzer(opts ...SizingOption) *SizingAnalyzer {
	a := &SizingAnalyzer{
		cvThreshold: DefaultCVThreshold,
		minTrades:   DefaultMinTradesForSizing,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns the detector identifier.
func (a *SizingAnalyzer) Name() string {
	return "sizing"
}

// Weight returns the detector weight for composite scoring.
func (a *SizingAnalyzer) Weight() float64 {
	return SizingWeight
}

// SizingStats contains statistical analysis of trade sizes.
type SizingStats struct {
	// Sizes contains all trade sizes analyzed.
	Sizes []float64
	// Mean is the average trade size.
	Mean float64
	// StdDev is the standard deviation of trade sizes.
	StdDev float64
	// Min is the minimum trade size.
	Min float64
	// Max is the maximum trade size.
	Max float64
	// TradeCount is the number of trades analyzed.
	TradeCount int
}

// CoefficientOfVariation returns the CV (std dev / mean).
// Lower values indicate more consistent sizing.
func (s *SizingStats) CoefficientOfVariation() float64 {
	if s.Mean == 0 {
		return 0
	}
	return s.StdDev / s.Mean
}

// RoundNumberRatio returns the proportion of trade sizes that are "round numbers".
// Round numbers are multiples of 5, 10, 25, 50, or 100 (whole numbers).
func (s *SizingStats) RoundNumberRatio() float64 {
	if len(s.Sizes) == 0 {
		return 0
	}

	roundCount := 0
	for _, size := range s.Sizes {
		if isRoundNumber(size) {
			roundCount++
		}
	}
	return float64(roundCount) / float64(len(s.Sizes))
}

// isRoundNumber checks if a number is a "round" algorithmic value.
// Round numbers are whole numbers that are multiples of 5.
func isRoundNumber(n float64) bool {
	// Check if it's a whole number
	if n != math.Floor(n) {
		return false
	}

	// Check if divisible by 5
	intVal := int64(n)
	return intVal%5 == 0
}

// calculateSizingStats computes statistical measures for trade sizes.
func calculateSizingStats(trades models.TradeList) SizingStats {
	stats := SizingStats{}

	if len(trades) == 0 {
		return stats
	}

	stats.TradeCount = len(trades)
	stats.Sizes = make([]float64, len(trades))

	// First pass: calculate mean, min, max
	var total float64
	stats.Min = trades[0].Size
	stats.Max = trades[0].Size
	for i, trade := range trades {
		stats.Sizes[i] = trade.Size
		total += trade.Size
		if trade.Size < stats.Min {
			stats.Min = trade.Size
		}
		if trade.Size > stats.Max {
			stats.Max = trade.Size
		}
	}
	stats.Mean = total / float64(len(trades))

	// Second pass: calculate standard deviation
	if len(trades) > 1 {
		var sumSquaredDiff float64
		for _, size := range stats.Sizes {
			diff := size - stats.Mean
			sumSquaredDiff += diff * diff
		}
		variance := sumSquaredDiff / float64(len(trades))
		stats.StdDev = math.Sqrt(variance)
	}

	return stats
}

// Analyze examines trade sizing patterns to detect algorithmic behavior.
// It flags wallets with suspiciously consistent position sizing (low CV).
func (a *SizingAnalyzer) Analyze(positions models.PositionList, trades models.TradeList) DetectionResult {
	result := DetectionResult{
		Score:   0.0,
		Signals: []models.DetectionSignal{},
	}

	// Need minimum trades for meaningful analysis
	if len(trades) < a.minTrades {
		return result
	}

	// Calculate sizing statistics
	stats := calculateSizingStats(trades)

	// Get coefficient of variation and round number ratio
	cv := stats.CoefficientOfVariation()
	roundRatio := stats.RoundNumberRatio()

	// Calculate score based on CV and round number patterns
	score := a.calculateScore(cv, roundRatio)

	if score <= 0 {
		return result
	}

	result.Score = score

	// Create detection signal with evidence
	signal := models.DetectionSignal{
		Type:        "sizing",
		Description: "Detected algorithmic position sizing pattern with low variance",
		Severity:    a.calculateSeverity(cv, roundRatio),
		Evidence: map[string]interface{}{
			"coefficient_of_variation": cv,
			"mean_size":                stats.Mean,
			"std_dev":                  stats.StdDev,
			"min_size":                 stats.Min,
			"max_size":                 stats.Max,
			"trade_count":              stats.TradeCount,
			"round_number_ratio":       roundRatio,
			"threshold":                a.cvThreshold,
		},
	}
	result.Signals = append(result.Signals, signal)

	return result
}

// calculateScore determines the sizing score based on statistics.
func (a *SizingAnalyzer) calculateScore(cv, roundRatio float64) float64 {
	// Very low CV (uniform sizing) is extremely suspicious
	if cv <= a.cvThreshold/2 {
		return 1.0
	}

	// Low CV combined with high round number ratio is suspicious
	if cv <= a.cvThreshold && roundRatio >= 0.7 {
		return 1.0
	}

	// Just below threshold
	if cv <= a.cvThreshold {
		// Scale based on how close to threshold
		return 1.0 - (cv/a.cvThreshold)*0.5
	}

	// High round number ratio alone is moderately suspicious
	// (even with higher CV, using only round numbers suggests automation)
	// This catches patterns like alternating between two round amounts (100, 50)
	if roundRatio >= 0.9 && cv <= a.cvThreshold*4 {
		return 0.6 + (0.4 * (1 - cv/(a.cvThreshold*4)))
	}

	// Very high round number ratio with moderate CV
	if roundRatio >= 0.8 && cv <= a.cvThreshold*3 {
		return 0.5
	}

	return 0
}

// calculateSeverity determines severity based on CV and round number ratio.
func (a *SizingAnalyzer) calculateSeverity(cv, roundRatio float64) string {
	// High severity: very low CV with many round numbers
	if cv <= a.cvThreshold/3 || (cv <= a.cvThreshold/2 && roundRatio >= 0.8) {
		return "high"
	}

	// Medium severity: low CV below threshold
	if cv <= a.cvThreshold*0.7 {
		return "medium"
	}

	// Low severity: borderline cases
	return "low"
}
