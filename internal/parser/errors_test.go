package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeParseErrorError(t *testing.T) {
	err := &TimeParseError{
		Input:   "badtime",
		Field:   "timestamp",
		Message: "could not parse time",
	}
	result := err.Error()
	assert.Contains(t, result, "invalid timestamp")
	assert.Contains(t, result, "badtime")
	assert.Contains(t, result, "could not parse time")
}

func TestNewTimeParseError(t *testing.T) {
	err := NewTimeParseError("duration", "xyz", "invalid format", "1h", "30m", "2h30m")
	assert.Equal(t, "duration", err.Field)
	assert.Equal(t, "xyz", err.Input)
	assert.Equal(t, "invalid format", err.Message)
	assert.Len(t, err.Examples, 3)
	assert.Equal(t, "1h", err.Examples[0])
}

func TestFormatWithExamples(t *testing.T) {
	t.Run("with_examples", func(t *testing.T) {
		err := &TimeParseError{
			Input:    "badtime",
			Field:    "timestamp",
			Message:  "could not parse",
			Examples: []string{"9am", "10:30", "2pm"},
		}
		result := err.FormatWithExamples()
		assert.Contains(t, result, "invalid timestamp")
		assert.Contains(t, result, "Valid examples:")
		assert.Contains(t, result, "9am")
		assert.Contains(t, result, "10:30")
		assert.Contains(t, result, "2pm")
	})

	t.Run("with_suggestion", func(t *testing.T) {
		err := &TimeParseError{
			Input:      "badtime",
			Field:      "timestamp",
			Message:    "could not parse",
			Examples:   []string{"9am"},
			Suggestion: "Try using natural language",
		}
		result := err.FormatWithExamples()
		assert.Contains(t, result, "Try using natural language")
	})

	t.Run("no_examples_no_suggestion", func(t *testing.T) {
		err := &TimeParseError{
			Input:   "badtime",
			Field:   "timestamp",
			Message: "could not parse",
		}
		result := err.FormatWithExamples()
		assert.Contains(t, result, "invalid timestamp")
		assert.NotContains(t, result, "Valid examples:")
	})
}

func TestNewDurationError(t *testing.T) {
	err := NewDurationError("badvalue")
	assert.Equal(t, "duration", err.Field)
	assert.Equal(t, "badvalue", err.Input)
	assert.Contains(t, err.Message, "could not parse duration")
	assert.Equal(t, DurationExamples, err.Examples)
	assert.Contains(t, err.Suggestion, "hours")
	assert.Contains(t, err.Suggestion, "minutes")
}

func TestNewTimestampError(t *testing.T) {
	err := NewTimestampError("notadate")
	assert.Equal(t, "timestamp", err.Field)
	assert.Equal(t, "notadate", err.Input)
	assert.Contains(t, err.Message, "could not parse time")
	assert.Equal(t, TimestampExamples, err.Examples)
	assert.Contains(t, err.Suggestion, "natural language")
}

func TestNewDeadlineError(t *testing.T) {
	err := NewDeadlineError("sometime")
	assert.Equal(t, "deadline", err.Field)
	assert.Equal(t, "sometime", err.Input)
	assert.Contains(t, err.Message, "could not parse deadline")
	assert.Equal(t, DeadlineExamples, err.Examples)
	assert.Contains(t, err.Suggestion, "relative")
}

func TestNewDateRangeError(t *testing.T) {
	err := NewDateRangeError("whenever")
	assert.Equal(t, "date range", err.Field)
	assert.Equal(t, "whenever", err.Input)
	assert.Contains(t, err.Message, "could not parse date range")
	assert.Equal(t, DateRangeExamples, err.Examples)
	assert.Contains(t, err.Suggestion, "period names")
}

func TestToUserError(t *testing.T) {
	t.Run("with_suggestion", func(t *testing.T) {
		err := &TimeParseError{
			Input:      "badtime",
			Field:      "timestamp",
			Message:    "could not parse",
			Suggestion: "Try using natural language",
		}
		userErr := err.ToUserError()
		assert.NotNil(t, userErr)
		assert.Contains(t, userErr.Error(), "badtime")
		assert.Equal(t, "timestamp", userErr.Field)
	})

	t.Run("with_examples_no_suggestion", func(t *testing.T) {
		err := &TimeParseError{
			Input:    "badtime",
			Field:    "timestamp",
			Message:  "could not parse",
			Examples: []string{"9am", "10:30", "2pm", "5pm"},
		}
		userErr := err.ToUserError()
		assert.NotNil(t, userErr)
	})
}

func TestValidateAndSuggest(t *testing.T) {
	t.Run("valid_duration", func(t *testing.T) {
		err := ValidateAndSuggest("duration", "1h30m")
		assert.NoError(t, err)
	})

	t.Run("invalid_duration", func(t *testing.T) {
		err := ValidateAndSuggest("duration", "notaduration")
		assert.Error(t, err)
		parseErr, ok := err.(*TimeParseError)
		assert.True(t, ok)
		assert.Equal(t, "duration", parseErr.Field)
	})

	t.Run("valid_timestamp", func(t *testing.T) {
		err := ValidateAndSuggest("timestamp", "9am")
		assert.NoError(t, err)
	})

	t.Run("invalid_timestamp", func(t *testing.T) {
		err := ValidateAndSuggest("timestamp", "notatime")
		assert.Error(t, err)
	})

	t.Run("valid_deadline", func(t *testing.T) {
		err := ValidateAndSuggest("deadline", "+1h")
		assert.NoError(t, err)
	})

	t.Run("invalid_deadline", func(t *testing.T) {
		err := ValidateAndSuggest("deadline", "notadeadline")
		assert.Error(t, err)
	})

	t.Run("unknown_type", func(t *testing.T) {
		err := ValidateAndSuggest("unknown", "anything")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown input type")
	})
}

func TestMin(t *testing.T) {
	t.Run("a_less_than_b", func(t *testing.T) {
		result := min(3, 5)
		assert.Equal(t, 3, result)
	})

	t.Run("b_less_than_a", func(t *testing.T) {
		result := min(7, 4)
		assert.Equal(t, 4, result)
	})

	t.Run("equal", func(t *testing.T) {
		result := min(5, 5)
		assert.Equal(t, 5, result)
	})
}
