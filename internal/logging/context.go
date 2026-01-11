package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

// contextKey is a type for context keys used by this package.
type contextKey int

const (
	requestIDKey contextKey = iota
)

// GenerateRequestID creates a new unique request ID.
// Format: 16 character hex string (8 random bytes).
func GenerateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to a simple counter if random fails (shouldn't happen).
		return "00000000"
	}
	return hex.EncodeToString(b)
}

// WithRequestID returns a new context with the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// NewRequestContext creates a new context with a generated request ID.
func NewRequestContext() context.Context {
	return WithRequestID(context.Background(), GenerateRequestID())
}

// RequestIDFromContext extracts the request ID from the context.
// Returns empty string if no request ID is set.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerFromContext returns a logger with the request ID from context.
// If no request ID is in the context, returns the default logger.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	logger := Logger()
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		logger = logger.With(KeyRequestID, requestID)
	}
	return logger
}

// ContextLogger is a helper for logging with context.
type ContextLogger struct {
	ctx    context.Context
	logger *slog.Logger
}

// FromContext creates a ContextLogger from a context.
func FromContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		ctx:    ctx,
		logger: LoggerFromContext(ctx),
	}
}

// With returns a new ContextLogger with additional attributes.
func (cl *ContextLogger) With(args ...any) *ContextLogger {
	return &ContextLogger{
		ctx:    cl.ctx,
		logger: cl.logger.With(args...),
	}
}

// Info logs at INFO level.
func (cl *ContextLogger) Info(msg string, args ...any) {
	cl.logger.InfoContext(cl.ctx, msg, args...)
}

// Debug logs at DEBUG level.
func (cl *ContextLogger) Debug(msg string, args ...any) {
	cl.logger.DebugContext(cl.ctx, msg, args...)
}

// Warn logs at WARN level.
func (cl *ContextLogger) Warn(msg string, args ...any) {
	cl.logger.WarnContext(cl.ctx, msg, args...)
}

// Error logs at ERROR level.
func (cl *ContextLogger) Error(msg string, args ...any) {
	cl.logger.ErrorContext(cl.ctx, msg, args...)
}

// RequestID returns the request ID from the logger's context.
func (cl *ContextLogger) RequestID() string {
	return RequestIDFromContext(cl.ctx)
}
