package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/manav03panchal/humantime/internal/runtime"
	"github.com/manav03panchal/humantime/internal/timer"
)

// Pomodoro command flags.
var (
	pomodoroFlagWork      int
	pomodoroFlagBreak     int
	pomodoroFlagLong      int
	pomodoroFlagSessions  int
	pomodoroFlagNote      string
)

// pomodoroCmd represents the pomodoro command.
var pomodoroCmd = &cobra.Command{
	Use:     "pomodoro on <project> [--work MINUTES] [--break MINUTES] [--long MINUTES] [--sessions COUNT]",
	Aliases: []string{"pom", "pomo", "tomato"},
	Short:   "Start a Pomodoro timer for focused work",
	Long: `Start a Pomodoro timer that alternates between work and break sessions.

The Pomodoro Technique uses 25-minute work sessions followed by 5-minute breaks.
After 4 work sessions, take a longer 15-minute break.

Time blocks are automatically created for completed work sessions.
Pressing Ctrl+C or Q will quit and create a partial block for the elapsed time.

Keyboard Controls:
  SPACE  Pause/Resume the timer
  S      Skip the current session
  Q      Quit (creates partial block if in work session)
  Ctrl+C Quit (same as Q)

Examples:
  humantime pomodoro on myproject
  humantime pomodoro on clientwork --work 50 --break 10
  humantime pomodoro on writing --sessions 2
  humantime pomo on study --long 20`,
	Args: func(cmd *cobra.Command, args []string) error {
		// We need at least "on <project>"
		if len(args) < 2 {
			return fmt.Errorf("requires 'on <project>' argument")
		}
		if args[0] != "on" {
			return fmt.Errorf("first argument must be 'on', got '%s'", args[0])
		}
		return nil
	},
	RunE: runPomodoro,
}

func init() {
	pomodoroCmd.Flags().IntVarP(&pomodoroFlagWork, "work", "w", 25, "Work session duration in minutes")
	pomodoroCmd.Flags().IntVarP(&pomodoroFlagBreak, "break", "b", 5, "Short break duration in minutes")
	pomodoroCmd.Flags().IntVarP(&pomodoroFlagLong, "long", "l", 15, "Long break duration in minutes")
	pomodoroCmd.Flags().IntVarP(&pomodoroFlagSessions, "sessions", "s", 4, "Number of work sessions (0 for infinite)")
	pomodoroCmd.Flags().StringVarP(&pomodoroFlagNote, "note", "n", "", "Note for the work blocks")

	// Add to root command
	rootCmd.AddCommand(pomodoroCmd)
}

