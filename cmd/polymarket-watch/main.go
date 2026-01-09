// Package main provides the CLI entry point for polymarket-watch.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/victorihuoma/polymarket-watch/internal/discovery"
	"github.com/victorihuoma/polymarket-watch/internal/models"
	"github.com/victorihuoma/polymarket-watch/internal/output"
	"github.com/victorihuoma/polymarket-watch/internal/scanner"
)

// Version information set at build time.
var (
	version = "0.1.0"
	commit  = "dev"
	verbose = false
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// rootCmd creates the root command for polymarket-watch.
func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "polymarket-watch",
		Short: "Detect automated trading bots on Polymarket",
		Long: `polymarket-watch is a CLI tool that scans Polymarket wallets
for automated/bot trading patterns by analyzing trading behavior,
timing patterns, and arbitrage strategies.`,
		Version: fmt.Sprintf("%s (%s)", version, commit),
	}

	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")

	cmd.AddCommand(scanCmd())
	cmd.AddCommand(batchCmd())
	cmd.AddCommand(monitorCmd())
	cmd.AddCommand(discoverCmd())

	return cmd
}

// scanCmd creates the scan subcommand.
func scanCmd() *cobra.Command {
	var (
		wallet     string
		jsonOutput bool
		outputFile string
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan a single wallet for bot activity",
		Long:  `Scan a wallet address and generate a bot probability report.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wallet == "" {
				return fmt.Errorf("wallet address is required")
			}

			// Set up context with signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Fprintln(os.Stderr, "\nInterrupted, cancelling scan...")
				cancel()
			}()

			// Create scanner and run scan
			var opts []scanner.ScannerOption
			if verbose {
				opts = append(opts, scanner.WithVerbose(true))
			}
			if limit > 0 {
				opts = append(opts, scanner.WithTradeLimit(limit))
			}
			s := scanner.NewScanner(opts...)
			report, err := s.Scan(ctx, wallet)
			if err != nil {
				return fmt.Errorf("scanning wallet: %w", err)
			}

			// Output results
			return outputReport(report, jsonOutput, outputFile)
		},
	}

	cmd.Flags().StringVarP(&wallet, "wallet", "w", "", "Wallet address to scan (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write output to file")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "Limit number of trades to fetch (0 = all)")
	cmd.MarkFlagRequired("wallet")

	return cmd
}

// batchCmd creates the batch subcommand for scanning multiple wallets.
func batchCmd() *cobra.Command {
	var (
		file        string
		concurrency int
		jsonOutput  bool
		outputFile  string
	)

	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Scan multiple wallets from a file",
		Long: `Scan multiple wallet addresses from a file with concurrent processing.
Each line in the file should contain a single wallet address.
Empty lines and lines starting with # are ignored.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("file path is required")
			}

			// Read wallets from file
			wallets, err := readWalletsFromFile(file)
			if err != nil {
				return fmt.Errorf("reading wallets file: %w", err)
			}

			if len(wallets) == 0 {
				return fmt.Errorf("no wallets found in file")
			}

			// Set up context with signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Fprintln(os.Stderr, "\nInterrupted, cancelling scans...")
				cancel()
			}()

			// Scan wallets concurrently
			reports, errs := scanWalletsConcurrently(ctx, wallets, concurrency)

			// Report any errors
			if len(errs) > 0 {
				fmt.Fprintf(os.Stderr, "\nEncountered %d errors:\n", len(errs))
				for _, e := range errs {
					fmt.Fprintf(os.Stderr, "  - %v\n", e)
				}
			}

			if len(reports) == 0 {
				return fmt.Errorf("no wallets scanned successfully")
			}

			// Output results
			return outputReports(reports, jsonOutput, outputFile)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "File containing wallet addresses (one per line)")
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 3, "Number of concurrent scans")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write output to file")
	cmd.MarkFlagRequired("file")

	return cmd
}

// readWalletsFromFile reads wallet addresses from a file.
// Each line should contain a single wallet address.
// Empty lines and lines starting with # are ignored.
func readWalletsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wallets []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		wallets = append(wallets, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return wallets, nil
}

