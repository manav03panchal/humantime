// Package contract provides contract tests for the parser package.
package contract

import (
	"strings"
	"testing"
	"time"

	"github.com/manav03panchal/humantime/internal/parser"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// SID Utilities Tests (sid.go)
// =============================================================================

func TestValidateSID_ValidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "simple alphanumeric",
			input: "project1",
			want:  true,
		},
		{
			name:  "with hyphens",
			input: "my-project",
			want:  true,
		},
		{
			name:  "with underscores",
			input: "my_project",
			want:  true,
		},
		{
			name:  "with periods",
			input: "my.project",
			want:  true,
		},
		{
			name:  "mixed characters",
			input: "my-project_v1.0",
			want:  true,
		},
		{
			name:  "uppercase letters",
			input: "MyProject",
			want:  true,
		},
		{
			name:  "numbers only",
			input: "12345",
			want:  true,
		},
		{
			name:  "single character",
			input: "a",
			want:  true,
		},
		{
			name:  "max length (32 chars)",
			input: strings.Repeat("a", 32),
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.ValidateSID(tt.input)
			assert.Equal(t, tt.want, got, "ValidateSID(%q) should return %v", tt.input, tt.want)
		})
	}
}

func TestValidateSID_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "exceeds max length",
			input: strings.Repeat("a", 33),
			want:  false,
		},
		{
			name:  "contains space",
			input: "my project",
			want:  false,
		},
		{
			name:  "contains special characters",
			input: "my@project",
			want:  false,
		},
		{
			name:  "contains exclamation",
			input: "project!",
			want:  false,
		},
		{
			name:  "reserved word: edit",
			input: "edit",
			want:  false,
		},
		{
			name:  "reserved word: create",
			input: "create",
			want:  false,
		},
		{
			name:  "reserved word: delete",
			input: "delete",
			want:  false,
		},
		{
			name:  "reserved word: list",
			input: "list",
			want:  false,
		},
		{
			name:  "reserved word: show",
			input: "show",
			want:  false,
		},
		{
			name:  "reserved word: set",
			input: "set",
			want:  false,
		},
		{
			name:  "reserved word uppercase: EDIT",
			input: "EDIT",
			want:  false,
		},
		{
			name:  "reserved word mixed case: Edit",
			input: "Edit",
			want:  false,
		},
		{
			name:  "contains slash",
			input: "project/task",
			want:  false,
		},
		{
			name:  "contains hash",
			input: "project#1",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.ValidateSID(tt.input)
			assert.Equal(t, tt.want, got, "ValidateSID(%q) should return %v", tt.input, tt.want)
		})
	}
}

func TestConvertToSID(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		want        string
	}{
		{
			name:        "simple conversion",
			displayName: "My Project",
			want:        "my-project",
		},
		{
			name:        "already lowercase",
			displayName: "my project",
			want:        "my-project",
		},
		{
			name:        "removes special characters",
			displayName: "My Client Project!",
			want:        "my-client-project",
		},
		{
			name:        "removes multiple special chars",
			displayName: "Project @#$ Name!!!",
			want:        "project-name",
		},
		{
			name:        "preserves underscores",
			displayName: "my_project_name",
			want:        "my_project_name",
		},
		{
			name:        "preserves periods",
			displayName: "version.1.0",
			want:        "version.1.0",
		},
		{
			name:        "handles consecutive spaces",
			displayName: "my    project",
			want:        "my-project",
		},
		{
			name:        "trims leading/trailing hyphens",
			displayName: " -My Project- ",
			want:        "my-project",
		},
		{
			name:        "truncates long names",
			displayName: "This Is A Very Long Project Name That Exceeds Maximum",
			want:        "this-is-a-very-long-project-name",
		},
		{
			name:        "empty string",
			displayName: "",
			want:        "",
		},
		{
			name:        "only special characters",
			displayName: "!@#$%",
			want:        "",
		},
		{
			name:        "mixed unicode",
			displayName: "Project Cafe",
			want:        "project-cafe",
		},
		{
			name:        "numbers preserved",
			displayName: "Project 123",
			want:        "project-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.ConvertToSID(tt.displayName)
			assert.Equal(t, tt.want, got, "ConvertToSID(%q)", tt.displayName)
		})
	}
}

