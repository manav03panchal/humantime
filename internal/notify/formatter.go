// Package notify provides notification dispatch and formatting for webhooks.
package notify

import (
	"github.com/manav03panchal/humantime/internal/model"
)

// Formatter formats notifications for a specific webhook type.
type Formatter interface {
	// Format converts a notification into the webhook-specific payload.
	Format(n *model.Notification) ([]byte, error)

	// ContentType returns the HTTP Content-Type for the payload.
	ContentType() string
}

// GetFormatter returns the appropriate formatter for a webhook type.
func GetFormatter(webhookType string) Formatter {
	switch webhookType {
	case model.WebhookTypeDiscord:
		return &DiscordFormatter{}
	case model.WebhookTypeSlack:
		return &SlackFormatter{}
	case model.WebhookTypeTeams:
		return &TeamsFormatter{}
	case model.WebhookTypeGeneric:
		return &GenericFormatter{}
	default:
		return &GenericFormatter{}
	}
}
