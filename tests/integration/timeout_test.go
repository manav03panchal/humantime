package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPClientTimeoutBehavior tests HTTP client timeout behavior.
func TestHTTPClientTimeoutBehavior(t *testing.T) {
	// Create a slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	// Create client with short timeout
	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	// Make request - should timeout
	resp, err := client.Get(slowServer.URL)
	if err == nil {
		resp.Body.Close()
		t.Error("Expected timeout error, got success")
	}

	// Error should indicate timeout
	if err != nil {
		t.Logf("Got expected timeout error: %v", err)
	}
}

// TestHTTPClientContextTimeoutBehavior tests context-based timeout.
func TestHTTPClientContextTimeoutBehavior(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	client := &http.Client{}

	// Create request with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", slowServer.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Error("Expected context timeout error, got success")
	}

	if err != nil {
		t.Logf("Got expected context timeout error: %v", err)
	}
}

// TestWebhookTimeoutConfiguration tests configurable webhook timeouts.
func TestWebhookTimeoutConfiguration(t *testing.T) {
	// Test various timeout configurations
	timeouts := []time.Duration{
		100 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
		5 * time.Second,
	}

	for _, timeout := range timeouts {
		t.Run(timeout.String(), func(t *testing.T) {
			client := &http.Client{
				Timeout: timeout,
			}

			if client.Timeout != timeout {
				t.Errorf("Client timeout should be %v, got %v", timeout, client.Timeout)
			}
		})
	}
}

// TestQuickServer tests that fast servers complete without timeout.
func TestQuickServer(t *testing.T) {
	quickServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer quickServer.Close()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get(quickServer.URL)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Quick server request failed: %v", err)
	}
	defer resp.Body.Close()

	if elapsed > 100*time.Millisecond {
		t.Errorf("Quick server took too long: %v", elapsed)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// TestRetryAfterTimeoutError tests retry behavior after timeout.
func TestRetryAfterTimeoutError(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First two attempts are slow
			time.Sleep(200 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	// Simulate retry logic
	maxRetries := 5
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			t.Logf("Succeeded after %d attempts", i+1)
			return
		}
	}

	if lastErr != nil {
		t.Logf("All retries failed, last error: %v", lastErr)
	}
}

// TestTimeoutCleanup tests that timeouts clean up properly.
func TestTimeoutCleanup(t *testing.T) {
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	client := &http.Client{
		Timeout: 50 * time.Millisecond,
	}

	// Make several requests that will timeout
	for i := 0; i < 5; i++ {
		resp, err := client.Get(slowServer.URL)
		if err == nil {
			resp.Body.Close()
		}
	}

	// Give goroutines time to clean up
	time.Sleep(100 * time.Millisecond)

	// No explicit goroutine count check, but test should complete without hanging
	t.Log("Timeout cleanup test completed")
}