func TestParseProjectTask(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantProjectSID string
		wantTaskSID    string
	}{
		{
			name:           "project only",
			input:          "myproject",
			wantProjectSID: "myproject",
			wantTaskSID:    "",
		},
		{
			name:           "project and task",
			input:          "myproject/mytask",
			wantProjectSID: "myproject",
			wantTaskSID:    "mytask",
		},
		{
			name:           "project and task with spaces",
			input:          " myproject / mytask ",
			wantProjectSID: "myproject",
			wantTaskSID:    "mytask",
		},
		{
			name:           "multiple slashes",
			input:          "project/task/subtask",
			wantProjectSID: "project",
			wantTaskSID:    "task/subtask",
		},
		{
			name:           "empty task",
			input:          "project/",
			wantProjectSID: "project",
			wantTaskSID:    "",
		},
		{
			name:           "empty string",
			input:          "",
			wantProjectSID: "",
			wantTaskSID:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProject, gotTask := parser.ParseProjectTask(tt.input)
			assert.Equal(t, tt.wantProjectSID, gotProject, "ProjectSID mismatch")
			assert.Equal(t, tt.wantTaskSID, gotTask, "TaskSID mismatch")
		})
	}
}

func TestNormalizeSID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already valid SID",
			input: "myproject",
			want:  "myproject",
		},
		{
			name:  "needs conversion",
			input: "My Project",
			want:  "my-project",
		},
		{
			name:  "with leading/trailing whitespace",
			input: "  myproject  ",
			want:  "myproject",
		},
		{
			name:  "reserved word gets converted",
			input: "edit",
			want:  "edit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.NormalizeSID(tt.input)
			assert.Equal(t, tt.want, got, "NormalizeSID(%q)", tt.input)
		})
	}
}

// =============================================================================
// Timestamp Parser Tests (timestamp.go)
// =============================================================================

func TestParseTimestamp_EmptyAndNow(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "now lowercase",
			input: "now",
		},
		{
			name:  "now uppercase",
			input: "NOW",
		},
		{
			name:  "now mixed case",
			input: "Now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tt.input)
			assert.NoError(t, result.Error, "ParseTimestamp(%q) should not error", tt.input)
			// Should be within 1 second of now
			assert.WithinDuration(t, time.Now(), result.Time, time.Second,
				"ParseTimestamp(%q) should return current time", tt.input)
		})
	}
}

func TestParseTimestamp_RelativeTimes(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		checkDuration bool
		minAgo        time.Duration
		maxAgo        time.Duration
	}{
		{
			name:          "2 hours ago",
			input:         "2 hours ago",
			expectError:   false,
			checkDuration: true,
			minAgo:        1*time.Hour + 59*time.Minute,
			maxAgo:        2*time.Hour + 1*time.Minute,
		},
		{
			name:          "1 hour ago",
			input:         "1 hour ago",
			expectError:   false,
			checkDuration: true,
			minAgo:        59 * time.Minute,
			maxAgo:        61 * time.Minute,
		},
		{
			name:          "30 minutes ago",
			input:         "30 minutes ago",
			expectError:   false,
			checkDuration: true,
			minAgo:        29 * time.Minute,
			maxAgo:        31 * time.Minute,
		},
		{
			name:        "yesterday",
			input:       "yesterday",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tt.input)
			if tt.expectError {
				assert.Error(t, result.Error, "ParseTimestamp(%q) should error", tt.input)
				return
			}
			assert.NoError(t, result.Error, "ParseTimestamp(%q) should not error", tt.input)

			if tt.checkDuration {
				elapsed := time.Since(result.Time)
				assert.GreaterOrEqual(t, elapsed, tt.minAgo,
					"ParseTimestamp(%q) time should be at least %v ago", tt.input, tt.minAgo)
				assert.LessOrEqual(t, elapsed, tt.maxAgo,
					"ParseTimestamp(%q) time should be at most %v ago", tt.input, tt.maxAgo)
			}
		})
	}
}

func TestParseTimestamp_AbsoluteTimes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "9am",
			input:       "9am",
			expectError: false,
		},
		{
			name:        "9 am",
			input:       "9 am",
			expectError: false,
		},
		{
			name:        "9:00am",
			input:       "9:00am",
			expectError: false,
		},
		{
			name:        "14:30",
			input:       "14:30",
			expectError: false,
		},
		{
			name:        "2:30pm",
			input:       "2:30pm",
			expectError: false,
		},
		{
			name:        "2:30 PM",
			input:       "2:30 PM",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tt.input)
			if tt.expectError {
				assert.Error(t, result.Error, "ParseTimestamp(%q) should error", tt.input)
			} else {
				assert.NoError(t, result.Error, "ParseTimestamp(%q) should not error", tt.input)
				assert.False(t, result.Time.IsZero(), "ParseTimestamp(%q) should return non-zero time", tt.input)
			}
		})
	}
}

