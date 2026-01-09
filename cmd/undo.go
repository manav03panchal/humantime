package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
)

// undoCmd represents the undo command.
var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the last action",
	Long: `Undo the last undoable action (start, stop, or delete).

Examples:
  humantime start on project
  humantime undo
  # Removes the block that was just started

  humantime stop
  humantime undo
  # Resumes tracking (reopens the stopped block)

  humantime blocks delete abc123 --force
  humantime undo
  # Restores the deleted block`,
	RunE: runUndo,
}

func init() {
	rootCmd.AddCommand(undoCmd)
}

func runUndo(cmd *cobra.Command, args []string) error {
	// Get undo state
	state, err := ctx.UndoRepo.Get()
	if err != nil {
		return err
	}
	if state == nil {
		if ctx.IsJSON() {
			return ctx.Formatter.JSON(map[string]string{
				"status":  "nothing_to_undo",
				"message": "Nothing to undo",
			})
		}
		ctx.CLIFormatter().Muted("Nothing to undo")
		return nil
	}

	cli := ctx.CLIFormatter()

	switch state.Action {
	case model.UndoActionStart:
		return undoStart(state, cli)
	case model.UndoActionStop:
		return undoStop(state, cli)
	case model.UndoActionDelete:
		return undoDelete(state, cli)
	default:
		ctx.CLIFormatter().Muted("Nothing to undo")
		return nil
	}
}

// undoStart undoes a start action by deleting the created block.
func undoStart(state *model.UndoState, cli *output.CLIFormatter) error {
	// Get the block that was started
	block, err := ctx.BlockRepo.Get(state.BlockKey)
	if err != nil {
		// Block may have already been deleted
		if err := ctx.UndoRepo.Clear(); err != nil {
			return err
		}
		cli.Muted("Nothing to undo (block no longer exists)")
		return nil
	}

	// Delete the block
	if err := ctx.BlockRepo.Delete(block.Key); err != nil {
		return err
	}

	// Clear active tracking if this was the active block
	activeBlock, _ := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if activeBlock != nil && activeBlock.Key == block.Key {
		ctx.ActiveBlockRepo.ClearActive()
	}

	// Clear undo state
	if err := ctx.UndoRepo.Clear(); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(map[string]interface{}{
			"status":     "undone",
			"action":     "start",
			"project":    block.ProjectSID,
			"task":       block.TaskSID,
		})
	}

	cli.Success("Undid start: removed block for " + cli.FormatProjectTask(block.ProjectSID, block.TaskSID))
	return nil
}

// undoStop undoes a stop action by reopening the block (removing end time).
func undoStop(state *model.UndoState, cli *output.CLIFormatter) error {
	// Check if there's already active tracking
	existingActive, _ := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if existingActive != nil {
		if ctx.IsJSON() {
			return ctx.Formatter.JSON(map[string]string{
				"status":  "error",
				"message": "Cannot undo stop: another block is already active",
			})
		}
		cli.Muted("Cannot undo stop: another block is already active. Stop it first.")
		return nil
	}

	// Get the block that was stopped
	block, err := ctx.BlockRepo.Get(state.BlockKey)
	if err != nil {
		// Block may have been deleted
		if err := ctx.UndoRepo.Clear(); err != nil {
			return err
		}
		cli.Muted("Nothing to undo (block no longer exists)")
		return nil
	}

	// Get elapsed time since stop
	elapsed := time.Since(block.TimestampEnd)

	// Reopen the block by clearing end time
	block.TimestampEnd = time.Time{}

	// Save the block
	if err := ctx.BlockRepo.Update(block); err != nil {
		return err
	}

	// Set as active block again
	if err := ctx.ActiveBlockRepo.SetActive(block.Key); err != nil {
		return err
	}

	// Clear undo state
	if err := ctx.UndoRepo.Clear(); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(map[string]interface{}{
			"status":     "undone",
			"action":     "stop",
			"project":    block.ProjectSID,
			"task":       block.TaskSID,
			"resumed_at": time.Now().Format(time.RFC3339),
		})
	}

	elapsedStr := output.FormatDuration(elapsed)
	cli.Success("Undid stop: tracking resumed on " + cli.FormatProjectTask(block.ProjectSID, block.TaskSID) + " (stopped " + elapsedStr + " ago)")
	return nil
}

// undoDelete undoes a delete action by restoring the block from snapshot.
func undoDelete(state *model.UndoState, cli *output.CLIFormatter) error {
	if state.BlockSnapshot == nil {
		if err := ctx.UndoRepo.Clear(); err != nil {
			return err
		}
		cli.Muted("Nothing to undo (no snapshot available)")
		return nil
	}

	// Restore the block from snapshot
	block := state.BlockSnapshot

	// We need to use a special restore that preserves the original key
	if err := ctx.DB.Set(block); err != nil {
		return err
	}

	// Clear undo state
	if err := ctx.UndoRepo.Clear(); err != nil {
		return err
	}

	if ctx.IsJSON() {
		return ctx.Formatter.JSON(map[string]interface{}{
			"status":     "undone",
			"action":     "delete",
			"project":    block.ProjectSID,
			"task":       block.TaskSID,
			"block_key":  block.Key,
		})
	}

	durationStr := output.FormatDuration(block.Duration())
	dateStr := block.TimestampStart.Format("2006-01-02")
	cli.Success("Undid delete: restored block for " + cli.FormatProjectTask(block.ProjectSID, block.TaskSID) + " (" + durationStr + " on " + dateStr + ")")
	return nil
}
