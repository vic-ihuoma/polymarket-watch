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
