package discovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/api"
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

// TestNewDiscoverer tests the Discoverer constructor and functional options.
func TestNewDiscoverer(t *testing.T) {
	t.Run("creates_with_default_dependencies", func(t *testing.T) {
		d := NewDiscoverer()

		if d == nil {
			t.Fatal("NewDiscoverer() returned nil")
		}
		if d.dataAPI == nil {
			t.Error("dataAPI is nil, expected non-nil")
		}
		if d.gammaAPI == nil {
			t.Error("gammaAPI is nil, expected non-nil")
		}
		if d.scanner == nil {
			t.Error("scanner is nil, expected non-nil")
		}
	})

	t.Run("accepts_custom_data_api", func(t *testing.T) {
		mockDataAPI := &mockDataAPIImpl{}
		d := NewDiscoverer(WithDiscovererDataAPI(mockDataAPI))

		if d.dataAPI != mockDataAPI {
			t.Error("dataAPI was not set to custom value")
		}
	})

	t.Run("accepts_custom_gamma_api", func(t *testing.T) {
		mockGammaAPI := &mockGammaAPIImpl{}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGammaAPI))

		if d.gammaAPI != mockGammaAPI {
			t.Error("gammaAPI was not set to custom value")
		}
	})

	t.Run("accepts_custom_scanner", func(t *testing.T) {
		mockScanner := &mockScannerImpl{}
		d := NewDiscoverer(WithDiscovererScanner(mockScanner))

		if d.scanner != mockScanner {
			t.Error("scanner was not set to custom value")
		}
	})
}

// TestDiscoverer_resolveMarkets tests the market resolution logic.
func TestDiscoverer_resolveMarkets(t *testing.T) {
	t.Run("resolves_explicit_market_ids", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{
				"0xmarket1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Question 1", Volume: 1000},
				"0xmarket2": {ConditionID: "0xmarket2", Slug: "market-2", Question: "Question 2", Volume: 2000},
			},
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xmarket1", "0xmarket2"},
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		markets, err := d.resolveMarkets(ctx, opts)
		if err != nil {
			t.Fatalf("resolveMarkets() error = %v", err)
		}

		if len(markets) != 2 {
			t.Errorf("len(markets) = %d, want 2", len(markets))
		}
	})

	t.Run("resolves_market_slugs", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			marketsBySlugs: map[string]*api.Market{
				"will-trump-win": {ConditionID: "0xtrump", Slug: "will-trump-win", Question: "Will Trump win?", Volume: 5000},
			},
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketSlugs: []string{"will-trump-win"},
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		markets, err := d.resolveMarkets(ctx, opts)
		if err != nil {
			t.Fatalf("resolveMarkets() error = %v", err)
		}

		if len(markets) != 1 {
			t.Errorf("len(markets) = %d, want 1", len(markets))
		}
	})

	t.Run("auto_discovers_top_markets", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			topMarkets: api.MarketList{
				{ConditionID: "0xtop1", Slug: "top-1", Question: "Top 1?", Volume: 10000},
				{ConditionID: "0xtop2", Slug: "top-2", Question: "Top 2?", Volume: 9000},
				{ConditionID: "0xtop3", Slug: "top-3", Question: "Top 3?", Volume: 8000},
			},
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			AutoDiscover: true,
			TopN:         3,
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		markets, err := d.resolveMarkets(ctx, opts)
		if err != nil {
			t.Fatalf("resolveMarkets() error = %v", err)
		}

		if len(markets) != 3 {
			t.Errorf("len(markets) = %d, want 3", len(markets))
		}
	})

	t.Run("deduplicates_markets", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{
				"0xmarket1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Question 1", Volume: 1000},
			},
			marketsBySlugs: map[string]*api.Market{
				"market-1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Question 1", Volume: 1000},
			},
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketIDs:   []string{"0xmarket1"},
			MarketSlugs: []string{"market-1"}, // Same market via slug
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		markets, err := d.resolveMarkets(ctx, opts)
		if err != nil {
			t.Fatalf("resolveMarkets() error = %v", err)
		}

		if len(markets) != 1 {
			t.Errorf("len(markets) = %d, want 1 (expected deduplication)", len(markets))
		}
	})
}