func runPomodoro(cmd *cobra.Command, args []string) error {
	// Parse project from "on <project>"
	projectArg := args[1]
	projectSID, taskSID := parser.ParseProjectTask(projectArg)

	// Validate project SID
	if !parser.ValidateSID(projectSID) {
		return runtime.ErrInvalidSID
	}
	if taskSID != "" && !parser.ValidateSID(taskSID) {
		return runtime.ErrInvalidSID
	}

	// Ensure project exists (auto-create if needed)
	_, _, err := ctx.ProjectRepo.GetOrCreate(projectSID, projectSID)
	if err != nil {
		return err
	}

	// Ensure task exists if specified
	if taskSID != "" {
		_, _, err := ctx.TaskRepo.GetOrCreate(projectSID, taskSID, taskSID)
		if err != nil {
			return err
		}
	}

	// End any active tracking first
	activeBlock, err := ctx.ActiveBlockRepo.GetActiveBlock(ctx.BlockRepo)
	if err != nil {
		return err
	}
	if activeBlock != nil {
		// End the current block
		activeBlock.TimestampEnd = time.Now()
		if err := ctx.BlockRepo.Update(activeBlock); err != nil {
			return err
		}
		if err := ctx.ActiveBlockRepo.ClearActive(); err != nil {
			return err
		}

		cli := ctx.CLIFormatter()
		cli.Muted(fmt.Sprintf("Stopped previous tracking: %s", activeBlock.ProjectSID))
	}

	// Create Pomodoro configuration
	config := timer.PomodoroConfig{
		WorkDuration:       time.Duration(pomodoroFlagWork) * time.Minute,
		BreakDuration:      time.Duration(pomodoroFlagBreak) * time.Minute,
		LongBreakDuration:  time.Duration(pomodoroFlagLong) * time.Minute,
		SessionsBeforeLong: 4,
		TotalSessions:      pomodoroFlagSessions,
	}

	// Create pomodoro timer
	pom := timer.NewPomodoro(config)

	// Track blocks created
	var currentWorkBlock *model.Block
	var blocksCreated []*model.Block

	// Set up callback to handle events
	pom.SetCallback(func(event timer.PomodoroEvent, state timer.PomodoroState) {
		switch event {
		case timer.EventSessionComplete:
			if state.CurrentType == timer.SessionWork && currentWorkBlock != nil {
				// Complete the work block
				currentWorkBlock.TimestampEnd = time.Now()
				if err := ctx.BlockRepo.Update(currentWorkBlock); err != nil {
					// Log error but don't stop
					ctx.Debugf("Error updating block: %v", err)
				}
				blocksCreated = append(blocksCreated, currentWorkBlock)
				currentWorkBlock = nil
			}

		case timer.EventTick:
			// Create work block at start of work session if not exists
			if state.CurrentType == timer.SessionWork && currentWorkBlock == nil && !state.Paused {
				// Only create at the very start of the session
				if state.Remaining >= state.TotalDuration-time.Second {
					currentWorkBlock = model.NewBlock(
						ctx.Config.UserKey,
						projectSID,
						taskSID,
						pomodoroFlagNote,
						time.Now(),
					)
					if err := ctx.BlockRepo.Create(currentWorkBlock); err != nil {
						ctx.Debugf("Error creating block: %v", err)
						currentWorkBlock = nil
					}
				}
			}

		case timer.EventSkipped:
			// If skipping a work session, save partial block
			if state.CurrentType == timer.SessionWork && currentWorkBlock != nil {
				currentWorkBlock.TimestampEnd = time.Now()
				currentWorkBlock.Note = appendNote(currentWorkBlock.Note, "skipped")
				if err := ctx.BlockRepo.Update(currentWorkBlock); err != nil {
					ctx.Debugf("Error updating block: %v", err)
				}
				blocksCreated = append(blocksCreated, currentWorkBlock)
				currentWorkBlock = nil
			}

		case timer.EventQuit:
			// If quitting during work session, save partial block
			if state.CurrentType == timer.SessionWork && currentWorkBlock != nil {
				currentWorkBlock.TimestampEnd = time.Now()
				currentWorkBlock.Note = appendNote(currentWorkBlock.Note, "interrupted")
				if err := ctx.BlockRepo.Update(currentWorkBlock); err != nil {
					ctx.Debugf("Error updating block: %v", err)
				}
				blocksCreated = append(blocksCreated, currentWorkBlock)
				currentWorkBlock = nil
			}
		}
	})

	// Show starting message
	cli := ctx.CLIFormatter()
	cli.Printf("\nStarting Pomodoro on %s\n", cli.FormatProjectTask(projectSID, taskSID))
	cli.Printf("  Work: %d min | Break: %d min | Long: %d min | Sessions: %d\n\n",
		pomodoroFlagWork, pomodoroFlagBreak, pomodoroFlagLong, pomodoroFlagSessions)
	cli.Muted("Press any key to start...")

	// Wait for key press to start
	buf := make([]byte, 1)
	os.Stdin.Read(buf)

	// Run the pomodoro timer
	err = pom.Run(context.Background())
	if err != nil {
		return err
	}

	// Print summary
	fmt.Println()
	state := pom.GetState()

	if len(blocksCreated) > 0 {
		cli.Success(fmt.Sprintf("Created %d time block(s)", len(blocksCreated)))
		cli.Printf("  Total tracked time: %s\n", timer.FormatDuration(state.TotalWorkTime))
	}

	if pom.WasInterrupted() {
		cli.Warning("Session was interrupted early")
	}

	return nil
}

// appendNote appends text to an existing note.
func appendNote(existing, addition string) string {
	if existing == "" {
		return "[" + addition + "]"
	}
	return existing + " [" + addition + "]"
}
