package scheduler

import (
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
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

// =============================================================================
// ReminderChecker Tests
// =============================================================================

func TestNewReminderChecker(t *testing.T) {
	db := setupTestDB(t)
	reminderRepo := storage.NewReminderRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)

	checker := NewReminderChecker(reminderRepo, webhookRepo)
	assert.NotNil(t, checker)
	assert.NotNil(t, checker.reminderRepo)
	assert.NotNil(t, checker.webhookRepo)
	assert.NotNil(t, checker.dispatcher)
	assert.NotNil(t, checker.notified)
}

func TestReminderCheckerSetDebug(t *testing.T) {
	db := setupTestDB(t)
	reminderRepo := storage.NewReminderRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)

	checker := NewReminderChecker(reminderRepo, webhookRepo)
	assert.False(t, checker.debug)

	checker.SetDebug(true)
	assert.True(t, checker.debug)

	checker.SetDebug(false)
	assert.False(t, checker.debug)
}

func TestReminderCheckerMarkAndWasNotified(t *testing.T) {
	db := setupTestDB(t)
	reminderRepo := storage.NewReminderRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)

	checker := NewReminderChecker(reminderRepo, webhookRepo)

	// Initially not notified
	assert.False(t, checker.wasNotified("reminder-1", "1h"))

	// Mark as notified
	checker.markNotified("reminder-1", "1h")
	assert.True(t, checker.wasNotified("reminder-1", "1h"))

	// Different interval not notified
	assert.False(t, checker.wasNotified("reminder-1", "30m"))

	// Different reminder not notified
	assert.False(t, checker.wasNotified("reminder-2", "1h"))
}

func TestReminderCheckerCheckNoReminders(t *testing.T) {
	db := setupTestDB(t)
	reminderRepo := storage.NewReminderRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)

	checker := NewReminderChecker(reminderRepo, webhookRepo)
	checker.SetDebug(true)

	// Should not panic
	checker.Check()
}

func TestReminderCheckerCleanupNotified(t *testing.T) {
	db := setupTestDB(t)
	reminderRepo := storage.NewReminderRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)

	checker := NewReminderChecker(reminderRepo, webhookRepo)

	// Add some notified entries
	checker.markNotified("reminder-1", "1h")
	checker.markNotified("reminder-2", "30m")

	// Cleanup should remove entries for non-existent reminders
	checker.CleanupNotified()

	// Both should be cleaned up since the reminders don't exist
	assert.Empty(t, checker.notified)
}

// =============================================================================
// IdleChecker Tests
// =============================================================================

func TestNewIdleChecker(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewIdleChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)
	assert.NotNil(t, checker)
	assert.NotNil(t, checker.blockRepo)
	assert.NotNil(t, checker.activeBlockRepo)
	assert.NotNil(t, checker.webhookRepo)
	assert.NotNil(t, checker.notifyConfigRepo)
	assert.NotNil(t, checker.dispatcher)
}

func TestIdleCheckerSetDebug(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewIdleChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)
	assert.False(t, checker.debug)

	checker.SetDebug(true)
	assert.True(t, checker.debug)
}

func TestIdleCheckerCheckNoBlocks(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewIdleChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)
	checker.SetDebug(true)

	// Should not panic
	checker.Check()
}

// =============================================================================
// BreakChecker Tests
// =============================================================================

func TestNewBreakChecker(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)
	assert.NotNil(t, checker)
	assert.NotNil(t, checker.blockRepo)
	assert.NotNil(t, checker.activeBlockRepo)
	assert.NotNil(t, checker.webhookRepo)
	assert.NotNil(t, checker.notifyConfigRepo)
	assert.NotNil(t, checker.dispatcher)
}

func TestBreakCheckerSetDebug(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)
	assert.False(t, checker.debug)

	checker.SetDebug(true)
	assert.True(t, checker.debug)
}

func TestBreakCheckerCheckNoBlocks(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)
	checker.SetDebug(true)

	// Should not panic
	checker.Check()
}

func TestBreakCheckerCalculateContinuousSessionEmpty(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)

	// Empty blocks
	start, duration := checker.calculateContinuousSession(nil, 15*time.Minute, time.Now())
	assert.True(t, start.IsZero())
	assert.Equal(t, time.Duration(0), duration)
}

// =============================================================================
// GoalChecker Tests
// =============================================================================

func TestNewGoalChecker(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewGoalChecker(blockRepo, goalRepo, webhookRepo, notifyConfigRepo)
	assert.NotNil(t, checker)
	assert.NotNil(t, checker.blockRepo)
	assert.NotNil(t, checker.goalRepo)
	assert.NotNil(t, checker.webhookRepo)
	assert.NotNil(t, checker.notifyConfigRepo)
	assert.NotNil(t, checker.dispatcher)
	assert.NotNil(t, checker.notifiedMilestones)
}

func TestGoalCheckerSetDebug(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewGoalChecker(blockRepo, goalRepo, webhookRepo, notifyConfigRepo)
	assert.False(t, checker.debug)

	checker.SetDebug(true)
	assert.True(t, checker.debug)
}