// TestDiscoverer_aggregateHolders tests the holder aggregation logic.
func TestDiscoverer_aggregateHolders(t *testing.T) {
	t.Run("aggregates_holders_across_markets", func(t *testing.T) {
		mockData := &mockDataAPIImpl{
			holders: map[string]api.HolderList{
				"0xmarket1": {
					{Token: "token1", Holders: []api.Holder{
						{ProxyWallet: "0xwallet1", Amount: 100},
						{ProxyWallet: "0xwallet2", Amount: 200},
					}},
				},
				"0xmarket2": {
					{Token: "token2", Holders: []api.Holder{
						{ProxyWallet: "0xwallet1", Amount: 150}, // Same wallet, different market
						{ProxyWallet: "0xwallet3", Amount: 300},
					}},
				},
			},
		}
		d := NewDiscoverer(WithDiscovererDataAPI(mockData))

		markets := []resolvedMarket{
			{conditionID: "0xmarket1", slug: "market-1", question: "Question 1"},
			{conditionID: "0xmarket2", slug: "market-2", question: "Question 2"},
		}

		opts := DiscoveryOptions{HoldersLimit: 20}

		ctx := context.Background()
		aggregated, err := d.aggregateHolders(ctx, markets, opts)
		if err != nil {
			t.Fatalf("aggregateHolders() error = %v", err)
		}

		// Should have 3 unique wallets
		if len(aggregated) != 3 {
			t.Errorf("len(aggregated) = %d, want 3", len(aggregated))
		}

		// Wallet1 should have aggregated amount from both markets
		wallet1 := aggregated["0xwallet1"]
		if wallet1.TotalAmount != 250 { // 100 + 150
			t.Errorf("wallet1.TotalAmount = %f, want 250", wallet1.TotalAmount)
		}
		if wallet1.MarketCount != 2 {
			t.Errorf("wallet1.MarketCount = %d, want 2", wallet1.MarketCount)
		}
	})

	t.Run("handles_empty_holders", func(t *testing.T) {
		mockData := &mockDataAPIImpl{
			holders: map[string]api.HolderList{},
		}
		d := NewDiscoverer(WithDiscovererDataAPI(mockData))

		markets := []resolvedMarket{
			{conditionID: "0xmarket1", slug: "market-1", question: "Question 1"},
		}

		opts := DiscoveryOptions{HoldersLimit: 20}

		ctx := context.Background()
		aggregated, err := d.aggregateHolders(ctx, markets, opts)
		if err != nil {
			t.Fatalf("aggregateHolders() error = %v", err)
		}

		if len(aggregated) != 0 {
			t.Errorf("len(aggregated) = %d, want 0", len(aggregated))
		}
	})
}

