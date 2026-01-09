// Package api provides HTTP client functionality for Polymarket APIs.
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// DefaultRateLimit is the default rate limit in requests per minute.
	DefaultRateLimit = 100
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultRetries is the default number of retry attempts.
	DefaultRetries = 3
	// UserAgent is the User-Agent header sent with requests.
	UserAgent = "polymarket-watch/0.1.0"
)

// retryableStatusCodes defines HTTP status codes that should trigger a retry.
var retryableStatusCodes = map[int]bool{
	http.StatusTooManyRequests:     true,
	http.StatusServiceUnavailable:  true,
	http.StatusGatewayTimeout:      true,
	http.StatusBadGateway:          true,
	http.StatusRequestTimeout:      true,
	http.StatusInternalServerError: true,
}

// Client is an HTTP client with rate limiting, retry logic, and timeout handling.
type Client struct {
	httpClient  *http.Client
	rateLimiter *rateLimiter
	retries     int
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithRateLimit sets the rate limit in requests per minute.
// If limit is 0 or negative, the default rate limit is used.
func WithRateLimit(requestsPerMinute int) ClientOption {
	return func(c *Client) {
		if requestsPerMinute <= 0 {
			requestsPerMinute = DefaultRateLimit
		}
		c.rateLimiter = newRateLimiter(requestsPerMinute)
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetries sets the number of retry attempts for failed requests.
func WithRetries(retries int) ClientOption {
	return func(c *Client) {
		if retries < 0 {
			retries = 0
		}
		c.retries = retries
	}
}

// NewClient creates a new HTTP client with the given options.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		rateLimiter: newRateLimiter(DefaultRateLimit),
		retries:     DefaultRetries,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Get performs an HTTP GET request to the specified URL.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	return c.doWithRetry(ctx, http.MethodGet, url, nil)
}

// GetWithParams performs an HTTP GET request with query parameters.
func (c *Client) GetWithParams(ctx context.Context, baseURL string, params map[string]string) (*http.Response, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	return c.doWithRetry(ctx, http.MethodGet, u.String(), nil)
}

// doWithRetry performs the request with retry logic for transient failures.
func (c *Client) doWithRetry(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	for attempt := 0; attempt <= c.retries; attempt++ {
		// Wait for rate limiter
		if err := c.rateLimiter.wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}
		req.Header.Set("User-Agent", UserAgent)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			// Don't retry on context cancellation
			if ctx.Err() != nil {
				return nil, err
			}
			// Wait before retry with exponential backoff
			if attempt < c.retries {
				c.backoff(ctx, attempt)
			}
			continue
		}

		// Check if we should retry based on status code
		if retryableStatusCodes[resp.StatusCode] && attempt < c.retries {
			// Close the body before retry to avoid resource leak
			resp.Body.Close()
			lastResp = nil
			c.backoff(ctx, attempt)
			continue
		}

		return resp, nil
	}

	// Return the last response if we have one, otherwise the last error
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

// backoff waits with exponential backoff before retry.
func (c *Client) backoff(ctx context.Context, attempt int) {
	// Exponential backoff: 100ms, 200ms, 400ms, ...
	delay := time.Duration(100<<attempt) * time.Millisecond
	if delay > 5*time.Second {
		delay = 5 * time.Second
	}

	timer := time.NewTimer(delay)
	select {
	case <-ctx.Done():
		timer.Stop()
	case <-timer.C:
	}
}

// rateLimiter implements a token bucket rate limiter.
type rateLimiter struct {
	mu             sync.Mutex
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
}

// newRateLimiter creates a new rate limiter with the specified requests per minute.
func newRateLimiter(requestsPerMinute int) *rateLimiter {
	rate := float64(requestsPerMinute) / 60.0 // Convert to per second
	return &rateLimiter{
		tokens:         float64(requestsPerMinute) / 60.0, // Start with 1 second worth
		maxTokens:      float64(requestsPerMinute) / 6.0,  // Allow bursts of ~10 seconds
		refillRate:     rate,
		lastRefillTime: time.Now(),
	}
}

// wait blocks until a token is available or the context is cancelled.
func (r *rateLimiter) wait(ctx context.Context) error {
	for {
		r.mu.Lock()
		r.refill()

		if r.tokens >= 1.0 {
			r.tokens--
			r.mu.Unlock()
			return nil
		}

		// Calculate time until next token
		waitTime := time.Duration((1.0 - r.tokens) / r.refillRate * float64(time.Second))
		r.mu.Unlock()

		// Wait for token or context cancellation
		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Try again
		}
	}
}

// refill adds tokens based on time elapsed since last refill.
func (r *rateLimiter) refill() {
	now := time.Now()
	elapsed := now.Sub(r.lastRefillTime).Seconds()
	r.tokens += elapsed * r.refillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefillTime = now
}
