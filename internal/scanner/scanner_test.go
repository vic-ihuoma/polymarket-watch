// Package scanner provides bot detection algorithms for Polymarket wallets.
package scanner

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/api"
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewScanner(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		s := NewScanner()
		if s == nil {
			t.Fatal("expected non-nil scanner")
		}
		if s.dataAPI == nil {
			t.Error("expected dataAPI to be initialized")
		}
		if len(s.detectors) != 4 {
			t.Errorf("expected 4 detectors, got %d", len(s.detectors))
		}
	})

	t.Run("custom data API", func(t *testing.T) {
		customAPI := api.NewDataAPI()
		s := NewScanner(WithDataAPI(customAPI))
		if s.dataAPI != customAPI {
			t.Error("expected custom dataAPI to be used")
		}
	})

	t.Run("custom detectors", func(t *testing.T) {
		customDetectors := []Detector{NewArbitrageDetector()}
		s := NewScanner(WithDetectors(customDetectors))
		if len(s.detectors) != 1 {
			t.Errorf("expected 1 detector, got %d", len(s.detectors))
		}
	})

	t.Run("add detector", func(t *testing.T) {
		s := NewScanner()
		initialCount := len(s.detectors)
		s2 := NewScanner(WithDetector(NewArbitrageDetector()))
		if len(s2.detectors) != initialCount+1 {
			t.Errorf("expected %d detectors, got %d", initialCount+1, len(s2.detectors))
		}
	})
}

func TestScanner_DetectorWeights(t *testing.T) {
	s := NewScanner()
	weights := s.DetectorWeights()

	// Verify all expected detectors are present
	expectedDetectors := []string{"arbitrage", "timing", "winrate", "sizing"}
	for _, name := range expectedDetectors {
		if _, ok := weights[name]; !ok {
			t.Errorf("expected detector %q in weights", name)
		}
	}

	// Verify weights sum to 1.0
	var totalWeight float64
	for _, weight := range weights {
		totalWeight += weight
	}
	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("expected weights to sum to 1.0, got %f", totalWeight)
	}
}

func TestScanner_AnalyzeData(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		s := NewScanner()
		report := s.AnalyzeData("0x1234567890abcdef", models.PositionList{}, models.TradeList{})

		if report.WalletAddress != "0x1234567890abcdef" {
			t.Errorf("expected wallet address 0x1234567890abcdef, got %s", report.WalletAddress)
		}
		if report.BotScore != 0 {
			t.Errorf("expected bot score 0 for empty data, got %d", report.BotScore)
		}
	})

	t.Run("with arbitrage positions", func(t *testing.T) {
		s := NewScanner()
		positions := models.PositionList{
			{ConditionID: "market1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
			{ConditionID: "market1", Outcome: "No", Size: 100, AvgPrice: 0.45},
		}
		report := s.AnalyzeData("0x1234567890abcdef", positions, models.TradeList{})

		if report.BotScore == 0 {
			t.Error("expected non-zero bot score for arbitrage positions")
		}
		if len(report.Signals) == 0 {
			t.Error("expected at least one detection signal")
		}

		// Check arbitrage score is set
		if _, ok := report.Scores["arbitrage"]; !ok {
			t.Error("expected arbitrage score in report")
		}
	})

	t.Run("with timing pattern", func(t *testing.T) {
		s := NewScanner()
		now := time.Now()
		trades := models.TradeList{
			{Market: "m1", Outcome: "Yes", Side: "BUY", Size: 100, Price: 0.5, MatchTime: now},
			{Market: "m1", Outcome: "Yes", Side: "BUY", Size: 100, Price: 0.5, MatchTime: now.Add(2 * time.Second)},
			{Market: "m1", Outcome: "Yes", Side: "BUY", Size: 100, Price: 0.5, MatchTime: now.Add(4 * time.Second)},
			{Market: "m1", Outcome: "Yes", Side: "BUY", Size: 100, Price: 0.5, MatchTime: now.Add(6 * time.Second)},
			{Market: "m1", Outcome: "Yes", Side: "BUY", Size: 100, Price: 0.5, MatchTime: now.Add(8 * time.Second)},
		}
		report := s.AnalyzeData("0x1234567890abcdef", models.PositionList{}, trades)

		// Should have timing detection
		if _, ok := report.Scores["timing"]; !ok {
			t.Error("expected timing score in report")
		}
	})

	t.Run("populates stats", func(t *testing.T) {
		s := NewScanner()
		positions := models.PositionList{
			{ConditionID: "market1", Outcome: "Yes", Size: 100, AvgPrice: 0.5},
			{ConditionID: "market2", Outcome: "No", Size: 50, AvgPrice: 0.3},
		}
		now := time.Now()
		trades := models.TradeList{
			{Market: "m1", Side: "BUY", Size: 100, Price: 0.5, MatchTime: now},
			{Market: "m2", Side: "BUY", Size: 200, Price: 0.6, MatchTime: now.Add(time.Hour)},
		}
		report := s.AnalyzeData("0x1234", positions, trades)

		if report.Stats.TotalTrades != 2 {
			t.Errorf("expected 2 total trades, got %d", report.Stats.TotalTrades)
		}
		if report.Stats.TotalPositions != 2 {
			t.Errorf("expected 2 total positions, got %d", report.Stats.TotalPositions)
		}
		if report.Stats.TotalVolume != 170 {
			t.Errorf("expected total volume 170, got %f", report.Stats.TotalVolume)
		}
		if report.Stats.UniqueMarkets != 2 {
			t.Errorf("expected 2 unique markets, got %d", report.Stats.UniqueMarkets)
		}
	})
}

