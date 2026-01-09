// Package tui provides the terminal user interface components for Humantime.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette for the TUI dashboard.
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#10B981") // Green
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorWarning   = lipgloss.Color("#F59E0B") // Yellow
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorActive    = lipgloss.Color("#3B82F6") // Blue
	ColorBorder    = lipgloss.Color("#4B5563") // Dark gray
)

// Base styles for the TUI.
var (
	// StyleTitle is used for section titles.
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// StyleSubtitle is used for subtitles and secondary information.
	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleProject is used for project names.
	StyleProject = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// StyleTask is used for task names.
	StyleTask = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	// StyleDuration is used for duration values.
	StyleDuration = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorActive)

	// StyleNote is used for notes.
	StyleNote = lipgloss.NewStyle().
			Italic(true).
			Foreground(ColorMuted)

	// StyleActive is used for active/tracking status.
	StyleActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	// StyleInactive is used for inactive status.
	StyleInactive = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleWarning is used for warning messages.
	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// StyleError is used for error messages.
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	// StyleSuccess is used for success messages.
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// StyleHelp is used for help text at the bottom.
	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorMuted).
			MarginTop(1)

	// StyleHelpKey is used for keyboard shortcut keys.
	StyleHelpKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	// StyleHelpDesc is used for keyboard shortcut descriptions.
	StyleHelpDesc = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// Box styles for different sections.
var (
	// StyleStatusBox is used for the current status section.
	StyleStatusBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2).
			MarginBottom(1)

	// StyleActiveStatusBox is used when tracking is active.
	StyleActiveStatusBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSuccess).
				Padding(1, 2).
				MarginBottom(1)

	// StyleBlocksBox is used for the recent blocks section.
	StyleBlocksBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2).
			MarginBottom(1)

	// StyleGoalBox is used for the goal progress section.
	StyleGoalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2).
			MarginBottom(1)

	// StyleGoalCompleteBox is used when goal is completed.
	StyleGoalCompleteBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSuccess).
				Padding(1, 2).
				MarginBottom(1)
)

// ProgressBar creates a progress bar string.
func ProgressBar(percentage float64, width int) string {
	if percentage > 100 {
		percentage = 100
	}
	if percentage < 0 {
		percentage = 0
	}

	filled := int(float64(width) * percentage / 100)
	empty := width - filled

	filledStyle := lipgloss.NewStyle().Foreground(ColorSuccess)
	emptyStyle := lipgloss.NewStyle().Foreground(ColorMuted)

	bar := ""
	for i := 0; i < filled; i++ {
		bar += filledStyle.Render("\u2588") // Full block
	}
	for i := 0; i < empty; i++ {
		bar += emptyStyle.Render("\u2591") // Light shade
	}

	return bar
}

// FormatProjectTask formats "project/task" notation with styles.
func FormatProjectTask(projectSID, taskSID string) string {
	if taskSID == "" {
		return StyleProject.Render(projectSID)
	}
	return StyleProject.Render(projectSID) + "/" + StyleTask.Render(taskSID)
}
