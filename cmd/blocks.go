package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
	"github.com/manav03panchal/humantime/internal/storage"
)

// Blocks command flags.
var (
	blocksFlagProject string
	blocksFlagTask    string
	blocksFlagFrom    string
	blocksFlagUntil   string
	blocksFlagLimit   int
)

// blocksCmd represents the blocks command.
var blocksCmd = &cobra.Command{
	Use:     "blocks [BLOCK_ID] [on PROJECT[/TASK]] [TIMEFRAME]",
	Aliases: []string{"block", "blk", "b"},
	Short:   "View time blocks",
	Long: `List time blocks with optional filtering by project, task, or time range.
Can also show a specific block by ID.

Examples:
  humantime blocks
  humantime blocks on clientwork
  humantime blocks on clientwork/bugfix
  humantime blocks from last week
  humantime blocks on clientwork from monday to friday
  humantime blocks today
  humantime blocks this month`,
	RunE: runBlocks,
}

// blocksEditCmd represents the blocks edit command.
var blocksEditCmd = &cobra.Command{
	Use:   "edit BLOCK_ID",
	Short: "Edit a time block",
	Long: `Edit an existing time block. You can update the note, start time, or end time.

Examples:
  humantime blocks edit abc123 --note "updated note"
  humantime blocks edit abc123 --start "9am" --end "11am"`,
	Args: cobra.ExactArgs(1),
	RunE: runBlocksEdit,
}

// Blocks edit flags.
var (
	blocksEditFlagNote  string
	blocksEditFlagStart string
	blocksEditFlagEnd   string
)

func init() {
	// List flags
	blocksCmd.Flags().StringVarP(&blocksFlagProject, "project", "p", "", "Filter by project SID")
	blocksCmd.Flags().StringVarP(&blocksFlagTask, "task", "t", "", "Filter by task SID")
	blocksCmd.Flags().StringVar(&blocksFlagFrom, "from", "", "Start of time range")
	blocksCmd.Flags().StringVar(&blocksFlagUntil, "until", "", "End of time range")
	blocksCmd.Flags().IntVarP(&blocksFlagLimit, "limit", "l", 50, "Maximum blocks to show")

	// Dynamic completion for projects/tasks
	blocksCmd.ValidArgsFunction = completeBlocksArgs
	blocksCmd.RegisterFlagCompletionFunc("project", completeProjects)

	// Edit flags
	blocksEditCmd.Flags().StringVarP(&blocksEditFlagNote, "note", "n", "", "Update note")
	blocksEditCmd.Flags().StringVarP(&blocksEditFlagStart, "start", "s", "", "Update start timestamp")
	blocksEditCmd.Flags().StringVarP(&blocksEditFlagEnd, "end", "e", "", "Update end timestamp")

	blocksCmd.AddCommand(blocksEditCmd)
	rootCmd.AddCommand(blocksCmd)
}

func runBlocks(cmd *cobra.Command, args []string) error {
	// Parse arguments for project/task and timeframe
	parsed := parser.Parse(args)
	parsed.Merge(blocksFlagProject, blocksFlagTask, "", blocksFlagFrom, blocksFlagUntil)

	// Check if first arg is a block ID (UUID format)
	if len(args) > 0 && isBlockID(args[0]) {
		return showSingleBlock(args[0])
	}

	// Process parsed arguments
	if err := parsed.Process(); err != nil {
		return err
	}

	// Build filter
	filter := storage.BlockFilter{
		ProjectSID: parsed.ProjectSID,
		TaskSID:    parsed.TaskSID,
		Limit:      blocksFlagLimit,
	}

	// Apply time range from parsed timestamps
	if parsed.HasStart {
		filter.StartAfter = parsed.TimestampStart
	}
	if parsed.HasEnd {
		filter.EndBefore = parsed.TimestampEnd
	}

	// If period phrase like "today", "this week", get the range
	if len(args) > 0 && !parsed.HasProject {
		periodInput := joinArgs(args)
		if isPeriodPhrase(periodInput) {
			timeRange := parser.GetPeriodRange(periodInput)
			filter.StartAfter = timeRange.Start
			filter.EndBefore = timeRange.End
		}
	}

	// Get blocks
	blocks, err := ctx.BlockRepo.ListFiltered(filter)
	if err != nil {
		return err
	}

	// Get total count
	allBlocks, err := ctx.BlockRepo.List()
	if err != nil {
		return err
	}
	totalCount := len(allBlocks)

	// Output
	if ctx.IsJSON() {
		return ctx.JSONFormatter().PrintBlocks(blocks, totalCount)
	}

	return printBlocksCLI(blocks, totalCount)
}