func TestParseTimestamp_PeriodExpressions(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		input     string
		checkFunc func(t *testing.T, result parser.TimestampResult)
	}{
		{
			name:  "this week",
			input: "this week",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				// Should be start of current week (Monday)
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				expected := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, expected.Month(), result.Time.Month())
				assert.Equal(t, expected.Day(), result.Time.Day())
			},
		},
		{
			name:  "last week",
			input: "last week",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				// Should be 7 days before start of current week
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				expected := time.Date(now.Year(), now.Month(), now.Day()-weekday+1-7, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, expected.Month(), result.Time.Month())
				assert.Equal(t, expected.Day(), result.Time.Day())
			},
		},
		{
			name:  "this month",
			input: "this month",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, expected.Month(), result.Time.Month())
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "last month",
			input: "last month",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, expected.Month(), result.Time.Month())
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "this year",
			input: "this year",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, time.January, result.Time.Month())
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "last year",
			input: "last year",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, time.January, result.Time.Month())
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "this day",
			input: "this day",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, expected.Month(), result.Time.Month())
				assert.Equal(t, expected.Day(), result.Time.Day())
			},
		},
		{
			name:  "current week",
			input: "current week",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				// Should behave same as "this week"
			},
		},
		{
			name:  "previous month",
			input: "previous month",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Year(), result.Time.Year())
				assert.Equal(t, expected.Month(), result.Time.Month())
			},
		},
		{
			name:  "this hour",
			input: "this hour",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
				assert.Equal(t, expected.Hour(), result.Time.Hour())
			},
		},
		{
			name:  "last hour",
			input: "last hour",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				expected := time.Date(now.Year(), now.Month(), now.Day(), now.Hour()-1, 0, 0, 0, now.Location())
				assert.Equal(t, expected.Hour(), result.Time.Hour())
			},
		},
		{
			name:  "this quarter",
			input: "this quarter",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				quarter := (int(now.Month()) - 1) / 3
				expectedMonth := time.Month(quarter*3 + 1)
				assert.Equal(t, expectedMonth, result.Time.Month())
				assert.Equal(t, 1, result.Time.Day())
			},
		},
		{
			name:  "last quarter",
			input: "last quarter",
			checkFunc: func(t *testing.T, result parser.TimestampResult) {
				assert.NoError(t, result.Error)
				quarter := (int(now.Month()) - 1) / 3
				expected := time.Date(now.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, now.Location())
				expected = expected.AddDate(0, -3, 0)
				assert.Equal(t, expected.Month(), result.Time.Month())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tt.input)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

func TestParseTimestamp_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "invalid input",
			input:       "not a valid time expression xyz123",
			expectError: true,
		},
		{
			name:        "whitespace only",
			input:       "   ",
			expectError: false, // treated as empty, returns now
		},
		{
			name:        "special characters only",
			input:       "!@#$%",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.ParseTimestamp(tt.input)
			if tt.expectError {
				assert.Error(t, result.Error, "ParseTimestamp(%q) should error", tt.input)
			} else {
				assert.NoError(t, result.Error, "ParseTimestamp(%q) should not error", tt.input)
			}
		})
	}
}

func TestGetPeriodRange(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		period    string
		checkFunc func(t *testing.T, tr parser.TimeRange)
	}{
		{
			name:   "today",
			period: "today",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 0, 1)
				assert.Equal(t, expectedStart, tr.Start, "Start should be beginning of today")
				assert.Equal(t, expectedEnd, tr.End, "End should be beginning of tomorrow")
			},
		},
		{
			name:   "yesterday",
			period: "yesterday",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 0, 1)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "this week",
			period: "this week",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				expectedStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 0, 7)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "last week",
			period: "last week",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				weekday := int(now.Weekday())
				if weekday == 0 {
					weekday = 7
				}
				expectedStart := time.Date(now.Year(), now.Month(), now.Day()-weekday+1-7, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 0, 7)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "this month",
			period: "this month",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 1, 0)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "last month",
			period: "last month",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 1, 0)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "this year",
			period: "this year",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(1, 0, 0)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "last year",
			period: "last year",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(1, 0, 0)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
		{
			name:   "unknown period defaults to today",
			period: "unknown",
			checkFunc: func(t *testing.T, tr parser.TimeRange) {
				expectedStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				expectedEnd := expectedStart.AddDate(0, 0, 1)
				assert.Equal(t, expectedStart, tr.Start)
				assert.Equal(t, expectedEnd, tr.End)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.GetPeriodRange(tt.period)
			tt.checkFunc(t, result)
		})
	}
}

