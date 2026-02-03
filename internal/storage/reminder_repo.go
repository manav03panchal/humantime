package storage

import (
	"time"

	"github.com/google/uuid"
	"github.com/manav03panchal/humantime/internal/model"
)

// ReminderRepo provides operations for Reminder entities.
type ReminderRepo struct {
	db *DB
}

// NewReminderRepo creates a new reminder repository.
func NewReminderRepo(db *DB) *ReminderRepo {
	return &ReminderRepo{db: db}
}

// Create creates a new reminder with a generated key.
func (r *ReminderRepo) Create(reminder *model.Reminder) error {
	if reminder.Key == "" {
		reminder.Key = model.GenerateReminderKey(uuid.New().String())
	}
	if reminder.CreatedAt.IsZero() {
		reminder.CreatedAt = time.Now()
	}
	if reminder.NotifyBefore == nil {
		reminder.NotifyBefore = []string{"1h", "15m"}
	}
	return r.db.Set(reminder)
}

// Get retrieves a reminder by key.
func (r *ReminderRepo) Get(key string) (*model.Reminder, error) {
	reminder := &model.Reminder{}
	if err := r.db.Get(key, reminder); err != nil {
		return nil, err
	}
	return reminder, nil
}

// GetByShortID retrieves a reminder by short ID prefix match.
// Uses filtered iteration to avoid loading all reminders into memory.
func (r *ReminderRepo) GetByShortID(shortID string) (*model.Reminder, error) {
	matches, err := GetFilteredByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	}, func(rem *model.Reminder) bool {
		return rem.ShortID() == shortID || len(rem.Key) > len(model.PrefixReminder)+1 &&
			len(shortID) <= len(rem.Key)-len(model.PrefixReminder)-1 &&
			rem.Key[len(model.PrefixReminder)+1:len(model.PrefixReminder)+1+len(shortID)] == shortID
	}, 0)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, ErrKeyNotFound
	}
	if len(matches) > 1 {
		return nil, &AmbiguousMatchError{Matches: len(matches)}
	}
	return matches[0], nil
}

// AmbiguousMatchError is returned when multiple reminders match a short ID.
type AmbiguousMatchError struct {
	Matches int
}

func (e *AmbiguousMatchError) Error() string {
	return "multiple reminders match the given ID"
}

// GetByTitle retrieves a reminder by exact title match.
// Uses filtered iteration with early termination to avoid loading all reminders.
func (r *ReminderRepo) GetByTitle(title string) (*model.Reminder, error) {
	matches, err := GetFilteredByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	}, func(rem *model.Reminder) bool {
		return rem.Title == title
	}, 1) // Limit to 1 - stop after first match
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, ErrKeyNotFound
	}
	return matches[0], nil
}

// List retrieves all reminders.
func (r *ReminderRepo) List() ([]*model.Reminder, error) {
	return GetAllByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	})
}

// ListPending retrieves all pending (not completed) reminders.
// Uses filtered iteration to avoid loading all reminders into memory.
func (r *ReminderRepo) ListPending() ([]*model.Reminder, error) {
	return GetFilteredByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	}, func(rem *model.Reminder) bool {
		return rem.IsPending()
	}, 0)
}

// ListDue retrieves reminders due within the given duration.
// Uses filtered iteration to avoid loading all reminders into memory.
func (r *ReminderRepo) ListDue(within time.Duration) ([]*model.Reminder, error) {
	return GetFilteredByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	}, func(rem *model.Reminder) bool {
		return rem.IsPending() && rem.IsDueWithin(within)
	}, 0)
}

// ListByProject retrieves all reminders for a project.
// Uses filtered iteration to avoid loading all reminders into memory.
func (r *ReminderRepo) ListByProject(projectSID string) ([]*model.Reminder, error) {
	return GetFilteredByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	}, func(rem *model.Reminder) bool {
		return rem.ProjectSID == projectSID
	}, 0)
}

// Update updates an existing reminder.
func (r *ReminderRepo) Update(reminder *model.Reminder) error {
	return r.db.Set(reminder)
}

// Delete removes a reminder by key.
func (r *ReminderRepo) Delete(key string) error {
	return r.db.Delete(key)
}

// MarkComplete marks a reminder as completed.
func (r *ReminderRepo) MarkComplete(key string) error {
	reminder, err := r.Get(key)
	if err != nil {
		return err
	}

	reminder.Completed = true
	reminder.CompletedAt = time.Now()

	return r.db.Set(reminder)
}

// CreateNextRecurrence creates the next occurrence for a recurring reminder.
func (r *ReminderRepo) CreateNextRecurrence(reminder *model.Reminder) (*model.Reminder, error) {
	if !reminder.IsRecurring() {
		return nil, nil
	}

	next := &model.Reminder{
		Key:          model.GenerateReminderKey(uuid.New().String()),
		Title:        reminder.Title,
		Deadline:     reminder.NextDeadline(),
		ProjectSID:   reminder.ProjectSID,
		RepeatRule:   reminder.RepeatRule,
		NotifyBefore: reminder.NotifyBefore,
		Completed:    false,
		CreatedAt:    time.Now(),
		OwnerKey:     reminder.OwnerKey,
	}

	if err := r.db.Set(next); err != nil {
		return nil, err
	}
	return next, nil
}

// Exists checks if a reminder exists.
func (r *ReminderRepo) Exists(key string) (bool, error) {
	return r.db.Exists(key)
}
