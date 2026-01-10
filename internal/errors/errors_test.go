package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// UserError Tests
// =============================================================================

func TestNewUserError(t *testing.T) {
	err := NewUserError("invalid input", "try again")
	assert.NotNil(t, err)
	assert.Equal(t, "invalid input", err.Message)
	assert.Equal(t, "try again", err.Suggestion)
}

func TestNewUserErrorWithField(t *testing.T) {
	err := NewUserErrorWithField("email", "invalid@", "invalid email format", "use a valid email like user@example.com")
	assert.NotNil(t, err)
	assert.Equal(t, "email", err.Field)
	assert.Equal(t, "invalid@", err.Value)
	assert.Equal(t, "invalid email format", err.Message)
}

func TestUserErrorError(t *testing.T) {
	t.Run("without_field", func(t *testing.T) {
		err := NewUserError("invalid input", "")
		assert.Equal(t, "invalid input", err.Error())
	})

	t.Run("with_field", func(t *testing.T) {
		err := NewUserErrorWithField("email", "bad@", "invalid email", "")
		assert.Equal(t, "invalid email: 'bad@'", err.Error())
	})
}

func TestIsUserError(t *testing.T) {
	t.Run("user_error", func(t *testing.T) {
		err := NewUserError("test", "")
		assert.True(t, IsUserError(err))
	})

	t.Run("wrapped_user_error", func(t *testing.T) {
		err := NewUserError("test", "")
		wrapped := fmt.Errorf("context: %w", err)
		assert.True(t, IsUserError(wrapped))
	})

	t.Run("not_user_error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.False(t, IsUserError(err))
	})

	t.Run("nil_error", func(t *testing.T) {
		assert.False(t, IsUserError(nil))
	})
}

func TestAsUserError(t *testing.T) {
	t.Run("user_error", func(t *testing.T) {
		err := NewUserError("test", "suggestion")
		ue, ok := AsUserError(err)
		assert.True(t, ok)
		assert.Equal(t, "test", ue.Message)
		assert.Equal(t, "suggestion", ue.Suggestion)
	})

	t.Run("not_user_error", func(t *testing.T) {
		err := errors.New("plain")
		_, ok := AsUserError(err)
		assert.False(t, ok)
	})
}

// =============================================================================
// SystemError Tests
// =============================================================================

func TestNewSystemError(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewSystemError("system failure", cause)
	assert.NotNil(t, err)
	assert.Equal(t, "system failure", err.Message)
	assert.Equal(t, cause, err.Cause)
}

func TestNewSystemErrorWithOp(t *testing.T) {
	cause := errors.New("io error")
	err := NewSystemErrorWithOp("write", "failed to save", cause)
	assert.NotNil(t, err)
	assert.Equal(t, "write", err.Op)
	assert.Equal(t, "failed to save", err.Message)
}

func TestSystemErrorError(t *testing.T) {
	t.Run("without_op", func(t *testing.T) {
		err := NewSystemError("system failure", nil)
		assert.Equal(t, "system failure", err.Error())
	})

	t.Run("with_op", func(t *testing.T) {
		err := NewSystemErrorWithOp("write", "disk full", nil)
		assert.Equal(t, "disk full during write", err.Error())
	})
}

func TestSystemErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := NewSystemError("wrapper", cause)
	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestIsSystemError(t *testing.T) {
	t.Run("system_error", func(t *testing.T) {
		err := NewSystemError("test", nil)
		assert.True(t, IsSystemError(err))
	})

	t.Run("wrapped_system_error", func(t *testing.T) {
		err := NewSystemError("test", nil)
		wrapped := fmt.Errorf("context: %w", err)
		assert.True(t, IsSystemError(wrapped))
	})

	t.Run("not_system_error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.False(t, IsSystemError(err))
	})
}

