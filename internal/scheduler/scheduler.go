// Package scheduler provides cron-based task scheduling for the daemon.
package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/manav03panchal/humantime/internal/storage"
)

// Scheduler manages scheduled tasks using cron.
type Scheduler struct {
	cron             *cron.Cron
	db               *storage.DB
	debug            bool
	lastCheck        time.Time
	mu               sync.Mutex
	reminderChecker  *ReminderChecker
	idleChecker      *IdleChecker
	breakChecker     *BreakChecker
	goalChecker      *GoalChecker
	summaryGenerator *SummaryGenerator
}

// NewScheduler creates a new scheduler.
func NewScheduler(db *storage.DB) *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
		db:   db,
	}
}

// SetDebug enables debug output.
func (s *Scheduler) SetDebug(debug bool) {
	s.debug = debug
	if s.reminderChecker != nil {
		s.reminderChecker.SetDebug(debug)
	}
	if s.idleChecker != nil {
		s.idleChecker.SetDebug(debug)
	}
	if s.breakChecker != nil {
		s.breakChecker.SetDebug(debug)
	}
	if s.goalChecker != nil {
		s.goalChecker.SetDebug(debug)
	}
	if s.summaryGenerator != nil {
		s.summaryGenerator.SetDebug(debug)
	}
}

// SetReminderChecker sets the reminder checker.
func (s *Scheduler) SetReminderChecker(checker *ReminderChecker) {
	s.reminderChecker = checker
	if s.debug {
		checker.SetDebug(s.debug)
	}
}

// SetIdleChecker sets the idle checker.
func (s *Scheduler) SetIdleChecker(checker *IdleChecker) {
	s.idleChecker = checker
	if s.debug {
		checker.SetDebug(s.debug)
	}
}

// SetBreakChecker sets the break checker.
func (s *Scheduler) SetBreakChecker(checker *BreakChecker) {
	s.breakChecker = checker
	if s.debug {
		checker.SetDebug(s.debug)
	}
}

// SetGoalChecker sets the goal checker.
func (s *Scheduler) SetGoalChecker(checker *GoalChecker) {
	s.goalChecker = checker
	if s.debug {
		checker.SetDebug(s.debug)
	}
}

// SetSummaryGenerator sets the summary generator.
func (s *Scheduler) SetSummaryGenerator(generator *SummaryGenerator) {
	s.summaryGenerator = generator
	if s.debug {
		generator.SetDebug(s.debug)
	}
}

// Start starts the scheduler with all configured jobs.
func (s *Scheduler) Start() error {
	s.lastCheck = time.Now()

	// Add minute-based checks
	_, err := s.cron.AddFunc("0 * * * * *", func() {
		s.runMinuteChecks()
	})
	if err != nil {
		return fmt.Errorf("failed to add minute checks: %w", err)
	}

	// Add 5-minute checks for less frequent tasks
	_, err = s.cron.AddFunc("0 */5 * * * *", func() {
		s.runFiveMinuteChecks()
	})
	if err != nil {
		return fmt.Errorf("failed to add 5-minute checks: %w", err)
	}

	// Start the cron scheduler
	s.cron.Start()

	if s.debug {
		fmt.Println("[DEBUG] Scheduler started")
	}

	return nil
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
	}
	if s.debug {
		fmt.Println("[DEBUG] Scheduler stopped")
	}
}

// runMinuteChecks runs checks that need to happen every minute.
func (s *Scheduler) runMinuteChecks() {
	s.mu.Lock()
	elapsed := time.Since(s.lastCheck)
	s.lastCheck = time.Now()
	s.mu.Unlock()

	// Skip if system was sleeping (gap > 1 hour)
	if elapsed > time.Hour {
		if s.debug {
			fmt.Printf("[DEBUG] Skipping stale checks after %v sleep\n", elapsed.Round(time.Second))
		}
		return
	}

	if s.debug {
		fmt.Printf("[DEBUG] Running minute checks (elapsed: %v)\n", elapsed.Round(time.Second))
	}

	// Run checks
	s.checkReminders()
	s.checkIdle()
	s.checkBreak()
	s.checkDailySummary()
	s.checkEndOfDay()
}

// runFiveMinuteChecks runs checks that happen every 5 minutes.
func (s *Scheduler) runFiveMinuteChecks() {
	if s.debug {
		fmt.Println("[DEBUG] Running 5-minute checks")
	}

	// Goal progress checks (implemented in Phase 9)
	s.checkGoalProgress()
}

// checkReminders checks for due reminders.
func (s *Scheduler) checkReminders() {
	if s.reminderChecker == nil {
		return
	}
	if s.debug {
		fmt.Println("[DEBUG] Checking reminders...")
	}
	s.reminderChecker.Check()
}

// checkIdle checks for idle detection.
func (s *Scheduler) checkIdle() {
	if s.idleChecker == nil {
		return
	}
	if s.debug {
		fmt.Println("[DEBUG] Checking idle status...")
	}
	s.idleChecker.Check()
}

// checkBreak checks for break reminders.
func (s *Scheduler) checkBreak() {
	if s.breakChecker == nil {
		return
	}
	if s.debug {
		fmt.Println("[DEBUG] Checking break status...")
	}
	s.breakChecker.Check()
}

// checkGoalProgress checks goal progress for notifications.
func (s *Scheduler) checkGoalProgress() {
	if s.goalChecker == nil {
		return
	}
	if s.debug {
		fmt.Println("[DEBUG] Checking goal progress...")
	}
	s.goalChecker.Check()
}

// checkDailySummary checks if it's time for the daily summary.
func (s *Scheduler) checkDailySummary() {
	if s.summaryGenerator == nil {
		return
	}
	s.summaryGenerator.CheckDailySummary()
}

// checkEndOfDay checks if it's time for the end-of-day recap.
func (s *Scheduler) checkEndOfDay() {
	if s.summaryGenerator == nil {
		return
	}
	s.summaryGenerator.CheckEndOfDay()
}

// AddJob adds a custom job to the scheduler.
func (s *Scheduler) AddJob(spec string, job func()) (cron.EntryID, error) {
	return s.cron.AddFunc(spec, job)
}

// RemoveJob removes a job from the scheduler.
func (s *Scheduler) RemoveJob(id cron.EntryID) {
	s.cron.Remove(id)
}

// Entries returns all scheduled entries.
func (s *Scheduler) Entries() []cron.Entry {
	return s.cron.Entries()
}

// NextRun returns the next scheduled run time for any job.
func (s *Scheduler) NextRun() time.Time {
	entries := s.cron.Entries()
	if len(entries) == 0 {
		return time.Time{}
	}

	next := entries[0].Next
	for _, e := range entries[1:] {
		if e.Next.Before(next) {
			next = e.Next
		}
	}
	return next
}