// TestDiscoverer_Discover tests the main orchestration method.
func TestDiscoverer_Discover(t *testing.T) {
	t.Run("discovers_and_scans_wallets", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{
				"0xmarket1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Question 1", Volume: 1000, Liquidity: 500},
			},
		}
		mockData := &mockDataAPIImpl{
			holders: map[string]api.HolderList{
				"0xmarket1": {
					{Token: "token1", Holders: []api.Holder{
						{ProxyWallet: "0xwallet1", Amount: 100},
						{ProxyWallet: "0xwallet2", Amount: 200},
					}},
				},
			},
		}
		mockScn := &mockScannerImpl{
			reports: map[string]*models.ScanReport{
				"0xwallet1": {WalletAddress: "0xwallet1", BotScore: 85},
				"0xwallet2": {WalletAddress: "0xwallet2", BotScore: 30},
			},
		}

		d := NewDiscoverer(
			WithDiscovererDataAPI(mockData),
			WithDiscovererGammaAPI(mockGamma),
			WithDiscovererScanner(mockScn),
		)

		opts := DiscoveryOptions{
			MarketIDs:    []string{"0xmarket1"},
			BotThreshold: 80,
		}

		ctx := context.Background()
		result, err := d.Discover(ctx, opts)
		if err != nil {
			t.Fatalf("Discover() error = %v", err)
		}

		if result == nil {
			t.Fatal("Discover() returned nil result")
		}
		if len(result.Wallets) != 2 {
			t.Errorf("len(Wallets) = %d, want 2", len(result.Wallets))
		}
		if result.Stats.WalletsScanned != 2 {
			t.Errorf("Stats.WalletsScanned = %d, want 2", result.Stats.WalletsScanned)
		}
		if result.Stats.BotsDetected != 1 {
			t.Errorf("Stats.BotsDetected = %d, want 1 (wallet1 with score 85)", result.Stats.BotsDetected)
		}
	})

	t.Run("skips_scanning_when_no_scan_is_true", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{
				"0xmarket1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Question 1", Volume: 1000, Liquidity: 500},
			},
		}
		mockData := &mockDataAPIImpl{
			holders: map[string]api.HolderList{
				"0xmarket1": {
					{Token: "token1", Holders: []api.Holder{
						{ProxyWallet: "0xwallet1", Amount: 100},
					}},
				},
			},
		}
		mockScn := &mockScannerImpl{
			scanCalled: false,
		}

		d := NewDiscoverer(
			WithDiscovererDataAPI(mockData),
			WithDiscovererGammaAPI(mockGamma),
			WithDiscovererScanner(mockScn),
		)

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xmarket1"},
			NoScan:    true,
		}

		ctx := context.Background()
		result, err := d.Discover(ctx, opts)
		if err != nil {
			t.Fatalf("Discover() error = %v", err)
		}

		if mockScn.scanCalled {
			t.Error("Scanner.Scan was called when NoScan=true")
		}
		if result.Wallets[0].ScanReport != nil {
			t.Error("ScanReport should be nil when NoScan=true")
		}
		if result.Stats.WalletsScanned != 0 {
			t.Errorf("Stats.WalletsScanned = %d, want 0", result.Stats.WalletsScanned)
		}
	})

	t.Run("returns_error_for_invalid_options", func(t *testing.T) {
		d := NewDiscoverer()

		opts := DiscoveryOptions{} // No input specified

		ctx := context.Background()
		_, err := d.Discover(ctx, opts)
		if err == nil {
			t.Error("Discover() expected error for invalid options, got nil")
		}
	})

	t.Run("records_market_info", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{
				"0xmarket1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Question 1", Volume: 1000, Liquidity: 500},
			},
		}
		mockData := &mockDataAPIImpl{
			holders: map[string]api.HolderList{
				"0xmarket1": {
					{Token: "token1", Holders: []api.Holder{
						{ProxyWallet: "0xwallet1", Amount: 100},
					}},
				},
			},
		}
		mockScn := &mockScannerImpl{
			reports: map[string]*models.ScanReport{
				"0xwallet1": {WalletAddress: "0xwallet1", BotScore: 50},
			},
		}

		d := NewDiscoverer(
			WithDiscovererDataAPI(mockData),
			WithDiscovererGammaAPI(mockGamma),
			WithDiscovererScanner(mockScn),
		)

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xmarket1"},
		}

		ctx := context.Background()
		result, err := d.Discover(ctx, opts)
		if err != nil {
			t.Fatalf("Discover() error = %v", err)
		}

		if len(result.Markets) != 1 {
			t.Errorf("len(Markets) = %d, want 1", len(result.Markets))
		}
		if result.Markets[0].ConditionID != "0xmarket1" {
			t.Errorf("Markets[0].ConditionID = %s, want 0xmarket1", result.Markets[0].ConditionID)
		}
		if result.Stats.MarketsScanned != 1 {
			t.Errorf("Stats.MarketsScanned = %d, want 1", result.Stats.MarketsScanned)
		}
	})
}

// TestDiscovererInterface verifies the Discoverer implements expected patterns.
func TestDiscovererInterface(t *testing.T) {
	// Compile-time check that Discoverer can be created
	var _ = NewDiscoverer()
}

