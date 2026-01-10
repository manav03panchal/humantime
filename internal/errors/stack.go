package errors

import (
	"fmt"
	"strings"
)

// FormatStackTrace formats a stack trace for display.
func FormatStackTrace(frames []StackFrame) string {
	if len(frames) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Stack trace:\n")

	for i, frame := range frames {
		sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, frame.Function))
		sb.WriteString(fmt.Sprintf("       at %s:%d\n", frame.File, frame.Line))
	}

	return sb.String()
}

// FormatDebugError formats an error with full debug information.
// This includes the error chain and stack trace if available.
func FormatDebugError(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Error message
	sb.WriteString("Error: ")
	sb.WriteString(err.Error())
	sb.WriteString("\n")

	// Error chain
	chain := Chain(err)
	if len(chain) > 1 {
		sb.WriteString("\nError chain:\n")
		for i, msg := range chain {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, msg))
		}
	}

	// Category
	category := GetCategory(err)
	sb.WriteString(fmt.Sprintf("\nCategory: %s\n", category.String()))

	// Suggestion
	if suggestion := GetSuggestion(err); suggestion != "" {
		sb.WriteString(fmt.Sprintf("\nSuggestion: %s\n", suggestion))
	}

	// Stack trace
	if stack := GetStack(err); len(stack) > 0 {
		sb.WriteString("\n")
		sb.WriteString(FormatStackTrace(stack))
	}

	// Root cause
	root := RootCause(err)
	if root != err {
		sb.WriteString(fmt.Sprintf("\nRoot cause: %v\n", root))
	}

	return sb.String()
}

// FormatUserError formats an error for display to the user.
// This provides a clean, actionable message without technical details.
func FormatUserError(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Main error message
	sb.WriteString(err.Error())

	// Add suggestion if available
	if suggestion := GetSuggestion(err); suggestion != "" {
		sb.WriteString("\n\n")
		sb.WriteString(suggestion)
	}

	// Add examples if it's a UserError with them
	if ue, ok := AsUserError(err); ok {
		examples := CommandExamples[err]
		if len(examples) == 0 {
			// Try to find examples for the underlying error
			for knownErr, ex := range CommandExamples {
				if Is(err, knownErr) {
					examples = ex
					break
				}
			}
		}
		if len(examples) > 0 {
			sb.WriteString("\n\nExamples:\n")
			for _, ex := range examples {
				sb.WriteString("  ")
				sb.WriteString(ex)
				sb.WriteString("\n")
			}
		}
		_ = ue // Silence unused variable warning
	}

	return sb.String()
}

// Is reports whether any error in err's tree matches target.
// Re-exported from standard errors package for convenience.
func Is(err, target error) bool {
	if target == nil {
		return err == target
	}

	for {
		if err == target {
			return true
		}
		if x, ok := err.(interface{ Is(error) bool }); ok && x.Is(target) {
			return true
		}
		err = Unwrap(err)
		if err == nil {
			return false
		}
	}
}
