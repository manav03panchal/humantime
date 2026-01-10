package notify

import (
	"encoding/json"

	"github.com/manav03panchal/humantime/internal/model"
)

// DiscordFormatter formats notifications for Discord webhooks.
type DiscordFormatter struct{}

// discordPayload represents a Discord webhook payload.
type discordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []discordEmbed `json:"embeds,omitempty"`
}

// discordEmbed represents a Discord embed.
type discordEmbed struct {
	Title       string               `json:"title,omitempty"`
	Description string               `json:"description,omitempty"`
	Color       int                  `json:"color,omitempty"`
	Fields      []discordEmbedField  `json:"fields,omitempty"`
	Footer      *discordEmbedFooter  `json:"footer,omitempty"`
	Timestamp   string               `json:"timestamp,omitempty"`
}

// discordEmbedField represents a field in a Discord embed.
type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// discordEmbedFooter represents a footer in a Discord embed.
type discordEmbedFooter struct {
	Text string `json:"text"`
}

// Format converts a notification to Discord webhook format.
func (f *DiscordFormatter) Format(n *model.Notification) ([]byte, error) {
	color := n.Color
	if color == 0 {
		color = model.DefaultColorForType(n.Type)
	}

	embed := discordEmbed{
		Title:       n.Title,
		Description: n.Message,
		Color:       color,
		Timestamp:   n.Timestamp.Format("2006-01-02T15:04:05Z"),
		Footer: &discordEmbedFooter{
			Text: "Humantime",
		},
	}

	// Add fields if present
	for key, value := range n.Fields {
		embed.Fields = append(embed.Fields, discordEmbedField{
			Name:   key,
			Value:  value,
			Inline: true,
		})
	}

	payload := discordPayload{
		Embeds: []discordEmbed{embed},
	}

	return json.Marshal(payload)
}

// ContentType returns the content type for Discord webhooks.
func (f *DiscordFormatter) ContentType() string {
	return "application/json"
}
