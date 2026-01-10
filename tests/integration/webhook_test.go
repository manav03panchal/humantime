package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Webhook Creation Tests
// =============================================================================

func TestWebhookCreation(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewWebhookRepo(db)

	t.Run("creates webhook with all fields", func(t *testing.T) {
		webhook := model.NewWebhook("work-discord", model.WebhookTypeDiscord, "https://discord.com/api/webhooks/123/abc")

		err := repo.Create(webhook)
		require.NoError(t, err)
		assert.NotEmpty(t, webhook.Key)

		retrieved, err := repo.Get("work-discord")
		require.NoError(t, err)

		assert.Equal(t, "work-discord", retrieved.Name)
		assert.Equal(t, model.WebhookTypeDiscord, retrieved.Type)
		assert.Equal(t, "https://discord.com/api/webhooks/123/abc", retrieved.URL)
		assert.True(t, retrieved.Enabled)
	})

	t.Run("creates webhooks of different types", func(t *testing.T) {
		slack := model.NewWebhook("team-slack", model.WebhookTypeSlack, "https://hooks.slack.com/services/T123/B456/xyz")
		teams := model.NewWebhook("office-teams", model.WebhookTypeTeams, "https://outlook.office.com/webhook/123")
		generic := model.NewWebhook("custom-hook", model.WebhookTypeGeneric, "https://example.com/webhook")

		require.NoError(t, repo.Create(slack))
		require.NoError(t, repo.Create(teams))
		require.NoError(t, repo.Create(generic))

		s, err := repo.Get("team-slack")
		require.NoError(t, err)
		assert.Equal(t, model.WebhookTypeSlack, s.Type)

		tm, err := repo.Get("office-teams")
		require.NoError(t, err)
		assert.Equal(t, model.WebhookTypeTeams, tm.Type)

		g, err := repo.Get("custom-hook")
		require.NoError(t, err)
		assert.Equal(t, model.WebhookTypeGeneric, g.Type)
	})

	t.Run("new webhooks are enabled by default", func(t *testing.T) {
		webhook := model.NewWebhook("enabled-test", model.WebhookTypeDiscord, "https://discord.com/api/webhooks/456/def")
		require.NoError(t, repo.Create(webhook))

		retrieved, err := repo.Get("enabled-test")
		require.NoError(t, err)
		assert.True(t, retrieved.Enabled)
	})
}

// =============================================================================
// Webhook Listing Tests
// =============================================================================

func TestWebhookListing(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewWebhookRepo(db)

	t.Run("lists empty webhooks", func(t *testing.T) {
		webhooks, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, webhooks)
	})

	t.Run("lists all webhooks", func(t *testing.T) {
		require.NoError(t, repo.Create(model.NewWebhook("hook1", model.WebhookTypeDiscord, "https://discord.com/1")))
		require.NoError(t, repo.Create(model.NewWebhook("hook2", model.WebhookTypeSlack, "https://hooks.slack.com/2")))
		require.NoError(t, repo.Create(model.NewWebhook("hook3", model.WebhookTypeTeams, "https://outlook.office.com/3")))

		webhooks, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, webhooks, 3)
	})

	t.Run("lists enabled webhooks only", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewWebhookRepo(db)

		w1 := model.NewWebhook("enabled1", model.WebhookTypeDiscord, "https://discord.com/enabled1")
		w2 := model.NewWebhook("enabled2", model.WebhookTypeDiscord, "https://discord.com/enabled2")
		w3 := model.NewWebhook("disabled1", model.WebhookTypeDiscord, "https://discord.com/disabled1")
		w3.Enabled = false

		require.NoError(t, repo.Create(w1))
		require.NoError(t, repo.Create(w2))
		require.NoError(t, repo.Create(w3))

		enabled, err := repo.ListEnabled()
		require.NoError(t, err)
		assert.Len(t, enabled, 2)

		for _, w := range enabled {
			assert.True(t, w.Enabled)
		}
	})
}

