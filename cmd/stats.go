package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Stats command flags.
var (
	statsFlagProject string
	statsFlagTask    string
	statsFlagFrom    string
	statsFlagUntil   string
	statsFlagGroup   string
	statsFlagTag     string
)

// statsCmd represents the stats command.
var statsCmd = &cobra.Command{
	Use:     "stats [on PROJECT[/TASK]] [TIMEFRAME]",
	Aliases: []string{"stat", "stt"},
	Short:   "Show time statistics",
	Long: `Show aggregated time statistics grouped by day, week, or month.

Examples:
  humantime stats
  humantime stats today
  humantime stats this week
  humantime stats on clientwork from last month
  humantime stats on clientwork/bugfix this quarter`,
	RunE: runStats,
}

func init() {
	statsCmd.Flags().StringVarP(&statsFlagProject, "project", "p", "", "Filter by project SID")
	statsCmd.Flags().StringVarP(&statsFlagTask, "task", "t", "", "Filter by task SID")
	statsCmd.Flags().StringVar(&statsFlagFrom, "from", "", "Start of time range")
	statsCmd.Flags().StringVar(&statsFlagUntil, "until", "", "End of time range")
	statsCmd.Flags().StringVarP(&statsFlagGroup, "group", "g", "auto", "Grouping: day, week, month, auto")
	statsCmd.Flags().StringVar(&statsFlagTag, "tag", "", "Filter by tag")

	// Dynamic completion for projects/tasks
	statsCmd.ValidArgsFunction = completeBlocksArgs
	statsCmd.RegisterFlagCompletionFunc("project", completeProjects)

	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	// Parse arguments
	parsed := parser.Parse(args)
	parsed.Merge(statsFlagProject, statsFlagTask, "", statsFlagFrom, statsFlagUntil)

	// Default to today if no time range specified
	var timeRange parser.TimeRange
	if len(args) > 0 && !parsed.HasProject {
		periodInput := joinArgs(args)
		if isPeriodPhrase(periodInput) {
			timeRange = parser.GetPeriodRange(periodInput)
		}
	}

	if timeRange.Start.IsZero() {
		if parsed.HasStart {
			if err := parsed.Process(); err != nil {
				return err
			}
			timeRange.Start = parsed.TimestampStart
			if parsed.HasEnd {
				timeRange.End = parsed.TimestampEnd
			} else {
				timeRange.End = time.Now()
			}
		} else {
			// Default to today
			now := time.Now()
			timeRange.Start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			timeRange.End = timeRange.Start.AddDate(0, 0, 1)
		}
	}

	// Build filter
	filter := storage.BlockFilter{
		ProjectSID: parsed.ProjectSID,
		TaskSID:    parsed.TaskSID,
		Tag:        statsFlagTag,
		StartAfter: timeRange.Start,
		EndBefore:  timeRange.End,
	}

	// Get blocks
	blocks, err := ctx.BlockRepo.ListFiltered(filter)
	if err != nil {
		return err
	}

	// Calculate aggregates
	projectAggs := storage.AggregateByProject(blocks)

	if ctx.IsJSON() {
		return printStatsJSON(blocks, projectAggs, timeRange)
	}

	return printStatsCLI(blocks, projectAggs, timeRange)
}

func printStatsCLI(blocks []*model.Block, projectAggs []storage.ProjectAggregate, timeRange parser.TimeRange) error {
	cli := ctx.CLIFormatter()

	// Format period label
	periodLabel := formatPeriodLabel(timeRange)
	cli.Title(fmt.Sprintf("Statistics: %s", periodLabel))
	cli.Println("")

	if len(blocks) == 0 {
		cli.Muted("No time tracked in this period.")
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

	cli.Println("By Project:")
	cli.Println("")
	for _, agg := range projectAggs {
		percentage := float64(agg.Duration) / float64(totalDuration) * 100
		barWidth := 20
		bar := output.ProgressBar(percentage, barWidth)

		padding := maxProjectLen - len(agg.ProjectSID)
		cli.Printf("  %s%s  %*s  %s  %5.1f%%\n",
			cli.ProjectName(agg.ProjectSID),
			strings.Repeat(" ", padding),
			maxDurationLen,
			cli.Duration(output.FormatDuration(agg.Duration)),
			bar,
			percentage)
	}

	cli.Println("")
	cli.Println(strings.Repeat("─", maxProjectLen+maxDurationLen+35))
	cli.Printf("  %-*s  %*s\n", maxProjectLen, "Total:", maxDurationLen, cli.Duration(output.FormatDuration(totalDuration)))

	return nil
}

func printStatsJSON(blocks []*model.Block, projectAggs []storage.ProjectAggregate, timeRange parser.TimeRange) error {
	// Calculate total duration
	var totalDuration time.Duration
	for _, agg := range projectAggs {
		totalDuration += agg.Duration
	}

	// Build response
	resp := output.StatsResponse{
		Period: &output.PeriodOutput{
			Start:    timeRange.Start.Format(time.RFC3339),
			End:      timeRange.End.Format(time.RFC3339),
			Grouping: "day",
		},
		Summary: &output.SummaryOutput{
			TotalDurationSeconds: int64(totalDuration.Seconds()),
			ByProject:            make([]*output.ProjectSummaryOutput, len(projectAggs)),
		},
	}

	for i, agg := range projectAggs {
		percentage := float64(agg.Duration) / float64(totalDuration) * 100
		resp.Summary.ByProject[i] = &output.ProjectSummaryOutput{
			ProjectSID:      agg.ProjectSID,
			DurationSeconds: int64(agg.Duration.Seconds()),
			Percentage:      percentage,
		}
	}

	return ctx.Formatter.JSON(resp)
}

func formatPeriodLabel(timeRange parser.TimeRange) string {
	now := time.Now()
	start := timeRange.Start
	end := timeRange.End

	// Check if it's today
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1)
	if start.Equal(todayStart) && end.Equal(todayEnd) {
		return fmt.Sprintf("%s (Today)", start.Format("2006-01-02"))
	}

	// Check if it's yesterday
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	if start.Equal(yesterdayStart) && end.Equal(todayStart) {
		return fmt.Sprintf("%s (Yesterday)", start.Format("2006-01-02"))
	}

	// Check if it's this week
	weekStart := todayStart.AddDate(0, 0, -int(now.Weekday()-time.Monday))
	if now.Weekday() == time.Sunday {
		weekStart = todayStart.AddDate(0, 0, -6)
	}
	weekEnd := weekStart.AddDate(0, 0, 7)
	if start.Equal(weekStart) && end.Equal(weekEnd) {
		return fmt.Sprintf("This Week (%s - %s)", start.Format("Jan 2"), end.AddDate(0, 0, -1).Format("Jan 2, 2006"))
	}

	// Default: show date range
	if start.Format("2006-01-02") == end.AddDate(0, 0, -1).Format("2006-01-02") {
		return start.Format("2006-01-02")
	}
	return fmt.Sprintf("%s - %s", start.Format("Jan 2"), end.AddDate(0, 0, -1).Format("Jan 2, 2006"))
}
