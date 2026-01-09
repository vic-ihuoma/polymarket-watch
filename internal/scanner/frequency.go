// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"math"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// DefaultIntervalThreshold is the default maximum interval between trades
// that is considered suspiciously fast (sub-5 seconds).
const DefaultIntervalThreshold = 5 * time.Second

// TimingWeight is the weight of the timing analyzer in the composite score.
// This weight matches the PRD detection_algorithms specification.
const TimingWeight = 0.25

// MinTradesForAnalysis is the minimum number of trades required for timing analysis.
// With fewer trades, statistical analysis is not meaningful.
const MinTradesForAnalysis = 3

// TimingAnalyzer analyzes trade interval patterns to detect automated trading.
// Bots typically execute trades at consistent, machine-like intervals that are
// impossible for humans to achieve manually.
type TimingAnalyzer struct {
	threshold time.Duration
}

// TimingOption is a functional option for configuring TimingAnalyzer.
type TimingOption func(*TimingAnalyzer)

// WithIntervalThreshold sets the maximum interval to consider as "fast" trading.
func WithIntervalThreshold(threshold time.Duration) TimingOption {
	return func(a *TimingAnalyzer) {
		a.threshold = threshold
	}
}

// NewTimingAnalyzer creates a new TimingAnalyzer with optional configuration.
func NewTimingAnalyzer(opts ...TimingOption) *TimingAnalyzer {
	a := &TimingAnalyzer{
		threshold: DefaultIntervalThreshold,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns the detector identifier.
func (a *TimingAnalyzer) Name() string {
	return "timing"
}

// Weight returns the detector weight for composite scoring.
func (a *TimingAnalyzer) Weight() float64 {
	return TimingWeight
}

// TimingStats contains statistical analysis of trade intervals.
type TimingStats struct {
	// Intervals contains all calculated time intervals between trades.
	Intervals []time.Duration
	// Mean is the average interval duration.
	Mean time.Duration
	// StdDev is the standard deviation of intervals.
	StdDev time.Duration
	// Min is the minimum interval observed.
	Min time.Duration
	// Max is the maximum interval observed.
	Max time.Duration
}

// FastTradeRatio returns the proportion of intervals below the threshold.
func (s *TimingStats) FastTradeRatio(threshold time.Duration) float64 {
	if len(s.Intervals) == 0 {
		return 0
	}
	fastCount := 0
	for _, interval := range s.Intervals {
		if interval < threshold {
			fastCount++
		}
	}
	return float64(fastCount) / float64(len(s.Intervals))
}

// calculateTimingStats computes statistical measures for a set of intervals.
func calculateTimingStats(intervals []time.Duration) TimingStats {
	stats := TimingStats{
		Intervals: intervals,
	}

	if len(intervals) == 0 {
		return stats
	}

	// Calculate mean
	var total time.Duration
	stats.Min = intervals[0]
	stats.Max = intervals[0]
	for _, interval := range intervals {
		total += interval
		if interval < stats.Min {
			stats.Min = interval
		}
		if interval > stats.Max {
			stats.Max = interval
		}
	}
	stats.Mean = total / time.Duration(len(intervals))

	// Calculate standard deviation
	if len(intervals) > 1 {
		var sumSquaredDiff float64
		meanSeconds := stats.Mean.Seconds()
		for _, interval := range intervals {
			diff := interval.Seconds() - meanSeconds
			sumSquaredDiff += diff * diff
		}
		variance := sumSquaredDiff / float64(len(intervals))
		stats.StdDev = time.Duration(math.Sqrt(variance) * float64(time.Second))
	}

	return stats
}

// Analyze examines trade timing patterns to detect bot-like behavior.
// It calculates intervals between consecutive trades and flags patterns
// that indicate automated execution (consistent sub-5s intervals).
func (a *TimingAnalyzer) Analyze(positions models.PositionList, trades models.TradeList) DetectionResult {
	result := DetectionResult{
		Score:   0.0,
		Signals: []models.DetectionSignal{},
	}

	// Need at least MinTradesForAnalysis trades to analyze timing
	if len(trades) < MinTradesForAnalysis {
		return result
	}

	// Sort trades by time (creates a copy to avoid modifying input)
	sortedTrades := make(models.TradeList, len(trades))
	copy(sortedTrades, trades)
	sortedTrades.SortByTime()

	// Calculate intervals between consecutive trades
	intervals := make([]time.Duration, 0, len(sortedTrades)-1)
	for i := 1; i < len(sortedTrades); i++ {
		interval := sortedTrades[i].MatchTime.Sub(sortedTrades[i-1].MatchTime)
		intervals = append(intervals, interval)
	}

	// Calculate timing statistics
	stats := calculateTimingStats(intervals)

	// Calculate fast trade ratio
	fastRatio := stats.FastTradeRatio(a.threshold)

	// Calculate score based on fast trade ratio and consistency
	// High score if many fast trades with low variance (bot-like)
	score := a.calculateScore(stats, fastRatio)

	if score <= 0 {
		return result
	}

	result.Score = score

	// Create detection signal with evidence
	signal := models.DetectionSignal{
		Type:        "timing",
		Description: "Detected consistent fast trading intervals suggesting automated execution",
		Severity:    a.calculateSeverity(stats, fastRatio),
		Evidence: map[string]interface{}{
			"mean_interval_seconds":   stats.Mean.Seconds(),
			"std_dev_seconds":         stats.StdDev.Seconds(),
			"min_interval_seconds":    stats.Min.Seconds(),
			"max_interval_seconds":    stats.Max.Seconds(),
			"fast_trade_ratio":        fastRatio,
			"total_intervals":         len(intervals),
			"threshold_seconds":       a.threshold.Seconds(),
			"coefficient_of_variance": a.coefficientOfVariance(stats),
		},
	}
	result.Signals = append(result.Signals, signal)

	return result
}

// calculateScore determines the timing score based on statistics.
func (a *TimingAnalyzer) calculateScore(stats TimingStats, fastRatio float64) float64 {
	// No fast trades = not bot-like
	if fastRatio == 0 {
		return 0
	}

	// Score is based on:
	// 1. Proportion of fast trades (primary factor)
	// 2. Consistency of intervals (low variance = more bot-like)
	cv := a.coefficientOfVariance(stats)

	// High variance (CV > 1.0) indicates sporadic fast trades mixed with slow ones
	// This is more indicative of human behavior with occasional quick trades
	// Not systematic bot behavior
	if cv > 1.0 {
		return 0
	}

	// If most trades are fast with reasonable consistency, it's highly suspicious
	if fastRatio >= 0.8 {
		return 1.0
	}

	// High consistency (low CV) combined with fast trades is suspicious
	// CV < 0.5 is very consistent for human behavior
	if fastRatio >= 0.5 && cv < 0.5 {
		return 1.0
	}

	// Partial scoring based on fast ratio with moderate consistency
	if fastRatio >= 0.3 && cv < 1.0 {
		return fastRatio
	}

	return 0
}

// coefficientOfVariance returns the CV (std dev / mean).
// Lower values indicate more consistent timing.
func (a *TimingAnalyzer) coefficientOfVariance(stats TimingStats) float64 {
	if stats.Mean == 0 {
		return 0
	}
	return stats.StdDev.Seconds() / stats.Mean.Seconds()
}

// calculateSeverity determines severity based on timing patterns.
func (a *TimingAnalyzer) calculateSeverity(stats TimingStats, fastRatio float64) string {
	cv := a.coefficientOfVariance(stats)

	// High severity: very consistent fast trades (CV < 0.3 and >80% fast)
	if fastRatio >= 0.8 && cv < 0.3 {
		return "high"
	}

	// Medium severity: mostly fast trades or moderately consistent
	if fastRatio >= 0.5 || (fastRatio >= 0.3 && cv < 0.5) {
		return "medium"
	}

	// Low severity: some suspicious activity but not definitive
	return "low"
}