// =============================================================================
// Webhook Enable/Disable Tests
// =============================================================================

func TestWebhookEnableDisable(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewWebhookRepo(db)

	t.Run("disables webhook", func(t *testing.T) {
		webhook := model.NewWebhook("to-disable", model.WebhookTypeDiscord, "https://discord.com/disable")
		require.NoError(t, repo.Create(webhook))

		err := repo.Disable("to-disable")
		require.NoError(t, err)

		retrieved, err := repo.Get("to-disable")
		require.NoError(t, err)
		assert.False(t, retrieved.Enabled)
	})

	t.Run("enables disabled webhook", func(t *testing.T) {
		webhook := model.NewWebhook("to-enable", model.WebhookTypeDiscord, "https://discord.com/enable")
		webhook.Enabled = false
		require.NoError(t, repo.Create(webhook))

		err := repo.Enable("to-enable")
		require.NoError(t, err)

		retrieved, err := repo.Get("to-enable")
		require.NoError(t, err)
		assert.True(t, retrieved.Enabled)
	})

	t.Run("enable non-existent webhook returns error", func(t *testing.T) {
		err := repo.Enable("nonexistent")
		assert.Error(t, err)
	})
}

// =============================================================================
// Webhook Last Used Tracking Tests
// =============================================================================

func TestWebhookLastUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewWebhookRepo(db)

	t.Run("updates last used timestamp on success", func(t *testing.T) {
		webhook := model.NewWebhook("track-usage", model.WebhookTypeDiscord, "https://discord.com/track")
		require.NoError(t, repo.Create(webhook))

		before := time.Now()
		err := repo.UpdateLastUsed("track-usage", nil)
		require.NoError(t, err)

		retrieved, err := repo.Get("track-usage")
		require.NoError(t, err)

		assert.True(t, retrieved.LastUsed.After(before) || retrieved.LastUsed.Equal(before))
		assert.Empty(t, retrieved.LastError)
	})

	t.Run("records last error on failure", func(t *testing.T) {
		webhook := model.NewWebhook("track-error", model.WebhookTypeDiscord, "https://discord.com/error")
		require.NoError(t, repo.Create(webhook))

		testErr := errors.New("HTTP 404: Not Found")
		err := repo.UpdateLastUsed("track-error", testErr)
		require.NoError(t, err)

		retrieved, err := repo.Get("track-error")
		require.NoError(t, err)

		assert.Equal(t, "HTTP 404: Not Found", retrieved.LastError)
	})

	t.Run("clears last error on success after failure", func(t *testing.T) {
		webhook := model.NewWebhook("track-clear", model.WebhookTypeDiscord, "https://discord.com/clear")
		require.NoError(t, repo.Create(webhook))

		// First, record an error
		require.NoError(t, repo.UpdateLastUsed("track-clear", errors.New("some error")))

		// Then, record success
		require.NoError(t, repo.UpdateLastUsed("track-clear", nil))

		retrieved, err := repo.Get("track-clear")
		require.NoError(t, err)

		assert.Empty(t, retrieved.LastError)
	})
}

// =============================================================================
// Webhook Deletion Tests
// =============================================================================

func TestWebhookDeletion(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewWebhookRepo(db)

	t.Run("deletes existing webhook", func(t *testing.T) {
		webhook := model.NewWebhook("to-delete", model.WebhookTypeDiscord, "https://discord.com/delete")
		require.NoError(t, repo.Create(webhook))

		exists, err := repo.Exists("to-delete")
		require.NoError(t, err)
		assert.True(t, exists)

		err = repo.Delete("to-delete")
		require.NoError(t, err)

		exists, err = repo.Exists("to-delete")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent webhook does not error", func(t *testing.T) {
		err := repo.Delete("nonexistent")
		assert.NoError(t, err)
	})
}

// =============================================================================
// Webhook Model Tests
// =============================================================================

