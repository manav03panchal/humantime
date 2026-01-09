package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Goal command flags.
var (
	goalSetFlagDaily  string
	goalSetFlagWeekly string
)

// goalCmd represents the goal command.
var goalCmd = &cobra.Command{
	Use:     "goal [PROJECT_SID]",
	Aliases: []string{"goals"},
	Short:   "Manage time goals for projects",
	Long: `View and manage time goals for projects.

Shows all goals with progress when called without arguments, or shows
the goal for a specific project when a project SID is provided.

Examples:
  humantime goal
  humantime goal clientwork
  humantime goal set clientwork --daily 4h
  humantime goal set clientwork --weekly 20h
  humantime goal delete clientwork`,
	RunE: runGoalList,
}

// goalSetCmd sets a goal for a project.
var goalSetCmd = &cobra.Command{
	Use:   "set PROJECT_SID",
	Short: "Set a time goal for a project",
	Long: `Set a daily or weekly time goal for a project.

Duration format: Use h for hours, m for minutes (e.g., "4h", "30m", "2h30m", "4.5h")

Examples:
  humantime goal set clientwork --daily 4h
  humantime goal set clientwork --weekly 20h
  humantime goal set sideproject --daily 2h30m`,
	Args: cobra.ExactArgs(1),
	RunE: runGoalSet,
}

// goalDeleteCmd deletes a goal for a project.
var goalDeleteCmd = &cobra.Command{
	Use:     "delete PROJECT_SID",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a goal for a project",
	Args:    cobra.ExactArgs(1),
	RunE:    runGoalDelete,
}

func init() {
	// Set flags
	goalSetCmd.Flags().StringVarP(&goalSetFlagDaily, "daily", "d", "", "Daily time goal (e.g., 4h, 2h30m)")
	goalSetCmd.Flags().StringVarP(&goalSetFlagWeekly, "weekly", "w", "", "Weekly time goal (e.g., 20h, 40h)")

	// Dynamic completion for projects
	goalCmd.ValidArgsFunction = completeProjectArgs
	goalSetCmd.ValidArgsFunction = completeProjectArgs
	goalDeleteCmd.ValidArgsFunction = completeProjectArgs

	goalCmd.AddCommand(goalSetCmd)
	goalCmd.AddCommand(goalDeleteCmd)
	rootCmd.AddCommand(goalCmd)
}

// parseDuration parses a human-friendly duration string like "4h", "30m", "2h30m", "4.5h".
func parseDuration(input string) (time.Duration, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Handle decimal hours like "4.5h"
	decimalHoursRegex := regexp.MustCompile(`^(\d+(?:\.\d+)?)h$`)
	if match := decimalHoursRegex.FindStringSubmatch(input); match != nil {
		hours, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hours: %s", match[1])
		}
		return time.Duration(hours * float64(time.Hour)), nil
	}

	// Try standard Go duration parsing first (handles "1h30m", "45m", etc.)
	d, err := time.ParseDuration(input)
	if err == nil {
		return d, nil
	}

	// Handle "Xh Ym" format with space
	withSpaceRegex := regexp.MustCompile(`^(\d+)h\s+(\d+)m$`)
	if match := withSpaceRegex.FindStringSubmatch(input); match != nil {
		hours, _ := strconv.Atoi(match[1])
		minutes, _ := strconv.Atoi(match[2])
		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
	}

	return 0, fmt.Errorf("invalid duration format: %s (use e.g., 4h, 30m, 2h30m)", input)
}

func runGoalList(cmd *cobra.Command, args []string) error {
	// If a project SID is provided, show that project's goal
	if len(args) > 0 {
		return showGoal(args[0])
	}

	// List all goals with progress
	goals, err := ctx.GoalRepo.List()
	if err != nil {
		return err
	}

	if len(goals) == 0 {
		if ctx.IsJSON() {
			return ctx.Formatter.JSON(GoalsResponse{Goals: []*GoalOutput{}})
		}
		cli := ctx.CLIFormatter()
		cli.Muted("No goals set.")
		cli.Muted("Use 'humantime goal set <project> --daily 4h' to set one.")
		return nil
	}

	// Calculate progress for each goal
	goalOutputs := make([]*GoalOutput, 0, len(goals))
	for _, goal := range goals {
		progress, err := calculateGoalProgress(goal)
		if err != nil {
			return err
		}
		goalOutputs = append(goalOutputs, NewGoalOutput(goal, progress))
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(GoalsResponse{Goals: goalOutputs})
	}

	return printGoalsCLI(goalOutputs)
}