// scanResult holds the result of a single wallet scan.
type scanResult struct {
	report *models.ScanReport
	err    error
}

// scanWalletsConcurrently scans multiple wallets with bounded concurrency.
func scanWalletsConcurrently(ctx context.Context, wallets []string, concurrency int) ([]*models.ScanReport, []error) {
	if concurrency < 1 {
		concurrency = 1
	}

	resultsChan := make(chan scanResult, len(wallets))
	semaphore := make(chan struct{}, concurrency)

	var wg sync.WaitGroup

	for _, wallet := range wallets {
		wg.Add(1)
		go func(w string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				resultsChan <- scanResult{err: fmt.Errorf("wallet %s: %w", w, ctx.Err())}
				return
			}

			// Create scanner and run scan
			s := scanner.NewScanner()
			report, err := s.Scan(ctx, w)
			if err != nil {
				resultsChan <- scanResult{err: fmt.Errorf("wallet %s: %w", w, err)}
				return
			}

			resultsChan <- scanResult{report: report}
		}(wallet)
	}

	// Wait for all scans to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var reports []*models.ScanReport
	var errs []error

	for result := range resultsChan {
		if result.err != nil {
			errs = append(errs, result.err)
		} else {
			reports = append(reports, result.report)
		}
	}

	return reports, errs
}

// outputReport outputs a single scan report.
func outputReport(report *models.ScanReport, jsonOutput bool, outputFile string) error {
	if outputFile != "" {
		if jsonOutput {
			return output.WriteReportToFile(report, outputFile)
		}
		// For terminal output to file, create file writer
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer file.Close()
		formatter := output.NewTerminalOutput(output.WithWriter(file), output.WithColor(false))
		return formatter.Print(report)
	}

	var formatter output.OutputFormatter
	if jsonOutput {
		formatter = output.NewJSONOutput()
	} else {
		formatter = output.NewTerminalOutput()
	}

	return formatter.Print(report)
}

// outputReports outputs multiple scan reports.
func outputReports(reports []*models.ScanReport, jsonOutput bool, outputFile string) error {
	if outputFile != "" {
		if jsonOutput {
			return output.WriteSummaryToFile(reports, outputFile)
		}
		// For terminal output to file, create file writer
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer file.Close()
		formatter := output.NewTerminalOutput(output.WithWriter(file), output.WithColor(false))
		return formatter.PrintSummary(reports)
	}

	var formatter output.OutputFormatter
	if jsonOutput {
		formatter = output.NewJSONOutput()
	} else {
		formatter = output.NewTerminalOutput()
	}

	return formatter.PrintSummary(reports)
}

// MonitorEvent represents a monitoring alert event.
type MonitorEvent struct {
	// Wallet is the wallet address that triggered the alert.
	Wallet string `json:"wallet"`
	// TradeCount is the number of new trades detected.
	TradeCount int `json:"trade_count"`
	// BotScore is the computed bot probability score (0-100).
	BotScore int `json:"bot_score"`
	// Severity is the severity level (low, medium, high).
	Severity string `json:"severity"`
	// Timestamp is when the alert was generated.
	Timestamp time.Time `json:"timestamp"`
	// AlertReason describes why the alert was triggered.
	AlertReason string `json:"alert_reason"`
}

// monitorState tracks the monitoring state for multiple wallets.
type monitorState struct {
	wallets    []string
	lastChecks map[string]time.Time
	mu         sync.RWMutex
}

// newMonitorState creates a new monitoring state for the given wallets.
func newMonitorState(wallets []string) *monitorState {
	return &monitorState{
		wallets:    wallets,
		lastChecks: make(map[string]time.Time),
	}
}

// getLastCheck returns the last check time for a wallet.
func (m *monitorState) getLastCheck(wallet string) time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastChecks[wallet]
}

// setLastCheck updates the last check time for a wallet.
func (m *monitorState) setLastCheck(wallet string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastChecks[wallet] = t
}

