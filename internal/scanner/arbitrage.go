// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// DefaultSpreadThreshold is the minimum spread required to flag as arbitrage.
// A spread of 0 means the sum of yes and no prices equals 1, which is break-even.
// Any positive spread indicates potential profit from opposing positions.
const DefaultSpreadThreshold = 0.0

// ArbitrageWeight is the weight of the arbitrage detector in the composite score.
// This weight matches the PRD detection_algorithms specification.
const ArbitrageWeight = 0.35

// Detector is the interface that all detection algorithms must implement.
type Detector interface {
	// Name returns the identifier for this detector.
	Name() string
	// Weight returns the weight for this detector in the composite score.
	Weight() float64
	// Analyze analyzes positions and trades, returning detection results.
	Analyze(positions models.PositionList, trades models.TradeList) DetectionResult
}

// DetectionResult contains the output of a detection algorithm.
type DetectionResult struct {
	// Score is the detection score (0.0-1.0).
	Score float64
	// Signals contains detection evidence.
	Signals []models.DetectionSignal
}

// ArbitrageDetector detects simultaneous opposing positions on the same market.
// This is a key indicator of automated spread arbitrage strategies like Account88888's
// UP/DOWN farming where bots buy both Yes and No positions when prices sum to <$1.
type ArbitrageDetector struct {
	threshold float64
}

// ArbitrageOption is a functional option for configuring ArbitrageDetector.
type ArbitrageOption func(*ArbitrageDetector)

// WithSpreadThreshold sets the minimum spread threshold for flagging arbitrage.
func WithSpreadThreshold(threshold float64) ArbitrageOption {
	return func(d *ArbitrageDetector) {
		d.threshold = threshold
	}
}

// NewArbitrageDetector creates a new ArbitrageDetector with optional configuration.
func NewArbitrageDetector(opts ...ArbitrageOption) *ArbitrageDetector {
	d := &ArbitrageDetector{
		threshold: DefaultSpreadThreshold,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Name returns the detector identifier.
func (d *ArbitrageDetector) Name() string {
	return "arbitrage"
}

// Weight returns the detector weight for composite scoring.
func (d *ArbitrageDetector) Weight() float64 {
	return ArbitrageWeight
}

// ProfitablePair represents an opposing position pair with a profitable spread.
type ProfitablePair struct {
	// ConditionID is the market condition ID.
	ConditionID string
	// YesPrice is the average entry price for the Yes position.
	YesPrice float64
	// NoPrice is the average entry price for the No position.
	NoPrice float64
	// YesSize is the size of the Yes position.
	YesSize float64
	// NoSize is the size of the No position.
	NoSize float64
	// Spread is the profit margin (1 - yesPrice - noPrice).
	Spread float64
}

// ArbitrageResult contains detailed arbitrage analysis results.
type ArbitrageResult struct {
	// ProfitablePairs contains all opposing position pairs with positive spread.
	ProfitablePairs []ProfitablePair
	// TotalPairs is the total number of opposing position pairs analyzed.
	TotalPairs int
}

// TotalSpread returns the sum of all profitable pair spreads.
func (r *ArbitrageResult) TotalSpread() float64 {
	var total float64
	for _, pair := range r.ProfitablePairs {
		total += pair.Spread
	}
	return total
}

// Analyze examines positions for arbitrage patterns.
// It identifies opposing Yes/No positions on the same market and calculates
// whether the entry prices create a profitable spread (sum < $1).
func (d *ArbitrageDetector) Analyze(positions models.PositionList, trades models.TradeList) DetectionResult {
	result := DetectionResult{
		Score:   0.0,
		Signals: []models.DetectionSignal{},
	}

	if len(positions) == 0 {
		return result
	}

	// Find all opposing position pairs
	opposingPairs := positions.FindOpposingPositions()
	if len(opposingPairs) == 0 {
		return result
	}

	// Analyze each pair for profitable spreads
	arbResult := ArbitrageResult{
		TotalPairs: len(opposingPairs),
	}

	for _, pair := range opposingPairs {
		spread := pair.Spread()
		if spread > d.threshold {
			arbResult.ProfitablePairs = append(arbResult.ProfitablePairs, ProfitablePair{
				ConditionID: pair.ConditionID,
				YesPrice:    pair.YesPosition.AvgPrice,
				NoPrice:     pair.NoPosition.AvgPrice,
				YesSize:     pair.YesPosition.Size,
				NoSize:      pair.NoPosition.Size,
				Spread:      spread,
			})
		}
	}

	// Calculate score based on profitable pairs
	if len(arbResult.ProfitablePairs) == 0 {
		return result
	}

	// Score is 1.0 if any profitable arbitrage pairs are found
	// The presence of any spread arbitrage position strongly indicates bot activity
	result.Score = 1.0

	// Create detection signal with evidence
	signal := models.DetectionSignal{
		Type:        "arbitrage",
		Description: "Detected simultaneous opposing positions with profitable spread",
		Severity:    d.scoreSeverity(len(arbResult.ProfitablePairs), arbResult.TotalSpread()),
		Evidence: map[string]interface{}{
			"profitable_pairs": len(arbResult.ProfitablePairs),
			"total_pairs":      arbResult.TotalPairs,
			"total_spread":     arbResult.TotalSpread(),
			"pairs":            arbResult.ProfitablePairs,
		},
	}
	result.Signals = append(result.Signals, signal)

	return result
}

// scoreSeverity determines the severity based on number of pairs and total spread.
func (d *ArbitrageDetector) scoreSeverity(pairCount int, totalSpread float64) string {
	// High severity: Multiple profitable pairs or large total spread
	if pairCount >= 3 || totalSpread >= 0.5 {
		return "high"
	}
	// Medium severity: At least 2 pairs or moderate spread
	if pairCount >= 2 || totalSpread >= 0.2 {
		return "medium"
	}
	// Low severity: Single pair with small spread
	return "low"
}