func showGoal(projectSID string) error {
	// Validate project SID
	if !parser.ValidateSID(projectSID) {
		return runtime.ErrInvalidSID
	}

	goal, err := ctx.GoalRepo.Get(projectSID)
	if err != nil {
		if storage.IsErrKeyNotFound(err) {
			if ctx.IsJSON() {
				return ctx.JSONFormatter().PrintError("error", "goal_not_found", fmt.Sprintf("No goal set for project '%s'", projectSID))
			}
			cli := ctx.CLIFormatter()
			cli.Muted(fmt.Sprintf("No goal set for project '%s'.", projectSID))
			cli.Muted("Use 'humantime goal set " + projectSID + " --daily 4h' to set one.")
			return nil
		}
		return err
	}

	progress, err := calculateGoalProgress(goal)
	if err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(NewGoalOutput(goal, progress))
	}

	return printSingleGoalCLI(goal, progress)
}

func runGoalSet(cmd *cobra.Command, args []string) error {
	projectSID := args[0]

	// Validate project SID
	if !parser.ValidateSID(projectSID) {
		return runtime.ErrInvalidSID
	}

	// Determine goal type and parse duration
	var goalType model.GoalType
	var target time.Duration
	var err error

	if goalSetFlagDaily != "" && goalSetFlagWeekly != "" {
		return fmt.Errorf("cannot set both --daily and --weekly; choose one")
	}

	if goalSetFlagDaily != "" {
		goalType = model.GoalTypeDaily
		target, err = parseDuration(goalSetFlagDaily)
		if err != nil {
			return fmt.Errorf("invalid daily duration: %w", err)
		}
	} else if goalSetFlagWeekly != "" {
		goalType = model.GoalTypeWeekly
		target, err = parseDuration(goalSetFlagWeekly)
		if err != nil {
			return fmt.Errorf("invalid weekly duration: %w", err)
		}
	} else {
		return fmt.Errorf("must specify either --daily or --weekly")
	}

	if target <= 0 {
		return fmt.Errorf("goal duration must be positive")
	}

	// Create or update goal
	goal := model.NewGoal(projectSID, goalType, target)
	if err := ctx.GoalRepo.Upsert(goal); err != nil {
		return err
	}

	// Calculate current progress
	progress, err := calculateGoalProgress(goal)
	if err != nil {
		return err
	}

	if ctx.IsJSON() {
		resp := GoalSetResponse{
			Status: "set",
			Goal:   NewGoalOutput(goal, progress),
		}
		return ctx.Formatter.JSON(resp)
	}

	cli := ctx.CLIFormatter()
	cli.Success(fmt.Sprintf("Set %s goal for %s: %s", goal.Type, cli.ProjectName(projectSID), output.FormatDuration(target)))
	printProgressLine(cli, progress)

	return nil
}

func runGoalDelete(cmd *cobra.Command, args []string) error {
	projectSID := args[0]

	// Validate project SID
	if !parser.ValidateSID(projectSID) {
		return runtime.ErrInvalidSID
	}

	// Check if goal exists
	exists, err := ctx.GoalRepo.Exists(projectSID)
	if err != nil {
		return err
	}
	if !exists {
		if ctx.IsJSON() {
			return ctx.JSONFormatter().PrintError("error", "goal_not_found", fmt.Sprintf("No goal set for project '%s'", projectSID))
		}
		return fmt.Errorf("no goal set for project '%s'", projectSID)
	}

	// Delete goal
	if err := ctx.GoalRepo.Delete(projectSID); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(map[string]string{
			"status":      "deleted",
			"project_sid": projectSID,
		})
	}

	cli := ctx.CLIFormatter()
	cli.Success(fmt.Sprintf("Deleted goal for project '%s'", projectSID))

	return nil
}

// calculateGoalProgress calculates the progress toward a goal.
func calculateGoalProgress(goal *model.Goal) (model.Progress, error) {
	// Determine time range based on goal type
	var timeRange parser.TimeRange
	now := time.Now()

	switch goal.Type {
	case model.GoalTypeDaily:
		// Today's range
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		timeRange = parser.TimeRange{
			Start: start,
			End:   start.AddDate(0, 0, 1),
		}
	case model.GoalTypeWeekly:
		// This week's range (Monday to Sunday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		start := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		timeRange = parser.TimeRange{
			Start: start,
			End:   start.AddDate(0, 0, 7),
		}
	}

	// Get blocks for this project within the time range
	filter := storage.BlockFilter{
		ProjectSID: goal.ProjectSID,
		StartAfter: timeRange.Start,
		EndBefore:  timeRange.End,
	}

	blocks, err := ctx.BlockRepo.ListFiltered(filter)
	if err != nil {
		return model.Progress{}, err
	}

	// Calculate total duration
	totalDuration := storage.TotalDuration(blocks)

	return goal.CalculateProgress(totalDuration), nil
}

