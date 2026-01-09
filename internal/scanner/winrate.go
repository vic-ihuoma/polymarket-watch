// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// DefaultWinRateThreshold is the win rate above which trading is considered suspicious.
// A 95% win rate over 100+ trades is statistically improbable for random trading.
const DefaultWinRateThreshold = 0.95

// DefaultMinTradesForWinRate is the minimum number of trades required for win rate analysis.
// Win rate is only meaningful with sufficient sample size.
const DefaultMinTradesForWinRate = 100

// WinRateWeight is the weight of the win rate analyzer in the composite score.
// This weight matches the PRD detection_algorithms specification.
const WinRateWeight = 0.25

// WinRateAnalyzer analyzes trading win rates to detect statistically improbable patterns.
// Bots using arbitrage or high-frequency strategies often achieve win rates that are
// impossible for human traders to sustain over large sample sizes.
type WinRateAnalyzer struct {
	winRateThreshold float64
	minTrades        int
}

// WinRateOption is a functional option for configuring WinRateAnalyzer.
type WinRateOption func(*WinRateAnalyzer)

// WithWinRateThreshold sets the minimum win rate to flag as suspicious.
func WithWinRateThreshold(threshold float64) WinRateOption {
	return func(a *WinRateAnalyzer) {
		a.winRateThreshold = threshold
	}
}

// WithMinTradesForWinRate sets the minimum number of trades required for analysis.
func WithMinTradesForWinRate(minTrades int) WinRateOption {
	return func(a *WinRateAnalyzer) {
		a.minTrades = minTrades
	}
}

// NewWinRateAnalyzer creates a new WinRateAnalyzer with optional configuration.
func NewWinRateAnalyzer(opts ...WinRateOption) *WinRateAnalyzer {
	a := &WinRateAnalyzer{
		winRateThreshold: DefaultWinRateThreshold,
		minTrades:        DefaultMinTradesForWinRate,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Name returns the detector identifier.
func (a *WinRateAnalyzer) Name() string {
	return "winrate"
}

// Weight returns the detector weight for composite scoring.
func (a *WinRateAnalyzer) Weight() float64 {
	return WinRateWeight
}

// WinRateStats contains win rate analysis statistics.
type WinRateStats struct {
	// WinRate is the ratio of profitable trades (0.0-1.0).
	WinRate float64
	// TotalTrades is the total number of complete round-trip trades.
	TotalTrades int
	// ProfitableTrades is the number of trades that were profitable.
	ProfitableTrades int
	// LossTrades is the number of trades that resulted in loss.
	LossTrades int
	// TotalProfit is the sum of profits from winning trades.
	TotalProfit float64
	// TotalLoss is the sum of losses from losing trades.
	TotalLoss float64
}

// IsImprobable returns true if the win rate is above the threshold.
func (s *WinRateStats) IsImprobable(threshold float64) bool {
	return s.WinRate >= threshold
}

// calculateWinRateStats analyzes trades to calculate win rate statistics.
// A trade is considered profitable if the exit price is higher than entry price
// for BUY positions (or lower for SELL/short positions).
func calculateWinRateStats(trades models.TradeList) WinRateStats {
	stats := WinRateStats{}

	if len(trades) == 0 {
		return stats
	}

	// Group trades by market and outcome to find matching BUY/SELL pairs
	type positionKey struct {
		market  string
		outcome string
	}

	// Track open positions: key -> list of buy trades
	openPositions := make(map[positionKey][]models.Trade)

	// Process trades chronologically
	sortedTrades := make(models.TradeList, len(trades))
	copy(sortedTrades, trades)
	sortedTrades.SortByTime()

	for _, trade := range sortedTrades {
		key := positionKey{
			market:  trade.Market,
			outcome: trade.Outcome,
		}

		if trade.IsBuy() {
			// Opening a position
			openPositions[key] = append(openPositions[key], trade)
		} else if trade.IsSell() {
			// Closing a position - match with earliest open position (FIFO)
			if opens, ok := openPositions[key]; ok && len(opens) > 0 {
				openTrade := opens[0]
				openPositions[key] = opens[1:]

				// Calculate profit/loss
				// For a BUY then SELL: profit = (sellPrice - buyPrice) * size
				profit := (trade.Price - openTrade.Price) * trade.Size

				stats.TotalTrades++
				if profit > 0 {
					stats.ProfitableTrades++
					stats.TotalProfit += profit
				} else {
					stats.LossTrades++
					stats.TotalLoss += -profit
				}
			}
		}
	}

	// Calculate win rate
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(stats.ProfitableTrades) / float64(stats.TotalTrades)
	}

	return stats
}

// Analyze examines trade win rates to detect statistically improbable patterns.
// It flags wallets with win rates above 95% over 100+ trades.
func (a *WinRateAnalyzer) Analyze(positions models.PositionList, trades models.TradeList) DetectionResult {
	result := DetectionResult{
		Score:   0.0,
		Signals: []models.DetectionSignal{},
	}

	// Need minimum trades for meaningful analysis
	if len(trades) < a.minTrades {
		return result
	}

	// Calculate win rate statistics
	stats := calculateWinRateStats(trades)

	// Need minimum completed round-trips
	if stats.TotalTrades < a.minTrades {
		return result
	}

	// Check if win rate is suspiciously high
	if stats.WinRate < a.winRateThreshold {
		return result
	}

	// Calculate score based on how much the win rate exceeds threshold
	result.Score = a.calculateScore(stats)

	// Create detection signal with evidence
	signal := models.DetectionSignal{
		Type:        "winrate",
		Description: "Detected statistically improbable win rate suggesting automated or insider trading",
		Severity:    a.calculateSeverity(stats.WinRate, stats.TotalTrades),
		Evidence: map[string]interface{}{
			"win_rate":          stats.WinRate,
			"total_trades":      stats.TotalTrades,
			"profitable_trades": stats.ProfitableTrades,
			"loss_trades":       stats.LossTrades,
			"total_profit":      stats.TotalProfit,
			"total_loss":        stats.TotalLoss,
			"threshold":         a.winRateThreshold,
		},
	}
	result.Signals = append(result.Signals, signal)

	return result
}

// calculateScore determines the score based on win rate statistics.
func (a *WinRateAnalyzer) calculateScore(stats WinRateStats) float64 {
	// Score increases with:
	// 1. How much win rate exceeds threshold
	// 2. Number of trades (higher sample size = more significant)

	// Base: win rate above threshold
	excessRate := stats.WinRate - a.winRateThreshold

	// Perfect or near-perfect win rate is extremely suspicious
	if stats.WinRate >= 0.98 {
		return 1.0
	}

	// Scale score based on excess rate
	// Each 1% above threshold adds ~0.2 to score
	score := excessRate * 20

	// Boost score for larger sample sizes
	// More trades with high win rate = more suspicious
	if stats.TotalTrades >= 200 {
		score += 0.2
	}

	// Cap at 1.0
	if score > 1.0 {
		return 1.0
	}

	return score
}

// calculateSeverity determines severity based on win rate and trade count.
func (a *WinRateAnalyzer) calculateSeverity(winRate float64, tradeCount int) string {
	// High severity: extreme win rate (>98%) or high rate with many trades
	if winRate >= 0.98 || (winRate >= 0.95 && tradeCount >= 300) {
		return "high"
	}

	// Medium severity: high win rate with moderate trades
	if winRate >= 0.90 && tradeCount >= 100 {
		return "medium"
	}

	// Low severity: borderline cases
	return "low"
}