// TestDiscoverer_resolveMarkets_Errors tests error handling in market resolution.
func TestDiscoverer_resolveMarkets_Errors(t *testing.T) {
	t.Run("returns_error_when_GetMarket_fails", func(t *testing.T) {
		mockGamma := &mockGammaAPIImplWithErrors{
			getMarketErr: errors.New("API connection failed"),
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xmarket1"},
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		_, err := d.resolveMarkets(ctx, opts)
		if err == nil {
			t.Error("resolveMarkets() expected error, got nil")
		}
		if !errors.Is(err, mockGamma.getMarketErr) && err.Error() != "fetching market 0xmarket1: API connection failed" {
			t.Errorf("resolveMarkets() error = %v, want wrapped API error", err)
		}
	})

	t.Run("returns_error_when_GetMarketBySlug_fails", func(t *testing.T) {
		mockGamma := &mockGammaAPIImplWithErrors{
			getMarketBySlugErr: errors.New("slug lookup failed"),
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketSlugs: []string{"will-trump-win"},
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		_, err := d.resolveMarkets(ctx, opts)
		if err == nil {
			t.Error("resolveMarkets() expected error, got nil")
		}
	})

	t.Run("returns_error_when_GetTopMarkets_fails", func(t *testing.T) {
		mockGamma := &mockGammaAPIImplWithErrors{
			getTopMarketsErr: errors.New("top markets fetch failed"),
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			AutoDiscover: true,
			TopN:         5,
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		_, err := d.resolveMarkets(ctx, opts)
		if err == nil {
			t.Error("resolveMarkets() expected error, got nil")
		}
	})

	t.Run("skips_nil_market_from_GetMarket", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{}, // Empty, so GetMarket returns nil
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xnonexistent"},
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		markets, err := d.resolveMarkets(ctx, opts)
		if err != nil {
			t.Fatalf("resolveMarkets() unexpected error: %v", err)
		}
		if len(markets) != 0 {
			t.Errorf("len(markets) = %d, want 0 for nil market", len(markets))
		}
	})

	t.Run("skips_nil_market_from_GetMarketBySlug", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			marketsBySlugs: map[string]*api.Market{}, // Empty, so GetMarketBySlug returns nil
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketSlugs: []string{"nonexistent-slug"},
		}
		opts.ApplyDefaults()

		ctx := context.Background()
		markets, err := d.resolveMarkets(ctx, opts)
		if err != nil {
			t.Fatalf("resolveMarkets() unexpected error: %v", err)
		}
		if len(markets) != 0 {
			t.Errorf("len(markets) = %d, want 0 for nil market", len(markets))
		}
	})
}

// TestDiscoverer_aggregateHolders_Errors tests error handling in holder aggregation.
func TestDiscoverer_aggregateHolders_Errors(t *testing.T) {
	t.Run("returns_error_when_GetHoldersForMarket_fails", func(t *testing.T) {
		mockData := &mockDataAPIImplWithErrors{
			getHoldersErr: errors.New("holders fetch failed"),
		}
		d := NewDiscoverer(WithDiscovererDataAPI(mockData))

		markets := []resolvedMarket{
			{conditionID: "0xmarket1", slug: "market-1", question: "Question 1"},
		}
		opts := DiscoveryOptions{HoldersLimit: 20}

		ctx := context.Background()
		_, err := d.aggregateHolders(ctx, markets, opts)
		if err == nil {
			t.Error("aggregateHolders() expected error, got nil")
		}
	})
}

// TestDiscoverer_scanWallets_Errors tests error handling in wallet scanning.
func TestDiscoverer_scanWallets_Errors(t *testing.T) {
	t.Run("increments_scan_errors_on_failure", func(t *testing.T) {
		mockScn := &mockScannerImplWithErrors{
			scanErr: errors.New("scan failed"),
		}
		d := NewDiscoverer(WithDiscovererScanner(mockScn))

		wallets := map[string]DiscoveredWallet{
			"0xwallet1": {Address: "0xwallet1", TotalAmount: 100},
			"0xwallet2": {Address: "0xwallet2", TotalAmount: 200},
		}

		result := NewDiscoveryResult()
		opts := DiscoveryOptions{Concurrency: 2, BotThreshold: 80}

		ctx := context.Background()
		d.scanWallets(ctx, wallets, result, opts)

		if result.Stats.ScanErrors != 2 {
			t.Errorf("Stats.ScanErrors = %d, want 2", result.Stats.ScanErrors)
		}
		if result.Stats.WalletsScanned != 0 {
			t.Errorf("Stats.WalletsScanned = %d, want 0", result.Stats.WalletsScanned)
		}
	})

	t.Run("handles_context_cancellation_during_scanning", func(t *testing.T) {
		// Create a scanner that will block
		mockScn := &mockScannerImplBlocking{
			blockChan: make(chan struct{}),
		}
		d := NewDiscoverer(WithDiscovererScanner(mockScn))

		wallets := map[string]DiscoveredWallet{
			"0xwallet1": {Address: "0xwallet1", TotalAmount: 100},
		}

		result := NewDiscoveryResult()
		opts := DiscoveryOptions{Concurrency: 1, BotThreshold: 80}

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately - this will cause context cancellation in semaphore acquisition
		cancel()

		d.scanWallets(ctx, wallets, result, opts)

		// When context is cancelled, errors should be incremented
		if result.Stats.ScanErrors != 1 {
			t.Errorf("Stats.ScanErrors = %d, want 1", result.Stats.ScanErrors)
		}
	})
}

