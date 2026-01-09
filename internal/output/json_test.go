package output

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewJSONOutput(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		jo := NewJSONOutput()
		if jo == nil {
			t.Fatal("NewJSONOutput returned nil")
		}
		if jo.writer == nil {
			t.Error("writer should not be nil")
		}
		if !jo.prettyPrint {
			t.Error("prettyPrint should default to true")
		}
	})

	t.Run("custom writer", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))
		if jo.writer != &buf {
			t.Error("writer should be the custom buffer")
		}
	})

	t.Run("pretty print disabled", func(t *testing.T) {
		jo := NewJSONOutput(WithPrettyPrint(false))
		if jo.prettyPrint {
			t.Error("prettyPrint should be false when disabled")
		}
	})
}

func TestJSONOutput_Print(t *testing.T) {
	fixedTime := time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)

	t.Run("outputs valid JSON", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))

		report := &models.ScanReport{
			WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
			ScanTime:      fixedTime,
			BotScore:      85,
			Scores: map[string]float64{
				"arbitrage": 1.0,
				"timing":    0.5,
				"winrate":   0.0,
				"sizing":    0.2,
			},
			Signals: []models.DetectionSignal{
				{
					Type:        "arbitrage",
					Description: "Detected opposing positions",
					Severity:    "high",
					Evidence: map[string]interface{}{
						"spread": 0.05,
					},
				},
			},
			Stats: models.ScanStats{
				TotalTrades:       100,
				TotalPositions:    10,
				TotalVolume:       50000.0,
				UniqueMarkets:     5,
				FirstTradeTime:    fixedTime.Add(-24 * time.Hour),
				LastTradeTime:     fixedTime,
				WinRate:           0.95,
				AvgPositionSize:   500.0,
				OpposingPairCount: 3,
			},
		}

		err := jo.Print(report)
		if err != nil {
			t.Fatalf("Print returned error: %v", err)
		}

		// Verify it's valid JSON
		var decoded models.ScanReport
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		// Verify key fields
		if decoded.WalletAddress != report.WalletAddress {
			t.Errorf("WalletAddress mismatch: got %s, want %s", decoded.WalletAddress, report.WalletAddress)
		}
		if decoded.BotScore != report.BotScore {
			t.Errorf("BotScore mismatch: got %d, want %d", decoded.BotScore, report.BotScore)
		}
		if len(decoded.Signals) != len(report.Signals) {
			t.Errorf("Signals count mismatch: got %d, want %d", len(decoded.Signals), len(report.Signals))
		}
	})

	t.Run("pretty print enabled", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf), WithPrettyPrint(true))

		report := &models.ScanReport{
			WalletAddress: "0x1234",
			ScanTime:      fixedTime,
			BotScore:      50,
			Scores:        map[string]float64{"arbitrage": 0.5},
			Signals:       []models.DetectionSignal{},
		}

		err := jo.Print(report)
		if err != nil {
			t.Fatalf("Print returned error: %v", err)
		}

		output := buf.String()
		// Pretty print should have newlines and indentation
		if !strings.Contains(output, "\n") {
			t.Error("pretty print output should contain newlines")
		}
		if !strings.Contains(output, "  ") {
			t.Error("pretty print output should contain indentation")
		}
	})

	t.Run("compact output", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf), WithPrettyPrint(false))

		report := &models.ScanReport{
			WalletAddress: "0x1234",
			ScanTime:      fixedTime,
			BotScore:      50,
			Scores:        map[string]float64{"arbitrage": 0.5},
			Signals:       []models.DetectionSignal{},
		}

		err := jo.Print(report)
		if err != nil {
			t.Fatalf("Print returned error: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		// Compact output should be a single line (no embedded newlines except for trailing)
		lines := strings.Split(output, "\n")
		if len(lines) != 1 {
			t.Errorf("compact output should be single line, got %d lines", len(lines))
		}
	})

	t.Run("nil report", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))

		err := jo.Print(nil)
		if err == nil {
			t.Error("Print should return error for nil report")
		}
	})
}

