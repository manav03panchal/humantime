package model

import (
	"time"
)

// KeyNotifyConfig is the database key for notification configuration.
const KeyNotifyConfig = "config:notify"

// NotifyConfig holds notification preferences.
type NotifyConfig struct {
	IdleAfter      time.Duration   `json:"idle_after"`       // Default: 30m
	BreakAfter     time.Duration   `json:"break_after"`      // Default: 2h
	BreakReset     time.Duration   `json:"break_reset"`      // Default: 15m
	GoalMilestones []int           `json:"goal_milestones"`  // Default: [50, 75, 100]
	DailySummaryAt string          `json:"daily_summary_at"` // Default: "09:00"
	EndOfDayAt     string          `json:"end_of_day_at"`    // Default: "18:00"
	Enabled        map[string]bool `json:"enabled"`          // Per-type toggles
}

// DefaultNotifyConfig returns the default notification configuration.
func DefaultNotifyConfig() *NotifyConfig {
	return &NotifyConfig{
		IdleAfter:      30 * time.Minute,
		BreakAfter:     2 * time.Hour,
		BreakReset:     15 * time.Minute,
		GoalMilestones: []int{50, 75, 100},
		DailySummaryAt: "09:00",
		EndOfDayAt:     "18:00",
		Enabled: map[string]bool{
			"idle":          true,
			"break":         true,
			"goal":          true,
			"daily_summary": true,
			"end_of_day":    true,
			"reminder":      true,
		},
	}
}

// IsTypeEnabled checks if a notification type is enabled.
func (c *NotifyConfig) IsTypeEnabled(notifyType string) bool {
	if c.Enabled == nil {
		return true // Default to enabled
	}
	enabled, exists := c.Enabled[notifyType]
	if !exists {
		return true // Default to enabled if not specified
	}
	return enabled
}

// SetTypeEnabled sets whether a notification type is enabled.
func (c *NotifyConfig) SetTypeEnabled(notifyType string, enabled bool) {
	if c.Enabled == nil {
		c.Enabled = make(map[string]bool)
	}
	c.Enabled[notifyType] = enabled
}

// Clone creates a deep copy of the config.
func (c *NotifyConfig) Clone() *NotifyConfig {
	clone := &NotifyConfig{
		IdleAfter:      c.IdleAfter,
		BreakAfter:     c.BreakAfter,
		BreakReset:     c.BreakReset,
		DailySummaryAt: c.DailySummaryAt,
		EndOfDayAt:     c.EndOfDayAt,
	}

	// Copy milestones
	if c.GoalMilestones != nil {
		clone.GoalMilestones = make([]int, len(c.GoalMilestones))
		copy(clone.GoalMilestones, c.GoalMilestones)
	}

	// Copy enabled map
	if c.Enabled != nil {
		clone.Enabled = make(map[string]bool)
		for k, v := range c.Enabled {
			clone.Enabled[k] = v
		}
	}

	return clone
}

// Validate checks if the configuration values are valid.
func (c *NotifyConfig) Validate() error {
	// IdleAfter: 5m to 4h
	if c.IdleAfter < 5*time.Minute || c.IdleAfter > 4*time.Hour {
		return &ValidationError{Field: "idle_after", Message: "must be between 5m and 4h"}
	}

	// BreakAfter: 0 (disabled) or 30m to 8h
	if c.BreakAfter != 0 && (c.BreakAfter < 30*time.Minute || c.BreakAfter > 8*time.Hour) {
		return &ValidationError{Field: "break_after", Message: "must be 0 (disabled) or between 30m and 8h"}
	}

	// GoalMilestones: each 1-100
	for _, m := range c.GoalMilestones {
		if m < 1 || m > 100 {
			return &ValidationError{Field: "goal_milestones", Message: "each milestone must be between 1 and 100"}
		}
	}

	return nil
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