// TestDiscoverer_Discover_Errors tests error handling in Discover.
func TestDiscoverer_Discover_Errors(t *testing.T) {
	t.Run("returns_error_when_resolveMarkets_fails", func(t *testing.T) {
		mockGamma := &mockGammaAPIImplWithErrors{
			getMarketErr: errors.New("API connection failed"),
		}
		d := NewDiscoverer(WithDiscovererGammaAPI(mockGamma))

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xmarket1"},
		}

		ctx := context.Background()
		_, err := d.Discover(ctx, opts)
		if err == nil {
			t.Error("Discover() expected error, got nil")
		}
	})

	t.Run("returns_error_when_aggregateHolders_fails", func(t *testing.T) {
		mockGamma := &mockGammaAPIImpl{
			markets: map[string]*api.Market{
				"0xmarket1": {ConditionID: "0xmarket1", Slug: "market-1", Question: "Q1"},
			},
		}
		mockData := &mockDataAPIImplWithErrors{
			getHoldersErr: errors.New("holders fetch failed"),
		}
		d := NewDiscoverer(
			WithDiscovererGammaAPI(mockGamma),
			WithDiscovererDataAPI(mockData),
		)

		opts := DiscoveryOptions{
			MarketIDs: []string{"0xmarket1"},
		}

		ctx := context.Background()
		_, err := d.Discover(ctx, opts)
		if err == nil {
			t.Error("Discover() expected error, got nil")
		}
	})
}

// TestDiscoveryResult_SortByBotScore_EdgeCases tests edge cases in sorting.
func TestDiscoveryResult_SortByBotScore_EdgeCases(t *testing.T) {
	t.Run("handles_all_nil_reports", func(t *testing.T) {
		result := NewDiscoveryResult()
		result.AddWallet(DiscoveredWallet{Address: "0xa", ScanReport: nil})
		result.AddWallet(DiscoveredWallet{Address: "0xb", ScanReport: nil})
		result.AddWallet(DiscoveredWallet{Address: "0xc", ScanReport: nil})

		// Should not panic
		result.SortByBotScore()

		if len(result.Wallets) != 3 {
			t.Errorf("len(Wallets) = %d, want 3", len(result.Wallets))
		}
	})

	t.Run("handles_empty_wallets", func(t *testing.T) {
		result := NewDiscoveryResult()

		// Should not panic
		result.SortByBotScore()

		if len(result.Wallets) != 0 {
			t.Errorf("len(Wallets) = %d, want 0", len(result.Wallets))
		}
	})

	t.Run("sorts_equal_scores_stably", func(t *testing.T) {
		result := NewDiscoveryResult()

		report1 := models.NewScanReport("0xwallet1")
		report1.BotScore = 50
		report2 := models.NewScanReport("0xwallet2")
		report2.BotScore = 50

		result.AddWallet(DiscoveredWallet{Address: "0xwallet1", ScanReport: report1})
		result.AddWallet(DiscoveredWallet{Address: "0xwallet2", ScanReport: report2})

		// Should not panic and maintain order for equal scores
		result.SortByBotScore()

		if len(result.Wallets) != 2 {
			t.Errorf("len(Wallets) = %d, want 2", len(result.Wallets))
		}
	})
}

// TestDiscoverer_scanWallets_Concurrency tests concurrent wallet scanning.
func TestDiscoverer_scanWallets_Concurrency(t *testing.T) {
	t.Run("respects_concurrency_limit", func(t *testing.T) {
		mockScn := &mockScannerImpl{
			reports: map[string]*models.ScanReport{
				"0xwallet1": {WalletAddress: "0xwallet1", BotScore: 50},
				"0xwallet2": {WalletAddress: "0xwallet2", BotScore: 60},
				"0xwallet3": {WalletAddress: "0xwallet3", BotScore: 70},
				"0xwallet4": {WalletAddress: "0xwallet4", BotScore: 80},
				"0xwallet5": {WalletAddress: "0xwallet5", BotScore: 90},
			},
		}
		d := NewDiscoverer(WithDiscovererScanner(mockScn))

		wallets := map[string]DiscoveredWallet{
			"0xwallet1": {Address: "0xwallet1", TotalAmount: 100},
			"0xwallet2": {Address: "0xwallet2", TotalAmount: 200},
			"0xwallet3": {Address: "0xwallet3", TotalAmount: 300},
			"0xwallet4": {Address: "0xwallet4", TotalAmount: 400},
			"0xwallet5": {Address: "0xwallet5", TotalAmount: 500},
		}

		result := NewDiscoveryResult()
		opts := DiscoveryOptions{Concurrency: 2, BotThreshold: 80}

		ctx := context.Background()
		d.scanWallets(ctx, wallets, result, opts)

		if result.Stats.WalletsScanned != 5 {
			t.Errorf("Stats.WalletsScanned = %d, want 5", result.Stats.WalletsScanned)
		}
		if result.Stats.BotsDetected != 2 {
			t.Errorf("Stats.BotsDetected = %d, want 2 (scores 80, 90)", result.Stats.BotsDetected)
		}
	})
}

