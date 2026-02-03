package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"syscall"
	"testing"

	"github.com/manav03panchal/humantime/internal/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Context Tests
// =============================================================================

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.NotEmpty(t, opts.DBPath)
	assert.False(t, opts.InMemory)
	assert.Equal(t, output.FormatCLI, opts.Format)
	assert.Equal(t, output.ColorAuto, opts.ColorMode)
	assert.False(t, opts.Debug)
}

func TestNew(t *testing.T) {
	ctx, err := New(Options{InMemory: true})
	require.NoError(t, err)
	defer ctx.Close()

	assert.NotNil(t, ctx.DB)
	assert.NotNil(t, ctx.Formatter)
	assert.NotNil(t, ctx.BlockRepo)
	assert.NotNil(t, ctx.ProjectRepo)
	assert.NotNil(t, ctx.ActiveBlockRepo)
	assert.NotNil(t, ctx.UndoRepo)
}

func TestNewWithOptions(t *testing.T) {
	ctx, err := New(Options{
		InMemory:  true,
		Format:    output.FormatJSON,
		ColorMode: output.ColorNever,
		Debug:     true,
	})
	require.NoError(t, err)
	defer ctx.Close()

	assert.Equal(t, output.FormatJSON, ctx.Formatter.Format)
	assert.Equal(t, output.ColorNever, ctx.Formatter.ColorMode)
	assert.True(t, ctx.Debug)
}

func TestNewWithEnvVariable(t *testing.T) {
	// Test with :memory: env var
	os.Setenv("HUMANTIME_DATABASE", ":memory:")
	defer os.Unsetenv("HUMANTIME_DATABASE")

	ctx, err := New(Options{})
	require.NoError(t, err)
	defer ctx.Close()

	assert.NotNil(t, ctx.DB)
}

func TestNewWithEnvVariablePath(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/humantime-test.db"

	os.Setenv("HUMANTIME_DATABASE", dbPath)
	defer os.Unsetenv("HUMANTIME_DATABASE")

	ctx, err := New(Options{})
	require.NoError(t, err)
	defer ctx.Close()

	assert.NotNil(t, ctx.DB)
}

func TestContextClose(t *testing.T) {
	ctx, err := New(Options{InMemory: true})
	require.NoError(t, err)

	err = ctx.Close()
	assert.NoError(t, err)

	// Closing nil DB should be safe
	nilCtx := &Context{}
	err = nilCtx.Close()
	assert.NoError(t, err)
}

func TestContextCLIFormatter(t *testing.T) {
	ctx, err := New(Options{InMemory: true})
	require.NoError(t, err)
	defer ctx.Close()

	cli := ctx.CLIFormatter()
	assert.NotNil(t, cli)
}

func TestContextJSONFormatter(t *testing.T) {
	ctx, err := New(Options{InMemory: true})
	require.NoError(t, err)
	defer ctx.Close()

	jf := ctx.JSONFormatter()
	assert.NotNil(t, jf)
}

func TestContextIsJSON(t *testing.T) {
	t.Run("json_format", func(t *testing.T) {
		ctx, err := New(Options{InMemory: true, Format: output.FormatJSON})
		require.NoError(t, err)
		defer ctx.Close()

		assert.True(t, ctx.IsJSON())
		assert.False(t, ctx.IsCLI())
	})

	t.Run("cli_format", func(t *testing.T) {
		ctx, err := New(Options{InMemory: true, Format: output.FormatCLI})
		require.NoError(t, err)
		defer ctx.Close()

		assert.False(t, ctx.IsJSON())
		assert.True(t, ctx.IsCLI())
	})
}

