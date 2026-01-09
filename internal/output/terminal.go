// Package output provides output formatters for scan results.
package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/victorihuoma/polymarket-watch/internal/models"
)

// OutputFormatter defines the interface for outputting scan results.
type OutputFormatter interface {
	// Print outputs a single scan report.
	Print(report *models.ScanReport) error
	// PrintSummary outputs a summary of multiple scan reports.
	PrintSummary(reports []*models.ScanReport) error
}

// TerminalOutput formats scan results for terminal display.
type TerminalOutput struct {
	writer       io.Writer
	colorEnabled bool
}

// Option is a functional option for configuring TerminalOutput.
type Option func(*TerminalOutput)

// NewTerminalOutput creates a new TerminalOutput with the given options.
func NewTerminalOutput(opts ...Option) *TerminalOutput {
	t := &TerminalOutput{
		writer:       os.Stdout,
		colorEnabled: true,
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(t *TerminalOutput) {
		t.writer = w
	}
}

// WithColor enables or disables color output.
func WithColor(enabled bool) Option {
	return func(t *TerminalOutput) {
		t.colorEnabled = enabled
	}
}

// Print outputs a single scan report to the terminal.
func (t *TerminalOutput) Print(report *models.ScanReport) error {
	// Header
	fmt.Fprintln(t.writer)
	fmt.Fprintln(t.writer, strings.Repeat("═", 60))
	fmt.Fprintf(t.writer, "  POLYMARKET WALLET SCAN REPORT\n")
	fmt.Fprintln(t.writer, strings.Repeat("═", 60))
	fmt.Fprintln(t.writer)

	// Wallet info
	fmt.Fprintf(t.writer, "  Wallet:    %s\n", truncateWallet(report.WalletAddress))
	fmt.Fprintf(t.writer, "  Scanned:   %s\n", report.ScanTime.Format("2006-01-02 15:04:05 UTC"))
	fmt.Fprintln(t.writer)

	// Bot score with color
	severity := report.Severity()
	scoreStr := fmt.Sprintf("%d", report.BotScore)
	severityStr := severityLabel(severity)

	if t.colorEnabled {
		scoreColor := t.getScoreColor(severity)
		scoreStr = scoreColor.Sprintf("%d", report.BotScore)
		severityStr = scoreColor.Sprintf("%s", severityLabel(severity))
	}

	fmt.Fprintf(t.writer, "  Bot Score: %s / 100\n", scoreStr)
	fmt.Fprintf(t.writer, "  Severity:  %s\n", severityStr)
	fmt.Fprintln(t.writer)

	// Detector scores table
	t.printDetectorScores(report)

	// Detection signals
	if len(report.Signals) > 0 {
		t.printSignals(report)
	}

	// Trading statistics
	t.printStats(report)

	fmt.Fprintln(t.writer, strings.Repeat("─", 60))
	fmt.Fprintln(t.writer)

	return nil
}

// printDetectorScores outputs the individual detector scores table.
func (t *TerminalOutput) printDetectorScores(report *models.ScanReport) {
	fmt.Fprintln(t.writer, "  DETECTOR SCORES")
	fmt.Fprintln(t.writer, strings.Repeat("─", 60))

	table := tablewriter.NewTable(t.writer)
	table.Header("Detector", "Score", "Contribution")

	// Sort detectors for consistent output
	detectors := make([]string, 0, len(report.Scores))
	for d := range report.Scores {
		detectors = append(detectors, d)
	}
	sort.Strings(detectors)

	weights := map[string]float64{
		"arbitrage": 0.35,
		"timing":    0.25,
		"winrate":   0.25,
		"sizing":    0.15,
	}

	for _, detector := range detectors {
		score := report.Scores[detector]
		weight := weights[detector]
		contribution := score * weight * 100

		scoreStr := fmt.Sprintf("%.1f%%", score*100)
		contribStr := fmt.Sprintf("%.1f pts", contribution)

		table.Append(detector, scoreStr, contribStr)
	}

	table.Render()
	fmt.Fprintln(t.writer)
}

// printSignals outputs the detection signals.
func (t *TerminalOutput) printSignals(report *models.ScanReport) {
	fmt.Fprintln(t.writer, "  DETECTION SIGNALS")
	fmt.Fprintln(t.writer, strings.Repeat("─", 60))

	for _, signal := range report.Signals {
		severityLabel := strings.ToUpper(signal.Severity)
		if t.colorEnabled {
			severityColor := t.getScoreColor(signal.Severity)
			severityLabel = severityColor.Sprint(severityLabel)
		}

		fmt.Fprintf(t.writer, "  [%s] %s\n", severityLabel, signal.Type)
		fmt.Fprintf(t.writer, "         %s\n", signal.Description)

		if len(signal.Evidence) > 0 {
			fmt.Fprint(t.writer, "         Evidence: ")
			evidenceStrs := make([]string, 0, len(signal.Evidence))
			for k, v := range signal.Evidence {
				evidenceStrs = append(evidenceStrs, fmt.Sprintf("%s=%v", k, v))
			}
			sort.Strings(evidenceStrs)
			fmt.Fprintln(t.writer, strings.Join(evidenceStrs, ", "))
		}
		fmt.Fprintln(t.writer)
	}
}

// printStats outputs the trading statistics.
func (t *TerminalOutput) printStats(report *models.ScanReport) {
	fmt.Fprintln(t.writer, "  TRADING STATISTICS")
	fmt.Fprintln(t.writer, strings.Repeat("─", 60))

	table := tablewriter.NewTable(t.writer)
	table.Header("Metric", "Value")

	stats := report.Stats

	table.Append("Total Trades", fmt.Sprintf("%d", stats.TotalTrades))
	table.Append("Total Positions", fmt.Sprintf("%d", stats.TotalPositions))
	table.Append("Total Volume", formatVolume(stats.TotalVolume))
	table.Append("Unique Markets", fmt.Sprintf("%d", stats.UniqueMarkets))
	table.Append("Win Rate", formatPercent(stats.WinRate))
	table.Append("Avg Position Size", fmt.Sprintf("%.2f", stats.AvgPositionSize))
	table.Append("Opposing Pairs", fmt.Sprintf("%d", stats.OpposingPairCount))

	if !stats.FirstTradeTime.IsZero() {
		table.Append("First Trade", stats.FirstTradeTime.Format("2006-01-02"))
	}
	if !stats.LastTradeTime.IsZero() {
		table.Append("Last Trade", stats.LastTradeTime.Format("2006-01-02"))
	}

	duration := stats.TradingDuration()
	if duration > 0 {
		days := int(duration.Hours() / 24)
		table.Append("Trading Duration", fmt.Sprintf("%d days", days))
	}

	tradesPerDay := stats.TradesPerDay()
	if tradesPerDay > 0 {
		table.Append("Trades/Day", fmt.Sprintf("%.1f", tradesPerDay))
	}

	table.Render()
	fmt.Fprintln(t.writer)
}

// PrintSummary outputs a summary of multiple scan reports.
func (t *TerminalOutput) PrintSummary(reports []*models.ScanReport) error {
	if len(reports) == 0 {
		fmt.Fprintln(t.writer, "No reports to summarize.")
		return nil
	}

	fmt.Fprintln(t.writer)
	fmt.Fprintln(t.writer, strings.Repeat("═", 80))
	fmt.Fprintf(t.writer, "  BATCH SCAN SUMMARY (%d wallets)\n", len(reports))
	fmt.Fprintln(t.writer, strings.Repeat("═", 80))
	fmt.Fprintln(t.writer)

	table := tablewriter.NewTable(t.writer)
	table.Header("Wallet", "Score", "Severity", "Signals")

	// Sort by bot score descending
	sorted := make([]*models.ScanReport, len(reports))
	copy(sorted, reports)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].BotScore > sorted[j].BotScore
	})

	var highCount, mediumCount, lowCount int

	for _, report := range sorted {
		severity := report.Severity()
		switch severity {
		case "high":
			highCount++
		case "medium":
			mediumCount++
		default:
			lowCount++
		}

		severityStr := severityLabel(severity)
		if t.colorEnabled {
			severityColor := t.getScoreColor(severity)
			severityStr = severityColor.Sprint(severityStr)
		}

		signalCount := len(report.Signals)

		table.Append(
			truncateWallet(report.WalletAddress),
			fmt.Sprintf("%d", report.BotScore),
			severityStr,
			fmt.Sprintf("%d", signalCount),
		)
	}

	table.Render()
	fmt.Fprintln(t.writer)

	// Summary statistics
	fmt.Fprintln(t.writer, "  SUMMARY")
	fmt.Fprintln(t.writer, strings.Repeat("─", 40))
	fmt.Fprintf(t.writer, "  Total Wallets:   %d\n", len(reports))
	fmt.Fprintf(t.writer, "  High Severity:   %d\n", highCount)
	fmt.Fprintf(t.writer, "  Medium Severity: %d\n", mediumCount)
	fmt.Fprintf(t.writer, "  Low Severity:    %d\n", lowCount)
	fmt.Fprintln(t.writer)

	return nil
}

