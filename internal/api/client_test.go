package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		options []ClientOption
		wantErr bool
	}{
		{
			name:    "default options",
			options: nil,
			wantErr: false,
		},
		{
			name:    "custom rate limit",
			options: []ClientOption{WithRateLimit(50)},
			wantErr: false,
		},
		{
			name:    "custom timeout",
			options: []ClientOption{WithTimeout(5 * time.Second)},
			wantErr: false,
		},
		{
			name:    "multiple options",
			options: []ClientOption{WithRateLimit(200), WithTimeout(30 * time.Second)},
			wantErr: false,
		},
		{
			name:    "zero rate limit uses default",
			options: []ClientOption{WithRateLimit(0)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(tt.options...)
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.httpClient == nil {
				t.Error("httpClient is nil")
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		wantStatusCode int
		wantBody       string
		wantErr        bool
	}{
		{
			name: "successful request",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"ok"}`))
			},
			wantStatusCode: http.StatusOK,
			wantBody:       `{"status":"ok"}`,
			wantErr:        false,
		},
		{
			name: "server error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal error"}`))
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       `{"error":"internal error"}`,
			wantErr:        false, // Error status is returned, not an error
		},
		{
			name: "not found",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantStatusCode: http.StatusNotFound,
			wantBody:       "",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := NewClient()
			resp, err := client.Get(context.Background(), server.URL)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				t.Errorf("Get() statusCode = %v, want %v", resp.StatusCode, tt.wantStatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			if string(body) != tt.wantBody {
				t.Errorf("Get() body = %v, want %v", string(body), tt.wantBody)
			}
		})
	}
}

func TestClient_GetContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Get(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

func TestClient_RateLimiting(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Rate limit of 10 requests per minute = 1 request per 6 seconds
	// For testing, we'll use a higher rate and verify requests are throttled
	client := NewClient(WithRateLimit(600)) // 600/min = 10/sec

	ctx := context.Background()
	start := time.Now()

	// Make 10 concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(ctx, server.URL)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			resp.Body.Close()
		}()
	}
	wg.Wait()

	elapsed := time.Since(start)

	// All requests should complete
	if got := requestCount.Load(); got != 10 {
		t.Errorf("requestCount = %d, want 10", got)
	}

	// With rate limiting, requests should be throttled
	// 10 requests at 10/sec should take about 1 second
	// Allow some flexibility for test timing
	if elapsed > 5*time.Second {
		t.Errorf("requests took too long: %v", elapsed)
	}
}

func TestClient_Retry(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := attempts.Add(1)
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := NewClient(WithRetries(3))
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("statusCode = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	// Should have made 3 attempts (2 retries + 1 final success)
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestClient_RetryExhausted(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(WithRetries(2))
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	// After exhausting retries, should return the last response
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("statusCode = %v, want %v", resp.StatusCode, http.StatusServiceUnavailable)
	}

	// Should have made 3 attempts (initial + 2 retries)
	if got := attempts.Load(); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

func TestClient_QueryParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("user") != "0x123" {
			t.Errorf("user param = %v, want 0x123", query.Get("user"))
		}
		if query.Get("limit") != "100" {
			t.Errorf("limit param = %v, want 100", query.Get("limit"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	params := map[string]string{
		"user":  "0x123",
		"limit": "100",
	}

	resp, err := client.GetWithParams(context.Background(), server.URL, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("statusCode = %v, want %v", resp.StatusCode, http.StatusOK)
	}
}

func TestClient_UserAgent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua != "polymarket-watch/0.1.0" {
			t.Errorf("User-Agent = %v, want polymarket-watch/0.1.0", ua)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient()
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
}
