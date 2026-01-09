package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestRootCmd(t *testing.T) {
	cmd := rootCmd()

	if cmd.Use != "polymarket-watch" {
		t.Errorf("expected Use 'polymarket-watch', got '%s'", cmd.Use)
	}

	if !strings.Contains(cmd.Short, "bot") {
		t.Error("expected Short to mention 'bot'")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	cmd := rootCmd()

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	if !subcommands["scan"] {
		t.Error("expected 'scan' subcommand")
	}

	if !subcommands["batch"] {
		t.Error("expected 'batch' subcommand")
	}
}

func TestScanCmd(t *testing.T) {
	cmd := scanCmd()

	if cmd.Use != "scan" {
		t.Errorf("expected Use 'scan', got '%s'", cmd.Use)
	}

	// Check flags exist
	flag := cmd.Flags().Lookup("wallet")
	if flag == nil {
		t.Error("expected --wallet flag")
	}

	flag = cmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("expected --json flag")
	}

	flag = cmd.Flags().Lookup("output")
	if flag == nil {
		t.Error("expected --output flag")
	}
}

func TestScanCmd_RequiresWallet(t *testing.T) {
	cmd := rootCmd()

	// Execute without wallet
	cmd.SetArgs([]string{"scan"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error when wallet not provided")
	}
}

func TestBatchCmd(t *testing.T) {
	cmd := batchCmd()

	if cmd.Use != "batch" {
		t.Errorf("expected Use 'batch', got '%s'", cmd.Use)
	}

	// Check flags exist
	flag := cmd.Flags().Lookup("file")
	if flag == nil {
		t.Error("expected --file flag")
	}

	flag = cmd.Flags().Lookup("concurrency")
	if flag == nil {
		t.Error("expected --concurrency flag")
	}
	if flag.DefValue != "3" {
		t.Errorf("expected default concurrency 3, got %s", flag.DefValue)
	}

	flag = cmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("expected --json flag")
	}

	flag = cmd.Flags().Lookup("output")
	if flag == nil {
		t.Error("expected --output flag")
	}
}

func TestBatchCmd_RequiresFile(t *testing.T) {
	cmd := rootCmd()

	// Execute without file
	cmd.SetArgs([]string{"batch"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error when file not provided")
	}
}

func TestReadWalletsFromFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
		wantErr  bool
	}{
		{
			name:     "single wallet",
			content:  "0x1234567890abcdef1234567890abcdef12345678",
			expected: []string{"0x1234567890abcdef1234567890abcdef12345678"},
			wantErr:  false,
		},
		{
			name: "multiple wallets",
			content: `0x1111111111111111111111111111111111111111
0x2222222222222222222222222222222222222222
0x3333333333333333333333333333333333333333`,
			expected: []string{
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
				"0x3333333333333333333333333333333333333333",
			},
			wantErr: false,
		},
		{
			name: "with comments and empty lines",
			content: `# This is a comment
0x1111111111111111111111111111111111111111

# Another comment
0x2222222222222222222222222222222222222222

`,
			expected: []string{
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
			},
			wantErr: false,
		},
		{
			name:     "empty file",
			content:  "",
			expected: nil,
			wantErr:  false,
		},
		{
			name: "only comments",
			content: `# comment 1
# comment 2`,
			expected: nil,
			wantErr:  false,
		},
		{
			name: "whitespace trimming",
			content: `  0x1111111111111111111111111111111111111111
	0x2222222222222222222222222222222222222222	`,
			expected: []string{
				"0x1111111111111111111111111111111111111111",
				"0x2222222222222222222222222222222222222222",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "wallets.txt")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			wallets, err := readWalletsFromFile(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(wallets) != len(tt.expected) {
				t.Errorf("expected %d wallets, got %d", len(tt.expected), len(wallets))
				return
			}

			for i, w := range wallets {
				if w != tt.expected[i] {
					t.Errorf("wallet[%d] = %s, expected %s", i, w, tt.expected[i])
				}
			}
		})
	}
}

