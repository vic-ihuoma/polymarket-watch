package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/victorihuoma/polymarket-watch/internal/models"
)

func TestNewDataAPI(t *testing.T) {
	t.Run("creates with default client", func(t *testing.T) {
		api := NewDataAPI()
		if api == nil {
			t.Fatal("expected non-nil DataAPI")
		}
		if api.baseURL != DataAPIBaseURL {
			t.Errorf("expected baseURL %s, got %s", DataAPIBaseURL, api.baseURL)
		}
	})

	t.Run("creates with custom client", func(t *testing.T) {
		client := NewClient(WithTimeout(10 * time.Second))
		api := NewDataAPI(WithClient(client))
		if api == nil {
			t.Fatal("expected non-nil DataAPI")
		}
	})

	t.Run("creates with custom base URL", func(t *testing.T) {
		customURL := "https://custom-api.example.com"
		api := NewDataAPI(WithBaseURL(customURL))
		if api.baseURL != customURL {
			t.Errorf("expected baseURL %s, got %s", customURL, api.baseURL)
		}
	})
}

func TestDataAPI_GetTrades(t *testing.T) {
	t.Run("fetches trades for wallet", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request parameters
			if r.URL.Path != "/trades" {
				t.Errorf("expected path /trades, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("user") != "0x123" {
				t.Errorf("expected user query param 0x123, got %s", r.URL.Query().Get("user"))
			}

			// Return mock trades
			trades := []map[string]interface{}{
				{
					"id":             "trade1",
					"taker_order_id": "order1",
					"market":         "0xmarket1",
					"asset_id":       "asset1",
					"side":           "BUY",
					"size":           "100.5",
					"fee_rate_bps":   "20",
					"price":          "0.65",
					"status":         "MATCHED",
					"match_time":     "2024-01-15T10:30:00Z",
					"last_update":    "2024-01-15T10:30:00Z",
					"outcome":        "Yes",
					"bucket_index":   "0",
					"owner":          "0x123",
					"trader_side":    "TAKER",
					"type":           "MARKET",
				},
				{
					"id":             "trade2",
					"taker_order_id": "order2",
					"market":         "0xmarket1",
					"asset_id":       "asset2",
					"side":           "SELL",
					"size":           "50.0",
					"fee_rate_bps":   "20",
					"price":          "0.35",
					"status":         "MATCHED",
					"match_time":     "2024-01-15T10:35:00Z",
					"last_update":    "2024-01-15T10:35:00Z",
					"outcome":        "No",
					"bucket_index":   "1",
					"owner":          "0x123",
					"trader_side":    "TAKER",
					"type":           "LIMIT",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(trades)
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		trades, err := api.GetTrades(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(trades) != 2 {
			t.Fatalf("expected 2 trades, got %d", len(trades))
		}
		if trades[0].ID != "trade1" {
			t.Errorf("expected first trade ID 'trade1', got %s", trades[0].ID)
		}
		if trades[0].Side != "BUY" {
			t.Errorf("expected first trade side 'BUY', got %s", trades[0].Side)
		}
		if trades[0].Size != 100.5 {
			t.Errorf("expected first trade size 100.5, got %f", trades[0].Size)
		}
		if trades[1].ID != "trade2" {
			t.Errorf("expected second trade ID 'trade2', got %s", trades[1].ID)
		}
	})

	t.Run("fetches trades with options", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify all query params are passed
			query := r.URL.Query()
			if query.Get("user") != "0x123" {
				t.Errorf("expected user 0x123, got %s", query.Get("user"))
			}
			if query.Get("limit") != "50" {
				t.Errorf("expected limit 50, got %s", query.Get("limit"))
			}
			if query.Get("offset") != "100" {
				t.Errorf("expected offset 100, got %s", query.Get("offset"))
			}
			if query.Get("market") != "0xmarket1" {
				t.Errorf("expected market 0xmarket1, got %s", query.Get("market"))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		opts := TradeQueryOptions{
			Limit:  50,
			Offset: 100,
			Market: "0xmarket1",
		}
		_, err := api.GetTradesWithOptions(context.Background(), "0x123", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		trades, err := api.GetTrades(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(trades) != 0 {
			t.Errorf("expected 0 trades, got %d", len(trades))
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		api := NewDataAPI(
			WithBaseURL(server.URL),
			WithClient(NewClient(WithRetries(0))), // No retries to speed up test
		)
		_, err := api.GetTrades(context.Background(), "0x123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		_, err := api.GetTrades(context.Background(), "0x123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second) // Delay response
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := api.GetTrades(ctx, "0x123")
		if err == nil {
			t.Fatal("expected error due to context cancellation")
		}
	})
}

func TestDataAPI_GetAllTrades(t *testing.T) {
	t.Run("fetches all trades with pagination", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var trades []map[string]interface{}

			offset := r.URL.Query().Get("offset")
			switch offset {
			case "", "0":
				// First page - return 100 trades (max per page)
				for i := 0; i < 100; i++ {
					trades = append(trades, map[string]interface{}{
						"id":           "trade" + string(rune(i)),
						"market":       "0xmarket1",
						"side":         "BUY",
						"size":         "10.0",
						"price":        "0.5",
						"status":       "MATCHED",
						"match_time":   "2024-01-15T10:30:00Z",
						"last_update":  "2024-01-15T10:30:00Z",
						"outcome":      "Yes",
						"bucket_index": "0",
						"owner":        "0x123",
					})
				}
			case "100":
				// Second page - return 50 trades (less than max, signals end)
				for i := 0; i < 50; i++ {
					trades = append(trades, map[string]interface{}{
						"id":           "trade" + string(rune(100+i)),
						"market":       "0xmarket1",
						"side":         "BUY",
						"size":         "10.0",
						"price":        "0.5",
						"status":       "MATCHED",
						"match_time":   "2024-01-15T10:30:00Z",
						"last_update":  "2024-01-15T10:30:00Z",
						"outcome":      "Yes",
						"bucket_index": "0",
						"owner":        "0x123",
					})
				}
			default:
				// No more trades
				trades = []map[string]interface{}{}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(trades)
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		trades, err := api.GetAllTrades(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(trades) != 150 {
			t.Errorf("expected 150 trades, got %d", len(trades))
		}
		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestDataAPI_GetPositions(t *testing.T) {
	t.Run("fetches positions for wallet", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request parameters
			if r.URL.Path != "/positions" {
				t.Errorf("expected path /positions, got %s", r.URL.Path)
			}
			if r.URL.Query().Get("user") != "0x123" {
				t.Errorf("expected user query param 0x123, got %s", r.URL.Query().Get("user"))
			}

			// Return mock positions
			positions := []map[string]interface{}{
				{
					"proxyWallet":  "0x123",
					"asset":        "asset1",
					"conditionId":  "cond1",
					"outcome":      "Yes",
					"outcomeIndex": "0",
					"size":         "100.0",
					"avgPrice":     "0.45",
					"initialValue": "45.0",
					"currentValue": "50.0",
					"cashBalance":  "0",
					"pnl":          "5.0",
					"realizedPnl":  "0",
					"percentPnl":   "11.11",
					"curPrice":     "0.50",
					"redeemable":   false,
					"mergeable":    false,
				},
				{
					"proxyWallet":  "0x123",
					"asset":        "asset2",
					"conditionId":  "cond1",
					"outcome":      "No",
					"outcomeIndex": "1",
					"size":         "100.0",
					"avgPrice":     "0.50",
					"initialValue": "50.0",
					"currentValue": "48.0",
					"cashBalance":  "0",
					"pnl":          "-2.0",
					"realizedPnl":  "0",
					"percentPnl":   "-4.0",
					"curPrice":     "0.48",
					"redeemable":   false,
					"mergeable":    false,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(positions)
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		positions, err := api.GetPositions(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(positions) != 2 {
			t.Fatalf("expected 2 positions, got %d", len(positions))
		}
		if positions[0].Outcome != "Yes" {
			t.Errorf("expected first position outcome 'Yes', got %s", positions[0].Outcome)
		}
		if positions[0].Size != 100.0 {
			t.Errorf("expected first position size 100.0, got %f", positions[0].Size)
		}
		if positions[1].Outcome != "No" {
			t.Errorf("expected second position outcome 'No', got %s", positions[1].Outcome)
		}
	})

	t.Run("fetches positions with options", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify all query params are passed
			query := r.URL.Query()
			if query.Get("user") != "0x123" {
				t.Errorf("expected user 0x123, got %s", query.Get("user"))
			}
			if query.Get("limit") != "25" {
				t.Errorf("expected limit 25, got %s", query.Get("limit"))
			}
			if query.Get("offset") != "50" {
				t.Errorf("expected offset 50, got %s", query.Get("offset"))
			}
			if query.Get("sizeThreshold") != "10.5" {
				t.Errorf("expected sizeThreshold 10.5, got %s", query.Get("sizeThreshold"))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		opts := PositionQueryOptions{
			Limit:         25,
			Offset:        50,
			SizeThreshold: 10.5,
		}
		_, err := api.GetPositionsWithOptions(context.Background(), "0x123", opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("handles empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		positions, err := api.GetPositions(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(positions) != 0 {
			t.Errorf("expected 0 positions, got %d", len(positions))
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		api := NewDataAPI(
			WithBaseURL(server.URL),
			WithClient(NewClient(WithRetries(0))),
		)
		_, err := api.GetPositions(context.Background(), "0x123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDataAPI_GetAllPositions(t *testing.T) {
	t.Run("fetches all positions with pagination", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			var positions []map[string]interface{}

			offset := r.URL.Query().Get("offset")
			switch offset {
			case "", "0":
				// First page - return 100 positions (max per page)
				for i := 0; i < 100; i++ {
					positions = append(positions, map[string]interface{}{
						"proxyWallet":  "0x123",
						"asset":        "asset" + string(rune(i)),
						"conditionId":  "cond" + string(rune(i)),
						"outcome":      "Yes",
						"outcomeIndex": "0",
						"size":         "10.0",
						"avgPrice":     "0.5",
						"curPrice":     "0.5",
						"redeemable":   false,
						"mergeable":    false,
					})
				}
			case "100":
				// Second page - return 25 positions (less than max, signals end)
				for i := 0; i < 25; i++ {
					positions = append(positions, map[string]interface{}{
						"proxyWallet":  "0x123",
						"asset":        "asset" + string(rune(100+i)),
						"conditionId":  "cond" + string(rune(100+i)),
						"outcome":      "Yes",
						"outcomeIndex": "0",
						"size":         "10.0",
						"avgPrice":     "0.5",
						"curPrice":     "0.5",
						"redeemable":   false,
						"mergeable":    false,
					})
				}
			default:
				// No more positions
				positions = []map[string]interface{}{}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(positions)
		}))
		defer server.Close()

		api := NewDataAPI(WithBaseURL(server.URL))
		positions, err := api.GetAllPositions(context.Background(), "0x123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(positions) != 125 {
			t.Errorf("expected 125 positions, got %d", len(positions))
		}
		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestTradeQueryOptions(t *testing.T) {
	t.Run("converts to params correctly", func(t *testing.T) {
		opts := TradeQueryOptions{
			Limit:  100,
			Offset: 50,
			Market: "0xmarket1",
			Side:   "BUY",
		}
		params := opts.toParams()

		if params["limit"] != "100" {
			t.Errorf("expected limit '100', got %s", params["limit"])
		}
		if params["offset"] != "50" {
			t.Errorf("expected offset '50', got %s", params["offset"])
		}
		if params["market"] != "0xmarket1" {
			t.Errorf("expected market '0xmarket1', got %s", params["market"])
		}
		if params["side"] != "BUY" {
			t.Errorf("expected side 'BUY', got %s", params["side"])
		}
	})

	t.Run("omits zero values", func(t *testing.T) {
		opts := TradeQueryOptions{
			Limit: 100,
			// Offset, Market, Side are zero values
		}
		params := opts.toParams()

		if params["limit"] != "100" {
			t.Errorf("expected limit '100', got %s", params["limit"])
		}
		if _, exists := params["offset"]; exists {
			t.Error("expected offset to be omitted")
		}
		if _, exists := params["market"]; exists {
			t.Error("expected market to be omitted")
		}
		if _, exists := params["side"]; exists {
			t.Error("expected side to be omitted")
		}
	})
}

func TestPositionQueryOptions(t *testing.T) {
	t.Run("converts to params correctly", func(t *testing.T) {
		opts := PositionQueryOptions{
			Limit:         100,
			Offset:        25,
			SizeThreshold: 5.5,
		}
		params := opts.toParams()

		if params["limit"] != "100" {
			t.Errorf("expected limit '100', got %s", params["limit"])
		}
		if params["offset"] != "25" {
			t.Errorf("expected offset '25', got %s", params["offset"])
		}
		if params["sizeThreshold"] != "5.5" {
			t.Errorf("expected sizeThreshold '5.5', got %s", params["sizeThreshold"])
		}
	})

	t.Run("omits zero values", func(t *testing.T) {
		opts := PositionQueryOptions{
			Limit: 50,
		}
		params := opts.toParams()

		if params["limit"] != "50" {
			t.Errorf("expected limit '50', got %s", params["limit"])
		}
		if _, exists := params["offset"]; exists {
			t.Error("expected offset to be omitted")
		}
		if _, exists := params["sizeThreshold"]; exists {
			t.Error("expected sizeThreshold to be omitted")
		}
	})
}

// TestDataAPIInterface ensures DataAPI implements expected interface contract.
func TestDataAPIInterface(t *testing.T) {
	// This test ensures the DataAPI type can be used as expected
	var _ interface {
		GetTrades(context.Context, string) (models.TradeList, error)
		GetTradesWithOptions(context.Context, string, TradeQueryOptions) (models.TradeList, error)
		GetAllTrades(context.Context, string) (models.TradeList, error)
		GetPositions(context.Context, string) (models.PositionList, error)
		GetPositionsWithOptions(context.Context, string, PositionQueryOptions) (models.PositionList, error)
		GetAllPositions(context.Context, string) (models.PositionList, error)
	} = &DataAPI{}
}
