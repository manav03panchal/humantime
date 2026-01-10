package scheduler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/notify"
	"github.com/manav03panchal/humantime/internal/storage"
)

// SummaryGenerator generates and sends daily/end-of-day summaries.
type SummaryGenerator struct {
	blockRepo        *storage.BlockRepo
	reminderRepo     *storage.ReminderRepo
	goalRepo         *storage.GoalRepo
	webhookRepo      *storage.WebhookRepo
	notifyConfigRepo *storage.NotifyConfigRepo
	dispatcher       *notify.Dispatcher
	lastDailySummary time.Time
	lastEndOfDay     time.Time
	debug            bool
}

// NewSummaryGenerator creates a new summary generator.
func NewSummaryGenerator(
	blockRepo *storage.BlockRepo,
	reminderRepo *storage.ReminderRepo,
	goalRepo *storage.GoalRepo,
	webhookRepo *storage.WebhookRepo,
	notifyConfigRepo *storage.NotifyConfigRepo,
) *SummaryGenerator {
	return &SummaryGenerator{
		blockRepo:        blockRepo,
		reminderRepo:     reminderRepo,
		goalRepo:         goalRepo,
		webhookRepo:      webhookRepo,
		notifyConfigRepo: notifyConfigRepo,
		dispatcher:       notify.NewDispatcher(webhookRepo),
	}
}

// SetDebug enables debug output.
func (g *SummaryGenerator) SetDebug(debug bool) {
	g.debug = debug
}

// CheckDailySummary checks if it's time to send the daily summary.
func (g *SummaryGenerator) CheckDailySummary() {
	config, err := g.notifyConfigRepo.Get()
	if err != nil {
		if g.debug {
			fmt.Printf("[DEBUG] Failed to get notify config: %v\n", err)
		}
		return
	}

	// Check if daily summary is enabled
	if !config.IsTypeEnabled("daily_summary") || config.DailySummaryAt == "" {
		if g.debug {
			fmt.Println("[DEBUG] Daily summary disabled")
		}
		return
	}

	// Parse the configured time
	targetTime, err := parseTimeOfDay(config.DailySummaryAt)
	if err != nil {
		if g.debug {
			fmt.Printf("[DEBUG] Invalid daily_summary_at time: %v\n", err)
		}
		return
	}

	// Check if we should send now
	if !g.shouldSendSummary(targetTime, g.lastDailySummary) {
		return
	}

	// Generate and send daily summary
	g.sendDailySummary()
	g.lastDailySummary = time.Now()
}

// CheckEndOfDay checks if it's time to send the end-of-day recap.
func (g *SummaryGenerator) CheckEndOfDay() {
	config, err := g.notifyConfigRepo.Get()
	if err != nil {
		if g.debug {
			fmt.Printf("[DEBUG] Failed to get notify config: %v\n", err)
		}
		return
	}

	// Check if end-of-day is enabled
	if !config.IsTypeEnabled("end_of_day") || config.EndOfDayAt == "" {
		if g.debug {
			fmt.Println("[DEBUG] End-of-day recap disabled")
		}
		return
	}

	// Parse the configured time
	targetTime, err := parseTimeOfDay(config.EndOfDayAt)
	if err != nil {
		if g.debug {
			fmt.Printf("[DEBUG] Invalid end_of_day_at time: %v\n", err)
		}
		return
	}

	// Check if we should send now
	if !g.shouldSendSummary(targetTime, g.lastEndOfDay) {
		return
	}

	// Generate and send end-of-day recap
	g.sendEndOfDayRecap()
	g.lastEndOfDay = time.Now()
}

// shouldSendSummary checks if we should send a summary at the target time.
func (g *SummaryGenerator) shouldSendSummary(targetTime time.Time, lastSent time.Time) bool {
	now := time.Now()

	// Build today's target time
	todayTarget := time.Date(now.Year(), now.Month(), now.Day(),
		targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())

	// Check if we're within 5 minutes after the target time
	if now.Before(todayTarget) || now.After(todayTarget.Add(5*time.Minute)) {
		return false
	}

	// Check if we already sent today
	if !lastSent.IsZero() {
		lastSentDay := time.Date(lastSent.Year(), lastSent.Month(), lastSent.Day(), 0, 0, 0, 0, lastSent.Location())
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if lastSentDay.Equal(today) {
			return false
		}
	}

	return true
}

