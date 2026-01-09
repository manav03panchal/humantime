// Package timer provides timer functionality for Humantime.
package timer

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// CountdownDisplay handles the visual display of a countdown timer.
type CountdownDisplay struct {
	Writer      io.Writer
	UseColor    bool
	ShowSeconds bool
}

// NewCountdownDisplay creates a new countdown display.
func NewCountdownDisplay() *CountdownDisplay {
	return &CountdownDisplay{
		Writer:      os.Stdout,
		UseColor:    true,
		ShowSeconds: true,
	}
}

// Styles for countdown display.
var (
	timerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")) // Purple

	workStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#10B981")) // Green

	breakStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F59E0B")) // Yellow

	longBreakStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3B82F6")) // Blue

	progressStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")) // Gray

	sessionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")) // Gray

	statusStyle = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#6B7280")) // Gray
)

// SessionType represents the type of pomodoro session.
type SessionType int

const (
	SessionWork SessionType = iota
	SessionBreak
	SessionLongBreak
)

// String returns a string representation of the session type.
func (s SessionType) String() string {
	switch s {
	case SessionWork:
		return "WORK"
	case SessionBreak:
		return "BREAK"
	case SessionLongBreak:
		return "LONG BREAK"
	default:
		return "UNKNOWN"
	}
}

// FormatDuration formats a duration as MM:SS or HH:MM:SS.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// RenderTimer renders the countdown timer display.
func (cd *CountdownDisplay) RenderTimer(remaining time.Duration, total time.Duration, sessionType SessionType, sessionNum int, totalSessions int, paused bool) string {
	// Choose style based on session type
	var typeStyle lipgloss.Style
	switch sessionType {
	case SessionWork:
		typeStyle = workStyle
	case SessionBreak:
		typeStyle = breakStyle
	case SessionLongBreak:
		typeStyle = longBreakStyle
	}

	// Build the display
	var output string

	// Session type header
	if cd.UseColor {
		output += typeStyle.Render(sessionType.String())
	} else {
		output += sessionType.String()
	}

	// Session counter
	sessionInfo := fmt.Sprintf(" [%d/%d]", sessionNum, totalSessions)
	if cd.UseColor {
		output += sessionStyle.Render(sessionInfo)
	} else {
		output += sessionInfo
	}
	output += "\n\n"

	// Timer display
	timeStr := FormatDuration(remaining)
	if cd.UseColor {
		output += timerStyle.Render(timeStr)
	} else {
		output += timeStr
	}
	output += "\n\n"

	// Progress bar
	progress := 1.0 - (float64(remaining) / float64(total))
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	progressBar := cd.renderProgressBar(progress, 30)
	if cd.UseColor {
		output += progressStyle.Render(progressBar)
	} else {
		output += progressBar
	}
	output += "\n\n"

	// Status/controls hint
	var status string
	if paused {
		status = "[PAUSED] Press SPACE to resume, Q to quit"
	} else {
		status = "Press SPACE to pause, S to skip, Q to quit"
	}
	if cd.UseColor {
		output += statusStyle.Render(status)
	} else {
		output += status
	}

	return output
}

// renderProgressBar creates a progress bar string.
func (cd *CountdownDisplay) renderProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	empty := width - filled

	bar := ""
	for i := 0; i < filled; i++ {
		bar += "\u2588" // Full block
	}
	for i := 0; i < empty; i++ {
		bar += "\u2591" // Light shade
	}

	percentage := int(progress * 100)
	return fmt.Sprintf("[%s] %d%%", bar, percentage)
}

// ClearScreen clears the terminal screen.
func (cd *CountdownDisplay) ClearScreen() {
	fmt.Fprint(cd.Writer, "\033[H\033[2J")
}

// MoveCursorHome moves cursor to home position.
func (cd *CountdownDisplay) MoveCursorHome() {
	fmt.Fprint(cd.Writer, "\033[H")
}

// RenderComplete renders a session completion message.
func (cd *CountdownDisplay) RenderComplete(sessionType SessionType, nextType SessionType) string {
	var output string

	// Completion message
	completeMsg := fmt.Sprintf("%s session complete!", sessionType.String())
	if cd.UseColor {
		output += workStyle.Render(completeMsg)
	} else {
		output += completeMsg
	}
	output += "\n\n"

	// Next session hint
	if nextType != sessionType {
		nextMsg := fmt.Sprintf("Starting %s in 3 seconds...", nextType.String())
		if cd.UseColor {
			output += statusStyle.Render(nextMsg)
		} else {
			output += nextMsg
		}
	}

	return output
}

// RenderAllComplete renders the final completion message.
func (cd *CountdownDisplay) RenderAllComplete(totalWorkTime time.Duration, sessionsCompleted int) string {
	var output string

	header := "Pomodoro session complete!"
	if cd.UseColor {
		output += workStyle.Render(header)
	} else {
		output += header
	}
	output += "\n\n"

	stats := fmt.Sprintf("Completed %d work sessions\n", sessionsCompleted)
	stats += fmt.Sprintf("Total work time: %s", FormatDuration(totalWorkTime))
	if cd.UseColor {
		output += sessionStyle.Render(stats)
	} else {
		output += stats
	}

	return output
}
