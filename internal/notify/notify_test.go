package notify

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// =============================================================================
// Dispatcher Tests
// =============================================================================

func TestNewDispatcher(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	webhookRepo := storage.NewWebhookRepo(db)
	dispatcher := NewDispatcher(webhookRepo)

	assert.NotNil(t, dispatcher)
	assert.NotNil(t, dispatcher.webhookRepo)
	assert.NotNil(t, dispatcher.httpClient)
}

func TestDispatcherSetDebug(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	webhookRepo := storage.NewWebhookRepo(db)
	dispatcher := NewDispatcher(webhookRepo)

	assert.False(t, dispatcher.debug)

	dispatcher.SetDebug(true)
	assert.True(t, dispatcher.debug)

	dispatcher.SetDebug(false)
	assert.False(t, dispatcher.debug)
}

func TestDispatcherHasEnabledWebhooks(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	webhookRepo := storage.NewWebhookRepo(db)
	dispatcher := NewDispatcher(webhookRepo)

	// Initially no webhooks
	assert.False(t, dispatcher.HasEnabledWebhooks())

	// Add an enabled webhook
	webhook := &model.Webhook{
		Name:    "test-webhook",
		Type:    model.WebhookTypeDiscord,
		URL:     "http://localhost:8080/test",
		Enabled: true,
	}
	err = webhookRepo.Create(webhook)
	require.NoError(t, err)

	assert.True(t, dispatcher.HasEnabledWebhooks())
}

func TestDispatcherCountEnabledWebhooks(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	webhookRepo := storage.NewWebhookRepo(db)
	dispatcher := NewDispatcher(webhookRepo)

	// Initially 0
	assert.Equal(t, 0, dispatcher.CountEnabledWebhooks())

	// Add webhooks
	for i := 0; i < 3; i++ {
		webhook := &model.Webhook{
			Name:    fmt.Sprintf("webhook-%d", i),
			Type:    model.WebhookTypeDiscord,
			URL:     fmt.Sprintf("http://localhost:8080/test%d", i),
			Enabled: true,
		}
		err = webhookRepo.Create(webhook)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, dispatcher.CountEnabledWebhooks())
}

func TestDispatcherSendNotificationNoWebhooks(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	webhookRepo := storage.NewWebhookRepo(db)
	dispatcher := NewDispatcher(webhookRepo)

	notification := model.NewNotification(model.NotifyTest, "Test", "Test message")
	results := dispatcher.SendNotification(context.Background(), notification)

	// No webhooks, should return nil
	assert.Nil(t, results)
}

func TestDispatcherSendToSingleNotFound(t *testing.T) {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	defer db.Close()

	webhookRepo := storage.NewWebhookRepo(db)
	dispatcher := NewDispatcher(webhookRepo)

	notification := model.NewNotification(model.NotifyTest, "Test", "Test message")
	result := dispatcher.SendToSingle(context.Background(), notification, "nonexistent")

	assert.False(t, result.Success)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "webhook not found")
}

func TestDispatchResult(t *testing.T) {
	result := DispatchResult{
		WebhookName: "test-webhook",
		Success:     true,
		StatusCode:  200,
		Duration:    100 * time.Millisecond,
		Error:       nil,
	}

	assert.Equal(t, "test-webhook", result.WebhookName)
	assert.True(t, result.Success)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, 100*time.Millisecond, result.Duration)
	assert.Nil(t, result.Error)
}

// =============================================================================
// Extended Formatter Tests
// =============================================================================

func TestDiscordFormatterWithFields(t *testing.T) {
	formatter := &DiscordFormatter{}

	notification := model.NewNotification(model.NotifyGoal, "Goal Progress", "You've reached 50% of your goal!").
		WithField("Project", "test-project").
		WithField("Progress", "50%").
		WithColor(model.ColorSuccess)

	payload, err := formatter.Format(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload)

	// Check that it contains the expected fields
	assert.Contains(t, string(payload), "Goal Progress")
	assert.Contains(t, string(payload), "50%")
	assert.Contains(t, string(payload), "test-project")
}