// =============================================================================
// Argument Parser Tests (args.go)
// =============================================================================

func TestParse_OnProjectSyntax(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantProject  string
		wantHasProj  bool
	}{
		{
			name:         "on project",
			args:         []string{"on", "myproject"},
			wantProject:  "myproject",
			wantHasProj:  true,
		},
		{
			name:         "to project",
			args:         []string{"to", "myproject"},
			wantProject:  "myproject",
			wantHasProj:  true,
		},
		{
			name:         "of project",
			args:         []string{"of", "myproject"},
			wantProject:  "myproject",
			wantHasProj:  true,
		},
		{
			name:         "on project with time",
			args:         []string{"on", "myproject", "2", "hours", "ago"},
			wantProject:  "myproject",
			wantHasProj:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.args)
			assert.Equal(t, tt.wantHasProj, result.HasProject, "HasProject mismatch")
			if tt.wantHasProj {
				assert.Equal(t, tt.wantProject, result.ProjectSID, "ProjectSID mismatch")
			}
		})
	}
}

func TestParse_OnProjectTaskSyntax(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantProject string
		wantTask    string
		wantHasProj bool
		wantHasTask bool
	}{
		{
			name:        "on project/task",
			args:        []string{"on", "myproject/mytask"},
			wantProject: "myproject",
			wantTask:    "mytask",
			wantHasProj: true,
			wantHasTask: true,
		},
		{
			name:        "to project/task with time",
			args:        []string{"to", "client/billing", "1", "hour", "ago"},
			wantProject: "client",
			wantTask:    "billing",
			wantHasProj: true,
			wantHasTask: true,
		},
		{
			name:        "project only no task",
			args:        []string{"on", "projectonly"},
			wantProject: "projectonly",
			wantTask:    "",
			wantHasProj: true,
			wantHasTask: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.args)
			assert.Equal(t, tt.wantHasProj, result.HasProject, "HasProject mismatch")
			assert.Equal(t, tt.wantHasTask, result.HasTask, "HasTask mismatch")
			assert.Equal(t, tt.wantProject, result.ProjectSID, "ProjectSID mismatch")
			assert.Equal(t, tt.wantTask, result.TaskSID, "TaskSID mismatch")
		})
	}
}

func TestParse_WithNoteSyntax(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantNote    string
		wantHasNote bool
	}{
		{
			name:        "with note double quotes",
			args:        []string{"on", "project", "with", "note", "\"my note text\""},
			wantNote:    "my note text",
			wantHasNote: true,
		},
		{
			name:        "with note single quotes",
			args:        []string{"on", "project", "with", "note", "'single quoted note'"},
			wantNote:    "single quoted note",
			wantHasNote: true,
		},
		{
			name:        "note with special characters",
			args:        []string{"with", "note", "\"note with numbers 123 and symbols!\""},
			wantNote:    "note with numbers 123 and symbols!",
			wantHasNote: true,
		},
		{
			name:        "no note",
			args:        []string{"on", "project"},
			wantNote:    "",
			wantHasNote: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.args)
			assert.Equal(t, tt.wantHasNote, result.HasNote, "HasNote mismatch")
			assert.Equal(t, tt.wantNote, result.Note, "Note mismatch")
		})
	}
}

func TestParse_TimeExpressions(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantRawStart string
		wantHasStart bool
	}{
		{
			name:         "2 hours ago",
			args:         []string{"2", "hours", "ago"},
			wantRawStart: "2 hours ago",
			wantHasStart: true,
		},
		{
			name:         "yesterday",
			args:         []string{"yesterday"},
			wantRawStart: "yesterday",
			wantHasStart: true,
		},
		{
			name:         "project with time",
			args:         []string{"on", "project", "2", "hours", "ago"},
			wantRawStart: "2 hours ago",
			wantHasStart: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.args)
			assert.Equal(t, tt.wantHasStart, result.HasStart, "HasStart mismatch")
			if tt.wantHasStart {
				assert.Equal(t, tt.wantRawStart, result.RawTimestampStart, "RawTimestampStart mismatch")
			}
		})
	}
}

