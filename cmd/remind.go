package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
)

// Remind command flags.
var (
	remindFlagProject string
	remindFlagRepeat  string
	remindFlagNotify  string
	remindDeleteForce bool
	remindListAll     bool
	remindListProject string
)

// remindCmd represents the remind command.
var remindCmd = &cobra.Command{
	Use:     "remind [TITLE] [DEADLINE]",
	Aliases: []string{"r", "rem"},
	Short:   "Manage deadline reminders",
	Long: `Create and manage deadline reminders with natural language support.

When called with arguments, creates a new reminder. Otherwise, lists pending reminders.

Deadline formats:
  - Relative: +5m, +1h, +2d, +1w
  - Natural language: "friday 5pm", "tomorrow 2pm", "next monday 10am"
  - Date/time: "2026-01-15 14:00"

Examples:
  humantime remind "Submit invoice" friday 5pm
  humantime remind "Client call" tomorrow 2pm --project clientwork
  humantime remind "Weekly report" friday 4pm --repeat weekly
  humantime remind "Standup" +30m`,
	RunE: runRemindCreate,
}

// remindListCmd lists reminders.
var remindListCmd = &cobra.Command{
	Use:   "list",
	Short: "List reminders",
	Long: `List pending reminders. Use --all to include completed reminders.

Examples:
  humantime remind list
  humantime remind list --all
  humantime remind list --project clientwork`,
	RunE: runRemindList,
}

// remindDoneCmd marks a reminder as complete.
var remindDoneCmd = &cobra.Command{
	Use:   "done ID_OR_TITLE",
	Short: "Mark a reminder as complete",
	Long: `Mark a reminder as completed by ID (prefix) or exact title.

Examples:
  humantime remind done abc123
  humantime remind done "Submit invoice"`,
	Args: cobra.ExactArgs(1),
	RunE: runRemindDone,
}

// remindDeleteCmd deletes a reminder.
var remindDeleteCmd = &cobra.Command{
	Use:     "delete ID",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a reminder",
	Args:    cobra.ExactArgs(1),
	RunE:    runRemindDelete,
}

func init() {
	// Create reminder flags
	remindCmd.Flags().StringVarP(&remindFlagProject, "project", "p", "",
		"Link reminder to a project")
	remindCmd.Flags().StringVarP(&remindFlagRepeat, "repeat", "r", "",
		"Recurrence: daily, weekly, monthly")
	remindCmd.Flags().StringVar(&remindFlagNotify, "notify", "1h,15m",
		"When to notify (e.g., '1h,15m')")

	// List flags
	remindListCmd.Flags().BoolVarP(&remindListAll, "all", "a", false,
		"Include completed reminders")
	remindListCmd.Flags().StringVarP(&remindListProject, "project", "p", "",
		"Filter by project")

	// Delete flags
	remindDeleteCmd.Flags().BoolVarP(&remindDeleteForce, "force", "f", false,
		"Skip confirmation")

	// Dynamic completion
	remindDoneCmd.ValidArgsFunction = completeReminderArgs
	remindDeleteCmd.ValidArgsFunction = completeReminderArgs

	// Add subcommands
	remindCmd.AddCommand(remindListCmd)
	remindCmd.AddCommand(remindDoneCmd)
	remindCmd.AddCommand(remindDeleteCmd)

	rootCmd.AddCommand(remindCmd)
}

// completeReminderArgs provides completion for reminder IDs.
func completeReminderArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Initialize context for completion
	if ctx == nil {
		opts := runtime.DefaultOptions()
		var err error
		ctx, err = runtime.New(opts)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		defer ctx.Close()
	}

	reminders, err := ctx.ReminderRepo.ListPending()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for _, r := range reminders {
		shortID := r.ShortID()
		if strings.HasPrefix(shortID, toComplete) {
			suggestions = append(suggestions, fmt.Sprintf("%s\t%s", shortID, r.Title))
		}
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}

// runRemindCreate handles creating a new reminder.
func runRemindCreate(cmd *cobra.Command, args []string) error {
	// If no args, show list
	if len(args) == 0 {
		return runRemindList(cmd, args)
	}

	// Need at least title and deadline
	if len(args) < 2 {
		return fmt.Errorf("usage: humantime remind \"TITLE\" DEADLINE")
	}

	title := args[0]
	if title == "" {
		return fmt.Errorf("reminder title is required")
	}
	if len(title) > 200 {
		return fmt.Errorf("title too long (max 200 characters)")
	}

	// Parse deadline from remaining args
	deadlineResult := parser.ParseDeadlineArgs(args[1:])
	if deadlineResult.Error != nil {
		return fmt.Errorf("could not parse deadline: %w", deadlineResult.Error)
	}

	// Validate repeat rule
	if remindFlagRepeat != "" && !model.IsValidRepeatRule(remindFlagRepeat) {
		return fmt.Errorf("invalid repeat rule: must be daily, weekly, or monthly")
	}

	// Create reminder
	reminder := model.NewReminder(title, deadlineResult.Time, ctx.Config.UserKey)
	reminder.ProjectSID = remindFlagProject
	reminder.RepeatRule = remindFlagRepeat

	// Parse notify intervals
	if remindFlagNotify != "" {
		reminder.NotifyBefore = strings.Split(remindFlagNotify, ",")
		for i := range reminder.NotifyBefore {
			reminder.NotifyBefore[i] = strings.TrimSpace(reminder.NotifyBefore[i])
		}
	}

	if err := ctx.ReminderRepo.Create(reminder); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"key":           reminder.Key,
			"title":         reminder.Title,
			"deadline":      reminder.Deadline,
			"notify_before": reminder.NotifyBefore,
			"project_sid":   reminder.ProjectSID,
			"repeat_rule":   reminder.RepeatRule,
			"created_at":    reminder.CreatedAt,
		})
	}

	ctx.Formatter.Printf("Created reminder: %s\n", reminder.Title)
	ctx.Formatter.Printf("Due: %s (%s)\n",
		parser.FormatDeadline(reminder.Deadline),
		parser.FormatTimeUntil(reminder.Deadline))

	notifyStr := strings.Join(reminder.NotifyBefore, ", ")
	ctx.Formatter.Printf("Notifications: %s before\n", notifyStr)

	if reminder.RepeatRule != "" {
		ctx.Formatter.Printf("Repeats: %s\n", reminder.RepeatRule)
	}

	return nil
}

