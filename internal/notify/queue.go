package notify

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/manav03panchal/humantime/internal/config"
	"github.com/manav03panchal/humantime/internal/logging"
)

// QueuedNotification represents a notification waiting to be sent.
type QueuedNotification struct {
	ID          string          `json:"id"`
	WebhookName string          `json:"webhook_name"`
	URL         string          `json:"url"`
	ContentType string          `json:"content_type"`
	Body        json.RawMessage `json:"body"`
	CreatedAt   time.Time       `json:"created_at"`
	NextRetry   time.Time       `json:"next_retry"`
	Attempts    int             `json:"attempts"`
	MaxRetries  int             `json:"max_retries"`
	LastError   string          `json:"last_error,omitempty"`
}

// RetryQueue manages a queue of failed notifications for retry.
type RetryQueue struct {
	mu       sync.RWMutex
	queue    []*QueuedNotification
	client   *HTTPClient
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	interval time.Duration

	// Metrics
	totalQueued int
	totalSent   int
	totalFailed int
}

// NewRetryQueue creates a new retry queue with the given HTTP client.
func NewRetryQueue(client *HTTPClient) *RetryQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &RetryQueue{
		queue:    make([]*QueuedNotification, 0),
		client:   client,
		ctx:      ctx,
		cancel:   cancel,
		interval: config.Global.RetryQueue.CheckInterval,
	}
}

// Start begins processing the retry queue in the background.
func (q *RetryQueue) Start() {
	q.mu.Lock()
	if q.running {
		q.mu.Unlock()
		return
	}
	q.running = true
	q.mu.Unlock()

	q.wg.Add(1)
	go q.processLoop()
}

// Stop stops the retry queue processor.
func (q *RetryQueue) Stop() {
	q.mu.Lock()
	if !q.running {
		q.mu.Unlock()
		return
	}
	q.running = false
	q.mu.Unlock()

	q.cancel()
	q.wg.Wait()
}

// Enqueue adds a failed notification to the retry queue.
func (q *RetryQueue) Enqueue(id, webhookName, url, contentType string, body []byte, maxRetries int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	notification := &QueuedNotification{
		ID:          id,
		WebhookName: webhookName,
		URL:         url,
		ContentType: contentType,
		Body:        body,
		CreatedAt:   time.Now(),
		NextRetry:   time.Now().Add(calculateBackoff(0)),
		Attempts:    0,
		MaxRetries:  maxRetries,
	}

	q.queue = append(q.queue, notification)
	q.totalQueued++

	logging.Info("notification queued for retry",
		logging.KeyWebhook, webhookName,
		"queue_size", len(q.queue))
}

// EnqueueWithError adds a failed notification with error context.
func (q *RetryQueue) EnqueueWithError(id, webhookName, url, contentType string, body []byte, maxRetries int, err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	notification := &QueuedNotification{
		ID:          id,
		WebhookName: webhookName,
		URL:         url,
		ContentType: contentType,
		Body:        body,
		CreatedAt:   time.Now(),
		NextRetry:   time.Now().Add(calculateBackoff(0)),
		Attempts:    0,
		MaxRetries:  maxRetries,
	}

	if err != nil {
		notification.LastError = err.Error()
	}

	q.queue = append(q.queue, notification)
	q.totalQueued++

	logging.Info("notification queued for retry",
		logging.KeyWebhook, webhookName,
		"queue_size", len(q.queue),
		logging.KeyError, err)
}

// processLoop runs in the background and processes queued notifications.
func (q *RetryQueue) processLoop() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.interval)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.processQueue()
		}
	}
}

// processQueue attempts to send all ready notifications.
func (q *RetryQueue) processQueue() {
	q.mu.Lock()
	now := time.Now()

	// Find notifications ready for retry
	var ready []*QueuedNotification
	var remaining []*QueuedNotification

	for _, n := range q.queue {
		if n.NextRetry.Before(now) || n.NextRetry.Equal(now) {
			ready = append(ready, n)
		} else {
			remaining = append(remaining, n)
		}
	}

	q.queue = remaining
	q.mu.Unlock()

	// Process ready notifications
	for _, n := range ready {
		q.processNotification(n)
	}
}

// processNotification attempts to send a single notification.
func (q *RetryQueue) processNotification(n *QueuedNotification) {
	n.Attempts++

	logging.DebugLog("retrying notification",
		logging.KeyWebhook, n.WebhookName,
		"attempt", n.Attempts,
		"max_retries", n.MaxRetries)

	result := q.client.Send(q.ctx, n.URL, n.ContentType, n.Body)

	if result.Error == nil {
		// Success
		q.mu.Lock()
		q.totalSent++
		q.mu.Unlock()

		logging.Info("queued notification sent successfully",
			logging.KeyWebhook, n.WebhookName,
			"attempts", n.Attempts,
			logging.KeyDuration, result.Duration.Milliseconds())
		return
	}

	// Failed
	n.LastError = result.Error.Error()

	if n.Attempts >= n.MaxRetries {
		// Max retries exceeded
		q.mu.Lock()
		q.totalFailed++
		q.mu.Unlock()

		logging.Warn("notification failed after max retries",
			logging.KeyWebhook, n.WebhookName,
			"attempts", n.Attempts,
			logging.KeyError, result.Error)
		return
	}

	// Re-queue with backoff
	n.NextRetry = time.Now().Add(calculateBackoff(n.Attempts))

	q.mu.Lock()
	q.queue = append(q.queue, n)
	q.mu.Unlock()

	logging.DebugLog("notification re-queued",
		logging.KeyWebhook, n.WebhookName,
		"next_retry", n.NextRetry,
		"attempts", n.Attempts)
}

// calculateBackoff returns the backoff duration for the given attempt number.
// Uses configurable exponential backoff schedule from config.
func calculateBackoff(attempt int) time.Duration {
	backoffs := config.Global.RetryQueue.BackoffSchedule

	if attempt >= len(backoffs) {
		return backoffs[len(backoffs)-1]
	}
	return backoffs[attempt]
}

// QueueStats returns statistics about the retry queue.
type QueueStats struct {
	QueueSize   int `json:"queue_size"`
	TotalQueued int `json:"total_queued"`
	TotalSent   int `json:"total_sent"`
	TotalFailed int `json:"total_failed"`
}

// Stats returns current queue statistics.
func (q *RetryQueue) Stats() QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return QueueStats{
		QueueSize:   len(q.queue),
		TotalQueued: q.totalQueued,
		TotalSent:   q.totalSent,
		TotalFailed: q.totalFailed,
	}
}

// Pending returns the number of pending notifications.
func (q *RetryQueue) Pending() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

// Clear removes all pending notifications from the queue.
func (q *RetryQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = make([]*QueuedNotification, 0)
}
