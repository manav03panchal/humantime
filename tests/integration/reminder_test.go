package integration

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Reminder Creation Tests
// =============================================================================

func TestReminderCreation(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewReminderRepo(db)

	t.Run("creates reminder with all fields", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Submit invoice", deadline, "user1")
		reminder.ProjectSID = "clientwork"
		reminder.RepeatRule = "weekly"

		err := repo.Create(reminder)
		require.NoError(t, err)
		assert.NotEmpty(t, reminder.Key)

		retrieved, err := repo.Get(reminder.Key)
		require.NoError(t, err)

		assert.Equal(t, "Submit invoice", retrieved.Title)
		assert.Equal(t, "clientwork", retrieved.ProjectSID)
		assert.Equal(t, "weekly", retrieved.RepeatRule)
		assert.Equal(t, []string{"1h", "15m"}, retrieved.NotifyBefore)
		assert.False(t, retrieved.Completed)
	})

	t.Run("creates reminder with default notify intervals", func(t *testing.T) {
		deadline := time.Now().Add(1 * time.Hour)
		reminder := model.NewReminder("Quick task", deadline, "user1")

		err := repo.Create(reminder)
		require.NoError(t, err)

		retrieved, err := repo.Get(reminder.Key)
		require.NoError(t, err)

		assert.Equal(t, []string{"1h", "15m"}, retrieved.NotifyBefore)
	})

	t.Run("generates unique keys", func(t *testing.T) {
		deadline := time.Now().Add(2 * time.Hour)
		r1 := model.NewReminder("Reminder 1", deadline, "user1")
		r2 := model.NewReminder("Reminder 2", deadline, "user1")

		require.NoError(t, repo.Create(r1))
		require.NoError(t, repo.Create(r2))

		assert.NotEqual(t, r1.Key, r2.Key)
	})
}

// =============================================================================
// Reminder Listing Tests
// =============================================================================

func TestReminderListing(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewReminderRepo(db)

	t.Run("lists empty reminders", func(t *testing.T) {
		reminders, err := repo.List()
		require.NoError(t, err)
		assert.Empty(t, reminders)
	})

	t.Run("lists all reminders", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		require.NoError(t, repo.Create(model.NewReminder("Task 1", deadline, "user1")))
		require.NoError(t, repo.Create(model.NewReminder("Task 2", deadline, "user1")))
		require.NoError(t, repo.Create(model.NewReminder("Task 3", deadline, "user1")))

		reminders, err := repo.List()
		require.NoError(t, err)
		assert.Len(t, reminders, 3)
	})

	t.Run("lists pending reminders only", func(t *testing.T) {
		db := setupTestDB(t) // Fresh db
		repo := storage.NewReminderRepo(db)

		deadline := time.Now().Add(24 * time.Hour)
		r1 := model.NewReminder("Pending 1", deadline, "user1")
		r2 := model.NewReminder("Pending 2", deadline, "user1")
		r3 := model.NewReminder("Completed", deadline, "user1")
		r3.Completed = true

		require.NoError(t, repo.Create(r1))
		require.NoError(t, repo.Create(r2))
		require.NoError(t, repo.Create(r3))

		pending, err := repo.ListPending()
		require.NoError(t, err)
		assert.Len(t, pending, 2)

		for _, p := range pending {
			assert.False(t, p.Completed)
		}
	})

	t.Run("lists due reminders", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewReminderRepo(db)

		now := time.Now()
		r1 := model.NewReminder("Due soon", now.Add(30*time.Minute), "user1")
		r2 := model.NewReminder("Due later", now.Add(2*time.Hour), "user1")
		r3 := model.NewReminder("Due much later", now.Add(24*time.Hour), "user1")

		require.NoError(t, repo.Create(r1))
		require.NoError(t, repo.Create(r2))
		require.NoError(t, repo.Create(r3))

		due, err := repo.ListDue(1 * time.Hour)
		require.NoError(t, err)
		assert.Len(t, due, 1)
		assert.Equal(t, "Due soon", due[0].Title)
	})

	t.Run("lists reminders by project", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewReminderRepo(db)

		deadline := time.Now().Add(24 * time.Hour)
		r1 := model.NewReminder("Project A task", deadline, "user1")
		r1.ProjectSID = "projecta"
		r2 := model.NewReminder("Project B task", deadline, "user1")
		r2.ProjectSID = "projectb"
		r3 := model.NewReminder("Project A task 2", deadline, "user1")
		r3.ProjectSID = "projecta"

		require.NoError(t, repo.Create(r1))
		require.NoError(t, repo.Create(r2))
		require.NoError(t, repo.Create(r3))

		projectA, err := repo.ListByProject("projecta")
		require.NoError(t, err)
		assert.Len(t, projectA, 2)

		projectB, err := repo.ListByProject("projectb")
		require.NoError(t, err)
		assert.Len(t, projectB, 1)
	})
}

