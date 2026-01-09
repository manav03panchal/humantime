// Package parser provides argument and timestamp parsing for Humantime.
package parser

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	// MaxSIDLength is the maximum length for a Simplified ID.
	MaxSIDLength = 32
)

var (
	// sidRegex validates SID format: alphanumeric, dash, underscore, period.
	sidRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_.]+$`)

	// reservedSIDs are SIDs that cannot be used to avoid command conflicts.
	reservedSIDs = map[string]bool{
		"edit":   true,
		"create": true,
		"delete": true,
		"list":   true,
		"show":   true,
		"set":    true,
	}
)

// ValidateSID checks if a string is a valid SID.
func ValidateSID(sid string) bool {
	if sid == "" || len(sid) > MaxSIDLength {
		return false
	}
	if reservedSIDs[strings.ToLower(sid)] {
		return false
	}
	return sidRegex.MatchString(sid)
}

// ConvertToSID converts a display name to a valid SID.
// Example: "My Client Project!" -> "my-client-project"
func ConvertToSID(displayName string) string {
	// Convert to lowercase
	result := strings.ToLower(displayName)

	// Replace spaces with hyphens
	result = strings.ReplaceAll(result, " ", "-")

	// Remove invalid characters
	var sb strings.Builder
	for _, r := range result {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			sb.WriteRune(r)
		}
	}
	result = sb.String()

	// Remove consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}

	// Trim leading/trailing hyphens
	result = strings.Trim(result, "-")

	// Truncate if too long
	if len(result) > MaxSIDLength {
		result = result[:MaxSIDLength]
	}

	return result
}

// ParseProjectTask parses a "project/task" notation.
// Returns (projectSID, taskSID). If no task, taskSID is empty.
func ParseProjectTask(input string) (projectSID, taskSID string) {
	parts := strings.SplitN(input, "/", 2)
	projectSID = strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		taskSID = strings.TrimSpace(parts[1])
	}

	return projectSID, taskSID
}

// NormalizeSID ensures a SID is valid, converting if necessary.
func NormalizeSID(input string) string {
	input = strings.TrimSpace(input)
	if ValidateSID(input) {
		return input
	}
	return ConvertToSID(input)
}
