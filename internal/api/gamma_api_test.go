package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewGammaAPI(t *testing.T) {
	t.Run("creates with default client", func(t *testing.T) {
		api := NewGammaAPI()
		if api == nil {
			t.Fatal("expected non-nil GammaAPI")
		}
		if api.baseURL != GammaAPIBaseURL {
			t.Errorf("expected baseURL %s, got %s", GammaAPIBaseURL, api.baseURL)
		}
	})

	t.Run("creates with custom client", func(t *testing.T) {
		client := NewClient(WithTimeout(10 * time.Second))
		api := NewGammaAPI(WithGammaClient(client))
		if api == nil {
			t.Fatal("expected non-nil GammaAPI")
		}
	})

	t.Run("creates with custom base URL", func(t *testing.T) {
		customURL := "https://custom-gamma.example.com"
		api := NewGammaAPI(WithGammaBaseURL(customURL))
		if api.baseURL != customURL {
			t.Errorf("expected baseURL %s, got %s", customURL, api.baseURL)
		}
	})
}

func TestGammaAPI_GetMarket(t *testing.T) {
	t.Run("fetches market by condition ID", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request path
			if r.URL.Path != "/markets/cond123" {
				t.Errorf("expected path /markets/cond123, got %s", r.URL.Path)
			}

			// Return mock market
			market := map[string]interface{}{
				"id":                 "market123",
				"question":           "Will Bitcoin reach $100k by end of 2024?",
				"conditionId":        "cond123",
				"slug":               "bitcoin-100k-2024",
				"resolutionSource":   "https://coinmarketcap.com",
				"endDate":            "2024-12-31T23:59:59Z",
				"liquidity":          "50000.0",
				"volume":             "1000000.0",
				"volume24hr":         "25000.0",
				"active":             true,
				"closed":             false,
				"marketMakerAddress": "0xmaker123",
				"outcomePrices":      "[\"0.65\",\"0.35\"]",
				"outcomes":           "[\"Yes\",\"No\"]",
				"clobTokenIds":       "[\"token1\",\"token2\"]",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(market)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		market, err := api.GetMarket(context.Background(), "cond123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if market.ID != "market123" {
			t.Errorf("expected market ID 'market123', got %s", market.ID)
		}
		if market.Question != "Will Bitcoin reach $100k by end of 2024?" {
			t.Errorf("unexpected market question: %s", market.Question)
		}
		if market.ConditionID != "cond123" {
			t.Errorf("expected conditionId 'cond123', got %s", market.ConditionID)
		}
		if !market.Active {
			t.Error("expected market to be active")
		}
	})

	t.Run("handles market not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "market not found"}`))
		}))
		defer server.Close()

		api := NewGammaAPI(
			WithGammaBaseURL(server.URL),
			WithGammaClient(NewClient(WithRetries(0))),
		)
		_, err := api.GetMarket(context.Background(), "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		api := NewGammaAPI(
			WithGammaBaseURL(server.URL),
			WithGammaClient(NewClient(WithRetries(0))),
		)
		_, err := api.GetMarket(context.Background(), "cond123")
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

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		_, err := api.GetMarket(context.Background(), "cond123")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{})
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := api.GetMarket(ctx, "cond123")
		if err == nil {
			t.Fatal("expected error due to context cancellation")
		}
	})
}

