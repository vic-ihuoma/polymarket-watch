// Package output provides output formatters for scan results.
package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/discovery"
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewTerminalOutput(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		out := NewTerminalOutput()
		if out == nil {
			t.Fatal("expected non-nil TerminalOutput")
		}
		if out.writer == nil {
			t.Error("expected non-nil writer")
		}
		if !out.colorEnabled {
			t.Error("expected color enabled by default")
		}
	})

	t.Run("with custom writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		out := NewTerminalOutput(WithWriter(buf))
		if out.writer != buf {
			t.Error("expected custom writer")
		}
	})

	t.Run("with color disabled", func(t *testing.T) {
		out := NewTerminalOutput(WithColor(false))
		if out.colorEnabled {
			t.Error("expected color disabled")
		}
	})
}

func TestTerminalOutput_Print(t *testing.T) {
	baseTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		report   *models.ScanReport
		contains []string
	}{
		{
			name: "high bot score",
			report: &models.ScanReport{
				WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
				ScanTime:      baseTime,
				BotScore:      85,
				Scores: map[string]float64{
					"arbitrage": 1.0,
					"timing":    0.8,
					"winrate":   0.7,
					"sizing":    0.5,
				},
				Signals: []models.DetectionSignal{
					{
						Type:        "arbitrage",
						Description: "Detected 3 profitable arbitrage pairs",
						Severity:    "high",
						Evidence: map[string]interface{}{
							"profitable_pairs": 3,
							"total_spread":     0.15,
						},
					},
				},
				Stats: models.ScanStats{
					TotalTrades:       250,
					TotalPositions:    15,
					TotalVolume:       50000.0,
					UniqueMarkets:     8,
					FirstTradeTime:    baseTime.Add(-30 * 24 * time.Hour),
					LastTradeTime:     baseTime.Add(-1 * time.Hour),
					WinRate:           0.96,
					AvgPositionSize:   500.0,
					OpposingPairCount: 3,
				},
			},
			contains: []string{
				"0x1234...5678",         // Truncated wallet address
				"85",                    // Bot score
				"HIGH",                  // Severity
				"arbitrage",             // Detector name
				"Detected 3 profitable", // Signal description
				"250",                   // Total trades
				"$50,000",               // Volume (formatted)
				"96.0%",                 // Win rate formatted
			},
		},
		{
			name: "low bot score",
			report: &models.ScanReport{
				WalletAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
				ScanTime:      baseTime,
				BotScore:      25,
				Scores: map[string]float64{
					"arbitrage": 0.0,
					"timing":    0.3,
					"winrate":   0.2,
					"sizing":    0.4,
				},
				Signals: []models.DetectionSignal{},
				Stats: models.ScanStats{
					TotalTrades:    50,
					TotalPositions: 5,
					TotalVolume:    5000.0,
					UniqueMarkets:  4,
				},
			},
			contains: []string{
				"0xabcd...ef12", // Truncated wallet
				"25",            // Bot score
				"LOW",           // Severity
				"50",            // Total trades
			},
		},
		{
			name: "medium bot score with multiple signals",
			report: &models.ScanReport{
				WalletAddress: "0x9876543210fedcba9876543210fedcba98765432",
				ScanTime:      baseTime,
				BotScore:      65,
				Scores: map[string]float64{
					"arbitrage": 0.5,
					"timing":    0.7,
					"winrate":   0.6,
					"sizing":    0.8,
				},
				Signals: []models.DetectionSignal{
					{
						Type:        "timing",
						Description: "Fast trade intervals detected",
						Severity:    "medium",
					},
					{
						Type:        "sizing",
						Description: "Consistent position sizing",
						Severity:    "medium",
					},
				},
				Stats: models.ScanStats{
					TotalTrades:    100,
					TotalPositions: 10,
					TotalVolume:    20000.0,
					UniqueMarkets:  6,
				},
			},
			contains: []string{
				"65",             // Bot score
				"MEDIUM",         // Severity
				"timing",         // Signal type
				"sizing",         // Signal type
				"Fast trade",     // Signal description
				"Consistent pos", // Signal description (partial)
			},
		},
		{
			name: "empty report",
			report: &models.ScanReport{
				WalletAddress: "0x0000000000000000000000000000000000000000",
				ScanTime:      baseTime,
				BotScore:      0,
				Scores:        map[string]float64{},
				Signals:       []models.DetectionSignal{},
				Stats:         models.ScanStats{},
			},
			contains: []string{
				"0x0000...0000", // Truncated wallet
				"0",             // Bot score
				"LOW",           // Severity
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			out := NewTerminalOutput(WithWriter(buf), WithColor(false))

			err := out.Print(tt.report)
			if err != nil {
				t.Fatalf("Print() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected content %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestTerminalOutput_PrintSummary(t *testing.T) {
	baseTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	reports := []*models.ScanReport{
		{
			WalletAddress: "0x1111111111111111111111111111111111111111",
			BotScore:      85,
			ScanTime:      baseTime,
		},
		{
			WalletAddress: "0x2222222222222222222222222222222222222222",
			BotScore:      45,
			ScanTime:      baseTime,
		},
		{
			WalletAddress: "0x3333333333333333333333333333333333333333",
			BotScore:      92,
			ScanTime:      baseTime,
		},
	}

	t.Run("multiple reports summary", func(t *testing.T) {
		buf := &bytes.Buffer{}
		out := NewTerminalOutput(WithWriter(buf), WithColor(false))

		err := out.PrintSummary(reports)
		if err != nil {
			t.Fatalf("PrintSummary() error = %v", err)
		}

		output := buf.String()

		// Should contain all wallet addresses (truncated)
		if !strings.Contains(output, "0x1111...1111") {
			t.Error("missing first wallet")
		}
		if !strings.Contains(output, "0x2222...2222") {
			t.Error("missing second wallet")
		}
		if !strings.Contains(output, "0x3333...3333") {
			t.Error("missing third wallet")
		}

		// Should contain scores
		if !strings.Contains(output, "85") {
			t.Error("missing first score")
		}
		if !strings.Contains(output, "92") {
			t.Error("missing third score")
		}

		// Should contain summary statistics
		if !strings.Contains(output, "3") {
			t.Error("missing total count")
		}
	})

	t.Run("empty reports", func(t *testing.T) {
		buf := &bytes.Buffer{}
		out := NewTerminalOutput(WithWriter(buf), WithColor(false))

		err := out.PrintSummary([]*models.ScanReport{})
		if err != nil {
			t.Fatalf("PrintSummary() error = %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No reports") {
			t.Error("expected 'No reports' message for empty input")
		}
	})
}

func TestTruncateWallet(t *testing.T) {
	tests := []struct {
		name   string
		wallet string
		want   string
	}{
		{
			name:   "full ethereum address",
			wallet: "0x1234567890abcdef1234567890abcdef12345678",
			want:   "0x1234...5678",
		},
		{
			name:   "short address",
			wallet: "0x12345678",
			want:   "0x12345678",
		},
		{
			name:   "very short",
			wallet: "0x123",
			want:   "0x123",
		},
		{
			name:   "empty",
			wallet: "",
			want:   "",
		},
		{
			name:   "exactly 14 chars",
			wallet: "0x123456789012",
			want:   "0x123456789012",
		},
		{
			name:   "15 chars - should truncate",
			wallet: "0x1234567890123",
			want:   "0x1234...0123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateWallet(tt.wallet)
			if got != tt.want {
				t.Errorf("truncateWallet(%q) = %q, want %q", tt.wallet, got, tt.want)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{
			name:  "zero",
			value: 0.0,
			want:  "0.0%",
		},
		{
			name:  "half",
			value: 0.5,
			want:  "50.0%",
		},
		{
			name:  "full",
			value: 1.0,
			want:  "100.0%",
		},
		{
			name:  "96 percent",
			value: 0.96,
			want:  "96.0%",
		},
		{
			name:  "fractional",
			value: 0.333,
			want:  "33.3%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPercent(tt.value)
			if got != tt.want {
				t.Errorf("formatPercent(%f) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestFormatVolume(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  string
	}{
		{
			name:  "zero",
			value: 0.0,
			want:  "$0",
		},
		{
			name:  "small",
			value: 500.0,
			want:  "$500",
		},
		{
			name:  "thousands",
			value: 5000.0,
			want:  "$5,000",
		},
		{
			name:  "large",
			value: 1234567.89,
			want:  "$1,234,567",
		},
		{
			name:  "millions",
			value: 50000000.0,
			want:  "$50,000,000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVolume(tt.value)
			if got != tt.want {
				t.Errorf("formatVolume(%f) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestSeverityLabel(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     string
	}{
		{
			name:     "low",
			severity: "low",
			want:     "LOW",
		},
		{
			name:     "medium",
			severity: "medium",
			want:     "MEDIUM",
		},
		{
			name:     "high",
			severity: "high",
			want:     "HIGH",
		},
		{
			name:     "unknown",
			severity: "unknown",
			want:     "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := severityLabel(tt.severity)
			if got != tt.want {
				t.Errorf("severityLabel(%q) = %q, want %q", tt.severity, got, tt.want)
			}
		})
	}
}

func TestTerminalOutput_Interface(t *testing.T) {
	// Compile-time check that TerminalOutput implements OutputFormatter
	var _ OutputFormatter = (*TerminalOutput)(nil)
}

func TestTerminalOutput_PrintDiscoveryResult(t *testing.T) {
	baseTime := time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		result   *discovery.DiscoveryResult
		contains []string
	}{
		{
			name: "discovery result with bots detected",
			result: &discovery.DiscoveryResult{
				Wallets: []discovery.DiscoveredWallet{
					{
						Address:       "0x1234567890abcdef1234567890abcdef12345678",
						TotalAmount:   5000.0,
						MarketCount:   3,
						PositionCount: 5,
						ScanReport: &models.ScanReport{
							WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
							BotScore:      92,
							ScanTime:      baseTime,
							Scores:        map[string]float64{"arbitrage": 1.0},
							Signals:       []models.DetectionSignal{},
						},
					},
					{
						Address:       "0xabcdef1234567890abcdef1234567890abcdef12",
						TotalAmount:   2000.0,
						MarketCount:   2,
						PositionCount: 3,
						ScanReport: &models.ScanReport{
							WalletAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
							BotScore:      45,
							ScanTime:      baseTime,
							Scores:        map[string]float64{"timing": 0.5},
							Signals:       []models.DetectionSignal{},
						},
					},
				},
				Markets: []discovery.MarketInfo{
					{
						ConditionID:  "0xmarket123",
						Slug:         "will-btc-reach-100k",
						Question:     "Will BTC reach $100k?",
						HoldersCount: 50,
						Volume:       1000000.0,
						Liquidity:    50000.0,
					},
				},
				Stats: discovery.DiscoveryStats{
					MarketsScanned: 1,
					WalletsFound:   2,
					WalletsScanned: 2,
					BotsDetected:   1,
					ScanErrors:     0,
					StartTime:      baseTime.Add(-5 * time.Minute),
					EndTime:        baseTime,
				},
			},
			contains: []string{
				"DISCOVERY",           // Header
				"0x1234...5678",       // First wallet truncated
				"92",                  // Bot score
				"HIGH",                // Severity
				"0xabcd...ef12",       // Second wallet truncated
				"45",                  // Second bot score
				"Markets Scanned",     // Stats label
				"1",                   // Markets scanned count
				"Wallets Found",       // Stats label
				"2",                   // Wallets found count
				"Bots Detected",       // Stats label
				"will-btc-reach-100k", // Market slug
			},
		},
		{
			name: "discovery result with no scan (no_scan mode)",
			result: &discovery.DiscoveryResult{
				Wallets: []discovery.DiscoveredWallet{
					{
						Address:       "0x9999999999999999999999999999999999999999",
						TotalAmount:   10000.0,
						MarketCount:   5,
						PositionCount: 8,
						ScanReport:    nil, // No scan report when NoScan is true
					},
				},
				Markets: []discovery.MarketInfo{
					{
						ConditionID:  "0xmarket456",
						Slug:         "us-election-2024",
						Question:     "Who will win the 2024 election?",
						HoldersCount: 100,
						Volume:       5000000.0,
						Liquidity:    200000.0,
					},
				},
				Stats: discovery.DiscoveryStats{
					MarketsScanned: 1,
					WalletsFound:   1,
					WalletsScanned: 0, // No scans performed
					BotsDetected:   0,
					ScanErrors:     0,
					StartTime:      baseTime.Add(-1 * time.Minute),
					EndTime:        baseTime,
				},
			},
			contains: []string{
				"DISCOVERY",
				"0x9999...9999",
				"10000",       // Total amount
				"5",           // Market count
				"us-election", // Market slug
				"N/A",         // No score available
			},
		},
		{
			name: "empty discovery result",
			result: &discovery.DiscoveryResult{
				Wallets: []discovery.DiscoveredWallet{},
				Markets: []discovery.MarketInfo{},
				Stats: discovery.DiscoveryStats{
					MarketsScanned: 0,
					WalletsFound:   0,
					WalletsScanned: 0,
					BotsDetected:   0,
					ScanErrors:     0,
					StartTime:      baseTime,
					EndTime:        baseTime,
				},
			},
			contains: []string{
				"DISCOVERY",
				"No wallets discovered",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			out := NewTerminalOutput(WithWriter(buf), WithColor(false))

			err := out.PrintDiscoveryResult(tt.result)
			if err != nil {
				t.Fatalf("PrintDiscoveryResult() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected content %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestTerminalOutput_PrintDiscoveryResult_NilResult(t *testing.T) {
	buf := &bytes.Buffer{}
	out := NewTerminalOutput(WithWriter(buf), WithColor(false))

	err := out.PrintDiscoveryResult(nil)
	if err == nil {
		t.Error("PrintDiscoveryResult should return error for nil result")
	}
}
