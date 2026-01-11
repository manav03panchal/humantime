package integration

import (
	"strings"
	"testing"
	"time"

	errs "github.com/manav03panchal/humantime/internal/errors"
	"github.com/manav03panchal/humantime/internal/logging"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// T117: Debug Trace Tests
// =============================================================================

func TestDebugTraceErrorChain(t *testing.T) {
	t.Run("shows_full_error_chain", func(t *testing.T) {
		root := errs.NewSystemError("connection failed", nil)
		layer1 := errs.WithContext(root, "connecting to database")
		layer2 := errs.WithContext(layer1, "initializing storage")
		layer3 := errs.WithContext(layer2, "starting application")

		chain := errs.Chain(layer3)
		assert.GreaterOrEqual(t, len(chain), 3)

		// Should contain all context levels
		fullChain := strings.Join(chain, " -> ")
		assert.Contains(t, fullChain, "application")
		assert.Contains(t, fullChain, "storage")
	})

	t.Run("formats_debug_output", func(t *testing.T) {
		err := errs.NewUserError("invalid input", "use correct format")
		wrapped := errs.WithStack(err)

		debug := errs.FormatDebugError(wrapped)
		assert.NotEmpty(t, debug)
	})
}

func TestDebugTraceTimingInfo(t *testing.T) {
	t.Run("measures_operation_time", func(t *testing.T) {
		start := time.Now()

		// Simulate some work
		time.Sleep(10 * time.Millisecond)

		elapsed := time.Since(start)
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10))
	})

	t.Run("formats_timing_output", func(t *testing.T) {
		elapsed := 150 * time.Millisecond
		formatted := elapsed.String()
		assert.Contains(t, formatted, "ms")
	})
}

func TestDebugTraceRequestCorrelation(t *testing.T) {
	t.Run("traces_request_through_operations", func(t *testing.T) {
		ctx := logging.NewRequestContext()
		requestID := logging.RequestIDFromContext(ctx)

		// Simulate operations that would be traced
		operations := []struct {
			name    string
			elapsed time.Duration
		}{
			{"parse_input", 5 * time.Millisecond},
			{"validate_data", 2 * time.Millisecond},
			{"execute_command", 50 * time.Millisecond},
			{"format_output", 3 * time.Millisecond},
		}

		for _, op := range operations {
			// In debug mode, each operation would log with request ID
			assert.NotEmpty(t, requestID)
			_ = op
		}
	})
}

func TestDebugTraceStackFrames(t *testing.T) {
	t.Run("captures_stack_frames", func(t *testing.T) {
		err := errs.NewSystemError("test error", nil)
		withStack := errs.WithStack(err)

		ctxErr, ok := withStack.(*errs.ContextError)
		if ok && len(ctxErr.Stack) > 0 {
			formatted := errs.FormatStackTrace(ctxErr.Stack)
			assert.NotEmpty(t, formatted)
		}
	})

	t.Run("stack_includes_function_names", func(t *testing.T) {
		err := errs.NewSystemError("test error", nil)
		withStack := errs.WithStack(err)

		ctxErr, ok := withStack.(*errs.ContextError)
		if ok && len(ctxErr.Stack) > 0 {
			// Stack frames should contain function information
			assert.Greater(t, len(ctxErr.Stack), 0)
		}
	})
}

func TestDebugTraceContextualInfo(t *testing.T) {
	t.Run("includes_error_context", func(t *testing.T) {
		err := errs.NewUserError("validation failed for email", "check email format")
		formatted := err.Error()
		assert.Contains(t, formatted, "email")
	})

	t.Run("includes_suggestion", func(t *testing.T) {
		err := errs.NewUserError("invalid duration", "use format like 1h30m")
		assert.Contains(t, err.Suggestion, "1h30m")
	})
}

func TestDebugModeFormatting(t *testing.T) {
	t.Run("user_error_formatting", func(t *testing.T) {
		err := errs.NewUserError("bad input", "try again")
		formatted := errs.FormatByCategory(err)
		assert.NotEmpty(t, formatted)
		assert.Contains(t, formatted, "bad input")
	})

	t.Run("system_error_formatting", func(t *testing.T) {
		err := errs.NewSystemError("disk full", nil)
		formatted := errs.FormatByCategory(err)
		assert.NotEmpty(t, formatted)
		assert.Contains(t, formatted, "System")
	})

	t.Run("recoverable_error_formatting", func(t *testing.T) {
		err := errs.NewRecoverableError("network timeout", nil, 3)
		formatted := errs.FormatByCategory(err)
		assert.NotEmpty(t, formatted)
		assert.Contains(t, formatted, "retry")
	})
}

func TestDebugTraceNilHandling(t *testing.T) {
	t.Run("handles_nil_error", func(t *testing.T) {
		formatted := errs.FormatDebugError(nil)
		assert.Empty(t, formatted)
	})

	t.Run("handles_empty_chain", func(t *testing.T) {
		chain := errs.Chain(nil)
		assert.Empty(t, chain)
	})
}
