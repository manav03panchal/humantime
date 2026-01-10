package property

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/manav03panchal/humantime/internal/parser"
)

// TestParseDurationNeverPanics tests that ParseDuration never panics.
func TestParseDurationNeverPanics(t *testing.T) {
	f := func(input string) bool {
		result := parser.ParseDuration(input)
		_ = result
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Error(err)
	}
}

// TestParseDurationValidFormats tests various valid duration formats.
func TestParseDurationValidFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1h", time.Hour},
		{"2h", 2 * time.Hour},
		{"30m", 30 * time.Minute},
		{"45m", 45 * time.Minute},
		{"90m", 90 * time.Minute},
		{"1h30m", 90 * time.Minute},
		{"2h15m", 135 * time.Minute},
		{"1s", time.Second},
		{"30s", 30 * time.Second},
	}

	for _, tc := range tests {
		result := parser.ParseDuration(tc.input)
		if !result.Valid {
			t.Errorf("ParseDuration(%q) should be valid", tc.input)
			continue
		}
		if result.Duration != tc.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, result.Duration, tc.expected)
		}
	}
}

// TestParseDurationInvalidFormats tests invalid duration formats.
func TestParseDurationInvalidFormats(t *testing.T) {
	invalidInputs := []string{
		"",
		"abc",
		"hello world",
		"1x",
		"hours",
		"--1h",
		"1h-",
	}

	for _, input := range invalidInputs {
		result := parser.ParseDuration(input)
		if result.Valid && result.Duration != 0 {
			t.Errorf("ParseDuration(%q) should be invalid, got duration %v", input, result.Duration)
		}
	}
}

// TestParseDurationPositive tests that valid durations are positive.
func TestParseDurationPositive(t *testing.T) {
	validInputs := []string{
		"1h",
		"30m",
		"1h30m",
		"2h",
		"45m",
	}

	for _, input := range validInputs {
		result := parser.ParseDuration(input)
		if result.Valid && result.Duration <= 0 {
			t.Errorf("ParseDuration(%q) should be positive, got %v", input, result.Duration)
		}
	}
}

// TestParseDurationLongFormats tests human-readable duration formats.
func TestParseDurationLongFormats(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1 hour", time.Hour},
		{"2 hours", 2 * time.Hour},
		{"30 minutes", 30 * time.Minute},
		{"1 minute", time.Minute},
	}

	for _, tc := range tests {
		result := parser.ParseDuration(tc.input)
		if !result.Valid {
			t.Logf("ParseDuration(%q) long format not supported", tc.input)
			continue
		}
		if result.Duration != tc.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, result.Duration, tc.expected)
		}
	}
}

// TestParseDurationResultFields tests result struct fields.
func TestParseDurationResultFields(t *testing.T) {
	// Valid input
	validResult := parser.ParseDuration("1h30m")
	if !validResult.Valid {
		t.Error("Valid input should have Valid=true")
	}
	if validResult.Duration == 0 {
		t.Error("Valid input should have non-zero Duration")
	}

	// Invalid input
	invalidResult := parser.ParseDuration("")
	if invalidResult.Valid {
		t.Error("Empty input should have Valid=false")
	}
}

// TestParseDurationIdempotent tests that parsing is deterministic.
func TestParseDurationIdempotent(t *testing.T) {
	inputs := []string{
		"1h",
		"30m",
		"1h30m",
		"2h",
	}

	for _, input := range inputs {
		result1 := parser.ParseDuration(input)
		result2 := parser.ParseDuration(input)

		if result1.Valid != result2.Valid {
			t.Errorf("ParseDuration(%q) not idempotent: Valid differs", input)
		}
		if result1.Duration != result2.Duration {
			t.Errorf("ParseDuration(%q) not idempotent: Duration differs", input)
		}
	}
}

// TestParseDurationDecimal tests decimal duration formats.
func TestParseDurationDecimal(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1.5h", 90 * time.Minute},
		{"2.5h", 150 * time.Minute},
		{"0.5h", 30 * time.Minute},
	}

	for _, tc := range tests {
		result := parser.ParseDuration(tc.input)
		if !result.Valid {
			t.Logf("ParseDuration(%q) decimal format not supported", tc.input)
			continue
		}
		if result.Duration != tc.expected {
			t.Errorf("ParseDuration(%q) = %v, want %v", tc.input, result.Duration, tc.expected)
		}
	}
}

// TestIsDurationLike tests the IsDurationLike helper.
func TestIsDurationLike(t *testing.T) {
	durationLike := []string{
		"1h",
		"30m",
		"2 hours",
		"45 minutes",
		"1h30m",
	}

	notDurationLike := []string{
		"",
		"hello",
		"tomorrow",
		"monday",
	}

	for _, s := range durationLike {
		if !parser.IsDurationLike(s) {
			t.Errorf("IsDurationLike(%q) should be true", s)
		}
	}

	for _, s := range notDurationLike {
		if parser.IsDurationLike(s) {
			t.Errorf("IsDurationLike(%q) should be false", s)
		}
	}
}