func TestContextDebugf(t *testing.T) {
	t.Run("debug_enabled", func(t *testing.T) {
		var buf bytes.Buffer
		ctx, err := New(Options{InMemory: true, Debug: true})
		require.NoError(t, err)
		defer ctx.Close()

		ctx.Formatter.Writer = &buf
		ctx.Debugf("test message %s", "arg1")

		assert.Contains(t, buf.String(), "[DEBUG]")
		assert.Contains(t, buf.String(), "test message arg1")
	})

	t.Run("debug_disabled", func(t *testing.T) {
		var buf bytes.Buffer
		ctx, err := New(Options{InMemory: true, Debug: false})
		require.NoError(t, err)
		defer ctx.Close()

		ctx.Formatter.Writer = &buf
		ctx.Debugf("test message")

		assert.Empty(t, buf.String())
	})
}

// =============================================================================
// Error Tests
// =============================================================================

func TestSentinelErrors(t *testing.T) {
	errors := []error{
		ErrNoActiveTracking,
		ErrProjectRequired,
		ErrInvalidSID,
		ErrInvalidTimestamp,
		ErrEndBeforeStart,
		ErrBlockNotFound,
		ErrProjectNotFound,
		ErrInvalidColor,
		ErrInvalidDuration,
		ErrDiskFull,
	}

	for _, err := range errors {
		t.Run(err.Error(), func(t *testing.T) {
			assert.NotNil(t, err)
			assert.NotEmpty(t, err.Error())
		})
	}
}

func TestParseError(t *testing.T) {
	err := NewParseError("duration", "abc", "must be a valid duration")

	assert.NotNil(t, err)
	assert.Equal(t, "duration", err.Field)
	assert.Equal(t, "abc", err.Value)
	assert.Equal(t, "must be a valid duration", err.Message)
	assert.Contains(t, err.Error(), "duration")
	assert.Contains(t, err.Error(), "abc")
	assert.Contains(t, err.Error(), "must be a valid duration")
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("project", "cannot be empty")

	assert.NotNil(t, err)
	assert.Equal(t, "project", err.Field)
	assert.Equal(t, "cannot be empty", err.Message)
	assert.Equal(t, "project: cannot be empty", err.Error())
}

func TestGetSuggestion(t *testing.T) {
	t.Run("known_error", func(t *testing.T) {
		suggestion := GetSuggestion(ErrNoActiveTracking)
		assert.NotEmpty(t, suggestion)
		assert.Contains(t, suggestion, "ht start")
	})

	t.Run("wrapped_error", func(t *testing.T) {
		wrapped := fmt.Errorf("context: %w", ErrProjectRequired)
		suggestion := GetSuggestion(wrapped)
		assert.NotEmpty(t, suggestion)
	})

	t.Run("unknown_error", func(t *testing.T) {
		unknown := errors.New("some random error")
		suggestion := GetSuggestion(unknown)
		assert.Empty(t, suggestion)
	})
}

func TestFormatError(t *testing.T) {
	t.Run("with_suggestion", func(t *testing.T) {
		formatted := FormatError(ErrNoActiveTracking)
		assert.Contains(t, formatted, "no active tracking")
		assert.Contains(t, formatted, "ht start")
	})

	t.Run("without_suggestion", func(t *testing.T) {
		err := errors.New("custom error")
		formatted := FormatError(err)
		assert.Equal(t, "custom error", formatted)
	})
}

func TestSuggestionsMap(t *testing.T) {
	// Verify all sentinel errors have suggestions
	sentinelErrors := []error{
		ErrNoActiveTracking,
		ErrProjectRequired,
		ErrInvalidSID,
		ErrInvalidTimestamp,
		ErrEndBeforeStart,
		ErrBlockNotFound,
		ErrProjectNotFound,
		ErrInvalidColor,
		ErrDiskFull,
	}

	for _, err := range sentinelErrors {
		t.Run(err.Error(), func(t *testing.T) {
			suggestion, exists := Suggestions[err]
			assert.True(t, exists, "missing suggestion for %v", err)
			assert.NotEmpty(t, suggestion)
		})
	}
}

// =============================================================================
// DiskFullError Tests
// =============================================================================

