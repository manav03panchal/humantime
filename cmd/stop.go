package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
)

// Stop command flags.
var (
	stopFlagNote string
	stopFlagEnd  string
)

// stopCmd represents the stop command.
var stopCmd = &cobra.Command{
	Use:     "stop [with note 'NOTE'] [TIMESTAMP]",
	Aliases: []string{"stp", "e", "end", "pause"},
	Short:   "Stop the current time tracking",
	Long: `Stop the currently active time tracking. Optionally add or update the note
and specify a custom end time.

Examples:
  humantime stop
  humantime stop with note 'completed the feature'
  humantime stop 10 minutes ago
  humantime stop at 5pm with note 'done for the day'`,
	RunE: runStop,
}

func init() {
	stopCmd.Flags().StringVarP(&stopFlagNote, "note", "n", "", "Note for the block (appends to existing)")
	stopCmd.Flags().StringVarP(&stopFlagEnd, "end", "e", "", "End timestamp")
}

func runStop(cmd *cobra.Command, args []string) error {
	// Get active block
	block, err := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if err != nil {
		return err
	}
	if block == nil {
		if ctx.IsJSON() {
			return ctx.JSONFormatter().PrintError("no_active_tracking", "No active tracking to stop", "")
		}
		ctx.CLIFormatter().PrintNoActiveTracking()
		return nil
	}

	// Parse arguments for note and end time
	parsed := parser.Parse(args)
	parsed.Merge("", "", stopFlagNote, "", stopFlagEnd)
	if err := parsed.Process(); err != nil {
		return err
	}

	// Update end time
	if parsed.HasEnd {
		block.TimestampEnd = parsed.TimestampEnd
	} else {
		block.TimestampEnd = parsed.TimestampStart // Uses "now" by default
	}

	// Validate end time is after start
	if block.TimestampEnd.Before(block.TimestampStart) {
		return runtime.ErrEndBeforeStart
	}

	// Validate end time is not in the future
	if block.TimestampEnd.After(time.Now().Add(time.Minute)) {
		return runtime.NewValidationError("stop", "end time cannot be in the future")
	}

	// Update note if provided
	if parsed.HasNote {
		if block.Note != "" {
			block.Note += " - " + parsed.Note
		} else {
			block.Note = parsed.Note
		}
	}

	// Save the block
	if err := ctx.BlockRepo.Update(block); err != nil {
		return err
	}

	// Save undo state (save after update so we have the final state)
	if err := ctx.UndoRepo.SaveUndoStop(block); err != nil {
		// Non-fatal error
		ctx.Debugf("Failed to save undo state: %v", err)
	}

	// Clear active tracking
	if err := ctx.ActiveBlockRepo.ClearActive(); err != nil {
		return err
	}

	// Output result
	if ctx.IsJSON() {
		return ctx.JSONFormatter().PrintStop(block)
	}

	ctx.CLIFormatter().PrintTrackingStopped(block)
	return nil
}
