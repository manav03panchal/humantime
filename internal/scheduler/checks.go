package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/manav03panchal/humantime/internal/config"
	"github.com/manav03panchal/humantime/internal/logging"
	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/notify"
	"github.com/manav03panchal/humantime/internal/storage"
)

// IdleChecker checks for idle detection and sends notifications.
type IdleChecker struct {
	blockRepo        *storage.BlockRepo
	activeBlockRepo  *storage.ActiveBlockRepo
	webhookRepo      *storage.WebhookRepo
	notifyConfigRepo *storage.NotifyConfigRepo
	dispatcher       *notify.Dispatcher
	lastNotified     time.Time
	debug            bool
}

// NewIdleChecker creates a new idle checker.
func NewIdleChecker(
	blockRepo *storage.BlockRepo,
	activeBlockRepo *storage.ActiveBlockRepo,
	webhookRepo *storage.WebhookRepo,
	notifyConfigRepo *storage.NotifyConfigRepo,
) *IdleChecker {
	return &IdleChecker{
		blockRepo:        blockRepo,
		activeBlockRepo:  activeBlockRepo,
		webhookRepo:      webhookRepo,
		notifyConfigRepo: notifyConfigRepo,
		dispatcher:       notify.NewDispatcher(webhookRepo),
	}
}

// SetDebug enables debug output.
func (c *IdleChecker) SetDebug(debug bool) {
	c.debug = debug
}

// Check checks for idle status and sends notification if needed.
func (c *IdleChecker) Check() {
	// Get notification config
	notifyConfig, err := c.notifyConfigRepo.Get()
	if err != nil {
		if c.debug {
			logging.DebugLog("failed to get notify config", logging.KeyError, err)
		}
		return
	}

	// Check if idle notifications are enabled
	if !notifyConfig.IsTypeEnabled("idle") {
		if c.debug {
			logging.DebugLog("idle notifications disabled")
		}
		return
	}

	// Check if currently tracking (active block exists)
	activeBlock, err := c.activeBlockRepo.Get()
	if err == nil && activeBlock != nil && activeBlock.ActiveBlockKey != "" {
		// Currently tracking, no idle notification needed
		if c.debug {
			logging.DebugLog("currently tracking, skipping idle check")
		}
		return
	}

	// Get today's blocks to check if any tracking occurred today
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	blocks, err := c.blockRepo.ListByTimeRange(startOfDay, now)
	if err != nil {
		if c.debug {
			logging.DebugLog("failed to list blocks", logging.KeyError, err)
		}
		return
	}

	// Skip if no tracking occurred today (FR-012)
	if len(blocks) == 0 {
		if c.debug {
			logging.DebugLog("no tracking today, skipping idle check")
		}
		return
	}

	// Find the most recent block end time
	var lastEndTime time.Time
	for _, block := range blocks {
		if !block.TimestampEnd.IsZero() && block.TimestampEnd.After(lastEndTime) {
			lastEndTime = block.TimestampEnd
		}
	}

	// If no completed blocks, skip
	if lastEndTime.IsZero() {
		if c.debug {
			logging.DebugLog("no completed blocks today, skipping idle check")
		}
		return
	}

	// Check if idle threshold exceeded
	idleDuration := time.Since(lastEndTime)
	if idleDuration < notifyConfig.IdleAfter {
		if c.debug {
			logging.DebugLog("not idle long enough", "idle_duration", idleDuration.Round(time.Minute), "threshold", notifyConfig.IdleAfter)
		}
		return
	}

	// Check for deduplication (don't spam - use configurable cooldown)
	if !c.lastNotified.IsZero() && time.Since(c.lastNotified) < config.Global.Scheduler.IdleNotificationCooldown {
		if c.debug {
			logging.DebugLog("already notified recently, skipping")
		}
		return
	}

	// Send idle notification
	c.sendIdleNotification(idleDuration)
	c.lastNotified = time.Now()
}

