package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSID(t *testing.T) {
	tests := []struct {
		name     string
		sid      string
		expected bool
	}{
		// Valid SIDs
		{"simple_lowercase", "myproject", true},
		{"with_hyphen", "my-project", true},
		{"with_underscore", "my_project", true},
		{"with_period", "my.project", true},
		{"with_numbers", "project123", true},
		{"mixed", "my-project_123.v2", true},
		{"uppercase", "MyProject", true},

		// Invalid SIDs
		{"empty", "", false},
		{"too_long", strings.Repeat("a", MaxSIDLength+1), false},
		{"with_space", "my project", false},
		{"with_special_chars", "my@project", false},
		{"with_slash", "my/project", false},

		// Reserved SIDs
		{"reserved_edit", "edit", false},
		{"reserved_create", "create", false},
		{"reserved_delete", "delete", false},
		{"reserved_list", "list", false},
		{"reserved_show", "show", false},
		{"reserved_set", "set", false},
		{"reserved_case_insensitive", "EDIT", false},
		{"reserved_mixed_case", "Edit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSID(tt.sid)
			assert.Equal(t, tt.expected, result, "ValidateSID(%q)", tt.sid)
		})
	}
}

func TestConvertToSID(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		expected    string
	}{
		{"simple", "My Project", "my-project"},
		{"with_special_chars", "My Client Project!", "my-client-project"},
		{"multiple_spaces", "My  Project", "my-project"},
		{"leading_trailing_spaces", "  My Project  ", "my-project"},
		{"uppercase", "MY PROJECT", "my-project"},
		{"with_numbers", "Project 123", "project-123"},
		{"underscore_preserved", "my_project", "my_project"},
		{"period_preserved", "my.project", "my.project"},
		{"complex", "Client: Acme Corp. (2023)", "client-acme-corp.-2023"},
		{"only_special", "!@#$%", ""},
		{"mixed_hyphens", "my---project", "my-project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToSID(tt.displayName)
			assert.Equal(t, tt.expected, result, "ConvertToSID(%q)", tt.displayName)
		})
	}
}

func TestConvertToSIDTruncation(t *testing.T) {
	longName := strings.Repeat("a", 100)
	result := ConvertToSID(longName)
	assert.LessOrEqual(t, len(result), MaxSIDLength)
}

func TestParseProjectTask(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		projectSID  string
		taskSID     string
	}{
		{"project_only", "myproject", "myproject", ""},
		{"project_and_task", "myproject/mytask", "myproject", "mytask"},
		{"with_whitespace", " myproject / mytask ", "myproject", "mytask"},
		{"empty_task", "myproject/", "myproject", ""},
		{"multiple_slashes", "a/b/c", "a", "b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, task := ParseProjectTask(tt.input)
			assert.Equal(t, tt.projectSID, project, "project for %q", tt.input)
			assert.Equal(t, tt.taskSID, task, "task for %q", tt.input)
		})
	}
}

func TestNormalizeSID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid_sid", "myproject", "myproject"},
		{"needs_conversion", "My Project", "my-project"},
		{"with_whitespace", "  myproject  ", "myproject"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeSID(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizeSID(%q)", tt.input)
		})
	}
}

func TestMaxSIDLength(t *testing.T) {
	assert.Equal(t, 32, MaxSIDLength)
}
