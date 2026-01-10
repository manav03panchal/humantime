package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDeadline(t *testing.T) {
	t.Run("empty_string_returns_error", func(t *testing.T) {
		result := ParseDeadline("")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "required")
	})

	t.Run("whitespace_returns_error", func(t *testing.T) {
		result := ParseDeadline("   ")
		assert.Error(t, result.Error)
	})
}

func TestParseDeadlineRelative(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		input  string
		minAdd time.Duration
		maxAdd time.Duration
	}{
		{"plus_5_minutes", "+5m", 4*time.Minute + 50*time.Second, 5*time.Minute + 10*time.Second},
		{"plus_1_hour", "+1h", 59*time.Minute + 50*time.Second, 60*time.Minute + 10*time.Second},
		{"plus_2_days", "+2d", 47*time.Hour + 50*time.Minute, 48*time.Hour + 10*time.Minute},
		{"plus_1_week", "+1w", 7*24*time.Hour - 10*time.Second, 7*24*time.Hour + 10*time.Second},
		{"plus_30_seconds", "+30s", 29*time.Second, 31*time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDeadline(tt.input)
			assert.Nil(t, result.Error)
			diff := result.Time.Sub(now)
			assert.True(t, diff >= tt.minAdd, "Time diff %v should be >= %v", diff, tt.minAdd)
			assert.True(t, diff <= tt.maxAdd, "Time diff %v should be <= %v", diff, tt.maxAdd)
		})
	}
}

func TestParseDeadlineNaturalRelative(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name   string
		input  string
		minAdd time.Duration
		maxAdd time.Duration
	}{
		{"1_minute", "1 minute", 50 * time.Second, 70 * time.Second},
		{"5_minutes", "5 minutes", 4*time.Minute + 50*time.Second, 5*time.Minute + 10*time.Second},
		{"in_5_minutes", "in 5 minutes", 4*time.Minute + 50*time.Second, 5*time.Minute + 10*time.Second},
		{"1_hour", "1 hour", 59*time.Minute + 50*time.Second, 60*time.Minute + 10*time.Second},
		{"2_hours", "2 hours", 119*time.Minute + 50*time.Second, 120*time.Minute + 10*time.Second},
		{"in_2_hours", "in 2 hours", 119*time.Minute + 50*time.Second, 120*time.Minute + 10*time.Second},
		{"1_day", "1 day", 23*time.Hour + 50*time.Minute, 24*time.Hour + 10*time.Minute},
		{"2_days", "2 days", 47*time.Hour + 50*time.Minute, 48*time.Hour + 10*time.Minute},
		{"1_week", "1 week", 7*24*time.Hour - 10*time.Minute, 7*24*time.Hour + 10*time.Minute},

		// Abbreviated forms
		{"5_mins", "5 mins", 4*time.Minute + 50*time.Second, 5*time.Minute + 10*time.Second},
		{"1_hr", "1 hr", 59*time.Minute + 50*time.Second, 60*time.Minute + 10*time.Second},
		{"2_hrs", "2 hrs", 119*time.Minute + 50*time.Second, 120*time.Minute + 10*time.Second},
		{"1_wk", "1 wk", 7*24*time.Hour - 10*time.Minute, 7*24*time.Hour + 10*time.Minute},
		{"2_wks", "2 wks", 14*24*time.Hour - 10*time.Minute, 14*24*time.Hour + 10*time.Minute},
		{"30_seconds", "30 seconds", 29 * time.Second, 31 * time.Second},
		{"30_secs", "30 second", 29 * time.Second, 31 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseDeadline(tt.input)
			assert.Nil(t, result.Error, "Error for input %q: %v", tt.input, result.Error)
			diff := result.Time.Sub(now)
			assert.True(t, diff >= tt.minAdd, "Time diff %v should be >= %v for input %q", diff, tt.minAdd, tt.input)
			assert.True(t, diff <= tt.maxAdd, "Time diff %v should be <= %v for input %q", diff, tt.maxAdd, tt.input)
		})
	}
}

func TestParseDeadlineArgs(t *testing.T) {
	t.Run("empty_args_returns_error", func(t *testing.T) {
		result := ParseDeadlineArgs([]string{})
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "required")
	})

	t.Run("single_arg", func(t *testing.T) {
		result := ParseDeadlineArgs([]string{"+5m"})
		assert.Nil(t, result.Error)
	})

	t.Run("multiple_args_joined", func(t *testing.T) {
		result := ParseDeadlineArgs([]string{"in", "5", "minutes"})
		assert.Nil(t, result.Error)
	})
}

