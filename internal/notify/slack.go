package notify

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/manav03panchal/humantime/internal/model"
)

// SlackFormatter formats notifications for Slack webhooks.
type SlackFormatter struct{}

// slackPayload represents a Slack webhook payload.
type slackPayload struct {
	Text        string         `json:"text,omitempty"`
	Blocks      []slackBlock   `json:"blocks,omitempty"`
	Attachments []slackAttach  `json:"attachments,omitempty"`
}

// slackBlock represents a Slack block.
type slackBlock struct {
	Type   string           `json:"type"`
	Text   *slackBlockText  `json:"text,omitempty"`
	Fields []slackBlockText `json:"fields,omitempty"`
}

// slackBlockText represents text in a Slack block.
type slackBlockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// slackAttach represents a Slack attachment (for color).
type slackAttach struct {
	Color    string `json:"color,omitempty"`
	Fallback string `json:"fallback,omitempty"`
}

// Format converts a notification to Slack webhook format.
func (f *SlackFormatter) Format(n *model.Notification) ([]byte, error) {
	color := n.Color
	if color == 0 {
		color = model.DefaultColorForType(n.Type)
	}

	// Build the header block
	headerText := fmt.Sprintf("*%s*", n.Title)

	blocks := []slackBlock{
		{
			Type: "header",
			Text: &slackBlockText{
				Type: "plain_text",
				Text: n.Title,
			},
		},
		{
			Type: "section",
			Text: &slackBlockText{
				Type: "mrkdwn",
				Text: n.Message,
			},
		},
	}

	// Add fields if present
	if len(n.Fields) > 0 {
		var fieldTexts []slackBlockText
		for key, value := range n.Fields {
			fieldTexts = append(fieldTexts, slackBlockText{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*%s*\n%s", key, value),
			})
		}
		blocks = append(blocks, slackBlock{
			Type:   "section",
			Fields: fieldTexts,
		})
	}

	// Add context block with timestamp
	blocks = append(blocks, slackBlock{
		Type: "context",
		Text: &slackBlockText{
			Type: "mrkdwn",
			Text: fmt.Sprintf("Humantime | %s", n.Timestamp.Format("Jan 2, 3:04 PM")),
		},
	})

	payload := slackPayload{
		Text:   headerText, // Fallback text
		Blocks: blocks,
		Attachments: []slackAttach{
			{
				Color:    colorToHex(color),
				Fallback: n.Title,
			},
		},
	}

	return json.Marshal(payload)
}

// ContentType returns the content type for Slack webhooks.
func (f *SlackFormatter) ContentType() string {
	return "application/json"
}

// colorToHex converts an integer color to hex string.
func colorToHex(color int) string {
	return fmt.Sprintf("#%06X", color)
}

// slackEscape escapes special characters for Slack mrkdwn.
func slackEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