func printGoalsCLI(goals []*GoalOutput) error {
	cli := ctx.CLIFormatter()

	cli.Title(fmt.Sprintf("Goals (%d)", len(goals)))
	cli.Println("")

	for _, g := range goals {
		typeLabel := "daily"
		if g.Type == string(model.GoalTypeWeekly) {
			typeLabel = "weekly"
		}

		// Progress bar
		barWidth := 20
		bar := output.ProgressBar(g.Progress.Percentage, barWidth)

		// Status indicator
		statusIndicator := ""
		if g.Progress.IsComplete {
			statusIndicator = " [DONE]"
		}

		cli.Printf("%s  %s  %s/%s  %s  %.0f%%%s\n",
			cli.ProjectName(g.ProjectSID),
			typeLabel,
			cli.Duration(output.FormatDuration(time.Duration(g.Progress.CurrentSeconds)*time.Second)),
			output.FormatDuration(time.Duration(g.TargetSeconds)*time.Second),
			bar,
			g.Progress.Percentage,
			statusIndicator,
		)
	}

	return nil
}

func printSingleGoalCLI(goal *model.Goal, progress model.Progress) error {
	cli := ctx.CLIFormatter()

	typeLabel := "Daily"
	periodLabel := "today"
	if goal.Type == model.GoalTypeWeekly {
		typeLabel = "Weekly"
		periodLabel = "this week"
	}

	cli.Title(fmt.Sprintf("%s Goal: %s", typeLabel, goal.ProjectSID))
	cli.Println("")
	cli.Printf("  Target: %s\n", cli.Duration(output.FormatDuration(goal.Target)))
	cli.Printf("  Tracked %s: %s\n", periodLabel, cli.Duration(output.FormatDuration(progress.Current)))
	cli.Println("")

	printProgressLine(cli, progress)

	return nil
}

func printProgressLine(cli *output.CLIFormatter, progress model.Progress) {
	barWidth := 30
	bar := output.ProgressBar(progress.Percentage, barWidth)

	statusText := ""
	if progress.IsComplete {
		statusText = " Complete!"
	} else {
		statusText = fmt.Sprintf(" (%s remaining)", output.FormatDuration(progress.Remaining))
	}

	cli.Printf("  Progress: %s %.1f%%%s\n", bar, progress.Percentage, statusText)
}

// JSON output types for goals.

// GoalOutput represents a goal in JSON output.
type GoalOutput struct {
	ProjectSID    string          `json:"project_sid"`
	Type          string          `json:"type"`
	TargetSeconds int64           `json:"target_seconds"`
	Progress      *ProgressOutput `json:"progress"`
}

// ProgressOutput represents progress in JSON output.
type ProgressOutput struct {
	CurrentSeconds   int64   `json:"current_seconds"`
	RemainingSeconds int64   `json:"remaining_seconds"`
	Percentage       float64 `json:"percentage"`
	IsComplete       bool    `json:"is_complete"`
}

// GoalsResponse represents the goals list output in JSON.
type GoalsResponse struct {
	Goals []*GoalOutput `json:"goals"`
}

// GoalSetResponse represents the goal set command output in JSON.
type GoalSetResponse struct {
	Status string      `json:"status"`
	Goal   *GoalOutput `json:"goal"`
}

// NewGoalOutput creates a GoalOutput from a Goal and Progress.
func NewGoalOutput(goal *model.Goal, progress model.Progress) *GoalOutput {
	return &GoalOutput{
		ProjectSID:    goal.ProjectSID,
		Type:          string(goal.Type),
		TargetSeconds: goal.TargetSeconds(),
		Progress: &ProgressOutput{
			CurrentSeconds:   int64(progress.Current.Seconds()),
			RemainingSeconds: int64(progress.Remaining.Seconds()),
			Percentage:       progress.Percentage,
			IsComplete:       progress.IsComplete,
		},
	}
}
