// Package logging provides structured logging for the Humantime CLI.
// It uses Go's standard library slog for structured logging with JSON output support.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	// defaultLogger is the package-level logger instance.
	defaultLogger *slog.Logger
	loggerMu      sync.RWMutex

	// Debug indicates if debug mode is enabled.
	Debug bool
)

func init() {
	// Initialize with a default text logger to stderr.
	defaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// Config holds logger configuration.
type Config struct {
	Level      slog.Level // Minimum log level
	JSON       bool       // Use JSON output format
	Output     io.Writer  // Output destination (default: stderr)
	AddSource  bool       // Include source file and line number
}

// DefaultConfig returns the default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:  slog.LevelInfo,
		JSON:   false,
		Output: os.Stderr,
	}
}

// DebugConfig returns a configuration suitable for debug mode.
func DebugConfig() Config {
	return Config{
		Level:     slog.LevelDebug,
		JSON:      true,
		Output:    os.Stderr,
		AddSource: true,
	}
}

// Init initializes the global logger with the given configuration.
func Init(cfg Config) {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	output := cfg.Output
	if output == nil {
		output = os.Stderr
	}

	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.JSON {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
	Debug = cfg.Level == slog.LevelDebug
}

// InitDebug initializes the logger in debug mode with JSON output.
func InitDebug() {
	Init(DebugConfig())
}

// Logger returns the current logger instance.
func Logger() *slog.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	return defaultLogger
}

// With returns a logger with additional attributes.
func With(args ...any) *slog.Logger {
	return Logger().With(args...)
}

// WithGroup returns a logger with a group prefix.
func WithGroup(name string) *slog.Logger {
	return Logger().WithGroup(name)
}

// Log logging functions that delegate to the default logger.

// Info logs at INFO level.
func Info(msg string, args ...any) {
	Logger().Info(msg, args...)
}

// Debug logs at DEBUG level.
func DebugLog(msg string, args ...any) {
	Logger().Debug(msg, args...)
}

// Warn logs at WARN level.
func Warn(msg string, args ...any) {
	Logger().Warn(msg, args...)
}

// Error logs at ERROR level.
func Error(msg string, args ...any) {
	Logger().Error(msg, args...)
}

// InfoContext logs at INFO level with context.
func InfoContext(ctx context.Context, msg string, args ...any) {
	Logger().InfoContext(ctx, msg, args...)
}

// DebugContext logs at DEBUG level with context.
func DebugContext(ctx context.Context, msg string, args ...any) {
	Logger().DebugContext(ctx, msg, args...)
}

// WarnContext logs at WARN level with context.
func WarnContext(ctx context.Context, msg string, args ...any) {
	Logger().WarnContext(ctx, msg, args...)
}

// ErrorContext logs at ERROR level with context.
func ErrorContext(ctx context.Context, msg string, args ...any) {
	Logger().ErrorContext(ctx, msg, args...)
}

// Common structured logging fields.
const (
	KeyRequestID  = "request_id"
	KeyOperation  = "op"
	KeyDuration   = "duration_ms"
	KeyError      = "error"
	KeyProject    = "project"
	KeyTask       = "task"
	KeyBlockID    = "block_id"
	KeyReminderID = "reminder_id"
	KeyWebhook    = "webhook"
	KeyStatus     = "status"
	KeyCount      = "count"
)

// LogOperation logs the start and duration of an operation.
// Usage: defer LogOperation("save_block", time.Now())
func LogOperation(op string, args ...any) {
	allArgs := append([]any{KeyOperation, op}, args...)
	Logger().Debug("operation", allArgs...)
}