func TestReadWalletsFromFile_FileNotFound(t *testing.T) {
	_, err := readWalletsFromFile("/nonexistent/path/wallets.txt")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestScanWalletsConcurrently(t *testing.T) {
	// This test verifies the concurrency mechanism works
	// but doesn't make real API calls

	t.Run("empty wallets", func(t *testing.T) {
		ctx := context.Background()
		reports, errs := scanWalletsConcurrently(ctx, []string{}, 3)

		if len(reports) != 0 {
			t.Errorf("expected 0 reports, got %d", len(reports))
		}
		if len(errs) != 0 {
			t.Errorf("expected 0 errors, got %d", len(errs))
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		wallets := []string{
			"0x1111111111111111111111111111111111111111",
			"0x2222222222222222222222222222222222222222",
		}

		// This should return quickly due to cancelled context
		done := make(chan struct{})
		go func() {
			scanWalletsConcurrently(ctx, wallets, 1)
			close(done)
		}()

		select {
		case <-done:
			// Success - function returned
		case <-time.After(5 * time.Second):
			t.Error("scanWalletsConcurrently did not return after context cancellation")
		}
	})

	t.Run("concurrency limit enforcement", func(t *testing.T) {
		// With concurrency 0, should default to 1
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		wallets := []string{
			"0x1111111111111111111111111111111111111111",
		}

		// Should not panic with concurrency 0
		_, _ = scanWalletsConcurrently(ctx, wallets, 0)
	})
}

func TestOutputReport(t *testing.T) {
	report := models.NewScanReport("0x1234567890abcdef1234567890abcdef12345678")
	report.BotScore = 50

	t.Run("terminal output to stdout", func(t *testing.T) {
		// This will output to stdout, which is fine for a quick test
		err := outputReport(report, false, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("json output to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "report.json")

		err := outputReport(report, true, tmpFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Error("output file was not created")
		}

		// Verify JSON content
		content, _ := os.ReadFile(tmpFile)
		if !bytes.Contains(content, []byte("0x1234567890abcdef1234567890abcdef12345678")) {
			t.Error("output file does not contain wallet address")
		}
	})

	t.Run("terminal output to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "report.txt")

		err := outputReport(report, false, tmpFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Error("output file was not created")
		}
	})
}

func TestOutputReports(t *testing.T) {
	reports := []*models.ScanReport{
		models.NewScanReport("0x1111111111111111111111111111111111111111"),
		models.NewScanReport("0x2222222222222222222222222222222222222222"),
	}
	reports[0].BotScore = 80
	reports[1].BotScore = 20

	t.Run("terminal summary to stdout", func(t *testing.T) {
		err := outputReports(reports, false, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("json output to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "reports.json")

		err := outputReports(reports, true, tmpFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Error("output file was not created")
		}

		// Verify JSON array content
		content, _ := os.ReadFile(tmpFile)
		if !bytes.Contains(content, []byte("[")) {
			t.Error("output should be a JSON array")
		}
	})

	t.Run("terminal summary to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "summary.txt")

		err := outputReports(reports, false, tmpFile)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
			t.Error("output file was not created")
		}
	})
}

func TestScanResult(t *testing.T) {
	// Test the scanResult struct
	result := scanResult{
		report: models.NewScanReport("0x1234"),
		err:    nil,
	}

	if result.report == nil {
		t.Error("expected non-nil report")
	}

	if result.err != nil {
		t.Error("expected nil error")
	}
}

func TestMonitorCmd(t *testing.T) {
	cmd := monitorCmd()

	if cmd.Use != "monitor" {
		t.Errorf("expected Use 'monitor', got '%s'", cmd.Use)
	}

	// Check flags exist
	flag := cmd.Flags().Lookup("wallet")
	if flag == nil {
		t.Error("expected --wallet flag")
	}

	flag = cmd.Flags().Lookup("file")
	if flag == nil {
		t.Error("expected --file flag")
	}

	flag = cmd.Flags().Lookup("interval")
	if flag == nil {
		t.Error("expected --interval flag")
	}
	if flag.DefValue != "30s" {
		t.Errorf("expected default interval 30s, got %s", flag.DefValue)
	}

	flag = cmd.Flags().Lookup("threshold")
	if flag == nil {
		t.Error("expected --threshold flag")
	}
	if flag.DefValue != "80" {
		t.Errorf("expected default threshold 80, got %s", flag.DefValue)
	}

	flag = cmd.Flags().Lookup("json")
	if flag == nil {
		t.Error("expected --json flag")
	}
}

func TestMonitorCmd_RequiresWalletOrFile(t *testing.T) {
	cmd := rootCmd()

	// Execute without wallet or file
	cmd.SetArgs([]string{"monitor"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error when neither wallet nor file provided")
	}
}

func TestRootCmd_HasMonitorSubcommand(t *testing.T) {
	cmd := rootCmd()

	subcommands := make(map[string]bool)
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	if !subcommands["monitor"] {
		t.Error("expected 'monitor' subcommand")
	}
}

func TestMonitorEvent(t *testing.T) {
	event := MonitorEvent{
		Wallet:      "0x1234567890abcdef1234567890abcdef12345678",
		TradeCount:  5,
		BotScore:    85,
		Severity:    "high",
		Timestamp:   time.Now(),
		AlertReason: "High bot score detected",
	}

	if event.Wallet == "" {
		t.Error("expected non-empty wallet")
	}

	if event.TradeCount != 5 {
		t.Errorf("expected TradeCount 5, got %d", event.TradeCount)
	}

	if event.BotScore != 85 {
		t.Errorf("expected BotScore 85, got %d", event.BotScore)
	}

	if event.Severity != "high" {
		t.Errorf("expected Severity 'high', got '%s'", event.Severity)
	}
}

func TestMonitorState(t *testing.T) {
	t.Run("new state", func(t *testing.T) {
		state := newMonitorState([]string{"0x1234", "0x5678"})

		if state == nil {
			t.Fatal("expected non-nil state")
		}

		if len(state.wallets) != 2 {
			t.Errorf("expected 2 wallets, got %d", len(state.wallets))
		}

		// Last check should be zero initially
		lastCheck := state.getLastCheck("0x1234")
		if !lastCheck.IsZero() {
			t.Error("expected zero last check for new wallet")
		}
	})

	t.Run("update last check", func(t *testing.T) {
		state := newMonitorState([]string{"0x1234"})
		now := time.Now()

		state.setLastCheck("0x1234", now)
		lastCheck := state.getLastCheck("0x1234")

		if !lastCheck.Equal(now) {
			t.Errorf("expected last check %v, got %v", now, lastCheck)
		}
	})

	t.Run("get wallets", func(t *testing.T) {
		wallets := []string{"0x1234", "0x5678", "0x9abc"}
		state := newMonitorState(wallets)

		stateWallets := state.getWallets()
		if len(stateWallets) != 3 {
			t.Errorf("expected 3 wallets, got %d", len(stateWallets))
		}
	})
}

func TestFilterNewTrades(t *testing.T) {
	now := time.Now()

	trades := models.TradeList{
		{ID: "1", MatchTime: now.Add(-5 * time.Minute)},
		{ID: "2", MatchTime: now.Add(-2 * time.Minute)},
		{ID: "3", MatchTime: now.Add(-30 * time.Second)},
	}

	t.Run("filter by time threshold", func(t *testing.T) {
		since := now.Add(-3 * time.Minute)
		newTrades := filterNewTrades(trades, since)

		if len(newTrades) != 2 {
			t.Errorf("expected 2 new trades, got %d", len(newTrades))
		}

		if newTrades[0].ID != "2" {
			t.Errorf("expected first trade ID '2', got '%s'", newTrades[0].ID)
		}
	})

	t.Run("zero since returns all trades", func(t *testing.T) {
		newTrades := filterNewTrades(trades, time.Time{})

		if len(newTrades) != 3 {
			t.Errorf("expected 3 trades, got %d", len(newTrades))
		}
	})

	t.Run("future since returns none", func(t *testing.T) {
		since := now.Add(1 * time.Hour)
		newTrades := filterNewTrades(trades, since)

		if len(newTrades) != 0 {
			t.Errorf("expected 0 trades, got %d", len(newTrades))
		}
	})

	t.Run("empty trades", func(t *testing.T) {
		newTrades := filterNewTrades(models.TradeList{}, now)

		if len(newTrades) != 0 {
			t.Errorf("expected 0 trades, got %d", len(newTrades))
		}
	})
}

func TestFormatMonitorAlert(t *testing.T) {
	event := MonitorEvent{
		Wallet:      "0x1234567890abcdef1234567890abcdef12345678",
		TradeCount:  10,
		BotScore:    92,
		Severity:    "high",
		Timestamp:   time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC),
		AlertReason: "High bot score detected",
	}

	alert := formatMonitorAlert(event)

	if !strings.Contains(alert, "0x1234...5678") {
		t.Errorf("alert should contain truncated wallet address, got: %s", alert)
	}

	if !strings.Contains(alert, "92") {
		t.Errorf("alert should contain bot score, got: %s", alert)
	}

	if !strings.Contains(alert, "10") {
		t.Errorf("alert should contain trade count, got: %s", alert)
	}

	// Check for HIGH (uppercase) in the alert
	if !strings.Contains(alert, "HIGH") {
		t.Errorf("alert should indicate high severity (HIGH), got: %s", alert)
	}
}

func TestShouldAlert(t *testing.T) {
	tests := []struct {
		name      string
		score     int
		threshold int
		expected  bool
	}{
		{"score above threshold", 85, 80, true},
		{"score equal to threshold", 80, 80, true},
		{"score below threshold", 75, 80, false},
		{"zero threshold alerts all", 50, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldAlert(tt.score, tt.threshold)
			if result != tt.expected {
				t.Errorf("shouldAlert(%d, %d) = %v, want %v", tt.score, tt.threshold, result, tt.expected)
			}
		})
	}
}
