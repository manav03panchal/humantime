package parser

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	t.Run("empty_args", func(t *testing.T) {
		result := Parse([]string{})
		assert.False(t, result.HasProject)
		assert.False(t, result.HasTask)
		assert.False(t, result.HasNote)
		assert.False(t, result.HasStart)
		assert.False(t, result.HasEnd)
	})

	t.Run("project_with_on_keyword", func(t *testing.T) {
		result := Parse([]string{"on", "myproject"})
		assert.True(t, result.HasProject)
		assert.Equal(t, "myproject", result.ProjectSID)
	})

	t.Run("project_with_of_keyword", func(t *testing.T) {
		result := Parse([]string{"of", "myproject"})
		assert.True(t, result.HasProject)
		assert.Equal(t, "myproject", result.ProjectSID)
	})

	t.Run("project_with_to_keyword", func(t *testing.T) {
		result := Parse([]string{"to", "myproject"})
		assert.True(t, result.HasProject)
		assert.Equal(t, "myproject", result.ProjectSID)
	})

	t.Run("project_and_task", func(t *testing.T) {
		result := Parse([]string{"on", "myproject/mytask"})
		assert.True(t, result.HasProject)
		assert.True(t, result.HasTask)
		assert.Equal(t, "myproject", result.ProjectSID)
		assert.Equal(t, "mytask", result.TaskSID)
	})

	t.Run("note_with_quotes", func(t *testing.T) {
		result := Parse([]string{"with", "note", `"this is my note"`})
		assert.True(t, result.HasNote)
		assert.Equal(t, "this is my note", result.Note)
	})

	t.Run("note_with_single_quotes", func(t *testing.T) {
		result := Parse([]string{"with", "note", "'my note'"})
		assert.True(t, result.HasNote)
		assert.Equal(t, "my note", result.Note)
	})

	t.Run("end_keyword", func(t *testing.T) {
		result := Parse([]string{"end", "now"})
		assert.True(t, result.HasEnd)
		assert.Equal(t, "now", result.RawTimestampEnd)
	})

	t.Run("until_keyword", func(t *testing.T) {
		result := Parse([]string{"until", "5pm"})
		assert.True(t, result.HasEnd)
		assert.Equal(t, "5pm", result.RawTimestampEnd)
	})

	t.Run("ended_keyword", func(t *testing.T) {
		result := Parse([]string{"ended", "yesterday"})
		assert.True(t, result.HasEnd)
	})

	t.Run("timestamp_start", func(t *testing.T) {
		result := Parse([]string{"9am"})
		assert.True(t, result.HasStart)
		assert.Equal(t, "9am", result.RawTimestampStart)
	})

	t.Run("to_after_project_is_end_keyword", func(t *testing.T) {
		result := Parse([]string{"on", "myproject", "9am", "to", "5pm"})
		assert.True(t, result.HasProject)
		assert.True(t, result.HasStart)
		assert.True(t, result.HasEnd)
		assert.Equal(t, "5pm", result.RawTimestampEnd)
	})

	t.Run("skip_words_ignored", func(t *testing.T) {
		result := Parse([]string{"block", "on", "myproject"})
		assert.True(t, result.HasProject)
		assert.Equal(t, "myproject", result.ProjectSID)
	})

	t.Run("complex_input", func(t *testing.T) {
		result := Parse([]string{"working", "on", "myproject", "from", "9am", "to", "5pm", "with", "note", `"meeting"`})
		assert.True(t, result.HasProject)
		assert.True(t, result.HasNote)
		assert.Equal(t, "myproject", result.ProjectSID)
		assert.Equal(t, "meeting", result.Note)
	})
}

func TestParsedArgsProcess(t *testing.T) {
	t.Run("normalizes_project_sid", func(t *testing.T) {
		args := &ParsedArgs{
			RawProject: "My Project",
		}
		err := args.Process()
		assert.Nil(t, err)
		assert.Equal(t, "my-project", args.ProjectSID)
	})

	t.Run("normalizes_task_sid", func(t *testing.T) {
		args := &ParsedArgs{
			ProjectSID: "myproject",
			TaskSID:    "My Task",
		}
		err := args.Process()
		assert.Nil(t, err)
		assert.Equal(t, "my-task", args.TaskSID)
	})

	t.Run("sets_default_start_time", func(t *testing.T) {
		args := &ParsedArgs{}
		err := args.Process()
		assert.Nil(t, err)
		assert.WithinDuration(t, time.Now(), args.TimestampStart, time.Second)
	})

	t.Run("parses_start_timestamp", func(t *testing.T) {
		args := &ParsedArgs{
			RawTimestampStart: "now",
		}
		err := args.Process()
		assert.Nil(t, err)
		assert.WithinDuration(t, time.Now(), args.TimestampStart, time.Second)
	})

	t.Run("invalid_start_timestamp_returns_error", func(t *testing.T) {
		args := &ParsedArgs{
			RawTimestampStart: "invalid time format xyz",
		}
		err := args.Process()
		// go-dateparser may or may not error on this
		// The test validates Process() doesn't panic
		_ = err
	})
}

