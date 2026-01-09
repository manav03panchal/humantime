package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
)

// Start command flags.
var (
	startFlagProject string
	startFlagTask    string
	startFlagNote    string
	startFlagStart   string
	startFlagEnd     string
)

// startCmd represents the start command.
var startCmd = &cobra.Command{
	Use:     "start [on PROJECT[/TASK]] [with note 'NOTE'] [TIMESTAMP]",
	Aliases: []string{"sta", "str", "s", "started", "switch", "sw"},
	Short:   "Start tracking time on a project",
	Long: `Start tracking time on a project, optionally with a task and note.

If tracking is already active, the current block is ended and a new one begins.

Examples:
  humantime start on myproject
  humantime start on clientwork/bugfix
  humantime start on clientwork with note 'fixing login issue'
  humantime start on myproject 2 hours ago
  humantime start on myproject from 9am to 11am`,
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
	startCmd.Flags().StringVarP(&startFlagTask, "task", "t", "", "Task SID")
	startCmd.Flags().StringVarP(&startFlagNote, "note", "n", "", "Note for the block")
	startCmd.Flags().StringVarP(&startFlagStart, "start", "s", "", "Start timestamp")
	startCmd.Flags().StringVarP(&startFlagEnd, "end", "e", "", "End timestamp (creates completed block)")

	// Add resume as subcommand
	startCmd.AddCommand(resumeCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	// Parse arguments
	parsed := parser.Parse(args)

	// Merge flags (flags override parsed args)
	parsed.Merge(startFlagProject, startFlagTask, startFlagNote, startFlagStart, startFlagEnd)

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
	if parsed.TaskSID != "" && !parser.ValidateSID(parsed.TaskSID) {
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

	// Ensure task exists if specified (auto-create if needed)
	if parsed.TaskSID != "" {
		_, _, err := ctx.TaskRepo.GetOrCreate(parsed.ProjectSID, parsed.TaskSID, parsed.TaskSID)
		if err != nil {
			return err
		}
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
		ctx.Config.UserKey,
		parsed.ProjectSID,
		parsed.TaskSID,
		parsed.Note,
		parsed.TimestampStart,
	)

	// If end time specified, create completed block
	if !parsed.TimestampEnd.IsZero() {
		block.TimestampEnd = parsed.TimestampEnd
	}

	// Save the block
	if err := ctx.BlockRepo.Create(block); err != nil {
		return err
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

	// Start tracking on the same project/task
	block := model.NewBlock(
		ctx.Config.UserKey,
		previousBlock.ProjectSID,
		previousBlock.TaskSID,
		"", // No note for resumed blocks
		time.Now(),
	)

	if err := ctx.BlockRepo.Create(block); err != nil {
		return err
	}

	if err := ctx.ActiveBlockRepo.SetActive(block.Key); err != nil {
		return err
	}

	// Output result
	if ctx.IsJSON() {
		return ctx.JSONFormatter().PrintStart(block, nil)
	}

	cli := ctx.CLIFormatter()
	cli.Printf("Resumed tracking on %s\n", cli.FormatProjectTask(block.ProjectSID, block.TaskSID))
	cli.Printf("  Started: %s\n", block.TimestampStart.Format("2006-01-02 15:04:05"))
	return nil
}