// sendDailySummary sends the morning daily summary.
func (g *SummaryGenerator) sendDailySummary() {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	startOfYesterday := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, now.Location())
	endOfYesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Get yesterday's blocks
	blocks, err := g.blockRepo.ListByTimeRange(startOfYesterday, endOfYesterday)
	if err != nil {
		if g.debug {
			fmt.Printf("[DEBUG] Failed to get yesterday's blocks: %v\n", err)
		}
		return
	}

	// Aggregate by project
	projectTotals := g.aggregateByProject(blocks)

	// Get today's reminders
	todayReminders, err := g.getTodayReminders()
	if err != nil && g.debug {
		fmt.Printf("[DEBUG] Failed to get today's reminders: %v\n", err)
	}

	// Build notification
	var message strings.Builder

	if len(projectTotals) == 0 {
		message.WriteString("No time tracked yesterday.\n")
	} else {
		var totalDuration time.Duration
		for _, pt := range projectTotals {
			totalDuration += pt.duration
		}
		message.WriteString(fmt.Sprintf("Yesterday: %s total\n", formatDuration(totalDuration)))

		for _, pt := range projectTotals {
			message.WriteString(fmt.Sprintf("  • %s: %s\n", pt.project, formatDuration(pt.duration)))
		}
	}

	if len(todayReminders) > 0 {
		message.WriteString("\nToday's deadlines:\n")
		for _, r := range todayReminders {
			message.WriteString(fmt.Sprintf("  • %s at %s\n", r.Title, r.Deadline.Format("3:04 PM")))
		}
	}

	notification := model.NewNotification(
		model.NotifyDailySummary,
		"Good Morning! Daily Summary",
		strings.TrimSpace(message.String()),
	).WithColor(model.ColorInfo)

	if len(projectTotals) > 0 {
		var total time.Duration
		for _, pt := range projectTotals {
			total += pt.duration
		}
		notification.WithField("Yesterday Total", formatDuration(total))
	}
	if len(todayReminders) > 0 {
		notification.WithField("Today's Reminders", fmt.Sprintf("%d", len(todayReminders)))
	}

	ctx := context.Background()
	results := g.dispatcher.SendNotification(ctx, notification)

	if g.debug {
		for _, result := range results {
			if result.Success {
				fmt.Printf("[DEBUG] Sent daily summary to %s\n", result.WebhookName)
			} else {
				fmt.Printf("[DEBUG] Failed to send to %s: %v\n", result.WebhookName, result.Error)
			}
		}
	}
}

// sendEndOfDayRecap sends the end-of-day recap.
func (g *SummaryGenerator) sendEndOfDayRecap() {
	now := time.Now()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Get today's blocks
	blocks, err := g.blockRepo.ListByTimeRange(startOfToday, now)
	if err != nil {
		if g.debug {
			fmt.Printf("[DEBUG] Failed to get today's blocks: %v\n", err)
		}
		return
	}

	// Aggregate by project
	projectTotals := g.aggregateByProject(blocks)

	// Get tomorrow's reminders
	tomorrowReminders, err := g.getTomorrowReminders()
	if err != nil && g.debug {
		fmt.Printf("[DEBUG] Failed to get tomorrow's reminders: %v\n", err)
	}

	// Get goal status
	goalStatus := g.getGoalStatus(blocks)

	// Build notification
	var message strings.Builder

	if len(projectTotals) == 0 {
		message.WriteString("No time tracked today.\n")
	} else {
		var totalDuration time.Duration
		for _, pt := range projectTotals {
			totalDuration += pt.duration
		}
		message.WriteString(fmt.Sprintf("Today: %s total\n", formatDuration(totalDuration)))

		for _, pt := range projectTotals {
			message.WriteString(fmt.Sprintf("  • %s: %s\n", pt.project, formatDuration(pt.duration)))
		}
	}

	if len(goalStatus) > 0 {
		message.WriteString("\nGoal Progress:\n")
		for _, gs := range goalStatus {
			status := "🔴"
			if gs.percentage >= 100 {
				status = "✅"
			} else if gs.percentage >= 75 {
				status = "🟡"
			}
			message.WriteString(fmt.Sprintf("  %s %s: %.0f%% (%s/%s)\n",
				status, gs.project, gs.percentage,
				formatDuration(gs.current), formatDuration(gs.target)))
		}
	}

	if len(tomorrowReminders) > 0 {
		message.WriteString("\nTomorrow's deadlines:\n")
		for _, r := range tomorrowReminders {
			message.WriteString(fmt.Sprintf("  • %s at %s\n", r.Title, r.Deadline.Format("3:04 PM")))
		}
	}

	notification := model.NewNotification(
		model.NotifyEndOfDay,
		"End of Day Recap",
		strings.TrimSpace(message.String()),
	).WithColor(model.ColorSuccess)

	if len(projectTotals) > 0 {
		var total time.Duration
		for _, pt := range projectTotals {
			total += pt.duration
		}
		notification.WithField("Today Total", formatDuration(total))
	}
	if len(tomorrowReminders) > 0 {
		notification.WithField("Tomorrow's Reminders", fmt.Sprintf("%d", len(tomorrowReminders)))
	}

	ctx := context.Background()
	results := g.dispatcher.SendNotification(ctx, notification)

	if g.debug {
		for _, result := range results {
			if result.Success {
				fmt.Printf("[DEBUG] Sent end-of-day recap to %s\n", result.WebhookName)
			} else {
				fmt.Printf("[DEBUG] Failed to send to %s: %v\n", result.WebhookName, result.Error)
			}
		}
	}
}