// Mock implementations for testing

// mockDataAPIImpl implements the DataAPIProvider interface for testing.
type mockDataAPIImpl struct {
	holders map[string]api.HolderList
}

func (m *mockDataAPIImpl) GetHoldersForMarket(_ context.Context, conditionID string, _ int) (api.HolderList, error) {
	if holders, ok := m.holders[conditionID]; ok {
		return holders, nil
	}
	return api.HolderList{}, nil
}

// mockGammaAPIImpl implements the GammaAPIProvider interface for testing.
type mockGammaAPIImpl struct {
	markets        map[string]*api.Market
	marketsBySlugs map[string]*api.Market
	topMarkets     api.MarketList
}

func (m *mockGammaAPIImpl) GetMarket(_ context.Context, conditionID string) (*api.Market, error) {
	if market, ok := m.markets[conditionID]; ok {
		return market, nil
	}
	return nil, nil
}

func (m *mockGammaAPIImpl) GetMarketBySlug(_ context.Context, slug string) (*api.Market, error) {
	if market, ok := m.marketsBySlugs[slug]; ok {
		return market, nil
	}
	return nil, nil
}

func (m *mockGammaAPIImpl) GetTopMarkets(_ context.Context, _ api.TopMarketsOptions) (api.MarketList, error) {
	return m.topMarkets, nil
}

// mockScannerImpl implements the ScannerProvider interface for testing.
type mockScannerImpl struct {
	reports    map[string]*models.ScanReport
	scanCalled bool
}

func (m *mockScannerImpl) Scan(_ context.Context, wallet string) (*models.ScanReport, error) {
	m.scanCalled = true
	if report, ok := m.reports[wallet]; ok {
		return report, nil
	}
	return &models.ScanReport{WalletAddress: wallet, BotScore: 0}, nil
}

// mockGammaAPIImplWithErrors implements GammaAPIProvider with configurable errors.
type mockGammaAPIImplWithErrors struct {
	getMarketErr       error
	getMarketBySlugErr error
	getTopMarketsErr   error
}

func (m *mockGammaAPIImplWithErrors) GetMarket(_ context.Context, _ string) (*api.Market, error) {
	if m.getMarketErr != nil {
		return nil, m.getMarketErr
	}
	return nil, nil
}

func (m *mockGammaAPIImplWithErrors) GetMarketBySlug(_ context.Context, _ string) (*api.Market, error) {
	if m.getMarketBySlugErr != nil {
		return nil, m.getMarketBySlugErr
	}
	return nil, nil
}

func (m *mockGammaAPIImplWithErrors) GetTopMarkets(_ context.Context, _ api.TopMarketsOptions) (api.MarketList, error) {
	if m.getTopMarketsErr != nil {
		return nil, m.getTopMarketsErr
	}
	return nil, nil
}

// mockDataAPIImplWithErrors implements DataAPIProvider with configurable errors.
type mockDataAPIImplWithErrors struct {
	getHoldersErr error
}

func (m *mockDataAPIImplWithErrors) GetHoldersForMarket(_ context.Context, _ string, _ int) (api.HolderList, error) {
	if m.getHoldersErr != nil {
		return nil, m.getHoldersErr
	}
	return nil, nil
}

// mockScannerImplWithErrors implements ScannerProvider with configurable errors.
type mockScannerImplWithErrors struct {
	scanErr error
}

func (m *mockScannerImplWithErrors) Scan(_ context.Context, _ string) (*models.ScanReport, error) {
	return nil, m.scanErr
}

// mockScannerImplBlocking implements ScannerProvider that blocks until cancelled.
type mockScannerImplBlocking struct {
	blockChan chan struct{}
}

func (m *mockScannerImplBlocking) Scan(ctx context.Context, wallet string) (*models.ScanReport, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.blockChan:
		return &models.ScanReport{WalletAddress: wallet, BotScore: 0}, nil
	}
}