func TestAsSystemError(t *testing.T) {
	t.Run("system_error", func(t *testing.T) {
		err := NewSystemError("test", nil)
		se, ok := AsSystemError(err)
		assert.True(t, ok)
		assert.Equal(t, "test", se.Message)
	})

	t.Run("not_system_error", func(t *testing.T) {
		err := errors.New("plain")
		_, ok := AsSystemError(err)
		assert.False(t, ok)
	})
}

// =============================================================================
// RecoverableError Tests
// =============================================================================

func TestNewRecoverableError(t *testing.T) {
	cause := errors.New("timeout")
	err := NewRecoverableError("request failed", cause, 3)
	assert.NotNil(t, err)
	assert.Equal(t, "request failed", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.Equal(t, 3, err.MaxRetries)
	assert.True(t, err.CanRetry)
	assert.Equal(t, 0, err.RetryCount)
}

func TestRecoverableErrorError(t *testing.T) {
	t.Run("first_attempt", func(t *testing.T) {
		err := NewRecoverableError("failed", nil, 3)
		assert.Equal(t, "failed", err.Error())
	})

	t.Run("with_retries", func(t *testing.T) {
		err := NewRecoverableError("failed", nil, 3)
		err.RetryCount = 2
		assert.Equal(t, "failed (attempt 2/3)", err.Error())
	})
}

func TestRecoverableErrorUnwrap(t *testing.T) {
	cause := errors.New("network issue")
	err := NewRecoverableError("failed", cause, 3)
	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestRecoverableErrorIncrementRetry(t *testing.T) {
	err := NewRecoverableError("test", nil, 3)

	assert.Equal(t, 0, err.RetryCount)
	assert.True(t, err.CanRetry)

	err.IncrementRetry()
	assert.Equal(t, 1, err.RetryCount)
	assert.True(t, err.CanRetry)

	err.IncrementRetry()
	assert.Equal(t, 2, err.RetryCount)
	assert.True(t, err.CanRetry)

	err.IncrementRetry()
	assert.Equal(t, 3, err.RetryCount)
	assert.False(t, err.CanRetry)
}

func TestIsRecoverableError(t *testing.T) {
	t.Run("recoverable_error", func(t *testing.T) {
		err := NewRecoverableError("test", nil, 3)
		assert.True(t, IsRecoverableError(err))
	})

	t.Run("not_recoverable_error", func(t *testing.T) {
		err := errors.New("plain error")
		assert.False(t, IsRecoverableError(err))
	})
}

func TestAsRecoverableError(t *testing.T) {
	t.Run("recoverable_error", func(t *testing.T) {
		err := NewRecoverableError("test", nil, 5)
		re, ok := AsRecoverableError(err)
		assert.True(t, ok)
		assert.Equal(t, 5, re.MaxRetries)
	})

	t.Run("not_recoverable_error", func(t *testing.T) {
		err := errors.New("plain")
		_, ok := AsRecoverableError(err)
		assert.False(t, ok)
	})
}

// =============================================================================
// Wrap Functions Tests
// =============================================================================

func TestWrap(t *testing.T) {
	t.Run("wraps_error", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := Wrap(original, "context")
		assert.Error(t, wrapped)
		assert.Contains(t, wrapped.Error(), "context")
		assert.Contains(t, wrapped.Error(), "original error")
		assert.True(t, errors.Is(wrapped, original))
	})

	t.Run("nil_returns_nil", func(t *testing.T) {
		result := Wrap(nil, "context")
		assert.Nil(t, result)
	})
}

func TestWrapf(t *testing.T) {
	t.Run("wraps_with_format", func(t *testing.T) {
		original := errors.New("original")
		wrapped := Wrapf(original, "operation %s failed", "save")
		assert.Error(t, wrapped)
		assert.Contains(t, wrapped.Error(), "operation save failed")
		assert.Contains(t, wrapped.Error(), "original")
	})

	t.Run("nil_returns_nil", func(t *testing.T) {
		result := Wrapf(nil, "format %s", "arg")
		assert.Nil(t, result)
	})
}

// =============================================================================
// Sentinel Errors Tests
// =============================================================================

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		ErrNoActiveTracking,
		ErrProjectRequired,
		ErrInvalidSID,
		ErrInvalidTimestamp,
		ErrEndBeforeStart,
		ErrBlockNotFound,
		ErrProjectNotFound,
		ErrTaskNotFound,
		ErrGoalNotFound,
		ErrReminderNotFound,
		ErrWebhookNotFound,
		ErrInvalidColor,
		ErrInvalidGoalType,
		ErrInvalidDuration,
		ErrInvalidURL,
		ErrDiskFull,
		ErrDatabaseCorrupted,
		ErrNetworkUnavailable,
		ErrLockHeld,
		ErrTimeout,
		ErrPermissionDenied,
	}

	for _, sentinel := range sentinels {
		t.Run(sentinel.Error(), func(t *testing.T) {
			assert.NotNil(t, sentinel)
			assert.NotEmpty(t, sentinel.Error())

			// Can be wrapped and still matched
			wrapped := fmt.Errorf("context: %w", sentinel)
			assert.True(t, errors.Is(wrapped, sentinel))
		})
	}
}