// =============================================================================
// Reminder Completion Tests
// =============================================================================

func TestReminderCompletion(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewReminderRepo(db)

	t.Run("marks reminder as complete", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Complete me", deadline, "user1")
		require.NoError(t, repo.Create(reminder))

		err := repo.MarkComplete(reminder.Key)
		require.NoError(t, err)

		retrieved, err := repo.Get(reminder.Key)
		require.NoError(t, err)

		assert.True(t, retrieved.Completed)
		assert.False(t, retrieved.CompletedAt.IsZero())
	})

	t.Run("completed reminders excluded from pending", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewReminderRepo(db)

		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Will complete", deadline, "user1")
		require.NoError(t, repo.Create(reminder))

		pending, err := repo.ListPending()
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		require.NoError(t, repo.MarkComplete(reminder.Key))

		pending, err = repo.ListPending()
		require.NoError(t, err)
		assert.Len(t, pending, 0)
	})
}

// =============================================================================
// Recurring Reminder Tests
// =============================================================================

func TestRecurringReminders(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewReminderRepo(db)

	t.Run("creates next daily occurrence", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Daily standup", deadline, "user1")
		reminder.RepeatRule = "daily"
		require.NoError(t, repo.Create(reminder))

		next, err := repo.CreateNextRecurrence(reminder)
		require.NoError(t, err)
		require.NotNil(t, next)

		assert.Equal(t, reminder.Title, next.Title)
		assert.Equal(t, reminder.RepeatRule, next.RepeatRule)
		assert.NotEqual(t, reminder.Key, next.Key)

		// Next deadline should be 1 day later
		expectedDeadline := reminder.Deadline.AddDate(0, 0, 1)
		assert.Equal(t, expectedDeadline.Unix(), next.Deadline.Unix())
	})

	t.Run("creates next weekly occurrence", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Weekly report", deadline, "user1")
		reminder.RepeatRule = "weekly"
		require.NoError(t, repo.Create(reminder))

		next, err := repo.CreateNextRecurrence(reminder)
		require.NoError(t, err)
		require.NotNil(t, next)

		expectedDeadline := reminder.Deadline.AddDate(0, 0, 7)
		assert.Equal(t, expectedDeadline.Unix(), next.Deadline.Unix())
	})

	t.Run("creates next monthly occurrence", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Monthly invoice", deadline, "user1")
		reminder.RepeatRule = "monthly"
		require.NoError(t, repo.Create(reminder))

		next, err := repo.CreateNextRecurrence(reminder)
		require.NoError(t, err)
		require.NotNil(t, next)

		expectedDeadline := reminder.Deadline.AddDate(0, 1, 0)
		assert.Equal(t, expectedDeadline.Unix(), next.Deadline.Unix())
	})

	t.Run("non-recurring reminder returns nil", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("One-time task", deadline, "user1")
		require.NoError(t, repo.Create(reminder))

		next, err := repo.CreateNextRecurrence(reminder)
		require.NoError(t, err)
		assert.Nil(t, next)
	})
}

// =============================================================================
// Reminder Deletion Tests
// =============================================================================