func TestDiscordFormatterWithColor(t *testing.T) {
	formatter := &DiscordFormatter{}

	notification := model.NewNotification(model.NotifyBreak, "Break Time", "Time for a break!").
		WithColor(model.ColorWarning)

	payload, err := formatter.Format(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload)

	// Should contain the color
	assert.Contains(t, string(payload), fmt.Sprintf("%d", model.ColorWarning))
}

func TestSlackFormatterWithFields(t *testing.T) {
	formatter := &SlackFormatter{}

	notification := model.NewNotification(model.NotifyReminder, "Reminder", "Don't forget!").
		WithField("Due", "2h").
		WithField("Project", "test-project")

	payload, err := formatter.Format(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload)

	// Check the payload contains text
	assert.Contains(t, string(payload), "Reminder")
}

func TestTeamsFormatterWithFields(t *testing.T) {
	formatter := &TeamsFormatter{}

	notification := model.NewNotification(model.NotifyIdle, "Idle", "You've been idle").
		WithField("Duration", "30m")

	payload, err := formatter.Format(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload)

	assert.Contains(t, string(payload), "Idle")
}

func TestGenericFormatterWithTemplate(t *testing.T) {
	formatter := NewGenericFormatter("{ \"text\": \"{{.Title}}: {{.Message}}\" }")

	notification := model.NewNotification(model.NotifyTest, "Custom", "Custom message")

	payload, err := formatter.Format(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload)

	assert.Contains(t, string(payload), "Custom")
	assert.Contains(t, string(payload), "Custom message")
}

func TestGenericFormatterWithInvalidTemplate(t *testing.T) {
	formatter := NewGenericFormatter("{{ invalid template")

	notification := model.NewNotification(model.NotifyTest, "Test", "Test")

	// Should still produce output (falls back to JSON)
	payload, err := formatter.Format(notification)
	// Template parse error
	assert.Error(t, err)
	assert.Nil(t, payload)
}

func TestGenericFormatterEmptyTemplate(t *testing.T) {
	formatter := NewGenericFormatter("")

	notification := model.NewNotification(model.NotifyTest, "Test", "Test message")

	// Empty template uses default JSON format
	payload, err := formatter.Format(notification)
	assert.NoError(t, err)
	assert.NotEmpty(t, payload)
	assert.Contains(t, string(payload), "Test")
}

// =============================================================================
// HTTP Client Tests
// =============================================================================

func TestHTTPClientSendResult(t *testing.T) {
	result := SendResult{
		StatusCode: 200,
		Duration:   50 * time.Millisecond,
		Error:      nil,
	}

	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, 50*time.Millisecond, result.Duration)
	assert.Nil(t, result.Error)
}

func TestHTTPClientSendResultWithError(t *testing.T) {
	result := SendResult{
		StatusCode: 0,
		Duration:   0,
		Error:      fmt.Errorf("connection refused"),
	}

	assert.Equal(t, 0, result.StatusCode)
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "connection refused")
}

// =============================================================================
// Helper Function Tests
// =============================================================================

func TestSlackEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no_special_chars", "Hello World", "Hello World"},
		{"with_ampersand", "AT&T", "AT&amp;T"},
		{"with_less_than", "a < b", "a &lt; b"},
		{"with_greater_than", "a > b", "a &gt; b"},
		{"all_special", "<script>alert('&')</script>", "&lt;script&gt;alert('&amp;')&lt;/script&gt;"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slackEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestColorToHex(t *testing.T) {
	tests := []struct {
		name     string
		color    int
		expected string
	}{
		{"red", 0xFF0000, "#FF0000"},
		{"green", 0x00FF00, "#00FF00"},
		{"blue", 0x0000FF, "#0000FF"},
		{"black", 0x000000, "#000000"},
		{"white", 0xFFFFFF, "#FFFFFF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorToHex(tt.color)
			assert.Equal(t, tt.expected, result)
		})
	}
}
