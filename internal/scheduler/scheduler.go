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
	cron            *cron.Cron
	db              *storage.DB
	debug           bool
	lastCheck       time.Time
	mu              sync.Mutex
	reminderChecker *ReminderChecker
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
}

// SetReminderChecker sets the reminder checker.
func (s *Scheduler) SetReminderChecker(checker *ReminderChecker) {
	s.reminderChecker = checker
	if s.debug {
		checker.SetDebug(s.debug)
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

	// Run checks (these will be implemented in Phase 5-7)
	s.checkReminders()
	s.checkIdle()
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
	// Will be implemented in Phase 6
	if s.debug {
		fmt.Println("[DEBUG] Checking idle status...")
	}
}

// checkGoalProgress checks goal progress for notifications.
func (s *Scheduler) checkGoalProgress() {
	// Will be implemented in Phase 9
	if s.debug {
		fmt.Println("[DEBUG] Checking goal progress...")
	}
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
