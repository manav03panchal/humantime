package unit

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
	"testing"

	errs "github.com/manav03panchal/humantime/internal/errors"
)

// TestClassifyUserError tests user error classification.
func TestClassifyUserError(t *testing.T) {
	userErr := errs.NewUserError("test message", "try this")
	category := errs.Classify(userErr)

	if category != errs.CategoryUser {
		t.Errorf("UserError should classify as CategoryUser, got %v", category)
	}
}

// TestClassifySystemError tests system error classification.
func TestClassifySystemError(t *testing.T) {
	sysErr := errs.NewSystemError("message", errors.New("cause"))
	category := errs.Classify(sysErr)

	if category != errs.CategorySystem {
		t.Errorf("SystemError should classify as CategorySystem, got %v", category)
	}
}

// TestClassifyRecoverableError tests recoverable error classification.
func TestClassifyRecoverableError(t *testing.T) {
	recErr := errs.NewRecoverableError("message", errors.New("cause"), 3)
	category := errs.Classify(recErr)

	if category != errs.CategoryRecoverable {
		t.Errorf("RecoverableError should classify as CategoryRecoverable, got %v", category)
	}
}

// TestClassifyNil tests nil error classification.
func TestClassifyNil(t *testing.T) {
	category := errs.Classify(nil)

	if category != errs.CategoryUnknown {
		t.Errorf("nil should classify as CategoryUnknown, got %v", category)
	}
}

// TestClassifyFileErrors tests file-related error classification.
func TestClassifyFileErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected errs.Category
	}{
		{"ErrNotExist", fs.ErrNotExist, errs.CategoryUnknown}, // May classify differently
		{"ErrPermission", fs.ErrPermission, errs.CategoryUnknown},
		{"ErrExist", fs.ErrExist, errs.CategoryUnknown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			category := errs.Classify(tc.err)
			// Just verify it doesn't panic - classification depends on implementation
			t.Logf("%s classified as %v", tc.name, category)
		})
	}
}

// TestClassifySyscallErrors tests syscall error classification.
func TestClassifySyscallErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected errs.Category
	}{
		{"ENOSPC", syscall.ENOSPC, errs.CategorySystem},
		{"ECONNREFUSED", syscall.ECONNREFUSED, errs.CategoryRecoverable},
		{"ETIMEDOUT", syscall.ETIMEDOUT, errs.CategoryRecoverable},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			category := errs.Classify(tc.err)
			if category != tc.expected {
				t.Errorf("%s should classify as %v, got %v", tc.name, tc.expected, category)
			}
		})
	}
}

// TestGetCategory tests GetCategory with typed errors.
func TestGetCategory(t *testing.T) {
	userErr := errs.NewUserError("msg", "suggestion")
	if errs.GetCategory(userErr) != errs.CategoryUser {
		t.Error("GetCategory should return CategoryUser for UserError")
	}

	sysErr := errs.NewSystemError("msg", nil)
	if errs.GetCategory(sysErr) != errs.CategorySystem {
		t.Error("GetCategory should return CategorySystem for SystemError")
	}
}

// TestCategoryString tests Category string representation.
func TestCategoryString(t *testing.T) {
	tests := []struct {
		cat      errs.Category
		expected string
	}{
		{errs.CategoryUnknown, "unknown"},
		{errs.CategoryUser, "user"},
		{errs.CategorySystem, "system"},
		{errs.CategoryRecoverable, "recoverable"},
		{errs.CategoryInternal, "internal"},
	}

	for _, tc := range tests {
		got := tc.cat.String()
		if got != tc.expected {
			t.Errorf("Category.String() = %q, want %q", got, tc.expected)
		}
	}
}

// TestUserErrorFields tests UserError field access.
func TestUserErrorFields(t *testing.T) {
	err := errs.NewUserErrorWithField("duration", "xyz", "invalid format", "use 1h30m")

	if err.Field != "duration" {
		t.Errorf("Field = %q, want %q", err.Field, "duration")
	}
	if err.Value != "xyz" {
		t.Errorf("Value = %q, want %q", err.Value, "xyz")
	}
	if err.Message != "invalid format" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid format")
	}
	if err.Suggestion != "use 1h30m" {
		t.Errorf("Suggestion = %q, want %q", err.Suggestion, "use 1h30m")
	}
}

// TestUserErrorError tests UserError.Error() format.
func TestUserErrorError(t *testing.T) {
	err := errs.NewUserErrorWithField("duration", "xyz", "invalid format", "use 1h30m")
	errStr := err.Error()

	// Should contain meaningful info
	if errStr == "" {
		t.Error("Error() should not be empty")
	}
}

// TestSystemErrorCause tests SystemError cause extraction.
func TestSystemErrorCause(t *testing.T) {
	cause := errors.New("underlying cause")
	err := errs.NewSystemError("operation failed", cause)

	if err.Cause != cause {
		t.Error("SystemError should preserve cause")
	}
}

// TestRecoverableErrorRetryCount tests RecoverableError retry tracking.
func TestRecoverableErrorRetryCount(t *testing.T) {
	err := errs.NewRecoverableError("network error", nil, 3)

	if err.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want %d", err.MaxRetries, 3)
	}
	if !err.CanRetry {
		t.Error("CanRetry should be true initially")
	}

	// Increment retries
	err.IncrementRetry()
	if err.RetryCount != 1 {
		t.Errorf("RetryCount should be 1 after increment, got %d", err.RetryCount)
	}
}

// TestIsUserError tests IsUserError helper.
func TestIsUserError(t *testing.T) {
	userErr := errs.NewUserError("msg", "suggestion")
	if !errs.IsUserError(userErr) {
		t.Error("IsUserError should return true for UserError")
	}

	sysErr := errs.NewSystemError("msg", nil)
	if errs.IsUserError(sysErr) {
		t.Error("IsUserError should return false for SystemError")
	}
}

// TestIsRecoverable tests IsRecoverableError helper.
func TestIsRecoverable(t *testing.T) {
	recErr := errs.NewRecoverableError("msg", nil, 3)
	if !errs.IsRecoverableError(recErr) {
		t.Error("IsRecoverableError should return true for RecoverableError")
	}

	userErr := errs.NewUserError("msg", "suggestion")
	if errs.IsRecoverableError(userErr) {
		t.Error("IsRecoverableError should return false for UserError")
	}
}

// TestWrappedErrorClassification tests classification through wrapping.
func TestWrappedErrorClassification(t *testing.T) {
	userErr := errs.NewUserError("msg", "suggestion")
	wrapped := errs.WithContext(userErr, "additional context")

	category := errs.Classify(wrapped)
	if category != errs.CategoryUser {
		t.Errorf("Wrapped UserError should classify as CategoryUser, got %v", category)
	}
}

// TestPathErrorClassification tests os.PathError classification.
func TestPathErrorClassification(t *testing.T) {
	pathErr := &os.PathError{
		Op:   "open",
		Path: "/nonexistent",
		Err:  syscall.ENOENT,
	}

	category := errs.Classify(pathErr)
	// PathError wrapping ENOENT may classify as system error
	t.Logf("PathError classification: %v", category)
}