func TestParse_EndTimestamp(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantRawStart string
		wantRawEnd   string
		wantHasStart bool
		wantHasEnd   bool
	}{
		{
			name:         "start end syntax",
			args:         []string{"9am", "end", "5pm"},
			wantRawStart: "9am",
			wantRawEnd:   "5pm",
			wantHasStart: true,
			wantHasEnd:   true,
		},
		{
			name:         "start until syntax",
			args:         []string{"10am", "until", "12pm"},
			wantRawStart: "10am",
			wantRawEnd:   "12pm",
			wantHasStart: true,
			wantHasEnd:   true,
		},
		{
			name:         "ended keyword",
			args:         []string{"9am", "ended", "3pm"},
			wantRawStart: "9am",
			wantRawEnd:   "3pm",
			wantHasStart: true,
			wantHasEnd:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.args)
			assert.Equal(t, tt.wantHasStart, result.HasStart, "HasStart mismatch")
			assert.Equal(t, tt.wantHasEnd, result.HasEnd, "HasEnd mismatch")
			if tt.wantHasStart {
				assert.Equal(t, tt.wantRawStart, result.RawTimestampStart, "RawTimestampStart mismatch")
			}
			if tt.wantHasEnd {
				assert.Equal(t, tt.wantRawEnd, result.RawTimestampEnd, "RawTimestampEnd mismatch")
			}
		})
	}
}

func TestParse_EmptyArgs(t *testing.T) {
	result := parser.Parse([]string{})

	assert.False(t, result.HasProject, "Should not have project")
	assert.False(t, result.HasTask, "Should not have task")
	assert.False(t, result.HasNote, "Should not have note")
	assert.False(t, result.HasStart, "Should not have start")
	assert.False(t, result.HasEnd, "Should not have end")
	assert.Empty(t, result.ProjectSID)
	assert.Empty(t, result.TaskSID)
	assert.Empty(t, result.Note)
}

func TestParsedArgs_Merge(t *testing.T) {
	tests := []struct {
		name         string
		initial      *parser.ParsedArgs
		projectFlag  string
		taskFlag     string
		noteFlag     string
		startFlag    string
		endFlag      string
		wantProject  string
		wantTask     string
		wantNote     string
		wantRawStart string
		wantRawEnd   string
	}{
		{
			name:         "flags override parsed values",
			initial:      &parser.ParsedArgs{ProjectSID: "parsed-proj", TaskSID: "parsed-task"},
			projectFlag:  "flag-proj",
			taskFlag:     "flag-task",
			noteFlag:     "flag note",
			startFlag:    "9am",
			endFlag:      "5pm",
			wantProject:  "flag-proj",
			wantTask:     "flag-task",
			wantNote:     "flag note",
			wantRawStart: "9am",
			wantRawEnd:   "5pm",
		},
		{
			name:        "empty flags preserve original values",
			initial:     &parser.ParsedArgs{ProjectSID: "original", TaskSID: "original-task", Note: "original note"},
			projectFlag: "",
			taskFlag:    "",
			noteFlag:    "",
			startFlag:   "",
			endFlag:     "",
			wantProject: "original",
			wantTask:    "original-task",
			wantNote:    "original note",
		},
		{
			name:        "partial override",
			initial:     &parser.ParsedArgs{ProjectSID: "original", TaskSID: "original-task"},
			projectFlag: "new-proj",
			taskFlag:    "",
			noteFlag:    "new note",
			wantProject: "new-proj",
			wantTask:    "original-task",
			wantNote:    "new note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Merge(tt.projectFlag, tt.taskFlag, tt.noteFlag, tt.startFlag, tt.endFlag)

			assert.Equal(t, tt.wantProject, tt.initial.ProjectSID, "ProjectSID mismatch after merge")
			assert.Equal(t, tt.wantTask, tt.initial.TaskSID, "TaskSID mismatch after merge")
			assert.Equal(t, tt.wantNote, tt.initial.Note, "Note mismatch after merge")
			if tt.startFlag != "" {
				assert.Equal(t, tt.wantRawStart, tt.initial.RawTimestampStart, "RawTimestampStart mismatch")
				assert.True(t, tt.initial.HasStart)
			}
			if tt.endFlag != "" {
				assert.Equal(t, tt.wantRawEnd, tt.initial.RawTimestampEnd, "RawTimestampEnd mismatch")
				assert.True(t, tt.initial.HasEnd)
			}
		})
	}
}