// projectTotal holds aggregated time for a project.
type projectTotal struct {
	project  string
	duration time.Duration
}

// aggregateByProject aggregates block durations by project.
func (g *SummaryGenerator) aggregateByProject(blocks []*model.Block) []projectTotal {
	totals := make(map[string]time.Duration)

	for _, b := range blocks {
		totals[b.ProjectSID] += b.Duration()
	}

	var result []projectTotal
	for project, duration := range totals {
		result = append(result, projectTotal{project: project, duration: duration})
	}

	// Sort by duration (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].duration > result[j].duration
	})

	return result
}

// goalStatus holds goal progress information.
type goalStatus struct {
	project    string
	current    time.Duration
	target     time.Duration
	percentage float64
}

// getGoalStatus gets the status of daily goals.
func (g *SummaryGenerator) getGoalStatus(todayBlocks []*model.Block) []goalStatus {
	goals, err := g.goalRepo.List()
	if err != nil {
		return nil
	}

	// Calculate time per project
	projectTime := make(map[string]time.Duration)
	for _, b := range todayBlocks {
		projectTime[b.ProjectSID] += b.Duration()
	}

	var result []goalStatus
	for _, goal := range goals {
		if goal.Type != model.GoalTypeDaily {
			continue
		}

		current := projectTime[goal.ProjectSID]
		percentage := float64(current) / float64(goal.Target) * 100

		result = append(result, goalStatus{
			project:    goal.ProjectSID,
			current:    current,
			target:     goal.Target,
			percentage: percentage,
		})
	}

	// Sort by percentage (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].percentage > result[j].percentage
	})

	return result
}

// getTodayReminders gets reminders due today.
func (g *SummaryGenerator) getTodayReminders() ([]*model.Reminder, error) {
	now := time.Now()
	endOfToday := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())

	reminders, err := g.reminderRepo.ListPending()
	if err != nil {
		return nil, err
	}

	var todayReminders []*model.Reminder
	for _, r := range reminders {
		if r.Deadline.Before(endOfToday) && r.Deadline.After(now) {
			todayReminders = append(todayReminders, r)
		}
	}

	// Sort by deadline
	sort.Slice(todayReminders, func(i, j int) bool {
		return todayReminders[i].Deadline.Before(todayReminders[j].Deadline)
	})

	return todayReminders, nil
}

// getTomorrowReminders gets reminders due tomorrow.
func (g *SummaryGenerator) getTomorrowReminders() ([]*model.Reminder, error) {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	startOfTomorrow := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, now.Location())
	endOfTomorrow := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, now.Location())

	reminders, err := g.reminderRepo.ListPending()
	if err != nil {
		return nil, err
	}

	var tomorrowReminders []*model.Reminder
	for _, r := range reminders {
		if r.Deadline.After(startOfTomorrow) && r.Deadline.Before(endOfTomorrow) {
			tomorrowReminders = append(tomorrowReminders, r)
		}
	}

	// Sort by deadline
	sort.Slice(tomorrowReminders, func(i, j int) bool {
		return tomorrowReminders[i].Deadline.Before(tomorrowReminders[j].Deadline)
	})

	return tomorrowReminders, nil
}

// parseTimeOfDay parses a time string in HH:MM format.
func parseTimeOfDay(s string) (time.Time, error) {
	// Try parsing as HH:MM
	t, err := time.Parse("15:04", s)
	if err == nil {
		return t, nil
	}

	// Try parsing as H:MM
	t, err = time.Parse("3:04", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time format: %s (expected HH:MM)", s)
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", minutes)
}