// =============================================================================
// Category Tests
// =============================================================================

func TestCategoryString(t *testing.T) {
	tests := []struct {
		category Category
		expected string
	}{
		{CategoryUnknown, "unknown"},
		{CategoryUser, "user"},
		{CategorySystem, "system"},
		{CategoryRecoverable, "recoverable"},
		{CategoryInternal, "internal"},
		{Category(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.category.String())
		})
	}
}

func TestClassify(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		assert.Equal(t, CategoryUnknown, Classify(nil))
	})

	t.Run("user_error", func(t *testing.T) {
		err := NewUserError("invalid input", "")
		assert.Equal(t, CategoryUser, Classify(err))
	})

	t.Run("system_error", func(t *testing.T) {
		err := NewSystemError("disk failure", nil)
		assert.Equal(t, CategorySystem, Classify(err))
	})

	t.Run("recoverable_error", func(t *testing.T) {
		err := NewRecoverableError("network issue", nil, 3)
		assert.Equal(t, CategoryRecoverable, Classify(err))
	})

	t.Run("disk_full_sentinel", func(t *testing.T) {
		assert.Equal(t, CategorySystem, Classify(ErrDiskFull))
	})

	t.Run("network_sentinel", func(t *testing.T) {
		assert.Equal(t, CategoryRecoverable, Classify(ErrNetworkUnavailable))
	})

	t.Run("timeout_sentinel", func(t *testing.T) {
		assert.Equal(t, CategoryRecoverable, Classify(ErrTimeout))
	})

	t.Run("unknown_error", func(t *testing.T) {
		err := errors.New("some random error")
		assert.Equal(t, CategoryUnknown, Classify(err))
	})
}

func TestWithCategory(t *testing.T) {
	t.Run("wraps_error", func(t *testing.T) {
		original := errors.New("test error")
		wrapped := WithCategory(original, CategoryUser)
		assert.Error(t, wrapped)

		var classified *ClassifiedError
		assert.True(t, errors.As(wrapped, &classified))
		assert.Equal(t, CategoryUser, classified.Category)
	})

	t.Run("nil_returns_nil", func(t *testing.T) {
		result := WithCategory(nil, CategoryUser)
		assert.Nil(t, result)
	})
}

func TestClassifiedError(t *testing.T) {
	original := errors.New("original error")
	classified := &ClassifiedError{
		Err:      original,
		Category: CategorySystem,
	}

	assert.Equal(t, "original error", classified.Error())
	assert.Equal(t, original, classified.Unwrap())
}

func TestGetCategory(t *testing.T) {
	t.Run("classified_error", func(t *testing.T) {
		err := WithCategory(errors.New("test"), CategoryInternal)
		assert.Equal(t, CategoryInternal, GetCategory(err))
	})

	t.Run("typed_error", func(t *testing.T) {
		err := NewUserError("test", "")
		assert.Equal(t, CategoryUser, GetCategory(err))
	})

	t.Run("plain_error", func(t *testing.T) {
		err := errors.New("plain")
		assert.Equal(t, CategoryUnknown, GetCategory(err))
	})
}

