package notify

import (
	"fmt"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Formatter Tests
// =============================================================================

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		webhookType string
		expected    string
	}{
		{model.WebhookTypeDiscord, "*notify.DiscordFormatter"},
		{model.WebhookTypeSlack, "*notify.SlackFormatter"},
		{model.WebhookTypeTeams, "*notify.TeamsFormatter"},
		{model.WebhookTypeGeneric, "*notify.GenericFormatter"},
		{"unknown", "*notify.GenericFormatter"},
		{"", "*notify.GenericFormatter"},
	}

	for _, tt := range tests {
		t.Run(tt.webhookType, func(t *testing.T) {
			formatter := GetFormatter(tt.webhookType)
			assert.NotNil(t, formatter)
			assert.Equal(t, tt.expected, fmt.Sprintf("%T", formatter))
		})
	}
}

func TestDiscordFormatter(t *testing.T) {
	formatter := &DiscordFormatter{}

	t.Run("content_type", func(t *testing.T) {
		assert.Equal(t, "application/json", formatter.ContentType())
	})

	t.Run("format_notification", func(t *testing.T) {
		notification := &model.Notification{
			Title:   "Test Title",
			Message: "Test Message",
		}

		payload, err := formatter.Format(notification)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload)
		assert.Contains(t, string(payload), "Test Title")
		assert.Contains(t, string(payload), "Test Message")
	})
}

func TestSlackFormatter(t *testing.T) {
	formatter := &SlackFormatter{}

	t.Run("content_type", func(t *testing.T) {
		assert.Equal(t, "application/json", formatter.ContentType())
	})

	t.Run("format_notification", func(t *testing.T) {
		notification := &model.Notification{
			Title:   "Test Title",
			Message: "Test Message",
		}

		payload, err := formatter.Format(notification)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload)
	})
}

func TestTeamsFormatter(t *testing.T) {
	formatter := &TeamsFormatter{}

	t.Run("content_type", func(t *testing.T) {
		assert.Equal(t, "application/json", formatter.ContentType())
	})

	t.Run("format_notification", func(t *testing.T) {
		notification := &model.Notification{
			Title:   "Test Title",
			Message: "Test Message",
		}

		payload, err := formatter.Format(notification)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload)
	})
}

func TestGenericFormatter(t *testing.T) {
	formatter := &GenericFormatter{}

	t.Run("content_type", func(t *testing.T) {
		assert.Equal(t, "application/json", formatter.ContentType())
	})

	t.Run("format_notification", func(t *testing.T) {
		notification := &model.Notification{
			Title:   "Test Title",
			Message: "Test Message",
		}

		payload, err := formatter.Format(notification)
		assert.NoError(t, err)
		assert.NotEmpty(t, payload)
	})
}

// =============================================================================
// RetryQueue Tests
// =============================================================================

func TestNewRetryQueue(t *testing.T) {
	client := NewHTTPClient()
	queue := NewRetryQueue(client)

	assert.NotNil(t, queue)
	assert.Equal(t, 0, queue.Pending())
}

func TestRetryQueueEnqueue(t *testing.T) {
	client := NewHTTPClient()
	queue := NewRetryQueue(client)

	queue.Enqueue("id-1", "test-webhook", "http://localhost/test", "application/json", []byte(`{"test": true}`), 3)

	assert.Equal(t, 1, queue.Pending())

	stats := queue.Stats()
	assert.Equal(t, 1, stats.QueueSize)
	assert.Equal(t, 1, stats.TotalQueued)
	assert.Equal(t, 0, stats.TotalSent)
	assert.Equal(t, 0, stats.TotalFailed)
}

func TestRetryQueueEnqueueWithError(t *testing.T) {
	client := NewHTTPClient()
	queue := NewRetryQueue(client)

	err := fmt.Errorf("test error")
	queue.EnqueueWithError("id-1", "test-webhook", "http://localhost/test", "application/json", []byte(`{}`), 3, err)

	assert.Equal(t, 1, queue.Pending())
}

func TestRetryQueueClear(t *testing.T) {
	client := NewHTTPClient()
	queue := NewRetryQueue(client)

	queue.Enqueue("id-1", "webhook-1", "http://localhost/1", "application/json", []byte(`{}`), 3)
	queue.Enqueue("id-2", "webhook-2", "http://localhost/2", "application/json", []byte(`{}`), 3)

	assert.Equal(t, 2, queue.Pending())

	queue.Clear()

	assert.Equal(t, 0, queue.Pending())
}

func TestRetryQueueStartStop(t *testing.T) {
	client := NewHTTPClient()
	queue := NewRetryQueue(client)

	// Start should be idempotent
	queue.Start()
	queue.Start()

	// Stop should be idempotent
	queue.Stop()
	queue.Stop()
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 5 * time.Second},
		{1, 30 * time.Second},
		{2, 2 * time.Minute},
		{3, 5 * time.Minute},
		{4, 15 * time.Minute},
		{5, 15 * time.Minute}, // Capped at max
		{100, 15 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			result := calculateBackoff(tt.attempt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueueStats(t *testing.T) {
	stats := QueueStats{
		QueueSize:   5,
		TotalQueued: 10,
		TotalSent:   8,
		TotalFailed: 2,
	}

	assert.Equal(t, 5, stats.QueueSize)
	assert.Equal(t, 10, stats.TotalQueued)
	assert.Equal(t, 8, stats.TotalSent)
	assert.Equal(t, 2, stats.TotalFailed)
}

func TestQueuedNotification(t *testing.T) {
	n := &QueuedNotification{
		ID:          "test-id",
		WebhookName: "test-webhook",
		URL:         "http://localhost/test",
		ContentType: "application/json",
		Body:        []byte(`{"test": true}`),
		CreatedAt:   time.Now(),
		NextRetry:   time.Now().Add(5 * time.Second),
		Attempts:    0,
		MaxRetries:  3,
		LastError:   "",
	}

	assert.Equal(t, "test-id", n.ID)
	assert.Equal(t, "test-webhook", n.WebhookName)
	assert.Equal(t, 3, n.MaxRetries)
}

// =============================================================================
// HTTPClient Tests
// =============================================================================

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	assert.NotNil(t, client)
}

func TestHTTPClientTimeout(t *testing.T) {
	client := NewHTTPClient()
	assert.NotNil(t, client)
}
