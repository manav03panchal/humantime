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

// =============================================================================
// Context Tests
// =============================================================================

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.Len(t, id1, 16) // 8 bytes = 16 hex chars
	assert.NotEqual(t, id1, id2)
}

func TestWithRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "test-request-123")
	id := RequestIDFromContext(ctx)
	assert.Equal(t, "test-request-123", id)
}

func TestNewRequestContext(t *testing.T) {
	ctx := NewRequestContext()
	id := RequestIDFromContext(ctx)
	assert.NotEmpty(t, id)
	assert.Len(t, id, 16)
}

func TestRequestIDFromContext(t *testing.T) {
	t.Run("nil_context", func(t *testing.T) {
		id := RequestIDFromContext(nil)
		assert.Empty(t, id)
	})

	t.Run("no_request_id", func(t *testing.T) {
		id := RequestIDFromContext(context.Background())
		assert.Empty(t, id)
	})

	t.Run("with_request_id", func(t *testing.T) {
		ctx := WithRequestID(context.Background(), "abc123")
		id := RequestIDFromContext(ctx)
		assert.Equal(t, "abc123", id)
	})
}

func TestLoggerFromContext(t *testing.T) {
	t.Run("without_request_id", func(t *testing.T) {
		logger := LoggerFromContext(context.Background())
		assert.NotNil(t, logger)
	})

	t.Run("with_request_id", func(t *testing.T) {
		ctx := WithRequestID(context.Background(), "test-123")
		logger := LoggerFromContext(ctx)
		assert.NotNil(t, logger)
	})
}

func TestContextLogger(t *testing.T) {
	var buf bytes.Buffer
	Init(Config{
		Level:  slog.LevelDebug,
		JSON:   true,
		Output: &buf,
	})

	ctx := WithRequestID(context.Background(), "test-ctx-123")

	t.Run("from_context", func(t *testing.T) {
		cl := FromContext(ctx)
		assert.NotNil(t, cl)
	})

	t.Run("with", func(t *testing.T) {
		cl := FromContext(ctx)
		cl2 := cl.With("key", "value")
		assert.NotNil(t, cl2)
	})

	t.Run("request_id", func(t *testing.T) {
		cl := FromContext(ctx)
		assert.Equal(t, "test-ctx-123", cl.RequestID())
	})

	t.Run("info", func(t *testing.T) {
		buf.Reset()
		cl := FromContext(ctx)
		cl.Info("info message", "key", "value")
		assert.Contains(t, buf.String(), "info message")
	})

	t.Run("debug", func(t *testing.T) {
		buf.Reset()
		cl := FromContext(ctx)
		cl.Debug("debug message", "key", "value")
		assert.Contains(t, buf.String(), "debug message")
	})

	t.Run("warn", func(t *testing.T) {
		buf.Reset()
		cl := FromContext(ctx)
		cl.Warn("warn message", "key", "value")
		assert.Contains(t, buf.String(), "warn message")
	})

	t.Run("error", func(t *testing.T) {
		buf.Reset()
		cl := FromContext(ctx)
		cl.Error("error message", "key", "value")
		assert.Contains(t, buf.String(), "error message")
	})
}

// =============================================================================
// Mask Tests
// =============================================================================

func TestMaskMap(t *testing.T) {
	t.Run("empty_map", func(t *testing.T) {
		result := MaskMap(map[string]any{})
		assert.Empty(t, result)
	})

	t.Run("with_sensitive_fields", func(t *testing.T) {
		input := map[string]any{
			"token":    "secret-token-123",
			"username": "john",
			"password": "my-password",
		}
		result := MaskMap(input)
		assert.Contains(t, result["token"], "*")
		assert.Equal(t, "john", result["username"])
		assert.Contains(t, result["password"], "*")
	})

	t.Run("with_nested_map", func(t *testing.T) {
		input := map[string]any{
			"config": map[string]any{
				"api_key": "secret-key",
				"name":    "test",
			},
		}
		result := MaskMap(input)
		nested := result["config"].(map[string]any)
		assert.Contains(t, nested["api_key"], "*")
		assert.Equal(t, "test", nested["name"])
	})

	t.Run("with_url_in_string", func(t *testing.T) {
		input := map[string]any{
			"webhook": "https://example.com/webhook/test-logging",
		}
		result := MaskMap(input)
		// URL should be masked
		masked := result["webhook"].(string)
		assert.Contains(t, masked, "https://example.com")
	})

	t.Run("with_non_string_sensitive", func(t *testing.T) {
		input := map[string]any{
			"secret": 12345, // non-string value
		}
		result := MaskMap(input)
		assert.Contains(t, result["secret"], "*")
	})
}

func TestSanitizeLogMessage(t *testing.T) {
	t.Run("no_urls", func(t *testing.T) {
		msg := "This is a simple log message"
		result := SanitizeLogMessage(msg)
		assert.Equal(t, msg, result)
	})

	t.Run("with_url", func(t *testing.T) {
		msg := "Sending request to https://example.com/api/v1/webhook/secret-token-12345"
		result := SanitizeLogMessage(msg)
		// URL should be masked but "Sending request to" preserved
		assert.Contains(t, result, "Sending request to")
		assert.Contains(t, result, "***")
	})

	t.Run("with_localhost", func(t *testing.T) {
		msg := "Connecting to http://localhost:8080/api"
		result := SanitizeLogMessage(msg)
		// Localhost URLs should not be masked
		assert.Contains(t, result, "localhost")
	})
}
