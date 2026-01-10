package model

import (
	"fmt"
	"time"
)

// PrefixReminder is the database key prefix for reminders.
const PrefixReminder = "reminder"

// Reminder represents a deadline reminder.
type Reminder struct {
	Key          string    `json:"key"`
	Title        string    `json:"title" validate:"required,max=200"`
	Deadline     time.Time `json:"deadline" validate:"required"`
	ProjectSID   string    `json:"project_sid,omitempty" validate:"max=32"`
	RepeatRule   string    `json:"repeat_rule,omitempty"` // "", "daily", "weekly", "monthly"
	NotifyBefore []string  `json:"notify_before"`         // ["1h", "15m"]
	Completed    bool      `json:"completed"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	OwnerKey     string    `json:"owner_key"`
}

// SetKey sets the database key for this reminder.
func (r *Reminder) SetKey(key string) {
	r.Key = key
}

// GetKey returns the database key for this reminder.
func (r *Reminder) GetKey() string {
	return r.Key
}

// IsPending returns true if the reminder is not completed.
func (r *Reminder) IsPending() bool {
	return !r.Completed
}

// IsDue returns true if the reminder deadline has passed.
func (r *Reminder) IsDue() bool {
	return time.Now().After(r.Deadline)
}

// IsDueWithin returns true if the reminder is due within the given duration.
func (r *Reminder) IsDueWithin(d time.Duration) bool {
	return time.Until(r.Deadline) <= d
}

// IsRecurring returns true if the reminder repeats.
func (r *Reminder) IsRecurring() bool {
	return r.RepeatRule != ""
}

// NextDeadline calculates the next deadline for recurring reminders.
func (r *Reminder) NextDeadline() time.Time {
	switch r.RepeatRule {
	case "daily":
		return r.Deadline.AddDate(0, 0, 1)
	case "weekly":
		return r.Deadline.AddDate(0, 0, 7)
	case "monthly":
		return r.Deadline.AddDate(0, 1, 0)
	default:
		return r.Deadline
	}
}

// TimeUntil returns the duration until the deadline.
func (r *Reminder) TimeUntil() time.Duration {
	return time.Until(r.Deadline)
}

// ShortID returns the first 6 characters of the UUID for display.
func (r *Reminder) ShortID() string {
	// Key format: "reminder:uuid"
	if len(r.Key) > 15 {
		return r.Key[9:15] // Skip "reminder:" prefix
	}
	return r.Key
}

// GenerateReminderKey generates a database key for a reminder using UUID.
func GenerateReminderKey(uuid string) string {
	return fmt.Sprintf("%s:%s", PrefixReminder, uuid)
}

// NewReminder creates a new reminder with default notification intervals.
func NewReminder(title string, deadline time.Time, ownerKey string) *Reminder {
	return &Reminder{
		Title:        title,
		Deadline:     deadline,
		NotifyBefore: []string{"1h", "15m"},
		Completed:    false,
		CreatedAt:    time.Now(),
		OwnerKey:     ownerKey,
	}
}

// ValidRepeatRules returns the valid repeat rule options.
func ValidRepeatRules() []string {
	return []string{"", "daily", "weekly", "monthly"}
}

// IsValidRepeatRule checks if a repeat rule is valid.
func IsValidRepeatRule(rule string) bool {
	for _, valid := range ValidRepeatRules() {
		if rule == valid {
			return true
		}
	}
	return false
}