// runRemindList handles listing reminders.
func runRemindList(cmd *cobra.Command, args []string) error {
	var reminders []*model.Reminder
	var err error

	if remindListAll {
		reminders, err = ctx.ReminderRepo.List()
	} else {
		reminders, err = ctx.ReminderRepo.ListPending()
	}

	if err != nil {
		return err
	}

	// Filter by project if specified
	if remindListProject != "" {
		var filtered []*model.Reminder
		for _, r := range reminders {
			if r.ProjectSID == remindListProject {
				filtered = append(filtered, r)
			}
		}
		reminders = filtered
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"reminders": reminders,
			"count":     len(reminders),
		})
	}

	if len(reminders) == 0 {
		if remindListAll {
			ctx.Formatter.Println("No reminders found.")
		} else {
			ctx.Formatter.Println("No pending reminders.")
		}
		ctx.Formatter.Println("")
		ctx.Formatter.Println("Create one with: humantime remind \"Title\" deadline")
		return nil
	}

	if remindListAll {
		ctx.Formatter.Println("All Reminders:")
	} else {
		ctx.Formatter.Println("Pending Reminders:")
	}
	ctx.Formatter.Println("")

	for _, r := range reminders {
		shortID := r.ShortID()
		deadline := parser.FormatDeadline(r.Deadline)
		timeUntil := parser.FormatTimeUntil(r.Deadline)

		status := ""
		if r.Completed {
			status = " [completed]"
		} else if r.RepeatRule != "" {
			status = fmt.Sprintf(" [%s]", r.RepeatRule)
		}

		ctx.Formatter.Printf("  %s  %-20s %s (%s)%s\n",
			shortID, truncate(r.Title, 20), deadline, timeUntil, status)
	}

	ctx.Formatter.Println("")
	ctx.Formatter.Printf("%d reminders\n", len(reminders))

	return nil
}

// runRemindDone handles marking a reminder as complete.
func runRemindDone(cmd *cobra.Command, args []string) error {
	idOrTitle := args[0]

	// Try by short ID first
	reminder, err := ctx.ReminderRepo.GetByShortID(idOrTitle)
	if err != nil {
		// Try by title
		reminder, err = ctx.ReminderRepo.GetByTitle(idOrTitle)
		if err != nil {
			return fmt.Errorf("reminder %q not found", idOrTitle)
		}
	}

	if reminder.Completed {
		return fmt.Errorf("reminder is already completed")
	}

	if err := ctx.ReminderRepo.MarkComplete(reminder.Key); err != nil {
		return err
	}

	// Create next occurrence for recurring reminders
	if reminder.IsRecurring() {
		next, err := ctx.ReminderRepo.CreateNextRecurrence(reminder)
		if err != nil {
			ctx.Formatter.Printf("Warning: failed to create next occurrence: %v\n", err)
		} else if next != nil && !ctx.IsJSON() {
			ctx.Formatter.Printf("Next occurrence created for %s\n",
				parser.FormatDeadline(next.Deadline))
		}
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status": "completed",
			"key":    reminder.Key,
			"title":  reminder.Title,
		})
	}

	ctx.Formatter.Printf("Completed: %s\n", reminder.Title)
	return nil
}

// runRemindDelete handles deleting a reminder.
func runRemindDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	reminder, err := ctx.ReminderRepo.GetByShortID(id)
	if err != nil {
		return fmt.Errorf("reminder %q not found", id)
	}

	// Confirmation (skip if --force)
	if !remindDeleteForce && !ctx.IsJSON() {
		ctx.Formatter.Printf("Delete reminder %q? [y/N] ", reminder.Title)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			ctx.Formatter.Println("Cancelled.")
			return nil
		}
	}

	if err := ctx.ReminderRepo.Delete(reminder.Key); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.PrintJSON(map[string]interface{}{
			"status": "deleted",
			"key":    reminder.Key,
			"title":  reminder.Title,
		})
	}

	ctx.Formatter.Printf("Deleted: %s\n", reminder.Title)
	return nil
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