func TestScanner_Scan(t *testing.T) {
	t.Run("successful scan", func(t *testing.T) {
		// Create mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/trades":
				w.Write([]byte(`[{"id":"t1","market":"m1","side":"BUY","size":"100","price":"0.5","match_time":"2024-01-01T00:00:00Z"}]`))
			case "/positions":
				w.Write([]byte(`[{"conditionId":"c1","outcome":"Yes","size":"100","avgPrice":"0.5"}]`))
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		dataAPI := api.NewDataAPI(api.WithBaseURL(server.URL))
		s := NewScanner(WithDataAPI(dataAPI))

		report, err := s.Scan(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report == nil {
			t.Fatal("expected non-nil report")
		}
		if report.WalletAddress != "0x123" {
			t.Errorf("expected wallet 0x123, got %s", report.WalletAddress)
		}
	})

	t.Run("API error on trades", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		dataAPI := api.NewDataAPI(api.WithBaseURL(server.URL))
		s := NewScanner(WithDataAPI(dataAPI))

		_, err := s.Scan(context.Background(), "0x123")
		if err == nil {
			t.Error("expected error for API failure")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Write([]byte(`[]`))
		}))
		defer server.Close()

		dataAPI := api.NewDataAPI(api.WithBaseURL(server.URL))
		s := NewScanner(WithDataAPI(dataAPI))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := s.Scan(ctx, "0x123")
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})
}