// getWallets returns the list of wallets being monitored.
func (m *monitorState) getWallets() []string {
	return m.wallets
}

// filterNewTrades returns only trades that occurred after the given time.
func filterNewTrades(trades models.TradeList, since time.Time) models.TradeList {
	if since.IsZero() {
		return trades
	}

	var newTrades models.TradeList
	for _, trade := range trades {
		if trade.MatchTime.After(since) {
			newTrades = append(newTrades, trade)
		}
	}
	return newTrades
}

// shouldAlert returns true if the score meets the alert threshold.
func shouldAlert(score, threshold int) bool {
	return score >= threshold
}

// formatMonitorAlert formats a monitoring event as a human-readable string.
func formatMonitorAlert(event MonitorEvent) string {
	wallet := truncateWalletAddr(event.Wallet)
	severityLabel := strings.ToUpper(event.Severity)

	return fmt.Sprintf("[%s] ALERT (%s): %s - Bot Score: %d, New Trades: %d - %s",
		event.Timestamp.Format("15:04:05"),
		severityLabel,
		wallet,
		event.BotScore,
		event.TradeCount,
		event.AlertReason,
	)
}

// truncateWalletAddr truncates a wallet address to a short form.
func truncateWalletAddr(wallet string) string {
	if len(wallet) <= 12 {
		return wallet
	}
	return wallet[:6] + "..." + wallet[len(wallet)-4:]
}

