// Package model defines the domain models for Humantime.
package model

// Model is the interface that all database models must implement.
type Model interface {
	// SetKey sets the database key for this model.
	SetKey(key string)
	// GetKey returns the database key for this model.
	GetKey() string
}

// KeyPrefix constants for database key generation.
const (
	PrefixBlock       = "block"
	PrefixProject     = "project"
	PrefixTask        = "task"
	PrefixGoal        = "goal"
	KeyActiveBlock    = "activeblock"
	KeyConfig         = "config"
	// New prefixes for reminders daemon feature
	// PrefixReminder = "reminder" - defined in reminder.go
	// PrefixWebhook  = "webhook"  - defined in webhook.go
	// KeyNotifyConfig = "config:notify" - defined in notify_config.go
)
