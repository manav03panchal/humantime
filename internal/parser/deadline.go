package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/markusmobius/go-dateparser"
)

// DeadlineResult holds the parsed deadline and any error.
type DeadlineResult struct {
	Time  time.Time
	Error error
}

// relativeRegex matches relative time expressions like "+5m", "+1h", "+2d".
var relativeRegex = regexp.MustCompile(`^\+(\d+)([smhdw])$`)

// ParseDeadline parses a natural language deadline expression.
// Supports formats like:
//   - "+5m", "+1h", "+2d" (relative)
//   - "friday 5pm", "tomorrow 2pm" (natural language)
//   - "2026-01-15 14:00" (ISO format)
func ParseDeadline(input string) DeadlineResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return DeadlineResult{Error: fmt.Errorf("deadline is required")}
	}

	// Check for relative time format (+5m, +1h, etc.)
	if match := relativeRegex.FindStringSubmatch(input); match != nil {
		return parseRelativeDeadline(match[1], match[2])
	}

	// Use go-dateparser for natural language parsing
	cfg := &dateparser.Configuration{
		CurrentTime: time.Now(),
	}

	result, err := dateparser.Parse(cfg, input)
	if err != nil {
		return DeadlineResult{Error: fmt.Errorf("could not parse deadline %q", input)}
	}

	// Ensure the date is in the future
	if result.Time.Before(time.Now()) {
		// If it's today but in the past, try to interpret as tomorrow
		if isSameDay(result.Time, time.Now()) {
			result.Time = result.Time.AddDate(0, 0, 1)
		} else {
			return DeadlineResult{Error: fmt.Errorf("deadline must be in the future")}
		}
	}

	return DeadlineResult{Time: result.Time}
}

// parseRelativeDeadline parses relative time expressions.
func parseRelativeDeadline(numStr, unit string) DeadlineResult {
	num, _ := strconv.Atoi(numStr)
	if num <= 0 {
		return DeadlineResult{Error: fmt.Errorf("invalid duration: must be positive")}
	}

	var d time.Duration
	switch unit {
	case "s":
		d = time.Duration(num) * time.Second
	case "m":
		d = time.Duration(num) * time.Minute
	case "h":
		d = time.Duration(num) * time.Hour
	case "d":
		d = time.Duration(num) * 24 * time.Hour
	case "w":
		d = time.Duration(num) * 7 * 24 * time.Hour
	default:
		return DeadlineResult{Error: fmt.Errorf("invalid time unit: %s", unit)}
	}

	return DeadlineResult{Time: time.Now().Add(d)}
}

// isSameDay checks if two times are on the same day.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// ParseDeadlineArgs parses deadline from command arguments.
// Joins multiple args into a single string for natural language parsing.
func ParseDeadlineArgs(args []string) DeadlineResult {
	if len(args) == 0 {
		return DeadlineResult{Error: fmt.Errorf("deadline is required")}
	}
	return ParseDeadline(strings.Join(args, " "))
}

// FormatDeadline formats a deadline for display.
func FormatDeadline(t time.Time) string {
	now := time.Now()
	diff := time.Until(t)

	// Format the date part
	var datePart string
	if isSameDay(t, now) {
		datePart = "Today"
	} else if isSameDay(t, now.AddDate(0, 0, 1)) {
		datePart = "Tomorrow"
	} else if diff < 7*24*time.Hour {
		datePart = t.Format("Monday")
	} else {
		datePart = t.Format("Mon, Jan 2")
	}

	// Format the time part
	timePart := t.Format("3:04 PM")

	return fmt.Sprintf("%s at %s", datePart, timePart)
}

// FormatTimeUntil formats the duration until a deadline.
func FormatTimeUntil(t time.Time) string {
	diff := time.Until(t)
	if diff < 0 {
		return "overdue"
	}

	if diff < time.Minute {
		return "less than a minute"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		mins := int(diff.Minutes()) % 60
		if hours == 1 {
			if mins > 0 {
				return fmt.Sprintf("in 1 hour %d minutes", mins)
			}
			return "in 1 hour"
		}
		if mins > 0 {
			return fmt.Sprintf("in %d hours %d minutes", hours, mins)
		}
		return fmt.Sprintf("in %d hours", hours)
	}
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "in 1 day"
		}
		return fmt.Sprintf("in %d days", days)
	}

	weeks := int(diff.Hours() / (24 * 7))
	if weeks == 1 {
		return "in 1 week"
	}
	return fmt.Sprintf("in %d weeks", weeks)
}