func TestReminderDeletion(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewReminderRepo(db)

	t.Run("deletes existing reminder", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Delete me", deadline, "user1")
		require.NoError(t, repo.Create(reminder))

		exists, err := repo.Exists(reminder.Key)
		require.NoError(t, err)
		assert.True(t, exists)

		err = repo.Delete(reminder.Key)
		require.NoError(t, err)

		exists, err = repo.Exists(reminder.Key)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("delete non-existent reminder does not error", func(t *testing.T) {
		err := repo.Delete("reminder:nonexistent")
		assert.NoError(t, err)
	})
}

// =============================================================================
// Short ID Lookup Tests
// =============================================================================

func TestReminderShortIDLookup(t *testing.T) {
	db := setupTestDB(t)
	repo := storage.NewReminderRepo(db)

	t.Run("finds reminder by short ID", func(t *testing.T) {
		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Find by short", deadline, "user1")
		require.NoError(t, repo.Create(reminder))

		shortID := reminder.ShortID()
		found, err := repo.GetByShortID(shortID)
		require.NoError(t, err)
		assert.Equal(t, reminder.Key, found.Key)
	})

	t.Run("finds reminder by title", func(t *testing.T) {
		db := setupTestDB(t)
		repo := storage.NewReminderRepo(db)

		deadline := time.Now().Add(24 * time.Hour)
		reminder := model.NewReminder("Unique Title", deadline, "user1")
		require.NoError(t, repo.Create(reminder))

		found, err := repo.GetByTitle("Unique Title")
		require.NoError(t, err)
		assert.Equal(t, reminder.Key, found.Key)
	})

	t.Run("returns error for non-existent short ID", func(t *testing.T) {
		_, err := repo.GetByShortID("nonexistent")
		assert.Error(t, err)
	})
}

// =============================================================================
// Reminder Model Tests
// =============================================================================

func TestReminderModel(t *testing.T) {
	t.Run("IsPending returns correct value", func(t *testing.T) {
		r := &model.Reminder{Completed: false}
		assert.True(t, r.IsPending())

		r.Completed = true
		assert.False(t, r.IsPending())
	})

	t.Run("IsDue returns correct value", func(t *testing.T) {
		r := &model.Reminder{Deadline: time.Now().Add(-1 * time.Hour)}
		assert.True(t, r.IsDue())

		r.Deadline = time.Now().Add(1 * time.Hour)
		assert.False(t, r.IsDue())
	})

	t.Run("IsDueWithin returns correct value", func(t *testing.T) {
		r := &model.Reminder{Deadline: time.Now().Add(30 * time.Minute)}
		assert.True(t, r.IsDueWithin(1*time.Hour))
		assert.False(t, r.IsDueWithin(15*time.Minute))
	})

	t.Run("IsRecurring returns correct value", func(t *testing.T) {
		r := &model.Reminder{RepeatRule: ""}
		assert.False(t, r.IsRecurring())

		r.RepeatRule = "daily"
		assert.True(t, r.IsRecurring())
	})

	t.Run("NextDeadline calculates correctly", func(t *testing.T) {
		now := time.Now()

		daily := &model.Reminder{Deadline: now, RepeatRule: "daily"}
		assert.Equal(t, now.AddDate(0, 0, 1).Unix(), daily.NextDeadline().Unix())

		weekly := &model.Reminder{Deadline: now, RepeatRule: "weekly"}
		assert.Equal(t, now.AddDate(0, 0, 7).Unix(), weekly.NextDeadline().Unix())

		monthly := &model.Reminder{Deadline: now, RepeatRule: "monthly"}
		assert.Equal(t, now.AddDate(0, 1, 0).Unix(), monthly.NextDeadline().Unix())

		// Non-recurring returns same deadline
		oneTime := &model.Reminder{Deadline: now, RepeatRule: ""}
		assert.Equal(t, now.Unix(), oneTime.NextDeadline().Unix())
	})

	t.Run("validates repeat rules", func(t *testing.T) {
		assert.True(t, model.IsValidRepeatRule(""))
		assert.True(t, model.IsValidRepeatRule("daily"))
		assert.True(t, model.IsValidRepeatRule("weekly"))
		assert.True(t, model.IsValidRepeatRule("monthly"))
		assert.False(t, model.IsValidRepeatRule("yearly"))
		assert.False(t, model.IsValidRepeatRule("invalid"))
	})
}
