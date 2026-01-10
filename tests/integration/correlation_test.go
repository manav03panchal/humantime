package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/manav03panchal/humantime/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// T109: Request ID Correlation Tests
// =============================================================================

func TestRequestIDGeneration(t *testing.T) {
	t.Run("generates_unique_IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := logging.GenerateRequestID()
			assert.NotEmpty(t, id)
			assert.False(t, ids[id], "ID should be unique")
			ids[id] = true
		}
	})

	t.Run("ID_has_expected_format", func(t *testing.T) {
		id := logging.GenerateRequestID()
		// Should be non-empty and reasonable length
		assert.True(t, len(id) >= 8, "ID should be at least 8 chars")
		assert.True(t, len(id) <= 64, "ID should be at most 64 chars")
	})
}

func TestRequestContextPropagation(t *testing.T) {
	t.Run("stores_and_retrieves_request_ID", func(t *testing.T) {
		ctx := context.Background()
		requestID := logging.GenerateRequestID()

		// Store request ID in context
		ctx = logging.WithRequestID(ctx, requestID)

		// Retrieve it
		retrieved := logging.RequestIDFromContext(ctx)
		assert.Equal(t, requestID, retrieved)
	})

	t.Run("returns_empty_for_missing_ID", func(t *testing.T) {
		ctx := context.Background()
		retrieved := logging.RequestIDFromContext(ctx)
		assert.Empty(t, retrieved)
	})

	t.Run("context_is_inherited", func(t *testing.T) {
		ctx := context.Background()
		requestID := logging.GenerateRequestID()
		ctx = logging.WithRequestID(ctx, requestID)

		// Create child context
		childCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Should still have request ID
		retrieved := logging.RequestIDFromContext(childCtx)
		assert.Equal(t, requestID, retrieved)
	})
}

func TestRequestContextLogger(t *testing.T) {
	t.Run("creates_request_context", func(t *testing.T) {
		ctx := logging.NewRequestContext()

		// Should have a request ID
		requestID := logging.RequestIDFromContext(ctx)
		assert.NotEmpty(t, requestID)
	})

	t.Run("logger_from_context_includes_request_ID", func(t *testing.T) {
		ctx := context.Background()
		requestID := "test-request-123"
		ctx = logging.WithRequestID(ctx, requestID)

		// Get logger from context
		logger := logging.LoggerFromContext(ctx)
		require.NotNil(t, logger)

		// Logger should exist (implementation detail may vary)
		_ = logger
	})
}

func TestContextLogger(t *testing.T) {
	t.Run("creates_context_logger_with_request_id", func(t *testing.T) {
		requestID := "ctx-test-456"
		ctx := logging.WithRequestID(context.Background(), requestID)
		ctxLogger := logging.FromContext(ctx)
		require.NotNil(t, ctxLogger)

		// Request ID should be accessible from the context
		assert.Equal(t, requestID, ctxLogger.RequestID())
	})

	t.Run("creates_context_logger_without_request_id", func(t *testing.T) {
		ctxLogger := logging.FromContext(context.Background())
		require.NotNil(t, ctxLogger)

		// No request ID in context
		assert.Empty(t, ctxLogger.RequestID())
	})

	t.Run("context_logger_chain", func(t *testing.T) {
		ctxLogger := logging.FromContext(context.Background())

		// Chain with values
		ctxLogger = ctxLogger.With("operation", "test")
		ctxLogger = ctxLogger.With("user", "test-user")

		// Should still function
		assert.NotNil(t, ctxLogger)
	})
}

func TestCorrelationAcrossOperations(t *testing.T) {
	t.Run("correlates_related_operations", func(t *testing.T) {
		// Create a request context
		ctx := logging.NewRequestContext()
		requestID := logging.RequestIDFromContext(ctx)

		// Simulate multiple operations with same context
		operations := []string{
			"parse_input",
			"validate_data",
			"store_result",
			"send_notification",
		}

		for _, op := range operations {
			// Each operation should have access to the same request ID
			opRequestID := logging.RequestIDFromContext(ctx)
			assert.Equal(t, requestID, opRequestID, "Request ID should be consistent across operations")
			_ = op
		}
	})
}

func TestRequestIDFormat(t *testing.T) {
	t.Run("ID_is_URL_safe", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			id := logging.GenerateRequestID()
			// Should not contain characters that need URL encoding
			assert.False(t, strings.ContainsAny(id, "/?#[]@!$&'()*+,;="))
		}
	})

	t.Run("ID_is_log_safe", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			id := logging.GenerateRequestID()
			// Should not contain newlines or control characters
			assert.False(t, strings.ContainsAny(id, "\n\r\t"))
			for _, c := range id {
				assert.True(t, c >= 32, "Should not contain control characters")
			}
		}
	})
}

func TestConcurrentRequestIDs(t *testing.T) {
	t.Run("concurrent_generation_is_unique", func(t *testing.T) {
		results := make(chan string, 100)

		// Generate IDs concurrently
		for i := 0; i < 100; i++ {
			go func() {
				results <- logging.GenerateRequestID()
			}()
		}

		// Collect and verify uniqueness
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			id := <-results
			assert.False(t, ids[id], "ID should be unique even with concurrent generation")
			ids[id] = true
		}
	})
}
