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
)