func TestScanner_ScanWithAllBotIndicators(t *testing.T) {
	// Create a scenario that triggers all detection algorithms
	s := NewScanner()

	// Arbitrage: opposing positions with profitable spread
	positions := models.PositionList{
		{ConditionID: "market1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
		{ConditionID: "market1", Outcome: "No", Size: 100, AvgPrice: 0.45},
	}

	// Timing: consistent fast intervals
	// Sizing: consistent position sizes (all 100)
	now := time.Now()
	trades := models.TradeList{}
	for i := 0; i < 150; i++ {
		side := "BUY"
		price := 0.45
		if i%2 == 1 {
			side = "SELL"
			price = 0.55 // Profitable exit
		}
		trades = append(trades, models.Trade{
			Market:    "market1",
			Outcome:   "Yes",
			Side:      side,
			Size:      100, // Consistent sizing
			Price:     price,
			MatchTime: now.Add(time.Duration(i*2) * time.Second), // 2 second intervals
		})
	}

	report := s.AnalyzeData("0x1234567890abcdef", positions, trades)

	// Should have high bot score with multiple signals
	if report.BotScore < 50 {
		t.Errorf("expected bot score >= 50, got %d", report.BotScore)
	}

	// Check all detectors produced scores
	detectors := []string{"arbitrage", "timing", "sizing"}
	for _, d := range detectors {
		if _, ok := report.Scores[d]; !ok {
			t.Errorf("expected score for detector %q", d)
		}
	}
}

func TestScanner_calculateStats(t *testing.T) {
	t.Run("calculates unique markets", func(t *testing.T) {
		s := NewScanner()
		trades := models.TradeList{
			{Market: "m1", Side: "BUY", Size: 100, Price: 0.5, MatchTime: time.Now()},
			{Market: "m1", Side: "SELL", Size: 100, Price: 0.6, MatchTime: time.Now()},
			{Market: "m2", Side: "BUY", Size: 50, Price: 0.3, MatchTime: time.Now()},
		}
		report := s.AnalyzeData("0x123", models.PositionList{}, trades)

		if report.Stats.UniqueMarkets != 2 {
			t.Errorf("expected 2 unique markets, got %d", report.Stats.UniqueMarkets)
		}
	})

	t.Run("calculates first and last trade times", func(t *testing.T) {
		s := NewScanner()
		firstTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		lastTime := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		trades := models.TradeList{
			{Market: "m1", Side: "BUY", Size: 100, Price: 0.5, MatchTime: firstTime},
			{Market: "m1", Side: "SELL", Size: 100, Price: 0.6, MatchTime: lastTime},
		}
		report := s.AnalyzeData("0x123", models.PositionList{}, trades)

		if !report.Stats.FirstTradeTime.Equal(firstTime) {
			t.Errorf("expected first trade time %v, got %v", firstTime, report.Stats.FirstTradeTime)
		}
		if !report.Stats.LastTradeTime.Equal(lastTime) {
			t.Errorf("expected last trade time %v, got %v", lastTime, report.Stats.LastTradeTime)
		}
	})

	t.Run("calculates total volume", func(t *testing.T) {
		s := NewScanner()
		trades := models.TradeList{
			{Market: "m1", Side: "BUY", Size: 100, Price: 0.5, MatchTime: time.Now()},  // Value: 50
			{Market: "m2", Side: "BUY", Size: 200, Price: 0.25, MatchTime: time.Now()}, // Value: 50
		}
		report := s.AnalyzeData("0x123", models.PositionList{}, trades)

		if report.Stats.TotalVolume != 100 {
			t.Errorf("expected total volume 100, got %f", report.Stats.TotalVolume)
		}
	})

	t.Run("calculates average position size", func(t *testing.T) {
		s := NewScanner()
		positions := models.PositionList{
			{ConditionID: "c1", Outcome: "Yes", Size: 100, AvgPrice: 0.5},
			{ConditionID: "c2", Outcome: "No", Size: 200, AvgPrice: 0.3},
		}
		report := s.AnalyzeData("0x123", positions, models.TradeList{})

		if report.Stats.AvgPositionSize != 150 {
			t.Errorf("expected avg position size 150, got %f", report.Stats.AvgPositionSize)
		}
	})
}

func TestScannerInterface(t *testing.T) {
	// Ensure Scanner implements expected interface
	var _ interface {
		Scan(ctx context.Context, wallet string) (*models.ScanReport, error)
		AnalyzeData(wallet string, positions models.PositionList, trades models.TradeList) *models.ScanReport
		DetectorWeights() map[string]float64
	} = (*Scanner)(nil)
}

// TestIntegrationAccount88888 verifies that the scanner correctly identifies
// Account88888 (known bot wallet) as having a high bot probability score (>80).
// This test simulates the expected trading patterns of a spread arbitrage bot:
// - Simultaneous opposing positions (Yes/No) on the same market
// - Consistent sub-5 second trade intervals
// - High win rate (>95%)
// - Algorithmic position sizing with round numbers
func TestIntegrationAccount88888(t *testing.T) {
	// Account88888 wallet address from PRD
	const account88888Wallet = "0x7f69983eb28245bba0d5083502a78744a8f66162"

	t.Run("simulated_bot_behavior_detected", func(t *testing.T) {
		s := NewScanner()

		// Create positions that simulate Account88888's spread arbitrage strategy:
		// Holding simultaneous Yes and No positions on the same market
		positions := models.PositionList{
			// Market 1: UP/DOWN spread (0.45 + 0.45 = 0.90, spread = 0.10)
			{ConditionID: "market_up_down_1", Outcome: "Yes", Size: 1000, AvgPrice: 0.45},
			{ConditionID: "market_up_down_1", Outcome: "No", Size: 1000, AvgPrice: 0.45},
			// Market 2: Another spread position
			{ConditionID: "market_up_down_2", Outcome: "Yes", Size: 500, AvgPrice: 0.48},
			{ConditionID: "market_up_down_2", Outcome: "No", Size: 500, AvgPrice: 0.47},
			// Market 3: Third spread position
			{ConditionID: "market_up_down_3", Outcome: "Yes", Size: 750, AvgPrice: 0.44},
			{ConditionID: "market_up_down_3", Outcome: "No", Size: 750, AvgPrice: 0.46},
		}

		// Create trades that simulate bot behavior:
		// - Consistent 2-second intervals (sub-5s threshold)
		// - Consistent position sizes (same 100 for all - low CV)
		// - High win rate through profitable BUY->SELL sequences on SAME market/outcome
		now := time.Now()
		var trades models.TradeList

		// Generate 240 trades (120 complete BUY->SELL round trips)
		// All on the same market/outcome to ensure win rate calculation works
		for i := 0; i < 240; i++ {
			// Keep market and outcome consistent for BUY/SELL pairs
			market := "market_up_down_1"
			outcome := "Yes"

			// Alternating BUY/SELL for profitable round trips
			var side string
			var price float64
			if i%2 == 0 {
				side = "BUY"
				price = 0.45 // Buy low
			} else {
				side = "SELL"
				price = 0.55 // Sell high - guaranteed profit (10% spread)
			}

			// Consistent size for low coefficient of variation
			size := 100.0 // Same size every trade = CV of 0

			trades = append(trades, models.Trade{
				ID:        "trade_" + time.Now().Format("20060102150405") + "_" + string(rune(i)),
				Market:    market,
				Outcome:   outcome,
				Side:      side,
				Size:      size,
				Price:     price,
				MatchTime: now.Add(time.Duration(i*2) * time.Second), // 2-second intervals
			})
		}

		report := s.AnalyzeData(account88888Wallet, positions, trades)

		// Verify wallet address is correct
		if report.WalletAddress != account88888Wallet {
			t.Errorf("expected wallet %s, got %s", account88888Wallet, report.WalletAddress)
		}

		// PRIMARY ASSERTION: Bot score should be >80 as per PRD specification
		if report.BotScore <= 80 {
			t.Errorf("expected bot score >80 for Account88888, got %d", report.BotScore)
		}

		// Verify severity classification
		if report.Severity() != "high" {
			t.Errorf("expected severity 'high' for Account88888, got %q", report.Severity())
		}

		// Verify IsProbableBot returns true with threshold 80
		if !report.IsProbableBot(80) {
			t.Error("expected IsProbableBot(80) to return true for Account88888")
		}

		// Verify all four detectors produced scores
		expectedDetectors := []string{"arbitrage", "timing", "winrate", "sizing"}
		for _, detector := range expectedDetectors {
			if score, ok := report.Scores[detector]; !ok {
				t.Errorf("expected score for detector %q", detector)
			} else if detector == "arbitrage" && score == 0 {
				t.Errorf("expected non-zero arbitrage score for spread positions")
			}
		}

		// Verify detection signals were generated
		if len(report.Signals) == 0 {
			t.Error("expected at least one detection signal")
		}

		// Verify arbitrage signal is present (key indicator for Account88888)
		hasArbitrageSignal := false
		for _, signal := range report.Signals {
			if signal.Type == "arbitrage" {
				hasArbitrageSignal = true
				if signal.Severity != "high" && signal.Severity != "medium" {
					t.Errorf("expected high or medium severity for arbitrage signal, got %q", signal.Severity)
				}
				break
			}
		}
		if !hasArbitrageSignal {
			t.Error("expected arbitrage detection signal for Account88888")
		}

		// Log the score breakdown for debugging
		t.Logf("Account88888 Bot Score: %d (severity: %s)", report.BotScore, report.Severity())
		t.Logf("Detector Scores: arbitrage=%.2f, timing=%.2f, winrate=%.2f, sizing=%.2f",
			report.Scores["arbitrage"], report.Scores["timing"],
			report.Scores["winrate"], report.Scores["sizing"])
		t.Logf("Detection Signals: %d signals generated", len(report.Signals))
	})

	t.Run("edge_case_minimal_bot_indicators", func(t *testing.T) {
		// Test with minimal indicators - should still detect arbitrage
		s := NewScanner()

		// Only arbitrage positions, no trades
		positions := models.PositionList{
			{ConditionID: "market1", Outcome: "Yes", Size: 100, AvgPrice: 0.45},
			{ConditionID: "market1", Outcome: "No", Size: 100, AvgPrice: 0.45},
		}

		report := s.AnalyzeData(account88888Wallet, positions, models.TradeList{})

		// Should have non-zero score from arbitrage alone
		if report.BotScore == 0 {
			t.Error("expected non-zero bot score for arbitrage positions")
		}

		// Arbitrage score should be 1.0 (100%)
		if score, ok := report.Scores["arbitrage"]; !ok || score != 1.0 {
			t.Errorf("expected arbitrage score 1.0, got %.2f", score)
		}
	})

	t.Run("human_trading_pattern_low_score", func(t *testing.T) {
		// Verify that human-like patterns don't trigger high bot scores
		s := NewScanner()

		// No opposing positions (normal trading)
		positions := models.PositionList{
			{ConditionID: "market1", Outcome: "Yes", Size: 100, AvgPrice: 0.60},
			{ConditionID: "market2", Outcome: "No", Size: 75, AvgPrice: 0.35},
		}

		// Irregular trade intervals and sizes (human-like)
		now := time.Now()
		trades := models.TradeList{
			{Market: "m1", Outcome: "Yes", Side: "BUY", Size: 47, Price: 0.55, MatchTime: now},
			{Market: "m1", Outcome: "Yes", Side: "SELL", Size: 23, Price: 0.65, MatchTime: now.Add(15 * time.Minute)},
			{Market: "m2", Outcome: "No", Side: "BUY", Size: 89, Price: 0.30, MatchTime: now.Add(2 * time.Hour)},
			{Market: "m2", Outcome: "No", Side: "SELL", Size: 33, Price: 0.40, MatchTime: now.Add(5 * time.Hour)},
			{Market: "m3", Outcome: "Yes", Side: "BUY", Size: 56, Price: 0.45, MatchTime: now.Add(8 * time.Hour)},
		}

		report := s.AnalyzeData("0x1234567890abcdef1234567890abcdef12345678", positions, trades)

		// Human patterns should result in low bot score (below 80 threshold)
		if report.BotScore >= 80 {
			t.Errorf("expected bot score <80 for human trading patterns, got %d", report.BotScore)
		}

		// Severity should not be "high"
		if report.Severity() == "high" {
			t.Error("expected severity not to be 'high' for human trading patterns")
		}
	})
}
