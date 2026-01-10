package notify

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/manav03panchal/humantime/internal/model"
)

// GenericFormatter formats notifications for generic webhooks.
type GenericFormatter struct {
	// Template is an optional custom template for the payload.
	Template string
}

// genericPayload is the default payload for generic webhooks.
type genericPayload struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	Timestamp string            `json:"timestamp"`
	Color     int               `json:"color,omitempty"`
}

// Format converts a notification to a generic webhook format.
func (f *GenericFormatter) Format(n *model.Notification) ([]byte, error) {
	color := n.Color
	if color == 0 {
		color = model.DefaultColorForType(n.Type)
	}

	// If template is provided, use it
	if f.Template != "" {
		return f.formatWithTemplate(n)
	}

	// Default format
	payload := genericPayload{
		Type:      string(n.Type),
		Title:     n.Title,
		Message:   n.Message,
		Fields:    n.Fields,
		Timestamp: n.Timestamp.Format("2006-01-02T15:04:05Z"),
		Color:     color,
	}

	return json.Marshal(payload)
}

// formatWithTemplate uses a custom template to format the notification.
func (f *GenericFormatter) formatWithTemplate(n *model.Notification) ([]byte, error) {
	tmpl, err := template.New("webhook").Parse(f.Template)
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"Type":      string(n.Type),
		"Title":     n.Title,
		"Message":   n.Message,
		"Fields":    n.Fields,
		"Timestamp": n.Timestamp,
		"Color":     n.Color,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ContentType returns the content type for generic webhooks.
func (f *GenericFormatter) ContentType() string {
	return "application/json"
}

// NewGenericFormatter creates a new generic formatter with an optional template.
func NewGenericFormatter(template string) *GenericFormatter {
	return &GenericFormatter{Template: template}
}