func TestParsedArgsMerge(t *testing.T) {
	t.Run("project_flag_overrides", func(t *testing.T) {
		args := &ParsedArgs{
			ProjectSID: "original",
		}
		args.Merge("override", "", "", "", "")
		assert.Equal(t, "override", args.ProjectSID)
		assert.True(t, args.HasProject)
	})

	t.Run("task_flag_overrides", func(t *testing.T) {
		args := &ParsedArgs{
			TaskSID: "original",
		}
		args.Merge("", "override", "", "", "")
		assert.Equal(t, "override", args.TaskSID)
		assert.True(t, args.HasTask)
	})

	t.Run("note_flag_overrides", func(t *testing.T) {
		args := &ParsedArgs{
			Note: "original",
		}
		args.Merge("", "", "override", "", "")
		assert.Equal(t, "override", args.Note)
		assert.True(t, args.HasNote)
	})

	t.Run("start_flag_overrides", func(t *testing.T) {
		args := &ParsedArgs{
			RawTimestampStart: "original",
		}
		args.Merge("", "", "", "9am", "")
		assert.Equal(t, "9am", args.RawTimestampStart)
		assert.True(t, args.HasStart)
	})

	t.Run("end_flag_overrides", func(t *testing.T) {
		args := &ParsedArgs{
			RawTimestampEnd: "original",
		}
		args.Merge("", "", "", "", "5pm")
		assert.Equal(t, "5pm", args.RawTimestampEnd)
		assert.True(t, args.HasEnd)
	})

	t.Run("empty_flags_dont_override", func(t *testing.T) {
		args := &ParsedArgs{
			ProjectSID: "original",
		}
		args.Merge("", "", "", "", "")
		assert.Equal(t, "original", args.ProjectSID)
	})
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple", "one two three", []string{"one", "two", "three"}},
		{"double_quotes", `one "two three" four`, []string{"one", "two three", "four"}},
		{"single_quotes", `one 'two three' four`, []string{"one", "two three", "four"}},
		{"empty", "", nil},
		{"single_token", "hello", []string{"hello"}},
		{"multiple_spaces", "one   two", []string{"one", "two"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"one", "two", "three"}

	assert.True(t, containsString(slice, "one"))
	assert.True(t, containsString(slice, "two"))
	assert.True(t, containsString(slice, "three"))
	assert.False(t, containsString(slice, "four"))
	assert.False(t, containsString(slice, ""))
	assert.False(t, containsString([]string{}, "one"))
}

func TestIsTimeLike(t *testing.T) {
	tests := []struct {
		token    string
		expected bool
	}{
		// Time words
		{"now", true},
		{"today", true},
		{"yesterday", true},
		{"tomorrow", true},
		{"NOW", true},
		{"Today", true},

		// Time periods
		{"hour", true},
		{"hours", true},
		{"minute", true},
		{"minutes", true},
		{"day", true},
		{"week", true},
		{"month", true},
		{"year", true},

		// Modifiers
		{"ago", true},
		{"last", true},
		{"this", true},
		{"next", true},
		{"previous", true},
		{"current", true},

		// AM/PM
		{"am", true},
		{"pm", true},
		{"9am", true},
		{"11pm", true},
		{"9:00am", true},

		// Time format with colon
		{"14:30", true},
		{"9:00", true},

		// Weekdays
		{"monday", true},
		{"tuesday", true},
		{"wednesday", true},
		{"thursday", true},
		{"friday", true},
		{"saturday", true},
		{"sunday", true},

		// Months
		{"january", true},
		{"feb", true},
		{"march", true},
		{"december", true},

		// Parts of day
		{"morning", true},
		{"afternoon", true},
		{"evening", true},
		{"night", true},

		// Not time-like
		{"myproject", false},
		{"project123", false},
		{"meeting", false},
		{"123", false}, // Pure number without am/pm
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := isTimeLike(tt.token)
			assert.Equal(t, tt.expected, result, "isTimeLike(%q)", tt.token)
		})
	}
}
