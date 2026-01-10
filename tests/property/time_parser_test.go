package property

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/manav03panchal/humantime/internal/parser"
)

// TestParseDeadlineProperty tests that ParseDeadline:
// 1. Never panics on any input
// 2. Returns a time in the future for valid future inputs
// 3. Returns consistent results for the same input
func TestParseDeadlineProperty(t *testing.T) {
	// Property: ParseDeadline should never panic
	f := func(input string) bool {
		result := parser.ParseDeadline(input)
		// If there's an error, that's fine - we just want no panic
		if result.Error != nil {
			return true
		}
		// If successful, time should be non-zero
		return !result.Time.IsZero()
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// TestParseDurationProperty tests that ParseDuration:
// 1. Never panics on any input
// 2. Returns positive duration for valid inputs
func TestParseDurationProperty(t *testing.T) {
	f := func(input string) bool {
		result := parser.ParseDuration(input)
		// If not valid, that's fine
		if !result.Valid {
			return true
		}
		// Valid durations should be positive
		return result.Duration > 0
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// TestParseTimestampProperty tests that ParseTimestamp:
// 1. Never panics on any input
// 2. Returns a valid time for valid inputs
func TestParseTimestampProperty(t *testing.T) {
	f := func(input string) bool {
		result := parser.ParseTimestamp(input)
		// If there's an error, that's fine
		if result.Error != nil {
			return true
		}
		// Valid timestamps should be non-zero
		return !result.Time.IsZero()
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Error(err)
	}
}

// TestRelativeDeadlinesFuture tests that relative deadlines like "+5m" produce future times.
func TestRelativeDeadlinesFuture(t *testing.T) {
	inputs := []string{
		"+1m",
		"+5m",
		"+1h",
		"+30m",
		"+2h",
		"in 1 minute",
		"in 5 minutes",
		"1 minute",
		"5 minutes",
	}

	now := time.Now()

	for _, input := range inputs {
		result := parser.ParseDeadline(input)
		if result.Error != nil {
			t.Errorf("ParseDeadline(%q) returned error: %v", input, result.Error)
			continue
		}
		if !result.Time.After(now) {
			t.Errorf("ParseDeadline(%q) = %v, expected time after %v", input, result.Time, now)
		}
	}
}

// TestDurationRoundTrip tests that parsing formatted durations produces correct values.
func TestDurationRoundTrip(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Duration
	}{
		{"1h", time.Hour},
		{"30m", 30 * time.Minute},
		{"1h30m", 90 * time.Minute},
		{"2h", 2 * time.Hour},
		{"45m", 45 * time.Minute},
		{"90m", 90 * time.Minute},
	}

	for _, tc := range testCases {
		result := parser.ParseDuration(tc.input)
		if !result.Valid {
			t.Errorf("ParseDuration(%q) returned invalid result", tc.input)
			continue
		}
		if result.Duration != tc.expected {
			t.Errorf("ParseDuration(%q) = %v, expected %v", tc.input, result.Duration, tc.expected)
		}
	}
}

// TestDeadlineIdempotency tests that parsing the same input twice gives the same result.
func TestDeadlineIdempotency(t *testing.T) {
	inputs := []string{
		"+5m",
		"tomorrow",
		"next monday",
		"friday 5pm",
	}

	for _, input := range inputs {
		result1 := parser.ParseDeadline(input)
		result2 := parser.ParseDeadline(input)

		// Both should have the same error status
		if (result1.Error == nil) != (result2.Error == nil) {
			t.Errorf("ParseDeadline(%q) inconsistent error status", input)
			continue
		}

		// If both succeeded, times should be very close (within a second due to "now" reference)
		if result1.Error == nil {
			diff := result1.Time.Sub(result2.Time)
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("ParseDeadline(%q) inconsistent results: %v vs %v", input, result1.Time, result2.Time)
			}
		}
	}
}
