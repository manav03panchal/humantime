package errors

import (
	"errors"
	"syscall"
)

// Category represents the type of error for display and handling purposes.
type Category int

const (
	// CategoryUnknown is the default for unclassified errors.
	CategoryUnknown Category = iota
	// CategoryUser indicates an error the user can fix (bad input, missing args).
	CategoryUser
	// CategorySystem indicates a system-level error (disk full, network down).
	CategorySystem
	// CategoryRecoverable indicates an error that can be automatically retried.
	CategoryRecoverable
	// CategoryInternal indicates an internal bug or unexpected state.
	CategoryInternal
)

// String returns the string representation of the category.
func (c Category) String() string {
	switch c {
	case CategoryUser:
		return "user"
	case CategorySystem:
		return "system"
	case CategoryRecoverable:
		return "recoverable"
	case CategoryInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// Classify determines the category of an error.
func Classify(err error) Category {
	if err == nil {
		return CategoryUnknown
	}

	// Check for our typed errors first
	if IsUserError(err) {
		return CategoryUser
	}
	if IsSystemError(err) {
		return CategorySystem
	}
	if IsRecoverableError(err) {
		return CategoryRecoverable
	}

	// Check for known system errors
	if isSystemLevel(err) {
		return CategorySystem
	}

	// Check for recoverable patterns
	if isRecoverablePattern(err) {
		return CategoryRecoverable
	}

	// Default to unknown
	return CategoryUnknown
}

// isSystemLevel checks if an error is a system-level error.
func isSystemLevel(err error) bool {
	// Check for syscall errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.ENOSPC: // No space left on device
			return true
		case syscall.EACCES, syscall.EPERM: // Permission denied
			return true
		case syscall.ENOENT: // No such file or directory
			return true
		case syscall.EIO: // I/O error
			return true
		case syscall.EROFS: // Read-only filesystem
			return true
		}
	}

	// Check for our sentinel errors
	if errors.Is(err, ErrDiskFull) ||
		errors.Is(err, ErrDatabaseCorrupted) ||
		errors.Is(err, ErrPermissionDenied) {
		return true
	}

	return false
}

// isRecoverablePattern checks if an error matches recoverable patterns.
func isRecoverablePattern(err error) bool {
	// Check for network-related sentinel errors
	if errors.Is(err, ErrNetworkUnavailable) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrLockHeld) {
		return true
	}

	// Check for syscall errors that are typically transient
	var errno syscall.Errno
	if errors.As(err, &errno) {
		switch errno {
		case syscall.EAGAIN: // Resource temporarily unavailable (EWOULDBLOCK is same on Darwin/Linux)
			return true
		case syscall.EINTR: // Interrupted system call
			return true
		case syscall.ETIMEDOUT: // Connection timed out
			return true
		case syscall.ECONNREFUSED: // Connection refused
			return true
		case syscall.ECONNRESET: // Connection reset by peer
			return true
		}
	}

	return false
}

// ClassifiedError wraps an error with its classification.
type ClassifiedError struct {
	Err      error
	Category Category
}

func (e *ClassifiedError) Error() string {
	return e.Err.Error()
}

func (e *ClassifiedError) Unwrap() error {
	return e.Err
}

// WithCategory wraps an error with an explicit category.
func WithCategory(err error, category Category) error {
	if err == nil {
		return nil
	}
	return &ClassifiedError{
		Err:      err,
		Category: category,
	}
}

// GetCategory returns the category of an error.
// If the error was wrapped with WithCategory, returns that category.
// Otherwise, uses Classify to determine the category.
func GetCategory(err error) Category {
	var classified *ClassifiedError
	if errors.As(err, &classified) {
		return classified.Category
	}
	return Classify(err)
}

// IsUserCategory returns true if the error is a user-fixable error.
func IsUserCategory(err error) bool {
	return GetCategory(err) == CategoryUser
}

// IsSystemCategory returns true if the error is a system-level error.
func IsSystemCategory(err error) bool {
	return GetCategory(err) == CategorySystem
}

// IsRecoverableCategory returns true if the error can be automatically retried.
func IsRecoverableCategory(err error) bool {
	return GetCategory(err) == CategoryRecoverable
}

// FormatByCategory returns a user-appropriate error message based on category.
func FormatByCategory(err error) string {
	if err == nil {
		return ""
	}

	category := GetCategory(err)
	msg := err.Error()

	switch category {
	case CategoryUser:
		// User errors: show the message directly, it should be actionable
		if suggestion := GetSuggestion(err); suggestion != "" {
			return msg + "\n\nTry: " + suggestion
		}
		return msg

	case CategorySystem:
		// System errors: provide context about the system issue
		if suggestion := GetSuggestion(err); suggestion != "" {
			return "System error: " + msg + "\n\n" + suggestion
		}
		return "System error: " + msg

	case CategoryRecoverable:
		// Recoverable errors: indicate automatic retry
		return msg + " (will retry automatically)"

	default:
		// Unknown errors: just return the message
		return msg
	}
}