// getScoreColor returns the appropriate color for a severity level.
func (t *TerminalOutput) getScoreColor(severity string) *color.Color {
	switch severity {
	case "high":
		return color.New(color.FgRed, color.Bold)
	case "medium":
		return color.New(color.FgYellow, color.Bold)
	default:
		return color.New(color.FgGreen)
	}
}

// truncateWallet shortens a wallet address for display.
// Full address: 0x1234567890abcdef1234567890abcdef12345678
// Truncated:    0x1234...5678
func truncateWallet(wallet string) string {
	if len(wallet) <= 14 {
		return wallet
	}
	return wallet[:6] + "..." + wallet[len(wallet)-4:]
}

// formatPercent formats a decimal value as a percentage.
func formatPercent(value float64) string {
	return fmt.Sprintf("%.1f%%", value*100)
}

// formatVolume formats a USD volume with commas.
func formatVolume(value float64) string {
	intVal := int64(value)
	str := fmt.Sprintf("%d", intVal)

	// Add commas
	n := len(str)
	if n <= 3 {
		return "$" + str
	}

	var result strings.Builder
	result.WriteByte('$')

	remainder := n % 3
	if remainder > 0 {
		result.WriteString(str[:remainder])
		if remainder < n {
			result.WriteByte(',')
		}
	}

	for i := remainder; i < n; i += 3 {
		result.WriteString(str[i : i+3])
		if i+3 < n {
			result.WriteByte(',')
		}
	}

	return result.String()
}

// severityLabel returns the uppercase label for a severity level.
func severityLabel(severity string) string {
	return strings.ToUpper(severity)
}
