package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/notify"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/storage"
)

// ReminderChecker checks for due reminders and sends notifications.
type ReminderChecker struct {
	reminderRepo *storage.ReminderRepo
	webhookRepo  *storage.WebhookRepo
	dispatcher   *notify.Dispatcher
	notified     map[string]map[string]time.Time // reminder_key -> interval -> last_notified
	debug        bool
}

// NewReminderChecker creates a new reminder checker.
func NewReminderChecker(reminderRepo *storage.ReminderRepo, webhookRepo *storage.WebhookRepo) *ReminderChecker {
	return &ReminderChecker{
		reminderRepo: reminderRepo,
		webhookRepo:  webhookRepo,
		dispatcher:   notify.NewDispatcher(webhookRepo),
		notified:     make(map[string]map[string]time.Time),
	}
}

// SetDebug enables debug output.
func (c *ReminderChecker) SetDebug(debug bool) {
	c.debug = debug
}

// Check checks for reminders that need notifications.
func (c *ReminderChecker) Check() {
	reminders, err := c.reminderRepo.ListPending()
	if err != nil {
		if c.debug {
			fmt.Printf("[DEBUG] Failed to list reminders: %v\n", err)
		}
		return
	}

	if len(reminders) == 0 {
		return
	}

	// Check each reminder
	var notifications []*model.Notification
	for _, reminder := range reminders {
		notifs := c.checkReminder(reminder)
		notifications = append(notifications, notifs...)
	}

	// Send notifications
	if len(notifications) > 0 {
		c.sendNotifications(notifications)
	}
}

// checkReminder checks a single reminder and returns notifications to send.
func (c *ReminderChecker) checkReminder(reminder *model.Reminder) []*model.Notification {
	var notifications []*model.Notification

	timeUntil := time.Until(reminder.Deadline)

	// Check each notification interval
	for _, intervalStr := range reminder.NotifyBefore {
		result := parser.ParseDuration(intervalStr)
		if !result.Valid {
			continue
		}
		interval := result.Duration

		// Should we notify for this interval?
		if !c.shouldNotify(reminder, intervalStr, interval, timeUntil) {
			continue
		}

		// Create notification
		notification := c.createNotification(reminder, intervalStr, timeUntil)
		notifications = append(notifications, notification)

		// Mark as notified
		c.markNotified(reminder.Key, intervalStr)
	}

	// Check for at-deadline notification (0-1 minute before)
	if timeUntil > 0 && timeUntil <= time.Minute {
		if !c.wasNotified(reminder.Key, "now") {
			notification := c.createNotification(reminder, "now", timeUntil)
			notifications = append(notifications, notification)
			c.markNotified(reminder.Key, "now")
		}
	}

	return notifications
}

// shouldNotify determines if we should send a notification for this interval.
func (c *ReminderChecker) shouldNotify(reminder *model.Reminder, intervalStr string, interval, timeUntil time.Duration) bool {
	// Already past the deadline
	if timeUntil <= 0 {
		return false
	}

	// Not within the notification window yet
	if timeUntil > interval {
		return false
	}

	// Already notified for this interval
	if c.wasNotified(reminder.Key, intervalStr) {
		return false
	}

	// Check if we're within the notification window (interval - 1 minute buffer)
	windowStart := interval
	windowEnd := interval - time.Minute
	if windowEnd < 0 {
		windowEnd = 0
	}

	return timeUntil <= windowStart && timeUntil >= windowEnd
}

// wasNotified checks if we already notified for this reminder and interval.
func (c *ReminderChecker) wasNotified(reminderKey, interval string) bool {
	intervals, ok := c.notified[reminderKey]
	if !ok {
		return false
	}
	lastNotified, ok := intervals[interval]
	if !ok {
		return false
	}
	// Consider it notified if within the last 5 minutes (to avoid duplicates)
	return time.Since(lastNotified) < 5*time.Minute
}

// markNotified records that we notified for this reminder and interval.
func (c *ReminderChecker) markNotified(reminderKey, interval string) {
	if c.notified[reminderKey] == nil {
		c.notified[reminderKey] = make(map[string]time.Time)
	}
	c.notified[reminderKey][interval] = time.Now()
}

// createNotification creates a notification for a reminder.
func (c *ReminderChecker) createNotification(reminder *model.Reminder, interval string, timeUntil time.Duration) *model.Notification {
	var title string
	if interval == "now" {
		title = fmt.Sprintf("Reminder Due: %s", reminder.Title)
	} else {
		title = fmt.Sprintf("Reminder: %s", reminder.Title)
	}

	var message string
	if interval == "now" {
		message = "This reminder is due now!"
	} else {
		message = fmt.Sprintf("Due %s", parser.FormatTimeUntil(reminder.Deadline))
	}

	notification := model.NewNotification(model.NotifyReminder, title, message).
		WithColor(model.ColorWarning)

	if reminder.ProjectSID != "" {
		notification.WithField("Project", reminder.ProjectSID)
	}

	notification.WithField("Deadline", parser.FormatDeadline(reminder.Deadline))

	if reminder.RepeatRule != "" {
		notification.WithField("Repeats", reminder.RepeatRule)
	}

	return notification
}

// sendNotifications sends all pending notifications.
func (c *ReminderChecker) sendNotifications(notifications []*model.Notification) {
	ctx := context.Background()

	for _, notification := range notifications {
		results := c.dispatcher.SendNotification(ctx, notification)

		if c.debug {
			for _, result := range results {
				if result.Success {
					fmt.Printf("[DEBUG] Sent reminder notification to %s\n", result.WebhookName)
				} else {
					fmt.Printf("[DEBUG] Failed to send to %s: %v\n", result.WebhookName, result.Error)
				}
			}
		}
	}
}

// CleanupNotified removes old notification records for completed reminders.
func (c *ReminderChecker) CleanupNotified() {
	for key := range c.notified {
		reminder, err := c.reminderRepo.Get(key)
		if err != nil || reminder.Completed {
			delete(c.notified, key)
		}
	}
}
