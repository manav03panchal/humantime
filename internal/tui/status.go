package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/manav03panchal/humantime/internal/model"
	"github.com/manav03panchal/humantime/internal/output"
)

// StatusComponent displays the current tracking status.
type StatusComponent struct {
	Block     *model.Block
	Width     int
	IsActive  bool
	StartTime time.Time
}

// NewStatusComponent creates a new status component.
func NewStatusComponent(block *model.Block, width int) *StatusComponent {
	sc := &StatusComponent{
		Block: block,
		Width: width,
	}
	if block != nil {
		sc.IsActive = block.IsActive()
		sc.StartTime = block.TimestampStart
	}
	return sc
}

// View renders the status component.
func (sc *StatusComponent) View() string {
	var content strings.Builder

	if sc.Block == nil {
		// No active tracking
		content.WriteString(StyleInactive.Render("Not tracking"))
		content.WriteString("\n\n")
		content.WriteString(StyleSubtitle.Render("Press 's' to start tracking"))

		box := StyleStatusBox.Width(sc.Width - 4)
		return box.Render(content.String())
	}

	// Active tracking
	content.WriteString(StyleActive.Render("\u25CF TRACKING"))
	content.WriteString("\n\n")

	// Project/Task
	content.WriteString(FormatProjectTask(sc.Block.ProjectSID, sc.Block.TaskSID))
	content.WriteString("\n")

	// Duration
	duration := sc.Block.Duration()
	durationStr := output.FormatDuration(duration)
	content.WriteString("\n")
	content.WriteString(StyleDuration.Render(durationStr))
	content.WriteString("\n")

	// Started at
	content.WriteString("\n")
	content.WriteString(StyleSubtitle.Render(fmt.Sprintf("Started: %s", output.FormatTime(sc.Block.TimestampStart))))

	// Note if present
	if sc.Block.Note != "" {
		content.WriteString("\n\n")
		content.WriteString(StyleNote.Render(fmt.Sprintf("\"%s\"", sc.Block.Note)))
	}

	box := StyleActiveStatusBox.Width(sc.Width - 4)
	return box.Render(content.String())
}

// BlocksComponent displays recent time blocks.
type BlocksComponent struct {
	Blocks []*model.Block
	Width  int
	Limit  int
}

// NewBlocksComponent creates a new blocks component.
func NewBlocksComponent(blocks []*model.Block, width, limit int) *BlocksComponent {
	if limit > 0 && len(blocks) > limit {
		blocks = blocks[:limit]
	}
	return &BlocksComponent{
		Blocks: blocks,
		Width:  width,
		Limit:  limit,
	}
}

// View renders the blocks component.
func (bc *BlocksComponent) View() string {
	var content strings.Builder

	content.WriteString(StyleTitle.Render("Recent Blocks"))
	content.WriteString("\n")

	if len(bc.Blocks) == 0 {
		content.WriteString(StyleMuted.Render("No blocks yet"))
	} else {
		for i, block := range bc.Blocks {
			if i > 0 {
				content.WriteString("\n")
			}
			content.WriteString(bc.renderBlock(block))
		}
	}

	box := StyleBlocksBox.Width(bc.Width - 4)
	return box.Render(content.String())
}

func (bc *BlocksComponent) renderBlock(block *model.Block) string {
	var sb strings.Builder

	// Project/Task and duration on the same line
	projectTask := FormatProjectTask(block.ProjectSID, block.TaskSID)
	duration := output.FormatDuration(block.Duration())

	sb.WriteString(projectTask)
	sb.WriteString("  ")
	sb.WriteString(StyleDuration.Render(duration))

	// Time range
	sb.WriteString("\n")
	if block.IsActive() {
		sb.WriteString(StyleSubtitle.Render(fmt.Sprintf("  %s - (active)", output.FormatTimeOnly(block.TimestampStart))))
	} else {
		sb.WriteString(StyleSubtitle.Render(fmt.Sprintf("  %s - %s",
			output.FormatTimeOnly(block.TimestampStart),
			output.FormatTimeOnly(block.TimestampEnd))))
	}

	return sb.String()
}

// StyleMuted is used for muted text (alias for convenience).
var StyleMuted = StyleSubtitle

// GoalComponent displays goal progress.
type GoalComponent struct {
	Goal     *model.Goal
	Current  time.Duration
	Progress model.Progress
	Width    int
}

// NewGoalComponent creates a new goal component.
func NewGoalComponent(goal *model.Goal, current time.Duration, width int) *GoalComponent {
	gc := &GoalComponent{
		Goal:    goal,
		Current: current,
		Width:   width,
	}
	if goal != nil {
		gc.Progress = goal.CalculateProgress(current)
	}
	return gc
}

// View renders the goal component.
func (gc *GoalComponent) View() string {
	if gc.Goal == nil {
		return ""
	}

	var content strings.Builder

	// Title
	goalType := "Daily"
	if gc.Goal.Type == model.GoalTypeWeekly {
		goalType = "Weekly"
	}
	title := fmt.Sprintf("%s Goal: %s", goalType, gc.Goal.ProjectSID)
	content.WriteString(StyleTitle.Render(title))
	content.WriteString("\n\n")

	// Progress bar
	barWidth := gc.Width - 12
	if barWidth < 10 {
		barWidth = 10
	}
	bar := ProgressBar(gc.Progress.Percentage, barWidth)
	content.WriteString(bar)
	content.WriteString("\n")

	// Progress text
	currentStr := output.FormatDuration(gc.Current)
	targetStr := output.FormatDuration(gc.Goal.Target)
	progressText := fmt.Sprintf("%s / %s (%.0f%%)", currentStr, targetStr, gc.Progress.Percentage)

	if gc.Progress.IsComplete {
		content.WriteString(StyleSuccess.Render(progressText))
		content.WriteString("\n")
		content.WriteString(StyleSuccess.Render("\u2713 Goal completed!"))
	} else {
		content.WriteString(StyleSubtitle.Render(progressText))
		content.WriteString("\n")
		remainingStr := output.FormatDuration(gc.Progress.Remaining)
		content.WriteString(StyleSubtitle.Render(fmt.Sprintf("%s remaining", remainingStr)))
	}

	// Choose box style based on completion
	var box lipgloss.Style
	if gc.Progress.IsComplete {
		box = StyleGoalCompleteBox.Width(gc.Width - 4)
	} else {
		box = StyleGoalBox.Width(gc.Width - 4)
	}

	return box.Render(content.String())
}

// HelpBar renders the help bar at the bottom.
func HelpBar() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"s", "start"},
		{"e", "stop"},
		{"r", "refresh"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		part := StyleHelpKey.Render(k.key) + " " + StyleHelpDesc.Render(k.desc)
		parts = append(parts, part)
	}

	return StyleHelp.Render(strings.Join(parts, "  \u2022  "))
}
