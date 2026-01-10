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