func TestGammaAPI_GetMarkets(t *testing.T) {
	t.Run("fetches all markets", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/markets" {
				t.Errorf("expected path /markets, got %s", r.URL.Path)
			}

			markets := []map[string]interface{}{
				{
					"id":          "market1",
					"question":    "Market 1?",
					"conditionId": "cond1",
					"active":      true,
				},
				{
					"id":          "market2",
					"question":    "Market 2?",
					"conditionId": "cond2",
					"active":      false,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		markets, err := api.GetMarkets(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(markets) != 2 {
			t.Fatalf("expected 2 markets, got %d", len(markets))
		}
		if markets[0].ID != "market1" {
			t.Errorf("expected first market ID 'market1', got %s", markets[0].ID)
		}
		if markets[1].ID != "market2" {
			t.Errorf("expected second market ID 'market2', got %s", markets[1].ID)
		}
	})

	t.Run("fetches markets with query options", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			if query.Get("active") != "true" {
				t.Errorf("expected active=true, got %s", query.Get("active"))
			}
			if query.Get("closed") != "false" {
				t.Errorf("expected closed=false, got %s", query.Get("closed"))
			}
			if query.Get("limit") != "50" {
				t.Errorf("expected limit=50, got %s", query.Get("limit"))
			}
			if query.Get("offset") != "100" {
				t.Errorf("expected offset=100, got %s", query.Get("offset"))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := MarketQueryOptions{
			Active: BoolPtr(true),
			Closed: BoolPtr(false),
			Limit:  50,
			Offset: 100,
		}
		_, err := api.GetMarketsWithOptions(context.Background(), opts)
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

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		markets, err := api.GetMarkets(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(markets) != 0 {
			t.Errorf("expected 0 markets, got %d", len(markets))
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		api := NewGammaAPI(
			WithGammaBaseURL(server.URL),
			WithGammaClient(NewClient(WithRetries(0))),
		)
		_, err := api.GetMarkets(context.Background())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestMarketQueryOptions(t *testing.T) {
	t.Run("converts to params correctly", func(t *testing.T) {
		opts := MarketQueryOptions{
			Active: BoolPtr(true),
			Closed: BoolPtr(false),
			Limit:  100,
			Offset: 50,
		}
		params := opts.toParams()

		if params["active"] != "true" {
			t.Errorf("expected active 'true', got %s", params["active"])
		}
		if params["closed"] != "false" {
			t.Errorf("expected closed 'false', got %s", params["closed"])
		}
		if params["limit"] != "100" {
			t.Errorf("expected limit '100', got %s", params["limit"])
		}
		if params["offset"] != "50" {
			t.Errorf("expected offset '50', got %s", params["offset"])
		}
	})

	t.Run("omits nil/zero values", func(t *testing.T) {
		opts := MarketQueryOptions{
			Limit: 50,
		}
		params := opts.toParams()

		if params["limit"] != "50" {
			t.Errorf("expected limit '50', got %s", params["limit"])
		}
		if _, exists := params["active"]; exists {
			t.Error("expected active to be omitted")
		}
		if _, exists := params["closed"]; exists {
			t.Error("expected closed to be omitted")
		}
		if _, exists := params["offset"]; exists {
			t.Error("expected offset to be omitted")
		}
	})
}

func TestGammaAPIInterface(t *testing.T) {
	// This test ensures the GammaAPI type can be used as expected
	var _ interface {
		GetMarket(context.Context, string) (*Market, error)
		GetMarkets(context.Context) (MarketList, error)
		GetMarketsWithOptions(context.Context, MarketQueryOptions) (MarketList, error)
	} = &GammaAPI{}
}

// BoolPtr is a helper function to create a pointer to a bool.
func BoolPtr(b bool) *bool {
	return &b
}

func TestTopMarketsOptions(t *testing.T) {
	t.Run("converts to params with all fields", func(t *testing.T) {
		opts := TopMarketsOptions{
			SortBy:        "volume",
			MinVolume:     10000.0,
			MinVolume24hr: 1000.0,
			MinLiquidity:  5000.0,
			Limit:         20,
		}
		params := opts.toParams()

		if params["sort_by"] != "volume" {
			t.Errorf("expected sort_by 'volume', got %s", params["sort_by"])
		}
		if params["min_volume"] != "10000" {
			t.Errorf("expected min_volume '10000', got %s", params["min_volume"])
		}
		if params["min_volume_24hr"] != "1000" {
			t.Errorf("expected min_volume_24hr '1000', got %s", params["min_volume_24hr"])
		}
		if params["min_liquidity"] != "5000" {
			t.Errorf("expected min_liquidity '5000', got %s", params["min_liquidity"])
		}
		if params["limit"] != "20" {
			t.Errorf("expected limit '20', got %s", params["limit"])
		}
	})

	t.Run("sorts by liquidity", func(t *testing.T) {
		opts := TopMarketsOptions{
			SortBy: "liquidity",
		}
		params := opts.toParams()

		if params["sort_by"] != "liquidity" {
			t.Errorf("expected sort_by 'liquidity', got %s", params["sort_by"])
		}
	})

	t.Run("sorts by volume24hr", func(t *testing.T) {
		opts := TopMarketsOptions{
			SortBy: "volume24hr",
		}
		params := opts.toParams()

		if params["sort_by"] != "volume24hr" {
			t.Errorf("expected sort_by 'volume24hr', got %s", params["sort_by"])
		}
	})

	t.Run("omits zero values", func(t *testing.T) {
		opts := TopMarketsOptions{
			SortBy: "volume",
			Limit:  10,
		}
		params := opts.toParams()

		if params["sort_by"] != "volume" {
			t.Errorf("expected sort_by 'volume', got %s", params["sort_by"])
		}
		if params["limit"] != "10" {
			t.Errorf("expected limit '10', got %s", params["limit"])
		}
		if _, exists := params["min_volume"]; exists {
			t.Error("expected min_volume to be omitted")
		}
		if _, exists := params["min_volume_24hr"]; exists {
			t.Error("expected min_volume_24hr to be omitted")
		}
		if _, exists := params["min_liquidity"]; exists {
			t.Error("expected min_liquidity to be omitted")
		}
	})

	t.Run("omits empty sort_by", func(t *testing.T) {
		opts := TopMarketsOptions{
			MinVolume: 5000.0,
		}
		params := opts.toParams()

		if _, exists := params["sort_by"]; exists {
			t.Error("expected sort_by to be omitted")
		}
		if params["min_volume"] != "5000" {
			t.Errorf("expected min_volume '5000', got %s", params["min_volume"])
		}
	})

	t.Run("handles decimal min values", func(t *testing.T) {
		opts := TopMarketsOptions{
			MinVolume:     1234.56,
			MinVolume24hr: 100.5,
			MinLiquidity:  999.99,
		}
		params := opts.toParams()

		if params["min_volume"] != "1234.56" {
			t.Errorf("expected min_volume '1234.56', got %s", params["min_volume"])
		}
		if params["min_volume_24hr"] != "100.5" {
			t.Errorf("expected min_volume_24hr '100.5', got %s", params["min_volume_24hr"])
		}
		if params["min_liquidity"] != "999.99" {
			t.Errorf("expected min_liquidity '999.99', got %s", params["min_liquidity"])
		}
	})
}

func TestGammaAPI_GetTopMarkets(t *testing.T) {
	t.Run("fetches and sorts markets by volume descending", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/markets" {
				t.Errorf("expected path /markets, got %s", r.URL.Path)
			}

			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "volume24hr": "1000", "liquidity": "5000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "volume24hr": "2000", "liquidity": "10000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "volume24hr": "1500", "liquidity": "7500", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			SortBy: "volume",
			Limit:  10,
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(markets) != 3 {
			t.Fatalf("expected 3 markets, got %d", len(markets))
		}
		// Should be sorted by volume descending
		if markets[0].ID != "market2" {
			t.Errorf("expected first market to be market2 (highest volume), got %s", markets[0].ID)
		}
		if markets[1].ID != "market3" {
			t.Errorf("expected second market to be market3, got %s", markets[1].ID)
		}
		if markets[2].ID != "market1" {
			t.Errorf("expected third market to be market1 (lowest volume), got %s", markets[2].ID)
		}
	})

	t.Run("sorts markets by liquidity descending", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "100000", "volume24hr": "1000", "liquidity": "5000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "50000", "volume24hr": "500", "liquidity": "15000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "volume24hr": "750", "liquidity": "10000", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			SortBy: "liquidity",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should be sorted by liquidity descending
		if markets[0].ID != "market2" {
			t.Errorf("expected first market to be market2 (highest liquidity), got %s", markets[0].ID)
		}
		if markets[1].ID != "market3" {
			t.Errorf("expected second market to be market3, got %s", markets[1].ID)
		}
		if markets[2].ID != "market1" {
			t.Errorf("expected third market to be market1 (lowest liquidity), got %s", markets[2].ID)
		}
	})

	t.Run("sorts markets by volume24hr descending", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "100000", "volume24hr": "500", "liquidity": "5000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "50000", "volume24hr": "2000", "liquidity": "15000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "volume24hr": "1000", "liquidity": "10000", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			SortBy: "volume24hr",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should be sorted by volume24hr descending
		if markets[0].ID != "market2" {
			t.Errorf("expected first market to be market2 (highest volume24hr), got %s", markets[0].ID)
		}
		if markets[1].ID != "market3" {
			t.Errorf("expected second market to be market3, got %s", markets[1].ID)
		}
		if markets[2].ID != "market1" {
			t.Errorf("expected third market to be market1 (lowest volume24hr), got %s", markets[2].ID)
		}
	})

	t.Run("filters by minimum volume", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "volume24hr": "1000", "liquidity": "5000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "volume24hr": "2000", "liquidity": "10000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "25000", "volume24hr": "500", "liquidity": "2500", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			MinVolume: 40000,
			SortBy:    "volume",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should only return markets with volume >= 40000
		if len(markets) != 2 {
			t.Fatalf("expected 2 markets after filtering, got %d", len(markets))
		}
		for _, m := range markets {
			if m.Volume < 40000 {
				t.Errorf("market %s has volume %f which is below threshold", m.ID, m.Volume)
			}
		}
	})

	t.Run("filters by minimum volume24hr", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "volume24hr": "1000", "liquidity": "5000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "volume24hr": "500", "liquidity": "10000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "volume24hr": "2000", "liquidity": "7500", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			MinVolume24hr: 800,
			SortBy:        "volume24hr",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should only return markets with volume24hr >= 800
		if len(markets) != 2 {
			t.Fatalf("expected 2 markets after filtering, got %d", len(markets))
		}
		for _, m := range markets {
			if m.Volume24hr < 800 {
				t.Errorf("market %s has volume24hr %f which is below threshold", m.ID, m.Volume24hr)
			}
		}
	})

	t.Run("filters by minimum liquidity", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "volume24hr": "1000", "liquidity": "3000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "volume24hr": "2000", "liquidity": "10000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "volume24hr": "1500", "liquidity": "7500", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			MinLiquidity: 5000,
			SortBy:       "liquidity",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should only return markets with liquidity >= 5000
		if len(markets) != 2 {
			t.Fatalf("expected 2 markets after filtering, got %d", len(markets))
		}
		for _, m := range markets {
			if m.Liquidity < 5000 {
				t.Errorf("market %s has liquidity %f which is below threshold", m.ID, m.Liquidity)
			}
		}
	})

	t.Run("limits results to specified count", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "active": true},
				{"id": "market4", "conditionId": "cond4", "volume": "90000", "active": true},
				{"id": "market5", "conditionId": "cond5", "volume": "60000", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			SortBy: "volume",
			Limit:  3,
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(markets) != 3 {
			t.Fatalf("expected 3 markets (limited), got %d", len(markets))
		}
		// Should have the top 3 by volume
		if markets[0].Volume != 100000 {
			t.Errorf("expected first market volume 100000, got %f", markets[0].Volume)
		}
		if markets[1].Volume != 90000 {
			t.Errorf("expected second market volume 90000, got %f", markets[1].Volume)
		}
		if markets[2].Volume != 75000 {
			t.Errorf("expected third market volume 75000, got %f", markets[2].Volume)
		}
	})

	t.Run("filters only active markets", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "active": true, "closed": false},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "active": false, "closed": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "active": true, "closed": false},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			SortBy: "volume",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should only return active, non-closed markets
		if len(markets) != 2 {
			t.Fatalf("expected 2 active markets, got %d", len(markets))
		}
		for _, m := range markets {
			if !m.Active || m.Closed {
				t.Errorf("market %s should be active and not closed", m.ID)
			}
		}
	})

	t.Run("handles empty result after filtering", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "1000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "2000", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			MinVolume: 100000, // Higher than any market
			SortBy:    "volume",
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(markets) != 0 {
			t.Errorf("expected 0 markets after filtering, got %d", len(markets))
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		api := NewGammaAPI(
			WithGammaBaseURL(server.URL),
			WithGammaClient(NewClient(WithRetries(0))),
		)
		opts := TopMarketsOptions{
			SortBy: "volume",
		}
		_, err := api.GetTopMarkets(context.Background(), opts)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		opts := TopMarketsOptions{
			SortBy: "volume",
		}
		_, err := api.GetTopMarkets(ctx, opts)
		if err == nil {
			t.Fatal("expected error due to context cancellation")
		}
	})

	t.Run("defaults to volume sort when sortBy is empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			markets := []map[string]interface{}{
				{"id": "market1", "conditionId": "cond1", "volume": "50000", "active": true},
				{"id": "market2", "conditionId": "cond2", "volume": "100000", "active": true},
				{"id": "market3", "conditionId": "cond3", "volume": "75000", "active": true},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		opts := TopMarketsOptions{
			// No SortBy specified - should default to volume
			Limit: 10,
		}
		markets, err := api.GetTopMarkets(context.Background(), opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should be sorted by volume descending (default)
		if markets[0].Volume != 100000 {
			t.Errorf("expected first market volume 100000 (sorted by volume), got %f", markets[0].Volume)
		}
	})
}

