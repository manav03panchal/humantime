package runtime

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

// Common errors.
var (
	ErrNoActiveTracking = errors.New("no active tracking")
	ErrProjectRequired  = errors.New("project is required")
	ErrInvalidSID       = errors.New("invalid simplified ID")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrEndBeforeStart   = errors.New("end time must be after start time")
	ErrBlockNotFound    = errors.New("block not found")
	ErrProjectNotFound  = errors.New("project not found")
	ErrInvalidColor     = errors.New("invalid color format (use #RRGGBB)")
	ErrInvalidDuration  = errors.New("invalid duration")
	ErrDiskFull         = errors.New("disk full: unable to write to database")
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
	ErrNoActiveTracking: "Use 'ht start <project>' to begin tracking.",
	ErrProjectRequired:  "Specify a project as the first argument.",
	ErrInvalidSID:       "SIDs must be alphanumeric with dashes, underscores, or periods (max 32 chars).",
	ErrInvalidTimestamp: "Try formats like '2 hours ago', 'yesterday at 3pm', or '9am'.",
	ErrEndBeforeStart:   "Check your timestamps - end must come after start.",
	ErrBlockNotFound:    "Use 'ht blocks' to see available blocks.",
	ErrProjectNotFound:  "Use 'ht projects' to see available projects.",
	ErrInvalidColor:     "Use hex color format like '#FF5733' or '#00FF00'.",
	ErrDiskFull:         "Free up disk space and try again. Your active tracking state is preserved in memory.",
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

// DiskFullError represents a disk full condition with additional context.
type DiskFullError struct {
	Op      string // The operation that failed (e.g., "write", "sync")
	Path    string // The path involved, if known
	wrapped error  // The underlying error
}

func (e *DiskFullError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("disk full during %s on %s: %v", e.Op, e.Path, e.wrapped)
	}
	return fmt.Sprintf("disk full during %s: %v", e.Op, e.wrapped)
}

func (e *DiskFullError) Unwrap() error {
	return ErrDiskFull
}

// NewDiskFullError creates a new DiskFullError.
func NewDiskFullError(op, path string, err error) *DiskFullError {
	return &DiskFullError{
		Op:      op,
		Path:    path,
		wrapped: err,
	}
}

// IsDiskFullError checks if an error indicates a disk full condition.
// It checks for ENOSPC (Linux/macOS) and common disk full error patterns.
func IsDiskFullError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's already our DiskFullError
	var diskFullErr *DiskFullError
	if errors.As(err, &diskFullErr) {
		return true
	}

	// Check if it's our sentinel error
	if errors.Is(err, ErrDiskFull) {
		return true
	}

	// Check for ENOSPC (no space left on device)
	var errno syscall.Errno
	if errors.As(err, &errno) {
		if errno == syscall.ENOSPC {
			return true
		}
	}

	// Check error message for disk full patterns
	errStr := strings.ToLower(err.Error())
	diskFullPatterns := []string{
		"no space left on device",
		"disk full",
		"enospc",
		"not enough space",
		"insufficient disk space",
		"out of disk space",
	}

	for _, pattern := range diskFullPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// WrapDiskFullError wraps an error as a DiskFullError if it indicates disk full.
// If the error is not a disk full error, it returns the original error unchanged.
func WrapDiskFullError(err error, op, path string) error {
	if err == nil {
		return nil
	}
	if IsDiskFullError(err) {
		return NewDiskFullError(op, path, err)
	}
	return err
}