// sendIdleNotification sends an idle detection notification.
func (c *IdleChecker) sendIdleNotification(idleDuration time.Duration) {
	minutes := int(idleDuration.Minutes())

	var durationStr string
	if minutes >= 60 {
		hours := minutes / 60
		mins := minutes % 60
		if mins > 0 {
			durationStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			durationStr = fmt.Sprintf("%d hour", hours)
			if hours > 1 {
				durationStr += "s"
			}
		}
	} else {
		durationStr = fmt.Sprintf("%d minutes", minutes)
	}

	notification := model.NewNotification(
		model.NotifyIdle,
		"Tracking Stopped",
		fmt.Sprintf("You stopped tracking %s ago. Still working?", durationStr),
	).WithColor(model.ColorInfo)

	notification.WithField("Idle Time", durationStr)

	ctx := context.Background()
	results := c.dispatcher.SendNotification(ctx, notification)

	if c.debug {
		for _, result := range results {
			if result.Success {
				logging.DebugLog("sent idle notification", logging.KeyWebhook, result.WebhookName)
			} else {
				logging.DebugLog("failed to send idle notification", logging.KeyWebhook, result.WebhookName, logging.KeyError, result.Error)
			}
		}
	}
}

// BreakChecker checks for break reminders and sends notifications.
type BreakChecker struct {
	blockRepo        *storage.BlockRepo
	activeBlockRepo  *storage.ActiveBlockRepo
	webhookRepo      *storage.WebhookRepo
	notifyConfigRepo *storage.NotifyConfigRepo
	dispatcher       *notify.Dispatcher
	sessionStart     time.Time
	lastNotified     time.Time
	debug            bool
}

// NewBreakChecker creates a new break checker.
func NewBreakChecker(
	blockRepo *storage.BlockRepo,
	activeBlockRepo *storage.ActiveBlockRepo,
	webhookRepo *storage.WebhookRepo,
	notifyConfigRepo *storage.NotifyConfigRepo,
) *BreakChecker {
	return &BreakChecker{
		blockRepo:        blockRepo,
		activeBlockRepo:  activeBlockRepo,
		webhookRepo:      webhookRepo,
		notifyConfigRepo: notifyConfigRepo,
		dispatcher:       notify.NewDispatcher(webhookRepo),
	}
}

// SetDebug enables debug output.
func (c *BreakChecker) SetDebug(debug bool) {
	c.debug = debug
}

// Check checks for continuous work and sends break reminder if needed.
func (c *BreakChecker) Check() {
	// Get notification config
	notifyConfig, err := c.notifyConfigRepo.Get()
	if err != nil {
		if c.debug {
			logging.DebugLog("failed to get notify config", logging.KeyError, err)
		}
		return
	}

	// Check if break notifications are enabled
	if !notifyConfig.IsTypeEnabled("break") || notifyConfig.BreakAfter == 0 {
		if c.debug {
			logging.DebugLog("break notifications disabled")
		}
		return
	}

	// Get today's blocks
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	blocks, err := c.blockRepo.ListByTimeRange(startOfDay, now)
	if err != nil {
		if c.debug {
			logging.DebugLog("failed to list blocks", logging.KeyError, err)
		}
		return
	}

	if len(blocks) == 0 {
		c.sessionStart = time.Time{}
		return
	}

	// Calculate continuous session time
	// A session is continuous if gaps between blocks are less than BreakReset
	continuousStart, continuousDuration := c.calculateContinuousSession(blocks, notifyConfig.BreakReset, now)

	if continuousDuration < notifyConfig.BreakAfter {
		if c.debug {
			logging.DebugLog("not working long enough", "duration", continuousDuration.Round(time.Minute), "threshold", notifyConfig.BreakAfter)
		}
		return
	}

	// Check for deduplication (notify at most once per session)
	if !c.lastNotified.IsZero() && c.lastNotified.After(continuousStart) {
		if c.debug {
			logging.DebugLog("already notified for this session")
		}
		return
	}

	// Get current project if tracking
	var projectName string
	activeBlock, err := c.activeBlockRepo.Get()
	if err == nil && activeBlock != nil && activeBlock.ActiveBlockKey != "" {
		// Try to get project name from most recent block
		if len(blocks) > 0 {
			projectName = blocks[0].ProjectSID
		}
	}

	// Send break notification
	c.sendBreakNotification(continuousDuration, projectName)
	c.lastNotified = time.Now()
}

