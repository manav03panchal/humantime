package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, slog.LevelInfo, cfg.Level)
	assert.False(t, cfg.JSON)
}

func TestDebugConfig(t *testing.T) {
	cfg := DebugConfig()
	assert.Equal(t, slog.LevelDebug, cfg.Level)
	assert.True(t, cfg.JSON)
	assert.True(t, cfg.AddSource)
}

func TestInit(t *testing.T) {
	t.Run("default_config", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := Config{
			Level:  slog.LevelInfo,
			JSON:   false,
			Output: &buf,
		}
		Init(cfg)

		logger := Logger()
		assert.NotNil(t, logger)
	})

	t.Run("json_config", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := Config{
			Level:  slog.LevelDebug,
			JSON:   true,
			Output: &buf,
		}
		Init(cfg)
		assert.True(t, Debug)
	})

	t.Run("nil_output_uses_stderr", func(t *testing.T) {
		cfg := Config{
			Level:  slog.LevelInfo,
			Output: nil,
		}
		Init(cfg)
		assert.NotNil(t, Logger())
	})
}

func TestInitDebug(t *testing.T) {
	InitDebug()
	assert.True(t, Debug)
}

func TestLogger(t *testing.T) {
	logger := Logger()
	assert.NotNil(t, logger)
}

func TestWith(t *testing.T) {
	logger := With("key", "value")
	assert.NotNil(t, logger)
}

func TestWithGroup(t *testing.T) {
	logger := WithGroup("test-group")
	assert.NotNil(t, logger)
}

func TestLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  slog.LevelDebug,
		JSON:   true,
		Output: &buf,
	})

	t.Run("info", func(t *testing.T) {
		buf.Reset()
		Info("test message", "key", "value")
		assert.Contains(t, buf.String(), "test message")
	})

	t.Run("debug", func(t *testing.T) {
		buf.Reset()
		DebugLog("debug message", "key", "value")
		assert.Contains(t, buf.String(), "debug message")
	})

	t.Run("warn", func(t *testing.T) {
		buf.Reset()
		Warn("warn message", "key", "value")
		assert.Contains(t, buf.String(), "warn message")
	})

	t.Run("error", func(t *testing.T) {
		buf.Reset()
		Error("error message", "key", "value")
		assert.Contains(t, buf.String(), "error message")
	})
}

func TestContextLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  slog.LevelDebug,
		JSON:   true,
		Output: &buf,
	})

	ctx := context.Background()

	t.Run("info_context", func(t *testing.T) {
		buf.Reset()
		InfoContext(ctx, "test message", "key", "value")
		assert.Contains(t, buf.String(), "test message")
	})

	t.Run("debug_context", func(t *testing.T) {
		buf.Reset()
		DebugContext(ctx, "debug message", "key", "value")
		assert.Contains(t, buf.String(), "debug message")
	})

	t.Run("warn_context", func(t *testing.T) {
		buf.Reset()
		WarnContext(ctx, "warn message", "key", "value")
		assert.Contains(t, buf.String(), "warn message")
	})

	t.Run("error_context", func(t *testing.T) {
		buf.Reset()
		ErrorContext(ctx, "error message", "key", "value")
		assert.Contains(t, buf.String(), "error message")
	})
}

func TestLogOperation(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  slog.LevelDebug,
		JSON:   true,
		Output: &buf,
	})

	buf.Reset()
	LogOperation("test-op", "extra", "data")
	assert.Contains(t, buf.String(), "test-op")
}

func TestKeyConstants(t *testing.T) {
	assert.Equal(t, "request_id", KeyRequestID)
	assert.Equal(t, "op", KeyOperation)
	assert.Equal(t, "duration_ms", KeyDuration)
	assert.Equal(t, "error", KeyError)
	assert.Equal(t, "project", KeyProject)
	assert.Equal(t, "task", KeyTask)
	assert.Equal(t, "block_id", KeyBlockID)
	assert.Equal(t, "reminder_id", KeyReminderID)
	assert.Equal(t, "webhook", KeyWebhook)
	assert.Equal(t, "status", KeyStatus)
	assert.Equal(t, "count", KeyCount)
}
