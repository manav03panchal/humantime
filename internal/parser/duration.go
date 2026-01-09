package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DurationResult represents the result of parsing a duration.
type DurationResult struct {
	Duration time.Duration
	Valid    bool
	Error    error
}

// durationPattern matches duration expressions like "2h", "30m", "1h30m", "2.5h", etc.
var durationPattern = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*(h|hr|hrs|hour|hours|m|min|mins|minute|minutes|s|sec|secs|second|seconds)?\s*(?:(\d+(?:\.\d+)?)\s*(m|min|mins|minute|minutes))?$`)

// ParseDuration parses a human-readable duration string.
// Supports formats like:
//   - "2h" or "2 hours"
//   - "30m" or "30 minutes"
//   - "1h30m" or "1 hour 30 minutes"
//   - "2.5h" (2 hours 30 minutes)
//   - "90m" (90 minutes)
func ParseDuration(input string) DurationResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return DurationResult{Valid: false}
	}

	// Try standard Go duration format first (e.g., "2h30m")
	if d, err := time.ParseDuration(input); err == nil {
		return DurationResult{Duration: d, Valid: true}
	}

	// Try our custom patterns
	matches := durationPattern.FindStringSubmatch(input)
	if matches == nil {
		return DurationResult{Valid: false}
	}

	var totalDuration time.Duration

	// First number and unit
	if matches[1] != "" {
		value, _ := strconv.ParseFloat(matches[1], 64)
		unit := strings.ToLower(matches[2])
		if unit == "" {
			// Default to hours if no unit
			unit = "h"
		}
		totalDuration += unitToDuration(value, unit)
	}

	// Second number and unit (for "1h30m" style)
	if matches[3] != "" {
		value, _ := strconv.ParseFloat(matches[3], 64)
		unit := strings.ToLower(matches[4])
		totalDuration += unitToDuration(value, unit)
	}

	if totalDuration == 0 {
		return DurationResult{Valid: false}
	}

	return DurationResult{Duration: totalDuration, Valid: true}
}

// unitToDuration converts a value and unit to a duration.
func unitToDuration(value float64, unit string) time.Duration {
	switch unit {
	case "h", "hr", "hrs", "hour", "hours":
		return time.Duration(value * float64(time.Hour))
	case "m", "min", "mins", "minute", "minutes":
		return time.Duration(value * float64(time.Minute))
	case "s", "sec", "secs", "second", "seconds":
		return time.Duration(value * float64(time.Second))
	default:
		// Default to hours
		return time.Duration(value * float64(time.Hour))
	}
}

// IsDurationLike checks if a string looks like a duration expression.
func IsDurationLike(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return false
	}

	// Check if it starts with a digit
	if len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		// And contains common duration indicators
		durationIndicators := []string{"h", "hr", "hour", "m", "min", "minute", "s", "sec", "second"}
		for _, ind := range durationIndicators {
			if strings.Contains(s, ind) {
				return true
			}
		}
		// If just a number without unit, could be hours
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return true
		}
	}
	return false
}
