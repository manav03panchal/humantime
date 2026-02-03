package parser

import (
	"fmt"
	"strings"

	"github.com/manav03panchal/humantime/internal/errors"
)

// TimeParseError represents a time parsing error with helpful suggestions.
type TimeParseError struct {
	Input      string
	Field      string
	Message    string
	Examples   []string
	Suggestion string
}

func (e *TimeParseError) Error() string {
	return fmt.Sprintf("invalid %s '%s': %s", e.Field, e.Input, e.Message)
}

// NewTimeParseError creates a new time parse error with examples.
func NewTimeParseError(field, input, message string, examples ...string) *TimeParseError {
	return &TimeParseError{
		Input:    input,
		Field:    field,
		Message:  message,
		Examples: examples,
	}
}

// FormatWithExamples returns the error message with example suggestions.
func (e *TimeParseError) FormatWithExamples() string {
	var sb strings.Builder
	sb.WriteString(e.Error())

	if len(e.Examples) > 0 {
		sb.WriteString("\n\nValid examples:\n")
		for _, ex := range e.Examples {
			sb.WriteString("  - ")
			sb.WriteString(ex)
			sb.WriteString("\n")
		}
	}

	if e.Suggestion != "" {
		sb.WriteString("\n")
		sb.WriteString(e.Suggestion)
	}

	return sb.String()
}

// DurationExamples provides example duration formats.
var DurationExamples = []string{
	"1h30m",
	"90m",
	"2 hours",
	"30 minutes",
	"1h 30m",
	"2.5h",
}

// TimestampExamples provides example timestamp formats.
var TimestampExamples = []string{
	"9am",
	"5:30pm",
	"14:30",
	"yesterday at 3pm",
	"2 hours ago",
	"now",
}

// DeadlineExamples provides example deadline formats.
var DeadlineExamples = []string{
	"+5m",
	"+1h30m",
	"in 5 minutes",
	"tomorrow at 3pm",
	"friday 5pm",
	"next monday",
}

// DateRangeExamples provides example date range formats.
var DateRangeExamples = []string{
	"today",
	"yesterday",
	"this week",
	"last week",
	"this month",
	"last month",
}

// NewDurationError creates a duration parse error with standard examples.
func NewDurationError(input string) *TimeParseError {
	return &TimeParseError{
		Input:    input,
		Field:    "duration",
		Message:  "could not parse duration",
		Examples: DurationExamples,
		Suggestion: "Durations can be specified as hours (h), minutes (m), or seconds (s).",
	}
}

// NewTimestampError creates a timestamp parse error with standard examples.
func NewTimestampError(input string) *TimeParseError {
	return &TimeParseError{
		Input:    input,
		Field:    "timestamp",
		Message:  "could not parse time",
		Examples: TimestampExamples,
		Suggestion: "Try using natural language like '9am', '2 hours ago', or '14:30'.",
	}
}

// NewDeadlineError creates a deadline parse error with standard examples.
func NewDeadlineError(input string) *TimeParseError {
	return &TimeParseError{
		Input:    input,
		Field:    "deadline",
		Message:  "could not parse deadline",
		Examples: DeadlineExamples,
		Suggestion: "Deadlines can be relative (+5m) or absolute (friday 5pm).",
	}
}

// NewDateRangeError creates a date range parse error with standard examples.
func NewDateRangeError(input string) *TimeParseError {
	return &TimeParseError{
		Input:    input,
		Field:    "date range",
		Message:  "could not parse date range",
		Examples: DateRangeExamples,
		Suggestion: "Use period names like 'today', 'this week', or 'last month'.",
	}
}

// ToUserError converts a TimeParseError to a UserError for consistent handling.
func (e *TimeParseError) ToUserError() *errors.UserError {
	suggestion := e.Suggestion
	if len(e.Examples) > 0 && suggestion == "" {
		suggestion = fmt.Sprintf("Try: %s", strings.Join(e.Examples[:min(3, len(e.Examples))], ", "))
	}

	return errors.NewUserErrorWithField(e.Field, e.Input, e.Message, suggestion)
}

// ValidateAndSuggest validates input and returns helpful error if invalid.
// This is a convenience function for command handlers.
func ValidateAndSuggest(inputType, input string) error {
	switch inputType {
	case "duration":
		result := ParseDuration(input)
		if !result.Valid {
			return NewDurationError(input)
		}
	case "timestamp":
		result := ParseTimestamp(input)
		if result.Error != nil {
			return NewTimestampError(input)
		}
	default:
		return fmt.Errorf("unknown input type: %s", inputType)
	}
	return nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
