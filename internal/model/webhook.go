package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// PrefixWebhook is the database key prefix for webhooks.
const PrefixWebhook = "webhook"

// Webhook type constants.
const (
	WebhookTypeDiscord = "discord"
	WebhookTypeSlack   = "slack"
	WebhookTypeTeams   = "teams"
	WebhookTypeGeneric = "generic"
)

// Webhook represents a notification webhook configuration.
type Webhook struct {
	Key       string    `json:"key"`
	Name      string    `json:"name" validate:"required,max=50"`
	Type      string    `json:"type" validate:"required,oneof=discord slack teams generic"`
	URL       string    `json:"url" validate:"required,url"`
	Enabled   bool      `json:"enabled"`
	Template  string    `json:"template,omitempty"` // For generic webhooks
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
	LastError string    `json:"last_error,omitempty"`
}

// SetKey sets the database key for this webhook.
func (w *Webhook) SetKey(key string) {
	w.Key = key
}

// GetKey returns the database key for this webhook.
func (w *Webhook) GetKey() string {
	return w.Key
}

// IsEnabled returns true if the webhook is enabled.
func (w *Webhook) IsEnabled() bool {
	return w.Enabled
}

// MaskedURL returns the URL with sensitive parts masked.
func (w *Webhook) MaskedURL() string {
	// Show first 30 chars and mask the rest
	if len(w.URL) > 40 {
		return w.URL[:30] + "***"
	}
	return w.URL
}

// GenerateWebhookKey generates a database key for a webhook.
func GenerateWebhookKey(name string) string {
	return fmt.Sprintf("%s:%s", PrefixWebhook, name)
}

// NewWebhook creates a new enabled webhook.
func NewWebhook(name, webhookType, url string) *Webhook {
	return &Webhook{
		Key:       GenerateWebhookKey(name),
		Name:      name,
		Type:      webhookType,
		URL:       url,
		Enabled:   true,
		CreatedAt: time.Now(),
	}
}

// ValidWebhookTypes returns the list of valid webhook types.
func ValidWebhookTypes() []string {
	return []string{WebhookTypeDiscord, WebhookTypeSlack, WebhookTypeTeams, WebhookTypeGeneric}
}

// IsValidWebhookType checks if a type is valid.
func IsValidWebhookType(t string) bool {
	for _, valid := range ValidWebhookTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// webhookNameRegex validates webhook names (alphanumeric, dash, underscore).
var webhookNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// IsValidWebhookName checks if a webhook name is valid.
func IsValidWebhookName(name string) bool {
	if len(name) == 0 || len(name) > 50 {
		return false
	}
	return webhookNameRegex.MatchString(name)
}

// DetectWebhookType attempts to detect the webhook type from the URL.
func DetectWebhookType(url string) string {
	urlLower := strings.ToLower(url)

	switch {
	case strings.Contains(urlLower, "discord.com/api/webhooks"):
		return WebhookTypeDiscord
	case strings.Contains(urlLower, "hooks.slack.com"):
		return WebhookTypeSlack
	case strings.Contains(urlLower, "outlook.office.com/webhook") ||
		strings.Contains(urlLower, "webhook.office.com"):
		return WebhookTypeTeams
	default:
		return WebhookTypeGeneric
	}
}