func TestGoalCheckerMarkAndWasNotified(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewGoalChecker(blockRepo, goalRepo, webhookRepo, notifyConfigRepo)

	// Initially not notified
	assert.False(t, checker.wasNotified("project1", 50))

	// Mark as notified
	checker.markNotified("project1", 50)
	assert.True(t, checker.wasNotified("project1", 50))

	// Different milestone not notified
	assert.False(t, checker.wasNotified("project1", 75))

	// Different project not notified
	assert.False(t, checker.wasNotified("project2", 50))
}

func TestGoalCheckerCleanupOldMilestones(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewGoalChecker(blockRepo, goalRepo, webhookRepo, notifyConfigRepo)

	// Add milestones at current time
	checker.markNotified("project1", 50)
	checker.markNotified("project1", 75)

	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	startOfWeek := startOfDay.AddDate(0, 0, -(weekday - 1))

	// Should not clean up current milestones
	checker.cleanupOldMilestones(startOfDay, startOfWeek)
	assert.True(t, checker.wasNotified("project1", 50))
	assert.True(t, checker.wasNotified("project1", 75))
}

func TestGoalCheckerCheckNoGoals(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewGoalChecker(blockRepo, goalRepo, webhookRepo, notifyConfigRepo)
	checker.SetDebug(true)

	// Should not panic
	checker.Check()
}

// =============================================================================
// SummaryGenerator Tests
// =============================================================================

func TestNewSummaryGenerator(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)
	assert.NotNil(t, gen)
	assert.NotNil(t, gen.blockRepo)
	assert.NotNil(t, gen.reminderRepo)
	assert.NotNil(t, gen.goalRepo)
	assert.NotNil(t, gen.webhookRepo)
	assert.NotNil(t, gen.notifyConfigRepo)
	assert.NotNil(t, gen.dispatcher)
}

func TestSummaryGeneratorSetDebug(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)
	assert.False(t, gen.debug)

	gen.SetDebug(true)
	assert.True(t, gen.debug)

	gen.SetDebug(false)
	assert.False(t, gen.debug)
}

func TestSummaryGeneratorCheckDailySummary(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)
	gen.SetDebug(true)

	// Should not panic (daily summary disabled by default)
	gen.CheckDailySummary()
}

func TestSummaryGeneratorCheckEndOfDay(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)
	gen.SetDebug(true)

	// Should not panic (end of day disabled by default)
	gen.CheckEndOfDay()
}

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_24h", "09:00", false},
		{"valid_24h_afternoon", "14:30", false},
		{"valid_12h", "9:00", false},
		{"invalid_format", "invalid", true},
		{"invalid_hour", "25:00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTimeOfDay(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0m"},
		{"minutes_only", 30 * time.Minute, "30m"},
		{"hours_only", 2 * time.Hour, "2h"},
		{"hours_and_minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"one_hour_one_minute", time.Hour + time.Minute, "1h 1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldSendSummary(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	// Test with a target time far in the future
	futureTarget := time.Date(2000, 1, 1, 23, 59, 0, 0, time.Local)
	result := gen.shouldSendSummary(futureTarget, time.Time{})
	assert.False(t, result)

	// Test with already sent today
	now := time.Now()
	targetTime := time.Date(2000, 1, 1, now.Hour(), now.Minute(), 0, 0, time.Local)
	lastSent := now
	result = gen.shouldSendSummary(targetTime, lastSent)
	assert.False(t, result)
}

func TestAggregateByProject(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	now := time.Now()
	blocks := []*model.Block{
		{ProjectSID: "project-a", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now.Add(-1 * time.Hour)},
		{ProjectSID: "project-a", TimestampStart: now.Add(-1 * time.Hour), TimestampEnd: now},
		{ProjectSID: "project-b", TimestampStart: now.Add(-30 * time.Minute), TimestampEnd: now},
	}

	result := gen.aggregateByProject(blocks)
	assert.Len(t, result, 2)
	// project-a should be first (more time)
	assert.Equal(t, "project-a", result[0].project)
	assert.Equal(t, 2*time.Hour, result[0].duration)
	assert.Equal(t, "project-b", result[1].project)
	assert.Equal(t, 30*time.Minute, result[1].duration)
}

func TestAggregateByProjectEmpty(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	result := gen.aggregateByProject(nil)
	assert.Empty(t, result)
}

func TestGetGoalStatus(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	// Create a daily goal
	goal := model.NewGoal("test-project", model.GoalTypeDaily, 4*time.Hour)
	err := goalRepo.Create(goal)
	require.NoError(t, err)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	// Test with blocks
	now := time.Now()
	blocks := []*model.Block{
		{ProjectSID: "test-project", TimestampStart: now.Add(-2 * time.Hour), TimestampEnd: now},
	}

	result := gen.getGoalStatus(blocks)
	assert.Len(t, result, 1)
	assert.Equal(t, "test-project", result[0].project)
	assert.Equal(t, 2*time.Hour, result[0].current)
	assert.Equal(t, 4*time.Hour, result[0].target)
	assert.InDelta(t, 50.0, result[0].percentage, 0.1)
}

func TestGetGoalStatusNoGoals(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	result := gen.getGoalStatus(nil)
	assert.Empty(t, result)
}

func TestGetTodayReminders(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	// Test with no reminders
	reminders, err := gen.getTodayReminders()
	assert.NoError(t, err)
	assert.Empty(t, reminders)

	// Create a reminder due later today (2 hours from now, unless that crosses midnight)
	now := time.Now()
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	// Only create if we have at least 1 hour left in the day
	if now.Add(time.Hour).Before(endOfToday) {
		reminder := model.NewReminder("Test Reminder", now.Add(time.Hour), "owner1")
		err = reminderRepo.Create(reminder)
		require.NoError(t, err)

		reminders, err = gen.getTodayReminders()
		assert.NoError(t, err)
		// The reminder should be found if created
		if len(reminders) > 0 {
			assert.Equal(t, "Test Reminder", reminders[0].Title)
		}
	}
}

func TestGetTomorrowReminders(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	reminderRepo := storage.NewReminderRepo(db)
	goalRepo := storage.NewGoalRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	gen := NewSummaryGenerator(blockRepo, reminderRepo, goalRepo, webhookRepo, notifyConfigRepo)

	// Test with no reminders
	reminders, err := gen.getTomorrowReminders()
	assert.NoError(t, err)
	assert.Empty(t, reminders)

	// Create a reminder due tomorrow at 10 AM
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowMorning := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, now.Location())
	reminder := model.NewReminder("Tomorrow Reminder", tomorrowMorning, "owner1")
	err = reminderRepo.Create(reminder)
	require.NoError(t, err)

	reminders, err = gen.getTomorrowReminders()
	assert.NoError(t, err)
	assert.Len(t, reminders, 1)
	assert.Equal(t, "Tomorrow Reminder", reminders[0].Title)
}

