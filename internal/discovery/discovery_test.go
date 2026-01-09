package discovery

import (
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestDiscoveryOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    DiscoveryOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_with_market_ids",
			opts: DiscoveryOptions{
				MarketIDs: []string{"0x123", "0x456"},
			},
			wantErr: false,
		},
		{
			name: "valid_with_slugs",
			opts: DiscoveryOptions{
				MarketSlugs: []string{"will-trump-win", "will-biden-win"},
			},
			wantErr: false,
		},
		{
			name: "valid_with_auto_discover",
			opts: DiscoveryOptions{
				AutoDiscover: true,
				TopN:         10,
			},
			wantErr: false,
		},
		{
			name: "invalid_auto_discover_without_top_n",
			opts: DiscoveryOptions{
				AutoDiscover: true,
				TopN:         0,
			},
			wantErr: true,
			errMsg:  "TopN must be > 0 when AutoDiscover is true",
		},
		{
			name: "invalid_no_input_specified",
			opts: DiscoveryOptions{
				AutoDiscover: false,
			},
			wantErr: true,
			errMsg:  "must specify MarketIDs, MarketSlugs, or AutoDiscover",
		},
		{
			name: "valid_with_all_options",
			opts: DiscoveryOptions{
				MarketIDs:    []string{"0x123"},
				HoldersLimit: 20,
				Concurrency:  5,
				NoScan:       false,
			},
			wantErr: false,
		},
		{
			name: "defaults_applied_for_zero_concurrency",
			opts: DiscoveryOptions{
				MarketIDs:   []string{"0x123"},
				Concurrency: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDiscoveryOptions_ApplyDefaults(t *testing.T) {
	tests := []struct {
		name          string
		opts          DiscoveryOptions
		wantHolders   int
		wantConcur    int
		wantSortBy    string
		wantBotThresh int
	}{
		{
			name:          "applies_all_defaults",
			opts:          DiscoveryOptions{},
			wantHolders:   DefaultHoldersLimit,
			wantConcur:    DefaultConcurrency,
			wantSortBy:    DefaultSortBy,
			wantBotThresh: DefaultBotThreshold,
		},
		{
			name: "preserves_custom_values",
			opts: DiscoveryOptions{
				HoldersLimit: 10,
				Concurrency:  3,
				SortBy:       "liquidity",
				BotThreshold: 90,
			},
			wantHolders:   10,
			wantConcur:    3,
			wantSortBy:    "liquidity",
			wantBotThresh: 90,
		},
		{
			name: "only_applies_missing_defaults",
			opts: DiscoveryOptions{
				HoldersLimit: 15,
				Concurrency:  0,
			},
			wantHolders:   15,
			wantConcur:    DefaultConcurrency,
			wantSortBy:    DefaultSortBy,
			wantBotThresh: DefaultBotThreshold,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.opts.ApplyDefaults()

			if tt.opts.HoldersLimit != tt.wantHolders {
				t.Errorf("HoldersLimit = %d, want %d", tt.opts.HoldersLimit, tt.wantHolders)
			}
			if tt.opts.Concurrency != tt.wantConcur {
				t.Errorf("Concurrency = %d, want %d", tt.opts.Concurrency, tt.wantConcur)
			}
			if tt.opts.SortBy != tt.wantSortBy {
				t.Errorf("SortBy = %s, want %s", tt.opts.SortBy, tt.wantSortBy)
			}
			if tt.opts.BotThreshold != tt.wantBotThresh {
				t.Errorf("BotThreshold = %d, want %d", tt.opts.BotThreshold, tt.wantBotThresh)
			}
		})
	}
}

func TestDiscoveredWallet(t *testing.T) {
	t.Run("new_discovered_wallet", func(t *testing.T) {
		wallet := DiscoveredWallet{
			Address:       "0x1234567890abcdef1234567890abcdef12345678",
			TotalAmount:   1000.50,
			MarketCount:   3,
			PositionCount: 5,
		}

		if wallet.Address != "0x1234567890abcdef1234567890abcdef12345678" {
			t.Errorf("Address = %s, want %s", wallet.Address, "0x1234567890abcdef1234567890abcdef12345678")
		}
		if wallet.TotalAmount != 1000.50 {
			t.Errorf("TotalAmount = %f, want %f", wallet.TotalAmount, 1000.50)
		}
		if wallet.MarketCount != 3 {
			t.Errorf("MarketCount = %d, want %d", wallet.MarketCount, 3)
		}
		if wallet.PositionCount != 5 {
			t.Errorf("PositionCount = %d, want %d", wallet.PositionCount, 5)
		}
	})

	t.Run("with_scan_report", func(t *testing.T) {
		report := models.NewScanReport("0x1234")
		report.BotScore = 85

		wallet := DiscoveredWallet{
			Address:    "0x1234",
			ScanReport: report,
		}

		if wallet.ScanReport == nil {
			t.Error("ScanReport is nil, expected non-nil")
		}
		if wallet.ScanReport.BotScore != 85 {
			t.Errorf("BotScore = %d, want %d", wallet.ScanReport.BotScore, 85)
		}
	})
}

func TestDiscoveryStats(t *testing.T) {
	t.Run("new_discovery_stats", func(t *testing.T) {
		now := time.Now()
		stats := DiscoveryStats{
			MarketsScanned: 10,
			WalletsFound:   50,
			WalletsScanned: 45,
			BotsDetected:   5,
			ScanErrors:     2,
			StartTime:      now.Add(-5 * time.Minute),
			EndTime:        now,
		}

		if stats.MarketsScanned != 10 {
			t.Errorf("MarketsScanned = %d, want %d", stats.MarketsScanned, 10)
		}
		if stats.WalletsFound != 50 {
			t.Errorf("WalletsFound = %d, want %d", stats.WalletsFound, 50)
		}
		if stats.WalletsScanned != 45 {
			t.Errorf("WalletsScanned = %d, want %d", stats.WalletsScanned, 45)
		}
		if stats.BotsDetected != 5 {
			t.Errorf("BotsDetected = %d, want %d", stats.BotsDetected, 5)
		}
		if stats.ScanErrors != 2 {
			t.Errorf("ScanErrors = %d, want %d", stats.ScanErrors, 2)
		}
	})

	t.Run("duration", func(t *testing.T) {
		now := time.Now()
		stats := DiscoveryStats{
			StartTime: now.Add(-5 * time.Minute),
			EndTime:   now,
		}

		duration := stats.Duration()
		if duration != 5*time.Minute {
			t.Errorf("Duration() = %v, want %v", duration, 5*time.Minute)
		}
	})

	t.Run("duration_zero_when_not_set", func(t *testing.T) {
		stats := DiscoveryStats{}
		duration := stats.Duration()
		if duration != 0 {
			t.Errorf("Duration() = %v, want 0", duration)
		}
	})

	t.Run("bot_detection_rate", func(t *testing.T) {
		stats := DiscoveryStats{
			WalletsScanned: 100,
			BotsDetected:   25,
		}

		rate := stats.BotDetectionRate()
		if rate != 0.25 {
			t.Errorf("BotDetectionRate() = %f, want %f", rate, 0.25)
		}
	})

	t.Run("bot_detection_rate_zero_wallets", func(t *testing.T) {
		stats := DiscoveryStats{
			WalletsScanned: 0,
			BotsDetected:   0,
		}

		rate := stats.BotDetectionRate()
		if rate != 0 {
			t.Errorf("BotDetectionRate() = %f, want 0", rate)
		}
	})
}

func TestDiscoveryResult(t *testing.T) {
	t.Run("new_discovery_result", func(t *testing.T) {
		result := NewDiscoveryResult()

		if result.Wallets == nil {
			t.Error("Wallets is nil, expected initialized slice")
		}
		if len(result.Wallets) != 0 {
			t.Errorf("len(Wallets) = %d, want 0", len(result.Wallets))
		}
		if result.Markets == nil {
			t.Error("Markets is nil, expected initialized slice")
		}
		if result.Stats.StartTime.IsZero() {
			t.Error("Stats.StartTime is zero, expected non-zero")
		}
	})

	t.Run("add_wallet", func(t *testing.T) {
		result := NewDiscoveryResult()

		wallet := DiscoveredWallet{
			Address:     "0x1234",
			TotalAmount: 500.0,
		}

		result.AddWallet(wallet)

		if len(result.Wallets) != 1 {
			t.Errorf("len(Wallets) = %d, want 1", len(result.Wallets))
		}
		if result.Wallets[0].Address != "0x1234" {
			t.Errorf("Wallets[0].Address = %s, want 0x1234", result.Wallets[0].Address)
		}
	})

	t.Run("get_bots", func(t *testing.T) {
		result := NewDiscoveryResult()

		// Add a bot (high score)
		botReport := models.NewScanReport("0xbot")
		botReport.BotScore = 85
		result.AddWallet(DiscoveredWallet{
			Address:    "0xbot",
			ScanReport: botReport,
		})

		// Add a human (low score)
		humanReport := models.NewScanReport("0xhuman")
		humanReport.BotScore = 30
		result.AddWallet(DiscoveredWallet{
			Address:    "0xhuman",
			ScanReport: humanReport,
		})

		// Add a wallet without scan
		result.AddWallet(DiscoveredWallet{
			Address:    "0xnoscan",
			ScanReport: nil,
		})

		bots := result.GetBots(80)
		if len(bots) != 1 {
			t.Errorf("len(GetBots(80)) = %d, want 1", len(bots))
		}
		if bots[0].Address != "0xbot" {
			t.Errorf("bots[0].Address = %s, want 0xbot", bots[0].Address)
		}
	})

	t.Run("get_bots_empty_when_no_matches", func(t *testing.T) {
		result := NewDiscoveryResult()

		humanReport := models.NewScanReport("0xhuman")
		humanReport.BotScore = 30
		result.AddWallet(DiscoveredWallet{
			Address:    "0xhuman",
			ScanReport: humanReport,
		})

		bots := result.GetBots(80)
		if len(bots) != 0 {
			t.Errorf("len(GetBots(80)) = %d, want 0", len(bots))
		}
	})

	t.Run("finalize", func(t *testing.T) {
		result := NewDiscoveryResult()
		result.Stats.WalletsScanned = 10
		result.Stats.BotsDetected = 3
		result.Stats.MarketsScanned = 5

		result.Finalize()

		if result.Stats.EndTime.IsZero() {
			t.Error("Stats.EndTime is zero after Finalize(), expected non-zero")
		}
		if result.Stats.Duration() <= 0 {
			t.Error("Stats.Duration() should be > 0 after Finalize()")
		}
	})

	t.Run("sort_by_bot_score", func(t *testing.T) {
		result := NewDiscoveryResult()

		report1 := models.NewScanReport("0xwallet1")
		report1.BotScore = 50
		report2 := models.NewScanReport("0xwallet2")
		report2.BotScore = 90
		report3 := models.NewScanReport("0xwallet3")
		report3.BotScore = 70

		result.AddWallet(DiscoveredWallet{Address: "0xwallet1", ScanReport: report1})
		result.AddWallet(DiscoveredWallet{Address: "0xwallet2", ScanReport: report2})
		result.AddWallet(DiscoveredWallet{Address: "0xwallet3", ScanReport: report3})

		result.SortByBotScore()

		if result.Wallets[0].ScanReport.BotScore != 90 {
			t.Errorf("Wallets[0].BotScore = %d, want 90", result.Wallets[0].ScanReport.BotScore)
		}
		if result.Wallets[1].ScanReport.BotScore != 70 {
			t.Errorf("Wallets[1].BotScore = %d, want 70", result.Wallets[1].ScanReport.BotScore)
		}
		if result.Wallets[2].ScanReport.BotScore != 50 {
			t.Errorf("Wallets[2].BotScore = %d, want 50", result.Wallets[2].ScanReport.BotScore)
		}
	})

	t.Run("sort_by_bot_score_nil_reports_last", func(t *testing.T) {
		result := NewDiscoveryResult()

		report1 := models.NewScanReport("0xwallet1")
		report1.BotScore = 50

		result.AddWallet(DiscoveredWallet{Address: "0xnoscan", ScanReport: nil})
		result.AddWallet(DiscoveredWallet{Address: "0xwallet1", ScanReport: report1})

		result.SortByBotScore()

		if result.Wallets[0].Address != "0xwallet1" {
			t.Errorf("Wallets[0].Address = %s, want 0xwallet1", result.Wallets[0].Address)
		}
		if result.Wallets[1].Address != "0xnoscan" {
			t.Errorf("Wallets[1].Address = %s, want 0xnoscan", result.Wallets[1].Address)
		}
	})
}

func TestMarketInfo(t *testing.T) {
	t.Run("new_market_info", func(t *testing.T) {
		info := MarketInfo{
			ConditionID:  "0xmarket123",
			Slug:         "will-trump-win",
			Question:     "Will Trump win?",
			HoldersCount: 50,
			Volume:       100000.0,
			Liquidity:    50000.0,
		}

		if info.ConditionID != "0xmarket123" {
			t.Errorf("ConditionID = %s, want 0xmarket123", info.ConditionID)
		}
		if info.Slug != "will-trump-win" {
			t.Errorf("Slug = %s, want will-trump-win", info.Slug)
		}
		if info.HoldersCount != 50 {
			t.Errorf("HoldersCount = %d, want 50", info.HoldersCount)
		}
	})
}

// TestDiscoveryOptionsJSON verifies JSON serialization works correctly.
func TestDiscoveryOptionsJSON(t *testing.T) {
	opts := DiscoveryOptions{
		MarketIDs:    []string{"0x123"},
		MarketSlugs:  []string{"test-market"},
		AutoDiscover: false,
		TopN:         10,
		SortBy:       "volume",
		MinVolume:    1000.0,
		HoldersLimit: 20,
		Concurrency:  5,
		NoScan:       false,
		BotThreshold: 80,
	}

	// Just verify all fields are accessible
	if len(opts.MarketIDs) != 1 {
		t.Errorf("len(MarketIDs) = %d, want 1", len(opts.MarketIDs))
	}
	if opts.TopN != 10 {
		t.Errorf("TopN = %d, want 10", opts.TopN)
	}
}
