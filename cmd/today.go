package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/storage"
)

// todayCmd represents the today command.
var todayCmd = &cobra.Command{
	Use:     "today",
	Aliases: []string{"t", "td"},
	Short:   "Show today's time tracking summary",
	Long: `Display a summary of all time tracked today, grouped by project.
Shows totals, active tracking, and goal progress if goals are set.

Examples:
  humantime today
  humantime t`,
	RunE: runToday,
}

func init() {
	rootCmd.AddCommand(todayCmd)
}

func runToday(cmd *cobra.Command, args []string) error {
	// Get today's date range
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)

	// Get today's blocks
	filter := storage.BlockFilter{
		StartAfter: todayStart,
		EndBefore:  todayEnd,
	}
	blocks, err := ctx.BlockRepo.ListFiltered(filter)
	if err != nil {
		return err
	}

	// Get active block
	activeBlock, err := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if err != nil {
		return err
	}

	// Include active block if it started today
	if activeBlock != nil && activeBlock.TimestampStart.After(todayStart) {
		// Check if it's not already in the list
		found := false
		for _, b := range blocks {
			if b.Key == activeBlock.Key {
				found = true
				break
			}
		}
		if !found {
			blocks = append(blocks, activeBlock)
		}
	}

	// Get goals for progress display
	goals, err := ctx.GoalRepo.List()
	if err != nil {
		return err
	}

	// Calculate aggregates by project
	projectAggs := storage.AggregateByProject(blocks)

	// JSON output
	if ctx.IsJSON() {
		return printTodayJSON(blocks, projectAggs, activeBlock)
	}

	// CLI output
	return printTodayCLI(projectAggs, activeBlock, goals, todayStart)
}

func printTodayCLI(projectAggs []storage.ProjectAggregate, activeBlock *model.Block, goals []*model.Goal, todayStart time.Time) error {
	cli := ctx.CLIFormatter()

	// Header
	cli.Title(fmt.Sprintf("Today (%s)", todayStart.Format("2006-01-02")))
	cli.Println("")

	// If no tracking today
	if len(projectAggs) == 0 && activeBlock == nil {
		cli.Muted("No time tracked today.")
		return nil
	}

	// Calculate total duration and max widths
	var totalDuration time.Duration
	maxProjectLen := 0
	maxDurationLen := 0
	for _, agg := range projectAggs {
		totalDuration += agg.Duration
		if len(agg.ProjectSID) > maxProjectLen {
			maxProjectLen = len(agg.ProjectSID)
		}
		dur := output.FormatDuration(agg.Duration)
		if len(dur) > maxDurationLen {
			maxDurationLen = len(dur)
		}
	}
	if maxProjectLen < 12 {
		maxProjectLen = 12
	}
	if maxDurationLen < 8 {
		maxDurationLen = 8
	}

	// Print project breakdown
	for _, agg := range projectAggs {
		// Check if this project has an active block
		isActive := activeBlock != nil && activeBlock.ProjectSID == agg.ProjectSID

		percentage := float64(agg.Duration) / float64(totalDuration) * 100
		barWidth := 20
		bar := output.ProgressBar(percentage, barWidth)

		padding := maxProjectLen - len(agg.ProjectSID)
		projectDisplay := cli.ProjectName(agg.ProjectSID)
		if isActive {
			projectDisplay = projectDisplay + " *"
			padding -= 2
		}
		if padding < 0 {
			padding = 0
		}

		cli.Printf("  %s%s  %*s  %s  %5.1f%%\n",
			projectDisplay,
			strings.Repeat(" ", padding),
			maxDurationLen,
			cli.Duration(output.FormatDuration(agg.Duration)),
			bar,
			percentage)
	}

	// Separator and total
	cli.Println("")
	cli.Println(strings.Repeat("─", maxProjectLen+maxDurationLen+35))
	cli.Printf("  %-*s  %*s\n", maxProjectLen, "Total:", maxDurationLen, cli.Duration(output.FormatDuration(totalDuration)))

	// Show active tracking
	if activeBlock != nil {
		cli.Println("")
		cli.Printf("Currently tracking: %s", cli.FormatProjectTask(activeBlock.ProjectSID, activeBlock.TaskSID))
		if activeBlock.Note != "" {
			cli.Printf(" (%s)", cli.Note(activeBlock.Note))
		}
		cli.Println("")
		elapsed := time.Since(activeBlock.TimestampStart)
		cli.Printf("  Elapsed: %s\n", cli.Duration(output.FormatDuration(elapsed)))
	}

	// Show goal progress for projects with daily goals
	if len(goals) > 0 {
		hasGoalProgress := false
		for _, goal := range goals {
			if goal.Type != model.GoalTypeDaily {
				continue
			}
			// Find project aggregate
			var projectDuration time.Duration
			for _, agg := range projectAggs {
				if agg.ProjectSID == goal.ProjectSID {
					projectDuration = agg.Duration
					break
				}
			}
			if goal.Target > 0 {
				if !hasGoalProgress {
					cli.Println("")
					cli.Println("Daily Goals:")
					hasGoalProgress = true
				}
				prog := goal.CalculateProgress(projectDuration)
				bar := output.ProgressBar(prog.Percentage, 20)
				cli.Printf("  %s: %s %.1f%% (%s remaining)\n",
					cli.ProjectName(goal.ProjectSID),
					bar,
					prog.Percentage,
					output.FormatDuration(prog.Remaining))
			}
		}
	}

	return nil
}

func printTodayJSON(blocks []*model.Block, projectAggs []storage.ProjectAggregate, activeBlock *model.Block) error {
	type ProjectSummary struct {
		ProjectSID      string `json:"project_sid"`
		DurationSeconds int64  `json:"duration_seconds"`
		BlockCount      int    `json:"block_count"`
		Percentage      float64 `json:"percentage"`
	}

	type TodayResponse struct {
		Date        string           `json:"date"`
		TotalSeconds int64           `json:"total_seconds"`
		Projects    []ProjectSummary `json:"projects"`
		ActiveBlock *output.BlockOutput `json:"active_block,omitempty"`
	}

	var totalDuration time.Duration
	for _, agg := range projectAggs {
		totalDuration += agg.Duration
	}

	projects := make([]ProjectSummary, len(projectAggs))
	for i, agg := range projectAggs {
		percentage := float64(agg.Duration) / float64(totalDuration) * 100
		projects[i] = ProjectSummary{
			ProjectSID:      agg.ProjectSID,
			DurationSeconds: int64(agg.Duration.Seconds()),
			BlockCount:      agg.BlockCount,
			Percentage:      percentage,
		}
	}

	resp := TodayResponse{
		Date:         time.Now().Format("2006-01-02"),
		TotalSeconds: int64(totalDuration.Seconds()),
		Projects:     projects,
	}

	if activeBlock != nil {
		resp.ActiveBlock = output.NewBlockOutput(activeBlock)
	}

	return ctx.Formatter.JSON(resp)
}