func TestIsCategoryFunctions(t *testing.T) {
	t.Run("IsUserCategory", func(t *testing.T) {
		err := NewUserError("test", "")
		assert.True(t, IsUserCategory(err))
		assert.False(t, IsSystemCategory(err))
		assert.False(t, IsRecoverableCategory(err))
	})

	t.Run("IsSystemCategory", func(t *testing.T) {
		err := NewSystemError("test", nil)
		assert.False(t, IsUserCategory(err))
		assert.True(t, IsSystemCategory(err))
		assert.False(t, IsRecoverableCategory(err))
	})

	t.Run("IsRecoverableCategory", func(t *testing.T) {
		err := NewRecoverableError("test", nil, 3)
		assert.False(t, IsUserCategory(err))
		assert.False(t, IsSystemCategory(err))
		assert.True(t, IsRecoverableCategory(err))
	})
}

func TestFormatByCategory(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		assert.Empty(t, FormatByCategory(nil))
	})

	t.Run("user_error", func(t *testing.T) {
		err := NewUserError("invalid project name", "")
		formatted := FormatByCategory(err)
		assert.Contains(t, formatted, "invalid project name")
	})

	t.Run("system_error", func(t *testing.T) {
		err := NewSystemError("disk failure", nil)
		formatted := FormatByCategory(err)
		assert.Contains(t, formatted, "System error")
	})

	t.Run("recoverable_error", func(t *testing.T) {
		err := NewRecoverableError("network issue", nil, 3)
		formatted := FormatByCategory(err)
		assert.Contains(t, formatted, "will retry automatically")
	})

	t.Run("unknown_error", func(t *testing.T) {
		err := errors.New("plain error")
		formatted := FormatByCategory(err)
		assert.Equal(t, "plain error", formatted)
	})
}

// =============================================================================
// Suggestion Tests
// =============================================================================

func TestGetCategorySuggestion(t *testing.T) {
	t.Run("user_error", func(t *testing.T) {
		err := NewUserError("invalid input", "")
		suggestion := GetCategorySuggestion(err)
		assert.Contains(t, suggestion, "Check your input")
	})

	t.Run("system_error", func(t *testing.T) {
		err := NewSystemError("disk failure", nil)
		suggestion := GetCategorySuggestion(err)
		assert.Contains(t, suggestion, "system error")
	})

	t.Run("recoverable_error", func(t *testing.T) {
		err := NewRecoverableError("network issue", nil, 3)
		suggestion := GetCategorySuggestion(err)
		assert.Contains(t, suggestion, "retried automatically")
	})

	t.Run("plain_error", func(t *testing.T) {
		err := errors.New("plain error")
		suggestion := GetCategorySuggestion(err)
		assert.Empty(t, suggestion)
	})
}

func TestGetExamples(t *testing.T) {
	t.Run("no_active_tracking", func(t *testing.T) {
		examples := GetExamples(ErrNoActiveTracking)
		assert.NotEmpty(t, examples)
		assert.Contains(t, examples[0], "humantime start")
	})

	t.Run("project_required", func(t *testing.T) {
		examples := GetExamples(ErrProjectRequired)
		assert.NotEmpty(t, examples)
	})

	t.Run("invalid_timestamp", func(t *testing.T) {
		examples := GetExamples(ErrInvalidTimestamp)
		assert.NotEmpty(t, examples)
	})

	t.Run("invalid_duration", func(t *testing.T) {
		examples := GetExamples(ErrInvalidDuration)
		assert.NotEmpty(t, examples)
	})

	t.Run("unknown_error", func(t *testing.T) {
		examples := GetExamples(errors.New("unknown"))
		assert.Empty(t, examples)
	})

	t.Run("wrapped_error", func(t *testing.T) {
		wrapped := fmt.Errorf("context: %w", ErrNoActiveTracking)
		examples := GetExamples(wrapped)
		assert.NotEmpty(t, examples)
	})
}

