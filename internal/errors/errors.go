// Package errors provides consistent error types for the Humantime CLI.
// It defines three main categories: UserError (fixable by user), SystemError (system issues),
// and RecoverableError (can be automatically retried).
package errors

import (
	"errors"
	"fmt"
)

// Standard sentinel errors for common conditions.
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
	ErrReminderNotFound  = errors.New("reminder not found")
	ErrWebhookNotFound   = errors.New("webhook not found")
	ErrInvalidColor      = errors.New("invalid color format")
	ErrInvalidGoalType   = errors.New("invalid goal type")
	ErrInvalidDuration   = errors.New("invalid duration")
	ErrInvalidURL        = errors.New("invalid URL")
	ErrDiskFull          = errors.New("disk full")
	ErrDatabaseCorrupted = errors.New("database corrupted")
	ErrNetworkUnavailable = errors.New("network unavailable")
	ErrLockHeld          = errors.New("database locked by another process")
	ErrTimeout           = errors.New("operation timed out")
	ErrPermissionDenied  = errors.New("permission denied")
)

// UserError represents an error that the user can fix.
// Examples: invalid input, missing required arguments, incorrect format.
type UserError struct {
	Message    string // What happened
	Reason     string // Why it happened (optional)
	Suggestion string // How to fix it
	Field      string // The field/input that caused the error (optional)
	Value      string // The invalid value (optional)
}

func (e *UserError) Error() string {
	msg := e.Message
	if e.Field != "" && e.Value != "" {
		msg = fmt.Sprintf("%s: '%s'", e.Message, e.Value)
	}
	return msg
}

// NewUserError creates a new UserError.
func NewUserError(message, suggestion string) *UserError {
	return &UserError{
		Message:    message,
		Suggestion: suggestion,
	}
}

// NewUserErrorWithField creates a new UserError with field context.
func NewUserErrorWithField(field, value, message, suggestion string) *UserError {
	return &UserError{
		Message:    message,
		Field:      field,
		Value:      value,
		Suggestion: suggestion,
	}
}

// SystemError represents a system-level error that the user cannot directly fix.
// Examples: disk full, network failure, database corruption.
type SystemError struct {
	Message string // What happened
	Cause   error  // The underlying error
	Op      string // The operation that failed (optional)
}

func (e *SystemError) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s during %s", e.Message, e.Op)
	}
	return e.Message
}

func (e *SystemError) Unwrap() error {
	return e.Cause
}

// NewSystemError creates a new SystemError.
func NewSystemError(message string, cause error) *SystemError {
	return &SystemError{
		Message: message,
		Cause:   cause,
	}
}

// NewSystemErrorWithOp creates a new SystemError with operation context.
func NewSystemErrorWithOp(op, message string, cause error) *SystemError {
	return &SystemError{
		Message: message,
		Cause:   cause,
		Op:      op,
	}
}

// RecoverableError represents an error that can be automatically retried.
// Examples: temporary network failure, transient database lock.
type RecoverableError struct {
	Message     string // What happened
	Cause       error  // The underlying error
	RetryCount  int    // Number of retries attempted so far
	MaxRetries  int    // Maximum number of retries allowed
	CanRetry    bool   // Whether retry is still possible
}

func (e *RecoverableError) Error() string {
	if e.RetryCount > 0 {
		return fmt.Sprintf("%s (attempt %d/%d)", e.Message, e.RetryCount, e.MaxRetries)
	}
	return e.Message
}

func (e *RecoverableError) Unwrap() error {
	return e.Cause
}

// NewRecoverableError creates a new RecoverableError.
func NewRecoverableError(message string, cause error, maxRetries int) *RecoverableError {
	return &RecoverableError{
		Message:    message,
		Cause:      cause,
		MaxRetries: maxRetries,
		CanRetry:   true,
	}
}

// IncrementRetry increments the retry count and updates CanRetry.
func (e *RecoverableError) IncrementRetry() {
	e.RetryCount++
	e.CanRetry = e.RetryCount < e.MaxRetries
}

// IsUserError checks if an error is a UserError.
func IsUserError(err error) bool {
	var ue *UserError
	return errors.As(err, &ue)
}

// IsSystemError checks if an error is a SystemError.
func IsSystemError(err error) bool {
	var se *SystemError
	return errors.As(err, &se)
}

// IsRecoverableError checks if an error is a RecoverableError.
func IsRecoverableError(err error) bool {
	var re *RecoverableError
	return errors.As(err, &re)
}

// AsUserError extracts a UserError from an error chain.
func AsUserError(err error) (*UserError, bool) {
	var ue *UserError
	ok := errors.As(err, &ue)
	return ue, ok
}

// AsSystemError extracts a SystemError from an error chain.
func AsSystemError(err error) (*SystemError, bool) {
	var se *SystemError
	ok := errors.As(err, &se)
	return se, ok
}

// AsRecoverableError extracts a RecoverableError from an error chain.
func AsRecoverableError(err error) (*RecoverableError, bool) {
	var re *RecoverableError
	ok := errors.As(err, &re)
	return re, ok
}

// Wrap wraps an error with additional context.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted additional context.
func Wrapf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
