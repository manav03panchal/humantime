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
func (r *ReminderRepo) GetByShortID(shortID string) (*model.Reminder, error) {
	reminders, err := r.List()
	if err != nil {
		return nil, err
	}

	var matches []*model.Reminder
	for _, rem := range reminders {
		if rem.ShortID() == shortID || len(rem.Key) > len(model.PrefixReminder)+1 &&
		   len(shortID) <= len(rem.Key)-len(model.PrefixReminder)-1 &&
		   rem.Key[len(model.PrefixReminder)+1:len(model.PrefixReminder)+1+len(shortID)] == shortID {
			matches = append(matches, rem)
		}
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
func (r *ReminderRepo) GetByTitle(title string) (*model.Reminder, error) {
	reminders, err := r.List()
	if err != nil {
		return nil, err
	}

	for _, rem := range reminders {
		if rem.Title == title {
			return rem, nil
		}
	}
	return nil, ErrKeyNotFound
}

// List retrieves all reminders.
func (r *ReminderRepo) List() ([]*model.Reminder, error) {
	return GetAllByPrefix(r.db, model.PrefixReminder+":", func() *model.Reminder {
		return &model.Reminder{}
	})
}

// ListPending retrieves all pending (not completed) reminders.
func (r *ReminderRepo) ListPending() ([]*model.Reminder, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}

	var pending []*model.Reminder
	for _, rem := range all {
		if rem.IsPending() {
			pending = append(pending, rem)
		}
	}
	return pending, nil
}

// ListDue retrieves reminders due within the given duration.
func (r *ReminderRepo) ListDue(within time.Duration) ([]*model.Reminder, error) {
	pending, err := r.ListPending()
	if err != nil {
		return nil, err
	}

	var due []*model.Reminder
	for _, rem := range pending {
		if rem.IsDueWithin(within) {
			due = append(due, rem)
		}
	}
	return due, nil
}

// ListByProject retrieves all reminders for a project.
func (r *ReminderRepo) ListByProject(projectSID string) ([]*model.Reminder, error) {
	all, err := r.List()
	if err != nil {
		return nil, err
	}

	var result []*model.Reminder
	for _, rem := range all {
		if rem.ProjectSID == projectSID {
			result = append(result, rem)
		}
	}
	return result, nil
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
