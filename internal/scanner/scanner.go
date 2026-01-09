// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/api"
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// Scanner orchestrates all detection algorithms and produces a composite bot score.
// It fetches trade and position data from the Polymarket API, runs all configured
// detectors, and aggregates results into a ScanReport.
type Scanner struct {
	dataAPI   *api.DataAPI
	detectors []Detector
}

// ScannerOption configures a Scanner instance.
type ScannerOption func(*Scanner)

// WithDataAPI sets a custom DataAPI client for the Scanner.
func WithDataAPI(dataAPI *api.DataAPI) ScannerOption {
	return func(s *Scanner) {
		s.dataAPI = dataAPI
	}
}

// WithDetectors replaces the default detectors with a custom set.
func WithDetectors(detectors []Detector) ScannerOption {
	return func(s *Scanner) {
		s.detectors = detectors
	}
}

// WithDetector adds a detector to the existing set.
func WithDetector(detector Detector) ScannerOption {
	return func(s *Scanner) {
		s.detectors = append(s.detectors, detector)
	}
}

// NewScanner creates a new Scanner with default or custom configuration.
// By default, it initializes all four detection algorithms:
// - ArbitrageDetector: detects simultaneous opposing positions
// - TimingAnalyzer: analyzes trade interval patterns
// - WinRateAnalyzer: flags statistically improbable win rates
// - SizingAnalyzer: detects algorithmic position sizing
func NewScanner(opts ...ScannerOption) *Scanner {
	s := &Scanner{
		dataAPI: api.NewDataAPI(),
		detectors: []Detector{
			NewArbitrageDetector(),
			NewTimingAnalyzer(),
			NewWinRateAnalyzer(),
			NewSizingAnalyzer(),
		},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// DetectorWeights returns a map of detector names to their weights.
// This is used for computing the composite bot score.
func (s *Scanner) DetectorWeights() map[string]float64 {
	weights := make(map[string]float64)
	for _, detector := range s.detectors {
		weights[detector.Name()] = detector.Weight()
	}
	return weights
}

// Scan fetches data for a wallet and runs all detection algorithms.
// It returns a ScanReport with the composite bot score and detection signals.
func (s *Scanner) Scan(ctx context.Context, wallet string) (*models.ScanReport, error) {
	// Fetch trades
	trades, err := s.dataAPI.GetAllTrades(ctx, wallet)
	if err != nil {
		return nil, fmt.Errorf("fetching trades: %w", err)
	}

	// Fetch positions
	positions, err := s.dataAPI.GetAllPositions(ctx, wallet)
	if err != nil {
		return nil, fmt.Errorf("fetching positions: %w", err)
	}

	// Analyze the data
	report := s.AnalyzeData(wallet, positions, trades)

	return report, nil
}

// AnalyzeData runs all detection algorithms on the provided data.
// This method is useful for testing or when data is already available.
func (s *Scanner) AnalyzeData(wallet string, positions models.PositionList, trades models.TradeList) *models.ScanReport {
	report := models.NewScanReport(wallet)

	// Run all detectors
	for _, detector := range s.detectors {
		result := detector.Analyze(positions, trades)
		report.AddScore(detector.Name(), result.Score)
		for _, signal := range result.Signals {
			report.AddSignal(signal)
		}
	}

	// Compute composite score using detector weights
	report.BotScore = report.ComputeScore(s.DetectorWeights())

	// Calculate statistics
	report.Stats = s.calculateStats(positions, trades)

	return report
}

// calculateStats computes trading statistics for the report.
func (s *Scanner) calculateStats(positions models.PositionList, trades models.TradeList) models.ScanStats {
	stats := models.ScanStats{
		TotalTrades:    len(trades),
		TotalPositions: len(positions),
	}

	// Calculate unique markets and total volume from trades
	marketSet := make(map[string]struct{})
	var totalVolume float64
	var firstTime, lastTime time.Time

	for _, trade := range trades {
		marketSet[trade.Market] = struct{}{}
		totalVolume += trade.Value()

		if firstTime.IsZero() || trade.MatchTime.Before(firstTime) {
			firstTime = trade.MatchTime
		}
		if trade.MatchTime.After(lastTime) {
			lastTime = trade.MatchTime
		}
	}

	stats.UniqueMarkets = len(marketSet)
	stats.TotalVolume = totalVolume
	stats.FirstTradeTime = firstTime
	stats.LastTradeTime = lastTime

	// Calculate average position size
	if len(positions) > 0 {
		var totalSize float64
		for _, pos := range positions {
			totalSize += pos.Size
		}
		stats.AvgPositionSize = totalSize / float64(len(positions))
	}

	// Count opposing pairs
	stats.OpposingPairCount = len(positions.FindOpposingPositions())

	return stats
}