func TestFormatDeadline(t *testing.T) {
	now := time.Now()

	t.Run("today", func(t *testing.T) {
		deadline := time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location())
		formatted := FormatDeadline(deadline)
		assert.Contains(t, formatted, "Today")
		assert.Contains(t, formatted, "2:30 PM")
	})

	t.Run("tomorrow", func(t *testing.T) {
		tomorrow := now.AddDate(0, 0, 1)
		deadline := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 10, 0, 0, 0, now.Location())
		formatted := FormatDeadline(deadline)
		assert.Contains(t, formatted, "Tomorrow")
		assert.Contains(t, formatted, "10:00 AM")
	})

	t.Run("within_week", func(t *testing.T) {
		// 3 days from now
		future := now.AddDate(0, 0, 3)
		deadline := time.Date(future.Year(), future.Month(), future.Day(), 15, 0, 0, 0, now.Location())
		formatted := FormatDeadline(deadline)
		// Should show weekday name
		assert.Contains(t, formatted, "at")
	})

	t.Run("far_future", func(t *testing.T) {
		// 2 weeks from now
		future := now.AddDate(0, 0, 14)
		deadline := time.Date(future.Year(), future.Month(), future.Day(), 15, 0, 0, 0, now.Location())
		formatted := FormatDeadline(deadline)
		// Should show date
		assert.Contains(t, formatted, "at")
	})
}

func TestFormatTimeUntil(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		deadline time.Time
		contains string
	}{
		{"overdue", now.Add(-1 * time.Hour), "overdue"},
		{"less_than_minute", now.Add(30 * time.Second), "less than a minute"},
		{"1_minute", now.Add(1*time.Minute + 30*time.Second), "in 1 minute"},
		{"5_minutes", now.Add(5*time.Minute + 30*time.Second), "in 5 minutes"},
		{"1_hour", now.Add(1*time.Hour + 30*time.Second), "in 1 hour"},
		{"2_hours", now.Add(2*time.Hour + 30*time.Second), "in 2 hours"},
		{"1_hour_30_minutes", now.Add(1*time.Hour + 30*time.Minute + 30*time.Second), "in 1 hour 30 minutes"},
		{"1_day", now.Add(24*time.Hour + 30*time.Second), "in 1 day"},
		{"3_days", now.Add(3*24*time.Hour + 30*time.Second), "in 3 days"},
		{"1_week", now.Add(7*24*time.Hour + 30*time.Second), "in 1 week"},
		{"2_weeks", now.Add(14*24*time.Hour + 30*time.Second), "in 2 weeks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeUntil(tt.deadline)
			assert.Contains(t, result, tt.contains, "FormatTimeUntil for %s", tt.name)
		})
	}
}

func TestIsSameDay(t *testing.T) {
	now := time.Now()

	t.Run("same_day_different_times", func(t *testing.T) {
		t1 := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
		t2 := time.Date(now.Year(), now.Month(), now.Day(), 17, 0, 0, 0, now.Location())
		assert.True(t, isSameDay(t1, t2))
	})

	t.Run("different_days", func(t *testing.T) {
		t1 := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
		t2 := time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location())
		assert.False(t, isSameDay(t1, t2))
	})

	t.Run("different_months", func(t *testing.T) {
		t1 := time.Date(2026, 1, 15, 9, 0, 0, 0, now.Location())
		t2 := time.Date(2026, 2, 15, 9, 0, 0, 0, now.Location())
		assert.False(t, isSameDay(t1, t2))
	})

	t.Run("different_years", func(t *testing.T) {
		t1 := time.Date(2025, 1, 15, 9, 0, 0, 0, now.Location())
		t2 := time.Date(2026, 1, 15, 9, 0, 0, 0, now.Location())
		assert.False(t, isSameDay(t1, t2))
	})
}

func TestParseRelativeDeadlineInvalid(t *testing.T) {
	t.Run("zero_value", func(t *testing.T) {
		result := parseRelativeDeadline("0", "m")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "positive")
	})

	t.Run("invalid_unit", func(t *testing.T) {
		result := parseRelativeDeadline("5", "x")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "invalid time unit")
	})
}

func TestParseNaturalRelativeDeadlineInvalid(t *testing.T) {
	t.Run("zero_value", func(t *testing.T) {
		result := parseNaturalRelativeDeadline("0", "minute")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "positive")
	})

	t.Run("invalid_unit", func(t *testing.T) {
		result := parseNaturalRelativeDeadline("5", "xyz")
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "invalid time unit")
	})
}
