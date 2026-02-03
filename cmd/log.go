package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
	"github.com/manav03panchal/humantime/internal/parser"
)

// Log command flags.
var (
	logFlagProject string
	logFlagNote    string
	logFlagTag     string
)

// logCmd represents the log command.
var logCmd = &cobra.Command{
	Use:     "log PROJECT DURATION [NOTE] [TIME_MODIFIER]",
	Aliases: []string{"l", "add"},
	Short:   "Log a completed time block",
	Long: `Log a completed time block with a specified duration.
Creates a block ending now (or at the specified time) with the given duration.

Duration formats:
  2h, 2hr, 2 hours     - 2 hours
  30m, 30min           - 30 minutes
  1h30m, 1.5h          - 1 hour 30 minutes

Examples:
  ht log clientwork 2h
  ht log clientwork 30m "fixed login issue"
  ht log clientwork 2h --tag billable
  ht log clientwork 3h yesterday
  ht log clientwork 2h "morning work" yesterday`,
	Args: cobra.MinimumNArgs(2),
	RunE: runLog,
}

func init() {
	logCmd.Flags().StringVarP(&logFlagProject, "project", "p", "", "Project SID (alternative to positional)")
	logCmd.Flags().StringVarP(&logFlagNote, "note", "n", "", "Note for the block")
	logCmd.Flags().StringVar(&logFlagTag, "tag", "", "Comma-separated tags (e.g., billable,urgent)")

	logCmd.RegisterFlagCompletionFunc("project", completeProjects)

	rootCmd.AddCommand(logCmd)
}

func runLog(cmd *cobra.Command, args []string) error {
	// Parse project from first argument
	projectSID := parser.NormalizeSID(args[0])

	// Parse duration from second argument
	durationResult := parser.ParseDuration(args[1])
	if !durationResult.Valid {
		return fmt.Errorf("invalid duration: %s (use formats like 2h, 30m, 1h30m)", args[1])
	}

	// Parse remaining arguments
	remainingArgs := args[2:]
	var note string
	var endTime time.Time

	for i := 0; i < len(remainingArgs); i++ {
		arg := remainingArgs[i]

		// Check for quoted note
		if strings.HasPrefix(arg, "\"") || strings.HasPrefix(arg, "'") {
			quote := arg[0:1]
			if strings.HasSuffix(arg, quote) && len(arg) > 2 {
				note = arg[1 : len(arg)-1]
			} else {
				noteBuilder := []string{arg[1:]}
				for j := i + 1; j < len(remainingArgs); j++ {
					if strings.HasSuffix(remainingArgs[j], quote) {
						noteBuilder = append(noteBuilder, remainingArgs[j][:len(remainingArgs[j])-1])
						i = j
						break
					}
					noteBuilder = append(noteBuilder, remainingArgs[j])
					i = j
				}
				note = strings.Join(noteBuilder, " ")
			}
			continue
		}

		// Check for time modifier (yesterday, etc.)
		if isTimeModifier(arg) {
			timeTokens := []string{arg}
			for j := i + 1; j < len(remainingArgs); j++ {
				if isTimeModifier(remainingArgs[j]) || isTimeLikeToken(remainingArgs[j]) {
					timeTokens = append(timeTokens, remainingArgs[j])
					i = j
				} else {
					break
				}
			}
			timeStr := strings.Join(timeTokens, " ")
			result := parser.ParseTimestamp(timeStr)
			if result.Error == nil {
				endTime = result.Time
			}
			continue
		}

		// Unquoted note
		if note == "" {
			note = arg
		}
	}

	// Override with flags
	if logFlagProject != "" {
		projectSID = parser.NormalizeSID(logFlagProject)
	}
	if logFlagNote != "" {
		note = logFlagNote
	}

	// Calculate start and end times
	if endTime.IsZero() {
		endTime = time.Now()
	}
	startTime := endTime.Add(-durationResult.Duration)

	// Ensure project exists (auto-create if needed)
	_, _, err := ctx.ProjectRepo.GetOrCreate(projectSID, projectSID)
	if err != nil {
		return err
	}

	// Create the block
	block := model.NewBlock("", projectSID, "", note, startTime)
	block.TimestampEnd = endTime

	// Add tags if specified
	if logFlagTag != "" {
		tags := strings.Split(logFlagTag, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		block.Tags = tags
	}

	// Save the block
	if err := ctx.BlockRepo.Create(block); err != nil {
		return err
	}

	// Save undo state
	if err := ctx.UndoRepo.SaveUndoStart(block.Key); err != nil {
		ctx.Debugf("Failed to save undo state: %v", err)
	}

	// Output result
	if ctx.IsJSON() {
		return ctx.Formatter.JSON(output.NewBlockOutput(block))
	}

	cli := ctx.CLIFormatter()
	durationStr := output.FormatDuration(durationResult.Duration)
	cli.Success(fmt.Sprintf("Logged %s to %s", durationStr, projectSID))
	cli.Printf("  %s - %s\n", output.FormatTime(block.TimestampStart), output.FormatTimeOnly(block.TimestampEnd))
	if note != "" {
		cli.Printf("  Note: %s\n", cli.Note(note))
	}
	if len(block.Tags) > 0 {
		cli.Printf("  Tags: %s\n", strings.Join(block.Tags, ", "))
	}
	return nil
}

// isTimeModifier checks if a token is a time modifier.
func isTimeModifier(s string) bool {
	modifiers := []string{
		"yesterday", "today", "tomorrow",
		"last", "this", "next",
		"ago", "before",
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
		"morning", "afternoon", "evening", "night",
	}
	sLower := strings.ToLower(s)
	for _, m := range modifiers {
		if sLower == m {
			return true
		}
	}
	return false
}

// isTimeLikeToken checks if a token looks like part of a time expression.
func isTimeLikeToken(s string) bool {
	timeLike := []string{
		"hour", "hours", "minute", "minutes", "day", "days", "week", "weeks",
		"am", "pm",
	}
	sLower := strings.ToLower(s)
	for _, t := range timeLike {
		if sLower == t {
			return true
		}
	}
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		return true
	}
	return false
}
