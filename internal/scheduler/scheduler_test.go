package scheduler

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *storage.DB {
	db, err := storage.Open(storage.Options{InMemory: true})
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestNewScheduler(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)
	assert.NotNil(t, scheduler)
	assert.NotNil(t, scheduler.cron)
}

func TestSchedulerSetDebug(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)

	scheduler.SetDebug(true)
	assert.True(t, scheduler.debug)

	scheduler.SetDebug(false)
	assert.False(t, scheduler.debug)
}

func TestSchedulerStartStop(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)

	err := scheduler.Start()
	assert.NoError(t, err)

	// Wait a bit then stop
	time.Sleep(100 * time.Millisecond)
	scheduler.Stop()
}

func TestSchedulerStartStopWithDebug(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)
	scheduler.SetDebug(true)

	err := scheduler.Start()
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	scheduler.Stop()
}

func TestSchedulerAddRemoveJob(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)

	// Add a job
	executed := false
	id, err := scheduler.AddJob("@every 1s", func() {
		executed = true
	})
	assert.NoError(t, err)

	entries := scheduler.Entries()
	assert.Len(t, entries, 1)

	// Remove the job
	scheduler.RemoveJob(id)

	entries = scheduler.Entries()
	assert.Len(t, entries, 0)
	_ = executed
}

func TestSchedulerNextRun(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)

	// No entries
	next := scheduler.NextRun()
	assert.True(t, next.IsZero())

	// Add an entry and start scheduler (entries get next run time after start)
	_, err := scheduler.AddJob("@every 1m", func() {})
	require.NoError(t, err)

	// Start the scheduler to calculate next run times
	scheduler.cron.Start()
	defer scheduler.cron.Stop()

	// Give it a moment to initialize
	time.Sleep(10 * time.Millisecond)

	entries := scheduler.Entries()
	assert.Greater(t, len(entries), 0)
}

func TestSchedulerSetCheckers(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)

	// Set reminder checker
	reminderChecker := &ReminderChecker{}
	scheduler.SetReminderChecker(reminderChecker)
	assert.Equal(t, reminderChecker, scheduler.reminderChecker)

	// Set with debug mode
	scheduler.SetDebug(true)

	idleChecker := &IdleChecker{}
	scheduler.SetIdleChecker(idleChecker)
	assert.Equal(t, idleChecker, scheduler.idleChecker)

	breakChecker := &BreakChecker{}
	scheduler.SetBreakChecker(breakChecker)
	assert.Equal(t, breakChecker, scheduler.breakChecker)

	goalChecker := &GoalChecker{}
	scheduler.SetGoalChecker(goalChecker)
	assert.Equal(t, goalChecker, scheduler.goalChecker)

	summaryGen := &SummaryGenerator{}
	scheduler.SetSummaryGenerator(summaryGen)
	assert.Equal(t, summaryGen, scheduler.summaryGenerator)
}

func TestSchedulerChecksWithNilCheckers(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)

	// Should not panic with nil checkers
	scheduler.checkReminders()
	scheduler.checkIdle()
	scheduler.checkBreak()
	scheduler.checkGoalProgress()
	scheduler.checkDailySummary()
	scheduler.checkEndOfDay()
}

func TestSchedulerRunMinuteChecks(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)
	scheduler.lastCheck = time.Now()

	// Should not panic
	scheduler.runMinuteChecks()
}

func TestSchedulerRunMinuteChecksAfterSleep(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)
	scheduler.SetDebug(true)

	// Simulate a sleep of more than an hour
	scheduler.lastCheck = time.Now().Add(-2 * time.Hour)

	// Should skip stale checks
	scheduler.runMinuteChecks()
}

func TestSchedulerRunFiveMinuteChecks(t *testing.T) {
	db := setupTestDB(t)
	scheduler := NewScheduler(db)
	scheduler.SetDebug(true)

	// Should not panic
	scheduler.runFiveMinuteChecks()
}