func TestParsedArgs_Process(t *testing.T) {
	t.Run("normalizes SIDs", func(t *testing.T) {
		args := &parser.ParsedArgs{
			RawProject: "My Project/My Task",
		}
		err := args.Process()

		assert.NoError(t, err)
		assert.Equal(t, "my-project", args.ProjectSID)
		assert.Equal(t, "my-task", args.TaskSID)
		assert.True(t, args.HasProject)
		assert.True(t, args.HasTask)
	})

	t.Run("processes timestamps", func(t *testing.T) {
		args := &parser.ParsedArgs{
			RawTimestampStart: "now",
			HasStart:          true,
		}
		err := args.Process()

		assert.NoError(t, err)
		assert.WithinDuration(t, time.Now(), args.TimestampStart, time.Second)
	})

	t.Run("sets default start time when not provided", func(t *testing.T) {
		args := &parser.ParsedArgs{}
		err := args.Process()

		assert.NoError(t, err)
		assert.WithinDuration(t, time.Now(), args.TimestampStart, time.Second)
	})

	t.Run("returns error for invalid start timestamp", func(t *testing.T) {
		args := &parser.ParsedArgs{
			RawTimestampStart: "not a valid time xyz123",
			HasStart:          true,
		}
		err := args.Process()

		assert.Error(t, err)
	})

	t.Run("returns error for invalid end timestamp", func(t *testing.T) {
		args := &parser.ParsedArgs{
			RawTimestampEnd: "not a valid time xyz123",
			HasEnd:          true,
		}
		err := args.Process()

		assert.Error(t, err)
	})

	t.Run("processes both start and end timestamps", func(t *testing.T) {
		args := &parser.ParsedArgs{
			RawTimestampStart: "9am",
			RawTimestampEnd:   "5pm",
			HasStart:          true,
			HasEnd:            true,
		}
		err := args.Process()

		assert.NoError(t, err)
		assert.False(t, args.TimestampStart.IsZero())
		assert.False(t, args.TimestampEnd.IsZero())
	})
}

func TestParse_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantProject string
		wantTask    string
		wantNote    string
		wantHasProj bool
		wantHasTask bool
		wantHasNote bool
		wantHasTime bool
	}{
		{
			name:        "full command with all parts",
			args:        []string{"on", "client/billing", "with", "note", "\"fixing invoice\"", "2", "hours", "ago"},
			wantProject: "client",
			wantTask:    "billing",
			wantNote:    "fixing invoice",
			wantHasProj: true,
			wantHasTask: true,
			wantHasNote: true,
			wantHasTime: true,
		},
		{
			name:        "project and time only",
			args:        []string{"on", "myproject", "yesterday"},
			wantProject: "myproject",
			wantTask:    "",
			wantNote:    "",
			wantHasProj: true,
			wantHasTask: false,
			wantHasNote: false,
			wantHasTime: true,
		},
		{
			name:        "note only",
			args:        []string{"with", "note", "\"just a note\""},
			wantProject: "",
			wantTask:    "",
			wantNote:    "just a note",
			wantHasProj: false,
			wantHasTask: false,
			wantHasNote: true,
			wantHasTime: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.args)

			assert.Equal(t, tt.wantHasProj, result.HasProject, "HasProject mismatch")
			assert.Equal(t, tt.wantHasTask, result.HasTask, "HasTask mismatch")
			assert.Equal(t, tt.wantHasNote, result.HasNote, "HasNote mismatch")
			assert.Equal(t, tt.wantProject, result.ProjectSID, "ProjectSID mismatch")
			assert.Equal(t, tt.wantTask, result.TaskSID, "TaskSID mismatch")
			assert.Equal(t, tt.wantNote, result.Note, "Note mismatch")

			if tt.wantHasTime {
				assert.True(t, result.HasStart || result.HasEnd, "Should have timestamp")
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkValidateSID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parser.ValidateSID("my-project-name")
	}
}

func BenchmarkConvertToSID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parser.ConvertToSID("My Client Project!")
	}
}

func BenchmarkParseTimestamp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		parser.ParseTimestamp("2 hours ago")
	}
}

func BenchmarkParse(b *testing.B) {
	args := []string{"on", "client/billing", "with", "note", "\"test note\"", "2", "hours", "ago"}
	for i := 0; i < b.N; i++ {
		parser.Parse(args)
	}
}