// calculateContinuousSession calculates the start time and duration of the continuous work session.
func (c *BreakChecker) calculateContinuousSession(blocks []*model.Block, breakReset time.Duration, now time.Time) (time.Time, time.Duration) {
	if len(blocks) == 0 {
		return time.Time{}, 0
	}

	// Sort blocks by start time (oldest first)
	// blocks are typically sorted newest first, so we need to reverse
	sortedBlocks := make([]*model.Block, len(blocks))
	for i, b := range blocks {
		sortedBlocks[len(blocks)-1-i] = b
	}

	var sessionStart time.Time
	var totalDuration time.Duration
	var lastEnd time.Time

	for _, block := range sortedBlocks {
		blockEnd := block.TimestampEnd
		if blockEnd.IsZero() {
			blockEnd = now // Active block
		}

		// Check if there's a gap that resets the session
		if !lastEnd.IsZero() && block.TimestampStart.Sub(lastEnd) > breakReset {
			// Reset session
			sessionStart = block.TimestampStart
			totalDuration = blockEnd.Sub(block.TimestampStart)
		} else {
			// Continue session
			if sessionStart.IsZero() {
				sessionStart = block.TimestampStart
			}
			totalDuration += blockEnd.Sub(block.TimestampStart)
		}

		lastEnd = blockEnd
	}

	return sessionStart, totalDuration
}

// sendBreakNotification sends a break reminder notification.
func (c *BreakChecker) sendBreakNotification(duration time.Duration, projectName string) {
	hours := int(duration.Hours())
	mins := int(duration.Minutes()) % 60

	var durationStr string
	if hours > 0 {
		if mins > 0 {
			durationStr = fmt.Sprintf("%dh %dm", hours, mins)
		} else {
			durationStr = fmt.Sprintf("%d hour", hours)
			if hours > 1 {
				durationStr += "s"
			}
		}
	} else {
		durationStr = fmt.Sprintf("%d minutes", mins)
	}

	var message string
	if projectName != "" {
		message = fmt.Sprintf("%s on %s - maybe take a break?", durationStr, projectName)
	} else {
		message = fmt.Sprintf("You've been working for %s - time for a break?", durationStr)
	}

	notification := model.NewNotification(
		model.NotifyBreak,
		"Break Reminder",
		message,
	).WithColor(model.ColorWarning)

	notification.WithField("Session Duration", durationStr)
	if projectName != "" {
		notification.WithField("Project", projectName)
	}

	ctx := context.Background()
	results := c.dispatcher.SendNotification(ctx, notification)

	if c.debug {
		for _, result := range results {
			if result.Success {
				logging.DebugLog("sent break notification", logging.KeyWebhook, result.WebhookName)
			} else {
				logging.DebugLog("failed to send break notification", logging.KeyWebhook, result.WebhookName, logging.KeyError, result.Error)
			}
		}
	}
}

// GoalChecker checks for goal progress and sends milestone notifications.
type GoalChecker struct {
	blockRepo          *storage.BlockRepo
	goalRepo           *storage.GoalRepo
	webhookRepo        *storage.WebhookRepo
	notifyConfigRepo   *storage.NotifyConfigRepo
	dispatcher         *notify.Dispatcher
	notifiedMilestones map[string]map[int]time.Time // project -> milestone -> last notified
	debug              bool
}

// NewGoalChecker creates a new goal checker.
func NewGoalChecker(
	blockRepo *storage.BlockRepo,
	goalRepo *storage.GoalRepo,
	webhookRepo *storage.WebhookRepo,
	notifyConfigRepo *storage.NotifyConfigRepo,
) *GoalChecker {
	return &GoalChecker{
		blockRepo:          blockRepo,
		goalRepo:           goalRepo,
		webhookRepo:        webhookRepo,
		notifyConfigRepo:   notifyConfigRepo,
		dispatcher:         notify.NewDispatcher(webhookRepo),
		notifiedMilestones: make(map[string]map[int]time.Time),
	}
}

// SetDebug enables debug output.
func (c *GoalChecker) SetDebug(debug bool) {
	c.debug = debug
}

// Check checks goal progress and sends milestone notifications.
func (c *GoalChecker) Check() {
	// Get notification config
	notifyConfig, err := c.notifyConfigRepo.Get()
	if err != nil {
		if c.debug {
			logging.DebugLog("failed to get notify config", logging.KeyError, err)
		}
		return
	}

	// Check if goal notifications are enabled
	if !notifyConfig.IsTypeEnabled("goal") {
		if c.debug {
			logging.DebugLog("goal notifications disabled")
		}
		return
	}

	// Get all goals
	goals, err := c.goalRepo.List()
	if err != nil {
		if c.debug {
			logging.DebugLog("failed to list goals", logging.KeyError, err)
		}
		return
	}

	if len(goals) == 0 {
		return
	}

	// Get today's blocks for daily goals
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Get this week's start for weekly goals
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday as last day of week
	}
	startOfWeek := startOfDay.AddDate(0, 0, -(weekday - 1))

	// Check each goal
	for _, goal := range goals {
		c.checkGoal(goal, notifyConfig, startOfDay, startOfWeek, now)
	}

	// Reset old milestone notifications (at day/week boundaries)
	c.cleanupOldMilestones(startOfDay, startOfWeek)
}