// monitorCmd creates the monitor subcommand for continuous monitoring.
func monitorCmd() *cobra.Command {
	var (
		wallet     string
		file       string
		interval   time.Duration
		threshold  int
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Continuously monitor wallets for suspicious activity",
		Long: `Monitor one or more wallet addresses continuously for new trading activity.
When suspicious patterns are detected (bot score above threshold), an alert is generated.

Use --wallet for a single wallet or --file for multiple wallets.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate input
			if wallet == "" && file == "" {
				return fmt.Errorf("either --wallet or --file must be provided")
			}

			// Collect wallets to monitor
			var wallets []string
			if wallet != "" {
				wallets = append(wallets, wallet)
			}

			if file != "" {
				fileWallets, err := readWalletsFromFile(file)
				if err != nil {
					return fmt.Errorf("reading wallets file: %w", err)
				}
				wallets = append(wallets, fileWallets...)
			}

			if len(wallets) == 0 {
				return fmt.Errorf("no wallets to monitor")
			}

			// Set up context with signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Fprintln(os.Stderr, "\nStopping monitor...")
				cancel()
			}()

			// Initialize monitoring state
			state := newMonitorState(wallets)

			fmt.Printf("Starting monitor for %d wallet(s), checking every %v\n", len(wallets), interval)
			fmt.Printf("Alert threshold: %d\n", threshold)
			fmt.Println("Press Ctrl+C to stop.")
			fmt.Println()

			// Run monitoring loop
			return runMonitorLoop(ctx, state, interval, threshold, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&wallet, "wallet", "w", "", "Single wallet address to monitor")
	cmd.Flags().StringVarP(&file, "file", "f", "", "File containing wallet addresses to monitor")
	cmd.Flags().DurationVarP(&interval, "interval", "i", 30*time.Second, "Check interval (e.g., 30s, 1m, 5m)")
	cmd.Flags().IntVarP(&threshold, "threshold", "t", 80, "Bot score threshold for alerts (0-100)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output alerts as JSON")

	return cmd
}

// runMonitorLoop runs the main monitoring loop.
func runMonitorLoop(ctx context.Context, state *monitorState, interval time.Duration, threshold int, jsonOutput bool) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run initial check immediately
	checkWallets(ctx, state, threshold, jsonOutput)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Monitor stopped.")
			return nil
		case <-ticker.C:
			checkWallets(ctx, state, threshold, jsonOutput)
		}
	}
}

// checkWallets checks all wallets for new suspicious activity.
func checkWallets(ctx context.Context, state *monitorState, threshold int, jsonOutput bool) {
	wallets := state.getWallets()

	for _, wallet := range wallets {
		select {
		case <-ctx.Done():
			return
		default:
		}

		event, err := checkWallet(ctx, wallet, state, threshold)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking wallet %s: %v\n", truncateWalletAddr(wallet), err)
			continue
		}

		// Update last check time
		state.setLastCheck(wallet, time.Now())

		// Output alert if triggered
		if event != nil {
			outputAlert(*event, jsonOutput)
		}
	}
}

// checkWallet checks a single wallet for suspicious activity.
func checkWallet(ctx context.Context, wallet string, state *monitorState, threshold int) (*MonitorEvent, error) {
	s := scanner.NewScanner()

	// Fetch latest trades
	report, err := s.Scan(ctx, wallet)
	if err != nil {
		return nil, err
	}

	// Get new trades since last check
	lastCheck := state.getLastCheck(wallet)

	// Count recent trades (simplified: we use total trades if first check)
	tradeCount := report.Stats.TotalTrades
	if !lastCheck.IsZero() {
		// For subsequent checks, estimate new trades based on time window
		// This is a simplification - in a real implementation we'd track seen trade IDs
		tradeCount = estimateNewTrades(report.Stats, lastCheck)
	}

	// Check if we should alert
	if shouldAlert(report.BotScore, threshold) && tradeCount > 0 {
		return &MonitorEvent{
			Wallet:      wallet,
			TradeCount:  tradeCount,
			BotScore:    report.BotScore,
			Severity:    report.Severity(),
			Timestamp:   time.Now(),
			AlertReason: fmt.Sprintf("High bot score detected (%d >= %d threshold)", report.BotScore, threshold),
		}, nil
	}

	return nil, nil
}

// estimateNewTrades estimates the number of new trades since last check.
func estimateNewTrades(stats models.ScanStats, lastCheck time.Time) int {
	if stats.LastTradeTime.IsZero() || stats.FirstTradeTime.IsZero() {
		return 0
	}

	// If all trades are before last check, no new trades
	if stats.LastTradeTime.Before(lastCheck) {
		return 0
	}

	// If all trades are after last check, all are new
	if stats.FirstTradeTime.After(lastCheck) {
		return stats.TotalTrades
	}

	// Estimate based on trades per day rate
	tradesPerDay := stats.TradesPerDay()
	if tradesPerDay <= 0 {
		return 0
	}

	duration := time.Since(lastCheck)
	estimated := int(tradesPerDay * duration.Hours() / 24)

	// Cap at total trades
	if estimated > stats.TotalTrades {
		estimated = stats.TotalTrades
	}

	return estimated
}

// outputAlert outputs a monitoring alert.
func outputAlert(event MonitorEvent, jsonOutput bool) {
	if jsonOutput {
		data, err := json.Marshal(event)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding alert: %v\n", err)
			return
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(formatMonitorAlert(event))
	}
}

// discoverCmd creates the discover subcommand for wallet discovery.
func discoverCmd() *cobra.Command {
	var (
		marketIDs     []string
		marketSlugs   []string
		autoDiscover  bool
		topN          int
		sortBy        string
		minVolume     float64
		minVolume24hr float64
		minLiquidity  float64
		holdersLimit  int
		concurrency   int
		noScan        bool
		botThreshold  int
		tradeLimit    int
		jsonOutput    bool
		outputFile    string
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover suspicious wallets from top market holders",
		Long: `Discover potential bot wallets by scanning holders of markets.

Markets can be specified explicitly by ID or slug, or discovered automatically
by selecting the top markets by volume or liquidity.

Examples:
  # Auto-discover top 5 markets by volume and scan their holders
  polymarket-watch discover --auto --top 5

  # Scan holders of specific markets by slug
  polymarket-watch discover --slug "will-trump-win-2024" --slug "bitcoin-100k"

  # Scan holders of specific markets by condition ID
  polymarket-watch discover --market "0x1234..." --market "0x5678..."

  # Auto-discover with filters
  polymarket-watch discover --auto --top 10 --sort liquidity --min-volume 100000

  # Skip wallet scanning (only list holders)
  polymarket-watch discover --auto --top 3 --no-scan`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate that at least one input source is provided
			if len(marketIDs) == 0 && len(marketSlugs) == 0 && !autoDiscover {
				return fmt.Errorf("must specify --market, --slug, or --auto")
			}

			// If auto-discover is enabled, topN must be specified
			if autoDiscover && topN <= 0 {
				return fmt.Errorf("--top must be > 0 when --auto is enabled")
			}

			// Build discovery options
			opts := discovery.DiscoveryOptions{
				MarketIDs:     marketIDs,
				MarketSlugs:   marketSlugs,
				AutoDiscover:  autoDiscover,
				TopN:          topN,
				SortBy:        sortBy,
				MinVolume:     minVolume,
				MinVolume24hr: minVolume24hr,
				MinLiquidity:  minLiquidity,
				HoldersLimit:  holdersLimit,
				Concurrency:   concurrency,
				NoScan:        noScan,
				BotThreshold:  botThreshold,
				TradeLimit:    tradeLimit,
			}

			// Set up context with signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Fprintln(os.Stderr, "\nInterrupted, cancelling discovery...")
				cancel()
			}()

			// Create discoverer and run discovery
			d := discovery.NewDiscoverer()
			result, err := d.Discover(ctx, opts)
			if err != nil {
				return fmt.Errorf("discovery failed: %w", err)
			}

			// Output results
			return outputDiscoveryResult(result, jsonOutput, outputFile)
		},
	}

	// Input source flags
	cmd.Flags().StringSliceVarP(&marketIDs, "market", "m", nil, "Market condition ID(s) to scan (can be specified multiple times)")
	cmd.Flags().StringSliceVarP(&marketSlugs, "slug", "s", nil, "Market slug(s) to scan (can be specified multiple times)")
	cmd.Flags().BoolVar(&autoDiscover, "auto", false, "Auto-discover top markets")
	cmd.Flags().IntVar(&topN, "top", 0, "Number of top markets to scan (requires --auto)")

	// Auto-discover filter flags
	cmd.Flags().StringVar(&sortBy, "sort", "volume", "Sort markets by: volume, liquidity, volume24hr")
	cmd.Flags().Float64Var(&minVolume, "min-volume", 0, "Minimum total volume for auto-discovered markets")
	cmd.Flags().Float64Var(&minVolume24hr, "min-volume-24hr", 0, "Minimum 24-hour volume for auto-discovered markets")
	cmd.Flags().Float64Var(&minLiquidity, "min-liquidity", 0, "Minimum liquidity for auto-discovered markets")

	// Scanning options
	cmd.Flags().IntVar(&holdersLimit, "limit", 20, "Maximum number of holders to fetch per market (max 20)")
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 3, "Number of concurrent wallet scans")
	cmd.Flags().BoolVar(&noScan, "no-scan", false, "Skip wallet scanning (only list holders)")
	cmd.Flags().IntVarP(&botThreshold, "threshold", "t", 80, "Bot score threshold for detection (0-100)")
	cmd.Flags().IntVar(&tradeLimit, "trade-limit", 100, "Limit trades fetched per wallet for faster scanning (0 = all)")

	// Output flags
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write output to file")

	return cmd
}

// outputDiscoveryResult outputs discovery results.
func outputDiscoveryResult(result *discovery.DiscoveryResult, jsonOutput bool, outputFile string) error {
	if outputFile != "" {
		if jsonOutput {
			return output.WriteDiscoveryToFile(result, outputFile)
		}
		// For terminal output to file, create file writer
		file, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer file.Close()
		formatter := output.NewTerminalOutput(output.WithWriter(file), output.WithColor(false))
		return formatter.PrintDiscoveryResult(result)
	}

	if jsonOutput {
		jo := output.NewJSONOutput()
		return jo.PrintDiscoveryResult(result)
	}

	formatter := output.NewTerminalOutput()
	return formatter.PrintDiscoveryResult(result)
}