func TestNewDiskFullError(t *testing.T) {
	original := errors.New("underlying error")
	err := NewDiskFullError("write", "/path/to/db", original)

	assert.NotNil(t, err)
	assert.Equal(t, "write", err.Op)
	assert.Equal(t, "/path/to/db", err.Path)
	assert.Contains(t, err.Error(), "disk full")
	assert.Contains(t, err.Error(), "write")
	assert.Contains(t, err.Error(), "/path/to/db")
}

func TestDiskFullErrorWithoutPath(t *testing.T) {
	original := errors.New("underlying error")
	err := NewDiskFullError("sync", "", original)

	assert.Contains(t, err.Error(), "disk full")
	assert.Contains(t, err.Error(), "sync")
	assert.NotContains(t, err.Error(), "on ")
}

func TestDiskFullErrorUnwrap(t *testing.T) {
	original := errors.New("underlying error")
	err := NewDiskFullError("write", "", original)

	// Unwrap should return ErrDiskFull
	assert.True(t, errors.Is(err, ErrDiskFull))
}

func TestIsDiskFullError(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		assert.False(t, IsDiskFullError(nil))
	})

	t.Run("disk_full_error_type", func(t *testing.T) {
		err := NewDiskFullError("write", "", nil)
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("sentinel_disk_full", func(t *testing.T) {
		assert.True(t, IsDiskFullError(ErrDiskFull))
	})

	t.Run("wrapped_sentinel", func(t *testing.T) {
		wrapped := fmt.Errorf("context: %w", ErrDiskFull)
		assert.True(t, IsDiskFullError(wrapped))
	})

	t.Run("enospc_errno", func(t *testing.T) {
		err := syscall.ENOSPC
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("error_message_no_space", func(t *testing.T) {
		err := errors.New("no space left on device")
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("error_message_disk_full", func(t *testing.T) {
		err := errors.New("DISK FULL")
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("error_message_enospc", func(t *testing.T) {
		err := errors.New("write failed: ENOSPC")
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("error_message_not_enough_space", func(t *testing.T) {
		err := errors.New("not enough space on disk")
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("error_message_insufficient_space", func(t *testing.T) {
		err := errors.New("insufficient disk space")
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("error_message_out_of_disk_space", func(t *testing.T) {
		err := errors.New("out of disk space")
		assert.True(t, IsDiskFullError(err))
	})

	t.Run("regular_error", func(t *testing.T) {
		err := errors.New("connection timeout")
		assert.False(t, IsDiskFullError(err))
	})
}

func TestWrapDiskFullError(t *testing.T) {
	t.Run("nil_error", func(t *testing.T) {
		result := WrapDiskFullError(nil, "write", "/path")
		assert.Nil(t, result)
	})

	t.Run("disk_full_error", func(t *testing.T) {
		err := errors.New("no space left on device")
		result := WrapDiskFullError(err, "write", "/path/to/db")

		var diskFullErr *DiskFullError
		assert.True(t, errors.As(result, &diskFullErr))
		assert.Equal(t, "write", diskFullErr.Op)
		assert.Equal(t, "/path/to/db", diskFullErr.Path)
	})

	t.Run("regular_error_not_wrapped", func(t *testing.T) {
		err := errors.New("connection timeout")
		result := WrapDiskFullError(err, "write", "/path")

		assert.Equal(t, err, result)
		var diskFullErr *DiskFullError
		assert.False(t, errors.As(result, &diskFullErr))
	})
}

// =============================================================================
// Options Tests
// =============================================================================

func TestOptionsStruct(t *testing.T) {
	opts := Options{
		DBPath:    "/path/to/db",
		InMemory:  true,
		Format:    output.FormatJSON,
		ColorMode: output.ColorAlways,
		Debug:     true,
	}

	assert.Equal(t, "/path/to/db", opts.DBPath)
	assert.True(t, opts.InMemory)
	assert.Equal(t, output.FormatJSON, opts.Format)
	assert.Equal(t, output.ColorAlways, opts.ColorMode)
	assert.True(t, opts.Debug)
}