func TestWebhookModel(t *testing.T) {
	t.Run("validates webhook types", func(t *testing.T) {
		assert.True(t, model.IsValidWebhookType(model.WebhookTypeDiscord))
		assert.True(t, model.IsValidWebhookType(model.WebhookTypeSlack))
		assert.True(t, model.IsValidWebhookType(model.WebhookTypeTeams))
		assert.True(t, model.IsValidWebhookType(model.WebhookTypeGeneric))
		assert.False(t, model.IsValidWebhookType("invalid"))
		assert.False(t, model.IsValidWebhookType(""))
	})

	t.Run("validates webhook names", func(t *testing.T) {
		assert.True(t, model.IsValidWebhookName("discord"))
		assert.True(t, model.IsValidWebhookName("work-discord"))
		assert.True(t, model.IsValidWebhookName("my_webhook_123"))
		assert.True(t, model.IsValidWebhookName("A-webhook"))

		assert.False(t, model.IsValidWebhookName(""))           // Empty
		assert.False(t, model.IsValidWebhookName("-invalid"))   // Starts with dash
		assert.False(t, model.IsValidWebhookName("_invalid"))   // Starts with underscore
		assert.False(t, model.IsValidWebhookName("has space"))  // Contains space
		assert.False(t, model.IsValidWebhookName("has.dot"))    // Contains dot
	})

	t.Run("detects webhook type from URL", func(t *testing.T) {
		assert.Equal(t, model.WebhookTypeDiscord, model.DetectWebhookType("https://discord.com/api/webhooks/123/abc"))
		assert.Equal(t, model.WebhookTypeSlack, model.DetectWebhookType("https://hooks.slack.com/services/T123/B456/xyz"))
		assert.Equal(t, model.WebhookTypeTeams, model.DetectWebhookType("https://outlook.office.com/webhook/123"))
		assert.Equal(t, model.WebhookTypeTeams, model.DetectWebhookType("https://webhook.office.com/something"))
		assert.Equal(t, model.WebhookTypeGeneric, model.DetectWebhookType("https://example.com/webhook"))
	})

	t.Run("masks URL correctly", func(t *testing.T) {
		short := &model.Webhook{URL: "https://short.url"}
		assert.Equal(t, "https://short.url", short.MaskedURL())

		long := &model.Webhook{URL: "https://discord.com/api/webhooks/1234567890/abcdefghijklmnopqrstuvwxyz"}
		assert.Equal(t, "https://discord.com/api/webhoo***", long.MaskedURL())
	})

	t.Run("generates correct key", func(t *testing.T) {
		key := model.GenerateWebhookKey("my-webhook")
		assert.Equal(t, "webhook:my-webhook", key)
	})
}

// =============================================================================
// Webhook Edge Cases
// =============================================================================

func TestWebhookEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewWebhookRepo(db)

	t.Run("overwrites webhook with same name", func(t *testing.T) {
		w1 := model.NewWebhook("same-name", model.WebhookTypeDiscord, "https://discord.com/first")
		require.NoError(t, repo.Create(w1))

		w2 := model.NewWebhook("same-name", model.WebhookTypeSlack, "https://hooks.slack.com/second")
		require.NoError(t, repo.Create(w2))

		retrieved, err := repo.Get("same-name")
		require.NoError(t, err)
		assert.Equal(t, model.WebhookTypeSlack, retrieved.Type)
		assert.Equal(t, "https://hooks.slack.com/second", retrieved.URL)
	})

	t.Run("webhook with template for generic type", func(t *testing.T) {
		webhook := model.NewWebhook("templated", model.WebhookTypeGeneric, "https://example.com/hook")
		webhook.Template = `{"text": "{{.Message}}"}`
		require.NoError(t, repo.Create(webhook))

		retrieved, err := repo.Get("templated")
		require.NoError(t, err)
		assert.Equal(t, `{"text": "{{.Message}}"}`, retrieved.Template)
	})
}
