package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		valid    bool
	}{
		// Standard Go duration formats
		{"go_duration_hours", "2h", 2 * time.Hour, true},
		{"go_duration_minutes", "30m", 30 * time.Minute, true},
		{"go_duration_seconds", "45s", 45 * time.Second, true},
		{"go_duration_combined", "1h30m", 90 * time.Minute, true},
		{"go_duration_complex", "2h30m15s", 2*time.Hour + 30*time.Minute + 15*time.Second, true},

		// Human-readable hour formats
		{"hours_h", "2h", 2 * time.Hour, true},
		{"hours_hr", "2hr", 2 * time.Hour, true},
		{"hours_hrs", "2hrs", 2 * time.Hour, true},
		{"hours_hour", "2 hour", 2 * time.Hour, true},
		{"hours_hours", "2 hours", 2 * time.Hour, true},

		// Human-readable minute formats
		{"minutes_m", "30m", 30 * time.Minute, true},
		{"minutes_min", "30min", 30 * time.Minute, true},
		{"minutes_mins", "30mins", 30 * time.Minute, true},
		{"minutes_minute", "30 minute", 30 * time.Minute, true},
		{"minutes_minutes", "30 minutes", 30 * time.Minute, true},

		// Human-readable second formats
		{"seconds_s", "45s", 45 * time.Second, true},
		{"seconds_sec", "45sec", 45 * time.Second, true},
		{"seconds_secs", "45secs", 45 * time.Second, true},
		{"seconds_second", "45 second", 45 * time.Second, true},
		{"seconds_seconds", "45 seconds", 45 * time.Second, true},

		// Decimal values
		{"decimal_hours", "1.5h", 90 * time.Minute, true},
		{"decimal_hours_2", "2.5h", 150 * time.Minute, true},
		{"decimal_minutes", "1.5m", 90 * time.Second, true},

		// Combined formats
		{"combined_hours_minutes", "1h 30m", 90 * time.Minute, true},

		// Just numbers (default to hours)
		{"number_only", "2", 2 * time.Hour, true},
		{"decimal_number", "1.5", 90 * time.Minute, true},

		// Edge cases - invalid
		{"empty_string", "", 0, false},
		{"whitespace_only", "   ", 0, false},
		{"invalid_format", "abc", 0, false},

		// Negative durations are supported by Go's time.ParseDuration
		{"negative_duration", "-1h", -1 * time.Hour, true},

		// Case insensitivity
		{"uppercase_H", "2H", 2 * time.Hour, true},
		{"uppercase_HOURS", "2 HOURS", 2 * time.Hour, true},
		{"mixed_case", "2 HoUrS", 2 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDuration(tt.input)
			assert.Equal(t, tt.valid, result.Valid, "Valid mismatch for input: %s", tt.input)
			if tt.valid {
				assert.Equal(t, tt.expected, result.Duration, "Duration mismatch for input: %s", tt.input)
			}
		})
	}
}

func TestParseDurationWhitespace(t *testing.T) {
	// Test with leading/trailing whitespace
	result := ParseDuration("  2h  ")
	assert.True(t, result.Valid)
	assert.Equal(t, 2*time.Hour, result.Duration)
}

func TestIsDurationLike(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Should be duration-like
		{"2h", true},
		{"30m", true},
		{"45s", true},
		{"2 hours", true},
		{"30 minutes", true},
		{"1hr", true},
		{"5min", true},
		{"10sec", true},
		{"2", true}, // Just a number

		// Should NOT be duration-like
		{"", false},
		{"abc", false},
		{"hours", false}, // No leading number
		{"meeting", false},
		{"project", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsDurationLike(tt.input)
			assert.Equal(t, tt.expected, result, "IsDurationLike(%q)", tt.input)
		})
	}
}

func TestUnitToDuration(t *testing.T) {
	tests := []struct {
		value    float64
		unit     string
		expected time.Duration
	}{
		// Hours
		{1, "h", time.Hour},
		{2, "hr", 2 * time.Hour},
		{3, "hrs", 3 * time.Hour},
		{1, "hour", time.Hour},
		{2, "hours", 2 * time.Hour},

		// Minutes
		{1, "m", time.Minute},
		{30, "min", 30 * time.Minute},
		{45, "mins", 45 * time.Minute},
		{1, "minute", time.Minute},
		{15, "minutes", 15 * time.Minute},

		// Seconds
		{1, "s", time.Second},
		{30, "sec", 30 * time.Second},
		{45, "secs", 45 * time.Second},
		{1, "second", time.Second},
		{60, "seconds", 60 * time.Second},

		// Decimal values
		{1.5, "h", 90 * time.Minute},
		{2.5, "m", 150 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			result := unitToDuration(tt.value, tt.unit)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test unknown unit defaults to hours
	t.Run("unknown_defaults_to_hours", func(t *testing.T) {
		result := unitToDuration(2, "xyz")
		assert.Equal(t, 2*time.Hour, result)
	})
}

func TestParseDurationZeroValue(t *testing.T) {
	// Zero duration is valid (e.g., "0h" parses to 0)
	result := ParseDuration("0h")
	assert.True(t, result.Valid)
	assert.Equal(t, time.Duration(0), result.Duration)
}

func TestParseDurationLargeValues(t *testing.T) {
	// Test large but valid durations
	result := ParseDuration("100h")
	assert.True(t, result.Valid)
	assert.Equal(t, 100*time.Hour, result.Duration)

	result = ParseDuration("1000m")
	assert.True(t, result.Valid)
	assert.Equal(t, 1000*time.Minute, result.Duration)
}
