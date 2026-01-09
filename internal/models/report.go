// Package models defines data structures for Polymarket API responses.
package models

import "time"

// ScanReport represents the results of scanning a wallet for bot activity.
type ScanReport struct {
	// WalletAddress is the wallet that was scanned.
	WalletAddress string `json:"wallet_address"`
	// ScanTime is when the scan was performed.
	ScanTime time.Time `json:"scan_time"`
	// BotScore is the composite bot probability score (0-100).
	BotScore int `json:"bot_score"`
	// Scores contains individual detector scores (0.0-1.0).
	Scores map[string]float64 `json:"scores"`
	// Signals contains detection evidence and explanations.
	Signals []DetectionSignal `json:"signals"`
	// Stats contains trading statistics for the wallet.
	Stats ScanStats `json:"stats"`
}

// DetectionSignal represents evidence of bot-like behavior.
type DetectionSignal struct {
	// Type identifies the detection algorithm (e.g., "arbitrage", "timing").
	Type string `json:"type"`
	// Description explains what was detected.
	Description string `json:"description"`
	// Severity indicates the signal strength: "low", "medium", or "high".
	Severity string `json:"severity"`
	// Evidence contains supporting data for the signal.
	Evidence map[string]interface{} `json:"evidence,omitempty"`
}

// ScanStats contains trading statistics for a wallet.
type ScanStats struct {
	// TotalTrades is the total number of trades analyzed.
	TotalTrades int `json:"total_trades"`
	// TotalPositions is the number of open positions.
	TotalPositions int `json:"total_positions"`
	// TotalVolume is the total trading volume in USD.
	TotalVolume float64 `json:"total_volume"`
	// UniqueMarkets is the number of distinct markets traded.
	UniqueMarkets int `json:"unique_markets"`
	// FirstTradeTime is the timestamp of the earliest trade.
	FirstTradeTime time.Time `json:"first_trade_time"`
	// LastTradeTime is the timestamp of the most recent trade.
	LastTradeTime time.Time `json:"last_trade_time"`
	// WinRate is the percentage of profitable trades (0.0-1.0).
	WinRate float64 `json:"win_rate"`
	// AvgPositionSize is the average position size in shares.
	AvgPositionSize float64 `json:"avg_position_size"`
	// OpposingPairCount is the number of Yes/No position pairs.
	OpposingPairCount int `json:"opposing_pair_count"`
}

// NewScanReport creates a new ScanReport for the given wallet address.
func NewScanReport(wallet string) *ScanReport {
	return &ScanReport{
		WalletAddress: wallet,
		ScanTime:      time.Now(),
		Scores:        make(map[string]float64),
		Signals:       make([]DetectionSignal, 0),
	}
}

// AddScore adds a detector score to the report.
func (r *ScanReport) AddScore(detector string, score float64) {
	r.Scores[detector] = score
}

// AddSignal adds a detection signal to the report.
func (r *ScanReport) AddSignal(signal DetectionSignal) {
	r.Signals = append(r.Signals, signal)
}

// ComputeScore calculates the composite bot score using weighted averaging.
// Weights should sum to 1.0 for proper scaling.
// Returns a score from 0-100.
func (r *ScanReport) ComputeScore(weights map[string]float64) int {
	var total float64
	for detector, score := range r.Scores {
		weight, ok := weights[detector]
		if ok {
			total += score * weight
		}
	}
	// Convert to 0-100 scale
	return int(total * 100)
}

// IsProbableBot returns true if the bot score meets or exceeds the threshold.
func (r *ScanReport) IsProbableBot(threshold int) bool {
	return r.BotScore >= threshold
}

// Severity returns a severity classification based on the bot score.
// Returns "low" (<50), "medium" (50-79), or "high" (>=80).
func (r *ScanReport) Severity() string {
	switch {
	case r.BotScore >= 80:
		return "high"
	case r.BotScore >= 50:
		return "medium"
	default:
		return "low"
	}
}

// TradingDuration returns the time span between first and last trade.
func (s *ScanStats) TradingDuration() time.Duration {
	if s.FirstTradeTime.IsZero() || s.LastTradeTime.IsZero() {
		return 0
	}
	return s.LastTradeTime.Sub(s.FirstTradeTime)
}

// TradesPerDay returns the average number of trades per day.
func (s *ScanStats) TradesPerDay() float64 {
	if s.TotalTrades == 0 {
		return 0
	}
	duration := s.TradingDuration()
	days := duration.Hours() / 24
	if days < 1 {
		return float64(s.TotalTrades)
	}
	return float64(s.TotalTrades) / days
}
