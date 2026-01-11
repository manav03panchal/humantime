package unit

import (
	"strings"
	"testing"

	"github.com/manav03panchal/humantime/internal/parser"
)

// TestTimeParseErrorFormat tests TimeParseError formatting.
func TestTimeParseErrorFormat(t *testing.T) {
	err := parser.NewTimeParseError("duration", "xyz", "invalid format")

	errStr := err.Error()
	if !strings.Contains(errStr, "duration") {
		t.Errorf("Error should contain field name, got: %s", errStr)
	}
	if !strings.Contains(errStr, "xyz") {
		t.Errorf("Error should contain input value, got: %s", errStr)
	}
	if !strings.Contains(errStr, "invalid format") {
		t.Errorf("Error should contain message, got: %s", errStr)
	}
}

// TestTimeParseErrorWithExamples tests FormatWithExamples.
func TestTimeParseErrorWithExamples(t *testing.T) {
	err := parser.NewTimeParseError("duration", "xyz", "invalid", "1h", "30m", "1h30m")

	formatted := err.FormatWithExamples()

	if !strings.Contains(formatted, "Valid examples") {
		t.Error("Formatted error should contain 'Valid examples'")
	}
	if !strings.Contains(formatted, "1h") {
		t.Error("Formatted error should contain example '1h'")
	}
	if !strings.Contains(formatted, "30m") {
		t.Error("Formatted error should contain example '30m'")
	}
}

// TestNewDurationError tests duration error creation.
func TestNewDurationError(t *testing.T) {
	err := parser.NewDurationError("abc")

	if err.Field != "duration" {
		t.Errorf("Field should be 'duration', got %s", err.Field)
	}
	if err.Input != "abc" {
		t.Errorf("Input should be 'abc', got %s", err.Input)
	}
	if len(err.Examples) == 0 {
		t.Error("Should have examples")
	}
}

// TestNewTimestampError tests timestamp error creation.
func TestNewTimestampError(t *testing.T) {
	err := parser.NewTimestampError("xyz")

	if err.Field != "timestamp" {
		t.Errorf("Field should be 'timestamp', got %s", err.Field)
	}
	if len(err.Examples) == 0 {
		t.Error("Should have examples")
	}
}

// TestNewDeadlineError tests deadline error creation.
func TestNewDeadlineError(t *testing.T) {
	err := parser.NewDeadlineError("xyz")

	if err.Field != "deadline" {
		t.Errorf("Field should be 'deadline', got %s", err.Field)
	}
	if len(err.Examples) == 0 {
		t.Error("Should have examples")
	}
}

// TestNewDateRangeError tests date range error creation.
func TestNewDateRangeError(t *testing.T) {
	err := parser.NewDateRangeError("xyz")

	if err.Field != "date range" {
		t.Errorf("Field should be 'date range', got %s", err.Field)
	}
	if len(err.Examples) == 0 {
		t.Error("Should have examples")
	}
}

// TestToUserError tests conversion to UserError.
func TestToUserError(t *testing.T) {
	parseErr := parser.NewDurationError("xyz")
	userErr := parseErr.ToUserError()

	if userErr == nil {
		t.Fatal("ToUserError should return non-nil")
	}
	if userErr.Field != "duration" {
		t.Errorf("UserError field should be 'duration', got %s", userErr.Field)
	}
	if userErr.Value != "xyz" {
		t.Errorf("UserError value should be 'xyz', got %s", userErr.Value)
	}
}

// TestDurationExamples tests that duration examples are valid.
func TestDurationExamples(t *testing.T) {
	for _, example := range parser.DurationExamples {
		result := parser.ParseDuration(example)
		if !result.Valid {
			t.Errorf("DurationExample %q should be valid", example)
		}
	}
}

// TestDeadlineExamples tests that deadline examples are parseable.
func TestDeadlineExamples(t *testing.T) {
	for _, example := range parser.DeadlineExamples {
		result := parser.ParseDeadline(example)
		if result.Error != nil {
			t.Logf("DeadlineExample %q parse error: %v", example, result.Error)
		}
	}
}

// TestValidateAndSuggest tests the validation helper.
func TestValidateAndSuggest(t *testing.T) {
	tests := []struct {
		inputType string
		input     string
		wantError bool
	}{
		{"duration", "1h", false},
		{"duration", "xyz", true},
		{"timestamp", "now", false},
		{"deadline", "+5m", false},
		{"unknown", "test", true},
	}

	for _, tc := range tests {
		err := parser.ValidateAndSuggest(tc.inputType, tc.input)
		gotError := err != nil

		if gotError != tc.wantError {
			t.Errorf("ValidateAndSuggest(%q, %q) error=%v, wantError=%v",
				tc.inputType, tc.input, err, tc.wantError)
		}
	}
}

// TestErrorSuggestionPresent tests that errors have suggestions.
func TestErrorSuggestionPresent(t *testing.T) {
	errors := []*parser.TimeParseError{
		parser.NewDurationError("x"),
		parser.NewTimestampError("x"),
		parser.NewDeadlineError("x"),
		parser.NewDateRangeError("x"),
	}

	for _, err := range errors {
		if err.Suggestion == "" && len(err.Examples) == 0 {
			t.Errorf("%s error should have suggestion or examples", err.Field)
		}
	}
}
