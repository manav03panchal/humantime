package unit

import (
	"errors"
	"strings"
	"testing"

	errs "github.com/manav03panchal/humantime/internal/errors"
)

// TestWithContext tests error context wrapping.
func TestWithContext(t *testing.T) {
	original := errors.New("original error")
	wrapped := errs.WithContext(original, "additional context")

	if wrapped == nil {
		t.Fatal("WithContext should return non-nil error")
	}

	errStr := wrapped.Error()
	if !strings.Contains(errStr, "original error") {
		t.Errorf("Wrapped error should contain original: %s", errStr)
	}
	if !strings.Contains(errStr, "additional context") {
		t.Errorf("Wrapped error should contain context: %s", errStr)
	}
}

// TestWithContextNil tests wrapping nil error.
func TestWithContextNil(t *testing.T) {
	wrapped := errs.WithContext(nil, "context")

	if wrapped != nil {
		t.Error("WithContext(nil) should return nil")
	}
}

// TestWithStack tests stack trace capture.
func TestWithStack(t *testing.T) {
	original := errors.New("test error")
	withStack := errs.WithStack(original)

	if withStack == nil {
		t.Fatal("WithStack should return non-nil error")
	}

	// Check it's a ContextError with stack
	ctxErr, ok := withStack.(*errs.ContextError)
	if !ok {
		t.Fatal("WithStack should return *ContextError")
	}

	if len(ctxErr.Stack) == 0 {
		t.Error("Stack should be captured")
	}
}

// TestChain tests error chain extraction.
func TestChain(t *testing.T) {
	original := errors.New("root cause")
	wrapped1 := errs.WithContext(original, "layer 1")
	wrapped2 := errs.WithContext(wrapped1, "layer 2")

	chain := errs.Chain(wrapped2)

	if len(chain) == 0 {
		t.Error("Chain should not be empty")
	}

	// First message should be outermost context
	if !strings.Contains(chain[0], "layer 2") {
		t.Errorf("First chain element should be outermost, got: %s", chain[0])
	}
}

// TestChainNil tests chain with nil error.
func TestChainNil(t *testing.T) {
	chain := errs.Chain(nil)

	if len(chain) != 0 {
		t.Error("Chain(nil) should return empty slice")
	}
}

// TestRootCause tests root cause extraction.
func TestRootCause(t *testing.T) {
	original := errors.New("root cause")
	wrapped1 := errs.WithContext(original, "layer 1")
	wrapped2 := errs.WithContext(wrapped1, "layer 2")

	root := errs.RootCause(wrapped2)

	if root != original {
		t.Errorf("RootCause should return original error, got: %v", root)
	}
}

// TestRootCauseNil tests root cause with nil.
func TestRootCauseNil(t *testing.T) {
	root := errs.RootCause(nil)

	if root != nil {
		t.Error("RootCause(nil) should return nil")
	}
}

// TestRootCauseUnwrapped tests root cause with unwrapped error.
func TestRootCauseUnwrapped(t *testing.T) {
	original := errors.New("simple error")
	root := errs.RootCause(original)

	if root != original {
		t.Error("RootCause of unwrapped should return same error")
	}
}

// TestContextErrorUnwrap tests Unwrap interface.
func TestContextErrorUnwrap(t *testing.T) {
	original := errors.New("original")
	wrapped := errs.WithContext(original, "context")

	ctxErr, ok := wrapped.(*errs.ContextError)
	if !ok {
		t.Fatal("Expected *ContextError")
	}

	unwrapped := ctxErr.Unwrap()
	if unwrapped != original {
		t.Error("Unwrap should return original error")
	}
}

// TestMultipleContextLayers tests deeply nested contexts.
func TestMultipleContextLayers(t *testing.T) {
	err := errors.New("base")

	for i := 0; i < 10; i++ {
		err = errs.WithContext(err, "layer")
	}

	// Should still be able to get root cause
	root := errs.RootCause(err)
	if root == nil {
		t.Error("Should find root cause through 10 layers")
	}

	chain := errs.Chain(err)
	if len(chain) < 10 {
		t.Errorf("Chain should have at least 10 elements, got %d", len(chain))
	}
}

// TestStackFrameFormat tests stack frame formatting.
func TestStackFrameFormat(t *testing.T) {
	original := errors.New("test")
	withStack := errs.WithStack(original)

	ctxErr, ok := withStack.(*errs.ContextError)
	if !ok {
		t.Fatal("Expected *ContextError")
	}

	if len(ctxErr.Stack) == 0 {
		t.Fatal("Expected stack frames")
	}

	frame := ctxErr.Stack[0]
	if frame.Function == "" {
		t.Error("Stack frame should have function name")
	}
	if frame.File == "" {
		t.Error("Stack frame should have file name")
	}
	if frame.Line == 0 {
		t.Error("Stack frame should have line number")
	}
}

// TestErrorsIsCompatibility tests errors.Is compatibility.
func TestErrorsIsCompatibility(t *testing.T) {
	sentinelErr := errors.New("sentinel")
	wrapped := errs.WithContext(sentinelErr, "context")

	if !errors.Is(wrapped, sentinelErr) {
		t.Error("errors.Is should find sentinel through context")
	}
}

// TestErrorsAsCompatibility tests errors.As compatibility.
func TestErrorsAsCompatibility(t *testing.T) {
	original := errors.New("test")
	wrapped := errs.WithContext(original, "context")

	var ctxErr *errs.ContextError
	if !errors.As(wrapped, &ctxErr) {
		t.Error("errors.As should find ContextError")
	}

	if ctxErr == nil {
		t.Error("ctxErr should not be nil after As")
	}
}