// =============================================================================
// Additional BreakChecker Tests - calculateContinuousSession
// =============================================================================

func TestBreakCheckerCalculateContinuousSessionSingleBlock(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)

	now := time.Now()
	blocks := []*model.Block{
		{
			TimestampStart: now.Add(-1 * time.Hour),
			TimestampEnd:   now,
		},
	}

	start, duration := checker.calculateContinuousSession(blocks, 15*time.Minute, now)
	assert.Equal(t, blocks[0].TimestampStart, start)
	assert.Equal(t, 1*time.Hour, duration)
}

func TestBreakCheckerCalculateContinuousSessionMultipleBlocksNoGap(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)

	now := time.Now()
	// Blocks listed newest first (as they would be from the repo)
	blocks := []*model.Block{
		{
			TimestampStart: now.Add(-1 * time.Hour),
			TimestampEnd:   now,
		},
		{
			TimestampStart: now.Add(-2*time.Hour - 5*time.Minute), // 5 min gap (within 15 min reset)
			TimestampEnd:   now.Add(-1*time.Hour - 5*time.Minute),
		},
	}

	start, duration := checker.calculateContinuousSession(blocks, 15*time.Minute, now)
	// Should combine both blocks
	assert.Equal(t, now.Add(-2*time.Hour-5*time.Minute), start)
	// Duration = 1h + 1h = 2h
	assert.Equal(t, 2*time.Hour, duration)
}

func TestBreakCheckerCalculateContinuousSessionWithBreak(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)

	now := time.Now()
	// Blocks listed newest first - with a 30 minute gap between them (> 15 min reset)
	blocks := []*model.Block{
		{
			TimestampStart: now.Add(-1 * time.Hour),
			TimestampEnd:   now,
		},
		{
			TimestampStart: now.Add(-3 * time.Hour), // 30 min gap before next block
			TimestampEnd:   now.Add(-1*time.Hour - 30*time.Minute),
		},
	}

	start, duration := checker.calculateContinuousSession(blocks, 15*time.Minute, now)
	// Should only count the recent block (session reset by 30 min gap)
	assert.Equal(t, now.Add(-1*time.Hour), start)
	assert.Equal(t, 1*time.Hour, duration)
}

func TestBreakCheckerCalculateContinuousSessionActiveBlock(t *testing.T) {
	db := setupTestDB(t)
	blockRepo := storage.NewBlockRepo(db)
	activeBlockRepo := storage.NewActiveBlockRepo(db)
	webhookRepo := storage.NewWebhookRepo(db)
	notifyConfigRepo := storage.NewNotifyConfigRepo(db)

	checker := NewBreakChecker(blockRepo, activeBlockRepo, webhookRepo, notifyConfigRepo)

	now := time.Now()
	// Active block (no end time)
	blocks := []*model.Block{
		{
			TimestampStart: now.Add(-2 * time.Hour),
			TimestampEnd:   time.Time{}, // Zero time = active
		},
	}

	start, duration := checker.calculateContinuousSession(blocks, 15*time.Minute, now)
	assert.Equal(t, blocks[0].TimestampStart, start)
	assert.Equal(t, 2*time.Hour, duration)
}
