package notify

import (
	"encoding/json"
	"fmt"

	"github.com/manav03panchal/humantime/internal/model"
)

// TeamsFormatter formats notifications for Microsoft Teams webhooks.
type TeamsFormatter struct{}

// teamsPayload represents a Teams webhook payload (MessageCard format).
type teamsPayload struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor string         `json:"themeColor,omitempty"`
	Summary    string         `json:"summary"`
	Sections   []teamsSection `json:"sections,omitempty"`
}

// teamsSection represents a section in a Teams message.
type teamsSection struct {
	ActivityTitle    string      `json:"activityTitle,omitempty"`
	ActivitySubtitle string      `json:"activitySubtitle,omitempty"`
	Text             string      `json:"text,omitempty"`
	Facts            []teamsFact `json:"facts,omitempty"`
	Markdown         bool        `json:"markdown"`
}

// teamsFact represents a fact (key-value pair) in a Teams section.
type teamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Format converts a notification to Teams webhook format.
func (f *TeamsFormatter) Format(n *model.Notification) ([]byte, error) {
	color := n.Color
	if color == 0 {
		color = model.DefaultColorForType(n.Type)
	}

	section := teamsSection{
		ActivityTitle:    n.Title,
		ActivitySubtitle: fmt.Sprintf("Humantime | %s", n.Timestamp.Format("Jan 2, 3:04 PM")),
		Text:             n.Message,
		Markdown:         true,
	}

	// Add fields as facts
	for key, value := range n.Fields {
		section.Facts = append(section.Facts, teamsFact{
			Name:  key,
			Value: value,
		})
	}

	payload := teamsPayload{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: fmt.Sprintf("%06X", color),
		Summary:    n.Title,
		Sections:   []teamsSection{section},
	}

	return json.Marshal(payload)
}

// ContentType returns the content type for Teams webhooks.
func (f *TeamsFormatter) ContentType() string {
	return "application/json"
}
