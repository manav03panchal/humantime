package cmd

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
)

// Start command flags.
var (
	startFlagProject string
	startFlagNote    string
	startFlagStart   string
	startFlagEnd     string
	startFlagTag     string
)

// startCmd represents the start command.
var startCmd = &cobra.Command{
	Use:     "start [PROJECT] [NOTE] [TIMESTAMP]",
	Aliases: []string{"s", "on"},
	Short:   "Start tracking time on a project",
	Long: `Start tracking time on a project.

If tracking is already active, the current block is ended and a new one begins.

Examples:
  ht start myproject
  ht start clientwork "fixing login issue"
  ht start myproject 2 hours ago`,
	RunE: runStart,
}

// resumeCmd represents the resume command.
var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume tracking on the last project/task",
	RunE:  runResume,
}

func init() {
	// Start flags
	startCmd.Flags().StringVarP(&startFlagProject, "project", "p", "", "Project SID")
	startCmd.Flags().StringVarP(&startFlagNote, "note", "n", "", "Note for the block")
	startCmd.Flags().StringVarP(&startFlagStart, "start", "s", "", "Start timestamp")
	startCmd.Flags().StringVarP(&startFlagEnd, "end", "e", "", "End timestamp (creates completed block)")
	startCmd.Flags().StringVar(&startFlagTag, "tag", "", "Comma-separated tags (e.g., billable,urgent)")

	// Dynamic completion for projects
	startCmd.ValidArgsFunction = completeStartArgs
	startCmd.RegisterFlagCompletionFunc("project", completeProjects)

	// Add resume as subcommand
	startCmd.AddCommand(resumeCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	// Parse arguments
	parsed := parser.Parse(args)

	// Merge flags (flags override parsed args)
	parsed.Merge(startFlagProject, "", startFlagNote, startFlagStart, startFlagEnd)

	// Process parsed arguments
	if err := parsed.Process(); err != nil {
		return err
	}

	// Validate project is specified
	if parsed.ProjectSID == "" {
		return runtime.ErrProjectRequired
	}

	// Validate SID
	if !parser.ValidateSID(parsed.ProjectSID) {
		return runtime.ErrInvalidSID
	}

	// Validate timestamps
	if !parsed.TimestampEnd.IsZero() && parsed.TimestampEnd.Before(parsed.TimestampStart) {
		return runtime.ErrEndBeforeStart
	}

	// Ensure project exists (auto-create if needed)
	_, _, err := ctx.ProjectRepo.GetOrCreate(parsed.ProjectSID, parsed.ProjectSID)
	if err != nil {
		return err
	}

	// End any active tracking first
	var previousBlock *model.Block
	activeBlock, err := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if err != nil {
		return err
	}
	if activeBlock != nil {
		// End the current block
		activeBlock.TimestampEnd = parsed.TimestampStart
		if err := ctx.BlockRepo.Update(activeBlock); err != nil {
			return err
		}
		previousBlock = activeBlock
	}

	// Create new block
	block := model.NewBlock(
		"",
		parsed.ProjectSID,
		"",
		parsed.Note,
		parsed.TimestampStart,
	)

	// Add tags if specified
	if startFlagTag != "" {
		tags := strings.Split(startFlagTag, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		block.Tags = tags
	}

	// If end time specified, create completed block
	if !parsed.TimestampEnd.IsZero() {
		block.TimestampEnd = parsed.TimestampEnd
	}

	// Save the block
	if err := ctx.BlockRepo.Create(block); err != nil {
		return err
	}

	// Save undo state
	if err := ctx.UndoRepo.SaveUndoStart(block.Key); err != nil {
		// Non-fatal error, just log if debug
		ctx.Debugf("Failed to save undo state: %v", err)
	}

	// Update active tracking (only if no end time)
	if parsed.TimestampEnd.IsZero() {
		if err := ctx.ActiveBlockRepo.SetActive(block.Key); err != nil {
			return err
		}
	} else {
		// Clear active tracking since this is a completed block
		if err := ctx.ActiveBlockRepo.ClearActive(); err != nil {
			return err
		}
	}

	// Output result
	if ctx.IsJSON() {
		return ctx.JSONFormatter().PrintStart(block, previousBlock)
	}

	cli := ctx.CLIFormatter()
	if previousBlock != nil {
		cli.Muted("Stopped previous tracking: " + previousBlock.ProjectSID)
	}
	cli.PrintTrackingStarted(block)
	return nil
}

func runResume(cmd *cobra.Command, args []string) error {
	// Get the previous block
	previousBlock, err := ctx.ActiveBlockRepo.GetPreviousBlock(ctx.BlockRepo)
	if err != nil {
		return err
	}
	if previousBlock == nil {
		return runtime.NewValidationError("resume", "no previous tracking to resume")
	}

	// Start tracking on the same project
	block := model.NewBlock(
		"",
		previousBlock.ProjectSID,
		"",
		"", // No note for resumed blocks
		time.Now(),
	)

	if err := ctx.BlockRepo.Create(block); err != nil {
		return err
	}

	// Save undo state
	if err := ctx.UndoRepo.SaveUndoStart(block.Key); err != nil {
		ctx.Debugf("Failed to save undo state: %v", err)
	}

	if err := ctx.ActiveBlockRepo.SetActive(block.Key); err != nil {
		return err
	}

	// Output result
	if ctx.IsJSON() {
		return ctx.JSONFormatter().PrintStart(block, nil)
	}

	cli := ctx.CLIFormatter()
	cli.Printf("Resumed tracking on %s\n", cli.ProjectName(block.ProjectSID))
	cli.Printf("  Started: %s\n", block.TimestampStart.Format("2006-01-02 15:04:05"))
	return nil
}
