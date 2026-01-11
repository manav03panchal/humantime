package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTimestamp(t *testing.T) {
	now := time.Now()

	t.Run("empty_string_returns_now", func(t *testing.T) {
		result := ParseTimestamp("")
		assert.Nil(t, result.Error)
		assert.WithinDuration(t, now, result.Time, time.Second)
	})

	t.Run("now_returns_now", func(t *testing.T) {
		result := ParseTimestamp("now")
		assert.Nil(t, result.Error)
		assert.WithinDuration(t, now, result.Time, time.Second)
	})

	t.Run("NOW_case_insensitive", func(t *testing.T) {
		result := ParseTimestamp("NOW")
		assert.Nil(t, result.Error)
		assert.WithinDuration(t, now, result.Time, time.Second)
	})

	t.Run("whitespace_trimmed", func(t *testing.T) {
		result := ParseTimestamp("  now  ")
		assert.Nil(t, result.Error)
		assert.WithinDuration(t, now, result.Time, time.Second)
	})
}

func TestParseTimestampPeriods(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, result TimestampResult)
	}{
		{
			name:  "this_hour",
			input: "this hour",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, now.Hour(), result.Time.Hour())
				assert.Equal(t, 0, result.Time.Minute())
				assert.Equal(t, 0, result.Time.Second())
			},
		},
		{
			name:  "last_hour",
			input: "last hour",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				expected := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Hour(), result.Time.Hour())
			},
		},
		{
			name:  "this_day",
			input: "this day",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, now.Day(), result.Time.Day())
				assert.Equal(t, 0, result.Time.Hour())
			},
		},
		{
			name:  "last_day",
			input: "last day",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				yesterday := now.AddDate(0, 0, -1)
				assert.Equal(t, yesterday.Day(), result.Time.Day())
			},
		},
		{
			name:  "this_week",
			input: "this week",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				// Should be Monday of this week
				assert.Equal(t, time.Monday, result.Time.Weekday())
			},
		},
		{
			name:  "last_week",
			input: "last week",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, time.Monday, result.Time.Weekday())
			},
		},
		{
			name:  "this_month",
			input: "this month",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, 1, result.Time.Day())
				assert.Equal(t, now.Month(), result.Time.Month())
			},
		},
		{
			name:  "last_month",
			input: "last month",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "this_year",
			input: "this year",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, 1, result.Time.Day())
				assert.Equal(t, time.January, result.Time.Month())
				assert.Equal(t, now.Year(), result.Time.Year())
			},
		},
		{
			name:  "last_year",
			input: "last year",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, 1, result.Time.Day())
				assert.Equal(t, time.January, result.Time.Month())
				assert.Equal(t, now.Year()-1, result.Time.Year())
			},
		},
		{
			name:  "current_week",
			input: "current week",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, time.Monday, result.Time.Weekday())
			},
		},
		{
			name:  "previous_month",
			input: "previous month",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "this_quarter",
			input: "this quarter",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
				// Quarter start month should be January, April, July, or October
				month := result.Time.Month()
				assert.True(t, month == time.January || month == time.April ||
					month == time.July || month == time.October)
			},
		},
		{
			name:  "last_quarter",
			input: "last quarter",
			validate: func(t *testing.T, result TimestampResult) {
				assert.Nil(t, result.Error)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTimestamp(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestParseTimestampCaseInsensitive(t *testing.T) {
	tests := []string{
		"THIS WEEK",
		"This Week",
		"this Week",
		"THIS week",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := ParseTimestamp(input)
			assert.Nil(t, result.Error)
			assert.Equal(t, time.Monday, result.Time.Weekday())
		})
	}
}

func TestGetPeriodRange(t *testing.T) {
	now := time.Now()

	t.Run("today", func(t *testing.T) {
		r := GetPeriodRange("today")
		assert.Equal(t, now.Day(), r.Start.Day())
		assert.Equal(t, 0, r.Start.Hour())
		assert.Equal(t, r.Start.AddDate(0, 0, 1), r.End)
	})

	t.Run("yesterday", func(t *testing.T) {
		r := GetPeriodRange("yesterday")
		yesterday := now.AddDate(0, 0, -1)
		assert.Equal(t, yesterday.Day(), r.Start.Day())
		assert.Equal(t, r.Start.AddDate(0, 0, 1), r.End)
	})

	t.Run("this_week", func(t *testing.T) {
		r := GetPeriodRange("this week")
		assert.Equal(t, time.Monday, r.Start.Weekday())
		assert.Equal(t, r.Start.AddDate(0, 0, 7), r.End)
	})

	t.Run("last_week", func(t *testing.T) {
		r := GetPeriodRange("last week")
		assert.Equal(t, time.Monday, r.Start.Weekday())
		assert.Equal(t, r.Start.AddDate(0, 0, 7), r.End)
	})

	t.Run("this_month", func(t *testing.T) {
		r := GetPeriodRange("this month")
		assert.Equal(t, 1, r.Start.Day())
		assert.Equal(t, now.Month(), r.Start.Month())
	})

	t.Run("last_month", func(t *testing.T) {
		r := GetPeriodRange("last month")
		assert.Equal(t, 1, r.Start.Day())
	})

	t.Run("this_year", func(t *testing.T) {
		r := GetPeriodRange("this year")
		assert.Equal(t, 1, r.Start.Day())
		assert.Equal(t, time.January, r.Start.Month())
		assert.Equal(t, now.Year(), r.Start.Year())
	})

	t.Run("last_year", func(t *testing.T) {
		r := GetPeriodRange("last year")
		assert.Equal(t, 1, r.Start.Day())
		assert.Equal(t, time.January, r.Start.Month())
		assert.Equal(t, now.Year()-1, r.Start.Year())
	})

	t.Run("current_week", func(t *testing.T) {
		r := GetPeriodRange("current week")
		assert.Equal(t, time.Monday, r.Start.Weekday())
	})

	t.Run("default_to_today", func(t *testing.T) {
		r := GetPeriodRange("invalid")
		assert.Equal(t, now.Day(), r.Start.Day())
	})
}

func TestTimeRange(t *testing.T) {
	r := GetPeriodRange("today")
	assert.True(t, r.End.After(r.Start))
	assert.Equal(t, 24*time.Hour, r.End.Sub(r.Start))
}
