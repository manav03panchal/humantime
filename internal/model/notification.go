package model

import (
	"time"
)

// NotificationType defines the type of notification.
type NotificationType string

// Notification types.
const (
	NotifyReminder     NotificationType = "reminder"
	NotifyIdle         NotificationType = "idle"
	NotifyBreak        NotificationType = "break"
	NotifyGoal         NotificationType = "goal"
	NotifyDailySummary NotificationType = "daily_summary"
	NotifyEndOfDay     NotificationType = "end_of_day"
	NotifyTest         NotificationType = "test"
)

// Notification represents a notification to be sent.
type Notification struct {
	Type      NotificationType  `json:"type"`
	Title     string            `json:"title"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Color     int               `json:"color,omitempty"` // Hex color for embeds
}

// NewNotification creates a new notification.
func NewNotification(t NotificationType, title, message string) *Notification {
	return &Notification{
		Type:      t,
		Title:     title,
		Message:   message,
		Fields:    make(map[string]string),
		Timestamp: time.Now(),
	}
}

// WithField adds a field to the notification.
func (n *Notification) WithField(key, value string) *Notification {
	if n.Fields == nil {
		n.Fields = make(map[string]string)
	}
	n.Fields[key] = value
	return n
}

// WithColor sets the embed color.
func (n *Notification) WithColor(color int) *Notification {
	n.Color = color
	return n
}

// Notification colors (Discord-compatible hex values).
const (
	ColorSuccess = 0x57F287 // Green
	ColorWarning = 0xFEE75C // Yellow
	ColorInfo    = 0x5865F2 // Blurple
	ColorError   = 0xED4245 // Red
	ColorPrimary = 0x3498DB // Blue
)

// DefaultColorForType returns the default color for a notification type.
func DefaultColorForType(t NotificationType) int {
	switch t {
	case NotifyReminder:
		return ColorWarning
	case NotifyIdle:
		return ColorInfo
	case NotifyBreak:
		return ColorPrimary
	case NotifyGoal:
		return ColorSuccess
	case NotifyDailySummary:
		return ColorInfo
	case NotifyEndOfDay:
		return ColorSuccess
	case NotifyTest:
		return ColorPrimary
	default:
		return ColorInfo
	}
}

// Icon returns an emoji icon for the notification type.
func (n *Notification) Icon() string {
	switch n.Type {
	case NotifyReminder:
		return "bell"
	case NotifyIdle:
		return "pause_button"
	case NotifyBreak:
		return "coffee"
	case NotifyGoal:
		return "dart"
	case NotifyDailySummary:
		return "sunrise"
	case NotifyEndOfDay:
		return "moon"
	case NotifyTest:
		return "test_tube"
	default:
		return "bell"
	}
}

// TypeLabel returns a human-readable label for the notification type.
func (n *Notification) TypeLabel() string {
	switch n.Type {
	case NotifyReminder:
		return "Reminder"
	case NotifyIdle:
		return "Idle Detection"
	case NotifyBreak:
		return "Break Reminder"
	case NotifyGoal:
		return "Goal Progress"
	case NotifyDailySummary:
		return "Daily Summary"
	case NotifyEndOfDay:
		return "End of Day Recap"
	case NotifyTest:
		return "Test Notification"
	default:
		return "Notification"
	}
}