func TestGammaAPIInterface_WithTopMarkets(t *testing.T) {
	// This test ensures the GammaAPI type includes GetTopMarkets
	var _ interface {
		GetMarket(context.Context, string) (*Market, error)
		GetMarkets(context.Context) (MarketList, error)
		GetMarketsWithOptions(context.Context, MarketQueryOptions) (MarketList, error)
		GetTopMarkets(context.Context, TopMarketsOptions) (MarketList, error)
	} = &GammaAPI{}
}

func TestGammaAPI_GetMarketBySlug(t *testing.T) {
	t.Run("fetches market by slug", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request path and query parameter
			if r.URL.Path != "/markets" {
				t.Errorf("expected path /markets, got %s", r.URL.Path)
			}
			slug := r.URL.Query().Get("slug")
			if slug != "bitcoin-100k-2024" {
				t.Errorf("expected slug query param 'bitcoin-100k-2024', got %s", slug)
			}

			// Return mock market array (API returns array even for slug query)
			markets := []map[string]interface{}{
				{
					"id":                 "market123",
					"question":           "Will Bitcoin reach $100k by end of 2024?",
					"conditionId":        "cond123",
					"slug":               "bitcoin-100k-2024",
					"resolutionSource":   "https://coinmarketcap.com",
					"endDate":            "2024-12-31T23:59:59Z",
					"liquidity":          "50000.0",
					"volume":             "1000000.0",
					"volume24hr":         "25000.0",
					"active":             true,
					"closed":             false,
					"marketMakerAddress": "0xmaker123",
					"outcomePrices":      "[\"0.65\",\"0.35\"]",
					"outcomes":           "[\"Yes\",\"No\"]",
					"clobTokenIds":       "[\"token1\",\"token2\"]",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		market, err := api.GetMarketBySlug(context.Background(), "bitcoin-100k-2024")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if market == nil {
			t.Fatal("expected non-nil market")
		}
		if market.ID != "market123" {
			t.Errorf("expected market ID 'market123', got %s", market.ID)
		}
		if market.Slug != "bitcoin-100k-2024" {
			t.Errorf("expected slug 'bitcoin-100k-2024', got %s", market.Slug)
		}
		if market.Question != "Will Bitcoin reach $100k by end of 2024?" {
			t.Errorf("unexpected market question: %s", market.Question)
		}
		if market.ConditionID != "cond123" {
			t.Errorf("expected conditionId 'cond123', got %s", market.ConditionID)
		}
		if !market.Active {
			t.Error("expected market to be active")
		}
		if market.Volume != 1000000.0 {
			t.Errorf("expected volume 1000000.0, got %f", market.Volume)
		}
	})

	t.Run("returns error when market not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return empty array when slug doesn't match
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		_, err := api.GetMarketBySlug(context.Background(), "nonexistent-market")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Verify error message mentions slug
		if err.Error() != "market not found for slug: nonexistent-market" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})

	t.Run("returns first match when multiple results", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return multiple markets (edge case)
			markets := []map[string]interface{}{
				{
					"id":          "market1",
					"conditionId": "cond1",
					"slug":        "test-market",
					"active":      true,
				},
				{
					"id":          "market2",
					"conditionId": "cond2",
					"slug":        "test-market",
					"active":      true,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(markets)
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		market, err := api.GetMarketBySlug(context.Background(), "test-market")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should return first match
		if market.ID != "market1" {
			t.Errorf("expected first market ID 'market1', got %s", market.ID)
		}
	})

	t.Run("handles server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		api := NewGammaAPI(
			WithGammaBaseURL(server.URL),
			WithGammaClient(NewClient(WithRetries(0))),
		)
		_, err := api.GetMarketBySlug(context.Background(), "test-slug")
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

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		_, err := api.GetMarketBySlug(context.Background(), "test-slug")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		}))
		defer server.Close()

		api := NewGammaAPI(WithGammaBaseURL(server.URL))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := api.GetMarketBySlug(ctx, "test-slug")
		if err == nil {
			t.Fatal("expected error due to context cancellation")
		}
	})

	t.Run("handles empty slug", func(t *testing.T) {
		api := NewGammaAPI()
		_, err := api.GetMarketBySlug(context.Background(), "")
		if err == nil {
			t.Fatal("expected error for empty slug, got nil")
		}
		if err.Error() != "slug cannot be empty" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})
}

func TestGammaAPIInterface_WithGetMarketBySlug(t *testing.T) {
	// This test ensures the GammaAPI type includes GetMarketBySlug
	var _ interface {
		GetMarket(context.Context, string) (*Market, error)
		GetMarkets(context.Context) (MarketList, error)
		GetMarketsWithOptions(context.Context, MarketQueryOptions) (MarketList, error)
		GetTopMarkets(context.Context, TopMarketsOptions) (MarketList, error)
		GetMarketBySlug(context.Context, string) (*Market, error)
	} = &GammaAPI{}
}