func TestJSONOutput_PrintSummary(t *testing.T) {
	fixedTime := time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)

	t.Run("outputs valid JSON array", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))

		reports := []*models.ScanReport{
			{
				WalletAddress: "0x1111111111111111111111111111111111111111",
				ScanTime:      fixedTime,
				BotScore:      85,
				Scores:        map[string]float64{"arbitrage": 1.0},
				Signals:       []models.DetectionSignal{},
			},
			{
				WalletAddress: "0x2222222222222222222222222222222222222222",
				ScanTime:      fixedTime,
				BotScore:      30,
				Scores:        map[string]float64{"arbitrage": 0.0},
				Signals:       []models.DetectionSignal{},
			},
		}

		err := jo.PrintSummary(reports)
		if err != nil {
			t.Fatalf("PrintSummary returned error: %v", err)
		}

		// Verify it's valid JSON array
		var decoded []*models.ScanReport
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("Output is not valid JSON array: %v", err)
		}

		if len(decoded) != len(reports) {
			t.Errorf("decoded length mismatch: got %d, want %d", len(decoded), len(reports))
		}
	})

	t.Run("empty reports", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))

		err := jo.PrintSummary([]*models.ScanReport{})
		if err != nil {
			t.Fatalf("PrintSummary returned error: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if output != "[]" {
			t.Errorf("empty reports should output empty array, got: %s", output)
		}
	})

	t.Run("nil reports", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))

		err := jo.PrintSummary(nil)
		if err != nil {
			t.Fatalf("PrintSummary returned error: %v", err)
		}

		output := strings.TrimSpace(buf.String())
		if output != "[]" {
			t.Errorf("nil reports should output empty array, got: %s", output)
		}
	})

	t.Run("preserves order", func(t *testing.T) {
		var buf bytes.Buffer
		jo := NewJSONOutput(WithJSONWriter(&buf))

		reports := []*models.ScanReport{
			{WalletAddress: "0xfirst", BotScore: 10, Scores: map[string]float64{}, Signals: []models.DetectionSignal{}},
			{WalletAddress: "0xsecond", BotScore: 50, Scores: map[string]float64{}, Signals: []models.DetectionSignal{}},
			{WalletAddress: "0xthird", BotScore: 90, Scores: map[string]float64{}, Signals: []models.DetectionSignal{}},
		}

		err := jo.PrintSummary(reports)
		if err != nil {
			t.Fatalf("PrintSummary returned error: %v", err)
		}

		var decoded []*models.ScanReport
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		for i, report := range reports {
			if decoded[i].WalletAddress != report.WalletAddress {
				t.Errorf("order not preserved at index %d: got %s, want %s",
					i, decoded[i].WalletAddress, report.WalletAddress)
			}
		}
	})
}

func TestJSONOutput_Interface(t *testing.T) {
	// Compile-time interface check
	var _ OutputFormatter = (*JSONOutput)(nil)
}

func TestWriteToFile(t *testing.T) {
	fixedTime := time.Date(2026, 1, 9, 12, 0, 0, 0, time.UTC)

	t.Run("writes report to file", func(t *testing.T) {
		// Use temp file
		tmpDir := t.TempDir()
		filePath := tmpDir + "/report.json"

		report := &models.ScanReport{
			WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
			ScanTime:      fixedTime,
			BotScore:      75,
			Scores: map[string]float64{
				"arbitrage": 0.8,
				"timing":    0.6,
			},
			Signals: []models.DetectionSignal{},
		}

		err := WriteReportToFile(report, filePath)
		if err != nil {
			t.Fatalf("WriteReportToFile returned error: %v", err)
		}

		// Read back and verify
		data, err := readFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var decoded models.ScanReport
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("file content is not valid JSON: %v", err)
		}

		if decoded.WalletAddress != report.WalletAddress {
			t.Errorf("WalletAddress mismatch: got %s, want %s",
				decoded.WalletAddress, report.WalletAddress)
		}
	})

	t.Run("writes summary to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := tmpDir + "/summary.json"

		reports := []*models.ScanReport{
			{WalletAddress: "0x1111", BotScore: 80, Scores: map[string]float64{}, Signals: []models.DetectionSignal{}},
			{WalletAddress: "0x2222", BotScore: 40, Scores: map[string]float64{}, Signals: []models.DetectionSignal{}},
		}

		err := WriteSummaryToFile(reports, filePath)
		if err != nil {
			t.Fatalf("WriteSummaryToFile returned error: %v", err)
		}

		data, err := readFile(filePath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var decoded []*models.ScanReport
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("file content is not valid JSON: %v", err)
		}

		if len(decoded) != len(reports) {
			t.Errorf("length mismatch: got %d, want %d", len(decoded), len(reports))
		}
	})

	t.Run("returns error for nil report", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := tmpDir + "/nil.json"

		err := WriteReportToFile(nil, filePath)
		if err == nil {
			t.Error("WriteReportToFile should return error for nil report")
		}
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		report := &models.ScanReport{
			WalletAddress: "0x1234",
			BotScore:      50,
			Scores:        map[string]float64{},
			Signals:       []models.DetectionSignal{},
		}

		err := WriteReportToFile(report, "/nonexistent/directory/file.json")
		if err == nil {
			t.Error("WriteReportToFile should return error for invalid path")
		}
	})
}

// readFile is a helper to read a file's contents.
func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
