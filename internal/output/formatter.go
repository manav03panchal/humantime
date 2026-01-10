// Package output provides output formatting for Humantime.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mattn/go-isatty"
)

// Format represents the output format type.
type Format string

const (
	FormatCLI   Format = "cli"
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
)

// ColorMode represents the color output mode.
type ColorMode string

const (
	ColorAuto   ColorMode = "auto"
	ColorAlways ColorMode = "always"
	ColorNever  ColorMode = "never"
)

// Formatter handles output formatting.
type Formatter struct {
	Writer    io.Writer
	Format    Format
	ColorMode ColorMode
	NoNewline bool
}

// NewFormatter creates a new formatter with default settings.
func NewFormatter() *Formatter {
	return &Formatter{
		Writer:    os.Stdout,
		Format:    FormatCLI,
		ColorMode: ColorAuto,
	}
}

// IsColorEnabled returns true if color output is enabled.
func (f *Formatter) IsColorEnabled() bool {
	switch f.ColorMode {
	case ColorAlways:
		return true
	case ColorNever:
		return false
	default:
		// Auto-detect based on terminal
		if w, ok := f.Writer.(*os.File); ok {
			return isatty.IsTerminal(w.Fd()) || isatty.IsCygwinTerminal(w.Fd())
		}
		return false
	}
}

// Print outputs formatted text.
func (f *Formatter) Print(a ...interface{}) {
	fmt.Fprint(f.Writer, a...)
}

// Println outputs formatted text with newline.
func (f *Formatter) Println(a ...interface{}) {
	fmt.Fprintln(f.Writer, a...)
}

// Printf outputs formatted text.
func (f *Formatter) Printf(format string, a ...interface{}) {
	fmt.Fprintf(f.Writer, format, a...)
}

// JSON outputs data as JSON.
func (f *Formatter) JSON(v interface{}) error {
	encoder := json.NewEncoder(f.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// PrintJSON is an alias for JSON for consistency.
func (f *Formatter) PrintJSON(v interface{}) error {
	return f.JSON(v)
}

// FormatDuration formats a duration in human-readable form.
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// FormatDurationShort formats a duration in short form (e.g., "2h 15m").
func FormatDurationShort(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// FormatTime formats a time in local timezone.
func FormatTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

// FormatTimeShort formats a time without seconds.
func FormatTimeShort(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}

// FormatDate formats a date only.
func FormatDate(t time.Time) string {
	return t.Local().Format("2006-01-02")
}

// FormatTimeOnly formats time without date.
func FormatTimeOnly(t time.Time) string {
	return t.Local().Format("15:04")
}
