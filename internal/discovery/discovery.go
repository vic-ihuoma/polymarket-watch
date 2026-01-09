// Package discovery provides wallet discovery functionality for Polymarket.
// It enables finding potential bot wallets by scanning holders of top markets.
package discovery

import (
	"errors"
	"sort"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// Default configuration values for discovery operations.
const (
	// DefaultHoldersLimit is the default number of holders to fetch per market.
	DefaultHoldersLimit = 20
	// DefaultConcurrency is the default number of concurrent wallet scans.
	DefaultConcurrency = 3
	// DefaultSortBy is the default field to sort markets by for auto-discovery.
	DefaultSortBy = "volume"
	// DefaultBotThreshold is the default bot score threshold (0-100).
	DefaultBotThreshold = 80
)

// DiscoveryOptions configures the discovery operation.
type DiscoveryOptions struct {
	// MarketIDs is a list of specific market condition IDs to scan.
	MarketIDs []string `json:"market_ids,omitempty"`
	// MarketSlugs is a list of market slugs to resolve and scan.
	MarketSlugs []string `json:"market_slugs,omitempty"`
	// AutoDiscover enables automatic discovery of top markets.
	AutoDiscover bool `json:"auto_discover"`
	// TopN is the number of top markets to scan when AutoDiscover is true.
	TopN int `json:"top_n,omitempty"`
	// SortBy specifies how to sort markets for auto-discovery ("volume", "liquidity", "volume24hr").
	SortBy string `json:"sort_by,omitempty"`
	// MinVolume is the minimum volume threshold for auto-discovered markets.
	MinVolume float64 `json:"min_volume,omitempty"`
	// MinVolume24hr is the minimum 24-hour volume threshold for auto-discovered markets.
	MinVolume24hr float64 `json:"min_volume_24hr,omitempty"`
	// MinLiquidity is the minimum liquidity threshold for auto-discovered markets.
	MinLiquidity float64 `json:"min_liquidity,omitempty"`
	// HoldersLimit is the maximum number of holders to fetch per market (max 20).
	HoldersLimit int `json:"holders_limit,omitempty"`
	// Concurrency is the number of concurrent wallet scans to perform.
	Concurrency int `json:"concurrency,omitempty"`
	// NoScan skips wallet scanning and only collects holder information.
	NoScan bool `json:"no_scan"`
	// BotThreshold is the bot score threshold for flagging wallets as bots (0-100).
	BotThreshold int `json:"bot_threshold,omitempty"`
}

// Validate checks if the options are valid.
// Returns an error if the configuration is invalid.
func (o *DiscoveryOptions) Validate() error {
	hasMarketIDs := len(o.MarketIDs) > 0
	hasMarketSlugs := len(o.MarketSlugs) > 0

	// Check for at least one input source
	if !hasMarketIDs && !hasMarketSlugs && !o.AutoDiscover {
		return errors.New("must specify MarketIDs, MarketSlugs, or AutoDiscover")
	}

	// If AutoDiscover is enabled, TopN must be specified
	if o.AutoDiscover && o.TopN <= 0 {
		return errors.New("TopN must be > 0 when AutoDiscover is true")
	}

	return nil
}

// ApplyDefaults sets default values for unspecified options.
func (o *DiscoveryOptions) ApplyDefaults() {
	if o.HoldersLimit <= 0 {
		o.HoldersLimit = DefaultHoldersLimit
	}
	if o.Concurrency <= 0 {
		o.Concurrency = DefaultConcurrency
	}
	if o.SortBy == "" {
		o.SortBy = DefaultSortBy
	}
	if o.BotThreshold <= 0 {
		o.BotThreshold = DefaultBotThreshold
	}
}

// DiscoveredWallet represents a wallet found during discovery.
type DiscoveredWallet struct {
	// Address is the wallet address.
	Address string `json:"address"`
	// TotalAmount is the total position size across all discovered markets.
	TotalAmount float64 `json:"total_amount"`
	// MarketCount is the number of markets this wallet was found in.
	MarketCount int `json:"market_count"`
	// PositionCount is the total number of positions held.
	PositionCount int `json:"position_count"`
	// ScanReport contains the bot detection results (nil if NoScan was true).
	ScanReport *models.ScanReport `json:"scan_report,omitempty"`
}

// MarketInfo contains information about a scanned market.
type MarketInfo struct {
	// ConditionID is the market's condition ID.
	ConditionID string `json:"condition_id"`
	// Slug is the URL-friendly market identifier.
	Slug string `json:"slug"`
	// Question is the market question.
	Question string `json:"question"`
	// HoldersCount is the number of holders found for this market.
	HoldersCount int `json:"holders_count"`
	// Volume is the total trading volume.
	Volume float64 `json:"volume"`
	// Liquidity is the market liquidity.
	Liquidity float64 `json:"liquidity"`
}

// DiscoveryStats contains statistics about the discovery operation.
type DiscoveryStats struct {
	// MarketsScanned is the number of markets that were scanned.
	MarketsScanned int `json:"markets_scanned"`
	// WalletsFound is the total number of unique wallets discovered.
	WalletsFound int `json:"wallets_found"`
	// WalletsScanned is the number of wallets that were scanned for bot activity.
	WalletsScanned int `json:"wallets_scanned"`
	// BotsDetected is the number of wallets flagged as probable bots.
	BotsDetected int `json:"bots_detected"`
	// ScanErrors is the number of wallet scans that failed.
	ScanErrors int `json:"scan_errors"`
	// StartTime is when the discovery operation started.
	StartTime time.Time `json:"start_time"`
	// EndTime is when the discovery operation completed.
	EndTime time.Time `json:"end_time"`
}

// Duration returns the total duration of the discovery operation.
func (s *DiscoveryStats) Duration() time.Duration {
	if s.StartTime.IsZero() || s.EndTime.IsZero() {
		return 0
	}
	return s.EndTime.Sub(s.StartTime)
}

// BotDetectionRate returns the percentage of scanned wallets that were detected as bots.
func (s *DiscoveryStats) BotDetectionRate() float64 {
	if s.WalletsScanned == 0 {
		return 0
	}
	return float64(s.BotsDetected) / float64(s.WalletsScanned)
}

// DiscoveryResult contains the results of a discovery operation.
type DiscoveryResult struct {
	// Wallets is the list of discovered wallets with their analysis.
	Wallets []DiscoveredWallet `json:"wallets"`
	// Markets contains information about the scanned markets.
	Markets []MarketInfo `json:"markets"`
	// Stats contains discovery statistics.
	Stats DiscoveryStats `json:"stats"`
}

// NewDiscoveryResult creates a new DiscoveryResult with initialized fields.
func NewDiscoveryResult() *DiscoveryResult {
	return &DiscoveryResult{
		Wallets: make([]DiscoveredWallet, 0),
		Markets: make([]MarketInfo, 0),
		Stats: DiscoveryStats{
			StartTime: time.Now(),
		},
	}
}

// AddWallet adds a discovered wallet to the result.
func (r *DiscoveryResult) AddWallet(wallet DiscoveredWallet) {
	r.Wallets = append(r.Wallets, wallet)
}

// GetBots returns all wallets with a bot score meeting or exceeding the threshold.
// Wallets without a scan report are excluded.
func (r *DiscoveryResult) GetBots(threshold int) []DiscoveredWallet {
	var bots []DiscoveredWallet
	for _, w := range r.Wallets {
		if w.ScanReport != nil && w.ScanReport.BotScore >= threshold {
			bots = append(bots, w)
		}
	}
	return bots
}

// Finalize sets the end time and finalizes the discovery stats.
func (r *DiscoveryResult) Finalize() {
	r.Stats.EndTime = time.Now()
}

// SortByBotScore sorts wallets by bot score in descending order.
// Wallets without scan reports are sorted to the end.
func (r *DiscoveryResult) SortByBotScore() {
	sort.Slice(r.Wallets, func(i, j int) bool {
		// Handle nil scan reports (put them last)
		if r.Wallets[i].ScanReport == nil && r.Wallets[j].ScanReport == nil {
			return false
		}
		if r.Wallets[i].ScanReport == nil {
			return false
		}
		if r.Wallets[j].ScanReport == nil {
			return true
		}
		// Sort by bot score descending
		return r.Wallets[i].ScanReport.BotScore > r.Wallets[j].ScanReport.BotScore
	})
}