func showSingleBlock(blockID string) error {
	// Try to find the block by partial key match
	key := "block:" + blockID
	block, err := ctx.BlockRepo.Get(key)
	if err != nil {
		// Try listing and searching
		blocks, err := ctx.BlockRepo.List()
		if err != nil {
			return err
		}
		for _, b := range blocks {
			if containsID(b.Key, blockID) {
				block = b
				break
			}
		}
		if block == nil {
			return runtime.ErrBlockNotFound
		}
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewBlockOutput(block))
	}

	cli := ctx.CLIFormatter()
	cli.Title("Block: " + block.Key)
	cli.Printf("  Project: %s\n", cli.ProjectName(block.ProjectSID))
	if block.TaskSID != "" {
		cli.Printf("  Task: %s\n", cli.TaskName(block.TaskSID))
	}
	if block.Note != "" {
		cli.Printf("  Note: %s\n", cli.Note(block.Note))
	}
	cli.Printf("  Started: %s\n", output.FormatTime(block.TimestampStart))
	if !block.TimestampEnd.IsZero() {
		cli.Printf("  Ended: %s\n", output.FormatTime(block.TimestampEnd))
	}
	cli.Printf("  Duration: %s\n", cli.Duration(output.FormatDuration(block.Duration())))
	if block.IsActive() {
		cli.Success("Currently active")
	}

	return nil
}

func runBlocksEdit(cmd *cobra.Command, args []string) error {
	blockID := args[0]

	// Find the block
	key := "block:" + blockID
	block, err := ctx.BlockRepo.Get(key)
	if err != nil {
		// Try partial match
		blocks, err := ctx.BlockRepo.List()
		if err != nil {
			return err
		}
		for _, b := range blocks {
			if containsID(b.Key, blockID) {
				block = b
				break
			}
		}
		if block == nil {
			return runtime.ErrBlockNotFound
		}
	}

	// Apply updates
	updated := false

	if blocksEditFlagNote != "" {
		block.Note = blocksEditFlagNote
		updated = true
	}

	if blocksEditFlagStart != "" {
		result := parser.ParseTimestamp(blocksEditFlagStart)
		if result.Error != nil {
			return result.Error
		}
		block.TimestampStart = result.Time
		updated = true
	}

	if blocksEditFlagEnd != "" {
		result := parser.ParseTimestamp(blocksEditFlagEnd)
		if result.Error != nil {
			return result.Error
		}
		block.TimestampEnd = result.Time
		updated = true
	}

	if !updated {
		return fmt.Errorf("no updates specified (use --note, --start, or --end)")
	}

	// Validate
	if !block.TimestampEnd.IsZero() && block.TimestampEnd.Before(block.TimestampStart) {
		return runtime.ErrEndBeforeStart
	}

	// Save
	if err := ctx.BlockRepo.Update(block); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewBlockOutput(block))
	}

	ctx.CLIFormatter().Success("Block updated")
	return showSingleBlock(blockID)
}

func printBlocksCLI(blocks []*model.Block, totalCount int) error {
	cli := ctx.CLIFormatter()

	if len(blocks) == 0 {
		cli.Muted("No blocks found.")
		return nil
	}

	cli.Title(fmt.Sprintf("Time Blocks (showing %d of %d)", len(blocks), totalCount))
	cli.Println("")

	var totalDuration int64
	for _, block := range blocks {
		// Format project/task
		location := cli.FormatProjectTask(block.ProjectSID, block.TaskSID)
		duration := output.FormatDuration(block.Duration())
		totalDuration += block.DurationSeconds()

		// Print block
		cli.Printf("%s  %s\n", location, cli.Duration(duration))
		if block.Note != "" {
			cli.Printf("  %s\n", cli.Note(block.Note))
		}
		if block.IsActive() {
			cli.Printf("  %s - (active)\n", output.FormatTimeShort(block.TimestampStart))
		} else {
			cli.Printf("  %s - %s\n",
				output.FormatTimeShort(block.TimestampStart),
				output.FormatTimeOnly(block.TimestampEnd))
		}
		cli.Println("")
	}

	cli.Printf("Total: %s\n", cli.Duration(output.FormatDuration(time.Duration(totalDuration)*time.Second)))
	return nil
}

// Helper functions

func isBlockID(s string) bool {
	// Block IDs are UUIDs - check for UUID-like pattern
	if len(s) < 8 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-') {
			return false
		}
	}
	return true
}

func containsID(key, partial string) bool {
	return len(key) >= len(partial) && key[len(key)-len(partial):] == partial
}

func isPeriodPhrase(s string) bool {
	periods := []string{"today", "yesterday", "this week", "last week", "this month", "last month", "this year", "last year"}
	for _, p := range periods {
		if s == p {
			return true
		}
	}
	return false
}

func joinArgs(args []string) string {
	result := ""
	for i, a := range args {
		if i > 0 {
			result += " "
		}
		result += a
	}
	return result
}
