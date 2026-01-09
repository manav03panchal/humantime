package runtime

import (
	"errors"
	"fmt"
)

// Common errors.
var (
	ErrNoActiveTracking  = errors.New("no active tracking")
	ErrProjectRequired   = errors.New("project is required")
	ErrInvalidSID        = errors.New("invalid simplified ID")
	ErrInvalidTimestamp  = errors.New("invalid timestamp")
	ErrEndBeforeStart    = errors.New("end time must be after start time")
	ErrBlockNotFound     = errors.New("block not found")
	ErrProjectNotFound   = errors.New("project not found")
	ErrTaskNotFound      = errors.New("task not found")
	ErrGoalNotFound      = errors.New("goal not found")
	ErrInvalidColor      = errors.New("invalid color format (use #RRGGBB)")
	ErrInvalidGoalType   = errors.New("invalid goal type (use daily or weekly)")
	ErrInvalidDuration   = errors.New("invalid duration")
)

// ParseError represents a parsing error with context.
type ParseError struct {
	Field   string
	Value   string
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("invalid %s '%s': %s", e.Field, e.Value, e.Message)
}

// NewParseError creates a new parse error.
func NewParseError(field, value, message string) *ParseError {
	return &ParseError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// Suggestions provides helpful suggestions for common errors.
var Suggestions = map[error]string{
	ErrNoActiveTracking: "Use 'humantime start on <project>' to begin tracking.",
	ErrProjectRequired:  "Specify a project with 'on <project>' or use the -p flag.",
	ErrInvalidSID:       "SIDs must be alphanumeric with dashes, underscores, or periods (max 32 chars).",
	ErrInvalidTimestamp: "Try formats like '2 hours ago', 'yesterday at 3pm', or '9am'.",
	ErrEndBeforeStart:   "Check your timestamps - end must come after start.",
	ErrBlockNotFound:    "Use 'humantime blocks' to see available blocks.",
	ErrProjectNotFound:  "Use 'humantime project' to see available projects.",
	ErrTaskNotFound:     "Use 'humantime task' to see tasks for a project.",
	ErrInvalidColor:     "Use hex color format like '#FF5733' or '#00FF00'.",
	ErrInvalidGoalType:  "Use --daily or --weekly to set goal type.",
}

// GetSuggestion returns a suggestion for an error, if available.
func GetSuggestion(err error) string {
	for knownErr, suggestion := range Suggestions {
		if errors.Is(err, knownErr) {
			return suggestion
		}
	}
	return ""
}

// FormatError formats an error with optional suggestion.
func FormatError(err error) string {
	msg := err.Error()
	if suggestion := GetSuggestion(err); suggestion != "" {
		msg += "\n" + suggestion
	}
	return msg
}