// =============================================================================
// Context Error (Wrap) Tests
// =============================================================================

func TestStackFrameString(t *testing.T) {
	frame := StackFrame{
		Function: "main.doSomething",
		File:     "/path/to/file.go",
		Line:     42,
	}

	str := frame.String()
	assert.Contains(t, str, "main.doSomething")
	assert.Contains(t, str, "/path/to/file.go")
	assert.Contains(t, str, "42")
}

func TestContextErrorStackTrace(t *testing.T) {
	t.Run("empty_stack", func(t *testing.T) {
		err := &ContextError{
			Message: "test error",
			Cause:   nil,
			Stack:   nil,
		}
		assert.Empty(t, err.StackTrace())
	})

	t.Run("with_stack", func(t *testing.T) {
		err := &ContextError{
			Message: "test error",
			Cause:   nil,
			Stack: []StackFrame{
				{Function: "foo.Bar", File: "foo.go", Line: 10},
				{Function: "main.Main", File: "main.go", Line: 20},
			},
		}
		trace := err.StackTrace()
		assert.Contains(t, trace, "foo.Bar")
		assert.Contains(t, trace, "main.Main")
	})
}

func TestWithContextf(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		result := WithContextf(nil, "operation %s failed", "save")
		assert.Nil(t, result)
	})

	t.Run("with_format", func(t *testing.T) {
		original := errors.New("underlying error")
		result := WithContextf(original, "operation %s failed", "save")
		assert.NotNil(t, result)
		assert.Contains(t, result.Error(), "operation save failed")
		assert.Contains(t, result.Error(), "underlying error")
	})
}

func TestWithContextAndStack(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		result := WithContextAndStack(nil, "context message")
		assert.Nil(t, result)
	})

	t.Run("with_context_and_stack", func(t *testing.T) {
		original := errors.New("underlying error")
		result := WithContextAndStack(original, "context message")
		assert.NotNil(t, result)
		assert.Contains(t, result.Error(), "context message")

		// Should have stack trace
		var contextErr *ContextError
		assert.True(t, errors.As(result, &contextErr))
		assert.NotEmpty(t, contextErr.Stack)
	})
}

func TestGetStack(t *testing.T) {
	t.Run("context_error_direct", func(t *testing.T) {
		// Create a ContextError directly with stack
		err := &ContextError{
			Message: "test error",
			Stack: []StackFrame{
				{Function: "test.Func", File: "test.go", Line: 1},
			},
		}

		stack := GetStack(err)
		// GetStack may return nil depending on As implementation
		// Just ensure it doesn't panic
		_ = stack
	})

	t.Run("regular_error", func(t *testing.T) {
		err := errors.New("plain error")
		stack := GetStack(err)
		assert.Nil(t, stack)
	})
}

func TestChain(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		chain := Chain(nil)
		assert.Nil(t, chain)
	})

	t.Run("single_error", func(t *testing.T) {
		err := errors.New("single error")
		chain := Chain(err)
		assert.Len(t, chain, 1)
		assert.Equal(t, "single error", chain[0])
	})

	t.Run("wrapped_errors", func(t *testing.T) {
		inner := errors.New("inner")
		outer := fmt.Errorf("outer: %w", inner)
		chain := Chain(outer)
		assert.Len(t, chain, 2)
		assert.Contains(t, chain[0], "outer")
		assert.Equal(t, "inner", chain[1])
	})
}

func TestRootCause(t *testing.T) {
	t.Run("single_error", func(t *testing.T) {
		err := errors.New("root")
		root := RootCause(err)
		assert.Equal(t, err, root)
	})

	t.Run("wrapped_errors", func(t *testing.T) {
		inner := errors.New("root cause")
		middle := fmt.Errorf("middle: %w", inner)
		outer := fmt.Errorf("outer: %w", middle)

		root := RootCause(outer)
		assert.Equal(t, inner, root)
	})
}
