# Polymarket Watch - Implementation Plan

## Overview
A Go CLI tool to scan Polymarket for automated/bot wallets by analyzing trading patterns.

## Target Example
**Account88888** (`0x7f69983eb28245bba0d5083502a78744a8f66162`)
- 11,599 trades, $541K+ profit, 99% win rate
- Strategy: Buys both UP and DOWN on BTC price windows during mispricing
- When UP=48¢ + DOWN=46¢ = 94¢, bot buys both, collects $1, keeps 6¢ profit

## Detection Algorithms

### 1. Arbitrage Detection
Detect simultaneous opposing positions. If `price_up + price_down < 1.0`, flag as arbitrage opportunity exploitation.

### 2. Timing Analysis
Calculate standard deviation of trade intervals:
- Human: σ > 60 seconds (high variance)
- Bot: σ < 5 seconds (consistent timing)

### 3. Win Rate Anomaly
Flag accounts with >95% win rate AND >100 trades (statistically improbable for random outcomes).

### 4. Position Sizing
Detect algorithmic patterns: round numbers, fixed percentages, consistent ratios.

### 5. Composite Score
```
bot_score = (arbitrage * 0.35) + (timing * 0.25) + (winrate * 0.25) + (sizing * 0.15)
```

## API Endpoints
- `GET https://data-api.polymarket.com/trades?user=0x...`
- `GET https://data-api.polymarket.com/positions?user=0x...`
- `GET https://gamma-api.polymarket.com/markets`

## CLI Commands
```bash
polymarket-watch scan --wallet <address>
polymarket-watch batch --file wallets.txt
polymarket-watch monitor --interval 30s
polymarket-watch discover --event-id <id>
```

## File Structure
```
cmd/polymarket-watch/main.go    # CLI entry point
internal/api/client.go          # HTTP client
internal/api/data_api.go        # Data API
internal/api/gamma_api.go       # Gamma API
internal/scanner/scanner.go     # Scanner orchestration
internal/scanner/arbitrage.go   # Arbitrage detection
internal/scanner/frequency.go   # Timing analysis
internal/scanner/winrate.go     # Win rate analysis
internal/scanner/sizing.go      # Position sizing
internal/models/*.go            # Data models
internal/output/terminal.go     # CLI output
internal/output/json.go         # JSON export
```
