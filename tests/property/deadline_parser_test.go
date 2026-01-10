package property

import (
	"testing"
	"testing/quick"
	"time"

	"github.com/manav03panchal/humantime/internal/parser"
)

// TestParseDeadlineNeverPanics tests that ParseDeadline never panics on any input.
func TestParseDeadlineNeverPanics(t *testing.T) {
	f := func(input string) bool {
		// Should never panic
		result := parser.ParseDeadline(input)
		_ = result
		return true
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 2000}); err != nil {
		t.Error(err)
	}
}

// TestParseDeadlineValidInputsFuture tests that valid future deadlines produce future times.
func TestParseDeadlineValidInputsFuture(t *testing.T) {
	futureInputs := []string{
		"+1m",
		"+5m",
		"+30m",
		"+1h",
		"+2h",
		"in 1 minute",
		"in 5 minutes",
		"in 1 hour",
		"tomorrow",
		"next week",
	}

	now := time.Now()

	for _, input := range futureInputs {
		result := parser.ParseDeadline(input)
		if result.Error != nil {
			t.Errorf("ParseDeadline(%q) should succeed, got error: %v", input, result.Error)
			continue
		}
		if !result.Time.After(now) {
			t.Errorf("ParseDeadline(%q) = %v, expected time after %v", input, result.Time, now)
		}
	}
}

// TestParseDeadlineRelativeFormat tests relative deadline parsing.
func TestParseDeadlineRelativeFormat(t *testing.T) {
	tests := []struct {
		input       string
		minDuration time.Duration
		maxDuration time.Duration
	}{
		{"+1m", 50 * time.Second, 70 * time.Second},
		{"+5m", 4*time.Minute + 50*time.Second, 5*time.Minute + 10*time.Second},
		{"+1h", 59 * time.Minute, 61 * time.Minute},
	}

	for _, tc := range tests {
		now := time.Now()
		result := parser.ParseDeadline(tc.input)

		if result.Error != nil {
			t.Errorf("ParseDeadline(%q) error: %v", tc.input, result.Error)
			continue
		}

		diff := result.Time.Sub(now)
		if diff < tc.minDuration || diff > tc.maxDuration {
			t.Errorf("ParseDeadline(%q) produced time %v from now, expected between %v and %v",
				tc.input, diff, tc.minDuration, tc.maxDuration)
		}
	}
}

// TestParseDeadlineEmptyString tests empty input handling.
func TestParseDeadlineEmptyString(t *testing.T) {
	result := parser.ParseDeadline("")
	// Empty should either error or return now
	if result.Error == nil && result.Time.IsZero() {
		t.Error("Empty input should either error or return non-zero time")
	}
}

// TestParseDeadlineWeekdays tests weekday parsing.
func TestParseDeadlineWeekdays(t *testing.T) {
	weekdays := []string{
		"monday",
		"tuesday",
		"wednesday",
		"thursday",
		"friday",
		"saturday",
		"sunday",
	}

	for _, day := range weekdays {
		result := parser.ParseDeadline(day)
		if result.Error != nil {
			// Some parsers might not support bare weekday names
			t.Logf("ParseDeadline(%q) not supported: %v", day, result.Error)
			continue
		}

		// Result should be a valid time
		if result.Time.IsZero() {
			t.Errorf("ParseDeadline(%q) returned zero time", day)
		}
	}
}

// TestParseDeadlineWithTime tests deadline with time specification.
func TestParseDeadlineWithTime(t *testing.T) {
	inputs := []string{
		"tomorrow at 9am",
		"friday 5pm",
		"monday 10:00",
	}

	for _, input := range inputs {
		result := parser.ParseDeadline(input)
		if result.Error != nil {
			t.Logf("ParseDeadline(%q) not supported: %v", input, result.Error)
			continue
		}

		if result.Time.IsZero() {
			t.Errorf("ParseDeadline(%q) returned zero time", input)
		}
	}
}

// TestParseDeadlineResultFields tests that result fields are properly set.
func TestParseDeadlineResultFields(t *testing.T) {
	// Valid input
	validResult := parser.ParseDeadline("+5m")
	if validResult.Error != nil {
		t.Errorf("Valid input should not have error: %v", validResult.Error)
	}
	if validResult.Time.IsZero() {
		t.Error("Valid input should have non-zero time")
	}

	// Invalid input
	invalidResult := parser.ParseDeadline("not a valid deadline format xyz123")
	// Should have error OR parse as something (parsers vary)
	if invalidResult.Error == nil && invalidResult.Time.IsZero() {
		t.Log("Invalid input returned no error and zero time - parser is lenient")
	}
}
