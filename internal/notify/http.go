package notify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/manav03panchal/humantime/internal/config"
	"github.com/manav03panchal/humantime/internal/logging"
)

// HTTPClient handles HTTP requests with retry logic.
type HTTPClient struct {
	client     *http.Client
	maxRetries int
	retryDelay []time.Duration
}

// NewHTTPClient creates a new HTTP client with default settings from config.
func NewHTTPClient() *HTTPClient {
	cfg := config.Global.HTTP
	return &HTTPClient{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelays,
	}
}

// SendResult contains the result of a send operation.
type SendResult struct {
	StatusCode int
	Duration   time.Duration
	Attempts   int
	Error      error
}

// Send sends a POST request to the given URL with retry logic.
func (c *HTTPClient) Send(ctx context.Context, url string, contentType string, body []byte) *SendResult {
	result := &SendResult{}
	start := time.Now()

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Wait before retry (except first attempt)
		if attempt > 0 && attempt < len(c.retryDelay) {
			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				result.Duration = time.Since(start)
				return result
			case <-time.After(c.retryDelay[attempt]):
			}
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			result.Error = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", contentType)
		req.Header.Set("User-Agent", "Humantime/1.0")

		// Send request
		resp, err := c.client.Do(req)
		if err != nil {
			result.Error = fmt.Errorf("request failed: %w", err)
			continue
		}

		// Read and close body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logging.Warn("failed to read response body", logging.KeyError, err, "url", url)
		}
		resp.Body.Close()

		result.StatusCode = resp.StatusCode

		// Check for success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			result.Error = nil
			result.Duration = time.Since(start)
			return result
		}

		// Rate limiting - should retry
		if resp.StatusCode == 429 {
			result.Error = fmt.Errorf("rate limited (HTTP 429)")
			continue
		}

		// Server error - should retry
		if resp.StatusCode >= 500 {
			result.Error = fmt.Errorf("server error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
			continue
		}

		// Client error - don't retry
		result.Error = fmt.Errorf("client error (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
		result.Duration = time.Since(start)
		return result
	}

	result.Duration = time.Since(start)
	if result.Error == nil {
		result.Error = fmt.Errorf("max retries exceeded")
	}
	return result
}

// SendWithTimeout sends a request with a specific timeout.
func (c *HTTPClient) SendWithTimeout(url string, contentType string, body []byte, timeout time.Duration) *SendResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Send(ctx, url, contentType, body)
}