// checkGoal checks a single goal for milestone notifications.
func (c *GoalChecker) checkGoal(goal *model.Goal, notifyConfig *model.NotifyConfig, startOfDay, startOfWeek, now time.Time) {
	var start time.Time
	switch goal.Type {
	case model.GoalTypeDaily:
		start = startOfDay
	case model.GoalTypeWeekly:
		start = startOfWeek
	default:
		return
	}

	// Get blocks for the goal's project in the relevant period
	blocks, err := c.blockRepo.ListByTimeRange(start, now)
	if err != nil {
		return
	}

	// Filter by project
	var projectBlocks []*model.Block
	for _, b := range blocks {
		if b.ProjectSID == goal.ProjectSID {
			projectBlocks = append(projectBlocks, b)
		}
	}

	// Calculate total tracked time
	var totalDuration time.Duration
	for _, b := range projectBlocks {
		totalDuration += b.Duration()
	}

	// Calculate progress percentage
	targetDuration := goal.Target
	if targetDuration == 0 {
		return
	}
	progress := int(float64(totalDuration) / float64(targetDuration) * 100)

	// Check milestones
	for _, milestone := range notifyConfig.GoalMilestones {
		if progress >= milestone && !c.wasNotified(goal.ProjectSID, milestone) {
			c.sendGoalNotification(goal, milestone, totalDuration, targetDuration)
			c.markNotified(goal.ProjectSID, milestone)
		}
	}
}

// wasNotified checks if a milestone was already notified.
func (c *GoalChecker) wasNotified(projectSID string, milestone int) bool {
	if c.notifiedMilestones[projectSID] == nil {
		return false
	}
	_, exists := c.notifiedMilestones[projectSID][milestone]
	return exists
}

// markNotified marks a milestone as notified.
func (c *GoalChecker) markNotified(projectSID string, milestone int) {
	if c.notifiedMilestones[projectSID] == nil {
		c.notifiedMilestones[projectSID] = make(map[int]time.Time)
	}
	c.notifiedMilestones[projectSID][milestone] = time.Now()
}

// cleanupOldMilestones removes milestone records from previous days/weeks.
func (c *GoalChecker) cleanupOldMilestones(startOfDay, startOfWeek time.Time) {
	for project, milestones := range c.notifiedMilestones {
		for milestone, notifiedAt := range milestones {
			// Remove if notified before today (for daily) or before this week (for weekly)
			if notifiedAt.Before(startOfWeek) {
				delete(milestones, milestone)
			}
		}
		if len(milestones) == 0 {
			delete(c.notifiedMilestones, project)
		}
	}
}

// sendGoalNotification sends a goal progress notification.
func (c *GoalChecker) sendGoalNotification(goal *model.Goal, milestone int, current, target time.Duration) {
	var title, message string
	var color int

	hours := int(current.Hours())
	targetHours := int(target.Hours())
	goalType := string(goal.Type)

	if milestone >= 100 {
		title = "Goal Achieved!"
		message = fmt.Sprintf("You've completed your %s goal for %s!", goalType, goal.ProjectSID)
		color = model.ColorSuccess
	} else if milestone >= 75 {
		title = "Almost There!"
		message = fmt.Sprintf("%d%% of your %s goal - %dh/%dh on %s", milestone, goalType, hours, targetHours, goal.ProjectSID)
		color = model.ColorInfo
	} else {
		title = "Goal Progress"
		message = fmt.Sprintf("Halfway there! %dh/%dh on %s", hours, targetHours, goal.ProjectSID)
		color = model.ColorInfo
	}

	notification := model.NewNotification(model.NotifyGoal, title, message).WithColor(color)
	notification.WithField("Progress", fmt.Sprintf("%d%%", milestone))
	notification.WithField("Project", goal.ProjectSID)
	notification.WithField("Period", goalType)

	ctx := context.Background()
	results := c.dispatcher.SendNotification(ctx, notification)

	if c.debug {
		for _, result := range results {
			if result.Success {
				logging.DebugLog("sent goal notification", logging.KeyWebhook, result.WebhookName)
			} else {
				logging.DebugLog("failed to send goal notification", logging.KeyWebhook, result.WebhookName, logging.KeyError, result.Error)
			}
		}
	}
}
