package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// String returns a formatted string representation of the stack frame.
func (f StackFrame) String() string {
	return fmt.Sprintf("%s\n\t%s:%d", f.Function, f.File, f.Line)
}

// ContextError wraps an error with additional context and optional stack trace.
type ContextError struct {
	Message string
	Cause   error
	Stack   []StackFrame
}

func (e *ContextError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *ContextError) Unwrap() error {
	return e.Cause
}

// StackTrace returns the stack trace as a formatted string.
func (e *ContextError) StackTrace() string {
	if len(e.Stack) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, frame := range e.Stack {
		sb.WriteString(frame.String())
		sb.WriteString("\n")
	}
	return sb.String()
}

// WithContext wraps an error with additional context message.
// The context is prepended to the error message.
func WithContext(err error, message string) error {
	if err == nil {
		return nil
	}
	return &ContextError{
		Message: message,
		Cause:   err,
	}
}

// WithContextf wraps an error with a formatted context message.
func WithContextf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &ContextError{
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WithStack wraps an error and captures the current stack trace.
// Use this when you need detailed debugging information.
func WithStack(err error) error {
	if err == nil {
		return nil
	}

	// Check if error already has a stack
	var contextErr *ContextError
	if As(err, &contextErr) && len(contextErr.Stack) > 0 {
		return err
	}

	return &ContextError{
		Message: err.Error(),
		Cause:   err,
		Stack:   captureStack(2), // Skip WithStack and captureStack
	}
}

// WithContextAndStack wraps an error with context and captures stack trace.
func WithContextAndStack(err error, message string) error {
	if err == nil {
		return nil
	}
	return &ContextError{
		Message: message,
		Cause:   err,
		Stack:   captureStack(2),
	}
}

// captureStack captures the current stack trace, skipping the specified number of frames.
func captureStack(skip int) []StackFrame {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+1, pcs[:])

	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]StackFrame, 0, n)

	for {
		frame, more := frames.Next()
		// Skip runtime and testing frames
		if strings.Contains(frame.Function, "runtime.") ||
			strings.Contains(frame.Function, "testing.") {
			if !more {
				break
			}
			continue
		}

		stack = append(stack, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})

		if !more {
			break
		}
	}

	return stack
}

// GetStack extracts the stack trace from an error if available.
func GetStack(err error) []StackFrame {
	var contextErr *ContextError
	if As(err, &contextErr) {
		return contextErr.Stack
	}
	return nil
}

// Chain returns the full error chain as a slice of error messages.
func Chain(err error) []string {
	if err == nil {
		return nil
	}

	var chain []string
	for err != nil {
		chain = append(chain, err.Error())
		err = Unwrap(err)
	}
	return chain
}

// RootCause returns the deepest wrapped error in the chain.
func RootCause(err error) error {
	for {
		unwrapped := Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// Unwrap returns the result of calling the Unwrap method on err.
// Re-exported from standard errors package for convenience.
func Unwrap(err error) error {
	u, ok := err.(interface{ Unwrap() error })
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// As is re-exported from standard errors package for convenience.
func As(err error, target interface{}) bool {
	return asError(err, target)
}

// asError is a helper that implements As without importing errors.
func asError(err error, target interface{}) bool {
	if target == nil {
		return false
	}
	for err != nil {
		// Use type assertion to check if target matches
		if x, ok := target.(*error); ok {
			*x = err
			return true
		}
		// Check if error can be assigned to target type
		if assignable, ok := err.(interface{ As(interface{}) bool }); ok {
			if assignable.As(target) {
				return true
			}
		}
		err = Unwrap(err)
	}
	return false
}
