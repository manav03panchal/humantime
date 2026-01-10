package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/notify"
)

// TestHTTPClientRetry verifies retry behavior on server errors.
func TestHTTPClientRetry(t *testing.T) {
	attempts := atomic.Int32{}

	// Server that fails first 2 attempts, then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("temporary error"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	client := notify.NewHTTPClient()
	result := client.Send(context.Background(), server.URL, "application/json", []byte(`{}`))

	if result.Error != nil {
		t.Errorf("Expected success after retries, got error: %v", result.Error)
	}
	if result.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", result.StatusCode)
	}
	if result.Attempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", result.Attempts)
	}
}

// TestHTTPClientTimeout verifies timeout handling.
func TestHTTPClientTimeout(t *testing.T) {
	// Server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client := notify.NewHTTPClient()

	// Use a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	result := client.Send(ctx, server.URL, "application/json", []byte(`{}`))

	if result.Error == nil {
		t.Error("Expected timeout error")
	}
}

// TestHTTPClientSuccessNoRetry verifies no retry on success.
func TestHTTPClientSuccessNoRetry(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := notify.NewHTTPClient()
	result := client.Send(context.Background(), server.URL, "application/json", []byte(`{}`))

	if result.Error != nil {
		t.Errorf("Unexpected error: %v", result.Error)
	}
	if attempts.Load() != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts.Load())
	}
}

// TestHTTPClientNoRetryOnClientError verifies no retry on 4xx errors.
func TestHTTPClientNoRetryOnClientError(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client := notify.NewHTTPClient()
	result := client.Send(context.Background(), server.URL, "application/json", []byte(`{}`))

	if result.Error == nil {
		t.Error("Expected error for 400 response")
	}
	if attempts.Load() != 1 {
		t.Errorf("Should not retry on client error, got %d attempts", attempts.Load())
	}
}

// TestHTTPClientRateLimitRetry verifies retry on 429 rate limit.
func TestHTTPClientRateLimitRetry(t *testing.T) {
	attempts := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := notify.NewHTTPClient()

	// Use shorter timeout for test
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := client.Send(ctx, server.URL, "application/json", []byte(`{}`))

	if result.Error != nil {
		t.Errorf("Expected success after rate limit retry, got: %v", result.Error)
	}
	if attempts.Load() < 2 {
		t.Errorf("Expected retry on rate limit, got %d attempts", attempts.Load())
	}
}

// TestRetryQueueBasic verifies basic retry queue functionality.
func TestRetryQueueBasic(t *testing.T) {
	client := notify.NewHTTPClient()
	queue := notify.NewRetryQueue(client)

	// Enqueue a notification
	queue.Enqueue("test-1", "webhook1", "https://example.com/hook", "application/json", []byte(`{}`), 3)

	// Check queue stats
	stats := queue.Stats()
	if stats.QueueSize != 1 {
		t.Errorf("Expected queue size 1, got %d", stats.QueueSize)
	}
	if stats.TotalQueued != 1 {
		t.Errorf("Expected total queued 1, got %d", stats.TotalQueued)
	}
}

// TestRetryQueuePending verifies pending count.
func TestRetryQueuePending(t *testing.T) {
	client := notify.NewHTTPClient()
	queue := notify.NewRetryQueue(client)

	if queue.Pending() != 0 {
		t.Errorf("Empty queue should have 0 pending, got %d", queue.Pending())
	}

	queue.Enqueue("test-1", "webhook1", "https://example.com", "application/json", []byte(`{}`), 3)
	queue.Enqueue("test-2", "webhook2", "https://example.com", "application/json", []byte(`{}`), 3)

	if queue.Pending() != 2 {
		t.Errorf("Expected 2 pending, got %d", queue.Pending())
	}
}

// TestRetryQueueClear verifies queue clearing.
func TestRetryQueueClear(t *testing.T) {
	client := notify.NewHTTPClient()
	queue := notify.NewRetryQueue(client)

	queue.Enqueue("test-1", "webhook1", "https://example.com", "application/json", []byte(`{}`), 3)
	queue.Enqueue("test-2", "webhook2", "https://example.com", "application/json", []byte(`{}`), 3)

	queue.Clear()

	if queue.Pending() != 0 {
		t.Errorf("Cleared queue should have 0 pending, got %d", queue.Pending())
	}
}

// TestRetryQueueWithError verifies enqueueing with error context.
func TestRetryQueueWithError(t *testing.T) {
	client := notify.NewHTTPClient()
	queue := notify.NewRetryQueue(client)

	testErr := &testError{msg: "connection refused"}
	queue.EnqueueWithError("test-1", "webhook1", "https://example.com", "application/json", []byte(`{}`), 3, testErr)

	if queue.Pending() != 1 {
		t.Errorf("Expected 1 pending, got %d", queue.Pending())
	}
}

// TestRetryQueueStartStop verifies queue lifecycle.
func TestRetryQueueStartStop(t *testing.T) {
	client := notify.NewHTTPClient()
	queue := notify.NewRetryQueue(client)

	// Start should not panic
	queue.Start()

	// Double start should not panic
	queue.Start()

	// Stop should not panic
	queue.Stop()

	// Double stop should not panic
	queue.Stop()
}
