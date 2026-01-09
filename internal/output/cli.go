package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/manav03panchal/humantime/internal/model"
)

// Styles for CLI output.
var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED") // Purple
	colorSecondary = lipgloss.Color("#10B981") // Green
	colorMuted     = lipgloss.Color("#6B7280") // Gray
	colorWarning   = lipgloss.Color("#F59E0B") // Yellow
	colorError     = lipgloss.Color("#EF4444") // Red
	colorSuccess   = lipgloss.Color("#10B981") // Green

	// Styles
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorWarning)

	styleError = lipgloss.NewStyle().
			Foreground(colorError)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleBold = lipgloss.NewStyle().
			Bold(true)

	styleProject = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleTask = lipgloss.NewStyle().
			Foreground(colorSecondary)

	styleDuration = lipgloss.NewStyle().
			Bold(true)

	styleNote = lipgloss.NewStyle().
			Italic(true).
			Foreground(colorMuted)
)

// CLIFormatter provides CLI-specific formatting.
type CLIFormatter struct {
	*Formatter
}

// NewCLIFormatter creates a new CLI formatter.
func NewCLIFormatter(f *Formatter) *CLIFormatter {
	return &CLIFormatter{Formatter: f}
}

// Title prints a title.
func (c *CLIFormatter) Title(text string) {
	if c.IsColorEnabled() {
		c.Println(styleTitle.Render(text))
	} else {
		c.Println(text)
	}
}

// Success prints a success message.
func (c *CLIFormatter) Success(text string) {
	if c.IsColorEnabled() {
		c.Println(styleSuccess.Render("✓ " + text))
	} else {
		c.Println("✓ " + text)
	}
}

// Warning prints a warning message.
func (c *CLIFormatter) Warning(text string) {
	if c.IsColorEnabled() {
		c.Println(styleWarning.Render("⚠ " + text))
	} else {
		c.Println("⚠ " + text)
	}
}

// Error prints an error message.
func (c *CLIFormatter) Error(text string) {
	if c.IsColorEnabled() {
		c.Println(styleError.Render("✗ " + text))
	} else {
		c.Println("✗ " + text)
	}
}

// Muted prints muted text.
func (c *CLIFormatter) Muted(text string) {
	if c.IsColorEnabled() {
		c.Println(styleMuted.Render(text))
	} else {
		c.Println(text)
	}
}

// ProjectName formats a project name.
func (c *CLIFormatter) ProjectName(name string) string {
	if c.IsColorEnabled() {
		return styleProject.Render(name)
	}
	return name
}

// TaskName formats a task name.
func (c *CLIFormatter) TaskName(name string) string {
	if c.IsColorEnabled() {
		return styleTask.Render(name)
	}
	return name
}

// Duration formats a duration.
func (c *CLIFormatter) Duration(text string) string {
	if c.IsColorEnabled() {
		return styleDuration.Render(text)
	}
	return text
}

// Note formats a note.
func (c *CLIFormatter) Note(text string) string {
	if c.IsColorEnabled() {
		return styleNote.Render(text)
	}
	return text
}

// FormatProjectTask formats "project/task" notation.
func (c *CLIFormatter) FormatProjectTask(projectSID, taskSID string) string {
	if taskSID == "" {
		return c.ProjectName(projectSID)
	}
	return c.ProjectName(projectSID) + "/" + c.TaskName(taskSID)
}

// PrintTrackingStarted prints a tracking started message.
func (c *CLIFormatter) PrintTrackingStarted(block *model.Block) {
	c.Printf("Started tracking on %s\n", c.FormatProjectTask(block.ProjectSID, block.TaskSID))
	if block.Note != "" {
		c.Printf("  Note: %s\n", c.Note(block.Note))
	}
	c.Printf("  Started: %s\n", FormatTime(block.TimestampStart))

	if !block.TimestampEnd.IsZero() {
		c.Printf("  Ended: %s\n", FormatTime(block.TimestampEnd))
		c.Printf("  Duration: %s\n", c.Duration(FormatDuration(block.Duration())))
	}
}

// PrintTrackingStopped prints a tracking stopped message.
func (c *CLIFormatter) PrintTrackingStopped(block *model.Block) {
	c.Printf("Stopped tracking on %s\n", c.FormatProjectTask(block.ProjectSID, block.TaskSID))
	c.Printf("  Duration: %s\n", c.Duration(FormatDuration(block.Duration())))
	if block.Note != "" {
		c.Printf("  Note: %s\n", c.Note(block.Note))
	}
	c.Printf("  Started: %s\n", FormatTime(block.TimestampStart))
	c.Printf("  Ended: %s\n", FormatTime(block.TimestampEnd))
}

// PrintStatus prints current tracking status.
func (c *CLIFormatter) PrintStatus(block *model.Block) {
	if block == nil {
		c.Muted("No active tracking.")
		c.Muted("Use 'humantime start on <project>' to begin.")
		return
	}

	c.Printf("Currently tracking: %s\n", c.FormatProjectTask(block.ProjectSID, block.TaskSID))
	c.Printf("  Started: %s\n", FormatTime(block.TimestampStart))
	c.Printf("  Duration: %s\n", c.Duration(FormatDuration(block.Duration())))
	if block.Note != "" {
		c.Printf("  Note: %s\n", c.Note(block.Note))
	}
}

// PrintNoActiveTracking prints a message when there's no active tracking.
func (c *CLIFormatter) PrintNoActiveTracking() {
	c.Warning("No active tracking to stop.")
	c.Muted("Use 'humantime start on <project>' to begin tracking.")
}

// ProgressBar creates a simple progress bar.
func ProgressBar(percentage float64, width int) string {
	if percentage > 100 {
		percentage = 100
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(float64(width) * percentage / 100)
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return bar
}

// Table helpers for CLI output.
type TableRow struct {
	Columns []string
}

// PrintTable prints a simple table.
func (c *CLIFormatter) PrintTable(headers []string, rows []TableRow) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, col := range row.Columns {
			if i < len(widths) && len(col) > widths[i] {
				widths[i] = len(col)
			}
		}
	}

	// Print headers
	var headerLine strings.Builder
	for i, h := range headers {
		headerLine.WriteString(fmt.Sprintf("%-*s  ", widths[i], h))
	}
	c.Println(styleBold.Render(headerLine.String()))

	// Print separator
	var sep strings.Builder
	for _, w := range widths {
		sep.WriteString(strings.Repeat("─", w) + "  ")
	}
	c.Println(sep.String())

	// Print rows
	for _, row := range rows {
		var rowLine strings.Builder
		for i, col := range row.Columns {
			if i < len(widths) {
				rowLine.WriteString(fmt.Sprintf("%-*s  ", widths[i], col))
			}
		}
		c.Println(rowLine.String())
	}
}
