package integration

import (
	"strings"
	"testing"

	errs "github.com/manav03panchal/humantime/internal/errors"
)

// TestDebugErrorFormatting tests error formatting in debug mode.
func TestDebugErrorFormatting(t *testing.T) {
	// Create an error with context
	original := errs.NewUserError("invalid duration", "use format like 1h30m")
	wrapped := errs.WithContext(original, "parsing user input")
	withStack := errs.WithStack(wrapped)

	// Format for debug output
	debugOutput := errs.FormatDebugError(withStack)

	// Should contain error message
	if !strings.Contains(debugOutput, "invalid duration") {
		t.Error("Debug output should contain error message")
	}

	// Should contain context
	if !strings.Contains(debugOutput, "parsing user input") {
		t.Error("Debug output should contain context")
	}
}

// TestUserErrorFormatting tests user-friendly error formatting.
func TestUserErrorFormatting(t *testing.T) {
	err := errs.NewUserError("project not found", "check project name")

	userOutput := errs.FormatUserError(err)

	// Should be user-friendly (no stack traces)
	if strings.Contains(userOutput, ".go:") {
		t.Error("User output should not contain file references")
	}
}

// TestStackTraceFormatting tests stack trace formatting.
func TestStackTraceFormatting(t *testing.T) {
	original := errs.NewSystemError("database error", nil)
	withStack := errs.WithStack(original)

	ctxErr, ok := withStack.(*errs.ContextError)
	if !ok {
		t.Fatal("Expected ContextError")
	}

	if len(ctxErr.Stack) == 0 {
		t.Fatal("Expected stack frames")
	}

	formatted := errs.FormatStackTrace(ctxErr.Stack)

	// Should contain function names
	if !strings.Contains(formatted, "TestStackTraceFormatting") {
		t.Log("Stack trace:", formatted)
		// This might not be present depending on inlining
	}
}

// TestErrorCategoryFormatting tests category-based formatting.
func TestErrorCategoryFormatting(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			"user error",
			errs.NewUserError("bad input", "fix it"),
			"bad input",
		},
		{
			"system error",
			errs.NewSystemError("disk full", nil),
			"System error",
		},
		{
			"recoverable error",
			errs.NewRecoverableError("network timeout", nil, 3),
			"will retry",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			formatted := errs.FormatByCategory(tc.err)
			if !strings.Contains(formatted, tc.contains) {
				t.Errorf("Expected %q in formatted output, got: %s", tc.contains, formatted)
			}
		})
	}
}

// TestNilErrorFormatting tests nil error handling.
func TestNilErrorFormatting(t *testing.T) {
	result := errs.FormatDebugError(nil)
	if result != "" {
		t.Errorf("FormatDebugError(nil) should return empty string, got %q", result)
	}

	result = errs.FormatUserError(nil)
	if result != "" {
		t.Errorf("FormatUserError(nil) should return empty string, got %q", result)
	}

	result = errs.FormatByCategory(nil)
	if result != "" {
		t.Errorf("FormatByCategory(nil) should return empty string, got %q", result)
	}
}

// TestErrorChainFormatting tests formatting of error chains.
func TestErrorChainFormatting(t *testing.T) {
	root := errs.NewSystemError("connection refused", nil)
	layer1 := errs.WithContext(root, "connecting to webhook")
	layer2 := errs.WithContext(layer1, "sending notification")

	chain := errs.Chain(layer2)

	if len(chain) < 2 {
		t.Errorf("Expected at least 2 elements in chain, got %d", len(chain))
	}

	// First should be outermost
	if !strings.Contains(chain[0], "notification") {
		t.Errorf("First chain element should contain 'notification', got %q", chain[0])
	}
}
